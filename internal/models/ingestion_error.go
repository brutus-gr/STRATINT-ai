package models

import (
	"time"
)

// IngestionError represents an error that occurred during data ingestion.
type IngestionError struct {
	ID         string     `json:"id"`
	Platform   string     `json:"platform"`   // e.g., "rss", "twitter", "telegram"
	ErrorType  string     `json:"error_type"` // e.g., "scrape_failed", "feed_fetch_failed"
	URL        string     `json:"url"`        // The URL that failed
	ErrorMsg   string     `json:"error_msg"`  // Error message
	Metadata   string     `json:"metadata"`   // Additional JSON metadata
	CreatedAt  time.Time  `json:"created_at"`
	Resolved   bool       `json:"resolved"`
	ResolvedAt *time.Time `json:"resolved_at,omitempty"`
}

// IngestionErrorType categorizes different types of ingestion errors.
type IngestionErrorType string

const (
	ErrorTypeRSSFetchFailed    IngestionErrorType = "rss_fetch_failed"
	ErrorTypeScrapeFailed      IngestionErrorType = "scrape_failed"
	ErrorTypeParsingFailed     IngestionErrorType = "parsing_failed"
	ErrorTypeConnectionFailed  IngestionErrorType = "connection_failed"
	ErrorTypeAuthFailed        IngestionErrorType = "auth_failed"
	ErrorTypeRateLimitExceeded IngestionErrorType = "rate_limit_exceeded"
	ErrorTypeEnrichmentFailed  IngestionErrorType = "enrichment_failed"
)
