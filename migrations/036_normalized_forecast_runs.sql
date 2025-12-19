-- Migration 036: Add normalized forecast runs table and update normalized_forecasts

-- First, modify the existing normalized_forecasts table to remove distribution/statistics
-- (these will now be stored in normalized_forecast_runs)
ALTER TABLE normalized_forecasts DROP COLUMN IF EXISTS distribution;
ALTER TABLE normalized_forecasts DROP COLUMN IF EXISTS statistics;

-- Table for normalized forecast run results (historical tracking)
CREATE TABLE IF NOT EXISTS normalized_forecast_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    normalized_forecast_id UUID NOT NULL REFERENCES normalized_forecasts(id) ON DELETE CASCADE,
    run_at TIMESTAMP NOT NULL DEFAULT NOW(),
    mean REAL NOT NULL,
    median REAL NOT NULL,
    mode REAL NOT NULL,
    std_dev REAL NOT NULL,
    variance REAL NOT NULL,
    percentile_10 REAL NOT NULL,
    percentile_25 REAL NOT NULL,
    percentile_75 REAL NOT NULL,
    percentile_90 REAL NOT NULL,
    distribution_data JSONB,  -- Full distribution for detailed analysis
    status VARCHAR(50) NOT NULL DEFAULT 'completed',
    error_message TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_normalized_forecast_runs_normalized_forecast_id ON normalized_forecast_runs(normalized_forecast_id);
CREATE INDEX IF NOT EXISTS idx_normalized_forecast_runs_run_at ON normalized_forecast_runs(run_at DESC);
CREATE INDEX IF NOT EXISTS idx_normalized_forecast_runs_status ON normalized_forecast_runs(status);

COMMENT ON TABLE normalized_forecast_runs IS 'Historical results of normalized forecast executions';
COMMENT ON COLUMN normalized_forecast_runs.distribution_data IS 'Full probability distribution data (x, cdf, pdf)';
