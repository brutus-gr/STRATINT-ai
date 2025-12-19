package database

import (
	"context"
	"testing"
	"time"

	"github.com/STRATINT/stratint/internal/models"
	"github.com/google/uuid"
)

func TestGetByTitleAndURL(t *testing.T) {
	// Skip if no database connection available
	// In real scenario, you'd use testcontainers or similar
	t.Skip("Requires database connection - run manually or with integration test setup")

	ctx := context.Background()

	// Setup test database connection
	dbURL := "postgresql://osintmcp:osintmcp_dev_password@localhost:5432/osintmcp_test?sslmode=disable"
	cfg := Config{URL: dbURL}
	db, err := Connect(ctx, cfg)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	repo := NewPostgresSourceRepository(db)

	// Create a test source
	testSource := models.Source{
		ID:          uuid.New().String(),
		Type:        models.SourceTypeTwitter,
		Title:       "Test Tweet Title",
		URL:         "https://twitter.com/user/status/123456789",
		Author:      "testuser",
		PublishedAt: time.Now(),
		RetrievedAt: time.Now(),
		RawContent:  "Test tweet content",
		Credibility: 0.8,
		Metadata:    models.SourceMetadata{},
	}

	// Store the test source
	err = repo.Store(ctx, testSource)
	if err != nil {
		t.Fatalf("failed to store test source: %v", err)
	}

	// Test 1: Find existing source by exact title and URL
	t.Run("find existing source", func(t *testing.T) {
		found, err := repo.GetByTitleAndURL(ctx, testSource.Title, testSource.URL)
		if err != nil {
			t.Errorf("GetByTitleAndURL returned error: %v", err)
		}
		if found == nil {
			t.Error("expected to find source, got nil")
		}
		if found != nil && found.ID != testSource.ID {
			t.Errorf("expected ID %s, got %s", testSource.ID, found.ID)
		}
	})

	// Test 2: Don't find source with different title
	t.Run("different title", func(t *testing.T) {
		found, err := repo.GetByTitleAndURL(ctx, "Different Title", testSource.URL)
		if err != nil {
			t.Errorf("GetByTitleAndURL returned error: %v", err)
		}
		if found != nil {
			t.Error("expected nil, found source with different title")
		}
	})

	// Test 3: Don't find source with different URL
	t.Run("different URL", func(t *testing.T) {
		found, err := repo.GetByTitleAndURL(ctx, testSource.Title, "https://different.com/url")
		if err != nil {
			t.Errorf("GetByTitleAndURL returned error: %v", err)
		}
		if found != nil {
			t.Error("expected nil, found source with different URL")
		}
	})

	// Test 4: Handle non-existent source
	t.Run("non-existent source", func(t *testing.T) {
		found, err := repo.GetByTitleAndURL(ctx, "Non Existent Title", "https://nonexistent.com/url")
		if err != nil {
			t.Errorf("GetByTitleAndURL returned error: %v", err)
		}
		if found != nil {
			t.Error("expected nil for non-existent source")
		}
	})

	// Test 5: Case sensitivity check
	t.Run("case sensitivity", func(t *testing.T) {
		found, err := repo.GetByTitleAndURL(ctx, "test tweet title", testSource.URL)
		if err != nil {
			t.Errorf("GetByTitleAndURL returned error: %v", err)
		}
		// PostgreSQL is case-sensitive by default for = operator
		if found != nil {
			t.Error("expected nil due to case mismatch")
		}
	})

	// Cleanup
	t.Cleanup(func() {
		// Delete test source
		_, err := repo.db.ExecContext(ctx, "DELETE FROM sources WHERE id = $1", testSource.ID)
		if err != nil {
			t.Logf("cleanup failed: %v", err)
		}
	})
}

func TestGetByTitleAndURL_EmptyValues(t *testing.T) {
	t.Skip("Requires database connection - run manually or with integration test setup")

	ctx := context.Background()

	dbURL := "postgresql://osintmcp:osintmcp_dev_password@localhost:5432/osintmcp_test?sslmode=disable"
	cfg := Config{URL: dbURL}
	db, err := Connect(ctx, cfg)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	repo := NewPostgresSourceRepository(db)

	// Test with empty title
	t.Run("empty title", func(t *testing.T) {
		found, err := repo.GetByTitleAndURL(ctx, "", "https://example.com/url")
		if err != nil {
			t.Errorf("GetByTitleAndURL returned error: %v", err)
		}
		if found != nil {
			t.Error("expected nil for empty title")
		}
	})

	// Test with empty URL
	t.Run("empty URL", func(t *testing.T) {
		found, err := repo.GetByTitleAndURL(ctx, "Some Title", "")
		if err != nil {
			t.Errorf("GetByTitleAndURL returned error: %v", err)
		}
		if found != nil {
			t.Error("expected nil for empty URL")
		}
	})

	// Test with both empty
	t.Run("both empty", func(t *testing.T) {
		found, err := repo.GetByTitleAndURL(ctx, "", "")
		if err != nil {
			t.Errorf("GetByTitleAndURL returned error: %v", err)
		}
		if found != nil {
			t.Error("expected nil for both empty")
		}
	})
}

func TestGetByTitleAndURL_MultipleSources(t *testing.T) {
	t.Skip("Requires database connection - run manually or with integration test setup")

	ctx := context.Background()

	dbURL := "postgresql://osintmcp:osintmcp_dev_password@localhost:5432/osintmcp_test?sslmode=disable"
	cfg := Config{URL: dbURL}
	db, err := Connect(ctx, cfg)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	repo := NewPostgresSourceRepository(db)

	// Create multiple test sources with different titles/URLs
	sources := []models.Source{
		{
			ID:          uuid.New().String(),
			Type:        models.SourceTypeTwitter,
			Title:       "First Tweet",
			URL:         "https://twitter.com/user/status/111",
			PublishedAt: time.Now().Add(-2 * time.Hour),
			RetrievedAt: time.Now(),
		},
		{
			ID:          uuid.New().String(),
			Type:        models.SourceTypeTwitter,
			Title:       "Second Tweet",
			URL:         "https://twitter.com/user/status/222",
			PublishedAt: time.Now().Add(-1 * time.Hour),
			RetrievedAt: time.Now(),
		},
		{
			ID:          uuid.New().String(),
			Type:        models.SourceTypeTwitter,
			Title:       "Twitter Post",
			URL:         "https://twitter.com/test/status/333",
			PublishedAt: time.Now(),
			RetrievedAt: time.Now(),
		},
	}

	// Store all sources
	for _, source := range sources {
		err := repo.Store(ctx, source)
		if err != nil {
			t.Fatalf("failed to store source %s: %v", source.ID, err)
		}
	}

	// Test finding each source individually
	for _, expected := range sources {
		t.Run("find_"+expected.Title, func(t *testing.T) {
			found, err := repo.GetByTitleAndURL(ctx, expected.Title, expected.URL)
			if err != nil {
				t.Errorf("GetByTitleAndURL returned error: %v", err)
			}
			if found == nil {
				t.Error("expected to find source, got nil")
			}
			if found != nil && found.ID != expected.ID {
				t.Errorf("expected ID %s, got %s", expected.ID, found.ID)
			}
		})
	}

	// Cleanup
	t.Cleanup(func() {
		for _, source := range sources {
			_, err := repo.db.ExecContext(ctx, "DELETE FROM sources WHERE id = $1", source.ID)
			if err != nil {
				t.Logf("cleanup failed for %s: %v", source.ID, err)
			}
		}
	})
}
