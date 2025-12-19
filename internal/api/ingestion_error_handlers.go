package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/STRATINT/stratint/internal/database"
)

type IngestionErrorHandler struct {
	repo   database.IngestionErrorRepository
	logger *slog.Logger
}

func NewIngestionErrorHandler(repo database.IngestionErrorRepository, logger *slog.Logger) *IngestionErrorHandler {
	return &IngestionErrorHandler{
		repo:   repo,
		logger: logger,
	}
}

// ListErrors returns ingestion errors with optional filtering
// GET /api/ingestion-errors?limit=100&unresolved_only=true
func (h *IngestionErrorHandler) ListErrors(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	// Parse query parameters
	limitStr := r.URL.Query().Get("limit")
	limit := 100
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	unresolvedOnly := r.URL.Query().Get("unresolved_only") == "true"

	ctx := context.Background()
	errors, err := h.repo.List(ctx, limit, unresolvedOnly)
	if err != nil {
		h.logger.Error("failed to list ingestion errors", "error", err)
		http.Error(w, "Failed to list errors", http.StatusInternalServerError)
		return
	}

	// Get count of unresolved errors
	unresolvedCount, err := h.repo.CountUnresolved(ctx)
	if err != nil {
		h.logger.Error("failed to count unresolved errors", "error", err)
		unresolvedCount = 0
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"errors":           errors,
		"count":            len(errors),
		"unresolved_count": unresolvedCount,
	})
}

// ResolveError marks an error as resolved
// POST /api/ingestion-errors/:id/resolve
func (h *IngestionErrorHandler) ResolveError(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	// Extract ID from path
	path := strings.TrimPrefix(r.URL.Path, "/api/ingestion-errors/")
	id := strings.TrimSuffix(path, "/resolve")

	ctx := context.Background()
	if err := h.repo.MarkResolved(ctx, id); err != nil {
		h.logger.Error("failed to resolve error", "id", id, "error", err)
		http.Error(w, "Failed to resolve error", http.StatusInternalServerError)
		return
	}

	h.logger.Info("resolved ingestion error", "id", id)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"id":      id,
	})
}

// DeleteError removes an error from the database
// DELETE /api/ingestion-errors/:id
func (h *IngestionErrorHandler) DeleteError(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	// Extract ID from path
	id := strings.TrimPrefix(r.URL.Path, "/api/ingestion-errors/")

	ctx := context.Background()
	if err := h.repo.Delete(ctx, id); err != nil {
		h.logger.Error("failed to delete error", "id", id, "error", err)
		http.Error(w, "Failed to delete error", http.StatusInternalServerError)
		return
	}

	h.logger.Info("deleted ingestion error", "id", id)

	w.WriteHeader(http.StatusNoContent)
}
