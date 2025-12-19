#!/bin/bash

# Apply GLD forecasts migration to production database
# This script adds forecast questions for GLD (gold ETF) price movements

set -e

# Configuration - set these environment variables before running
PROJECT_ID="${GCP_PROJECT_ID:?Error: Set GCP_PROJECT_ID environment variable}"
REGION="${GCP_REGION:-us-central1}"
INSTANCE_NAME="${CLOUD_SQL_INSTANCE:-osint-db}"
INSTANCE_CONNECTION_NAME="${PROJECT_ID}:${REGION}:${INSTANCE_NAME}"
DB_NAME="${DB_NAME:-stratint}"
DB_USER="${DB_USER:-postgres}"

echo "🔧 Applying GLD Forecasts Migration to Production Database"
echo "Project: $PROJECT_ID"
echo "Instance: $INSTANCE_CONNECTION_NAME"
echo ""

# Get DB password from secret manager
echo "📝 Retrieving database password from Secret Manager..."
DB_PASSWORD=$(gcloud secrets versions access latest --secret="db-password" --project="$PROJECT_ID")

# Start cloud-sql-proxy in background if not already running
if ! pgrep -f "cloud-sql-proxy.*$INSTANCE_CONNECTION_NAME" > /dev/null; then
    echo "🚀 Starting Cloud SQL Proxy..."
    ./cloud-sql-proxy "$INSTANCE_CONNECTION_NAME" &
    PROXY_PID=$!
    echo "Started proxy with PID: $PROXY_PID"
    sleep 3
else
    echo "✓ Cloud SQL Proxy already running"
    PROXY_PID=""
fi

# Apply migration
echo ""
echo "📊 Applying migration 038_add_gld_forecasts.sql..."
PGPASSWORD="$DB_PASSWORD" psql \
    -h 127.0.0.1 \
    -p 5432 \
    -U "$DB_USER" \
    -d "$DB_NAME" \
    -f migrations/038_add_gld_forecasts.sql

echo ""
echo "✅ Migration applied successfully!"

# Verify forecasts were created
echo ""
echo "🔍 Verifying GLD forecasts..."
PGPASSWORD="$DB_PASSWORD" psql \
    -h 127.0.0.1 \
    -p 5432 \
    -U "$DB_USER" \
    -d "$DB_NAME" \
    -c "SELECT id, name, threshold_percent, threshold_direction, threshold_operator FROM forecasts WHERE id LIKE 'gld-%' ORDER BY threshold_direction DESC, threshold_percent ASC;"

# Clean up proxy if we started it
if [ ! -z "$PROXY_PID" ]; then
    echo ""
    echo "🛑 Stopping Cloud SQL Proxy..."
    kill $PROXY_PID 2>/dev/null || true
fi

echo ""
echo "✨ Done! GLD forecasts have been added to the database."
echo ""
echo "You can now:"
echo "  1. Go to the admin panel: https://[YOUR-SERVICE-URL]/admin"
echo "  2. Navigate to the 'Forecasts' tab"
echo "  3. Find the GLD forecasts and add your API keys to the models"
echo "  4. Execute the forecasts to get probability estimates"
echo "  5. Use the 'Normalized Forecasts' tab to combine them into a distribution"
echo ""
