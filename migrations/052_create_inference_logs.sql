-- Create inference_logs table to track all LLM API calls
CREATE TABLE IF NOT EXISTS inference_logs (
    id SERIAL PRIMARY KEY,
    provider VARCHAR(50) NOT NULL,           -- 'openai', 'anthropic', etc.
    model VARCHAR(100) NOT NULL,             -- 'gpt-4o', 'claude-sonnet-4', etc.
    operation VARCHAR(100) NOT NULL,         -- 'event_creation', 'twitter_post', 'forecast', 'strategy', etc.
    tokens_used INTEGER NOT NULL DEFAULT 0,  -- Total tokens (input + output)
    input_tokens INTEGER,                    -- Input tokens (if available)
    output_tokens INTEGER,                   -- Output tokens (if available)
    cost_usd DECIMAL(10, 6),                 -- Estimated cost in USD (if calculable)
    latency_ms INTEGER,                      -- Response time in milliseconds
    status VARCHAR(20) NOT NULL DEFAULT 'success', -- 'success', 'error'
    error_message TEXT,                      -- Error details if failed
    metadata JSONB,                          -- Additional context (event_id, strategy_id, etc.)
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Add indexes for common queries
CREATE INDEX IF NOT EXISTS idx_inference_logs_created_at ON inference_logs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_inference_logs_provider ON inference_logs(provider);
CREATE INDEX IF NOT EXISTS idx_inference_logs_operation ON inference_logs(operation);
CREATE INDEX IF NOT EXISTS idx_inference_logs_model ON inference_logs(model);
CREATE INDEX IF NOT EXISTS idx_inference_logs_status ON inference_logs(status);

-- Add composite index for filtering
CREATE INDEX IF NOT EXISTS idx_inference_logs_provider_created_at ON inference_logs(provider, created_at DESC);
