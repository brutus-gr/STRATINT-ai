package models

import (
	"encoding/json"
	"time"
)

// TwitterConfig holds configuration for Twitter/X API integration and auto-posting.
type TwitterConfig struct {
	ID                    int             `json:"id"`
	APIKey                string          `json:"api_key"`
	APISecret             string          `json:"api_secret"`
	AccessToken           string          `json:"access_token"`
	AccessTokenSecret     string          `json:"access_token_secret"`
	BearerToken           string          `json:"bearer_token"`
	TweetGenerationPrompt string          `json:"tweet_generation_prompt"`
	MinMagnitudeForTweet  float64         `json:"min_magnitude_for_tweet"`
	MinConfidenceForTweet float64         `json:"min_confidence_for_tweet"`
	MaxTweetAgeHours      int             `json:"max_tweet_age_hours"` // Maximum age of events to auto-tweet (in hours)
	EnabledCategories     json.RawMessage `json:"enabled_categories"`  // JSON array of category strings
	Enabled               bool            `json:"enabled"`
	UpdatedAt             time.Time       `json:"updated_at"`
	CreatedAt             time.Time       `json:"created_at"`
}

// TwitterConfigUpdate represents the fields that can be updated via API.
type TwitterConfigUpdate struct {
	APIKey                string          `json:"api_key"`
	APISecret             string          `json:"api_secret"`
	AccessToken           string          `json:"access_token"`
	AccessTokenSecret     string          `json:"access_token_secret"`
	BearerToken           string          `json:"bearer_token"`
	TweetGenerationPrompt string          `json:"tweet_generation_prompt"`
	MinMagnitudeForTweet  float64         `json:"min_magnitude_for_tweet"`
	MinConfidenceForTweet float64         `json:"min_confidence_for_tweet"`
	MaxTweetAgeHours      int             `json:"max_tweet_age_hours"`
	EnabledCategories     json.RawMessage `json:"enabled_categories"`
	Enabled               bool            `json:"enabled"`
}

// PostedTweet represents a tweet that has been posted for an event.
type PostedTweet struct {
	ID        int       `json:"id"`
	EventID   string    `json:"event_id"`
	TweetID   string    `json:"tweet_id"`
	TweetText string    `json:"tweet_text"`
	PostedAt  time.Time `json:"posted_at"`
}
