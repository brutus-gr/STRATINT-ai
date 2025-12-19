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

// IngestionErrorRepository defines the interface for storing and retrieving ingestion errors.
type IngestionErrorRepository interface {
	// Store saves an ingestion error to the repository.
	Store(ctx context.Context, err models.IngestionError) error

	// List retrieves ingestion errors with optional filtering.
	List(ctx context.Context, limit int, unresolvedOnly bool) ([]models.IngestionError, error)

	// GetByID retrieves an error by its ID.
	GetByID(ctx context.Context, id string) (*models.IngestionError, error)

	// MarkResolved marks an error as resolved.
	MarkResolved(ctx context.Context, id string) error

	// Delete removes an error from the repository.
	Delete(ctx context.Context, id string) error

	// CountUnresolved returns the count of unresolved errors.
	CountUnresolved(ctx context.Context) (int, error)
}

// PostgresIngestionErrorRepository implements the IngestionErrorRepository using PostgreSQL.
type PostgresIngestionErrorRepository struct {
	db *sql.DB
}

// NewPostgresIngestionErrorRepository creates a new PostgreSQL-based ingestion error repository.
func NewPostgresIngestionErrorRepository(db *sql.DB) *PostgresIngestionErrorRepository {
	return &PostgresIngestionErrorRepository{db: db}
}

// Store saves an ingestion error to the database.
func (r *PostgresIngestionErrorRepository) Store(ctx context.Context, err models.IngestionError) error {
	// Generate ID if not provided
	if err.ID == "" {
		err.ID = uuid.New().String()
	}

	// Set created_at if not provided
	if err.CreatedAt.IsZero() {
		err.CreatedAt = time.Now()
	}

	query := `
		INSERT INTO ingestion_errors (id, platform, error_type, url, error_msg, metadata, created_at, resolved, resolved_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id) DO UPDATE SET
			error_msg = EXCLUDED.error_msg,
			metadata = EXCLUDED.metadata,
			created_at = EXCLUDED.created_at
	`

	_, execErr := r.db.ExecContext(ctx, query,
		err.ID,
		err.Platform,
		err.ErrorType,
		err.URL,
		err.ErrorMsg,
		err.Metadata,
		err.CreatedAt,
		err.Resolved,
		err.ResolvedAt,
	)

	return execErr
}

// List retrieves ingestion errors with optional filtering.
func (r *PostgresIngestionErrorRepository) List(ctx context.Context, limit int, unresolvedOnly bool) ([]models.IngestionError, error) {
	query := `
		SELECT id, platform, error_type, url, error_msg, metadata, created_at, resolved, resolved_at
		FROM ingestion_errors
	`

	if unresolvedOnly {
		query += " WHERE resolved = FALSE"
	}

	query += " ORDER BY created_at DESC LIMIT $1"

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query ingestion errors: %w", err)
	}
	defer rows.Close()

	var errors []models.IngestionError
	for rows.Next() {
		var e models.IngestionError
		var metadata sql.NullString
		var resolvedAt sql.NullTime

		if err := rows.Scan(
			&e.ID,
			&e.Platform,
			&e.ErrorType,
			&e.URL,
			&e.ErrorMsg,
			&metadata,
			&e.CreatedAt,
			&e.Resolved,
			&resolvedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan ingestion error: %w", err)
		}

		if metadata.Valid {
			e.Metadata = metadata.String
		}
		if resolvedAt.Valid {
			e.ResolvedAt = &resolvedAt.Time
		}

		errors = append(errors, e)
	}

	return errors, rows.Err()
}

// GetByID retrieves an error by its ID.
func (r *PostgresIngestionErrorRepository) GetByID(ctx context.Context, id string) (*models.IngestionError, error) {
	query := `
		SELECT id, platform, error_type, url, error_msg, metadata, created_at, resolved, resolved_at
		FROM ingestion_errors
		WHERE id = $1
	`

	var e models.IngestionError
	var metadata sql.NullString
	var resolvedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&e.ID,
		&e.Platform,
		&e.ErrorType,
		&e.URL,
		&e.ErrorMsg,
		&metadata,
		&e.CreatedAt,
		&e.Resolved,
		&resolvedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get ingestion error: %w", err)
	}

	if metadata.Valid {
		e.Metadata = metadata.String
	}
	if resolvedAt.Valid {
		e.ResolvedAt = &resolvedAt.Time
	}

	return &e, nil
}

// MarkResolved marks an error as resolved.
func (r *PostgresIngestionErrorRepository) MarkResolved(ctx context.Context, id string) error {
	query := `
		UPDATE ingestion_errors
		SET resolved = TRUE, resolved_at = NOW()
		WHERE id = $1
	`

	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// Delete removes an error from the repository.
func (r *PostgresIngestionErrorRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM ingestion_errors WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// CountUnresolved returns the count of unresolved errors.
func (r *PostgresIngestionErrorRepository) CountUnresolved(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM ingestion_errors WHERE resolved = FALSE`

	var count int
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count unresolved errors: %w", err)
	}

	return count, nil
}

// Helper function to create error metadata JSON
func CreateErrorMetadata(data map[string]interface{}) (string, error) {
	if data == nil {
		return "", nil
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	return string(jsonData), nil
}
