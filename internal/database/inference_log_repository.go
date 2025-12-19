package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/STRATINT/stratint/internal/models"
)

// InferenceLogRepository handles inference log database operations
type InferenceLogRepository struct {
	db *sql.DB
}

// NewInferenceLogRepository creates a new repository
func NewInferenceLogRepository(db *sql.DB) *InferenceLogRepository {
	return &InferenceLogRepository{db: db}
}

// Create logs a new inference call
func (r *InferenceLogRepository) Create(ctx context.Context, log models.InferenceLog) error {
	query := `
		INSERT INTO inference_logs (
			provider, model, operation, tokens_used, input_tokens, output_tokens,
			cost_usd, latency_ms, status, error_message, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err := r.db.ExecContext(ctx, query,
		log.Provider,
		log.Model,
		log.Operation,
		log.TokensUsed,
		log.InputTokens,
		log.OutputTokens,
		log.CostUSD,
		log.LatencyMs,
		log.Status,
		log.ErrorMessage,
		log.Metadata,
	)

	return err
}

// List retrieves inference logs with optional filtering
func (r *InferenceLogRepository) List(ctx context.Context, query models.InferenceLogQuery) ([]models.InferenceLog, error) {
	sqlQuery := `
		SELECT id, provider, model, operation, tokens_used, input_tokens, output_tokens,
		       cost_usd, latency_ms, status, error_message, metadata, created_at
		FROM inference_logs
		WHERE 1=1
	`
	args := []interface{}{}
	argPos := 1

	if query.Provider != "" {
		sqlQuery += fmt.Sprintf(" AND provider = $%d", argPos)
		args = append(args, query.Provider)
		argPos++
	}

	if query.Model != "" {
		sqlQuery += fmt.Sprintf(" AND model = $%d", argPos)
		args = append(args, query.Model)
		argPos++
	}

	if query.Operation != "" {
		sqlQuery += fmt.Sprintf(" AND operation = $%d", argPos)
		args = append(args, query.Operation)
		argPos++
	}

	if query.Status != "" {
		sqlQuery += fmt.Sprintf(" AND status = $%d", argPos)
		args = append(args, query.Status)
		argPos++
	}

	if query.StartDate != nil {
		sqlQuery += fmt.Sprintf(" AND created_at >= $%d", argPos)
		args = append(args, query.StartDate)
		argPos++
	}

	if query.EndDate != nil {
		sqlQuery += fmt.Sprintf(" AND created_at <= $%d", argPos)
		args = append(args, query.EndDate)
		argPos++
	}

	sqlQuery += " ORDER BY created_at DESC"

	if query.Limit > 0 {
		sqlQuery += fmt.Sprintf(" LIMIT $%d", argPos)
		args = append(args, query.Limit)
		argPos++
	}

	if query.Offset > 0 {
		sqlQuery += fmt.Sprintf(" OFFSET $%d", argPos)
		args = append(args, query.Offset)
	}

	rows, err := r.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query inference logs: %w", err)
	}
	defer rows.Close()

	var logs []models.InferenceLog
	for rows.Next() {
		var log models.InferenceLog
		var metadata sql.NullString

		err := rows.Scan(
			&log.ID,
			&log.Provider,
			&log.Model,
			&log.Operation,
			&log.TokensUsed,
			&log.InputTokens,
			&log.OutputTokens,
			&log.CostUSD,
			&log.LatencyMs,
			&log.Status,
			&log.ErrorMessage,
			&metadata,
			&log.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan inference log: %w", err)
		}

		if metadata.Valid {
			log.Metadata = metadata.String
		}

		logs = append(logs, log)
	}

	return logs, nil
}

// GetStats retrieves aggregated statistics
func (r *InferenceLogRepository) GetStats(ctx context.Context, startDate, endDate *time.Time) (*models.InferenceLogStats, error) {
	query := `
		SELECT
			COUNT(*) as total_calls,
			COALESCE(SUM(tokens_used), 0) as total_tokens,
			COALESCE(SUM(cost_usd), 0) as total_cost_usd,
			SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END) as successful_calls,
			SUM(CASE WHEN status = 'error' THEN 1 ELSE 0 END) as failed_calls,
			COALESCE(AVG(latency_ms), 0) as avg_latency_ms
		FROM inference_logs
		WHERE 1=1
	`
	args := []interface{}{}
	argPos := 1

	if startDate != nil {
		query += fmt.Sprintf(" AND created_at >= $%d", argPos)
		args = append(args, startDate)
		argPos++
	}

	if endDate != nil {
		query += fmt.Sprintf(" AND created_at <= $%d", argPos)
		args = append(args, endDate)
	}

	var stats models.InferenceLogStats
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&stats.TotalCalls,
		&stats.TotalTokens,
		&stats.TotalCostUSD,
		&stats.SuccessfulCalls,
		&stats.FailedCalls,
		&stats.AvgLatencyMs,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get inference stats: %w", err)
	}

	return &stats, nil
}
