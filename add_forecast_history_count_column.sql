-- Add the forecast_history_count column to strategies table
ALTER TABLE strategies ADD COLUMN forecast_history_count INTEGER NOT NULL DEFAULT 1;

-- Verify it was added
SELECT column_name, data_type, column_default
FROM information_schema.columns
WHERE table_name = 'strategies' AND column_name = 'forecast_history_count';

-- Show updated strategies
SELECT id, name, forecast_history_count FROM strategies;
