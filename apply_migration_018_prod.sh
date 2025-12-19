#!/bin/bash
# Apply migration 018 to production Cloud SQL database

set -e

# Cloud SQL Configuration - set these environment variables before running
PROJECT_ID="${GCP_PROJECT_ID:?Error: Set GCP_PROJECT_ID environment variable}"
INSTANCE_NAME="${CLOUD_SQL_INSTANCE:-osint-db}"
DB_NAME="${DB_NAME:-stratint}"
DB_USER="${DB_USER:-postgres}"

echo "======================================"
echo "Applying Migration 018 to Production"
echo "======================================"
echo "Project: $PROJECT_ID"
echo "Instance: $INSTANCE_NAME"
echo "Database: $DB_NAME"
echo ""
echo "This will update enrichment prompts with source fidelity instructions."
echo ""

read -p "Continue? (y/n) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Cancelled"
    exit 0
fi

echo ""
echo "Retrieving database password from Secret Manager..."

# Get the database password from Secret Manager
DB_PASSWORD=$(gcloud secrets versions access latest --secret=db-password --project=$PROJECT_ID 2>/dev/null)

if [ -z "$DB_PASSWORD" ]; then
    echo "❌ Failed to retrieve database password from Secret Manager"
    echo "You can manually run the migration by connecting to Cloud SQL and running:"
    echo "  psql -U postgres -d $DB_NAME < migrations/018_add_source_fidelity_prompts.sql"
    exit 1
fi

echo "✓ Password retrieved"
echo ""
echo "Applying migration via Cloud SQL..."

# Create temporary password file for psql
PGPASSFILE=$(mktemp)
echo "34.29.23.163:5432:$DB_NAME:$DB_USER:$DB_PASSWORD" > "$PGPASSFILE"
chmod 600 "$PGPASSFILE"

# Get the public IP and apply migration
PUBLIC_IP=$(gcloud sql instances describe $INSTANCE_NAME --project=$PROJECT_ID --format="value(ipAddresses[0].ipAddress)")

echo "Whitelisting your IP and connecting..."

# Use gcloud sql connect with password file
PGPASSFILE="$PGPASSFILE" gcloud sql connect $INSTANCE_NAME \
    --project=$PROJECT_ID \
    --database=$DB_NAME \
    --user=$DB_USER \
    --quiet < migrations/018_add_source_fidelity_prompts.sql

# Clean up password file
rm -f "$PGPASSFILE"

echo ""
echo "✓ Migration 018 applied successfully to production!"
echo ""
