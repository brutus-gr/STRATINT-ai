package enrichment

import (
	"context"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/STRATINT/stratint/internal/models"
)

// CredibilityCache caches domain credibility scores to avoid excessive LLM calls.
type CredibilityCache struct {
	cache    map[string]cacheEntry
	mu       sync.RWMutex
	enricher *OpenAIClient
	ttl      time.Duration
}

type cacheEntry struct {
	score     float64
	timestamp time.Time
}

// NewCredibilityCache creates a new credibility cache with TTL.
func NewCredibilityCache(enricher *OpenAIClient, ttl time.Duration) *CredibilityCache {
	return &CredibilityCache{
		cache:    make(map[string]cacheEntry),
		enricher: enricher,
		ttl:      ttl,
	}
}

// GetCredibility returns cached credibility or fetches from LLM.
func (c *CredibilityCache) GetCredibility(ctx context.Context, sourceURL string, sourceType models.SourceType) (float64, error) {
	domain := extractDomain(sourceURL)
	if domain == "" {
		// Fallback to default if we can't parse domain
		return c.enricher.getDefaultCredibility(sourceType), nil
	}

	// Check cache first
	c.mu.RLock()
	entry, exists := c.cache[domain]
	c.mu.RUnlock()

	if exists && time.Since(entry.timestamp) < c.ttl {
		return entry.score, nil
	}

	// Fetch from LLM
	score, err := c.enricher.AssessSourceCredibility(ctx, sourceURL, sourceType)
	if err != nil {
		return c.enricher.getDefaultCredibility(sourceType), err
	}

	// Cache the result
	c.mu.Lock()
	c.cache[domain] = cacheEntry{
		score:     score,
		timestamp: time.Now(),
	}
	c.mu.Unlock()

	return score, nil
}

// extractDomain extracts the domain from a URL.
func extractDomain(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}

	host := parsed.Host
	// Remove port if present
	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}

	return host
}
