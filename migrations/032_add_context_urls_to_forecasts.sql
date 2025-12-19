-- Add context_urls column to forecasts table
ALTER TABLE forecasts ADD COLUMN IF NOT EXISTS context_urls TEXT[] DEFAULT '{}';

-- Add iterations column (was missing from original migration)
ALTER TABLE forecasts ADD COLUMN IF NOT EXISTS iterations INTEGER NOT NULL DEFAULT 1;

-- Comment
COMMENT ON COLUMN forecasts.context_urls IS 'URLs to fetch and inject as context before headlines';
COMMENT ON COLUMN forecasts.iterations IS 'Number of times to query each model for consensus';
