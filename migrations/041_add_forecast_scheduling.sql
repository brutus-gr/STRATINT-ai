-- Migration 041: Add scheduling fields to forecasts table
-- This migration adds the ability to schedule forecasts to run automatically

-- Add scheduling columns to forecasts table
ALTER TABLE forecasts ADD COLUMN IF NOT EXISTS schedule_enabled BOOLEAN DEFAULT FALSE;
ALTER TABLE forecasts ADD COLUMN IF NOT EXISTS schedule_interval INTEGER DEFAULT 0; -- Interval in minutes (e.g., 60 for hourly)
ALTER TABLE forecasts ADD COLUMN IF NOT EXISTS last_run_at TIMESTAMP;
ALTER TABLE forecasts ADD COLUMN IF NOT EXISTS next_run_at TIMESTAMP;

-- Create index for efficient scheduling queries
CREATE INDEX IF NOT EXISTS idx_forecasts_next_run ON forecasts(next_run_at) WHERE schedule_enabled = TRUE;

-- Add comment for documentation
COMMENT ON COLUMN forecasts.schedule_enabled IS 'Whether automatic scheduling is enabled for this forecast';
COMMENT ON COLUMN forecasts.schedule_interval IS 'Interval in minutes between forecast runs (e.g., 60 = hourly, 1440 = daily)';
COMMENT ON COLUMN forecasts.last_run_at IS 'Timestamp of when the forecast was last executed';
COMMENT ON COLUMN forecasts.next_run_at IS 'Timestamp of when the forecast should run next';
