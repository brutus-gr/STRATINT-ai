# STRATINT Deployment Guide

This guide covers deploying STRATINT to various environments.

## Table of Contents

- [Docker Deployment](#docker-deployment)
- [Google Cloud Run](#google-cloud-run)
- [Traditional VPS](#traditional-vps)
- [Environment Variables](#environment-variables)
- [Production Checklist](#production-checklist)

## Docker Deployment

### Using Docker Compose (Recommended)

1. **Clone the repository**:
   ```bash
   git clone https://github.com/brutus-gr/STRATINT.git
   cd STRATINT
   ```

2. **Configure environment**:
   ```bash
   cp .env.example .env
   # Edit .env with your settings
   nano .env
   ```

   Required variables:
   ```env
   OPENAI_API_KEY=your_openai_key_here
   ADMIN_PASSWORD=your_secure_password
   POSTGRES_PASSWORD=your_postgres_password
   ```

3. **Start services**:
   ```bash
   docker-compose up -d
   ```

4. **Verify deployment**:
   ```bash
   curl http://localhost:8080/healthz
   # Should return: {"status":"healthy"}
   ```

5. **Access the application**:
   - Web UI: http://localhost:8080
   - Admin Panel: http://localhost:8080/admin
   - API: http://localhost:8080/api
   - RSS Feed: http://localhost:8080/api/feed.rss

### Building Custom Image

```bash
docker build -t stratint:latest .
docker run -d \
  -p 8080:8080 \
  -e DATABASE_URL="your_database_url" \
  -e OPENAI_API_KEY="your_key" \
  --name stratint \
  stratint:latest
```

## Google Cloud Run

### Prerequisites

- Google Cloud account
- `gcloud` CLI installed
- Cloud SQL instance (PostgreSQL 15)
- Cloud Build API enabled

### Deployment Steps

1. **Set up Cloud SQL**:
   ```bash
   gcloud sql instances create stratint-db \
     --database-version=POSTGRES_15 \
     --tier=db-f1-micro \
     --region=us-central1

   gcloud sql databases create stratint \
     --instance=stratint-db

   gcloud sql users create stratint \
     --instance=stratint-db \
     --password=YOUR_PASSWORD
   ```

2. **Build and push image**:
   ```bash
   gcloud builds submit --tag gcr.io/YOUR_PROJECT_ID/stratint
   ```

3. **Deploy to Cloud Run**:
   ```bash
   gcloud run deploy stratint \
     --image gcr.io/YOUR_PROJECT_ID/stratint \
     --platform managed \
     --region us-central1 \
     --allow-unauthenticated \
     --add-cloudsql-instances YOUR_PROJECT_ID:us-central1:stratint-db \
     --set-env-vars DATABASE_URL="postgresql://stratint:PASSWORD@/stratint?host=/cloudsql/YOUR_PROJECT_ID:us-central1:stratint-db" \
     --set-env-vars OPENAI_API_KEY="your_key" \
     --set-env-vars ADMIN_PASSWORD="your_password" \
     --memory 1Gi \
     --cpu 1 \
     --max-instances 10
   ```

4. **Run migrations**:
   ```bash
   # Connect to Cloud SQL
   gcloud sql connect stratint-db --user=stratint

   # Run migrations
   \i /path/to/migrations/001_initial_schema.sql
   # ... run all migrations
   ```

### Cloud Run Configuration

**Memory & CPU**:
- Minimum: 512Mi / 0.5 CPU
- Recommended: 1Gi / 1 CPU
- Heavy load: 2Gi / 2 CPU

**Concurrency**:
- Default: 80 requests per instance
- Adjust based on scraping load

**Timeout**:
- Increase to 300s for scraping operations

## Traditional VPS

### System Requirements

- OS: Ubuntu 22.04 LTS (recommended)
- RAM: 2GB minimum, 4GB+ recommended
- Storage: 20GB minimum
- CPU: 2 cores recommended

### Installation

1. **Install dependencies**:
   ```bash
   # Update system
   sudo apt update && sudo apt upgrade -y

   # Install Go
   wget https://go.dev/dl/go1.24.linux-amd64.tar.gz
   sudo tar -C /usr/local -xzf go1.24.linux-amd64.tar.gz
   export PATH=$PATH:/usr/local/go/bin

   # Install Node.js
   curl -fsSL https://deb.nodesource.com/setup_18.x | sudo -E bash -
   sudo apt install -y nodejs

   # Install PostgreSQL
   sudo apt install -y postgresql postgresql-contrib

   # Install Playwright dependencies
   npx playwright install-deps chromium
   ```

2. **Set up PostgreSQL**:
   ```bash
   sudo -u postgres psql
   CREATE DATABASE stratint;
   CREATE USER stratint WITH PASSWORD 'your_password';
   GRANT ALL PRIVILEGES ON DATABASE stratint TO stratint;
   \q
   ```

3. **Clone and build**:
   ```bash
   git clone https://github.com/brutus-gr/STRATINT.git
   cd STRATINT

   # Build backend
   go build -o server ./cmd/server

   # Build frontend
   cd web
   npm install
   npm run build
   cd ..
   ```

4. **Configure environment**:
   ```bash
   cp .env.example .env
   nano .env
   ```

5. **Run migrations**:
   ```bash
   export DATABASE_URL="postgresql://stratint:your_password@localhost:5432/stratint?sslmode=disable"
   psql $DATABASE_URL -f migrations/001_initial_schema.sql
   # Run all migrations
   ```

6. **Create systemd service**:
   ```bash
   sudo nano /etc/systemd/system/stratint.service
   ```

   ```ini
   [Unit]
   Description=STRATINT Intelligence Platform
   After=network.target postgresql.service

   [Service]
   Type=simple
   User=stratint
   WorkingDirectory=/home/stratint/STRATINT
   EnvironmentFile=/home/stratint/STRATINT/.env
   ExecStart=/home/stratint/STRATINT/server
   Restart=on-failure
   RestartSec=10

   [Install]
   WantedBy=multi-user.target
   ```

   ```bash
   sudo systemctl daemon-reload
   sudo systemctl enable stratint
   sudo systemctl start stratint
   ```

7. **Set up Nginx reverse proxy**:
   ```bash
   sudo apt install -y nginx

   sudo nano /etc/nginx/sites-available/stratint
   ```

   ```nginx
   server {
       listen 80;
       server_name your-domain.com;

       location / {
           proxy_pass http://localhost:8080;
           proxy_http_version 1.1;
           proxy_set_header Upgrade $http_upgrade;
           proxy_set_header Connection 'upgrade';
           proxy_set_header Host $host;
           proxy_set_header X-Real-IP $remote_addr;
           proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
           proxy_set_header X-Forwarded-Proto $scheme;
           proxy_cache_bypass $http_upgrade;
       }
   }
   ```

   ```bash
   sudo ln -s /etc/nginx/sites-available/stratint /etc/nginx/sites-enabled/
   sudo nginx -t
   sudo systemctl restart nginx
   ```

8. **SSL with Let's Encrypt**:
   ```bash
   sudo apt install -y certbot python3-certbot-nginx
   sudo certbot --nginx -d your-domain.com
   ```

## Environment Variables

### Required

| Variable | Description | Example |
|----------|-------------|---------|
| `DATABASE_URL` | PostgreSQL connection string | `postgresql://user:pass@host:5432/db` |
| `OPENAI_API_KEY` | OpenAI API key | `sk-...` |
| `ADMIN_PASSWORD` | Admin panel password | `secure_password_123` |

### Optional

| Variable | Description | Default |
|----------|-------------|---------|
| `SERVER_PORT` | HTTP server port | `8080` |
| `LOG_LEVEL` | Logging level | `info` |
| `LOG_FORMAT` | Log format (json/text) | `json` |
| `ENVIRONMENT` | Environment name | `production` |
| `DATABASE_MAX_CONNECTIONS` | Max DB connections | `100` |
| `DATABASE_MAX_IDLE_CONNECTIONS` | Max idle connections | `10` |

## Production Checklist

### Security

- [ ] Change default admin password
- [ ] Use strong PostgreSQL password
- [ ] Enable SSL/TLS (HTTPS)
- [ ] Set `ENVIRONMENT=production`
- [ ] Restrict CORS origins (if needed)
- [ ] Enable firewall (allow 80, 443, SSH only)
- [ ] Keep API keys in secrets manager
- [ ] Regular security updates

### Performance

- [ ] Configure database connection pool
- [ ] Set up database indexes
- [ ] Enable query caching (optional)
- [ ] Configure scraper worker count
- [ ] Set appropriate OpenAI rate limits
- [ ] Monitor resource usage

### Monitoring

- [ ] Set up Prometheus scraping
- [ ] Configure alerting
- [ ] Log aggregation
- [ ] Error tracking
- [ ] Uptime monitoring
- [ ] Database backups

### Backups

```bash
# Automated PostgreSQL backup script
#!/bin/bash
DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR="/backup/stratint"
pg_dump $DATABASE_URL > $BACKUP_DIR/backup_$DATE.sql
find $BACKUP_DIR -type f -mtime +7 -delete  # Keep 7 days
```

### Scaling

- [ ] Monitor API response times
- [ ] Track scraping throughput
- [ ] Watch database performance
- [ ] Plan for horizontal scaling if needed
- [ ] Consider CDN for static assets

## Troubleshooting

### Common Issues

**Database connection failed**:
```bash
# Check PostgreSQL is running
sudo systemctl status postgresql

# Verify connection string
psql $DATABASE_URL -c "SELECT 1"
```

**Scraping failures**:
```bash
# Install Playwright dependencies
npx playwright install-deps

# Check Chromium availability
which chromium-browser
```

**High memory usage**:
- Reduce scraper worker count
- Decrease database connection pool
- Enable garbage collection tuning

**OpenAI API errors**:
- Verify API key
- Check rate limits
- Monitor quota usage

### Health Checks

```bash
# Application health
curl http://localhost:8080/healthz

# Database health
psql $DATABASE_URL -c "SELECT 1"

# Check logs
journalctl -u stratint -f  # systemd
docker logs stratint       # Docker
tail -f server.log         # standalone
```

## Updates & Maintenance

### Updating STRATINT

```bash
# Backup database
pg_dump $DATABASE_URL > backup_$(date +%Y%m%d).sql

# Pull latest code
git pull origin main

# Rebuild
go build -o server ./cmd/server
cd web && npm install && npm run build && cd ..

# Run new migrations
psql $DATABASE_URL -f migrations/XXX_new_migration.sql

# Restart service
sudo systemctl restart stratint  # VPS
docker-compose restart           # Docker
```

### Database Maintenance

```bash
# Vacuum database
psql $DATABASE_URL -c "VACUUM ANALYZE"

# Check database size
psql $DATABASE_URL -c "SELECT pg_size_pretty(pg_database_size('stratint'))"

# Archive old events
psql $DATABASE_URL -c "UPDATE events SET status='archived' WHERE created_at < NOW() - INTERVAL '90 days' AND status='published'"
```

## Support

For deployment issues:
1. Check the logs
2. Review this documentation
3. Open an issue on GitHub
4. Join the community discussions

---

For architecture details, see [ARCHITECTURE.md](../ARCHITECTURE.md)
