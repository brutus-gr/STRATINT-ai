-- Migration 014: Add performance indexes for common query patterns
-- These indexes significantly improve query performance for:
-- 1. Event detail loading (sources and entities relations)
-- 2. Enrichment worker claim queries
-- 3. Full-text search with status filtering

-- HIGH PRIORITY: Junction table indexes for event relations
-- These are used on EVERY event detail query and event listing with relations
-- Without these, we're doing sequential scans of junction tables

CREATE INDEX IF NOT EXISTS idx_event_sources_event_id
ON event_sources(event_id);

CREATE INDEX IF NOT EXISTS idx_event_entities_event_id
ON event_entities(event_id);

-- MEDIUM PRIORITY: Partial index for enrichment worker queries
-- The ClaimForEnrichment query filters on raw_content != ''
-- This partial index only includes sources with content, making it smaller and faster
-- Estimated 30-50% smaller index size vs full table index

CREATE INDEX IF NOT EXISTS idx_sources_has_content
ON sources(scrape_status, enrichment_status, created_at ASC)
WHERE raw_content IS NOT NULL AND raw_content != '';

-- MEDIUM PRIORITY: Partial index for stale enrichment claim detection
-- Used to find enrichments that were claimed but never completed (crashed workers)
-- Only indexes rows where enrichment is in progress

CREATE INDEX IF NOT EXISTS idx_sources_enrichment_claimed
ON sources(enrichment_claimed_at)
WHERE enrichment_status = 'enriching';

-- Add comments for documentation
COMMENT ON INDEX idx_event_sources_event_id IS 'Improves event detail queries - loads sources for an event';
COMMENT ON INDEX idx_event_entities_event_id IS 'Improves event detail queries - loads entities for an event';
COMMENT ON INDEX idx_sources_has_content IS 'Partial index for enrichment worker - only sources with content';
COMMENT ON INDEX idx_sources_enrichment_claimed IS 'Partial index for stale enrichment detection';
