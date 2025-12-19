package models

import "time"

// TrackedAccount represents a social media account being monitored for OSINT
type TrackedAccount struct {
	ID                   string                 `json:"id"`
	Platform             string                 `json:"platform"` // twitter, telegram, rss
	AccountIdentifier    string                 `json:"account_identifier"`
	DisplayName          string                 `json:"display_name,omitempty"`
	Enabled              bool                   `json:"enabled"`
	LastFetchedID        string                 `json:"last_fetched_id,omitempty"`
	LastFetchedAt        *time.Time             `json:"last_fetched_at,omitempty"`
	FetchIntervalMinutes int                    `json:"fetch_interval_minutes"`
	Metadata             map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt            time.Time              `json:"created_at"`
	UpdatedAt            time.Time              `json:"updated_at"`
}

// TrackedAccountRepository defines operations for tracked accounts
type TrackedAccountRepository interface {
	// Store creates or updates a tracked account
	Store(account *TrackedAccount) error

	// GetByID retrieves an account by ID
	GetByID(id string) (*TrackedAccount, error)

	// GetByPlatformAndIdentifier retrieves an account by platform and identifier
	GetByPlatformAndIdentifier(platform, identifier string) (*TrackedAccount, error)

	// ListByPlatform returns all accounts for a given platform
	ListByPlatform(platform string, enabledOnly bool) ([]*TrackedAccount, error)

	// ListAll returns all tracked accounts
	ListAll(enabledOnly bool) ([]*TrackedAccount, error)

	// UpdateLastFetched updates the last fetched ID and timestamp
	UpdateLastFetched(id, lastFetchedID string, lastFetchedAt time.Time) error

	// Delete removes a tracked account
	Delete(id string) error

	// SetEnabled enables or disables an account
	SetEnabled(id string, enabled bool) error
}
