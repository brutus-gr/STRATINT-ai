-- Migration: Add title and author_id to sources table
-- These fields were present in the Go model but missing from the database schema

BEGIN;

-- Add title column for storing article/post titles from RSS and other sources
ALTER TABLE sources
ADD COLUMN IF NOT EXISTS title VARCHAR(512) DEFAULT '';

-- Add author_id column for unique author identifiers (usernames, IDs, etc.)
ALTER TABLE sources
ADD COLUMN IF NOT EXISTS author_id VARCHAR(255) DEFAULT '';

COMMIT;
