#!/bin/bash
# Apply proxy configuration migration
PROJECT_ID="${GCP_PROJECT_ID:?Error: Set GCP_PROJECT_ID environment variable}"
DB_PASSWORD=$(gcloud secrets versions access latest --secret="db-password" --project="$PROJECT_ID")
export PGPASSWORD="$DB_PASSWORD"

echo "Applying migration 021 (proxy configuration)..."
psql -h 127.0.0.1 -p 54321 -U postgres -d stratint < migrations/021_proxy_config.sql

echo "Verifying proxy_config table..."
psql -h 127.0.0.1 -p 54321 -U postgres -d stratint << 'SQL'
\dt proxy_config
SELECT * FROM proxy_config;
SQL

echo "Migration 021 applied successfully!"
