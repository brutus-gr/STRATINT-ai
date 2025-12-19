package enrichment

import (
	"context"
	"testing"
	"time"

	"github.com/STRATINT/stratint/internal/models"
)

func TestMockEnricher_Enrich(t *testing.T) {
	enricher := NewMockEnricher()
	ctx := context.Background()

	source := models.Source{
		ID:          "src-test-1",
		Type:        models.SourceTypeTwitter,
		URL:         "https://twitter.com/user/status/123",
		Author:      "TestUser",
		PublishedAt: time.Now(),
		RawContent:  "Breaking: Military exercises conducted near the border involving troops from United States and allied nations. #Military #Defense",
		Credibility: 0.7,
		Metadata: models.SourceMetadata{
			TweetID:      "123",
			RetweetCount: 50,
			LikeCount:    200,
			Hashtags:     []string{"Military", "Defense"},
			Language:     "en",
		},
	}

	event, err := enricher.Enrich(ctx, source)
	if err != nil {
		t.Fatalf("Enrich failed: %v", err)
	}

	// Verify event structure
	if event.ID == "" {
		t.Error("Event ID should be set")
	}

	if event.Title == "" {
		t.Error("Event title should be set")
	}

	if event.Summary == "" {
		t.Error("Event summary should be set")
	}

	if event.Category == "" {
		t.Error("Event category should be set")
	}

	// Should infer military category
	if event.Category != models.CategoryMilitary {
		t.Errorf("Expected military category, got %v", event.Category)
	}

	// Verify confidence scoring
	if event.Confidence.Score < 0.0 || event.Confidence.Score > 1.0 {
		t.Errorf("Confidence score out of range: %v", event.Confidence.Score)
	}

	if event.Confidence.Level == "" {
		t.Error("Confidence level should be set")
	}

	// Verify magnitude estimation
	if event.Magnitude < 0.0 || event.Magnitude > 10.0 {
		t.Errorf("Magnitude out of range: %v", event.Magnitude)
	}

	// Military events should have higher magnitude
	if event.Magnitude < 5.0 {
		t.Errorf("Expected military event to have magnitude >= 5.0, got %v", event.Magnitude)
	}

	// Verify entities
	if len(event.Entities) == 0 {
		t.Error("Should extract at least one entity")
	}

	// Verify source attribution
	if len(event.Sources) != 1 {
		t.Errorf("Expected 1 source, got %d", len(event.Sources))
	}

	// Verify status
	if event.Status != models.EventStatusEnriched {
		t.Errorf("Expected status enriched, got %v", event.Status)
	}
}

func TestMockEnricher_EnrichBatch(t *testing.T) {
	enricher := NewMockEnricher()
	ctx := context.Background()

	sources := []models.Source{
		{
			ID:          "src-1",
			Type:        models.SourceTypeTwitter,
			PublishedAt: time.Now(),
			RawContent:  "Diplomatic summit between leaders concludes with treaty signing. #Diplomacy",
			Credibility: 0.8,
		},
		{
			ID:          "src-2",
			Type:        models.SourceTypeTelegram,
			PublishedAt: time.Now(),
			RawContent:  "Earthquake strikes major city, humanitarian relief efforts underway. #Disaster",
			Credibility: 0.75,
		},
		{
			ID:          "src-3",
			Type:        models.SourceTypeNewsMedia,
			PublishedAt: time.Now(),
			RawContent:  "Cyber attack disrupts critical infrastructure, investigation ongoing. #Cyber #Security",
			Credibility: 0.9,
		},
	}

	events, err := enricher.EnrichBatch(ctx, sources)
	if err != nil {
		t.Fatalf("EnrichBatch failed: %v", err)
	}

	if len(events) != len(sources) {
		t.Errorf("Expected %d events, got %d", len(sources), len(events))
	}

	// Verify each event
	for i, event := range events {
		if event.ID == "" {
			t.Errorf("Event %d missing ID", i)
		}

		if event.Category == "" {
			t.Errorf("Event %d missing category", i)
		}

		if event.Confidence.Score == 0 {
			t.Errorf("Event %d has zero confidence", i)
		}

		if event.Magnitude == 0 {
			t.Errorf("Event %d has zero magnitude", i)
		}
	}

	// Verify category inference
	if events[0].Category != models.CategoryDiplomacy {
		t.Errorf("Expected diplomacy category for event 0, got %v", events[0].Category)
	}

	if events[1].Category != models.CategoryDisaster {
		t.Errorf("Expected disaster category for event 1, got %v", events[1].Category)
	}

	if events[2].Category != models.CategoryCyber {
		t.Errorf("Expected cyber category for event 2, got %v", events[2].Category)
	}
}

func TestInferCategory(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected models.Category
	}{
		{
			name:     "military content",
			content:  "Army troops deployed to the region for military exercises",
			expected: models.CategoryMilitary,
		},
		{
			name:     "terrorism content",
			content:  "Terror attack reported in city center, explosion heard",
			expected: models.CategoryTerrorism,
		},
		{
			name:     "cyber content",
			content:  "Cyber attack targets government systems, hackers breach network",
			expected: models.CategoryCyber,
		},
		{
			name:     "disaster content",
			content:  "Earthquake strikes coastal region, emergency response activated",
			expected: models.CategoryDisaster,
		},
		{
			name:     "economic content",
			content:  "Financial markets react to new trade policies, economic impact assessed",
			expected: models.CategoryEconomic,
		},
		{
			name:     "generic content",
			content:  "Some random text without specific keywords",
			expected: models.CategoryOther,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := inferCategory(tt.content)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestExtractTags(t *testing.T) {
	content := "Breaking news about #Military operations in #Europe. Multiple sources confirm #Breaking developments."

	tags := extractTags(content)

	if len(tags) == 0 {
		t.Fatal("Expected to extract tags")
	}

	// Check for hashtag extraction
	hasHashtag := false
	for _, tag := range tags {
		if tag == "Military" || tag == "Europe" || tag == "Breaking" {
			hasHashtag = true
			break
		}
	}

	if !hasHashtag {
		t.Error("Should extract hashtags from content")
	}
}

func TestExtractMockEntities(t *testing.T) {
	enricher := NewMockEnricher()

	tests := []struct {
		name          string
		content       string
		expectedTypes []models.EntityType
	}{
		{
			name:          "country entities",
			content:       "Discussions between United States and China regarding trade policies",
			expectedTypes: []models.EntityType{models.EntityTypeCountry, models.EntityTypeCountry},
		},
		{
			name:          "person entity with title",
			content:       "President announced new policy measures during press conference",
			expectedTypes: []models.EntityType{models.EntityTypePerson},
		},
		{
			name:          "mixed entities",
			content:       "General from United States met with Minister from Russia to discuss military cooperation",
			expectedTypes: []models.EntityType{models.EntityTypePerson, models.EntityTypeCountry, models.EntityTypeCountry},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entities := enricher.extractMockEntities(tt.content)

			if len(entities) < len(tt.expectedTypes) {
				t.Errorf("Expected at least %d entities, got %d", len(tt.expectedTypes), len(entities))
			}

			// Verify entity structure
			for _, entity := range entities {
				if entity.ID == "" {
					t.Error("Entity should have ID")
				}
				if entity.Type == "" {
					t.Error("Entity should have type")
				}
				if entity.Name == "" {
					t.Error("Entity should have name")
				}
				if entity.Confidence <= 0 || entity.Confidence > 1 {
					t.Errorf("Entity confidence out of range: %v", entity.Confidence)
				}
			}
		})
	}
}

func TestEnrichment_HighQualitySource(t *testing.T) {
	enricher := NewMockEnricher()
	ctx := context.Background()

	// High-quality government source
	source := models.Source{
		ID:          "src-gov-1",
		Type:        models.SourceTypeGovernment,
		URL:         "https://state.gov/press/release",
		Author:      "State Department",
		PublishedAt: time.Now(),
		RawContent:  "Official statement: Diplomatic relations established with neighboring nation. Treaty signed by both parties. Joint economic cooperation framework initiated.",
		Credibility: 0.95,
		Metadata: models.SourceMetadata{
			Language: "en",
		},
	}

	event, err := enricher.Enrich(ctx, source)
	if err != nil {
		t.Fatalf("Enrich failed: %v", err)
	}

	// High-quality government source should have high confidence
	if event.Confidence.Score < 0.7 {
		t.Errorf("Expected high confidence (>0.7) for government source, got %v", event.Confidence.Score)
	}

	// Should be high or verified confidence level
	if event.Confidence.Level != models.ConfidenceHigh && event.Confidence.Level != models.ConfidenceVerified {
		t.Errorf("Expected high or verified confidence level, got %v", event.Confidence.Level)
	}
}

func TestEnrichment_LowQualitySource(t *testing.T) {
	enricher := NewMockEnricher()
	ctx := context.Background()

	// Low-quality blog source
	source := models.Source{
		ID:          "src-blog-1",
		Type:        models.SourceTypeBlog,
		URL:         "https://example.com/blog/post/123",
		Author:      "Anonymous",
		PublishedAt: time.Now().Add(-48 * time.Hour), // Old
		RawContent:  "BREAKING!!!!! URGENT!!!!! YOU WONT BELIEVE THIS!!!!!",
		Credibility: 0.2,
		Metadata:    models.SourceMetadata{},
	}

	event, err := enricher.Enrich(ctx, source)
	if err != nil {
		t.Fatalf("Enrich failed: %v", err)
	}

	// Low-quality source should have lower confidence
	if event.Confidence.Score > 0.5 {
		t.Errorf("Expected low confidence (<0.5) for low-quality blog source, got %v", event.Confidence.Score)
	}

	// Should be low or medium confidence level
	if event.Confidence.Level == models.ConfidenceVerified || event.Confidence.Level == models.ConfidenceHigh {
		t.Errorf("Expected low or medium confidence level, got %v", event.Confidence.Level)
	}
}
