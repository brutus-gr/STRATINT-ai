package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/STRATINT/stratint/internal/database"
	"github.com/STRATINT/stratint/internal/models"
	"github.com/STRATINT/stratint/internal/strategist"
)

// StrategyHandler handles HTTP requests for strategy management
type StrategyHandler struct {
	repo       *database.StrategyRepository
	strategist *strategist.Strategist
	logger     *slog.Logger
}

// NewStrategyHandler creates a new StrategyHandler
func NewStrategyHandler(repo *database.StrategyRepository, s *strategist.Strategist, logger *slog.Logger) *StrategyHandler {
	return &StrategyHandler{
		repo:       repo,
		strategist: s,
		logger:     logger,
	}
}

// CreateStrategy handles POST /api/admin/strategies
func (h *StrategyHandler) CreateStrategy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.CreateStrategyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("failed to decode create strategy request", "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Name == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}
	if req.Prompt == "" {
		http.Error(w, "Prompt is required", http.StatusBadRequest)
		return
	}
	if len(req.InvestmentSymbols) == 0 {
		http.Error(w, "At least one investment symbol is required", http.StatusBadRequest)
		return
	}
	if req.HeadlineCount <= 0 {
		req.HeadlineCount = 500 // Default
	}
	if req.Iterations <= 0 {
		req.Iterations = 3 // Default
	}
	if len(req.Models) == 0 {
		http.Error(w, "At least one model is required", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	strategy, err := h.repo.CreateStrategy(ctx, req)
	if err != nil {
		h.logger.Error("failed to create strategy", "error", err)
		http.Error(w, "Failed to create strategy", http.StatusInternalServerError)
		return
	}

	h.logger.Info("strategy created", "id", strategy.ID, "name", strategy.Name)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(strategy)
}

// ListStrategies handles GET /api/admin/strategies
func (h *StrategyHandler) ListStrategies(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := context.Background()
	strategies, err := h.repo.ListStrategies(ctx)
	if err != nil {
		h.logger.Error("failed to list strategies", "error", err)
		http.Error(w, "Failed to list strategies: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(strategies)
}

// GetStrategy handles GET /api/admin/strategies/{id}
func (h *StrategyHandler) GetStrategy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from URL
	id := strings.TrimPrefix(r.URL.Path, "/api/admin/strategies/")
	if id == "" || strings.Contains(id, "/") {
		http.Error(w, "Invalid strategy ID", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	strategy, err := h.repo.GetStrategy(ctx, id)
	if err != nil {
		h.logger.Error("failed to get strategy", "id", id, "error", err)
		http.Error(w, "Strategy not found", http.StatusNotFound)
		return
	}

	// Fetch associated models
	strategyModels, err := h.repo.GetStrategyModels(ctx, id)
	if err != nil {
		h.logger.Error("failed to get strategy models", "id", id, "error", err)
		// Don't fail the request, just log the error and return strategy without models
		strategyModels = []models.StrategyModel{}
	}
	strategy.Models = strategyModels

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(strategy)
}

// UpdateStrategy handles PUT /api/admin/strategies/{id}
func (h *StrategyHandler) UpdateStrategy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from URL path
	id := strings.TrimPrefix(r.URL.Path, "/api/admin/strategies/")
	if id == "" {
		http.Error(w, "Strategy ID is required", http.StatusBadRequest)
		return
	}

	var req models.CreateStrategyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("failed to decode update strategy request", "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Name == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}
	if req.Prompt == "" {
		http.Error(w, "Prompt is required", http.StatusBadRequest)
		return
	}
	if len(req.InvestmentSymbols) == 0 {
		http.Error(w, "At least one investment symbol is required", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	strategy, err := h.repo.UpdateStrategy(ctx, id, req)
	if err != nil {
		h.logger.Error("failed to update strategy", "id", id, "error", err)
		http.Error(w, "Failed to update strategy", http.StatusInternalServerError)
		return
	}

	h.logger.Info("strategy updated", "id", strategy.ID, "name", strategy.Name)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(strategy)
}

// DeleteStrategy handles DELETE /api/admin/strategies/{id}
func (h *StrategyHandler) DeleteStrategy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from URL path
	id := strings.TrimPrefix(r.URL.Path, "/api/admin/strategies/")
	if id == "" {
		http.Error(w, "Strategy ID is required", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	if err := h.repo.DeleteStrategy(ctx, id); err != nil {
		h.logger.Error("failed to delete strategy", "id", id, "error", err)
		http.Error(w, "Failed to delete strategy", http.StatusInternalServerError)
		return
	}

	h.logger.Info("strategy deleted", "id", id)

	w.WriteHeader(http.StatusNoContent)
}

// ExecuteStrategy handles POST /api/admin/strategies/{id}/execute
func (h *StrategyHandler) ExecuteStrategy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from URL
	path := strings.TrimPrefix(r.URL.Path, "/api/admin/strategies/")
	id := strings.TrimSuffix(path, "/execute")
	if id == "" {
		http.Error(w, "Strategy ID is required", http.StatusBadRequest)
		return
	}

	ctx := context.Background()

	// Verify strategy exists
	strategy, err := h.repo.GetStrategy(ctx, id)
	if err != nil {
		h.logger.Error("failed to get strategy for execution", "id", id, "error", err)
		http.Error(w, "Strategy not found", http.StatusNotFound)
		return
	}

	if !strategy.Active {
		http.Error(w, "Strategy is not active", http.StatusBadRequest)
		return
	}

	// Execute strategy (this runs asynchronously)
	runID, err := h.strategist.ExecuteStrategy(ctx, id)
	if err != nil {
		h.logger.Error("failed to execute strategy", "id", id, "error", err)
		http.Error(w, "Failed to execute strategy", http.StatusInternalServerError)
		return
	}

	h.logger.Info("strategy execution started", "strategy_id", id, "run_id", runID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"run_id": runID,
		"status": "pending",
	})
}

// GetStrategyRuns handles GET /api/admin/strategies/{id}/runs
func (h *StrategyHandler) GetStrategyRuns(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from URL
	path := strings.TrimPrefix(r.URL.Path, "/api/admin/strategies/")
	id := strings.TrimSuffix(path, "/runs")
	if id == "" {
		http.Error(w, "Strategy ID is required", http.StatusBadRequest)
		return
	}

	// Get limit from query params (default 10)
	limitStr := r.URL.Query().Get("limit")
	limit := 10
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	ctx := context.Background()
	runs, err := h.repo.ListStrategyRuns(ctx, id, limit)
	if err != nil {
		h.logger.Error("failed to list strategy runs", "strategy_id", id, "error", err)
		http.Error(w, "Failed to list strategy runs", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(runs)
}

// GetStrategyRun handles GET /api/admin/strategies/runs/{runId}
func (h *StrategyHandler) GetStrategyRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract run ID from URL
	runID := strings.TrimPrefix(r.URL.Path, "/api/admin/strategies/runs/")
	if runID == "" {
		http.Error(w, "Run ID is required", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	runDetail, err := h.repo.GetStrategyRun(ctx, runID)
	if err != nil {
		h.logger.Error("failed to get strategy run", "run_id", runID, "error", err)
		http.Error(w, "Run not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(runDetail)
}

// UpdateStrategyPublic handles PUT /api/admin/strategies/{id}/publish
func (h *StrategyHandler) UpdateStrategyPublic(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from URL
	path := strings.TrimPrefix(r.URL.Path, "/api/admin/strategies/")
	id := strings.TrimSuffix(path, "/publish")
	if id == "" {
		http.Error(w, "Strategy ID is required", http.StatusBadRequest)
		return
	}

	var req struct {
		Public bool `json:"public"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("failed to decode update public request", "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	if err := h.repo.UpdateStrategyPublic(ctx, id, req.Public); err != nil {
		h.logger.Error("failed to update strategy public status", "id", id, "error", err)
		http.Error(w, "Failed to update strategy", http.StatusInternalServerError)
		return
	}

	h.logger.Info("strategy public status updated", "id", id, "public", req.Public)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"public": req.Public})
}

// UpdateStrategyDisplayOrder handles PUT /api/admin/strategies/{id}/order
func (h *StrategyHandler) UpdateStrategyDisplayOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from URL
	path := strings.TrimPrefix(r.URL.Path, "/api/admin/strategies/")
	id := strings.TrimSuffix(path, "/order")
	if id == "" {
		http.Error(w, "Strategy ID is required", http.StatusBadRequest)
		return
	}

	var req struct {
		DisplayOrder int `json:"display_order"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("failed to decode update display order request", "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	if err := h.repo.UpdateStrategyDisplayOrder(ctx, id, req.DisplayOrder); err != nil {
		h.logger.Error("failed to update strategy display order", "id", id, "error", err)
		http.Error(w, "Failed to update strategy", http.StatusInternalServerError)
		return
	}

	h.logger.Info("strategy display order updated", "id", id, "order", req.DisplayOrder)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"display_order": req.DisplayOrder})
}

// UpdateStrategySchedule handles PUT /api/admin/strategies/{id}/schedule
func (h *StrategyHandler) UpdateStrategySchedule(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from URL
	path := strings.TrimPrefix(r.URL.Path, "/api/admin/strategies/")
	id := strings.TrimSuffix(path, "/schedule")
	if id == "" {
		http.Error(w, "Strategy ID is required", http.StatusBadRequest)
		return
	}

	var req struct {
		ScheduleEnabled  bool `json:"schedule_enabled"`
		ScheduleInterval int  `json:"schedule_interval"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("failed to decode update schedule request", "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate schedule interval if enabled
	if req.ScheduleEnabled && req.ScheduleInterval <= 0 {
		http.Error(w, "Schedule interval must be greater than 0 when scheduling is enabled", http.StatusBadRequest)
		return
	}

	ctx := context.Background()

	// Calculate next run time if enabling schedule
	var nextRunAt *time.Time
	if req.ScheduleEnabled {
		next := time.Now().Add(time.Duration(req.ScheduleInterval) * time.Minute)
		nextRunAt = &next
	}

	if err := h.repo.UpdateStrategySchedule(ctx, id, req.ScheduleEnabled, req.ScheduleInterval, nextRunAt); err != nil {
		h.logger.Error("failed to update strategy schedule", "id", id, "error", err, "schedule_enabled", req.ScheduleEnabled, "schedule_interval", req.ScheduleInterval, "next_run_at", nextRunAt)
		// Return detailed error to help diagnose the issue
		errMsg := fmt.Sprintf("Failed to update strategy schedule: %v (id=%s, enabled=%v, interval=%d, next_run_at=%v)", err, id, req.ScheduleEnabled, req.ScheduleInterval, nextRunAt)
		http.Error(w, errMsg, http.StatusInternalServerError)
		return
	}

	h.logger.Info("strategy schedule updated", "id", id, "enabled", req.ScheduleEnabled, "interval", req.ScheduleInterval)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"schedule_enabled":  req.ScheduleEnabled,
		"schedule_interval": req.ScheduleInterval,
		"next_run_at":       nextRunAt,
	})
}

// --- Public endpoints ---

// ListPublicStrategies handles GET /api/strategies
func (h *StrategyHandler) ListPublicStrategies(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := context.Background()
	strategies, err := h.repo.ListPublicStrategies(ctx)
	if err != nil {
		h.logger.Error("failed to list public strategies", "error", err)
		http.Error(w, "Failed to list strategies: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(strategies)
}

// GetPublicStrategy handles GET /api/strategies/{id}
func (h *StrategyHandler) GetPublicStrategy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from URL
	id := strings.TrimPrefix(r.URL.Path, "/api/strategies/")
	if id == "" {
		http.Error(w, "Strategy ID is required", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	strategy, err := h.repo.GetStrategy(ctx, id)
	if err != nil {
		h.logger.Error("failed to get public strategy", "id", id, "error", err)
		http.Error(w, "Strategy not found", http.StatusNotFound)
		return
	}

	// Verify it's public
	if !strategy.Public {
		http.Error(w, "Strategy not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(strategy)
}

// GetLatestStrategyResult handles GET /api/strategies/{id}/latest
func (h *StrategyHandler) GetLatestStrategyResult(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from URL
	path := strings.TrimPrefix(r.URL.Path, "/api/strategies/")
	id := strings.TrimSuffix(path, "/latest")
	if id == "" {
		http.Error(w, "Strategy ID is required", http.StatusBadRequest)
		return
	}

	ctx := context.Background()

	// Verify strategy exists and is public
	strategy, err := h.repo.GetStrategy(ctx, id)
	if err != nil {
		h.logger.Error("failed to get strategy for latest result", "id", id, "error", err)
		http.Error(w, "Strategy not found", http.StatusNotFound)
		return
	}

	if !strategy.Public {
		http.Error(w, "Strategy not found", http.StatusNotFound)
		return
	}

	// Get latest completed run
	latestRun, err := h.repo.GetLatestStrategyResult(ctx, id)
	if err != nil {
		h.logger.Error("failed to get latest strategy result", "id", id, "error", err)
		http.Error(w, "No results found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(latestRun)
}
