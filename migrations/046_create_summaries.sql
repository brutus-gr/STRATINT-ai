-- Create summaries table
CREATE TABLE IF NOT EXISTS summaries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    prompt TEXT NOT NULL,
    time_of_day TIME, -- Optional: specific time to run (e.g., 09:00)
    lookback_hours INTEGER NOT NULL DEFAULT 24,
    categories TEXT[] DEFAULT '{}',
    headline_count INTEGER NOT NULL DEFAULT 100,
    models JSONB NOT NULL DEFAULT '[]'::jsonb,
    active BOOLEAN NOT NULL DEFAULT true,
    schedule_enabled BOOLEAN NOT NULL DEFAULT false,
    schedule_interval INTEGER NOT NULL DEFAULT 1440, -- Default: daily (in minutes)
    last_run_at TIMESTAMP WITH TIME ZONE,
    next_run_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create summary_runs table
CREATE TABLE IF NOT EXISTS summary_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    summary_id UUID NOT NULL REFERENCES summaries(id) ON DELETE CASCADE,
    run_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    headline_count INTEGER NOT NULL,
    lookback_start TIMESTAMP WITH TIME ZONE NOT NULL,
    lookback_end TIMESTAMP WITH TIME ZONE NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    error_message TEXT,
    completed_at TIMESTAMP WITH TIME ZONE
);

-- Create summary_results table
CREATE TABLE IF NOT EXISTS summary_results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id UUID NOT NULL REFERENCES summary_runs(id) ON DELETE CASCADE,
    summary_text TEXT NOT NULL,
    model_provider TEXT NOT NULL,
    model_name TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_summaries_active ON summaries(active);
CREATE INDEX IF NOT EXISTS idx_summaries_schedule ON summaries(schedule_enabled, next_run_at);
CREATE INDEX IF NOT EXISTS idx_summary_runs_summary_id ON summary_runs(summary_id);
CREATE INDEX IF NOT EXISTS idx_summary_runs_run_at ON summary_runs(run_at DESC);
CREATE INDEX IF NOT EXISTS idx_summary_results_run_id ON summary_results(run_id);

-- Create trigger to update updated_at
CREATE OR REPLACE FUNCTION update_summaries_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_summaries_updated_at
    BEFORE UPDATE ON summaries
    FOR EACH ROW
    EXECUTE FUNCTION update_summaries_updated_at();
