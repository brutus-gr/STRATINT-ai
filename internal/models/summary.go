package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

type Summary struct {
	ID                string         `json:"id"`
	Name              string         `json:"name"`
	Prompt            string         `json:"prompt"`
	TimeOfDay         *string        `json:"time_of_day,omitempty"` // Format: "09:00"
	LookbackHours     int            `json:"lookback_hours"`
	Categories        []string       `json:"categories"`
	HeadlineCount     int            `json:"headline_count"`
	Models            []SummaryModel `json:"models"`
	Active            bool           `json:"active"`
	ScheduleEnabled   bool           `json:"schedule_enabled"`
	ScheduleInterval  int            `json:"schedule_interval"` // in minutes
	AutoPostToTwitter bool           `json:"auto_post_to_twitter"`
	IncludeForecasts  bool           `json:"include_forecasts"`
	LastRunAt         *time.Time     `json:"last_run_at,omitempty"`
	NextRunAt         *time.Time     `json:"next_run_at,omitempty"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
}

type SummaryModel struct {
	Provider  string  `json:"provider"`
	ModelName string  `json:"model_name"`
	APIKey    string  `json:"api_key"`
	Weight    float64 `json:"weight"`
}

type SummaryRun struct {
	ID            string     `json:"id"`
	SummaryID     string     `json:"summary_id"`
	RunAt         time.Time  `json:"run_at"`
	HeadlineCount int        `json:"headline_count"`
	LookbackStart time.Time  `json:"lookback_start"`
	LookbackEnd   time.Time  `json:"lookback_end"`
	Status        string     `json:"status"` // pending, running, completed, failed
	ErrorMessage  *string    `json:"error_message,omitempty"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
}

type SummaryResult struct {
	ID            string    `json:"id"`
	RunID         string    `json:"run_id"`
	SummaryText   string    `json:"summary_text"`
	ModelProvider string    `json:"model_provider"`
	ModelName     string    `json:"model_name"`
	CreatedAt     time.Time `json:"created_at"`
}

type SummaryRunDetail struct {
	Run     SummaryRun      `json:"run"`
	Results []SummaryResult `json:"results"`
}

// Custom JSON marshaling for SummaryModel array
type SummaryModels []SummaryModel

func (sm SummaryModels) Value() (driver.Value, error) {
	return json.Marshal(sm)
}

func (sm *SummaryModels) Scan(value interface{}) error {
	if value == nil {
		*sm = []SummaryModel{}
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}

	return json.Unmarshal(bytes, sm)
}
