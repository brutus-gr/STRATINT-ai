-- Create normalized_forecasts table to store pre-calculated distributions
CREATE TABLE IF NOT EXISTS normalized_forecasts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    forecast_ids UUID[] NOT NULL,
    distribution JSONB NOT NULL,
    statistics JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_normalized_forecasts_created_at ON normalized_forecasts(created_at DESC);
CREATE INDEX idx_normalized_forecasts_forecast_ids ON normalized_forecasts USING GIN(forecast_ids);

COMMENT ON TABLE normalized_forecasts IS 'Pre-calculated normalized forecast distributions';
COMMENT ON COLUMN normalized_forecasts.forecast_ids IS 'Array of forecast IDs included in this normalized view';
COMMENT ON COLUMN normalized_forecasts.distribution IS 'JSON array of {x, pdf, cdf} points';
COMMENT ON COLUMN normalized_forecasts.statistics IS 'JSON object with mean, median, mode, variance, std_dev';
