package enrichment

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/STRATINT/stratint/internal/database"
	"github.com/STRATINT/stratint/internal/inference"
	"github.com/STRATINT/stratint/internal/models"
	openai "github.com/sashabaranov/go-openai"
)

// Enricher processes raw OSINT sources into structured events with AI-powered analysis.
type Enricher interface {
	// Enrich processes a source and returns an enriched event.
	Enrich(ctx context.Context, source models.Source) (*models.Event, error)

	// EnrichBatch processes multiple sources concurrently.
	EnrichBatch(ctx context.Context, sources []models.Source) ([]models.Event, error)

	// ExtractArticleText extracts article content from HTML (for scraping)
	ExtractArticleText(ctx context.Context, html, url string) (string, error)
}

// OpenAIClient wraps the OpenAI API for OSINT enrichment.
type OpenAIClient struct {
	client          *openai.Client
	config          OpenAIConfig
	prompts         *PromptTemplates
	extractor       *EntityExtractor
	scorer          *ConfidenceScorer
	estimator       *MagnitudeEstimator
	correlator      *EventCorrelator
	configRepo      *database.OpenAIConfigRepository
	logger          *slog.Logger
	inferenceLogger *inference.Logger
}

// OpenAIConfig holds configuration for OpenAI API usage.
type OpenAIConfig struct {
	APIKey      string
	Model       string
	Temperature float32
	MaxTokens   int
	Timeout     int // seconds
}

// DefaultOpenAIConfig returns sensible defaults for OSINT processing.
func DefaultOpenAIConfig() OpenAIConfig {
	return OpenAIConfig{
		Model:       openai.GPT4TurboPreview,
		Temperature: 0.3, // Lower temperature for factual analysis
		MaxTokens:   2000,
		Timeout:     180, // 180 seconds for o1 models (can take 60-180s for extended reasoning)
	}
}

// ConfigFromEnv creates config from environment variables with sensible defaults.
func ConfigFromEnv() OpenAIConfig {
	config := DefaultOpenAIConfig()

	// Override model from environment if set
	if model := os.Getenv("OPENAI_MODEL"); model != "" {
		config.Model = model
	}

	// Override temperature from environment if set
	if tempStr := os.Getenv("OPENAI_TEMPERATURE"); tempStr != "" {
		if temp, err := strconv.ParseFloat(tempStr, 32); err == nil {
			config.Temperature = float32(temp)
		}
	}

	return config
}

// NewOpenAIClient creates a new OpenAI-powered enricher.
// Deprecated: Use NewOpenAIClientFromDB for database-backed configuration.
func NewOpenAIClient(apiKey string, config OpenAIConfig) *OpenAIClient {
	config.APIKey = apiKey

	client := openai.NewClient(apiKey)

	prompts := NewPromptTemplates()

	return &OpenAIClient{
		client:     client,
		config:     config,
		prompts:    prompts,
		extractor:  NewEntityExtractor(),
		scorer:     NewConfidenceScorer(),
		estimator:  NewMagnitudeEstimator(),
		correlator: NewEventCorrelator(client, config, prompts, slog.Default()),
		configRepo: nil,
		logger:     slog.Default(),
	}
}

// NewOpenAIClientFromDB creates a new OpenAI-powered enricher using database configuration.
func NewOpenAIClientFromDB(ctx context.Context, configRepo *database.OpenAIConfigRepository, logger *slog.Logger, inferenceLogger *inference.Logger) (*OpenAIClient, error) {
	// Load configuration from database
	dbConfig, err := configRepo.Get(ctx)
	if err != nil {
		// Check if it's a "not found" error
		if err.Error() == "openai configuration not found" {
			return nil, fmt.Errorf("openai configuration not found in database - please configure in admin panel")
		}
		return nil, fmt.Errorf("failed to load openai config from database: %w", err)
	}

	// Check if OpenAI is enabled
	if !dbConfig.Enabled {
		return nil, fmt.Errorf("openai enrichment is disabled in configuration")
	}

	// Validate API key
	if dbConfig.APIKey == "" {
		return nil, fmt.Errorf("openai api key not configured - please set in admin panel")
	}

	// Create OpenAI client
	client := openai.NewClient(dbConfig.APIKey)

	// Convert database config to internal config
	config := OpenAIConfig{
		APIKey:      dbConfig.APIKey,
		Model:       dbConfig.Model,
		Temperature: dbConfig.Temperature,
		MaxTokens:   dbConfig.MaxTokens,
		Timeout:     dbConfig.TimeoutSeconds,
	}

	// Create prompts from database configuration
	prompts := &PromptTemplates{
		SystemPrompt:            dbConfig.SystemPrompt,
		AnalysisTemplate:        dbConfig.AnalysisTemplate,
		EntityExtractionPrompt:  dbConfig.EntityExtractionPrompt,
		CorrelationSystemPrompt: dbConfig.CorrelationSystemPrompt,
	}

	logger.Info("initialized openai enricher from database config",
		"model", config.Model,
		"temperature", config.Temperature,
		"enabled", dbConfig.Enabled)

	return &OpenAIClient{
		client:          client,
		config:          config,
		prompts:         prompts,
		extractor:       NewEntityExtractor(),
		scorer:          NewConfidenceScorer(),
		estimator:       NewMagnitudeEstimator(),
		correlator:      NewEventCorrelator(client, config, prompts, logger),
		configRepo:      configRepo,
		logger:          logger,
		inferenceLogger: inferenceLogger,
	}, nil
}

// GetCorrelator returns the event correlator for this client.
func (c *OpenAIClient) GetCorrelator() *EventCorrelator {
	return c.correlator
}

// GetScorer returns the confidence scorer for this client.
func (c *OpenAIClient) GetScorer() *ConfidenceScorer {
	return c.scorer
}

// Enrich processes a single source into an enriched event.
func (c *OpenAIClient) Enrich(ctx context.Context, source models.Source) (*models.Event, error) {
	enrichStart := time.Now()
	c.logger.Info("[ENRICH START]",
		"source_id", source.ID,
		"url", source.URL)

	// Skip enrichment if source has insufficient content
	// Lowered threshold for RSS descriptions (was 500)
	if len(source.RawContent) < 50 {
		return nil, fmt.Errorf("insufficient content for enrichment: only %d chars (minimum 50 required)", len(source.RawContent))
	}

	// Generate prompt for analysis
	promptStart := time.Now()
	prompt := c.prompts.BuildAnalysisPrompt(source)
	c.logger.Debug("[PROMPT BUILT]",
		"source_id", source.ID,
		"duration_ms", time.Since(promptStart).Milliseconds())

	// Create a timeout context for the API call
	// GPT-5/o1 models require much longer timeouts (60-180s) due to extended reasoning
	// GPT-4 models typically respond in 5-15s
	timeout := 180 // Default to 180 seconds for o1 models
	if c.config.Timeout > 0 {
		timeout = c.config.Timeout
	}

	// Retry logic for rate limiting
	maxRetries := 3
	baseDelay := 1 * time.Second

	var resp openai.ChatCompletionResponse
	var err error

	for attempt := 0; attempt < maxRetries; attempt++ {
		apiCallStart := time.Now()
		c.logger.Info("[OPENAI API CALL START]",
			"source_id", source.ID,
			"attempt", attempt+1,
			"timeout_sec", timeout)

		apiCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)

		// Detect o1/o4/gpt-5 reasoning models which have different API requirements (no JSON mode, merged system prompt)
		isO1Model := strings.Contains(strings.ToLower(c.config.Model), "o1") ||
			strings.Contains(strings.ToLower(c.config.Model), "o4") ||
			strings.Contains(strings.ToLower(c.config.Model), "gpt-5")

		var request openai.ChatCompletionRequest

		if isO1Model {
			// o1 models don't support: response_format, system messages (must merge into user)
			// Combine system prompt and user prompt into a single user message
			combinedPrompt := c.prompts.SystemPrompt + "\n\n" + prompt

			request = openai.ChatCompletionRequest{
				Model:               c.config.Model,
				MaxCompletionTokens: c.config.MaxTokens,
				Messages: []openai.ChatCompletionMessage{
					{
						Role:    openai.ChatMessageRoleUser,
						Content: combinedPrompt,
					},
				},
			}

			c.logger.Debug("[O1 MODEL DETECTED]", "model", c.config.Model, "no_json_mode", true)
		} else {
			// Standard models (gpt-4, gpt-4o, gpt-4o-mini) support JSON mode and system messages
			request = openai.ChatCompletionRequest{
				Model:               c.config.Model,
				MaxCompletionTokens: c.config.MaxTokens,
				ResponseFormat: &openai.ChatCompletionResponseFormat{
					Type: openai.ChatCompletionResponseFormatTypeJSONObject,
				},
				Messages: []openai.ChatCompletionMessage{
					{
						Role:    openai.ChatMessageRoleSystem,
						Content: c.prompts.SystemPrompt,
					},
					{
						Role:    openai.ChatMessageRoleUser,
						Content: prompt,
					},
				},
			}
		}

		resp, err = c.client.CreateChatCompletion(apiCtx, request)

		cancel()

		apiCallDuration := time.Since(apiCallStart)
		c.logger.Info("[OPENAI API CALL COMPLETE]",
			"source_id", source.ID,
			"attempt", attempt+1,
			"duration_ms", apiCallDuration.Milliseconds(),
			"success", err == nil)

		// Log inference call if logger is available
		if c.inferenceLogger != nil {
			usage := struct {
				PromptTokens     int
				CompletionTokens int
				TotalTokens      int
			}{}
			metadata := map[string]interface{}{
				"source_id": source.ID,
				"attempt":   attempt + 1,
			}

			if err == nil {
				usage.PromptTokens = resp.Usage.PromptTokens
				usage.CompletionTokens = resp.Usage.CompletionTokens
				usage.TotalTokens = resp.Usage.TotalTokens
			} else {
				// Add rate limit information to metadata if present in error
				errStr := err.Error()
				if strings.Contains(errStr, "429") || strings.Contains(errStr, "Rate limit") {
					metadata["is_rate_limit"] = true
					metadata["rate_limit_error"] = errStr
				}
			}

			c.inferenceLogger.LogOpenAICall(ctx, c.config.Model, "event_creation", usage, apiCallDuration, err, metadata)
		}

		// If successful, break out of retry loop
		if err == nil {
			break
		}

		// Check if it's a rate limit error (429)
		if err != nil && err.Error() != "" {
			errStr := err.Error()
			if strings.Contains(errStr, "429") || strings.Contains(errStr, "Too Many Requests") || strings.Contains(errStr, "Rate limit") {
				// Log detailed rate limit information
				c.logger.Warn("OpenAI rate limit hit",
					"source_id", source.ID,
					"attempt", attempt+1,
					"error", errStr)

				// Try to parse rate limit reset time from error message
				// OpenAI errors often include messages like "Rate limit reached. Try again in 2h3m4s"
				if strings.Contains(errStr, "Try again in") {
					c.logger.Warn("Rate limit reset info found in error",
						"error", errStr)
				}

				// Calculate exponential backoff with jitter
				delay := baseDelay * time.Duration(1<<uint(attempt))
				// Add jitter (0-500ms)
				jitter := time.Duration(rand.Intn(500)) * time.Millisecond
				delay += jitter

				if attempt < maxRetries-1 {
					c.logger.Warn("rate limited, retrying with backoff",
						"source_id", source.ID,
						"attempt", attempt+1,
						"delay_ms", delay.Milliseconds(),
						"max_retries", maxRetries)
					time.Sleep(delay)
					continue
				} else {
					c.logger.Error("rate limit exceeded, max retries reached",
						"source_id", source.ID,
						"attempts", maxRetries,
						"error", errStr)
				}
			}
		}

		// For non-rate-limit errors or final attempt, return the error
		break
	}

	if err != nil {
		return nil, fmt.Errorf("openai api call failed for source %s: %w", source.ID, err)
	}

	if len(resp.Choices) == 0 {
		c.logger.Error("[OPENAI NO CHOICES]",
			"source_id", source.ID,
			"model", c.config.Model,
			"response_id", resp.ID)
		return nil, fmt.Errorf("no completion choices returned from model %s", c.config.Model)
	}

	analysis := resp.Choices[0].Message.Content

	// Log if content is empty
	if analysis == "" {
		c.logger.Error("[OPENAI EMPTY RESPONSE]",
			"source_id", source.ID,
			"model", c.config.Model,
			"finish_reason", resp.Choices[0].FinishReason,
			"response_id", resp.ID)
		return nil, fmt.Errorf("empty response from model %s (finish_reason: %s)", c.config.Model, resp.Choices[0].FinishReason)
	}

	// Parse analysis into structured event
	parseStart := time.Now()
	event, err := c.parseAnalysis(source, analysis)
	c.logger.Debug("[PARSE ANALYSIS]",
		"source_id", source.ID,
		"duration_ms", time.Since(parseStart).Milliseconds())
	if err != nil {
		return nil, fmt.Errorf("failed to parse analysis: %w", err)
	}

	// Extract entities using the configured entity extraction prompt
	entityStart := time.Now()
	c.logger.Info("[ENTITY EXTRACTION START]", "source_id", source.ID)
	entityPrompt := c.prompts.BuildEntityExtractionPrompt(source.RawContent)
	entities, err := c.extractor.Extract(ctx, source.RawContent, c.client, c.config, entityPrompt)
	c.logger.Info("[ENTITY EXTRACTION COMPLETE]",
		"source_id", source.ID,
		"duration_ms", time.Since(entityStart).Milliseconds(),
		"entity_count", len(entities))
	if err != nil {
		// Non-fatal: log warning and continue with empty entities
		if c.logger != nil {
			c.logger.Warn("entity extraction failed, continuing without entities", "error", err, "source_id", source.ID)
		}
		entities = []models.Entity{}
	}
	event.Entities = entities

	// If location wasn't populated by AI, try to extract from entities
	if event.Location == nil {
		event.Location = extractLocationFromEntities(entities)
	}

	// Calculate confidence score
	scoreStart := time.Now()
	confidence := c.scorer.Score(source, event, entities)
	event.Confidence = confidence
	c.logger.Debug("[CONFIDENCE SCORE]",
		"source_id", source.ID,
		"duration_ms", time.Since(scoreStart).Milliseconds())

	// Magnitude is now determined by OpenAI in the analysis phase
	c.logger.Debug("[MAGNITUDE]",
		"source_id", source.ID,
		"magnitude", event.Magnitude,
		"source", "openai")

	// Set metadata
	event.Sources = []models.Source{source}
	event.Status = models.EventStatusEnriched

	totalDuration := time.Since(enrichStart)
	c.logger.Info("[ENRICH COMPLETE]",
		"source_id", source.ID,
		"total_duration_ms", totalDuration.Milliseconds())

	return event, nil
}

// ExtractArticleText uses OpenAI to extract article content from raw HTML
func (c *OpenAIClient) ExtractArticleText(ctx context.Context, html, url string) (string, error) {
	// Truncate HTML if too long to stay under token limits
	// Aiming for ~3k tokens max (12k chars at ~4 chars/token)
	maxHTMLLength := 15000
	originalLength := len(html)
	if len(html) > maxHTMLLength {
		c.logger.Warn("truncating HTML for extraction",
			"url", url,
			"original_length", originalLength,
			"truncated_length", maxHTMLLength)
		html = html[:maxHTMLLength]
	}

	prompt := fmt.Sprintf(`Extract the main article content from this HTML page. Return only the clean article text without any HTML tags, navigation menus, advertisements, or other non-article content.

URL: %s

Return the extracted article text in plain text format. If the page is blocked, paywalled, or contains no article content, return "ERROR: No article content found".`, url)

	// Create timeout context
	apiCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Call OpenAI API
	startTime := time.Now()
	resp, err := c.client.CreateChatCompletion(apiCtx, openai.ChatCompletionRequest{
		Model:               c.config.Model,
		MaxCompletionTokens: 4000, // Allow longer responses for article content
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: "You are an expert at extracting article content from HTML. Return only the clean article text without any formatting or explanations.",
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: prompt + "\n\nHTML:\n" + html,
			},
		},
	})
	latency := time.Since(startTime)

	// Log inference call
	if c.inferenceLogger != nil {
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
		c.inferenceLogger.LogOpenAICall(ctx, c.config.Model, "article_extraction", usage, latency, err, map[string]interface{}{
			"url": url,
		})
	}

	if err != nil {
		return "", fmt.Errorf("OpenAI API call failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from OpenAI")
	}

	articleText := strings.TrimSpace(resp.Choices[0].Message.Content)

	// Check if extraction failed
	if strings.HasPrefix(articleText, "ERROR:") || len(articleText) < 100 {
		return "", fmt.Errorf("failed to extract article content: %s", articleText)
	}

	c.logger.Info("extracted article with OpenAI", "url", url, "length", len(articleText))

	return articleText, nil
}

// EnrichBatch processes multiple sources concurrently using a worker pool.
func (c *OpenAIClient) EnrichBatch(ctx context.Context, sources []models.Source) ([]models.Event, error) {
	if len(sources) == 0 {
		return []models.Event{}, nil
	}

	// Use worker pool for concurrent processing - balanced for rate limits
	// Each enrichment = 2 API calls (analysis + entities)
	// 200k TPM limit ~= 10 concurrent enrichments max before rate limiting
	const maxWorkers = 10
	workerCount := maxWorkers
	if len(sources) < workerCount {
		workerCount = len(sources)
	}

	batchStart := time.Now()
	c.logger.Info("[BATCH ENRICH START]",
		"total_sources", len(sources),
		"workers", workerCount)

	// Create channels for work distribution
	type job struct {
		index  int
		source models.Source
	}
	type result struct {
		index int
		event *models.Event
		err   error
	}

	jobChan := make(chan job, len(sources))
	resultChan := make(chan result, len(sources))

	// Start workers
	for w := 0; w < workerCount; w++ {
		go func(workerID int) {
			c.logger.Info("[WORKER START]", "worker_id", workerID)
			jobCount := 0
			for job := range jobChan {
				jobCount++
				jobStart := time.Now()
				c.logger.Info("[WORKER JOB START]",
					"worker_id", workerID,
					"job_num", jobCount,
					"source_id", job.source.ID)

				event, err := c.Enrich(ctx, job.source)

				c.logger.Info("[WORKER JOB COMPLETE]",
					"worker_id", workerID,
					"job_num", jobCount,
					"source_id", job.source.ID,
					"duration_ms", time.Since(jobStart).Milliseconds(),
					"success", err == nil)

				resultChan <- result{
					index: job.index,
					event: event,
					err:   err,
				}
			}
			c.logger.Info("[WORKER DONE]", "worker_id", workerID, "jobs_processed", jobCount)
		}(w)
	}

	// Send jobs
	for i, source := range sources {
		jobChan <- job{index: i, source: source}
	}
	close(jobChan)

	// Collect results (preserve order by index)
	results := make([]result, len(sources))
	for i := 0; i < len(sources); i++ {
		res := <-resultChan
		results[res.index] = res
	}
	close(resultChan)

	// Process results in order
	events := make([]models.Event, 0, len(sources))
	errors := make([]error, 0)

	for i, res := range results {
		if res.err != nil {
			c.logger.Error("enrichment failed",
				"source_id", sources[i].ID,
				"error", res.err)
			errors = append(errors, fmt.Errorf("source %s: %w", sources[i].ID, res.err))
			continue
		}
		if res.event != nil {
			events = append(events, *res.event)
		}
	}

	batchDuration := time.Since(batchStart)
	c.logger.Info("[BATCH ENRICH COMPLETE]",
		"total_sources", len(sources),
		"events_created", len(events),
		"errors", len(errors),
		"total_duration_ms", batchDuration.Milliseconds(),
		"avg_per_source_ms", batchDuration.Milliseconds()/int64(len(sources)))

	if len(errors) > 0 {
		return events, fmt.Errorf("batch enrichment had %d errors (first: %w)", len(errors), errors[0])
	}

	return events, nil
}

// parseAnalysis converts OpenAI's text response into a structured event.
func (c *OpenAIClient) parseAnalysis(source models.Source, analysis string) (*models.Event, error) {
	// Parse the structured analysis response
	parsed, err := ParseStructuredAnalysis(analysis)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	event := &models.Event{
		ID:         generateEventID(source),
		Timestamp:  source.PublishedAt,
		Title:      parsed.Title,
		Summary:    "", // No longer generating summaries from RSS descriptions
		RawContent: source.RawContent,
		Category:   parsed.Category,
		Magnitude:  parsed.Magnitude,
		Tags:       parsed.Tags,
		Location:   parsed.Location,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	return event, nil
}

// generateEventID creates a deterministic event identifier based on source.
// This ensures that enriching the same source multiple times produces the same event ID,
// preventing duplicate events from race conditions in the enrichment pipeline.
func generateEventID(source models.Source) string {
	// Use source content hash if available (most reliable)
	if source.ContentHash != "" {
		return fmt.Sprintf("evt-%s", source.ContentHash)
	}
	// Fallback to source ID (also deterministic)
	return fmt.Sprintf("evt-%s", source.ID)
}

func generateTimestamp() int64 {
	return time.Now().UnixNano()
}

// extractLocationFromEntities attempts to build a Location from extracted country/city entities.
func extractLocationFromEntities(entities []models.Entity) *models.Location {
	var country, city string

	// Look for the first country and city entities
	for _, entity := range entities {
		if entity.Type == models.EntityTypeCountry && country == "" {
			// Prefer normalized name if available
			if entity.NormalizedName != "" {
				country = entity.NormalizedName
			} else {
				country = entity.Name
			}
		}
		if entity.Type == models.EntityTypeCity && city == "" {
			// Prefer normalized name if available
			if entity.NormalizedName != "" {
				city = entity.NormalizedName
			} else {
				city = entity.Name
			}
		}

		// Stop early if we have both
		if country != "" && city != "" {
			break
		}
	}

	// Only create location if we found at least a country
	if country == "" {
		return nil
	}

	return &models.Location{
		Country:   country,
		City:      city,
		Latitude:  0.0,
		Longitude: 0.0,
	}
}

// AssessSourceCredibility uses LLM to evaluate the credibility of a source based on its domain/URL.
// Returns a score between 0.0 (not credible) and 1.0 (highly credible).
func (c *OpenAIClient) AssessSourceCredibility(ctx context.Context, url string, sourceType models.SourceType) (float64, error) {
	prompt := fmt.Sprintf(`Assess the credibility of this source for OSINT analysis.

URL: %s
Source Type: %s

Consider:
- Domain reputation and authority
- Known track record for accuracy
- Editorial standards
- Bias/reliability ratings
- Historical trustworthiness

Respond with ONLY a decimal number between 0.0 (not credible) and 1.0 (highly credible).
Examples:
- Reuters, AP News: 0.95
- CNN, BBC: 0.85
- Local news sites: 0.70
- Personal blogs: 0.40
- Twitter/social media: 0.60
- Unknown/suspicious sites: 0.20

Score:`, url, sourceType)

	startTime := time.Now()
	resp, err := c.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: c.config.Model,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: "You are an OSINT analyst expert at assessing source credibility. Respond only with a decimal number.",
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: prompt,
			},
		},
		MaxCompletionTokens: 50,
	})
	latency := time.Since(startTime)

	// Log inference call
	if c.inferenceLogger != nil {
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
		c.inferenceLogger.LogOpenAICall(ctx, c.config.Model, "source_credibility", usage, latency, err, map[string]interface{}{
			"url":         url,
			"source_type": string(sourceType),
		})
	}

	if err != nil {
		c.logger.Error("failed to assess source credibility",
			"url", url,
			"error", err)
		// Return default score based on source type on error
		return c.getDefaultCredibility(sourceType), nil
	}

	if len(resp.Choices) == 0 {
		c.logger.Debug("no response from LLM for credibility assessment, using default", "url", url)
		return c.getDefaultCredibility(sourceType), nil
	}

	scoreStr := strings.TrimSpace(resp.Choices[0].Message.Content)
	if scoreStr == "" {
		c.logger.Debug("empty response from LLM for credibility assessment, using default", "url", url)
		return c.getDefaultCredibility(sourceType), nil
	}

	score, err := strconv.ParseFloat(scoreStr, 64)
	if err != nil {
		c.logger.Debug("failed to parse credibility score, using default",
			"url", url,
			"response", scoreStr)
		return c.getDefaultCredibility(sourceType), nil
	}

	// Clamp to valid range
	if score < 0.0 {
		score = 0.0
	}
	if score > 1.0 {
		score = 1.0
	}

	c.logger.Debug("assessed source credibility",
		"url", url,
		"score", score)

	return score, nil
}

// getDefaultCredibility returns a fallback credibility score based on source type.
func (c *OpenAIClient) getDefaultCredibility(sourceType models.SourceType) float64 {
	defaults := map[models.SourceType]float64{
		models.SourceTypeGovernment: 0.95,
		models.SourceTypeNewsMedia:  0.85,
		models.SourceTypeTwitter:    0.60,
		models.SourceTypeTelegram:   0.55,
		models.SourceTypeBlog:       0.45,
		models.SourceTypeGLP:        0.25,
		models.SourceTypeOther:      0.40,
	}

	if score, ok := defaults[sourceType]; ok {
		return score
	}
	return 0.40
}

// GenerateText generates text using OpenAI with a simple system/user prompt
// This is useful for generating tweets, summaries, or other text based on templates
func (c *OpenAIClient) GenerateText(ctx context.Context, systemPrompt, userPrompt string, temperature float32, maxTokens int) (string, error) {
	// Create timeout context
	timeout := 180 // Default to 180 seconds for o1 models
	if c.config.Timeout > 0 {
		timeout = c.config.Timeout
	}

	apiCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	// Detect reasoning models (o1, o3, o4, gpt-5) which have API restrictions
	isReasoningModel := strings.Contains(strings.ToLower(c.config.Model), "o1") ||
		strings.Contains(strings.ToLower(c.config.Model), "o3") ||
		strings.Contains(strings.ToLower(c.config.Model), "o4") ||
		strings.Contains(strings.ToLower(c.config.Model), "gpt-5")

	var request openai.ChatCompletionRequest

	if isReasoningModel {
		// Reasoning models don't support temperature, top_p, system messages
		// Merge system prompt into user message
		combinedPrompt := systemPrompt + "\n\n" + userPrompt

		request = openai.ChatCompletionRequest{
			Model: c.config.Model,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: combinedPrompt,
				},
			},
		}

		// Only set MaxCompletionTokens if specified (0 means use OpenAI defaults)
		if maxTokens > 0 {
			request.MaxCompletionTokens = maxTokens
		}
	} else {
		// Standard models support all parameters
		request = openai.ChatCompletionRequest{
			Model:       c.config.Model,
			Temperature: temperature,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: systemPrompt,
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: userPrompt,
				},
			},
		}

		// Only set MaxCompletionTokens if specified (0 means use OpenAI defaults)
		if maxTokens > 0 {
			request.MaxCompletionTokens = maxTokens
		}
	}

	// Call OpenAI API
	startTime := time.Now()
	resp, err := c.client.CreateChatCompletion(apiCtx, request)
	latency := time.Since(startTime)

	// Log inference call
	if c.inferenceLogger != nil {
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
		c.inferenceLogger.LogOpenAICall(ctx, c.config.Model, "text_generation", usage, latency, err, map[string]interface{}{
			"temperature": temperature,
			"max_tokens":  maxTokens,
		})
	}

	if err != nil {
		return "", fmt.Errorf("openai api call failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from openai")
	}

	content := resp.Choices[0].Message.Content

	// For reasoning models, check if there's reasoning content
	if isReasoningModel && content == "" {
		c.logger.Warn("reasoning model returned empty content, checking for reasoning field",
			"model", c.config.Model,
			"choices_count", len(resp.Choices),
			"finish_reason", resp.Choices[0].FinishReason)
	}

	c.logger.Info("openai generate text response",
		"model", c.config.Model,
		"content_length", len(content),
		"finish_reason", resp.Choices[0].FinishReason,
		"is_reasoning_model", isReasoningModel)

	return content, nil
}
