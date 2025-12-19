package models

import (
	"time"
)

// Forecast represents a value-based forecast configuration
type Forecast struct {
	ID               string     `json:"id"`
	Name             string     `json:"name"`
	Proposition      string     `json:"proposition"`           // e.g., "What will be the % change of the S&P 500 1 year from today?"
	PredictionType   string     `json:"prediction_type"`       // "percentile" (full distribution) or "point_estimate" (single value)
	Units            string     `json:"units"`                 // e.g., "percent_change", "dollars", "points"
	TargetDate       *time.Time `json:"target_date,omitempty"` // When the prediction is for
	Categories       []string   `json:"categories"`            // Categories to include in analysis
	HeadlineCount    int        `json:"headline_count"`        // Number of headlines to use
	Iterations       int        `json:"iterations"`            // Number of times to query each model
	ContextURLs      []string   `json:"context_urls"`          // URLs to fetch and inject before headlines
	Active           bool       `json:"active"`
	Public           bool       `json:"public"`                // Whether the forecast is publicly visible on homepage
	DisplayOrder     int        `json:"display_order"`         // Sort order for homepage display (higher = earlier)
	ScheduleEnabled  bool       `json:"schedule_enabled"`      // Whether automatic scheduling is enabled
	ScheduleInterval int        `json:"schedule_interval"`     // Interval in minutes (e.g., 60 for hourly, 1440 for daily)
	LastRunAt        *time.Time `json:"last_run_at,omitempty"` // When the forecast was last executed
	NextRunAt        *time.Time `json:"next_run_at,omitempty"` // When the forecast should run next
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// ForecastModel represents a model configuration for a forecast
type ForecastModel struct {
	ID         string    `json:"id"`
	ForecastID string    `json:"forecast_id"`
	Provider   string    `json:"provider"`   // 'anthropic' or 'openai'
	ModelName  string    `json:"model_name"` // e.g., 'claude-sonnet-4.5', 'gpt-4'
	APIKey     string    `json:"api_key"`    // Should be encrypted in DB
	Weight     float64   `json:"weight"`     // Weight for averaging
	Active     bool      `json:"active"`
	CreatedAt  time.Time `json:"created_at"`
}

// ForecastRun represents a single execution of a forecast
type ForecastRun struct {
	ID                string             `json:"id"`
	ForecastID        string             `json:"forecast_id"`
	RunAt             time.Time          `json:"run_at"`
	HeadlineCount     int                `json:"headline_count"`
	HeadlinesSnapshot []ForecastHeadline `json:"headlines_snapshot"`
	Status            string             `json:"status"` // 'pending', 'running', 'completed', 'failed'
	ErrorMessage      string             `json:"error_message,omitempty"`
	CompletedAt       *time.Time         `json:"completed_at,omitempty"`
}

// ForecastHeadline represents a headline used in a forecast
type ForecastHeadline struct {
	EventID   string    `json:"event_id"`
	Title     string    `json:"title"`
	Category  string    `json:"category"`
	Magnitude float64   `json:"magnitude"`
	Timestamp time.Time `json:"timestamp"`
}

// PercentilePredictions represents a distribution via percentiles
type PercentilePredictions struct {
	P10 float64 `json:"p10"` // 10th percentile
	P25 float64 `json:"p25"` // 25th percentile (Q1)
	P50 float64 `json:"p50"` // 50th percentile (median)
	P75 float64 `json:"p75"` // 75th percentile (Q3)
	P90 float64 `json:"p90"` // 90th percentile
}

// ForecastModelResponse represents a response from a single model
type ForecastModelResponse struct {
	ID                    string                 `json:"id"`
	RunID                 string                 `json:"run_id"`
	ModelID               string                 `json:"model_id"`
	Provider              string                 `json:"provider"`
	ModelName             string                 `json:"model_name"`
	PercentilePredictions *PercentilePredictions `json:"percentile_predictions,omitempty"` // For distribution forecasts
	PointEstimate         *float64               `json:"point_estimate,omitempty"`         // For single-value forecasts
	Reasoning             string                 `json:"reasoning,omitempty"`
	RawResponse           map[string]interface{} `json:"raw_response,omitempty"`
	TokensUsed            *int                   `json:"tokens_used,omitempty"`
	ResponseTimeMs        *int                   `json:"response_time_ms,omitempty"`
	Status                string                 `json:"status"` // 'pending', 'completed', 'failed'
	ErrorMessage          string                 `json:"error_message,omitempty"`
	CreatedAt             time.Time              `json:"created_at"`
}

// ForecastResult represents the aggregated result of a forecast run
type ForecastResult struct {
	ID                      string                 `json:"id"`
	RunID                   string                 `json:"run_id"`
	AggregatedPercentiles   *PercentilePredictions `json:"aggregated_percentiles,omitempty"`    // Weighted avg of model percentiles
	AggregatedPointEstimate *float64               `json:"aggregated_point_estimate,omitempty"` // Weighted avg of point estimates
	ModelCount              int                    `json:"model_count"`
	ConsensusLevel          *float64               `json:"consensus_level,omitempty"` // Standard deviation across models
	CreatedAt               time.Time              `json:"created_at"`
}

// ForecastRunDetail combines run info with responses and result
type ForecastRunDetail struct {
	Run       ForecastRun             `json:"run"`
	Responses []ForecastModelResponse `json:"responses"`
	Result    *ForecastResult         `json:"result,omitempty"`
}

// CreateForecastRequest represents the request to create a new value-based forecast
type CreateForecastRequest struct {
	Name           string          `json:"name"`
	Proposition    string          `json:"proposition"`     // e.g., "What will be the % change of the S&P 500 1 year from today?"
	PredictionType string          `json:"prediction_type"` // "percentile" or "point_estimate"
	Units          string          `json:"units"`           // e.g., "percent_change", "dollars"
	TargetDate     *time.Time      `json:"target_date,omitempty"`
	Categories     []string        `json:"categories"`
	HeadlineCount  int             `json:"headline_count"`
	Iterations     int             `json:"iterations"`
	ContextURLs    []string        `json:"context_urls"`
	Models         []ForecastModel `json:"models"`
}

// ExecuteForecastRequest represents the request to run a forecast
type ExecuteForecastRequest struct {
	ForecastID string `json:"forecast_id"`
}
