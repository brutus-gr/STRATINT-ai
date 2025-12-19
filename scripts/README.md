# Scripts

Development and utility scripts for OSINTMCP.

## Development Scripts

### start-dev.sh
Starts the development environment with Docker Compose (PostgreSQL).

```bash
./scripts/start-dev.sh
```

**Note:** Redis is referenced in this script but is not currently used by the application.

## Utility Scripts

Located in `utilities/` subdirectory.

### clear_data.go
Clears events, sources, logs, and errors from the database while preserving configuration.

```bash
cd scripts/utilities
DATABASE_URL="postgres://..." go run clear_data.go
```

**Preserves:**
- tracked_accounts
- openai_config
- threshold_config
- connector_config
- scraper_config

### clear_db.go
Complete database wipe (removes everything).

```bash
cd scripts/utilities
DATABASE_URL="postgres://..." go run clear_db.go
```

### check_sources.go
Checks the status of sources in the database.

```bash
cd scripts/utilities
DATABASE_URL="postgres://..." go run check_sources.go
```

### verify_reddit_feeds.go
Verifies Reddit feed configurations.

```bash
cd scripts/utilities
DATABASE_URL="postgres://..." go run verify_reddit_feeds.go
```

### update_reddit_feeds.go
Updates Reddit feed configurations in the database.

```bash
cd scripts/utilities
DATABASE_URL="postgres://..." go run update_reddit_feeds.go
```

## Usage Notes

All utility scripts require the `DATABASE_URL` environment variable:

```bash
export DATABASE_URL="postgres://user:password@localhost:5432/osintmcp?sslmode=disable"
```

Or pass inline:

```bash
DATABASE_URL="postgres://..." go run script.go
```

## Safety

Most utility scripts modify the database. Review the code before running in production environments.
