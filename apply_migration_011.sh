#!/bin/bash

# Apply migration 011 to add location text fields
# This script MUST be run before deploying the location fix

set -e

# Configure these for your environment
PROJECT_ID="${GCP_PROJECT_ID:?Error: Set GCP_PROJECT_ID environment variable}"
INSTANCE_NAME="${CLOUD_SQL_INSTANCE:-osint-db}"
DB_NAME="${DB_NAME:-stratint}"

echo "=================================="
echo "Applying Migration 011"
echo "=================================="
echo "This will add location_country, location_city, and location_region columns"
echo "to the events table."
echo ""
echo "Instance: $INSTANCE_NAME"
echo "Database: $DB_NAME"
echo ""

# Check if psql is installed
if ! command -v psql &> /dev/null; then
    echo "ERROR: psql is not installed"
    echo "Install it with: sudo apt-get install postgresql-client"
    exit 1
fi

# Check if proxy is running
if ! pgrep -f "cloud-sql-proxy.*osint-db" > /dev/null; then
    echo "ERROR: Cloud SQL Proxy is not running"
    echo ""
    echo "Start it with:"
    echo "  ./cloud-sql-proxy ${PROJECT_ID}:us-central1:${INSTANCE_NAME} --port 54321 &"
    echo ""
    echo "Or run: ./setup-db.sh"
    exit 1
fi

# Find the proxy port
PROXY_PORT=$(ps aux | grep "cloud-sql-proxy.*osint-db.*--port" | grep -v grep | sed -n 's/.*--port \([0-9]*\).*/\1/p' | head -1)

if [ -z "$PROXY_PORT" ]; then
    echo "ERROR: Could not determine Cloud SQL Proxy port"
    echo "Make sure the proxy is running with --port option"
    exit 1
fi

echo "Found Cloud SQL Proxy on port: $PROXY_PORT"
echo ""

# Prompt for password
echo "Enter database password (from Secret Manager: db-password):"
read -s DB_PASSWORD
echo ""

# Set password for psql
export PGPASSWORD="$DB_PASSWORD"

# Test connection
echo "Testing database connection..."
if ! psql -h localhost -p "$PROXY_PORT" -U postgres -d "$DB_NAME" -c '\conninfo' &> /dev/null; then
    echo "ERROR: Cannot connect to database"
    echo "Check that:"
    echo "  1. Cloud SQL Proxy is running"
    echo "  2. Password is correct"
    echo "  3. Port $PROXY_PORT is correct"
    exit 1
fi
echo "✓ Connected to database"
echo ""

# Apply migration
echo "Applying migration 011..."
if psql -h localhost -p "$PROXY_PORT" -U postgres -d "$DB_NAME" -f migrations/011_add_location_fields.sql; then
    echo ""
    echo "=================================="
    echo "✓ Migration 011 applied successfully!"
    echo "=================================="
    echo ""
    echo "You can now deploy the location fix code:"
    echo "  git revert HEAD  # Undo the revert of location fix"
    echo "  ./deploy-quick.sh"
    echo ""
else
    echo ""
    echo "ERROR: Migration failed"
    echo "Check the error message above"
    exit 1
fi
