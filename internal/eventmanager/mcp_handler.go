package eventmanager

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/STRATINT/stratint/internal/models"
)

// MCPHandler handles MCP (Model Context Protocol) function calls for OSINT events.
type MCPHandler struct {
	lifecycle *EventLifecycleManager
}

// NewMCPHandler creates a new MCP handler.
func NewMCPHandler(lifecycle *EventLifecycleManager) *MCPHandler {
	return &MCPHandler{
		lifecycle: lifecycle,
	}
}

// MCPEvent represents an event without internal fields for MCP responses
// Excludes: status, summary, raw_content
type MCPEvent struct {
	ID         string            `json:"id"`
	Timestamp  time.Time         `json:"timestamp"`
	Title      string            `json:"title"`
	Magnitude  float64           `json:"magnitude"`
	Confidence models.Confidence `json:"confidence"`
	Category   models.Category   `json:"category"`
	Entities   []models.Entity   `json:"entities"`
	Sources    []models.Source   `json:"sources"`
	Tags       []string          `json:"tags"`
	Location   *models.Location  `json:"location,omitempty"`
	CreatedAt  time.Time         `json:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at"`
}

// MCPEventResponse represents the MCP-specific response format
type MCPEventResponse struct {
	Events []MCPEvent `json:"events"`
	Total  int        `json:"total"`
	Page   int        `json:"page"`
	Limit  int        `json:"limit"`
}

// GetEvents implements the get_events MCP function.
// Accepts an EventQuery JSON object and returns an EventResponse.
// Only returns published events and omits the status field.
func (h *MCPHandler) GetEvents(ctx context.Context, queryJSON string) (string, error) {
	// Parse query from JSON
	var query models.EventQuery
	if err := json.Unmarshal([]byte(queryJSON), &query); err != nil {
		return "", fmt.Errorf("invalid query JSON: %w", err)
	}

	// Force status to published - MCP clients should only see published events
	publishedStatus := models.EventStatusPublished
	query.Status = &publishedStatus

	// Validate query
	if err := query.Validate(); err != nil {
		return "", fmt.Errorf("invalid query parameters: %w", err)
	}

	// Get published events
	response, err := h.lifecycle.GetPublishedEvents(ctx, query)
	if err != nil {
		return "", fmt.Errorf("query failed: %w", err)
	}

	// Convert to MCP format (without internal fields)
	mcpEvents := make([]MCPEvent, len(response.Events))
	for i, event := range response.Events {
		mcpEvents[i] = MCPEvent{
			ID:         event.ID,
			Timestamp:  event.Timestamp,
			Title:      event.Title,
			Magnitude:  event.Magnitude,
			Confidence: event.Confidence,
			Category:   event.Category,
			Entities:   event.Entities,
			Sources:    event.Sources,
			Tags:       event.Tags,
			Location:   event.Location,
			CreatedAt:  event.CreatedAt,
			UpdatedAt:  event.UpdatedAt,
		}
	}

	mcpResponse := MCPEventResponse{
		Events: mcpEvents,
		Total:  response.Total,
		Page:   response.Page,
		Limit:  response.Limit,
	}

	// Serialize response to JSON
	responseJSON, err := json.Marshal(mcpResponse)
	if err != nil {
		return "", fmt.Errorf("failed to serialize response: %w", err)
	}

	return string(responseJSON), nil
}

// MCPToolDefinition returns the MCP tool definition for get_events.
func (h *MCPHandler) MCPToolDefinition() map[string]interface{} {
	return map[string]interface{}{
		"name":        "get_events",
		"description": "Query OSINT events with comprehensive filtering including search, time ranges, thresholds, categories, and source types",
		"inputSchema": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"search_query": map[string]interface{}{
					"type":        "string",
					"description": "Full-text search across event title and summary",
				},
				"since_timestamp": map[string]interface{}{
					"type":        "string",
					"format":      "date-time",
					"description": "Start of time range (RFC3339 format, e.g., '2024-01-01T00:00:00Z')",
				},
				"until_timestamp": map[string]interface{}{
					"type":        "string",
					"format":      "date-time",
					"description": "End of time range (RFC3339 format)",
				},
				"min_magnitude": map[string]interface{}{
					"type":        "number",
					"minimum":     0,
					"maximum":     10,
					"description": "Minimum event magnitude/severity (0-10 scale)",
				},
				"min_confidence": map[string]interface{}{
					"type":        "number",
					"minimum":     0,
					"maximum":     1,
					"description": "Minimum confidence score (0-1 scale)",
				},
				"categories": map[string]interface{}{
					"type": "array",
					"items": map[string]interface{}{
						"type": "string",
						"enum": []string{
							"geopolitics", "military", "economic", "cyber",
							"disaster", "terrorism", "diplomacy", "intelligence",
							"humanitarian", "other",
						},
					},
					"description": "Filter by event categories (military, cyber, geopolitics, etc.)",
				},
				"source_types": map[string]interface{}{
					"type": "array",
					"items": map[string]interface{}{
						"type": "string",
						"enum": []string{
							"twitter", "telegram", "glp",
							"government", "news_media", "blog", "other",
						},
					},
					"description": "Filter by source types (twitter, government, news, etc.)",
				},
				"tags": map[string]interface{}{
					"type": "array",
					"items": map[string]interface{}{
						"type": "string",
					},
					"description": "Filter by tags",
				},
				"entity_types": map[string]interface{}{
					"type": "array",
					"items": map[string]interface{}{
						"type": "string",
						"enum": []string{
							"country", "city", "region", "person", "organization",
							"military_unit", "vessel", "weapon_system", "facility",
							"event", "other",
						},
					},
					"description": "Filter by entity types present in events",
				},
				"page": map[string]interface{}{
					"type":        "integer",
					"minimum":     1,
					"default":     1,
					"description": "Page number for pagination (1-indexed)",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"minimum":     1,
					"maximum":     200,
					"default":     20,
					"description": "Number of results per page (max 200)",
				},
				"sort_by": map[string]interface{}{
					"type": "string",
					"enum": []string{
						"timestamp", "magnitude", "confidence",
						"created_at", "updated_at",
					},
					"default":     "timestamp",
					"description": "Field to sort results by",
				},
				"sort_order": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"asc", "desc"},
					"default":     "desc",
					"description": "Sort direction (ascending or descending)",
				},
			},
			"required": []string{}, // All parameters are optional
		},
	}
}

// GetEventByID retrieves a specific event by ID (helper function).
func (h *MCPHandler) GetEventByID(ctx context.Context, eventID string) (*models.Event, error) {
	return h.lifecycle.eventRepo.GetByID(ctx, eventID)
}

// GetStats returns event statistics (helper function).
func (h *MCPHandler) GetStats(ctx context.Context) (LifecycleStats, error) {
	return h.lifecycle.GetStats(ctx)
}
