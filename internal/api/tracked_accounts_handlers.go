package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/STRATINT/stratint/internal/database"
	"github.com/STRATINT/stratint/internal/enrichment"
	"github.com/STRATINT/stratint/internal/ingestion"
	"github.com/STRATINT/stratint/internal/models"
)

type TrackedAccountsHandler struct {
	repo                models.TrackedAccountRepository
	sourceRepo          ingestion.SourceRepository
	errorRepo           database.IngestionErrorRepository
	activityLogRepo     *database.ActivityLogRepository
	connectorConfigRepo *database.ConnectorConfigRepository
	credibilityCache    *enrichment.CredibilityCache
	enricher            enrichment.Enricher
	logger              *slog.Logger
}

func NewTrackedAccountsHandler(repo models.TrackedAccountRepository, sourceRepo ingestion.SourceRepository, errorRepo database.IngestionErrorRepository, activityLogRepo *database.ActivityLogRepository, connectorConfigRepo *database.ConnectorConfigRepository, credibilityCache *enrichment.CredibilityCache, enricher enrichment.Enricher, logger *slog.Logger) *TrackedAccountsHandler {
	return &TrackedAccountsHandler{
		repo:                repo,
		sourceRepo:          sourceRepo,
		errorRepo:           errorRepo,
		activityLogRepo:     activityLogRepo,
		connectorConfigRepo: connectorConfigRepo,
		credibilityCache:    credibilityCache,
		enricher:            enricher,
		logger:              logger,
	}
}

// ListTrackedAccounts returns all tracked accounts
// GET /api/tracked-accounts?platform=twitter&enabled_only=true
func (h *TrackedAccountsHandler) ListTrackedAccounts(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	platform := r.URL.Query().Get("platform")
	enabledOnly := r.URL.Query().Get("enabled_only") == "true"

	var accounts []*models.TrackedAccount
	var err error

	if platform != "" {
		accounts, err = h.repo.ListByPlatform(platform, enabledOnly)
	} else {
		accounts, err = h.repo.ListAll(enabledOnly)
	}

	if err != nil {
		h.logger.Error("failed to list tracked accounts", "error", err)
		http.Error(w, "Failed to list accounts", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"accounts": accounts,
		"count":    len(accounts),
	})
}

// AddTrackedAccount adds a new account to track
// POST /api/tracked-accounts
// Body: {"platform": "twitter", "account_identifier": "@elonmusk", "display_name": "Elon Musk"}
func (h *TrackedAccountsHandler) AddTrackedAccount(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	var account models.TrackedAccount
	if err := json.NewDecoder(r.Body).Decode(&account); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Set defaults
	if account.FetchIntervalMinutes == 0 {
		account.FetchIntervalMinutes = 5
	}

	// Validate the account
	if err := ValidateTrackedAccount(account.Platform, account.AccountIdentifier, account.FetchIntervalMinutes); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Normalize account identifier
	account.AccountIdentifier = normalizeAccountIdentifier(account.Platform, account.AccountIdentifier)
	if account.Metadata == nil {
		account.Metadata = make(map[string]interface{})
	}
	account.Enabled = true

	if err := h.repo.Store(&account); err != nil {
		h.logger.Error("failed to store tracked account", "error", err)
		http.Error(w, "Failed to store account", http.StatusInternalServerError)
		return
	}

	h.logger.Info("added tracked account",
		"platform", account.Platform,
		"identifier", account.AccountIdentifier,
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(account)
}

// GetTrackedAccount returns a specific tracked account
// GET /api/tracked-accounts/:id
func (h *TrackedAccountsHandler) GetTrackedAccount(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/tracked-accounts/")

	account, err := h.repo.GetByID(id)
	if err != nil {
		h.logger.Error("failed to get tracked account", "error", err)
		http.Error(w, "Failed to get account", http.StatusInternalServerError)
		return
	}

	if account == nil {
		http.Error(w, "Account not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(account)
}

// UpdateTrackedAccount updates an existing account
// PUT /api/tracked-accounts/:id
func (h *TrackedAccountsHandler) UpdateTrackedAccount(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/tracked-accounts/")

	var updates models.TrackedAccount
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get existing account
	existing, err := h.repo.GetByID(id)
	if err != nil {
		h.logger.Error("failed to get tracked account", "error", err)
		http.Error(w, "Failed to get account", http.StatusInternalServerError)
		return
	}

	if existing == nil {
		http.Error(w, "Account not found", http.StatusNotFound)
		return
	}

	// Update fields
	existing.DisplayName = updates.DisplayName
	existing.FetchIntervalMinutes = updates.FetchIntervalMinutes
	if updates.Metadata != nil {
		existing.Metadata = updates.Metadata
	}

	if err := h.repo.Store(existing); err != nil {
		h.logger.Error("failed to update tracked account", "error", err)
		http.Error(w, "Failed to update account", http.StatusInternalServerError)
		return
	}

	h.logger.Info("updated tracked account", "id", id)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(existing)
}

// DeleteTrackedAccount removes an account from tracking
// DELETE /api/tracked-accounts/:id
func (h *TrackedAccountsHandler) DeleteTrackedAccount(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/tracked-accounts/")

	if err := h.repo.Delete(id); err != nil {
		h.logger.Error("failed to delete tracked account", "error", err)
		http.Error(w, "Failed to delete account", http.StatusInternalServerError)
		return
	}

	h.logger.Info("deleted tracked account", "id", id)

	w.WriteHeader(http.StatusNoContent)
}

// ToggleTrackedAccount enables/disables an account
// POST /api/tracked-accounts/:id/toggle
func (h *TrackedAccountsHandler) ToggleTrackedAccount(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/tracked-accounts/")
	id = strings.TrimSuffix(id, "/toggle")

	var body struct {
		Enabled bool `json:"enabled"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.repo.SetEnabled(id, body.Enabled); err != nil {
		h.logger.Error("failed to toggle tracked account", "error", err)
		http.Error(w, "Failed to toggle account", http.StatusInternalServerError)
		return
	}

	h.logger.Info("toggled tracked account", "id", id, "enabled", body.Enabled)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":      id,
		"enabled": body.Enabled,
	})
}

// FetchNow triggers an immediate fetch for a tracked account
// POST /api/tracked-accounts/:id/fetch
func (h *TrackedAccountsHandler) FetchNow(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	id := strings.TrimPrefix(r.URL.Path, "/api/tracked-accounts/")
	id = strings.TrimSuffix(id, "/fetch")

	// Get the tracked account
	account, err := h.repo.GetByID(id)
	if err != nil {
		h.logger.Error("failed to get tracked account", "error", err)
		http.Error(w, "Failed to get account", http.StatusInternalServerError)
		return
	}

	if account == nil {
		http.Error(w, "Account not found", http.StatusNotFound)
		return
	}

	// Fetch based on platform
	var sources []*models.Source
	ctx := context.Background()

	switch account.Platform {
	case "twitter":
		// Get Twitter config from database
		twitterConfig, err := h.connectorConfigRepo.Get(ctx, "twitter")
		if err != nil || !twitterConfig.Enabled {
			h.logger.Error("Twitter not configured or disabled", "error", err)
			http.Error(w, "Twitter not configured", http.StatusServiceUnavailable)
			return
		}

		bearerToken := twitterConfig.Config["bearer_token"]
		if bearerToken == "" {
			http.Error(w, "Twitter bearer token not configured", http.StatusServiceUnavailable)
			return
		}

		h.logger.Info("manual fetch triggered", "platform", "twitter", "account", account.AccountIdentifier)
		twitterConnector := ingestion.NewTwitterConnector(bearerToken, h.logger, h.credibilityCache)
		sources, err = twitterConnector.FetchAccountTweets(account)
		if err != nil {
			h.logger.Error("failed to fetch tweets", "account", account.AccountIdentifier, "error", err)
			http.Error(w, "Failed to fetch tweets: "+err.Error(), http.StatusInternalServerError)
			return
		}

	case "rss":
		h.logger.Info("manual fetch triggered", "platform", "rss", "feed", account.AccountIdentifier)
		rssConnector, err := ingestion.NewRSSConnector([]string{account.AccountIdentifier}, h.logger, h.errorRepo, h.activityLogRepo)
		if err != nil {
			h.logger.Error("failed to create rss connector", "feed", account.AccountIdentifier, "error", err)
			http.Error(w, "Failed to create RSS connector: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer rssConnector.Close()

		rssSources, err := rssConnector.Fetch()
		if err != nil {
			h.logger.Error("failed to fetch rss feed", "feed", account.AccountIdentifier, "error", err)
			http.Error(w, "Failed to fetch RSS feed: "+err.Error(), http.StatusInternalServerError)
			return
		}
		// Convert []models.Source to []*models.Source
		for i := range rssSources {
			sources = append(sources, &rssSources[i])
		}

	default:
		http.Error(w, "Unsupported platform", http.StatusBadRequest)
		return
	}

	// Store sources (check for duplicates first)
	storedCount := 0
	skippedCount := 0
	for i, source := range sources {
		h.logger.Info("processing source",
			"index", i,
			"title", source.Title,
			"url", source.URL,
			"has_title", source.Title != "",
			"has_url", source.URL != "")

		// Check for duplicates by title + URL
		if source.Title != "" && source.URL != "" {
			existing, err := h.sourceRepo.GetByTitleAndURL(ctx, source.Title, source.URL)
			if err != nil {
				h.logger.Error("failed to check for duplicate source", "error", err, "title", source.Title)
				continue
			}
			if existing != nil {
				h.logger.Info("skipping duplicate source",
					"title", source.Title,
					"url", source.URL,
					"existing_id", existing.ID)
				skippedCount++
				continue
			}
			h.logger.Info("no duplicate found, storing source", "title", source.Title)
		} else {
			h.logger.Warn("source missing title or URL, storing anyway",
				"title", source.Title,
				"url", source.URL)
		}

		if err := h.sourceRepo.StoreRaw(ctx, *source); err != nil {
			h.logger.Error("failed to store source", "error", err, "title", source.Title)
		} else {
			storedCount++
			h.logger.Info("successfully stored source", "title", source.Title, "url", source.URL)
		}
	}

	// Update last fetched timestamp
	if len(sources) > 0 {
		var latestID string
		switch account.Platform {
		case "twitter":
			latestID = ingestion.GetLatestTweetID(sources)
		case "rss":
			// For RSS, use the URL of the most recent article as the ID
			if len(sources) > 0 {
				latestID = sources[0].URL
			}
		}

		if latestID != "" {
			h.repo.UpdateLastFetched(account.ID, latestID, time.Now())
		}
	}

	h.logger.Info("manual fetch complete",
		"account", account.AccountIdentifier,
		"platform", account.Platform,
		"fetched", len(sources),
		"stored", storedCount,
		"skipped", skippedCount)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"fetched": len(sources),
		"stored":  storedCount,
		"message": "Fetch triggered successfully. Sources will be enriched in the background.",
	})
}

// normalizeAccountIdentifier standardizes account identifiers
func normalizeAccountIdentifier(platform, identifier string) string {
	switch platform {
	case "twitter":
		// Ensure Twitter handles start with @
		if !strings.HasPrefix(identifier, "@") {
			return "@" + identifier
		}
		return identifier
	case "telegram":
		// Telegram channels start with @
		if !strings.HasPrefix(identifier, "@") && !strings.HasPrefix(identifier, "https://") {
			return "@" + identifier
		}
		return identifier
	default:
		return identifier
	}
}
