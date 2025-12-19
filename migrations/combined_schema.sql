-- Initial database schema for OSINTMCP
-- PostgreSQL 15 with PostGIS extension

-- Enable extensions
CREATE EXTENSION IF NOT EXISTS postgis;
CREATE EXTENSION IF NOT EXISTS pg_trgm; -- For fast text search

-- Events table
CREATE TABLE events (
    id VARCHAR(64) PRIMARY KEY,
    timestamp TIMESTAMPTZ NOT NULL,
    title TEXT NOT NULL,
    summary TEXT NOT NULL,
    raw_content TEXT,
    magnitude DECIMAL(3,1) CHECK (magnitude >= 0 AND magnitude <= 10),
    confidence JSONB NOT NULL,
    category VARCHAR(32) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    tags TEXT[] DEFAULT '{}',
    location GEOGRAPHY(POINT, 4326), -- WGS84 coordinate system
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Sources table
CREATE TABLE sources (
    id VARCHAR(64) PRIMARY KEY,
    type VARCHAR(32) NOT NULL,
    url TEXT,
    author VARCHAR(255),
    published_at TIMESTAMPTZ NOT NULL,
    retrieved_at TIMESTAMPTZ NOT NULL,
    raw_content TEXT NOT NULL,
    content_hash VARCHAR(64) NOT NULL,
    credibility DECIMAL(3,2) CHECK (credibility >= 0 AND credibility <= 1),
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Entities table
CREATE TABLE entities (
    id VARCHAR(64) PRIMARY KEY,
    type VARCHAR(32) NOT NULL,
    name VARCHAR(255) NOT NULL,
    normalized_name VARCHAR(255) NOT NULL,
    confidence DECIMAL(3,2) CHECK (confidence >= 0 AND confidence <= 1),
    attributes JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Event-Source junction table (many-to-many)
CREATE TABLE event_sources (
    event_id VARCHAR(64) NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    source_id VARCHAR(64) NOT NULL REFERENCES sources(id) ON DELETE CASCADE,
    PRIMARY KEY (event_id, source_id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Event-Entity junction table (many-to-many)
CREATE TABLE event_entities (
    event_id VARCHAR(64) NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    entity_id VARCHAR(64) NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
    PRIMARY KEY (event_id, entity_id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for query performance

-- Events indexes
CREATE INDEX idx_events_timestamp ON events(timestamp DESC);
CREATE INDEX idx_events_magnitude ON events(magnitude DESC);
CREATE INDEX idx_events_confidence ON events(((confidence->>'score')::NUMERIC) DESC);
CREATE INDEX idx_events_category ON events(category);
CREATE INDEX idx_events_status ON events(status);
CREATE INDEX idx_events_created_at ON events(created_at DESC);
CREATE INDEX idx_events_updated_at ON events(updated_at DESC);
CREATE INDEX idx_events_tags ON events USING GIN(tags);
CREATE INDEX idx_events_composite_status_mag ON events(status, magnitude DESC, timestamp DESC);
CREATE INDEX idx_events_composite_category_mag ON events(category, magnitude DESC, timestamp DESC);

-- Geospatial index
CREATE INDEX idx_events_location ON events USING GIST(location);

-- Full-text search index
CREATE INDEX idx_events_search ON events USING GIN(
    to_tsvector('english', title || ' ' || COALESCE(summary, ''))
);

-- Sources indexes
CREATE INDEX idx_sources_type ON sources(type);
CREATE INDEX idx_sources_published_at ON sources(published_at DESC);
CREATE INDEX idx_sources_retrieved_at ON sources(retrieved_at DESC);
CREATE INDEX idx_sources_content_hash ON sources(content_hash);
CREATE INDEX idx_sources_created_at ON sources(created_at DESC);

-- Entities indexes
CREATE INDEX idx_entities_type ON entities(type);
CREATE INDEX idx_entities_name ON entities(name);
CREATE INDEX idx_entities_normalized_name ON entities(normalized_name);
CREATE INDEX idx_entities_name_trgm ON entities USING GIN(normalized_name gin_trgm_ops);

-- Junction table indexes
CREATE INDEX idx_event_sources_source_id ON event_sources(source_id);
CREATE INDEX idx_event_entities_entity_id ON event_entities(entity_id);

-- Update timestamp trigger
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_events_updated_at BEFORE UPDATE ON events
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Comments for documentation
COMMENT ON TABLE events IS 'Processed OSINT events with AI enrichment';
COMMENT ON TABLE sources IS 'Raw source data from various platforms';
COMMENT ON TABLE entities IS 'Named entities extracted from events';
COMMENT ON COLUMN events.confidence IS 'Multi-factor confidence scoring (JSON with score, level, reasoning)';
COMMENT ON COLUMN events.magnitude IS 'Event severity/importance on 0-10 scale';
COMMENT ON COLUMN events.location IS 'Geographic location (PostGIS POINT)';
COMMENT ON COLUMN sources.content_hash IS 'SHA-256 hash for deduplication';
-- Migration 002: Add tracked social media accounts
-- This allows admins to specify which accounts to monitor

CREATE TABLE tracked_accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    platform VARCHAR(32) NOT NULL, -- 'twitter', 'telegram', 'rss', etc.
    account_identifier VARCHAR(255) NOT NULL, -- username, handle, or channel ID
    display_name VARCHAR(255),
    enabled BOOLEAN NOT NULL DEFAULT true,
    last_fetched_id VARCHAR(255), -- Last tweet/post ID fetched (for pagination)
    last_fetched_at TIMESTAMP WITH TIME ZONE,
    fetch_interval_minutes INT NOT NULL DEFAULT 5, -- How often to check for new posts
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    UNIQUE(platform, account_identifier)
);

-- Index for efficient queries
CREATE INDEX idx_tracked_accounts_platform_enabled ON tracked_accounts(platform, enabled);
CREATE INDEX idx_tracked_accounts_last_fetched ON tracked_accounts(last_fetched_at) WHERE enabled = true;

-- Add comments
COMMENT ON TABLE tracked_accounts IS 'Social media accounts being monitored for OSINT';
COMMENT ON COLUMN tracked_accounts.account_identifier IS 'Platform-specific identifier (e.g., @elonmusk for Twitter)';
COMMENT ON COLUMN tracked_accounts.last_fetched_id IS 'Used for pagination to avoid re-fetching old posts';
-- Migration: Add title and author_id to sources table
-- These fields were present in the Go model but missing from the database schema

BEGIN;

-- Add title column for storing article/post titles from RSS and other sources
ALTER TABLE sources
ADD COLUMN IF NOT EXISTS title VARCHAR(512) DEFAULT '';

-- Add author_id column for unique author identifiers (usernames, IDs, etc.)
ALTER TABLE sources
ADD COLUMN IF NOT EXISTS author_id VARCHAR(255) DEFAULT '';

COMMIT;
-- Create ingestion_errors table to track RSS feed and scraping failures
CREATE TABLE IF NOT EXISTS ingestion_errors (
    id TEXT PRIMARY KEY,
    platform TEXT NOT NULL,
    error_type TEXT NOT NULL,
    url TEXT NOT NULL,
    error_msg TEXT NOT NULL,
    metadata TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    resolved BOOLEAN NOT NULL DEFAULT FALSE,
    resolved_at TIMESTAMP
);

-- Create indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_ingestion_errors_platform ON ingestion_errors(platform);
CREATE INDEX IF NOT EXISTS idx_ingestion_errors_error_type ON ingestion_errors(error_type);
CREATE INDEX IF NOT EXISTS idx_ingestion_errors_created_at ON ingestion_errors(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ingestion_errors_resolved ON ingestion_errors(resolved);
CREATE INDEX IF NOT EXISTS idx_ingestion_errors_url ON ingestion_errors(url);
-- Threshold configuration table
CREATE TABLE IF NOT EXISTS threshold_config (
    id SERIAL PRIMARY KEY,
    min_confidence DECIMAL(3,2) NOT NULL DEFAULT 0.10,
    min_magnitude DECIMAL(3,1) NOT NULL DEFAULT 0.0,
    max_source_age_hours INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Insert default values
INSERT INTO threshold_config (min_confidence, min_magnitude, max_source_age_hours)
VALUES (0.10, 0.0, 0);
-- Activity logs table for tracking non-error activities
CREATE TABLE IF NOT EXISTS activity_logs (
    id TEXT PRIMARY KEY,
    timestamp TIMESTAMP NOT NULL DEFAULT NOW(),
    activity_type TEXT NOT NULL, -- 'rss_fetch', 'enrichment', 'playwright_scrape', 'correlation', etc.
    platform TEXT, -- 'rss', 'twitter', 'telegram', etc.
    message TEXT NOT NULL,
    details JSONB, -- Additional structured details
    source_count INTEGER, -- Number of sources processed (if applicable)
    duration_ms INTEGER -- Duration in milliseconds (if applicable)
);

-- Create index on timestamp for efficient querying
CREATE INDEX IF NOT EXISTS idx_activity_logs_timestamp ON activity_logs(timestamp DESC);

-- Create index on activity_type for filtering
CREATE INDEX IF NOT EXISTS idx_activity_logs_type ON activity_logs(activity_type);

-- Create index on platform for filtering
CREATE INDEX IF NOT EXISTS idx_activity_logs_platform ON activity_logs(platform);
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
-- Scraper configuration table
CREATE TABLE IF NOT EXISTS scraper_config (
    id SERIAL PRIMARY KEY,
    max_concurrent_workers INTEGER NOT NULL DEFAULT 5,
    scrape_timeout_seconds INTEGER NOT NULL DEFAULT 60,
    max_retries INTEGER NOT NULL DEFAULT 2,
    rate_limit_per_minute INTEGER NOT NULL DEFAULT 60,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Insert default configuration
INSERT INTO scraper_config (max_concurrent_workers, scrape_timeout_seconds, max_retries, rate_limit_per_minute, enabled)
VALUES (5, 60, 2, 60, TRUE)
ON CONFLICT DO NOTHING;
-- Connector Configuration Table
-- Stores configuration for data source connectors (Twitter, Telegram, etc.)

CREATE TABLE IF NOT EXISTS connector_config (
    id TEXT PRIMARY KEY, -- connector name: 'twitter', 'telegram', 'rss'
    enabled BOOLEAN NOT NULL DEFAULT false,
    config JSONB NOT NULL DEFAULT '{}', -- Connector-specific configuration
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create index on enabled for quick filtering
CREATE INDEX IF NOT EXISTS idx_connector_config_enabled ON connector_config(enabled);

-- Insert default configurations
INSERT INTO connector_config (id, enabled, config) VALUES
    ('twitter', false, '{"bearer_token": ""}'),
    ('telegram', false, '{"bot_token": ""}'),
    ('rss', true, '{}')
ON CONFLICT (id) DO NOTHING;

-- Add comments
COMMENT ON TABLE connector_config IS 'Configuration for data source connectors';
COMMENT ON COLUMN connector_config.id IS 'Connector identifier (twitter, telegram, rss)';
COMMENT ON COLUMN connector_config.config IS 'JSON configuration specific to each connector';
-- Add correlation_system_prompt to openai_config table
-- This prompt is used for event correlation and deduplication

ALTER TABLE openai_config
ADD COLUMN IF NOT EXISTS correlation_system_prompt TEXT NOT NULL DEFAULT '';

-- Update with default correlation prompt
UPDATE openai_config SET
    correlation_system_prompt = 'You are an expert OSINT analyst specializing in event correlation and deduplication.

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
New Source: "Russia''s Kyiv attack killed 5 civilians and damaged power station, officials say 15 injured"
Output: {"similarity": 0.9, "should_merge": true, "has_novel_facts": true, "novel_facts": ["15 people injured", "Power station damaged"], "reasoning": "Same event with additional casualty figures and infrastructure damage"}

EXAMPLE 3 - Low similarity, should not merge:
Existing Event: "Russia launches missile strikes on Kyiv, 5 civilians killed"
New Source: "Ukraine drone strike on Russian oil refinery causes major fire"
Output: {"similarity": 0.3, "should_merge": false, "has_novel_facts": false, "novel_facts": [], "reasoning": "Related to same conflict but completely different events"}

Remember: Output ONLY the JSON object. No additional text.',
    updated_at = NOW()
WHERE id = 1;
-- Add scraping status tracking to sources table
-- This allows RSS fetching to be decoupled from content scraping

-- Add scrape_status column with enum-like constraint
ALTER TABLE sources
ADD COLUMN scrape_status VARCHAR(32) NOT NULL DEFAULT 'pending'
CHECK (scrape_status IN ('pending', 'in_progress', 'completed', 'failed', 'skipped'));

-- Add scrape error message column (only populated if scraping fails)
ALTER TABLE sources
ADD COLUMN scrape_error TEXT;

-- Add scraped_at timestamp (when content was actually scraped)
ALTER TABLE sources
ADD COLUMN scraped_at TIMESTAMPTZ;

-- Create index on scrape_status for efficient filtering
CREATE INDEX idx_sources_scrape_status ON sources(scrape_status);

-- Create composite index for common query pattern (status + timestamp)
CREATE INDEX idx_sources_scrape_status_created ON sources(scrape_status, created_at DESC);

-- Update existing sources to 'completed' status (already scraped)
-- This maintains backward compatibility with existing data
UPDATE sources
SET scrape_status = 'completed',
    scraped_at = created_at
WHERE scrape_status = 'pending';

-- Add comment for documentation
COMMENT ON COLUMN sources.scrape_status IS 'Status of content scraping: pending, in_progress, completed, failed, or skipped';
COMMENT ON COLUMN sources.scrape_error IS 'Error message if scraping failed';
COMMENT ON COLUMN sources.scraped_at IS 'Timestamp when content was successfully scraped';
-- Create table for Firecrawl configuration
CREATE TABLE IF NOT EXISTS firecrawl_config (
    id SERIAL PRIMARY KEY,
    api_key TEXT NOT NULL,
    concurrent_requests INTEGER DEFAULT 5,
    enabled BOOLEAN DEFAULT true,
    timeout_seconds INTEGER DEFAULT 30,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Insert default configuration
INSERT INTO firecrawl_config (api_key, concurrent_requests, enabled, timeout_seconds)
VALUES ('fc-f2cd1254f14b40099e82c59ec6c45c67', 5, true, 30)
ON CONFLICT DO NOTHING;

-- Add trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_firecrawl_config_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_firecrawl_config_timestamp
BEFORE UPDATE ON firecrawl_config
FOR EACH ROW
EXECUTE FUNCTION update_firecrawl_config_updated_at();