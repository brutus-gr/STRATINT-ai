package forecaster

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/STRATINT/stratint/internal/inference"
	"github.com/STRATINT/stratint/internal/models"
	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/sashabaranov/go-openai"
)

const (
	// Temperature for sampling (higher = more randomness)
	samplingTemperature = 1.0
)

// EventRepository defines methods needed to fetch events for forecasting
type EventRepository interface {
	Query(ctx context.Context, query models.EventQuery) (*models.EventResponse, error)
}

// ForecastRepository defines methods needed for forecast storage
type ForecastRepository interface {
	GetForecast(ctx context.Context, id string) (*models.Forecast, error)
	GetForecastModels(ctx context.Context, forecastID string) ([]models.ForecastModel, error)
	CreateForecastRun(ctx context.Context, forecastID string, headlines []models.ForecastHeadline) (string, error)
	UpdateForecastRunStatus(ctx context.Context, runID, status, errorMsg string) error
	CreateModelResponse(ctx context.Context, response models.ForecastModelResponse) error
	CreateForecastResult(ctx context.Context, result models.ForecastResult) error
	GetForecastRun(ctx context.Context, runID string) (*models.ForecastRunDetail, error)
}

// Forecaster executes forecasts using multiple AI models
type Forecaster struct {
	eventRepo       EventRepository
	forecastRepo    ForecastRepository
	logger          *slog.Logger
	inferenceLogger *inference.Logger
}

// NewForecaster creates a new forecaster
func NewForecaster(eventRepo EventRepository, forecastRepo ForecastRepository, logger *slog.Logger, inferenceLogger *inference.Logger) *Forecaster {
	return &Forecaster{
		eventRepo:       eventRepo,
		forecastRepo:    forecastRepo,
		logger:          logger,
		inferenceLogger: inferenceLogger,
	}
}

// parsePercentiles extracts five comma-separated percentile values from model response
// Returns PercentilePredictions or error if not found/invalid
func parsePercentiles(content string) (*models.PercentilePredictions, error) {
	// Trim and clean the response
	content = strings.TrimSpace(content)

	// Look for comma-separated numbers in the last few lines
	lines := strings.Split(content, "\n")
	for i := len(lines) - 1; i >= 0 && i >= len(lines)-3; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		// Try to parse as comma-separated values
		parts := strings.Split(line, ",")
		if len(parts) == 5 {
			var values [5]float64
			allValid := true
			for j, part := range parts {
				part = strings.TrimSpace(part)
				// Remove any % symbols or other non-numeric characters except . and -
				cleaned := ""
				for _, ch := range part {
					if (ch >= '0' && ch <= '9') || ch == '.' || ch == '-' {
						cleaned += string(ch)
					}
				}
				if n, err := fmt.Sscanf(cleaned, "%f", &values[j]); err != nil || n != 1 {
					allValid = false
					break
				}
			}
			if allValid {
				// Validate that percentiles are in ascending order
				if values[0] <= values[1] && values[1] <= values[2] && values[2] <= values[3] && values[3] <= values[4] {
					return &models.PercentilePredictions{
						P10: values[0],
						P25: values[1],
						P50: values[2],
						P75: values[3],
						P90: values[4],
					}, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("could not parse percentiles from response: %s", content)
}

// parsePointEstimate extracts a single numeric value from model response
// Returns the value as float64 or error if not found
func parsePointEstimate(content string) (float64, error) {
	// Trim and clean the response
	content = strings.TrimSpace(content)

	// Look in the last few lines
	lines := strings.Split(content, "\n")
	for i := len(lines) - 1; i >= 0 && i >= len(lines)-3; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		// Remove any non-numeric characters except . and -
		cleaned := ""
		for _, ch := range line {
			if (ch >= '0' && ch <= '9') || ch == '.' || ch == '-' {
				cleaned += string(ch)
			}
		}

		var num float64
		if n, err := fmt.Sscanf(cleaned, "%f", &num); err == nil && n == 1 {
			return num, nil
		}
	}

	return 0, fmt.Errorf("could not parse point estimate from response: %s", content)
}

// ExecuteForecast runs a forecast
func (f *Forecaster) ExecuteForecast(ctx context.Context, forecastID string) (string, error) {
	f.logger.Info("starting forecast execution", "forecast_id", forecastID)

	// Get forecast config
	forecast, err := f.forecastRepo.GetForecast(ctx, forecastID)
	if err != nil {
		return "", fmt.Errorf("failed to get forecast: %w", err)
	}
	if forecast == nil {
		return "", fmt.Errorf("forecast not found: %s", forecastID)
	}

	// Get forecast models
	models, err := f.forecastRepo.GetForecastModels(ctx, forecastID)
	if err != nil {
		return "", fmt.Errorf("failed to get forecast models: %w", err)
	}
	if len(models) == 0 {
		return "", fmt.Errorf("no models configured for forecast: %s", forecastID)
	}

	// Fetch recent headlines
	headlines, err := f.fetchHeadlines(ctx, forecast)
	if err != nil {
		return "", fmt.Errorf("failed to fetch headlines: %w", err)
	}

	f.logger.Info("fetched headlines for forecast",
		"forecast_id", forecastID,
		"headline_count", len(headlines))

	// Create forecast run
	runID, err := f.forecastRepo.CreateForecastRun(ctx, forecastID, headlines)
	if err != nil {
		return "", fmt.Errorf("failed to create forecast run: %w", err)
	}

	// Update status to running
	if err := f.forecastRepo.UpdateForecastRunStatus(ctx, runID, "running", ""); err != nil {
		return "", fmt.Errorf("failed to update run status: %w", err)
	}

	// Execute forecast asynchronously
	go f.executeForecastAsync(context.Background(), runID, forecast, models, headlines)

	return runID, nil
}

func (f *Forecaster) executeForecastAsync(ctx context.Context, runID string, forecast *models.Forecast, forecastModels []models.ForecastModel, headlines []models.ForecastHeadline) {
	defer func() {
		if r := recover(); r != nil {
			f.logger.Error("panic in forecast execution", "run_id", runID, "panic", r)
			f.forecastRepo.UpdateForecastRunStatus(ctx, runID, "failed", fmt.Sprintf("panic: %v", r))
		}
	}()

	// Query each model
	var responses []models.ForecastModelResponse
	var totalWeight float64

	// Use iterations as the number of samples (configurable 1-50)
	numSamples := forecast.Iterations

	for _, model := range forecastModels {
		f.logger.Info("querying model",
			"run_id", runID,
			"provider", model.Provider,
			"model", model.ModelName,
			"num_samples", numSamples)

		startTime := time.Now()
		response, err := f.queryModel(ctx, forecast, &model, headlines, numSamples)
		responseTime := int(time.Since(startTime).Milliseconds())

		if err != nil {
			f.logger.Error("model query failed",
				"run_id", runID,
				"provider", model.Provider,
				"model", model.ModelName,
				"error", err)

			// Store failed response
			failedResp := models.ForecastModelResponse{
				RunID:          runID,
				ModelID:        model.ID,
				Provider:       model.Provider,
				ModelName:      model.ModelName,
				Status:         "failed",
				ErrorMessage:   err.Error(),
				ResponseTimeMs: &responseTime,
			}
			f.forecastRepo.CreateModelResponse(ctx, failedResp)
			continue
		}

		// Update response with run metadata
		response.RunID = runID
		response.ResponseTimeMs = &responseTime

		responses = append(responses, *response)
		totalWeight += model.Weight

		// Store response
		if err := f.forecastRepo.CreateModelResponse(ctx, *response); err != nil {
			f.logger.Error("failed to store model response", "error", err)
		}
	}

	if len(responses) == 0 {
		f.forecastRepo.UpdateForecastRunStatus(ctx, runID, "failed", "all models failed")
		return
	}

	// Calculate weighted average
	result := f.calculateWeightedResult(responses, forecastModels, totalWeight)
	result.RunID = runID

	// Store result
	if err := f.forecastRepo.CreateForecastResult(ctx, result); err != nil {
		f.logger.Error("failed to store forecast result", "error", err)
		f.forecastRepo.UpdateForecastRunStatus(ctx, runID, "failed", fmt.Sprintf("failed to store result: %v", err))
		return
	}

	// Mark run as completed
	f.forecastRepo.UpdateForecastRunStatus(ctx, runID, "completed", "")

	f.logger.Info("forecast execution completed",
		"run_id", runID,
		"model_count", result.ModelCount)
}

func (f *Forecaster) fetchHeadlines(ctx context.Context, forecast *models.Forecast) ([]models.ForecastHeadline, error) {
	// Build query
	query := models.EventQuery{
		Limit:     forecast.HeadlineCount,
		Page:      1,
		SortBy:    "timestamp",
		SortOrder: "desc",
	}

	// Filter by categories if specified
	if len(forecast.Categories) > 0 {
		categories := make([]models.Category, len(forecast.Categories))
		for i, cat := range forecast.Categories {
			categories[i] = models.Category(cat)
		}
		query.Categories = categories
	}

	// Query events
	resp, err := f.eventRepo.Query(ctx, query)
	if err != nil {
		return nil, err
	}

	f.logger.Info("fetched headlines from database",
		"requested", forecast.HeadlineCount,
		"received", len(resp.Events),
		"categories", forecast.Categories)

	// Convert to headlines
	headlines := make([]models.ForecastHeadline, 0, len(resp.Events))
	for _, event := range resp.Events {
		headlines = append(headlines, models.ForecastHeadline{
			EventID:   event.ID,
			Title:     event.Title,
			Category:  string(event.Category),
			Magnitude: event.Magnitude,
			Timestamp: event.Timestamp,
		})
	}

	return headlines, nil
}

func (f *Forecaster) queryModel(ctx context.Context, forecast *models.Forecast, model *models.ForecastModel, headlines []models.ForecastHeadline, numSamples int) (*models.ForecastModelResponse, error) {
	// Get max context length for this model
	maxTokens := f.getModelContextLength(model)

	// Truncate headlines if needed to fit in context window
	// Reserve ~1500 tokens for system prompt, proposition, and response
	// Estimate ~80 tokens per headline on average
	maxHeadlines := (maxTokens - 1500) / 80
	if maxHeadlines < 10 {
		maxHeadlines = 10 // Always include at least 10 headlines
	}

	truncatedHeadlines := headlines
	if len(headlines) > maxHeadlines {
		truncatedHeadlines = headlines[:maxHeadlines]
		f.logger.Info("truncating headlines for model context window",
			"model", model.ModelName,
			"original_count", len(headlines),
			"truncated_count", maxHeadlines,
			"max_tokens", maxTokens)
	} else {
		f.logger.Info("no truncation needed",
			"model", model.ModelName,
			"headline_count", len(headlines),
			"max_headlines", maxHeadlines,
			"max_tokens", maxTokens)
	}

	// Build prompt with context from URLs if provided
	prompt, err := f.buildForecastPrompt(ctx, forecast, truncatedHeadlines)
	if err != nil {
		return nil, fmt.Errorf("failed to build prompt: %w", err)
	}

	// Log FULL prompt for debugging
	f.logger.Info("FULL PROMPT BEING SENT TO MODEL",
		"model", model.ModelName,
		"headlines_in_prompt", len(truncatedHeadlines),
		"prompt_length", len(prompt),
		"prediction_type", forecast.PredictionType)

	// Use unified query function for all providers
	return f.queryModelUnified(ctx, forecast, model, prompt, numSamples)
}

func (f *Forecaster) queryModelUnified(ctx context.Context, forecast *models.Forecast, model *models.ForecastModel, prompt string, numSamples int) (*models.ForecastModelResponse, error) {
	// System prompt adapted for value-based predictions
	systemPrompt := "You are an expert intelligence analyst providing forecasts based on evidence. Analyze the data carefully and provide your forecast in the exact format requested."

	isPercentile := forecast.PredictionType == "percentile"

	var allResponses []string
	var totalTokens int
	var firstContent string

	// For percentile forecasts
	var percentileSamples []models.PercentilePredictions

	// For point estimate forecasts
	var pointEstimates []float64

	f.logger.Info("starting forecast sampling",
		"model", model.ModelName,
		"provider", model.Provider,
		"num_samples", numSamples,
		"prediction_type", forecast.PredictionType)

	// Run multiple samples
	for i := 0; i < numSamples; i++ {
		var content string
		var tokens int
		var err error

		switch model.Provider {
		case "openai":
			content, tokens, err = f.callOpenAI(ctx, model, systemPrompt, prompt)
		case "anthropic":
			content, tokens, err = f.callAnthropic(ctx, model, systemPrompt, prompt)
		default:
			return nil, fmt.Errorf("unsupported provider: %s", model.Provider)
		}

		if err != nil {
			f.logger.Error("sample failed", "sample", i+1, "error", err)
			continue
		}

		if content == "" {
			f.logger.Error("empty content in sample", "sample", i+1)
			continue
		}

		allResponses = append(allResponses, content)
		totalTokens += tokens

		if firstContent == "" {
			firstContent = content
		}

		// Parse based on prediction type
		if isPercentile {
			percentiles, err := parsePercentiles(content)
			if err != nil {
				f.logger.Warn("failed to parse percentiles", "sample", i+1, "error", err, "content", content)
				continue
			}

			f.logger.Info("PARSED PERCENTILES",
				"sample", i+1,
				"p10", percentiles.P10,
				"p25", percentiles.P25,
				"p50", percentiles.P50,
				"p75", percentiles.P75,
				"p90", percentiles.P90)

			percentileSamples = append(percentileSamples, *percentiles)
		} else {
			// Point estimate
			value, err := parsePointEstimate(content)
			if err != nil {
				f.logger.Warn("failed to parse point estimate", "sample", i+1, "error", err, "content", content)
				continue
			}

			f.logger.Info("PARSED POINT ESTIMATE",
				"sample", i+1,
				"value", value)

			pointEstimates = append(pointEstimates, value)
		}

		if (i+1)%10 == 0 {
			if isPercentile {
				f.logger.Info("sampling progress", "completed", i+1, "valid_samples", len(percentileSamples))
			} else {
				f.logger.Info("sampling progress", "completed", i+1, "valid_samples", len(pointEstimates))
			}
		}
	}

	// Check if we got any valid samples
	if isPercentile && len(percentileSamples) == 0 {
		return &models.ForecastModelResponse{
			ModelID:      model.ID,
			Provider:     model.Provider,
			ModelName:    model.ModelName,
			Status:       "failed",
			ErrorMessage: fmt.Sprintf("no valid percentile responses after %d samples", numSamples),
		}, fmt.Errorf("no valid percentile responses")
	}

	if !isPercentile && len(pointEstimates) == 0 {
		return &models.ForecastModelResponse{
			ModelID:      model.ID,
			Provider:     model.Provider,
			ModelName:    model.ModelName,
			Status:       "failed",
			ErrorMessage: fmt.Sprintf("no valid point estimate responses after %d samples", numSamples),
		}, fmt.Errorf("no valid point estimate responses")
	}

	response := &models.ForecastModelResponse{
		ModelID:    model.ID,
		Provider:   model.Provider,
		ModelName:  model.ModelName,
		Reasoning:  firstContent,
		TokensUsed: &totalTokens,
		Status:     "completed",
		RawResponse: map[string]interface{}{
			"model":          model.ModelName,
			"num_samples":    numSamples,
			"first_response": firstContent,
			"total_tokens":   totalTokens,
		},
	}

	if isPercentile {
		// Average the percentile samples
		avgPercentiles := averagePercentiles(percentileSamples)
		response.PercentilePredictions = &avgPercentiles
		response.RawResponse["valid_samples"] = len(percentileSamples)
		response.RawResponse["all_samples"] = percentileSamples

		f.logger.Info("percentile sampling complete",
			"valid_samples", len(percentileSamples),
			"avg_p10", avgPercentiles.P10,
			"avg_p25", avgPercentiles.P25,
			"avg_p50", avgPercentiles.P50,
			"avg_p75", avgPercentiles.P75,
			"avg_p90", avgPercentiles.P90)
	} else {
		// Average the point estimates
		var sum float64
		for _, v := range pointEstimates {
			sum += v
		}
		avgValue := sum / float64(len(pointEstimates))
		response.PointEstimate = &avgValue
		response.RawResponse["valid_samples"] = len(pointEstimates)
		response.RawResponse["all_estimates"] = pointEstimates

		f.logger.Info("point estimate sampling complete",
			"valid_samples", len(pointEstimates),
			"avg_estimate", avgValue)
	}

	return response, nil
}

// averagePercentiles calculates the average of multiple percentile predictions
func averagePercentiles(samples []models.PercentilePredictions) models.PercentilePredictions {
	if len(samples) == 0 {
		return models.PercentilePredictions{}
	}

	var sumP10, sumP25, sumP50, sumP75, sumP90 float64
	for _, s := range samples {
		sumP10 += s.P10
		sumP25 += s.P25
		sumP50 += s.P50
		sumP75 += s.P75
		sumP90 += s.P90
	}

	n := float64(len(samples))
	return models.PercentilePredictions{
		P10: sumP10 / n,
		P25: sumP25 / n,
		P50: sumP50 / n,
		P75: sumP75 / n,
		P90: sumP90 / n,
	}
}

func (f *Forecaster) getModelContextLength(model *models.ForecastModel) int {
	// Return max context length based on model name
	modelName := strings.ToLower(model.ModelName)

	// OpenAI o-series models (o1, o3, o4)
	if strings.Contains(modelName, "o1") || strings.Contains(modelName, "o3") || strings.Contains(modelName, "o4") {
		return 200000
	}

	// OpenAI GPT-5 models
	if strings.Contains(modelName, "gpt-5") {
		return 200000
	}

	// OpenAI GPT-4 models
	if strings.Contains(modelName, "gpt-4o") || strings.Contains(modelName, "gpt-4-turbo") {
		return 128000
	}
	if strings.Contains(modelName, "gpt-4") {
		return 8192
	}

	// OpenAI GPT-3.5 models
	if strings.Contains(modelName, "gpt-3.5-turbo-16k") {
		return 16384
	}
	if strings.Contains(modelName, "gpt-3.5") {
		return 4096
	}

	// Anthropic Claude models (Claude 2, 3, 4)
	if strings.Contains(modelName, "claude") {
		return 200000
	}

	// Default conservative estimate
	return 4096
}

func (f *Forecaster) buildForecastPrompt(ctx context.Context, forecast *models.Forecast, headlines []models.ForecastHeadline) (string, error) {
	var sb strings.Builder

	sb.WriteString("You are an expert intelligence analyst providing objective forecasts based on OSINT signals.\n\n")

	sb.WriteString(fmt.Sprintf("QUESTION: %s\n\n", forecast.Proposition))

	// Determine if this is a percentile or point estimate forecast
	isPercentile := forecast.PredictionType == "percentile"

	if isPercentile {
		sb.WriteString(fmt.Sprintf("Review the %d intelligence signals below and provide a percentile-based forecast distribution.\n\n", len(headlines)))
	} else {
		sb.WriteString(fmt.Sprintf("Review the %d intelligence signals below and provide a point estimate forecast.\n\n", len(headlines)))
	}

	sb.WriteString("FORECASTING METHODOLOGY:\n")
	sb.WriteString("1. Consider base rates and historical patterns for this type of question\n")
	sb.WriteString("2. Review the intelligence signals for relevant evidence\n")
	sb.WriteString("3. Weight signals by relevance, magnitude, and recency\n")
	sb.WriteString("4. Apply economic reasoning and domain knowledge\n")
	sb.WriteString("5. Synthesize all factors into a forecast estimate\n\n")

	sb.WriteString("IMPORTANT: Even if few signals directly address the question, you must still provide a well-reasoned forecast based on:\n")
	sb.WriteString("- Historical base rates and trends\n")
	sb.WriteString("- Economic fundamentals and market dynamics\n")
	sb.WriteString("- Indirect signals that might affect the outcome\n")
	sb.WriteString("- Broader geopolitical and economic context\n\n")

	// Fetch and inject context from URLs if provided
	if len(forecast.ContextURLs) > 0 {
		sb.WriteString("CONTEXT DATA (recent factual information):\n\n")

		for i, url := range forecast.ContextURLs {
			f.logger.Info("fetching context from URL", "url", url, "index", i+1)

			content, err := f.fetchURLContent(ctx, url)
			if err != nil {
				f.logger.Error("failed to fetch URL content", "url", url, "error", err)
				sb.WriteString(fmt.Sprintf("%d. [FAILED TO FETCH: %s] Error: %v\n\n", i+1, url, err))
				continue
			}

			sb.WriteString(fmt.Sprintf("%d. Source: %s\n%s\n\n", i+1, url, content))
		}

		sb.WriteString("---\n\n")
	}

	sb.WriteString("INTELLIGENCE SIGNALS (most recent first):\n")
	for i, headline := range headlines {
		sb.WriteString(fmt.Sprintf("%d. [%s | MAG %.1f] %s (%s)\n",
			i+1,
			headline.Category,
			headline.Magnitude,
			headline.Title,
			headline.Timestamp.Format("2006-01-02")))
	}

	sb.WriteString("\n\n=== RESPONSE INSTRUCTIONS ===\n")

	if isPercentile {
		sb.WriteString("Provide your forecast as five percentile values (p10, p25, p50, p75, p90).\n")
		sb.WriteString(fmt.Sprintf("These values represent your uncertainty distribution for: %s\n\n", forecast.Proposition))
		sb.WriteString("CRITICAL: Your response MUST contain EXACTLY five numbers in this order:\n")
		sb.WriteString("p10: The value you're 90% confident the actual result will exceed\n")
		sb.WriteString("p25: The value you're 75% confident the actual result will exceed (Q1)\n")
		sb.WriteString("p50: Your median estimate (50% above, 50% below)\n")
		sb.WriteString("p75: The value you're 25% confident the actual result will exceed (Q3)\n")
		sb.WriteString("p90: The value you're 10% confident the actual result will exceed\n\n")
		sb.WriteString(fmt.Sprintf("Express values in %s.\n", forecast.Units))
		sb.WriteString("Format: p10,p25,p50,p75,p90\n")
		sb.WriteString("Example valid response: -5.2,2.1,8.5,15.3,22.7\n")
		sb.WriteString("Do NOT include:\n")
		sb.WriteString("- Labels or text\n")
		sb.WriteString("- Reasoning or explanation\n")
		sb.WriteString("- Units or % symbols\n")
		sb.WriteString("- Any other text\n\n")
		sb.WriteString("Respond now with ONLY the five comma-separated numbers:")
	} else {
		sb.WriteString("Provide your best point estimate for the question.\n")
		sb.WriteString(fmt.Sprintf("Express your answer in %s.\n\n", forecast.Units))
		sb.WriteString("CRITICAL: Your response MUST contain ONLY a single number.\n")
		sb.WriteString("Do NOT include:\n")
		sb.WriteString("- Reasoning or explanation\n")
		sb.WriteString("- Units or labels\n")
		sb.WriteString("- Any other text\n\n")
		sb.WriteString("Example valid responses: 12.5 or -3.2 or 850\n")
		sb.WriteString("Respond now with ONLY the number:")
	}

	return sb.String(), nil
}

// callOpenAI makes a single OpenAI API call and returns (content, tokens, error)
func (f *Forecaster) callOpenAI(ctx context.Context, model *models.ForecastModel, systemPrompt, userPrompt string) (string, int, error) {
	client := openai.NewClient(model.APIKey)
	modelNameLower := strings.ToLower(model.ModelName)

	// Reasoning models (o1, o3, o4) don't support system messages or temperature
	isReasoningModel := strings.Contains(modelNameLower, "o1") ||
		strings.Contains(modelNameLower, "o3") ||
		strings.Contains(modelNameLower, "o4")

	isGPT5 := strings.Contains(modelNameLower, "gpt-5")

	var req openai.ChatCompletionRequest
	var finalPrompt string

	if isReasoningModel {
		// Merge system prompt into user message for reasoning models
		combinedPrompt := systemPrompt + "\n\n" + userPrompt
		finalPrompt = combinedPrompt
		req = openai.ChatCompletionRequest{
			Model:               model.ModelName,
			MaxCompletionTokens: 1000,
			Messages: []openai.ChatCompletionMessage{
				{Role: openai.ChatMessageRoleUser, Content: combinedPrompt},
			},
		}
	} else if isGPT5 {
		// GPT-5 also prefers merged prompt
		combinedPrompt := systemPrompt + "\n\n" + userPrompt
		finalPrompt = combinedPrompt
		req = openai.ChatCompletionRequest{
			Model:       model.ModelName,
			Temperature: samplingTemperature,
			Messages: []openai.ChatCompletionMessage{
				{Role: openai.ChatMessageRoleUser, Content: combinedPrompt},
			},
		}
	} else {
		// Standard models
		finalPrompt = "SYSTEM: " + systemPrompt + "\n\nUSER: " + userPrompt
		req = openai.ChatCompletionRequest{
			Model:       model.ModelName,
			Temperature: samplingTemperature,
			MaxTokens:   500,
			Messages: []openai.ChatCompletionMessage{
				{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
				{Role: openai.ChatMessageRoleUser, Content: userPrompt},
			},
		}
	}

	f.logger.Info("ACTUAL PROMPT SENT TO OPENAI",
		"model", model.ModelName,
		"is_reasoning_model", isReasoningModel,
		"is_gpt5", isGPT5,
		"FINAL_PROMPT", finalPrompt)

	startTime := time.Now()
	resp, err := client.CreateChatCompletion(ctx, req)
	latency := time.Since(startTime)

	// Log inference call
	if f.inferenceLogger != nil {
		usage := struct {
			PromptTokens     int
			CompletionTokens int
			TotalTokens      int
		}{}
		if err == nil {
			usage.PromptTokens = resp.Usage.PromptTokens
			usage.CompletionTokens = resp.Usage.CompletionTokens
			usage.TotalTokens = resp.Usage.TotalTokens
		}
		f.inferenceLogger.LogOpenAICall(ctx, model.ModelName, "forecast_generation", usage, latency, err, map[string]interface{}{
			"model_id": model.ID,
		})
	}

	if err != nil {
		return "", 0, err
	}

	if len(resp.Choices) == 0 {
		return "", 0, fmt.Errorf("no response choices")
	}

	content := resp.Choices[0].Message.Content
	tokens := resp.Usage.TotalTokens

	f.logger.Info("RESPONSE FROM OPENAI",
		"model", model.ModelName,
		"content", content)

	return content, tokens, nil
}

// callAnthropic makes a single Anthropic API call and returns (content, tokens, error)
func (f *Forecaster) callAnthropic(ctx context.Context, model *models.ForecastModel, systemPrompt, userPrompt string) (string, int, error) {
	client := anthropic.NewClient(option.WithAPIKey(model.APIKey))

	req := anthropic.MessageNewParams{
		Model:       anthropic.Model(model.ModelName),
		MaxTokens:   500,
		Temperature: anthropic.Float(samplingTemperature),
		System: []anthropic.TextBlockParam{
			{Text: systemPrompt},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(userPrompt)),
		},
	}

	startTime := time.Now()
	resp, err := client.Messages.New(ctx, req)
	latency := time.Since(startTime)

	// Log inference call
	if f.inferenceLogger != nil {
		usage := struct {
			InputTokens  int
			OutputTokens int
		}{}
		if err == nil {
			usage.InputTokens = int(resp.Usage.InputTokens)
			usage.OutputTokens = int(resp.Usage.OutputTokens)
		}
		f.inferenceLogger.LogAnthropicCall(ctx, model.ModelName, "forecast_generation", usage, latency, err, map[string]interface{}{
			"model_id": model.ID,
		})
	}

	if err != nil {
		return "", 0, err
	}

	if len(resp.Content) == 0 {
		return "", 0, fmt.Errorf("no response content")
	}

	var content string
	for _, block := range resp.Content {
		if block.Type == "text" {
			content = block.Text
			break
		}
	}

	if content == "" {
		return "", 0, fmt.Errorf("no text content in response")
	}

	tokens := int(resp.Usage.InputTokens + resp.Usage.OutputTokens)

	return content, tokens, nil
}

func (f *Forecaster) calculateWeightedResult(responses []models.ForecastModelResponse, modelConfigs []models.ForecastModel, totalWeight float64) models.ForecastResult {
	// Build model weight map
	weights := make(map[string]float64)
	for _, config := range modelConfigs {
		weights[config.ID] = config.Weight
	}

	// Determine if this is percentile or point estimate based on first valid response
	var isPercentile bool
	for _, resp := range responses {
		if resp.Status == "completed" {
			isPercentile = resp.PercentilePredictions != nil
			break
		}
	}

	var validCount int
	var consensus *float64

	if isPercentile {
		// Calculate weighted average of percentiles
		var weightedP10, weightedP25, weightedP50, weightedP75, weightedP90 float64

		for _, resp := range responses {
			if resp.Status != "completed" || resp.PercentilePredictions == nil {
				continue
			}

			weight := weights[resp.ModelID]
			weightedP10 += resp.PercentilePredictions.P10 * weight
			weightedP25 += resp.PercentilePredictions.P25 * weight
			weightedP50 += resp.PercentilePredictions.P50 * weight
			weightedP75 += resp.PercentilePredictions.P75 * weight
			weightedP90 += resp.PercentilePredictions.P90 * weight
			validCount++
		}

		if totalWeight > 0 {
			weightedP10 /= totalWeight
			weightedP25 /= totalWeight
			weightedP50 /= totalWeight
			weightedP75 /= totalWeight
			weightedP90 /= totalWeight
		}

		// Calculate consensus based on variance in median estimates (P50)
		if validCount > 1 {
			var sumSquaredDiff float64
			for _, resp := range responses {
				if resp.Status != "completed" || resp.PercentilePredictions == nil {
					continue
				}
				diff := resp.PercentilePredictions.P50 - weightedP50
				sumSquaredDiff += diff * diff
			}
			stdDev := math.Sqrt(sumSquaredDiff / float64(validCount))
			consensus = &stdDev
		}

		return models.ForecastResult{
			AggregatedPercentiles: &models.PercentilePredictions{
				P10: weightedP10,
				P25: weightedP25,
				P50: weightedP50,
				P75: weightedP75,
				P90: weightedP90,
			},
			ModelCount:     validCount,
			ConsensusLevel: consensus,
		}
	} else {
		// Calculate weighted average of point estimates
		var weightedEstimate float64

		for _, resp := range responses {
			if resp.Status != "completed" || resp.PointEstimate == nil {
				continue
			}

			weight := weights[resp.ModelID]
			weightedEstimate += *resp.PointEstimate * weight
			validCount++
		}

		if totalWeight > 0 {
			weightedEstimate /= totalWeight
		}

		// Calculate consensus based on variance in point estimates
		if validCount > 1 {
			var sumSquaredDiff float64
			for _, resp := range responses {
				if resp.Status != "completed" || resp.PointEstimate == nil {
					continue
				}
				diff := *resp.PointEstimate - weightedEstimate
				sumSquaredDiff += diff * diff
			}
			stdDev := math.Sqrt(sumSquaredDiff / float64(validCount))
			consensus = &stdDev
		}

		return models.ForecastResult{
			AggregatedPointEstimate: &weightedEstimate,
			ModelCount:              validCount,
			ConsensusLevel:          consensus,
		}
	}
}

// fetchURLContent fetches content from a URL and returns it as a string
func (f *Forecaster) fetchURLContent(ctx context.Context, url string) (string, error) {
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set user agent to avoid bot blocking
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Read response body with size limit (1MB max)
	bodyBytes := make([]byte, 1024*1024)
	n, err := resp.Body.Read(bodyBytes)
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	return string(bodyBytes[:n]), nil
}
