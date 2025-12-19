# Google Cloud Deployment Architecture

## Overview

STRATINT will be deployed on Google Cloud Platform using Cloud Run for compute, Cloud SQL for PostgreSQL, and associated services for a fully managed, scalable architecture.

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                      Internet / Users                       │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│                  Cloud Load Balancer                        │
│                  (Global HTTPS LB)                          │
└───────────┬─────────────────────────────────┬───────────────┘
            │                                 │
            ▼                                 ▼
┌───────────────────────┐         ┌───────────────────────┐
│   Cloud Run (API)     │         │  Cloud Run (Admin)    │
│   - MCP Server        │         │  - Admin Panel        │
│   - Event API         │         │  - Management API     │
│   - Autoscale 0-100   │         │  - Autoscale 0-10     │
└───────────┬───────────┘         └───────────┬───────────┘
            │                                 │
            └─────────────┬───────────────────┘
                          │
            ┌─────────────┴─────────────┐
            │                           │
            ▼                           ▼
┌───────────────────────┐   ┌───────────────────────┐
│   Cloud SQL           │   │   Redis (Memorystore) │
│   (PostgreSQL 15)     │   │   - Cache Layer       │
│   - Private IP        │   │   - Session Store     │
│   - HA Config         │   │   - Rate Limiting     │
│   - Auto Backups      │   │                       │
└───────────┬───────────┘   └───────────────────────┘
            │
            ▼
┌─────────────────────────────────────────────────────────────┐
│                     Supporting Services                      │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐     │
│  │ Secret Mgr   │  │ Cloud        │  │ Cloud        │     │
│  │ - API Keys   │  │ Storage      │  │ Logging      │     │
│  │ - DB Creds   │  │ - Backups    │  │ - Structured │     │
│  └──────────────┘  │ - Archives   │  │ - Real-time  │     │
│                    └──────────────┘  └──────────────┘     │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐     │
│  │ Cloud        │  │ Cloud        │  │ Cloud        │     │
│  │ Monitoring   │  │ Build        │  │ Scheduler    │     │
│  │ - Metrics    │  │ - CI/CD      │  │ - Cron Jobs  │     │
│  │ - Alerts     │  │ - Auto Build │  │ - Cleanup    │     │
│  └──────────────┘  └──────────────┘  └──────────────┘     │
└─────────────────────────────────────────────────────────────┘
```

## Service Configuration

### 1. Cloud Run (API Service)

**Service Name:** `stratint-api`

**Configuration:**
```yaml
apiVersion: serving.knative.dev/v1
kind: Service
metadata:
  name: stratint-api
  labels:
    environment: production
spec:
  template:
    metadata:
      annotations:
        autoscaling.knative.dev/minScale: "1"
        autoscaling.knative.dev/maxScale: "100"
        run.googleapis.com/cpu-throttling: "false"
        run.googleapis.com/vpc-access-connector: stratint-connector
        run.googleapis.com/vpc-access-egress: private-ranges-only
    spec:
      containerConcurrency: 80
      timeoutSeconds: 300
      containers:
      - image: gcr.io/PROJECT_ID/stratint-api:latest
        ports:
        - containerPort: 8080
        resources:
          limits:
            memory: 2Gi
            cpu: "2"
        env:
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: database-url
              key: url
        - name: REDIS_URL
          valueFrom:
            secretKeyRef:
              name: redis-url
              key: url
        - name: OPENAI_API_KEY
          valueFrom:
            secretKeyRef:
              name: openai-key
              key: api_key
```

**Environment Variables:**
- `PORT=8080`
- `DATABASE_URL` (from Secret Manager)
- `REDIS_URL` (from Secret Manager)
- `OPENAI_API_KEY` (from Secret Manager)
- `LOG_LEVEL=info`
- `LOG_FORMAT=json`
- `ENVIRONMENT=production`

**Resource Limits:**
- Memory: 2 GiB per instance
- CPU: 2 vCPU per instance
- Max instances: 100 (adjustable)
- Min instances: 1 (always warm)

**Networking:**
- VPC Connector for private Cloud SQL access
- Egress: Private ranges only (to Cloud SQL)
- Ingress: All traffic (behind Load Balancer)

### 2. Cloud Run (Admin Panel)

**Service Name:** `stratint-admin`

**Configuration:**
```yaml
apiVersion: serving.knative.dev/v1
kind: Service
metadata:
  name: stratint-admin
spec:
  template:
    metadata:
      annotations:
        autoscaling.knative.dev/minScale: "0"
        autoscaling.knative.dev/maxScale: "10"
        run.googleapis.com/vpc-access-connector: stratint-connector
    spec:
      containers:
      - image: gcr.io/PROJECT_ID/stratint-admin:latest
        resources:
          limits:
            memory: 1Gi
            cpu: "1"
```

**Features:**
- Scale to zero when idle
- Lower resource allocation (admin traffic is lower)
- Same VPC connector for database access

### 3. Cloud SQL (PostgreSQL)

**Instance Name:** `stratint-db`

**Configuration:**
- **Edition:** Enterprise
- **Version:** PostgreSQL 15
- **Tier:** db-custom-4-16384 (4 vCPU, 16 GB RAM)
- **Storage:** 100 GB SSD (auto-increase enabled)
- **High Availability:** Regional HA (automatic failover)
- **Backup:** Daily automated backups, 7-day retention
- **Point-in-time Recovery:** Enabled

**Connection:**
- Private IP only (no public IP)
- VPC peering with Cloud Run services
- SSL/TLS required
- Connection pooling via PgBouncer

**Database Schema:**
```sql
-- Main tables
CREATE TABLE events (
    id VARCHAR(64) PRIMARY KEY,
    timestamp TIMESTAMPTZ NOT NULL,
    title TEXT NOT NULL,
    summary TEXT NOT NULL,
    raw_content TEXT,
    magnitude DECIMAL(3,1) CHECK (magnitude >= 0 AND magnitude <= 10),
    confidence JSONB NOT NULL,
    category VARCHAR(32) NOT NULL,
    status VARCHAR(32) NOT NULL,
    location GEOGRAPHY(POINT),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE sources (
    id VARCHAR(64) PRIMARY KEY,
    type VARCHAR(32) NOT NULL,
    url TEXT,
    author VARCHAR(255),
    published_at TIMESTAMPTZ NOT NULL,
    retrieved_at TIMESTAMPTZ NOT NULL,
    raw_content TEXT NOT NULL,
    credibility DECIMAL(3,2),
    metadata JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE entities (
    id VARCHAR(64) PRIMARY KEY,
    type VARCHAR(32) NOT NULL,
    name VARCHAR(255) NOT NULL,
    normalized_name VARCHAR(255) NOT NULL,
    confidence DECIMAL(3,2),
    attributes JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE event_sources (
    event_id VARCHAR(64) REFERENCES events(id) ON DELETE CASCADE,
    source_id VARCHAR(64) REFERENCES sources(id) ON DELETE CASCADE,
    PRIMARY KEY (event_id, source_id)
);

CREATE TABLE event_entities (
    event_id VARCHAR(64) REFERENCES events(id) ON DELETE CASCADE,
    entity_id VARCHAR(64) REFERENCES entities(id) ON DELETE CASCADE,
    PRIMARY KEY (event_id, entity_id)
);

-- Indexes for query performance
CREATE INDEX idx_events_timestamp ON events(timestamp DESC);
CREATE INDEX idx_events_magnitude ON events(magnitude DESC);
CREATE INDEX idx_events_category ON events(category);
CREATE INDEX idx_events_status ON events(status);
CREATE INDEX idx_events_composite ON events(category, magnitude, timestamp DESC);
CREATE INDEX idx_events_location ON events USING GIST(location);
CREATE INDEX idx_sources_type ON sources(type);
CREATE INDEX idx_sources_published ON sources(published_at DESC);
CREATE INDEX idx_entities_type ON entities(type);
CREATE INDEX idx_entities_normalized ON entities(normalized_name);

-- Full-text search
CREATE INDEX idx_events_search ON events USING GIN(
    to_tsvector('english', title || ' ' || summary)
);
```

### 4. Redis (Memorystore)

**Instance Name:** `stratint-cache`

**Configuration:**
- **Tier:** Standard (HA with automatic failover)
- **Version:** Redis 7.0
- **Capacity:** 5 GB
- **Network:** Same VPC as Cloud Run
- **Eviction Policy:** allkeys-lru

**Use Cases:**
- Event query result caching (5-min TTL)
- Rate limiting counters
- Session storage for admin panel
- Deduplication bloom filters
- Real-time metrics aggregation

**Cache Keys:**
```
events:query:{hash}:{page}:{limit}  -> EventResponse JSON
rate_limit:ip:{ip}                  -> Request counter
session:{session_id}                -> User session data
dedup:fingerprint:{hash}            -> Source fingerprint
metrics:connector:{name}            -> Connector metrics
```

### 5. Cloud Storage

**Buckets:**

**1. Backups:** `stratint-backups`
- Database exports (weekly)
- Configuration snapshots
- Lifecycle: Delete after 90 days

**2. Archives:** `stratint-archives`
- Historical events (>1 year old)
- Cold storage class
- Lifecycle: Delete after 7 years

**3. Logs:** `stratint-logs`
- Application logs (if needed beyond Cloud Logging)
- Retention: 30 days

**4. Assets:** `stratint-assets`
- Static admin panel assets
- CDN-enabled

### 6. Secret Manager

**Secrets:**
- `database-url` - PostgreSQL connection string
- `redis-url` - Redis connection URL
- `openai-api-key` - OpenAI API key
- `twitter-api-key` - Twitter API credentials
- `telegram-bot-token` - Telegram bot token
- `reddit-client-secret` - Reddit OAuth secret
- `admin-jwt-secret` - JWT signing key
- `session-secret` - Session encryption key

**Access Control:**
- Cloud Run service accounts have read-only access
- Secrets rotated quarterly
- Audit logging enabled

### 7. Cloud Build (CI/CD)

**Build Triggers:**

**API Build:** `cloudbuild-api.yaml`
```yaml
steps:
  # Run tests
  - name: 'golang:1.24'
    args: ['go', 'test', './...']
  
  # Build binary
  - name: 'golang:1.24'
    args: ['go', 'build', '-o', 'stratint-server', './cmd/server']
  
  # Build Docker image
  - name: 'gcr.io/cloud-builders/docker'
    args: ['build', '-t', 'gcr.io/$PROJECT_ID/stratint-api:$COMMIT_SHA', '.']
  
  # Push to Container Registry
  - name: 'gcr.io/cloud-builders/docker'
    args: ['push', 'gcr.io/$PROJECT_ID/stratint-api:$COMMIT_SHA']
  
  # Deploy to Cloud Run
  - name: 'gcr.io/google.com/cloudsdktool/cloud-sdk'
    entrypoint: gcloud
    args:
      - 'run'
      - 'deploy'
      - 'stratint-api'
      - '--image'
      - 'gcr.io/$PROJECT_ID/stratint-api:$COMMIT_SHA'
      - '--region'
      - 'us-central1'
      - '--platform'
      - 'managed'
```

**Triggers:**
- Push to `main` branch: Deploy to production
- Pull requests: Run tests only
- Tagged releases: Deploy with version tag

### 8. Cloud Scheduler (Cron Jobs)

**Scheduled Jobs:**

**1. Cleanup Old Events:** Daily at 2 AM
```bash
0 2 * * * curl -X POST https://stratint-api-...run.app/admin/cleanup
```

**2. Database Vacuum:** Weekly Sunday 3 AM
```sql
VACUUM ANALYZE events;
VACUUM ANALYZE sources;
```

**3. Metrics Aggregation:** Every 15 minutes
```bash
*/15 * * * * curl -X POST https://stratint-api-...run.app/internal/aggregate-metrics
```

**4. Backup Verification:** Daily at 4 AM
```bash
0 4 * * * gcloud sql backups list --instance=stratint-db --limit=1
```

### 9. Cloud Monitoring & Logging

**Metrics to Track:**
- Request latency (p50, p95, p99)
- Request rate (QPS)
- Error rate (4xx, 5xx)
- Database query latency
- Cache hit rate
- Memory usage
- CPU utilization
- Active connections

**Log Sinks:**
- Errors: Export to BigQuery for analysis
- Audit logs: Long-term retention (7 years)
- Access logs: 30-day retention

**Alerting Policies:**
- Error rate > 5% for 5 minutes
- Latency p95 > 1s for 5 minutes
- Database CPU > 80% for 10 minutes
- Instance count = max scale
- SSL certificate expiring in 30 days

## Deployment Steps

### Initial Setup

**1. Enable Required APIs:**
```bash
gcloud services enable \
  run.googleapis.com \
  sql-component.googleapis.com \
  sqladmin.googleapis.com \
  secretmanager.googleapis.com \
  redis.googleapis.com \
  cloudbuild.googleapis.com \
  cloudscheduler.googleapis.com
```

**2. Create VPC Connector:**
```bash
gcloud compute networks vpc-access connectors create stratint-connector \
  --region=us-central1 \
  --subnet-project=$PROJECT_ID \
  --subnet=default
```

**3. Create Cloud SQL Instance:**
```bash
gcloud sql instances create stratint-db \
  --database-version=POSTGRES_15 \
  --tier=db-custom-4-16384 \
  --region=us-central1 \
  --network=default \
  --no-assign-ip \
  --availability-type=REGIONAL \
  --backup
```

**4. Create Redis Instance:**
```bash
gcloud redis instances create stratint-cache \
  --size=5 \
  --region=us-central1 \
  --redis-version=redis_7_0 \
  --tier=standard
```

**5. Store Secrets:**
```bash
echo -n "$DATABASE_URL" | gcloud secrets create database-url --data-file=-
echo -n "$OPENAI_API_KEY" | gcloud secrets create openai-api-key --data-file=-
```

**6. Build & Deploy:**
```bash
gcloud builds submit --config=cloudbuild-api.yaml
```

## Cost Estimation

### Monthly Costs (Production)

**Cloud Run (API):**
- 10M requests/month @ $0.40/million = $4
- 100 GB-seconds @ $0.00002448/GB-second = $2.50
- 100 vCPU-seconds @ $0.00002400/vCPU-second = $2.40
- **Subtotal: ~$9/month**

**Cloud SQL:**
- db-custom-4-16384 instance = $280/month
- 100 GB SSD storage = $17/month
- Backups (100 GB) = $8/month
- **Subtotal: ~$305/month**

**Redis (Memorystore):**
- Standard tier 5 GB = $150/month

**Cloud Storage:**
- 500 GB standard storage = $10/month
- 100 GB backups (nearline) = $1/month

**Networking:**
- VPC Connector = $8/month
- Egress (estimate 100 GB) = $12/month

**Other Services:**
- Cloud Logging = $5/month
- Cloud Monitoring = $5/month
- Secret Manager = $1/month

**External APIs:**
- OpenAI (1K sources/day @ $0.01) = $300/month

**Total Estimated Cost: ~$800-900/month**

### Cost Optimization Strategies

1. **Use Committed Use Discounts** - 37% savings on Cloud SQL
2. **Scale to Zero** - Admin panel when not in use
3. **Implement Aggressive Caching** - Reduce database queries
4. **Use GPT-3.5-Turbo** - 10x cheaper than GPT-4 (~$30/month)
5. **Archive Old Data** - Move to cold storage after 1 year
6. **Right-size Instances** - Monitor and adjust based on usage

## Security Hardening

### Network Security
- ✅ Private IP for Cloud SQL (no public access)
- ✅ VPC Service Controls
- ✅ Cloud Armor for DDoS protection
- ✅ TLS 1.3 only
- ✅ Certificate pinning

### Access Control
- ✅ Service accounts with minimal permissions
- ✅ Workload Identity for pod-level IAM
- ✅ Secret rotation policies
- ✅ IP allowlisting for admin panel
- ✅ Multi-factor authentication

### Data Protection
- ✅ Encryption at rest (Cloud SQL, Redis, Storage)
- ✅ Encryption in transit (TLS everywhere)
- ✅ Automated backups with encryption
- ✅ Point-in-time recovery enabled
- ✅ Audit logging for all access

### Compliance
- ✅ GDPR compliance (data residency, deletion)
- ✅ SOC 2 compliant infrastructure
- ✅ Regular security audits
- ✅ Vulnerability scanning
- ✅ Penetration testing quarterly

## Disaster Recovery

### Backup Strategy
- **Database:** Daily automated backups, 7-day retention
- **Point-in-Time Recovery:** 7-day window
- **Configuration:** Version controlled in Git
- **Secrets:** Backed up in separate project

### Recovery Procedures

**RTO (Recovery Time Objective):** 1 hour  
**RPO (Recovery Point Objective):** 24 hours

**Scenario 1: Database Failure**
- Automatic failover to standby replica (~30 seconds)
- Manual restore from backup if needed (15-30 minutes)

**Scenario 2: Regional Outage**
- Deploy to alternate region (30-45 minutes)
- Restore latest backup to new region
- Update DNS/Load Balancer

**Scenario 3: Data Corruption**
- Point-in-time recovery to before corruption
- Restore to temporary instance for validation
- Switchover to restored instance

## Monitoring & Alerts

### Critical Alerts (PagerDuty)
- API error rate > 5%
- Database CPU > 90%
- All instances at max scale
- SSL certificate expiring < 7 days
- Backup failure

### Warning Alerts (Email/Slack)
- High latency (p95 > 1s)
- Cache hit rate < 70%
- Disk usage > 80%
- Unusual traffic patterns
- Connector failures

### Dashboards
- **Operations:** Real-time service health
- **Performance:** Latency, throughput, errors
- **Business:** Event metrics, source statistics
- **Cost:** Resource usage and billing

## Next Steps

1. [ ] Create GCP project and enable billing
2. [ ] Set up VPC and networking
3. [ ] Provision Cloud SQL and Redis
4. [ ] Create Dockerfile for Cloud Run
5. [ ] Set up Cloud Build CI/CD
6. [ ] Configure secrets in Secret Manager
7. [ ] Deploy initial version
8. [ ] Set up monitoring and alerts
9. [ ] Configure backups and disaster recovery
10. [ ] Security audit and penetration testing
