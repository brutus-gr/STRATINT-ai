package models

import (
	"testing"
	"time"
)

func TestConfidence_DeriveLevel(t *testing.T) {
	tests := []struct {
		name     string
		score    float64
		expected ConfidenceLevel
	}{
		{"Verified high score", 0.95, ConfidenceVerified},
		{"Verified threshold", 0.85, ConfidenceVerified},
		{"High score", 0.75, ConfidenceHigh},
		{"High threshold", 0.6, ConfidenceHigh},
		{"Medium score", 0.45, ConfidenceMedium},
		{"Medium threshold", 0.3, ConfidenceMedium},
		{"Low score", 0.15, ConfidenceLow},
		{"Zero score", 0.0, ConfidenceLow},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Confidence{Score: tt.score}
			if got := c.DeriveLevel(); got != tt.expected {
				t.Errorf("DeriveLevel() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestEvent_IsPublishable(t *testing.T) {
	baseEvent := Event{
		ID:         "test-1",
		Title:      "Test Event",
		Magnitude:  5.0,
		Confidence: Confidence{Score: 0.5},
		Sources:    []Source{{ID: "source-1"}},
	}

	tests := []struct {
		name     string
		event    Event
		expected bool
	}{
		{
			name:     "Publishable event",
			event:    baseEvent,
			expected: true,
		},
		{
			name: "Low confidence",
			event: Event{
				ID:         "test-2",
				Magnitude:  5.0,
				Confidence: Confidence{Score: 0.2},
				Sources:    []Source{{ID: "source-1"}},
			},
			expected: false,
		},
		{
			name: "Low magnitude",
			event: Event{
				ID:         "test-3",
				Magnitude:  0.5,
				Confidence: Confidence{Score: 0.5},
				Sources:    []Source{{ID: "source-1"}},
			},
			expected: false,
		},
		{
			name: "No sources",
			event: Event{
				ID:         "test-4",
				Magnitude:  5.0,
				Confidence: Confidence{Score: 0.5},
				Sources:    []Source{},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.event.IsPublishable(); got != tt.expected {
				t.Errorf("IsPublishable() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestEventStatus(t *testing.T) {
	statuses := []EventStatus{
		EventStatusPending,
		EventStatusEnriched,
		EventStatusPublished,
		EventStatusArchived,
		EventStatusRejected,
	}

	for _, status := range statuses {
		if status == "" {
			t.Errorf("EventStatus should not be empty")
		}
	}
}

func TestCategory(t *testing.T) {
	categories := []Category{
		CategoryGeopolitics,
		CategoryMilitary,
		CategoryEconomic,
		CategoryCyber,
		CategoryDisaster,
		CategoryTerrorism,
		CategoryDiplomacy,
		CategoryIntelligence,
		CategoryHumanitarian,
		CategoryOther,
	}

	for _, cat := range categories {
		if cat == "" {
			t.Errorf("Category should not be empty")
		}
	}
}

func TestLocation(t *testing.T) {
	loc := Location{
		Latitude:  40.7128,
		Longitude: -74.0060,
		City:      "New York",
		Country:   "United States",
		Region:    "New York",
	}

	if loc.Latitude == 0 || loc.Longitude == 0 {
		t.Error("Location coordinates should be set")
	}
	if loc.City == "" {
		t.Error("City should be set")
	}
}

func TestEvent_FullLifecycle(t *testing.T) {
	now := time.Now()
	event := Event{
		ID:        "evt-123",
		Timestamp: now,
		Title:     "Test Event",
		Summary:   "A test OSINT event",
		Magnitude: 7.5,
		Confidence: Confidence{
			Score:       0.85,
			SourceCount: 3,
			Reasoning:   "Multiple corroborating sources",
		},
		Category: CategoryGeopolitics,
		Sources: []Source{
			{ID: "src-1"},
			{ID: "src-2"},
			{ID: "src-3"},
		},
		Status: EventStatusEnriched,
	}

	// Test derived level
	event.Confidence.Level = event.Confidence.DeriveLevel()
	if event.Confidence.Level != ConfidenceVerified {
		t.Errorf("Expected ConfidenceVerified, got %v", event.Confidence.Level)
	}

	// Test publishability
	if !event.IsPublishable() {
		t.Error("Event should be publishable")
	}
}
