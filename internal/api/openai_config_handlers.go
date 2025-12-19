package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/STRATINT/stratint/internal/database"
	"github.com/STRATINT/stratint/internal/models"
)

type OpenAIConfigHandlers struct {
	repo   *database.OpenAIConfigRepository
	logger *slog.Logger
}

func NewOpenAIConfigHandlers(repo *database.OpenAIConfigRepository, logger *slog.Logger) *OpenAIConfigHandlers {
	return &OpenAIConfigHandlers{
		repo:   repo,
		logger: logger,
	}
}

// GetOpenAIConfig handles GET /api/openai-config
func (h *OpenAIConfigHandlers) GetOpenAIConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	config, err := h.repo.Get(context.Background())
	if err != nil {
		h.logger.Error("failed to get openai config", "error", err)
		http.Error(w, "Failed to get OpenAI configuration", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(config)
}

// UpdateOpenAIConfig handles PUT /api/openai-config
func (h *OpenAIConfigHandlers) UpdateOpenAIConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var update models.OpenAIConfigUpdate
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get current config to validate the complete config
	currentConfig, err := h.repo.Get(context.Background())
	if err != nil {
		h.logger.Error("failed to get current config", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Apply updates to a copy for validation
	testConfig := *currentConfig
	if update.APIKey != nil {
		testConfig.APIKey = *update.APIKey
	}
	if update.Model != nil {
		testConfig.Model = *update.Model
	}
	if update.Temperature != nil {
		testConfig.Temperature = *update.Temperature
	}
	if update.MaxTokens != nil {
		testConfig.MaxTokens = *update.MaxTokens
	}
	if update.TimeoutSeconds != nil {
		testConfig.TimeoutSeconds = *update.TimeoutSeconds
	}

	// Validate the config
	if err := ValidateOpenAIConfig(&testConfig); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Update configuration in database
	config, err := h.repo.Update(context.Background(), update)
	if err != nil {
		h.logger.Error("failed to update openai config", "error", err)
		http.Error(w, "Failed to update OpenAI configuration", http.StatusInternalServerError)
		return
	}

	h.logger.Info("openai config updated",
		"model", config.Model,
		"temperature", config.Temperature,
		"enabled", config.Enabled,
	)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "OpenAI configuration updated successfully. Changes will apply to new sources.",
		"config":  config,
	})
}

// TestOpenAIConfig handles POST /api/openai-config/test
func (h *OpenAIConfigHandlers) TestOpenAIConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := h.repo.TestConnection(context.Background())
	if err != nil {
		h.logger.Error("openai config test failed", "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Configuration test failed: " + err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "OpenAI configuration is valid",
	})
}
