package models

import "time"

// ThresholdConfig holds auto-publish threshold configuration.
type ThresholdConfig struct {
	MinConfidence     float64   `json:"min_confidence"`
	MinMagnitude      float64   `json:"min_magnitude"`
	MaxSourceAgeHours int       `json:"max_source_age_hours"`
	UpdatedAt         time.Time `json:"updated_at"`
}
