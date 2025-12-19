package enrichment

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/STRATINT/stratint/internal/models"
)

// ConfidenceScorer calculates confidence scores for OSINT events.
type ConfidenceScorer struct {
	sourceWeights map[models.SourceType]float64
}

// NewConfidenceScorer creates a new confidence scorer with default weights.
func NewConfidenceScorer() *ConfidenceScorer {
	return &ConfidenceScorer{
		sourceWeights: map[models.SourceType]float64{
			models.SourceTypeGovernment: 0.95,
			models.SourceTypeNewsMedia:  0.85,
			models.SourceTypeTwitter:    0.60,
			models.SourceTypeTelegram:   0.55,
			models.SourceTypeBlog:       0.45,
			models.SourceTypeGLP:        0.25,
			models.SourceTypeOther:      0.40,
		},
	}
}

// Score calculates a comprehensive confidence score for an event.
func (s *ConfidenceScorer) Score(source models.Source, event *models.Event, entities []models.Entity) models.Confidence {
	// Check if the event indicates insufficient data for analysis
	insufficientDataPhrases := []string{
		"insufficient data",
		"lacks sufficient detail",
		"not enough information",
		"missing critical details",
		"unable to provide",
		"cannot be determined",
		"information is too limited",
		"provided information lacks",
	}

	summaryLower := strings.ToLower(event.Summary)
	hasInsufficientData := false
	for _, phrase := range insufficientDataPhrases {
		if strings.Contains(summaryLower, phrase) {
			hasInsufficientData = true
			break
		}
	}

	factors := []scoreFactor{
		{name: "source_credibility", weight: 0.35, score: source.Credibility},
		{name: "source_type", weight: 0.25, score: s.sourceWeights[source.Type]},
		{name: "entity_confidence", weight: 0.15, score: s.averageEntityConfidence(entities)},
		{name: "content_quality", weight: 0.15, score: s.assessContentQuality(source)},
		{name: "recency", weight: 0.10, score: s.recencyScore(source.PublishedAt)},
	}

	// Calculate weighted average
	totalScore := 0.0
	totalWeight := 0.0

	for _, factor := range factors {
		totalScore += factor.score * factor.weight
		totalWeight += factor.weight
	}

	finalScore := totalScore / totalWeight

	// If analysis indicates insufficient data, cap confidence at 0.05
	if hasInsufficientData {
		finalScore = math.Min(finalScore, 0.05)
	}

	// Clamp to [0, 1]
	finalScore = math.Max(0.0, math.Min(1.0, finalScore))

	confidence := models.Confidence{
		Score:       finalScore,
		Level:       models.ConfidenceLow, // Will be set by DeriveLevel
		SourceCount: 1,
		Reasoning:   s.buildReasoning(factors, finalScore),
	}

	confidence.Level = confidence.DeriveLevel()

	return confidence
}

type scoreFactor struct {
	name   string
	weight float64
	score  float64
}

// averageEntityConfidence calculates mean confidence across extracted entities.
func (s *ConfidenceScorer) averageEntityConfidence(entities []models.Entity) float64 {
	if len(entities) == 0 {
		return 0.5 // Neutral score if no entities
	}

	total := 0.0
	for _, entity := range entities {
		total += entity.Confidence
	}

	return total / float64(len(entities))
}

// assessContentQuality evaluates the quality of the source content.
func (s *ConfidenceScorer) assessContentQuality(source models.Source) float64 {
	score := 0.5 // Start at neutral

	content := source.RawContent
	contentLen := len(content)

	// Length factor (prefer substantive content)
	if contentLen < 50 {
		score -= 0.2 // Too short
	} else if contentLen > 200 && contentLen < 2000 {
		score += 0.2 // Good length
	} else if contentLen > 5000 {
		score -= 0.1 // Very long, harder to verify
	}

	// Check for indicators of quality
	if strings.Contains(content, "http") {
		score += 0.05 // Contains links (can be verified)
	}

	// Check for spam indicators
	if strings.Count(content, "!") > 5 {
		score -= 0.1 // Excessive exclamation (sensationalism)
	}

	if isAllCaps(content) {
		score -= 0.15 // ALL CAPS suggests low quality
	}

	// Check for balanced language
	if !containsSensationalism(content) {
		score += 0.1
	}

	return math.Max(0.0, math.Min(1.0, score))
}

// recencyScore gives higher scores to more recent content.
func (s *ConfidenceScorer) recencyScore(publishedAt time.Time) float64 {
	age := time.Since(publishedAt)

	// Decay function: 1.0 for <1h, 0.9 for 1-6h, 0.7 for 6-24h, etc.
	hours := age.Hours()

	if hours < 1 {
		return 1.0
	} else if hours < 6 {
		return 0.9
	} else if hours < 24 {
		return 0.75
	} else if hours < 72 {
		return 0.6
	} else if hours < 168 { // 1 week
		return 0.45
	} else {
		return 0.3
	}
}

// metadataScore evaluates richness of source metadata.
func (s *ConfidenceScorer) metadataScore(source models.Source) float64 {
	score := 0.0
	meta := source.Metadata

	// Twitter metadata
	if meta.TweetID != "" {
		score += 0.2
		if meta.RetweetCount > 10 {
			score += 0.1
		}
		if meta.LikeCount > 50 {
			score += 0.1
		}
	}

	// Telegram metadata
	if meta.ChannelID != "" {
		score += 0.2
		if meta.ViewCount > 1000 {
			score += 0.1
		}
	}

	// Hashtags and mentions
	if len(meta.Hashtags) > 0 {
		score += 0.05
	}
	if len(meta.Mentions) > 0 {
		score += 0.05
	}

	// Language indicator
	if meta.Language != "" {
		score += 0.05
	}

	return math.Min(1.0, score)
}

// buildReasoning generates human-readable explanation for confidence score.
func (s *ConfidenceScorer) buildReasoning(factors []scoreFactor, finalScore float64) string {
	parts := []string{}

	for _, factor := range factors {
		if factor.score > 0.7 {
			parts = append(parts, fmt.Sprintf("High %s (%.2f)", factor.name, factor.score))
		} else if factor.score < 0.4 {
			parts = append(parts, fmt.Sprintf("Low %s (%.2f)", factor.name, factor.score))
		}
	}

	if len(parts) == 0 {
		return "Moderate confidence across all factors"
	}

	reasoning := strings.Join(parts, "; ")
	return fmt.Sprintf("Final score: %.2f. %s", finalScore, reasoning)
}

// isAllCaps checks if text is predominantly uppercase.
func isAllCaps(text string) bool {
	if len(text) < 10 {
		return false
	}

	upper := 0
	lower := 0

	for _, ch := range text {
		if ch >= 'A' && ch <= 'Z' {
			upper++
		} else if ch >= 'a' && ch <= 'z' {
			lower++
		}
	}

	if upper+lower == 0 {
		return false
	}

	return float64(upper)/float64(upper+lower) > 0.7
}

// containsSensationalism checks for sensationalist language.
func containsSensationalism(text string) bool {
	lower := strings.ToLower(text)

	sensationalWords := []string{
		"breaking:", "urgent:", "must see", "you won't believe",
		"shocking", "unbelievable", "exposed", "destroyed",
	}

	for _, word := range sensationalWords {
		if strings.Contains(lower, word) {
			return true
		}
	}

	return false
}

// MagnitudeEstimator calculates event severity/importance scores.
type MagnitudeEstimator struct {
	categoryWeights map[models.Category]float64
}

// NewMagnitudeEstimator creates a new magnitude estimator.
func NewMagnitudeEstimator() *MagnitudeEstimator {
	return &MagnitudeEstimator{
		categoryWeights: map[models.Category]float64{
			models.CategoryTerrorism:    9.0, // Highest base magnitude
			models.CategoryMilitary:     8.0,
			models.CategoryDisaster:     7.5,
			models.CategoryGeopolitics:  7.0,
			models.CategoryIntelligence: 6.5,
			models.CategoryCyber:        6.0,
			models.CategoryDiplomacy:    5.5,
			models.CategoryHumanitarian: 5.0,
			models.CategoryEconomic:     4.5,
			models.CategoryOther:        3.0,
		},
	}
}

// Estimate calculates a 0-10 magnitude score for an event.
func (e *MagnitudeEstimator) Estimate(event *models.Event, source models.Source) float64 {
	// Start with category base magnitude
	baseMagnitude := e.categoryWeights[event.Category]

	// Apply modifiers
	modifiers := []float64{
		e.entityCountModifier(event.Entities),
		e.engagementModifier(source.Metadata),
		e.urgencyModifier(event.Title, event.Summary),
		e.scopeModifier(event.Entities),
	}

	totalModifier := 0.0
	for _, mod := range modifiers {
		totalModifier += mod
	}

	magnitude := baseMagnitude + totalModifier

	// Clamp to [0, 10]
	magnitude = math.Max(0.0, math.Min(10.0, magnitude))

	return magnitude
}

// entityCountModifier adjusts magnitude based on number of entities.
func (e *MagnitudeEstimator) entityCountModifier(entities []models.Entity) float64 {
	count := len(entities)

	if count < 2 {
		return -0.5 // Few entities suggests smaller scope
	} else if count >= 5 {
		return 1.0 // Many entities suggests larger scope
	}

	return 0.0
}

// engagementModifier considers social metrics.
func (e *MagnitudeEstimator) engagementModifier(meta models.SourceMetadata) float64 {
	modifier := 0.0

	// Twitter engagement
	if meta.RetweetCount > 1000 || meta.LikeCount > 5000 {
		modifier += 0.5
	}

	// Telegram views
	if meta.ViewCount > 10000 {
		modifier += 0.5
	}

	return math.Min(1.5, modifier)
}

// urgencyModifier checks for urgent language.
func (e *MagnitudeEstimator) urgencyModifier(title, summary string) float64 {
	text := strings.ToLower(title + " " + summary)

	urgentTerms := []string{
		"breaking", "urgent", "emergency", "crisis",
		"attack", "explosion", "killed", "war",
		"invasion", "strike", "deployed",
	}

	matches := 0
	for _, term := range urgentTerms {
		if strings.Contains(text, term) {
			matches++
		}
	}

	return math.Min(1.0, float64(matches)*0.3)
}

// scopeModifier considers geographic and organizational scope.
func (e *MagnitudeEstimator) scopeModifier(entities []models.Entity) float64 {
	countries := 0
	militaryUnits := 0
	organizations := 0

	for _, entity := range entities {
		switch entity.Type {
		case models.EntityTypeCountry:
			countries++
		case models.EntityTypeMilitaryUnit:
			militaryUnits++
		case models.EntityTypeOrganization:
			organizations++
		}
	}

	modifier := 0.0

	// Multiple countries = international scope
	if countries >= 2 {
		modifier += 0.8
	} else if countries == 1 {
		modifier += 0.2
	}

	// Military involvement
	if militaryUnits > 0 {
		modifier += 0.5
	}

	// Major organizations
	if organizations >= 2 {
		modifier += 0.3
	}

	return math.Min(1.5, modifier)
}
