-- Add public field to forecasts table
ALTER TABLE forecasts ADD COLUMN IF NOT EXISTS public BOOLEAN NOT NULL DEFAULT false;

-- Create index for filtering public forecasts
CREATE INDEX IF NOT EXISTS idx_forecasts_public ON forecasts(public) WHERE public = true;

-- Comment
COMMENT ON COLUMN forecasts.public IS 'Whether the forecast is publicly visible on the homepage';
