package enrichment

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/STRATINT/stratint/internal/models"
	openai "github.com/sashabaranov/go-openai"
)

// EntityExtractor extracts named entities from text using AI.
type EntityExtractor struct {
	normalizer *EntityNormalizer
}

// NewEntityExtractor creates a new entity extractor.
func NewEntityExtractor() *EntityExtractor {
	return &EntityExtractor{
		normalizer: NewEntityNormalizer(),
	}
}

// Extract pulls named entities from content using OpenAI.
func (e *EntityExtractor) Extract(ctx context.Context, content string, client *openai.Client, config OpenAIConfig, entityPrompt string) ([]models.Entity, error) {
	// Use the provided entity extraction prompt (should already have content substituted)
	if entityPrompt == "" {
		return nil, fmt.Errorf("entity extraction prompt is empty")
	}

	resp, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:               config.Model,
		MaxCompletionTokens: 2000, // Increased to handle larger entity lists
		ResponseFormat: &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONObject,
		},
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: "You are a precise entity extraction system. You must respond with ONLY valid JSON. Wrap all entities in an object with an 'entities' key. Structure: {\"entities\": [{\"type\": \"...\", \"name\": \"...\", \"normalized_name\": \"...\", \"confidence\": 0.0, \"context\": \"...\"}]}",
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: entityPrompt,
			},
		},
	})

	if err != nil {
		return nil, fmt.Errorf("entity extraction failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return []models.Entity{}, nil
	}

	rawResponse := resp.Choices[0].Message.Content

	// Parse JSON response
	entities, err := e.parseEntityResponse(rawResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to parse entities: %w", err)
	}

	// Normalize entities
	for i := range entities {
		e.normalizer.Normalize(&entities[i])
	}

	return entities, nil
}

// parseEntityResponse converts JSON response to entity structs.
func (e *EntityExtractor) parseEntityResponse(response string) ([]models.Entity, error) {
	// Simple struct for JSON parsing
	type rawEntity struct {
		Type           string  `json:"type"`
		Name           string  `json:"name"`
		NormalizedName string  `json:"normalized_name"`
		Confidence     float64 `json:"confidence"`
		Context        string  `json:"context"`
	}

	// Try to parse as wrapped object first: {"entities": [...]}
	type wrappedEntities struct {
		Entities []rawEntity `json:"entities"`
	}
	var wrapped wrappedEntities
	if err := json.Unmarshal([]byte(response), &wrapped); err == nil {
		// Success with wrapped format (even if empty)
		entities := make([]models.Entity, 0, len(wrapped.Entities))
		for _, r := range wrapped.Entities {
			entities = append(entities, models.Entity{
				ID:             generateEntityID(),
				Type:           parseEntityType(r.Type),
				Name:           r.Name,
				NormalizedName: r.NormalizedName,
				Confidence:     r.Confidence,
				Context:        r.Context,
			})
		}
		return entities, nil
	}

	// Try direct array format: [...]
	var raw []rawEntity
	if err := json.Unmarshal([]byte(response), &raw); err != nil {
		// Try to extract JSON from text if wrapped in markdown or other text
		start := findJSONStart(response)
		end := findJSONEnd(response)
		if start >= 0 && end > start {
			jsonStr := response[start : end+1]
			if err := json.Unmarshal([]byte(jsonStr), &raw); err != nil {
				return nil, fmt.Errorf("json parse error: %w", err)
			}
		} else {
			return nil, fmt.Errorf("json parse error: %w", err)
		}
	}

	// Convert to models.Entity
	entities := make([]models.Entity, 0, len(raw))
	for _, r := range raw {
		entity := models.Entity{
			ID:             generateEntityID(),
			Type:           parseEntityType(r.Type),
			Name:           r.Name,
			NormalizedName: r.NormalizedName,
			Confidence:     r.Confidence,
			Context:        r.Context,
		}
		entities = append(entities, entity)
	}

	return entities, nil
}

// findJSONStart looks for the start of a JSON array in text.
func findJSONStart(text string) int {
	for i, ch := range text {
		if ch == '[' || ch == '{' {
			return i
		}
	}
	return -1
}

// findJSONEnd looks for the end of a JSON array in text.
func findJSONEnd(text string) int {
	depth := 0
	inString := false
	escape := false

	for i, ch := range text {
		if escape {
			escape = false
			continue
		}

		if ch == '\\' {
			escape = true
			continue
		}

		if ch == '"' {
			inString = !inString
			continue
		}

		if inString {
			continue
		}

		if ch == '[' || ch == '{' {
			depth++
		} else if ch == ']' || ch == '}' {
			depth--
			if depth == 0 {
				return i
			}
		}
	}

	return -1
}

// parseEntityType converts string to EntityType enum.
func parseEntityType(typeStr string) models.EntityType {
	switch typeStr {
	case "country":
		return models.EntityTypeCountry
	case "city":
		return models.EntityTypeCity
	case "region":
		return models.EntityTypeRegion
	case "person":
		return models.EntityTypePerson
	case "organization":
		return models.EntityTypeOrganization
	case "military_unit":
		return models.EntityTypeMilitaryUnit
	case "vessel":
		return models.EntityTypeVessel
	case "weapon_system":
		return models.EntityTypeWeaponSystem
	case "facility":
		return models.EntityTypeFacility
	case "event":
		return models.EntityTypeEvent
	default:
		return models.EntityTypeOther
	}
}

// generateEntityID creates a unique entity identifier.
func generateEntityID() string {
	return fmt.Sprintf("ent-%d", generateTimestamp())
}

// EntityNormalizer standardizes entity names using reference data.
type EntityNormalizer struct {
	countryAliases map[string]string
	cityAliases    map[string]string
}

// NewEntityNormalizer creates a new entity normalizer with reference data.
func NewEntityNormalizer() *EntityNormalizer {
	return &EntityNormalizer{
		countryAliases: buildCountryAliases(),
		cityAliases:    buildCityAliases(),
	}
}

// Normalize standardizes an entity's name and adds metadata.
func (n *EntityNormalizer) Normalize(entity *models.Entity) {
	switch entity.Type {
	case models.EntityTypeCountry:
		if normalized, ok := n.countryAliases[entity.Name]; ok {
			entity.NormalizedName = normalized
			// Add country code if available
			if code := getCountryCode(normalized); code != "" {
				entity.Attributes.CountryCode = code
			}
		}

	case models.EntityTypeCity:
		if normalized, ok := n.cityAliases[entity.Name]; ok {
			entity.NormalizedName = normalized
		}
	}

	// If no normalization happened, use the original name
	if entity.NormalizedName == "" {
		entity.NormalizedName = entity.Name
	}
}

// buildCountryAliases returns common country name variations.
func buildCountryAliases() map[string]string {
	return map[string]string{
		"USA":              "United States",
		"U.S.":             "United States",
		"U.S.A.":           "United States",
		"US":               "United States",
		"America":          "United States",
		"UK":               "United Kingdom",
		"U.K.":             "United Kingdom",
		"Britain":          "United Kingdom",
		"Great Britain":    "United Kingdom",
		"Russia":           "Russian Federation",
		"USSR":             "Soviet Union",
		"Soviet Union":     "Soviet Union",
		"PRC":              "China",
		"P.R.C.":           "China",
		"Peoples Republic": "China",
		"ROK":              "South Korea",
		"DPRK":             "North Korea",
		"North Korea":      "North Korea",
		"South Korea":      "South Korea",
		"UAE":              "United Arab Emirates",
		"U.A.E.":           "United Arab Emirates",
		"KSA":              "Saudi Arabia",
		"Deutschland":      "Germany",
		"BRD":              "Germany",
	}
}

// buildCityAliases returns common city name variations.
func buildCityAliases() map[string]string {
	return map[string]string{
		"NYC":            "New York City",
		"New York":       "New York City",
		"LA":             "Los Angeles",
		"SF":             "San Francisco",
		"DC":             "Washington",
		"Kyiv":           "Kyiv",
		"Kiev":           "Kyiv", // Use Ukrainian spelling
		"Peking":         "Beijing",
		"Bombay":         "Mumbai",
		"Leningrad":      "Saint Petersburg",
		"Constantinople": "Istanbul",
	}
}

// getCountryCode returns ISO 3166-1 alpha-2 code for a country.
func getCountryCode(country string) string {
	codes := map[string]string{
		"United States":        "US",
		"United Kingdom":       "GB",
		"Russian Federation":   "RU",
		"China":                "CN",
		"Germany":              "DE",
		"France":               "FR",
		"Japan":                "JP",
		"South Korea":          "KR",
		"North Korea":          "KP",
		"Ukraine":              "UA",
		"Israel":               "IL",
		"Iran":                 "IR",
		"Saudi Arabia":         "SA",
		"United Arab Emirates": "AE",
		"Turkey":               "TR",
		"India":                "IN",
		"Pakistan":             "PK",
		"Australia":            "AU",
		"Canada":               "CA",
		"Mexico":               "MX",
		"Brazil":               "BR",
	}

	return codes[country]
}
