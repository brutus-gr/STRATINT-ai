package api

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/STRATINT/stratint/internal/database"
)

// ConnectorConfigHandlers manages connector configuration endpoints
type ConnectorConfigHandlers struct {
	repo   *database.ConnectorConfigRepository
	logger *slog.Logger
}

// NewConnectorConfigHandlers creates connector config handlers
func NewConnectorConfigHandlers(repo *database.ConnectorConfigRepository, logger *slog.Logger) *ConnectorConfigHandlers {
	return &ConnectorConfigHandlers{
		repo:   repo,
		logger: logger,
	}
}

// ConnectorConfig represents configuration for a connector
type ConnectorConfig struct {
	Config map[string]string `json:"config"`
}

// ConnectorListResponse represents a connector in list responses
type ConnectorListResponse struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
	Status  string `json:"status"`
}

// ListConnectors lists all available connectors
func (h *ConnectorConfigHandlers) ListConnectors(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("listing all connectors")

	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	// Get all connectors from database
	ctx := context.Background()
	configs, err := h.repo.GetAll(ctx)
	if err != nil {
		h.logger.Error("failed to get all connector configs", "error", err)
		http.Error(w, "Failed to get connector configurations", http.StatusInternalServerError)
		return
	}

	// Map connector IDs to friendly names
	nameMap := map[string]string{
		"twitter":  "Twitter API",
		"telegram": "Telegram Bot",
		"rss":      "RSS Feeds",
	}

	// Build response
	var connectors []ConnectorListResponse
	for _, config := range configs {
		name := nameMap[config.ID]
		if name == "" {
			name = config.ID
		}
		status := "disabled"
		if config.Enabled {
			status = "active"
		}
		connectors = append(connectors, ConnectorListResponse{
			ID:      config.ID,
			Name:    name,
			Enabled: config.Enabled,
			Status:  status,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"connectors": connectors,
	})
}

// ToggleConnector toggles a connector's enabled status
func (h *ConnectorConfigHandlers) ToggleConnector(w http.ResponseWriter, r *http.Request) {
	// Extract connector ID from URL path: /api/connectors/{id}/toggle
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 3 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	connectorID := pathParts[2]

	h.logger.Info("toggling connector", "connector_id", connectorID)

	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	// Get current config
	ctx := context.Background()
	config, err := h.repo.Get(ctx, connectorID)
	if err != nil {
		h.logger.Error("failed to get connector config", "error", err)
		http.Error(w, "Failed to get connector configuration", http.StatusInternalServerError)
		return
	}

	// Toggle enabled status
	newEnabled := !config.Enabled
	err = h.repo.SetEnabled(ctx, connectorID, newEnabled)
	if err != nil {
		h.logger.Error("failed to toggle connector", "error", err)
		http.Error(w, "Failed to toggle connector", http.StatusInternalServerError)
		return
	}

	h.logger.Info("connector toggled", "connector_id", connectorID, "enabled", newEnabled)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "Connector toggled successfully",
		"enabled": newEnabled,
	})
}

// GetConnectorConfig retrieves configuration for a specific connector
func (h *ConnectorConfigHandlers) GetConnectorConfig(w http.ResponseWriter, r *http.Request) {
	// Extract connector ID from URL path: /api/connectors/{id}/config
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 3 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	connectorID := pathParts[2]

	h.logger.Info("fetching connector config", "connector_id", connectorID)

	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	// Get config from database
	ctx := context.Background()
	dbConfig, err := h.repo.Get(ctx, connectorID)
	if err != nil {
		h.logger.Error("failed to get connector config", "error", err)
		http.Error(w, "Failed to get connector configuration", http.StatusInternalServerError)
		return
	}

	// Mask sensitive values for security
	maskedConfig := make(map[string]string)
	for key, value := range dbConfig.Config {
		if strings.Contains(strings.ToLower(key), "token") || strings.Contains(strings.ToLower(key), "key") || strings.Contains(strings.ToLower(key), "secret") {
			if value != "" && len(value) > 4 {
				maskedConfig[key] = "***" + value[len(value)-4:]
			} else if value != "" {
				maskedConfig[key] = "***"
			} else {
				maskedConfig[key] = ""
			}
		} else {
			maskedConfig[key] = value
		}
	}

	response := ConnectorConfig{Config: maskedConfig}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// UpdateConnectorConfig updates configuration for a specific connector
func (h *ConnectorConfigHandlers) UpdateConnectorConfig(w http.ResponseWriter, r *http.Request) {
	// Extract connector ID from URL path: /api/connectors/{id}/config
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 3 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	connectorID := pathParts[2]

	h.logger.Info("updating connector config", "connector_id", connectorID)

	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	// Parse request
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Error("failed to read body", "error", err)
		http.Error(w, "Failed to read request", http.StatusBadRequest)
		return
	}

	var request ConnectorConfig
	if err := json.Unmarshal(body, &request); err != nil {
		h.logger.Error("failed to parse json", "error", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Update config in database
	ctx := context.Background()
	_, err = h.repo.Update(ctx, connectorID, nil, request.Config)
	if err != nil {
		h.logger.Error("failed to update connector config", "error", err)
		http.Error(w, "Failed to update configuration", http.StatusInternalServerError)
		return
	}

	h.logger.Info("connector config updated", "connector_id", connectorID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Configuration updated successfully.",
	})
}
