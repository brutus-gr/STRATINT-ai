# Threshold-Based Forecast Refactor Plan

## Overview

This document outlines the plan to refactor forecasts to support a parent-child model where a single "parent forecast" can define multiple thresholds (e.g., -25%, -20%, -15%, etc.) and automatically manage "sub-forecasts" for each threshold.

**Current State**: Each threshold is a separate independent forecast
**Desired State**: One parent forecast manages multiple threshold-based sub-forecasts

## Design Decisions

### 1. Parent-Child Relationship Model

```
Parent Forecast (Template)
├── Defines: name, base proposition, categories, headline_count, iterations, models, context_urls
├── Thresholds: [-25, -20, -15, -10, -5, 0, 5, 10, 15, 20, 25] (configurable array)
└── Auto-manages Child Forecasts
    ├── Child 1: "SP500 >= 25%" (threshold: 25, direction: up, operator: >=)
    ├── Child 2: "SP500 >= 20%" (threshold: 20, direction: up, operator: >=)
    ├── ... (one child per threshold)
    └── Child N: "SP500 <= -25%" (threshold: 25, direction: down, operator: <=)
```

**Key Properties**:
- Parent forecasts are **templates** - they cannot be executed directly
- Child forecasts are **executable** - they inherit all settings from parent + threshold
- Changes to parent cascade to all children (except completed runs)
- Deleting parent deletes all children (CASCADE)
- Thresholds are stored as a simple array of numbers in the parent

### 2. Threshold Configuration Format

**Parent Forecast Model**:
```go
type ParentForecast struct {
    ID                 string    `json:"id"`
    Name               string    `json:"name"`
    BaseProposition    string    `json:"base_proposition"`  // Template: "Will {ASSET} change by {THRESHOLD} or more?"
    Thresholds         []float64 `json:"thresholds"`        // [-25, -20, -15, ..., 15, 20, 25]
    ConfidenceInterval float64   `json:"confidence_interval"`
    Categories         []string  `json:"categories"`
    HeadlineCount      int       `json:"headline_count"`
    Iterations         int       `json:"iterations"`
    ContextURLs        []string  `json:"context_urls"`
    Active             bool      `json:"active"`
    CreatedAt          time.Time `json:"created_at"`
    UpdatedAt          time.Time `json:"updated_at"`
}
```

**Child Forecast Enhancement**:
```go
type Forecast struct {
    ID                 string    `json:"id"`
    ParentForecastID   *string   `json:"parent_forecast_id"` // NULL for standalone forecasts
    Name               string    `json:"name"`
    Proposition        string    `json:"proposition"`
    ConfidenceInterval float64   `json:"confidence_interval"`
    Categories         []string  `json:"categories"`
    HeadlineCount      int       `json:"headline_count"`
    Iterations         int       `json:"iterations"`
    ContextURLs        []string  `json:"context_urls"`
    ThresholdPercent   *float64  `json:"threshold_percent"`
    ThresholdDirection *string   `json:"threshold_direction"`
    ThresholdOperator  *string   `json:"threshold_operator"`
    Active             bool      `json:"active"`
    CreatedAt          time.Time `json:"created_at"`
    UpdatedAt          time.Time `json:"updated_at"`
}
```

### 3. Proposition Template System

The parent's `base_proposition` uses placeholders:
- `{THRESHOLD}` - replaced with absolute value (e.g., "15")
- `{DIRECTION}` - replaced with "increase"/"decrease" or "up"/"down"
- `{SIGN}` - replaced with "+" or "-"

**Example Templates**:
```
"Will the S&P 500 increase by {THRESHOLD}% or more in the next year?"
"Will gold change by {SIGN}{THRESHOLD}% or more by September 2026?"
```

**Generated Propositions**:
```
Threshold: 15  → "Will the S&P 500 increase by 15% or more in the next year?"
Threshold: -20 → "Will the S&P 500 decrease by 20% or more in the next year?"
```

### 4. Database Schema Changes

**New Table: parent_forecasts**
```sql
CREATE TABLE parent_forecasts (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    base_proposition TEXT NOT NULL,
    thresholds REAL[] NOT NULL,  -- Array of threshold percentages
    confidence_interval REAL NOT NULL DEFAULT 0.95,
    categories TEXT[] NOT NULL DEFAULT '{}',
    headline_count INTEGER NOT NULL DEFAULT 500,
    iterations INTEGER NOT NULL DEFAULT 1,
    context_urls TEXT[] NOT NULL DEFAULT '{}',
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_parent_forecasts_active ON parent_forecasts(active);
```

**Modify Table: forecasts**
```sql
ALTER TABLE forecasts
    ADD COLUMN parent_forecast_id TEXT REFERENCES parent_forecasts(id) ON DELETE CASCADE;

CREATE INDEX idx_forecasts_parent_id ON forecasts(parent_forecast_id);
```

**New Table: parent_forecast_models**
```sql
CREATE TABLE parent_forecast_models (
    id TEXT PRIMARY KEY,
    parent_forecast_id TEXT NOT NULL REFERENCES parent_forecasts(id) ON DELETE CASCADE,
    provider TEXT NOT NULL,
    model_name TEXT NOT NULL,
    api_key TEXT NOT NULL,
    weight REAL NOT NULL DEFAULT 1.0,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_parent_forecast_models_parent_id ON parent_forecast_models(parent_forecast_id);
```

### 5. Lifecycle Management

**Creating a Parent Forecast**:
1. User creates parent with thresholds: `[-25, -20, -15, -10, -5, 0, 5, 10, 15, 20, 25]`
2. System creates 11 child forecasts, one per threshold
3. Each child inherits: name (+ suffix), proposition (rendered), categories, headline_count, iterations, context_urls, models
4. Each child gets threshold-specific: threshold_percent, threshold_direction, threshold_operator

**Updating a Parent Forecast**:
1. Detect which thresholds were added/removed/unchanged
2. **Added thresholds**: Create new child forecasts
3. **Removed thresholds**: Delete corresponding child forecasts (CASCADE deletes runs)
4. **Unchanged thresholds**: Update child forecasts with new parent settings
5. Update timestamp on parent

**Deleting a Parent Forecast**:
1. CASCADE deletes all child forecasts
2. CASCADE deletes all child forecast runs
3. Normalized forecasts referencing this parent need to be updated (see below)

**Edge Cases**:
- Prevent deletion if any child has runs in 'running' status
- Option to "orphan" children (set parent_forecast_id to NULL) instead of deletion
- Validation: thresholds array must not be empty
- Validation: thresholds should be sorted and unique

## API Design

### Parent Forecast Endpoints

**POST /api/admin/parent-forecasts**
```json
{
  "name": "S&P 500 1-Year Change",
  "base_proposition": "Will the S&P 500 change by {SIGN}{THRESHOLD}% or more in the next year?",
  "thresholds": [-25, -20, -15, -10, -5, 5, 10, 15, 20, 25],
  "confidence_interval": 0.80,
  "categories": ["economic", "market"],
  "headline_count": 500,
  "iterations": 3,
  "context_urls": ["https://api.example.com/fred", "https://api.example.com/spy-risk"],
  "models": [
    {
      "provider": "openai",
      "model_name": "gpt-4o",
      "api_key": "...",
      "weight": 1.0
    },
    {
      "provider": "anthropic",
      "model_name": "claude-sonnet-4.5",
      "api_key": "...",
      "weight": 1.5
    }
  ]
}
```

**Response**: Returns parent forecast + array of created child forecasts

**GET /api/admin/parent-forecasts**
- Lists all parent forecasts

**GET /api/admin/parent-forecasts/:id**
- Returns parent forecast + all child forecasts + models

**PUT /api/admin/parent-forecasts/:id**
- Updates parent and syncs children (add/remove/update as needed)

**DELETE /api/admin/parent-forecasts/:id**
- Deletes parent and all children (or orphans them if query param set)

### Child Forecast Endpoints

**GET /api/admin/forecasts**
- Add query param `?include_children=false` to exclude parent-managed forecasts
- Default includes all forecasts

**GET /api/admin/forecasts/:id**
- Returns child forecast + parent info if applicable

**POST /api/admin/forecasts/:id/execute**
- Execute a specific child forecast (unchanged behavior)
- Can still execute standalone forecasts

**Note**: Direct creation/update of child forecasts via API is **disabled** if they have a parent_forecast_id

### Execution Flow

**Unchanged**: Forecasts are still executed individually
- User clicks "Execute" on a specific child forecast
- Execution logic remains the same
- Each child can have independent runs/results

**Future Enhancement**: Bulk execution endpoint
```
POST /api/admin/parent-forecasts/:id/execute-all
```
Would queue execution of all child forecasts

## Normalized Forecasts Integration

### Current Model
```go
type NormalizedForecast struct {
    ID          uuid.UUID `json:"id"`
    Name        string    `json:"name"`
    ForecastIDs []string  `json:"forecast_ids"` // Array of individual forecast IDs
    ...
}
```

### Enhanced Model
```go
type NormalizedForecast struct {
    ID               uuid.UUID `json:"id"`
    Name             string    `json:"name"`
    ParentForecastID *string   `json:"parent_forecast_id"` // Reference to parent
    ForecastIDs      []string  `json:"forecast_ids"`       // Still support manual selection
    ...
}
```

**Behavior**:
- If `parent_forecast_id` is set: automatically use ALL child forecasts from that parent
- If `forecast_ids` is set: use specific forecasts (supports mixed or standalone)
- Cannot set both (validation error)

**Advantages**:
- Normalized forecast automatically includes new thresholds when parent is updated
- Cleaner UI: "Normalize S&P 500 Parent Forecast" instead of selecting 20+ individual forecasts

### Database Schema Update
```sql
ALTER TABLE normalized_forecasts
    ADD COLUMN parent_forecast_id TEXT REFERENCES parent_forecasts(id) ON DELETE SET NULL;

-- Add constraint: either parent_forecast_id OR forecast_ids, not both
ALTER TABLE normalized_forecasts
    ADD CONSTRAINT check_forecast_reference
    CHECK (
        (parent_forecast_id IS NOT NULL AND forecast_ids = '{}') OR
        (parent_forecast_id IS NULL AND forecast_ids != '{}')
    );
```

## Migration Strategy

### Phase 1: Schema Migration (Zero Downtime)
1. Create `parent_forecasts` table
2. Create `parent_forecast_models` table
3. Add `parent_forecast_id` column to `forecasts` (nullable)
4. Add `parent_forecast_id` column to `normalized_forecasts` (nullable)

### Phase 2: Data Migration (Offline/Maintenance)

**Option A: Manual Migration**
- Admin manually creates parent forecasts for existing threshold groups
- System provides a helper API to "group" existing forecasts into a parent

**Option B: Automatic Migration**
- Detect forecast groups by name pattern (e.g., "SP500 +15%", "SP500 +10%", etc.)
- Auto-create parent forecasts
- Link existing forecasts as children
- Requires careful validation

**Recommendation**: Start with Option A, provide tooling for Option B later

### Phase 3: Deprecation
- Existing standalone forecasts continue to work (parent_forecast_id = NULL)
- New threshold-based forecasts must use parent model
- Eventually migrate all old forecasts

## Testing Strategy

### Unit Tests
- Threshold parsing and proposition rendering
- Child forecast creation from parent
- Update synchronization (add/remove thresholds)
- Cascade deletion behavior

### Integration Tests
- Full CRUD lifecycle for parent forecasts
- Execute child forecasts
- Normalized forecast aggregation with parent reference

### Edge Cases to Test
1. Empty thresholds array → validation error
2. Duplicate thresholds → deduplicate or error
3. Updating parent while child is running → prevent or queue
4. Deleting parent with running children → prevent
5. Orphaning children vs cascade delete
6. Normalized forecast with deleted parent → graceful degradation

## Implementation Phases

### Phase 1: Foundation (1-2 days)
- [ ] Create database migrations
- [ ] Update models (ParentForecast, enhance Forecast)
- [ ] Create ParentForecastRepository
- [ ] Implement proposition template rendering

### Phase 2: API Layer (2-3 days)
- [ ] Create ParentForecastHandler
- [ ] CRUD endpoints for parent forecasts
- [ ] Child sync logic (add/remove/update)
- [ ] Update existing forecast endpoints to handle parent context

### Phase 3: Integration (1-2 days)
- [ ] Update normalized forecast logic to support parent reference
- [ ] Update normalized forecast API endpoints
- [ ] Frontend changes (not covered in this backend plan)

### Phase 4: Migration & Testing (2-3 days)
- [ ] Write migration scripts/tooling
- [ ] Comprehensive testing
- [ ] Documentation
- [ ] Deploy and monitor

**Total Estimated Time**: 6-10 days for backend

## Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| Breaking existing forecasts | Keep parent_forecast_id nullable; maintain backward compatibility |
| Accidental deletion of children | Implement soft delete or confirmation flow |
| Performance with many children | Index parent_forecast_id; batch operations |
| Race conditions during update | Use database transactions; lock parent during sync |
| Frontend complexity | Provide clear API contracts; comprehensive error messages |

## Success Criteria

1. ✅ Can create a parent forecast with 15+ thresholds in one API call
2. ✅ Updating parent thresholds automatically adds/removes children
3. ✅ Deleting parent cleanly removes all children
4. ✅ Child forecasts execute independently
5. ✅ Normalized forecasts can reference parent and auto-include all children
6. ✅ Existing standalone forecasts continue to work
7. ✅ Zero data loss during migration

## Open Questions

1. **Naming convention for children?**
   - Suggestion: `{parent_name} [{SIGN}{threshold}%]`
   - Example: "S&P 500 1-Year Change [+15%]"

2. **Should we support mixed normalized forecasts?**
   - Can a normalized forecast include BOTH a parent reference AND manual forecast_ids?
   - Recommendation: No, keep it simple (either/or)

3. **Bulk execution API?**
   - Do we need POST /parent-forecasts/:id/execute-all?
   - Recommendation: Add later if needed

4. **Webhook/notification when children change?**
   - Should we notify when thresholds are added/removed?
   - Recommendation: Audit log only for now

## Next Steps

1. Review and approve this plan
2. Create GitHub issues for each phase
3. Begin Phase 1: Database schema + models
4. Iterative implementation with frequent reviews
