-- Migration 037: Fix normalized_forecasts to use TEXT[] for forecast_ids instead of UUID[]

-- Drop and recreate the forecast_ids column with the correct type
ALTER TABLE normalized_forecasts DROP COLUMN IF EXISTS forecast_ids;
ALTER TABLE normalized_forecasts ADD COLUMN forecast_ids TEXT[] NOT NULL DEFAULT '{}';

-- Recreate the GIN index
DROP INDEX IF EXISTS idx_normalized_forecasts_forecast_ids;
CREATE INDEX idx_normalized_forecasts_forecast_ids ON normalized_forecasts USING GIN(forecast_ids);
