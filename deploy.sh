#!/bin/bash

# OSINTMCP Cloud Run Deployment Script
# This script helps deploy OSINTMCP to Google Cloud Run manually

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration (can be overridden by environment variables)
PROJECT_ID="${GCP_PROJECT_ID:-}"
REGION="${GCP_REGION:-us-central1}"
SERVICE_NAME="${SERVICE_NAME:-stratint}"
IMAGE_NAME="gcr.io/${PROJECT_ID}/${SERVICE_NAME}"

# Cloud Run configuration
MEMORY="${MEMORY:-2Gi}"
CPU="${CPU:-2}"
TIMEOUT="${TIMEOUT:-300}"
MAX_INSTANCES="${MAX_INSTANCES:-10}"
MIN_INSTANCES="${MIN_INSTANCES:-0}"
CONCURRENCY="${CONCURRENCY:-80}"

# Database configuration
INSTANCE_CONNECTION_NAME="${INSTANCE_CONNECTION_NAME:-}"
DB_NAME="${DB_NAME:-stratint}"
DB_USER="${DB_USER:-stratint}"
DB_PASSWORD_SECRET="${DB_PASSWORD_SECRET:-db-password}"
JWT_SECRET_NAME="${JWT_SECRET_NAME:-admin-jwt-secret}"
ADMIN_PASSWORD_SECRET="${ADMIN_PASSWORD_SECRET:-admin-password}"

# Functions
print_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

check_prerequisites() {
    print_info "Checking prerequisites..."

    # Check if gcloud is installed
    if ! command -v gcloud &> /dev/null; then
        print_error "gcloud CLI is not installed. Please install it from https://cloud.google.com/sdk/docs/install"
        exit 1
    fi

    # Check if docker is installed
    if ! command -v docker &> /dev/null; then
        print_error "Docker is not installed. Please install it from https://docs.docker.com/get-docker/"
        exit 1
    fi

    print_info "Prerequisites check passed"
}

check_config() {
    print_info "Checking configuration..."

    if [ -z "$PROJECT_ID" ]; then
        print_error "GCP_PROJECT_ID is not set. Please set it as an environment variable or in this script."
        exit 1
    fi

    if [ -z "$INSTANCE_CONNECTION_NAME" ]; then
        print_warning "INSTANCE_CONNECTION_NAME is not set. Cloud SQL connection will not be configured."
        print_warning "Set it in the format: PROJECT_ID:REGION:INSTANCE_NAME"
    fi

    print_info "Configuration check passed"
}

build_image() {
    print_info "Building Docker image..."

    # Tag with both latest and timestamp
    TIMESTAMP=$(date +%Y%m%d%H%M%S)
    IMAGE_TAG="${IMAGE_NAME}:${TIMESTAMP}"
    IMAGE_LATEST="${IMAGE_NAME}:latest"

    docker build -t "${IMAGE_TAG}" -t "${IMAGE_LATEST}" .

    print_info "Image built: ${IMAGE_TAG}"
}

push_image() {
    print_info "Pushing image to Google Container Registry..."

    # Configure docker to use gcloud as a credential helper
    gcloud auth configure-docker --quiet

    docker push "${IMAGE_TAG}"
    docker push "${IMAGE_LATEST}"

    print_info "Image pushed successfully"
}

deploy_to_cloud_run() {
    print_info "Deploying to Cloud Run..."

    # Build the gcloud run deploy command
    DEPLOY_CMD="gcloud run deploy ${SERVICE_NAME} \
        --image=${IMAGE_TAG} \
        --region=${REGION} \
        --platform=managed \
        --allow-unauthenticated \
        --memory=${MEMORY} \
        --cpu=${CPU} \
        --timeout=${TIMEOUT} \
        --max-instances=${MAX_INSTANCES} \
        --min-instances=${MIN_INSTANCES} \
        --concurrency=${CONCURRENCY} \
        --project=${PROJECT_ID}"

    # Add Cloud SQL instance if configured
    if [ -n "$INSTANCE_CONNECTION_NAME" ]; then
        DEPLOY_CMD="${DEPLOY_CMD} --add-cloudsql-instances=${INSTANCE_CONNECTION_NAME}"
    fi

    # Add environment variables
    DEPLOY_CMD="${DEPLOY_CMD} \
        --set-env-vars=ENVIRONMENT=production,LOG_LEVEL=info,LOG_FORMAT=json"

    # Add Cloud SQL connection variables if configured
    if [ -n "$INSTANCE_CONNECTION_NAME" ]; then
        DEPLOY_CMD="${DEPLOY_CMD} \
            --set-env-vars=INSTANCE_CONNECTION_NAME=${INSTANCE_CONNECTION_NAME},DB_NAME=${DB_NAME},DB_USER=${DB_USER}"
    fi

    # Add secrets (passwords should be stored in Google Secret Manager)
    if [ -n "$INSTANCE_CONNECTION_NAME" ]; then
        DEPLOY_CMD="${DEPLOY_CMD} \
            --set-secrets=DB_PASSWORD=${DB_PASSWORD_SECRET}:latest,ADMIN_JWT_SECRET=${JWT_SECRET_NAME}:latest,ADMIN_PASSWORD=${ADMIN_PASSWORD_SECRET}:latest"
    fi

    # Execute deployment
    eval $DEPLOY_CMD

    print_info "Deployment completed successfully"
}

get_service_url() {
    print_info "Retrieving service URL..."

    SERVICE_URL=$(gcloud run services describe ${SERVICE_NAME} \
        --region=${REGION} \
        --project=${PROJECT_ID} \
        --format='value(status.url)')

    print_info "Service URL: ${SERVICE_URL}"
}

# Main deployment flow
main() {
    echo "======================================"
    echo "OSINTMCP Cloud Run Deployment"
    echo "======================================"
    echo ""

    check_prerequisites
    check_config

    # Confirm deployment
    echo ""
    echo "Deployment configuration:"
    echo "  Project: ${PROJECT_ID}"
    echo "  Region: ${REGION}"
    echo "  Service: ${SERVICE_NAME}"
    echo "  Image: ${IMAGE_NAME}"
    echo "  Memory: ${MEMORY}"
    echo "  CPU: ${CPU}"
    echo "  Cloud SQL: ${INSTANCE_CONNECTION_NAME:-Not configured}"
    echo ""

    read -p "Continue with deployment? (y/n) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        print_warning "Deployment cancelled"
        exit 0
    fi

    build_image
    push_image
    deploy_to_cloud_run
    get_service_url

    echo ""
    print_info "Deployment complete!"
    echo ""
}

# Run main function
main
