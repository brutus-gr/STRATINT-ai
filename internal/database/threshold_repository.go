package database

import (
	"context"
	"database/sql"
	"time"

	"github.com/STRATINT/stratint/internal/models"
)

// ThresholdRepository handles threshold configuration storage.
type ThresholdRepository struct {
	db *sql.DB
}

// NewThresholdRepository creates a new threshold repository.
func NewThresholdRepository(db *sql.DB) *ThresholdRepository {
	return &ThresholdRepository{db: db}
}

// Get retrieves the current threshold configuration.
func (r *ThresholdRepository) Get(ctx context.Context) (*models.ThresholdConfig, error) {
	query := `
		SELECT min_confidence, min_magnitude, max_source_age_hours, updated_at
		FROM threshold_config
		ORDER BY id DESC
		LIMIT 1
	`

	var config models.ThresholdConfig
	err := r.db.QueryRowContext(ctx, query).Scan(
		&config.MinConfidence,
		&config.MinMagnitude,
		&config.MaxSourceAgeHours,
		&config.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

// Update updates the threshold configuration.
func (r *ThresholdRepository) Update(ctx context.Context, config *models.ThresholdConfig) error {
	query := `
		UPDATE threshold_config
		SET min_confidence = $1,
		    min_magnitude = $2,
		    max_source_age_hours = $3,
		    updated_at = $4
		WHERE id = (SELECT id FROM threshold_config ORDER BY id DESC LIMIT 1)
	`

	config.UpdatedAt = time.Now()

	_, err := r.db.ExecContext(ctx, query,
		config.MinConfidence,
		config.MinMagnitude,
		config.MaxSourceAgeHours,
		config.UpdatedAt,
	)

	return err
}
