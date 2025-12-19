#!/bin/bash

# Simple database setup using gcloud commands (no psql required)

set -e

# Configure these for your environment
PROJECT_ID="${GCP_PROJECT_ID:?Error: Set GCP_PROJECT_ID environment variable}"
INSTANCE_NAME="${CLOUD_SQL_INSTANCE:-osint-db}"
DB_NAME="${DB_NAME:-stratint}"

echo "=================================="
echo "OSINTMCP Database Setup (Simple)"
echo "=================================="
echo ""

# Check instance is ready
echo "Checking Cloud SQL instance status..."
STATE=$(gcloud sql instances describe $INSTANCE_NAME \
    --project=$PROJECT_ID \
    --format="value(state)" 2>/dev/null)

if [ "$STATE" != "RUNNABLE" ]; then
    echo "ERROR: Instance not ready (state: $STATE)"
    exit 1
fi
echo "✓ Instance is RUNNABLE"
echo ""

# Create database using gcloud
echo "Creating database '$DB_NAME'..."
gcloud sql databases create $DB_NAME \
    --instance=$INSTANCE_NAME \
    --project=$PROJECT_ID 2>&1 || echo "Database may already exist (this is OK)"

echo ""
echo "✓ Database '$DB_NAME' is ready"
echo ""

# List databases to confirm
echo "Current databases:"
gcloud sql databases list \
    --instance=$INSTANCE_NAME \
    --project=$PROJECT_ID

echo ""
echo "=================================="
echo "✓ Database setup complete!"
echo "=================================="
echo ""
echo "NOTE: Migrations need to be run separately."
echo "The database schema will be created automatically when the"
echo "application starts, or you can run ./setup-db.sh to apply"
echo "migrations manually using Cloud SQL Proxy."
echo ""
echo "Next step: Run ./setup-secrets.sh"
echo ""
