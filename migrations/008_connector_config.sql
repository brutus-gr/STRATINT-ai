-- Connector Configuration Table
-- Stores configuration for data source connectors (Twitter, Telegram, etc.)

CREATE TABLE IF NOT EXISTS connector_config (
    id TEXT PRIMARY KEY, -- connector name: 'twitter', 'telegram', 'rss'
    enabled BOOLEAN NOT NULL DEFAULT false,
    config JSONB NOT NULL DEFAULT '{}', -- Connector-specific configuration
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create index on enabled for quick filtering
CREATE INDEX IF NOT EXISTS idx_connector_config_enabled ON connector_config(enabled);

-- Insert default configurations
INSERT INTO connector_config (id, enabled, config) VALUES
    ('twitter', false, '{"bearer_token": ""}'),
    ('telegram', false, '{"bot_token": ""}'),
    ('rss', true, '{}')
ON CONFLICT (id) DO NOTHING;

-- Add comments
COMMENT ON TABLE connector_config IS 'Configuration for data source connectors';
COMMENT ON COLUMN connector_config.id IS 'Connector identifier (twitter, telegram, rss)';
COMMENT ON COLUMN connector_config.config IS 'JSON configuration specific to each connector';
