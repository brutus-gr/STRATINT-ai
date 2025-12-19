-- Add include_forecasts field to summaries table
ALTER TABLE summaries ADD COLUMN IF NOT EXISTS include_forecasts BOOLEAN NOT NULL DEFAULT false;
