-- Create forecasts table
CREATE TABLE IF NOT EXISTS forecasts (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    proposition TEXT NOT NULL,
    confidence_interval REAL NOT NULL DEFAULT 0.95, -- e.g., 0.80, 0.90, 0.95
    categories TEXT[] NOT NULL DEFAULT '{}', -- Array of category names to include
    headline_count INTEGER NOT NULL DEFAULT 500, -- Number of headlines to include
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create forecast models table (which models to use and their weights)
CREATE TABLE IF NOT EXISTS forecast_models (
    id TEXT PRIMARY KEY,
    forecast_id TEXT NOT NULL REFERENCES forecasts(id) ON DELETE CASCADE,
    provider TEXT NOT NULL, -- 'anthropic' or 'openai'
    model_name TEXT NOT NULL, -- e.g., 'claude-sonnet-4.5', 'gpt-4'
    api_key TEXT NOT NULL, -- Encrypted API key
    weight REAL NOT NULL DEFAULT 1.0, -- Weight for weighted average
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create forecast runs table (results of running a forecast)
CREATE TABLE IF NOT EXISTS forecast_runs (
    id TEXT PRIMARY KEY,
    forecast_id TEXT NOT NULL REFERENCES forecasts(id) ON DELETE CASCADE,
    run_at TIMESTAMP NOT NULL DEFAULT NOW(),
    headline_count INTEGER NOT NULL, -- Actual number of headlines used
    headlines_snapshot JSONB NOT NULL, -- Store the headlines used
    status TEXT NOT NULL DEFAULT 'pending', -- 'pending', 'running', 'completed', 'failed'
    error_message TEXT,
    completed_at TIMESTAMP
);

-- Create forecast model responses table (individual model responses)
CREATE TABLE IF NOT EXISTS forecast_model_responses (
    id TEXT PRIMARY KEY,
    run_id TEXT NOT NULL REFERENCES forecast_runs(id) ON DELETE CASCADE,
    model_id TEXT NOT NULL REFERENCES forecast_models(id),
    provider TEXT NOT NULL,
    model_name TEXT NOT NULL,
    probability REAL, -- The probability returned by the model (0-1)
    reasoning TEXT, -- Model's reasoning
    confidence_interval_lower REAL, -- Lower bound of confidence interval
    confidence_interval_upper REAL, -- Upper bound of confidence interval
    raw_response JSONB, -- Full raw response from the model
    tokens_used INTEGER,
    response_time_ms INTEGER,
    status TEXT NOT NULL DEFAULT 'pending', -- 'pending', 'completed', 'failed'
    error_message TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create forecast results table (weighted average results)
CREATE TABLE IF NOT EXISTS forecast_results (
    id TEXT PRIMARY KEY,
    run_id TEXT NOT NULL UNIQUE REFERENCES forecast_runs(id) ON DELETE CASCADE,
    weighted_probability REAL NOT NULL, -- Weighted average probability
    weighted_confidence_lower REAL, -- Weighted lower bound
    weighted_confidence_upper REAL, -- Weighted upper bound
    model_count INTEGER NOT NULL, -- Number of models that responded
    consensus_level REAL, -- How much models agree (std dev or similar metric)
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_forecasts_active ON forecasts(active);
CREATE INDEX IF NOT EXISTS idx_forecast_models_forecast_id ON forecast_models(forecast_id);
CREATE INDEX IF NOT EXISTS idx_forecast_runs_forecast_id ON forecast_runs(forecast_id);
CREATE INDEX IF NOT EXISTS idx_forecast_runs_run_at ON forecast_runs(run_at);
CREATE INDEX IF NOT EXISTS idx_forecast_model_responses_run_id ON forecast_model_responses(run_id);
CREATE INDEX IF NOT EXISTS idx_forecast_results_run_id ON forecast_results(run_id);

-- Comments
COMMENT ON TABLE forecasts IS 'Forecast configurations - propositions to be evaluated';
COMMENT ON TABLE forecast_models IS 'AI models to use for each forecast with their weights';
COMMENT ON TABLE forecast_runs IS 'Individual runs of a forecast at a point in time';
COMMENT ON TABLE forecast_model_responses IS 'Individual model responses for each run';
COMMENT ON TABLE forecast_results IS 'Aggregated weighted results for each run';
