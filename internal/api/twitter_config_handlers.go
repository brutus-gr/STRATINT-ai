package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/STRATINT/stratint/internal/database"
	"github.com/STRATINT/stratint/internal/ingestion"
	"github.com/STRATINT/stratint/internal/models"
	"github.com/STRATINT/stratint/internal/social"
)

type TwitterConfigHandlers struct {
	repo          *database.TwitterRepository
	eventRepo     ingestion.EventRepository
	twitterPoster *social.TwitterPoster
	logger        *slog.Logger
}

func NewTwitterConfigHandlers(repo *database.TwitterRepository, logger *slog.Logger) *TwitterConfigHandlers {
	return &TwitterConfigHandlers{
		repo:   repo,
		logger: logger,
	}
}

// SetTwitterPoster sets the Twitter poster after initialization
func (h *TwitterConfigHandlers) SetTwitterPoster(poster *social.TwitterPoster) {
	h.twitterPoster = poster
}

// SetEventRepo sets the event repository after initialization
func (h *TwitterConfigHandlers) SetEventRepo(repo ingestion.EventRepository) {
	h.eventRepo = repo
}

// GetTwitterConfig handles GET /api/twitter-config
func (h *TwitterConfigHandlers) GetTwitterConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	config, err := h.repo.Get(context.Background())
	if err != nil {
		h.logger.Error("failed to get twitter config", "error", err)
		http.Error(w, "Failed to get Twitter configuration", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(config)
}

// UpdateTwitterConfig handles PUT /api/twitter-config
func (h *TwitterConfigHandlers) UpdateTwitterConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var update models.TwitterConfigUpdate
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate config
	if err := ValidateTwitterConfig(&update); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Update configuration in database
	if err := h.repo.Update(context.Background(), &update); err != nil {
		h.logger.Error("failed to update twitter config", "error", err)
		http.Error(w, "Failed to update Twitter configuration", http.StatusInternalServerError)
		return
	}

	// Get updated config to return
	config, err := h.repo.Get(context.Background())
	if err != nil {
		h.logger.Error("failed to get updated twitter config", "error", err)
		http.Error(w, "Configuration updated but failed to retrieve", http.StatusInternalServerError)
		return
	}

	h.logger.Info("twitter config updated",
		"min_magnitude", update.MinMagnitudeForTweet,
		"min_confidence", update.MinConfidenceForTweet,
		"enabled", update.Enabled,
	)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Twitter configuration updated successfully. Changes are active immediately.",
		"config":  config,
	})
}

// PostEventToTwitter handles POST /api/events/:id/post-to-twitter
func (h *TwitterConfigHandlers) PostEventToTwitter(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract event ID from URL path
	// URL format: /api/events/{id}/post-to-twitter
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/events/"), "/")
	if len(pathParts) < 2 {
		http.Error(w, "Invalid URL format", http.StatusBadRequest)
		return
	}
	eventID := pathParts[0]

	if h.twitterPoster == nil {
		h.logger.Error("twitter poster is nil - service not initialized properly", "event_id", eventID)
		http.Error(w, "Twitter poster not initialized - service may be restarting", http.StatusServiceUnavailable)
		return
	}

	if h.eventRepo == nil {
		h.logger.Error("event repository is nil - service not initialized properly", "event_id", eventID)
		http.Error(w, "Event repository not initialized - service may be restarting", http.StatusServiceUnavailable)
		return
	}

	// Get the event
	event, err := h.eventRepo.GetByID(context.Background(), eventID)
	if err != nil {
		h.logger.Error("failed to get event", "event_id", eventID, "error", err)
		http.Error(w, "Event not found", http.StatusNotFound)
		return
	}

	// Check if already tweeted
	alreadyTweeted, err := h.repo.HasBeenTweeted(context.Background(), eventID)
	if err != nil {
		h.logger.Error("failed to check if event was tweeted", "event_id", eventID, "error", err)
		http.Error(w, "Failed to check tweet status", http.StatusInternalServerError)
		return
	}

	if alreadyTweeted {
		http.Error(w, "Event has already been posted to Twitter", http.StatusConflict)
		return
	}

	// Refresh config to get latest credentials
	err = h.twitterPoster.RefreshConfig(context.Background())
	if err != nil {
		h.logger.Error("failed to refresh twitter config", "event_id", eventID, "error", err)
		http.Error(w, fmt.Sprintf("Failed to load Twitter configuration: %v", err), http.StatusInternalServerError)
		return
	}

	// Post to Twitter
	err = h.twitterPoster.PostTweetForEvent(context.Background(), event)
	if err != nil {
		h.logger.Error("failed to post tweet", "event_id", eventID, "error", err)
		http.Error(w, fmt.Sprintf("Failed to post to Twitter: %v", err), http.StatusInternalServerError)
		return
	}

	h.logger.Info("manually posted event to twitter", "event_id", eventID)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Event successfully posted to Twitter",
	})
}

// GetPostedTweets handles GET /api/admin/posted-tweets
func (h *TwitterConfigHandlers) GetPostedTweets(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Default limit of 100, max 500
	limit := 100
	offset := 0

	// Parse query parameters for pagination
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		fmt.Sscanf(limitStr, "%d", &limit)
		if limit > 500 {
			limit = 500
		}
		if limit < 1 {
			limit = 100
		}
	}
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		fmt.Sscanf(offsetStr, "%d", &offset)
		if offset < 0 {
			offset = 0
		}
	}

	tweets, err := h.repo.GetAllPostedTweets(context.Background(), limit, offset)
	if err != nil {
		h.logger.Error("failed to get posted tweets", "error", err)
		http.Error(w, "Failed to get posted tweets", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(tweets)
}
