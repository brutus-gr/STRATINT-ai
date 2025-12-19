-- Add forecast_history_count column to the OSINTMCP database (not stratint!)
ALTER TABLE strategies ADD COLUMN forecast_history_count INTEGER NOT NULL DEFAULT 1;

-- Verify
SELECT id, name, forecast_history_count FROM strategies;
