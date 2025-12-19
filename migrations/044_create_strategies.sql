-- Create strategies feature for AI-generated portfolio allocation
-- Strategies take user prompts, inject headlines + forecast data, and return structured allocations

-- Main strategies table
CREATE TABLE IF NOT EXISTS strategies (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL,
  prompt TEXT NOT NULL, -- User-defined directive (e.g., "Build an aggressive portfolio allocation based on geopolitical risk")
  investment_symbols TEXT[] NOT NULL, -- Available investments (e.g., ["SPY", "VNQ", "TLT", "GLD", "CASH"])
  categories TEXT[] NOT NULL DEFAULT '{}', -- Signal categories to include in analysis
  headline_count INTEGER NOT NULL DEFAULT 500,
  iterations INTEGER NOT NULL DEFAULT 3, -- Number of times to run before averaging
  forecast_ids TEXT[] DEFAULT '{}', -- IDs of forecasts to inject (latest run with full percentiles)
  active BOOLEAN NOT NULL DEFAULT true,
  public BOOLEAN NOT NULL DEFAULT false, -- Whether visible on homepage
  display_order INTEGER NOT NULL DEFAULT 0, -- Sort order for homepage display (higher = earlier)
  schedule_enabled BOOLEAN NOT NULL DEFAULT false,
  schedule_interval INTEGER DEFAULT 0, -- Interval in minutes (e.g., 60 for hourly, 1440 for daily)
  last_run_at TIMESTAMP,
  next_run_at TIMESTAMP,
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_strategies_public ON strategies(public) WHERE public = true;
CREATE INDEX idx_strategies_display_order ON strategies(display_order DESC) WHERE public = true;
CREATE INDEX idx_strategies_next_run_at ON strategies(next_run_at) WHERE schedule_enabled = true AND active = true;

-- Strategy models configuration (similar to forecast_models)
CREATE TABLE IF NOT EXISTS strategy_models (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  strategy_id UUID NOT NULL REFERENCES strategies(id) ON DELETE CASCADE,
  provider TEXT NOT NULL, -- 'anthropic' or 'openai'
  model_name TEXT NOT NULL,
  api_key TEXT NOT NULL, -- Should be encrypted in production
  weight FLOAT NOT NULL DEFAULT 1.0,
  active BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_strategy_models_strategy_id ON strategy_models(strategy_id);

-- Strategy execution runs
CREATE TABLE IF NOT EXISTS strategy_runs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  strategy_id UUID NOT NULL REFERENCES strategies(id) ON DELETE CASCADE,
  run_at TIMESTAMP NOT NULL DEFAULT NOW(),
  headline_count INTEGER NOT NULL,
  headlines_snapshot JSONB, -- Snapshot of headlines used in this run
  forecast_snapshots JSONB, -- Snapshot of forecast data injected (full percentiles P10-P90)
  status TEXT NOT NULL DEFAULT 'pending', -- 'pending', 'running', 'completed', 'failed'
  error_message TEXT,
  completed_at TIMESTAMP
);

CREATE INDEX idx_strategy_runs_strategy_id ON strategy_runs(strategy_id);
CREATE INDEX idx_strategy_runs_run_at ON strategy_runs(run_at DESC);

-- Individual model responses per iteration
CREATE TABLE IF NOT EXISTS strategy_model_responses (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  run_id UUID NOT NULL REFERENCES strategy_runs(id) ON DELETE CASCADE,
  model_id UUID NOT NULL REFERENCES strategy_models(id) ON DELETE CASCADE,
  iteration INTEGER NOT NULL, -- Which iteration (1 to N)
  provider TEXT NOT NULL,
  model_name TEXT NOT NULL,
  allocations JSONB NOT NULL, -- Percentage allocation per symbol: {"SPY": 40.0, "VNQ": 25.0, "TLT": 20.0, "GLD": 10.0, "CASH": 5.0}
  reasoning TEXT, -- Model's explanation for this allocation
  raw_response JSONB, -- Full API response for debugging
  tokens_used INTEGER,
  response_time_ms INTEGER,
  status TEXT NOT NULL DEFAULT 'pending', -- 'pending', 'completed', 'failed'
  error_message TEXT,
  created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_strategy_model_responses_run_id ON strategy_model_responses(run_id);
CREATE INDEX idx_strategy_model_responses_iteration ON strategy_model_responses(run_id, iteration);

-- Aggregated strategy result
CREATE TABLE IF NOT EXISTS strategy_results (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  run_id UUID NOT NULL REFERENCES strategy_runs(id) ON DELETE CASCADE,
  averaged_allocations JSONB NOT NULL, -- Simple average across all iterations: {"SPY": 42.3, "VNQ": 23.1, ...}
  normalized_allocations JSONB NOT NULL, -- Final AI-normalized to sum to exactly 100%: {"SPY": 42.0, "VNQ": 23.0, ...}
  normalization_reasoning TEXT, -- AI's explanation for normalization adjustments
  model_count INTEGER NOT NULL,
  iteration_count INTEGER NOT NULL,
  consensus_variance JSONB, -- Standard deviation per symbol showing iteration agreement
  created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_strategy_results_run_id ON strategy_results(run_id);

-- Comments
COMMENT ON TABLE strategies IS 'AI-generated portfolio allocation strategies combining intelligence signals and forecasts';
COMMENT ON COLUMN strategies.investment_symbols IS 'List of ticker symbols available for allocation (e.g., ["SPY", "VNQ", "TLT", "GLD", "CASH"])';
COMMENT ON COLUMN strategies.iterations IS 'Number of times to run each model before averaging allocations';
COMMENT ON COLUMN strategies.forecast_ids IS 'Forecast IDs to inject as context (latest run with full P10-P90 percentiles)';
COMMENT ON COLUMN strategy_model_responses.allocations IS 'Percentage allocation per symbol from this iteration (structured JSON)';
COMMENT ON COLUMN strategy_results.averaged_allocations IS 'Simple mathematical average of allocations across all (models × iterations) responses';
COMMENT ON COLUMN strategy_results.normalized_allocations IS 'AI-adjusted allocations ensuring sum = exactly 100.0%';
COMMENT ON COLUMN strategy_results.consensus_variance IS 'Standard deviation per symbol showing agreement/disagreement across iterations';
