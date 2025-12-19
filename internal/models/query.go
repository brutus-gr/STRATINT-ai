package models

import (
	"time"
)

// EventQuery represents filters and pagination for retrieving events via the MCP API.
type EventQuery struct {
	// Search and time filters
	Search         *string    `json:"search,omitempty"`
	SearchQuery    string     `json:"search_query,omitempty"` // Alias for MCP compatibility
	Since          *time.Time `json:"since,omitempty"`
	SinceTimestamp *time.Time `json:"since_timestamp,omitempty"` // Alias for MCP compatibility
	Until          *time.Time `json:"until,omitempty"`
	UntilTimestamp *time.Time `json:"until_timestamp,omitempty"` // Alias for MCP compatibility

	// Magnitude filters
	MinMagnitude *float64 `json:"min_magnitude,omitempty"`
	MaxMagnitude *float64 `json:"max_magnitude,omitempty"`

	// Confidence filters
	MinConfidence *float64 `json:"min_confidence,omitempty"`
	MaxConfidence *float64 `json:"max_confidence,omitempty"`

	// Category and type filters
	Categories  []Category   `json:"categories,omitempty"`
	SourceTypes []SourceType `json:"source_types,omitempty"`
	Tags        []string     `json:"tags,omitempty"`
	EntityTypes []EntityType `json:"entity_types,omitempty"`
	Status      *EventStatus `json:"status,omitempty"`

	// Pagination
	Page   int `json:"page"`
	Limit  int `json:"limit,omitempty"`
	Offset int `json:"offset,omitempty"`

	// Sorting
	SortBy    EventSortField `json:"sort_by,omitempty"`
	SortOrder SortOrder      `json:"sort_order,omitempty"`
}

// EventSortField specifies which field to sort events by.
type EventSortField string

const (
	SortByTimestamp  EventSortField = "timestamp"
	SortByMagnitude  EventSortField = "magnitude"
	SortByConfidence EventSortField = "confidence"
	SortByCreatedAt  EventSortField = "created_at"
	SortByUpdatedAt  EventSortField = "updated_at"
)

// SortOrder specifies ascending or descending sort direction.
type SortOrder string

const (
	SortOrderAsc  SortOrder = "asc"
	SortOrderDesc SortOrder = "desc"
)

// Validate ensures the query parameters are valid and applies defaults.
func (q *EventQuery) Validate() error {
	// Set defaults for page
	if q.Page < 1 {
		q.Page = 1
	}

	// Set defaults for limit
	if q.Limit == 0 {
		q.Limit = 20
	}
	if q.Limit < 1 {
		q.Limit = 20
	}
	if q.Limit > 1000 {
		q.Limit = 1000
	}

	// Set defaults for sorting
	if q.SortBy == "" {
		q.SortBy = SortByTimestamp
	}
	if q.SortOrder == "" {
		q.SortOrder = SortOrderDesc
	}

	// Sync aliases for MCP compatibility
	if q.Search != nil && q.SearchQuery == "" {
		q.SearchQuery = *q.Search
	}
	if q.Since != nil && q.SinceTimestamp == nil {
		q.SinceTimestamp = q.Since
	}
	if q.Until != nil && q.UntilTimestamp == nil {
		q.UntilTimestamp = q.Until
	}

	return nil
}

// GetOffset calculates the database offset for pagination.
func (q *EventQuery) GetOffset() int {
	if q.Offset > 0 {
		return q.Offset
	}
	if q.Limit > 0 {
		return (q.Page - 1) * q.Limit
	}
	return 0
}

// EventResponse represents a paginated list of events with metadata.
type EventResponse struct {
	Events  []Event `json:"events"`
	Page    int     `json:"page"`
	Limit   int     `json:"limit"`
	Total   int     `json:"total"`
	HasMore bool    `json:"has_more"`
	Query   string  `json:"query,omitempty"`
}
