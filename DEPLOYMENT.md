# STRATINT - Google Cloud Run Deployment Guide

This guide walks you through deploying STRATINT to Google Cloud Run for your BETA environment.

## Architecture Overview

For this BETA deployment:
- **Application Container**: Single Cloud Run service containing both backend and frontend
- **Database**: Google Cloud SQL (PostgreSQL)
- **Secrets**: Google Secret Manager
- **Container Registry**: Google Container Registry (GCR)

## Prerequisites

1. **Google Cloud Account** with billing enabled
2. **Google Cloud CLI** (`gcloud`) installed and configured
   ```bash
   # Install gcloud CLI
   # https://cloud.google.com/sdk/docs/install

   # Login and set project
   gcloud auth login
   gcloud config set project YOUR_PROJECT_ID
   ```

3. **Docker** installed locally
   ```bash
   docker --version
   ```

4. **Enable required Google Cloud APIs**:
   ```bash
   gcloud services enable \
     run.googleapis.com \
     sqladmin.googleapis.com \
     secretmanager.googleapis.com \
     cloudbuild.googleapis.com \
     containerregistry.googleapis.com
   ```

## Step 1: Create Cloud SQL Instance

Create a PostgreSQL instance for your database:

```bash
# Set your variables
export PROJECT_ID="your-project-id"
export REGION="us-central1"
export DB_INSTANCE_NAME="stratint-db"
export DB_NAME="stratint"
export DB_USER="stratint"

# Create Cloud SQL instance
gcloud sql instances create $DB_INSTANCE_NAME \
  --database-version=POSTGRES_15 \
  --tier=db-f1-micro \
  --region=$REGION \
  --root-password=CHANGE_THIS_ROOT_PASSWORD \
  --database-flags=max_connections=100

# Create database
gcloud sql databases create $DB_NAME \
  --instance=$DB_INSTANCE_NAME

# Create user
gcloud sql users create $DB_USER \
  --instance=$DB_INSTANCE_NAME \
  --password=CHANGE_THIS_PASSWORD
```

**Important**: Save the password securely. For production, use a strong random password.

### Run Database Migrations

You'll need to run the database migrations. You can do this via Cloud SQL Proxy:

```bash
# Download Cloud SQL Proxy
wget https://dl.google.com/cloudsql/cloud_sql_proxy.linux.amd64 -O cloud_sql_proxy
chmod +x cloud_sql_proxy

# Get your instance connection name
export INSTANCE_CONNECTION_NAME="${PROJECT_ID}:${REGION}:${DB_INSTANCE_NAME}"

# Start the proxy (in a separate terminal)
./cloud_sql_proxy -instances=${INSTANCE_CONNECTION_NAME}=tcp:5432

# Run migrations (from another terminal)
# You'll need to install a PostgreSQL migration tool or use psql
export DATABASE_URL="postgresql://${DB_USER}:YOUR_PASSWORD@localhost:5432/${DB_NAME}?sslmode=disable"

# Apply your migrations from the migrations/ directory
psql $DATABASE_URL -f migrations/001_initial_schema.sql
# Continue with other migration files...
```

## Step 2: Store Secrets in Secret Manager

Store sensitive data in Google Secret Manager:

```bash
# Create secrets
echo -n "YOUR_DB_PASSWORD" | gcloud secrets create db-password --data-file=-
echo -n "$(openssl rand -base64 32)" | gcloud secrets create admin-jwt-secret --data-file=-
echo -n "YOUR_ADMIN_PASSWORD" | gcloud secrets create admin-password --data-file=-

# Grant Cloud Run service account access to secrets
PROJECT_NUMBER=$(gcloud projects describe $PROJECT_ID --format="value(projectNumber)")
SERVICE_ACCOUNT="${PROJECT_NUMBER}-compute@developer.gserviceaccount.com"

for SECRET in db-password admin-jwt-secret admin-password; do
  gcloud secrets add-iam-policy-binding $SECRET \
    --member="serviceAccount:${SERVICE_ACCOUNT}" \
    --role="roles/secretmanager.secretAccessor"
done
```

## Step 3: Deploy to Cloud Run

### Option A: Manual Deployment (using deploy.sh)

1. Set environment variables:
   ```bash
   export GCP_PROJECT_ID="your-project-id"
   export GCP_REGION="us-central1"
   export INSTANCE_CONNECTION_NAME="${PROJECT_ID}:${REGION}:${DB_INSTANCE_NAME}"
   export DB_NAME="stratint"
   export DB_USER="stratint"
   ```

2. Run the deployment script:
   ```bash
   ./deploy.sh
   ```

### Option B: Manual Deployment (step by step)

1. **Build the Docker image**:
   ```bash
   docker build -t gcr.io/$PROJECT_ID/stratint:latest .
   ```

2. **Push to Container Registry**:
   ```bash
   gcloud auth configure-docker
   docker push gcr.io/$PROJECT_ID/stratint:latest
   ```

3. **Deploy to Cloud Run**:
   ```bash
   gcloud run deploy stratint \
     --image=gcr.io/$PROJECT_ID/stratint:latest \
     --region=$REGION \
     --platform=managed \
     --allow-unauthenticated \
     --memory=2Gi \
     --cpu=2 \
     --timeout=300 \
     --max-instances=10 \
     --min-instances=0 \
     --concurrency=80 \
     --add-cloudsql-instances=$INSTANCE_CONNECTION_NAME \
     --set-env-vars=ENVIRONMENT=production,LOG_LEVEL=info,LOG_FORMAT=json \
     --set-env-vars=INSTANCE_CONNECTION_NAME=$INSTANCE_CONNECTION_NAME \
     --set-env-vars=DB_NAME=$DB_NAME \
     --set-env-vars=DB_USER=$DB_USER \
     --set-secrets=DB_PASSWORD=db-password:latest \
     --set-secrets=ADMIN_JWT_SECRET=admin-jwt-secret:latest \
     --set-secrets=ADMIN_PASSWORD=admin-password:latest
   ```

### Option C: Automated Deployment (using Cloud Build)

For continuous deployment from a Git repository:

1. **Connect your repository** to Cloud Build:
   - Go to Cloud Build > Triggers in the Google Cloud Console
   - Click "Connect Repository"
   - Follow the steps to connect your GitHub/GitLab/Bitbucket repo

2. **Create a build trigger**:
   ```bash
   gcloud builds triggers create github \
     --repo-name=stratint \
     --repo-owner=YOUR_GITHUB_USERNAME \
     --branch-pattern="^main$" \
     --build-config=cloudbuild.yaml \
     --substitutions=_REGION=$REGION,_INSTANCE_CONNECTION_NAME=$INSTANCE_CONNECTION_NAME
   ```

3. **Push to your repository**:
   ```bash
   git push origin main
   ```

   Cloud Build will automatically build and deploy your application.

## Step 4: Configure Your Application

After deployment, you need to configure your application through the admin panel:

1. Get your Cloud Run service URL:
   ```bash
   gcloud run services describe stratint \
     --region=$REGION \
     --format='value(status.url)'
   ```

2. Access the admin panel:
   ```
   https://your-service-url.run.app/admin
   ```

3. Login with your admin credentials (set in Secret Manager)

4. Configure:
   - OpenAI API settings (if using AI enrichment)
   - Connector configurations (Twitter, RSS, etc.)
   - Scraper settings
   - Thresholds

## Step 5: Verify Deployment

1. **Check service health**:
   ```bash
   curl https://your-service-url.run.app/healthz
   ```

   Expected response:
   ```json
   {"status":"ok"}
   ```

2. **Check logs**:
   ```bash
   gcloud run services logs read stratint --region=$REGION --limit=50
   ```

3. **Test the application**:
   - Visit the web interface: `https://your-service-url.run.app`
   - Check metrics: `https://your-service-url.run.app/metrics`

## Monitoring and Maintenance

### View Logs

```bash
# Stream logs in real-time
gcloud run services logs tail stratint --region=$REGION

# View recent logs
gcloud run services logs read stratint --region=$REGION --limit=100
```

### Monitor Resources

Visit the Cloud Run console:
```
https://console.cloud.google.com/run/detail/$REGION/stratint/metrics
```

### Update the Application

To deploy a new version:

```bash
# Build and push new image
docker build -t gcr.io/$PROJECT_ID/stratint:v2 .
docker push gcr.io/$PROJECT_ID/stratint:v2

# Deploy update
gcloud run services update stratint \
  --image=gcr.io/$PROJECT_ID/stratint:v2 \
  --region=$REGION
```

Or simply run `./deploy.sh` again.

### Scale Configuration

Adjust scaling settings:

```bash
gcloud run services update stratint \
  --region=$REGION \
  --min-instances=1 \
  --max-instances=20
```

## Cost Optimization for BETA

For a BETA deployment, consider:

1. **Cloud SQL**: Use `db-f1-micro` (cheapest tier)
2. **Cloud Run**:
   - Set `--min-instances=0` (scale to zero when not in use)
   - Set `--max-instances=5` (limit concurrent instances)
3. **Enable Cloud SQL automatic backups** but reduce retention period

## Troubleshooting

### Container fails to start

Check logs:
```bash
gcloud run services logs read stratint --region=$REGION --limit=50
```

Common issues:
- Database connection failed: Check `INSTANCE_CONNECTION_NAME` format
- Secrets not accessible: Verify Secret Manager permissions
- Migration not run: Ensure database schema is initialized

### Database connection issues

1. Verify Cloud SQL instance is running:
   ```bash
   gcloud sql instances describe $DB_INSTANCE_NAME
   ```

2. Check if Cloud Run has permission to connect:
   ```bash
   gcloud projects get-iam-policy $PROJECT_ID \
     --flatten="bindings[].members" \
     --filter="bindings.members:${SERVICE_ACCOUNT}"
   ```

### High costs

1. Check Cloud Run metrics to see if traffic is higher than expected
2. Reduce `--max-instances` if needed
3. Consider setting up budget alerts:
   ```bash
   gcloud billing budgets create \
     --billing-account=YOUR_BILLING_ACCOUNT \
     --display-name="STRATINT Budget Alert" \
     --budget-amount=50USD
   ```

## Security Considerations

1. **Always use Secret Manager** for passwords and API keys
2. **Restrict access** to Cloud Run service:
   ```bash
   # Remove public access
   gcloud run services remove-iam-policy-binding stratint \
     --region=$REGION \
     --member="allUsers" \
     --role="roles/run.invoker"
   ```

3. **Enable Cloud Armor** for DDoS protection (if needed)
4. **Regular backups** of Cloud SQL database
5. **Enable Cloud SQL SSL** for connections

## Next Steps

- Set up Cloud Monitoring alerts
- Configure custom domain with Cloud Run
- Set up Cloud CDN for static assets
- Implement Cloud Armor security policies
- Configure automated backups and disaster recovery

## Support

For issues or questions:
- Check logs first: `gcloud run services logs read stratint --region=$REGION`
- Review Cloud Run documentation: https://cloud.google.com/run/docs
- Check Cloud SQL documentation: https://cloud.google.com/sql/docs
