package models

import "time"

// OpenAIConfig represents the configuration for OpenAI API integration.
type OpenAIConfig struct {
	ID                      int       `json:"id"`
	APIKey                  string    `json:"api_key"`
	Model                   string    `json:"model"`
	Temperature             float32   `json:"temperature"`
	MaxTokens               int       `json:"max_tokens"`
	TimeoutSeconds          int       `json:"timeout_seconds"`
	SystemPrompt            string    `json:"system_prompt"`
	AnalysisTemplate        string    `json:"analysis_template"`
	EntityExtractionPrompt  string    `json:"entity_extraction_prompt"`
	CorrelationSystemPrompt string    `json:"correlation_system_prompt"`
	Enabled                 bool      `json:"enabled"`
	UpdatedAt               time.Time `json:"updated_at"`
	CreatedAt               time.Time `json:"created_at"`
}

// OpenAIConfigUpdate represents fields that can be updated.
type OpenAIConfigUpdate struct {
	APIKey                  *string  `json:"api_key,omitempty"`
	Model                   *string  `json:"model,omitempty"`
	Temperature             *float32 `json:"temperature,omitempty"`
	MaxTokens               *int     `json:"max_tokens,omitempty"`
	TimeoutSeconds          *int     `json:"timeout_seconds,omitempty"`
	SystemPrompt            *string  `json:"system_prompt,omitempty"`
	AnalysisTemplate        *string  `json:"analysis_template,omitempty"`
	EntityExtractionPrompt  *string  `json:"entity_extraction_prompt,omitempty"`
	CorrelationSystemPrompt *string  `json:"correlation_system_prompt,omitempty"`
	Enabled                 *bool    `json:"enabled,omitempty"`
}
