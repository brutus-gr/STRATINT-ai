package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/STRATINT/stratint/internal/eventmanager"
	"github.com/STRATINT/stratint/internal/ingestion"
	"github.com/STRATINT/stratint/internal/models"
	"log/slog"
)

type Handler struct {
	manager            *eventmanager.EventLifecycleManager
	logger             *slog.Logger
	sourceRepo         ingestion.SourceRepository
	trackedAccountRepo models.TrackedAccountRepository
	startTime          time.Time
}

func NewHandler(manager *eventmanager.EventLifecycleManager, sourceRepo ingestion.SourceRepository, trackedAccountRepo models.TrackedAccountRepository, logger *slog.Logger) *Handler {
	return &Handler{
		manager:            manager,
		logger:             logger,
		sourceRepo:         sourceRepo,
		trackedAccountRepo: trackedAccountRepo,
		startTime:          time.Now(),
	}
}

// GetEventsHandler handles GET /api/events
func (h *Handler) GetEventsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters into EventQuery
	query := h.parseQueryParams(r)

	// Get events from manager
	events, err := h.manager.GetEvents(query)
	if err != nil {
		h.logger.Error("failed to get events", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*") // CORS for dev
	w.WriteHeader(http.StatusOK)

	response := EventsResponse{
		Events: events,
		Count:  len(events),
		Query:  query,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("failed to encode response", "error", err)
	}
}

// GetEventByIDHandler handles GET /api/events/:id
func (h *Handler) GetEventByIDHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from path
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		http.Error(w, "Event ID required", http.StatusBadRequest)
		return
	}
	eventID := parts[3]

	// Get event by ID from event manager
	ctx := r.Context()
	event, err := h.manager.GetEventByID(ctx, eventID)
	if err != nil {
		h.logger.Error("failed to get event by ID", "id", eventID, "error", err)
		http.Error(w, "Event not found", http.StatusNotFound)
		return
	}

	if event == nil {
		http.Error(w, "Event not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(event)
}

// GetStatsHandler handles GET /api/stats
func (h *Handler) GetStatsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get total event count efficiently
	totalEvents, err := h.manager.GetEventCount(models.EventQuery{})
	if err != nil {
		h.logger.Error("failed to get event count for stats", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Get a sample of events for calculating averages
	query := models.EventQuery{
		Limit: 200, // Sample for stats calculation
	}
	events, err := h.manager.GetEvents(query)
	if err != nil {
		h.logger.Error("failed to get events for stats", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Get source count
	sourceCount := 0
	ctx := context.Background()
	if count, err := h.sourceRepo.Count(ctx); err == nil {
		sourceCount = count
	}

	// Get tracked account count
	trackedCount := 0
	if accounts, err := h.trackedAccountRepo.ListAll(false); err == nil {
		trackedCount = len(accounts)
	}

	// Calculate uptime
	uptime := time.Since(h.startTime)
	uptimeSeconds := int64(uptime.Seconds())
	hours := int64(uptime.Hours())
	minutes := int64(uptime.Minutes()) % 60
	seconds := uptimeSeconds % 60
	uptimeFormatted := fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)

	// Calculate stats using sample
	stats := calculateStats(events)
	// Override with actual total count
	stats.TotalEvents = totalEvents
	stats.TotalSources = sourceCount
	stats.TrackedAccounts = trackedCount
	stats.UptimeSeconds = uptimeSeconds
	stats.UptimeFormatted = uptimeFormatted

	// Calculate enrichment rate (percentage of sources that have been enriched into events)
	if sourceCount > 0 {
		stats.EnrichmentRate = (float64(totalEvents) / float64(sourceCount)) * 100.0
	} else {
		stats.EnrichmentRate = 0.0
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(stats)
}

// parseQueryParams converts URL query parameters to EventQuery
func (h *Handler) parseQueryParams(r *http.Request) models.EventQuery {
	q := r.URL.Query()
	query := models.EventQuery{}

	// Text search
	if search := q.Get("search"); search != "" {
		query.Search = &search
	}

	// Time range
	if since := q.Get("since"); since != "" {
		if t, err := time.Parse(time.RFC3339, since); err == nil {
			query.Since = &t
		}
	}
	if until := q.Get("until"); until != "" {
		if t, err := time.Parse(time.RFC3339, until); err == nil {
			query.Until = &t
		}
	}

	// Time range shortcuts
	if timeRange := q.Get("time_range"); timeRange != "" {
		now := time.Now()
		var since time.Time
		switch timeRange {
		case "1h":
			since = now.Add(-1 * time.Hour)
		case "6h":
			since = now.Add(-6 * time.Hour)
		case "24h":
			since = now.Add(-24 * time.Hour)
		case "7d":
			since = now.Add(-7 * 24 * time.Hour)
		case "30d":
			since = now.Add(-30 * 24 * time.Hour)
		}
		if !since.IsZero() {
			query.Since = &since
		}
	}

	// Magnitude
	if minMag := q.Get("min_magnitude"); minMag != "" {
		if val, err := strconv.ParseFloat(minMag, 64); err == nil {
			query.MinMagnitude = &val
		}
	}
	if maxMag := q.Get("max_magnitude"); maxMag != "" {
		if val, err := strconv.ParseFloat(maxMag, 64); err == nil {
			query.MaxMagnitude = &val
		}
	}

	// Confidence
	if minConf := q.Get("min_confidence"); minConf != "" {
		if val, err := strconv.ParseFloat(minConf, 64); err == nil {
			query.MinConfidence = &val
		}
	}
	if maxConf := q.Get("max_confidence"); maxConf != "" {
		if val, err := strconv.ParseFloat(maxConf, 64); err == nil {
			query.MaxConfidence = &val
		}
	}

	// Categories
	if categories := q.Get("categories"); categories != "" {
		cats := strings.Split(categories, ",")
		modelCats := make([]models.Category, 0, len(cats))
		for _, c := range cats {
			modelCats = append(modelCats, models.Category(strings.TrimSpace(c)))
		}
		query.Categories = modelCats
	}

	// Tags
	if tags := q.Get("tags"); tags != "" {
		query.Tags = strings.Split(tags, ",")
	}

	// Status
	if status := q.Get("status"); status != "" {
		s := models.EventStatus(status)
		query.Status = &s
	}

	// Sorting
	if sortBy := q.Get("sort_by"); sortBy != "" {
		query.SortBy = models.EventSortField(sortBy)
	}
	if sortOrder := q.Get("sort_order"); sortOrder != "" {
		query.SortOrder = models.SortOrder(sortOrder)
	}

	// Pagination
	if limit := q.Get("limit"); limit != "" {
		if val, err := strconv.Atoi(limit); err == nil {
			query.Limit = val
		}
	}
	if offset := q.Get("offset"); offset != "" {
		if val, err := strconv.Atoi(offset); err == nil {
			query.Offset = val
		}
	}

	return query
}

// Helper functions
func intPtr(i int) *int {
	return &i
}

func calculateStats(events []models.Event) StatsResponse {
	if len(events) == 0 {
		return StatsResponse{
			TotalEvents:    0,
			AvgConfidence:  0,
			AvgMagnitude:   0,
			CategoryCounts: make(map[string]int),
		}
	}

	stats := StatsResponse{
		TotalEvents:    len(events),
		CategoryCounts: make(map[string]int),
	}

	var totalConf, totalMag float64
	for _, e := range events {
		totalConf += e.Confidence.Score
		totalMag += e.Magnitude
		stats.CategoryCounts[string(e.Category)]++
	}

	stats.AvgConfidence = totalConf / float64(len(events))
	stats.AvgMagnitude = totalMag / float64(len(events))

	return stats
}

// HandleSources handles GET /api/sources and POST /api/sources
func (h *Handler) HandleSources(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.getSourcesHandler(w, r)
	case http.MethodPost:
		h.createSourceHandler(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleSourceByID handles GET /api/sources/:id, PUT /api/sources/:id, DELETE /api/sources/:id
func (h *Handler) HandleSourceByID(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.getSourceByIDHandler(w, r)
	case http.MethodPut:
		h.updateSourceHandler(w, r)
	case http.MethodDelete:
		h.deleteSourceHandler(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// getSourcesHandler handles GET /api/sources
func (h *Handler) getSourcesHandler(w http.ResponseWriter, r *http.Request) {
	sources, err := h.manager.GetAllSources(r.Context())
	if err != nil {
		h.logger.Error("failed to get sources", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Filter by tracked account ID if provided
	trackedAccountID := r.URL.Query().Get("tracked_account_id")
	if trackedAccountID != "" {
		// Get the tracked account to know its identifier (feed URL, username, etc.)
		account, err := h.trackedAccountRepo.GetByID(trackedAccountID)
		if err != nil || account == nil {
			h.logger.Error("failed to get tracked account", "id", trackedAccountID, "error", err)
			http.Error(w, "Tracked account not found", http.StatusNotFound)
			return
		}

		// Filter sources based on the account identifier
		// For RSS feeds, we match by URL
		// For Twitter, we match by author or URL pattern
		var filteredSources []models.Source
		for _, source := range sources {
			matchesAccount := false

			switch account.Platform {
			case "rss":
				// For RSS, match sources that came from this feed URL
				if source.Metadata.FeedURL == account.AccountIdentifier {
					matchesAccount = true
				}
			case "twitter":
				// For Twitter, match by author handle
				authorHandle := strings.TrimPrefix(account.AccountIdentifier, "@")
				sourceAuthor := strings.TrimPrefix(source.Author, "@")
				if sourceAuthor == authorHandle || source.AuthorID == account.AccountIdentifier {
					matchesAccount = true
				}
			}

			if matchesAccount {
				filteredSources = append(filteredSources, source)
			}
		}
		sources = filteredSources
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(SourcesResponse{
		Sources: sources,
		Count:   len(sources),
	})
}

// createSourceHandler handles POST /api/sources
func (h *Handler) createSourceHandler(w http.ResponseWriter, r *http.Request) {
	var source models.Source
	if err := json.NewDecoder(r.Body).Decode(&source); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if source.Type == "" || source.URL == "" || source.RawContent == "" {
		http.Error(w, "Missing required fields: type, url, raw_content", http.StatusBadRequest)
		return
	}

	// Check for duplicates by title + URL
	if source.Title != "" && source.URL != "" {
		existing, err := h.sourceRepo.GetByTitleAndURL(r.Context(), source.Title, source.URL)
		if err != nil {
			h.logger.Error("failed to check for duplicate source", "error", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		if existing != nil {
			h.logger.Warn("attempted to create duplicate source", "title", source.Title, "url", source.URL)
			http.Error(w, "Source with same title and URL already exists", http.StatusConflict)
			return
		}
	}

	// Set timestamps
	if source.PublishedAt.IsZero() {
		source.PublishedAt = time.Now()
	}

	if err := h.manager.CreateSource(r.Context(), &source); err != nil {
		h.logger.Error("failed to create source", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(source)
}

// getSourceByIDHandler handles GET /api/sources/:id
func (h *Handler) getSourceByIDHandler(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		http.Error(w, "Source ID required", http.StatusBadRequest)
		return
	}
	sourceID := parts[3]

	source, err := h.manager.GetSourceByID(r.Context(), sourceID)
	if err != nil {
		h.logger.Error("failed to get source", "error", err)
		http.Error(w, "Source not found", http.StatusNotFound)
		return
	}

	if source == nil {
		http.Error(w, "Source not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(source)
}

// updateSourceHandler handles PUT /api/sources/:id
func (h *Handler) updateSourceHandler(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		http.Error(w, "Source ID required", http.StatusBadRequest)
		return
	}
	sourceID := parts[3]

	var source models.Source
	if err := json.NewDecoder(r.Body).Decode(&source); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	source.ID = sourceID // Ensure ID matches URL

	if err := h.manager.UpdateSource(r.Context(), &source); err != nil {
		h.logger.Error("failed to update source", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(source)
}

// deleteSourceHandler handles DELETE /api/sources/:id
func (h *Handler) deleteSourceHandler(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		http.Error(w, "Source ID required", http.StatusBadRequest)
		return
	}
	sourceID := parts[3]

	if err := h.manager.DeleteSource(r.Context(), sourceID); err != nil {
		h.logger.Error("failed to delete source", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusNoContent)
}

// UpdateEventStatusHandler handles PUT /api/events/:id/status
func (h *Handler) UpdateEventStatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from path
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 5 {
		http.Error(w, "Event ID required", http.StatusBadRequest)
		return
	}
	eventID := parts[3]

	var request struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	var err error
	switch request.Status {
	case "published":
		err = h.manager.PublishEvent(r.Context(), eventID)
	case "rejected":
		err = h.manager.RejectEvent(r.Context(), eventID)
	case "archived":
		err = h.manager.ArchiveEvent(r.Context(), eventID)
	default:
		http.Error(w, "Invalid status. Must be: published, rejected, or archived", http.StatusBadRequest)
		return
	}

	if err != nil {
		h.logger.Error("failed to update event status", "event_id", eventID, "status", request.Status, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.logger.Info("event status updated", "event_id", eventID, "status", request.Status)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"message":  fmt.Sprintf("Event %s successfully", request.Status),
		"event_id": eventID,
		"status":   request.Status,
	})
}

// Response types
type EventsResponse struct {
	Events []models.Event    `json:"events"`
	Count  int               `json:"count"`
	Query  models.EventQuery `json:"query,omitempty"`
}

type StatsResponse struct {
	TotalEvents     int            `json:"total_events"`
	TotalSources    int            `json:"total_sources"`
	TrackedAccounts int            `json:"tracked_accounts"`
	AvgConfidence   float64        `json:"avg_confidence"`
	AvgMagnitude    float64        `json:"avg_magnitude"`
	EnrichmentRate  float64        `json:"enrichment_rate"`
	CategoryCounts  map[string]int `json:"category_counts"`
	UptimeSeconds   int64          `json:"uptime_seconds"`
	UptimeFormatted string         `json:"uptime_formatted"`
}

type SourcesResponse struct {
	Sources []models.Source `json:"sources"`
	Count   int             `json:"count"`
}
