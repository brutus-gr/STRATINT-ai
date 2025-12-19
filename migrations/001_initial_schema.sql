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
