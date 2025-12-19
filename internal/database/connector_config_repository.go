package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/STRATINT/stratint/internal/models"
)

// ConnectorConfigRepository manages connector configuration in the database.
type ConnectorConfigRepository struct {
	db *sql.DB
}

// NewConnectorConfigRepository creates a new repository for connector configuration.
func NewConnectorConfigRepository(db *sql.DB) *ConnectorConfigRepository {
	return &ConnectorConfigRepository{db: db}
}

// Get retrieves the configuration for a specific connector.
func (r *ConnectorConfigRepository) Get(ctx context.Context, connectorID string) (*models.ConnectorConfig, error) {
	query := `
		SELECT id, enabled, config, updated_at, created_at
		FROM connector_config
		WHERE id = $1
	`

	config := &models.ConnectorConfig{}
	var configJSON []byte

	err := r.db.QueryRowContext(ctx, query, connectorID).Scan(
		&config.ID,
		&config.Enabled,
		&configJSON,
		&config.UpdatedAt,
		&config.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("connector configuration not found: %s", connectorID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get connector configuration: %w", err)
	}

	// Parse JSON config
	if err := json.Unmarshal(configJSON, &config.Config); err != nil {
		return nil, fmt.Errorf("failed to parse connector config: %w", err)
	}

	return config, nil
}

// GetAll retrieves all connector configurations.
func (r *ConnectorConfigRepository) GetAll(ctx context.Context) ([]models.ConnectorConfig, error) {
	query := `
		SELECT id, enabled, config, updated_at, created_at
		FROM connector_config
		ORDER BY id
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query connector configs: %w", err)
	}
	defer rows.Close()

	var configs []models.ConnectorConfig
	for rows.Next() {
		var config models.ConnectorConfig
		var configJSON []byte

		if err := rows.Scan(
			&config.ID,
			&config.Enabled,
			&configJSON,
			&config.UpdatedAt,
			&config.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan connector config: %w", err)
		}

		// Parse JSON config
		if err := json.Unmarshal(configJSON, &config.Config); err != nil {
			return nil, fmt.Errorf("failed to parse connector config: %w", err)
		}

		configs = append(configs, config)
	}

	return configs, nil
}

// Update updates the configuration for a specific connector.
func (r *ConnectorConfigRepository) Update(ctx context.Context, connectorID string, enabled *bool, config map[string]string) (*models.ConnectorConfig, error) {
	// Build dynamic update query
	query := `UPDATE connector_config SET updated_at = $1`
	args := []interface{}{time.Now()}
	argCount := 1

	if enabled != nil {
		argCount++
		query += fmt.Sprintf(", enabled = $%d", argCount)
		args = append(args, *enabled)
	}

	if config != nil {
		argCount++
		configJSON, err := json.Marshal(config)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal config: %w", err)
		}
		query += fmt.Sprintf(", config = $%d", argCount)
		args = append(args, configJSON)
	}

	argCount++
	query += fmt.Sprintf(` WHERE id = $%d`, argCount)
	args = append(args, connectorID)

	query += ` RETURNING id, enabled, config, updated_at, created_at`

	result := &models.ConnectorConfig{}
	var configJSON []byte

	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&result.ID,
		&result.Enabled,
		&configJSON,
		&result.UpdatedAt,
		&result.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to update connector configuration: %w", err)
	}

	// Parse JSON config
	if err := json.Unmarshal(configJSON, &result.Config); err != nil {
		return nil, fmt.Errorf("failed to parse connector config: %w", err)
	}

	return result, nil
}

// SetEnabled enables or disables a connector.
func (r *ConnectorConfigRepository) SetEnabled(ctx context.Context, connectorID string, enabled bool) error {
	query := `
		UPDATE connector_config
		SET enabled = $1, updated_at = $2
		WHERE id = $3
	`

	result, err := r.db.ExecContext(ctx, query, enabled, time.Now(), connectorID)
	if err != nil {
		return fmt.Errorf("failed to update connector enabled status: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("connector not found: %s", connectorID)
	}

	return nil
}
