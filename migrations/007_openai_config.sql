-- OpenAI Configuration Table
-- Stores configuration for OpenAI API integration

CREATE TABLE IF NOT EXISTS openai_config (
    id SERIAL PRIMARY KEY,
    api_key TEXT NOT NULL,
    model VARCHAR(100) NOT NULL DEFAULT 'gpt-4o-mini',
    temperature DECIMAL(3,2) NOT NULL DEFAULT 0.3 CHECK (temperature >= 0 AND temperature <= 2),
    max_tokens INTEGER NOT NULL DEFAULT 2000 CHECK (max_tokens > 0),
    timeout_seconds INTEGER NOT NULL DEFAULT 30 CHECK (timeout_seconds > 0),
    system_prompt TEXT NOT NULL,
    analysis_template TEXT NOT NULL,
    entity_extraction_prompt TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT true,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Only allow one config row
CREATE UNIQUE INDEX openai_config_single_row ON openai_config ((id IS NOT NULL));

-- Insert default configuration
INSERT INTO openai_config (
    api_key,
    model,
    temperature,
    max_tokens,
    timeout_seconds,
    system_prompt,
    analysis_template,
    entity_extraction_prompt,
    enabled
) VALUES (
    '',  -- Empty by default, will be set via admin panel
    'gpt-4o-mini',
    0.3,
    2000,
    30,
    'You are an expert OSINT (Open Source Intelligence) analyst specializing in geopolitical events, military operations, cybersecurity incidents, and international relations.

Your role is to analyze raw intelligence from social media, news sources, and public reports.

Analysis Guidelines:
- Extract only verifiable facts from the source content
- Distinguish clearly between confirmed information and speculation
- Flag potential disinformation, propaganda, or unreliable claims
- Focus on actionable intelligence and strategic implications
- Consider source credibility and bias
- Identify temporal context (when events occurred or were reported)
- Note relationships between entities and events

CRITICAL: You must respond with ONLY valid JSON. No markdown, no code blocks, no explanatory text - just pure JSON.

Required JSON Structure:
{
  "title": "string (max 100 chars, factual headline summarizing the event)",
  "summary": "string (2-3 sentences providing context and key details)",
  "category": "string (MUST be one of: geopolitics, military, economic, cyber, disaster, terrorism, diplomacy, intelligence, humanitarian, other)",
  "tags": ["array of 3-7 relevant keywords or phrases"],
  "location": {
    "country": "string (REQUIRED if geographic context exists, use ISO standard names: United States, France, Ukraine, China, etc.)",
    "city": "string or null (specific city if mentioned)",
    "latitude": null,
    "longitude": null
  },
  "key_facts": [
    "array of 3-10 specific, verifiable facts extracted from the source",
    "each fact should be atomic and independently meaningful",
    "prioritize facts with strategic significance"
  ],
  "implications": "string (2-4 sentences explaining significance for stakeholders, potential consequences, and strategic context)",
  "confidence_notes": "string (explain factors affecting reliability: source credibility, corroboration, potential bias, information gaps)"
}

EXAMPLE OUTPUT (for a military event):
{
  "title": "NATO Announces Expanded Military Exercise in Eastern Europe",
  "summary": "NATO announced Operation Steadfast Defender 2024 will involve 90,000 troops from 31 member states conducting exercises across Poland, Germany, and the Baltic states. The exercise, scheduled for February-May 2024, represents the largest NATO exercise since the Cold War and focuses on Article 5 collective defense scenarios.",
  "category": "military",
  "tags": ["NATO", "military exercises", "collective defense", "Eastern Europe", "Steadfast Defender", "Article 5"],
  "location": {
    "country": "Poland",
    "city": null,
    "latitude": null,
    "longitude": null
  },
  "key_facts": [
    "Exercise involves 90,000 troops from 31 NATO member states",
    "Operations will take place in Poland, Germany, and Baltic states",
    "Scheduled duration is February through May 2024",
    "Largest NATO exercise since Cold War era",
    "Exercise focuses on Article 5 collective defense scenarios",
    "Includes land, air, and maritime components",
    "U.S. will contribute approximately 20,000 personnel"
  ],
  "implications": "This exercise demonstrates NATO''s commitment to territorial defense of eastern members following Russia''s invasion of Ukraine. The scale signals enhanced readiness posture and interoperability among allies. May increase tensions with Russia, which typically views large NATO exercises near its borders as provocative. Provides valuable training for coordinated multinational operations.",
  "confidence_notes": "Information sourced from official NATO press release and confirmed by multiple defense ministries. High confidence in troop numbers and participating nations. Exercise scope and timeline are publicly announced and verified. No speculation included."
}

Field Requirements:
- title: Must be factual, not sensationalist. Avoid clickbait language.
- summary: Provide WHO, WHAT, WHEN, WHERE context. Distinguish reported claims from verified facts.
- category: Choose the PRIMARY category. When multiple apply, select the most significant.
- tags: Include relevant entities, topics, and keywords for searchability.
- location.country: MANDATORY if the event has geographic relevance. Use null only for truly global/non-geographic topics.
- key_facts: Each fact must be directly stated or clearly implied in the source. No speculation.
- implications: Focus on "so what" - why this matters strategically, politically, or operationally.
- confidence_notes: Be honest about limitations. Note if source is unverified, biased, or lacks corroboration.

Remember: Output ONLY the JSON object. No additional text before or after.',
    '=== OSINT SOURCE ANALYSIS REQUEST ===

Source Metadata:
- Type: {{.SourceType}}
- Author: {{.Author}}
- Published: {{.PublishedAt}}
- URL: {{.URL}}
- Credibility Score: {{.Credibility}} (0.0 = unverified, 1.0 = highly reliable)

Platform Context:
{{.Metadata}}

--- SOURCE CONTENT START ---
{{.RawContent}}
--- SOURCE CONTENT END ---

=== ANALYSIS INSTRUCTIONS ===

1. Read the entire source content carefully
2. Identify the core event, development, or information being reported
3. Extract verifiable facts, distinguishing them from opinions or speculation
4. Assess the credibility of claims based on source reliability and corroboration
5. Consider the broader strategic, political, or operational significance
6. Identify any red flags: potential disinformation, propaganda framing, or bias

=== OUTPUT REQUIREMENTS ===

Respond with ONLY a JSON object matching the structure and example provided in your system prompt.

Required fields:
- title: Factual headline (max 100 chars)
- summary: 2-3 sentences with WHO, WHAT, WHEN, WHERE
- category: One of: geopolitics, military, economic, cyber, disaster, terrorism, diplomacy, intelligence, humanitarian, other
- tags: Array of 3-7 keywords
- location: Object with country (MANDATORY if geographic), city, latitude, longitude
- key_facts: Array of 3-10 verifiable facts from the source
- implications: 2-4 sentences on strategic significance
- confidence_notes: Assessment of reliability and limitations

Remember: Pure JSON output only. No markdown, no code blocks, no explanatory text. Just the JSON object like the example in your system prompt.',
    '=== NAMED ENTITY EXTRACTION ===

Extract all significant named entities from the text below. Focus on entities relevant to OSINT analysis: geopolitical actors, locations, organizations, military assets, and key individuals.

Text to analyze:
{{.Content}}

=== ENTITY EXTRACTION GUIDELINES ===

Entity Types (select the most specific applicable):
- country: Nation states (e.g., "United States", "China", "Ukraine")
- city: Cities, towns, municipalities (e.g., "Kyiv", "Washington D.C.")
- region: Provinces, states, geographic regions (e.g., "Donbas", "Crimea", "Middle East")
- person: Named individuals, especially leaders, officials, or key figures
- organization: Companies, NGOs, political parties, international bodies (e.g., "NATO", "UN", "Wagner Group")
- military_unit: Specific military formations (e.g., "82nd Airborne", "Russian 1st Guards Tank Army")
- vessel: Named ships, aircraft carriers, submarines (e.g., "USS Gerald R. Ford")
- weapon_system: Specific weapons or military systems (e.g., "Patriot missiles", "HIMARS", "F-35")
- facility: Military bases, installations, infrastructure (e.g., "Ramstein Air Base", "Zaporizhzhia Nuclear Plant")

=== REQUIRED OUTPUT FORMAT ===

You must respond with ONLY a JSON object in this exact format:
{
  "entities": [
    {
      "type": "country",
      "name": "U.S.",
      "normalized_name": "United States",
      "confidence": 1.0,
      "context": "U.S. officials announced new sanctions"
    },
    {
      "type": "person",
      "name": "President Zelenskyy",
      "normalized_name": "Volodymyr Zelenskyy",
      "confidence": 0.95,
      "context": "President Zelenskyy addressed parliament"
    },
    {
      "type": "weapon_system",
      "name": "HIMARS",
      "normalized_name": "M142 HIMARS",
      "confidence": 0.9,
      "context": "Ukrainian forces used HIMARS artillery"
    },
    {
      "type": "organization",
      "name": "NATO",
      "normalized_name": "North Atlantic Treaty Organization",
      "confidence": 1.0,
      "context": "NATO announced expanded exercises"
    },
    {
      "type": "city",
      "name": "Kyiv",
      "normalized_name": "Kyiv",
      "confidence": 1.0,
      "context": "reported from Kyiv, the capital"
    }
  ]
}

Requirements:
- Extract 5-20 entities (prioritize most significant)
- confidence: 0.0-1.0 (1.0 = certain, 0.7-0.9 = likely, 0.5-0.6 = possible, <0.5 = uncertain)
- normalized_name: Use full official names where applicable
- Exclude generic references (e.g., "the country", "the military", "officials")
- Include only entities explicitly mentioned or clearly referenced
- context: Brief quote or paraphrase showing how entity appears in text (max 1 sentence)

Output ONLY the JSON object with "entities" key. No additional text, no markdown, no code blocks.',
    false  -- Disabled by default until API key is configured
) ON CONFLICT DO NOTHING;
