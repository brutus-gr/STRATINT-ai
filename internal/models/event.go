package models

import (
	"time"
)

// Event represents a processed OSINT intelligence event with metadata, sources, and extracted entities.
type Event struct {
	ID         string      `json:"id"`
	Timestamp  time.Time   `json:"timestamp"`
	Title      string      `json:"title"`
	Summary    string      `json:"summary"`
	RawContent string      `json:"raw_content"`
	Magnitude  float64     `json:"magnitude"` // 0-10 scale for event importance/severity
	Confidence Confidence  `json:"confidence"`
	Category   Category    `json:"category"`
	Entities   []Entity    `json:"entities"`
	Sources    []Source    `json:"sources"`
	Tags       []string    `json:"tags"`
	Location   *Location   `json:"location,omitempty"`
	CreatedAt  time.Time   `json:"created_at"`
	UpdatedAt  time.Time   `json:"updated_at"`
	Status     EventStatus `json:"status"`
}

// EventStatus represents the lifecycle state of an event.
type EventStatus string

const (
	EventStatusPending   EventStatus = "pending"   // Raw data ingested, not yet processed
	EventStatusEnriched  EventStatus = "enriched"  // NLP processing completed
	EventStatusPublished EventStatus = "published" // Available via API
	EventStatusArchived  EventStatus = "archived"  // Moved to cold storage
	EventStatusRejected  EventStatus = "rejected"  // Failed validation or moderation
)

// Category represents the primary classification of an OSINT event.
type Category string

const (
	CategoryGeopolitics  Category = "geopolitics"
	CategoryMilitary     Category = "military"
	CategoryEconomic     Category = "economic"
	CategoryCyber        Category = "cyber"
	CategoryDisaster     Category = "disaster"
	CategoryTerrorism    Category = "terrorism"
	CategoryDiplomacy    Category = "diplomacy"
	CategoryIntelligence Category = "intelligence"
	CategoryHumanitarian Category = "humanitarian"
	CategoryOther        Category = "other"
)

// Location represents geographic coordinates and place information.
type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Country   string  `json:"country,omitempty"`
	City      string  `json:"city,omitempty"`
	Region    string  `json:"region,omitempty"`
}

// Confidence represents the reliability assessment of an event.
type Confidence struct {
	Score       float64         `json:"score"`        // 0-1 scale
	Level       ConfidenceLevel `json:"level"`        // Human-readable level
	Reasoning   string          `json:"reasoning"`    // Explanation for the score
	SourceCount int             `json:"source_count"` // Number of corroborating sources
}

// ConfidenceLevel provides human-readable confidence assessment.
type ConfidenceLevel string

const (
	ConfidenceLow      ConfidenceLevel = "low"      // 0.0-0.3
	ConfidenceMedium   ConfidenceLevel = "medium"   // 0.3-0.6
	ConfidenceHigh     ConfidenceLevel = "high"     // 0.6-0.85
	ConfidenceVerified ConfidenceLevel = "verified" // 0.85-1.0
)

// DeriveLevel calculates the confidence level from a numeric score.
func (c *Confidence) DeriveLevel() ConfidenceLevel {
	switch {
	case c.Score >= 0.85:
		return ConfidenceVerified
	case c.Score >= 0.6:
		return ConfidenceHigh
	case c.Score >= 0.3:
		return ConfidenceMedium
	default:
		return ConfidenceLow
	}
}

// IsPublishable returns true if the event meets minimum quality thresholds for publication.
func (e *Event) IsPublishable() bool {
	return e.Confidence.Score >= 0.3 && e.Magnitude >= 1.0 && len(e.Sources) > 0
}
