package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/STRATINT/stratint/internal/models"
)

// PostgresSourceRepository implements SourceRepository using PostgreSQL.
type PostgresSourceRepository struct {
	db *sql.DB
}

// NewPostgresSourceRepository creates a new PostgreSQL source repository.
func NewPostgresSourceRepository(db *sql.DB) *PostgresSourceRepository {
	return &PostgresSourceRepository{db: db}
}

// StoreRaw saves a raw source to the repository (alias for Store).
func (r *PostgresSourceRepository) StoreRaw(ctx context.Context, source models.Source) error {
	return r.Store(ctx, source)
}

// Exists checks if a source with the given ID exists.
func (r *PostgresSourceRepository) Exists(ctx context.Context, id string) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM sources WHERE id = $1)", id).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check source existence: %w", err)
	}
	return exists, nil
}

// GetByURL retrieves a source by its URL.
func (r *PostgresSourceRepository) GetByURL(ctx context.Context, url string) (*models.Source, error) {
	query := `
		SELECT id, type, url, title, author, author_id, published_at, retrieved_at,
		       raw_content, content_hash, credibility, metadata, created_at
		FROM sources
		WHERE url = $1
		ORDER BY created_at DESC
		LIMIT 1
	`

	var source models.Source
	var metadataJSON []byte

	err := r.db.QueryRowContext(ctx, query, url).Scan(
		&source.ID,
		&source.Type,
		&source.URL,
		&source.Title,
		&source.Author,
		&source.AuthorID,
		&source.PublishedAt,
		&source.RetrievedAt,
		&source.RawContent,
		&source.ContentHash,
		&source.Credibility,
		&metadataJSON,
		&source.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query source by URL: %w", err)
	}

	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &source.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &source, nil
}

// GetByTitleAndURL checks if a source with the same title and URL exists.
func (r *PostgresSourceRepository) GetByTitleAndURL(ctx context.Context, title, url string) (*models.Source, error) {
	query := `
		SELECT id, type, url, title, author, author_id, published_at, retrieved_at,
		       raw_content, content_hash, credibility, metadata, created_at
		FROM sources
		WHERE title = $1 AND url = $2
		ORDER BY created_at DESC
		LIMIT 1
	`

	var source models.Source
	var metadataJSON []byte

	err := r.db.QueryRowContext(ctx, query, title, url).Scan(
		&source.ID,
		&source.Type,
		&source.URL,
		&source.Title,
		&source.Author,
		&source.AuthorID,
		&source.PublishedAt,
		&source.RetrievedAt,
		&source.RawContent,
		&source.ContentHash,
		&source.Credibility,
		&metadataJSON,
		&source.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query source by title and URL: %w", err)
	}

	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &source.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &source, nil
}

// Store inserts a single source into the database.
func (r *PostgresSourceRepository) Store(ctx context.Context, source models.Source) error {
	metadataJSON, err := json.Marshal(source.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// First try to insert. If there's a conflict on URL (duplicate article), ignore it.
	// If there's a conflict on ID (same source being re-inserted), update the fields.
	// Note: PostgreSQL doesn't support multiple ON CONFLICT clauses, so we prioritize URL uniqueness.
	// The unique URL constraint will prevent duplicate sources with different IDs.
	query := `
		INSERT INTO sources (
			id, type, url, title, author, author_id, published_at, retrieved_at,
			raw_content, content_hash, credibility, metadata,
			scrape_status, scrape_error, scraped_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		ON CONFLICT (id) DO UPDATE SET
			type = EXCLUDED.type,
			url = EXCLUDED.url,
			title = EXCLUDED.title,
			author = EXCLUDED.author,
			author_id = EXCLUDED.author_id,
			published_at = EXCLUDED.published_at,
			retrieved_at = EXCLUDED.retrieved_at,
			raw_content = EXCLUDED.raw_content,
			content_hash = EXCLUDED.content_hash,
			credibility = EXCLUDED.credibility,
			metadata = EXCLUDED.metadata,
			scrape_status = EXCLUDED.scrape_status,
			scrape_error = EXCLUDED.scrape_error,
			scraped_at = EXCLUDED.scraped_at
	`

	_, err = r.db.ExecContext(ctx, query,
		source.ID,
		source.Type,
		source.URL,
		source.Title,
		source.Author,
		source.AuthorID,
		source.PublishedAt,
		source.RetrievedAt,
		source.RawContent,
		source.ContentHash,
		source.Credibility,
		metadataJSON,
		source.ScrapeStatus,
		source.ScrapeError,
		source.ScrapedAt,
		source.CreatedAt,
	)

	if err != nil {
		// Check if this is a unique constraint violation on URL
		// This means we're trying to insert a duplicate URL with a different ID - ignore it
		if strings.Contains(err.Error(), "idx_sources_url_unique") ||
			strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			// Silently ignore URL duplicates
			return nil
		}
		return fmt.Errorf("failed to store source: %w", err)
	}

	return nil
}

// StoreBatch inserts multiple sources in a single transaction.
func (r *PostgresSourceRepository) StoreBatch(ctx context.Context, sources []models.Source) error {
	if len(sources) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO sources (
			id, type, url, title, author, author_id, published_at, retrieved_at,
			raw_content, content_hash, credibility, metadata,
			scrape_status, scrape_error, scraped_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		ON CONFLICT (id) DO NOTHING
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, source := range sources {
		metadataJSON, err := json.Marshal(source.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}

		_, err = stmt.ExecContext(ctx,
			source.ID,
			source.Type,
			source.URL,
			source.Title,
			source.Author,
			source.AuthorID,
			source.PublishedAt,
			source.RetrievedAt,
			source.RawContent,
			source.ContentHash,
			source.Credibility,
			metadataJSON,
			source.ScrapeStatus,
			source.ScrapeError,
			source.ScrapedAt,
			source.CreatedAt,
		)
		if err != nil {
			// Check if this is a unique constraint violation on URL - if so, skip it
			if strings.Contains(err.Error(), "idx_sources_url_unique") ||
				strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
				// Silently ignore URL duplicates and continue with the rest
				continue
			}
			return fmt.Errorf("failed to insert source %s: %w", source.ID, err)
		}
	}

	return tx.Commit()
}

// GetByID retrieves a source by its ID.
func (r *PostgresSourceRepository) GetByID(ctx context.Context, id string) (*models.Source, error) {
	query := `
		SELECT id, type, url, title, author, author_id, published_at, retrieved_at,
		       raw_content, content_hash, credibility, metadata, created_at
		FROM sources
		WHERE id = $1
	`

	var source models.Source
	var metadataJSON []byte

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&source.ID,
		&source.Type,
		&source.URL,
		&source.Title,
		&source.Author,
		&source.AuthorID,
		&source.PublishedAt,
		&source.RetrievedAt,
		&source.RawContent,
		&source.ContentHash,
		&source.Credibility,
		&metadataJSON,
		&source.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query source: %w", err)
	}

	// Deserialize metadata
	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &source.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &source, nil
}

// GetByContentHash retrieves a source by its content hash (for deduplication).
func (r *PostgresSourceRepository) GetByContentHash(ctx context.Context, hash string) (*models.Source, error) {
	query := `
		SELECT id, type, url, title, author, author_id, published_at, retrieved_at,
		       raw_content, content_hash, credibility, metadata, created_at
		FROM sources
		WHERE content_hash = $1
		ORDER BY created_at DESC
		LIMIT 1
	`

	var source models.Source
	var metadataJSON []byte

	err := r.db.QueryRowContext(ctx, query, hash).Scan(
		&source.ID,
		&source.Type,
		&source.URL,
		&source.Title,
		&source.Author,
		&source.AuthorID,
		&source.PublishedAt,
		&source.RetrievedAt,
		&source.RawContent,
		&source.ContentHash,
		&source.Credibility,
		&metadataJSON,
		&source.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query source by hash: %w", err)
	}

	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &source.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &source, nil
}

// ListRecent retrieves sources created within a time window.
func (r *PostgresSourceRepository) ListRecent(ctx context.Context, since time.Time, limit int) ([]models.Source, error) {
	query := `
		SELECT id, type, url, title, author, author_id, published_at, retrieved_at,
		       raw_content, content_hash, credibility, metadata, created_at, scrape_status, scrape_error, scraped_at
		FROM sources
		WHERE created_at >= $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := r.db.QueryContext(ctx, query, since, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent sources: %w", err)
	}
	defer rows.Close()

	sources := []models.Source{}
	for rows.Next() {
		var source models.Source
		var metadataJSON []byte
		var scrapeError sql.NullString
		var scrapedAt sql.NullTime

		err := rows.Scan(
			&source.ID,
			&source.Type,
			&source.URL,
			&source.Title,
			&source.Author,
			&source.AuthorID,
			&source.PublishedAt,
			&source.RetrievedAt,
			&source.RawContent,
			&source.ContentHash,
			&source.Credibility,
			&metadataJSON,
			&source.CreatedAt,
			&source.ScrapeStatus,
			&scrapeError,
			&scrapedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan source: %w", err)
		}

		// Handle nullable fields
		if scrapeError.Valid {
			source.ScrapeError = scrapeError.String
		}
		if scrapedAt.Valid {
			source.ScrapedAt = &scrapedAt.Time
		}

		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &source.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		sources = append(sources, source)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return sources, nil
}

// ListByType retrieves sources of a specific type.
func (r *PostgresSourceRepository) ListByType(ctx context.Context, sourceType models.SourceType, limit int) ([]models.Source, error) {
	query := `
		SELECT id, type, url, title, author, author_id, published_at, retrieved_at,
		       raw_content, content_hash, credibility, metadata, created_at
		FROM sources
		WHERE type = $1
		ORDER BY published_at DESC
		LIMIT $2
	`

	rows, err := r.db.QueryContext(ctx, query, sourceType, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query sources by type: %w", err)
	}
	defer rows.Close()

	sources := []models.Source{}
	for rows.Next() {
		var source models.Source
		var metadataJSON []byte

		err := rows.Scan(
			&source.ID,
			&source.Type,
			&source.URL,
			&source.Title,
			&source.Author,
			&source.AuthorID,
			&source.PublishedAt,
			&source.RetrievedAt,
			&source.RawContent,
			&source.ContentHash,
			&source.Credibility,
			&metadataJSON,
			&source.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan source: %w", err)
		}

		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &source.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		sources = append(sources, source)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return sources, nil
}

// Count returns the total number of sources.
func (r *PostgresSourceRepository) Count(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sources").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count sources: %w", err)
	}
	return count, nil
}

// CountByType returns the count of sources by type.
func (r *PostgresSourceRepository) CountByType(ctx context.Context, sourceType models.SourceType) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sources WHERE type = $1", sourceType).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count sources by type: %w", err)
	}
	return count, nil
}

// DeleteOlderThan removes sources older than the specified time (for retention policies).
func (r *PostgresSourceRepository) DeleteOlderThan(ctx context.Context, before time.Time) (int, error) {
	result, err := r.db.ExecContext(ctx, "DELETE FROM sources WHERE published_at < $1", before)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old sources: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return int(rows), nil
}

// GetByStatus retrieves sources with a specific scrape status.
func (r *PostgresSourceRepository) GetByStatus(ctx context.Context, status models.ScrapeStatus, limit int) ([]models.Source, error) {
	query := `
		SELECT id, type, url, title, author, author_id, published_at, retrieved_at,
		       raw_content, content_hash, credibility, metadata,
		       scrape_status, scrape_error, scraped_at, created_at
		FROM sources
		WHERE scrape_status = $1
		ORDER BY created_at ASC
		LIMIT $2
	`

	rows, err := r.db.QueryContext(ctx, query, status, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query sources by status: %w", err)
	}
	defer rows.Close()

	var sources []models.Source
	for rows.Next() {
		source, err := r.scanSource(rows)
		if err != nil {
			return nil, err
		}
		sources = append(sources, *source)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return sources, nil
}

// Update modifies an existing source in the database.
func (r *PostgresSourceRepository) Update(ctx context.Context, source models.Source) error {
	metadataJSON, err := json.Marshal(source.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		UPDATE sources
		SET type = $2,
		    url = $3,
		    title = $4,
		    author = $5,
		    author_id = $6,
		    published_at = $7,
		    retrieved_at = $8,
		    raw_content = $9,
		    content_hash = $10,
		    credibility = $11,
		    metadata = $12,
		    scrape_status = $13,
		    scrape_error = $14,
		    scraped_at = $15
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query,
		source.ID,
		source.Type,
		source.URL,
		source.Title,
		source.Author,
		source.AuthorID,
		source.PublishedAt,
		source.RetrievedAt,
		source.RawContent,
		source.ContentHash,
		source.Credibility,
		metadataJSON,
		source.ScrapeStatus,
		source.ScrapeError,
		source.ScrapedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to update source: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("source with id %s not found", source.ID)
	}

	return nil
}

// Delete removes a source by its ID from the database.
func (r *PostgresSourceRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM sources WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete source: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("source with id %s not found", id)
	}

	return nil
}

// scanSource is a helper to consistently scan source rows including new scrape fields.
func (r *PostgresSourceRepository) scanSource(scanner interface {
	Scan(dest ...interface{}) error
}) (*models.Source, error) {
	var source models.Source
	var metadataJSON []byte
	var scrapeError sql.NullString
	var scrapedAt sql.NullTime

	err := scanner.Scan(
		&source.ID,
		&source.Type,
		&source.URL,
		&source.Title,
		&source.Author,
		&source.AuthorID,
		&source.PublishedAt,
		&source.RetrievedAt,
		&source.RawContent,
		&source.ContentHash,
		&source.Credibility,
		&metadataJSON,
		&source.ScrapeStatus,
		&scrapeError,
		&scrapedAt,
		&source.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to scan source: %w", err)
	}

	// Handle nullable fields
	if scrapeError.Valid {
		source.ScrapeError = scrapeError.String
	}
	if scrapedAt.Valid {
		source.ScrapedAt = &scrapedAt.Time
	}

	// Deserialize metadata
	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &source.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &source, nil
}

// ClaimSourcesForEnrichment atomically claims sources for enrichment processing.
// This prevents race conditions when multiple Cloud Run instances are running.
// Returns the list of sources that were successfully claimed by THIS call.
func (r *PostgresSourceRepository) ClaimSourcesForEnrichment(ctx context.Context, limit int, staleAfter time.Duration) ([]models.Source, error) {
	// Atomic UPDATE + RETURNING to claim sources
	// Claim sources that are:
	// 1. Scrape status is 'completed'
	// 2. Enrichment status is 'pending' OR
	// 3. Enrichment status is 'enriching' but claim is stale (>15 min old, indicating crashed worker)
	query := `
		UPDATE sources
		SET enrichment_status = 'enriching',
		    enrichment_claimed_at = NOW(),
		    enrichment_error = NULL
		WHERE id IN (
			SELECT id FROM sources
			WHERE scrape_status = 'completed'
			  AND raw_content != ''
			  AND (
			    enrichment_status = 'pending'
			    OR (enrichment_status = 'enriching' AND enrichment_claimed_at < NOW() - $2::interval)
			  )
			ORDER BY created_at ASC
			LIMIT $1
			FOR UPDATE SKIP LOCKED  -- Critical: prevents blocking and ensures atomicity
		)
		RETURNING id, type, url, title, author, author_id, published_at, retrieved_at,
		          raw_content, content_hash, credibility, metadata,
		          scrape_status, scrape_error, scraped_at,
		          enrichment_status, enrichment_error, enriched_at, enrichment_claimed_at,
		          created_at
	`

	staleInterval := fmt.Sprintf("%d minutes", int(staleAfter.Minutes()))
	rows, err := r.db.QueryContext(ctx, query, limit, staleInterval)
	if err != nil {
		return nil, fmt.Errorf("failed to claim sources for enrichment: %w", err)
	}
	defer rows.Close()

	var sources []models.Source
	for rows.Next() {
		var source models.Source
		var metadataJSON []byte
		var scrapeError, enrichmentError sql.NullString
		var scrapedAt, enrichedAt, enrichmentClaimedAt sql.NullTime

		err := rows.Scan(
			&source.ID,
			&source.Type,
			&source.URL,
			&source.Title,
			&source.Author,
			&source.AuthorID,
			&source.PublishedAt,
			&source.RetrievedAt,
			&source.RawContent,
			&source.ContentHash,
			&source.Credibility,
			&metadataJSON,
			&source.ScrapeStatus,
			&scrapeError,
			&scrapedAt,
			&source.EnrichmentStatus,
			&enrichmentError,
			&enrichedAt,
			&enrichmentClaimedAt,
			&source.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan claimed source: %w", err)
		}

		// Handle nullable fields
		if scrapeError.Valid {
			source.ScrapeError = scrapeError.String
		}
		if scrapedAt.Valid {
			source.ScrapedAt = &scrapedAt.Time
		}
		if enrichmentError.Valid {
			source.EnrichmentError = enrichmentError.String
		}
		if enrichedAt.Valid {
			source.EnrichedAt = &enrichedAt.Time
		}
		if enrichmentClaimedAt.Valid {
			source.EnrichmentClaimedAt = &enrichmentClaimedAt.Time
		}

		// Deserialize metadata
		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &source.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		sources = append(sources, source)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating claimed sources: %w", err)
	}

	return sources, nil
}

// UpdateEnrichmentStatus updates the enrichment status of a source.
func (r *PostgresSourceRepository) UpdateEnrichmentStatus(ctx context.Context, sourceID string, status models.EnrichmentStatus, errorMsg string) error {
	var enrichedAt *time.Time
	if status == models.EnrichmentStatusCompleted {
		now := time.Now()
		enrichedAt = &now
	}

	query := `
		UPDATE sources
		SET enrichment_status = $1,
		    enrichment_error = $2,
		    enriched_at = $3,
		    enrichment_claimed_at = NULL
		WHERE id = $4
	`

	_, err := r.db.ExecContext(ctx, query, status, errorMsg, enrichedAt, sourceID)
	if err != nil {
		return fmt.Errorf("failed to update enrichment status: %w", err)
	}

	return nil
}

// SetEventID sets the event_id for a source after enrichment.
func (r *PostgresSourceRepository) SetEventID(ctx context.Context, sourceID, eventID string) error {
	query := `
		UPDATE sources
		SET event_id = $1
		WHERE id = $2
	`

	_, err := r.db.ExecContext(ctx, query, eventID, sourceID)
	if err != nil {
		return fmt.Errorf("failed to set event_id: %w", err)
	}

	return nil
}

// GetRecentEnrichments retrieves recent sources with their enrichment status and event IDs.
func (r *PostgresSourceRepository) GetRecentEnrichments(ctx context.Context, limit int) ([]models.Source, error) {
	query := `
		SELECT id, type, url, title, author, author_id, published_at, retrieved_at,
		       raw_content, content_hash, credibility, metadata,
		       scrape_status, scrape_error, scraped_at,
		       enrichment_status, enrichment_error, enriched_at, enrichment_claimed_at,
		       event_id, created_at
		FROM sources
		WHERE enrichment_status != 'pending'
		ORDER BY enriched_at DESC NULLS LAST, created_at DESC
		LIMIT $1
	`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent enrichments: %w", err)
	}
	defer rows.Close()

	var sources []models.Source
	for rows.Next() {
		var source models.Source
		var metadataJSON []byte
		var scrapeError, enrichmentError sql.NullString
		var scrapedAt, enrichedAt, enrichmentClaimedAt sql.NullTime
		var eventID sql.NullString

		err := rows.Scan(
			&source.ID,
			&source.Type,
			&source.URL,
			&source.Title,
			&source.Author,
			&source.AuthorID,
			&source.PublishedAt,
			&source.RetrievedAt,
			&source.RawContent,
			&source.ContentHash,
			&source.Credibility,
			&metadataJSON,
			&source.ScrapeStatus,
			&scrapeError,
			&scrapedAt,
			&source.EnrichmentStatus,
			&enrichmentError,
			&enrichedAt,
			&enrichmentClaimedAt,
			&eventID,
			&source.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan source: %w", err)
		}

		// Handle nullable fields
		if scrapeError.Valid {
			source.ScrapeError = scrapeError.String
		}
		if scrapedAt.Valid {
			source.ScrapedAt = &scrapedAt.Time
		}
		if enrichmentError.Valid {
			source.EnrichmentError = enrichmentError.String
		}
		if enrichedAt.Valid {
			source.EnrichedAt = &enrichedAt.Time
		}
		if enrichmentClaimedAt.Valid {
			source.EnrichmentClaimedAt = &enrichmentClaimedAt.Time
		}
		if eventID.Valid {
			source.EventID = eventID.String
		}

		// Deserialize metadata
		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &source.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		sources = append(sources, source)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return sources, nil
}
