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

// PostgresEventRepository implements EventRepository using PostgreSQL.
type PostgresEventRepository struct {
	db *sql.DB
}

// NewPostgresEventRepository creates a new PostgreSQL event repository.
func NewPostgresEventRepository(db *sql.DB) *PostgresEventRepository {
	return &PostgresEventRepository{db: db}
}

// Create inserts a new event into the database.
func (r *PostgresEventRepository) Create(ctx context.Context, event models.Event) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Serialize confidence to JSON
	confidenceJSON, err := json.Marshal(event.Confidence)
	if err != nil {
		return fmt.Errorf("failed to marshal confidence: %w", err)
	}

	// Insert event with location fields
	query := `
		INSERT INTO events (
			id, timestamp, title, summary, raw_content, magnitude, confidence,
			category, status, tags, location, location_country, location_city, location_region,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, ST_SetSRID(ST_MakePoint($11, $12), 4326), $13, $14, $15, $16, $17)
	`

	var lon, lat *float64
	var country, city, region *string
	if event.Location != nil {
		lon = &event.Location.Longitude
		lat = &event.Location.Latitude
		if event.Location.Country != "" {
			country = &event.Location.Country
		}
		if event.Location.City != "" {
			city = &event.Location.City
		}
		if event.Location.Region != "" {
			region = &event.Location.Region
		}
	}

	_, err = tx.ExecContext(ctx, query,
		event.ID,
		event.Timestamp,
		event.Title,
		event.Summary,
		event.RawContent,
		event.Magnitude,
		confidenceJSON,
		event.Category,
		event.Status,
		pq.Array(event.Tags),
		lon,
		lat,
		country,
		city,
		region,
		event.CreatedAt,
		event.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert event: %w", err)
	}

	// Insert event-source relationships
	if err := r.insertEventSources(ctx, tx, event.ID, event.Sources); err != nil {
		return err
	}

	// Insert event-entity relationships
	if err := r.insertEventEntities(ctx, tx, event.ID, event.Entities); err != nil {
		return err
	}

	return tx.Commit()
}

// GetByID retrieves an event by its ID.
func (r *PostgresEventRepository) GetByID(ctx context.Context, id string) (*models.Event, error) {
	// Query with location text fields (migration 011)
	query := `
		SELECT id, timestamp, title, summary, raw_content, magnitude, confidence,
		       category, status, tags, ST_X(location::geometry), ST_Y(location::geometry),
		       location_country, location_city, location_region,
		       created_at, updated_at
		FROM events
		WHERE id = $1
	`

	var event models.Event
	var confidenceJSON []byte
	var lon, lat sql.NullFloat64
	var locationCountry, locationCity, locationRegion sql.NullString
	var tags pq.StringArray

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&event.ID,
		&event.Timestamp,
		&event.Title,
		&event.Summary,
		&event.RawContent,
		&event.Magnitude,
		&confidenceJSON,
		&event.Category,
		&event.Status,
		&tags,
		&lon,
		&lat,
		&locationCountry,
		&locationCity,
		&locationRegion,
		&event.CreatedAt,
		&event.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query event: %w", err)
	}

	// Deserialize confidence
	if err := json.Unmarshal(confidenceJSON, &event.Confidence); err != nil {
		return nil, fmt.Errorf("failed to unmarshal confidence: %w", err)
	}

	event.Tags = tags

	// Set location if any location data is present
	if lon.Valid || lat.Valid || locationCountry.Valid || locationCity.Valid || locationRegion.Valid {
		event.Location = &models.Location{}
		if lon.Valid {
			event.Location.Longitude = lon.Float64
		}
		if lat.Valid {
			event.Location.Latitude = lat.Float64
		}
		if locationCountry.Valid {
			event.Location.Country = locationCountry.String
		}
		if locationCity.Valid {
			event.Location.City = locationCity.String
		}
		if locationRegion.Valid {
			event.Location.Region = locationRegion.String
		}
	}

	// Load sources and entities
	if err := r.loadEventRelations(ctx, &event); err != nil {
		return nil, err
	}

	return &event, nil
}

// Update updates an existing event.
func (r *PostgresEventRepository) Update(ctx context.Context, event models.Event) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	confidenceJSON, err := json.Marshal(event.Confidence)
	if err != nil {
		return fmt.Errorf("failed to marshal confidence: %w", err)
	}

	query := `
		UPDATE events SET
			timestamp = $2, title = $3, summary = $4, raw_content = $5,
			magnitude = $6, confidence = $7, category = $8, status = $9,
			tags = $10, location = ST_SetSRID(ST_MakePoint($11, $12), 4326),
			updated_at = $13
		WHERE id = $1
	`

	var lon, lat *float64
	if event.Location != nil {
		lon = &event.Location.Longitude
		lat = &event.Location.Latitude
	}

	result, err := tx.ExecContext(ctx, query,
		event.ID,
		event.Timestamp,
		event.Title,
		event.Summary,
		event.RawContent,
		event.Magnitude,
		confidenceJSON,
		event.Category,
		event.Status,
		pq.Array(event.Tags),
		lon,
		lat,
		time.Now(),
	)
	if err != nil {
		return fmt.Errorf("failed to update event: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("event not found: %s", event.ID)
	}

	// Update relationships (delete old, insert new)
	if _, err := tx.ExecContext(ctx, "DELETE FROM event_sources WHERE event_id = $1", event.ID); err != nil {
		return fmt.Errorf("failed to delete event sources: %w", err)
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM event_entities WHERE event_id = $1", event.ID); err != nil {
		return fmt.Errorf("failed to delete event entities: %w", err)
	}

	if err := r.insertEventSources(ctx, tx, event.ID, event.Sources); err != nil {
		return err
	}
	if err := r.insertEventEntities(ctx, tx, event.ID, event.Entities); err != nil {
		return err
	}

	return tx.Commit()
}

// Delete removes an event from the database.
func (r *PostgresEventRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, "DELETE FROM events WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("failed to delete event: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("event not found: %s", id)
	}

	return nil
}

// UpdateStatus updates only the status of an event.
func (r *PostgresEventRepository) UpdateStatus(ctx context.Context, id string, status models.EventStatus) error {
	query := "UPDATE events SET status = $1, updated_at = $2 WHERE id = $3"
	result, err := r.db.ExecContext(ctx, query, status, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("event not found: %s", id)
	}

	return nil
}

// Query retrieves events based on filter criteria.
func (r *PostgresEventRepository) Query(ctx context.Context, query models.EventQuery) (*models.EventResponse, error) {
	// Validate query
	if err := query.Validate(); err != nil {
		return nil, err
	}

	// Build SQL query
	sqlQuery, args := r.buildQuery(query)

	// Execute count query
	countQuery := r.buildCountQuery(query)
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args[:len(args)-2]...).Scan(&total); err != nil {
		return nil, fmt.Errorf("failed to count events: %w", err)
	}

	// Execute main query
	rows, err := r.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query events: %w", err)
	}
	defer rows.Close()

	events := []models.Event{}
	for rows.Next() {
		var event models.Event
		var confidenceJSON []byte
		var lon, lat sql.NullFloat64
		var locationCountry, locationCity, locationRegion sql.NullString
		var tags pq.StringArray

		err := rows.Scan(
			&event.ID,
			&event.Timestamp,
			&event.Title,
			&event.Summary,
			&event.RawContent,
			&event.Magnitude,
			&confidenceJSON,
			&event.Category,
			&event.Status,
			&tags,
			&lon,
			&lat,
			&locationCountry,
			&locationCity,
			&locationRegion,
			&event.CreatedAt,
			&event.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}

		if err := json.Unmarshal(confidenceJSON, &event.Confidence); err != nil {
			return nil, fmt.Errorf("failed to unmarshal confidence: %w", err)
		}

		event.Tags = tags

		// Set location if any location data is present
		if lon.Valid || lat.Valid || locationCountry.Valid || locationCity.Valid || locationRegion.Valid {
			event.Location = &models.Location{}
			if lon.Valid {
				event.Location.Longitude = lon.Float64
			}
			if lat.Valid {
				event.Location.Latitude = lat.Float64
			}
			if locationCountry.Valid {
				event.Location.Country = locationCountry.String
			}
			if locationCity.Valid {
				event.Location.City = locationCity.String
			}
			if locationRegion.Valid {
				event.Location.Region = locationRegion.String
			}
		}

		// Load relations
		if err := r.loadEventRelations(ctx, &event); err != nil {
			return nil, err
		}

		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return &models.EventResponse{
		Events:  events,
		Page:    query.Page,
		Limit:   query.Limit,
		Total:   total,
		HasMore: (query.Page * query.Limit) < total,
		Query:   query.SearchQuery,
	}, nil
}

// buildQuery constructs the SQL query from EventQuery.
func (r *PostgresEventRepository) buildQuery(q models.EventQuery) (string, []interface{}) {
	args := []interface{}{}
	argIdx := 1
	conditions := []string{}

	// Status filter (default to published)
	if q.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *q.Status)
		argIdx++
	} else {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, models.EventStatusPublished)
		argIdx++
	}

	// Full-text search
	if q.SearchQuery != "" {
		conditions = append(conditions, fmt.Sprintf("to_tsvector('english', title || ' ' || summary) @@ plainto_tsquery('english', $%d)", argIdx))
		args = append(args, q.SearchQuery)
		argIdx++
	}

	// Time range
	if q.SinceTimestamp != nil {
		conditions = append(conditions, fmt.Sprintf("timestamp >= $%d", argIdx))
		args = append(args, *q.SinceTimestamp)
		argIdx++
	}
	if q.UntilTimestamp != nil {
		conditions = append(conditions, fmt.Sprintf("timestamp <= $%d", argIdx))
		args = append(args, *q.UntilTimestamp)
		argIdx++
	}

	// Thresholds
	if q.MinMagnitude != nil {
		conditions = append(conditions, fmt.Sprintf("magnitude >= $%d", argIdx))
		args = append(args, *q.MinMagnitude)
		argIdx++
	}
	if q.MinConfidence != nil {
		conditions = append(conditions, fmt.Sprintf("(confidence->>'score')::DECIMAL >= $%d", argIdx))
		args = append(args, *q.MinConfidence)
		argIdx++
	}

	// Category filter
	if len(q.Categories) > 0 {
		conditions = append(conditions, fmt.Sprintf("category = ANY($%d)", argIdx))
		args = append(args, pq.Array(q.Categories))
		argIdx++
	}

	// Tags filter
	if len(q.Tags) > 0 {
		conditions = append(conditions, fmt.Sprintf("tags && $%d", argIdx))
		args = append(args, pq.Array(q.Tags))
		argIdx++
	}

	// Build WHERE clause
	whereClause := "WHERE " + strings.Join(conditions, " AND ")

	// Build ORDER BY clause
	orderBy := fmt.Sprintf("ORDER BY %s %s", q.SortBy, q.SortOrder)

	// Add LIMIT and OFFSET
	args = append(args, q.Limit, q.GetOffset())

	query := fmt.Sprintf(`
		SELECT id, timestamp, title, summary, raw_content, magnitude, confidence,
		       category, status, tags, ST_X(location::geometry), ST_Y(location::geometry),
		       location_country, location_city, location_region,
		       created_at, updated_at
		FROM events
		%s
		%s
		LIMIT $%d OFFSET $%d
	`, whereClause, orderBy, argIdx, argIdx+1)

	return query, args
}

// buildCountQuery constructs the count query.
func (r *PostgresEventRepository) buildCountQuery(q models.EventQuery) string {
	conditions := []string{}
	argIdx := 1

	if q.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		argIdx++
	} else {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		argIdx++
	}

	if q.SearchQuery != "" {
		conditions = append(conditions, fmt.Sprintf("to_tsvector('english', title || ' ' || summary) @@ plainto_tsquery('english', $%d)", argIdx))
		argIdx++
	}

	if q.SinceTimestamp != nil {
		conditions = append(conditions, fmt.Sprintf("timestamp >= $%d", argIdx))
		argIdx++
	}
	if q.UntilTimestamp != nil {
		conditions = append(conditions, fmt.Sprintf("timestamp <= $%d", argIdx))
		argIdx++
	}

	if q.MinMagnitude != nil {
		conditions = append(conditions, fmt.Sprintf("magnitude >= $%d", argIdx))
		argIdx++
	}
	if q.MinConfidence != nil {
		conditions = append(conditions, fmt.Sprintf("(confidence->>'score')::DECIMAL >= $%d", argIdx))
		argIdx++
	}

	if len(q.Categories) > 0 {
		conditions = append(conditions, fmt.Sprintf("category = ANY($%d)", argIdx))
		argIdx++
	}

	if len(q.Tags) > 0 {
		conditions = append(conditions, fmt.Sprintf("tags && $%d", argIdx))
		argIdx++
	}

	whereClause := "WHERE " + strings.Join(conditions, " AND ")

	return fmt.Sprintf("SELECT COUNT(*) FROM events %s", whereClause)
}

// Helper functions

func (r *PostgresEventRepository) insertEventSources(ctx context.Context, tx *sql.Tx, eventID string, sources []models.Source) error {
	if len(sources) == 0 {
		return nil
	}

	for _, source := range sources {
		_, err := tx.ExecContext(ctx,
			"INSERT INTO event_sources (event_id, source_id) VALUES ($1, $2) ON CONFLICT DO NOTHING",
			eventID, source.ID,
		)
		if err != nil {
			return fmt.Errorf("failed to insert event source: %w", err)
		}
	}

	return nil
}

func (r *PostgresEventRepository) insertEventEntities(ctx context.Context, tx *sql.Tx, eventID string, entities []models.Entity) error {
	if len(entities) == 0 {
		return nil
	}

	for _, entity := range entities {
		// First, ensure the entity exists in the entities table
		attrsJSON, err := json.Marshal(entity.Attributes)
		if err != nil {
			return fmt.Errorf("failed to marshal entity attributes: %w", err)
		}

		_, err = tx.ExecContext(ctx, `
			INSERT INTO entities (id, type, name, normalized_name, confidence, attributes, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (id) DO UPDATE SET
				confidence = EXCLUDED.confidence,
				attributes = EXCLUDED.attributes
		`, entity.ID, entity.Type, entity.Name, entity.NormalizedName, entity.Confidence, attrsJSON, time.Now())
		if err != nil {
			return fmt.Errorf("failed to insert/update entity: %w", err)
		}

		// Then insert the event-entity relationship
		_, err = tx.ExecContext(ctx,
			"INSERT INTO event_entities (event_id, entity_id) VALUES ($1, $2) ON CONFLICT DO NOTHING",
			eventID, entity.ID,
		)
		if err != nil {
			return fmt.Errorf("failed to insert event entity relationship: %w", err)
		}
	}

	return nil
}

func (r *PostgresEventRepository) loadEventRelations(ctx context.Context, event *models.Event) error {
	// Load sources
	sourcesQuery := `
		SELECT s.id, s.type, s.url, s.author, s.published_at, s.retrieved_at,
		       s.raw_content, s.content_hash, s.credibility, s.metadata
		FROM sources s
		JOIN event_sources es ON s.id = es.source_id
		WHERE es.event_id = $1
	`

	rows, err := r.db.QueryContext(ctx, sourcesQuery, event.ID)
	if err != nil {
		return fmt.Errorf("failed to load sources: %w", err)
	}
	defer rows.Close()

	event.Sources = []models.Source{}
	for rows.Next() {
		var source models.Source
		var metadataJSON []byte

		err := rows.Scan(
			&source.ID,
			&source.Type,
			&source.URL,
			&source.Author,
			&source.PublishedAt,
			&source.RetrievedAt,
			&source.RawContent,
			&source.ContentHash,
			&source.Credibility,
			&metadataJSON,
		)
		if err != nil {
			return fmt.Errorf("failed to scan source: %w", err)
		}

		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &source.Metadata); err != nil {
				return fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		event.Sources = append(event.Sources, source)
	}

	// Load entities
	entitiesQuery := `
		SELECT e.id, e.type, e.name, e.normalized_name, e.confidence, e.attributes
		FROM entities e
		JOIN event_entities ee ON e.id = ee.entity_id
		WHERE ee.event_id = $1
	`

	rows, err = r.db.QueryContext(ctx, entitiesQuery, event.ID)
	if err != nil {
		return fmt.Errorf("failed to load entities: %w", err)
	}
	defer rows.Close()

	event.Entities = []models.Entity{}
	for rows.Next() {
		var entity models.Entity
		var attrsJSON []byte

		err := rows.Scan(
			&entity.ID,
			&entity.Type,
			&entity.Name,
			&entity.NormalizedName,
			&entity.Confidence,
			&attrsJSON,
		)
		if err != nil {
			return fmt.Errorf("failed to scan entity: %w", err)
		}

		if len(attrsJSON) > 0 {
			if err := json.Unmarshal(attrsJSON, &entity.Attributes); err != nil {
				return fmt.Errorf("failed to unmarshal attributes: %w", err)
			}
		}

		event.Entities = append(event.Entities, entity)
	}

	return nil
}

// HasSourceEvents checks if a source has any associated events.
func (r *PostgresEventRepository) HasSourceEvents(ctx context.Context, sourceID string) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM event_sources WHERE source_id = $1`,
		sourceID,
	).Scan(&count)

	if err != nil {
		return false, fmt.Errorf("failed to check source events: %w", err)
	}

	return count > 0, nil
}

// Count returns the total number of events matching the given query.
func (r *PostgresEventRepository) Count(ctx context.Context, query models.EventQuery) (int, error) {
	// Build count query using the existing helper
	countQuery, args := r.buildCountQueryWithArgs(query)

	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return 0, fmt.Errorf("failed to count events: %w", err)
	}

	return total, nil
}

// buildCountQueryWithArgs constructs the count query with arguments.
func (r *PostgresEventRepository) buildCountQueryWithArgs(q models.EventQuery) (string, []interface{}) {
	args := []interface{}{}
	argIdx := 1
	conditions := []string{}

	if q.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *q.Status)
		argIdx++
	} else {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, models.EventStatusPublished)
		argIdx++
	}

	if q.SearchQuery != "" {
		conditions = append(conditions, fmt.Sprintf("to_tsvector('english', title || ' ' || summary) @@ plainto_tsquery('english', $%d)", argIdx))
		args = append(args, q.SearchQuery)
		argIdx++
	}

	if q.SinceTimestamp != nil {
		conditions = append(conditions, fmt.Sprintf("timestamp >= $%d", argIdx))
		args = append(args, *q.SinceTimestamp)
		argIdx++
	}
	if q.UntilTimestamp != nil {
		conditions = append(conditions, fmt.Sprintf("timestamp <= $%d", argIdx))
		args = append(args, *q.UntilTimestamp)
		argIdx++
	}

	if q.MinMagnitude != nil {
		conditions = append(conditions, fmt.Sprintf("magnitude >= $%d", argIdx))
		args = append(args, *q.MinMagnitude)
		argIdx++
	}
	if q.MinConfidence != nil {
		conditions = append(conditions, fmt.Sprintf("(confidence->>'score')::DECIMAL >= $%d", argIdx))
		args = append(args, *q.MinConfidence)
		argIdx++
	}

	if len(q.Categories) > 0 {
		conditions = append(conditions, fmt.Sprintf("category = ANY($%d)", argIdx))
		args = append(args, pq.Array(q.Categories))
		argIdx++
	}

	if len(q.Tags) > 0 {
		conditions = append(conditions, fmt.Sprintf("tags && $%d", argIdx))
		args = append(args, pq.Array(q.Tags))
		argIdx++
	}

	whereClause := "WHERE " + strings.Join(conditions, " AND ")

	return fmt.Sprintf("SELECT COUNT(*) FROM events %s", whereClause), args
}

// GetEventsBetween retrieves events within a time range
func (r *PostgresEventRepository) GetEventsBetween(ctx context.Context, startTime, endTime time.Time, categories []string, limit int) ([]models.Event, error) {
	query := `
		SELECT id, timestamp, title, summary, category, tags, created_at
		FROM events
		WHERE timestamp >= $1 AND timestamp <= $2
	`

	args := []interface{}{startTime, endTime}
	argIdx := 3

	if len(categories) > 0 {
		query += fmt.Sprintf(" AND category = ANY($%d)", argIdx)
		args = append(args, pq.Array(categories))
		argIdx++
	}

	query += fmt.Sprintf(" ORDER BY timestamp DESC LIMIT $%d", argIdx)
	args = append(args, limit)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query events: %w", err)
	}
	defer rows.Close()

	var events []models.Event
	for rows.Next() {
		var event models.Event
		var tags pq.StringArray

		err := rows.Scan(
			&event.ID,
			&event.Timestamp,
			&event.Title,
			&event.Summary,
			&event.Category,
			&tags,
			&event.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}

		event.Tags = tags

		events = append(events, event)
	}

	return events, rows.Err()
}
