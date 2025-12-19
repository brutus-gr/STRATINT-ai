package enrichment

import (
	"testing"
	"time"

	"github.com/STRATINT/stratint/internal/models"
)

func TestConfidenceScorer_Score(t *testing.T) {
	scorer := NewConfidenceScorer()

	source := models.Source{
		Type:        models.SourceTypeNewsMedia,
		Credibility: 0.85,
		PublishedAt: time.Now().Add(-1 * time.Hour),
		RawContent:  "A well-written, substantive article about recent developments in international relations.",
		Metadata: models.SourceMetadata{
			Language: "en",
		},
	}

	event := &models.Event{
		Title:   "International Summit Concluded",
		Summary: "Leaders from multiple nations reached agreement on key issues.",
	}

	entities := []models.Entity{
		{Confidence: 0.9},
		{Confidence: 0.85},
		{Confidence: 0.8},
	}

	confidence := scorer.Score(source, event, entities)

	if confidence.Score < 0.0 || confidence.Score > 1.0 {
		t.Errorf("confidence score out of range: %v", confidence.Score)
	}

	if confidence.Level == "" {
		t.Error("confidence level not set")
	}

	if confidence.Reasoning == "" {
		t.Error("reasoning should be provided")
	}
}
func TestConfidenceScorer_SourceTypeWeighting(t *testing.T) {
	scorer := NewConfidenceScorer()

	tests := []struct {
		sourceType      models.SourceType
		expectedMinimum float64
	}{
		{models.SourceTypeGovernment, 0.6},
		{models.SourceTypeNewsMedia, 0.6},
		{models.SourceTypeTwitter, 0.4},
		{models.SourceTypeGLP, 0.2},
	}

	for _, tt := range tests {
		t.Run(string(tt.sourceType), func(t *testing.T) {
			source := models.Source{
				Type:        tt.sourceType,
				Credibility: 0.8,
				PublishedAt: time.Now(),
				RawContent:  "Test content with reasonable length and quality.",
			}

			confidence := scorer.Score(source, &models.Event{}, []models.Entity{})

			if confidence.Score < tt.expectedMinimum {
				t.Errorf("expected score >= %v for %s, got %v",
					tt.expectedMinimum, tt.sourceType, confidence.Score)
			}
		})
	}
}

func TestConfidenceScorer_RecencyScore(t *testing.T) {
	scorer := NewConfidenceScorer()

	tests := []struct {
		name          string
		age           time.Duration
		expectedScore float64
	}{
		{"very recent", 30 * time.Minute, 1.0},
		{"recent", 3 * time.Hour, 0.9},
		{"today", 12 * time.Hour, 0.75},
		{"yesterday", 36 * time.Hour, 0.6},
		{"last week", 5 * 24 * time.Hour, 0.45},
		{"old", 10 * 24 * time.Hour, 0.3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			publishedAt := time.Now().Add(-tt.age)
			score := scorer.recencyScore(publishedAt)

			if score != tt.expectedScore {
				t.Errorf("expected score %v, got %v", tt.expectedScore, score)
			}
		})
	}
}

func TestConfidenceScorer_ContentQuality(t *testing.T) {
	scorer := NewConfidenceScorer()

	tests := []struct {
		name       string
		content    string
		expectHigh bool
	}{
		{
			name:       "quality content",
			content:    "This is a well-written, substantive article about recent developments. It contains balanced analysis and cites sources.",
			expectHigh: true,
		},
		{
			name:       "too short",
			content:    "Short!",
			expectHigh: false,
		},
		{
			name:       "all caps",
			content:    "THIS IS ALL CAPS CONTENT THAT SUGGESTS LOW QUALITY AND SENSATIONALISM",
			expectHigh: false,
		},
		{
			name:       "excessive exclamation",
			content:    "Breaking!!!!! Urgent!!!!! Must see!!!!! This is crazy!!!!! Unbelievable!!!!!",
			expectHigh: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := models.Source{RawContent: tt.content}
			score := scorer.assessContentQuality(source)

			if tt.expectHigh && score < 0.5 {
				t.Errorf("expected high quality score, got %v", score)
			} else if !tt.expectHigh && score > 0.6 {
				t.Errorf("expected low quality score, got %v", score)
			}
		})
	}
}

func TestIsAllCaps(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"THIS IS ALL CAPS", true},
		{"This is Mixed Case", false},
		{"this is lowercase", false},
		{"MOSTLY CAPS with some lowercase", false},
		{"Short", false},
		{"", false},
	}

	for _, tt := range tests {
		result := isAllCaps(tt.input)
		if result != tt.expected {
			t.Errorf("isAllCaps(%q) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestContainsSensationalism(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"BREAKING: Major event happening now", true},
		{"You won't believe what happened next", true},
		{"Shocking discovery exposed", true},
		{"Regular news article about policy changes", false},
		{"Analysis of recent economic trends", false},
	}

	for _, tt := range tests {
		result := containsSensationalism(tt.input)
		if result != tt.expected {
			t.Errorf("containsSensationalism(%q) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestMagnitudeEstimator_Estimate(t *testing.T) {
	estimator := NewMagnitudeEstimator()

	tests := []struct {
		name            string
		category        models.Category
		entityCount     int
		expectedMinimum float64
		expectedMaximum float64
	}{
		{
			name:            "terrorism event",
			category:        models.CategoryTerrorism,
			entityCount:     5,
			expectedMinimum: 8.0,
			expectedMaximum: 10.0,
		},
		{
			name:            "military event",
			category:        models.CategoryMilitary,
			entityCount:     3,
			expectedMinimum: 7.0,
			expectedMaximum: 10.0,
		},
		{
			name:            "economic event",
			category:        models.CategoryEconomic,
			entityCount:     2,
			expectedMinimum: 3.0,
			expectedMaximum: 6.0,
		},
		{
			name:            "other event",
			category:        models.CategoryOther,
			entityCount:     1,
			expectedMinimum: 2.0,
			expectedMaximum: 5.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entities := make([]models.Entity, tt.entityCount)
			for i := range entities {
				entities[i] = models.Entity{
					Type: models.EntityTypeCountry,
				}
			}

			event := &models.Event{
				Category: tt.category,
				Title:    "Test Event",
				Summary:  "Test summary",
				Entities: entities,
			}

			source := models.Source{}

			magnitude := estimator.Estimate(event, source)

			if magnitude < 0.0 || magnitude > 10.0 {
				t.Errorf("magnitude out of range [0, 10]: %v", magnitude)
			}

			if magnitude < tt.expectedMinimum || magnitude > tt.expectedMaximum {
				t.Errorf("expected magnitude in range [%v, %v], got %v",
					tt.expectedMinimum, tt.expectedMaximum, magnitude)
			}
		})
	}
}

func TestMagnitudeEstimator_UrgencyModifier(t *testing.T) {
	estimator := NewMagnitudeEstimator()

	tests := []struct {
		title    string
		summary  string
		expected float64
	}{
		{"Breaking: Attack reported", "Emergency situation", 0.6},
		{"Routine policy update", "No urgent action required", 0.0},
		{"War declared", "Invasion underway", 0.6},
	}

	for _, tt := range tests {
		result := estimator.urgencyModifier(tt.title, tt.summary)
		if result < 0.0 || result > 1.0 {
			t.Errorf("urgency modifier out of range: %v", result)
		}
	}
}

func TestMagnitudeEstimator_ScopeModifier(t *testing.T) {
	estimator := NewMagnitudeEstimator()

	tests := []struct {
		name     string
		entities []models.Entity
		expected float64
	}{
		{
			name: "international scope",
			entities: []models.Entity{
				{Type: models.EntityTypeCountry, Name: "USA"},
				{Type: models.EntityTypeCountry, Name: "China"},
				{Type: models.EntityTypeMilitaryUnit, Name: "5th Fleet"},
			},
			expected: 1.3, // 0.8 (2 countries) + 0.5 (military)
		},
		{
			name: "single country",
			entities: []models.Entity{
				{Type: models.EntityTypeCountry, Name: "France"},
			},
			expected: 0.2,
		},
		{
			name:     "no geographic entities",
			entities: []models.Entity{},
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := estimator.scopeModifier(tt.entities)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}
