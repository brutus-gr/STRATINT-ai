package models

import (
	"testing"
	"time"
)

func TestSource_GetDisplayName(t *testing.T) {
	tests := []struct {
		name     string
		source   Source
		expected string
	}{
		{
			name: "Title present",
			source: Source{
				Title:  "Breaking News",
				Author: "JohnDoe",
				Type:   SourceTypeTwitter,
			},
			expected: "Breaking News",
		},
		{
			name: "Author present, no title",
			source: Source{
				Author: "JaneDoe",
				Type:   SourceTypeTelegram,
			},
			expected: "JaneDoe (telegram)",
		},
		{
			name: "Only type present",
			source: Source{
				Type: SourceTypeTwitter,
			},
			expected: "twitter source",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.source.GetDisplayName(); got != tt.expected {
				t.Errorf("GetDisplayName() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestSource_IsRecent(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name        string
		publishedAt time.Time
		window      time.Duration
		expected    bool
	}{
		{
			name:        "Recent within 1 hour",
			publishedAt: now.Add(-30 * time.Minute),
			window:      1 * time.Hour,
			expected:    true,
		},
		{
			name:        "Not recent beyond 1 hour",
			publishedAt: now.Add(-2 * time.Hour),
			window:      1 * time.Hour,
			expected:    false,
		},
		{
			name:        "Recent within 24 hours",
			publishedAt: now.Add(-12 * time.Hour),
			window:      24 * time.Hour,
			expected:    true,
		},
		{
			name:        "Just published",
			publishedAt: now,
			window:      1 * time.Hour,
			expected:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Source{PublishedAt: tt.publishedAt}
			if got := s.IsRecent(tt.window); got != tt.expected {
				t.Errorf("IsRecent() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestSource_IsCredible(t *testing.T) {
	tests := []struct {
		name        string
		credibility float64
		expected    bool
	}{
		{"High credibility", 0.9, true},
		{"Threshold credibility", 0.4, true},
		{"Low credibility", 0.3, false},
		{"Zero credibility", 0.0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Source{Credibility: tt.credibility}
			if got := s.IsCredible(); got != tt.expected {
				t.Errorf("IsCredible() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestSourceType(t *testing.T) {
	types := []SourceType{
		SourceTypeTwitter,
		SourceTypeTelegram,
		SourceTypeGLP,
		SourceTypeGovernment,
		SourceTypeNewsMedia,
		SourceTypeBlog,
		SourceTypeOther,
	}

	for _, st := range types {
		if st == "" {
			t.Errorf("SourceType should not be empty")
		}
	}
}

func TestSourceMetadata_Twitter(t *testing.T) {
	meta := SourceMetadata{
		TweetID:      "1234567890",
		RetweetCount: 150,
		LikeCount:    500,
		Hashtags:     []string{"OSINT", "Breaking"},
		Mentions:     []string{"@user1", "@user2"},
		Language:     "en",
	}

	if meta.TweetID == "" {
		t.Error("TweetID should be set")
	}
	if len(meta.Hashtags) != 2 {
		t.Errorf("Expected 2 hashtags, got %d", len(meta.Hashtags))
	}
}

func TestSourceMetadata_Telegram(t *testing.T) {
	meta := SourceMetadata{
		ChannelID:   "chan-123",
		ChannelName: "OSINT Channel",
		MessageID:   "msg-456",
		ViewCount:   1000,
	}

	if meta.ChannelID == "" {
		t.Error("ChannelID should be set")
	}
	if meta.ViewCount <= 0 {
		t.Error("ViewCount should be positive")
	}
}

func TestSource_FullLifecycle(t *testing.T) {
	now := time.Now()
	source := Source{
		ID:          "src-123",
		Type:        SourceTypeTwitter,
		URL:         "https://twitter.com/user/status/123",
		Title:       "Breaking: Major Event",
		Author:      "OSINTAnalyst",
		AuthorID:    "user-456",
		PublishedAt: now.Add(-1 * time.Hour),
		RetrievedAt: now,
		RawContent:  "Breaking news content here",
		Credibility: 0.75,
		Metadata: SourceMetadata{
			TweetID:      "123",
			RetweetCount: 50,
			LikeCount:    200,
			Hashtags:     []string{"Breaking"},
			Language:     "en",
		},
	}

	// Test display name
	displayName := source.GetDisplayName()
	if displayName != "Breaking: Major Event" {
		t.Errorf("Expected title as display name, got %s", displayName)
	}

	// Test recency
	if !source.IsRecent(2 * time.Hour) {
		t.Error("Source should be recent within 2 hours")
	}

	// Test credibility
	if !source.IsCredible() {
		t.Error("Source should be credible")
	}
}
