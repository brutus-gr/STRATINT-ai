package models

import (
	"time"
)

// Source represents an OSINT data source (e.g., Twitter post, Telegram message, RSS feed).
type Source struct {
	ID                  string           `json:"id"`
	Type                SourceType       `json:"type"`
	URL                 string           `json:"url"`
	Title               string           `json:"title,omitempty"`
	Author              string           `json:"author,omitempty"`
	AuthorID            string           `json:"author_id,omitempty"`
	PublishedAt         time.Time        `json:"published_at"`
	RetrievedAt         time.Time        `json:"retrieved_at"`
	RawContent          string           `json:"raw_content"`
	ContentHash         string           `json:"content_hash"` // SHA-256 hash for deduplication
	Metadata            SourceMetadata   `json:"metadata"`
	Credibility         float64          `json:"credibility"` // 0-1 scale for source reliability
	VerificationURL     string           `json:"verification_url,omitempty"`
	ScrapeStatus        ScrapeStatus     `json:"scrape_status"`                   // Status of content scraping
	ScrapeError         string           `json:"scrape_error,omitempty"`          // Error message if scraping failed
	ScrapedAt           *time.Time       `json:"scraped_at,omitempty"`            // When content was scraped
	EnrichmentStatus    EnrichmentStatus `json:"enrichment_status"`               // Status of AI enrichment
	EnrichmentError     string           `json:"enrichment_error,omitempty"`      // Error message if enrichment failed
	EnrichedAt          *time.Time       `json:"enriched_at,omitempty"`           // When enrichment completed
	EnrichmentClaimedAt *time.Time       `json:"enrichment_claimed_at,omitempty"` // When enrichment was claimed (for stale lock detection)
	EventID             string           `json:"event_id,omitempty"`              // ID of the event created from this source
	CreatedAt           time.Time        `json:"created_at"`                      // Database timestamp
}

// SourceType categorizes the origin platform of OSINT data.
type SourceType string

const (
	SourceTypeTwitter    SourceType = "twitter"
	SourceTypeTelegram   SourceType = "telegram"
	SourceTypeGLP        SourceType = "glp" // Godlike Productions
	SourceTypeGovernment SourceType = "government"
	SourceTypeNewsMedia  SourceType = "news_media"
	SourceTypeBlog       SourceType = "blog"
	SourceTypeOther      SourceType = "other"
)

// ScrapeStatus indicates the scraping state of a source.
type ScrapeStatus string

const (
	ScrapeStatusPending    ScrapeStatus = "pending"     // Source created, scraping not yet attempted
	ScrapeStatusInProgress ScrapeStatus = "in_progress" // Currently being scraped
	ScrapeStatusCompleted  ScrapeStatus = "completed"   // Successfully scraped
	ScrapeStatusFailed     ScrapeStatus = "failed"      // Scraping failed
	ScrapeStatusSkipped    ScrapeStatus = "skipped"     // Scraping skipped (e.g., problematic domain)
)

// EnrichmentStatus indicates the AI enrichment state of a source.
type EnrichmentStatus string

const (
	EnrichmentStatusPending   EnrichmentStatus = "pending"   // Source ready for enrichment
	EnrichmentStatusEnriching EnrichmentStatus = "enriching" // Currently being enriched
	EnrichmentStatusCompleted EnrichmentStatus = "completed" // Successfully enriched
	EnrichmentStatusFailed    EnrichmentStatus = "failed"    // Enrichment failed
)

// SourceMetadata holds platform-specific metadata for attribution and traceability.
type SourceMetadata struct {
	// Twitter-specific
	TweetID      string `json:"tweet_id,omitempty"`
	RetweetCount int    `json:"retweet_count,omitempty"`
	LikeCount    int    `json:"like_count,omitempty"`

	// Telegram-specific
	ChannelID   string `json:"channel_id,omitempty"`
	ChannelName string `json:"channel_name,omitempty"`
	MessageID   string `json:"message_id,omitempty"`
	ViewCount   int    `json:"view_count,omitempty"`

	// RSS-specific
	FeedURL   string `json:"feed_url,omitempty"`
	RedditURL string `json:"reddit_url,omitempty"` // Original Reddit discussion URL (when sourced via Reddit)

	// Common fields
	Hashtags []string `json:"hashtags,omitempty"`
	Mentions []string `json:"mentions,omitempty"`
	Language string   `json:"language,omitempty"`
}

// GetDisplayName returns a human-readable identifier for the source.
func (s *Source) GetDisplayName() string {
	if s.Title != "" {
		return s.Title
	}
	if s.Author != "" {
		return s.Author + " (" + string(s.Type) + ")"
	}
	return string(s.Type) + " source"
}

// IsRecent returns true if the source was published within the specified duration.
func (s *Source) IsRecent(window time.Duration) bool {
	return time.Since(s.PublishedAt) <= window
}

// IsCredible returns true if the source meets minimum credibility threshold.
func (s *Source) IsCredible() bool {
	return s.Credibility >= 0.4
}
