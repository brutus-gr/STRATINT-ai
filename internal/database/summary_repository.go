package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/STRATINT/stratint/internal/models"
	"github.com/lib/pq"
)

type SummaryRepository struct {
	db *sql.DB
}

func NewSummaryRepository(db *sql.DB) *SummaryRepository {
	return &SummaryRepository{db: db}
}

// formatTimeOfDay converts PostgreSQL TIME format (HH:MM:SS) to HTML time input format (HH:MM)
func formatTimeOfDay(tod *string) *string {
	if tod == nil || *tod == "" {
		return nil
	}
	// PostgreSQL returns TIME as HH:MM:SS, but HTML input expects HH:MM
	parts := strings.Split(*tod, ":")
	if len(parts) >= 2 {
		formatted := parts[0] + ":" + parts[1]
		return &formatted
	}
	return tod
}

func (r *SummaryRepository) Create(ctx context.Context, summary *models.Summary) error {
	modelsJSON, err := json.Marshal(summary.Models)
	if err != nil {
		return fmt.Errorf("failed to marshal models: %w", err)
	}

	query := `
		INSERT INTO summaries (name, prompt, time_of_day, lookback_hours, categories, headline_count, models, active, schedule_enabled, schedule_interval, auto_post_to_twitter, include_forecasts)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, created_at, updated_at
	`
	return r.db.QueryRowContext(ctx, query,
		summary.Name,
		summary.Prompt,
		summary.TimeOfDay,
		summary.LookbackHours,
		pq.Array(summary.Categories),
		summary.HeadlineCount,
		modelsJSON,
		summary.Active,
		summary.ScheduleEnabled,
		summary.ScheduleInterval,
		summary.AutoPostToTwitter,
		summary.IncludeForecasts,
	).Scan(&summary.ID, &summary.CreatedAt, &summary.UpdatedAt)
}

func (r *SummaryRepository) List(ctx context.Context) ([]models.Summary, error) {
	query := `
		SELECT id, name, prompt, time_of_day::text, lookback_hours, categories, headline_count, models, active, schedule_enabled, schedule_interval, auto_post_to_twitter, include_forecasts, last_run_at, next_run_at, created_at, updated_at
		FROM summaries
		ORDER BY created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list summaries: %w", err)
	}
	defer rows.Close()

	var summaries []models.Summary
	for rows.Next() {
		var s models.Summary
		var modelsJSON []byte
		err := rows.Scan(
			&s.ID, &s.Name, &s.Prompt, &s.TimeOfDay, &s.LookbackHours,
			pq.Array(&s.Categories), &s.HeadlineCount, &modelsJSON,
			&s.Active, &s.ScheduleEnabled, &s.ScheduleInterval, &s.AutoPostToTwitter, &s.IncludeForecasts,
			&s.LastRunAt, &s.NextRunAt, &s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(modelsJSON, &s.Models); err != nil {
			return nil, err
		}
		// Format time_of_day from HH:MM:SS to HH:MM for HTML time inputs
		s.TimeOfDay = formatTimeOfDay(s.TimeOfDay)
		summaries = append(summaries, s)
	}
	return summaries, nil
}

func (r *SummaryRepository) Get(ctx context.Context, id string) (*models.Summary, error) {
	query := `
		SELECT id, name, prompt, time_of_day::text, lookback_hours, categories, headline_count, models, active, schedule_enabled, schedule_interval, auto_post_to_twitter, include_forecasts, last_run_at, next_run_at, created_at, updated_at
		FROM summaries
		WHERE id = $1
	`
	var s models.Summary
	var modelsJSON []byte
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&s.ID, &s.Name, &s.Prompt, &s.TimeOfDay, &s.LookbackHours,
		pq.Array(&s.Categories), &s.HeadlineCount, &modelsJSON,
		&s.Active, &s.ScheduleEnabled, &s.ScheduleInterval, &s.AutoPostToTwitter, &s.IncludeForecasts,
		&s.LastRunAt, &s.NextRunAt, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(modelsJSON, &s.Models); err != nil {
		return nil, err
	}
	// Format time_of_day from HH:MM:SS to HH:MM for HTML time inputs
	s.TimeOfDay = formatTimeOfDay(s.TimeOfDay)
	return &s, nil
}

func (r *SummaryRepository) Update(ctx context.Context, summary *models.Summary) error {
	modelsJSON, err := json.Marshal(summary.Models)
	if err != nil {
		return err
	}

	query := `
		UPDATE summaries
		SET name = $1, prompt = $2, time_of_day = $3, lookback_hours = $4, categories = $5, headline_count = $6, models = $7, active = $8, schedule_enabled = $9, schedule_interval = $10, auto_post_to_twitter = $11, include_forecasts = $12
		WHERE id = $13
	`
	_, err = r.db.ExecContext(ctx, query,
		summary.Name, summary.Prompt, summary.TimeOfDay, summary.LookbackHours,
		pq.Array(summary.Categories), summary.HeadlineCount, modelsJSON,
		summary.Active, summary.ScheduleEnabled, summary.ScheduleInterval, summary.AutoPostToTwitter, summary.IncludeForecasts, summary.ID,
	)
	return err
}

func (r *SummaryRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM summaries WHERE id = $1", id)
	return err
}

func (r *SummaryRepository) CreateRun(ctx context.Context, summaryID string, headlineCount int, lookbackStart, lookbackEnd time.Time) (string, error) {
	var runID string
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO summary_runs (summary_id, headline_count, lookback_start, lookback_end, status)
		 VALUES ($1, $2, $3, $4, 'pending') RETURNING id`,
		summaryID, headlineCount, lookbackStart, lookbackEnd,
	).Scan(&runID)
	return runID, err
}

func (r *SummaryRepository) SaveResult(ctx context.Context, runID, summaryText, provider, modelName string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO summary_results (run_id, summary_text, model_provider, model_name)
		 VALUES ($1, $2, $3, $4)`,
		runID, summaryText, provider, modelName,
	)
	return err
}

func (r *SummaryRepository) CompleteRun(ctx context.Context, runID string, status string, errorMsg *string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE summary_runs SET status = $1, error_message = $2, completed_at = CURRENT_TIMESTAMP WHERE id = $3`,
		status, errorMsg, runID,
	)
	return err
}

func (r *SummaryRepository) GetLatestRun(ctx context.Context, summaryID string) (*models.SummaryRunDetail, error) {
	// Get latest completed run
	var runID string
	err := r.db.QueryRowContext(ctx,
		`SELECT id FROM summary_runs WHERE summary_id = $1 AND status = 'completed' ORDER BY run_at DESC LIMIT 1`,
		summaryID,
	).Scan(&runID)
	if err != nil {
		return nil, err
	}

	// Get run details
	var run models.SummaryRun
	err = r.db.QueryRowContext(ctx,
		`SELECT id, summary_id, run_at, headline_count, lookback_start, lookback_end, status, error_message, completed_at
		 FROM summary_runs WHERE id = $1`,
		runID,
	).Scan(&run.ID, &run.SummaryID, &run.RunAt, &run.HeadlineCount, &run.LookbackStart,
		&run.LookbackEnd, &run.Status, &run.ErrorMessage, &run.CompletedAt)
	if err != nil {
		return nil, err
	}

	// Get results
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, run_id, summary_text, model_provider, model_name, created_at
		 FROM summary_results WHERE run_id = $1 ORDER BY created_at`,
		runID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []models.SummaryResult
	for rows.Next() {
		var r models.SummaryResult
		if err := rows.Scan(&r.ID, &r.RunID, &r.SummaryText, &r.ModelProvider, &r.ModelName, &r.CreatedAt); err != nil {
			return nil, err
		}
		results = append(results, r)
	}

	return &models.SummaryRunDetail{Run: run, Results: results}, nil
}

func (r *SummaryRepository) GetRunByID(ctx context.Context, runID string) (*models.SummaryRunDetail, error) {
	// Get run details
	var run models.SummaryRun
	err := r.db.QueryRowContext(ctx,
		`SELECT id, summary_id, run_at, headline_count, lookback_start, lookback_end, status, error_message, completed_at
		 FROM summary_runs WHERE id = $1`,
		runID,
	).Scan(&run.ID, &run.SummaryID, &run.RunAt, &run.HeadlineCount, &run.LookbackStart,
		&run.LookbackEnd, &run.Status, &run.ErrorMessage, &run.CompletedAt)
	if err != nil {
		return nil, err
	}

	// Get results
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, run_id, summary_text, model_provider, model_name, created_at
		 FROM summary_results WHERE run_id = $1 ORDER BY created_at`,
		runID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []models.SummaryResult
	for rows.Next() {
		var r models.SummaryResult
		if err := rows.Scan(&r.ID, &r.RunID, &r.SummaryText, &r.ModelProvider, &r.ModelName, &r.CreatedAt); err != nil {
			return nil, err
		}
		results = append(results, r)
	}

	return &models.SummaryRunDetail{Run: run, Results: results}, nil
}

func (r *SummaryRepository) ListRuns(ctx context.Context, summaryID string) ([]models.SummaryRun, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, summary_id, run_at, headline_count, lookback_start, lookback_end, status, error_message, completed_at
		 FROM summary_runs WHERE summary_id = $1 ORDER BY run_at DESC LIMIT 50`,
		summaryID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runs []models.SummaryRun
	for rows.Next() {
		var r models.SummaryRun
		if err := rows.Scan(&r.ID, &r.SummaryID, &r.RunAt, &r.HeadlineCount, &r.LookbackStart,
			&r.LookbackEnd, &r.Status, &r.ErrorMessage, &r.CompletedAt); err != nil {
			return nil, err
		}
		runs = append(runs, r)
	}
	return runs, nil
}
