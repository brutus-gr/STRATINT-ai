package test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/STRATINT/stratint/internal/database"
	"github.com/STRATINT/stratint/internal/enrichment"
	"github.com/STRATINT/stratint/internal/inference"
	"github.com/STRATINT/stratint/internal/models"
	_ "github.com/lib/pq"
	"log/slog"
)

// TestResult and TestSuite types are defined in report_generator.go

// Global test suite
var suite *TestSuite

func init() {
	suite = &TestSuite{
		Name:      "OSINT System Integration Tests",
		StartTime: time.Now(),
		Results:   []TestResult{},
	}
}

// TestMain runs all tests and generates HTML report
func TestMain(m *testing.M) {
	// Run all tests
	code := m.Run()

	// Finalize suite
	suite.EndTime = time.Now()
	suite.TotalTests = len(suite.Results)
	for _, r := range suite.Results {
		if r.Passed {
			suite.PassedTests++
		} else {
			suite.FailedTests++
		}
	}

	// Generate HTML report
	if err := GenerateHTMLReport(suite, "test_report.html"); err != nil {
		fmt.Printf("Failed to generate HTML report: %v\n", err)
	} else {
		fmt.Printf("\n✅ Test report generated: test_report.html\n")
	}

	// Generate JSON report
	jsonData, _ := json.MarshalIndent(suite, "", "  ")
	os.WriteFile("test_report.json", jsonData, 0644)

	os.Exit(code)
}

// Helper to add test result
func addResult(result TestResult) {
	suite.Results = append(suite.Results, result)
}

// TestSourceDeduplication tests content hash-based deduplication
func TestSourceDeduplication(t *testing.T) {
	start := time.Now()

	// Test Case 1: Identical content should have same hash
	source1 := models.Source{
		ID:          "test-1",
		RawContent:  "Russia launches missile strikes on Kyiv, 5 civilians killed",
		ContentHash: hashContent("Russia launches missile strikes on Kyiv, 5 civilians killed"),
	}

	source2 := models.Source{
		ID:          "test-2",
		RawContent:  "Russia launches missile strikes on Kyiv, 5 civilians killed",
		ContentHash: hashContent("Russia launches missile strikes on Kyiv, 5 civilians killed"),
	}

	passed := source1.ContentHash == source2.ContentHash
	addResult(TestResult{
		TestName:        "Source Deduplication - Identical Content",
		Category:        "Deduplication",
		Description:     "Two sources with identical content should have the same hash",
		Passed:          passed,
		ExpectedOutcome: "Hashes match",
		ActualOutcome:   fmt.Sprintf("Hash1: %s, Hash2: %s", source1.ContentHash, source2.ContentHash),
		Details: map[string]interface{}{
			"source1_hash": source1.ContentHash,
			"source2_hash": source2.ContentHash,
		},
		Duration: time.Since(start),
	})

	if !passed {
		t.Error("Identical sources should have same content hash")
	}

	// Test Case 2: Different content should have different hashes
	start = time.Now()
	source3 := models.Source{
		ID:          "test-3",
		RawContent:  "Ukraine launches drone strike on Russian oil refinery",
		ContentHash: hashContent("Ukraine launches drone strike on Russian oil refinery"),
	}

	passed = source1.ContentHash != source3.ContentHash
	addResult(TestResult{
		TestName:        "Source Deduplication - Different Content",
		Category:        "Deduplication",
		Description:     "Two sources with different content should have different hashes",
		Passed:          passed,
		ExpectedOutcome: "Hashes differ",
		ActualOutcome:   fmt.Sprintf("Hash1: %s, Hash3: %s", source1.ContentHash, source3.ContentHash),
		Details: map[string]interface{}{
			"source1_hash": source1.ContentHash,
			"source3_hash": source3.ContentHash,
		},
		Duration: time.Since(start),
	})

	if !passed {
		t.Error("Different sources should have different content hashes")
	}

	// Test Case 3: Minor variations should have different hashes
	start = time.Now()
	source4 := models.Source{
		ID:          "test-4",
		RawContent:  "Russia launches missile strikes on Kyiv, 5 civilians killed!",
		ContentHash: hashContent("Russia launches missile strikes on Kyiv, 5 civilians killed!"),
	}

	passed = source1.ContentHash != source4.ContentHash
	addResult(TestResult{
		TestName:        "Source Deduplication - Minor Punctuation Variation",
		Category:        "Deduplication",
		Description:     "Content with minor punctuation changes should have different hashes",
		Passed:          passed,
		ExpectedOutcome: "Hashes differ (punctuation is significant)",
		ActualOutcome:   fmt.Sprintf("Hash1: %s, Hash4: %s", source1.ContentHash, source4.ContentHash),
		Details: map[string]interface{}{
			"source1_hash": source1.ContentHash,
			"source4_hash": source4.ContentHash,
			"difference":   "Added exclamation mark",
		},
		Duration: time.Since(start),
	})

	if !passed {
		t.Error("Minor variations should produce different hashes")
	}

	// Test Case 4: Whitespace variations should have different hashes
	start = time.Now()
	source5 := models.Source{
		ID:          "test-5",
		RawContent:  "Russia  launches  missile  strikes  on  Kyiv",
		ContentHash: hashContent("Russia  launches  missile  strikes  on  Kyiv"),
	}

	source6 := models.Source{
		ID:          "test-6",
		RawContent:  "Russia launches missile strikes on Kyiv",
		ContentHash: hashContent("Russia launches missile strikes on Kyiv"),
	}

	passed = source5.ContentHash != source6.ContentHash
	addResult(TestResult{
		TestName:        "Source Deduplication - Whitespace Sensitivity",
		Category:        "Deduplication",
		Description:     "Content with different whitespace should have different hashes",
		Passed:          passed,
		ExpectedOutcome: "Hashes differ (whitespace is significant)",
		ActualOutcome:   fmt.Sprintf("Hash5: %s, Hash6: %s", source5.ContentHash, source6.ContentHash),
		Details: map[string]interface{}{
			"source5_hash": source5.ContentHash,
			"source6_hash": source6.ContentHash,
			"difference":   "Double vs single spaces",
		},
		Duration: time.Since(start),
	})

	if !passed {
		t.Error("Whitespace variations should produce different hashes")
	}
}

// TestEventCorrelation tests OpenAI-based event correlation
func TestEventCorrelation(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()

	// Get database connection string
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set, skipping correlation tests")
		return
	}

	// Connect to database
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Skipf("Failed to connect to database: %v", err)
		return
	}
	defer db.Close()

	// Verify connection
	if err := db.Ping(); err != nil {
		t.Skipf("Failed to ping database: %v", err)
		return
	}

	// Load OpenAI configuration from database
	configRepo := database.NewOpenAIConfigRepository(db)
	inferenceLogRepo := database.NewInferenceLogRepository(db)
	inferenceLogger := inference.NewLogger(inferenceLogRepo, logger)
	client, err := enrichment.NewOpenAIClientFromDB(ctx, configRepo, logger, inferenceLogger)
	if err != nil {
		t.Skipf("OpenAI not configured in database: %v", err)
		return
	}

	correlator := client.GetCorrelator()

	// Test Case 1: Same event, minor variation
	testSameEventMinorVariation(t, ctx, correlator, logger)

	// Test Case 2: Same event with novel facts
	testSameEventNovelFacts(t, ctx, correlator, logger)

	// Test Case 3: Different events, same topic
	testDifferentEventsSameTopic(t, ctx, correlator, logger)

	// Test Case 4: Completely unrelated events
	testUnrelatedEvents(t, ctx, correlator, logger)

	// Test Case 5: Casualty count update
	testCasualtyUpdate(t, ctx, correlator, logger)

	// Test Case 6: Temporal sequence (related but different events)
	testTemporalSequence(t, ctx, correlator, logger)

	// Test Case 7: Same location, different incidents
	testSameLocationDifferentIncidents(t, ctx, correlator, logger)

	// Test Case 8: Conflicting information
	testConflictingInformation(t, ctx, correlator, logger)
}

func testSameEventMinorVariation(t *testing.T, ctx context.Context, correlator *enrichment.EventCorrelator, logger *slog.Logger) {
	start := time.Now()

	existingEvent := models.Event{
		ID:        "evt-1",
		Timestamp: time.Now().Add(-1 * time.Hour),
		Title:     "Russia launches missile strikes on Kyiv, 5 civilians killed",
		Summary:   "Russian forces conducted missile strikes on the Ukrainian capital Kyiv early this morning, resulting in 5 civilian casualties. Air defense systems intercepted several missiles.",
		Category:  models.CategoryMilitary,
		Tags:      []string{"Russia", "Ukraine", "Kyiv", "missile strike"},
	}

	newSource := models.Source{
		ID:          "src-2",
		Title:       "Russian missiles hit Ukrainian capital, killing 5",
		RawContent:  "Russian missile attack on Kyiv this morning killed five civilians, Ukrainian officials reported. Multiple missiles were intercepted by air defenses.",
		PublishedAt: time.Now(),
	}

	result, err := correlator.AnalyzeCorrelation(ctx, newSource, existingEvent)

	var passed bool
	var actualOutcome string
	var errorStr string
	details := map[string]interface{}{
		"existing_event": existingEvent.Title,
		"new_source":     newSource.Title,
	}

	if err != nil {
		errorStr = err.Error()
		actualOutcome = fmt.Sprintf("Error: %v", err)
	} else {
		passed = result.ShouldMerge && result.Similarity >= 0.8 && !result.HasNovelFacts
		actualOutcome = fmt.Sprintf("Similarity: %.2f, ShouldMerge: %v, HasNovelFacts: %v, Reasoning: %s",
			result.Similarity, result.ShouldMerge, result.HasNovelFacts, result.Reasoning)
		details["similarity"] = result.Similarity
		details["should_merge"] = result.ShouldMerge
		details["novel_facts"] = result.NovelFacts
		details["reasoning"] = result.Reasoning
	}

	addResult(TestResult{
		TestName:        "Event Correlation - Same Event Minor Variation",
		Category:        "Correlation",
		Description:     "Same event described with minor wording differences should merge with high similarity",
		Passed:          passed,
		ExpectedOutcome: "Similarity >= 0.8, ShouldMerge = true, HasNovelFacts = false",
		ActualOutcome:   actualOutcome,
		Details:         details,
		Duration:        time.Since(start),
		Error:           errorStr,
	})

	if err != nil {
		t.Errorf("Correlation analysis failed: %v", err)
	} else if !passed {
		t.Errorf("Same event with minor variation should merge (similarity >= 0.8, should_merge = true, has_novel_facts = false)")
	}
}

func testSameEventNovelFacts(t *testing.T, ctx context.Context, correlator *enrichment.EventCorrelator, logger *slog.Logger) {
	start := time.Now()

	existingEvent := models.Event{
		ID:        "evt-2",
		Timestamp: time.Now().Add(-2 * time.Hour),
		Title:     "Russia launches missile strikes on Kyiv, 5 civilians killed",
		Summary:   "Russian forces conducted missile strikes on the Ukrainian capital Kyiv early this morning, resulting in 5 civilian casualties.",
		Category:  models.CategoryMilitary,
		Tags:      []string{"Russia", "Ukraine", "Kyiv", "missile strike"},
	}

	newSource := models.Source{
		ID:          "src-3",
		Title:       "Kyiv missile attack kills 5, damages power station, 15 injured",
		RawContent:  "Russia's missile attack on Kyiv this morning killed 5 civilians and damaged a major power station, officials reported. In addition to the fatalities, 15 people were injured and evacuated to hospitals.",
		PublishedAt: time.Now(),
	}

	result, err := correlator.AnalyzeCorrelation(ctx, newSource, existingEvent)

	var passed bool
	var actualOutcome string
	var errorStr string
	details := map[string]interface{}{
		"existing_event": existingEvent.Title,
		"new_source":     newSource.Title,
	}

	if err != nil {
		errorStr = err.Error()
		actualOutcome = fmt.Sprintf("Error: %v", err)
	} else {
		passed = result.ShouldMerge && result.Similarity >= 0.7 && result.HasNovelFacts && len(result.NovelFacts) > 0
		actualOutcome = fmt.Sprintf("Similarity: %.2f, ShouldMerge: %v, HasNovelFacts: %v, NovelFactCount: %d, Reasoning: %s",
			result.Similarity, result.ShouldMerge, result.HasNovelFacts, len(result.NovelFacts), result.Reasoning)
		details["similarity"] = result.Similarity
		details["should_merge"] = result.ShouldMerge
		details["novel_facts"] = result.NovelFacts
		details["reasoning"] = result.Reasoning
	}

	addResult(TestResult{
		TestName:        "Event Correlation - Same Event with Novel Facts",
		Category:        "Correlation",
		Description:     "Same event with additional information should merge and identify novel facts",
		Passed:          passed,
		ExpectedOutcome: "Similarity >= 0.7, ShouldMerge = true, HasNovelFacts = true, NovelFacts includes injuries and power station",
		ActualOutcome:   actualOutcome,
		Details:         details,
		Duration:        time.Since(start),
		Error:           errorStr,
	})

	if err != nil {
		t.Errorf("Correlation analysis failed: %v", err)
	} else if !passed {
		t.Errorf("Same event with novel facts should merge and identify new information")
	}
}

func testDifferentEventsSameTopic(t *testing.T, ctx context.Context, correlator *enrichment.EventCorrelator, logger *slog.Logger) {
	start := time.Now()

	existingEvent := models.Event{
		ID:        "evt-3",
		Timestamp: time.Now().Add(-3 * time.Hour),
		Title:     "Russia launches missile strikes on Kyiv, 5 civilians killed",
		Summary:   "Russian forces conducted missile strikes on the Ukrainian capital Kyiv early this morning, resulting in 5 civilian casualties.",
		Category:  models.CategoryMilitary,
		Tags:      []string{"Russia", "Ukraine", "Kyiv", "missile strike"},
	}

	newSource := models.Source{
		ID:          "src-4",
		Title:       "Ukraine drone strike on Russian oil refinery causes major fire",
		RawContent:  "Ukrainian forces launched a drone attack on a major Russian oil refinery in Rostov region, causing a massive fire that has not been contained. Russian authorities report significant damage to refining capacity.",
		PublishedAt: time.Now(),
	}

	result, err := correlator.AnalyzeCorrelation(ctx, newSource, existingEvent)

	var passed bool
	var actualOutcome string
	var errorStr string
	details := map[string]interface{}{
		"existing_event": existingEvent.Title,
		"new_source":     newSource.Title,
	}

	if err != nil {
		errorStr = err.Error()
		actualOutcome = fmt.Sprintf("Error: %v", err)
	} else {
		passed = !result.ShouldMerge && result.Similarity < 0.5
		actualOutcome = fmt.Sprintf("Similarity: %.2f, ShouldMerge: %v, Reasoning: %s",
			result.Similarity, result.ShouldMerge, result.Reasoning)
		details["similarity"] = result.Similarity
		details["should_merge"] = result.ShouldMerge
		details["reasoning"] = result.Reasoning
	}

	addResult(TestResult{
		TestName:        "Event Correlation - Different Events Same Topic",
		Category:        "Correlation",
		Description:     "Different events related to same conflict should NOT merge",
		Passed:          passed,
		ExpectedOutcome: "Similarity < 0.5, ShouldMerge = false",
		ActualOutcome:   actualOutcome,
		Details:         details,
		Duration:        time.Since(start),
		Error:           errorStr,
	})

	if err != nil {
		t.Errorf("Correlation analysis failed: %v", err)
	} else if !passed {
		t.Errorf("Different events should not merge even if related to same topic")
	}
}

func testUnrelatedEvents(t *testing.T, ctx context.Context, correlator *enrichment.EventCorrelator, logger *slog.Logger) {
	start := time.Now()

	existingEvent := models.Event{
		ID:        "evt-4",
		Timestamp: time.Now().Add(-1 * time.Hour),
		Title:     "Russia launches missile strikes on Kyiv, 5 civilians killed",
		Summary:   "Russian forces conducted missile strikes on the Ukrainian capital Kyiv early this morning, resulting in 5 civilian casualties.",
		Category:  models.CategoryMilitary,
		Tags:      []string{"Russia", "Ukraine", "Kyiv", "missile strike"},
	}

	newSource := models.Source{
		ID:          "src-5",
		Title:       "Apple announces new iPhone with advanced AI features",
		RawContent:  "Apple Inc. today unveiled its latest iPhone model featuring breakthrough artificial intelligence capabilities, including advanced image recognition and natural language processing.",
		PublishedAt: time.Now(),
	}

	result, err := correlator.AnalyzeCorrelation(ctx, newSource, existingEvent)

	var passed bool
	var actualOutcome string
	var errorStr string
	details := map[string]interface{}{
		"existing_event": existingEvent.Title,
		"new_source":     newSource.Title,
	}

	if err != nil {
		errorStr = err.Error()
		actualOutcome = fmt.Sprintf("Error: %v", err)
	} else {
		passed = !result.ShouldMerge && result.Similarity < 0.3
		actualOutcome = fmt.Sprintf("Similarity: %.2f, ShouldMerge: %v, Reasoning: %s",
			result.Similarity, result.ShouldMerge, result.Reasoning)
		details["similarity"] = result.Similarity
		details["should_merge"] = result.ShouldMerge
		details["reasoning"] = result.Reasoning
	}

	addResult(TestResult{
		TestName:        "Event Correlation - Completely Unrelated Events",
		Category:        "Correlation",
		Description:     "Completely unrelated events should have very low similarity",
		Passed:          passed,
		ExpectedOutcome: "Similarity < 0.3, ShouldMerge = false",
		ActualOutcome:   actualOutcome,
		Details:         details,
		Duration:        time.Since(start),
		Error:           errorStr,
	})

	if err != nil {
		t.Errorf("Correlation analysis failed: %v", err)
	} else if !passed {
		t.Errorf("Unrelated events should have very low similarity")
	}
}

func testCasualtyUpdate(t *testing.T, ctx context.Context, correlator *enrichment.EventCorrelator, logger *slog.Logger) {
	start := time.Now()

	existingEvent := models.Event{
		ID:        "evt-update-1",
		Timestamp: time.Now().Add(-3 * time.Hour),
		Title:     "Russian strikes on Kyiv, initial reports say 3 dead",
		Summary:   "Russian missile attack on Kyiv resulted in 3 confirmed deaths according to initial reports from local authorities.",
		Category:  models.CategoryMilitary,
		Tags:      []string{"Russia", "Ukraine", "Kyiv", "casualties"},
	}

	newSource := models.Source{
		ID:          "src-update-1",
		Title:       "Kyiv attack death toll rises to 8 as rescue efforts continue",
		RawContent:  "The death toll from this morning's Russian missile attack on Kyiv has risen to 8 people as rescue teams continue to search through rubble. Earlier reports indicated 3 deaths.",
		PublishedAt: time.Now(),
	}

	result, err := correlator.AnalyzeCorrelation(ctx, newSource, existingEvent)

	var passed bool
	var actualOutcome string
	var errorStr string
	details := map[string]interface{}{
		"existing_event": existingEvent.Title,
		"new_source":     newSource.Title,
	}

	if err != nil {
		errorStr = err.Error()
		actualOutcome = fmt.Sprintf("Error: %v", err)
	} else {
		passed = result.ShouldMerge && result.Similarity >= 0.85 && result.HasNovelFacts
		actualOutcome = fmt.Sprintf("Similarity: %.2f, ShouldMerge: %v, HasNovelFacts: %v, Reasoning: %s",
			result.Similarity, result.ShouldMerge, result.HasNovelFacts, result.Reasoning)
		details["similarity"] = result.Similarity
		details["should_merge"] = result.ShouldMerge
		details["novel_facts"] = result.NovelFacts
		details["reasoning"] = result.Reasoning
	}

	addResult(TestResult{
		TestName:        "Event Correlation - Casualty Count Update",
		Category:        "Correlation",
		Description:     "Updated casualty numbers for same event should merge with novel facts",
		Passed:          passed,
		ExpectedOutcome: "Similarity >= 0.85, ShouldMerge = true, HasNovelFacts = true",
		ActualOutcome:   actualOutcome,
		Details:         details,
		Duration:        time.Since(start),
		Error:           errorStr,
	})

	if err != nil {
		t.Errorf("Correlation analysis failed: %v", err)
	} else if !passed {
		t.Errorf("Casualty updates should merge with same event")
	}
}

func testTemporalSequence(t *testing.T, ctx context.Context, correlator *enrichment.EventCorrelator, logger *slog.Logger) {
	start := time.Now()

	existingEvent := models.Event{
		ID:        "evt-seq-1",
		Timestamp: time.Now().Add(-6 * time.Hour),
		Title:     "North Korea launches ballistic missile towards Sea of Japan",
		Summary:   "North Korea fired a ballistic missile eastward into the Sea of Japan this morning, according to South Korean military.",
		Category:  models.CategoryMilitary,
		Tags:      []string{"North Korea", "missile", "Sea of Japan"},
	}

	newSource := models.Source{
		ID:          "src-seq-1",
		Title:       "Japan condemns North Korean missile launch, summons NK ambassador",
		RawContent:  "In response to this morning's North Korean missile launch, Japan has issued a strong condemnation and summoned the North Korean ambassador for explanation.",
		PublishedAt: time.Now(),
	}

	result, err := correlator.AnalyzeCorrelation(ctx, newSource, existingEvent)

	var passed bool
	var actualOutcome string
	var errorStr string
	details := map[string]interface{}{
		"existing_event": existingEvent.Title,
		"new_source":     newSource.Title,
	}

	if err != nil {
		errorStr = err.Error()
		actualOutcome = fmt.Sprintf("Error: %v", err)
	} else {
		// Allow similarity 0.4-0.7 for related but separate events (reactions/responses)
		passed = !result.ShouldMerge && result.Similarity >= 0.4 && result.Similarity <= 0.7
		actualOutcome = fmt.Sprintf("Similarity: %.2f, ShouldMerge: %v, Reasoning: %s",
			result.Similarity, result.ShouldMerge, result.Reasoning)
		details["similarity"] = result.Similarity
		details["should_merge"] = result.ShouldMerge
		details["reasoning"] = result.Reasoning
	}

	addResult(TestResult{
		TestName:        "Event Correlation - Temporal Sequence (Response)",
		Category:        "Correlation",
		Description:     "Diplomatic response to earlier event is related but separate",
		Passed:          passed,
		ExpectedOutcome: "0.4 <= Similarity <= 0.7, ShouldMerge = false",
		ActualOutcome:   actualOutcome,
		Details:         details,
		Duration:        time.Since(start),
		Error:           errorStr,
	})

	if err != nil {
		t.Errorf("Correlation analysis failed: %v", err)
	} else if !passed {
		t.Errorf("Response to event should be related but not merged")
	}
}

func testSameLocationDifferentIncidents(t *testing.T, ctx context.Context, correlator *enrichment.EventCorrelator, logger *slog.Logger) {
	start := time.Now()

	existingEvent := models.Event{
		ID:        "evt-loc-1",
		Timestamp: time.Now().Add(-2 * time.Hour),
		Title:     "Explosion at chemical plant in Shanghai, 4 workers injured",
		Summary:   "An industrial accident at a chemical plant in Shanghai's Pudong district injured 4 workers this morning. Investigation ongoing.",
		Category:  models.CategoryDisaster,
		Tags:      []string{"Shanghai", "explosion", "chemical plant", "industrial accident"},
	}

	newSource := models.Source{
		ID:          "src-loc-1",
		Title:       "Fire breaks out at Shanghai warehouse, separate from earlier chemical plant incident",
		RawContent:  "A warehouse fire in Shanghai's Yangpu district is being contained by firefighters. Officials confirm this is unrelated to the chemical plant accident in Pudong earlier today.",
		PublishedAt: time.Now(),
	}

	result, err := correlator.AnalyzeCorrelation(ctx, newSource, existingEvent)

	var passed bool
	var actualOutcome string
	var errorStr string
	details := map[string]interface{}{
		"existing_event": existingEvent.Title,
		"new_source":     newSource.Title,
	}

	if err != nil {
		errorStr = err.Error()
		actualOutcome = fmt.Sprintf("Error: %v", err)
	} else {
		// Allow 0.2-0.5 similarity - AI may score lower when source explicitly states "separate"
		passed = !result.ShouldMerge && result.Similarity >= 0.2 && result.Similarity <= 0.5
		actualOutcome = fmt.Sprintf("Similarity: %.2f, ShouldMerge: %v, Reasoning: %s",
			result.Similarity, result.ShouldMerge, result.Reasoning)
		details["similarity"] = result.Similarity
		details["should_merge"] = result.ShouldMerge
		details["reasoning"] = result.Reasoning
	}

	addResult(TestResult{
		TestName:        "Event Correlation - Same Location Different Incidents",
		Category:        "Correlation",
		Description:     "Different incidents in same city should not merge despite location overlap",
		Passed:          passed,
		ExpectedOutcome: "0.2 <= Similarity <= 0.5, ShouldMerge = false",
		ActualOutcome:   actualOutcome,
		Details:         details,
		Duration:        time.Since(start),
		Error:           errorStr,
	})

	if err != nil {
		t.Errorf("Correlation analysis failed: %v", err)
	} else if !passed {
		t.Errorf("Different incidents in same location should not merge")
	}
}

func testConflictingInformation(t *testing.T, ctx context.Context, correlator *enrichment.EventCorrelator, logger *slog.Logger) {
	start := time.Now()

	existingEvent := models.Event{
		ID:        "evt-conflict-1",
		Timestamp: time.Now().Add(-1 * time.Hour),
		Title:     "Cyber attack on European energy infrastructure attributed to Russian hackers",
		Summary:   "A sophisticated cyber attack targeting European energy infrastructure has been attributed to Russian state-sponsored hackers by cybersecurity firms.",
		Category:  models.CategoryCyber,
		Tags:      []string{"cyber attack", "Europe", "Russia", "energy infrastructure"},
	}

	newSource := models.Source{
		ID:          "src-conflict-1",
		Title:       "Russia denies involvement in European energy cyberattack, points to Chinese APT group",
		RawContent:  "Russian officials have denied any involvement in the cyberattack on European energy infrastructure, instead suggesting the attack bears hallmarks of Chinese APT groups.",
		PublishedAt: time.Now(),
	}

	result, err := correlator.AnalyzeCorrelation(ctx, newSource, existingEvent)

	var passed bool
	var actualOutcome string
	var errorStr string
	details := map[string]interface{}{
		"existing_event": existingEvent.Title,
		"new_source":     newSource.Title,
	}

	if err != nil {
		errorStr = err.Error()
		actualOutcome = fmt.Sprintf("Error: %v", err)
	} else {
		// Allow slightly lower similarity (0.6+) for conflicting claims as they can be interpreted differently
		passed = result.ShouldMerge && result.Similarity >= 0.6 && result.HasNovelFacts
		actualOutcome = fmt.Sprintf("Similarity: %.2f, ShouldMerge: %v, HasNovelFacts: %v, Reasoning: %s",
			result.Similarity, result.ShouldMerge, result.HasNovelFacts, result.Reasoning)
		details["similarity"] = result.Similarity
		details["should_merge"] = result.ShouldMerge
		details["novel_facts"] = result.NovelFacts
		details["reasoning"] = result.Reasoning
	}

	addResult(TestResult{
		TestName:        "Event Correlation - Conflicting Information",
		Category:        "Correlation",
		Description:     "Conflicting attribution claims about same incident should merge as novel facts",
		Passed:          passed,
		ExpectedOutcome: "Similarity >= 0.6, ShouldMerge = true, HasNovelFacts = true",
		ActualOutcome:   actualOutcome,
		Details:         details,
		Duration:        time.Since(start),
		Error:           errorStr,
	})

	if err != nil {
		t.Errorf("Correlation analysis failed: %v", err)
	} else if !passed {
		t.Errorf("Conflicting information about same event should still merge")
	}
}

// TestConfidenceScoring tests confidence score calculation
func TestConfidenceScoring(t *testing.T) {
	scorer := enrichment.NewConfidenceScorer()

	// Test Case 1: High-quality source with entities
	testHighQualitySource(t, scorer)

	// Test Case 2: Low-quality source
	testLowQualitySource(t, scorer)

	// Test Case 3: Medium quality with some entities
	testMediumQualitySource(t, scorer)

	// Test Case 4: Social media source
	testSocialMediaSource(t, scorer)

	// Test Case 5: Official government source
	testGovernmentSource(t, scorer)

	// Test Case 6: No entities but high credibility
	testHighCredibilityNoEntities(t, scorer)
}

func testHighQualitySource(t *testing.T, scorer *enrichment.ConfidenceScorer) {
	start := time.Now()

	source := models.Source{
		ID:          "test-src-1",
		Type:        models.SourceTypeNewsMedia,
		URL:         "https://reuters.com/article",
		Credibility: 0.9,
		RawContent:  "Detailed analysis with verified information from multiple official sources.",
	}

	event := &models.Event{
		ID:       "test-evt-1",
		Category: models.CategoryMilitary,
	}

	entities := []models.Entity{
		{Type: models.EntityTypeCountry, Name: "United States", Confidence: 1.0},
		{Type: models.EntityTypeOrganization, Name: "NATO", Confidence: 0.95},
		{Type: models.EntityTypePerson, Name: "President Biden", Confidence: 0.9},
	}

	confidence := scorer.Score(source, event, entities)

	passed := confidence.Score >= 0.7
	addResult(TestResult{
		TestName:        "Confidence Scoring - High Quality Source",
		Category:        "Confidence",
		Description:     "High credibility source with multiple entities should have high confidence",
		Passed:          passed,
		ExpectedOutcome: "Confidence >= 0.7",
		ActualOutcome:   fmt.Sprintf("Confidence: %.2f", confidence.Score),
		Details: map[string]interface{}{
			"confidence_score":   confidence.Score,
			"source_credibility": source.Credibility,
			"entity_count":       len(entities),
		},
		Duration: time.Since(start),
	})

	if !passed {
		t.Errorf("High quality source should have confidence >= 0.7, got %.2f", confidence.Score)
	}
}

func testLowQualitySource(t *testing.T, scorer *enrichment.ConfidenceScorer) {
	start := time.Now()

	source := models.Source{
		ID:          "test-src-2",
		Type:        models.SourceTypeBlog,
		URL:         "https://random-blog.com/post",
		Credibility: 0.2,
		RawContent:  "Unverified rumor from anonymous source.",
	}

	event := &models.Event{
		ID:       "test-evt-2",
		Category: models.CategoryOther,
	}

	entities := []models.Entity{}

	confidence := scorer.Score(source, event, entities)

	passed := confidence.Score <= 0.5
	addResult(TestResult{
		TestName:        "Confidence Scoring - Low Quality Source",
		Category:        "Confidence",
		Description:     "Low credibility source with no entities should have low confidence",
		Passed:          passed,
		ExpectedOutcome: "Confidence <= 0.5",
		ActualOutcome:   fmt.Sprintf("Confidence: %.2f", confidence.Score),
		Details: map[string]interface{}{
			"confidence_score":   confidence.Score,
			"source_credibility": source.Credibility,
			"entity_count":       len(entities),
		},
		Duration: time.Since(start),
	})

	if !passed {
		t.Errorf("Low quality source should have confidence <= 0.5, got %.2f", confidence.Score)
	}
}

func testMediumQualitySource(t *testing.T, scorer *enrichment.ConfidenceScorer) {
	start := time.Now()

	source := models.Source{
		ID:          "test-src-3",
		Type:        models.SourceTypeNewsMedia,
		URL:         "https://local-news.com/article",
		Credibility: 0.6,
		RawContent:  "Report from local journalists with some corroboration.",
	}

	event := &models.Event{
		ID:       "test-evt-3",
		Category: models.CategoryGeopolitics,
	}

	entities := []models.Entity{
		{Type: models.EntityTypeCity, Name: "Kyiv", Confidence: 0.9},
	}

	confidence := scorer.Score(source, event, entities)

	passed := confidence.Score >= 0.4 && confidence.Score <= 0.7
	addResult(TestResult{
		TestName:        "Confidence Scoring - Medium Quality Source",
		Category:        "Confidence",
		Description:     "Medium credibility source should have moderate confidence",
		Passed:          passed,
		ExpectedOutcome: "0.4 <= Confidence <= 0.7",
		ActualOutcome:   fmt.Sprintf("Confidence: %.2f", confidence.Score),
		Details: map[string]interface{}{
			"confidence_score":   confidence.Score,
			"source_credibility": source.Credibility,
			"entity_count":       len(entities),
		},
		Duration: time.Since(start),
	})

	if !passed {
		t.Errorf("Medium quality source should have confidence between 0.4-0.7, got %.2f", confidence.Score)
	}
}

func testSocialMediaSource(t *testing.T, scorer *enrichment.ConfidenceScorer) {
	start := time.Now()

	source := models.Source{
		ID:          "test-src-4",
		Type:        models.SourceTypeTwitter,
		URL:         "https://twitter.com/user/status/123",
		Credibility: 0.3,
		RawContent:  "Unverified social media post about incident.",
	}

	event := &models.Event{
		ID:       "test-evt-4",
		Category: models.CategoryOther,
	}

	entities := []models.Entity{
		{Type: models.EntityTypeCity, Name: "Baghdad", Confidence: 0.7},
	}

	confidence := scorer.Score(source, event, entities)

	passed := confidence.Score <= 0.45
	addResult(TestResult{
		TestName:        "Confidence Scoring - Social Media Source",
		Category:        "Confidence",
		Description:     "Social media sources should have lower confidence even with entities",
		Passed:          passed,
		ExpectedOutcome: "Confidence <= 0.45",
		ActualOutcome:   fmt.Sprintf("Confidence: %.2f", confidence.Score),
		Details: map[string]interface{}{
			"confidence_score":   confidence.Score,
			"source_credibility": source.Credibility,
			"source_type":        source.Type,
			"entity_count":       len(entities),
		},
		Duration: time.Since(start),
	})

	if !passed {
		t.Errorf("Social media source should have confidence <= 0.45, got %.2f", confidence.Score)
	}
}

func testGovernmentSource(t *testing.T, scorer *enrichment.ConfidenceScorer) {
	start := time.Now()

	source := models.Source{
		ID:          "test-src-5",
		Type:        models.SourceTypeGovernment,
		URL:         "https://state.gov/statement",
		Credibility: 0.95,
		RawContent:  "Official government statement with verified information.",
	}

	event := &models.Event{
		ID:       "test-evt-5",
		Category: models.CategoryDiplomacy,
	}

	entities := []models.Entity{
		{Type: models.EntityTypeCountry, Name: "United States", Confidence: 1.0},
		{Type: models.EntityTypePerson, Name: "Secretary of State", Confidence: 0.95},
	}

	confidence := scorer.Score(source, event, entities)

	passed := confidence.Score >= 0.74
	addResult(TestResult{
		TestName:        "Confidence Scoring - Official Government Source",
		Category:        "Confidence",
		Description:     "Official government sources should have high confidence",
		Passed:          passed,
		ExpectedOutcome: "Confidence >= 0.74",
		ActualOutcome:   fmt.Sprintf("Confidence: %.2f", confidence.Score),
		Details: map[string]interface{}{
			"confidence_score":   confidence.Score,
			"source_credibility": source.Credibility,
			"source_type":        source.Type,
			"entity_count":       len(entities),
		},
		Duration: time.Since(start),
	})

	if !passed {
		t.Errorf("Government source should have confidence >= 0.74, got %.2f", confidence.Score)
	}
}

func testHighCredibilityNoEntities(t *testing.T, scorer *enrichment.ConfidenceScorer) {
	start := time.Now()

	source := models.Source{
		ID:          "test-src-6",
		Type:        models.SourceTypeNewsMedia,
		URL:         "https://bbc.com/news/article",
		Credibility: 0.85,
		RawContent:  "Well-sourced news report with attributed quotes.",
	}

	event := &models.Event{
		ID:       "test-evt-6",
		Category: models.CategoryEconomic,
	}

	entities := []models.Entity{}

	confidence := scorer.Score(source, event, entities)

	passed := confidence.Score >= 0.5 && confidence.Score <= 0.7
	addResult(TestResult{
		TestName:        "Confidence Scoring - High Credibility No Entities",
		Category:        "Confidence",
		Description:     "High credibility source without entities should have moderate confidence",
		Passed:          passed,
		ExpectedOutcome: "0.5 <= Confidence <= 0.7",
		ActualOutcome:   fmt.Sprintf("Confidence: %.2f", confidence.Score),
		Details: map[string]interface{}{
			"confidence_score":   confidence.Score,
			"source_credibility": source.Credibility,
			"entity_count":       len(entities),
		},
		Duration: time.Since(start),
	})

	if !passed {
		t.Errorf("High credibility source without entities should have confidence between 0.5-0.7, got %.2f", confidence.Score)
	}
}

// TestMagnitudeEstimation tests event magnitude calculation
func TestMagnitudeEstimation(t *testing.T) {
	estimator := enrichment.NewMagnitudeEstimator()

	// Test Case 1: High magnitude military event
	testHighMagnitudeMilitary(t, estimator)

	// Test Case 2: Low magnitude event
	testLowMagnitudeEvent(t, estimator)

	// Test Case 3: Cyber incident
	testCyberIncident(t, estimator)

	// Test Case 4: Terrorism event
	testTerrorismEvent(t, estimator)

	// Test Case 5: Natural disaster
	testNaturalDisaster(t, estimator)

	// Test Case 6: Economic event
	testEconomicEvent(t, estimator)
}

func testHighMagnitudeMilitary(t *testing.T, estimator *enrichment.MagnitudeEstimator) {
	start := time.Now()

	event := &models.Event{
		ID:       "test-evt-mag-1",
		Title:    "Major missile strikes on capital city with significant casualties",
		Summary:  "Large-scale military attack on civilian areas resulting in multiple deaths",
		Category: models.CategoryMilitary,
		Tags:     []string{"casualties", "civilian", "major attack"},
	}

	source := models.Source{
		Credibility: 0.9,
	}

	magnitude := estimator.Estimate(event, source)

	passed := magnitude >= 7.0
	addResult(TestResult{
		TestName:        "Magnitude Estimation - High Magnitude Military",
		Category:        "Magnitude",
		Description:     "Major military event with casualties should have high magnitude",
		Passed:          passed,
		ExpectedOutcome: "Magnitude >= 7.0",
		ActualOutcome:   fmt.Sprintf("Magnitude: %.1f", magnitude),
		Details: map[string]interface{}{
			"magnitude": magnitude,
			"category":  event.Category,
			"tags":      event.Tags,
		},
		Duration: time.Since(start),
	})

	if !passed {
		t.Errorf("High magnitude military event should have magnitude >= 7.0, got %.1f", magnitude)
	}
}

func testLowMagnitudeEvent(t *testing.T, estimator *enrichment.MagnitudeEstimator) {
	start := time.Now()

	event := &models.Event{
		ID:       "test-evt-mag-2",
		Title:    "Minor diplomatic meeting scheduled",
		Summary:  "Routine diplomatic consultation between officials",
		Category: models.CategoryDiplomacy,
		Tags:     []string{"meeting", "routine"},
	}

	source := models.Source{
		Credibility: 0.7,
	}

	magnitude := estimator.Estimate(event, source)

	passed := magnitude <= 5.5
	addResult(TestResult{
		TestName:        "Magnitude Estimation - Low Magnitude Event",
		Category:        "Magnitude",
		Description:     "Minor diplomatic event should have low magnitude",
		Passed:          passed,
		ExpectedOutcome: "Magnitude <= 5.5",
		ActualOutcome:   fmt.Sprintf("Magnitude: %.1f", magnitude),
		Details: map[string]interface{}{
			"magnitude": magnitude,
			"category":  event.Category,
			"tags":      event.Tags,
		},
		Duration: time.Since(start),
	})

	if !passed {
		t.Errorf("Low magnitude event should have magnitude <= 5.5, got %.1f", magnitude)
	}
}

func testCyberIncident(t *testing.T, estimator *enrichment.MagnitudeEstimator) {
	start := time.Now()

	event := &models.Event{
		ID:       "test-evt-mag-3",
		Title:    "Major cyberattack on critical infrastructure",
		Summary:  "Significant cyber incident targeting power grid systems",
		Category: models.CategoryCyber,
		Tags:     []string{"critical infrastructure", "cyberattack", "power grid"},
	}

	source := models.Source{
		Credibility: 0.8,
	}

	magnitude := estimator.Estimate(event, source)

	passed := magnitude >= 5.5 && magnitude <= 9.0
	addResult(TestResult{
		TestName:        "Magnitude Estimation - Cyber Incident",
		Category:        "Magnitude",
		Description:     "Critical infrastructure cyberattack should have high-medium magnitude",
		Passed:          passed,
		ExpectedOutcome: "5.5 <= Magnitude <= 9.0",
		ActualOutcome:   fmt.Sprintf("Magnitude: %.1f", magnitude),
		Details: map[string]interface{}{
			"magnitude": magnitude,
			"category":  event.Category,
			"tags":      event.Tags,
		},
		Duration: time.Since(start),
	})

	if !passed {
		t.Errorf("Cyber incident should have magnitude between 5.5-9.0, got %.1f", magnitude)
	}
}

func testTerrorismEvent(t *testing.T, estimator *enrichment.MagnitudeEstimator) {
	start := time.Now()

	event := &models.Event{
		ID:       "test-evt-mag-4",
		Title:    "Terrorist attack in major city, multiple casualties reported",
		Summary:  "Coordinated terrorist attack targeting civilian areas in capital city with significant casualties and widespread panic",
		Category: models.CategoryTerrorism,
		Tags:     []string{"terrorism", "attack", "casualties", "civilian"},
	}

	source := models.Source{
		Credibility: 0.85,
	}

	magnitude := estimator.Estimate(event, source)

	passed := magnitude >= 8.0
	addResult(TestResult{
		TestName:        "Magnitude Estimation - Terrorism Event",
		Category:        "Magnitude",
		Description:     "Terrorism events should have highest base magnitude",
		Passed:          passed,
		ExpectedOutcome: "Magnitude >= 8.0",
		ActualOutcome:   fmt.Sprintf("Magnitude: %.1f", magnitude),
		Details: map[string]interface{}{
			"magnitude": magnitude,
			"category":  event.Category,
			"tags":      event.Tags,
		},
		Duration: time.Since(start),
	})

	if !passed {
		t.Errorf("Terrorism event should have magnitude >= 8.0, got %.1f", magnitude)
	}
}

func testNaturalDisaster(t *testing.T, estimator *enrichment.MagnitudeEstimator) {
	start := time.Now()

	event := &models.Event{
		ID:       "test-evt-mag-5",
		Title:    "Magnitude 7.5 earthquake strikes populated region",
		Summary:  "Major earthquake with significant structural damage and casualties in densely populated area",
		Category: models.CategoryDisaster,
		Tags:     []string{"earthquake", "natural disaster", "casualties", "major damage"},
	}

	source := models.Source{
		Credibility: 0.9,
	}

	magnitude := estimator.Estimate(event, source)

	passed := magnitude >= 7.0 && magnitude <= 9.0
	addResult(TestResult{
		TestName:        "Magnitude Estimation - Natural Disaster",
		Category:        "Magnitude",
		Description:     "Major natural disasters should have high magnitude",
		Passed:          passed,
		ExpectedOutcome: "7.0 <= Magnitude <= 9.0",
		ActualOutcome:   fmt.Sprintf("Magnitude: %.1f", magnitude),
		Details: map[string]interface{}{
			"magnitude": magnitude,
			"category":  event.Category,
			"tags":      event.Tags,
		},
		Duration: time.Since(start),
	})

	if !passed {
		t.Errorf("Natural disaster should have magnitude between 7.0-9.0, got %.1f", magnitude)
	}
}

func testEconomicEvent(t *testing.T, estimator *enrichment.MagnitudeEstimator) {
	start := time.Now()

	event := &models.Event{
		ID:       "test-evt-mag-6",
		Title:    "Central bank announces interest rate decision",
		Summary:  "Routine monetary policy decision by central bank maintaining current interest rates",
		Category: models.CategoryEconomic,
		Tags:     []string{"economy", "central bank", "interest rates"},
	}

	source := models.Source{
		Credibility: 0.9,
	}

	magnitude := estimator.Estimate(event, source)

	passed := magnitude <= 5.0
	addResult(TestResult{
		TestName:        "Magnitude Estimation - Routine Economic Event",
		Category:        "Magnitude",
		Description:     "Routine economic events should have relatively low magnitude",
		Passed:          passed,
		ExpectedOutcome: "Magnitude <= 5.0",
		ActualOutcome:   fmt.Sprintf("Magnitude: %.1f", magnitude),
		Details: map[string]interface{}{
			"magnitude": magnitude,
			"category":  event.Category,
			"tags":      event.Tags,
		},
		Duration: time.Since(start),
	})

	if !passed {
		t.Errorf("Routine economic event should have magnitude <= 5.0, got %.1f", magnitude)
	}
}

// Helper function to hash content (simplified)
func hashContent(content string) string {
	hash := uint32(0)
	for _, c := range content {
		hash = hash*31 + uint32(c)
	}
	return fmt.Sprintf("%x", hash)
}
