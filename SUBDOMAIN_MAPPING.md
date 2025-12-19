# STRATINT Subdomain Mapping Guide

## Overview

STRATINT is deployed as three separate Cloud Run services, each optimized for its specific purpose:

| Subdomain | Purpose | Service | Image Size |
|-----------|---------|---------|------------|
| **stratint.com** | Web frontend + API | `stratint` | ~500MB (includes Playwright) |
| **api.stratint.com** | REST API only | `stratint-api` | ~50MB (lightweight) |
| **mcp.stratint.com** | MCP protocol server | `stratint-mcp` | ~30MB (minimal) |

---

## Service Details

### 1. Main Web Service (`stratint`)
**Subdomain:** `stratint.com`
**Cloud Run URL:** `https://[YOUR-SERVICE]-[HASH]-uc.a.run.app`

**Features:**
- React web frontend (cyberpunk UI)
- Full REST API
- Admin panel at `/admin`
- Content scraping with Playwright
- RSS feed generation

**Resources:**
- Memory: 2Gi
- CPU: 2 vCPU
- Includes: Frontend, Backend, Playwright, all features

**Endpoints:**
```
GET  /                    - Web interface
GET  /admin               - Admin panel
GET  /api/events          - Events API
GET  /api/events/:id      - Single event
PUT  /api/events/:id/status - Update event status (auth required)
GET  /api/sources         - Sources list
POST /api/admin/*         - Admin endpoints (auth required)
```

---

### 2. API Service (`stratint-api`)
**Subdomain:** `api.stratint.com`
**Cloud Run URL:** `https://[YOUR-API-SERVICE]-[HASH]-uc.a.run.app`

**Features:**
- Pure REST API endpoints
- Lightweight (no frontend, no Playwright)
- Fast startup and scaling
- Optimized for API requests

**Resources:**
- Memory: 2Gi
- CPU: 2 vCPU
- Minimal image size (~50MB vs ~500MB)

**Endpoints:**
```
GET  /api/events                    - Query events
GET  /api/events/:id                - Get single event
PUT  /api/events/:id/status         - Update event status (auth)
POST /api/auth/login                - Admin login
GET  /api/auth/validate             - Validate token
GET  /api/admin/config/openai       - OpenAI config (auth)
PUT  /api/admin/config/openai       - Update OpenAI config (auth)
GET  /api/admin/config/scraper      - Scraper settings (auth)
PUT  /api/admin/config/scraper      - Update scraper settings (auth)
GET  /api/admin/tracked-accounts    - Tracked accounts (auth)
POST /api/admin/tracked-accounts    - Add tracked account (auth)
GET  /api/admin/thresholds          - Thresholds config (auth)
PUT  /api/admin/thresholds          - Update thresholds (auth)
```

**Why Separate API Service?**
1. **Performance:** Faster cold starts (no Playwright dependencies)
2. **Cost:** Lower memory usage for API-only requests
3. **Scaling:** Independent scaling from web frontend
4. **Clean Architecture:** API consumers don't need frontend assets

---

### 3. MCP Server (`stratint-mcp`)
**Subdomain:** `mcp.stratint.com`
**Cloud Run URL:** `https://[YOUR-MCP-SERVICE]-[HASH]-uc.a.run.app`

**Features:**
- Model Context Protocol (MCP) server
- JSON-RPC 2.0 interface
- AI assistant integration (Claude Desktop, Cline, etc.)
- Single `get_events` tool with 13+ parameters

**Resources:**
- Memory: 1Gi
- CPU: 1 vCPU
- Minimal image size (~30MB)

**Endpoints:**
```
POST /                    - MCP JSON-RPC endpoint
GET  /healthz             - Health check
```

**MCP Methods:**
- `initialize` - Initialize MCP session
- `tools/list` - List available tools
- `tools/call` - Execute tool (get_events)

**Example MCP Request:**
```bash
curl -X POST https://mcp.stratint.com \
  -H 'Content-Type: application/json' \
  -d '{
    "jsonrpc":"2.0",
    "id":1,
    "method":"tools/call",
    "params":{
      "name":"get_events",
      "arguments":{
        "min_magnitude":7.0,
        "limit":10,
        "categories":["cyber","military"]
      }
    }
  }'
```

---

## DNS Configuration

### Using Google Cloud Run Domain Mapping

**Prerequisites:**
1. Domain ownership verified in Google Cloud Console
2. Cloud Run Admin permissions

**Steps:**

#### 1. Map Main Web Service
```bash
gcloud run domain-mappings create \
  --service=stratint \
  --domain=stratint.com \
  --region=us-central1 \
  --project=YOUR_PROJECT_ID
```

#### 2. Map API Service
```bash
gcloud run domain-mappings create \
  --service=stratint-api \
  --domain=api.stratint.com \
  --region=us-central1 \
  --project=YOUR_PROJECT_ID
```

#### 3. Map MCP Service
```bash
gcloud run domain-mappings create \
  --service=stratint-mcp \
  --domain=mcp.stratint.com \
  --region=us-central1 \
  --project=YOUR_PROJECT_ID
```

#### 4. Update DNS Records

After creating the domain mappings, Google will provide DNS records. Add these to your domain registrar:

**Example DNS Records (Cloudflare/Namecheap/etc.):**
```
Type    Name    Value                           TTL
CNAME   @       ghs.googlehosted.com           Auto
CNAME   api     ghs.googlehosted.com           Auto
CNAME   mcp     ghs.googlehosted.com           Auto
```

**Note:** Actual DNS records will be provided by Google Cloud after running the domain mapping commands.

---

### Alternative: Using Cloudflare/External DNS

If you prefer using Cloudflare or external DNS provider:

**DNS Records:**
```
Type    Name    Value                                              TTL    Proxy
CNAME   @       [YOUR-SERVICE]-[HASH]-uc.a.run.app               Auto   Yes
CNAME   api     [YOUR-API-SERVICE]-[HASH]-uc.a.run.app           Auto   Yes
CNAME   mcp     [YOUR-MCP-SERVICE]-[HASH]-uc.a.run.app           Auto   Yes
```

**Cloudflare Benefits:**
- DDoS protection
- CDN for static assets
- Advanced caching
- Analytics
- Free SSL/TLS

---

## Testing Subdomains

### Test Web Frontend
```bash
curl https://stratint.com/
curl https://stratint.com/api/events
```

### Test API Service
```bash
# List events
curl https://api.stratint.com/api/events

# Get single event
curl https://api.stratint.com/api/events/{event-id}

# Filter by magnitude
curl "https://api.stratint.com/api/events?min_magnitude=7.0&limit=10"
```

### Test MCP Server
```bash
# Health check
curl https://mcp.stratint.com/healthz

# List tools
curl -X POST https://mcp.stratint.com \
  -H 'Content-Type: application/json' \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}'

# Query events
curl -X POST https://mcp.stratint.com \
  -H 'Content-Type: application/json' \
  -d '{
    "jsonrpc":"2.0",
    "id":2,
    "method":"tools/call",
    "params":{
      "name":"get_events",
      "arguments":{"limit":5}
    }
  }'
```

---

## SSL/TLS Certificates

### Automatic (Google-Managed)
Cloud Run automatically provisions and manages SSL certificates when using domain mapping.

- **Provisioning Time:** 15-30 minutes
- **Renewal:** Automatic
- **Cert Type:** Let's Encrypt via Google

### Cloudflare (Recommended for Production)
- **Free SSL:** Yes (Universal SSL)
- **Cert Type:** Cloudflare-managed
- **Full (Strict) Mode:** Recommended

---

## Deployment Scripts

### Deploy All Services
```bash
# Deploy web service (main)
./deploy-quick.sh

# Deploy API service
./deploy-api.sh

# Deploy MCP service
./deploy-mcp.sh
```

### Deploy Individual Service
```bash
# Web only
./deploy-quick.sh

# API only
./deploy-api.sh

# MCP only
./deploy-mcp.sh
```

---

## Service Comparison

| Feature | Web Service | API Service | MCP Service |
|---------|-------------|-------------|-------------|
| **Frontend** | ✅ React SPA | ❌ | ❌ |
| **REST API** | ✅ Full | ✅ Full | ❌ |
| **MCP Protocol** | ❌ | ❌ | ✅ |
| **Admin Panel** | ✅ | ✅ (API only) | ❌ |
| **Playwright** | ✅ | ❌ | ❌ |
| **Image Size** | ~500MB | ~50MB | ~30MB |
| **Cold Start** | ~5-10s | ~2-3s | ~1-2s |
| **Memory** | 2Gi | 2Gi | 1Gi |
| **Use Case** | End users | API clients | AI assistants |

---

## Cost Optimization

### Current Configuration

**Web Service (stratint):**
- Min instances: 0 (scales to zero)
- Max instances: 10
- Cost: ~$3-5/month (low traffic)

**API Service (stratint-api):**
- Min instances: 0 (scales to zero)
- Max instances: 10
- Cost: ~$1-2/month (low traffic)

**MCP Service (stratint-mcp):**
- Min instances: 0 (scales to zero)
- Max instances: 10
- Cost: ~$1/month (low traffic)

**Total Cloud Run Cost:** ~$5-8/month for all services

**Other Costs:**
- Cloud SQL: ~$10/month (db-f1-micro)
- Secret Manager: Free (< 6 secrets)
- Container Registry: ~$0.50/month
- **Total:** ~$15-20/month

### Production Scaling
For production with traffic:
- Set min instances to 1-2 for warm services
- Expected cost: $50-100/month at moderate load
- Scale max instances based on traffic

---

## Monitoring

### View Logs
```bash
# Web service
gcloud run services logs read stratint \
  --region=us-central1 \
  --project=YOUR_PROJECT_ID

# API service
gcloud run services logs read stratint-api \
  --region=us-central1 \
  --project=YOUR_PROJECT_ID

# MCP service
gcloud run services logs read stratint-mcp \
  --region=us-central1 \
  --project=YOUR_PROJECT_ID
```

### Metrics Dashboard
Visit: https://console.cloud.google.com/run?project=YOUR_PROJECT_ID

---

## Troubleshooting

### Domain Mapping Not Working
1. Wait 15-30 minutes for SSL provisioning
2. Check DNS propagation: `dig stratint.com`
3. Verify domain ownership in Cloud Console
4. Check domain mapping status:
   ```bash
   gcloud run domain-mappings describe stratint.com \
     --region=us-central1 \
     --project=YOUR_PROJECT_ID
   ```

### Service Not Responding
1. Check service status:
   ```bash
   gcloud run services describe {service-name} \
     --region=us-central1 \
     --project=YOUR_PROJECT_ID
   ```
2. Check logs for errors
3. Verify Cloud SQL connection
4. Check secrets are accessible

### API Returning 404
- Ensure you're using correct endpoints (see Endpoints section)
- Web service and API service have identical endpoints
- MCP service only responds to JSON-RPC POST requests

---

## Security

### HTTPS Only
- All Cloud Run services force HTTPS
- HTTP requests automatically redirect to HTTPS

### CORS
- Enabled on all services
- Allows cross-origin requests for API access

### Authentication
- Public endpoints: `/api/events`, `/api/events/:id`
- Protected endpoints: `/api/admin/*`, require JWT token
- Admin login: `POST /api/auth/login`

---

## Next Steps

1. **Purchase Domain:** Get `stratint.com` from registrar
2. **Verify Domain:** Add to Google Cloud Console
3. **Create Mappings:** Run gcloud domain mapping commands
4. **Update DNS:** Add CNAME records from registrar
5. **Wait for SSL:** 15-30 minutes for certificate provisioning
6. **Test:** Verify all subdomains work with HTTPS
7. **Update Frontend:** Change API URLs in React app to use custom domains

---

## Support

For issues or questions:
- Check logs: `gcloud run services logs read {service}`
- Cloud Run docs: https://cloud.google.com/run/docs
- DNS propagation: https://www.whatsmydns.net/

---

## Summary

✅ **All services deployed successfully**

- **Web:** `https://[YOUR-SERVICE]-[HASH]-uc.a.run.app` → `stratint.com`
- **API:** `https://[YOUR-API-SERVICE]-[HASH]-uc.a.run.app` → `api.stratint.com`
- **MCP:** `https://[YOUR-MCP-SERVICE]-[HASH]-uc.a.run.app` → `mcp.stratint.com`

Ready to map custom domains when you're ready!
