package inference

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/STRATINT/stratint/internal/database"
	"github.com/STRATINT/stratint/internal/models"
)

// Logger logs inference calls to the database
type Logger struct {
	repo   *database.InferenceLogRepository
	logger *slog.Logger
}

// NewLogger creates a new inference logger
func NewLogger(repo *database.InferenceLogRepository, logger *slog.Logger) *Logger {
	return &Logger{
		repo:   repo,
		logger: logger,
	}
}

// LogCall logs an inference API call
type LogCallParams struct {
	Provider     string
	Model        string
	Operation    string
	TokensUsed   int
	InputTokens  *int
	OutputTokens *int
	CostUSD      *float64
	LatencyMs    *int
	Status       string // "success" or "error"
	ErrorMessage *string
	Metadata     map[string]interface{} // Additional context
}

// LogCall logs an inference call to the database
func (l *Logger) LogCall(ctx context.Context, params LogCallParams) {
	// Marshal metadata to JSON string
	var metadataJSON string
	if params.Metadata != nil {
		if jsonBytes, err := json.Marshal(params.Metadata); err == nil {
			metadataJSON = string(jsonBytes)
		}
	}

	log := models.InferenceLog{
		Provider:     params.Provider,
		Model:        params.Model,
		Operation:    params.Operation,
		TokensUsed:   params.TokensUsed,
		InputTokens:  params.InputTokens,
		OutputTokens: params.OutputTokens,
		CostUSD:      params.CostUSD,
		LatencyMs:    params.LatencyMs,
		Status:       params.Status,
		ErrorMessage: params.ErrorMessage,
		Metadata:     metadataJSON,
	}

	// Log asynchronously to avoid blocking the main operation
	go func() {
		bgCtx := context.Background()
		if err := l.repo.Create(bgCtx, log); err != nil {
			l.logger.Error("failed to log inference call", "error", err)
		}
	}()
}

// LogOpenAICall is a helper for OpenAI API calls
func (l *Logger) LogOpenAICall(ctx context.Context, model, operation string, usage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}, latency time.Duration, err error, metadata map[string]interface{}) {
	params := LogCallParams{
		Provider:     "openai",
		Model:        model,
		Operation:    operation,
		TokensUsed:   usage.TotalTokens,
		InputTokens:  &usage.PromptTokens,
		OutputTokens: &usage.CompletionTokens,
		Metadata:     metadata,
	}

	latencyMs := int(latency.Milliseconds())
	params.LatencyMs = &latencyMs

	if err != nil {
		params.Status = "error"
		errMsg := err.Error()
		params.ErrorMessage = &errMsg
	} else {
		params.Status = "success"
	}

	// Estimate cost (rough estimates - update with actual pricing)
	cost := estimateOpenAICost(model, usage.PromptTokens, usage.CompletionTokens)
	params.CostUSD = &cost

	l.LogCall(ctx, params)
}

// LogAnthropicCall is a helper for Anthropic API calls
func (l *Logger) LogAnthropicCall(ctx context.Context, model, operation string, usage struct {
	InputTokens  int
	OutputTokens int
}, latency time.Duration, err error, metadata map[string]interface{}) {
	totalTokens := usage.InputTokens + usage.OutputTokens
	params := LogCallParams{
		Provider:     "anthropic",
		Model:        model,
		Operation:    operation,
		TokensUsed:   totalTokens,
		InputTokens:  &usage.InputTokens,
		OutputTokens: &usage.OutputTokens,
		Metadata:     metadata,
	}

	latencyMs := int(latency.Milliseconds())
	params.LatencyMs = &latencyMs

	if err != nil {
		params.Status = "error"
		errMsg := err.Error()
		params.ErrorMessage = &errMsg
	} else {
		params.Status = "success"
	}

	// Estimate cost (rough estimates - update with actual pricing)
	cost := estimateAnthropicCost(model, usage.InputTokens, usage.OutputTokens)
	params.CostUSD = &cost

	l.LogCall(ctx, params)
}

// estimateOpenAICost provides rough cost estimates (update with actual pricing)
func estimateOpenAICost(model string, inputTokens, outputTokens int) float64 {
	// Rough estimates per 1M tokens (as of late 2024)
	var inputCostPer1M, outputCostPer1M float64

	switch model {
	case "gpt-4o":
		inputCostPer1M = 2.50
		outputCostPer1M = 10.00
	case "gpt-4o-mini":
		inputCostPer1M = 0.15
		outputCostPer1M = 0.60
	case "gpt-4-turbo", "gpt-4-turbo-preview":
		inputCostPer1M = 10.00
		outputCostPer1M = 30.00
	case "gpt-3.5-turbo":
		inputCostPer1M = 0.50
		outputCostPer1M = 1.50
	default:
		inputCostPer1M = 5.00
		outputCostPer1M = 15.00
	}

	inputCost := (float64(inputTokens) / 1_000_000) * inputCostPer1M
	outputCost := (float64(outputTokens) / 1_000_000) * outputCostPer1M

	return inputCost + outputCost
}

// estimateAnthropicCost provides rough cost estimates (update with actual pricing)
func estimateAnthropicCost(model string, inputTokens, outputTokens int) float64 {
	// Rough estimates per 1M tokens (as of late 2024)
	var inputCostPer1M, outputCostPer1M float64

	switch model {
	case "claude-sonnet-4-20250514", "claude-sonnet-4-20241022":
		inputCostPer1M = 3.00
		outputCostPer1M = 15.00
	case "claude-3-5-sonnet-20240620":
		inputCostPer1M = 3.00
		outputCostPer1M = 15.00
	case "claude-3-opus-20240229":
		inputCostPer1M = 15.00
		outputCostPer1M = 75.00
	case "claude-3-haiku-20240307":
		inputCostPer1M = 0.25
		outputCostPer1M = 1.25
	default:
		inputCostPer1M = 3.00
		outputCostPer1M = 15.00
	}

	inputCost := (float64(inputTokens) / 1_000_000) * inputCostPer1M
	outputCost := (float64(outputTokens) / 1_000_000) * outputCostPer1M

	return inputCost + outputCost
}
