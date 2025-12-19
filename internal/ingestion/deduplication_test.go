package ingestion

import (
	"strings"
	"testing"
	"time"

	"github.com/STRATINT/stratint/internal/models"
)

func TestMemoryDeduplicator_IsNew(t *testing.T) {
	dedup := NewMemoryDeduplicator(1 * time.Hour)

	source := models.Source{
		ID:         "src-1",
		Type:       models.SourceTypeTwitter,
		URL:        "https://twitter.com/user/status/123",
		RawContent: "Breaking news about event X",
		Author:     "NewsSource",
	}

	// First time should be new
	if !dedup.IsNew(source) {
		t.Error("source should be new on first check")
	}

	// Mark as seen
	dedup.Mark(source)

	// Second time should not be new
	if dedup.IsNew(source) {
		t.Error("source should not be new after marking")
	}
}

func TestMemoryDeduplicator_Cleanup(t *testing.T) {
	dedup := NewMemoryDeduplicator(1 * time.Hour)

	source1 := models.Source{
		ID:         "src-1",
		RawContent: "Content 1",
	}
	source2 := models.Source{
		ID:         "src-2",
		RawContent: "Content 2",
	}

	dedup.Mark(source1)
	time.Sleep(10 * time.Millisecond)
	dedup.Mark(source2)

	if dedup.Size() != 2 {
		t.Errorf("expected 2 fingerprints, got %d", dedup.Size())
	}

	// Cleanup old entries
	cutoff := time.Now().Add(-5 * time.Millisecond)
	dedup.Cleanup(cutoff)

	// source1 should be removed, source2 should remain
	if dedup.Size() != 1 {
		t.Errorf("expected 1 fingerprint after cleanup, got %d", dedup.Size())
	}
}

func TestComputeContentHash(t *testing.T) {
	source1 := models.Source{
		Type:       models.SourceTypeTwitter,
		URL:        "https://example.com",
		Author:     "user1",
		RawContent: "Hello World",
	}

	source2 := models.Source{
		Type:       models.SourceTypeTwitter,
		URL:        "https://example.com",
		Author:     "user1",
		RawContent: "hello world", // Different case
	}

	hash1 := ComputeContentHash(source1)
	hash2 := ComputeContentHash(source2)

	// Should be the same (case-insensitive)
	if hash1 != hash2 {
		t.Error("hashes should match for case-insensitive content")
	}

	// Different author should produce different hash
	source3 := source1
	source3.Author = "user2"
	hash3 := ComputeContentHash(source3)

	if hash1 == hash3 {
		t.Error("hashes should differ for different authors")
	}
}

func TestNormalizeContent(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "lowercase conversion",
			input:    "Hello World",
			expected: "hello world",
		},
		{
			name:     "whitespace normalization",
			input:    "hello    world\n\t\ntest",
			expected: "hello world test",
		},
		{
			name:     "url replacement",
			input:    "check this https://example.com/article",
			expected: "check this [URL]",
		},
		{
			name:     "mention replacement",
			input:    "hey @user1 and @user2",
			expected: "hey [MENTION] and [MENTION]",
		},
		{
			name:     "hashtag replacement",
			input:    "news #breaking #urgent",
			expected: "news [TAG] [TAG]",
		},
		{
			name:     "punctuation removal",
			input:    "Hello, world! How are you?",
			expected: "hello world how are you",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeContent(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestSimilarityScore(t *testing.T) {
	tests := []struct {
		name       string
		source1    models.Source
		source2    models.Source
		minScore   float64
		expectHigh bool
	}{
		{
			name: "exact URL match",
			source1: models.Source{
				URL: "https://example.com/1",
			},
			source2: models.Source{
				URL: "https://example.com/1",
			},
			minScore:   1.0,
			expectHigh: true,
		},
		{
			name: "same author similar content",
			source1: models.Source{
				Author:     "user1",
				RawContent: "This is a breaking news story",
			},
			source2: models.Source{
				Author:     "user1",
				RawContent: "This is a breaking news story today",
			},
			minScore:   0.7,
			expectHigh: true,
		},
		{
			name: "different content",
			source1: models.Source{
				RawContent: "Completely different story",
			},
			source2: models.Source{
				RawContent: "Unrelated news article",
			},
			minScore:   0.0,
			expectHigh: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := SimilarityScore(tt.source1, tt.source2)

			if tt.expectHigh && score < tt.minScore {
				t.Errorf("expected score >= %v, got %v", tt.minScore, score)
			} else if !tt.expectHigh && score >= 0.5 {
				t.Errorf("expected low score, got %v", score)
			}
		})
	}
}

func TestJaccardSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		s1       string
		s2       string
		expected float64
	}{
		{
			name:     "identical strings",
			s1:       "hello world",
			s2:       "hello world",
			expected: 1.0,
		},
		{
			name:     "completely different",
			s1:       "abc def",
			s2:       "xyz uvw",
			expected: 0.0,
		},
		{
			name:     "partial overlap",
			s1:       "hello world",
			s2:       "hello universe",
			expected: 0.333, // 1 common word / 3 total unique words (hello, world, universe)
		},
		{
			name:     "empty strings",
			s1:       "",
			s2:       "",
			expected: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := jaccardSimilarity(tt.s1, tt.s2)
			// Allow small floating point tolerance
			tolerance := 0.01
			if result < tt.expected-tolerance || result > tt.expected+tolerance {
				t.Errorf("expected ~%v, got %v", tt.expected, result)
			}
		})
	}
}

func TestTokenize(t *testing.T) {
	input := "Hello, world! This is a test."
	tokens := tokenize(input)

	expected := []string{"Hello", "world", "This", "is", "a", "test"}
	if len(tokens) != len(expected) {
		t.Errorf("expected %d tokens, got %d", len(expected), len(tokens))
	}

	for i, token := range tokens {
		if token != expected[i] {
			t.Errorf("token %d: expected %q, got %q", i, expected[i], token)
		}
	}
}

func TestDeduplicationFilter(t *testing.T) {
	dedup := NewMemoryDeduplicator(1 * time.Hour)
	filter := NewDeduplicationFilter(dedup)

	sources := []models.Source{
		{
			ID:         "src-1",
			RawContent: "Unique content 1",
		},
		{
			ID:         "src-2",
			RawContent: "Unique content 2",
		},
		{
			ID:         "src-3",
			RawContent: "Unique content 1", // Duplicate of src-1
		},
		{
			ID:         "src-4",
			RawContent: "Unique content 3",
		},
	}

	unique := filter.Filter(sources)

	if len(unique) != 3 {
		t.Errorf("expected 3 unique sources, got %d", len(unique))
	}

	stats := filter.GetStats()
	if stats.TotalProcessed != 4 {
		t.Errorf("expected 4 processed, got %d", stats.TotalProcessed)
	}
	if stats.Duplicates != 1 {
		t.Errorf("expected 1 duplicate, got %d", stats.Duplicates)
	}
	if stats.Unique != 3 {
		t.Errorf("expected 3 unique, got %d", stats.Unique)
	}
}

func TestDeduplicationFilter_ResetStats(t *testing.T) {
	dedup := NewMemoryDeduplicator(1 * time.Hour)
	filter := NewDeduplicationFilter(dedup)

	sources := []models.Source{
		{ID: "src-1", RawContent: "Content 1"},
		{ID: "src-2", RawContent: "Content 2"},
	}

	filter.Filter(sources)

	stats := filter.GetStats()
	if stats.TotalProcessed == 0 {
		t.Error("stats should not be empty")
	}

	filter.ResetStats()

	stats = filter.GetStats()
	if stats.TotalProcessed != 0 {
		t.Error("stats should be reset to zero")
	}
}

func TestNormalizeContent_Complex(t *testing.T) {
	input := `
			BREAKING: Major event happening NOW!
			Check out https://example.com/article?id=123
			@journalist1 @journalist2 reported it first.
			#Breaking #News #URGENT
			More details: http://another-site.com
		`

	normalized := NormalizeContent(input)

	// Should not contain URLs
	if strings.Contains(normalized, "http") {
		t.Error("normalized content should not contain URLs")
	}

	// Should not contain @ mentions
	if strings.Contains(normalized, "@") {
		t.Error("normalized content should not contain mentions")
	}

	// Should not contain # hashtags
	if strings.Contains(normalized, "#") {
		t.Error("normalized content should not contain hashtags")
	}

	// Should be mostly lowercase (except for placeholders like [URL])
	lowerParts := strings.Split(normalized, "[")
	if len(lowerParts) > 0 && lowerParts[0] != strings.ToLower(lowerParts[0]) {
		t.Error("normalized content text should be lowercase")
	}

	// Should not have excessive whitespace
	if strings.Contains(normalized, "  ") {
		t.Error("normalized content should not have double spaces")
	}
}
