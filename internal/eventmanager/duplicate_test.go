package eventmanager

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/STRATINT/stratint/internal/models"
)

// TestProcessEvent_NoDuplicateCreation tests that the same event isn't created twice
func TestProcessEvent_NoDuplicateCreation(t *testing.T) {
	// Create mock repos
	eventRepo := &mockEventRepo{
		events: make(map[string]*models.Event),
	}

	thresholdRepo := &mockThresholdRepo{
		config: models.ThresholdConfig{
			MinConfidence:     0.5,
			MinMagnitude:      3.0,
			MaxSourceAgeHours: 24,
		},
	}

	manager := &EventLifecycleManager{
		eventRepo:     eventRepo,
		thresholdRepo: thresholdRepo,
		config: LifecycleConfig{
			AutoPublish: true,
			MinSources:  1,
		},
		logger: slog.Default(),
	}

	ctx := context.Background()

	// Create a test source
	source := models.Source{
		ID:          "source-1",
		Type:        models.SourceTypeNewsMedia,
		URL:         "https://test.com/article",
		RawContent:  "Test content",
		Credibility: 0.8,
	}

	// Create an event
	event := &models.Event{
		ID:       "test-event-1",
		Title:    "Test Event",
		Summary:  "Test summary",
		Category: models.CategoryGeopolitics,
		Sources:  []models.Source{source},
		Confidence: models.Confidence{
			Score: 0.8,
			Level: models.ConfidenceHigh,
		},
		Magnitude: 5.0,
		Timestamp: time.Now(),
		CreatedAt: time.Now(),
	}

	// Try to process the same event from 3 goroutines simultaneously
	var wg sync.WaitGroup
	errors := make([]error, 3)

	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			// Create a copy of the event
			eventCopy := *event
			errors[idx] = manager.ProcessEvent(ctx, &eventCopy)
		}(i)
	}

	wg.Wait()

	// Check how many times the event was created
	createCount := eventRepo.GetCreateCount(event.ID)

	if createCount > 1 {
		t.Errorf("Event %s was created %d times, expected 1", event.ID, createCount)
	}

	// Verify the event exists
	stored, err := eventRepo.GetByID(ctx, event.ID)
	if err != nil {
		t.Fatalf("Failed to get event: %v", err)
	}
	if stored == nil {
		t.Fatal("Event was not stored")
	}
}

// TestMultipleSourcesSameEvent tests that multiple sources can be enriched into separate events
func TestMultipleSourcesSameEvent(t *testing.T) {
	eventRepo := &mockEventRepo{
		events: make(map[string]*models.Event),
	}

	thresholdRepo := &mockThresholdRepo{
		config: models.ThresholdConfig{
			MinConfidence:     0.5,
			MinMagnitude:      3.0,
			MaxSourceAgeHours: 24,
		},
	}

	manager := &EventLifecycleManager{
		eventRepo:     eventRepo,
		thresholdRepo: thresholdRepo,
		config: LifecycleConfig{
			AutoPublish: true,
			MinSources:  1,
		},
		logger: slog.Default(),
	}

	ctx := context.Background()

	// Create 3 events from different sources
	sourceIDs := []string{"source-1", "source-2", "source-3"}
	for i, sourceID := range sourceIDs {
		source := models.Source{
			ID:          sourceID,
			Type:        models.SourceTypeNewsMedia,
			URL:         fmt.Sprintf("https://test.com/article-%d", i),
			RawContent:  fmt.Sprintf("Test content %d", i),
			Credibility: 0.8,
		}

		event := &models.Event{
			ID:       fmt.Sprintf("event-%d", i),
			Title:    fmt.Sprintf("Event %d", i),
			Summary:  "Test summary",
			Category: models.CategoryGeopolitics,
			Sources:  []models.Source{source},
			Confidence: models.Confidence{
				Score: 0.8,
				Level: models.ConfidenceHigh,
			},
			Magnitude: 5.0,
			Timestamp: time.Now(),
			CreatedAt: time.Now(),
		}

		if err := manager.ProcessEvent(ctx, event); err != nil {
			t.Errorf("Failed to process event for source %s: %v", sourceID, err)
		}
	}

	// Verify 3 events were created
	if len(eventRepo.events) != 3 {
		t.Errorf("Expected 3 events, got %d", len(eventRepo.events))
	}

	// Verify each event has the correct source
	for i, sourceID := range sourceIDs {
		eventID := fmt.Sprintf("event-%d", i)
		event, exists := eventRepo.events[eventID]
		if !exists {
			t.Errorf("Event %s doesn't exist", eventID)
			continue
		}
		if len(event.Sources) != 1 || event.Sources[0].ID != sourceID {
			t.Errorf("Event %s has wrong sources: %v", eventID, event.Sources)
		}
	}
}

// Mock implementations

type mockEventRepo struct {
	mu           sync.Mutex
	events       map[string]*models.Event
	createCounts map[string]int
}

func (m *mockEventRepo) Create(ctx context.Context, event models.Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if event already exists
	if _, exists := m.events[event.ID]; exists {
		return fmt.Errorf("event %s already exists", event.ID)
	}

	// Store the event
	eventCopy := event
	m.events[event.ID] = &eventCopy

	// Track create count
	if m.createCounts == nil {
		m.createCounts = make(map[string]int)
	}
	m.createCounts[event.ID]++

	return nil
}

func (m *mockEventRepo) GetByID(ctx context.Context, id string) (*models.Event, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	event, exists := m.events[id]
	if !exists {
		return nil, nil
	}

	return event, nil
}

func (m *mockEventRepo) GetCreateCount(eventID string) int {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.createCounts[eventID]
}

func (m *mockEventRepo) Update(ctx context.Context, event models.Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.events[event.ID]; !exists {
		return fmt.Errorf("event %s doesn't exist", event.ID)
	}

	eventCopy := event
	m.events[event.ID] = &eventCopy
	return nil
}

func (m *mockEventRepo) Delete(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.events, id)
	return nil
}

func (m *mockEventRepo) Query(ctx context.Context, query models.EventQuery) (*models.EventResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	events := make([]models.Event, 0)
	for _, event := range m.events {
		events = append(events, *event)
	}

	return &models.EventResponse{
		Events: events,
		Total:  len(events),
		Page:   1,
		Limit:  len(events),
	}, nil
}

func (m *mockEventRepo) Count(ctx context.Context, query models.EventQuery) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.events), nil
}

func (m *mockEventRepo) UpdateStatus(ctx context.Context, id string, status models.EventStatus) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	event, exists := m.events[id]
	if !exists {
		return fmt.Errorf("event %s doesn't exist", id)
	}

	event.Status = status
	return nil
}

func (m *mockEventRepo) HasSourceEvents(ctx context.Context, sourceID string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, event := range m.events {
		for _, src := range event.Sources {
			if src.ID == sourceID {
				return true, nil
			}
		}
	}

	return false, nil
}

type mockThresholdRepo struct {
	config models.ThresholdConfig
}

func (m *mockThresholdRepo) Get(ctx context.Context) (*models.ThresholdConfig, error) {
	return &m.config, nil
}

func (m *mockThresholdRepo) Update(ctx context.Context, config *models.ThresholdConfig) error {
	m.config = *config
	return nil
}
