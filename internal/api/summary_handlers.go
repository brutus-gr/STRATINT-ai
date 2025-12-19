package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/STRATINT/stratint/internal/database"
	"github.com/STRATINT/stratint/internal/models"
)

type SummaryHandler struct {
	repo     *database.SummaryRepository
	executor *SummaryExecutor
	logger   *slog.Logger
}

type TwitterClient interface {
	PostTweet(text string) (tweetID string, err error)
}

func NewSummaryHandler(repo *database.SummaryRepository, executor *SummaryExecutor, logger *slog.Logger) *SummaryHandler {
	return &SummaryHandler{
		repo:     repo,
		executor: executor,
		logger:   logger,
	}
}

// List summaries
func (h *SummaryHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	summaries, err := h.repo.List(context.Background())
	if err != nil {
		h.logger.Error("failed to list summaries", "error", err)
		http.Error(w, "Failed to list summaries", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summaries)
}

// Get summary by ID
func (h *SummaryHandler) Get(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/api/admin/summaries/")
	if id == "" {
		http.Error(w, "ID is required", http.StatusBadRequest)
		return
	}

	summary, err := h.repo.Get(context.Background(), id)
	if err != nil {
		http.Error(w, "Summary not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

// Create summary
func (h *SummaryHandler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var summary models.Summary
	if err := json.NewDecoder(r.Body).Decode(&summary); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.repo.Create(context.Background(), &summary); err != nil {
		h.logger.Error("failed to create summary", "error", err)
		http.Error(w, "Failed to create summary", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(summary)
}

// Update summary
func (h *SummaryHandler) Update(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/api/admin/summaries/")
	if id == "" {
		http.Error(w, "ID is required", http.StatusBadRequest)
		return
	}

	var summary models.Summary
	if err := json.NewDecoder(r.Body).Decode(&summary); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	summary.ID = id
	if err := h.repo.Update(context.Background(), &summary); err != nil {
		h.logger.Error("failed to update summary", "error", err)
		http.Error(w, "Failed to update summary", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(summary)
}

// Delete summary
func (h *SummaryHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/api/admin/summaries/")
	if id == "" {
		http.Error(w, "ID is required", http.StatusBadRequest)
		return
	}

	if err := h.repo.Delete(context.Background(), id); err != nil {
		h.logger.Error("failed to delete summary", "error", err)
		http.Error(w, "Failed to delete summary", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Clone summary
func (h *SummaryHandler) Clone(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/api/admin/summaries/")
	id = strings.TrimSuffix(id, "/clone")

	// Get the original summary
	original, err := h.repo.Get(context.Background(), id)
	if err != nil {
		http.Error(w, "Summary not found", http.StatusNotFound)
		return
	}

	// Create a clone with modified name
	// Deep copy categories and models to avoid sharing slices
	categories := make([]string, len(original.Categories))
	copy(categories, original.Categories)

	summaryModels := make([]models.SummaryModel, len(original.Models))
	copy(summaryModels, original.Models)

	// Copy TimeOfDay if it exists and parse it properly
	var timeOfDay *string
	if original.TimeOfDay != nil {
		// Parse the time to handle any timestamp format and extract just the time portion
		t, err := time.Parse(time.RFC3339, *original.TimeOfDay)
		if err != nil {
			// If parsing fails, try as just a time string
			t, err = time.Parse("15:04:05", *original.TimeOfDay)
			if err != nil {
				// If still fails, try HH:MM format
				t, err = time.Parse("15:04", *original.TimeOfDay)
			}
		}
		if err == nil {
			// Format as HH:MM:SS which PostgreSQL TIME accepts
			tod := t.Format("15:04:05")
			timeOfDay = &tod
		}
		// If all parsing fails, leave timeOfDay as nil
	}

	clone := &models.Summary{
		Name:              original.Name + " (Copy)",
		Prompt:            original.Prompt,
		TimeOfDay:         timeOfDay,
		LookbackHours:     original.LookbackHours,
		Categories:        categories,
		HeadlineCount:     original.HeadlineCount,
		Models:            summaryModels,
		Active:            original.Active,
		ScheduleEnabled:   false, // Disable schedule for clones
		ScheduleInterval:  original.ScheduleInterval,
		AutoPostToTwitter: original.AutoPostToTwitter,
		IncludeForecasts:  original.IncludeForecasts,
	}

	if err := h.repo.Create(context.Background(), clone); err != nil {
		h.logger.Error("failed to clone summary", "error", err)
		http.Error(w, "Failed to clone summary", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(clone)
}

// Execute summary
func (h *SummaryHandler) Execute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/api/admin/summaries/")
	id = strings.TrimSuffix(id, "/execute")

	runID, err := h.executor.Execute(context.Background(), id)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to execute summary: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"run_id": runID, "status": "started"})
}

// Get latest run
func (h *SummaryHandler) GetLatestRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/admin/summaries/")
	id := strings.TrimSuffix(path, "/latest")

	runDetail, err := h.repo.GetLatestRun(context.Background(), id)
	if err != nil {
		http.Error(w, "No runs found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(runDetail)
}

// List runs
func (h *SummaryHandler) ListRuns(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/admin/summaries/")
	id := strings.TrimSuffix(path, "/runs")

	runs, err := h.repo.ListRuns(context.Background(), id)
	if err != nil {
		http.Error(w, "Failed to list runs", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(runs)
}

// Get run by ID
func (h *SummaryHandler) GetRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	runID := strings.TrimPrefix(r.URL.Path, "/api/admin/summaries/runs/")
	if runID == "" {
		http.Error(w, "Run ID is required", http.StatusBadRequest)
		return
	}

	runDetail, err := h.repo.GetRunByID(context.Background(), runID)
	if err != nil {
		http.Error(w, "Run not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(runDetail)
}

// PostToTwitter posts a summary result to Twitter
func (h *SummaryHandler) PostToTwitter(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request body to get result ID
	var req struct {
		ResultID string `json:"result_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.ResultID == "" {
		http.Error(w, "result_id is required", http.StatusBadRequest)
		return
	}

	// Get the run ID from the path
	runID := strings.TrimPrefix(r.URL.Path, "/api/admin/summaries/runs/")
	runID = strings.TrimSuffix(runID, "/tweet")

	// Get run detail to find the specific result
	runDetail, err := h.repo.GetRunByID(context.Background(), runID)
	if err != nil {
		http.Error(w, "Run not found", http.StatusNotFound)
		return
	}

	// Find the specific result
	var summaryText string
	var modelName string
	for _, result := range runDetail.Results {
		if result.ID == req.ResultID {
			summaryText = result.SummaryText
			modelName = result.ModelName
			break
		}
	}

	if summaryText == "" {
		http.Error(w, "Result not found", http.StatusNotFound)
		return
	}

	// Check if Twitter client is configured
	if h.executor.TwitterClient == nil {
		http.Error(w, "Twitter integration not configured", http.StatusServiceUnavailable)
		return
	}

	// Post to Twitter
	tweetID, err := h.executor.TwitterClient.PostTweet(summaryText)
	if err != nil {
		h.logger.Error("failed to post to twitter", "error", err)
		http.Error(w, fmt.Sprintf("Failed to post to Twitter: %v", err), http.StatusInternalServerError)
		return
	}

	// Record the tweet in database
	ctx := context.Background()
	tweetURL := fmt.Sprintf("https://twitter.com/i/web/status/%s", tweetID)

	config, err := h.executor.TwitterRepo.Get(ctx)
	if err == nil && config != nil {
		if err := h.executor.TwitterRepo.RecordPostedTweet(ctx, "", tweetID, summaryText); err != nil {
			h.logger.Error("failed to record tweet", "error", err)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"tweet_id":  tweetID,
		"tweet_url": tweetURL,
		"model":     modelName,
	})
}
