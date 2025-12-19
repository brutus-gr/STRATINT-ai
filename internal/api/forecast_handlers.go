package api

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/STRATINT/stratint/internal/database"
	"github.com/STRATINT/stratint/internal/forecaster"
	"github.com/STRATINT/stratint/internal/inference"
	"github.com/STRATINT/stratint/internal/models"
)

// ForecastHandler handles forecast-related HTTP requests
type ForecastHandler struct {
	forecastRepo *database.ForecastRepository
	forecaster   *forecaster.Forecaster
	logger       *slog.Logger
}

// NewForecastHandler creates a new forecast handler
func NewForecastHandler(db *sql.DB, eventRepo *database.PostgresEventRepository, logger *slog.Logger, inferenceLogger *inference.Logger) *ForecastHandler {
	forecastRepo := database.NewForecastRepository(db)
	forecasterInstance := forecaster.NewForecaster(eventRepo, forecastRepo, logger, inferenceLogger)

	return &ForecastHandler{
		forecastRepo: forecastRepo,
		forecaster:   forecasterInstance,
		logger:       logger,
	}
}

// CreateForecast handles POST /api/admin/forecasts
func (h *ForecastHandler) CreateForecast(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.CreateForecastRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode request", "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate
	if req.Name == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}
	if req.Proposition == "" {
		http.Error(w, "Proposition is required", http.StatusBadRequest)
		return
	}
	if len(req.Models) == 0 {
		http.Error(w, "At least one model is required", http.StatusBadRequest)
		return
	}
	if req.HeadlineCount <= 0 {
		req.HeadlineCount = 500 // Default
	}
	if req.Iterations <= 0 {
		req.Iterations = 1 // Default
	}

	ctx := r.Context()
	forecast, err := h.forecastRepo.CreateForecast(ctx, req)
	if err != nil {
		h.logger.Error("Failed to create forecast", "error", err)
		http.Error(w, "Failed to create forecast", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(forecast)
}

// ListForecasts handles GET /api/admin/forecasts
func (h *ForecastHandler) ListForecasts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	forecasts, err := h.forecastRepo.ListForecasts(ctx)
	if err != nil {
		h.logger.Error("Failed to list forecasts", "error", err)
		http.Error(w, "Failed to list forecasts", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"forecasts": forecasts,
		"count":     len(forecasts),
	})
}

// UpdateForecast handles PUT /api/admin/forecasts/:id
func (h *ForecastHandler) UpdateForecast(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from URL
	path := strings.TrimPrefix(r.URL.Path, "/api/admin/forecasts/")
	if path == "" || strings.Contains(path, "/") {
		http.Error(w, "Invalid forecast ID", http.StatusBadRequest)
		return
	}
	forecastID := path

	var req models.CreateForecastRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode request", "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate
	if req.Name == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}
	if req.Proposition == "" {
		http.Error(w, "Proposition is required", http.StatusBadRequest)
		return
	}
	if len(req.Models) == 0 {
		http.Error(w, "At least one model is required", http.StatusBadRequest)
		return
	}
	if req.HeadlineCount <= 0 {
		req.HeadlineCount = 500 // Default
	}
	if req.Iterations <= 0 {
		req.Iterations = 1 // Default
	}

	ctx := r.Context()
	forecast, err := h.forecastRepo.UpdateForecast(ctx, forecastID, req)
	if err != nil {
		h.logger.Error("Failed to update forecast", "error", err)
		http.Error(w, "Failed to update forecast", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(forecast)
}

// GetForecast handles GET /api/admin/forecasts/:id
func (h *ForecastHandler) GetForecast(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from URL
	path := strings.TrimPrefix(r.URL.Path, "/api/admin/forecasts/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "Forecast ID required", http.StatusBadRequest)
		return
	}
	forecastID := parts[0]

	ctx := r.Context()
	forecast, err := h.forecastRepo.GetForecast(ctx, forecastID)
	if err != nil {
		h.logger.Error("Failed to get forecast", "error", err)
		http.Error(w, "Failed to get forecast", http.StatusInternalServerError)
		return
	}
	if forecast == nil {
		http.Error(w, "Forecast not found", http.StatusNotFound)
		return
	}

	// Also get models
	models, err := h.forecastRepo.GetForecastModels(ctx, forecastID)
	if err != nil {
		h.logger.Error("Failed to get forecast models", "error", err)
		http.Error(w, "Failed to get forecast models", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"forecast": forecast,
		"models":   models,
	})
}

// ExecuteForecast handles POST /api/admin/forecasts/:id/execute
func (h *ForecastHandler) ExecuteForecast(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from URL
	path := strings.TrimPrefix(r.URL.Path, "/api/admin/forecasts/")
	path = strings.TrimSuffix(path, "/execute")
	if path == "" {
		http.Error(w, "Forecast ID required", http.StatusBadRequest)
		return
	}
	forecastID := path

	ctx := r.Context()
	runID, err := h.forecaster.ExecuteForecast(ctx, forecastID)
	if err != nil {
		h.logger.Error("Failed to execute forecast", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Forecast execution started",
		"run_id":  runID,
	})
}

// GetForecastRun handles GET /api/admin/forecasts/runs/:runId
func (h *ForecastHandler) GetForecastRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract run ID from URL
	path := strings.TrimPrefix(r.URL.Path, "/api/admin/forecasts/runs/")
	if path == "" {
		http.Error(w, "Run ID required", http.StatusBadRequest)
		return
	}
	runID := path

	ctx := r.Context()
	runDetail, err := h.forecastRepo.GetForecastRun(ctx, runID)
	if err != nil {
		h.logger.Error("Failed to get forecast run", "error", err)
		http.Error(w, "Failed to get forecast run", http.StatusInternalServerError)
		return
	}
	if runDetail == nil {
		http.Error(w, "Forecast run not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(runDetail)
}

// ListForecastRuns handles GET /api/admin/forecasts/:id/runs
func (h *ForecastHandler) ListForecastRuns(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from URL
	path := strings.TrimPrefix(r.URL.Path, "/api/admin/forecasts/")
	path = strings.TrimSuffix(path, "/runs")
	if path == "" {
		http.Error(w, "Forecast ID required", http.StatusBadRequest)
		return
	}
	forecastID := path

	ctx := r.Context()
	runs, err := h.forecastRepo.ListForecastRuns(ctx, forecastID, 50)
	if err != nil {
		h.logger.Error("Failed to list forecast runs", "error", err)
		http.Error(w, "Failed to list forecast runs", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"runs":  runs,
		"count": len(runs),
	})
}

// GetForecastHistory handles GET /api/admin/forecasts/:id/history
func (h *ForecastHandler) GetForecastHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from URL
	path := strings.TrimPrefix(r.URL.Path, "/api/admin/forecasts/")
	path = strings.TrimSuffix(path, "/history")
	if path == "" {
		http.Error(w, "Forecast ID required", http.StatusBadRequest)
		return
	}
	forecastID := path

	ctx := r.Context()
	history, err := h.forecastRepo.GetForecastHistory(ctx, forecastID)
	if err != nil {
		h.logger.Error("Failed to get forecast history", "error", err)
		http.Error(w, "Failed to get forecast history", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"history": history,
		"count":   len(history),
	})
}

// GetForecastHistoryDaily handles GET /api/admin/forecasts/:id/history/daily
func (h *ForecastHandler) GetForecastHistoryDaily(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from URL
	path := strings.TrimPrefix(r.URL.Path, "/api/admin/forecasts/")
	path = strings.TrimSuffix(path, "/history/daily")
	if path == "" {
		http.Error(w, "Forecast ID required", http.StatusBadRequest)
		return
	}
	forecastID := path

	ctx := r.Context()
	ohlcData, err := h.forecastRepo.GetForecastHistoryDaily(ctx, forecastID)
	if err != nil {
		h.logger.Error("Failed to get daily OHLC data", "error", err)
		http.Error(w, "Failed to get daily OHLC data", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data":  ohlcData,
		"count": len(ohlcData),
	})
}

// GetForecastHistory4Hour handles GET /api/admin/forecasts/:id/history/4h
func (h *ForecastHandler) GetForecastHistory4Hour(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from URL
	path := strings.TrimPrefix(r.URL.Path, "/api/admin/forecasts/")
	path = strings.TrimSuffix(path, "/history/4h")
	if path == "" {
		http.Error(w, "Forecast ID required", http.StatusBadRequest)
		return
	}
	forecastID := path

	ctx := r.Context()
	ohlcData, err := h.forecastRepo.GetForecastHistory4Hour(ctx, forecastID)
	if err != nil {
		h.logger.Error("Failed to get 4-hour OHLC data", "error", err)
		http.Error(w, "Failed to get 4-hour OHLC data", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data":  ohlcData,
		"count": len(ohlcData),
	})
}

// UpdateForecastSchedule handles PUT /api/admin/forecasts/:id/schedule
func (h *ForecastHandler) UpdateForecastSchedule(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from URL
	path := strings.TrimPrefix(r.URL.Path, "/api/admin/forecasts/")
	path = strings.TrimSuffix(path, "/schedule")
	if path == "" {
		http.Error(w, "Forecast ID required", http.StatusBadRequest)
		return
	}
	forecastID := path

	var req struct {
		Enabled  bool `json:"enabled"`
		Interval int  `json:"interval"` // Interval in minutes
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode request", "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	err := h.forecastRepo.UpdateForecastSchedule(ctx, forecastID, req.Enabled, req.Interval)
	if err != nil {
		h.logger.Error("Failed to update forecast schedule", "error", err)
		http.Error(w, "Failed to update forecast schedule", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Schedule updated successfully",
	})
}

// DeleteForecastRun handles DELETE /api/admin/forecasts/runs/:runId
func (h *ForecastHandler) DeleteForecastRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract run ID from URL
	path := strings.TrimPrefix(r.URL.Path, "/api/admin/forecasts/runs/")
	if path == "" {
		http.Error(w, "Run ID required", http.StatusBadRequest)
		return
	}
	runID := path

	ctx := r.Context()
	err := h.forecastRepo.DeleteForecastRun(ctx, runID)
	if err != nil {
		h.logger.Error("Failed to delete forecast run", "error", err)
		if err.Error() == "forecast run not found" {
			http.Error(w, "Forecast run not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to delete forecast run", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Forecast run deleted successfully",
	})
}

// DeleteAllForecastRuns handles DELETE /api/admin/forecasts/:id/runs
func (h *ForecastHandler) DeleteAllForecastRuns(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from URL path like /api/admin/forecasts/:id/runs
	path := strings.TrimPrefix(r.URL.Path, "/api/admin/forecasts/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] != "runs" {
		http.Error(w, "Invalid forecast ID", http.StatusBadRequest)
		return
	}
	forecastID := parts[0]

	ctx := r.Context()
	err := h.forecastRepo.DeleteAllRunsForForecast(ctx, forecastID)
	if err != nil {
		h.logger.Error("Failed to delete all runs", "error", err, "forecast_id", forecastID)
		http.Error(w, "Failed to delete all runs", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "All forecast runs deleted successfully",
	})
}

// ToggleForecastPublic handles PUT /api/admin/forecasts/:id/public
func (h *ForecastHandler) ToggleForecastPublic(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from URL path like /api/admin/forecasts/:id/public
	path := strings.TrimPrefix(r.URL.Path, "/api/admin/forecasts/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] != "public" {
		http.Error(w, "Invalid forecast ID", http.StatusBadRequest)
		return
	}
	forecastID := parts[0]

	var req struct {
		Public bool `json:"public"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode request", "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	err := h.forecastRepo.UpdateForecastPublic(ctx, forecastID, req.Public)
	if err != nil {
		h.logger.Error("Failed to update forecast public status", "error", err, "forecast_id", forecastID)
		if err.Error() == "forecast not found" {
			http.Error(w, "Forecast not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to update forecast public status", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Forecast public status updated successfully",
		"public":  req.Public,
	})
}

// UpdateForecastDisplayOrder handles PUT /api/admin/forecasts/:id/display-order
func (h *ForecastHandler) UpdateForecastDisplayOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from URL path like /api/admin/forecasts/:id/display-order
	path := strings.TrimPrefix(r.URL.Path, "/api/admin/forecasts/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] != "display-order" {
		http.Error(w, "Invalid forecast ID", http.StatusBadRequest)
		return
	}
	forecastID := parts[0]

	var req struct {
		DisplayOrder int `json:"display_order"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode request", "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	err := h.forecastRepo.UpdateForecastDisplayOrder(ctx, forecastID, req.DisplayOrder)
	if err != nil {
		h.logger.Error("Failed to update forecast display order", "error", err, "forecast_id", forecastID)
		if err.Error() == "forecast not found" {
			http.Error(w, "Forecast not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to update forecast display order", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":       "Forecast display order updated successfully",
		"display_order": req.DisplayOrder,
	})
}

// DeleteForecast handles DELETE /api/admin/forecasts/:id
func (h *ForecastHandler) DeleteForecast(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from URL
	path := strings.TrimPrefix(r.URL.Path, "/api/admin/forecasts/")
	if path == "" || strings.Contains(path, "/") {
		http.Error(w, "Invalid forecast ID", http.StatusBadRequest)
		return
	}
	forecastID := path

	ctx := r.Context()
	err := h.forecastRepo.DeleteForecast(ctx, forecastID)
	if err != nil {
		h.logger.Error("Failed to delete forecast", "error", err)
		if err.Error() == "forecast not found" {
			http.Error(w, "Forecast not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to delete forecast", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Forecast deleted successfully",
	})
}

// ListPublicForecasts handles GET /api/forecasts
func (h *ForecastHandler) ListPublicForecasts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	forecasts, err := h.forecastRepo.ListPublicForecasts(ctx)
	if err != nil {
		h.logger.Error("Failed to list public forecasts", "error", err)
		http.Error(w, "Failed to list public forecasts", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"forecasts": forecasts,
		"count":     len(forecasts),
	})
}

// GetPublicForecastHistory handles GET /api/forecasts/:id/history (public endpoint)
func (h *ForecastHandler) GetPublicForecastHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from URL
	path := strings.TrimPrefix(r.URL.Path, "/api/forecasts/")
	path = strings.TrimSuffix(path, "/history")
	if path == "" {
		http.Error(w, "Forecast ID required", http.StatusBadRequest)
		return
	}
	forecastID := path

	ctx := r.Context()

	// First verify the forecast is public
	forecast, err := h.forecastRepo.GetForecast(ctx, forecastID)
	if err != nil {
		h.logger.Error("Failed to get forecast", "error", err)
		http.Error(w, "Forecast not found", http.StatusNotFound)
		return
	}

	if !forecast.Public {
		http.Error(w, "Forecast not found", http.StatusNotFound)
		return
	}

	history, err := h.forecastRepo.GetForecastHistory(ctx, forecastID)
	if err != nil {
		h.logger.Error("Failed to get forecast history", "error", err)
		http.Error(w, "Failed to get forecast history", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"history": history,
		"count":   len(history),
	})
}

// GetPublicForecastHistoryDaily handles GET /api/forecasts/:id/history/daily (public)
func (h *ForecastHandler) GetPublicForecastHistoryDaily(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from URL
	path := strings.TrimPrefix(r.URL.Path, "/api/forecasts/")
	path = strings.TrimSuffix(path, "/history/daily")
	if path == "" {
		http.Error(w, "Forecast ID required", http.StatusBadRequest)
		return
	}
	forecastID := path

	ctx := r.Context()

	// First verify the forecast is public
	forecast, err := h.forecastRepo.GetForecast(ctx, forecastID)
	if err != nil {
		h.logger.Error("Failed to get forecast", "error", err)
		http.Error(w, "Forecast not found", http.StatusNotFound)
		return
	}

	if !forecast.Public {
		http.Error(w, "Forecast not found", http.StatusNotFound)
		return
	}

	ohlcData, err := h.forecastRepo.GetForecastHistoryDaily(ctx, forecastID)
	if err != nil {
		h.logger.Error("Failed to get daily OHLC data", "error", err)
		http.Error(w, "Failed to get daily OHLC data", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data":  ohlcData,
		"count": len(ohlcData),
	})
}

// GetPublicForecastHistory4Hour handles GET /api/forecasts/:id/history/4h (public)
func (h *ForecastHandler) GetPublicForecastHistory4Hour(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from URL
	path := strings.TrimPrefix(r.URL.Path, "/api/forecasts/")
	path = strings.TrimSuffix(path, "/history/4h")
	if path == "" {
		http.Error(w, "Forecast ID required", http.StatusBadRequest)
		return
	}
	forecastID := path

	ctx := r.Context()

	// First verify the forecast is public
	forecast, err := h.forecastRepo.GetForecast(ctx, forecastID)
	if err != nil {
		h.logger.Error("Failed to get forecast", "error", err)
		http.Error(w, "Forecast not found", http.StatusNotFound)
		return
	}

	if !forecast.Public {
		http.Error(w, "Forecast not found", http.StatusNotFound)
		return
	}

	ohlcData, err := h.forecastRepo.GetForecastHistory4Hour(ctx, forecastID)
	if err != nil {
		h.logger.Error("Failed to get 4-hour OHLC data", "error", err)
		http.Error(w, "Failed to get 4-hour OHLC data", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data":  ohlcData,
		"count": len(ohlcData),
	})
}
