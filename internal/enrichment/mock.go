package enrichment

import (
	"context"
	"fmt"
	"time"

	"github.com/STRATINT/stratint/internal/models"
)

// MockEnricher provides a test implementation of the Enricher interface.
type MockEnricher struct {
	scorer    *ConfidenceScorer
	estimator *MagnitudeEstimator
}

// NewMockEnricher creates a mock enricher for testing without OpenAI API calls.
func NewMockEnricher() *MockEnricher {
	return &MockEnricher{
		scorer:    NewConfidenceScorer(),
		estimator: NewMagnitudeEstimator(),
	}
}

// Enrich processes a source using rule-based analysis (no AI calls).
func (m *MockEnricher) Enrich(ctx context.Context, source models.Source) (*models.Event, error) {
	// Simple rule-based enrichment for testing
	event := &models.Event{
		ID:         generateEventID(source),
		Timestamp:  source.PublishedAt,
		Title:      extractTitle(source.RawContent),
		Summary:    extractSummary(source.RawContent),
		RawContent: source.RawContent,
		Category:   inferCategory(source.RawContent),
		Tags:       extractTags(source.RawContent),
		Sources:    []models.Source{source},
		Status:     models.EventStatusEnriched,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// Extract mock entities
	entities := m.extractMockEntities(source.RawContent)
	event.Entities = entities

	// Calculate confidence
	confidence := m.scorer.Score(source, event, entities)
	event.Confidence = confidence

	// Estimate magnitude
	magnitude := m.estimator.Estimate(event, source)
	event.Magnitude = magnitude

	return event, nil
}

// EnrichBatch processes multiple sources.
func (m *MockEnricher) EnrichBatch(ctx context.Context, sources []models.Source) ([]models.Event, error) {
	events := make([]models.Event, 0, len(sources))

	for _, source := range sources {
		event, err := m.Enrich(ctx, source)
		if err != nil {
			return events, err
		}
		events = append(events, *event)
	}

	return events, nil
}

// ExtractArticleText returns error for mock enricher (not supported)
func (m *MockEnricher) ExtractArticleText(ctx context.Context, html, url string) (string, error) {
	return "", fmt.Errorf("mock enricher does not support article text extraction")
}

// extractTitle creates a title from content.
func extractTitle(content string) string {
	if len(content) > 100 {
		return content[:100] + "..."
	}
	return content
}

// extractSummary creates a summary from content.
func extractSummary(content string) string {
	if len(content) > 250 {
		return content[:250] + "..."
	}
	return content
}

// inferCategory guesses the category from keywords.
func inferCategory(content string) models.Category {
	lower := content
	if len(content) > 500 {
		lower = content[:500]
	}

	// Check categories in order (specific to general)
	type categoryKeywords struct {
		category models.Category
		keywords []string
	}

	orderedCategories := []categoryKeywords{
		{models.CategoryCyber, []string{"cyber", "hack", "breach", "malware", "ransomware"}},
		{models.CategoryTerrorism, []string{"terror", "bombing", "hostage"}},
		{models.CategoryDiplomacy, []string{"diplomatic", "ambassador", "embassy", "foreign minister", "summit", "treaty"}},
		{models.CategoryMilitary, []string{"military", "army", "navy", "troops", "soldiers", "war", "combat"}},
		{models.CategoryDisaster, []string{"earthquake", "flood", "hurricane", "disaster"}},
		{models.CategoryGeopolitics, []string{"sanctions", "alliance", "geopolitical"}},
		{models.CategoryEconomic, []string{"economic", "trade", "market", "financial", "economy"}},
		{models.CategoryIntelligence, []string{"intelligence", "spy", "surveillance", "classified"}},
		{models.CategoryHumanitarian, []string{"refugee", "humanitarian", "aid", "relief"}},
	}

	for _, ck := range orderedCategories {
		for _, word := range ck.keywords {
			if containsWord(lower, word) {
				return ck.category
			}
		}
	}

	return models.CategoryOther
}

// containsWord checks if text contains a word (case-insensitive).
func containsWord(text, word string) bool {
	if len(text) == 0 || len(word) == 0 {
		return false
	}

	// Convert to lowercase for case-insensitive matching
	lowerText := ""
	lowerWord := ""

	for _, ch := range text {
		if ch >= 'A' && ch <= 'Z' {
			lowerText += string(ch + 32)
		} else {
			lowerText += string(ch)
		}
	}

	for _, ch := range word {
		if ch >= 'A' && ch <= 'Z' {
			lowerWord += string(ch + 32)
		} else {
			lowerWord += string(ch)
		}
	}

	// Simple substring check
	for i := 0; i <= len(lowerText)-len(lowerWord); i++ {
		if lowerText[i:i+len(lowerWord)] == lowerWord {
			return true
		}
	}

	return false
}

// extractTags pulls hashtags and keywords from content.
func extractTags(content string) []string {
	tags := []string{}

	// Extract hashtags
	for i := 0; i < len(content)-1; i++ {
		if content[i] == '#' {
			end := i + 1
			for end < len(content) && isAlphanumeric(content[end]) {
				end++
			}
			if end > i+1 {
				tag := content[i+1 : end]
				tags = append(tags, tag)
			}
		}
	}

	// Add category-based tags
	category := inferCategory(content)
	tags = append(tags, string(category))

	return tags
}

// isAlphanumeric checks if a byte is alphanumeric.
func isAlphanumeric(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')
}

// extractMockEntities creates simple mock entities for testing.
func (m *MockEnricher) extractMockEntities(content string) []models.Entity {
	entities := []models.Entity{}

	// Simple pattern matching for common entities
	countries := []string{"United States", "Russia", "China", "Ukraine", "Israel", "Iran"}
	for _, country := range countries {
		if containsWord(content, country) {
			entities = append(entities, models.Entity{
				ID:             generateEntityID(),
				Type:           models.EntityTypeCountry,
				Name:           country,
				NormalizedName: country,
				Confidence:     0.85,
				Context:        fmt.Sprintf("mentioned in content"),
			})
		}
	}

	// Add mock person entity if content mentions common titles
	titles := []string{"President", "Minister", "General", "Ambassador"}
	for _, title := range titles {
		if containsWord(content, title) {
			entities = append(entities, models.Entity{
				ID:             generateEntityID(),
				Type:           models.EntityTypePerson,
				Name:           title + " (unnamed)",
				NormalizedName: title,
				Confidence:     0.60,
				Context:        "title mentioned",
				Attributes: models.EntityAttrs{
					Title: title,
				},
			})
			break // Only add one
		}
	}

	return entities
}
