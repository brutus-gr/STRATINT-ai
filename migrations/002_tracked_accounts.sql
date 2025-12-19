-- Migration 002: Add tracked social media accounts
-- This allows admins to specify which accounts to monitor

CREATE TABLE tracked_accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    platform VARCHAR(32) NOT NULL, -- 'twitter', 'telegram', 'rss', etc.
    account_identifier VARCHAR(255) NOT NULL, -- username, handle, or channel ID
    display_name VARCHAR(255),
    enabled BOOLEAN NOT NULL DEFAULT true,
    last_fetched_id VARCHAR(255), -- Last tweet/post ID fetched (for pagination)
    last_fetched_at TIMESTAMP WITH TIME ZONE,
    fetch_interval_minutes INT NOT NULL DEFAULT 5, -- How often to check for new posts
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    UNIQUE(platform, account_identifier)
);

-- Index for efficient queries
CREATE INDEX idx_tracked_accounts_platform_enabled ON tracked_accounts(platform, enabled);
CREATE INDEX idx_tracked_accounts_last_fetched ON tracked_accounts(last_fetched_at) WHERE enabled = true;

-- Add comments
COMMENT ON TABLE tracked_accounts IS 'Social media accounts being monitored for OSINT';
COMMENT ON COLUMN tracked_accounts.account_identifier IS 'Platform-specific identifier (e.g., @elonmusk for Twitter)';
COMMENT ON COLUMN tracked_accounts.last_fetched_id IS 'Used for pagination to avoid re-fetching old posts';
