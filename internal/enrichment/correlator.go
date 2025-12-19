package enrichment

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/STRATINT/stratint/internal/models"
	openai "github.com/sashabaranov/go-openai"
)

// EventCorrelator analyzes relationships between sources and events using AI.
type EventCorrelator struct {
	client  *openai.Client
	config  OpenAIConfig
	prompts *PromptTemplates
	logger  *slog.Logger
}

// NewEventCorrelator creates a new event correlator.
func NewEventCorrelator(client *openai.Client, config OpenAIConfig, prompts *PromptTemplates, logger *slog.Logger) *EventCorrelator {
	return &EventCorrelator{
		client:  client,
		config:  config,
		prompts: prompts,
		logger:  logger,
	}
}

// CorrelationResult describes how a new source relates to an existing event.
type CorrelationResult struct {
	// Similarity score from 0.0 (unrelated) to 1.0 (identical)
	Similarity float64 `json:"similarity"`

	// Whether this source should be added to the existing event
	ShouldMerge bool `json:"should_merge"`

	// Whether this source contains novel facts not in the existing event
	HasNovelFacts bool `json:"has_novel_facts"`

	// List of novel facts found in the new source
	NovelFacts []string `json:"novel_facts"`

	// Explanation of the correlation decision
	Reasoning string `json:"reasoning"`
}

// AnalyzeCorrelation determines if a new source should be merged with an existing event,
// create a new event, or both.
func (c *EventCorrelator) AnalyzeCorrelation(ctx context.Context, newSource models.Source, existingEvent models.Event) (*CorrelationResult, error) {
	// Build correlation analysis prompt
	prompt := c.buildCorrelationPrompt(newSource, existingEvent)

	// Call OpenAI with JSON mode
	apiCtx, cancel := context.WithTimeout(ctx, time.Duration(c.config.Timeout)*time.Second)
	defer cancel()

	resp, err := c.client.CreateChatCompletion(apiCtx, openai.ChatCompletionRequest{
		Model:               c.config.Model,
		MaxCompletionTokens: 1000, // Shorter response for correlation
		ResponseFormat: &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONObject,
		},
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: c.prompts.CorrelationSystemPrompt,
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: prompt,
			},
		},
	})

	if err != nil {
		return nil, fmt.Errorf("openai correlation analysis failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no completion choices returned")
	}

	// Parse JSON response
	var result CorrelationResult
	if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &result); err != nil {
		return nil, fmt.Errorf("failed to parse correlation result: %w", err)
	}

	c.logger.Debug("analyzed source correlation",
		"source_id", newSource.ID,
		"event_id", existingEvent.ID,
		"similarity", result.Similarity,
		"should_merge", result.ShouldMerge,
		"has_novel_facts", result.HasNovelFacts,
		"novel_fact_count", len(result.NovelFacts))

	return &result, nil
}

// FindBestMatch finds the most relevant existing event for a new source.
// Returns the best matching event and its correlation result, or nil if no good match.
func (c *EventCorrelator) FindBestMatch(ctx context.Context, newSource models.Source, existingEvents []models.Event) (*models.Event, *CorrelationResult, error) {
	if len(existingEvents) == 0 {
		return nil, nil, nil
	}

	// For efficiency, we'll check recent events first (within last 7 days)
	recentEvents := filterRecentEvents(existingEvents, 7*24*time.Hour)

	var bestEvent *models.Event
	var bestResult *CorrelationResult
	var bestSimilarity float64 = 0.0

	// Analyze correlation with recent events
	for i := range recentEvents {
		event := &recentEvents[i]

		result, err := c.AnalyzeCorrelation(ctx, newSource, *event)
		if err != nil {
			c.logger.Warn("failed to analyze correlation", "error", err, "event_id", event.ID)
			continue
		}

		// Track best match
		if result.Similarity > bestSimilarity {
			bestSimilarity = result.Similarity
			bestResult = result
			bestEvent = event
		}

		// If we find a very high match (>0.8), we can stop early
		if result.Similarity > 0.8 {
			c.logger.Debug("found high-confidence match early", "similarity", result.Similarity)
			break
		}
	}

	// Only return a match if similarity is above threshold (0.6)
	if bestSimilarity >= 0.6 && bestResult.ShouldMerge {
		return bestEvent, bestResult, nil
	}

	return nil, nil, nil
}

// buildCorrelationPrompt creates the prompt for correlation analysis.
func (c *EventCorrelator) buildCorrelationPrompt(newSource models.Source, existingEvent models.Event) string {
	// Format existing event facts
	existingFacts := "- " + formatKeyFacts(existingEvent)

	prompt := fmt.Sprintf(`=== CORRELATION ANALYSIS REQUEST ===

You are analyzing whether a new intelligence source should be merged with an existing event.

EXISTING EVENT:
Title: %s
Summary: %s
Category: %s
Key Facts:
%s

NEW SOURCE:
Title: %s
URL: %s
Published: %s
Content Preview:
%s

=== ANALYSIS TASK ===

Compare the new source against the existing event and determine:

1. SIMILARITY (0.0-1.0): How closely related are they?
   - 1.0 = Same event, same facts (duplicate)
   - 0.8-0.9 = Same event, minor variations or updates
   - 0.6-0.7 = Related event, significant overlap
   - 0.4-0.5 = Tangentially related (e.g., reactions, responses, consequences)
   - 0.2-0.3 = Same topic but different events
   - 0.0-0.1 = Unrelated

2. SHOULD_MERGE: Should this source be added to the existing event?
   - true if similarity >= 0.6 AND sources discuss the SAME core event/incident
   - false if discussing different events, even if related topic or cause/effect

   DO NOT MERGE if the new source is:
   - A REACTION or RESPONSE to the event (condemnations, statements ABOUT the event)
   - A CONSEQUENCE or follow-on event (investigations, arrests, policy changes)
   - A DIFFERENT INCIDENT in the same location or same topic area
   - General commentary or analysis rather than factual reporting

   DO MERGE if the new source is:
   - Additional details about the SAME incident
   - Updated information (casualty counts, damage assessments)
   - Different perspective on the SAME event
   - Conflicting claims about the SAME incident (e.g., disputes about attribution, casualty numbers, causes)
   - Denials or counter-claims that directly contradict facts about the incident itself

   KEY DISTINCTION: Statements ABOUT an event (reactions) ≠ Statements about the FACTS of an event (conflicting claims)

3. NOVEL FACTS: Does the new source contain facts NOT in the existing event?
   - Identify specific new information, claims, or developments
   - Ignore stylistic variations or rephrasing of same facts
   - Focus on substantive new information
   - If the new source is a reaction/response, those reactions ARE novel facts

Output ONLY valid JSON in this format:
{
  "similarity": 0.85,
  "should_merge": true,
  "has_novel_facts": true,
  "novel_facts": [
    "Specific new fact 1 not in existing event",
    "Specific new fact 2 not in existing event"
  ],
  "reasoning": "Brief explanation of your decision"
}`,
		existingEvent.Title,
		existingEvent.Summary,
		existingEvent.Category,
		existingFacts,
		newSource.Title,
		newSource.URL,
		newSource.PublishedAt.Format(time.RFC3339),
		truncateText(newSource.RawContent, 2000), // Limit content for token efficiency
	)

	return prompt
}

// formatKeyFacts formats event key facts for display.
// Since Event model doesn't have KeyFacts field, we use summary instead.
func formatKeyFacts(event models.Event) string {
	if event.Summary != "" {
		return event.Summary
	}
	return "(No detailed information available)"
}

// truncateText truncates text to a maximum length.
func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "... [truncated]"
}

// filterRecentEvents returns events published within the specified duration.
func filterRecentEvents(events []models.Event, window time.Duration) []models.Event {
	cutoff := time.Now().Add(-window)
	recent := make([]models.Event, 0)

	for _, event := range events {
		if event.Timestamp.After(cutoff) {
			recent = append(recent, event)
		}
	}

	return recent
}
