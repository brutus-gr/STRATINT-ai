# Quick Start - Deploy to Cloud Run (Simplified)

Since you're getting auth issues with Cloud SQL Proxy locally, let's use a simpler approach:

## 1. Create the database (no migrations needed yet)

```bash
./setup-db-simple.sh
```

This just creates the empty database using gcloud commands.

## 2. Setup secrets

```bash
./setup-secrets.sh
```

When prompted:
- Database password: Press Enter (uses your root password)
- Admin password: Enter a password for the admin panel

## 3. Deploy to Cloud Run

```bash
./deploy-quick.sh
```

This will deploy your app to Cloud Run.

## 4. Initialize database from Cloud Run

Once deployed, you have two options:

### Option A: Let the app auto-create schema
Your Go application can create the schema on first startup if it has migration logic built-in.

### Option B: Run migrations from Cloud Run
```bash
# Get a shell in your Cloud Run instance
gcloud run services proxy stratint --project=YOUR_PROJECT_ID --region=us-central1

# Or run a one-off job to apply migrations
gcloud run jobs create migrate-db \
  --image=gcr.io/YOUR_PROJECT_ID/stratint:latest \
  --region=us-central1 \
  --project=YOUR_PROJECT_ID \
  --set-cloudsql-instances=YOUR_PROJECT_ID:us-central1:osint-db \
  --set-env-vars=INSTANCE_CONNECTION_NAME=YOUR_PROJECT_ID:us-central1:osint-db,DB_NAME=stratint,DB_USER=postgres \
  --set-secrets=DB_PASSWORD=db-password:latest \
  --command=psql \
  --args="-h /cloudsql/YOUR_PROJECT_ID:us-central1:osint-db -U postgres -d stratint -f /app/migrations/schema.sql"
```

### Option C: Manual SQL via Cloud Console
1. Go to: https://console.cloud.google.com/sql/instances/osint-db?project=YOUR_PROJECT_ID
2. Click "DATABASES" → verify "stratint" exists
3. Click "QUERY" → paste your SQL schema
4. Execute

## Why This Is Easier

- **No local psql needed**
- **No auth issues** with Cloud SQL Proxy
- **Cloud Run has built-in auth** to Cloud SQL
- **Simpler workflow** for BETA deployment

The local Cloud SQL Proxy is useful for development, but for initial deployment, this approach is much faster.
