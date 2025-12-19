-- Migration 013: Add unique constraint on URL to prevent duplicate sources
--
-- Problem: The same article can be stored multiple times with different source IDs,
-- causing each duplicate to be enriched separately, creating duplicate events with
-- slightly different confidence scores and summaries.
--
-- Solution: Add a unique constraint on the URL column to prevent duplicate sources.

-- First, let's identify and clean up any existing duplicates
-- Keep the earliest source for each URL and delete the rest
WITH duplicates AS (
  SELECT
    id,
    url,
    ROW_NUMBER() OVER (PARTITION BY url ORDER BY created_at ASC) as rn
  FROM sources
  WHERE url IS NOT NULL AND url != ''
)
DELETE FROM sources
WHERE id IN (
  SELECT id FROM duplicates WHERE rn > 1
);

-- Now add the unique constraint
-- Using a partial index to allow NULL urls (though we should have very few)
CREATE UNIQUE INDEX idx_sources_url_unique
ON sources (url)
WHERE url IS NOT NULL AND url != '';

-- Add an index on (title, url) for faster duplicate checks
CREATE INDEX idx_sources_title_url ON sources (title, url) WHERE title IS NOT NULL AND url IS NOT NULL;
