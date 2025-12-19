-- Add threshold_percent to forecasts to specify the underlying % change the question is asking about
ALTER TABLE forecasts ADD COLUMN IF NOT EXISTS threshold_percent REAL;
ALTER TABLE forecasts ADD COLUMN IF NOT EXISTS threshold_direction VARCHAR(10);

COMMENT ON COLUMN forecasts.threshold_percent IS 'The % change threshold this forecast is asking about (e.g., 5.0 for "5% change")';
COMMENT ON COLUMN forecasts.threshold_direction IS 'Direction of threshold: "up" for gains, "down" for losses';
