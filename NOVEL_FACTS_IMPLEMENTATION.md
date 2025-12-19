# Novel Facts Event Creation Implementation

## Overview
Implemented the nuanced correlation logic where sources with novel facts are both merged with existing events AND create separate events for the new information.

## User Requirement
> "its not as simple as merge or not... sometimes it is merge or not, but sometimes its merge + new event too because it provides new details."

**Example Scenario:**
- **Event A** (from Source A): States facts Z and Y
- **Source B**: States fact Z → Merge with Event A only (no new event)
- **Source C**: States facts Z, Y, AND X → Merge with Event A AND create new Event for fact X

## Implementation Location
File: `/home/brutus/Documents/GitHub/STRATINT/internal/eventmanager/lifecycle.go`

### Changes Made

#### 1. Updated Merge Logic (lines 161-186)
Added detection and handling of novel facts during the merge operation:

```go
} else if bestMatch != nil && corrResult.ShouldMerge {
    m.logger.Info("found similar event via OpenAI correlation",
        "new_event_id", event.ID,
        "existing_event_id", bestMatch.ID,
        "similarity", corrResult.Similarity,
        "has_novel_facts", corrResult.HasNovelFacts,
        "novel_fact_count", len(corrResult.NovelFacts),
    )

    // Add source to existing event (merge operation)
    bestMatch.Sources = append(bestMatch.Sources, event.Sources...)

    // If this source contains novel facts, create a separate event for them
    if corrResult.HasNovelFacts && len(corrResult.NovelFacts) > 0 {
        if err := m.createNovelFactsEvent(ctx, event, bestMatch, corrResult); err != nil {
            m.logger.Error("failed to create novel facts event",
                "error", err,
                "original_event_id", bestMatch.ID,
            )
            // Continue with merge even if novel facts event creation fails
        }
    }

    // Update the existing event with merged sources
    return m.eventRepo.Update(ctx, *bestMatch)
}
```

#### 2. New Helper Method: `createNovelFactsEvent` (lines 212-285)
Creates a separate event containing only the novel facts:

**Key Features:**
- Generates title: `"{Original Event Title} - Additional Details"`
- Summary: Joins all novel facts with semicolons
- ID: `"novel-{original-event-id}"` for traceability
- Magnitude: 70% of original event (supplementary nature)
- Confidence: 90% of original event (derivative information)
- Status: Evaluated against same publication thresholds
- Sources: Includes the source that provided the novel facts
- Inherits: Category, tags, entities, location from original event

**Example Output:**
```
Original Event:
  - Title: "Russia launches missile strikes on Kyiv, 5 civilians killed"
  - ID: evt-12345

Novel Facts Event:
  - Title: "Russia launches missile strikes on Kyiv, 5 civilians killed - Additional Details"
  - ID: novel-evt-12345
  - Summary: "Power station damaged; 15 people injured and evacuated to hospitals"
  - RawContent: "Novel facts discovered in relation to event evt-12345: Same event with additional details about infrastructure damage and the number of injured."
```

## How It Works

### Flow Diagram
```
New Source Arrives
       ↓
OpenAI Correlation Analysis
       ↓
   ShouldMerge?
       ↓
   ┌────Yes────┐
   ↓           ↓
Merge      HasNovelFacts?
Source          ↓
into        ┌───Yes────┐
Existing    ↓          ↓
Event   Create     Update
        Novel      Existing
        Facts      Event
        Event
```

### Correlation Result Structure
The `enrichment.CorrelationResult` from OpenAI contains:
- `Similarity` (0.0-1.0): How closely related the sources are
- `ShouldMerge` (bool): Whether to add source to existing event
- `HasNovelFacts` (bool): Whether source contains new information
- `NovelFacts` ([]string): List of specific new facts identified
- `Reasoning` (string): Explanation of the decision

### Test Coverage
The integration test suite (`test/integration_test.go`) includes test cases that verify:
- ✅ Sources with only existing facts merge without creating new events
- ✅ Sources with novel facts are detected by OpenAI
- ✅ Novel facts are extracted and listed correctly

Example passing test:
```
Test: "Event Correlation - Same Event with Novel Facts"
Existing Event: "Russia launches missile strikes on Kyiv, 5 civilians killed"
New Source: "Kyiv missile attack kills 5, damages power station, 15 injured"

Result:
  Similarity: 0.90
  ShouldMerge: true
  HasNovelFacts: true
  NovelFacts:
    - "Power station damaged"
    - "15 people injured and evacuated to hospitals"
```

## Behavior Examples

### Scenario 1: No Novel Facts (Merge Only)
```
Existing Event: "Kyiv missile attack kills 5 civilians"
New Source: "Russian missiles hit Kyiv, killing 5"

Action: Merge source into existing event
Result: 1 event with 2 sources
```

### Scenario 2: Novel Facts (Merge + New Event)
```
Existing Event: "Kyiv missile attack kills 5 civilians"
New Source: "Kyiv attack kills 5, power station damaged, 15 injured"

Action 1: Merge source into existing event
Action 2: Create novel facts event
Result:
  - Original event with 2 sources
  - New event: "Kyiv missile attack kills 5 civilians - Additional Details"
    Summary: "Power station damaged; 15 injured"
```

### Scenario 3: Different Event (No Merge)
```
Existing Event: "Kyiv missile attack kills 5 civilians"
New Source: "Ukraine drone strikes Russian oil refinery"

Action: Create completely new event
Result: 2 separate unrelated events
```

## Configuration
The novel facts events are subject to the same publication thresholds as regular events:
- Minimum confidence score (from `threshold_config` table)
- Minimum magnitude (from `threshold_config` table)
- Minimum source count (from lifecycle config)

Novel fact events that don't meet thresholds are marked as `rejected` but still stored for potential later retrieval.

## Logging
Enhanced logging provides visibility into the novel facts workflow:
```
INFO: found similar event via OpenAI correlation
  new_event_id=evt-789
  existing_event_id=evt-123
  similarity=0.90
  has_novel_facts=true
  novel_fact_count=2

INFO: created novel facts event
  novel_event_id=novel-evt-789
  related_event_id=evt-123
  novel_facts=["Power station damaged", "15 injured"]

INFO: novel facts event published
  novel_event_id=novel-evt-789
  related_event_id=evt-123
  fact_count=2
```

## Error Handling
If novel facts event creation fails:
- Error is logged with context
- Original merge operation continues
- System remains operational
- No data loss on the merge operation

## Future Enhancements
Potential improvements:
1. Add explicit relationship field in Event model to link novel fact events to originals
2. Create bidirectional links (original → novel, novel → original)
3. Add UI to display related novel facts events
4. Track "lineage" of events for complex correlation chains
5. Allow manual review of novel facts extraction
