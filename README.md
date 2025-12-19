# STRATINT

An AI-powered Open Source Intelligence (OSINT) platform that continuously monitors RSS feeds, enriches data with AI analysis, and provides a real-time intelligence feed with event correlation and deduplication.

## Features

### 🔍 **Intelligent Data Pipeline**
- **RSS Feed Monitoring** - Track multiple news sources with configurable feed URLs
- **Simplified Architecture** - Direct RSS content processing without scraping
- **AI-Powered Enrichment** - OpenAI GPT-4 analysis for entity extraction and summarization
- **Event Correlation** - Automatic deduplication and novel facts detection
- **Threshold-based Publishing** - Configurable confidence and magnitude filters

### 📊 **Admin Dashboard**
- **Pipeline Funnel Visualization** - Real-time bottleneck detection
- **Source Management** - Track and configure RSS feeds
- **Event Moderation** - Review and manage enriched events
- **System Monitoring** - Activity logs, error tracking, and metrics
- **AI Configuration** - OpenAI settings and threshold tuning

### 🎨 **Brutalist Cyberpunk UI**
- Terminal-style event cards with real-time updates
- Dark theme with scan line effects and glitch animations
- Comprehensive filtering (magnitude, confidence, category, time range)
- Responsive design with custom scrollbars

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        RSS FEED SOURCES                         │
└───────────────────────────┬─────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│                   1. RSS INGESTION                              │
│  • Fetch RSS feed content directly                              │
│  • Use feed descriptions as source content                      │
│  • Store to PostgreSQL with status="completed"                  │
└───────────────────────────┬─────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│               2. AI ENRICHMENT (OpenAI GPT-4)                   │
│  • Entity extraction (people, orgs, locations)                  │
│  • Event summarization and categorization                       │
│  • Confidence and magnitude scoring                             │
│  • Create events from RSS sources                               │
└───────────────────────────┬─────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│           3. EVENT CORRELATION & DEDUPLICATION                  │
│  • OpenAI-based similarity analysis                             │
│  • Merge duplicate events                                       │
│  • Detect and extract novel facts                               │
│  • Create "Additional Details" events                           │
└───────────────────────────┬─────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│              4. THRESHOLD FILTERING & PUBLISHING                │
│  • Configurable confidence threshold                            │
│  • Configurable magnitude threshold                             │
│  • Auto-publish qualifying events                               │
│  • Reject low-quality events                                    │
└───────────────────────────┬─────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│                    PUBLISHED EVENT FEED                         │
│  • REST API with filtering                                      │
│  • Real-time web interface                                      │
│  • Admin moderation dashboard                                   │
└─────────────────────────────────────────────────────────────────┘
```

## Quick Start

### Prerequisites

- **Go 1.21+** - Backend server
- **Node.js 18+** - Frontend build
- **PostgreSQL 15+** - Database

### Installation

1. **Clone the repository**
   ```bash
   git clone https://github.com/brutus-gr/STRATINT-ai.git
   cd STRATINT-ai
   ```

2. **Set up PostgreSQL**
   ```bash
   createdb stratint
   export DATABASE_URL="postgres://user:password@localhost:5432/stratint?sslmode=disable"
   ```

3. **Run migrations**
   ```bash
   # Migrations are auto-applied on startup
   # Or manually: psql $DATABASE_URL < migrations/*.sql
   ```

4. **Configure environment**
   ```bash
   cp .env.example .env
   # Edit .env with your settings:
   # - DATABASE_URL
   # - OPENAI_API_KEY
   # - ADMIN_JWT_SECRET
   ```

5. **Build and run backend**
   ```bash
   go build -o server ./cmd/server
   ./server
   # Server starts on http://localhost:8080
   ```

6. **Build and run frontend** (separate terminal)
   ```bash
   cd web
   npm install
   npm run dev
   # Frontend runs on http://localhost:5173
   ```

7. **Access the application**
   - **Main UI**: http://localhost:5173
   - **Admin Panel**: http://localhost:5173/admin (password: see .env)
   - **API Health**: http://localhost:8080/healthz
   - **Metrics**: http://localhost:8080/metrics

### Using the Admin Dashboard

1. Navigate to http://localhost:5173/admin
2. Enter admin password
3. Configure your first RSS source:
   - Go to "SOURCES" tab
   - Click "Add Source"
   - Enter feed URL and settings
4. Trigger scraping:
   - Go to "SCRAPER" or "PIPELINE" tab
   - Click "Scrape Pending Sources"
5. Monitor the pipeline:
   - Check "PIPELINE" tab for funnel visualization
   - Watch bottlenecks and conversion rates
6. View enriched events:
   - Go to main UI (http://localhost:5173)
   - Events appear as they're published

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `DATABASE_URL` | PostgreSQL connection string | Required |
| `OPENAI_API_KEY` | OpenAI API key for enrichment | Required |
| `ADMIN_JWT_SECRET` | Secret key for admin JWT tokens | `change-this-secret` |
| `SERVER_PORT` | HTTP server port | `8080` |
| `LOG_LEVEL` | Logging level (debug/info/warn/error) | `info` |
| `LOG_FORMAT` | Log format (json/text) | `json` |

### Database Configuration

All configuration is stored in PostgreSQL and manageable via the admin UI:

- **OpenAI Settings** - Model, temperature, max tokens
- **Threshold Config** - Min confidence, min magnitude
- **RSS Sources** - Feed URLs, fetch intervals, status
- **Scraper Config** - Worker count, timeout settings

## API Endpoints

### Public API

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/events` | GET | List published events with filtering |
| `/api/events/:id` | GET | Get single event by ID |
| `/api/feed.rss` | GET | RSS 2.0 feed of recent events |
| `/api/stats` | GET | System statistics |
| `/healthz` | GET | Health check |
| `/metrics` | GET | Prometheus metrics |

### Admin API

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/sources` | GET/POST | Manage sources |
| `/api/pipeline/metrics` | GET | Pipeline funnel metrics |
| `/api/scraper/scrape` | POST | Trigger scraping |
| `/api/scraper/status` | GET | Scraping status |
| `/api/openai-config` | GET/PUT | OpenAI configuration |
| `/api/thresholds` | GET/POST | Threshold settings |
| `/api/activity-logs` | GET | Activity logs |
| `/api/ingestion-errors` | GET | Error tracking |

## Key Features Explained

### Split Scraping Architecture

The system separates RSS fetching from content scraping for better performance:

1. **Fast RSS Ingestion** - Fetches feed metadata in seconds
2. **Async Scraping** - Content scraped independently with worker pool
3. **Status Tracking** - Sources have `scrape_status`: pending/in_progress/completed/failed/skipped
4. **Retry Logic** - Failed scrapes can be retried without re-fetching RSS

See: [SCRAPING_SPLIT_IMPLEMENTATION.md](SCRAPING_SPLIT_IMPLEMENTATION.md)

### Event Correlation & Novel Facts

The system intelligently merges duplicate events while preserving new information:

1. **OpenAI Similarity** - Compares new sources against existing events
2. **Smart Merging** - Adds sources to existing events when similar
3. **Novel Facts Detection** - Extracts new information from merged sources
4. **Additional Events** - Creates separate events for novel details

See: [NOVEL_FACTS_IMPLEMENTATION.md](NOVEL_FACTS_IMPLEMENTATION.md)

### Pipeline Funnel Visualization

Real-time monitoring of the processing pipeline:

- **Bottleneck Detection** - Automatically identifies where processing is stuck
- **Conversion Metrics** - Track scrape completion, enrichment, and publish rates
- **Status Breakdown** - Detailed view of sources and events by status
- **Auto-refresh** - Updates every 5 seconds

## Development

### Running Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package
go test ./internal/enrichment/
```

### Database Migrations

Migrations are in `migrations/` and auto-applied on startup. Manual application:

```bash
psql $DATABASE_URL -f migrations/001_initial_schema.sql
psql $DATABASE_URL -f migrations/002_tracked_accounts.sql
# ... etc
```

### Frontend Development

```bash
cd web
npm run dev      # Development server with hot reload
npm run build    # Production build
npm run preview  # Preview production build
npm run lint     # Lint code
```

## Project Structure

```
stratint/
├── cmd/
│   └── server/          # Main server entry point
├── internal/
│   ├── api/             # REST API handlers
│   ├── config/          # Configuration management
│   ├── database/        # PostgreSQL repositories
│   ├── enrichment/      # AI enrichment (OpenAI)
│   ├── eventmanager/    # Event lifecycle management
│   ├── ingestion/       # RSS + scraping pipeline
│   ├── logging/         # Structured logging
│   ├── metrics/         # Prometheus metrics
│   ├── models/          # Data models
│   └── server/          # HTTP server
├── migrations/          # Database migrations
├── web/                 # React + TypeScript frontend
│   ├── src/
│   │   ├── admin/       # Admin dashboard
│   │   ├── components/  # Shared components
│   │   └── pages/       # Main UI pages
│   └── public/
├── docs/                # Design documents
├── archive/             # Historical documentation
└── README.md            # This file
```

## Documentation

- **[ARCHITECTURE.md](ARCHITECTURE.md)** - System architecture overview
- **[SCRAPING_SPLIT_IMPLEMENTATION.md](SCRAPING_SPLIT_IMPLEMENTATION.md)** - Split scraping design
- **[NOVEL_FACTS_IMPLEMENTATION.md](NOVEL_FACTS_IMPLEMENTATION.md)** - Event correlation design
- **[TEST_FAILURE_ANALYSIS.md](TEST_FAILURE_ANALYSIS.md)** - Integration test analysis
- **[CONTRIBUTING.md](CONTRIBUTING.md)** - Contribution guidelines
- **[docs/DEPLOYMENT.md](docs/DEPLOYMENT.md)** - Production deployment guide

### Module Documentation

- **[internal/models/README.md](internal/models/README.md)** - Data models
- **[internal/ingestion/README.md](internal/ingestion/README.md)** - Ingestion pipeline
- **[internal/enrichment/README.md](internal/enrichment/README.md)** - AI enrichment
- **[internal/database/README.md](internal/database/README.md)** - Database layer

## Deployment

The application can be deployed to various platforms:

- **Google Cloud Run** - See [docs/GOOGLE_CLOUD_DEPLOYMENT.md](docs/GOOGLE_CLOUD_DEPLOYMENT.md)
- **Docker** - Dockerfile included (under development)
- **Traditional VPS** - Binary + PostgreSQL + reverse proxy

## Performance Considerations

- **Scraping Speed**: ~5 concurrent workers, ~5-10 seconds per article
- **Enrichment**: ~2-8 seconds per source (OpenAI API dependent)
- **Database**: Indexed for fast querying, supports 10k+ events efficiently
- **Bottleneck**: Typically scraping or OpenAI API rate limits

## Troubleshooting

### Sources stuck in "pending"
- Check Playwright installation: `npx playwright install`
- Verify scraper service is running
- Check network connectivity
- Review error logs

### High scraping failure rate
- Check `scrape_error` field in sources table
- Verify target sites are accessible
- Consider adding domains to skip list
- Increase timeout/retry settings

### Low enrichment rate
- Verify OpenAI API key is valid
- Check OpenAI API quota/rate limits
- Review enrichment prompt effectiveness
- Check event correlation thresholds

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for detailed guidelines on:

- Setting up your development environment
- Code style and standards
- Submitting pull requests
- Testing requirements

Quick start:

1. Fork the repository
2. Create a feature branch
3. Make your changes with tests
4. Submit a pull request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Contact & Support

- **Issues**: Report bugs or request features via [GitHub Issues](https://github.com/brutus-gr/STRATINT-ai/issues)
- **Discussions**: Join community discussions on [GitHub Discussions](https://github.com/brutus-gr/STRATINT-ai/discussions)
- **Documentation**: Full docs available in the [docs/](docs/) directory
- **Security**: Report security vulnerabilities privately via GitHub Security Advisories

For deployment and production support, see [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md).
