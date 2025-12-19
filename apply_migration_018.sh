#!/bin/bash
# Apply migration 018 - Add source fidelity instructions to prompts

set -e

# Get database connection info from environment or defaults
DB_HOST=${DB_HOST:-localhost}
DB_PORT=${DB_PORT:-5432}
DB_NAME=${DB_NAME:-stratint}
DB_USER=${DB_USER:-stratint}

echo "Applying migration 018: Add source fidelity prompts..."
echo "Database: $DB_NAME at $DB_HOST:$DB_PORT"
echo

# Apply migration
psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -f migrations/018_add_source_fidelity_prompts.sql

echo
echo "Migration 018 applied successfully!"
echo "The AI will now trust article content instead of 'correcting' it based on training data."
