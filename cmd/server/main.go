package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/STRATINT/stratint/internal/api"
	"github.com/STRATINT/stratint/internal/auth"
	"github.com/STRATINT/stratint/internal/cloudsql"
	"github.com/STRATINT/stratint/internal/config"
	"github.com/STRATINT/stratint/internal/database"
	"github.com/STRATINT/stratint/internal/enrichment"
	"github.com/STRATINT/stratint/internal/eventmanager"
	"github.com/STRATINT/stratint/internal/forecaster"
	"github.com/STRATINT/stratint/internal/inference"
	"github.com/STRATINT/stratint/internal/ingestion"
	"github.com/STRATINT/stratint/internal/logging"
	"github.com/STRATINT/stratint/internal/metrics"
	"github.com/STRATINT/stratint/internal/models"
	"github.com/STRATINT/stratint/internal/scheduler"
	"github.com/STRATINT/stratint/internal/server"
	"github.com/STRATINT/stratint/internal/social"
	"github.com/STRATINT/stratint/internal/strategist"
	_ "github.com/lib/pq"
	"log/slog"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.New(slog.NewJSONHandler(os.Stdout, nil)).Error("failed to load config", "error", err)
		os.Exit(1)
	}

	logger, err := logging.New(cfg.Logging)
	if err != nil {
		slog.New(slog.NewJSONHandler(os.Stdout, nil)).Error("failed to init logger", "error", err)
		os.Exit(1)
	}

	logger.Info("starting OSINTMCP")

	// Connect to database (supports both local DATABASE_URL and Cloud SQL)
	dbURL, err := cloudsql.BuildDatabaseURL()
	if err != nil {
		logger.Error("failed to build database URL", "error", err)
		os.Exit(1)
	}

	// Log connection config (without sensitive data)
	connConfig := cloudsql.GetConnectionConfig()
	logger.Info("database configuration", "config", connConfig)

	logger.Info("connecting to database")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		logger.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		logger.Error("failed to ping database", "error", err)
		os.Exit(1)
	}
	logger.Info("database connected")

	// Run pending migrations (non-fatal to allow app to start even if migrations fail)
	if err := database.RunMigrations(db, "./migrations", logger); err != nil {
		logger.Warn("failed to run migrations, continuing anyway", "error", err)
	}

	// Create repositories
	sourceRepo := database.NewPostgresSourceRepository(db)
	eventRepo := database.NewPostgresEventRepository(db)
	trackedAccountRepo := database.NewPostgresTrackedAccountRepository(db)
	errorRepo := database.NewPostgresIngestionErrorRepository(db)
	thresholdRepo := database.NewThresholdRepository(db)
	activityLogRepo := database.NewActivityLogRepository(db)
	openaiConfigRepo := database.NewOpenAIConfigRepository(db)
	connectorConfigRepo := database.NewConnectorConfigRepository(db)
	// Scraping functionality removed - using RSS content only
	twitterRepo := database.NewTwitterRepository(db)
	inferenceLogRepo := database.NewInferenceLogRepository(db)

	// Create inference logger
	inferenceLogger := inference.NewLogger(inferenceLogRepo, logger)

	// Create enricher using database configuration
	var enricher enrichment.Enricher
	var credibilityCache *enrichment.CredibilityCache
	openaiEnricher, err := enrichment.NewOpenAIClientFromDB(context.Background(), openaiConfigRepo, logger, inferenceLogger)
	if err != nil {
		logger.Warn("failed to initialize OpenAI enricher from database, using mock enricher", "error", err)
		enricher = enrichment.NewMockEnricher()
	} else {
		logger.Info("using OpenAI enricher from database config")
		enricher = openaiEnricher
		// Create credibility cache with 24h TTL
		credibilityCache = enrichment.NewCredibilityCache(openaiEnricher, 24*time.Hour)
	}

	// Create Twitter poster if OpenAI is available
	var twitterPoster eventmanager.TwitterPoster
	if openaiEnricher != nil {
		poster, err := social.NewTwitterPoster(twitterRepo, openaiEnricher, logger)
		if err != nil {
			logger.Warn("failed to initialize twitter poster", "error", err)
		} else {
			twitterPoster = poster
			logger.Info("twitter poster initialized")
		}
	}

	// Create event manager
	lifecycleConfig := eventmanager.DefaultLifecycleConfig()
	eventManager := eventmanager.NewEventLifecycleManager(
		sourceRepo,
		eventRepo,
		enricher,
		thresholdRepo,
		twitterPoster,
		activityLogRepo,
		logger,
		lifecycleConfig,
	)

	// Scraping functionality removed - using RSS content only
	logger.Info("application running with RSS-only ingestion (no web scraping)")

	// Setup HTTP routes
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	// Service info endpoint
	mux.HandleFunc("/api/info", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"service":"osintmcp","status":"ready","version":"0.1.0"}`))
	})

	collector, err := metrics.NewHTTPCollector()
	if err != nil {
		logger.Error("failed to init metrics", "error", err)
		os.Exit(1)
	}
	mux.Handle("/metrics", collector.Handler())

	// Load auth configuration
	authConfig := auth.LoadConfigFromEnv()
	logger.Info("auth configured", "jwt_secret_set", authConfig.JWTSecret != "change-this-secret")

	// Get FRED API key from environment
	fredAPIKey := os.Getenv("FRED_API_KEY")
	if fredAPIKey == "" {
		logger.Warn("FRED_API_KEY not set, FRED endpoints will not work")
	}

	// Add REST API routes
	logger.Info("setting up REST API")
	api.SetupRoutes(mux, db, eventManager, sourceRepo, eventRepo, trackedAccountRepo, errorRepo, thresholdRepo, activityLogRepo, openaiConfigRepo, connectorConfigRepo, twitterRepo, twitterPoster, credibilityCache, enricher, authConfig, fredAPIKey, logger)

	// MCP endpoint (Model Context Protocol)
	mcpHandler := eventmanager.NewMCPHandler(eventManager)
	mux.HandleFunc("/mcp/events", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var request struct {
			Query string `json:"query"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		response, err := mcpHandler.GetEvents(r.Context(), request.Query)
		if err != nil {
			logger.Error("MCP query failed", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(response))
	})

	// Start RSS feed monitoring
	logger.Info("starting RSS monitoring")
	go func() {
		ticker := time.NewTicker(1 * time.Minute) // Check every 1 minute
		defer ticker.Stop()

		time.Sleep(5 * time.Second) // Initial delay

		for {
			accounts, err := trackedAccountRepo.ListByPlatform("rss", true)
			if err != nil {
				logger.Error("failed to list tracked RSS feeds", "error", err)
			} else if len(accounts) > 0 {
				logger.Debug("checking tracked RSS feeds", "count", len(accounts))

				for _, account := range accounts {
					// Check if enough time has elapsed since last fetch
					now := time.Now()
					if account.LastFetchedAt != nil {
						intervalDuration := time.Duration(account.FetchIntervalMinutes) * time.Minute
						nextFetchTime := account.LastFetchedAt.Add(intervalDuration)

						if now.Before(nextFetchTime) {
							timeUntilNext := nextFetchTime.Sub(now)
							logger.Debug("skipping RSS feed - not time yet",
								"feed", account.AccountIdentifier,
								"interval_minutes", account.FetchIntervalMinutes,
								"last_fetched", account.LastFetchedAt.Format(time.RFC3339),
								"next_fetch_in", timeUntilNext.Round(time.Second).String())
							continue
						}
					}

					logger.Info("fetching RSS feed",
						"feed", account.AccountIdentifier,
						"interval_minutes", account.FetchIntervalMinutes)

					// Create connector with single feed
					rssConnector, err := ingestion.NewRSSConnector([]string{account.AccountIdentifier}, logger, errorRepo, activityLogRepo)
					if err != nil {
						logger.Error("failed to create RSS connector",
							"feed", account.AccountIdentifier,
							"error", err)
						continue
					}

					sources, err := rssConnector.Fetch()
					if err != nil {
						rssConnector.Close()
						logger.Error("failed to fetch RSS feed",
							"feed", account.AccountIdentifier,
							"error", err)
						continue
					}

					if len(sources) > 0 {
						logger.Info("fetched new RSS items",
							"feed", account.AccountIdentifier,
							"count", len(sources))

						storedCount := 0
						for _, source := range sources {
							// Check if source already exists (deduplicate by title + URL)
							existing, err := sourceRepo.GetByTitleAndURL(context.Background(), source.Title, source.URL)
							if err != nil {
								logger.Error("failed to check for duplicate source", "error", err)
								continue
							}
							if existing != nil {
								logger.Debug("skipping duplicate source", "title", source.Title)
								continue
							}

							if err := sourceRepo.Store(context.Background(), source); err != nil {
								logger.Error("failed to store RSS source", "error", err)
							} else {
								storedCount++
							}
						}

						if storedCount > 0 {
							logger.Info("stored new sources", "count", storedCount)
						}

						// Update last fetched timestamp
						if len(sources) > 0 {
							// Use the first source's ID as the marker
							trackedAccountRepo.UpdateLastFetched(account.ID, sources[0].ID, time.Now())
						}
					}

					// Close the RSS connector after processing
					rssConnector.Close()
				}
			}

			<-ticker.C
		}
	}()

	// Start Twitter account monitoring if enabled in database
	logger.Info("starting Twitter monitoring")
	go func() {
		ticker := time.NewTicker(2 * time.Minute) // Check every 2 minutes
		defer ticker.Stop()

		// Initial check after 10 seconds
		time.Sleep(10 * time.Second)

		for {
			// Get Twitter config from database
			ctx := context.Background()
			twitterConfig, err := connectorConfigRepo.Get(ctx, "twitter")
			if err != nil || !twitterConfig.Enabled {
				logger.Debug("Twitter connector not enabled, skipping")
				<-ticker.C
				continue
			}

			bearerToken := twitterConfig.Config["bearer_token"]
			if bearerToken == "" {
				logger.Debug("Twitter bearer token not configured")
				<-ticker.C
				continue
			}

			twitterConnector := ingestion.NewTwitterConnector(bearerToken, logger, credibilityCache)

			accounts, err := trackedAccountRepo.ListByPlatform("twitter", true)
			if err != nil {
				logger.Error("failed to list tracked Twitter accounts", "error", err)
			} else if len(accounts) > 0 {
				logger.Debug("checking tracked Twitter accounts", "count", len(accounts))

				for _, account := range accounts {
					sources, err := twitterConnector.FetchAccountTweets(account)
					if err != nil {
						logger.Error("failed to fetch tweets",
							"account", account.AccountIdentifier,
							"error", err)
						continue
					}

					if len(sources) > 0 {
						logger.Info("fetched new tweets",
							"account", account.AccountIdentifier,
							"count", len(sources))

						// Store sources
						for _, source := range sources {
							if err := sourceRepo.Store(context.Background(), *source); err != nil {
								logger.Error("failed to store tweet source", "error", err)
							}
						}

						// Update last fetched ID
						latestID := ingestion.GetLatestTweetID(sources)
						if latestID != "" {
							trackedAccountRepo.UpdateLastFetched(account.ID, latestID, time.Now())
						}
					}
				}
			}

			// Wait for next tick
			<-ticker.C
		}
	}()

	// Start forecast scheduler
	logger.Info("starting forecast scheduler")
	forecastRepo := database.NewForecastRepository(db)
	forecastScheduler := scheduler.NewForecastScheduler(
		forecastRepo,
		forecaster.NewForecaster(eventRepo, forecastRepo, logger, inferenceLogger),
		logger,
	)
	go forecastScheduler.Start(context.Background())

	// Start summary scheduler
	logger.Info("starting summary scheduler")
	summaryRepo := database.NewSummaryRepository(db)
	// Get twitter poster for summary scheduler
	var summaryTwitterPoster api.TwitterPoster
	if twitterPoster != nil {
		if poster, ok := twitterPoster.(*social.TwitterPoster); ok {
			summaryTwitterPoster = poster
		}
	}
	summaryExecutor := api.NewSummaryExecutor(summaryRepo, eventRepo, forecastRepo, twitterRepo, summaryTwitterPoster, logger)
	summaryScheduler := scheduler.NewSummaryScheduler(summaryRepo, summaryExecutor, logger)
	go summaryScheduler.Start(context.Background())

	// Start strategy scheduler
	logger.Info("starting strategy scheduler")
	strategyRepo := database.NewStrategyRepository(db)
	strategistEngine := strategist.NewStrategist(eventRepo, strategyRepo, forecastRepo, logger, inferenceLogger)
	strategyScheduler := scheduler.NewStrategyScheduler(strategyRepo, strategistEngine, logger)
	go strategyScheduler.Start(context.Background())

	// Start background enrichment worker with database-level locking
	logger.Info("starting enrichment worker with database-level locking")

	go func() {
		// Run continuously with minimal delay between batches
		time.Sleep(5 * time.Second) // Initial delay

		for {
			enrichStart := time.Now()
			ctx := context.Background()

			// Atomically claim sources for enrichment (database-level locking)
			// This prevents race conditions across multiple Cloud Run instances
			// Stale claims (>15 min) are automatically reclaimed
			claimedSources, err := sourceRepo.ClaimSourcesForEnrichment(ctx, 1, 15*time.Minute)
			if err != nil {
				logger.Error("failed to claim sources for enrichment", "error", err)
				time.Sleep(5 * time.Second) // Brief pause on error
				continue
			}

			if len(claimedSources) == 0 {
				// No sources to process, pause before checking again
				logger.Debug("no sources available for enrichment, pausing")
				time.Sleep(10 * time.Second)
				continue
			}

			logger.Info("claimed sources for enrichment", "count", len(claimedSources))

			// Create a timeout context for the entire batch (10 minutes max)
			batchCtx, batchCancel := context.WithTimeout(ctx, 10*time.Minute)

			// Directly enrich the sources we claimed
			logger.Info("enriching claimed sources", "num_sources", len(claimedSources))
			events, enrichErr := enricher.EnrichBatch(batchCtx, claimedSources)
			logger.Info("enrichment batch returned", "num_events", len(events), "has_error", enrichErr != nil)

			var eventsPublished, eventsRejected, errorCount int

			// Track which sources successfully produced events
			successfulSourceIDs := make(map[string]bool)
			for _, event := range events {
				// Each event has a Sources field with the source(s) it came from
				for _, source := range event.Sources {
					successfulSourceIDs[source.ID] = true
				}
			}

			// Identify and log failures for individual sources
			for _, source := range claimedSources {
				if !successfulSourceIDs[source.ID] {
					// This source failed to produce an event
					errorCount++

					// Determine error message
					errorMsg := "enrichment failed"
					if enrichErr != nil {
						errorMsg = enrichErr.Error()
					}

					// Update source status as failed
					if err := sourceRepo.UpdateEnrichmentStatus(ctx, source.ID, models.EnrichmentStatusFailed, errorMsg); err != nil {
						logger.Error("failed to update enrichment status", "source_id", source.ID, "error", err)
					}

					// Log enrichment failure to ingestion_errors table
					ingestionErr := models.IngestionError{
						Platform:  "enrichment",
						ErrorType: string(models.ErrorTypeEnrichmentFailed),
						URL:       source.URL,
						ErrorMsg:  errorMsg,
						Metadata:  fmt.Sprintf(`{"source_id":"%s","title":"%s"}`, source.ID, source.Title),
						CreatedAt: time.Now(),
						Resolved:  false,
					}
					if err := errorRepo.Store(ctx, ingestionErr); err != nil {
						logger.Error("failed to log enrichment error", "source_id", source.ID, "error", err)
					} else {
						logger.Debug("logged enrichment error for source", "source_id", source.ID, "url", source.URL)
					}
				}
			}

			// If no events were created at all, skip to next iteration
			if len(events) == 0 {
				logger.Warn("no events created from batch", "source_count", len(claimedSources))
				batchCancel()
				continue
			}

			// CRITICAL: Mark successful sources as completed IMMEDIATELY after enrichment, before ProcessEvent
			// This prevents race conditions where another instance claims the same source
			// while this instance is still processing the event (which can be slow)
			for _, source := range claimedSources {
				// Only mark as completed if it successfully produced an event
				if successfulSourceIDs[source.ID] {
					if err := sourceRepo.UpdateEnrichmentStatus(ctx, source.ID, models.EnrichmentStatusCompleted, ""); err != nil {
						logger.Error("failed to mark source as enriched", "source_id", source.ID, "error", err)
					} else {
						logger.Debug("marked source as completed", "source_id", source.ID)
					}
				}
			}

			// Process each enriched event through the lifecycle manager
			for i := range events {
				event := &events[i]

				// Process the event (this handles correlation, thresholds, and storage)
				if err := eventManager.ProcessEvent(batchCtx, event); err != nil {
					logger.Error("event processing failed",
						"event_id", event.ID,
						"error", err)
					errorCount++
					continue
				}

				// Count by status
				switch event.Status {
				case models.EventStatusPublished:
					eventsPublished++
				case models.EventStatusRejected:
					eventsRejected++
				}
			}

			// Cancel context after all processing is complete
			batchCancel()

			// Log completion
			enrichDuration := int(time.Since(enrichStart).Milliseconds())
			logger.Info("enrichment batch complete",
				"sources_ingested", len(claimedSources),
				"events_enriched", len(events),
				"events_published", eventsPublished,
				"events_rejected", eventsRejected,
				"errors", errorCount,
				"duration_ms", enrichDuration)

			// Log enrichment activity
			sourcesIngested := len(claimedSources)
			activityLogRepo.Log(ctx, models.ActivityLog{
				ActivityType: models.ActivityTypeEnrichment,
				Message:      fmt.Sprintf("Enriched %d sources into %d events (%d published, %d rejected)", sourcesIngested, len(events), eventsPublished, eventsRejected),
				Details: map[string]interface{}{
					"sources_ingested": sourcesIngested,
					"events_enriched":  len(events),
					"events_published": eventsPublished,
					"events_rejected":  eventsRejected,
					"error_count":      errorCount,
				},
				SourceCount: &sourcesIngested,
				DurationMs:  &enrichDuration,
			})

			// No delay if we processed sources, continue immediately
		}
	}()

	// Scraper worker removed - no longer scraping articles

	// Wrap with SPA middleware to serve frontend for non-API routes
	logger.Info("setting up static file server for web UI")
	handler := server.SPAMiddleware(collector.InstrumentHandler(mux), "./web/dist", "./web/dist/index.html")

	// Start server
	srv := server.New(cfg.Server, logger, handler)

	go func() {
		logger.Info("starting server", "port", cfg.Server.Port)
		if err := srv.Start(); err != nil {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	logger.Info("OSINTMCP started successfully")
	logger.Info("API available", "url", fmt.Sprintf("http://localhost:%s", cfg.Server.Port))

	waitForSignal(logger)

	logger.Info("shutting down")
	if err := srv.Shutdown(context.Background()); err != nil {
		logger.Error("shutdown error", "error", err)
	}
	logger.Info("shutdown complete")
}

func waitForSignal(logger *slog.Logger) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	sig := <-c
	logger.Info("received signal", "signal", sig.String())
	signal.Stop(c)
	close(c)
}
