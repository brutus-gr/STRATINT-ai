-- Migration 048: Add auto_post_to_twitter field to summaries
-- This allows summaries to automatically post their first result to Twitter when run

ALTER TABLE summaries ADD COLUMN IF NOT EXISTS auto_post_to_twitter BOOLEAN NOT NULL DEFAULT false;
