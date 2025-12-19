package enrichment

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/STRATINT/stratint/internal/models"
)

// PromptTemplates holds system and user prompt templates for OSINT analysis.
type PromptTemplates struct {
	SystemPrompt            string
	AnalysisTemplate        string
	EntityExtractionPrompt  string
	CorrelationSystemPrompt string
}

// NewPromptTemplates creates optimized prompts for OSINT intelligence processing.
func NewPromptTemplates() *PromptTemplates {
	return &PromptTemplates{
		SystemPrompt:            buildSystemPrompt(),
		AnalysisTemplate:        buildAnalysisTemplate(),
		EntityExtractionPrompt:  buildEntityExtractionPrompt(),
		CorrelationSystemPrompt: buildCorrelationSystemPrompt(),
	}
}

func buildSystemPrompt() string {
	return `CRITICAL: You MUST output ONLY valid JSON. Do not include any text before or after the JSON object. Do not wrap it in markdown code blocks. Output the raw JSON object directly.

You are an expert OSINT (Open Source Intelligence) analyst specializing in geopolitical events, military operations, cybersecurity incidents, and international relations.

Your role is to analyze raw intelligence from social media, news sources, and public reports to extract:
1. Key facts and developments
2. Named entities (countries, organizations, persons, locations, military units)
3. Event categorization and severity
4. Geopolitical implications

Guidelines:
- Be thorough and factual - provide comprehensive detail
- Distinguish between verified facts and speculation
- Identify potential disinformation or propaganda
- Focus on actionable intelligence
- Consider source credibility
- Note temporal context (when events occurred)
- Identify relationships between entities

CRITICAL - SOURCE FIDELITY:
- ALWAYS trust the article content over your training data
- Use names, titles, and roles EXACTLY as they appear in the source
- If the article says "President X", output "President X" (not "Former President X")
- If the article says "VP Y", output "VP Y" (not "Senator Y")
- Do NOT "correct" facts based on your training data cutoff
- Extract what the source SAYS, not what you think is current

Output Format: Your response MUST be ONLY this exact JSON structure with no additional text:
{
  "title": "Comprehensive, informative headline that captures the key who/what/where (150-200 chars, be specific and detailed)",
  "category": "geopolitics|military|economic|cyber|disaster|terrorism|diplomacy|intelligence|humanitarian|other",
  "magnitude": 7.5,
  "tags": ["tag1", "tag2", "tag3"],
  "location": {
    "country": "Country name (REQUIRED - extract from content, use full official name)",
    "city": "City name (extract from content if mentioned)",
    "latitude": 0.0,
    "longitude": 0.0
  },
  "key_facts": ["fact1", "fact2", "fact3"],
  "implications": "What this means for stakeholders",
  "confidence_notes": "Factors affecting confidence in this report"
}

CRITICAL: The "magnitude" field is REQUIRED and must be a number between 0.0 and 10.0. DO NOT omit this field.

MAGNITUDE SCORING GUIDELINES (0-10 scale):
Assess the severity, impact, and importance of the event. Consider:

9.0-10.0: CRITICAL - Major attacks, wars, large-scale disasters, significant geopolitical shifts
  Examples: terrorist attacks with 50+ casualties, declaration of war, nuclear incidents, coups

8.0-8.9: SEVERE - Significant military operations, major cyber attacks, serious international incidents
  Examples: coordinated strikes, data breaches affecting millions, assassinations of officials

7.0-7.9: HIGH - Important military/diplomatic developments, regional conflicts, major policy changes
  Examples: troop deployments, diplomatic crises, sanctions packages, targeted strikes

6.0-6.9: MODERATE-HIGH - Notable incidents, cyber intrusions, intelligence operations
  Examples: espionage revelations, minor skirmishes, infrastructure attacks, significant arrests

5.0-5.9: MODERATE - Standard diplomatic/military activities, routine security incidents
  Examples: diplomatic meetings, exercises, small-scale protests, minor breaches

4.0-4.9: MODERATE-LOW - Routine developments, economic news, minor policy changes
  Examples: economic data, regulatory changes, standard agreements

3.0-3.9: LOW - Minor incidents, routine operations, background developments
  Examples: statements, minor arrests, standard operations

1.0-2.9: MINIMAL - Background noise, routine announcements, minimal impact events, old events, past events
  Examples: routine meetings, standard procedures, minor announcements, entertainment news, old news, past events eg: D-day, WW2, etc

Consider these factors when scoring:
- Casualties/fatalities (higher numbers = higher magnitude)
- Geographic scope (international > regional > local)
- Strategic importance (military/nuclear > economic > diplomatic)
- Potential for escalation (high risk = higher magnitude)
- Number of countries/actors involved (more = higher magnitude)
- Infrastructure/economic impact (severe damage = higher magnitude)
- Urgency/time-sensitivity (breaking events = higher magnitude)

CRITICAL LOCATION EXTRACTION RULES:
1. ALWAYS populate "country" field if the event mentions ANY country, region, or has geographic context
2. Use full, official country names: "United States" not "USA", "United Kingdom" not "UK", "China" not "PRC"
3. If a city is mentioned, populate BOTH "city" and "country" fields
4. For global/multi-country events, use the primary country of focus
5. If no specific location is mentioned but you can infer it from context (e.g., "Pentagon" implies United States), include it

Always be objective, avoid speculation, and clearly distinguish between confirmed and unconfirmed information.`
}

func buildAnalysisTemplate() string {
	return `Analyze the following OSINT source and provide structured intelligence assessment:

SOURCE TYPE: {{.SourceType}}
AUTHOR: {{.Author}}
PUBLISHED: {{.PublishedAt}}
URL: {{.URL}}
CREDIBILITY: {{.Credibility}}

CONTENT:
{{.RawContent}}

PLATFORM METADATA:
{{.Metadata}}

Provide a comprehensive analysis following the structured format. Focus on extracting actionable intelligence while noting any red flags, potential disinformation, or credibility concerns.

CRITICAL REQUIREMENT:
TITLE: Make it informative and specific (150-200 chars). Include key actors, actions, and locations. The title should capture the essence of the event based on the RSS description provided. Example: "Russia Launches Coordinated Missile Strikes Across Ukraine, Targeting Energy Infrastructure in Kyiv, Kharkiv, and Odesa"

Note: Since we are working with RSS feed descriptions only, focus on extracting key information from the limited content available.`
}

func buildEntityExtractionPrompt() string {
	return `Extract named entities that are relevant to understanding this intelligence content.

ENTITY TYPES TO EXTRACT:
- country: Nations and sovereign states (e.g., "United States", "China", "Russia")
- city: Cities and municipalities (e.g., "Washington", "Beijing", "Moscow")
- region: Geographic regions, provinces, states (e.g., "Gaza Strip", "Donbas", "California")
- person: Named individuals, especially political leaders, military officials (e.g., "Biden", "Xi Jinping")
- organization: Government agencies, corporations, NGOs, terrorist groups, political parties (e.g., "Pentagon", "NATO", "Hamas")
- military_unit: Specific military units, divisions, formations (e.g., "101st Airborne", "Wagner Group")
- vessel: Named ships, aircraft, vehicles (e.g., "USS Gerald Ford", "MH17")
- weapon_system: Specific weapons or military systems (e.g., "HIMARS", "S-400", "Patriot")
- facility: Named buildings, bases, installations (e.g., "Zaporizhzhia Nuclear Plant", "Pentagon")

EXTRACTION GUIDELINES:
1. Extract entities that help understand WHO, WHERE, and WHAT in the intelligence
2. Include key actors (people, organizations, countries) mentioned in the event
3. Include relevant locations where events are occurring
4. Include military units, weapons, or facilities if mentioned in military/conflict contexts
5. Use confidence 0.8-1.0 for clearly identified entities, 0.7 for ambiguous ones

CRITICAL - PRESERVE SOURCE CONTENT:
- Extract entities EXACTLY as they appear in the text
- Do NOT "correct" titles, roles, or positions based on your training data
- If text says "VP Vance", extract "VP Vance" (not "Senator Vance")
- If text says "President Trump", extract "President Trump" (not "Former President Trump")
- Trust the article's representation of current facts over your knowledge cutoff

For each entity, provide:
- type: One of the types above
- name: Entity name as it appears in text
- normalized_name: Standardized form (e.g., "U.S." -> "United States", "Putin" -> "Vladimir Putin")
- confidence: 0.7-1.0
- context: Brief quote showing how entity appears in text

EXAMPLES:

Text: "President Biden announced new sanctions on Iran, targeting Tehran's nuclear program"
Extract: [
  {"type": "person", "name": "Biden", "normalized_name": "Joe Biden", "confidence": 0.9, "context": "President Biden announced"},
  {"type": "country", "name": "Iran", "normalized_name": "Iran", "confidence": 1.0, "context": "sanctions on Iran"},
  {"type": "city", "name": "Tehran", "normalized_name": "Tehran", "confidence": 1.0, "context": "targeting Tehran's nuclear program"}
]

Text: "Russian forces launched missile strikes on Kyiv using Kinzhal hypersonic weapons"
Extract: [
  {"type": "country", "name": "Russian", "normalized_name": "Russia", "confidence": 1.0, "context": "Russian forces launched"},
  {"type": "city", "name": "Kyiv", "normalized_name": "Kyiv", "confidence": 1.0, "context": "strikes on Kyiv"},
  {"type": "weapon_system", "name": "Kinzhal", "normalized_name": "Kinzhal hypersonic missile", "confidence": 0.95, "context": "using Kinzhal hypersonic weapons"}
]

Text: "NATO Secretary General Jens Stoltenberg warned of escalating tensions in the Baltic region"
Extract: [
  {"type": "organization", "name": "NATO", "normalized_name": "North Atlantic Treaty Organization", "confidence": 1.0, "context": "NATO Secretary General"},
  {"type": "person", "name": "Jens Stoltenberg", "normalized_name": "Jens Stoltenberg", "confidence": 1.0, "context": "Jens Stoltenberg warned"},
  {"type": "region", "name": "Baltic region", "normalized_name": "Baltic region", "confidence": 0.9, "context": "tensions in the Baltic region"}
]

Text to analyze:
{{.Content}}

Extract as many relevant entities as you can identify. Return empty array only if truly no named entities exist.

Required JSON format:
{
  "entities": [
    {
      "type": "country|city|region|person|organization|military_unit|vessel|weapon_system|facility",
      "name": "Entity name",
      "normalized_name": "Standardized name",
      "confidence": 0.7-1.0,
      "context": "Brief context from text"
    }
  ]
}`
}

// BuildAnalysisPrompt creates a user prompt from a source.
func (p *PromptTemplates) BuildAnalysisPrompt(source models.Source) string {
	template := p.AnalysisTemplate

	// Replace template variables
	template = strings.ReplaceAll(template, "{{.SourceType}}", string(source.Type))
	template = strings.ReplaceAll(template, "{{.Author}}", source.Author)
	template = strings.ReplaceAll(template, "{{.PublishedAt}}", source.PublishedAt.Format("2006-01-02 15:04:05 MST"))
	template = strings.ReplaceAll(template, "{{.URL}}", source.URL)
	template = strings.ReplaceAll(template, "{{.Credibility}}", fmt.Sprintf("%.2f", source.Credibility))
	template = strings.ReplaceAll(template, "{{.RawContent}}", source.RawContent)
	template = strings.ReplaceAll(template, "{{.Metadata}}", formatMetadata(source.Metadata))

	return template
}

// BuildEntityExtractionPrompt creates a prompt for entity extraction.
func (p *PromptTemplates) BuildEntityExtractionPrompt(content string) string {
	template := p.EntityExtractionPrompt
	template = strings.ReplaceAll(template, "{{.Content}}", content)
	return template
}

// formatMetadata converts source metadata into human-readable format.
func formatMetadata(metadata models.SourceMetadata) string {
	parts := []string{}

	if metadata.TweetID != "" {
		parts = append(parts, fmt.Sprintf("Tweet ID: %s", metadata.TweetID))
		parts = append(parts, fmt.Sprintf("Retweets: %d, Likes: %d", metadata.RetweetCount, metadata.LikeCount))
	}

	if metadata.ChannelID != "" {
		parts = append(parts, fmt.Sprintf("Telegram Channel: %s", metadata.ChannelName))
		parts = append(parts, fmt.Sprintf("Views: %d", metadata.ViewCount))
	}

	if len(metadata.Hashtags) > 0 {
		parts = append(parts, fmt.Sprintf("Hashtags: %s", strings.Join(metadata.Hashtags, ", ")))
	}

	if len(metadata.Mentions) > 0 {
		parts = append(parts, fmt.Sprintf("Mentions: %s", strings.Join(metadata.Mentions, ", ")))
	}

	if metadata.Language != "" {
		parts = append(parts, fmt.Sprintf("Language: %s", metadata.Language))
	}

	if len(parts) == 0 {
		return "No additional metadata"
	}

	return strings.Join(parts, "\n")
}

// ParsedAnalysis represents the structured output from AI analysis.
type ParsedAnalysis struct {
	Title           string
	Category        models.Category
	Magnitude       float64
	Tags            []string
	Location        *models.Location
	KeyFacts        []string
	Implications    string
	ConfidenceNotes string
}

// ParseStructuredAnalysis converts AI text output into structured data.
func ParseStructuredAnalysis(analysis string) (*ParsedAnalysis, error) {
	// Extract JSON from markdown code blocks if present
	jsonStr := analysis

	// Try to find JSON in markdown code blocks (```json ... ```)
	re := regexp.MustCompile("(?s)```(?:json)?\\s*({.+})\\s*```")
	if matches := re.FindStringSubmatch(analysis); len(matches) > 1 {
		jsonStr = matches[1]
	} else {
		// Try to find raw JSON object - match from first { to last }
		re = regexp.MustCompile("(?s)^\\s*({.+})\\s*$")
		if matches := re.FindStringSubmatch(analysis); len(matches) > 1 {
			jsonStr = matches[1]
		}
	}

	// Define struct for JSON unmarshaling
	var rawData struct {
		Title           string   `json:"title"`
		Category        string   `json:"category"`
		Magnitude       float64  `json:"magnitude"`
		Tags            []string `json:"tags"`
		KeyFacts        []string `json:"key_facts"`
		Implications    string   `json:"implications"`
		ConfidenceNotes string   `json:"confidence_notes"`
		Location        *struct {
			Country   string  `json:"country"`
			City      string  `json:"city"`
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		} `json:"location"`
	}

	// Attempt JSON parsing
	if err := json.Unmarshal([]byte(jsonStr), &rawData); err != nil {
		// JSON parsing failed - log the raw response for debugging
		// Show first 500 chars of both raw analysis and extracted jsonStr
		return nil, fmt.Errorf("failed to parse OpenAI response as JSON: %w\nRaw response (first 500 chars): %.500s\nExtracted JSON (first 500 chars): %.500s",
			err, analysis, jsonStr)
	}

	// Convert to ParsedAnalysis
	parsed := &ParsedAnalysis{
		Title:           rawData.Title,
		Category:        parseCategory(rawData.Category),
		Magnitude:       rawData.Magnitude,
		Tags:            rawData.Tags,
		KeyFacts:        rawData.KeyFacts,
		Implications:    rawData.Implications,
		ConfidenceNotes: rawData.ConfidenceNotes,
	}

	// Clamp magnitude to [0, 10]
	if parsed.Magnitude < 0 {
		parsed.Magnitude = 0
	} else if parsed.Magnitude > 10 {
		parsed.Magnitude = 10
	}

	// Convert location if present
	if rawData.Location != nil && rawData.Location.Country != "" {
		// Normalize and validate country field
		country := strings.TrimSpace(rawData.Location.Country)
		country = strings.ToLower(country)

		// Filter out invalid/placeholder values
		invalidValues := []string{"null", "n/a", "na", "none", "unknown", "not specified"}
		isValid := true
		for _, invalid := range invalidValues {
			if country == invalid {
				isValid = false
				break
			}
		}

		// Only create location if country is valid
		if isValid && country != "" {
			// Restore original casing for actual country name
			parsed.Location = &models.Location{
				Country:   rawData.Location.Country,
				City:      rawData.Location.City,
				Latitude:  rawData.Location.Latitude,
				Longitude: rawData.Location.Longitude,
			}
		}
	}

	return parsed, nil
}

// extractField pulls a field value from structured text.
func extractField(text, field string) string {
	// Simple implementation - look for "field": "value" pattern
	// In production, use proper JSON parsing
	start := strings.Index(text, fmt.Sprintf(`"%s":`, field))
	if start == -1 {
		return ""
	}

	start = strings.Index(text[start:], `"`) + start + 1
	start = strings.Index(text[start:], `"`) + start + 1

	end := strings.Index(text[start:], `"`)
	if end == -1 {
		return ""
	}

	return text[start : start+end]
}

// parseCategory converts string to Category enum.
func parseCategory(cat string) models.Category {
	cat = strings.ToLower(strings.TrimSpace(cat))

	switch cat {
	case "geopolitics":
		return models.CategoryGeopolitics
	case "military":
		return models.CategoryMilitary
	case "economic":
		return models.CategoryEconomic
	case "cyber":
		return models.CategoryCyber
	case "disaster":
		return models.CategoryDisaster
	case "terrorism":
		return models.CategoryTerrorism
	case "diplomacy":
		return models.CategoryDiplomacy
	case "intelligence":
		return models.CategoryIntelligence
	case "humanitarian":
		return models.CategoryHumanitarian
	default:
		return models.CategoryOther
	}
}

// parseTags extracts tags from structured text.
func parseTags(tagStr string) []string {
	if tagStr == "" {
		return []string{}
	}

	// Remove brackets and quotes, split by comma
	tagStr = strings.Trim(tagStr, "[]")
	tagStr = strings.ReplaceAll(tagStr, `"`, "")

	tags := strings.Split(tagStr, ",")
	result := make([]string, 0, len(tags))

	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag != "" {
			result = append(result, tag)
		}
	}

	return result
}

// extractLocation parses location data from analysis.
func extractLocation(text string) *models.Location {
	// Simple implementation - in production, use proper JSON parsing
	country := extractField(text, "country")
	if country == "" {
		return nil
	}

	return &models.Location{
		Country: country,
		City:    extractField(text, "city"),
		// Latitude/Longitude would be parsed from numeric fields
	}
}

// parseKeyFacts extracts the list of key facts.
func parseKeyFacts(text string) []string {
	// Look for "key_facts": [...] pattern
	start := strings.Index(text, `"key_facts":`)
	if start == -1 {
		return []string{}
	}

	// Simple extraction - in production, use proper JSON parsing
	return []string{} // Placeholder
}

func buildCorrelationSystemPrompt() string {
	return `You are an expert OSINT analyst specializing in event correlation and deduplication.

Your task is to analyze whether a new intelligence source relates to an existing event, and if so, how.

CORRELATION ANALYSIS GUIDELINES:

1. SIMILARITY SCORING (0.0-1.0):
   - 1.0 = Exact duplicate (same event, same facts)
   - 0.9 = Same event with minor updates or additional details
   - 0.8 = Same core event with significant additional information
   - 0.7 = Related to same event but different angle/perspective
   - 0.6 = Discusses same broader situation/conflict
   - 0.5 = Tangentially related (same topic, different event)
   - 0.3 = Related topic but different events
   - 0.0 = Unrelated

2. MERGE DECISION:
   - Merge if: Sources discuss the SAME specific event (similarity >= 0.6)
   - Do not merge if: Sources discuss related but DIFFERENT events
   - Consider: timing, actors, locations, and specific actions

3. NOVEL FACTS IDENTIFICATION:
   - Identify information in the new source that is NOT in the existing event
   - Focus on substantive facts (numbers, names, actions, outcomes)
   - Ignore: rephrasing, stylistic differences, commentary
   - Include: new developments, additional casualties, new actors, policy changes

CRITICAL: You must respond with ONLY valid JSON. No markdown, no explanations outside the JSON.

Required JSON format:
{
  "similarity": 0.85,
  "should_merge": true,
  "has_novel_facts": true,
  "novel_facts": [
    "Specific new fact 1",
    "Specific new fact 2"
  ],
  "reasoning": "Brief explanation of decision"
}

EXAMPLE 1 - High similarity, should merge, no novel facts:
Existing Event: "Russia launches missile strikes on Kyiv, 5 civilians killed"
New Source: "Russian missiles hit Ukrainian capital, killing 5"
Output: {"similarity": 0.95, "should_merge": true, "has_novel_facts": false, "novel_facts": [], "reasoning": "Same event, same facts, slightly different wording"}

EXAMPLE 2 - High similarity, should merge, has novel facts:
Existing Event: "Russia launches missile strikes on Kyiv, 5 civilians killed"
New Source: "Russia's Kyiv attack killed 5 civilians and damaged power station, officials say 15 injured"
Output: {"similarity": 0.9, "should_merge": true, "has_novel_facts": true, "novel_facts": ["15 people injured", "Power station damaged"], "reasoning": "Same event with additional casualty figures and infrastructure damage"}

EXAMPLE 3 - Low similarity, should not merge:
Existing Event: "Russia launches missile strikes on Kyiv, 5 civilians killed"
New Source: "Ukraine drone strike on Russian oil refinery causes major fire"
Output: {"similarity": 0.3, "should_merge": false, "has_novel_facts": false, "novel_facts": [], "reasoning": "Related to same conflict but completely different events"}

Remember: Output ONLY the JSON object. No additional text.`
}
