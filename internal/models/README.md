# OSINTMCP Models Package

This package contains the core data models for the OSINTMCP system, defining the structure for OSINT events, sources, entities, and queries.

## Overview

The models package provides type-safe representations for:
- **Events**: Processed OSINT intelligence with metadata, confidence scores, and relationships
- **Sources**: Origin platforms and attribution for raw OSINT data
- **Entities**: Extracted named entities (countries, persons, organizations, military units, etc.)
- **Queries**: Request/response models for the MCP API

## Core Models

### Event (`event.go`)

The `Event` struct represents a fully processed OSINT intelligence event:

```go
type Event struct {
    ID          string
    Timestamp   time.Time
    Title       string
    Summary     string
    Magnitude   float64      // 0-10 importance scale
    Confidence  Confidence
    Category    Category
    Entities    []Entity
    Sources     []Source
    Location    *Location
    Status      EventStatus
}
```

**Key Features:**
- `Magnitude`: 0-10 scale representing event importance/severity
- `Confidence`: Structured reliability assessment (0-1 score with reasoning)
- `IsPublishable()`: Helper method to check if event meets quality thresholds

**Event Status Lifecycle:**
1. `pending` - Raw data ingested, not yet processed
2. `enriched` - NLP processing completed
3. `published` - Available via API
4. `archived` - Moved to cold storage
5. `rejected` - Failed validation

**Categories:**
- Geopolitics, Military, Economic, Cyber, Disaster, Terrorism, Diplomacy, Intelligence, Humanitarian, Other

### Source (`source.go`)

The `Source` struct represents an OSINT data origin:

```go
type Source struct {
    ID              string
    Type            SourceType
    URL             string
    Author          string
    PublishedAt     time.Time
    RawContent      string
    Metadata        SourceMetadata
    Credibility     float64  // 0-1 reliability scale
}
```

**Supported Source Types:**
- Twitter, Telegram, Reddit, 4chan, Godlike Productions (GLP)
- Government, News Media, Blog, Other

**Platform-Specific Metadata:**
- **Twitter**: Tweet ID, retweet/like counts, hashtags, mentions
- **Telegram**: Channel ID, message ID, view count
- **Reddit**: Subreddit, post/comment ID, score
- **4chan**: Board, thread ID, post number

**Helper Methods:**
- `GetDisplayName()`: Human-readable source identifier
- `IsRecent(duration)`: Check if published within time window
- `IsCredible()`: Check if meets minimum credibility threshold (≥0.4)

### Entity (`entity.go`)

The `Entity` struct represents extracted named entities:

```go
type Entity struct {
    ID             string
    Type           EntityType
    Name           string
    NormalizedName string  // Canonical form for deduplication
    Aliases        []string
    Confidence     float64  // NLP extraction confidence
    Attributes     EntityAttrs
}
```

**Entity Types:**
- Geographic: Country, City, Region
- Person, Organization, Military Unit
- Vessel, Weapon System, Facility, Event

**Type-Specific Attributes:**
- **Geographic**: Country code, lat/long coordinates
- **Person**: Title, affiliation
- **Organization**: Type, headquarters
- **Military**: Branch, command level
- **Vessel**: Type, IMO number, flag state
- **Weapon System**: Class, designation

**Helper Methods:**
- `IsPrimaryEntity()`: Returns true for high-confidence core entity types
- `GetDisplayIdentifier()`: Best identifier for display

### Query (`query.go`)

The `EventQuery` struct defines filters and pagination for retrieving events:

```go
type EventQuery struct {
    SearchQuery    string
    SinceTimestamp *time.Time
    MinMagnitude   *float64
    MinConfidence  *float64
    Categories     []Category
    SourceTypes    []SourceType
    Tags           []string
    Page           int
    Limit          int
    SortBy         EventSortField
    SortOrder      SortOrder
}
```

**Features:**
- Automatic validation with defaults (page=1, limit=20)
- Limit capped at 200 items per page
- Sort by: timestamp, magnitude, confidence, created_at, updated_at
- `GetOffset()`: Calculate database offset for pagination

**Response Format:**
```go
type EventResponse struct {
    Events  []Event
    Page    int
    Limit   int
    Total   int
    HasMore bool
}
```

## Confidence System

The `Confidence` struct provides structured reliability assessment:

```go
type Confidence struct {
    Score       float64         // 0-1 numeric score
    Level       ConfidenceLevel // Human-readable level
    Reasoning   string          // Explanation
    SourceCount int             // Corroborating sources
}
```

**Confidence Levels:**
- **Low** (0.0-0.3): Single source, unverified
- **Medium** (0.3-0.6): Multiple sources, partial corroboration
- **High** (0.6-0.85): Strong corroboration, credible sources
- **Verified** (0.85-1.0): Official confirmation, multiple high-credibility sources

Use `DeriveLevel()` to automatically calculate the level from the numeric score.

## Usage Examples

### Creating an Event

```go
event := models.Event{
    ID:        "evt-123",
    Timestamp: time.Now(),
    Title:     "Military Exercise Announced",
    Summary:   "Country X announced joint military exercises...",
    Magnitude: 6.5,
    Confidence: models.Confidence{
        Score:       0.75,
        SourceCount: 3,
        Reasoning:   "Confirmed by official government sources",
    },
    Category: models.CategoryMilitary,
    Status:   models.EventStatusEnriched,
}

// Check if publishable
if event.IsPublishable() {
    // Event meets quality thresholds
}
```

### Querying Events

```go
query := models.EventQuery{
    SearchQuery:   "Ukraine",
    MinMagnitude:  &[]float64{5.0}[0],
    MinConfidence: &[]float64{0.6}[0],
    Categories:    []models.Category{models.CategoryMilitary},
    Page:          1,
    Limit:         50,
    SortBy:        models.SortByTimestamp,
    SortOrder:     models.SortOrderDesc,
}

// Validate and get offset
if err := query.Validate(); err != nil {
    // Handle validation error
}
offset := query.GetOffset()
```

### Working with Entities

```go
entity := models.Entity{
    Type:           models.EntityTypeCountry,
    Name:           "United States",
    NormalizedName: "United States",
    Confidence:     0.95,
    Attributes: models.EntityAttrs{
        CountryCode: "US",
        WikidataID:  "Q30",
    },
}

if entity.IsPrimaryEntity() {
    displayName := entity.GetDisplayIdentifier()
    // Use in UI
}
```

## Testing

All models have comprehensive unit tests. Run tests with:

```bash
go test ./internal/models/...
```

Test coverage includes:
- Validation logic
- Helper methods
- Edge cases (zero values, boundary conditions)
- Full lifecycle scenarios

## Design Principles

1. **Type Safety**: Strong typing with enums for categories, statuses, and types
2. **Validation**: Built-in validation methods with sensible defaults
3. **Extensibility**: Flexible metadata structures for platform-specific data
4. **Traceability**: Full source attribution and confidence reasoning
5. **Performance**: Efficient pagination and filtering support

## Future Enhancements

Planned additions for subsequent phases:
- Graph relationships between entities
- Time-series data for trending analysis
- Advanced deduplication fingerprinting
- Webhook event streaming models
- Mobile-optimized response formats
