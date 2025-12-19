package models

import "time"

// ActivityType represents the type of activity being logged.
type ActivityType string

const (
	ActivityTypeRSSFetch         ActivityType = "rss_fetch"
	ActivityTypeTwitterFetch     ActivityType = "twitter_fetch"
	ActivityTypePlaywrightScrape ActivityType = "playwright_scrape"
	ActivityTypeFirecrawlScrape  ActivityType = "firecrawl_scrape"
	ActivityTypeEnrichment       ActivityType = "enrichment"
	ActivityTypeCorrelation      ActivityType = "correlation"
	ActivityTypePublish          ActivityType = "publish"
)

// ActivityLog represents a logged activity in the system.
type ActivityLog struct {
	ID           string                 `json:"id"`
	Timestamp    time.Time              `json:"timestamp"`
	ActivityType ActivityType           `json:"activity_type"`
	Platform     string                 `json:"platform,omitempty"`
	Message      string                 `json:"message"`
	Details      map[string]interface{} `json:"details,omitempty"`
	SourceCount  *int                   `json:"source_count,omitempty"`
	DurationMs   *int                   `json:"duration_ms,omitempty"`
}
