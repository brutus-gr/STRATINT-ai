#!/bin/bash

# Deploy MCP Server to Cloud Run
# Separate from main API server

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
export DB_NAME="${DB_NAME:-stratint}"
export DB_USER="${DB_USER:-postgres}"
export SERVICE_NAME="${SERVICE_NAME:-stratint-mcp}"

echo -e "${GREEN}OSINTMCP MCP Server Deployment${NC}"
echo "Project: $PROJECT_ID"
echo "Region: $REGION"
echo "Service: $SERVICE_NAME"
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
  echo -e "${YELLOW}Run ./setup-secrets.sh to create missing secrets${NC}"
  read -p "Press Enter to continue anyway, or Ctrl+C to exit..."
fi

# Build and push image
TIMESTAMP=$(date +%Y%m%d%H%M%S)
IMAGE_TAG="gcr.io/$PROJECT_ID/$SERVICE_NAME:$TIMESTAMP"
IMAGE_LATEST="gcr.io/$PROJECT_ID/$SERVICE_NAME:latest"

echo ""
echo -e "${GREEN}Building MCP server Docker image...${NC}"
docker build -f Dockerfile.mcp -t "$IMAGE_TAG" -t "$IMAGE_LATEST" .

echo ""
echo -e "${GREEN}Pushing to Container Registry...${NC}"
gcloud auth configure-docker --quiet
docker push "$IMAGE_TAG"
docker push "$IMAGE_LATEST"

echo ""
echo -e "${GREEN}Deploying MCP server to Cloud Run...${NC}"
gcloud run deploy $SERVICE_NAME \
  --image="$IMAGE_TAG" \
  --region=$REGION \
  --platform=managed \
  --allow-unauthenticated \
  --memory=1Gi \
  --cpu=1 \
  --timeout=300 \
  --max-instances=10 \
  --min-instances=0 \
  --concurrency=80 \
  --project=$PROJECT_ID \
  --add-cloudsql-instances=$INSTANCE_CONNECTION_NAME \
  --set-env-vars=ENVIRONMENT=production,LOG_LEVEL=info,LOG_FORMAT=json,INSTANCE_CONNECTION_NAME=$INSTANCE_CONNECTION_NAME,DB_NAME=$DB_NAME,DB_USER=$DB_USER \
  --set-secrets=DB_PASSWORD=db-password:latest,ADMIN_JWT_SECRET=admin-jwt-secret:latest,ADMIN_PASSWORD=admin-password:latest

echo ""
echo -e "${GREEN}Getting service URL...${NC}"
SERVICE_URL=$(gcloud run services describe $SERVICE_NAME \
  --region=$REGION \
  --project=$PROJECT_ID \
  --format='value(status.url)')

echo ""
echo -e "${GREEN}✓ MCP Server deployment complete!${NC}"
echo ""
echo "MCP Server URL: $SERVICE_URL"
echo "Health Check: $SERVICE_URL/healthz"
echo ""
echo "Test it with: curl -X POST $SERVICE_URL -H 'Content-Type: application/json' -d '{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"tools/list\"}'"
echo ""
echo "Add to Claude Desktop config:"
echo "{
  \"mcpServers\": {
    \"stratint\": {
      \"url\": \"$SERVICE_URL\",
      \"transport\": \"http\"
    }
  }
}"
echo ""
