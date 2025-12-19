-- Migration 025: Clean up scraping-related columns and tables
-- This migration removes all scraping-related functionality after switching to RSS-only model

-- Drop scraping-related tables if they exist
DROP TABLE IF EXISTS scraper_config CASCADE;
DROP TABLE IF EXISTS firecrawl_config CASCADE;
DROP TABLE IF EXISTS proxy_config CASCADE;
DROP TABLE IF EXISTS captcha_config CASCADE;

-- Drop scraping-related columns from connector_config if they exist
ALTER TABLE connector_config
DROP COLUMN IF EXISTS firecrawl_api_key,
DROP COLUMN IF EXISTS firecrawl_enabled,
DROP COLUMN IF EXISTS firecrawl_max_retries,
DROP COLUMN IF EXISTS proxy_enabled,
DROP COLUMN IF EXISTS proxy_url,
DROP COLUMN IF EXISTS captcha_enabled,
DROP COLUMN IF EXISTS captcha_api_key;

-- Note: We're keeping scrape_status columns in the sources table as they're used
-- to indicate that RSS content is "complete" and ready for enrichment