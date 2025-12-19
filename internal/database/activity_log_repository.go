package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/STRATINT/stratint/internal/models"
	"github.com/google/uuid"
)

// ActivityLogRepository handles activity log storage and retrieval.
type ActivityLogRepository struct {
	db *sql.DB
}

// NewActivityLogRepository creates a new activity log repository.
func NewActivityLogRepository(db *sql.DB) *ActivityLogRepository {
	return &ActivityLogRepository{db: db}
}

// Log stores a new activity log entry.
func (r *ActivityLogRepository) Log(ctx context.Context, log models.ActivityLog) error {
	if log.ID == "" {
		log.ID = uuid.New().String()
	}
	if log.Timestamp.IsZero() {
		log.Timestamp = time.Now()
	}

	var detailsJSON []byte
	var err error
	if log.Details != nil {
		detailsJSON, err = json.Marshal(log.Details)
		if err != nil {
			return fmt.Errorf("failed to marshal details: %w", err)
		}
	}

	query := `
		INSERT INTO activity_logs (id, timestamp, activity_type, platform, message, details, source_count, duration_ms)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err = r.db.ExecContext(ctx, query,
		log.ID,
		log.Timestamp,
		log.ActivityType,
		log.Platform,
		log.Message,
		detailsJSON,
		log.SourceCount,
		log.DurationMs,
	)

	return err
}

// List retrieves activity logs with optional filtering.
func (r *ActivityLogRepository) List(ctx context.Context, limit int, activityType string, platform string) ([]models.ActivityLog, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}

	query := `
		SELECT id, timestamp, activity_type, platform, message, details, source_count, duration_ms
		FROM activity_logs
		WHERE 1=1
	`
	args := []interface{}{}
	argPos := 1

	if activityType != "" {
		query += fmt.Sprintf(" AND activity_type = $%d", argPos)
		args = append(args, activityType)
		argPos++
	}

	if platform != "" {
		query += fmt.Sprintf(" AND platform = $%d", argPos)
		args = append(args, platform)
		argPos++
	}

	query += " ORDER BY timestamp DESC"
	query += fmt.Sprintf(" LIMIT $%d", argPos)
	args = append(args, limit)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	logs := []models.ActivityLog{}
	for rows.Next() {
		var log models.ActivityLog
		var detailsJSON []byte

		err := rows.Scan(
			&log.ID,
			&log.Timestamp,
			&log.ActivityType,
			&log.Platform,
			&log.Message,
			&detailsJSON,
			&log.SourceCount,
			&log.DurationMs,
		)
		if err != nil {
			return nil, err
		}

		if len(detailsJSON) > 0 {
			if err := json.Unmarshal(detailsJSON, &log.Details); err != nil {
				return nil, fmt.Errorf("failed to unmarshal details: %w", err)
			}
		}

		logs = append(logs, log)
	}

	return logs, rows.Err()
}

// DeleteOlderThan deletes activity logs older than the specified duration.
func (r *ActivityLogRepository) DeleteOlderThan(ctx context.Context, age time.Duration) (int64, error) {
	query := `DELETE FROM activity_logs WHERE timestamp < $1`
	cutoff := time.Now().Add(-age)

	result, err := r.db.ExecContext(ctx, query, cutoff)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}
