-- Create parent_forecasts table for managing threshold-based forecast templates
CREATE TABLE IF NOT EXISTS parent_forecasts (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    base_proposition TEXT NOT NULL,
    thresholds REAL[] NOT NULL,
    confidence_interval REAL NOT NULL DEFAULT 0.95,
    categories TEXT[] NOT NULL DEFAULT '{}',
    headline_count INTEGER NOT NULL DEFAULT 500,
    iterations INTEGER NOT NULL DEFAULT 1,
    context_urls TEXT[] NOT NULL DEFAULT '{}',
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create parent_forecast_models table for model configurations
CREATE TABLE IF NOT EXISTS parent_forecast_models (
    id TEXT PRIMARY KEY,
    parent_forecast_id TEXT NOT NULL REFERENCES parent_forecasts(id) ON DELETE CASCADE,
    provider TEXT NOT NULL,
    model_name TEXT NOT NULL,
    api_key TEXT NOT NULL,
    weight REAL NOT NULL DEFAULT 1.0,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Add parent_forecast_id to existing forecasts table
ALTER TABLE forecasts ADD COLUMN IF NOT EXISTS parent_forecast_id TEXT REFERENCES parent_forecasts(id) ON DELETE CASCADE;

-- Add parent_forecast_id to normalized_forecasts table
ALTER TABLE normalized_forecasts ADD COLUMN IF NOT EXISTS parent_forecast_id TEXT REFERENCES parent_forecasts(id) ON DELETE SET NULL;

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_parent_forecasts_active ON parent_forecasts(active);
CREATE INDEX IF NOT EXISTS idx_parent_forecast_models_parent_id ON parent_forecast_models(parent_forecast_id);
CREATE INDEX IF NOT EXISTS idx_forecasts_parent_id ON forecasts(parent_forecast_id);
CREATE INDEX IF NOT EXISTS idx_normalized_forecasts_parent_id ON normalized_forecasts(parent_forecast_id);

-- Add constraint to ensure thresholds array is not empty
ALTER TABLE parent_forecasts ADD CONSTRAINT check_thresholds_not_empty
    CHECK (array_length(thresholds, 1) > 0);

-- Add constraint to normalized_forecasts: either parent_forecast_id OR forecast_ids, not both
-- Note: We'll handle this in application logic since forecast_ids is TEXT[] and checking for empty array is complex
-- ALTER TABLE normalized_forecasts ADD CONSTRAINT check_forecast_reference
--     CHECK (
--         (parent_forecast_id IS NOT NULL AND forecast_ids = '{}') OR
--         (parent_forecast_id IS NULL AND array_length(forecast_ids, 1) > 0)
--     );

-- Comments
COMMENT ON TABLE parent_forecasts IS 'Parent forecast templates that manage multiple threshold-based child forecasts';
COMMENT ON TABLE parent_forecast_models IS 'AI models to use for parent forecasts (inherited by children)';
COMMENT ON COLUMN parent_forecasts.base_proposition IS 'Proposition template with placeholders: {THRESHOLD}, {SIGN}, {DIRECTION}';
COMMENT ON COLUMN parent_forecasts.thresholds IS 'Array of threshold percentages (e.g., [-25, -20, -15, ..., 15, 20, 25])';
COMMENT ON COLUMN forecasts.parent_forecast_id IS 'Reference to parent forecast if this is an auto-managed child forecast';
COMMENT ON COLUMN normalized_forecasts.parent_forecast_id IS 'Reference to parent forecast to auto-include all its children';
