# STRATINT Documentation Index

Last Updated: 2025-10-09

## Core Documentation

### [README.md](README.md)
**Main project documentation** - Quick start, installation, architecture overview, API reference, and troubleshooting guide.

### [ARCHITECTURE.md](ARCHITECTURE.md)
**System architecture** - Detailed technical architecture including data flow, components, and design decisions.

## Feature Documentation

### [SCRAPING_SPLIT_IMPLEMENTATION.md](SCRAPING_SPLIT_IMPLEMENTATION.md)
**Split scraping architecture** - Documentation for the two-phase RSS/scraping pipeline that separates fast feed fetching from slow content scraping.

**Key Points:**
- Phase 1: Fast RSS metadata ingestion
- Phase 2: Async content scraping with worker pool
- Status tracking: pending/in_progress/completed/failed/skipped
- Retry logic and error handling

### [NOVEL_FACTS_IMPLEMENTATION.md](NOVEL_FACTS_IMPLEMENTATION.md)
**Event correlation & novel facts** - How the system intelligently merges duplicate events while extracting new information.

**Key Points:**
- OpenAI-based similarity analysis
- Smart merging of duplicate events
- Novel facts detection and extraction
- Creation of "Additional Details" events

### [TEST_FAILURE_ANALYSIS.md](TEST_FAILURE_ANALYSIS.md)
**Test suite analysis** - Analysis of integration test results, including AI-based correlation tests.

**Test Coverage:**
- 22 of 24 tests passing (91.7%)
- Deduplication: 100% passing
- Correlation: 75% passing (AI-based, some subjectivity)
- Confidence: 100% passing
- Magnitude: 100% passing

## Design Specifications

### [docs/ADMIN_PANEL_SPEC.md](docs/ADMIN_PANEL_SPEC.md)
**Admin dashboard specification** - Design spec for the brutalist cyberpunk admin interface.

**Covers:**
- Authentication system
- Source management UI
- Event moderation dashboard
- System monitoring views
- Configuration management

### [docs/FRONTEND_DESIGN.md](docs/FRONTEND_DESIGN.md)
**UI design system** - Complete design specification for the brutalist cyberpunk aesthetic.

**Includes:**
- Color palette and typography
- Component library
- Animation effects (glitch, scan lines)
- Responsive design patterns

### [docs/DATA_SOURCES.md](docs/DATA_SOURCES.md)
**OSINT data sources research** - Research on available data sources, APIs, rate limits, and terms of service.

**Sources Covered:**
- Twitter/X API
- Telegram
- Reddit
- 4chan
- Government RSS feeds
- News APIs

### [docs/MCP_FUNCTION_DESIGN.md](docs/MCP_FUNCTION_DESIGN.md)
**MCP function design** - Design for Model Context Protocol (MCP) integration.

**Note:** MCP integration is planned but not currently the primary focus. The system provides a REST API instead.

### [docs/GOOGLE_CLOUD_DEPLOYMENT.md](docs/GOOGLE_CLOUD_DEPLOYMENT.md)
**Google Cloud deployment guide** - Complete deployment architecture and cost estimates for Google Cloud Platform.

**Covers:**
- Cloud Run configuration
- Cloud SQL (PostgreSQL) setup
- Secret Manager integration
- Cloud Logging and Monitoring
- CI/CD with Cloud Build
- Cost estimates and scaling

## Module Documentation

### [internal/models/README.md](internal/models/README.md)
**Data models package** - Documentation for core data structures.

**Models:**
- Event - Processed intelligence events
- Source - Raw OSINT data
- Entity - Extracted named entities
- EventQuery - Query/filter parameters

### [internal/ingestion/README.md](internal/ingestion/README.md)
**Ingestion pipeline** - RSS fetching, scraping, and deduplication.

**Components:**
- RSSConnector - Feed fetching
- PlaywrightScraper - Content extraction
- ScraperService - Async scraping worker pool
- Repositories - Data storage interfaces

### [internal/enrichment/README.md](internal/enrichment/README.md)
**AI enrichment system** - OpenAI integration for event analysis.

**Components:**
- OpenAIClient - GPT-4 integration
- EventCorrelator - Similarity analysis
- PromptTemplates - OSINT-optimized prompts
- MockEnricher - Testing implementation

### [internal/database/README.md](internal/database/README.md)
**Database layer** - PostgreSQL repositories and migrations.

**Repositories:**
- PostgresSourceRepository
- PostgresEventRepository
- TrackedAccountRepository
- IngestionErrorRepository
- ThresholdRepository

### [internal/cache/README.md](internal/cache/README.md)
**Caching layer** - Currently unused, Redis integration stub.

**Status:** Not currently implemented. Database queries are fast enough without caching.

## Archived Documentation

Historical session notes and status reports have been moved to `archive/session-docs/`. These are kept for reference but are no longer actively maintained.

## Quick Navigation

**Getting Started:**
1. Read [README.md](README.md) for installation and quick start
2. Review [ARCHITECTURE.md](ARCHITECTURE.md) for system understanding
3. Check feature docs for specific implementations

**Developing:**
1. Module READMEs in `internal/*/README.md` for package details
2. [SCRAPING_SPLIT_IMPLEMENTATION.md](SCRAPING_SPLIT_IMPLEMENTATION.md) for pipeline understanding
3. [NOVEL_FACTS_IMPLEMENTATION.md](NOVEL_FACTS_IMPLEMENTATION.md) for correlation logic

**Deploying:**
1. [docs/GOOGLE_CLOUD_DEPLOYMENT.md](docs/GOOGLE_CLOUD_DEPLOYMENT.md) for GCP deployment
2. [README.md](README.md) for configuration and environment setup

**Design Reference:**
1. [docs/FRONTEND_DESIGN.md](docs/FRONTEND_DESIGN.md) for UI patterns
2. [docs/ADMIN_PANEL_SPEC.md](docs/ADMIN_PANEL_SPEC.md) for admin features
3. [docs/DATA_SOURCES.md](docs/DATA_SOURCES.md) for OSINT sources

## Contributing to Documentation

When adding new features:
1. Update relevant module README
2. Add feature documentation in root (like SCRAPING_SPLIT_IMPLEMENTATION.md)
3. Update this index
4. Update README.md if it affects installation/usage

## Documentation Standards

- **Feature Docs**: Markdown with code examples, architecture diagrams (ASCII art), and usage examples
- **Module Docs**: Package-level README with API reference and examples
- **Session Notes**: Move to `archive/session-docs/` when done
- **Specs**: Keep in `docs/` folder for reference

## Need Help?

- **Installation Issues**: See [README.md](README.md) troubleshooting section
- **API Reference**: See [README.md](README.md) API endpoints section
- **Architecture Questions**: See [ARCHITECTURE.md](ARCHITECTURE.md)
- **Specific Features**: See feature documentation files
