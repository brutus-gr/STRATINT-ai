-- Add enrichment status fields to sources table for cross-instance locking
-- This prevents race conditions when multiple Cloud Run instances are running

BEGIN;

-- Add enrichment_status column with CHECK constraint
ALTER TABLE sources
ADD COLUMN IF NOT EXISTS enrichment_status VARCHAR(32) NOT NULL DEFAULT 'pending'
CHECK (enrichment_status IN ('pending', 'enriching', 'completed', 'failed'));

-- Add enrichment_error column to store error messages
ALTER TABLE sources
ADD COLUMN IF NOT EXISTS enrichment_error TEXT;

-- Add enriched_at timestamp (when enrichment completed)
ALTER TABLE sources
ADD COLUMN IF NOT EXISTS enriched_at TIMESTAMPTZ;

-- Add enrichment_claimed_at (for detecting stale locks)
ALTER TABLE sources
ADD COLUMN IF NOT EXISTS enrichment_claimed_at TIMESTAMPTZ;

-- Create index on enrichment_status for efficient filtering
CREATE INDEX IF NOT EXISTS idx_sources_enrichment_status ON sources(enrichment_status);

-- Create composite index for efficient unenriched source queries
CREATE INDEX IF NOT EXISTS idx_sources_scrape_enrichment ON sources(scrape_status, enrichment_status, created_at DESC);

COMMIT;
