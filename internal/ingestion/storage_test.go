package ingestion

import (
	"context"
	"testing"
	"time"

	"github.com/STRATINT/stratint/internal/models"
	"github.com/google/uuid"
)

// TestMemorySourceRepository_GetByTitleAndURL tests the in-memory duplicate detection
func TestMemorySourceRepository_GetByTitleAndURL(t *testing.T) {
	repo := NewMemorySourceRepository()
	ctx := context.Background()

	// Create test sources
	source1 := models.Source{
		ID:          uuid.New().String(),
		Type:        models.SourceTypeTwitter,
		Title:       "Test Tweet 1",
		URL:         "https://twitter.com/user/status/111",
		Author:      "testuser",
		PublishedAt: time.Now(),
		RetrievedAt: time.Now(),
	}

	source2 := models.Source{
		ID:          uuid.New().String(),
		Type:        models.SourceTypeTwitter,
		Title:       "Test Twitter Post",
		URL:         "https://twitter.com/test/status/222",
		Author:      "twitteruser",
		PublishedAt: time.Now(),
		RetrievedAt: time.Now(),
	}

	// Store sources
	if err := repo.StoreRaw(ctx, source1); err != nil {
		t.Fatalf("failed to store source1: %v", err)
	}
	if err := repo.StoreRaw(ctx, source2); err != nil {
		t.Fatalf("failed to store source2: %v", err)
	}

	// Test 1: Find existing source by exact title and URL
	t.Run("find existing source", func(t *testing.T) {
		found, err := repo.GetByTitleAndURL(ctx, source1.Title, source1.URL)
		if err != nil {
			t.Errorf("GetByTitleAndURL returned error: %v", err)
		}
		if found == nil {
			t.Fatal("expected to find source, got nil")
		}
		if found.ID != source1.ID {
			t.Errorf("expected ID %s, got %s", source1.ID, found.ID)
		}
		if found.Title != source1.Title {
			t.Errorf("expected title %s, got %s", source1.Title, found.Title)
		}
		if found.URL != source1.URL {
			t.Errorf("expected URL %s, got %s", source1.URL, found.URL)
		}
	})

	// Test 2: Don't find source with different title
	t.Run("different title", func(t *testing.T) {
		found, err := repo.GetByTitleAndURL(ctx, "Different Title", source1.URL)
		if err != nil {
			t.Errorf("GetByTitleAndURL returned error: %v", err)
		}
		if found != nil {
			t.Error("expected nil, found source with different title")
		}
	})

	// Test 3: Don't find source with different URL
	t.Run("different URL", func(t *testing.T) {
		found, err := repo.GetByTitleAndURL(ctx, source1.Title, "https://different.com/url")
		if err != nil {
			t.Errorf("GetByTitleAndURL returned error: %v", err)
		}
		if found != nil {
			t.Error("expected nil, found source with different URL")
		}
	})

	// Test 4: Non-existent source
	t.Run("non-existent source", func(t *testing.T) {
		found, err := repo.GetByTitleAndURL(ctx, "Non Existent", "https://nonexistent.com")
		if err != nil {
			t.Errorf("GetByTitleAndURL returned error: %v", err)
		}
		if found != nil {
			t.Error("expected nil for non-existent source")
		}
	})

	// Test 5: Find second source
	t.Run("find second source", func(t *testing.T) {
		found, err := repo.GetByTitleAndURL(ctx, source2.Title, source2.URL)
		if err != nil {
			t.Errorf("GetByTitleAndURL returned error: %v", err)
		}
		if found == nil {
			t.Fatal("expected to find source2, got nil")
		}
		if found.ID != source2.ID {
			t.Errorf("expected ID %s, got %s", source2.ID, found.ID)
		}
	})
}

// TestMemorySourceRepository_DuplicatePrevention tests that duplicate detection prevents duplicates
func TestMemorySourceRepository_DuplicatePrevention(t *testing.T) {
	repo := NewMemorySourceRepository()
	ctx := context.Background()

	source := models.Source{
		ID:          uuid.New().String(),
		Type:        models.SourceTypeTwitter,
		Title:       "Duplicate Test",
		URL:         "https://twitter.com/user/status/444",
		PublishedAt: time.Now(),
		RetrievedAt: time.Now(),
	}

	// Store the source
	if err := repo.StoreRaw(ctx, source); err != nil {
		t.Fatalf("failed to store source: %v", err)
	}

	// Check for duplicates before storing again
	existing, err := repo.GetByTitleAndURL(ctx, source.Title, source.URL)
	if err != nil {
		t.Fatalf("GetByTitleAndURL failed: %v", err)
	}

	if existing == nil {
		t.Fatal("expected to find existing source for duplicate check")
	}

	if existing.ID != source.ID {
		t.Errorf("expected existing ID %s, got %s", source.ID, existing.ID)
	}

	// Verify we can prevent duplicates in application logic
	duplicate := models.Source{
		ID:          uuid.New().String(), // Different ID
		Type:        models.SourceTypeTwitter,
		Title:       "Duplicate Test",                      // Same title
		URL:         "https://twitter.com/user/status/444", // Same URL
		PublishedAt: time.Now(),
		RetrievedAt: time.Now(),
	}

	// Check before storing
	existing, err = repo.GetByTitleAndURL(ctx, duplicate.Title, duplicate.URL)
	if err != nil {
		t.Fatalf("GetByTitleAndURL failed: %v", err)
	}

	if existing != nil {
		t.Log("Duplicate detected - not storing")
		// Don't store the duplicate
	} else {
		repo.StoreRaw(ctx, duplicate)
		t.Error("should have detected duplicate")
	}

	// Verify only one source exists
	if repo.Size() != 1 {
		t.Errorf("expected 1 source, got %d", repo.Size())
	}
}
