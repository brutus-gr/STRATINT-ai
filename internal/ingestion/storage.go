package ingestion

import (
	"context"
	"time"

	"github.com/STRATINT/stratint/internal/models"
)

// SourceRepository defines the interface for storing and retrieving sources.
type SourceRepository interface {
	// StoreRaw saves a raw source to the repository.
	StoreRaw(ctx context.Context, source models.Source) error

	// StoreBatch saves multiple raw sources in a single operation.
	StoreBatch(ctx context.Context, sources []models.Source) error

	// GetByID retrieves a source by its ID.
	GetByID(ctx context.Context, id string) (*models.Source, error)

	// GetByURL retrieves a source by its URL.
	GetByURL(ctx context.Context, url string) (*models.Source, error)

	// GetByTitleAndURL checks if a source with the same title and URL exists.
	GetByTitleAndURL(ctx context.Context, title, url string) (*models.Source, error)

	// ListRecent retrieves sources published since the given timestamp.
	ListRecent(ctx context.Context, since time.Time, limit int) ([]models.Source, error)

	// ListByType retrieves sources of a specific type.
	ListByType(ctx context.Context, sourceType models.SourceType, limit int) ([]models.Source, error)

	// GetByStatus retrieves sources with a specific scrape status.
	GetByStatus(ctx context.Context, status models.ScrapeStatus, limit int) ([]models.Source, error)

	// Update modifies an existing source.
	Update(ctx context.Context, source models.Source) error

	// Delete removes a source by its ID.
	Delete(ctx context.Context, id string) error

	// Exists checks if a source with the given ID exists.
	Exists(ctx context.Context, id string) (bool, error)

	// Count returns the total number of sources.
	Count(ctx context.Context) (int, error)
}

// EventRepository defines the interface for storing and retrieving events.
type EventRepository interface {
	// Create stores a new event.
	Create(ctx context.Context, event models.Event) error

	// Update modifies an existing event.
	Update(ctx context.Context, event models.Event) error

	// GetByID retrieves an event by its ID.
	GetByID(ctx context.Context, id string) (*models.Event, error)

	// Query retrieves events matching the given query parameters.
	Query(ctx context.Context, query models.EventQuery) (*models.EventResponse, error)

	// Delete removes an event by its ID.
	Delete(ctx context.Context, id string) error

	// UpdateStatus changes the status of an event.
	UpdateStatus(ctx context.Context, id string, status models.EventStatus) error

	// HasSourceEvents checks if a source has any associated events.
	HasSourceEvents(ctx context.Context, sourceID string) (bool, error)

	// Count returns the total number of events matching the given query.
	Count(ctx context.Context, query models.EventQuery) (int, error)
}

// MemorySourceRepository implements an in-memory source repository for testing/development.
type MemorySourceRepository struct {
	sources map[string]models.Source
	urlIdx  map[string]string // URL -> ID mapping
}

// NewMemorySourceRepository creates a new in-memory source repository.
func NewMemorySourceRepository() *MemorySourceRepository {
	return &MemorySourceRepository{
		sources: make(map[string]models.Source),
		urlIdx:  make(map[string]string),
	}
}

// StoreRaw saves a raw source to memory.
func (r *MemorySourceRepository) StoreRaw(ctx context.Context, source models.Source) error {
	r.sources[source.ID] = source
	if source.URL != "" {
		r.urlIdx[source.URL] = source.ID
	}
	return nil
}

// StoreBatch saves multiple sources to memory.
func (r *MemorySourceRepository) StoreBatch(ctx context.Context, sources []models.Source) error {
	for _, source := range sources {
		if err := r.StoreRaw(ctx, source); err != nil {
			return err
		}
	}
	return nil
}

// GetByID retrieves a source by ID.
func (r *MemorySourceRepository) GetByID(ctx context.Context, id string) (*models.Source, error) {
	source, ok := r.sources[id]
	if !ok {
		return nil, nil
	}
	return &source, nil
}

// GetByURL retrieves a source by URL.
func (r *MemorySourceRepository) GetByURL(ctx context.Context, url string) (*models.Source, error) {
	id, ok := r.urlIdx[url]
	if !ok {
		return nil, nil
	}
	return r.GetByID(ctx, id)
}

// GetByTitleAndURL checks if a source with the same title and URL exists.
func (r *MemorySourceRepository) GetByTitleAndURL(ctx context.Context, title, url string) (*models.Source, error) {
	for _, source := range r.sources {
		if source.Title == title && source.URL == url {
			return &source, nil
		}
	}
	return nil, nil
}

// ListRecent retrieves recent sources.
func (r *MemorySourceRepository) ListRecent(ctx context.Context, since time.Time, limit int) ([]models.Source, error) {
	result := make([]models.Source, 0, limit)

	for _, source := range r.sources {
		if source.PublishedAt.After(since) {
			result = append(result, source)
			if len(result) >= limit {
				break
			}
		}
	}

	return result, nil
}

// ListByType retrieves sources by type.
func (r *MemorySourceRepository) ListByType(ctx context.Context, sourceType models.SourceType, limit int) ([]models.Source, error) {
	result := make([]models.Source, 0, limit)

	for _, source := range r.sources {
		if source.Type == sourceType {
			result = append(result, source)
			if len(result) >= limit {
				break
			}
		}
	}

	return result, nil
}

// GetByStatus retrieves sources by scrape status.
func (r *MemorySourceRepository) GetByStatus(ctx context.Context, status models.ScrapeStatus, limit int) ([]models.Source, error) {
	result := make([]models.Source, 0, limit)

	for _, source := range r.sources {
		if source.ScrapeStatus == status {
			result = append(result, source)
			if len(result) >= limit {
				break
			}
		}
	}

	return result, nil
}

// Update modifies an existing source.
func (r *MemorySourceRepository) Update(ctx context.Context, source models.Source) error {
	r.sources[source.ID] = source
	if source.URL != "" {
		r.urlIdx[source.URL] = source.ID
	}
	return nil
}

// Delete removes a source by its ID.
func (r *MemorySourceRepository) Delete(ctx context.Context, id string) error {
	source, ok := r.sources[id]
	if ok && source.URL != "" {
		delete(r.urlIdx, source.URL)
	}
	delete(r.sources, id)
	return nil
}

// Exists checks if a source exists.
func (r *MemorySourceRepository) Exists(ctx context.Context, id string) (bool, error) {
	_, ok := r.sources[id]
	return ok, nil
}

// Count returns the total number of sources in the repository.
func (r *MemorySourceRepository) Count(ctx context.Context) (int, error) {
	return len(r.sources), nil
}

// Size returns the number of sources in the repository.
func (r *MemorySourceRepository) Size() int {
	return len(r.sources)
}

// MemoryEventRepository implements an in-memory event repository for testing/development.
type MemoryEventRepository struct {
	events map[string]models.Event
}

// NewMemoryEventRepository creates a new in-memory event repository.
func NewMemoryEventRepository() *MemoryEventRepository {
	return &MemoryEventRepository{
		events: make(map[string]models.Event),
	}
}

// Create stores a new event.
func (r *MemoryEventRepository) Create(ctx context.Context, event models.Event) error {
	event.CreatedAt = time.Now()
	event.UpdatedAt = time.Now()
	r.events[event.ID] = event
	return nil
}

// Update modifies an existing event.
func (r *MemoryEventRepository) Update(ctx context.Context, event models.Event) error {
	event.UpdatedAt = time.Now()
	r.events[event.ID] = event
	return nil
}

// GetByID retrieves an event by ID.
func (r *MemoryEventRepository) GetByID(ctx context.Context, id string) (*models.Event, error) {
	event, ok := r.events[id]
	if !ok {
		return nil, nil
	}
	return &event, nil
}

// Query retrieves events matching query parameters.
func (r *MemoryEventRepository) Query(ctx context.Context, query models.EventQuery) (*models.EventResponse, error) {
	if err := query.Validate(); err != nil {
		return nil, err
	}

	// Simple in-memory filtering (not optimized)
	matching := make([]models.Event, 0)

	for _, event := range r.events {
		if matchesQuery(event, query) {
			matching = append(matching, event)
		}
	}

	// Apply pagination
	total := len(matching)
	offset := query.GetOffset()
	end := offset + query.Limit

	if offset >= total {
		return &models.EventResponse{
			Events:  []models.Event{},
			Page:    query.Page,
			Limit:   query.Limit,
			Total:   total,
			HasMore: false,
		}, nil
	}

	if end > total {
		end = total
	}

	page := matching[offset:end]

	return &models.EventResponse{
		Events:  page,
		Page:    query.Page,
		Limit:   query.Limit,
		Total:   total,
		HasMore: end < total,
	}, nil
}

// Delete removes an event.
func (r *MemoryEventRepository) Delete(ctx context.Context, id string) error {
	delete(r.events, id)
	return nil
}

// UpdateStatus changes an event's status.
func (r *MemoryEventRepository) UpdateStatus(ctx context.Context, id string, status models.EventStatus) error {
	event, ok := r.events[id]
	if !ok {
		return nil
	}

	event.Status = status
	event.UpdatedAt = time.Now()
	r.events[id] = event

	return nil
}

// HasSourceEvents checks if a source has any associated events (in-memory implementation).
func (r *MemoryEventRepository) HasSourceEvents(ctx context.Context, sourceID string) (bool, error) {
	// For in-memory implementation, check if any event has this source
	for _, event := range r.events {
		for _, source := range event.Sources {
			if source.ID == sourceID {
				return true, nil
			}
		}
	}
	return false, nil
}

// Count returns the total number of events matching the query.
func (r *MemoryEventRepository) Count(ctx context.Context, query models.EventQuery) (int, error) {
	matching := 0
	for _, event := range r.events {
		if matchesQuery(event, query) {
			matching++
		}
	}
	return matching, nil
}

// Size returns the number of events in the repository.
func (r *MemoryEventRepository) Size() int {
	return len(r.events)
}

// matchesQuery checks if an event matches query filters.
func matchesQuery(event models.Event, query models.EventQuery) bool {
	// Status filter
	if query.Status != nil && event.Status != *query.Status {
		return false
	}

	// Time range filters
	if query.SinceTimestamp != nil && event.Timestamp.Before(*query.SinceTimestamp) {
		return false
	}
	if query.UntilTimestamp != nil && event.Timestamp.After(*query.UntilTimestamp) {
		return false
	}

	// Magnitude filter
	if query.MinMagnitude != nil && event.Magnitude < *query.MinMagnitude {
		return false
	}

	// Confidence filter
	if query.MinConfidence != nil && event.Confidence.Score < *query.MinConfidence {
		return false
	}

	// Category filter
	if len(query.Categories) > 0 {
		found := false
		for _, cat := range query.Categories {
			if event.Category == cat {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}
