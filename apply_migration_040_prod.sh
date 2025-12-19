#!/bin/bash
# Apply migration 040 to production Cloud SQL database

set -e

# Cloud SQL Configuration - set these environment variables before running
PROJECT_ID="${GCP_PROJECT_ID:?Error: Set GCP_PROJECT_ID environment variable}"
INSTANCE_NAME="${CLOUD_SQL_INSTANCE:-osint-db}"
DB_NAME="${DB_NAME:-stratint}"
DB_USER="${DB_USER:-postgres}"

echo "======================================"
echo "Applying Migration 040 to Production"
echo "======================================"
echo "Project: $PROJECT_ID"
echo "Instance: $INSTANCE_NAME"
echo "Database: $DB_NAME"
echo ""
echo "This will simplify forecasts to value-based predictions."
echo "Changes:"
echo "  - Drop parent_forecasts table"
echo "  - Remove threshold fields (threshold_percent, threshold_direction, threshold_operator)"
echo "  - Add prediction_type, units, and target_date columns"
echo "  - Update forecast_model_responses to use percentile_predictions JSONB"
echo "  - Update forecast_results to use aggregated_percentiles JSONB"
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
    echo "  psql -U postgres -d $DB_NAME < migrations/040_simplify_forecasts_to_value_based.sql"
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
    --quiet < migrations/040_simplify_forecasts_to_value_based.sql

# Clean up password file
rm -f "$PGPASSFILE"

echo ""
echo "✓ Migration 040 applied successfully to production!"
echo ""
echo "Forecast model changes:"
echo "  - Forecasts now support value-based predictions (percentile or point_estimate)"
echo "  - Parent forecast complexity has been removed"
echo "  - Model responses now use percentile_predictions JSONB"
echo "  - Forecast results now use aggregated_percentiles JSONB"
echo ""
