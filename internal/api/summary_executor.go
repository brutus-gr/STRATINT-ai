package api

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/STRATINT/stratint/internal/database"
	"github.com/STRATINT/stratint/internal/models"
	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/sashabaranov/go-openai"
)

// TwitterPoster interface for posting tweets
type TwitterPoster interface {
	PostTweet(text string) (tweetID string, err error)
}

// SummaryExecutor handles executing summaries
type SummaryExecutor struct {
	repo          *database.SummaryRepository
	eventRepo     *database.PostgresEventRepository
	forecastRepo  *database.ForecastRepository
	TwitterRepo   *database.TwitterRepository // Exported for handler access
	TwitterClient TwitterPoster               // Exported for handler access
	logger        *slog.Logger
}

// NewSummaryExecutor creates a new summary executor
func NewSummaryExecutor(
	repo *database.SummaryRepository,
	eventRepo *database.PostgresEventRepository,
	forecastRepo *database.ForecastRepository,
	twitterRepo *database.TwitterRepository,
	twitterClient TwitterPoster,
	logger *slog.Logger,
) *SummaryExecutor {
	return &SummaryExecutor{
		repo:          repo,
		eventRepo:     eventRepo,
		forecastRepo:  forecastRepo,
		TwitterRepo:   twitterRepo,
		TwitterClient: twitterClient,
		logger:        logger,
	}
}

// Execute starts a summary execution and returns the run ID
func (e *SummaryExecutor) Execute(ctx context.Context, summaryID string) (string, error) {
	summary, err := e.repo.Get(ctx, summaryID)
	if err != nil {
		return "", fmt.Errorf("summary not found: %w", err)
	}

	// Calculate lookback window
	endTime := time.Now()
	startTime := endTime.Add(-time.Duration(summary.LookbackHours) * time.Hour)

	// Create run record
	runID, err := e.repo.CreateRun(ctx, summaryID, summary.HeadlineCount, startTime, endTime)
	if err != nil {
		return "", fmt.Errorf("failed to create run: %w", err)
	}

	// Start execution asynchronously
	go e.executeSummary(summary, runID, startTime, endTime)

	return runID, nil
}

func (e *SummaryExecutor) executeSummary(summary *models.Summary, runID string, startTime, endTime time.Time) {
	ctx := context.Background()

	// Fetch headlines
	headlines, err := e.eventRepo.GetEventsBetween(ctx, startTime, endTime, summary.Categories, summary.HeadlineCount)
	if err != nil {
		errMsg := err.Error()
		e.repo.CompleteRun(ctx, runID, "failed", &errMsg)
		return
	}

	if len(headlines) == 0 {
		errMsg := "no headlines found in time range"
		e.repo.CompleteRun(ctx, runID, "failed", &errMsg)
		return
	}

	// Build prompt with headlines
	headlinesText := ""
	for _, h := range headlines {
		headlinesText += fmt.Sprintf("- [%s] %s\n", h.Timestamp.Format("2006-01-02 15:04"), h.Title)
	}

	// Optionally include forecasts
	forecastsText := ""
	if summary.IncludeForecasts {
		forecasts, err := e.forecastRepo.ListForecasts(ctx)
		if err != nil {
			e.logger.Warn("failed to fetch forecasts for summary", "error", err)
		} else {
			forecastsText = "\n\nCurrent Forecasts:\n"
			for _, f := range forecasts {
				if !f.Active {
					continue
				}
				// Get latest run to get probability
				latestRun, err := e.forecastRepo.GetLatestCompletedForecastRun(ctx, f.ID)
				if err == nil && latestRun != nil && latestRun.Result != nil && latestRun.Result.AggregatedPercentiles != nil {
					// Use the median (P50) as the probability
					forecastsText += fmt.Sprintf("- %s: %.1f%%\n", f.Name, latestRun.Result.AggregatedPercentiles.P50*100)
				} else {
					// No run yet, just show the forecast name
					forecastsText += fmt.Sprintf("- %s: (no recent forecast available)\n", f.Name)
				}
			}
		}
	}

	fullPrompt := fmt.Sprintf("%s\n\nHeadlines from the last %d hours:\n%s%s", summary.Prompt, summary.LookbackHours, headlinesText, forecastsText)

	// Execute with each model and track first result
	var firstSummaryText string
	for _, model := range summary.Models {
		summaryText, err := e.callLLM(model, fullPrompt)
		if err != nil {
			e.logger.Error("failed to call LLM", "model", model.ModelName, "error", err)
			continue
		}

		if err := e.repo.SaveResult(ctx, runID, summaryText, model.Provider, model.ModelName); err != nil {
			e.logger.Error("failed to save result", "error", err)
		}

		// Capture first successful result for auto-posting
		if firstSummaryText == "" {
			firstSummaryText = summaryText
		}
	}

	e.repo.CompleteRun(ctx, runID, "completed", nil)

	// Auto-post to Twitter if enabled
	if summary.AutoPostToTwitter && firstSummaryText != "" && e.TwitterClient != nil {
		tweetID, err := e.TwitterClient.PostTweet(firstSummaryText)
		if err != nil {
			e.logger.Error("failed to auto-post to twitter", "run_id", runID, "error", err)
		} else {
			e.logger.Info("auto-posted summary to twitter", "run_id", runID, "tweet_id", tweetID)
			// Record the tweet in database
			config, err := e.TwitterRepo.Get(ctx)
			if err == nil && config != nil {
				if err := e.TwitterRepo.RecordPostedTweet(ctx, "", tweetID, firstSummaryText); err != nil {
					e.logger.Error("failed to record auto-posted tweet", "error", err)
				}
			}
		}
	}
}

func (e *SummaryExecutor) callLLM(model models.SummaryModel, prompt string) (string, error) {
	ctx := context.Background()

	switch model.Provider {
	case "openai":
		return e.callOpenAI(ctx, model, prompt)
	case "anthropic":
		return e.callAnthropic(ctx, model, prompt)
	default:
		return "", fmt.Errorf("unsupported provider: %s", model.Provider)
	}
}

func (e *SummaryExecutor) callOpenAI(ctx context.Context, model models.SummaryModel, prompt string) (string, error) {
	client := openai.NewClient(model.APIKey)

	// Build request - don't set temperature for beta models (gpt-5, o1, etc)
	req := openai.ChatCompletionRequest{
		Model: model.ModelName,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleUser, Content: prompt},
		},
	}

	// Only set temperature for stable models
	if !strings.Contains(model.ModelName, "gpt-5") && !strings.Contains(model.ModelName, "o1") {
		req.Temperature = 0.7
	}

	resp, err := client.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("openai api error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from openai")
	}

	return resp.Choices[0].Message.Content, nil
}

func (e *SummaryExecutor) callAnthropic(ctx context.Context, model models.SummaryModel, prompt string) (string, error) {
	client := anthropic.NewClient(option.WithAPIKey(model.APIKey))

	req := anthropic.MessageNewParams{
		Model:       anthropic.Model(model.ModelName),
		MaxTokens:   4096,
		Temperature: anthropic.Float(0.7),
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	}

	message, err := client.Messages.New(ctx, req)
	if err != nil {
		return "", fmt.Errorf("anthropic api error: %w", err)
	}

	if len(message.Content) == 0 {
		return "", fmt.Errorf("no response from anthropic")
	}

	return message.Content[0].Text, nil
}
