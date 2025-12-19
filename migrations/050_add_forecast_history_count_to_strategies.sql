-- Add forecast_history_count to strategies table
ALTER TABLE strategies ADD COLUMN IF NOT EXISTS forecast_history_count INTEGER NOT NULL DEFAULT 1;

-- Set a reasonable default for existing strategies
UPDATE strategies SET forecast_history_count = 1 WHERE forecast_history_count IS NULL OR forecast_history_count < 1;
