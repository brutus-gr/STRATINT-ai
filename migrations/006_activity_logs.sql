-- Activity logs table for tracking non-error activities
CREATE TABLE IF NOT EXISTS activity_logs (
    id TEXT PRIMARY KEY,
    timestamp TIMESTAMP NOT NULL DEFAULT NOW(),
    activity_type TEXT NOT NULL, -- 'rss_fetch', 'enrichment', 'playwright_scrape', 'correlation', etc.
    platform TEXT, -- 'rss', 'twitter', 'telegram', etc.
    message TEXT NOT NULL,
    details JSONB, -- Additional structured details
    source_count INTEGER, -- Number of sources processed (if applicable)
    duration_ms INTEGER -- Duration in milliseconds (if applicable)
);

-- Create index on timestamp for efficient querying
CREATE INDEX IF NOT EXISTS idx_activity_logs_timestamp ON activity_logs(timestamp DESC);

-- Create index on activity_type for filtering
CREATE INDEX IF NOT EXISTS idx_activity_logs_type ON activity_logs(activity_type);

-- Create index on platform for filtering
CREATE INDEX IF NOT EXISTS idx_activity_logs_platform ON activity_logs(platform);
