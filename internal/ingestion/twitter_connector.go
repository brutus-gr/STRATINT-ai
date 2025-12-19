package ingestion

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/STRATINT/stratint/internal/enrichment"
	"github.com/STRATINT/stratint/internal/models"
)

// TwitterConnector fetches tweets from tracked accounts using Twitter API v2
type TwitterConnector struct {
	bearerToken      string
	logger           *slog.Logger
	client           *http.Client
	credibilityCache *enrichment.CredibilityCache
}

// NewTwitterConnector creates a new Twitter connector
func NewTwitterConnector(bearerToken string, logger *slog.Logger, credibilityCache *enrichment.CredibilityCache) *TwitterConnector {
	return &TwitterConnector{
		bearerToken:      bearerToken,
		logger:           logger,
		credibilityCache: credibilityCache,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// TwitterTweet represents a tweet from the API
type TwitterTweet struct {
	ID        string    `json:"id"`
	Text      string    `json:"text"`
	AuthorID  string    `json:"author_id"`
	CreatedAt time.Time `json:"created_at"`
}

// TwitterUser represents a user from the API
type TwitterUser struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name"`
}

// TwitterResponse represents the API response
type TwitterResponse struct {
	Data     []TwitterTweet         `json:"data"`
	Includes map[string]interface{} `json:"includes"`
	Meta     map[string]interface{} `json:"meta"`
}

// FetchAccountTweets fetches recent tweets from a specific account
// username: Twitter handle without @ (e.g., "elonmusk")
// sinceID: Optional - only return tweets after this ID
func (tc *TwitterConnector) FetchAccountTweets(account *models.TrackedAccount) ([]*models.Source, error) {
	if account.Platform != "twitter" {
		return nil, fmt.Errorf("invalid platform: %s", account.Platform)
	}

	// Remove @ if present
	username := strings.TrimPrefix(account.AccountIdentifier, "@")

	tc.logger.Info("fetching tweets", "username", username, "since_id", account.LastFetchedID)

	// Step 1: Get user ID from username
	userID, err := tc.getUserID(username)
	if err != nil {
		return nil, fmt.Errorf("failed to get user ID: %w", err)
	}

	// Step 2: Fetch tweets
	tweets, err := tc.getUserTweets(userID, account.LastFetchedID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch tweets: %w", err)
	}

	tc.logger.Info("fetched tweets", "username", username, "count", len(tweets))

	// Step 3: Convert to Source objects
	sources := make([]*models.Source, 0, len(tweets))
	ctx := context.Background()

	for _, tweet := range tweets {
		tweetURL := fmt.Sprintf("https://twitter.com/%s/status/%s", username, tweet.ID)

		// Assess source credibility using LLM (with domain caching)
		credibility := 0.60 // default fallback for Twitter
		if tc.credibilityCache != nil {
			if score, err := tc.credibilityCache.GetCredibility(ctx, tweetURL, models.SourceTypeTwitter); err == nil {
				credibility = score
			} else {
				tc.logger.Warn("failed to assess source credibility, using default",
					"url", tweetURL,
					"error", err)
			}
		}

		source := &models.Source{
			ID:          fmt.Sprintf("twitter-%s", tweet.ID),
			Type:        models.SourceTypeTwitter,
			URL:         tweetURL,
			Author:      fmt.Sprintf("@%s", username),
			AuthorID:    tweet.AuthorID,
			PublishedAt: tweet.CreatedAt,
			RetrievedAt: time.Now(),
			RawContent:  tweet.Text,
			ContentHash: hashContent(tweet.Text),
			Credibility: credibility, // LLM-assessed credibility score
			CreatedAt:   time.Now(),
			Metadata: models.SourceMetadata{
				TweetID: tweet.ID,
			},
		}
		sources = append(sources, source)
	}

	return sources, nil
}

// getUserID fetches the Twitter user ID from username
func (tc *TwitterConnector) getUserID(username string) (string, error) {
	url := fmt.Sprintf("https://api.twitter.com/2/users/by/username/%s", username)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+tc.bearerToken)

	resp, err := tc.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("twitter API error: %d - %s", resp.StatusCode, string(body))
	}

	var result struct {
		Data TwitterUser `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.Data.ID, nil
}

// getUserTweets fetches tweets from a user
func (tc *TwitterConnector) getUserTweets(userID, sinceID string) ([]TwitterTweet, error) {
	url := fmt.Sprintf("https://api.twitter.com/2/users/%s/tweets", userID)

	// Build query parameters
	params := []string{
		"tweet.fields=created_at,author_id",
		"max_results=10", // Fetch last 10 tweets
	}

	if sinceID != "" {
		params = append(params, fmt.Sprintf("since_id=%s", sinceID))
	}

	url += "?" + strings.Join(params, "&")

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+tc.bearerToken)

	resp, err := tc.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("twitter API error: %d - %s", resp.StatusCode, string(body))
	}

	var result TwitterResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Data, nil
}

// GetLatestTweetID returns the most recent tweet ID from a list of sources
func GetLatestTweetID(sources []*models.Source) string {
	var latestID string
	for _, source := range sources {
		if source.Metadata.TweetID != "" {
			if latestID == "" || source.Metadata.TweetID > latestID {
				latestID = source.Metadata.TweetID
			}
		}
	}
	return latestID
}

// hashContent computes SHA-256 hash of content
func hashContent(content string) string {
	h := sha256.New()
	h.Write([]byte(content))
	return hex.EncodeToString(h.Sum(nil))
}
