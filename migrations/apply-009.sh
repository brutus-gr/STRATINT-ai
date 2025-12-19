#!/bin/bash
# Quick migration applier
PROJECT_ID="${GCP_PROJECT_ID:?Error: Set GCP_PROJECT_ID environment variable}"
DB_PASSWORD=$(gcloud secrets versions access latest --secret="db-password" --project="$PROJECT_ID")
export PGPASSWORD="$DB_PASSWORD"

echo "Applying migration 009..."
psql -h 127.0.0.1 -p 54321 -U postgres -d stratint << 'SQL'
ALTER TABLE firecrawl_config ADD COLUMN IF NOT EXISTS max_retries INTEGER DEFAULT 3 NOT NULL;
\dt firecrawl_config
SQL

echo "Migration applied!"
