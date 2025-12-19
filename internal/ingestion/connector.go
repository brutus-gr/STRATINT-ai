package ingestion

import (
	"context"
	"time"

	"github.com/STRATINT/stratint/internal/models"
)

// Connector defines the interface that all source connectors must implement.
type Connector interface {
	// Name returns the unique identifier for this connector.
	Name() string

	// SourceType returns the type of sources this connector handles.
	SourceType() models.SourceType

	// Fetch retrieves new content from the source since the given timestamp.
	// Returns a list of raw sources and any error encountered.
	Fetch(ctx context.Context, since time.Time) ([]models.Source, error)

	// FetchByID retrieves a specific item by its source-specific identifier.
	FetchByID(ctx context.Context, id string) (*models.Source, error)

	// HealthCheck verifies the connector can reach its data source.
	HealthCheck(ctx context.Context) error

	// RateLimit returns the connector's rate limiting configuration.
	RateLimit() RateLimitConfig
}

// RateLimitConfig defines rate limiting parameters for a connector.
type RateLimitConfig struct {
	RequestsPerMinute int           // Maximum requests per minute
	BurstSize         int           // Maximum burst size
	Cooldown          time.Duration // Cooldown period after hitting rate limit
}

// ConnectorConfig holds common configuration for all connectors.
type ConnectorConfig struct {
	Name             string
	Enabled          bool
	MaxRetries       int
	RetryBackoff     time.Duration
	Timeout          time.Duration
	CredibilityBase  float64 // Base credibility score for this source type
	StoreRawContent  bool
	NormalizeContent bool
}

// FetchResult contains the outcome of a fetch operation.
type FetchResult struct {
	Sources    []models.Source
	NewCount   int
	ErrorCount int
	FetchedAt  time.Time
	Duration   time.Duration
}

// ConnectorStatus represents the current state of a connector.
type ConnectorStatus struct {
	Name           string
	Enabled        bool
	Healthy        bool
	LastFetch      time.Time
	LastError      string
	TotalFetched   int64
	TotalErrors    int64
	AverageLatency time.Duration
}

// BaseConnector provides common functionality for all connector implementations.
type BaseConnector struct {
	Config ConnectorConfig
	Status ConnectorStatus
}

// NewBaseConnector creates a new base connector with the given configuration.
func NewBaseConnector(cfg ConnectorConfig) *BaseConnector {
	return &BaseConnector{
		Config: cfg,
		Status: ConnectorStatus{
			Name:    cfg.Name,
			Enabled: cfg.Enabled,
			Healthy: true,
		},
	}
}

// UpdateStatus updates the connector status after a fetch operation.
func (b *BaseConnector) UpdateStatus(result FetchResult, err error) {
	b.Status.LastFetch = result.FetchedAt
	b.Status.TotalFetched += int64(result.NewCount)
	b.Status.TotalErrors += int64(result.ErrorCount)

	if err != nil {
		b.Status.Healthy = false
		b.Status.LastError = err.Error()
	} else {
		b.Status.Healthy = true
		b.Status.LastError = ""
	}

	// Update average latency (simple moving average)
	if b.Status.AverageLatency == 0 {
		b.Status.AverageLatency = result.Duration
	} else {
		b.Status.AverageLatency = (b.Status.AverageLatency + result.Duration) / 2
	}
}

// GetStatus returns the current connector status.
func (b *BaseConnector) GetStatus() ConnectorStatus {
	return b.Status
}
