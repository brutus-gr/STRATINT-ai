package models

import "time"

// ConnectorConfig represents configuration for a data source connector.
type ConnectorConfig struct {
	ID        string            `json:"id"`         // Connector identifier (twitter, telegram, rss)
	Enabled   bool              `json:"enabled"`    // Whether the connector is enabled
	Config    map[string]string `json:"config"`     // Connector-specific configuration
	UpdatedAt time.Time         `json:"updated_at"` // Last update timestamp
	CreatedAt time.Time         `json:"created_at"` // Creation timestamp
}
