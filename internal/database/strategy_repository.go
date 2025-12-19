package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/STRATINT/stratint/internal/models"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

// StrategyRepository handles strategy database operations
type StrategyRepository struct {
	db *sql.DB
}

// NewStrategyRepository creates a new strategy repository
func NewStrategyRepository(db *sql.DB) *StrategyRepository {
	return &StrategyRepository{db: db}
}

// CreateStrategy creates a new strategy with its models
func (r *StrategyRepository) CreateStrategy(ctx context.Context, req models.CreateStrategyRequest) (*models.Strategy, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Create strategy
	strategyID := uuid.New().String()
	now := time.Now()

	query := `
		INSERT INTO strategies (id, name, prompt, investment_symbols, categories, headline_count, iterations, forecast_ids, forecast_history_count, active, schedule_enabled, schedule_interval, last_run_at, next_run_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`

	iterations := req.Iterations
	if iterations < 1 {
		iterations = 1
	}

	forecastHistoryCount := req.ForecastHistoryCount
	if forecastHistoryCount < 1 {
		forecastHistoryCount = 1
	}

	_, err = tx.ExecContext(ctx, query, strategyID, req.Name, req.Prompt, pq.Array(req.InvestmentSymbols), pq.Array(req.Categories), req.HeadlineCount, iterations, pq.Array(req.ForecastIDs), forecastHistoryCount, true, false, 0, nil, nil, now, now)
	if err != nil {
		return nil, fmt.Errorf("failed to create strategy: %w", err)
	}

	// Create strategy models
	for _, model := range req.Models {
		modelID := uuid.New().String()
		modelQuery := `
			INSERT INTO strategy_models (id, strategy_id, provider, model_name, api_key, weight, active, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`
		_, err = tx.ExecContext(ctx, modelQuery, modelID, strategyID, model.Provider, model.ModelName, model.APIKey, model.Weight, true, now)
		if err != nil {
			return nil, fmt.Errorf("failed to create strategy model: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Return the created strategy
	return r.GetStrategy(ctx, strategyID)
}

// UpdateStrategy updates an existing strategy
func (r *StrategyRepository) UpdateStrategy(ctx context.Context, id string, req models.CreateStrategyRequest) (*models.Strategy, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	now := time.Now()

	iterations := req.Iterations
	if iterations < 1 {
		iterations = 1
	}

	forecastHistoryCount := req.ForecastHistoryCount
	if forecastHistoryCount < 1 {
		forecastHistoryCount = 1
	}

	query := `
		UPDATE strategies
		SET name = $2, prompt = $3, investment_symbols = $4, categories = $5, headline_count = $6, iterations = $7, forecast_ids = $8, forecast_history_count = $9, updated_at = $10
		WHERE id = $1
	`

	result, err := tx.ExecContext(ctx, query, id, req.Name, req.Prompt, pq.Array(req.InvestmentSymbols), pq.Array(req.Categories), req.HeadlineCount, iterations, pq.Array(req.ForecastIDs), forecastHistoryCount, now)
	if err != nil {
		return nil, fmt.Errorf("failed to update strategy: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return nil, fmt.Errorf("strategy not found")
	}

	// Delete existing models
	_, err = tx.ExecContext(ctx, `DELETE FROM strategy_models WHERE strategy_id = $1`, id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete existing models: %w", err)
	}

	// Create new models
	for _, model := range req.Models {
		modelID := uuid.New().String()
		modelQuery := `
			INSERT INTO strategy_models (id, strategy_id, provider, model_name, api_key, weight, active, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`
		_, err = tx.ExecContext(ctx, modelQuery, modelID, id, model.Provider, model.ModelName, model.APIKey, model.Weight, true, now)
		if err != nil {
			return nil, fmt.Errorf("failed to create strategy model: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return r.GetStrategy(ctx, id)
}

// GetStrategy retrieves a single strategy by ID
func (r *StrategyRepository) GetStrategy(ctx context.Context, id string) (*models.Strategy, error) {
	query := `
		SELECT id, name, prompt, investment_symbols, categories, headline_count, iterations, forecast_ids, forecast_history_count, active, public, display_order, schedule_enabled, schedule_interval, last_run_at, next_run_at, created_at, updated_at
		FROM strategies
		WHERE id = $1
	`

	var strategy models.Strategy
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&strategy.ID,
		&strategy.Name,
		&strategy.Prompt,
		pq.Array(&strategy.InvestmentSymbols),
		pq.Array(&strategy.Categories),
		&strategy.HeadlineCount,
		&strategy.Iterations,
		pq.Array(&strategy.ForecastIDs),
		&strategy.ForecastHistoryCount,
		&strategy.Active,
		&strategy.Public,
		&strategy.DisplayOrder,
		&strategy.ScheduleEnabled,
		&strategy.ScheduleInterval,
		&strategy.LastRunAt,
		&strategy.NextRunAt,
		&strategy.CreatedAt,
		&strategy.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("strategy not found")
		}
		return nil, fmt.Errorf("failed to get strategy: %w", err)
	}

	return &strategy, nil
}

// ListStrategies retrieves all strategies
func (r *StrategyRepository) ListStrategies(ctx context.Context) ([]models.Strategy, error) {
	query := `
		SELECT id, name, prompt, investment_symbols, categories, headline_count, iterations, forecast_ids, forecast_history_count, active, public, display_order, schedule_enabled, schedule_interval, last_run_at, next_run_at, created_at, updated_at
		FROM strategies
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list strategies: %w", err)
	}
	defer rows.Close()

	var strategies []models.Strategy
	for rows.Next() {
		var strategy models.Strategy
		err := rows.Scan(
			&strategy.ID,
			&strategy.Name,
			&strategy.Prompt,
			pq.Array(&strategy.InvestmentSymbols),
			pq.Array(&strategy.Categories),
			&strategy.HeadlineCount,
			&strategy.Iterations,
			pq.Array(&strategy.ForecastIDs),
			&strategy.ForecastHistoryCount,
			&strategy.Active,
			&strategy.Public,
			&strategy.DisplayOrder,
			&strategy.ScheduleEnabled,
			&strategy.ScheduleInterval,
			&strategy.LastRunAt,
			&strategy.NextRunAt,
			&strategy.CreatedAt,
			&strategy.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan strategy: %w", err)
		}

		strategies = append(strategies, strategy)
	}

	return strategies, nil
}

// ListPublicStrategies retrieves all public strategies ordered by display_order
func (r *StrategyRepository) ListPublicStrategies(ctx context.Context) ([]models.Strategy, error) {
	query := `
		SELECT id, name, prompt, investment_symbols, categories, headline_count, iterations, forecast_ids, forecast_history_count, active, public, display_order, schedule_enabled, schedule_interval, last_run_at, next_run_at, created_at, updated_at
		FROM strategies
		WHERE public = true
		ORDER BY display_order DESC, created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list public strategies: %w", err)
	}
	defer rows.Close()

	var strategies []models.Strategy
	for rows.Next() {
		var strategy models.Strategy
		err := rows.Scan(
			&strategy.ID,
			&strategy.Name,
			&strategy.Prompt,
			pq.Array(&strategy.InvestmentSymbols),
			pq.Array(&strategy.Categories),
			&strategy.HeadlineCount,
			&strategy.Iterations,
			pq.Array(&strategy.ForecastIDs),
			&strategy.ForecastHistoryCount,
			&strategy.Active,
			&strategy.Public,
			&strategy.DisplayOrder,
			&strategy.ScheduleEnabled,
			&strategy.ScheduleInterval,
			&strategy.LastRunAt,
			&strategy.NextRunAt,
			&strategy.CreatedAt,
			&strategy.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan strategy: %w", err)
		}

		strategies = append(strategies, strategy)
	}

	return strategies, nil
}

// DeleteStrategy deletes a strategy by ID
func (r *StrategyRepository) DeleteStrategy(ctx context.Context, id string) error {
	// CASCADE constraints will handle deletion of related records
	result, err := r.db.ExecContext(ctx, `DELETE FROM strategies WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete strategy: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("strategy not found")
	}

	return nil
}

// GetStrategyModels retrieves all models for a strategy
func (r *StrategyRepository) GetStrategyModels(ctx context.Context, strategyID string) ([]models.StrategyModel, error) {
	query := `
		SELECT id, strategy_id, provider, model_name, api_key, weight, active, created_at
		FROM strategy_models
		WHERE strategy_id = $1 AND active = true
		ORDER BY created_at
	`

	rows, err := r.db.QueryContext(ctx, query, strategyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get strategy models: %w", err)
	}
	defer rows.Close()

	var strategyModels []models.StrategyModel
	for rows.Next() {
		var model models.StrategyModel
		err := rows.Scan(
			&model.ID,
			&model.StrategyID,
			&model.Provider,
			&model.ModelName,
			&model.APIKey,
			&model.Weight,
			&model.Active,
			&model.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan strategy model: %w", err)
		}

		strategyModels = append(strategyModels, model)
	}

	return strategyModels, nil
}

// CreateStrategyRun creates a new strategy execution run
func (r *StrategyRepository) CreateStrategyRun(ctx context.Context, strategyID string, headlines []models.StrategyHeadline, forecastSnapshots []models.ForecastSnapshot) (string, error) {
	runID := uuid.New().String()

	headlinesJSON, err := json.Marshal(headlines)
	if err != nil {
		return "", fmt.Errorf("failed to marshal headlines: %w", err)
	}

	forecastsJSON, err := json.Marshal(forecastSnapshots)
	if err != nil {
		return "", fmt.Errorf("failed to marshal forecast snapshots: %w", err)
	}

	query := `
		INSERT INTO strategy_runs (id, strategy_id, run_at, headline_count, headlines_snapshot, forecast_snapshots, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err = r.db.ExecContext(ctx, query, runID, strategyID, time.Now(), len(headlines), headlinesJSON, forecastsJSON, "pending")
	if err != nil {
		return "", fmt.Errorf("failed to create strategy run: %w", err)
	}

	return runID, nil
}

// UpdateStrategyRunStatus updates the status of a strategy run
func (r *StrategyRepository) UpdateStrategyRunStatus(ctx context.Context, runID, status, errorMsg string) error {
	var query string
	var args []interface{}

	if status == "completed" {
		query = `
			UPDATE strategy_runs
			SET status = $2, completed_at = $3, error_message = $4
			WHERE id = $1
		`
		args = []interface{}{runID, status, time.Now(), errorMsg}
	} else {
		query = `
			UPDATE strategy_runs
			SET status = $2, error_message = $3
			WHERE id = $1
		`
		args = []interface{}{runID, status, errorMsg}
	}

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update strategy run status: %w", err)
	}

	return nil
}

// CreateModelResponse creates a model response record
func (r *StrategyRepository) CreateModelResponse(ctx context.Context, response models.StrategyModelResponse) error {
	responseID := uuid.New().String()

	allocationsJSON, err := json.Marshal(response.Allocations)
	if err != nil {
		return fmt.Errorf("failed to marshal allocations: %w", err)
	}

	rawResponseJSON, err := json.Marshal(response.RawResponse)
	if err != nil {
		return fmt.Errorf("failed to marshal raw response: %w", err)
	}

	query := `
		INSERT INTO strategy_model_responses (id, run_id, model_id, iteration, provider, model_name, allocations, reasoning, raw_response, tokens_used, response_time_ms, status, error_message, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`

	_, err = r.db.ExecContext(ctx, query, responseID, response.RunID, response.ModelID, response.Iteration, response.Provider, response.ModelName, allocationsJSON, response.Reasoning, rawResponseJSON, response.TokensUsed, response.ResponseTimeMs, response.Status, response.ErrorMessage, time.Now())
	if err != nil {
		return fmt.Errorf("failed to create model response: %w", err)
	}

	return nil
}

// CreateStrategyResult creates a strategy result record
func (r *StrategyRepository) CreateStrategyResult(ctx context.Context, result models.StrategyResult) error {
	resultID := uuid.New().String()

	averagedJSON, err := json.Marshal(result.AveragedAllocations)
	if err != nil {
		return fmt.Errorf("failed to marshal averaged allocations: %w", err)
	}

	normalizedJSON, err := json.Marshal(result.NormalizedAllocations)
	if err != nil {
		return fmt.Errorf("failed to marshal normalized allocations: %w", err)
	}

	varianceJSON, err := json.Marshal(result.ConsensusVariance)
	if err != nil {
		return fmt.Errorf("failed to marshal consensus variance: %w", err)
	}

	query := `
		INSERT INTO strategy_results (id, run_id, averaged_allocations, normalized_allocations, normalization_reasoning, model_count, iteration_count, consensus_variance, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err = r.db.ExecContext(ctx, query, resultID, result.RunID, averagedJSON, normalizedJSON, result.NormalizationReasoning, result.ModelCount, result.IterationCount, varianceJSON, time.Now())
	if err != nil {
		return fmt.Errorf("failed to create strategy result: %w", err)
	}

	return nil
}

// GetStrategyRun retrieves a strategy run with all responses and result
func (r *StrategyRepository) GetStrategyRun(ctx context.Context, runID string) (*models.StrategyRunDetail, error) {
	// Get the run
	runQuery := `
		SELECT id, strategy_id, run_at, headline_count, headlines_snapshot, forecast_snapshots, status, error_message, completed_at
		FROM strategy_runs
		WHERE id = $1
	`

	var run models.StrategyRun
	var headlinesJSON, forecastsJSON []byte
	err := r.db.QueryRowContext(ctx, runQuery, runID).Scan(
		&run.ID,
		&run.StrategyID,
		&run.RunAt,
		&run.HeadlineCount,
		&headlinesJSON,
		&forecastsJSON,
		&run.Status,
		&run.ErrorMessage,
		&run.CompletedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("strategy run not found")
		}
		return nil, fmt.Errorf("failed to get strategy run: %w", err)
	}

	if err := json.Unmarshal(headlinesJSON, &run.HeadlinesSnapshot); err != nil {
		return nil, fmt.Errorf("failed to unmarshal headlines: %w", err)
	}

	if err := json.Unmarshal(forecastsJSON, &run.ForecastSnapshots); err != nil {
		return nil, fmt.Errorf("failed to unmarshal forecast snapshots: %w", err)
	}

	// Get all responses
	responsesQuery := `
		SELECT id, run_id, model_id, iteration, provider, model_name, allocations, reasoning, raw_response, tokens_used, response_time_ms, status, error_message, created_at
		FROM strategy_model_responses
		WHERE run_id = $1
		ORDER BY iteration, created_at
	`

	rows, err := r.db.QueryContext(ctx, responsesQuery, runID)
	if err != nil {
		return nil, fmt.Errorf("failed to get model responses: %w", err)
	}
	defer rows.Close()

	var responses []models.StrategyModelResponse
	for rows.Next() {
		var response models.StrategyModelResponse
		var allocationsJSON, rawResponseJSON []byte

		err := rows.Scan(
			&response.ID,
			&response.RunID,
			&response.ModelID,
			&response.Iteration,
			&response.Provider,
			&response.ModelName,
			&allocationsJSON,
			&response.Reasoning,
			&rawResponseJSON,
			&response.TokensUsed,
			&response.ResponseTimeMs,
			&response.Status,
			&response.ErrorMessage,
			&response.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan model response: %w", err)
		}

		if err := json.Unmarshal(allocationsJSON, &response.Allocations); err != nil {
			return nil, fmt.Errorf("failed to unmarshal allocations: %w", err)
		}

		if err := json.Unmarshal(rawResponseJSON, &response.RawResponse); err != nil {
			return nil, fmt.Errorf("failed to unmarshal raw response: %w", err)
		}

		responses = append(responses, response)
	}

	// Get the result if it exists
	var result *models.StrategyResult
	resultQuery := `
		SELECT id, run_id, averaged_allocations, normalized_allocations, normalization_reasoning, model_count, iteration_count, consensus_variance, created_at
		FROM strategy_results
		WHERE run_id = $1
	`

	var resultData models.StrategyResult
	var averagedJSON, normalizedJSON, varianceJSON []byte
	err = r.db.QueryRowContext(ctx, resultQuery, runID).Scan(
		&resultData.ID,
		&resultData.RunID,
		&averagedJSON,
		&normalizedJSON,
		&resultData.NormalizationReasoning,
		&resultData.ModelCount,
		&resultData.IterationCount,
		&varianceJSON,
		&resultData.CreatedAt,
	)
	if err == nil {
		if err := json.Unmarshal(averagedJSON, &resultData.AveragedAllocations); err == nil {
			if err := json.Unmarshal(normalizedJSON, &resultData.NormalizedAllocations); err == nil {
				if err := json.Unmarshal(varianceJSON, &resultData.ConsensusVariance); err == nil {
					result = &resultData
				}
			}
		}
	}

	return &models.StrategyRunDetail{
		Run:       run,
		Responses: responses,
		Result:    result,
	}, nil
}

// UpdateStrategyPublic toggles the public visibility of a strategy
func (r *StrategyRepository) UpdateStrategyPublic(ctx context.Context, id string, public bool) error {
	query := `UPDATE strategies SET public = $2, updated_at = $3 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id, public, time.Now())
	if err != nil {
		return fmt.Errorf("failed to update strategy public status: %w", err)
	}
	return nil
}

// UpdateStrategyDisplayOrder updates the display order of a strategy
func (r *StrategyRepository) UpdateStrategyDisplayOrder(ctx context.Context, id string, displayOrder int) error {
	query := `UPDATE strategies SET display_order = $2, updated_at = $3 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id, displayOrder, time.Now())
	if err != nil {
		return fmt.Errorf("failed to update strategy display order: %w", err)
	}
	return nil
}

// UpdateStrategyLastRun updates the last run timestamp for a strategy
func (r *StrategyRepository) UpdateStrategyLastRun(ctx context.Context, id string, lastRunAt time.Time) error {
	query := `UPDATE strategies SET last_run_at = $2, updated_at = $3 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id, lastRunAt, time.Now())
	if err != nil {
		return fmt.Errorf("failed to update strategy last run: %w", err)
	}
	return nil
}

// UpdateStrategySchedule updates the schedule settings for a strategy
func (r *StrategyRepository) UpdateStrategySchedule(ctx context.Context, id string, enabled bool, interval int, nextRunAt *time.Time) error {
	query := `UPDATE strategies SET schedule_enabled = $2, schedule_interval = $3, next_run_at = $4, updated_at = $5 WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id, enabled, interval, nextRunAt, time.Now())
	if err != nil {
		return fmt.Errorf("failed to update strategy schedule (query failed): %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("no strategy found with id %s", id)
	}
	return nil
}

// ListStrategyRuns retrieves runs for a specific strategy
func (r *StrategyRepository) ListStrategyRuns(ctx context.Context, strategyID string, limit int) ([]models.StrategyRun, error) {
	query := `
		SELECT id, strategy_id, run_at, headline_count, headlines_snapshot, forecast_snapshots, status, error_message, completed_at
		FROM strategy_runs
		WHERE strategy_id = $1
		ORDER BY run_at DESC
		LIMIT $2
	`

	rows, err := r.db.QueryContext(ctx, query, strategyID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list strategy runs: %w", err)
	}
	defer rows.Close()

	var runs []models.StrategyRun
	for rows.Next() {
		var run models.StrategyRun
		var headlinesJSON, forecastsJSON []byte

		err := rows.Scan(
			&run.ID,
			&run.StrategyID,
			&run.RunAt,
			&run.HeadlineCount,
			&headlinesJSON,
			&forecastsJSON,
			&run.Status,
			&run.ErrorMessage,
			&run.CompletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan strategy run: %w", err)
		}

		if err := json.Unmarshal(headlinesJSON, &run.HeadlinesSnapshot); err != nil {
			return nil, fmt.Errorf("failed to unmarshal headlines: %w", err)
		}

		if err := json.Unmarshal(forecastsJSON, &run.ForecastSnapshots); err != nil {
			return nil, fmt.Errorf("failed to unmarshal forecast snapshots: %w", err)
		}

		runs = append(runs, run)
	}

	return runs, nil
}

// GetLatestStrategyResult retrieves the latest completed run with result for a strategy
func (r *StrategyRepository) GetLatestStrategyResult(ctx context.Context, strategyID string) (*models.StrategyRunDetail, error) {
	// Get the most recent completed run
	runQuery := `
		SELECT sr.id
		FROM strategy_runs sr
		INNER JOIN strategy_results sres ON sr.id = sres.run_id
		WHERE sr.strategy_id = $1 AND sr.status = 'completed'
		ORDER BY sr.run_at DESC
		LIMIT 1
	`

	var runID string
	err := r.db.QueryRowContext(ctx, runQuery, strategyID).Scan(&runID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no completed runs found")
		}
		return nil, fmt.Errorf("failed to get latest strategy result: %w", err)
	}

	// Use the existing GetStrategyRun method to get full details
	return r.GetStrategyRun(ctx, runID)
}

// GetScheduledStrategies retrieves all strategies that are due to run
// Uses atomic UPDATE with SKIP LOCKED to prevent duplicate execution across multiple instances
func (r *StrategyRepository) GetScheduledStrategies(ctx context.Context) ([]models.Strategy, error) {
	// Use UPDATE with SKIP LOCKED to atomically claim strategies and prevent duplicates
	// This ensures only ONE instance can claim each strategy, even across multiple Cloud Run instances
	query := `
		UPDATE strategies
		SET last_run_at = $1::timestamp,
		    next_run_at = $1::timestamp + (schedule_interval || ' minutes')::interval
		WHERE id IN (
			SELECT id
			FROM strategies
			WHERE schedule_enabled = TRUE
			  AND active = TRUE
			  AND schedule_interval > 0
			  AND (next_run_at IS NULL OR next_run_at <= $1)
			ORDER BY next_run_at ASC NULLS FIRST
			FOR UPDATE SKIP LOCKED
		)
		RETURNING id, name, prompt, investment_symbols, categories, headline_count, iterations, forecast_ids, forecast_history_count, active, public, display_order, schedule_enabled, schedule_interval, last_run_at, next_run_at, created_at, updated_at
	`

	now := time.Now()
	rows, err := r.db.QueryContext(ctx, query, now)
	if err != nil {
		return nil, fmt.Errorf("failed to get scheduled strategies: %w", err)
	}
	defer rows.Close()

	var strategies []models.Strategy
	for rows.Next() {
		var strategy models.Strategy
		var lastRunAt, nextRunAt sql.NullTime
		var forecastHistoryCount sql.NullInt32
		err := rows.Scan(
			&strategy.ID,
			&strategy.Name,
			&strategy.Prompt,
			pq.Array(&strategy.InvestmentSymbols),
			pq.Array(&strategy.Categories),
			&strategy.HeadlineCount,
			&strategy.Iterations,
			pq.Array(&strategy.ForecastIDs),
			&forecastHistoryCount,
			&strategy.Active,
			&strategy.Public,
			&strategy.DisplayOrder,
			&strategy.ScheduleEnabled,
			&strategy.ScheduleInterval,
			&lastRunAt,
			&nextRunAt,
			&strategy.CreatedAt,
			&strategy.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan strategy: %w", err)
		}

		if lastRunAt.Valid {
			strategy.LastRunAt = &lastRunAt.Time
		}
		if nextRunAt.Valid {
			strategy.NextRunAt = &nextRunAt.Time
		}
		if forecastHistoryCount.Valid {
			strategy.ForecastHistoryCount = int(forecastHistoryCount.Int32)
		}

		strategies = append(strategies, strategy)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating strategies: %w", err)
	}

	return strategies, nil
}
