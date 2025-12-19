#!/bin/bash

# Setup OSINTMCP database on Cloud SQL
# Creates the database if it doesn't exist and runs migrations

set -e

# Configure these for your environment
PROJECT_ID="${GCP_PROJECT_ID:?Error: Set GCP_PROJECT_ID environment variable}"
REGION="${GCP_REGION:-us-central1}"
INSTANCE_NAME="${CLOUD_SQL_INSTANCE:-osint-db}"
DB_NAME="${DB_NAME:-stratint}"
DB_USER="${DB_USER:-postgres}"

# Prompt for password if not set
if [ -z "$DB_PASSWORD" ]; then
  read -sp "Enter Cloud SQL database password: " DB_PASSWORD
  echo ""
fi

if [ -z "$DB_PASSWORD" ]; then
  echo "ERROR: Database password is required (set DB_PASSWORD or enter when prompted)"
  exit 1
fi

echo "=================================="
echo "OSINTMCP Database Setup"
echo "=================================="
echo "Instance: $INSTANCE_NAME"
echo "Database: $DB_NAME"
echo "User: $DB_USER"
echo ""

# Check if instance is ready
echo "Checking Cloud SQL instance status..."
STATE=$(gcloud sql instances describe $INSTANCE_NAME \
    --project=$PROJECT_ID \
    --format="value(state)" 2>/dev/null)

if [ "$STATE" != "RUNNABLE" ]; then
    echo "ERROR: Cloud SQL instance is not ready yet (state: $STATE)"
    echo ""
    echo "The instance is still being created. Please wait for it to finish."
    echo "Run this command to wait for it:"
    echo "  ./wait-for-db.sh"
    echo ""
    echo "Or check status manually:"
    echo "  gcloud sql instances describe $INSTANCE_NAME --project=$PROJECT_ID"
    exit 1
fi
echo "✓ Instance is RUNNABLE"
echo ""

# Check if psql is installed
if ! command -v psql &> /dev/null; then
    echo "PostgreSQL client (psql) not found. Installing..."
    sudo apt-get update -qq
    sudo apt-get install -y postgresql-client
    echo "✓ psql installed"
fi
echo ""

# Check if Cloud SQL Proxy is available
if ! command -v cloud-sql-proxy &> /dev/null; then
    echo "Cloud SQL Proxy not found. Downloading..."

    # Detect OS
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)

    if [ "$ARCH" = "x86_64" ]; then
        ARCH="amd64"
    elif [ "$ARCH" = "aarch64" ] || [ "$ARCH" = "arm64" ]; then
        ARCH="arm64"
    fi

    PROXY_URL="https://storage.googleapis.com/cloud-sql-connectors/cloud-sql-proxy/v2.13.0/cloud-sql-proxy.${OS}.${ARCH}"

    echo "Downloading from: $PROXY_URL"
    curl -o cloud-sql-proxy "$PROXY_URL"
    chmod +x cloud-sql-proxy
    PROXY_CMD="./cloud-sql-proxy"
else
    PROXY_CMD="cloud-sql-proxy"
fi

# Start Cloud SQL Proxy in background
echo ""
echo "Starting Cloud SQL Proxy..."
INSTANCE_CONNECTION_NAME="${PROJECT_ID}:${REGION}:${INSTANCE_NAME}"
$PROXY_CMD $INSTANCE_CONNECTION_NAME --port 5433 &
PROXY_PID=$!

# Wait for proxy to be ready
echo "Waiting for proxy to start..."
sleep 5

# Cleanup function
cleanup() {
    echo ""
    echo "Stopping Cloud SQL Proxy..."
    kill $PROXY_PID 2>/dev/null || true
}
trap cleanup EXIT

# Test connection
echo ""
echo "Testing connection to Cloud SQL..."
export PGPASSWORD="$DB_PASSWORD"

# Try to connect with verbose error output
echo "Attempting to connect with: psql -h localhost -p 5433 -U $DB_USER -d postgres"
if ! psql -h localhost -p 5433 -U $DB_USER -d postgres -c '\l' 2>&1; then
    echo ""
    echo "ERROR: Cannot connect to Cloud SQL instance"
    echo ""
    echo "Troubleshooting steps:"
    echo "  1. Verify the root password is correct"
    echo "  2. Check if 'postgres' user exists:"
    echo "     gcloud sql users list --instance=$INSTANCE_NAME --project=$PROJECT_ID"
    echo ""
    echo "Trying to list users via gcloud..."
    gcloud sql users list --instance=$INSTANCE_NAME --project=$PROJECT_ID
    exit 1
fi

echo "✓ Connected to Cloud SQL"

# Create database if it doesn't exist
echo ""
echo "Creating database '$DB_NAME' if it doesn't exist..."
psql -h localhost -p 5433 -U $DB_USER -d postgres -c "CREATE DATABASE $DB_NAME;" 2>/dev/null || echo "Database '$DB_NAME' already exists"

# Run migrations
echo ""
echo "Running migrations..."
if [ -d "migrations" ]; then
    for migration in migrations/*.sql; do
        if [ -f "$migration" ]; then
            echo "  Applying: $(basename $migration)"
            psql -h localhost -p 5433 -U $DB_USER -d $DB_NAME -f "$migration" || echo "    (migration may have already been applied)"
        fi
    done
    echo "✓ Migrations complete"
else
    echo "WARNING: No migrations directory found"
    echo "Create migrations manually or copy them to ./migrations/"
fi

# Verify database
echo ""
echo "Verifying database tables..."
psql -h localhost -p 5433 -U $DB_USER -d $DB_NAME -c '\dt'

echo ""
echo "=================================="
echo "✓ Database setup complete!"
echo "=================================="
echo ""
echo "Database: $DB_NAME"
echo "Connection: $INSTANCE_CONNECTION_NAME"
echo ""
echo "To connect manually:"
echo "  $PROXY_CMD $INSTANCE_CONNECTION_NAME --port 5433"
echo "  psql -h localhost -p 5433 -U $DB_USER -d $DB_NAME"
echo ""
echo "Next step: Run ./setup-secrets.sh and then ./deploy-quick.sh"
echo ""
