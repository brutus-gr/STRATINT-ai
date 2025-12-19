package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/STRATINT/stratint/internal/database"
	"github.com/STRATINT/stratint/internal/models"
)

// InferenceLogHandler handles HTTP requests for inference log management
type InferenceLogHandler struct {
	repo   *database.InferenceLogRepository
	logger *slog.Logger
}

// NewInferenceLogHandler creates a new handler
func NewInferenceLogHandler(repo *database.InferenceLogRepository, logger *slog.Logger) *InferenceLogHandler {
	return &InferenceLogHandler{
		repo:   repo,
		logger: logger,
	}
}

// ListInferenceLogs handles GET /api/admin/inference-logs
func (h *InferenceLogHandler) ListInferenceLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters
	query := models.InferenceLogQuery{
		Provider:  r.URL.Query().Get("provider"),
		Model:     r.URL.Query().Get("model"),
		Operation: r.URL.Query().Get("operation"),
		Status:    r.URL.Query().Get("status"),
		Limit:     100, // Default limit
		Offset:    0,
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			query.Limit = limit
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			query.Offset = offset
		}
	}

	// Parse date filters
	if startDateStr := r.URL.Query().Get("start_date"); startDateStr != "" {
		if startDate, err := time.Parse(time.RFC3339, startDateStr); err == nil {
			query.StartDate = &startDate
		}
	}

	if endDateStr := r.URL.Query().Get("end_date"); endDateStr != "" {
		if endDate, err := time.Parse(time.RFC3339, endDateStr); err == nil {
			query.EndDate = &endDate
		}
	}

	ctx := context.Background()
	logs, err := h.repo.List(ctx, query)
	if err != nil {
		h.logger.Error("failed to list inference logs", "error", err)
		http.Error(w, "Failed to list inference logs: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"logs":   logs,
		"limit":  query.Limit,
		"offset": query.Offset,
	})
}

// GetInferenceStats handles GET /api/admin/inference-logs/stats
func (h *InferenceLogHandler) GetInferenceStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var startDate, endDate *time.Time

	// Parse date filters
	if startDateStr := r.URL.Query().Get("start_date"); startDateStr != "" {
		if parsed, err := time.Parse(time.RFC3339, startDateStr); err == nil {
			startDate = &parsed
		}
	}

	if endDateStr := r.URL.Query().Get("end_date"); endDateStr != "" {
		if parsed, err := time.Parse(time.RFC3339, endDateStr); err == nil {
			endDate = &parsed
		}
	}

	ctx := context.Background()
	stats, err := h.repo.GetStats(ctx, startDate, endDate)
	if err != nil {
		h.logger.Error("failed to get inference stats", "error", err)
		http.Error(w, "Failed to get inference stats: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}
