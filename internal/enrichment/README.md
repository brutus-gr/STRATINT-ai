# Enrichment Package

AI-powered enrichment system for processing raw OSINT sources into structured, analyzed intelligence events.

## Overview

The enrichment package transforms raw sources into high-value intelligence by:
- Summarizing content with AI analysis
- Extracting and normalizing named entities
- Calculating multi-factor confidence scores
- Estimating event magnitude/severity
- Categorizing events

## Components

### OpenAIClient (`client.go`)
Production enricher using GPT-4 Turbo for analysis.

```go
client := NewOpenAIClient(apiKey, DefaultOpenAIConfig())
event, err := client.Enrich(ctx, source)
```

### MockEnricher (`mock.go`)
Rule-based enricher for testing without API calls.

```go
enricher := NewMockEnricher()
event, err := enricher.Enrich(ctx, source)
```

### PromptTemplates (`prompts.go`)
OSINT-optimized prompts for AI analysis with structured output.

### ConfidenceScorer (`scoring.go`)
6-factor weighted confidence algorithm (0-1 scale):
- Source Credibility (30%)
- Source Type (20%)
- Entity Confidence (15%)
- Content Quality (15%)
- Recency (10%)
- Metadata Richness (10%)

### MagnitudeEstimator (`scoring.go`)
Event severity scoring (0-10 scale) based on category, entities, engagement, urgency, and scope.

### EntityExtractor (`entities.go`)
Named entity recognition with normalization and reference data mapping.

## Usage

### Basic Enrichment

```go
// With OpenAI
config := DefaultOpenAIConfig()
enricher := NewOpenAIClient(os.Getenv("OPENAI_API_KEY"), config)

event, err := enricher.Enrich(ctx, source)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Event: %s (Magnitude: %.1f, Confidence: %.2f)\n", 
    event.Title, event.Magnitude, event.Confidence.Score)
```

### Testing Without API

```go
// Use MockEnricher for tests
enricher := NewMockEnricher()
event, err := enricher.Enrich(ctx, source)
// No API calls, instant results
```

## Configuration

```bash
# OpenAI
OPENAI_API_KEY=sk-...
OPENAI_MODEL=gpt-4-turbo-preview  # or gpt-3.5-turbo for cost savings
OPENAI_TEMPERATURE=0.3
OPENAI_MAX_TOKENS=2000
```

## Testing

```bash
go test ./internal/enrichment/... -v
```

**Test Coverage:** 23 tests covering confidence scoring, magnitude estimation, entity extraction, and category inference.

## Cost Considerations

**GPT-4 Turbo:** ~$0.01 per source (2K tokens avg)
- 1,000 sources/day = $10/day = $300/month

**GPT-3.5 Turbo:** ~$0.001 per source
- 1,000 sources/day = $1/day = $30/month

## See Also

- [ARCHITECTURE.md](../../ARCHITECTURE.md) - System architecture
- [/docs/DATA_SOURCES.md](../../docs/DATA_SOURCES.md) - Source research
