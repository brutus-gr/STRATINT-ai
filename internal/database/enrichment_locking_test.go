package database

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/STRATINT/stratint/internal/models"
	_ "github.com/lib/pq"
)

// TestClaimSourcesForEnrichment_AtomicBehavior tests that claiming is truly atomic
func TestClaimSourcesForEnrichment_AtomicBehavior(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := setupTestDB(t)
	defer db.Close()

	repo := NewPostgresSourceRepository(db)
	ctx := context.Background()

	// Create test sources
	sources := createTestSources(t, db, 10)

	// Try to claim same sources from 5 goroutines simultaneously
	var wg sync.WaitGroup
	claimedByGoroutine := make([][]models.Source, 5)
	errors := make([]error, 5)

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			claimed, err := repo.ClaimSourcesForEnrichment(ctx, 10, 15*time.Minute)
			claimedByGoroutine[idx] = claimed
			errors[idx] = err
		}(i)
	}

	wg.Wait()

	// Check errors
	for i, err := range errors {
		if err != nil {
			t.Errorf("goroutine %d failed: %v", i, err)
		}
	}

	// Count total claimed sources
	totalClaimed := 0
	claimedIDs := make(map[string]int)
	for i, claimed := range claimedByGoroutine {
		t.Logf("Goroutine %d claimed %d sources", i, len(claimed))
		totalClaimed += len(claimed)
		for _, source := range claimed {
			claimedIDs[source.ID]++
			if claimedIDs[source.ID] > 1 {
				t.Errorf("Source %s was claimed by multiple goroutines!", source.ID)
			}
		}
	}

	if totalClaimed != len(sources) {
		t.Errorf("Expected %d total claims, got %d", len(sources), totalClaimed)
	}

	// Verify all sources are marked as "enriching"
	for _, source := range sources {
		var status string
		err := db.QueryRow("SELECT enrichment_status FROM sources WHERE id = $1", source.ID).Scan(&status)
		if err != nil {
			t.Errorf("Failed to check status for source %s: %v", source.ID, err)
			continue
		}
		if status != "enriching" {
			t.Errorf("Source %s has status %s, expected 'enriching'", source.ID, status)
		}
	}
}

// TestClaimSourcesForEnrichment_StaleReclaim tests that stale claims are reclaimed
func TestClaimSourcesForEnrichment_StaleReclaim(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := setupTestDB(t)
	defer db.Close()

	repo := NewPostgresSourceRepository(db)
	ctx := context.Background()

	// Create a source with stale claim (20 minutes ago)
	source := models.Source{
		ID:                  "stale-source-1",
		Type:                models.SourceTypeNewsMedia,
		URL:                 "https://test.com/stale",
		PublishedAt:         time.Now(),
		RetrievedAt:         time.Now(),
		RawContent:          "Test content for stale source",
		ContentHash:         "stale123",
		Credibility:         0.8,
		ScrapeStatus:        models.ScrapeStatusCompleted,
		EnrichmentStatus:    models.EnrichmentStatusEnriching,           // Stuck in enriching
		EnrichmentClaimedAt: ptrTime(time.Now().Add(-20 * time.Minute)), // Stale claim
		CreatedAt:           time.Now(),
	}

	if err := repo.Store(ctx, source); err != nil {
		t.Fatalf("Failed to create stale source: %v", err)
	}

	// Try to claim with 15 minute stale threshold
	claimed, err := repo.ClaimSourcesForEnrichment(ctx, 10, 15*time.Minute)
	if err != nil {
		t.Fatalf("Failed to claim: %v", err)
	}

	// Should reclaim the stale source
	if len(claimed) != 1 {
		t.Fatalf("Expected to reclaim 1 stale source, got %d", len(claimed))
	}

	if claimed[0].ID != source.ID {
		t.Errorf("Claimed wrong source: %s", claimed[0].ID)
	}
}

// TestEnrichmentFlow_NoDuplicates tests full enrichment flow for duplicates
func TestEnrichmentFlow_NoDuplicates(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := setupTestDB(t)
	defer db.Close()

	sourceRepo := NewPostgresSourceRepository(db)
	eventRepo := NewPostgresEventRepository(db)
	ctx := context.Background()

	// Create test sources
	sources := createTestSources(t, db, 5)

	// Simulate 3 worker instances claiming and processing
	var wg sync.WaitGroup
	allEvents := make([][]string, 3) // Store event IDs created by each worker
	mu := sync.Mutex{}

	for worker := 0; worker < 3; worker++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			// Each worker tries to claim sources
			claimed, err := sourceRepo.ClaimSourcesForEnrichment(ctx, 10, 15*time.Minute)
			if err != nil {
				t.Errorf("Worker %d failed to claim: %v", workerID, err)
				return
			}

			t.Logf("Worker %d claimed %d sources", workerID, len(claimed))

			// Process each claimed source
			for _, source := range claimed {
				// Create event
				event := models.Event{
					ID:        fmt.Sprintf("event-%s-%d", source.ID, workerID),
					Title:     fmt.Sprintf("Event from %s by worker %d", source.ID, workerID),
					Summary:   "Test event",
					Category:  models.CategoryGeopolitics,
					Status:    models.EventStatusPublished,
					Timestamp: time.Now(),
					Sources:   []models.Source{source},
					Confidence: models.Confidence{
						Score: 0.8,
						Level: models.ConfidenceHigh,
					},
					CreatedAt: time.Now(),
				}

				if err := eventRepo.Create(ctx, event); err != nil {
					t.Errorf("Worker %d failed to create event: %v", workerID, err)
					continue
				}

				mu.Lock()
				allEvents[workerID] = append(allEvents[workerID], event.ID)
				mu.Unlock()

				// Mark source as enriched
				if err := sourceRepo.UpdateEnrichmentStatus(ctx, source.ID, models.EnrichmentStatusCompleted, ""); err != nil {
					t.Errorf("Worker %d failed to mark source as enriched: %v", workerID, err)
				}
			}
		}(worker)
	}

	wg.Wait()

	// Count total events created
	totalEvents := 0
	for i, events := range allEvents {
		t.Logf("Worker %d created %d events", i, len(events))
		totalEvents += len(events)
	}

	// Should have exactly 5 events (one per source)
	if totalEvents != len(sources) {
		t.Errorf("Expected %d events, got %d", len(sources), totalEvents)
	}

	// Verify each source has exactly one event
	for _, source := range sources {
		var count int
		err := db.QueryRow(`
			SELECT COUNT(*) FROM event_sources WHERE source_id = $1
		`, source.ID).Scan(&count)
		if err != nil {
			t.Errorf("Failed to count events for source %s: %v", source.ID, err)
			continue
		}
		if count != 1 {
			t.Errorf("Source %s has %d events, expected 1", source.ID, count)
		}
	}
}

// TestUpdateEnrichmentStatus tests status updates
func TestUpdateEnrichmentStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := setupTestDB(t)
	defer db.Close()

	repo := NewPostgresSourceRepository(db)
	ctx := context.Background()

	// Create source
	source := models.Source{
		ID:               "test-status-1",
		Type:             models.SourceTypeNewsMedia,
		URL:              "https://test.com/status",
		PublishedAt:      time.Now(),
		RetrievedAt:      time.Now(),
		RawContent:       "Test content",
		ContentHash:      "status123",
		Credibility:      0.8,
		ScrapeStatus:     models.ScrapeStatusCompleted,
		EnrichmentStatus: models.EnrichmentStatusPending,
		CreatedAt:        time.Now(),
	}

	if err := repo.Store(ctx, source); err != nil {
		t.Fatalf("Failed to create source: %v", err)
	}

	// Update to completed
	if err := repo.UpdateEnrichmentStatus(ctx, source.ID, models.EnrichmentStatusCompleted, ""); err != nil {
		t.Fatalf("Failed to update status: %v", err)
	}

	// Verify status
	var status string
	var enrichedAt sql.NullTime
	err := db.QueryRow(`
		SELECT enrichment_status, enriched_at FROM sources WHERE id = $1
	`, source.ID).Scan(&status, &enrichedAt)
	if err != nil {
		t.Fatalf("Failed to check status: %v", err)
	}

	if status != "completed" {
		t.Errorf("Status is %s, expected 'completed'", status)
	}

	if !enrichedAt.Valid {
		t.Error("enriched_at should be set for completed status")
	}

	// Update to failed
	errorMsg := "Test error"
	if err := repo.UpdateEnrichmentStatus(ctx, source.ID, models.EnrichmentStatusFailed, errorMsg); err != nil {
		t.Fatalf("Failed to update to failed: %v", err)
	}

	// Verify error stored
	var storedError sql.NullString
	err = db.QueryRow(`
		SELECT enrichment_error FROM sources WHERE id = $1
	`, source.ID).Scan(&storedError)
	if err != nil {
		t.Fatalf("Failed to check error: %v", err)
	}

	if !storedError.Valid || storedError.String != errorMsg {
		t.Errorf("Error is %v, expected '%s'", storedError, errorMsg)
	}
}

// Helper functions

func setupTestDB(t *testing.T) *sql.DB {
	// Try to connect to test database
	dbURL := "postgres://postgres:postgres@localhost:5432/osintmcp_test?sslmode=disable"
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Skipf("Skipping test: cannot connect to test database: %v", err)
	}

	if err := db.Ping(); err != nil {
		t.Skipf("Skipping test: test database not available: %v", err)
	}

	// Clean up test data
	db.Exec("DELETE FROM event_sources")
	db.Exec("DELETE FROM events")
	db.Exec("DELETE FROM sources")

	// Verify enrichment columns exist
	var exists bool
	err = db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_name = 'sources' AND column_name = 'enrichment_status'
		)
	`).Scan(&exists)
	if err != nil || !exists {
		t.Skipf("Skipping test: enrichment_status column doesn't exist. Run migration 012 first.")
	}

	return db
}

func createTestSources(t *testing.T, db *sql.DB, count int) []models.Source {
	sources := make([]models.Source, count)
	for i := 0; i < count; i++ {
		source := models.Source{
			ID:               fmt.Sprintf("test-source-%d", i),
			Type:             models.SourceTypeNewsMedia,
			URL:              fmt.Sprintf("https://test.com/article-%d", i),
			PublishedAt:      time.Now(),
			RetrievedAt:      time.Now(),
			RawContent:       fmt.Sprintf("Test content %d", i),
			ContentHash:      fmt.Sprintf("hash%d", i),
			Credibility:      0.8,
			ScrapeStatus:     models.ScrapeStatusCompleted,
			EnrichmentStatus: models.EnrichmentStatusPending,
			CreatedAt:        time.Now(),
		}

		_, err := db.Exec(`
			INSERT INTO sources (
				id, type, url, published_at, retrieved_at, raw_content, content_hash,
				credibility, scrape_status, enrichment_status, created_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		`, source.ID, source.Type, source.URL, source.PublishedAt, source.RetrievedAt,
			source.RawContent, source.ContentHash, source.Credibility, source.ScrapeStatus,
			source.EnrichmentStatus, source.CreatedAt)

		if err != nil {
			t.Fatalf("Failed to create test source: %v", err)
		}

		sources[i] = source
	}

	return sources
}

func ptrTime(t time.Time) *time.Time {
	return &t
}
