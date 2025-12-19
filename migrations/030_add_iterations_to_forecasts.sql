-- Add iterations column to forecasts table
ALTER TABLE forecasts ADD COLUMN iterations INTEGER NOT NULL DEFAULT 1;

-- Update existing forecasts to have 1 iteration
UPDATE forecasts SET iterations = 1 WHERE iterations IS NULL;
