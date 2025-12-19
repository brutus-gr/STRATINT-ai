#!/bin/bash

# Setup Google Secret Manager secrets for OSINTMCP
# Run this ONCE before deploying

set -e

# Configure these for your environment
PROJECT_ID="${GCP_PROJECT_ID:?Error: Set GCP_PROJECT_ID environment variable}"
REGION="${GCP_REGION:-us-central1}"

echo "=================================="
echo "OSINTMCP Secrets Setup"
echo "=================================="
echo "Project: $PROJECT_ID"
echo ""

# Get project number for service account
PROJECT_NUMBER=$(gcloud projects describe $PROJECT_ID --format="value(projectNumber)")
SERVICE_ACCOUNT="${PROJECT_NUMBER}-compute@developer.gserviceaccount.com"

echo "Service Account: $SERVICE_ACCOUNT"
echo ""

# Database password
echo "1. Setting up database password..."
echo "Using root user 'postgres' for Cloud SQL"
read -sp "Enter your Cloud SQL root password: " DB_PASSWORD
echo ""
if [ -z "$DB_PASSWORD" ]; then
  echo "ERROR: Database password is required"
  exit 1
fi
echo -n "$DB_PASSWORD" | gcloud secrets create db-password \
  --project=$PROJECT_ID \
  --data-file=- \
  --replication-policy="automatic" || echo "Secret 'db-password' may already exist. Use 'gcloud secrets versions add' to update."

# Admin JWT Secret (auto-generate)
echo ""
echo "2. Generating admin JWT secret (random)..."
JWT_SECRET=$(openssl rand -base64 32)
echo -n "$JWT_SECRET" | gcloud secrets create admin-jwt-secret \
  --project=$PROJECT_ID \
  --data-file=- \
  --replication-policy="automatic" || echo "Secret 'admin-jwt-secret' may already exist."

# Admin password
echo ""
echo "3. Setting up admin panel password..."
read -sp "Enter password for admin panel login: " ADMIN_PASSWORD
echo ""
echo -n "$ADMIN_PASSWORD" | gcloud secrets create admin-password \
  --project=$PROJECT_ID \
  --data-file=- \
  --replication-policy="automatic" || echo "Secret 'admin-password' may already exist."

# Grant permissions
echo ""
echo "4. Granting Cloud Run service account access to secrets..."
for SECRET in db-password admin-jwt-secret admin-password; do
  echo "  Granting access to: $SECRET"
  gcloud secrets add-iam-policy-binding $SECRET \
    --project=$PROJECT_ID \
    --member="serviceAccount:${SERVICE_ACCOUNT}" \
    --role="roles/secretmanager.secretAccessor" 2>/dev/null || true
done

echo ""
echo "=================================="
echo "✓ Secrets setup complete!"
echo "=================================="
echo ""
echo "Created secrets:"
echo "  - db-password"
echo "  - admin-jwt-secret"
echo "  - admin-password"
echo ""
echo "Next step: Run ./deploy-quick.sh to deploy your application"
echo ""
