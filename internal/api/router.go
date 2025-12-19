package api

import (
	"database/sql"
	"net/http"
	"strings"

	"github.com/STRATINT/stratint/internal/auth"
	"github.com/STRATINT/stratint/internal/database"
	"github.com/STRATINT/stratint/internal/enrichment"
	"github.com/STRATINT/stratint/internal/eventmanager"
	"github.com/STRATINT/stratint/internal/inference"
	"github.com/STRATINT/stratint/internal/ingestion"
	"github.com/STRATINT/stratint/internal/models"
	"github.com/STRATINT/stratint/internal/social"
	"github.com/STRATINT/stratint/internal/strategist"
	"log/slog"
)

// SetupRoutes configures all API routes
func SetupRoutes(mux *http.ServeMux, db *sql.DB, manager *eventmanager.EventLifecycleManager, sourceRepo ingestion.SourceRepository, eventRepo ingestion.EventRepository, trackedAccountRepo models.TrackedAccountRepository, errorRepo database.IngestionErrorRepository, thresholdRepo *database.ThresholdRepository, activityLogRepo *database.ActivityLogRepository, openaiConfigRepo *database.OpenAIConfigRepository, connectorConfigRepo *database.ConnectorConfigRepository, twitterRepo *database.TwitterRepository, twitterPoster eventmanager.TwitterPoster, credibilityCache *enrichment.CredibilityCache, enricher enrichment.Enricher, authConfig auth.Config, fredAPIKey string, logger *slog.Logger) {
	handler := NewHandler(manager, sourceRepo, trackedAccountRepo, logger)
	trackedAccountsHandler := NewTrackedAccountsHandler(trackedAccountRepo, sourceRepo, errorRepo, activityLogRepo, connectorConfigRepo, credibilityCache, enricher, logger)
	connectorConfigHandler := NewConnectorConfigHandlers(connectorConfigRepo, logger)
	thresholdHandler := NewThresholdHandlers(thresholdRepo, logger)
	errorHandler := NewIngestionErrorHandler(errorRepo, logger)
	activityHandler := NewActivityLogHandlers(activityLogRepo, logger)
	openaiConfigHandler := NewOpenAIConfigHandlers(openaiConfigRepo, logger)
	twitterConfigHandler := NewTwitterConfigHandlers(twitterRepo, logger)
	// Inject dependencies for Twitter posting
	if twitterPoster != nil {
		if poster, ok := twitterPoster.(*social.TwitterPoster); ok {
			twitterConfigHandler.SetTwitterPoster(poster)
		}
	}
	twitterConfigHandler.SetEventRepo(eventRepo)
	pipelineHandler := NewPipelineHandler(sourceRepo, eventRepo, db, logger)
	rssHandler := NewRSSHandler(manager, logger)
	authHandler := NewAuthHandler(authConfig, logger)
	adminHandler := NewAdminHandler(db, logger)

	// Initialize inference log components
	inferenceLogRepo := database.NewInferenceLogRepository(db)
	inferenceLogger := inference.NewLogger(inferenceLogRepo, logger)
	inferenceLogHandler := NewInferenceLogHandler(inferenceLogRepo, logger)

	forecastHandler := NewForecastHandler(db, eventRepo.(*database.PostgresEventRepository), logger, inferenceLogger)

	// Initialize strategy components
	strategyRepo := database.NewStrategyRepository(db)
	forecastRepo := database.NewForecastRepository(db)
	strategistEngine := strategist.NewStrategist(eventRepo.(*database.PostgresEventRepository), strategyRepo, forecastRepo, logger, inferenceLogger)
	strategyHandler := NewStrategyHandler(strategyRepo, strategistEngine, logger)

	// Initialize summary components
	summaryRepo := database.NewSummaryRepository(db)
	// Determine twitter poster for executor
	var twitterPosterForExecutor TwitterPoster
	if twitterPoster != nil {
		if poster, ok := twitterPoster.(*social.TwitterPoster); ok {
			twitterPosterForExecutor = poster
		}
	}
	summaryExecutor := NewSummaryExecutor(summaryRepo, eventRepo.(*database.PostgresEventRepository), forecastRepo, twitterRepo, twitterPosterForExecutor, logger)
	summaryHandler := NewSummaryHandler(summaryRepo, summaryExecutor, logger)

	optionsHandler := NewOptionsAnalysisHandler(logger)
	fredHandler := NewFREDHandler(logger, fredAPIKey)

	// Auth middleware
	authMiddleware := auth.AuthMiddleware(authConfig)

	// Authentication routes (public)
	mux.HandleFunc("/api/auth/login", authHandler.Login)
	mux.HandleFunc("/api/auth/validate", func(w http.ResponseWriter, r *http.Request) {
		authMiddleware(http.HandlerFunc(authHandler.ValidateToken)).ServeHTTP(w, r)
	})

	// Event routes (public for reading)
	mux.HandleFunc("/api/events", handler.GetEventsHandler)
	mux.HandleFunc("/api/events/", func(w http.ResponseWriter, r *http.Request) {
		// Handle POST /api/events/:id/post-to-twitter (requires auth)
		if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/post-to-twitter") {
			authMiddleware(http.HandlerFunc(twitterConfigHandler.PostEventToTwitter)).ServeHTTP(w, r)
			return
		}
		// Check if this is a status update request (requires auth)
		if strings.HasSuffix(r.URL.Path, "/status") && r.Method == http.MethodPut {
			authMiddleware(http.HandlerFunc(handler.UpdateEventStatusHandler)).ServeHTTP(w, r)
			return
		}
		// Otherwise handle as get by ID (public)
		handler.GetEventByIDHandler(w, r)
	})
	mux.HandleFunc("/api/stats", handler.GetStatsHandler)

	// Public forecast routes
	mux.HandleFunc("/api/forecasts", forecastHandler.ListPublicForecasts)
	mux.HandleFunc("/api/forecasts/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/history/daily") {
			forecastHandler.GetPublicForecastHistoryDaily(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/history/4h") {
			forecastHandler.GetPublicForecastHistory4Hour(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/history") {
			forecastHandler.GetPublicForecastHistory(w, r)
			return
		}
		http.Error(w, "Not found", http.StatusNotFound)
	})

	// Public strategy routes
	mux.HandleFunc("/api/strategies", strategyHandler.ListPublicStrategies)
	mux.HandleFunc("/api/strategies/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/latest") {
			strategyHandler.GetLatestStrategyResult(w, r)
			return
		}
		strategyHandler.GetPublicStrategy(w, r)
	})

	// Market analysis routes (public)
	mux.HandleFunc("/api/market/spy-risk-analysis", optionsHandler.HandleSPYRiskAnalysis)
	mux.HandleFunc("/api/market/ibit-risk-analysis", optionsHandler.HandleIBITRiskAnalysis)
	mux.HandleFunc("/api/market/gld-risk-analysis", optionsHandler.HandleGLDRiskAnalysis)
	mux.HandleFunc("/api/market/tlt-risk-analysis", optionsHandler.HandleTLTRiskAnalysis)
	mux.HandleFunc("/api/market/vnq-risk-analysis", optionsHandler.HandleVNQRiskAnalysis)
	mux.HandleFunc("/api/market/uso-risk-analysis", optionsHandler.HandleUSORiskAnalysis)

	// FRED economic data routes (public)
	mux.HandleFunc("/api/market/fred/", func(w http.ResponseWriter, r *http.Request) {
		// Check if this is a multi-series request (has ?series query param)
		if r.URL.Query().Get("series") != "" {
			fredHandler.HandleFREDMultiSeries(w, r)
			return
		}
		// Otherwise handle as single series
		fredHandler.HandleFREDSeries(w, r)
	})

	// Source management routes
	mux.HandleFunc("/api/sources", handler.HandleSources)
	mux.HandleFunc("/api/sources/", handler.HandleSourceByID)

	// Tracked accounts routes (admin only)
	mux.HandleFunc("/api/tracked-accounts", func(w http.ResponseWriter, r *http.Request) {
		// Handle CORS preflight
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.WriteHeader(http.StatusOK)
			return
		}

		// Require authentication for all methods
		authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				trackedAccountsHandler.ListTrackedAccounts(w, r)
			case http.MethodPost:
				trackedAccountsHandler.AddTrackedAccount(w, r)
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		})).ServeHTTP(w, r)
	})

	mux.HandleFunc("/api/tracked-accounts/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/tracked-accounts/" {
			http.NotFound(w, r)
			return
		}

		// Handle CORS preflight for subroutes
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.WriteHeader(http.StatusOK)
			return
		}

		// Require authentication for all subroutes
		authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Handle /api/tracked-accounts/:id/toggle
			if r.Method == http.MethodPost && len(r.URL.Path) > 7 && r.URL.Path[len(r.URL.Path)-7:] == "/toggle" {
				trackedAccountsHandler.ToggleTrackedAccount(w, r)
				return
			}

			// Handle /api/tracked-accounts/:id/fetch
			if r.Method == http.MethodPost && len(r.URL.Path) > 6 && r.URL.Path[len(r.URL.Path)-6:] == "/fetch" {
				trackedAccountsHandler.FetchNow(w, r)
				return
			}

			// Handle /api/tracked-accounts/:id
			switch r.Method {
			case http.MethodGet:
				trackedAccountsHandler.GetTrackedAccount(w, r)
			case http.MethodPut:
				trackedAccountsHandler.UpdateTrackedAccount(w, r)
			case http.MethodDelete:
				trackedAccountsHandler.DeleteTrackedAccount(w, r)
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		})).ServeHTTP(w, r)
	})

	// Connector configuration routes (admin only)
	mux.HandleFunc("/api/connectors", func(w http.ResponseWriter, r *http.Request) {
		// Handle CORS preflight
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.WriteHeader(http.StatusOK)
			return
		}

		// Require authentication
		authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet {
				connectorConfigHandler.ListConnectors(w, r)
			} else {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		})).ServeHTTP(w, r)
	})

	mux.HandleFunc("/api/connectors/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/connectors/" {
			http.NotFound(w, r)
			return
		}

		// Handle CORS preflight for all subroutes
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.WriteHeader(http.StatusOK)
			return
		}

		// Require authentication
		authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Handle /api/connectors/:id/toggle
			if r.Method == http.MethodPost && len(r.URL.Path) > 7 && r.URL.Path[len(r.URL.Path)-7:] == "/toggle" {
				connectorConfigHandler.ToggleConnector(w, r)
				return
			}

			// Handle /api/connectors/:id/config
			if r.Method == http.MethodGet && len(r.URL.Path) > 20 && r.URL.Path[len(r.URL.Path)-7:] == "/config" {
				connectorConfigHandler.GetConnectorConfig(w, r)
				return
			}
			if r.Method == http.MethodPost && len(r.URL.Path) > 20 && r.URL.Path[len(r.URL.Path)-7:] == "/config" {
				connectorConfigHandler.UpdateConnectorConfig(w, r)
				return
			}

			http.NotFound(w, r)
		})).ServeHTTP(w, r)
	})

	// Threshold configuration routes (admin only)
	mux.HandleFunc("/api/thresholds", func(w http.ResponseWriter, r *http.Request) {
		// Handle CORS preflight
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.WriteHeader(http.StatusOK)
			return
		}

		// Require authentication
		authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				thresholdHandler.GetThresholds(w, r)
			case http.MethodPost:
				thresholdHandler.UpdateThresholds(w, r)
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		})).ServeHTTP(w, r)
	})

	// Ingestion error routes (admin only)
	mux.HandleFunc("/api/ingestion-errors", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.WriteHeader(http.StatusOK)
			return
		}
		authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet {
				errorHandler.ListErrors(w, r)
			} else {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		})).ServeHTTP(w, r)
	})

	mux.HandleFunc("/api/ingestion-errors/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.WriteHeader(http.StatusOK)
			return
		}
		authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost && len(r.URL.Path) > 8 && r.URL.Path[len(r.URL.Path)-8:] == "/resolve" {
				errorHandler.ResolveError(w, r)
				return
			}
			if r.Method == http.MethodDelete {
				errorHandler.DeleteError(w, r)
				return
			}
			http.Error(w, "Not found", http.StatusNotFound)
		})).ServeHTTP(w, r)
	})

	// Activity log routes (admin only)
	mux.HandleFunc("/api/activity-logs", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.WriteHeader(http.StatusOK)
			return
		}
		authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet {
				activityHandler.ListActivities(w, r)
			} else {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		})).ServeHTTP(w, r)
	})

	// OpenAI configuration routes (admin only)
	mux.HandleFunc("/api/openai-config", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, PUT, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.WriteHeader(http.StatusOK)
			return
		}
		authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				openaiConfigHandler.GetOpenAIConfig(w, r)
			case http.MethodPut:
				openaiConfigHandler.UpdateOpenAIConfig(w, r)
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		})).ServeHTTP(w, r)
	})

	mux.HandleFunc("/api/openai-config/test", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.WriteHeader(http.StatusOK)
			return
		}
		authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost {
				openaiConfigHandler.TestOpenAIConfig(w, r)
			} else {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		})).ServeHTTP(w, r)
	})

	// Scraping functionality removed - now using RSS content only

	// Twitter configuration routes (admin only)
	mux.HandleFunc("/api/twitter-config", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, PUT, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.WriteHeader(http.StatusOK)
			return
		}
		authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				twitterConfigHandler.GetTwitterConfig(w, r)
			case http.MethodPut:
				twitterConfigHandler.UpdateTwitterConfig(w, r)
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		})).ServeHTTP(w, r)
	})

	// Posted tweets history route (admin only)
	mux.HandleFunc("/api/admin/posted-tweets", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.WriteHeader(http.StatusOK)
			return
		}
		authMiddleware(http.HandlerFunc(twitterConfigHandler.GetPostedTweets)).ServeHTTP(w, r)
	})

	// Delete all data route (admin only - DANGEROUS)
	mux.HandleFunc("/api/admin/delete-all", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.WriteHeader(http.StatusOK)
			return
		}
		authMiddleware(http.HandlerFunc(adminHandler.DeleteAllData)).ServeHTTP(w, r)
	})

	// Requeue failed enrichments route (admin only)
	mux.HandleFunc("/api/admin/requeue-enrichments", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.WriteHeader(http.StatusOK)
			return
		}
		authMiddleware(http.HandlerFunc(adminHandler.RequeueFailedEnrichments)).ServeHTTP(w, r)
	})

	// Delete failed enrichments route (admin only)
	mux.HandleFunc("/api/admin/delete-failed-enrichments", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.WriteHeader(http.StatusOK)
			return
		}
		authMiddleware(http.HandlerFunc(adminHandler.DeleteFailedEnrichments)).ServeHTTP(w, r)
	})

	// Delete pending sources route (admin only)
	mux.HandleFunc("/api/admin/delete-pending-sources", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.WriteHeader(http.StatusOK)
			return
		}
		authMiddleware(http.HandlerFunc(adminHandler.DeletePendingSources)).ServeHTTP(w, r)
	})

	// List Cloudflare debug HTML files (admin only)
	mux.HandleFunc("/api/admin/cloudflare-debug-files", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.WriteHeader(http.StatusOK)
			return
		}
		authMiddleware(http.HandlerFunc(adminHandler.ListCloudflareDebugFiles)).ServeHTTP(w, r)
	})

	// Download Cloudflare debug HTML file (admin only)
	mux.HandleFunc("/api/admin/cloudflare-debug-files/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.WriteHeader(http.StatusOK)
			return
		}
		authMiddleware(http.HandlerFunc(adminHandler.DownloadCloudflareDebugFile)).ServeHTTP(w, r)
	})

	// Source enrichment tracking (admin only)
	mux.HandleFunc("/api/admin/recent-enrichments", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.WriteHeader(http.StatusOK)
			return
		}
		authMiddleware(http.HandlerFunc(adminHandler.GetRecentEnrichments)).ServeHTTP(w, r)
	})

	// Forecast routes (admin only)
	mux.HandleFunc("/api/admin/forecasts", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.WriteHeader(http.StatusOK)
			return
		}
		authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				forecastHandler.ListForecasts(w, r)
			case http.MethodPost:
				forecastHandler.CreateForecast(w, r)
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		})).ServeHTTP(w, r)
	})

	mux.HandleFunc("/api/admin/forecasts/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.WriteHeader(http.StatusOK)
			return
		}
		authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Handle /api/admin/forecasts/runs/:runId
			if strings.HasPrefix(r.URL.Path, "/api/admin/forecasts/runs/") {
				if r.Method == http.MethodDelete {
					forecastHandler.DeleteForecastRun(w, r)
				} else {
					forecastHandler.GetForecastRun(w, r)
				}
				return
			}

			// Handle /api/admin/forecasts/:id/execute
			if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/execute") {
				forecastHandler.ExecuteForecast(w, r)
				return
			}

			// Handle /api/admin/forecasts/:id/schedule
			if r.Method == http.MethodPut && strings.HasSuffix(r.URL.Path, "/schedule") {
				forecastHandler.UpdateForecastSchedule(w, r)
				return
			}

			// Handle /api/admin/forecasts/:id/history/daily
			if r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/history/daily") {
				forecastHandler.GetForecastHistoryDaily(w, r)
				return
			}

			// Handle /api/admin/forecasts/:id/history/4h
			if r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/history/4h") {
				forecastHandler.GetForecastHistory4Hour(w, r)
				return
			}

			// Handle /api/admin/forecasts/:id/history
			if r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/history") {
				forecastHandler.GetForecastHistory(w, r)
				return
			}

			// Handle /api/admin/forecasts/:id/public (PUT)
			if r.Method == http.MethodPut && strings.HasSuffix(r.URL.Path, "/public") {
				forecastHandler.ToggleForecastPublic(w, r)
				return
			}

			// Handle /api/admin/forecasts/:id/display-order (PUT)
			if r.Method == http.MethodPut && strings.HasSuffix(r.URL.Path, "/display-order") {
				forecastHandler.UpdateForecastDisplayOrder(w, r)
				return
			}

			// Handle /api/admin/forecasts/:id/runs (DELETE - delete all runs)
			if r.Method == http.MethodDelete && strings.HasSuffix(r.URL.Path, "/runs") {
				forecastHandler.DeleteAllForecastRuns(w, r)
				return
			}

			// Handle /api/admin/forecasts/:id/runs (GET - list runs)
			if r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/runs") {
				forecastHandler.ListForecastRuns(w, r)
				return
			}

			// Handle /api/admin/forecasts/:id
			if r.Method == http.MethodGet {
				forecastHandler.GetForecast(w, r)
				return
			}

			// Handle /api/admin/forecasts/:id (PUT)
			if r.Method == http.MethodPut {
				forecastHandler.UpdateForecast(w, r)
				return
			}

			// Handle /api/admin/forecasts/:id (DELETE)
			if r.Method == http.MethodDelete {
				forecastHandler.DeleteForecast(w, r)
				return
			}

			http.Error(w, "Not found", http.StatusNotFound)
		})).ServeHTTP(w, r)
	})

	// Strategy routes (admin only)
	mux.HandleFunc("/api/admin/strategies", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.WriteHeader(http.StatusOK)
			return
		}
		authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				strategyHandler.ListStrategies(w, r)
			case http.MethodPost:
				strategyHandler.CreateStrategy(w, r)
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		})).ServeHTTP(w, r)
	})

	mux.HandleFunc("/api/admin/strategies/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.WriteHeader(http.StatusOK)
			return
		}
		authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Handle /api/admin/strategies/runs/:runId
			if strings.HasPrefix(r.URL.Path, "/api/admin/strategies/runs/") {
				strategyHandler.GetStrategyRun(w, r)
				return
			}

			// Handle /api/admin/strategies/:id/execute
			if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/execute") {
				strategyHandler.ExecuteStrategy(w, r)
				return
			}

			// Handle /api/admin/strategies/:id/schedule
			if r.Method == http.MethodPut && strings.HasSuffix(r.URL.Path, "/schedule") {
				strategyHandler.UpdateStrategySchedule(w, r)
				return
			}

			// Handle /api/admin/strategies/:id/publish
			if r.Method == http.MethodPut && strings.HasSuffix(r.URL.Path, "/publish") {
				strategyHandler.UpdateStrategyPublic(w, r)
				return
			}

			// Handle /api/admin/strategies/:id/order
			if r.Method == http.MethodPut && strings.HasSuffix(r.URL.Path, "/order") {
				strategyHandler.UpdateStrategyDisplayOrder(w, r)
				return
			}

			// Handle /api/admin/strategies/:id/runs (GET - list runs)
			if r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/runs") {
				strategyHandler.GetStrategyRuns(w, r)
				return
			}

			// Handle /api/admin/strategies/:id (GET)
			if r.Method == http.MethodGet {
				strategyHandler.GetStrategy(w, r)
				return
			}

			// Handle /api/admin/strategies/:id (PUT)
			if r.Method == http.MethodPut {
				strategyHandler.UpdateStrategy(w, r)
				return
			}

			// Handle /api/admin/strategies/:id (DELETE)
			if r.Method == http.MethodDelete {
				strategyHandler.DeleteStrategy(w, r)
				return
			}

			http.Error(w, "Not found", http.StatusNotFound)
		})).ServeHTTP(w, r)
	})

	// Summary routes (admin only)
	mux.HandleFunc("/api/admin/summaries", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.WriteHeader(http.StatusOK)
			return
		}
		authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				summaryHandler.List(w, r)
			case http.MethodPost:
				summaryHandler.Create(w, r)
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		})).ServeHTTP(w, r)
	})

	mux.HandleFunc("/api/admin/summaries/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.WriteHeader(http.StatusOK)
			return
		}
		authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Handle /api/admin/summaries/:id/execute
			if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/execute") {
				summaryHandler.Execute(w, r)
				return
			}

			// Handle /api/admin/summaries/:id/clone
			if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/clone") {
				summaryHandler.Clone(w, r)
				return
			}

			// Handle /api/admin/summaries/:id/latest
			if r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/latest") {
				summaryHandler.GetLatestRun(w, r)
				return
			}

			// Handle /api/admin/summaries/:id/runs
			if r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/runs") {
				summaryHandler.ListRuns(w, r)
				return
			}

			// Handle /api/admin/summaries/:id (GET)
			if r.Method == http.MethodGet {
				summaryHandler.Get(w, r)
				return
			}

			// Handle /api/admin/summaries/:id (PUT)
			if r.Method == http.MethodPut {
				summaryHandler.Update(w, r)
				return
			}

			// Handle /api/admin/summaries/:id (DELETE)
			if r.Method == http.MethodDelete {
				summaryHandler.Delete(w, r)
				return
			}

			http.Error(w, "Not found", http.StatusNotFound)
		})).ServeHTTP(w, r)
	})

	// Summary run detail route (admin only)
	mux.HandleFunc("/api/admin/summaries/runs/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.WriteHeader(http.StatusOK)
			return
		}
		authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Handle POST /api/admin/summaries/runs/:runId/tweet
			if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/tweet") {
				summaryHandler.PostToTwitter(w, r)
				return
			}
			// Handle GET /api/admin/summaries/runs/:runId
			summaryHandler.GetRun(w, r)
		})).ServeHTTP(w, r)
	})

	// Inference log routes (admin only)
	mux.HandleFunc("/api/admin/inference-logs", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.WriteHeader(http.StatusOK)
			return
		}
		authMiddleware(http.HandlerFunc(inferenceLogHandler.ListInferenceLogs)).ServeHTTP(w, r)
	})

	mux.HandleFunc("/api/admin/inference-logs/stats", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.WriteHeader(http.StatusOK)
			return
		}
		authMiddleware(http.HandlerFunc(inferenceLogHandler.GetInferenceStats)).ServeHTTP(w, r)
	})

	// Pipeline metrics routes (admin only)
	mux.HandleFunc("/api/pipeline/metrics", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.WriteHeader(http.StatusOK)
			return
		}
		authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet {
				pipelineHandler.GetPipelineMetricsHandler(w, r)
			} else {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		})).ServeHTTP(w, r)
	})

	// RSS feed route
	mux.HandleFunc("/api/feed.rss", rssHandler.GetRSSFeedHandler)

	// CORS preflight
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.WriteHeader(http.StatusOK)
			return
		}
		http.NotFound(w, r)
	})
}
