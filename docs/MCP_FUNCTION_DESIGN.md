# MCP Function Design Analysis

## Current Proposed Signature

```
get_events(searchQuery, sinceTimestamp, minMagnitude, minConfidence, page, limit)
```

## Analysis: ❌ Not Perfect - Needs Significant Improvement

### Issues with Current Design

1. **Too Many Individual Parameters** - Violates clean API design (6 parameters is borderline)
2. **Lacks Flexibility** - Missing critical filters we've already built into `EventQuery`
3. **No Time Range End** - Only `sinceTimestamp` but no `untilTimestamp` for ranges
4. **Missing Category Filters** - Can't filter by Military, Cyber, etc.
5. **No Source Type Filtering** - Can't focus on Twitter, Government sources, etc.
6. **No Sorting Control** - Can't sort by magnitude or confidence, only implicit timestamp
7. **Missing Entity Filters** - Can't filter by entities like countries or military units
8. **No Tag Support** - Can't leverage the tag system
9. **No Status Filter** - Can't query draft vs published events

### Our Existing Model is Much Better

We already have a comprehensive `EventQuery` model in `internal/models/query.go`:

```go
type EventQuery struct {
    SearchQuery    string         // Full-text search
    SinceTimestamp *time.Time     // Start of time range
    UntilTimestamp *time.Time     // End of time range ✅ MISSING IN PROPOSAL
    MinMagnitude   *float64       // Magnitude threshold
    MinConfidence  *float64       // Confidence threshold
    Categories     []Category     // ✅ MISSING IN PROPOSAL
    SourceTypes    []SourceType   // ✅ MISSING IN PROPOSAL
    Tags           []string       // ✅ MISSING IN PROPOSAL
    EntityTypes    []EntityType   // ✅ MISSING IN PROPOSAL
    Status         *EventStatus   // ✅ MISSING IN PROPOSAL
    Page           int            // Pagination
    Limit          int            // Results per page
    SortBy         EventSortField // ✅ MISSING IN PROPOSAL
    SortOrder      SortOrder      // ✅ MISSING IN PROPOSAL
}
```

## Recommended Approaches

### Option 1: Single Object Parameter (✅ RECOMMENDED)

```typescript
get_events(query: EventQuery): EventResponse
```

**Advantages:**
- ✅ All 13+ parameters in one object
- ✅ Easy to extend without breaking API
- ✅ Optional parameters are clear
- ✅ Already implemented in our models
- ✅ Consistent with REST API best practices
- ✅ JSON serialization friendly

**MCP Tool Definition:**
```json
{
  "name": "get_events",
  "description": "Query OSINT events with comprehensive filtering",
  "inputSchema": {
    "type": "object",
    "properties": {
      "search_query": {
        "type": "string",
        "description": "Full-text search across title and summary"
      },
      "since_timestamp": {
        "type": "string",
        "format": "date-time",
        "description": "Start of time range (RFC3339)"
      },
      "until_timestamp": {
        "type": "string",
        "format": "date-time",
        "description": "End of time range (RFC3339)"
      },
      "min_magnitude": {
        "type": "number",
        "minimum": 0,
        "maximum": 10,
        "description": "Minimum event magnitude (0-10)"
      },
      "min_confidence": {
        "type": "number",
        "minimum": 0,
        "maximum": 1,
        "description": "Minimum confidence score (0-1)"
      },
      "categories": {
        "type": "array",
        "items": {
          "type": "string",
          "enum": ["geopolitics", "military", "economic", "cyber", "disaster", "terrorism", "diplomacy", "intelligence", "humanitarian", "other"]
        },
        "description": "Filter by event categories"
      },
      "source_types": {
        "type": "array",
        "items": {
          "type": "string",
          "enum": ["twitter", "telegram", "reddit", "4chan", "glp", "government", "news_media", "blog", "other"]
        },
        "description": "Filter by source types"
      },
      "tags": {
        "type": "array",
        "items": {"type": "string"},
        "description": "Filter by tags"
      },
      "entity_types": {
        "type": "array",
        "items": {
          "type": "string",
          "enum": ["country", "city", "region", "person", "organization", "military_unit", "vessel", "weapon_system", "facility", "event", "other"]
        },
        "description": "Filter by entity types present"
      },
      "status": {
        "type": "string",
        "enum": ["pending", "enriched", "published", "archived", "rejected"],
        "description": "Filter by event status"
      },
      "page": {
        "type": "integer",
        "minimum": 1,
        "default": 1,
        "description": "Page number (1-indexed)"
      },
      "limit": {
        "type": "integer",
        "minimum": 1,
        "maximum": 200,
        "default": 20,
        "description": "Results per page"
      },
      "sort_by": {
        "type": "string",
        "enum": ["timestamp", "magnitude", "confidence", "created_at", "updated_at"],
        "default": "timestamp",
        "description": "Field to sort by"
      },
      "sort_order": {
        "type": "string",
        "enum": ["asc", "desc"],
        "default": "desc",
        "description": "Sort direction"
      }
    },
    "required": []
  }
}
```

**Usage Examples:**

```javascript
// Simple query - breaking news in last hour
await get_events({
  since_timestamp: new Date(Date.now() - 3600000).toISOString(),
  min_magnitude: 7.0,
  sort_by: "magnitude"
})

// Complex query - cyber attacks on US, high confidence
await get_events({
  search_query: "United States",
  categories: ["cyber"],
  min_confidence: 0.7,
  entity_types: ["country"],
  page: 1,
  limit: 50
})

// Military events from reliable sources
await get_events({
  categories: ["military"],
  source_types: ["government", "news_media"],
  min_confidence: 0.6,
  sort_by: "confidence",
  sort_order: "desc"
})

// Time-boxed analysis
await get_events({
  since_timestamp: "2024-01-01T00:00:00Z",
  until_timestamp: "2024-01-31T23:59:59Z",
  categories: ["geopolitics", "diplomacy"],
  min_magnitude: 5.0
})
```

### Option 2: Positional + Options Object (Alternative)

```typescript
get_events(
  searchQuery?: string,
  options?: {
    timeRange?: { since?: string, until?: string },
    thresholds?: { minMagnitude?: number, minConfidence?: number },
    filters?: { categories?: string[], sourceTypes?: string[], tags?: string[] },
    pagination?: { page?: number, limit?: number },
    sorting?: { sortBy?: string, sortOrder?: string }
  }
): EventResponse
```

**Advantages:**
- ✅ Common parameter (search) is easily accessible
- ✅ Grouping makes intent clear
- ❌ More complex to implement
- ❌ Nested structure harder to document

### Option 3: Keep Simple, Add More Functions (❌ NOT RECOMMENDED)

```typescript
get_events(search, page, limit)
get_events_filtered(filters)
get_events_by_category(category, page, limit)
get_events_by_time_range(since, until, page, limit)
```

**Disadvantages:**
- ❌ API explosion - too many functions
- ❌ Can't combine filters easily
- ❌ Harder to maintain
- ❌ Poor user experience

## Final Recommendation

### ✅ Use Option 1: Single EventQuery Object

**Signature:**
```go
func GetEvents(ctx context.Context, query EventQuery) (*EventResponse, error)
```

**MCP Tool Name:** `get_events`

**MCP Input:** JSON object mapping to `EventQuery` struct

**MCP Output:** JSON object mapping to `EventResponse` struct

### Why This is Perfect:

1. **Backwards Compatible** - All parameters optional with sensible defaults
2. **Future-Proof** - Add new filters without breaking changes
3. **Already Implemented** - `EventQuery` model exists with validation
4. **Clean** - One parameter, unlimited flexibility
5. **Testable** - Easy to construct queries for testing
6. **Documented** - JSON schema provides full documentation
7. **Type-Safe** - Strong typing in Go, validated at runtime
8. **Composable** - Users can build queries incrementally

### Implementation Checklist

- [ ] Create MCP tool definition with full JSON schema
- [ ] Implement handler that accepts `EventQuery` JSON
- [ ] Add input validation using `EventQuery.Validate()`
- [ ] Implement database query builder from `EventQuery`
- [ ] Add comprehensive examples to documentation
- [ ] Create client library helpers for common queries
- [ ] Add query templates for frequent use cases

## Common Query Templates

### Template 1: Breaking News Feed
```json
{
  "since_timestamp": "2024-01-01T00:00:00Z",
  "min_magnitude": 7.0,
  "sort_by": "timestamp",
  "sort_order": "desc",
  "limit": 50
}
```

### Template 2: High Confidence Military Events
```json
{
  "categories": ["military"],
  "min_confidence": 0.8,
  "source_types": ["government", "news_media"],
  "sort_by": "magnitude",
  "limit": 100
}
```

### Template 3: Cyber Threat Intelligence
```json
{
  "categories": ["cyber", "intelligence"],
  "entity_types": ["organization", "country"],
  "min_magnitude": 5.0,
  "tags": ["breach", "attack", "malware"]
}
```

### Template 4: Geopolitical Analysis Dashboard
```json
{
  "categories": ["geopolitics", "diplomacy", "military"],
  "since_timestamp": "2024-01-01T00:00:00Z",
  "min_confidence": 0.6,
  "sort_by": "magnitude"
}
```

## Performance Considerations

### Database Indexes Needed

Based on `EventQuery` fields, create indexes on:
- `timestamp` (DESC) - Primary sorting field
- `magnitude` (DESC) - Common filter/sort
- `confidence` (DESC) - Common filter/sort
- `category` - Enum filter
- `status` - Lifecycle filter
- Composite: `(category, magnitude, timestamp)`
- Composite: `(status, timestamp)`
- Full-text: `title, summary` - For search_query

### Query Optimization

```sql
-- Efficient query with proper indexes
SELECT * FROM events 
WHERE 
  status = 'published'
  AND magnitude >= 7.0
  AND confidence >= 0.7
  AND category = ANY($1::category_enum[])
  AND timestamp >= $2
  AND timestamp <= $3
ORDER BY magnitude DESC, timestamp DESC
LIMIT 20 OFFSET 0;

-- Use prepared statements for frequently used filters
-- Enable query plan caching for common patterns
```

### Caching Strategy

```go
// Cache key format: query_hash:page:limit
func buildCacheKey(query EventQuery) string {
    hash := hashQuery(query)
    return fmt.Sprintf("events:%s:%d:%d", hash, query.Page, query.Limit)
}

// Cache hot queries for 1-5 minutes
// Invalidate on new events published
```

## Migration Path

If we want to maintain backwards compatibility with simpler signature:

```go
// Legacy wrapper (deprecated)
func GetEventsSimple(search string, since time.Time, minMag, minConf float64, page, limit int) (*EventResponse, error) {
    query := EventQuery{
        SearchQuery:    search,
        SinceTimestamp: &since,
        MinMagnitude:   &minMag,
        MinConfidence:  &minConf,
        Page:           page,
        Limit:          limit,
    }
    return GetEvents(context.Background(), query)
}
```

## Conclusion

**The original signature is NOT perfect.** It lacks critical functionality we've already designed and limits future extensibility.

**Recommended:** Use `get_events(query: EventQuery)` with the full 13-parameter object for maximum flexibility, maintainability, and user experience.
