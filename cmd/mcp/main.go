package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/STRATINT/stratint/internal/cloudsql"
	"github.com/STRATINT/stratint/internal/config"
	"github.com/STRATINT/stratint/internal/database"
	"github.com/STRATINT/stratint/internal/enrichment"
	"github.com/STRATINT/stratint/internal/eventmanager"
	"github.com/STRATINT/stratint/internal/inference"
	"github.com/STRATINT/stratint/internal/logging"
	"github.com/STRATINT/stratint/internal/models"
	_ "github.com/lib/pq"
	"log/slog"
)

// MCPServer implements the Model Context Protocol HTTP/SSE server
type MCPServer struct {
	mcpHandler *eventmanager.MCPHandler
	logger     *slog.Logger
}

// MCPRequest represents an MCP protocol request
type MCPRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// MCPResponse represents an MCP protocol response
type MCPResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *MCPError   `json:"error,omitempty"`
}

// MCPError represents an MCP error
type MCPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ToolDefinition represents an MCP tool definition
type ToolDefinition struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"inputSchema"`
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("failed to load config:", err)
	}

	logger, err := logging.New(cfg.Logging)
	if err != nil {
		log.Fatal("failed to init logger:", err)
	}

	logger.Info("starting OSINTMCP MCP server")

	// Connect to database
	dbURL, err := cloudsql.BuildDatabaseURL()
	if err != nil {
		logger.Error("failed to build database URL", "error", err)
		os.Exit(1)
	}

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

	// Create repositories
	sourceRepo := database.NewPostgresSourceRepository(db)
	eventRepo := database.NewPostgresEventRepository(db)
	thresholdRepo := database.NewThresholdRepository(db)
	openaiConfigRepo := database.NewOpenAIConfigRepository(db)
	activityRepo := database.NewActivityLogRepository(db)
	inferenceLogRepo := database.NewInferenceLogRepository(db)

	// Create inference logger
	inferenceLogger := inference.NewLogger(inferenceLogRepo, logger)

	// Create enricher
	var enricher enrichment.Enricher
	openaiEnricher, err := enrichment.NewOpenAIClientFromDB(context.Background(), openaiConfigRepo, logger, inferenceLogger)
	if err != nil {
		logger.Warn("failed to initialize OpenAI enricher, using mock", "error", err)
		enricher = enrichment.NewMockEnricher()
	} else {
		enricher = openaiEnricher
	}

	// Create event manager
	lifecycleConfig := eventmanager.DefaultLifecycleConfig()
	eventManager := eventmanager.NewEventLifecycleManager(
		sourceRepo,
		eventRepo,
		enricher,
		thresholdRepo,
		nil, // Twitter poster not used in MCP mode
		activityRepo,
		logger,
		lifecycleConfig,
	)

	// Create MCP handler
	mcpHandler := eventmanager.NewMCPHandler(eventManager)

	// Create MCP server
	server := &MCPServer{
		mcpHandler: mcpHandler,
		logger:     logger,
	}

	// Setup HTTP routes
	mux := http.NewServeMux()

	// MCP protocol endpoint at /mcp
	mux.HandleFunc("/mcp", server.HandleMCPRequest)

	// Root endpoint proxies to /mcp for convenience
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			// Proxy root to /mcp handler
			server.HandleMCPRequest(w, r)
		} else {
			http.NotFound(w, r)
		}
	})

	// Health check
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	logger.Info("MCP server listening", "port", port)
	if err := http.ListenAndServe(":"+port, enableCORS(mux)); err != nil {
		logger.Error("server failed", "error", err)
		os.Exit(1)
	}
}

// HandleMCPRequest handles MCP JSON-RPC requests
func (s *MCPServer) HandleMCPRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req MCPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, nil, -32700, "Parse error: "+err.Error())
		return
	}

	s.logger.Info("MCP request received", "method", req.Method, "id", req.ID)

	switch req.Method {
	case "initialize":
		s.handleInitialize(w, req)
	case "tools/list":
		s.handleToolsList(w, req)
	case "tools/call":
		s.handleToolCall(w, req)
	default:
		s.sendError(w, req.ID, -32601, "Method not found: "+req.Method)
	}
}

// handleInitialize handles MCP initialize request
func (s *MCPServer) handleInitialize(w http.ResponseWriter, req MCPRequest) {
	result := map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]interface{}{
			"tools": map[string]interface{}{},
		},
		"serverInfo": map[string]interface{}{
			"name":    "osintmcp",
			"version": "1.0.0",
		},
	}

	s.sendResult(w, req.ID, result)
}

// handleToolsList returns available MCP tools
func (s *MCPServer) handleToolsList(w http.ResponseWriter, req MCPRequest) {
	tools := []ToolDefinition{
		{
			Name:        "get_events",
			Description: "Query OSINT events with comprehensive filtering options including search, time range, magnitude/confidence thresholds, categories, source types, tags, entity types, and pagination.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"search_query": map[string]interface{}{
						"type":        "string",
						"description": "Full-text search across event title and summary",
					},
					"since_timestamp": map[string]interface{}{
						"type":        "string",
						"format":      "date-time",
						"description": "Start of time range (RFC3339 format)",
					},
					"until_timestamp": map[string]interface{}{
						"type":        "string",
						"format":      "date-time",
						"description": "End of time range (RFC3339 format)",
					},
					"min_magnitude": map[string]interface{}{
						"type":        "number",
						"minimum":     0,
						"maximum":     10,
						"description": "Minimum event magnitude (0-10 scale)",
					},
					"min_confidence": map[string]interface{}{
						"type":        "number",
						"minimum":     0,
						"maximum":     1,
						"description": "Minimum confidence score (0-1 scale)",
					},
					"categories": map[string]interface{}{
						"type": "array",
						"items": map[string]interface{}{
							"type": "string",
							"enum": []string{"geopolitics", "military", "economic", "cyber", "disaster", "terrorism", "diplomacy", "intelligence", "humanitarian", "other"},
						},
						"description": "Filter by event categories",
					},
					"source_types": map[string]interface{}{
						"type": "array",
						"items": map[string]interface{}{
							"type": "string",
							"enum": []string{"twitter", "telegram", "reddit", "4chan", "glp", "government", "news_media", "blog", "other"},
						},
						"description": "Filter by source types",
					},
					"tags": map[string]interface{}{
						"type":        "array",
						"items":       map[string]interface{}{"type": "string"},
						"description": "Filter by event tags",
					},
					"status": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"pending", "enriched", "published", "archived", "rejected"},
						"description": "Filter by event status (default: published)",
					},
					"page": map[string]interface{}{
						"type":        "integer",
						"minimum":     1,
						"default":     1,
						"description": "Page number for pagination (1-indexed)",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"minimum":     1,
						"maximum":     200,
						"default":     20,
						"description": "Number of results per page",
					},
					"sort_by": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"timestamp", "magnitude", "confidence", "created_at", "updated_at"},
						"default":     "timestamp",
						"description": "Field to sort results by",
					},
					"sort_order": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"asc", "desc"},
						"default":     "desc",
						"description": "Sort order (ascending or descending)",
					},
				},
			},
		},
	}

	result := map[string]interface{}{
		"tools": tools,
	}

	s.sendResult(w, req.ID, result)
}

// handleToolCall handles MCP tool execution
func (s *MCPServer) handleToolCall(w http.ResponseWriter, req MCPRequest) {
	var params struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		s.sendError(w, req.ID, -32602, "Invalid params: "+err.Error())
		return
	}

	if params.Name != "get_events" {
		s.sendError(w, req.ID, -32601, "Unknown tool: "+params.Name)
		return
	}

	// Parse query arguments
	var queryArgs map[string]interface{}
	if err := json.Unmarshal(params.Arguments, &queryArgs); err != nil {
		s.sendError(w, req.ID, -32602, "Invalid arguments: "+err.Error())
		return
	}

	// Convert to EventQuery
	query, err := s.parseEventQuery(queryArgs)
	if err != nil {
		s.sendError(w, req.ID, -32602, "Invalid query: "+err.Error())
		return
	}

	// Convert query to JSON for MCP handler
	queryJSON, err := json.Marshal(query)
	if err != nil {
		s.sendError(w, req.ID, -32603, "Internal error: "+err.Error())
		return
	}

	// Call MCP handler
	ctx := context.Background()
	resultJSON, err := s.mcpHandler.GetEvents(ctx, string(queryJSON))
	if err != nil {
		s.sendError(w, req.ID, -32603, "Query failed: "+err.Error())
		return
	}

	// Parse result
	var result interface{}
	if err := json.Unmarshal([]byte(resultJSON), &result); err != nil {
		s.sendError(w, req.ID, -32603, "Invalid response: "+err.Error())
		return
	}

	// Return tool result in MCP format
	toolResult := map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": resultJSON,
			},
		},
	}

	s.sendResult(w, req.ID, toolResult)
}

// parseEventQuery converts map to EventQuery struct
func (s *MCPServer) parseEventQuery(args map[string]interface{}) (*models.EventQuery, error) {
	query := &models.EventQuery{
		Page:  1,
		Limit: 20,
	}

	if searchQuery, ok := args["search_query"].(string); ok {
		query.SearchQuery = searchQuery
	}

	if since, ok := args["since_timestamp"].(string); ok {
		t, err := time.Parse(time.RFC3339, since)
		if err != nil {
			return nil, fmt.Errorf("invalid since_timestamp: %w", err)
		}
		query.SinceTimestamp = &t
	}

	if until, ok := args["until_timestamp"].(string); ok {
		t, err := time.Parse(time.RFC3339, until)
		if err != nil {
			return nil, fmt.Errorf("invalid until_timestamp: %w", err)
		}
		query.UntilTimestamp = &t
	}

	if minMag, ok := args["min_magnitude"].(float64); ok {
		query.MinMagnitude = &minMag
	}

	if minConf, ok := args["min_confidence"].(float64); ok {
		query.MinConfidence = &minConf
	}

	if page, ok := args["page"].(float64); ok {
		query.Page = int(page)
	}

	if limit, ok := args["limit"].(float64); ok {
		query.Limit = int(limit)
	}

	if sortBy, ok := args["sort_by"].(string); ok {
		query.SortBy = models.EventSortField(sortBy)
	}

	if sortOrder, ok := args["sort_order"].(string); ok {
		query.SortOrder = models.SortOrder(sortOrder)
	}

	if status, ok := args["status"].(string); ok {
		s := models.EventStatus(status)
		query.Status = &s
	}

	// Handle array fields
	if categories, ok := args["categories"].([]interface{}); ok {
		for _, cat := range categories {
			if catStr, ok := cat.(string); ok {
				query.Categories = append(query.Categories, models.Category(catStr))
			}
		}
	}

	if sourceTypes, ok := args["source_types"].([]interface{}); ok {
		for _, st := range sourceTypes {
			if stStr, ok := st.(string); ok {
				query.SourceTypes = append(query.SourceTypes, models.SourceType(stStr))
			}
		}
	}

	if tags, ok := args["tags"].([]interface{}); ok {
		for _, tag := range tags {
			if tagStr, ok := tag.(string); ok {
				query.Tags = append(query.Tags, tagStr)
			}
		}
	}

	return query, nil
}

// sendResult sends a successful MCP response
func (s *MCPServer) sendResult(w http.ResponseWriter, id interface{}, result interface{}) {
	resp := MCPResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// sendError sends an MCP error response
func (s *MCPServer) sendError(w http.ResponseWriter, id interface{}, code int, message string) {
	resp := MCPResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &MCPError{
			Code:    code,
			Message: message,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK) // MCP uses 200 even for errors
	json.NewEncoder(w).Encode(resp)
}

// enableCORS adds CORS headers
func enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
