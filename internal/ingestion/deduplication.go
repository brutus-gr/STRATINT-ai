package ingestion

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/STRATINT/stratint/internal/models"
)

// Deduplicator identifies and filters duplicate content.
type Deduplicator interface {
	// IsNew checks if a source is new (not a duplicate).
	IsNew(source models.Source) bool

	// Mark records a source as seen.
	Mark(source models.Source)

	// Cleanup removes old entries from the deduplication cache.
	Cleanup(olderThan time.Time)
}

// ContentFingerprint represents a unique identifier for content.
type ContentFingerprint struct {
	ContentHash string // SHA-256 hash of normalized content
	URL         string // Source URL
	SourceType  models.SourceType
	CreatedAt   time.Time
}

// MemoryDeduplicator implements in-memory deduplication using fingerprints.
type MemoryDeduplicator struct {
	fingerprints map[string]ContentFingerprint
	window       time.Duration // How long to keep fingerprints
}

// NewMemoryDeduplicator creates a new in-memory deduplicator.
func NewMemoryDeduplicator(window time.Duration) *MemoryDeduplicator {
	return &MemoryDeduplicator{
		fingerprints: make(map[string]ContentFingerprint),
		window:       window,
	}
}

// IsNew checks if a source has been seen before.
func (d *MemoryDeduplicator) IsNew(source models.Source) bool {
	hash := ComputeContentHash(source)
	_, exists := d.fingerprints[hash]
	return !exists
}

// Mark records a source as seen.
func (d *MemoryDeduplicator) Mark(source models.Source) {
	hash := ComputeContentHash(source)
	d.fingerprints[hash] = ContentFingerprint{
		ContentHash: hash,
		URL:         source.URL,
		SourceType:  source.Type,
		CreatedAt:   time.Now(),
	}
}

// Cleanup removes fingerprints older than the specified time.
func (d *MemoryDeduplicator) Cleanup(olderThan time.Time) {
	for hash, fp := range d.fingerprints {
		if fp.CreatedAt.Before(olderThan) {
			delete(d.fingerprints, hash)
		}
	}
}

// Size returns the number of fingerprints in the cache.
func (d *MemoryDeduplicator) Size() int {
	return len(d.fingerprints)
}

// ComputeContentHash generates a fingerprint hash for a source.
func ComputeContentHash(source models.Source) string {
	// Normalize content before hashing
	normalized := NormalizeContent(source.RawContent)

	// Include source-specific identifiers for exact duplicate detection
	data := fmt.Sprintf("%s|%s|%s|%s",
		source.Type,
		source.URL,
		source.Author,
		normalized,
	)

	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// NormalizeContent standardizes content for comparison.
func NormalizeContent(content string) string {
	// Convert to lowercase
	normalized := strings.ToLower(content)

	// Remove extra whitespace
	normalized = regexp.MustCompile(`\s+`).ReplaceAllString(normalized, " ")

	// Trim leading/trailing whitespace
	normalized = strings.TrimSpace(normalized)

	// Remove common URL patterns (to catch content shared across platforms)
	normalized = regexp.MustCompile(`https?://[^\s]+`).ReplaceAllString(normalized, "[URL]")

	// Remove mentions and hashtags for better fuzzy matching
	normalized = regexp.MustCompile(`@\w+`).ReplaceAllString(normalized, "[MENTION]")
	normalized = regexp.MustCompile(`#\w+`).ReplaceAllString(normalized, "[TAG]")

	// Remove common punctuation that doesn't affect meaning
	normalized = regexp.MustCompile(`[.,!?;:""''""]+`).ReplaceAllString(normalized, "")

	return normalized
}

// SimilarityScore calculates how similar two sources are (0.0 = different, 1.0 = identical).
func SimilarityScore(s1, s2 models.Source) float64 {
	// Exact URL match
	if s1.URL != "" && s1.URL == s2.URL {
		return 1.0
	}

	// Same author and very similar content
	if s1.Author != "" && s1.Author == s2.Author {
		contentSimilarity := jaccardSimilarity(s1.RawContent, s2.RawContent)
		if contentSimilarity > 0.8 {
			return contentSimilarity
		}
	}

	// Pure content similarity
	return jaccardSimilarity(s1.RawContent, s2.RawContent)
}

// jaccardSimilarity computes Jaccard similarity coefficient between two strings.
func jaccardSimilarity(s1, s2 string) float64 {
	// Tokenize strings
	tokens1 := tokenize(NormalizeContent(s1))
	tokens2 := tokenize(NormalizeContent(s2))

	if len(tokens1) == 0 && len(tokens2) == 0 {
		return 1.0
	}
	if len(tokens1) == 0 || len(tokens2) == 0 {
		return 0.0
	}

	// Create sets
	set1 := make(map[string]bool)
	set2 := make(map[string]bool)

	for _, token := range tokens1 {
		set1[token] = true
	}
	for _, token := range tokens2 {
		set2[token] = true
	}

	// Calculate intersection and union
	intersection := 0
	for token := range set1 {
		if set2[token] {
			intersection++
		}
	}

	union := len(set1) + len(set2) - intersection

	if union == 0 {
		return 0.0
	}

	return float64(intersection) / float64(union)
}

// tokenize splits a string into words.
func tokenize(s string) []string {
	return regexp.MustCompile(`\w+`).FindAllString(s, -1)
}

// DeduplicationStats tracks deduplication metrics.
type DeduplicationStats struct {
	TotalProcessed int
	Duplicates     int
	Unique         int
	DuplicateRate  float64
}

// DeduplicationFilter wraps a deduplicator to track statistics.
type DeduplicationFilter struct {
	dedup Deduplicator
	stats DeduplicationStats
}

// NewDeduplicationFilter creates a new filter with stats tracking.
func NewDeduplicationFilter(dedup Deduplicator) *DeduplicationFilter {
	return &DeduplicationFilter{
		dedup: dedup,
	}
}

// Filter removes duplicates from a list of sources.
func (f *DeduplicationFilter) Filter(sources []models.Source) []models.Source {
	unique := make([]models.Source, 0, len(sources))

	for _, source := range sources {
		f.stats.TotalProcessed++

		if f.dedup.IsNew(source) {
			f.dedup.Mark(source)
			unique = append(unique, source)
			f.stats.Unique++
		} else {
			f.stats.Duplicates++
		}
	}

	// Update duplicate rate
	if f.stats.TotalProcessed > 0 {
		f.stats.DuplicateRate = float64(f.stats.Duplicates) / float64(f.stats.TotalProcessed)
	}

	return unique
}

// GetStats returns the current deduplication statistics.
func (f *DeduplicationFilter) GetStats() DeduplicationStats {
	return f.stats
}

// ResetStats clears the statistics counters.
func (f *DeduplicationFilter) ResetStats() {
	f.stats = DeduplicationStats{}
}
