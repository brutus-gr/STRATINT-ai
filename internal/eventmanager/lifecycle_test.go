package eventmanager

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/STRATINT/stratint/internal/config"
	"github.com/STRATINT/stratint/internal/enrichment"
	"github.com/STRATINT/stratint/internal/ingestion"
	"github.com/STRATINT/stratint/internal/logging"
	"github.com/STRATINT/stratint/internal/models"
)

// mockThresholdRepository is a simple mock for testing
type mockThresholdRepository struct {
	cfg *models.ThresholdConfig
}

func newMockThresholdRepository() *mockThresholdRepository {
	return &mockThresholdRepository{
		cfg: &models.ThresholdConfig{
			MinConfidence:     0.3,
			MinMagnitude:      1.0,
			MaxSourceAgeHours: 0,
		},
	}
}

func (m *mockThresholdRepository) Get(ctx context.Context) (*models.ThresholdConfig, error) {
	return m.cfg, nil
}

func (m *mockThresholdRepository) Update(ctx context.Context, cfg *models.ThresholdConfig) error {
	m.cfg = cfg
	return nil
}

func TestEventLifecycleManager_ProcessSources(t *testing.T) {
	sourceRepo := ingestion.NewMemorySourceRepository()
	eventRepo := ingestion.NewMemoryEventRepository()
	enricher := enrichment.NewMockEnricher()
	thresholdRepo := newMockThresholdRepository()
	// Set higher thresholds so low-quality source gets rejected
	thresholdRepo.cfg.MinConfidence = 0.5
	thresholdRepo.cfg.MinMagnitude = 4.0
	logger, _ := logging.New(config.LoggingConfig{Level: slog.LevelDebug, Format: "json"})

	config := DefaultLifecycleConfig()
	manager := NewEventLifecycleManager(sourceRepo, eventRepo, enricher, thresholdRepo, nil, nil, logger, config)

	ctx := context.Background()

	// Create test sources with completed scrape status
	sources := []models.Source{
		{
			ID:           "src-1",
			Type:         models.SourceTypeNewsMedia,
			URL:          "https://news.example.com/article1",
			Author:       "Journalist",
			PublishedAt:  time.Now(),
			RawContent:   "Breaking: Military exercises announced near border involving troops from United States. High-level diplomatic talks scheduled.",
			Credibility:  0.85,
			ScrapeStatus: models.ScrapeStatusCompleted,
		},
		{
			ID:           "src-2",
			Type:         models.SourceTypeBlog,
			URL:          "https://example.com/blog/123",
			Author:       "Anonymous",
			PublishedAt:  time.Now().Add(-48 * time.Hour),
			RawContent:   "!!!BREAKING!!! URGENT!!! YOU WONT BELIEVE!!!",
			Credibility:  0.2,
			ScrapeStatus: models.ScrapeStatusCompleted,
		},
	}

	// Process sources through lifecycle
	result, err := manager.ProcessSources(ctx, sources)
	if err != nil {
		t.Fatalf("ProcessSources failed: %v", err)
	}

	// Verify result
	if result.SourcesIngested != 2 {
		t.Errorf("Expected 2 sources ingested, got %d", result.SourcesIngested)
	}

	if result.EventsEnriched != 2 {
		t.Errorf("Expected 2 events enriched, got %d", result.EventsEnriched)
	}

	// High-quality source should be published
	if result.EventsPublished < 1 {
		t.Errorf("Expected at least 1 event published, got %d", result.EventsPublished)
	}

	// Low-quality source should be rejected
	if result.EventsRejected < 1 {
		t.Errorf("Expected at least 1 event rejected, got %d", result.EventsRejected)
	}
}

func TestEventLifecycleManager_ShouldPublish(t *testing.T) {
	logger, _ := logging.New(config.LoggingConfig{Level: slog.LevelDebug, Format: "json"})
	thresholdRepo := newMockThresholdRepository()
	// Update threshold repository to match test expectations
	thresholdRepo.cfg.MinConfidence = 0.5
	thresholdRepo.cfg.MinMagnitude = 5.0
	config := DefaultLifecycleConfig()
	config.MinSources = 1

	manager := NewEventLifecycleManager(nil, nil, nil, thresholdRepo, nil, nil, logger, config)

	tests := []struct {
		name     string
		event    *models.Event
		expected bool
	}{
		{
			name: "meets all criteria",
			event: &models.Event{
				Confidence: models.Confidence{Score: 0.8},
				Magnitude:  7.0,
				Sources:    []models.Source{{ID: "src-1"}},
			},
			expected: true,
		},
		{
			name: "low confidence",
			event: &models.Event{
				Confidence: models.Confidence{Score: 0.3},
				Magnitude:  7.0,
				Sources:    []models.Source{{ID: "src-1"}},
			},
			expected: false,
		},
		{
			name: "low magnitude",
			event: &models.Event{
				Confidence: models.Confidence{Score: 0.8},
				Magnitude:  3.0,
				Sources:    []models.Source{{ID: "src-1"}},
			},
			expected: false,
		},
		{
			name: "no sources",
			event: &models.Event{
				Confidence: models.Confidence{Score: 0.8},
				Magnitude:  7.0,
				Sources:    []models.Source{},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := manager.shouldPublish(tt.event)
			if result != tt.expected {
				t.Errorf("shouldPublish() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestEventLifecycleManager_PublishEvent(t *testing.T) {
	sourceRepo := ingestion.NewMemorySourceRepository()
	eventRepo := ingestion.NewMemoryEventRepository()
	enricher := enrichment.NewMockEnricher()
	thresholdRepo := newMockThresholdRepository()
	logger, _ := logging.New(config.LoggingConfig{Level: slog.LevelDebug, Format: "json"})

	config := DefaultLifecycleConfig()
	manager := NewEventLifecycleManager(sourceRepo, eventRepo, enricher, thresholdRepo, nil, nil, logger, config)

	ctx := context.Background()

	// Create a rejected event
	event := models.Event{
		ID:         "evt-1",
		Title:      "Test Event",
		Confidence: models.Confidence{Score: 0.5},
		Magnitude:  5.0,
		Status:     models.EventStatusRejected,
	}

	eventRepo.Create(ctx, event)

	// Manually publish it
	err := manager.PublishEvent(ctx, "evt-1")
	if err != nil {
		t.Fatalf("PublishEvent failed: %v", err)
	}

	// Verify status changed
	updated, _ := eventRepo.GetByID(ctx, "evt-1")
	if updated.Status != models.EventStatusPublished {
		t.Errorf("Expected status published, got %v", updated.Status)
	}
}

func TestEventLifecycleManager_RejectEvent(t *testing.T) {
	sourceRepo := ingestion.NewMemorySourceRepository()
	eventRepo := ingestion.NewMemoryEventRepository()
	enricher := enrichment.NewMockEnricher()
	thresholdRepo := newMockThresholdRepository()
	logger, _ := logging.New(config.LoggingConfig{Level: slog.LevelDebug, Format: "json"})

	config := DefaultLifecycleConfig()
	manager := NewEventLifecycleManager(sourceRepo, eventRepo, enricher, thresholdRepo, nil, nil, logger, config)

	ctx := context.Background()

	// Create a published event
	event := models.Event{
		ID:         "evt-1",
		Title:      "Test Event",
		Confidence: models.Confidence{Score: 0.8},
		Magnitude:  7.0,
		Status:     models.EventStatusPublished,
	}

	eventRepo.Create(ctx, event)

	// Manually reject it
	err := manager.RejectEvent(ctx, "evt-1")
	if err != nil {
		t.Fatalf("RejectEvent failed: %v", err)
	}

	// Verify status changed
	updated, _ := eventRepo.GetByID(ctx, "evt-1")
	if updated.Status != models.EventStatusRejected {
		t.Errorf("Expected status rejected, got %v", updated.Status)
	}
}

func TestEventLifecycleManager_GetPublishedEvents(t *testing.T) {
	sourceRepo := ingestion.NewMemorySourceRepository()
	eventRepo := ingestion.NewMemoryEventRepository()
	enricher := enrichment.NewMockEnricher()
	thresholdRepo := newMockThresholdRepository()
	logger, _ := logging.New(config.LoggingConfig{Level: slog.LevelDebug, Format: "json"})

	config := DefaultLifecycleConfig()
	manager := NewEventLifecycleManager(sourceRepo, eventRepo, enricher, thresholdRepo, nil, nil, logger, config)

	ctx := context.Background()

	// Create mix of published and rejected events
	events := []models.Event{
		{
			ID:         "evt-1",
			Magnitude:  8.0,
			Confidence: models.Confidence{Score: 0.9},
			Status:     models.EventStatusPublished,
		},
		{
			ID:         "evt-2",
			Magnitude:  3.0,
			Confidence: models.Confidence{Score: 0.4},
			Status:     models.EventStatusRejected,
		},
		{
			ID:         "evt-3",
			Magnitude:  6.0,
			Confidence: models.Confidence{Score: 0.7},
			Status:     models.EventStatusPublished,
		},
	}

	for _, e := range events {
		eventRepo.Create(ctx, e)
	}

	// Query published events
	query := models.EventQuery{
		Page:  1,
		Limit: 10,
	}

	response, err := manager.GetPublishedEvents(ctx, query)
	if err != nil {
		t.Fatalf("GetPublishedEvents failed: %v", err)
	}

	// Should only get published events
	if len(response.Events) != 2 {
		t.Errorf("Expected 2 published events, got %d", len(response.Events))
	}

	// Verify all are published
	for _, event := range response.Events {
		if event.Status != models.EventStatusPublished {
			t.Errorf("Expected published status, got %v", event.Status)
		}
	}
}

func TestEventLifecycleManager_GetStats(t *testing.T) {
	sourceRepo := ingestion.NewMemorySourceRepository()
	eventRepo := ingestion.NewMemoryEventRepository()
	enricher := enrichment.NewMockEnricher()
	thresholdRepo := newMockThresholdRepository()
	logger, _ := logging.New(config.LoggingConfig{Level: slog.LevelDebug, Format: "json"})

	config := DefaultLifecycleConfig()
	manager := NewEventLifecycleManager(sourceRepo, eventRepo, enricher, thresholdRepo, nil, nil, logger, config)

	ctx := context.Background()

	// Create events with different statuses
	events := []models.Event{
		{ID: "evt-1", Status: models.EventStatusPublished},
		{ID: "evt-2", Status: models.EventStatusPublished},
		{ID: "evt-3", Status: models.EventStatusRejected},
		{ID: "evt-4", Status: models.EventStatusEnriched},
	}

	for _, e := range events {
		eventRepo.Create(ctx, e)
	}

	// Get stats
	stats, err := manager.GetStats(ctx)
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}

	if stats.Published != 2 {
		t.Errorf("Expected 2 published, got %d", stats.Published)
	}

	if stats.Rejected != 1 {
		t.Errorf("Expected 1 rejected, got %d", stats.Rejected)
	}

	if stats.Enriched != 1 {
		t.Errorf("Expected 1 enriched, got %d", stats.Enriched)
	}
}

func TestDefaultLifecycleConfig(t *testing.T) {
	config := DefaultLifecycleConfig()

	if config.MinConfidence != 0.30 {
		t.Errorf("Expected MinConfidence 0.30, got %v", config.MinConfidence)
	}

	if config.MinMagnitude != 1.0 {
		t.Errorf("Expected MinMagnitude 1.0, got %v", config.MinMagnitude)
	}

	if config.MinSources != 1 {
		t.Errorf("Expected MinSources 1, got %v", config.MinSources)
	}

	if !config.AutoPublish {
		t.Error("Expected AutoPublish true")
	}
}
