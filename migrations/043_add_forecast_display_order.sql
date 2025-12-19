-- Add display_order field to forecasts for controlling homepage sort order
ALTER TABLE forecasts ADD COLUMN IF NOT EXISTS display_order INTEGER DEFAULT 0;

-- Create index for efficient ordering
CREATE INDEX IF NOT EXISTS idx_forecasts_display_order ON forecasts(display_order DESC);

COMMENT ON COLUMN forecasts.display_order IS 'Sort order for displaying forecasts on homepage (higher = earlier)';
