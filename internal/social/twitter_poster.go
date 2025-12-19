package social

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/STRATINT/stratint/internal/database"
	"github.com/STRATINT/stratint/internal/enrichment"
	"github.com/STRATINT/stratint/internal/models"
	"log/slog"
)

// TwitterPoster handles automatic tweet posting for events
type TwitterPoster struct {
	twitterRepo   *database.TwitterRepository
	openaiClient  *enrichment.OpenAIClient
	logger        *slog.Logger
	twitterClient *TwitterClient
	enabled       bool
}

// NewTwitterPoster creates a new Twitter poster service
func NewTwitterPoster(
	twitterRepo *database.TwitterRepository,
	openaiClient *enrichment.OpenAIClient,
	logger *slog.Logger,
) (*TwitterPoster, error) {
	poster := &TwitterPoster{
		twitterRepo:  twitterRepo,
		openaiClient: openaiClient,
		logger:       logger,
		enabled:      false,
	}

	// Try to initialize Twitter client from config
	if err := poster.RefreshConfig(context.Background()); err != nil {
		logger.Warn("twitter poster initialized but disabled", "error", err)
	}

	return poster, nil
}

// RefreshConfig reloads the Twitter configuration and reinitializes the client
func (tp *TwitterPoster) RefreshConfig(ctx context.Context) error {
	config, err := tp.twitterRepo.Get(ctx)
	if err != nil {
		return fmt.Errorf("failed to get twitter config: %w", err)
	}

	if !config.Enabled {
		tp.enabled = false
		tp.twitterClient = nil
		tp.logger.Info("twitter posting is disabled in config")
		return nil
	}

	// Create Twitter client
	tp.twitterClient = NewTwitterClient(
		config.APIKey,
		config.APISecret,
		config.AccessToken,
		config.AccessTokenSecret,
		config.BearerToken,
		tp.logger,
	)

	tp.enabled = true
	tp.logger.Info("twitter poster enabled and configured")

	return nil
}

// ShouldTweetEvent determines if an event meets the criteria for auto-tweeting
func (tp *TwitterPoster) ShouldTweetEvent(ctx context.Context, event *models.Event) (bool, error) {
	if !tp.enabled {
		return false, nil
	}

	// Check if already tweeted
	alreadyTweeted, err := tp.twitterRepo.HasBeenTweeted(ctx, event.ID)
	if err != nil {
		return false, fmt.Errorf("failed to check if event was tweeted: %w", err)
	}
	if alreadyTweeted {
		tp.logger.Debug("event already tweeted", "event_id", event.ID)
		return false, nil
	}

	// Get config to check thresholds
	config, err := tp.twitterRepo.Get(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get twitter config: %w", err)
	}

	// Check magnitude threshold
	if event.Magnitude < config.MinMagnitudeForTweet {
		tp.logger.Debug("event magnitude below threshold",
			"event_id", event.ID,
			"magnitude", event.Magnitude,
			"threshold", config.MinMagnitudeForTweet)
		return false, nil
	}

	// Check confidence threshold
	if event.Confidence.Score < config.MinConfidenceForTweet {
		tp.logger.Debug("event confidence below threshold",
			"event_id", event.ID,
			"confidence", event.Confidence.Score,
			"threshold", config.MinConfidenceForTweet)
		return false, nil
	}

	// Check event age threshold (don't tweet old events)
	if config.MaxTweetAgeHours > 0 {
		eventAge := time.Since(event.Timestamp)
		maxAge := time.Duration(config.MaxTweetAgeHours) * time.Hour
		if eventAge > maxAge {
			tp.logger.Debug("event too old to tweet",
				"event_id", event.ID,
				"event_timestamp", event.Timestamp,
				"event_age_hours", eventAge.Hours(),
				"max_age_hours", config.MaxTweetAgeHours)
			return false, nil
		}
	}

	// Check if category is enabled
	var enabledCategories []string
	if err := json.Unmarshal(config.EnabledCategories, &enabledCategories); err != nil {
		return false, fmt.Errorf("failed to parse enabled categories: %w", err)
	}

	categoryEnabled := false
	for _, cat := range enabledCategories {
		if strings.EqualFold(cat, string(event.Category)) {
			categoryEnabled = true
			break
		}
	}

	if !categoryEnabled {
		tp.logger.Debug("event category not enabled for tweeting",
			"event_id", event.ID,
			"category", event.Category)
		return false, nil
	}

	return true, nil
}

// TweetDecision represents the AI's decision about whether and how to tweet
type TweetDecision struct {
	Action    string `json:"action"`    // "POST", "UPDATE", or "SKIP"
	Tweet     string `json:"tweet"`     // The tweet text (empty if SKIP)
	Reasoning string `json:"reasoning"` // Why this action was chosen
}

// GenerateTweetText uses OpenAI to generate tweet text for an event with context awareness
func (tp *TwitterPoster) GenerateTweetText(ctx context.Context, event *models.Event) (string, error) {
	// Fetch recent tweets for context (last 24 hours)
	recentTweets, err := tp.twitterRepo.GetRecentTweets(ctx, 24)
	if err != nil {
		tp.logger.Warn("failed to fetch recent tweets for context",
			"event_id", event.ID,
			"error", err)
		// Continue without recent tweets context
		recentTweets = []models.PostedTweet{}
	}

	// Build recent tweets context
	var recentTweetsContext string
	if len(recentTweets) > 0 {
		recentTweetsContext = "\n\n=== RECENT TWEETS (LAST 24 HOURS) ===\n\n"
		for _, tweet := range recentTweets {
			// Format: [2 hours ago] tweet text
			hoursAgo := time.Since(tweet.PostedAt).Hours()
			timeStr := fmt.Sprintf("%.0fh ago", hoursAgo)
			if hoursAgo < 1 {
				timeStr = fmt.Sprintf("%.0fm ago", time.Since(tweet.PostedAt).Minutes())
			}
			recentTweetsContext += fmt.Sprintf("- [%s] %s\n", timeStr, tweet.TweetText)
		}
	} else {
		recentTweetsContext = "\n\n=== RECENT TWEETS ===\n\nNo tweets posted in the last 24 hours.\n"
	}

	// Build locations string from event
	var locations []string
	if event.Location.Country != "" {
		locations = append(locations, event.Location.Country)
	}
	if event.Location.City != "" {
		locations = append(locations, event.Location.City)
	}
	locationsStr := strings.Join(locations, ", ")
	if locationsStr == "" {
		locationsStr = "Global"
	}

	// Build the enhanced prompt
	basePrompt := `You are an expert at crafting compelling, concise tweets for breaking OSINT intelligence updates.

%s

=== NEW EVENT TO EVALUATE ===

Title: %s
Summary: %s
Category: %s
Locations: %s
Magnitude: %.1f/10
Confidence: %.2f

Event URL: https://stratint.ai/events/%s

=== YOUR TASK ===

Analyze the new event against recent tweets and decide on the appropriate action:

1. **SKIP** - If this event has substantially similar content to a recent tweet and adds no significant new information. We want to avoid posting redundant tweets.

2. **UPDATE** - If we already tweeted about this topic BUT there is significant new information. Start with "UPDATE:" or "DEVELOPING:" and focus on what's NEW.

3. **POST** - If this is genuinely new information that hasn't been covered, or is a distinct event. Use "BREAKING:" for urgent/breaking news (events within last 2 hours).

=== RESPONSE FORMAT ===

Respond with ONLY valid JSON in this exact format:

{
  "action": "POST|UPDATE|SKIP",
  "tweet": "your tweet text here (empty string if SKIP)",
  "reasoning": "brief explanation of why you chose this action"
}

=== TWEET REQUIREMENTS (IF NOT SKIPPING) ===

1. Start with flag emoji(s) for location(s) if applicable (e.g., 🇺🇸 🇺🇦 🇷🇺)
2. Use 🚨 after flags for breaking/urgent events
3. Use "BREAKING:" for truly breaking news (last 2 hours)
4. Use "UPDATE:" or "DEVELOPING:" if this is new info on a recent topic
5. Keep the main content under 200 characters to leave room for link
6. Make it attention-grabbing but factual - NO sensationalism
7. End with the event link: https://stratint.ai/events/%s
8. Add 2-3 relevant hashtags if they add value
9. Professional intelligence community tone

=== EXAMPLE OUTPUT ===

Example 1 (POST - New event):
{
  "action": "POST",
  "tweet": "🇺🇦🇷🇺 🚨 BREAKING: Ukrainian forces report downing Russian Su-34 fighter-bomber over Donetsk region. Pilot status unknown.\n\nhttps://stratint.ai/events/abc123\n\n#Ukraine #Russia #OSINT",
  "reasoning": "This is a new military development not covered in recent tweets. Magnitude 8.5 warrants BREAKING label."
}

Example 2 (UPDATE - New details):
{
  "action": "UPDATE",
  "tweet": "🇺🇦🇷🇺 UPDATE: Russian pilot from earlier Su-34 shootdown confirmed captured by Ukrainian forces. Video evidence circulating.\n\nhttps://stratint.ai/events/xyz789\n\n#Ukraine #Russia",
  "reasoning": "We tweeted about the shootdown 3 hours ago. This is significant new information about the pilot's status."
}

Example 3 (SKIP - Redundant):
{
  "action": "SKIP",
  "tweet": "",
  "reasoning": "We already tweeted about Chinese military exercises near Taiwan 2 hours ago. This event provides similar information without significant new details."
}

Respond with ONLY the JSON. No markdown, no code blocks, no explanations outside the JSON.`

	// Format the full prompt
	fullPrompt := fmt.Sprintf(basePrompt,
		recentTweetsContext,
		event.Title,
		event.Summary,
		string(event.Category),
		locationsStr,
		event.Magnitude,
		event.Confidence.Score,
		event.ID,
		event.ID,
	)

	// Call OpenAI to generate decision
	tp.logger.Info("generating tweet decision with OpenAI",
		"event_id", event.ID,
		"recent_tweets_count", len(recentTweets),
		"prompt_length", len(fullPrompt))

	systemPrompt := "You are an expert OSINT analyst and social media strategist. You make smart decisions about when to tweet and what to say. Always respond with valid JSON only."

	// Use 0 for maxTokens to let OpenAI use defaults
	responseText, err := tp.openaiClient.GenerateText(ctx, systemPrompt, fullPrompt, 0.7, 0)
	if err != nil {
		return "", fmt.Errorf("failed to generate tweet decision: %w", err)
	}

	tp.logger.Info("received response from OpenAI",
		"event_id", event.ID,
		"response_length", len(responseText),
		"response", responseText)

	// Parse JSON response
	var decision TweetDecision
	if err := json.Unmarshal([]byte(responseText), &decision); err != nil {
		tp.logger.Error("failed to parse JSON response from OpenAI",
			"event_id", event.ID,
			"response", responseText,
			"error", err)
		return "", fmt.Errorf("failed to parse AI decision: %w", err)
	}

	// Log the decision
	tp.logger.Info("tweet decision made",
		"event_id", event.ID,
		"action", decision.Action,
		"reasoning", decision.Reasoning)

	// Handle SKIP action
	if strings.ToUpper(decision.Action) == "SKIP" {
		tp.logger.Info("skipping tweet - AI determined it would be redundant",
			"event_id", event.ID,
			"reasoning", decision.Reasoning)
		return "", fmt.Errorf("SKIP: %s", decision.Reasoning)
	}

	// Validate tweet text
	tweetText := strings.TrimSpace(decision.Tweet)
	if tweetText == "" {
		return "", fmt.Errorf("AI returned empty tweet text for action %s", decision.Action)
	}

	// Truncate if too long
	if len(tweetText) > 280 {
		tp.logger.Warn("generated tweet exceeds 280 characters, truncating",
			"event_id", event.ID,
			"length", len(tweetText))
		tweetText = tweetText[:277] + "..."
	}

	tp.logger.Info("final tweet text prepared",
		"event_id", event.ID,
		"action", decision.Action,
		"final_length", len(tweetText),
		"final_text", tweetText)

	return tweetText, nil
}

// PostTweetForEvent generates and posts a tweet for an event
func (tp *TwitterPoster) PostTweetForEvent(ctx context.Context, event *models.Event) error {
	if !tp.enabled {
		return fmt.Errorf("twitter posting is disabled")
	}

	if tp.twitterClient == nil {
		return fmt.Errorf("twitter client not initialized")
	}

	// Generate tweet text
	tweetText, err := tp.GenerateTweetText(ctx, event)
	if err != nil {
		return fmt.Errorf("failed to generate tweet text: %w", err)
	}

	// Validate tweet text is not empty
	if strings.TrimSpace(tweetText) == "" {
		return fmt.Errorf("generated tweet text is empty")
	}

	tp.logger.Info("posting tweet for event",
		"event_id", event.ID,
		"tweet_text", tweetText,
		"tweet_length", len(tweetText))

	// Post tweet
	tweetID, err := tp.twitterClient.PostTweet(tweetText)
	if err != nil {
		return fmt.Errorf("failed to post tweet: %w", err)
	}

	// Record in database
	if err := tp.twitterRepo.RecordPostedTweet(ctx, event.ID, tweetID, tweetText); err != nil {
		tp.logger.Error("failed to record posted tweet in database",
			"event_id", event.ID,
			"tweet_id", tweetID,
			"error", err)
		// Don't return error here - tweet was posted successfully
	}

	tp.logger.Info("tweet posted successfully",
		"event_id", event.ID,
		"tweet_id", tweetID,
		"url", fmt.Sprintf("https://twitter.com/i/web/status/%s", tweetID))

	return nil
}

// TryPostTweetForEvent attempts to post a tweet for an event if it meets criteria
// This is the main entry point that should be called from the event lifecycle
func (tp *TwitterPoster) TryPostTweetForEvent(ctx context.Context, event *models.Event) {
	// Check if we should tweet this event
	shouldTweet, err := tp.ShouldTweetEvent(ctx, event)
	if err != nil {
		tp.logger.Error("error checking if event should be tweeted",
			"event_id", event.ID,
			"error", err)
		return
	}

	if !shouldTweet {
		return
	}

	// Post tweet
	if err := tp.PostTweetForEvent(ctx, event); err != nil {
		tp.logger.Error("failed to post tweet for event",
			"event_id", event.ID,
			"error", err)
		return
	}
}

// PostTweet posts a tweet with the given text and returns the tweet ID
func (tp *TwitterPoster) PostTweet(text string) (tweetID string, err error) {
	if tp.twitterClient == nil {
		return "", fmt.Errorf("twitter client not initialized")
	}
	return tp.twitterClient.PostTweet(text)
}
