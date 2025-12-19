package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/STRATINT/stratint/internal/database"
	"github.com/STRATINT/stratint/internal/models"
)

type ThresholdHandlers struct {
	repo   *database.ThresholdRepository
	logger *slog.Logger
}

func NewThresholdHandlers(repo *database.ThresholdRepository, logger *slog.Logger) *ThresholdHandlers {
	return &ThresholdHandlers{
		repo:   repo,
		logger: logger,
	}
}

// GetThresholds handles GET /api/thresholds
func (h *ThresholdHandlers) GetThresholds(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	config, err := h.repo.Get(context.Background())
	if err != nil {
		h.logger.Error("failed to get thresholds", "error", err)
		http.Error(w, "Failed to get thresholds", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(config)
}

// UpdateThresholds handles POST /api/thresholds
func (h *ThresholdHandlers) UpdateThresholds(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var config models.ThresholdConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate config
	if err := ValidateThresholdConfig(&config); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Update thresholds in database
	if err := h.repo.Update(context.Background(), &config); err != nil {
		h.logger.Error("failed to update thresholds", "error", err)
		http.Error(w, "Failed to update thresholds", http.StatusInternalServerError)
		return
	}

	h.logger.Info("thresholds updated",
		"min_confidence", config.MinConfidence,
		"min_magnitude", config.MinMagnitude,
		"max_source_age_hours", config.MaxSourceAgeHours,
	)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Thresholds updated successfully. Changes are active immediately.",
		"config":  config,
	})
}
