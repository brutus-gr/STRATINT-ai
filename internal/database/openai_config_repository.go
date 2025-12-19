package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/STRATINT/stratint/internal/models"
)

// OpenAIConfigRepository manages OpenAI configuration in the database.
type OpenAIConfigRepository struct {
	db *sql.DB
}

// NewOpenAIConfigRepository creates a new repository for OpenAI configuration.
func NewOpenAIConfigRepository(db *sql.DB) *OpenAIConfigRepository {
	return &OpenAIConfigRepository{db: db}
}

// Get retrieves the OpenAI configuration.
func (r *OpenAIConfigRepository) Get(ctx context.Context) (*models.OpenAIConfig, error) {
	query := `
		SELECT id, api_key, model, temperature, max_tokens, timeout_seconds,
		       system_prompt, analysis_template, entity_extraction_prompt, correlation_system_prompt,
		       enabled, updated_at, created_at
		FROM openai_config
		LIMIT 1
	`

	config := &models.OpenAIConfig{}
	err := r.db.QueryRowContext(ctx, query).Scan(
		&config.ID,
		&config.APIKey,
		&config.Model,
		&config.Temperature,
		&config.MaxTokens,
		&config.TimeoutSeconds,
		&config.SystemPrompt,
		&config.AnalysisTemplate,
		&config.EntityExtractionPrompt,
		&config.CorrelationSystemPrompt,
		&config.Enabled,
		&config.UpdatedAt,
		&config.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("openai configuration not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get openai configuration: %w", err)
	}

	return config, nil
}

// Update updates the OpenAI configuration.
func (r *OpenAIConfigRepository) Update(ctx context.Context, update models.OpenAIConfigUpdate) (*models.OpenAIConfig, error) {
	// Build dynamic update query
	query := `UPDATE openai_config SET updated_at = $1`
	args := []interface{}{time.Now()}
	argCount := 1

	if update.APIKey != nil {
		argCount++
		query += fmt.Sprintf(", api_key = $%d", argCount)
		args = append(args, *update.APIKey)
	}
	if update.Model != nil {
		argCount++
		query += fmt.Sprintf(", model = $%d", argCount)
		args = append(args, *update.Model)
	}
	if update.Temperature != nil {
		argCount++
		query += fmt.Sprintf(", temperature = $%d", argCount)
		args = append(args, *update.Temperature)
	}
	if update.MaxTokens != nil {
		argCount++
		query += fmt.Sprintf(", max_tokens = $%d", argCount)
		args = append(args, *update.MaxTokens)
	}
	if update.TimeoutSeconds != nil {
		argCount++
		query += fmt.Sprintf(", timeout_seconds = $%d", argCount)
		args = append(args, *update.TimeoutSeconds)
	}
	if update.SystemPrompt != nil {
		argCount++
		query += fmt.Sprintf(", system_prompt = $%d", argCount)
		args = append(args, *update.SystemPrompt)
	}
	if update.AnalysisTemplate != nil {
		argCount++
		query += fmt.Sprintf(", analysis_template = $%d", argCount)
		args = append(args, *update.AnalysisTemplate)
	}
	if update.EntityExtractionPrompt != nil {
		argCount++
		query += fmt.Sprintf(", entity_extraction_prompt = $%d", argCount)
		args = append(args, *update.EntityExtractionPrompt)
	}
	if update.CorrelationSystemPrompt != nil {
		argCount++
		query += fmt.Sprintf(", correlation_system_prompt = $%d", argCount)
		args = append(args, *update.CorrelationSystemPrompt)
	}
	if update.Enabled != nil {
		argCount++
		query += fmt.Sprintf(", enabled = $%d", argCount)
		args = append(args, *update.Enabled)
	}

	query += ` RETURNING id, api_key, model, temperature, max_tokens, timeout_seconds,
	                     system_prompt, analysis_template, entity_extraction_prompt, correlation_system_prompt,
	                     enabled, updated_at, created_at`

	config := &models.OpenAIConfig{}
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&config.ID,
		&config.APIKey,
		&config.Model,
		&config.Temperature,
		&config.MaxTokens,
		&config.TimeoutSeconds,
		&config.SystemPrompt,
		&config.AnalysisTemplate,
		&config.EntityExtractionPrompt,
		&config.CorrelationSystemPrompt,
		&config.Enabled,
		&config.UpdatedAt,
		&config.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to update openai configuration: %w", err)
	}

	return config, nil
}

// TestConnection tests the OpenAI API connection with the current configuration.
func (r *OpenAIConfigRepository) TestConnection(ctx context.Context) error {
	// This method could be implemented to actually test the OpenAI API
	// For now, just verify we can read the config
	_, err := r.Get(ctx)
	return err
}
