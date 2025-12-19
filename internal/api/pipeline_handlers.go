package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/STRATINT/stratint/internal/ingestion"
	"github.com/STRATINT/stratint/internal/models"
	"log/slog"
)

// noiseCache stores the current noise values with a timestamp
type noiseCache struct {
	mu             sync.RWMutex
	scrapingNoise  int
	enrichingNoise int
	lastUpdate     time.Time
	updateInterval time.Duration
}

// PipelineHandler handles pipeline metrics endpoints.
type PipelineHandler struct {
	sourceRepo ingestion.SourceRepository
	eventRepo  ingestion.EventRepository
	db         *sql.DB
	logger     *slog.Logger
	noise      *noiseCache
}

// NewPipelineHandler creates a new pipeline handler.
func NewPipelineHandler(
	sourceRepo ingestion.SourceRepository,
	eventRepo ingestion.EventRepository,
	db *sql.DB,
	logger *slog.Logger,
) *PipelineHandler {
	return &PipelineHandler{
		sourceRepo: sourceRepo,
		eventRepo:  eventRepo,
		db:         db,
		logger:     logger,
		noise: &noiseCache{
			updateInterval: 5 * time.Second, // Update noise every 5 seconds
		},
	}
}

// PipelineMetricsResponse represents the processing pipeline metrics.
type PipelineMetricsResponse struct {
	// Source stage
	SourcesTotal       int            `json:"sources_total"`
	SourcesByStatus    map[string]int `json:"sources_by_status"`
	SourcesRecentCount int            `json:"sources_recent_count"` // Last 24h

	// Enrichment stage (sources waiting for AI enrichment)
	EnrichmentByStatus map[string]int `json:"enrichment_by_status"` // pending, enriching, completed, failed

	// Event stage
	EventsTotal       int            `json:"events_total"`
	EventsByStatus    map[string]int `json:"events_by_status"`
	EventsRecentCount int            `json:"events_recent_count"` // Last 24h

	// Display values with noise for activity monitor
	DisplayScraping  int `json:"display_scraping"`  // Noisy pending sources
	DisplayEnriching int `json:"display_enriching"` // Noisy pending events

	// Conversion rates
	ScrapeCompletionRate float64 `json:"scrape_completion_rate"` // completed / (completed + pending + failed)
	EnrichmentRate       float64 `json:"enrichment_rate"`        // events / completed sources
	PublishRate          float64 `json:"publish_rate"`           // published / total events

	// Bottleneck indicators
	Bottleneck       string `json:"bottleneck"` // "scraping", "enrichment", "thresholds", or "none"
	BottleneckReason string `json:"bottleneck_reason"`
}

// updateNoise generates new noise values if enough time has passed
func (nc *noiseCache) updateNoise(scrapingBase, enrichingBase int) {
	nc.mu.Lock()
	defer nc.mu.Unlock()

	now := time.Now()
	if now.Sub(nc.lastUpdate) < nc.updateInterval {
		// Not time to update yet
		return
	}

	// Use current time as seed for deterministic but changing noise
	// This ensures all clients see the same noise at the same time
	seed := now.Unix() / int64(nc.updateInterval.Seconds())
	rng := rand.New(rand.NewSource(seed))

	// Generate noise: ±20% of base value, or ±2-5 if base is low
	if scrapingBase > 10 {
		nc.scrapingNoise = int((rng.Float64() - 0.5) * float64(scrapingBase) * 0.4)
	} else {
		nc.scrapingNoise = int((rng.Float64() - 0.5) * 8)
	}

	if enrichingBase > 10 {
		nc.enrichingNoise = int((rng.Float64() - 0.5) * float64(enrichingBase) * 0.4)
	} else {
		nc.enrichingNoise = int((rng.Float64() - 0.5) * 8)
	}

	nc.lastUpdate = now
}

// getNoise returns the current noise values
func (nc *noiseCache) getNoise() (int, int) {
	nc.mu.RLock()
	defer nc.mu.RUnlock()
	return nc.scrapingNoise, nc.enrichingNoise
}

// GetPipelineMetricsHandler returns processing pipeline metrics.
// GET /api/pipeline/metrics
func (h *PipelineHandler) GetPipelineMetricsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	metrics, err := h.calculatePipelineMetrics(ctx)
	if err != nil {
		h.logger.Error("failed to calculate pipeline metrics", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(metrics); err != nil {
		h.logger.Error("failed to encode response", "error", err)
	}
}

// calculatePipelineMetrics computes the pipeline metrics.
func (h *PipelineHandler) calculatePipelineMetrics(ctx context.Context) (*PipelineMetricsResponse, error) {
	// Initialize with all expected keys to avoid empty/sparse maps
	metrics := &PipelineMetricsResponse{
		SourcesByStatus: map[string]int{
			"pending":     0,
			"in_progress": 0,
			"completed":   0,
			"failed":      0,
			"skipped":     0,
		},
		EnrichmentByStatus: map[string]int{
			"pending":   0,
			"enriching": 0,
			"completed": 0,
			"failed":    0,
		},
		EventsByStatus: map[string]int{
			"pending":   0,
			"published": 0,
			"rejected":  0,
		},
	}

	// Get source counts by status
	statuses := []models.ScrapeStatus{
		models.ScrapeStatusPending,
		models.ScrapeStatusInProgress,
		models.ScrapeStatusCompleted,
		models.ScrapeStatusFailed,
		models.ScrapeStatusSkipped,
	}

	totalSources := 0
	for _, status := range statuses {
		sources, err := h.sourceRepo.GetByStatus(ctx, status, 10000) // High limit to get all
		if err != nil {
			return nil, err
		}
		count := len(sources)
		metrics.SourcesByStatus[string(status)] = count
		totalSources += count
	}
	metrics.SourcesTotal = totalSources

	// Get enrichment counts by status using direct database query
	rows, err := h.db.QueryContext(ctx, `
		SELECT enrichment_status, COUNT(*)
		FROM sources
		GROUP BY enrichment_status
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query enrichment status: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("failed to scan enrichment status: %w", err)
		}
		if _, exists := metrics.EnrichmentByStatus[status]; exists {
			metrics.EnrichmentByStatus[status] = count
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating enrichment rows: %w", err)
	}

	// Get event counts by status
	eventStatuses := []models.EventStatus{
		models.EventStatusPending,
		models.EventStatusPublished,
		models.EventStatusRejected,
	}

	totalEvents := 0
	for _, status := range eventStatuses {
		query := models.EventQuery{
			Status: &status,
			Limit:  10000,
			Page:   1,
		}
		count, err := h.eventRepo.Count(ctx, query)
		if err != nil {
			return nil, err
		}
		metrics.EventsByStatus[string(status)] = count
		totalEvents += count
	}
	metrics.EventsTotal = totalEvents

	// Calculate "pending enrichment" as completed sources that haven't been enriched yet
	// This represents sources waiting to be processed by AI enrichment
	completedSources := metrics.SourcesByStatus["completed"]
	pendingEnrichment := completedSources - totalEvents
	if pendingEnrichment < 0 {
		pendingEnrichment = 0
	}
	metrics.EventsByStatus["pending"] = pendingEnrichment

	// Calculate conversion rates
	completed := float64(metrics.SourcesByStatus["completed"])
	pending := float64(metrics.SourcesByStatus["pending"])
	failed := float64(metrics.SourcesByStatus["failed"])

	scrapableTotal := completed + pending + failed
	if scrapableTotal > 0 {
		metrics.ScrapeCompletionRate = (completed / scrapableTotal) * 100
	}

	if completed > 0 {
		metrics.EnrichmentRate = (float64(totalEvents) / completed) * 100
	}

	if totalEvents > 0 {
		published := float64(metrics.EventsByStatus["published"])
		metrics.PublishRate = (published / float64(totalEvents)) * 100
	}

	// Identify bottleneck
	metrics.Bottleneck, metrics.BottleneckReason = h.identifyBottleneck(metrics)

	// Update noise and apply to display values
	scrapingBase := metrics.SourcesByStatus["pending"]
	enrichingBase := metrics.EventsByStatus["pending"]

	h.noise.updateNoise(scrapingBase, enrichingBase)
	scrapingNoise, enrichingNoise := h.noise.getNoise()

	// Apply noise and clamp to ±20% of base value to prevent drift
	metrics.DisplayScraping = scrapingBase + scrapingNoise
	minScraping := int(float64(scrapingBase) * 0.8)
	maxScraping := int(float64(scrapingBase) * 1.2)
	if metrics.DisplayScraping < minScraping {
		metrics.DisplayScraping = minScraping
	}
	if metrics.DisplayScraping > maxScraping {
		metrics.DisplayScraping = maxScraping
	}
	if metrics.DisplayScraping < 0 {
		metrics.DisplayScraping = 0
	}

	metrics.DisplayEnriching = enrichingBase + enrichingNoise
	minEnriching := int(float64(enrichingBase) * 0.8)
	maxEnriching := int(float64(enrichingBase) * 1.2)
	if metrics.DisplayEnriching < minEnriching {
		metrics.DisplayEnriching = minEnriching
	}
	if metrics.DisplayEnriching > maxEnriching {
		metrics.DisplayEnriching = maxEnriching
	}
	if metrics.DisplayEnriching < 0 {
		metrics.DisplayEnriching = 0
	}

	return metrics, nil
}

// identifyBottleneck determines where the pipeline is bottlenecked.
func (h *PipelineHandler) identifyBottleneck(metrics *PipelineMetricsResponse) (string, string) {
	pending := metrics.SourcesByStatus["pending"]
	failed := metrics.SourcesByStatus["failed"]
	completed := metrics.SourcesByStatus["completed"]
	rejected := metrics.EventsByStatus["rejected"]
	published := metrics.EventsByStatus["published"]
	totalEvents := metrics.EventsTotal

	// Check for scraping bottleneck
	if pending > completed/2 && pending > 10 {
		return "scraping", "Large number of sources waiting to be scraped"
	}

	// Check for failed scrapes
	if failed > completed/3 && failed > 5 {
		return "scraping", "High scraping failure rate - check scraper errors"
	}

	// Check for enrichment bottleneck
	if completed > totalEvents*2 && completed > 20 {
		return "enrichment", "Many scraped sources not yet enriched into events"
	}

	// Check for threshold bottleneck (high rejection rate)
	if totalEvents > 0 && rejected > published && rejected > 5 {
		rejectionRate := float64(rejected) / float64(totalEvents) * 100
		return "thresholds", fmt.Sprintf("%.1f%% of events rejected - consider lowering thresholds", rejectionRate)
	}

	// Check for low enrichment rate
	if metrics.EnrichmentRate < 50 && completed > 10 {
		return "enrichment", "Low enrichment rate - AI may be filtering out many sources"
	}

	return "none", "Pipeline flowing smoothly"
}
