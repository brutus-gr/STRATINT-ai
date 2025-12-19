-- Create ingestion_errors table to track RSS feed and scraping failures
CREATE TABLE IF NOT EXISTS ingestion_errors (
    id TEXT PRIMARY KEY,
    platform TEXT NOT NULL,
    error_type TEXT NOT NULL,
    url TEXT NOT NULL,
    error_msg TEXT NOT NULL,
    metadata TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    resolved BOOLEAN NOT NULL DEFAULT FALSE,
    resolved_at TIMESTAMP
);

-- Create indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_ingestion_errors_platform ON ingestion_errors(platform);
CREATE INDEX IF NOT EXISTS idx_ingestion_errors_error_type ON ingestion_errors(error_type);
CREATE INDEX IF NOT EXISTS idx_ingestion_errors_created_at ON ingestion_errors(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ingestion_errors_resolved ON ingestion_errors(resolved);
CREATE INDEX IF NOT EXISTS idx_ingestion_errors_url ON ingestion_errors(url);
