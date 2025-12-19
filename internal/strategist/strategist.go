package strategist

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"time"

	"github.com/STRATINT/stratint/internal/inference"
	"github.com/STRATINT/stratint/internal/models"
	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/sashabaranov/go-openai"
)

const (
	samplingTemperature = 1.0
	systemPrompt        = "You are a financial analyst AI assistant. Analyze the provided information and respond with valid JSON according to the requested format."
)

// EventRepository defines methods needed to fetch events for strategies
type EventRepository interface {
	Query(ctx context.Context, query models.EventQuery) (*models.EventResponse, error)
}

// StrategyRepository defines methods needed for strategy storage
type StrategyRepository interface {
	GetStrategy(ctx context.Context, id string) (*models.Strategy, error)
	GetStrategyModels(ctx context.Context, strategyID string) ([]models.StrategyModel, error)
	CreateStrategyRun(ctx context.Context, strategyID string, headlines []models.StrategyHeadline, forecastSnapshots []models.ForecastSnapshot) (string, error)
	UpdateStrategyRunStatus(ctx context.Context, runID, status, errorMsg string) error
	CreateModelResponse(ctx context.Context, response models.StrategyModelResponse) error
	CreateStrategyResult(ctx context.Context, result models.StrategyResult) error
	GetStrategyRun(ctx context.Context, runID string) (*models.StrategyRunDetail, error)
	UpdateStrategyLastRun(ctx context.Context, id string, lastRunAt time.Time) error
}

// ForecastRepository defines methods needed to fetch forecast data
type ForecastRepository interface {
	GetForecast(ctx context.Context, id string) (*models.Forecast, error)
	GetForecastRun(ctx context.Context, runID string) (*models.ForecastRunDetail, error)
	GetLatestCompletedForecastRun(ctx context.Context, forecastID string) (*models.ForecastRunDetail, error)
	GetLatestNCompletedForecastRuns(ctx context.Context, forecastID string, n int) ([]models.ForecastRunDetail, error)
}

// Strategist executes strategies using multiple AI models
type Strategist struct {
	eventRepo       EventRepository
	strategyRepo    StrategyRepository
	forecastRepo    ForecastRepository
	logger          *slog.Logger
	inferenceLogger *inference.Logger
}

// NewStrategist creates a new strategist
func NewStrategist(eventRepo EventRepository, strategyRepo StrategyRepository, forecastRepo ForecastRepository, logger *slog.Logger, inferenceLogger *inference.Logger) *Strategist {
	return &Strategist{
		eventRepo:       eventRepo,
		strategyRepo:    strategyRepo,
		forecastRepo:    forecastRepo,
		logger:          logger,
		inferenceLogger: inferenceLogger,
	}
}

// ExecuteStrategy executes a strategy and returns the run ID
func (s *Strategist) ExecuteStrategy(ctx context.Context, strategyID string) (string, error) {
	s.logger.Info("starting strategy execution", "strategy_id", strategyID)

	// Get strategy configuration
	strategy, err := s.strategyRepo.GetStrategy(ctx, strategyID)
	if err != nil {
		return "", fmt.Errorf("failed to get strategy: %w", err)
	}

	// Get strategy models
	models, err := s.strategyRepo.GetStrategyModels(ctx, strategyID)
	if err != nil {
		return "", fmt.Errorf("failed to get strategy models: %w", err)
	}
	if len(models) == 0 {
		return "", fmt.Errorf("no models configured for strategy: %s", strategyID)
	}

	// Fetch recent headlines
	headlines, err := s.fetchHeadlines(ctx, strategy)
	if err != nil {
		return "", fmt.Errorf("failed to fetch headlines: %w", err)
	}

	s.logger.Info("fetched headlines for strategy",
		"strategy_id", strategyID,
		"headline_count", len(headlines))

	// Fetch forecast data if forecast IDs are specified
	historyCount := strategy.ForecastHistoryCount
	if historyCount < 1 {
		historyCount = 1 // Default to 1 if not set
	}
	forecastSnapshots, err := s.fetchForecastData(ctx, strategy.ForecastIDs, historyCount)
	if err != nil {
		return "", fmt.Errorf("failed to fetch forecast data: %w", err)
	}

	s.logger.Info("fetched forecast data for strategy",
		"strategy_id", strategyID,
		"forecast_count", len(forecastSnapshots))

	// Create strategy run
	runID, err := s.strategyRepo.CreateStrategyRun(ctx, strategyID, headlines, forecastSnapshots)
	if err != nil {
		return "", fmt.Errorf("failed to create strategy run: %w", err)
	}

	// Update status to running
	if err := s.strategyRepo.UpdateStrategyRunStatus(ctx, runID, "running", ""); err != nil {
		return "", fmt.Errorf("failed to update run status: %w", err)
	}

	// Execute strategy asynchronously
	go s.executeStrategyAsync(context.Background(), runID, strategy, models, headlines, forecastSnapshots)

	return runID, nil
}

func (s *Strategist) executeStrategyAsync(ctx context.Context, runID string, strategy *models.Strategy, strategyModels []models.StrategyModel, headlines []models.StrategyHeadline, forecastSnapshots []models.ForecastSnapshot) {
	defer func() {
		if r := recover(); r != nil {
			s.logger.Error("panic in strategy execution", "run_id", runID, "panic", r)
			s.strategyRepo.UpdateStrategyRunStatus(ctx, runID, "failed", fmt.Sprintf("panic: %v", r))
		}
	}()

	// Run multiple iterations across all models
	var allResponses []models.StrategyModelResponse

	for _, model := range strategyModels {
		for iteration := 1; iteration <= strategy.Iterations; iteration++ {
			s.logger.Info("executing iteration",
				"run_id", runID,
				"provider", model.Provider,
				"model", model.ModelName,
				"iteration", iteration,
				"total_iterations", strategy.Iterations)

			startTime := time.Now()
			response, err := s.executeIteration(ctx, strategy, &model, iteration, headlines, forecastSnapshots)
			responseTime := int(time.Since(startTime).Milliseconds())

			if err != nil {
				s.logger.Error("iteration failed",
					"run_id", runID,
					"provider", model.Provider,
					"model", model.ModelName,
					"iteration", iteration,
					"error", err)

				// Store failed response
				failedResp := models.StrategyModelResponse{
					RunID:          runID,
					ModelID:        model.ID,
					Iteration:      iteration,
					Provider:       model.Provider,
					ModelName:      model.ModelName,
					Status:         "failed",
					ErrorMessage:   err.Error(),
					ResponseTimeMs: &responseTime,
				}
				s.strategyRepo.CreateModelResponse(ctx, failedResp)
				continue
			}

			// Update response with metadata
			response.RunID = runID
			response.ResponseTimeMs = &responseTime

			allResponses = append(allResponses, *response)

			// Store response
			if err := s.strategyRepo.CreateModelResponse(ctx, *response); err != nil {
				s.logger.Error("failed to store model response", "error", err)
			}
		}
	}

	if len(allResponses) == 0 {
		s.strategyRepo.UpdateStrategyRunStatus(ctx, runID, "failed", "all iterations failed")
		return
	}

	// Calculate averaged allocations
	averaged := s.averageAllocations(allResponses, strategy.InvestmentSymbols)

	// Calculate variance
	variance := s.calculateVariance(allResponses, strategy.InvestmentSymbols)

	// Perform normalization pass with AI
	normalized, reasoning, err := s.normalizeAllocations(ctx, averaged, &strategyModels[0], strategy.InvestmentSymbols)
	if err != nil {
		s.logger.Error("normalization failed", "error", err)
		normalized = averaged // Fallback to averaged if normalization fails
		reasoning = "Normalization pass failed, using raw averages"
	}

	// Create result
	result := models.StrategyResult{
		RunID:                  runID,
		AveragedAllocations:    averaged,
		NormalizedAllocations:  normalized,
		NormalizationReasoning: reasoning,
		ModelCount:             len(strategyModels),
		IterationCount:         strategy.Iterations,
		ConsensusVariance:      variance,
	}

	// Store result
	if err := s.strategyRepo.CreateStrategyResult(ctx, result); err != nil {
		s.logger.Error("failed to store strategy result", "error", err)
		s.strategyRepo.UpdateStrategyRunStatus(ctx, runID, "failed", fmt.Sprintf("failed to store result: %v", err))
		return
	}

	// Update strategy last run time
	s.strategyRepo.UpdateStrategyLastRun(ctx, strategy.ID, time.Now())

	// Mark run as completed
	s.strategyRepo.UpdateStrategyRunStatus(ctx, runID, "completed", "")

	s.logger.Info("strategy execution completed",
		"run_id", runID,
		"model_count", result.ModelCount,
		"iteration_count", result.IterationCount)
}

func (s *Strategist) fetchHeadlines(ctx context.Context, strategy *models.Strategy) ([]models.StrategyHeadline, error) {
	// Build query
	query := models.EventQuery{
		Limit:     strategy.HeadlineCount,
		Page:      1,
		SortBy:    "timestamp",
		SortOrder: "desc",
	}

	// Filter by categories if specified
	if len(strategy.Categories) > 0 {
		categories := make([]models.Category, len(strategy.Categories))
		for i, cat := range strategy.Categories {
			categories[i] = models.Category(cat)
		}
		query.Categories = categories
	}

	// Query events
	resp, err := s.eventRepo.Query(ctx, query)
	if err != nil {
		return nil, err
	}

	s.logger.Info("fetched headlines from database",
		"requested", strategy.HeadlineCount,
		"received", len(resp.Events),
		"categories", strategy.Categories)

	// Convert to headlines
	headlines := make([]models.StrategyHeadline, 0, len(resp.Events))
	for _, event := range resp.Events {
		headlines = append(headlines, models.StrategyHeadline{
			EventID:   event.ID,
			Title:     event.Title,
			Category:  string(event.Category),
			Magnitude: event.Magnitude,
			Timestamp: event.Timestamp,
		})
	}

	return headlines, nil
}

func (s *Strategist) fetchForecastData(ctx context.Context, forecastIDs []string, historyCount int) ([]models.ForecastSnapshot, error) {
	if len(forecastIDs) == 0 {
		return []models.ForecastSnapshot{}, nil
	}

	if historyCount < 1 {
		historyCount = 1
	}

	snapshots := make([]models.ForecastSnapshot, 0, len(forecastIDs)*historyCount)

	for _, forecastID := range forecastIDs {
		// Get forecast details
		forecast, err := s.forecastRepo.GetForecast(ctx, forecastID)
		if err != nil {
			s.logger.Warn("failed to get forecast", "forecast_id", forecastID, "error", err)
			continue
		}

		// Get latest N completed runs with results
		runDetails, err := s.forecastRepo.GetLatestNCompletedForecastRuns(ctx, forecastID, historyCount)
		if err != nil {
			s.logger.Warn("failed to get latest forecast runs", "forecast_id", forecastID, "history_count", historyCount, "error", err)
			continue
		}
		if len(runDetails) == 0 {
			s.logger.Warn("forecast has no completed runs with results", "forecast_id", forecastID)
			continue
		}

		s.logger.Info("fetched forecast runs",
			"forecast_id", forecastID,
			"name", forecast.Name,
			"requested_count", historyCount,
			"retrieved_count", len(runDetails))

		// Create a snapshot for each run (most recent first)
		for _, runDetail := range runDetails {
			if runDetail.Result == nil {
				s.logger.Warn("forecast run missing result", "run_id", runDetail.Run.ID)
				continue
			}

			snapshot := models.ForecastSnapshot{
				ForecastID:   forecast.ID,
				ForecastName: forecast.Name,
				Percentiles:  runDetail.Result.AggregatedPercentiles,
				RunAt:        runDetail.Run.RunAt,
			}

			s.logger.Debug("forecast snapshot created with percentiles",
				"forecast_id", forecastID,
				"name", forecast.Name,
				"run_at", runDetail.Run.RunAt,
				"p50", runDetail.Result.AggregatedPercentiles.P50)

			snapshots = append(snapshots, snapshot)
		}
	}

	s.logger.Info("fetched forecast data for strategy",
		"total_snapshots", len(snapshots),
		"requested_forecasts", len(forecastIDs),
		"history_per_forecast", historyCount)

	return snapshots, nil
}

func (s *Strategist) executeIteration(ctx context.Context, strategy *models.Strategy, model *models.StrategyModel, iteration int, headlines []models.StrategyHeadline, forecastSnapshots []models.ForecastSnapshot) (*models.StrategyModelResponse, error) {
	// Build prompt
	prompt := s.buildPrompt(strategy, headlines, forecastSnapshots)

	s.logger.Info("built prompt",
		"model", model.ModelName,
		"headline_count", len(headlines),
		"forecast_count", len(forecastSnapshots),
		"iteration", iteration,
		"prompt_length", len(prompt))

	// Call AI model
	content, tokens, err := s.callModel(ctx, model, prompt)
	if err != nil {
		return nil, err
	}

	// Parse allocations from response
	allocations, reasoning, err := s.parseAllocations(content, strategy.InvestmentSymbols)
	if err != nil {
		return nil, fmt.Errorf("failed to parse allocations: %w", err)
	}

	return &models.StrategyModelResponse{
		ModelID:     model.ID,
		Iteration:   iteration,
		Provider:    model.Provider,
		ModelName:   model.ModelName,
		Allocations: allocations,
		Reasoning:   reasoning,
		TokensUsed:  &tokens,
		Status:      "completed",
	}, nil
}

func (s *Strategist) buildPrompt(strategy *models.Strategy, headlines []models.StrategyHeadline, forecastSnapshots []models.ForecastSnapshot) string {
	var sb strings.Builder

	sb.WriteString("You are a portfolio allocation strategist. Based on the intelligence signals and forecast data below, ")
	sb.WriteString("provide a recommended portfolio allocation across these investments:\n\n")

	sb.WriteString("AVAILABLE INVESTMENTS: ")
	sb.WriteString(strings.Join(strategy.InvestmentSymbols, ", "))
	sb.WriteString("\n\n")

	sb.WriteString("USER DIRECTIVE:\n")
	sb.WriteString(strategy.Prompt)
	sb.WriteString("\n\n")

	// Add forecast data if available
	if len(forecastSnapshots) > 0 {
		sb.WriteString("=== FORECAST DATA (Historical Runs) ===\n\n")

		// Group snapshots by forecast name
		forecastGroups := make(map[string][]models.ForecastSnapshot)
		for _, snapshot := range forecastSnapshots {
			forecastGroups[snapshot.ForecastName] = append(forecastGroups[snapshot.ForecastName], snapshot)
		}

		// Display each forecast with its history
		for forecastName, snapshots := range forecastGroups {
			sb.WriteString(fmt.Sprintf("%s (showing %d most recent runs):\n", forecastName, len(snapshots)))

			// Snapshots are already in reverse chronological order (newest first)
			for i, snapshot := range snapshots {
				timeLabel := "Latest"
				if i > 0 {
					timeLabel = fmt.Sprintf("-%dh", i)
				}

				if snapshot.Percentiles != nil {
					sb.WriteString(fmt.Sprintf("  [%s] P10: %.2f%%  P25: %.2f%%  P50: %.2f%%  P75: %.2f%%  P90: %.2f%%  (Run: %s)\n",
						timeLabel,
						snapshot.Percentiles.P10, snapshot.Percentiles.P25, snapshot.Percentiles.P50,
						snapshot.Percentiles.P75, snapshot.Percentiles.P90,
						snapshot.RunAt.Format("Jan 2 15:04")))

					s.logger.Debug("added forecast to prompt",
						"forecast_name", snapshot.ForecastName,
						"run_at", snapshot.RunAt,
						"p50", snapshot.Percentiles.P50)
				} else {
					s.logger.Warn("forecast missing percentiles in prompt",
						"forecast_name", snapshot.ForecastName,
						"forecast_id", snapshot.ForecastID)
				}
			}
			sb.WriteString("\n")
		}
	}

	// Add headlines
	sb.WriteString("=== RECENT INTELLIGENCE SIGNALS ===\n\n")
	for i, headline := range headlines {
		if i >= 50 {
			sb.WriteString(fmt.Sprintf("... [%d more headlines]\n", len(headlines)-50))
			break
		}
		sb.WriteString(fmt.Sprintf("[%s] %s (Magnitude: %.1f)\n", headline.Category, headline.Title, headline.Magnitude))
	}

	sb.WriteString("\n=== RESPONSE FORMAT ===\n\n")
	sb.WriteString("Provide your allocation as a JSON object with percentages that sum to approximately 100%:\n\n")
	sb.WriteString("{\n")
	for i, symbol := range strategy.InvestmentSymbols {
		if i == 0 {
			sb.WriteString(fmt.Sprintf("  \"%s\": 40.0", symbol))
		} else {
			sb.WriteString(fmt.Sprintf(",\n  \"%s\": %.1f", symbol, 100.0/float64(len(strategy.InvestmentSymbols))))
		}
	}
	sb.WriteString("\n}\n\n")
	sb.WriteString("After the JSON, provide a brief reasoning (2-3 sentences) explaining your allocation strategy.\n")

	return sb.String()
}

func (s *Strategist) callModel(ctx context.Context, model *models.StrategyModel, prompt string) (string, int, error) {

	switch model.Provider {
	case "openai":
		return s.callOpenAI(ctx, model, systemPrompt, prompt)
	case "anthropic":
		return s.callAnthropic(ctx, model, systemPrompt, prompt)
	default:
		return "", 0, fmt.Errorf("unsupported provider: %s", model.Provider)
	}
}

func (s *Strategist) callOpenAI(ctx context.Context, model *models.StrategyModel, systemPrompt, userPrompt string) (string, int, error) {
	client := openai.NewClient(model.APIKey)

	startTime := time.Now()
	resp, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:       model.ModelName,
		Temperature: float32(samplingTemperature),
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
			{Role: openai.ChatMessageRoleUser, Content: userPrompt},
		},
	})
	latency := time.Since(startTime)

	// Log inference call
	if s.inferenceLogger != nil {
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
		s.inferenceLogger.LogOpenAICall(ctx, model.ModelName, "strategy_execution", usage, latency, err, map[string]interface{}{
			"model_id": model.ID,
		})
	}

	if err != nil {
		return "", 0, fmt.Errorf("openai api error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", 0, fmt.Errorf("no response from openai")
	}

	return resp.Choices[0].Message.Content, resp.Usage.TotalTokens, nil
}

func (s *Strategist) callAnthropic(ctx context.Context, model *models.StrategyModel, systemPrompt, userPrompt string) (string, int, error) {
	client := anthropic.NewClient(option.WithAPIKey(model.APIKey))

	req := anthropic.MessageNewParams{
		Model:       anthropic.Model(model.ModelName),
		MaxTokens:   4096,
		Temperature: anthropic.Float(samplingTemperature),
		System: []anthropic.TextBlockParam{
			{Text: systemPrompt},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(userPrompt)),
		},
	}

	startTime := time.Now()
	message, err := client.Messages.New(ctx, req)
	latency := time.Since(startTime)

	// Log inference call
	if s.inferenceLogger != nil {
		usage := struct {
			InputTokens  int
			OutputTokens int
		}{}
		if err == nil {
			usage.InputTokens = int(message.Usage.InputTokens)
			usage.OutputTokens = int(message.Usage.OutputTokens)
		}
		s.inferenceLogger.LogAnthropicCall(ctx, model.ModelName, "strategy_execution", usage, latency, err, map[string]interface{}{
			"model_id": model.ID,
		})
	}

	if err != nil {
		return "", 0, fmt.Errorf("anthropic api error: %w", err)
	}

	if len(message.Content) == 0 {
		return "", 0, fmt.Errorf("no response from anthropic")
	}

	textBlock := message.Content[0].Text
	tokens := int(message.Usage.InputTokens + message.Usage.OutputTokens)

	return textBlock, tokens, nil
}

// extractJSON finds and extracts the first valid JSON object from text using brace matching
func (s *Strategist) extractJSON(text string) string {
	// Find the first opening brace
	startIdx := strings.Index(text, "{")
	if startIdx == -1 {
		return ""
	}

	// Use brace counting to find matching closing brace
	braceCount := 0
	inString := false
	escaped := false

	for i := startIdx; i < len(text); i++ {
		ch := text[i]

		// Handle escape sequences in strings
		if escaped {
			escaped = false
			continue
		}

		if ch == '\\' {
			escaped = true
			continue
		}

		// Track if we're inside a string
		if ch == '"' {
			inString = !inString
			continue
		}

		// Only count braces outside of strings
		if !inString {
			if ch == '{' {
				braceCount++
			} else if ch == '}' {
				braceCount--
				if braceCount == 0 {
					// Found matching closing brace
					return text[startIdx : i+1]
				}
			}
		}
	}

	return ""
}

func (s *Strategist) parseAllocations(responseText string, symbols []string) (map[string]float64, string, error) {
	allocations := make(map[string]float64)

	// Try to extract JSON object using brace matching to handle nested objects and multiline JSON
	jsonMatch := s.extractJSON(responseText)

	if jsonMatch == "" {
		s.logger.Error("no JSON found in response", "response", responseText)
		return nil, "", fmt.Errorf("no valid JSON allocation found in response")
	}

	// Parse JSON
	var rawAllocations map[string]interface{}
	if err := json.Unmarshal([]byte(jsonMatch), &rawAllocations); err != nil {
		s.logger.Error("failed to parse JSON",
			"error", err,
			"json_match", jsonMatch,
			"full_response", responseText)
		return nil, "", fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Verify this JSON contains our symbols
	containsSymbols := true
	for _, symbol := range symbols {
		if _, ok := rawAllocations[symbol]; !ok {
			containsSymbols = false
			break
		}
	}

	if !containsSymbols {
		s.logger.Error("JSON found but missing expected symbols",
			"expected_symbols", symbols,
			"found_keys", rawAllocations)
		return nil, "", fmt.Errorf("JSON does not contain all expected symbols")
	}

	// Extract allocations for each symbol
	for _, symbol := range symbols {
		if val, ok := rawAllocations[symbol]; ok {
			switch v := val.(type) {
			case float64:
				allocations[symbol] = v
			case int:
				allocations[symbol] = float64(v)
			default:
				return nil, "", fmt.Errorf("invalid allocation value for %s", symbol)
			}
		} else {
			return nil, "", fmt.Errorf("missing allocation for symbol: %s", symbol)
		}
	}

	// Extract reasoning (text after JSON)
	jsonIndex := strings.Index(responseText, jsonMatch)
	reasoning := strings.TrimSpace(responseText[jsonIndex+len(jsonMatch):])
	if len(reasoning) > 500 {
		reasoning = reasoning[:500] + "..."
	}

	return allocations, reasoning, nil
}

func (s *Strategist) averageAllocations(responses []models.StrategyModelResponse, symbols []string) map[string]float64 {
	averaged := make(map[string]float64)

	for _, symbol := range symbols {
		sum := 0.0
		count := 0

		for _, resp := range responses {
			if val, ok := resp.Allocations[symbol]; ok {
				sum += val
				count++
			}
		}

		if count > 0 {
			averaged[symbol] = sum / float64(count)
		} else {
			averaged[symbol] = 0.0
		}
	}

	return averaged
}

func (s *Strategist) calculateVariance(responses []models.StrategyModelResponse, symbols []string) map[string]float64 {
	variance := make(map[string]float64)

	// First calculate means
	means := s.averageAllocations(responses, symbols)

	for _, symbol := range symbols {
		sumSquaredDiff := 0.0
		count := 0

		for _, resp := range responses {
			if val, ok := resp.Allocations[symbol]; ok {
				diff := val - means[symbol]
				sumSquaredDiff += diff * diff
				count++
			}
		}

		if count > 1 {
			variance[symbol] = math.Sqrt(sumSquaredDiff / float64(count-1))
		} else {
			variance[symbol] = 0.0
		}
	}

	return variance
}

func (s *Strategist) normalizeAllocations(ctx context.Context, averaged map[string]float64, model *models.StrategyModel, symbols []string) (map[string]float64, string, error) {
	// Calculate current total
	total := 0.0
	for _, val := range averaged {
		total += val
	}

	// Build normalization prompt
	allocationsJSON, _ := json.MarshalIndent(averaged, "", "  ")

	prompt := fmt.Sprintf(`You are a portfolio allocation strategist performing a final review.

We ran multiple AI models with several iterations each, and calculated the average allocation:

AVERAGED ALLOCATIONS:
%s

Total: %.2f%%

Please adjust these allocations to sum to exactly 100.0%% while:
1. Maintaining the relative strategic intent
2. Making minimal adjustments
3. Ensuring all percentages are realistic (no negative values)

IMPORTANT: Respond with ONLY a flat JSON object where keys are ticker symbols and values are percentages. Do NOT nest the allocations inside another object. Use this exact format:

{
  %s
}

After the JSON object, on a new line, provide a brief explanation (1-2 sentences) of what adjustments you made.`, string(allocationsJSON), total, s.buildExampleJSON(symbols))

	// Call AI for normalization
	content, _, err := s.callModel(ctx, model, prompt)
	if err != nil {
		return nil, "", err
	}

	// Parse normalized allocations
	normalized, reasoning, err := s.parseAllocations(content, symbols)
	if err != nil {
		return nil, "", err
	}

	return normalized, reasoning, nil
}

func (s *Strategist) buildExampleJSON(symbols []string) string {
	parts := make([]string, len(symbols))
	for i, symbol := range symbols {
		parts[i] = fmt.Sprintf("\"%s\": %.1f", symbol, 100.0/float64(len(symbols)))
	}
	return strings.Join(parts, ",\n  ")
}
