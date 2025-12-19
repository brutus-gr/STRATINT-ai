# Admin Panel Specification

## Overview

The STRATINT admin panel provides a web-based interface for managing system configuration, monitoring health, and controlling the ingestion pipeline.

## Authentication & Authorization

### Authentication Methods
1. **Google OAuth 2.0** (primary for admin users)
2. **API Key Authentication** (for programmatic access)
3. **Service Account** (for internal automation)

### Role-Based Access Control (RBAC)

**Roles:**
- **Super Admin** - Full system access
- **Admin** - Configuration and monitoring
- **Operator** - Read-only access to metrics
- **API User** - Programmatic API access only

**Permissions Matrix:**

| Action | Super Admin | Admin | Operator | API User |
|--------|------------|-------|----------|----------|
| View metrics | ✅ | ✅ | ✅ | ✅ |
| View events | ✅ | ✅ | ✅ | ✅ |
| Manage connectors | ✅ | ✅ | ❌ | ❌ |
| Adjust thresholds | ✅ | ✅ | ❌ | ❌ |
| Manage users | ✅ | ❌ | ❌ | ❌ |
| View audit logs | ✅ | ✅ | ❌ | ❌ |
| Delete events | ✅ | ❌ | ❌ | ❌ |

## Core Features

### 1. Dashboard (Home)

**Metrics Overview:**
- Real-time event count (last hour, day, week)
- Events by category breakdown (pie chart)
- Events by magnitude distribution (histogram)
- Average confidence score trend
- Ingestion rate (sources/minute)
- Enrichment queue depth
- System health status

**Recent Activity:**
- Last 20 events published
- Recent connector errors
- Admin action audit trail

**Alerts:**
- Failed connector notifications
- Low confidence event warnings
- Rate limit threshold alerts
- Database connection issues

### 2. Source Connector Management

**Connector List View:**
```
┌─────────────────────────────────────────────────────┐
│ Twitter Connector                        [ENABLED]  │
│ Status: ✅ Healthy | Last Fetch: 2m ago             │
│ Fetched: 1,234 | Errors: 2 | Avg Latency: 450ms   │
│ [Configure] [Disable] [View Logs]                  │
└─────────────────────────────────────────────────────┘
```

**Configuration Panel:**
- **Enable/Disable** toggle
- **Polling Interval** (minutes): [5] ⚙️
- **Max Retries**: [3] ⚙️
- **Timeout** (seconds): [30] ⚙️
- **Credibility Base Score**: [0.60] ⚙️
- **API Credentials**: [Edit] 🔒
- **Rate Limit**: [15 req/15min] ⚙️

**Per-Connector Settings:**

*Twitter:*
- API Key / Bearer Token
- Search queries to monitor
- User accounts to track
- Hashtags to follow

*Telegram:*
- Channel IDs to monitor
- Bot token
- Update polling method

*Reddit:*
- Subreddits to monitor
- Client ID/Secret
- User agent string

*Government Sources:*
- RSS feed URLs
- Refresh intervals
- Source priorities

### 3. Threshold & Scoring Configuration

**Confidence Scoring Weights:**
```
Source Credibility:    [30%] ────────────────██████
Source Type:           [20%] ──────────████
Entity Confidence:     [15%] ──────███
Content Quality:       [15%] ──────███
Recency:              [10%] ────██
Metadata Richness:    [10%] ────██
```

**Source Type Weights:**
```
Government:     [0.95] █████████▌
News Media:     [0.85] ████████▌
Twitter:        [0.60] ██████
Telegram:       [0.55] █████▌
Reddit:         [0.50] █████
Blog:           [0.45] ████▌
4chan:          [0.30] ███
GLP:            [0.25] ██▌
```

**Magnitude Category Base:**
```
Terrorism:      [9.0] █████████
Military:       [8.0] ████████
Disaster:       [7.5] ███████▌
Geopolitics:    [7.0] ███████
Cyber:          [6.0] ██████
Economic:       [4.5] ████▌
```

**Publication Thresholds:**
- Minimum Confidence: [0.30] ⚙️
- Minimum Magnitude: [1.0] ⚙️
- Require Sources: [1] ⚙️

### 4. Content Moderation

**Keyword Filters:**
```
BLOCKED KEYWORDS (prevent publication):
- [keyword1]  [✕ Remove]
- [keyword2]  [✕ Remove]
[+ Add Keyword]

FLAGGED KEYWORDS (manual review):
- [keyword3]  [✕ Remove]
- [keyword4]  [✕ Remove]
[+ Add Keyword]
```

**Content Rules:**
- Block ALL CAPS content: [✓]
- Block excessive exclamation: [✓]
- Minimum content length: [50] characters
- Maximum content length: [10000] characters
- Require at least [1] entity
- Auto-reject low credibility (<0.2): [✓]

**Manual Review Queue:**
- Events flagged for review
- Approve/Reject actions
- Add to blocklist option

### 5. User & API Key Management

**User List:**
```
┌──────────────────────────────────────────────────┐
│ admin@example.com          | Super Admin | Active│
│ operator@example.com       | Operator    | Active│
│ [+ Add User]                                     │
└──────────────────────────────────────────────────┘
```

**API Key Management:**
```
┌──────────────────────────────────────────────────┐
│ Production API (prod-key-abc123)      | Active  │
│ Created: 2024-01-01 | Last Used: 1h ago         │
│ Requests: 1.2M | Rate Limit: 1000/min          │
│ [Regenerate] [Revoke] [View Usage]             │
└──────────────────────────────────────────────────┘
```

**API Key Creation:**
- Name/Description
- Rate limit (requests/minute)
- Expiration date (optional)
- Scope restrictions (read-only, write, admin)

### 6. System Health & Monitoring

**Service Status:**
```
✅ API Server          | Healthy
✅ Ingestion Pipeline  | Healthy
✅ Enrichment Service  | Healthy
⚠️ PostgreSQL         | High CPU (85%)
✅ Redis Cache        | Healthy
✅ Cloud Storage      | Healthy
```

**Performance Metrics:**
- Request latency (p50, p95, p99)
- Cache hit rate
- Database connection pool usage
- Memory usage
- CPU usage
- Network I/O

**Resource Usage:**
- Cloud Run instances: [3/10]
- Database connections: [45/100]
- Redis memory: [2.1GB / 4GB]
- Storage used: [125GB / 500GB]

### 7. Audit Log Viewer

**Log Filters:**
- Date range: [Last 7 days ▼]
- Action type: [All ▼]
- User: [All ▼]
- Severity: [All ▼]

**Log Entries:**
```
┌────────────────────────────────────────────────────────┐
│ 2024-01-15 14:23:45 | admin@example.com               │
│ Action: DISABLE_CONNECTOR                             │
│ Target: twitter-connector                             │
│ Reason: "Exceeding rate limits"                       │
│ IP: 192.168.1.100                                     │
└────────────────────────────────────────────────────────┘
```

**Audited Actions:**
- Configuration changes
- User management (add, remove, role change)
- API key operations
- Content moderation actions
- Threshold adjustments
- Manual event deletion

### 8. Event Browser & Search

**Search Interface:**
```
┌────────────────────────────────────────────────────────┐
│ Search: [               ]  [Advanced Filters ▼]       │
│                                                        │
│ Filters:                                              │
│ Category:     [All ▼]                                 │
│ Confidence:   [0.0 ──────●───────── 1.0]             │
│ Magnitude:    [0 ────────●──────── 10]               │
│ Date Range:   [Last 24h ▼]                           │
│ Source Type:  [All ▼]                                 │
│                                           [Search]    │
└────────────────────────────────────────────────────────┘
```

**Event Detail View:**
- Full event metadata
- Source attribution with links
- Entity list with types
- Confidence breakdown
- Magnitude calculation details
- Related events (similar entities/tags)
- Actions: [Edit] [Delete] [Flag]

## Admin API Endpoints

### Authentication
```
POST   /admin/api/auth/login
POST   /admin/api/auth/logout
GET    /admin/api/auth/me
```

### Connectors
```
GET    /admin/api/connectors
GET    /admin/api/connectors/:id
PUT    /admin/api/connectors/:id/config
POST   /admin/api/connectors/:id/enable
POST   /admin/api/connectors/:id/disable
POST   /admin/api/connectors/:id/trigger
GET    /admin/api/connectors/:id/logs
```

### Configuration
```
GET    /admin/api/config/thresholds
PUT    /admin/api/config/thresholds
GET    /admin/api/config/weights
PUT    /admin/api/config/weights
GET    /admin/api/config/moderation
PUT    /admin/api/config/moderation
```

### Users & API Keys
```
GET    /admin/api/users
POST   /admin/api/users
DELETE /admin/api/users/:id
PUT    /admin/api/users/:id/role

GET    /admin/api/keys
POST   /admin/api/keys
DELETE /admin/api/keys/:id
POST   /admin/api/keys/:id/regenerate
```

### Monitoring
```
GET    /admin/api/metrics
GET    /admin/api/health
GET    /admin/api/stats
GET    /admin/api/audit-logs
```

### Events
```
GET    /admin/api/events
DELETE /admin/api/events/:id
PUT    /admin/api/events/:id/status
POST   /admin/api/events/:id/flag
```

## Technology Stack

### Frontend
- **Framework**: React 18 + TypeScript
- **UI Library**: shadcn/ui + Radix UI
- **Styling**: TailwindCSS
- **State Management**: Zustand or React Query
- **Charts**: Recharts or Chart.js
- **Forms**: React Hook Form + Zod validation
- **Auth**: Firebase Auth or Auth0

### Backend API
- **Language**: Go (same codebase)
- **Router**: Chi or Gin
- **Auth**: JWT tokens + middleware
- **Validation**: Go struct tags
- **Database**: PostgreSQL (Cloud SQL)
- **Cache**: Redis

### Deployment
- **Frontend**: Cloud Run (separate service)
- **API**: Cloud Run (admin endpoints)
- **Build**: Cloud Build
- **CDN**: Cloud CDN for static assets

## Security Considerations

### Authentication Security
- ✅ OAuth 2.0 with Google Workspace
- ✅ JWT tokens with short expiration (1h)
- ✅ Refresh token rotation
- ✅ API keys stored hashed (SHA-256)
- ✅ Rate limiting per user/key
- ✅ IP allowlisting option

### Authorization
- ✅ Role-based access control (RBAC)
- ✅ Permission checks on every request
- ✅ Audit logging of all actions
- ✅ Least privilege principle

### Data Protection
- ✅ HTTPS only (TLS 1.3)
- ✅ Secrets in Cloud Secret Manager
- ✅ Database encryption at rest
- ✅ Encrypted backups
- ✅ No sensitive data in logs

### Input Validation
- ✅ Schema validation on all inputs
- ✅ SQL injection prevention (prepared statements)
- ✅ XSS protection (CSP headers)
- ✅ CSRF tokens
- ✅ Rate limiting per endpoint

## Development Roadmap

### Phase 1: Core Admin Backend
- [ ] Admin API authentication system
- [ ] RBAC middleware
- [ ] Connector management endpoints
- [ ] Configuration endpoints
- [ ] Audit logging system

### Phase 2: Admin Frontend
- [ ] React app scaffolding
- [ ] Authentication flow
- [ ] Dashboard page
- [ ] Connector management UI
- [ ] Configuration UI

### Phase 3: Advanced Features
- [ ] Real-time metrics websocket
- [ ] Event browser and search
- [ ] Content moderation queue
- [ ] User management UI
- [ ] API key management UI

### Phase 4: Monitoring & Alerts
- [ ] Prometheus metrics integration
- [ ] Alert configuration UI
- [ ] Email/Slack notifications
- [ ] Incident management

## Estimated Development Time

- **Backend API**: 2-3 weeks
- **Frontend UI**: 3-4 weeks
- **Integration & Testing**: 1-2 weeks
- **Security Hardening**: 1 week
- **Documentation**: 1 week

**Total**: 8-11 weeks (2-3 months)
