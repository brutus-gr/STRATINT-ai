package eventmanager

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/STRATINT/stratint/internal/enrichment"
	"github.com/STRATINT/stratint/internal/ingestion"
	"github.com/STRATINT/stratint/internal/models"
)

// EventLifecycleManager orchestrates the complete event lifecycle:
// Raw Source → Ingest → Enrich → Correlate → Publish
// TwitterPoster defines the interface for Twitter posting functionality
type TwitterPoster interface {
	TryPostTweetForEvent(ctx context.Context, event *models.Event)
}

type EventLifecycleManager struct {
	sourceRepo    ingestion.SourceRepository
	eventRepo     ingestion.EventRepository
	enricher      enrichment.Enricher
	correlator    *enrichment.EventCorrelator
	scorer        *enrichment.ConfidenceScorer
	thresholdRepo ThresholdRepository
	twitterPoster TwitterPoster
	activityRepo  ActivityLogger
	config        LifecycleConfig
	logger        *slog.Logger
}

// ActivityLogger defines the interface for logging activity.
type ActivityLogger interface {
	Log(ctx context.Context, log models.ActivityLog) error
}

// ThresholdRepository defines the interface for threshold configuration storage.
type ThresholdRepository interface {
	Get(ctx context.Context) (*models.ThresholdConfig, error)
	Update(ctx context.Context, config *models.ThresholdConfig) error
}

// LifecycleConfig holds configuration for event lifecycle management.
type LifecycleConfig struct {
	MinConfidence float64       // Minimum confidence to publish
	MinMagnitude  float64       // Minimum magnitude to publish
	MaxAge        time.Duration // Maximum age of source to consider (0 = no limit)
	MinSources    int           // Minimum number of sources required
	AutoPublish   bool          // Automatically publish events that meet criteria
	BatchSize     int           // Batch size for processing
}

// DefaultLifecycleConfig returns sensible defaults.
func DefaultLifecycleConfig() LifecycleConfig {
	return LifecycleConfig{
		MinConfidence: 0.30,
		MinMagnitude:  1.0,
		MinSources:    1,
		AutoPublish:   true,
		BatchSize:     50,
	}
}

// NewEventLifecycleManager creates a new lifecycle manager.
func NewEventLifecycleManager(
	sourceRepo ingestion.SourceRepository,
	eventRepo ingestion.EventRepository,
	enricher enrichment.Enricher,
	thresholdRepo ThresholdRepository,
	twitterPoster TwitterPoster,
	activityRepo ActivityLogger,
	logger *slog.Logger,
	config LifecycleConfig,
) *EventLifecycleManager {
	// Try to get correlator from enricher (if it's an OpenAI client)
	var correlator *enrichment.EventCorrelator
	var scorer *enrichment.ConfidenceScorer
	if openaiClient, ok := enricher.(interface {
		GetCorrelator() *enrichment.EventCorrelator
		GetScorer() *enrichment.ConfidenceScorer
	}); ok {
		correlator = openaiClient.GetCorrelator()
		scorer = openaiClient.GetScorer()
		logger.Info("initialized lifecycle manager with OpenAI-based correlator and scorer")
	} else {
		logger.Warn("enricher does not support correlation - events will not be deduplicated")
	}

	return &EventLifecycleManager{
		sourceRepo:    sourceRepo,
		eventRepo:     eventRepo,
		enricher:      enricher,
		correlator:    correlator,
		scorer:        scorer,
		thresholdRepo: thresholdRepo,
		twitterPoster: twitterPoster,
		activityRepo:  activityRepo,
		config:        config,
		logger:        logger,
	}
}

// ProcessScrapedSources processes already-stored sources that have been scraped.
// This is used after the scraping service has updated sources to "completed" status.
func (m *EventLifecycleManager) ProcessScrapedSources(ctx context.Context, limit int) (ProcessResult, error) {
	result := ProcessResult{
		ProcessedAt: time.Now(),
	}

	// Get scraped sources that haven't been processed yet
	sources, err := m.sourceRepo.GetByStatus(ctx, models.ScrapeStatusCompleted, limit)
	if err != nil {
		return result, fmt.Errorf("failed to get scraped sources: %w", err)
	}

	if len(sources) == 0 {
		m.logger.Debug("no scraped sources to process")
		return result, nil
	}

	m.logger.Info("processing scraped sources", "count", len(sources))

	// Enrich sources into events (one at a time to track individual failures)
	startTime := time.Now()
	successCount := 0
	failureCount := 0

	for _, source := range sources {
		// Try to enrich this source
		event, err := m.enricher.Enrich(ctx, source)

		if err != nil {
			// Mark source as failed enrichment
			failureCount++
			m.logger.Error("enrichment failed for source",
				"source_id", source.ID,
				"error", err)

			// Update source status to failed
			if updateErr := m.updateSourceEnrichmentStatus(ctx, source.ID, models.EnrichmentStatusFailed, err.Error(), ""); updateErr != nil {
				m.logger.Error("failed to update source enrichment status",
					"source_id", source.ID,
					"error", updateErr)
			}

			result.ErrorCount++
			continue
		}

		// Enrichment succeeded
		successCount++
		result.EventsEnriched++

		// Process the enriched event
		if err := m.ProcessEvent(ctx, event); err != nil {
			m.logger.Error("failed to process event",
				"event_id", event.ID,
				"error", err,
			)

			// Mark source as failed (event creation failed)
			if updateErr := m.updateSourceEnrichmentStatus(ctx, source.ID, models.EnrichmentStatusFailed, "event creation failed: "+err.Error(), ""); updateErr != nil {
				m.logger.Error("failed to update source enrichment status",
					"source_id", source.ID,
					"error", updateErr)
			}

			result.ErrorCount++
			continue
		}

		// Update source status to completed and link to event
		if updateErr := m.updateSourceEnrichmentStatus(ctx, source.ID, models.EnrichmentStatusCompleted, "", event.ID); updateErr != nil {
			m.logger.Error("failed to update source enrichment status",
				"source_id", source.ID,
				"event_id", event.ID,
				"error", updateErr)
		}

		// Count by status
		switch event.Status {
		case models.EventStatusPublished:
			result.EventsPublished++
		case models.EventStatusRejected:
			result.EventsRejected++
		}
	}

	duration := time.Since(startTime)
	m.logger.Info("enrichment batch completed",
		"total_sources", len(sources),
		"successful", successCount,
		"failed", failureCount,
		"duration_ms", duration.Milliseconds())

	// Log enrichment activity
	if m.activityRepo != nil {
		durationMs := int(duration.Milliseconds())
		m.activityRepo.Log(ctx, models.ActivityLog{
			ActivityType: models.ActivityTypeEnrichment,
			Platform:     "ai",
			Message:      fmt.Sprintf("Enriched %d sources: %d successful, %d failed", len(sources), successCount, failureCount),
			Details: map[string]interface{}{
				"total_sources":    len(sources),
				"successful_count": successCount,
				"failed_count":     failureCount,
			},
			SourceCount: &successCount,
			DurationMs:  &durationMs,
		})
	}

	return result, nil
}

// ProcessSources ingests raw sources through the complete lifecycle.
// With the split scraping workflow, sources will be stored with status="pending",
// then scraped separately, then enriched when they have status="completed".
func (m *EventLifecycleManager) ProcessSources(ctx context.Context, sources []models.Source) (ProcessResult, error) {
	result := ProcessResult{
		ProcessedAt: time.Now(),
	}

	// Step 1: Store raw sources
	if err := m.sourceRepo.StoreBatch(ctx, sources); err != nil {
		return result, fmt.Errorf("failed to store sources: %w", err)
	}
	result.SourcesIngested = len(sources)

	m.logger.Info("sources ingested",
		"count", len(sources),
	)

	// Step 2: Filter sources that have been scraped (skip pending/failed/skipped)
	scrapedSources := make([]models.Source, 0, len(sources))
	for _, source := range sources {
		if source.ScrapeStatus == models.ScrapeStatusCompleted {
			scrapedSources = append(scrapedSources, source)
		}
	}

	if len(scrapedSources) < len(sources) {
		m.logger.Info("filtered sources by scrape status",
			"total", len(sources),
			"scraped", len(scrapedSources),
			"skipped", len(sources)-len(scrapedSources),
		)
	}

	// Only enrich sources that have been successfully scraped
	if len(scrapedSources) == 0 {
		m.logger.Info("no scraped sources to enrich")
		return result, nil
	}

	// Step 3: Enrich scraped sources into events (one at a time to track individual failures)
	startTime := time.Now()
	successCount := 0
	failureCount := 0

	for _, source := range scrapedSources {
		// Try to enrich this source
		event, err := m.enricher.Enrich(ctx, source)

		if err != nil {
			// Mark source as failed enrichment
			failureCount++
			m.logger.Error("enrichment failed for source",
				"source_id", source.ID,
				"error", err)

			// Update source status to failed
			if updateErr := m.updateSourceEnrichmentStatus(ctx, source.ID, models.EnrichmentStatusFailed, err.Error(), ""); updateErr != nil {
				m.logger.Error("failed to update source enrichment status",
					"source_id", source.ID,
					"error", updateErr)
			}

			result.ErrorCount++
			continue
		}

		// Enrichment succeeded
		successCount++
		result.EventsEnriched++

		// Process the enriched event
		if err := m.ProcessEvent(ctx, event); err != nil {
			m.logger.Error("failed to process event",
				"event_id", event.ID,
				"error", err,
			)

			// Mark source as failed (event creation failed)
			if updateErr := m.updateSourceEnrichmentStatus(ctx, source.ID, models.EnrichmentStatusFailed, "event creation failed: "+err.Error(), ""); updateErr != nil {
				m.logger.Error("failed to update source enrichment status",
					"source_id", source.ID,
					"error", updateErr)
			}

			result.ErrorCount++
			continue
		}

		// Update source status to completed and link to event
		if updateErr := m.updateSourceEnrichmentStatus(ctx, source.ID, models.EnrichmentStatusCompleted, "", event.ID); updateErr != nil {
			m.logger.Error("failed to update source enrichment status",
				"source_id", source.ID,
				"event_id", event.ID,
				"error", updateErr)
		}

		// Count by status
		switch event.Status {
		case models.EventStatusPublished:
			result.EventsPublished++
		case models.EventStatusRejected:
			result.EventsRejected++
		}
	}

	duration := time.Since(startTime)
	m.logger.Info("enrichment batch completed",
		"total_sources", len(scrapedSources),
		"successful", successCount,
		"failed", failureCount,
		"duration_ms", duration.Milliseconds())

	// Log enrichment activity
	if m.activityRepo != nil {
		durationMs := int(duration.Milliseconds())
		m.activityRepo.Log(ctx, models.ActivityLog{
			ActivityType: models.ActivityTypeEnrichment,
			Platform:     "ai",
			Message:      fmt.Sprintf("Enriched %d sources: %d successful, %d failed", len(scrapedSources), successCount, failureCount),
			Details: map[string]interface{}{
				"total_sources":    len(scrapedSources),
				"successful_count": successCount,
				"failed_count":     failureCount,
			},
			SourceCount: &successCount,
			DurationMs:  &durationMs,
		})
	}

	return result, nil
}

// ProcessEvent handles a single event through its lifecycle.
// It checks for duplicates, performs correlation, applies thresholds, and saves the event.
func (m *EventLifecycleManager) ProcessEvent(ctx context.Context, event *models.Event) error {
	m.logger.Debug("ProcessEvent: Entered",
		"event_id", event.ID,
		"title", event.Title,
		"status", event.Status,
		"confidence", event.Confidence.Score,
		"magnitude", event.Magnitude,
		"sources_count", len(event.Sources))

	// Check if event already exists by ID
	m.logger.Debug("ProcessEvent: Checking for existing event", "event_id", event.ID)
	existing, err := m.eventRepo.GetByID(ctx, event.ID)
	if err != nil {
		m.logger.Debug("ProcessEvent: Failed to check existing event",
			"event_id", event.ID,
			"error", err,
			"error_type", fmt.Sprintf("%T", err))
		return fmt.Errorf("failed to check existing event: %w", err)
	}

	if existing != nil {
		// Event already exists, potentially update
		m.logger.Debug("ProcessEvent: Event already exists, updating",
			"event_id", event.ID,
			"existing_status", existing.Status)
		return m.updateExistingEvent(ctx, existing, event)
	}
	m.logger.Debug("ProcessEvent: Event is new, will check correlation", "event_id", event.ID)

	// TEMPORARILY DISABLED: Check for similar events using OpenAI-based correlation (if available)
	// This was making 50+ OpenAI calls per event, causing 2-minute delays
	if false && m.correlator != nil {
		m.logger.Debug("ProcessEvent: Correlator available, checking for similar events", "event_id", event.ID)
		// Get recent events for correlation analysis
		since := time.Now().Add(-7 * 24 * time.Hour)
		query := models.EventQuery{
			Since: &since,
			Limit: 100,
			Page:  1,
		}

		resp, err := m.eventRepo.Query(ctx, query)
		if err != nil {
			m.logger.Debug("ProcessEvent: Failed to query events for correlation",
				"event_id", event.ID,
				"error", err,
				"error_type", fmt.Sprintf("%T", err))
		} else if resp != nil && len(resp.Events) > 0 {
			m.logger.Debug("ProcessEvent: Found existing events for correlation",
				"event_id", event.ID,
				"existing_events_count", len(resp.Events))

			// Find best matching event using OpenAI
			bestMatch, corrResult, err := m.correlator.FindBestMatch(ctx, event.Sources[0], resp.Events)
			if err != nil {
				m.logger.Debug("ProcessEvent: Correlation analysis failed",
					"event_id", event.ID,
					"error", err)
			} else if bestMatch != nil && corrResult.ShouldMerge {
				m.logger.Debug("ProcessEvent: Found similar event, will merge",
					"new_event_id", event.ID,
					"existing_event_id", bestMatch.ID,
					"similarity", corrResult.Similarity,
					"should_merge", corrResult.ShouldMerge,
					"has_novel_facts", corrResult.HasNovelFacts,
					"novel_fact_count", len(corrResult.NovelFacts),
				)

				// Add source to existing event (merge operation)
				bestMatch.Sources = append(bestMatch.Sources, event.Sources...)

				// If this source contains novel facts, create a separate event for them
				if corrResult.HasNovelFacts && len(corrResult.NovelFacts) > 0 {
					m.logger.Debug("ProcessEvent: Creating novel facts event",
						"event_id", event.ID,
						"related_to", bestMatch.ID)
					if err := m.createNovelFactsEvent(ctx, event, bestMatch, corrResult); err != nil {
						m.logger.Debug("ProcessEvent: Failed to create novel facts event",
							"error", err,
							"original_event_id", bestMatch.ID,
						)
						// Continue with merge even if novel facts event creation fails
					}
				}

				// Update the existing event with merged sources
				m.logger.Debug("ProcessEvent: Updating existing event with merged sources",
					"existing_event_id", bestMatch.ID,
					"source_count", len(bestMatch.Sources))
				return m.eventRepo.Update(ctx, *bestMatch)
			} else {
				m.logger.Debug("ProcessEvent: No similar events found or merge not needed",
					"event_id", event.ID,
					"best_match_id", func() string {
						if bestMatch != nil {
							return bestMatch.ID
						}
						return "none"
					}(),
					"should_merge", corrResult != nil && corrResult.ShouldMerge)
			}
		} else {
			m.logger.Debug("ProcessEvent: No existing events for correlation", "event_id", event.ID)
		}
	} else {
		m.logger.Debug("ProcessEvent: No correlator available, skipping similarity check", "event_id", event.ID)
	}

	// New event - evaluate for publication
	m.logger.Debug("ProcessEvent: Evaluating event for publication",
		"event_id", event.ID,
		"auto_publish", m.config.AutoPublish,
		"current_status", event.Status)

	shouldPub := m.shouldPublish(event)
	m.logger.Debug("ProcessEvent: shouldPublish result",
		"event_id", event.ID,
		"should_publish", shouldPub,
		"auto_publish", m.config.AutoPublish)

	if m.config.AutoPublish && shouldPub {
		event.Status = models.EventStatusPublished
		m.logger.Debug("ProcessEvent: Event marked as PUBLISHED",
			"event_id", event.ID,
			"magnitude", event.Magnitude,
			"confidence", event.Confidence.Score,
			"status", event.Status)

		// Try to post to Twitter if enabled
		m.tryPostToTwitter(ctx, event)
	} else {
		event.Status = models.EventStatusRejected
		reason := m.rejectionReason(event)
		m.logger.Debug("ProcessEvent: Event marked as REJECTED",
			"event_id", event.ID,
			"magnitude", event.Magnitude,
			"confidence", event.Confidence.Score,
			"reason", reason,
			"status", event.Status)
	}

	// Store the event
	m.logger.Debug("ProcessEvent: About to call eventRepo.Create",
		"event_id", event.ID,
		"status", event.Status,
		"title", event.Title)

	err = m.eventRepo.Create(ctx, *event)
	if err != nil {
		m.logger.Debug("ProcessEvent: Failed to create event in database",
			"event_id", event.ID,
			"error", err,
			"error_type", fmt.Sprintf("%T", err),
			"status", event.Status)
		return fmt.Errorf("failed to create event: %w", err)
	}

	m.logger.Debug("ProcessEvent: Successfully created event in database",
		"event_id", event.ID,
		"status", event.Status)

	return nil
}

// createNovelFactsEvent creates a separate event containing only novel facts.
// This is called when a source is merged with an existing event but contains new information.
func (m *EventLifecycleManager) createNovelFactsEvent(
	ctx context.Context,
	originalEvent *models.Event,
	existingEvent *models.Event,
	corrResult *enrichment.CorrelationResult,
) error {
	// Create title indicating this is additional information
	novelTitle := fmt.Sprintf("%s - Additional Details", existingEvent.Title)

	// Create a descriptive summary highlighting the novel facts
	novelSummary := fmt.Sprintf("New details discovered: %s", formatNovelFacts(corrResult.NovelFacts))

	// Calculate confidence based on the new source (not inherited from existing event)
	var confidence models.Confidence
	if m.scorer != nil && len(originalEvent.Sources) > 0 {
		// Use the new source to calculate confidence
		newSource := originalEvent.Sources[0]
		confidence = m.scorer.Score(newSource, originalEvent, originalEvent.Entities)
		m.logger.Debug("recalculated confidence for novel facts event",
			"novel_event_id", fmt.Sprintf("novel-%s", originalEvent.ID),
			"new_score", confidence.Score,
			"source_url", newSource.URL)
	} else {
		// Fallback if scorer not available
		confidence = models.Confidence{
			Score:       existingEvent.Confidence.Score * 0.9,
			SourceCount: len(originalEvent.Sources),
			Reasoning:   fmt.Sprintf("Novel facts related to existing event %s", existingEvent.ID),
		}
	}

	// Create new event for novel facts
	novelEvent := &models.Event{
		ID:         fmt.Sprintf("novel-%s", originalEvent.ID),
		Title:      novelTitle,
		Summary:    novelSummary,
		RawContent: fmt.Sprintf("Novel facts discovered in relation to event %s: %s", existingEvent.ID, corrResult.Reasoning),
		Category:   existingEvent.Category,
		Tags:       existingEvent.Tags,
		Sources:    originalEvent.Sources, // Include the source that provided the novel facts
		Entities:   originalEvent.Entities,
		Location:   originalEvent.Location,
		Timestamp:  time.Now(),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Status:     models.EventStatusEnriched,
		Magnitude:  existingEvent.Magnitude * 0.7, // Slightly lower magnitude as it's supplementary
		Confidence: confidence,
	}

	// Evaluate if this novel facts event should be published
	if m.config.AutoPublish && m.shouldPublish(novelEvent) {
		novelEvent.Status = models.EventStatusPublished
		m.logger.Info("novel facts event published",
			"novel_event_id", novelEvent.ID,
			"related_event_id", existingEvent.ID,
			"fact_count", len(corrResult.NovelFacts),
		)

		// Try to post to Twitter if enabled
		m.tryPostToTwitter(ctx, novelEvent)
	} else {
		novelEvent.Status = models.EventStatusRejected
		m.logger.Debug("novel facts event rejected",
			"novel_event_id", novelEvent.ID,
			"related_event_id", existingEvent.ID,
			"reason", m.rejectionReason(novelEvent),
		)
	}

	// Store the novel facts event
	if err := m.eventRepo.Create(ctx, *novelEvent); err != nil {
		return fmt.Errorf("failed to create novel facts event: %w", err)
	}

	m.logger.Info("created novel facts event",
		"novel_event_id", novelEvent.ID,
		"related_event_id", existingEvent.ID,
		"novel_facts", corrResult.NovelFacts,
	)

	return nil
}

// shouldPublish determines if an event meets publication criteria.
// Reads thresholds from database to allow runtime updates.
func (m *EventLifecycleManager) shouldPublish(event *models.Event) bool {
	m.logger.Debug("shouldPublish: Evaluating event",
		"event_id", event.ID,
		"confidence", event.Confidence.Score,
		"magnitude", event.Magnitude,
		"sources", len(event.Sources))

	// Read thresholds from database
	thresholds, err := m.thresholdRepo.Get(context.Background())
	if err != nil {
		m.logger.Debug("shouldPublish: Failed to get thresholds, using defaults",
			"event_id", event.ID,
			"error", err)
		// Fall back to config defaults
		thresholds = &models.ThresholdConfig{
			MinConfidence:     0.1,
			MinMagnitude:      0.0,
			MaxSourceAgeHours: 0,
		}
	}

	m.logger.Debug("shouldPublish: Using thresholds",
		"event_id", event.ID,
		"min_confidence", thresholds.MinConfidence,
		"min_magnitude", thresholds.MinMagnitude,
		"min_sources", m.config.MinSources,
		"max_age_hours", thresholds.MaxSourceAgeHours)

	if event.Confidence.Score < thresholds.MinConfidence {
		m.logger.Debug("shouldPublish: Failed confidence check",
			"event_id", event.ID,
			"event_confidence", event.Confidence.Score,
			"min_confidence", thresholds.MinConfidence)
		return false
	}

	if event.Magnitude < thresholds.MinMagnitude {
		m.logger.Debug("shouldPublish: Failed magnitude check",
			"event_id", event.ID,
			"event_magnitude", event.Magnitude,
			"min_magnitude", thresholds.MinMagnitude)
		return false
	}

	if len(event.Sources) < m.config.MinSources {
		m.logger.Debug("shouldPublish: Failed sources check",
			"event_id", event.ID,
			"event_sources", len(event.Sources),
			"min_sources", m.config.MinSources)
		return false
	}

	// Check source age if MaxSourceAgeHours is set
	if thresholds.MaxSourceAgeHours > 0 {
		maxAge := time.Duration(thresholds.MaxSourceAgeHours) * time.Hour
		now := time.Now()
		for _, source := range event.Sources {
			age := now.Sub(source.PublishedAt)
			if age > maxAge {
				m.logger.Debug("shouldPublish: Failed age check",
					"event_id", event.ID,
					"source_age", age,
					"max_age", maxAge,
					"source_published", source.PublishedAt)
				return false
			}
		}
	}

	m.logger.Debug("shouldPublish: Event meets all criteria",
		"event_id", event.ID)
	return true
}

// tryPostToTwitter attempts to post the event to Twitter if enabled
func (m *EventLifecycleManager) tryPostToTwitter(ctx context.Context, event *models.Event) {
	if m.twitterPoster == nil {
		return
	}

	// Post in a goroutine to not block event processing
	go m.twitterPoster.TryPostTweetForEvent(context.Background(), event)
}

// rejectionReason returns a human-readable rejection reason.
func (m *EventLifecycleManager) rejectionReason(event *models.Event) string {
	// Read thresholds from database
	thresholds, err := m.thresholdRepo.Get(context.Background())
	if err != nil {
		return "failed to get thresholds"
	}

	if event.Confidence.Score < thresholds.MinConfidence {
		return fmt.Sprintf("confidence %.2f < %.2f", event.Confidence.Score, thresholds.MinConfidence)
	}

	if event.Magnitude < thresholds.MinMagnitude {
		return fmt.Sprintf("magnitude %.1f < %.1f", event.Magnitude, thresholds.MinMagnitude)
	}

	if len(event.Sources) < m.config.MinSources {
		return fmt.Sprintf("sources %d < %d", len(event.Sources), m.config.MinSources)
	}

	// Check source age if MaxSourceAgeHours is set
	if thresholds.MaxSourceAgeHours > 0 {
		maxAge := time.Duration(thresholds.MaxSourceAgeHours) * time.Hour
		now := time.Now()
		for _, source := range event.Sources {
			age := now.Sub(source.PublishedAt)
			if age > maxAge {
				return fmt.Sprintf("source too old: %s > %s", age.Round(time.Hour), maxAge)
			}
		}
	}

	return "unknown"
}

// updateExistingEvent handles updates to existing events.
func (m *EventLifecycleManager) updateExistingEvent(ctx context.Context, existing, updated *models.Event) error {
	// Merge sources
	sourceMap := make(map[string]models.Source)
	for _, s := range existing.Sources {
		sourceMap[s.ID] = s
	}
	for _, s := range updated.Sources {
		sourceMap[s.ID] = s
	}

	// Convert back to slice
	mergedSources := make([]models.Source, 0, len(sourceMap))
	for _, s := range sourceMap {
		mergedSources = append(mergedSources, s)
	}

	// Update event with merged sources
	existing.Sources = mergedSources
	existing.UpdatedAt = time.Now()

	// Recalculate if we have more sources now
	if len(mergedSources) > m.config.MinSources {
		existing.Confidence.SourceCount = len(mergedSources)

		// Re-evaluate publication status
		if existing.Status == models.EventStatusRejected && m.shouldPublish(existing) {
			existing.Status = models.EventStatusPublished
			m.logger.Info("event promoted to published",
				"event_id", existing.ID,
				"source_count", len(mergedSources),
			)

			// Try to post to Twitter if enabled
			m.tryPostToTwitter(ctx, existing)
		}
	}

	return m.eventRepo.Update(ctx, *existing)
}

// ProcessResult contains the outcome of processing a batch of sources.
type ProcessResult struct {
	SourcesIngested int
	EventsEnriched  int
	EventsPublished int
	EventsRejected  int
	ErrorCount      int
	ProcessedAt     time.Time
}

// PublishEvent manually publishes a rejected event.
func (m *EventLifecycleManager) PublishEvent(ctx context.Context, eventID string) error {
	event, err := m.eventRepo.GetByID(ctx, eventID)
	if err != nil {
		return fmt.Errorf("failed to get event: %w", err)
	}

	if event == nil {
		return fmt.Errorf("event not found: %s", eventID)
	}

	if event.Status == models.EventStatusPublished {
		return fmt.Errorf("event already published")
	}

	err = m.eventRepo.UpdateStatus(ctx, eventID, models.EventStatusPublished)
	if err != nil {
		return err
	}

	// Try to post to Twitter if enabled (after status is updated)
	event.Status = models.EventStatusPublished
	m.tryPostToTwitter(ctx, event)

	return nil
}

// RejectEvent manually rejects a published event.
func (m *EventLifecycleManager) RejectEvent(ctx context.Context, eventID string) error {
	event, err := m.eventRepo.GetByID(ctx, eventID)
	if err != nil {
		return fmt.Errorf("failed to get event: %w", err)
	}

	if event == nil {
		return fmt.Errorf("event not found: %s", eventID)
	}

	if event.Status == models.EventStatusRejected {
		return fmt.Errorf("event already rejected")
	}

	return m.eventRepo.UpdateStatus(ctx, eventID, models.EventStatusRejected)
}

// ArchiveEvent moves an old event to archived status.
func (m *EventLifecycleManager) ArchiveEvent(ctx context.Context, eventID string) error {
	return m.eventRepo.UpdateStatus(ctx, eventID, models.EventStatusArchived)
}

// GetPublishedEvents retrieves published events with filtering.
func (m *EventLifecycleManager) GetPublishedEvents(ctx context.Context, query models.EventQuery) (*models.EventResponse, error) {
	// Ensure we only get published events
	if query.Status == nil {
		published := models.EventStatusPublished
		query.Status = &published
	}

	return m.eventRepo.Query(ctx, query)
}

// GetStats returns lifecycle statistics.
func (m *EventLifecycleManager) GetStats(ctx context.Context) (LifecycleStats, error) {
	// Query counts by status
	stats := LifecycleStats{}

	// Published
	published := models.EventStatusPublished
	publishedQuery := models.EventQuery{Status: &published, Limit: 1}
	resp, err := m.eventRepo.Query(ctx, publishedQuery)
	if err != nil {
		return stats, err
	}
	stats.Published = resp.Total

	// Enriched
	enriched := models.EventStatusEnriched
	enrichedQuery := models.EventQuery{Status: &enriched, Limit: 1}
	resp, err = m.eventRepo.Query(ctx, enrichedQuery)
	if err != nil {
		return stats, err
	}
	stats.Enriched = resp.Total

	// Rejected
	rejected := models.EventStatusRejected
	rejectedQuery := models.EventQuery{Status: &rejected, Limit: 1}
	resp, err = m.eventRepo.Query(ctx, rejectedQuery)
	if err != nil {
		return stats, err
	}
	stats.Rejected = resp.Total

	return stats, nil
}

// LifecycleStats contains event statistics by status.
type LifecycleStats struct {
	Published int
	Enriched  int
	Rejected  int
	Archived  int
}

// GetEvents retrieves events based on the provided query.
// This method is used by the REST API and MCP server.
func (m *EventLifecycleManager) GetEvents(query models.EventQuery) ([]models.Event, error) {
	// Validate query parameters
	if err := query.Validate(); err != nil {
		return nil, fmt.Errorf("invalid query: %w", err)
	}

	m.logger.Debug("querying events",
		"limit", query.Limit,
		"page", query.Page,
		"categories", len(query.Categories),
	)

	// Query events from repository
	resp, err := m.eventRepo.Query(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("failed to query events: %w", err)
	}

	m.logger.Info("retrieved events",
		"count", len(resp.Events),
		"total", resp.Total,
	)

	return resp.Events, nil
}

// GetEventCount returns the total count of events matching the query.
func (m *EventLifecycleManager) GetEventCount(query models.EventQuery) (int, error) {
	// Note: We don't call Validate() here to allow counting without limit restrictions
	return m.eventRepo.Count(context.Background(), query)
}

// GetEventByID retrieves a specific event by its ID.
func (m *EventLifecycleManager) GetEventByID(ctx context.Context, eventID string) (*models.Event, error) {
	return m.eventRepo.GetByID(ctx, eventID)
}

// Source management methods

// GetAllSources retrieves all sources from the database.
func (m *EventLifecycleManager) GetAllSources(ctx context.Context) ([]models.Source, error) {
	// Use ListRecent with a very old date to get all sources
	since := time.Now().Add(-365 * 24 * time.Hour) // 1 year ago
	return m.sourceRepo.ListRecent(ctx, since, 10000)
}

// GetSourceByID retrieves a single source by ID.
func (m *EventLifecycleManager) GetSourceByID(ctx context.Context, id string) (*models.Source, error) {
	return m.sourceRepo.GetByID(ctx, id)
}

// CreateSource creates a new source.
func (m *EventLifecycleManager) CreateSource(ctx context.Context, source *models.Source) error {
	// Set timestamps if not provided
	if source.CreatedAt.IsZero() {
		source.CreatedAt = time.Now()
	}
	if source.RetrievedAt.IsZero() {
		source.RetrievedAt = time.Now()
	}

	return m.sourceRepo.StoreRaw(ctx, *source)
}

// UpdateSource updates an existing source.
func (m *EventLifecycleManager) UpdateSource(ctx context.Context, source *models.Source) error {
	return m.sourceRepo.StoreRaw(ctx, *source)
}

// DeleteSource deletes a source (note: this would need to be added to the repository interface).
func (m *EventLifecycleManager) DeleteSource(ctx context.Context, id string) error {
	// For now, we don't have a Delete method in the repository
	// We could add it, or we could just return an error
	return fmt.Errorf("delete not implemented yet")
}

// formatNovelFacts creates a readable summary of novel facts.
func formatNovelFacts(facts []string) string {
	if len(facts) == 0 {
		return "Additional information discovered"
	}

	if len(facts) == 1 {
		return facts[0]
	}

	// For multiple facts, create a bulleted list
	formatted := ""
	for i, fact := range facts {
		if i > 0 {
			formatted += "; "
		}
		formatted += fact
	}

	// Limit summary length to keep it concise
	if len(formatted) > 300 {
		formatted = formatted[:297] + "..."
	}

	return formatted
}

// updateSourceEnrichmentStatus updates the enrichment status and optionally links to an event.
// This is a helper method that wraps the repository calls with proper interface checking.
func (m *EventLifecycleManager) updateSourceEnrichmentStatus(ctx context.Context, sourceID string, status models.EnrichmentStatus, errorMsg, eventID string) error {
	// Check if the source repository implements the extended interface
	type enrichmentUpdater interface {
		UpdateEnrichmentStatus(ctx context.Context, sourceID string, status models.EnrichmentStatus, errorMsg string) error
		SetEventID(ctx context.Context, sourceID, eventID string) error
	}

	repo, ok := m.sourceRepo.(enrichmentUpdater)
	if !ok {
		m.logger.Warn("source repository does not support enrichment status updates",
			"source_id", sourceID)
		return nil
	}

	// Update enrichment status
	if err := repo.UpdateEnrichmentStatus(ctx, sourceID, status, errorMsg); err != nil {
		return fmt.Errorf("failed to update enrichment status: %w", err)
	}

	// Set event ID if provided and enrichment was successful
	if eventID != "" && status == models.EnrichmentStatusCompleted {
		if err := repo.SetEventID(ctx, sourceID, eventID); err != nil {
			return fmt.Errorf("failed to set event_id: %w", err)
		}
	}

	return nil
}
