package models

import "time"

// InferenceLog represents a single LLM API call
type InferenceLog struct {
	ID           int       `json:"id"`
	Provider     string    `json:"provider"`      // 'openai', 'anthropic', etc.
	Model        string    `json:"model"`         // 'gpt-4o', 'claude-sonnet-4', etc.
	Operation    string    `json:"operation"`     // 'event_creation', 'twitter_post', 'forecast', 'strategy', etc.
	TokensUsed   int       `json:"tokens_used"`   // Total tokens
	InputTokens  *int      `json:"input_tokens"`  // Input tokens if available
	OutputTokens *int      `json:"output_tokens"` // Output tokens if available
	CostUSD      *float64  `json:"cost_usd"`      // Estimated cost in USD
	LatencyMs    *int      `json:"latency_ms"`    // Response time in milliseconds
	Status       string    `json:"status"`        // 'success', 'error'
	ErrorMessage *string   `json:"error_message"` // Error details if failed
	Metadata     string    `json:"metadata"`      // JSONB metadata
	CreatedAt    time.Time `json:"created_at"`
}

// InferenceLogStats represents aggregated statistics
type InferenceLogStats struct {
	TotalCalls      int     `json:"total_calls"`
	TotalTokens     int64   `json:"total_tokens"`
	TotalCostUSD    float64 `json:"total_cost_usd"`
	SuccessfulCalls int     `json:"successful_calls"`
	FailedCalls     int     `json:"failed_calls"`
	AvgLatencyMs    float64 `json:"avg_latency_ms"`
}

// InferenceLogQuery represents query parameters for filtering logs
type InferenceLogQuery struct {
	Provider  string
	Model     string
	Operation string
	Status    string
	StartDate *time.Time
	EndDate   *time.Time
	Limit     int
	Offset    int
}
