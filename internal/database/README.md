# Database Package

PostgreSQL database layer with PostGIS for OSINTMCP.

## Overview

This package provides PostgreSQL implementations of the repository interfaces, with support for:
- Full-text search
- Geospatial queries (PostGIS)
- JSONB for flexible metadata
- Connection pooling
- Transaction support

## Components

### Database Connection (`database.go`)
Manages PostgreSQL connection with pooling and health checks.

```go
cfg := database.DefaultConfig()
cfg.URL = os.Getenv("DATABASE_URL")

db, err := database.Connect(ctx, cfg)
if err != nil {
    log.Fatal(err)
}
defer db.Close()
```

### PostgresEventRepository (`postgres_event_repository.go`)
Full event CRUD with relationships (sources, entities).

**Features:**
- Transaction-based operations
- Automatic relationship management
- Complex query building with filters
- Full-text search support
- Geospatial queries

### PostgresSourceRepository (`postgres_source_repository.go`)
Source storage with deduplication support.

**Features:**
- Batch inserts with prepared statements
- Content hash-based deduplication
- Retention policy methods
- Type-based filtering

## Schema

The database schema is defined in `/migrations/001_initial_schema.sql`:

**Tables:**
- `events` - Processed OSINT events
- `sources` - Raw source data
- `entities` - Named entities
- `event_sources` - Many-to-many junction
- `event_entities` - Many-to-many junction

**Key Features:**
- PostGIS GEOGRAPHY for locations
- Full-text search indexes (pg_trgm)
- JSONB for flexible metadata
- Composite indexes for common queries
- Automatic timestamp updates

## Usage

### Creating Events

```go
repo := database.NewPostgresEventRepository(db)

event := models.Event{
    ID:        "evt-123",
    Title:     "Military Exercise Announced",
    Summary:   "...",
    Magnitude: 7.5,
    Confidence: models.Confidence{Score: 0.85},
    Status:    models.EventStatusPublished,
}

err := repo.Create(ctx, event)
```

### Querying Events

```go
query := models.EventQuery{
    MinMagnitude:  &magnitude,
    Categories:    []models.Category{models.CategoryMilitary},
    Page:          1,
    Limit:         20,
    SortBy:        models.SortByMagnitude,
    SortOrder:     models.SortOrderDesc,
}

response, err := repo.Query(ctx, query)
```

### Batch Source Insert

```go
sourceRepo := database.NewPostgresSourceRepository(db)

sources := []models.Source{ /* ... */ }
err := sourceRepo.StoreBatch(ctx, sources)
```

## Migrations

Run migrations using your preferred tool:

```bash
# Using psql
psql $DATABASE_URL < migrations/001_initial_schema.sql

# Using golang-migrate
migrate -database $DATABASE_URL -path migrations up
```

## Configuration

Environment variables:

```bash
DATABASE_URL=postgresql://user:pass@host:5432/osintmcp?sslmode=require
DATABASE_MAX_CONNECTIONS=100
DATABASE_MAX_IDLE_CONNECTIONS=10
```

## Performance

**Indexes:**
- Events: timestamp, magnitude, confidence, category, status
- Full-text: title + summary (GIN index)
- Geospatial: location (GIST index)
- Composite: category+magnitude+timestamp

**Query Optimization:**
- Prepared statements for batch operations
- Transaction batching
- Index-only scans where possible
- Connection pooling (100 max, 10 idle)

## Health Checks

```go
if err := database.HealthCheck(ctx, db); err != nil {
    log.Error("database unhealthy", "error", err)
}

stats := database.Stats(db)
log.Info("database stats", "stats", stats)
```

## See Also

- [/internal/cache/README.md](../cache/README.md) - Redis caching layer
- [ARCHITECTURE.md](../../ARCHITECTURE.md) - System architecture
- [/docs/GOOGLE_CLOUD_DEPLOYMENT.md](../../docs/GOOGLE_CLOUD_DEPLOYMENT.md) - Cloud SQL setup
