-- Simplify forecasts to direct value predictions
-- Remove threshold-based yes/no approach and parent forecast complexity

-- Drop parent forecast tables
DROP TABLE IF EXISTS parent_forecasts CASCADE;

-- Remove threshold fields from forecasts (they're no longer needed)
ALTER TABLE forecasts
  DROP COLUMN IF EXISTS parent_forecast_id,
  DROP COLUMN IF EXISTS threshold_percent,
  DROP COLUMN IF EXISTS threshold_direction,
  DROP COLUMN IF EXISTS threshold_operator;

-- Add new fields for value-based predictions
ALTER TABLE forecasts
  ADD COLUMN IF NOT EXISTS prediction_type TEXT NOT NULL DEFAULT 'percentile', -- 'percentile' or 'point_estimate'
  ADD COLUMN IF NOT EXISTS units TEXT, -- e.g., 'percent_change', 'dollars', 'points'
  ADD COLUMN IF NOT EXISTS target_date TIMESTAMP; -- When the prediction is for

-- Update forecast_model_responses to store percentile predictions
ALTER TABLE forecast_model_responses
  DROP COLUMN IF EXISTS probability,
  DROP COLUMN IF EXISTS confidence_interval_lower,
  DROP COLUMN IF EXISTS confidence_interval_upper;

ALTER TABLE forecast_model_responses
  ADD COLUMN IF NOT EXISTS percentile_predictions JSONB;
  -- Will store: {"p10": 5.2, "p25": 7.1, "p50": 9.5, "p75": 12.3, "p90": 15.7}

-- Update forecast_results to store aggregated percentiles
ALTER TABLE forecast_results
  DROP COLUMN IF EXISTS weighted_probability,
  DROP COLUMN IF EXISTS weighted_confidence_lower,
  DROP COLUMN IF EXISTS weighted_confidence_upper;

ALTER TABLE forecast_results
  ADD COLUMN IF NOT EXISTS aggregated_percentiles JSONB;
  -- Will store weighted average of all model percentiles

-- Comments
COMMENT ON COLUMN forecasts.prediction_type IS 'Type of prediction: percentile (full distribution) or point_estimate (single value)';
COMMENT ON COLUMN forecasts.units IS 'Units of prediction: percent_change, dollars, points, etc.';
COMMENT ON COLUMN forecasts.target_date IS 'Date/time the prediction is for';
COMMENT ON COLUMN forecast_model_responses.percentile_predictions IS 'Model predictions as percentiles: {p10, p25, p50, p75, p90}';
COMMENT ON COLUMN forecast_results.aggregated_percentiles IS 'Weighted average of model percentile predictions';
