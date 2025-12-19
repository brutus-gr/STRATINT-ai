package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/STRATINT/stratint/internal/database"
)

type ActivityLogHandlers struct {
	repo   *database.ActivityLogRepository
	logger *slog.Logger
}

func NewActivityLogHandlers(repo *database.ActivityLogRepository, logger *slog.Logger) *ActivityLogHandlers {
	return &ActivityLogHandlers{
		repo:   repo,
		logger: logger,
	}
}

// ListActivities handles GET /api/activity-logs
func (h *ActivityLogHandlers) ListActivities(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters
	limitStr := r.URL.Query().Get("limit")
	limit := 100
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	activityType := r.URL.Query().Get("activity_type")
	platform := r.URL.Query().Get("platform")

	// Get activity logs
	logs, err := h.repo.List(context.Background(), limit, activityType, platform)
	if err != nil {
		h.logger.Error("failed to list activity logs", "error", err)
		http.Error(w, "Failed to retrieve activity logs", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"logs":  logs,
		"count": len(logs),
	})
}
