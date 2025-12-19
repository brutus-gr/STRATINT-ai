# STRATINT System Architecture

## Overview

STRATINT is an AI-powered Open Source Intelligence (OSINT) platform that continuously monitors multiple data sources, enriches content with AI analysis, and provides a real-time intelligence feed via MCP (Model Context Protocol) server.

**Current Status:** Phase 3 Complete (33% - Foundation, Ingestion, NLP)

## System Components

### 1. Data Models (`internal/models/`)
✅ **Status:** Complete

**Core Types:**
- `Event` - Processed intelligence events with confidence, magnitude, entities
- `Source` - Raw OSINT data with platform metadata and attribution
- `Entity` - Extracted named entities (countries, persons, organizations, etc.)
- `EventQuery` - Comprehensive filtering and pagination for event retrieval
- `Confidence` - Multi-factor reliability assessment (0-1 scale)

**Key Features:**
- 10 event categories (military, cyber, geopolitics, etc.)
- 9 source types (Twitter, Telegram, Reddit, government, etc.)
- 11 entity types with normalized names and reference data
- Flexible querying with 13+ filter parameters

### 2. Data Ingestion (`internal/ingestion/`)
✅ **Status:** Complete

**Components:**
- `Connector` interface for pluggable data sources
- `Pipeline` orchestrator with concurrent fetching and retry logic
- `Retry` system with exponential backoff and jitter
- `Deduplicator` using SHA-256 content fingerprinting
- `SourceRepository` / `EventRepository` storage abstractions

**Features:**
- Exponential backoff retry (max 3 retries, 1s-30s delays)
- SHA-256 content hashing with normalization
- Time-windowed deduplication (24-hour default)
- Concurrent connector fetching with semaphore control
- In-memory repositories for testing/development

**Connectors Planned:**
- Twitter (Free/Basic API)
- Telegram (MTProto, unlimited)
- Reddit (100 req/min free)
- 4chan (1 req/sec, no auth)
- Government RSS feeds
- News APIs

### 3. NLP Enrichment (`internal/enrichment/`)
✅ **Status:** Complete

**Components:**
- `OpenAIClient` - GPT-4 Turbo integration for analysis
- `PromptTemplates` - OSINT-optimized system/user prompts
- `EntityExtractor` - Named entity recognition with normalization
- `ConfidenceScorer` - 6-factor weighted confidence algorithm
- `MagnitudeEstimator` - 0-10 event severity scoring
- `MockEnricher` - Rule-based testing implementation

**Confidence Scoring (6 Factors):**
1. Source Credibility (30%)
2. Source Type (20%)
3. Entity Confidence (15%)
4. Content Quality (15%)
5. Recency (10%)
6. Metadata Richness (10%)

**Magnitude Estimation:**
- Base score by category (Terrorism: 9.0, Military: 8.0, etc.)
- Modifiers: entity count, engagement, urgency, scope
- Final range: 0-10 (clamped)

**AI Configuration:**
- Model: GPT-4-turbo-preview (or GPT-3.5-turbo for cost savings)
- Temperature: 0.3 (factual analysis)
- Max tokens: 2000
- Structured JSON output with fallback parsing

### 4. Event Management & MCP Functions
🔄 **Status:** Planned (Phase 4)

**Core Function:**
```typescript
get_events(query: EventQuery): EventResponse
```

**Query Capabilities:**
- Full-text search across title/summary
- Time ranges (since/until timestamps)
- Magnitude/confidence thresholds
- Category, source type, tag, entity type filters
- Status filtering (pending, enriched, published, archived)
- Flexible sorting (timestamp, magnitude, confidence)
- Pagination (1-200 results per page)

**Event Lifecycle:**
```
Raw Source → [Ingest] → Pending
           ↓
    [Enrich with AI] → Enriched
           ↓
     [Quality Check] → Published (if meets thresholds)
           ↓
  [Archive after 1yr] → Archived
```

**See:** `/docs/MCP_FUNCTION_DESIGN.md`

### 5. Storage & Caching
🔄 **Status:** Planned (Phase 5)

**Google Cloud SQL (PostgreSQL 15):**
- Regional HA with automatic failover
- PostGIS for geospatial queries
- Full-text search with tsvector indexes
- Composite indexes for query optimization
- Daily automated backups (7-day retention)

**Redis (Memorystore):**
- Standard tier 5GB with HA
- Query result caching (5-min TTL)
- Rate limiting counters
- Session storage
- Bloom filters for deduplication

**Database Schema:**
- `events` - Main events table with JSONB confidence
- `sources` - Raw source data with metadata
- `entities` - Normalized entity catalog
- `event_sources` - Many-to-many relationship
- `event_entities` - Entity-event associations

### 6. Frontend (Vite + React + TypeScript)
🔄 **Status:** Planned (Phase 6)

**Public Homepage:**
- Real-time intelligence feed
- Event cards with magnitude/confidence indicators
- Category filtering
- Source attribution
- Interactive map for geospatial events
- Mobile-responsive design

**Technologies:**
- React 18 + TypeScript
- TailwindCSS + shadcn/ui
- React Query for state management
- Recharts for visualizations
- Lucide icons

### 7. Google Cloud Deployment
🔄 **Status:** Planned (Phase 7)

**Architecture:**
```
Internet → Cloud Load Balancer
              ↓
    ┌─────────┴─────────┐
    ↓                   ↓
Cloud Run (API)    Cloud Run (Admin)
    ↓                   ↓
    └─────────┬─────────┘
              ↓
    ┌─────────┴─────────┐
    ↓                   ↓
Cloud SQL          Redis
(PostgreSQL)    (Memorystore)
```

**Cloud Run Configuration:**
- API: 1-100 instances autoscale, 2 vCPU, 2GB RAM
- Admin: 0-10 instances, 1 vCPU, 1GB RAM
- VPC private networking to Cloud SQL/Redis
- Secret Manager for credentials

**CI/CD:**
- Cloud Build triggers on GitHub push
- Automated testing in build pipeline
- Container image to Container Registry
- Automated deployment to Cloud Run

**Cost Estimate:** $800-900/month
- Cloud SQL: $305
- Redis: $150
- OpenAI API: $300 (GPT-4) or $30 (GPT-3.5)
- Cloud Run: $9
- Storage/Networking: $50

**See:** `/docs/GOOGLE_CLOUD_DEPLOYMENT.md`

### 8. Compliance, Security & Ethics
🔄 **Status:** Planned (Phase 8)

**Security Measures:**
- Private VPC networking (no public IPs)
- TLS 1.3 encryption everywhere
- Secret Manager for credentials
- Workload Identity for service accounts
- Cloud Armor for DDoS protection
- Automated vulnerability scanning

**Content Moderation:**
- Keyword blocklists
- ALL CAPS rejection
- Minimum confidence thresholds
- Manual review queue
- DMCA compliance

**Privacy:**
- GDPR-compliant data handling
- Right to deletion
- Anonymous source handling
- No PII collection

### 9. Marketing & Community
🔄 **Status:** Planned (Phase 9)

- Landing page with value proposition
- Documentation site
- Social media presence
- OSINT community outreach
- Free mobile app roadmap

### 10. Admin Panel & Management
🔄 **Status:** Planned (Phase 10)

**Authentication:**
- Google OAuth 2.0
- JWT tokens
- RBAC (Super Admin, Admin, Operator, API User)

**Features:**
- **Dashboard:** Real-time metrics, health status, recent activity
- **Connector Management:** Enable/disable, configure, monitor
- **Threshold Tuning:** Adjust confidence weights, publication thresholds
- **Content Moderation:** Keyword filters, review queue, blocklists
- **User Management:** Add/remove users, manage roles
- **API Keys:** Generate, revoke, monitor usage
- **Audit Logs:** Track all admin actions
- **Metrics Dashboard:** Request rates, latency, errors

**Technology Stack:**
- React + TypeScript frontend
- Go backend with Chi router
- PostgreSQL for persistence
- Redis for sessions

**Development Time:** 8-11 weeks

**See:** `/docs/ADMIN_PANEL_SPEC.md`

## Data Flow

### Ingestion Flow
```
1. Source APIs → Connectors (with rate limiting)
2. Connectors → Retry Logic (exponential backoff)
3. Raw Sources → Deduplicator (SHA-256 fingerprinting)
4. Unique Sources → SourceRepository (PostgreSQL)
5. Sources → Enrichment Queue
```

### Enrichment Flow
```
1. Source → OpenAI API (GPT-4 analysis)
2. AI Response → Entity Extraction
3. Entities → EntityNormalizer (reference data)
4. Source + Entities → ConfidenceScorer (6 factors)
5. Event + Confidence → MagnitudeEstimator
6. Complete Event → EventRepository
7. If publishable → Published status
```

### Query Flow
```
1. Client → MCP get_events(query)
2. Query → Validation (EventQuery.Validate)
3. Query → Cache Check (Redis)
4. Cache Miss → Database Query (PostgreSQL)
5. Results → Cache Store (5-min TTL)
6. EventResponse → Client
```

## API Endpoints

### MCP Functions (Phase 4)
```
get_events(query: EventQuery): EventResponse
```

### Admin API (Phase 10)
```
POST   /admin/auth/login
GET    /admin/auth/me

GET    /admin/connectors
PUT    /admin/connectors/:id/config
POST   /admin/connectors/:id/enable

GET    /admin/config/thresholds
PUT    /admin/config/thresholds

GET    /admin/users
POST   /admin/users

GET    /admin/metrics
GET    /admin/audit-logs
```

### Public API (Phase 4)
```
GET    /healthz
GET    /metrics (Prometheus)
```

## Configuration

**Environment Variables:**
- Database: `DATABASE_URL`, connection pool settings
- Redis: `REDIS_URL`, cache TTLs
- OpenAI: `OPENAI_API_KEY`, model selection
- Connectors: API keys, enable flags
- Thresholds: Confidence/magnitude minimums
- Admin: JWT secret, session duration

**See:** `.env.example`

## Testing Strategy

**Current Test Coverage:**
- ✅ Models: 36 tests (100% coverage)
- ✅ Ingestion: 36 tests (retry, dedup, storage)
- ✅ Enrichment: 23 tests (scoring, extraction)

**Testing Approach:**
- Unit tests for all components
- Integration tests for pipelines
- MockEnricher for API-free testing
- In-memory repositories for fast tests
- CI/CD automated testing

## Performance Targets

**Latency:**
- p50: < 200ms
- p95: < 500ms
- p99: < 1s

**Throughput:**
- 1,000 requests/minute sustained
- 10,000 events/hour ingestion
- 100 sources/minute enrichment

**Availability:**
- 99.9% uptime (43 min downtime/month)
- Automatic failover < 30 seconds
- Zero-downtime deployments

## Monitoring & Observability

**Metrics (Prometheus):**
- Request rate, latency, errors
- Database query performance
- Cache hit rate
- Enrichment queue depth
- Connector health status

**Logging (Cloud Logging):**
- Structured JSON logs
- Request/response logging
- Error tracking
- Audit trail

**Alerting:**
- PagerDuty for critical issues
- Slack for warnings
- Email for informational

## Development Roadmap

### ✅ Phase 1: Foundation (Complete)
- Go module, project structure
- Configuration system
- Logging and metrics
- CI/CD pipeline
- Data models

### ✅ Phase 2: Ingestion (Complete)
- Connector interface
- Retry/backoff system
- Deduplication
- Storage abstractions
- Pipeline orchestrator

### ✅ Phase 3: NLP Enrichment (Complete)
- OpenAI integration
- Prompt engineering
- Confidence scoring
- Magnitude estimation
- Entity extraction

### 🔄 Phase 4: MCP Functions (Next)
- Event lifecycle implementation
- get_events MCP function
- Query optimization
- Caching layer

### 🔄 Phase 5: Storage (Following)
- Cloud SQL setup
- Redis integration
- Database migrations
- Query optimization

### 🔄 Phase 6-10: Remaining phases
- Frontend, deployment, compliance, marketing, admin panel

**Total Progress:** 3/10 phases (33%)

## Key Decisions

1. **Single Object API** - Use `EventQuery` object instead of many parameters
2. **Google Cloud** - Cloud Run + Cloud SQL for managed infrastructure
3. **PostgreSQL** - Relational DB with PostGIS for geospatial
4. **OpenAI GPT-4** - Best quality (can downgrade to GPT-3.5 for cost)
5. **Admin Panel** - Web-based configuration instead of config files
6. **MCP Protocol** - Provides LLM integration via Model Context Protocol

## Repository Structure

```
/home/brutus/Documents/GitHub/STRATINT/
├── cmd/
│   └── server/          # Main application entry point
├── internal/
│   ├── config/          # Configuration management
│   ├── enrichment/      # AI-powered enrichment ✅
│   ├── ingestion/       # Data ingestion pipeline ✅
│   ├── logging/         # Structured logging
│   ├── metrics/         # Prometheus metrics
│   ├── models/          # Data models ✅
│   └── server/          # HTTP server
├── docs/
│   ├── DATA_SOURCES.md           # Source research
│   ├── MCP_FUNCTION_DESIGN.md    # API design
│   ├── ADMIN_PANEL_SPEC.md       # Admin panel spec
│   └── GOOGLE_CLOUD_DEPLOYMENT.md # Deployment guide
├── .env.example         # Configuration template
├── go.mod              # Go dependencies
├── Makefile            # Build commands
└── README.md           # Project overview
```

## Contributing

Development follows test-driven approach:
1. Write tests first
2. Implement functionality
3. Ensure all tests pass
4. Document changes
5. Create pull request

**Current test status:** All 95 tests passing ✅

## License

[To be determined]

## Contact

[Project details to be added]
