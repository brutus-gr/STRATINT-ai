package models

import (
	"time"
)

// Strategy represents an AI-generated portfolio allocation strategy
type Strategy struct {
	ID                   string          `json:"id"`
	Name                 string          `json:"name"`
	Prompt               string          `json:"prompt"`             // User-defined directive
	InvestmentSymbols    []string        `json:"investment_symbols"` // Available investments (e.g., ["SPY", "VNQ", "TLT", "GLD", "CASH"])
	Categories           []string        `json:"categories"`         // Signal categories to include
	HeadlineCount        int             `json:"headline_count"`
	Iterations           int             `json:"iterations"`             // Number of times to run before averaging
	ForecastIDs          []string        `json:"forecast_ids"`           // Forecast IDs to inject
	ForecastHistoryCount int             `json:"forecast_history_count"` // Number of past forecast runs to include (default: 1)
	Models               []StrategyModel `json:"models,omitempty"`       // Associated models (populated when fetching single strategy)
	Active               bool            `json:"active"`
	Public               bool            `json:"public"`            // Whether visible on homepage
	DisplayOrder         int             `json:"display_order"`     // Sort order for homepage
	ScheduleEnabled      bool            `json:"schedule_enabled"`  // Whether automatic scheduling is enabled
	ScheduleInterval     int             `json:"schedule_interval"` // Interval in minutes
	LastRunAt            *time.Time      `json:"last_run_at,omitempty"`
	NextRunAt            *time.Time      `json:"next_run_at,omitempty"`
	CreatedAt            time.Time       `json:"created_at"`
	UpdatedAt            time.Time       `json:"updated_at"`
}

// StrategyModel represents a model configuration for a strategy
type StrategyModel struct {
	ID         string    `json:"id"`
	StrategyID string    `json:"strategy_id"`
	Provider   string    `json:"provider"`   // 'anthropic' or 'openai'
	ModelName  string    `json:"model_name"` // e.g., 'claude-sonnet-4.5', 'gpt-4'
	APIKey     string    `json:"api_key"`    // Should be encrypted in DB
	Weight     float64   `json:"weight"`     // Weight for averaging
	Active     bool      `json:"active"`
	CreatedAt  time.Time `json:"created_at"`
}

// StrategyRun represents a single execution of a strategy
type StrategyRun struct {
	ID                string             `json:"id"`
	StrategyID        string             `json:"strategy_id"`
	RunAt             time.Time          `json:"run_at"`
	HeadlineCount     int                `json:"headline_count"`
	HeadlinesSnapshot []StrategyHeadline `json:"headlines_snapshot"`
	ForecastSnapshots []ForecastSnapshot `json:"forecast_snapshots"`
	Status            string             `json:"status"` // 'pending', 'running', 'completed', 'failed'
	ErrorMessage      string             `json:"error_message,omitempty"`
	CompletedAt       *time.Time         `json:"completed_at,omitempty"`
}

// StrategyHeadline represents a headline used in a strategy run
type StrategyHeadline struct {
	EventID   string    `json:"event_id"`
	Title     string    `json:"title"`
	Category  string    `json:"category"`
	Magnitude float64   `json:"magnitude"`
	Timestamp time.Time `json:"timestamp"`
}

// ForecastSnapshot represents forecast data injected into a strategy
type ForecastSnapshot struct {
	ForecastID   string                 `json:"forecast_id"`
	ForecastName string                 `json:"forecast_name"`
	Symbol       string                 `json:"symbol,omitempty"` // e.g., "SPY" if tracking a ticker
	Percentiles  *PercentilePredictions `json:"percentiles"`
	RunAt        time.Time              `json:"run_at"`
}

// StrategyModelResponse represents a response from a single model in one iteration
type StrategyModelResponse struct {
	ID             string                 `json:"id"`
	RunID          string                 `json:"run_id"`
	ModelID        string                 `json:"model_id"`
	Iteration      int                    `json:"iteration"` // Which iteration (1 to N)
	Provider       string                 `json:"provider"`
	ModelName      string                 `json:"model_name"`
	Allocations    map[string]float64     `json:"allocations"`         // {"SPY": 40.0, "VNQ": 25.0, ...}
	Reasoning      string                 `json:"reasoning,omitempty"` // Model's explanation
	RawResponse    map[string]interface{} `json:"raw_response,omitempty"`
	TokensUsed     *int                   `json:"tokens_used,omitempty"`
	ResponseTimeMs *int                   `json:"response_time_ms,omitempty"`
	Status         string                 `json:"status"` // 'pending', 'completed', 'failed'
	ErrorMessage   string                 `json:"error_message,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
}

// StrategyResult represents the aggregated result of a strategy run
type StrategyResult struct {
	ID                     string             `json:"id"`
	RunID                  string             `json:"run_id"`
	AveragedAllocations    map[string]float64 `json:"averaged_allocations"`    // Simple average across iterations
	NormalizedAllocations  map[string]float64 `json:"normalized_allocations"`  // AI-normalized to sum to 100%
	NormalizationReasoning string             `json:"normalization_reasoning"` // AI's explanation
	ModelCount             int                `json:"model_count"`
	IterationCount         int                `json:"iteration_count"`
	ConsensusVariance      map[string]float64 `json:"consensus_variance"` // Std dev per symbol
	CreatedAt              time.Time          `json:"created_at"`
}

// StrategyRunDetail combines run info with responses and result
type StrategyRunDetail struct {
	Run       StrategyRun             `json:"run"`
	Responses []StrategyModelResponse `json:"responses"`
	Result    *StrategyResult         `json:"result,omitempty"`
}

// CreateStrategyRequest represents the request to create a new strategy
type CreateStrategyRequest struct {
	Name                 string          `json:"name"`
	Prompt               string          `json:"prompt"`
	InvestmentSymbols    []string        `json:"investment_symbols"`
	Categories           []string        `json:"categories"`
	HeadlineCount        int             `json:"headline_count"`
	Iterations           int             `json:"iterations"`
	ForecastIDs          []string        `json:"forecast_ids"`
	ForecastHistoryCount int             `json:"forecast_history_count"`
	Models               []StrategyModel `json:"models"`
}

// ExecuteStrategyRequest represents the request to run a strategy
type ExecuteStrategyRequest struct {
	StrategyID string `json:"strategy_id"`
}
