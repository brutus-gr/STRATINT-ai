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

// ForecastRepository handles forecast database operations
type ForecastRepository struct {
	db *sql.DB
}

// NewForecastRepository creates a new forecast repository
func NewForecastRepository(db *sql.DB) *ForecastRepository {
	return &ForecastRepository{db: db}
}

// CreateForecast creates a new forecast with its models
func (r *ForecastRepository) CreateForecast(ctx context.Context, req models.CreateForecastRequest) (*models.Forecast, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Create forecast
	forecastID := uuid.New().String()
	now := time.Now()

	query := `
		INSERT INTO forecasts (id, name, proposition, prediction_type, units, target_date, categories, headline_count, iterations, context_urls, active, schedule_enabled, schedule_interval, last_run_at, next_run_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
	`

	iterations := req.Iterations
	if iterations < 1 {
		iterations = 1
	}

	_, err = tx.ExecContext(ctx, query, forecastID, req.Name, req.Proposition, req.PredictionType, req.Units, req.TargetDate, pq.Array(req.Categories), req.HeadlineCount, iterations, pq.Array(req.ContextURLs), true, false, 0, nil, nil, now, now)
	if err != nil {
		return nil, fmt.Errorf("failed to create forecast: %w", err)
	}

	// Create forecast models
	for _, model := range req.Models {
		modelID := uuid.New().String()
		modelQuery := `
			INSERT INTO forecast_models (id, forecast_id, provider, model_name, api_key, weight, active, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`
		_, err = tx.ExecContext(ctx, modelQuery, modelID, forecastID, model.Provider, model.ModelName, model.APIKey, model.Weight, true, now)
		if err != nil {
			return nil, fmt.Errorf("failed to create forecast model: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Return the created forecast
	return r.GetForecast(ctx, forecastID)
}

// UpdateForecast updates an existing forecast
func (r *ForecastRepository) UpdateForecast(ctx context.Context, id string, req models.CreateForecastRequest) (*models.Forecast, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	now := time.Now()

	// Update forecast (preserve existing schedule settings)
	query := `
		UPDATE forecasts
		SET name = $1, proposition = $2, prediction_type = $3, units = $4, target_date = $5, categories = $6, headline_count = $7, iterations = $8, context_urls = $9, updated_at = $10
		WHERE id = $11
	`

	iterations := req.Iterations
	if iterations < 1 {
		iterations = 1
	}

	_, err = tx.ExecContext(ctx, query, req.Name, req.Proposition, req.PredictionType, req.Units, req.TargetDate, pq.Array(req.Categories), req.HeadlineCount, iterations, pq.Array(req.ContextURLs), now, id)
	if err != nil {
		return nil, fmt.Errorf("failed to update forecast: %w", err)
	}

	// Deactivate existing models
	_, err = tx.ExecContext(ctx, "UPDATE forecast_models SET active = false WHERE forecast_id = $1", id)
	if err != nil {
		return nil, fmt.Errorf("failed to deactivate existing models: %w", err)
	}

	// Create new models
	for _, model := range req.Models {
		modelID := uuid.New().String()
		modelQuery := `
			INSERT INTO forecast_models (id, forecast_id, provider, model_name, api_key, weight, active, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`
		_, err = tx.ExecContext(ctx, modelQuery, modelID, id, model.Provider, model.ModelName, model.APIKey, model.Weight, true, now)
		if err != nil {
			return nil, fmt.Errorf("failed to create forecast model: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Return the updated forecast
	return r.GetForecast(ctx, id)
}

// GetForecast retrieves a forecast by ID
func (r *ForecastRepository) GetForecast(ctx context.Context, id string) (*models.Forecast, error) {
	query := `
		SELECT id, name, proposition, prediction_type, units, target_date, categories, headline_count, iterations, context_urls, active, public, display_order, schedule_enabled, schedule_interval, last_run_at, next_run_at, created_at, updated_at
		FROM forecasts
		WHERE id = $1
	`

	var forecast models.Forecast

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&forecast.ID,
		&forecast.Name,
		&forecast.Proposition,
		&forecast.PredictionType,
		&forecast.Units,
		&forecast.TargetDate,
		pq.Array(&forecast.Categories),
		&forecast.HeadlineCount,
		&forecast.Iterations,
		pq.Array(&forecast.ContextURLs),
		&forecast.Active,
		&forecast.Public,
		&forecast.DisplayOrder,
		&forecast.ScheduleEnabled,
		&forecast.ScheduleInterval,
		&forecast.LastRunAt,
		&forecast.NextRunAt,
		&forecast.CreatedAt,
		&forecast.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get forecast: %w", err)
	}

	return &forecast, nil
}

// ListForecasts retrieves all forecasts
func (r *ForecastRepository) ListForecasts(ctx context.Context) ([]models.Forecast, error) {
	query := `
		SELECT id, name, proposition, prediction_type, units, target_date, categories, headline_count, iterations, context_urls, active, public, display_order, schedule_enabled, schedule_interval, last_run_at, next_run_at, created_at, updated_at
		FROM forecasts
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list forecasts: %w", err)
	}
	defer rows.Close()

	var forecasts []models.Forecast
	for rows.Next() {
		var forecast models.Forecast

		err := rows.Scan(
			&forecast.ID,
			&forecast.Name,
			&forecast.Proposition,
			&forecast.PredictionType,
			&forecast.Units,
			&forecast.TargetDate,
			pq.Array(&forecast.Categories),
			&forecast.HeadlineCount,
			&forecast.Iterations,
			pq.Array(&forecast.ContextURLs),
			&forecast.Active,
			&forecast.Public,
			&forecast.DisplayOrder,
			&forecast.ScheduleEnabled,
			&forecast.ScheduleInterval,
			&forecast.LastRunAt,
			&forecast.NextRunAt,
			&forecast.CreatedAt,
			&forecast.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan forecast: %w", err)
		}

		forecasts = append(forecasts, forecast)
	}

	return forecasts, nil
}

// DeleteForecast deletes a forecast by ID (manually deletes related records in correct order)
func (r *ForecastRepository) DeleteForecast(ctx context.Context, id string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete forecast_model_responses (references both runs and models)
	_, err = tx.ExecContext(ctx, `
		DELETE FROM forecast_model_responses
		WHERE run_id IN (SELECT id FROM forecast_runs WHERE forecast_id = $1)
	`, id)
	if err != nil {
		return fmt.Errorf("failed to delete forecast model responses: %w", err)
	}

	// Delete forecast_results (references runs)
	_, err = tx.ExecContext(ctx, `
		DELETE FROM forecast_results
		WHERE run_id IN (SELECT id FROM forecast_runs WHERE forecast_id = $1)
	`, id)
	if err != nil {
		return fmt.Errorf("failed to delete forecast results: %w", err)
	}

	// Delete forecast_runs (references forecasts)
	_, err = tx.ExecContext(ctx, `DELETE FROM forecast_runs WHERE forecast_id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete forecast runs: %w", err)
	}

	// Delete forecast_models (references forecasts)
	_, err = tx.ExecContext(ctx, `DELETE FROM forecast_models WHERE forecast_id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete forecast models: %w", err)
	}

	// Finally, delete the forecast itself
	result, err := tx.ExecContext(ctx, `DELETE FROM forecasts WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete forecast: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("forecast not found")
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetForecastModels retrieves all models for a forecast
func (r *ForecastRepository) GetForecastModels(ctx context.Context, forecastID string) ([]models.ForecastModel, error) {
	query := `
		SELECT id, forecast_id, provider, model_name, api_key, weight, active, created_at
		FROM forecast_models
		WHERE forecast_id = $1 AND active = true
		ORDER BY created_at
	`

	rows, err := r.db.QueryContext(ctx, query, forecastID)
	if err != nil {
		return nil, fmt.Errorf("failed to get forecast models: %w", err)
	}
	defer rows.Close()

	var forecastModels []models.ForecastModel
	for rows.Next() {
		var model models.ForecastModel
		err := rows.Scan(
			&model.ID,
			&model.ForecastID,
			&model.Provider,
			&model.ModelName,
			&model.APIKey,
			&model.Weight,
			&model.Active,
			&model.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan forecast model: %w", err)
		}
		forecastModels = append(forecastModels, model)
	}

	return forecastModels, nil
}

// CreateForecastRun creates a new forecast run
func (r *ForecastRepository) CreateForecastRun(ctx context.Context, forecastID string, headlines []models.ForecastHeadline) (string, error) {
	runID := uuid.New().String()
	now := time.Now()

	headlinesJSON, err := json.Marshal(headlines)
	if err != nil {
		return "", fmt.Errorf("failed to marshal headlines: %w", err)
	}

	query := `
		INSERT INTO forecast_runs (id, forecast_id, run_at, headline_count, headlines_snapshot, status)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err = r.db.ExecContext(ctx, query, runID, forecastID, now, len(headlines), headlinesJSON, "pending")
	if err != nil {
		return "", fmt.Errorf("failed to create forecast run: %w", err)
	}

	return runID, nil
}

// UpdateForecastRunStatus updates the status of a forecast run
func (r *ForecastRepository) UpdateForecastRunStatus(ctx context.Context, runID, status, errorMsg string) error {
	var completedAt *time.Time
	if status == "completed" || status == "failed" {
		now := time.Now()
		completedAt = &now
	}

	query := `
		UPDATE forecast_runs
		SET status = $1, error_message = $2, completed_at = $3
		WHERE id = $4
	`

	_, err := r.db.ExecContext(ctx, query, status, errorMsg, completedAt, runID)
	return err
}

// CreateModelResponse creates a model response
func (r *ForecastRepository) CreateModelResponse(ctx context.Context, response models.ForecastModelResponse) error {
	if response.ID == "" {
		response.ID = uuid.New().String()
	}
	if response.CreatedAt.IsZero() {
		response.CreatedAt = time.Now()
	}

	rawResponseJSON, err := json.Marshal(response.RawResponse)
	if err != nil {
		return fmt.Errorf("failed to marshal raw response: %w", err)
	}

	var percentilesJSON []byte
	if response.PercentilePredictions != nil {
		percentilesJSON, err = json.Marshal(response.PercentilePredictions)
		if err != nil {
			return fmt.Errorf("failed to marshal percentile predictions: %w", err)
		}
	}

	query := `
		INSERT INTO forecast_model_responses (
			id, run_id, model_id, provider, model_name, percentile_predictions, reasoning,
			raw_response, tokens_used, response_time_ms, status, error_message, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	_, err = r.db.ExecContext(ctx, query,
		response.ID, response.RunID, response.ModelID, response.Provider, response.ModelName,
		percentilesJSON, response.Reasoning, rawResponseJSON, response.TokensUsed,
		response.ResponseTimeMs, response.Status, response.ErrorMessage, response.CreatedAt,
	)

	return err
}

// CreateForecastResult creates a forecast result
func (r *ForecastRepository) CreateForecastResult(ctx context.Context, result models.ForecastResult) error {
	if result.ID == "" {
		result.ID = uuid.New().String()
	}
	if result.CreatedAt.IsZero() {
		result.CreatedAt = time.Now()
	}

	var percentilesJSON []byte
	var err error
	if result.AggregatedPercentiles != nil {
		percentilesJSON, err = json.Marshal(result.AggregatedPercentiles)
		if err != nil {
			return fmt.Errorf("failed to marshal aggregated percentiles: %w", err)
		}
	}

	query := `
		INSERT INTO forecast_results (
			id, run_id, aggregated_percentiles, aggregated_point_estimate,
			model_count, consensus_level, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err = r.db.ExecContext(ctx, query,
		result.ID, result.RunID, percentilesJSON, result.AggregatedPointEstimate,
		result.ModelCount, result.ConsensusLevel, result.CreatedAt,
	)

	return err
}

// GetForecastRun retrieves a forecast run with all details
func (r *ForecastRepository) GetForecastRun(ctx context.Context, runID string) (*models.ForecastRunDetail, error) {
	// Get run
	runQuery := `
		SELECT id, forecast_id, run_at, headline_count, headlines_snapshot, status, error_message, completed_at
		FROM forecast_runs
		WHERE id = $1
	`

	var run models.ForecastRun
	var headlinesJSON []byte
	var errorMsg sql.NullString
	var completedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, runQuery, runID).Scan(
		&run.ID, &run.ForecastID, &run.RunAt, &run.HeadlineCount,
		&headlinesJSON, &run.Status, &errorMsg, &completedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get forecast run: %w", err)
	}

	if errorMsg.Valid {
		run.ErrorMessage = errorMsg.String
	}
	if completedAt.Valid {
		run.CompletedAt = &completedAt.Time
	}

	if err := json.Unmarshal(headlinesJSON, &run.HeadlinesSnapshot); err != nil {
		return nil, fmt.Errorf("failed to unmarshal headlines: %w", err)
	}

	// Get responses
	responsesQuery := `
		SELECT id, run_id, model_id, provider, model_name, percentile_predictions, point_estimate,
		       reasoning, raw_response, tokens_used, response_time_ms, status, error_message, created_at
		FROM forecast_model_responses
		WHERE run_id = $1
		ORDER BY created_at
	`

	rows, err := r.db.QueryContext(ctx, responsesQuery, runID)
	if err != nil {
		return nil, fmt.Errorf("failed to get model responses: %w", err)
	}
	defer rows.Close()

	var responses []models.ForecastModelResponse
	for rows.Next() {
		var resp models.ForecastModelResponse
		var percentilesJSON []byte
		var pointEstimate sql.NullFloat64
		var tokensUsed, responseTime sql.NullInt64
		var rawResponseJSON []byte
		var errMsg sql.NullString

		err := rows.Scan(
			&resp.ID, &resp.RunID, &resp.ModelID, &resp.Provider, &resp.ModelName,
			&percentilesJSON, &pointEstimate, &resp.Reasoning, &rawResponseJSON,
			&tokensUsed, &responseTime, &resp.Status, &errMsg, &resp.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan model response: %w", err)
		}

		if len(percentilesJSON) > 0 {
			var percentiles models.PercentilePredictions
			if err := json.Unmarshal(percentilesJSON, &percentiles); err != nil {
				return nil, fmt.Errorf("failed to unmarshal percentile predictions: %w", err)
			}
			resp.PercentilePredictions = &percentiles
		}
		if pointEstimate.Valid {
			resp.PointEstimate = &pointEstimate.Float64
		}
		if tokensUsed.Valid {
			tokens := int(tokensUsed.Int64)
			resp.TokensUsed = &tokens
		}
		if responseTime.Valid {
			respTime := int(responseTime.Int64)
			resp.ResponseTimeMs = &respTime
		}
		if errMsg.Valid {
			resp.ErrorMessage = errMsg.String
		}

		if len(rawResponseJSON) > 0 {
			if err := json.Unmarshal(rawResponseJSON, &resp.RawResponse); err != nil {
				return nil, fmt.Errorf("failed to unmarshal raw response: %w", err)
			}
		}

		responses = append(responses, resp)
	}

	// Get result
	resultQuery := `
		SELECT id, run_id, aggregated_percentiles, aggregated_point_estimate,
		       model_count, consensus_level, created_at
		FROM forecast_results
		WHERE run_id = $1
	`

	var result models.ForecastResult
	var percentilesJSON []byte
	var pointEstimate sql.NullFloat64
	var consensus sql.NullFloat64

	err = r.db.QueryRowContext(ctx, resultQuery, runID).Scan(
		&result.ID, &result.RunID, &percentilesJSON, &pointEstimate,
		&result.ModelCount, &consensus, &result.CreatedAt,
	)

	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get forecast result: %w", err)
	}

	var resultPtr *models.ForecastResult
	if err != sql.ErrNoRows {
		if len(percentilesJSON) > 0 {
			var percentiles models.PercentilePredictions
			if err := json.Unmarshal(percentilesJSON, &percentiles); err != nil {
				return nil, fmt.Errorf("failed to unmarshal aggregated percentiles: %w", err)
			}
			result.AggregatedPercentiles = &percentiles
		}
		if pointEstimate.Valid {
			result.AggregatedPointEstimate = &pointEstimate.Float64
		}
		if consensus.Valid {
			result.ConsensusLevel = &consensus.Float64
		}
		resultPtr = &result
	}

	return &models.ForecastRunDetail{
		Run:       run,
		Responses: responses,
		Result:    resultPtr,
	}, nil
}

// GetLatestCompletedForecastRun gets the most recent completed run for a forecast with its result
func (r *ForecastRepository) GetLatestCompletedForecastRun(ctx context.Context, forecastID string) (*models.ForecastRunDetail, error) {
	// Get the latest completed run
	runQuery := `
		SELECT fr.id
		FROM forecast_runs fr
		INNER JOIN forecast_results res ON res.run_id = fr.id
		WHERE fr.forecast_id = $1 AND fr.status = 'completed'
		ORDER BY fr.run_at DESC
		LIMIT 1
	`

	var runID string
	err := r.db.QueryRowContext(ctx, runQuery, forecastID).Scan(&runID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get latest completed run: %w", err)
	}

	// Use existing GetForecastRun method to get full details
	return r.GetForecastRun(ctx, runID)
}

// GetLatestNCompletedForecastRuns gets the N most recent completed runs for a forecast with their results
func (r *ForecastRepository) GetLatestNCompletedForecastRuns(ctx context.Context, forecastID string, n int) ([]models.ForecastRunDetail, error) {
	if n < 1 {
		n = 1
	}

	// Get the latest N completed runs
	runQuery := `
		SELECT fr.id
		FROM forecast_runs fr
		INNER JOIN forecast_results res ON res.run_id = fr.id
		WHERE fr.forecast_id = $1 AND fr.status = 'completed'
		ORDER BY fr.run_at DESC
		LIMIT $2
	`

	rows, err := r.db.QueryContext(ctx, runQuery, forecastID, n)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest completed runs: %w", err)
	}
	defer rows.Close()

	var runIDs []string
	for rows.Next() {
		var runID string
		if err := rows.Scan(&runID); err != nil {
			return nil, fmt.Errorf("failed to scan run ID: %w", err)
		}
		runIDs = append(runIDs, runID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	// Fetch full details for each run
	var runs []models.ForecastRunDetail
	for _, runID := range runIDs {
		runDetail, err := r.GetForecastRun(ctx, runID)
		if err != nil {
			return nil, fmt.Errorf("failed to get forecast run %s: %w", runID, err)
		}
		if runDetail != nil {
			runs = append(runs, *runDetail)
		}
	}

	return runs, nil
}

// ListForecastRuns lists all runs for a forecast
func (r *ForecastRepository) ListForecastRuns(ctx context.Context, forecastID string, limit int) ([]models.ForecastRun, error) {
	query := `
		SELECT id, forecast_id, run_at, headline_count, status, error_message, completed_at
		FROM forecast_runs
		WHERE forecast_id = $1
		ORDER BY run_at DESC
		LIMIT $2
	`

	rows, err := r.db.QueryContext(ctx, query, forecastID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list forecast runs: %w", err)
	}
	defer rows.Close()

	var runs []models.ForecastRun
	for rows.Next() {
		var run models.ForecastRun
		var errorMsg sql.NullString
		var completedAt sql.NullTime

		err := rows.Scan(
			&run.ID, &run.ForecastID, &run.RunAt, &run.HeadlineCount,
			&run.Status, &errorMsg, &completedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan forecast run: %w", err)
		}

		if errorMsg.Valid {
			run.ErrorMessage = errorMsg.String
		}
		if completedAt.Valid {
			run.CompletedAt = &completedAt.Time
		}

		runs = append(runs, run)
	}

	return runs, nil
}

// GetForecastHistory retrieves completed runs with results for a forecast (optimized for charting)
func (r *ForecastRepository) GetForecastHistory(ctx context.Context, forecastID string) ([]models.ForecastRunDetail, error) {
	// Get completed runs with their results
	query := `
		SELECT
			fr.id, fr.forecast_id, fr.run_at, fr.headline_count, fr.status, fr.error_message, fr.completed_at,
			fres.id, fres.aggregated_percentiles, fres.aggregated_point_estimate, fres.model_count, fres.consensus_level
		FROM forecast_runs fr
		LEFT JOIN forecast_results fres ON fr.id = fres.run_id
		WHERE fr.forecast_id = $1 AND fr.status = 'completed'
		ORDER BY fr.run_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, forecastID)
	if err != nil {
		return nil, fmt.Errorf("failed to get forecast history: %w", err)
	}
	defer rows.Close()

	var history []models.ForecastRunDetail
	for rows.Next() {
		var run models.ForecastRun
		var errorMsg sql.NullString
		var completedAt sql.NullTime

		var resultID sql.NullString
		var percentilesJSON []byte
		var pointEstimate sql.NullFloat64
		var modelCount sql.NullInt64
		var consensus sql.NullFloat64

		err := rows.Scan(
			&run.ID, &run.ForecastID, &run.RunAt, &run.HeadlineCount,
			&run.Status, &errorMsg, &completedAt,
			&resultID, &percentilesJSON, &pointEstimate, &modelCount, &consensus,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan forecast history: %w", err)
		}

		if errorMsg.Valid {
			run.ErrorMessage = errorMsg.String
		}
		if completedAt.Valid {
			run.CompletedAt = &completedAt.Time
		}

		var resultPtr *models.ForecastResult
		if resultID.Valid {
			result := models.ForecastResult{
				ID:    resultID.String,
				RunID: run.ID,
			}

			if len(percentilesJSON) > 0 {
				var percentiles models.PercentilePredictions
				if err := json.Unmarshal(percentilesJSON, &percentiles); err != nil {
					return nil, fmt.Errorf("failed to unmarshal percentiles: %w", err)
				}
				result.AggregatedPercentiles = &percentiles
			}
			if pointEstimate.Valid {
				result.AggregatedPointEstimate = &pointEstimate.Float64
			}
			if modelCount.Valid {
				result.ModelCount = int(modelCount.Int64)
			}
			if consensus.Valid {
				result.ConsensusLevel = &consensus.Float64
			}

			resultPtr = &result
		}

		history = append(history, models.ForecastRunDetail{
			Run:       run,
			Responses: []models.ForecastModelResponse{}, // Empty for history view
			Result:    resultPtr,
		})
	}

	return history, nil
}

// DailyOHLC represents OHLC data for a single day
type DailyOHLC struct {
	Date  string  `json:"date"`
	Open  float64 `json:"open"`
	High  float64 `json:"high"`
	Low   float64 `json:"low"`
	Close float64 `json:"close"`
}

// GetForecastHistoryDaily returns P50 values aggregated into daily OHLC bars
func (r *ForecastRepository) GetForecastHistoryDaily(ctx context.Context, forecastID string) ([]DailyOHLC, error) {
	// Aggregate P50 values by day using window functions
	query := `
		WITH daily_p50 AS (
			SELECT
				DATE(fr.run_at) as date,
				(fres.aggregated_percentiles->>'p50')::float as p50,
				fr.run_at,
				ROW_NUMBER() OVER (PARTITION BY DATE(fr.run_at) ORDER BY fr.run_at ASC) as first_run,
				ROW_NUMBER() OVER (PARTITION BY DATE(fr.run_at) ORDER BY fr.run_at DESC) as last_run
			FROM forecast_runs fr
			INNER JOIN forecast_results fres ON fr.id = fres.run_id
			WHERE fr.forecast_id = $1
				AND fr.status = 'completed'
				AND fres.aggregated_percentiles IS NOT NULL
		)
		SELECT
			date::text,
			MAX(CASE WHEN first_run = 1 THEN p50 END) as open,
			MAX(p50) as high,
			MIN(p50) as low,
			MAX(CASE WHEN last_run = 1 THEN p50 END) as close
		FROM daily_p50
		GROUP BY date
		ORDER BY date ASC
	`

	rows, err := r.db.QueryContext(ctx, query, forecastID)
	if err != nil {
		return nil, fmt.Errorf("failed to get daily OHLC: %w", err)
	}
	defer rows.Close()

	var ohlcData []DailyOHLC
	for rows.Next() {
		var ohlc DailyOHLC
		var open, close sql.NullFloat64

		err := rows.Scan(&ohlc.Date, &open, &ohlc.High, &ohlc.Low, &close)
		if err != nil {
			return nil, fmt.Errorf("failed to scan OHLC data: %w", err)
		}

		// Handle NULL for open/close (shouldn't happen but be safe)
		if open.Valid {
			ohlc.Open = open.Float64
		}
		if close.Valid {
			ohlc.Close = close.Float64
		}

		ohlcData = append(ohlcData, ohlc)
	}

	return ohlcData, nil
}

// GetForecastHistory4Hour returns P50 values aggregated into 4-hour OHLC bars
func (r *ForecastRepository) GetForecastHistory4Hour(ctx context.Context, forecastID string) ([]DailyOHLC, error) {
	// Aggregate P50 values by 4-hour intervals using window functions
	// 14400 seconds = 4 hours
	query := `
		WITH bucketed_p50 AS (
			SELECT
				to_timestamp(floor((extract(epoch from fr.run_at) / 14400)) * 14400) as bucket,
				(fres.aggregated_percentiles->>'p50')::float as p50,
				fr.run_at,
				ROW_NUMBER() OVER (PARTITION BY floor((extract(epoch from fr.run_at) / 14400)) ORDER BY fr.run_at ASC) as first_run,
				ROW_NUMBER() OVER (PARTITION BY floor((extract(epoch from fr.run_at) / 14400)) ORDER BY fr.run_at DESC) as last_run
			FROM forecast_runs fr
			INNER JOIN forecast_results fres ON fr.id = fres.run_id
			WHERE fr.forecast_id = $1
				AND fr.status = 'completed'
				AND fres.aggregated_percentiles IS NOT NULL
		)
		SELECT
			EXTRACT(EPOCH FROM bucket)::bigint as time,
			MAX(CASE WHEN first_run = 1 THEN p50 END) as open,
			MAX(p50) as high,
			MIN(p50) as low,
			MAX(CASE WHEN last_run = 1 THEN p50 END) as close
		FROM bucketed_p50
		GROUP BY bucket
		ORDER BY bucket ASC
	`

	rows, err := r.db.QueryContext(ctx, query, forecastID)
	if err != nil {
		return nil, fmt.Errorf("failed to get 4-hour OHLC: %w", err)
	}
	defer rows.Close()

	var ohlcData []DailyOHLC
	for rows.Next() {
		var ohlc DailyOHLC
		var open, close sql.NullFloat64
		var timestamp int64

		err := rows.Scan(&timestamp, &open, &ohlc.High, &ohlc.Low, &close)
		if err != nil {
			return nil, fmt.Errorf("failed to scan OHLC data: %w", err)
		}

		// Convert Unix timestamp to string for JSON (frontend will parse as number)
		ohlc.Date = fmt.Sprintf("%d", timestamp)

		// Handle NULL for open/close (shouldn't happen but be safe)
		if open.Valid {
			ohlc.Open = open.Float64
		}
		if close.Valid {
			ohlc.Close = close.Float64
		}

		ohlcData = append(ohlcData, ohlc)
	}

	return ohlcData, nil
}

// UpdateForecastSchedule updates the schedule settings for a forecast
func (r *ForecastRepository) UpdateForecastSchedule(ctx context.Context, forecastID string, enabled bool, intervalMinutes int) error {
	var nextRunAt *time.Time
	if enabled && intervalMinutes > 0 {
		next := time.Now().Add(time.Duration(intervalMinutes) * time.Minute)
		nextRunAt = &next
	}

	query := `
		UPDATE forecasts
		SET schedule_enabled = $1, schedule_interval = $2, next_run_at = $3, updated_at = $4
		WHERE id = $5
	`

	_, err := r.db.ExecContext(ctx, query, enabled, intervalMinutes, nextRunAt, time.Now(), forecastID)
	return err
}

// UpdateForecastLastRun updates the last_run_at and next_run_at for a forecast
func (r *ForecastRepository) UpdateForecastLastRun(ctx context.Context, forecastID string) error {
	query := `
		UPDATE forecasts
		SET last_run_at = $1::timestamp,
		    next_run_at = CASE
		        WHEN schedule_enabled AND schedule_interval > 0
		        THEN $1::timestamp + (schedule_interval || ' minutes')::interval
		        ELSE NULL
		    END
		WHERE id = $2
	`

	_, err := r.db.ExecContext(ctx, query, time.Now(), forecastID)
	return err
}

// GetScheduledForecasts retrieves all forecasts that are due to run
// Uses atomic UPDATE with SKIP LOCKED to prevent duplicate execution across multiple instances
func (r *ForecastRepository) GetScheduledForecasts(ctx context.Context) ([]models.Forecast, error) {
	// Use UPDATE with SKIP LOCKED to atomically claim forecasts and prevent duplicates
	// This ensures only ONE instance can claim each forecast, even across multiple Cloud Run instances
	query := `
		UPDATE forecasts
		SET last_run_at = $1::timestamp,
		    next_run_at = $1::timestamp + (schedule_interval || ' minutes')::interval
		WHERE id IN (
			SELECT id
			FROM forecasts
			WHERE schedule_enabled = TRUE
			  AND active = TRUE
			  AND schedule_interval > 0
			  AND (next_run_at IS NULL OR next_run_at <= $1)
			ORDER BY next_run_at ASC NULLS FIRST
			FOR UPDATE SKIP LOCKED
		)
		RETURNING id, name, proposition, prediction_type, units, target_date, categories, headline_count, iterations, context_urls, active, public, display_order, schedule_enabled, schedule_interval, last_run_at, next_run_at, created_at, updated_at
	`

	now := time.Now()
	rows, err := r.db.QueryContext(ctx, query, now)
	if err != nil {
		return nil, fmt.Errorf("failed to get scheduled forecasts: %w", err)
	}
	defer rows.Close()

	var forecasts []models.Forecast
	for rows.Next() {
		var forecast models.Forecast
		var targetDate sql.NullTime
		var lastRunAt sql.NullTime
		var nextRunAt sql.NullTime
		err := rows.Scan(
			&forecast.ID,
			&forecast.Name,
			&forecast.Proposition,
			&forecast.PredictionType,
			&forecast.Units,
			&targetDate,
			pq.Array(&forecast.Categories),
			&forecast.HeadlineCount,
			&forecast.Iterations,
			pq.Array(&forecast.ContextURLs),
			&forecast.Active,
			&forecast.Public,
			&forecast.DisplayOrder,
			&forecast.ScheduleEnabled,
			&forecast.ScheduleInterval,
			&lastRunAt,
			&nextRunAt,
			&forecast.CreatedAt,
			&forecast.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan scheduled forecast: %w", err)
		}

		if targetDate.Valid {
			forecast.TargetDate = &targetDate.Time
		}
		if lastRunAt.Valid {
			forecast.LastRunAt = &lastRunAt.Time
		}
		if nextRunAt.Valid {
			forecast.NextRunAt = &nextRunAt.Time
		}
		forecasts = append(forecasts, forecast)
	}

	return forecasts, nil
}

// DeleteForecastRun deletes a forecast run by ID
func (r *ForecastRepository) DeleteForecastRun(ctx context.Context, runID string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete model responses for this run
	_, err = tx.ExecContext(ctx, `DELETE FROM forecast_model_responses WHERE run_id = $1`, runID)
	if err != nil {
		return fmt.Errorf("failed to delete model responses: %w", err)
	}

	// Delete result for this run
	_, err = tx.ExecContext(ctx, `DELETE FROM forecast_results WHERE run_id = $1`, runID)
	if err != nil {
		return fmt.Errorf("failed to delete result: %w", err)
	}

	// Delete the run itself
	result, err := tx.ExecContext(ctx, `DELETE FROM forecast_runs WHERE id = $1`, runID)
	if err != nil {
		return fmt.Errorf("failed to delete forecast run: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("forecast run not found")
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// DeleteAllRunsForForecast deletes all runs for a forecast
func (r *ForecastRepository) DeleteAllRunsForForecast(ctx context.Context, forecastID string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Get all run IDs for this forecast
	rows, err := tx.QueryContext(ctx, `SELECT id FROM forecast_runs WHERE forecast_id = $1`, forecastID)
	if err != nil {
		return fmt.Errorf("failed to get run IDs: %w", err)
	}
	defer rows.Close()

	var runIDs []string
	for rows.Next() {
		var runID string
		if err := rows.Scan(&runID); err != nil {
			return fmt.Errorf("failed to scan run ID: %w", err)
		}
		runIDs = append(runIDs, runID)
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating run IDs: %w", err)
	}

	// Delete model responses for all runs
	_, err = tx.ExecContext(ctx, `DELETE FROM forecast_model_responses WHERE run_id = ANY($1)`, pq.Array(runIDs))
	if err != nil {
		return fmt.Errorf("failed to delete model responses: %w", err)
	}

	// Delete results for all runs
	_, err = tx.ExecContext(ctx, `DELETE FROM forecast_results WHERE run_id = ANY($1)`, pq.Array(runIDs))
	if err != nil {
		return fmt.Errorf("failed to delete results: %w", err)
	}

	// Delete all runs
	_, err = tx.ExecContext(ctx, `DELETE FROM forecast_runs WHERE forecast_id = $1`, forecastID)
	if err != nil {
		return fmt.Errorf("failed to delete forecast runs: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// UpdateForecastPublic updates the public status of a forecast
func (r *ForecastRepository) UpdateForecastPublic(ctx context.Context, forecastID string, public bool) error {
	query := `UPDATE forecasts SET public = $1, updated_at = $2 WHERE id = $3`
	result, err := r.db.ExecContext(ctx, query, public, time.Now(), forecastID)
	if err != nil {
		return fmt.Errorf("failed to update forecast public status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("forecast not found")
	}

	return nil
}

// UpdateForecastDisplayOrder updates the display order of a forecast
func (r *ForecastRepository) UpdateForecastDisplayOrder(ctx context.Context, forecastID string, displayOrder int) error {
	query := `UPDATE forecasts SET display_order = $1, updated_at = $2 WHERE id = $3`
	result, err := r.db.ExecContext(ctx, query, displayOrder, time.Now(), forecastID)
	if err != nil {
		return fmt.Errorf("failed to update forecast display order: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("forecast not found")
	}

	return nil
}

// ListPublicForecasts returns all public forecasts with their latest runs
func (r *ForecastRepository) ListPublicForecasts(ctx context.Context) ([]models.Forecast, error) {
	query := `
		SELECT
			id, name, proposition, prediction_type, units, target_date, categories, headline_count, iterations, context_urls, active, public, display_order, schedule_enabled, schedule_interval, last_run_at, next_run_at, created_at, updated_at
		FROM forecasts
		WHERE public = true AND active = true
		ORDER BY display_order DESC, updated_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query public forecasts: %w", err)
	}
	defer rows.Close()

	var forecasts []models.Forecast
	for rows.Next() {
		var f models.Forecast
		var targetDate sql.NullTime
		var lastRunAt sql.NullTime
		var nextRunAt sql.NullTime
		err := rows.Scan(
			&f.ID, &f.Name, &f.Proposition, &f.PredictionType, &f.Units, &targetDate, pq.Array(&f.Categories), &f.HeadlineCount, &f.Iterations, pq.Array(&f.ContextURLs), &f.Active, &f.Public, &f.DisplayOrder, &f.ScheduleEnabled, &f.ScheduleInterval, &lastRunAt, &nextRunAt, &f.CreatedAt, &f.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan forecast: %w", err)
		}

		if targetDate.Valid {
			f.TargetDate = &targetDate.Time
		}
		if lastRunAt.Valid {
			f.LastRunAt = &lastRunAt.Time
		}
		if nextRunAt.Valid {
			f.NextRunAt = &nextRunAt.Time
		}

		forecasts = append(forecasts, f)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating forecasts: %w", err)
	}

	return forecasts, nil
}
