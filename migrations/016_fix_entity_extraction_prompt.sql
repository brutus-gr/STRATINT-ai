-- Migration 016: Simplify entity extraction prompt to extract more entities
-- The previous prompt was too restrictive and filtered out too many valid entities

UPDATE openai_config
SET
  entity_extraction_prompt = 'Extract named entities that are relevant to understanding this intelligence content.

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

For each entity, provide:
- type: One of the types above
- name: Entity name as it appears in text
- normalized_name: Standardized form (e.g., "U.S." -> "United States", "Putin" -> "Vladimir Putin")
- confidence: 0.7-1.0
- context: Brief quote showing how entity appears in text

EXAMPLES:

Text: "President Biden announced new sanctions on Iran, targeting Tehran''s nuclear program"
Extract: [
  {"type": "person", "name": "Biden", "normalized_name": "Joe Biden", "confidence": 0.9, "context": "President Biden announced"},
  {"type": "country", "name": "Iran", "normalized_name": "Iran", "confidence": 1.0, "context": "sanctions on Iran"},
  {"type": "city", "name": "Tehran", "normalized_name": "Tehran", "confidence": 1.0, "context": "targeting Tehran''s nuclear program"}
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
}',
  updated_at = NOW()
WHERE id IS NOT NULL;

-- Verify the update
SELECT
  'Entity extraction prompt updated successfully' as status,
  LEFT(entity_extraction_prompt, 100) as prompt_preview,
  updated_at
FROM openai_config;
