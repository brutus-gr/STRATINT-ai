package database

import (
	"context"
	"database/sql"
	"time"

	"github.com/STRATINT/stratint/internal/models"
)

// TwitterRepository handles Twitter configuration storage.
type TwitterRepository struct {
	db *sql.DB
}

// NewTwitterRepository creates a new Twitter repository.
func NewTwitterRepository(db *sql.DB) *TwitterRepository {
	return &TwitterRepository{db: db}
}

// Get retrieves the current Twitter configuration.
func (r *TwitterRepository) Get(ctx context.Context) (*models.TwitterConfig, error) {
	query := `
		SELECT
			id,
			api_key,
			api_secret,
			access_token,
			access_token_secret,
			bearer_token,
			tweet_generation_prompt,
			min_magnitude_for_tweet,
			min_confidence_for_tweet,
			max_tweet_age_hours,
			enabled_categories,
			enabled,
			updated_at,
			created_at
		FROM twitter_config
		ORDER BY id DESC
		LIMIT 1
	`

	var config models.TwitterConfig
	err := r.db.QueryRowContext(ctx, query).Scan(
		&config.ID,
		&config.APIKey,
		&config.APISecret,
		&config.AccessToken,
		&config.AccessTokenSecret,
		&config.BearerToken,
		&config.TweetGenerationPrompt,
		&config.MinMagnitudeForTweet,
		&config.MinConfidenceForTweet,
		&config.MaxTweetAgeHours,
		&config.EnabledCategories,
		&config.Enabled,
		&config.UpdatedAt,
		&config.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

// Update updates the Twitter configuration.
func (r *TwitterRepository) Update(ctx context.Context, update *models.TwitterConfigUpdate) error {
	query := `
		UPDATE twitter_config
		SET
			api_key = $1,
			api_secret = $2,
			access_token = $3,
			access_token_secret = $4,
			bearer_token = $5,
			tweet_generation_prompt = $6,
			min_magnitude_for_tweet = $7,
			min_confidence_for_tweet = $8,
			max_tweet_age_hours = $9,
			enabled_categories = $10,
			enabled = $11,
			updated_at = $12
		WHERE id = (SELECT id FROM twitter_config ORDER BY id DESC LIMIT 1)
	`

	now := time.Now()

	_, err := r.db.ExecContext(ctx, query,
		update.APIKey,
		update.APISecret,
		update.AccessToken,
		update.AccessTokenSecret,
		update.BearerToken,
		update.TweetGenerationPrompt,
		update.MinMagnitudeForTweet,
		update.MinConfidenceForTweet,
		update.MaxTweetAgeHours,
		update.EnabledCategories,
		update.Enabled,
		now,
	)

	return err
}

// RecordPostedTweet records a tweet that was posted for an event.
func (r *TwitterRepository) RecordPostedTweet(ctx context.Context, eventID, tweetID, tweetText string) error {
	query := `
		INSERT INTO posted_tweets (event_id, tweet_id, tweet_text, posted_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (event_id) DO NOTHING
	`

	_, err := r.db.ExecContext(ctx, query, eventID, tweetID, tweetText, time.Now())
	return err
}

// HasBeenTweeted checks if an event has already been tweeted.
func (r *TwitterRepository) HasBeenTweeted(ctx context.Context, eventID string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM posted_tweets WHERE event_id = $1
		)
	`

	var exists bool
	err := r.db.QueryRowContext(ctx, query, eventID).Scan(&exists)
	return exists, err
}

// GetPostedTweet retrieves the posted tweet for an event.
func (r *TwitterRepository) GetPostedTweet(ctx context.Context, eventID string) (*models.PostedTweet, error) {
	query := `
		SELECT id, event_id, tweet_id, tweet_text, posted_at
		FROM posted_tweets
		WHERE event_id = $1
	`

	var tweet models.PostedTweet
	err := r.db.QueryRowContext(ctx, query, eventID).Scan(
		&tweet.ID,
		&tweet.EventID,
		&tweet.TweetID,
		&tweet.TweetText,
		&tweet.PostedAt,
	)
	if err != nil {
		return nil, err
	}

	return &tweet, nil
}

// GetRecentTweets retrieves tweets posted within the last N hours, ordered by most recent first.
func (r *TwitterRepository) GetRecentTweets(ctx context.Context, hours int) ([]models.PostedTweet, error) {
	query := `
		SELECT id, event_id, tweet_id, tweet_text, posted_at
		FROM posted_tweets
		WHERE posted_at >= NOW() - INTERVAL '1 hour' * $1
		ORDER BY posted_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, hours)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tweets []models.PostedTweet
	for rows.Next() {
		var tweet models.PostedTweet
		if err := rows.Scan(
			&tweet.ID,
			&tweet.EventID,
			&tweet.TweetID,
			&tweet.TweetText,
			&tweet.PostedAt,
		); err != nil {
			return nil, err
		}
		tweets = append(tweets, tweet)
	}

	return tweets, rows.Err()
}

// GetAllPostedTweets retrieves all posted tweets, ordered by most recent first.
// Supports pagination with limit and offset.
func (r *TwitterRepository) GetAllPostedTweets(ctx context.Context, limit, offset int) ([]models.PostedTweet, error) {
	query := `
		SELECT id, event_id, tweet_id, tweet_text, posted_at
		FROM posted_tweets
		ORDER BY posted_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tweets []models.PostedTweet
	for rows.Next() {
		var tweet models.PostedTweet
		if err := rows.Scan(
			&tweet.ID,
			&tweet.EventID,
			&tweet.TweetID,
			&tweet.TweetText,
			&tweet.PostedAt,
		); err != nil {
			return nil, err
		}
		tweets = append(tweets, tweet)
	}

	return tweets, rows.Err()
}
