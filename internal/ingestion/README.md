# Ingestion Package

The ingestion package provides a robust, modular framework for collecting OSINT data from multiple sources with retry logic, deduplication, and content normalization.

## Architecture

```
┌─────────────┐
│  Pipeline   │ ← Orchestrates all connectors
└──────┬──────┘
       │
       ├─────────┬─────────┬─────────┐
       ▼         ▼         ▼         ▼
  Connector  Connector Connector  ...
  (Twitter)  (Telegram) (Reddit)
       │         │         │
       └─────────┴─────────┴────────┐
                                    ▼
                            ┌───────────────┐
                            │ Deduplicator  │
                            └───────┬───────┘
                                    ▼
                            ┌───────────────┐
                            │   Storage     │
                            └───────────────┘
```

## Components

### 1. Connector Interface (`connector.go`)

All source connectors must implement the `Connector` interface:

```go
type Connector interface {
    Name() string
    SourceType() models.SourceType
    Fetch(ctx context.Context, since time.Time) ([]models.Source, error)
    FetchByID(ctx context.Context, id string) (*models.Source, error)
    HealthCheck(ctx context.Context) error
    RateLimit() RateLimitConfig
}
```

**Key Features:**
- Standardized interface for all data sources
- Health checking for monitoring
- Rate limit configuration per connector
- Status tracking with `BaseConnector`

### 2. Retry Mechanism (`retry.go`)

Implements exponential backoff retry logic with jitter:

```go
policy := RetryPolicy{
    MaxRetries:     3,
    InitialBackoff: 1 * time.Second,
    MaxBackoff:     30 * time.Second,
    BackoffFactor:  2.0,
    Jitter:         true,
}

err := Retry(ctx, policy, func() error {
    return someFailableOperation()
})
```

**Features:**
- Exponential backoff with configurable factor
- Maximum backoff cap
- Jitter to prevent thundering herd
- Context cancellation support
- Retryable error types

**Example:**
```go
// Mark an error as retryable
if networkError {
    return NewRetryableError(err)
}

// Suggest specific retry delay (e.g., from rate limit header)
if rateLimited {
    return NewRetryableErrorWithDelay(err, 60*time.Second)
}
```

### 3. Deduplication (`deduplication.go`)

Content-based duplicate detection using normalized hashing:

```go
dedup := NewMemoryDeduplicator(24 * time.Hour)

if dedup.IsNew(source) {
    dedup.Mark(source)
    // Process new source
}
```

**Normalization Process:**
1. Lowercase conversion
2. Whitespace normalization
3. URL replacement with `[URL]` placeholder
4. @mention replacement with `[MENTION]`
5. #hashtag replacement with `[TAG]`
6. Punctuation removal

**Features:**
- SHA-256 content hashing
- Jaccard similarity scoring
- Time-windowed caching
- Batch filtering with statistics

**Example:**
```go
filter := NewDeduplicationFilter(dedup)
uniqueSources := filter.Filter(sources)
stats := filter.GetStats()
// stats.DuplicateRate, stats.Unique, stats.Duplicates
```

### 4. Storage Interfaces (`storage.go`)

Abstract storage layer for sources and events:

```go
type SourceRepository interface {
    StoreRaw(ctx context.Context, source models.Source) error
    StoreBatch(ctx context.Context, sources []models.Source) error
    GetByID(ctx context.Context, id string) (*models.Source, error)
    ListRecent(ctx context.Context, since time.Time, limit int) ([]models.Source, error)
}

type EventRepository interface {
    Create(ctx context.Context, event models.Event) error
    Query(ctx context.Context, query models.EventQuery) (*models.EventResponse, error)
    UpdateStatus(ctx context.Context, id string, status models.EventStatus) error
}
```

**Implementations:**
- `MemorySourceRepository` - In-memory storage for testing/development
- `MemoryEventRepository` - In-memory event storage
- `PostgresSourceRepository` - Production PostgreSQL implementation ✅
- `PostgresEventRepository` - Production PostgreSQL implementation ✅

### 5. Pipeline (`pipeline.go`)

Orchestrates data ingestion from multiple connectors:

```go
pipeline := NewPipeline(
    connectors,
    sourceRepo,
    eventRepo,
    logger,
    DefaultPipelineConfig(),
)

// Start periodic ingestion
go pipeline.Start(ctx)

// Manual fetch
pipeline.FetchOne(ctx, "twitter-connector", time.Now().Add(-1*time.Hour))

// Health checks
health := pipeline.HealthCheck(ctx)
```

**Configuration:**
```go
type PipelineConfig struct {
    PollInterval      time.Duration  // How often to poll connectors
    BatchSize         int            // Max sources per batch
    EnableDedup       bool           // Enable deduplication
    DedupWindow       time.Duration  // How long to track duplicates
    ConcurrentFetches int            // Max parallel connector fetches
    RetryPolicy       RetryPolicy    // Retry configuration
}
```

**Features:**
- Concurrent connector fetching with semaphore
- Automatic retry with backoff
- Deduplication integration
- Periodic polling
- Graceful shutdown
- Per-connector health monitoring

## Usage Examples

### Creating a Connector

```go
type TwitterConnector struct {
    *BaseConnector
    client *twitter.Client
}

func (c *TwitterConnector) Name() string {
    return "twitter"
}

func (c *TwitterConnector) SourceType() models.SourceType {
    return models.SourceTypeTwitter
}

func (c *TwitterConnector) Fetch(ctx context.Context, since time.Time) ([]models.Source, error) {
    tweets, err := c.client.Search(ctx, "OSINT", since)
    if err != nil {
        return nil, NewRetryableError(err)
    }
    
    sources := make([]models.Source, len(tweets))
    for i, tweet := range tweets {
        sources[i] = c.convertTweet(tweet)
    }
    
    return sources, nil
}

func (c *TwitterConnector) RateLimit() RateLimitConfig {
    return RateLimitConfig{
        RequestsPerMinute: 15,
        BurstSize:         5,
        Cooldown:          15 * time.Minute,
    }
}
```

### Setting Up Pipeline

```go
// Create connectors
twitterConn := NewTwitterConnector(cfg.Twitter)
telegramConn := NewTelegramConnector(cfg.Telegram)
redditConn := NewRedditConnector(cfg.Reddit)

// Create repositories
sourceRepo := NewMemorySourceRepository()
eventRepo := NewMemoryEventRepository()

// Create pipeline
config := PipelineConfig{
    PollInterval:      5 * time.Minute,
    BatchSize:         100,
    EnableDedup:       true,
    DedupWindow:       24 * time.Hour,
    ConcurrentFetches: 3,
    RetryPolicy:       DefaultRetryPolicy(),
}

pipeline := NewPipeline(
    []Connector{twitterConn, telegramConn, redditConn},
    sourceRepo,
    eventRepo,
    logger,
    config,
)

// Start pipeline
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

go pipeline.Start(ctx)
```

### Manual Deduplication

```go
// Create deduplicator with 24-hour window
dedup := NewMemoryDeduplicator(24 * time.Hour)

// Check if new
if dedup.IsNew(source) {
    // Mark as seen
    dedup.Mark(source)
    
    // Process source
    processSource(source)
}

// Periodic cleanup (remove old entries)
ticker := time.NewTicker(1 * time.Hour)
for range ticker.C {
    cutoff := time.Now().Add(-24 * time.Hour)
    dedup.Cleanup(cutoff)
}
```

### Custom Retry Logic

```go
// Create custom retry policy
policy := RetryPolicy{
    MaxRetries:     5,
    InitialBackoff: 500 * time.Millisecond,
    MaxBackoff:     60 * time.Second,
    BackoffFactor:  2.5,
    Jitter:         true,
}

// Wrap function with retry
fetchWithRetry := WithRetry(policy, func(ctx context.Context) error {
    return connector.Fetch(ctx, since)
})

// Execute
if err := fetchWithRetry(ctx); err != nil {
    log.Error("fetch failed after retries", "error", err)
}
```

## Testing

The package includes comprehensive tests:

```bash
# Run all ingestion tests
go test ./internal/ingestion/... -v

# Run specific test
go test ./internal/ingestion -run TestRetry_Success

# Run with coverage
go test ./internal/ingestion/... -cover
```

**Test Coverage:**
- Retry logic with various failure scenarios
- Deduplication with content normalization
- Storage operations (CRUD)
- Query filtering and pagination
- Context cancellation handling

## Performance Considerations

### Deduplication Memory Usage
- Each fingerprint: ~200 bytes
- 10,000 sources cached: ~2 MB
- 100,000 sources: ~20 MB
- Recommend 24-48 hour window max

### Concurrent Fetching
- Default: 3 concurrent connectors
- Consider rate limits when increasing
- Each connector spawns goroutine
- Semaphore prevents overload

### Batch Operations
- `StoreBatch` more efficient than individual stores
- Recommended batch size: 50-100 sources
- Balance memory usage vs database calls

## Error Handling

### Retryable Errors
Use `NewRetryableError` for:
- Network timeouts
- Rate limit errors (with delay hint)
- Temporary API failures
- 5xx server errors

### Non-Retryable Errors
Return regular errors for:
- Authentication failures (401)
- Not found errors (404)
- Validation errors (400)
- Permanent API deprecation

```go
if resp.StatusCode == 401 {
    return nil, fmt.Errorf("authentication failed")
}
if resp.StatusCode == 429 {
    retryAfter := parseRetryAfter(resp.Header)
    return nil, NewRetryableErrorWithDelay(
        fmt.Errorf("rate limited"), 
        retryAfter,
    )
}
```

## Monitoring Metrics

Track these metrics for production:

```go
// Per connector
- fetch_duration_seconds (histogram)
- fetch_total (counter)
- fetch_errors_total (counter)
- sources_fetched_total (counter)
- duplicates_total (counter)

// Pipeline level
- pipeline_running (gauge)
- connector_health (gauge per connector)
- dedup_cache_size (gauge)
```

## Future Enhancements

### Planned Features
- [ ] Rate limit token bucket implementation
- [ ] Distributed deduplication with Redis
- [ ] PostgreSQL repository implementation
- [ ] Connector plugin system
- [ ] Real-time streaming support (WebSocket/SSE)
- [ ] Connector auto-disable on persistent failures
- [ ] Content-based similarity clustering
- [ ] Incremental backfill support

### Performance Optimizations
- [ ] Bloom filter pre-screening for dedup
- [ ] Parallel batch storage
- [ ] Connector priority queuing
- [ ] Adaptive polling intervals
- [ ] Database connection pooling

## See Also

- [`/docs/DATA_SOURCES.md`](../../docs/DATA_SOURCES.md) - API research and strategies
- [`/internal/models/`](../models/) - Data model definitions
- [`/internal/config/`](../config/) - Configuration management
