package ingestion

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/STRATINT/stratint/internal/models"
)

// Pipeline orchestrates the ingestion process from multiple connectors.
type Pipeline struct {
	connectors   []Connector
	sourceRepo   SourceRepository
	eventRepo    EventRepository
	deduplicator Deduplicator
	logger       *slog.Logger
	config       PipelineConfig
	mu           sync.RWMutex
	running      bool
}

// PipelineConfig holds configuration for the ingestion pipeline.
type PipelineConfig struct {
	PollInterval      time.Duration
	BatchSize         int
	EnableDedup       bool
	DedupWindow       time.Duration
	ConcurrentFetches int
	RetryPolicy       RetryPolicy
}

// DefaultPipelineConfig returns sensible defaults.
func DefaultPipelineConfig() PipelineConfig {
	return PipelineConfig{
		PollInterval:      5 * time.Minute,
		BatchSize:         100,
		EnableDedup:       true,
		DedupWindow:       24 * time.Hour,
		ConcurrentFetches: 3,
		RetryPolicy:       DefaultRetryPolicy(),
	}
}

// NewPipeline creates a new ingestion pipeline.
func NewPipeline(
	connectors []Connector,
	sourceRepo SourceRepository,
	eventRepo EventRepository,
	logger *slog.Logger,
	config PipelineConfig,
) *Pipeline {
	dedup := NewMemoryDeduplicator(config.DedupWindow)

	return &Pipeline{
		connectors:   connectors,
		sourceRepo:   sourceRepo,
		eventRepo:    eventRepo,
		deduplicator: dedup,
		logger:       logger,
		config:       config,
	}
}

// Start begins the ingestion pipeline.
func (p *Pipeline) Start(ctx context.Context) error {
	p.mu.Lock()
	if p.running {
		p.mu.Unlock()
		return fmt.Errorf("pipeline already running")
	}
	p.running = true
	p.mu.Unlock()

	p.logger.Info("starting ingestion pipeline",
		"connectors", len(p.connectors),
		"poll_interval", p.config.PollInterval,
	)

	ticker := time.NewTicker(p.config.PollInterval)
	defer ticker.Stop()

	// Initial fetch
	if err := p.fetchAll(ctx); err != nil {
		p.logger.Error("initial fetch failed", "error", err)
	}

	// Periodic fetching
	for {
		select {
		case <-ctx.Done():
			p.logger.Info("pipeline shutting down")
			p.mu.Lock()
			p.running = false
			p.mu.Unlock()
			return ctx.Err()

		case <-ticker.C:
			if err := p.fetchAll(ctx); err != nil {
				p.logger.Error("fetch cycle failed", "error", err)
			}
		}
	}
}

// fetchAll fetches from all connectors concurrently.
func (p *Pipeline) fetchAll(ctx context.Context) error {
	since := time.Now().Add(-p.config.PollInterval * 2)

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, p.config.ConcurrentFetches)

	for _, connector := range p.connectors {
		wg.Add(1)

		go func(conn Connector) {
			defer wg.Done()

			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			if err := p.fetchFromConnector(ctx, conn, since); err != nil {
				p.logger.Error("connector fetch failed",
					"connector", conn.Name(),
					"error", err,
				)
			}
		}(connector)
	}

	wg.Wait()

	// Cleanup old deduplication entries
	if p.config.EnableDedup {
		cutoff := time.Now().Add(-p.config.DedupWindow)
		p.deduplicator.Cleanup(cutoff)
	}

	return nil
}

// fetchFromConnector fetches data from a single connector with retry logic.
func (p *Pipeline) fetchFromConnector(ctx context.Context, conn Connector, since time.Time) error {
	start := time.Now()

	p.logger.Info("fetching from connector",
		"connector", conn.Name(),
		"since", since,
	)

	var sources []models.Source

	// Fetch with retry
	err := Retry(ctx, p.config.RetryPolicy, func() error {
		var err error
		sources, err = conn.Fetch(ctx, since)
		if err != nil {
			return NewRetryableError(err)
		}
		return nil
	})

	result := FetchResult{
		Sources:   sources,
		FetchedAt: time.Now(),
		Duration:  time.Since(start),
	}

	if err != nil {
		result.ErrorCount = 1
		p.logger.Error("fetch failed after retries",
			"connector", conn.Name(),
			"error", err,
		)
		return err
	}

	p.logger.Info("fetch completed",
		"connector", conn.Name(),
		"raw_count", len(sources),
		"duration", result.Duration,
	)

	// Deduplicate
	if p.config.EnableDedup {
		original := len(sources)
		sources = p.filterDuplicates(sources)
		duplicates := original - len(sources)

		p.logger.Info("deduplication complete",
			"connector", conn.Name(),
			"duplicates", duplicates,
			"unique", len(sources),
		)
	}

	// Store sources
	if len(sources) > 0 {
		if err := p.sourceRepo.StoreBatch(ctx, sources); err != nil {
			p.logger.Error("failed to store sources",
				"connector", conn.Name(),
				"error", err,
			)
			return err
		}

		result.NewCount = len(sources)

		p.logger.Info("sources stored",
			"connector", conn.Name(),
			"count", len(sources),
		)
	}

	return nil
}

// filterDuplicates removes duplicate sources using the deduplicator.
func (p *Pipeline) filterDuplicates(sources []models.Source) []models.Source {
	unique := make([]models.Source, 0, len(sources))

	for _, source := range sources {
		if p.deduplicator.IsNew(source) {
			p.deduplicator.Mark(source)
			unique = append(unique, source)
		}
	}

	return unique
}

// FetchOne manually triggers a fetch from a specific connector.
func (p *Pipeline) FetchOne(ctx context.Context, connectorName string, since time.Time) error {
	for _, conn := range p.connectors {
		if conn.Name() == connectorName {
			return p.fetchFromConnector(ctx, conn, since)
		}
	}
	return fmt.Errorf("connector not found: %s", connectorName)
}

// IsRunning returns whether the pipeline is currently running.
func (p *Pipeline) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.running
}

// GetConnectorStatus returns the status of all connectors.
// Note: This requires connectors to expose their status through a custom interface.
func (p *Pipeline) GetConnectorStatus() []ConnectorStatus {
	statuses := make([]ConnectorStatus, 0, len(p.connectors))

	// For now, return empty - connectors can expose status through other means
	// In future, add a StatusProvider interface that connectors can optionally implement

	return statuses
}

// HealthCheck checks the health of all connectors.
func (p *Pipeline) HealthCheck(ctx context.Context) map[string]error {
	results := make(map[string]error)

	for _, conn := range p.connectors {
		results[conn.Name()] = conn.HealthCheck(ctx)
	}

	return results
}

// AddConnector adds a new connector to the pipeline.
func (p *Pipeline) AddConnector(conn Connector) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.connectors = append(p.connectors, conn)
}

// RemoveConnector removes a connector from the pipeline.
func (p *Pipeline) RemoveConnector(name string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	for i, conn := range p.connectors {
		if conn.Name() == name {
			p.connectors = append(p.connectors[:i], p.connectors[i+1:]...)
			return true
		}
	}

	return false
}
