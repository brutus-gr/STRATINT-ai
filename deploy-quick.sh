#!/bin/bash

# Quick Deploy Script for OSINTMCP to Cloud Run
# Pre-configured for your Cloud SQL instance

set -e

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Configuration - set these environment variables before running
export PROJECT_ID="${GCP_PROJECT_ID:?Error: Set GCP_PROJECT_ID environment variable}"
export REGION="${GCP_REGION:-us-central1}"
export INSTANCE_NAME="${CLOUD_SQL_INSTANCE:-osint-db}"
export INSTANCE_CONNECTION_NAME="${PROJECT_ID}:${REGION}:${INSTANCE_NAME}"
export DB_NAME="${DB_NAME:-osintmcp}"
export DB_USER="${DB_USER:-postgres}"
export SERVICE_NAME="${SERVICE_NAME:-osintmcp}"

echo -e "${GREEN}OSINTMCP Quick Deploy${NC}"
echo "Project: $PROJECT_ID"
echo "Region: $REGION"
echo "Cloud SQL: $INSTANCE_CONNECTION_NAME"
echo ""

# Check if secrets exist
echo -e "${YELLOW}Checking secrets...${NC}"
SECRETS_EXIST=true

for SECRET in db-password admin-jwt-secret admin-password; do
  if ! gcloud secrets describe $SECRET --project=$PROJECT_ID &>/dev/null; then
    echo -e "${YELLOW}Warning: Secret '$SECRET' not found${NC}"
    SECRETS_EXIST=false
  else
    echo "✓ Secret '$SECRET' exists"
  fi
done

if [ "$SECRETS_EXIST" = false ]; then
  echo ""
  echo -e "${YELLOW}Some secrets are missing. Create them with:${NC}"
  echo ""
  echo "# Database password"
  echo "echo -n 'YOUR_DB_PASSWORD' | gcloud secrets create db-password --project=$PROJECT_ID --data-file=-"
  echo ""
  echo "# Admin JWT secret (random)"
  echo "echo -n \"\$(openssl rand -base64 32)\" | gcloud secrets create admin-jwt-secret --project=$PROJECT_ID --data-file=-"
  echo ""
  echo "# Admin panel password"
  echo "echo -n 'YOUR_ADMIN_PASSWORD' | gcloud secrets create admin-password --project=$PROJECT_ID --data-file=-"
  echo ""
  read -p "Press Enter to continue anyway, or Ctrl+C to exit and create secrets first..."
fi

# Build and push image
TIMESTAMP=$(date +%Y%m%d%H%M%S)
IMAGE_TAG="gcr.io/$PROJECT_ID/$SERVICE_NAME:$TIMESTAMP"
IMAGE_LATEST="gcr.io/$PROJECT_ID/$SERVICE_NAME:latest"

echo ""
echo -e "${GREEN}Building Docker image...${NC}"
docker build -t "$IMAGE_TAG" -t "$IMAGE_LATEST" .

echo ""
echo -e "${GREEN}Pushing to Container Registry...${NC}"
gcloud auth configure-docker --quiet
docker push "$IMAGE_TAG"
docker push "$IMAGE_LATEST"

echo ""
echo -e "${GREEN}Deploying to Cloud Run...${NC}"
gcloud run deploy $SERVICE_NAME \
  --image="$IMAGE_TAG" \
  --region=$REGION \
  --platform=managed \
  --allow-unauthenticated \
  --memory=2Gi \
  --cpu=2 \
  --timeout=300 \
  --max-instances=10 \
  --min-instances=0 \
  --concurrency=80 \
  --project=$PROJECT_ID \
  --add-cloudsql-instances=$INSTANCE_CONNECTION_NAME \
  --set-env-vars=ENVIRONMENT=production,LOG_LEVEL=debug,LOG_FORMAT=json,INSTANCE_CONNECTION_NAME=$INSTANCE_CONNECTION_NAME,DB_NAME=$DB_NAME,DB_USER=$DB_USER,ADMIN_ENABLED=true \
  --set-secrets=DB_PASSWORD=db-password:latest,ADMIN_JWT_SECRET=admin-jwt-secret:latest,ADMIN_PASSWORD=admin-password:latest \
  --no-cpu-throttling \
  --cpu-boost

echo ""
echo -e "${GREEN}Getting service URL...${NC}"
SERVICE_URL=$(gcloud run services describe $SERVICE_NAME \
  --region=$REGION \
  --project=$PROJECT_ID \
  --format='value(status.url)')

echo ""
echo -e "${GREEN}✓ Deployment complete!${NC}"
echo ""
echo "Service URL: $SERVICE_URL"
echo "Admin Panel: $SERVICE_URL/admin"
echo "Health Check: $SERVICE_URL/healthz"
echo ""
echo "Test it with: curl $SERVICE_URL/healthz"
echo ""
