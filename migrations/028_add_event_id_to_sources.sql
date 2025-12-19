-- Add event_id column to sources table for tracking which event was created from each source
ALTER TABLE sources
ADD COLUMN event_id TEXT;

-- Add index for faster lookups of sources by event
CREATE INDEX IF NOT EXISTS idx_sources_event_id ON sources(event_id);

-- Add comment explaining the column
COMMENT ON COLUMN sources.event_id IS 'ID of the event that was created from this source during enrichment';
