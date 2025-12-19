package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"log/slog"
)

// Cache entry for FRED API responses
type cacheEntry struct {
	data      interface{}
	expiresAt time.Time
}

// FREDHandler handles GET /api/market/fred/:series_id
type FREDHandler struct {
	logger *slog.Logger
	apiKey string
	cache  map[string]*cacheEntry
	mutex  sync.RWMutex
}

const cacheTTL = 1 * time.Hour

func NewFREDHandler(logger *slog.Logger, apiKey string) *FREDHandler {
	return &FREDHandler{
		logger: logger,
		apiKey: apiKey,
		cache:  make(map[string]*cacheEntry),
	}
}

// getCached retrieves a cached response if it exists and is not expired
func (h *FREDHandler) getCached(key string) (interface{}, bool) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	entry, exists := h.cache[key]
	if !exists {
		return nil, false
	}

	if time.Now().After(entry.expiresAt) {
		return nil, false
	}

	return entry.data, true
}

// setCached stores a response in the cache with TTL
func (h *FREDHandler) setCached(key string, data interface{}) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	h.cache[key] = &cacheEntry{
		data:      data,
		expiresAt: time.Now().Add(cacheTTL),
	}
}

// FREDSeriesResponse represents the cleaned-up FRED series data
type FREDSeriesResponse struct {
	Title       string             `json:"title"`
	SeriesID    string             `json:"series_id"`
	Units       string             `json:"units"`
	Frequency   string             `json:"frequency"`
	Description string             `json:"description,omitempty"`
	Data        []string           `json:"data"`
	Metadata    FREDSeriesMetadata `json:"metadata"`
}

type FREDSeriesMetadata struct {
	LastUpdated      string `json:"last_updated"`
	ObservationStart string `json:"observation_start"`
	ObservationEnd   string `json:"observation_end"`
	DataPoints       int    `json:"data_points"`
}

// FREDMultiSeriesResponse represents multiple series
type FREDMultiSeriesResponse struct {
	Series   map[string]FREDSeriesInfo `json:"series"`
	Metadata FREDMultiSeriesMetadata   `json:"metadata"`
}

type FREDSeriesInfo struct {
	Title string   `json:"title"`
	Units string   `json:"units"`
	Data  []string `json:"data"` // Format: "date: value"
}

type FREDMultiSeriesMetadata struct {
	SeriesCount int `json:"series_count"`
}

// Raw FRED API structures (for unmarshaling the noisy API response)
type fredAPISeriesResponse struct {
	Seriess []struct {
		ID               string `json:"id"`
		Title            string `json:"title"`
		Units            string `json:"units"`
		UnitsShort       string `json:"units_short"`
		Frequency        string `json:"frequency"`
		LastUpdated      string `json:"last_updated"`
		Notes            string `json:"notes"`
		ObservationStart string `json:"observation_start"`
		ObservationEnd   string `json:"observation_end"`
	} `json:"seriess"`
}

type fredAPIObservationsResponse struct {
	RealtimeStart    string `json:"realtime_start"`
	RealtimeEnd      string `json:"realtime_end"`
	ObservationStart string `json:"observation_start"`
	ObservationEnd   string `json:"observation_end"`
	Units            string `json:"units"`
	Count            int    `json:"count"`
	Observations     []struct {
		RealtimeStart string `json:"realtime_start"`
		RealtimeEnd   string `json:"realtime_end"`
		Date          string `json:"date"`
		Value         string `json:"value"`
	} `json:"observations"`
}

func (h *FREDHandler) HandleFREDMultiSeries(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get comma-separated series IDs from query parameter
	// Example: /api/market/fred?series=DFF,DGS10,DGS2
	seriesParam := r.URL.Query().Get("series")
	if seriesParam == "" {
		http.Error(w, "series parameter required (comma-separated series IDs)", http.StatusBadRequest)
		return
	}

	seriesIDs := strings.Split(seriesParam, ",")
	if len(seriesIDs) == 0 {
		http.Error(w, "At least one series ID required", http.StatusBadRequest)
		return
	}

	// Trim whitespace from series IDs
	for i := range seriesIDs {
		seriesIDs[i] = strings.TrimSpace(seriesIDs[i])
	}

	// Get observation start date from query parameter (default to 6 months ago)
	observationStart := r.URL.Query().Get("start")
	if observationStart == "" {
		observationStart = time.Now().AddDate(0, -6, 0).Format("2006-01-02")
	}

	// Check cache
	cacheKey := fmt.Sprintf("multi:%s:%s", seriesParam, observationStart)
	if cached, found := h.getCached(cacheKey); found {
		h.logger.Info("returning cached FRED multi-series data", "series_ids", seriesIDs, "cache_key", cacheKey)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("X-Cache", "HIT")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(cached)
		return
	}

	h.logger.Info("fetching multiple FRED series", "series_ids", seriesIDs)

	// Fetch all series in parallel
	type seriesResult struct {
		seriesID string
		metadata *seriesMetadata
		data     map[string]string // map[date]value for easy alignment
		err      error
	}

	results := make(chan seriesResult, len(seriesIDs))

	for _, seriesID := range seriesIDs {
		go func(sid string) {
			metadata, err := h.fetchSeriesMetadata(sid)
			if err != nil {
				results <- seriesResult{seriesID: sid, err: err}
				return
			}

			observations, err := h.fetchObservationsRaw(sid, observationStart)
			if err != nil {
				results <- seriesResult{seriesID: sid, err: err}
				return
			}

			// Convert to map for easy date alignment
			dataMap := make(map[string]string)
			for _, obs := range observations {
				dataMap[obs.Date] = obs.Value
			}

			results <- seriesResult{
				seriesID: sid,
				metadata: metadata,
				data:     dataMap,
			}
		}(seriesID)
	}

	// Collect results
	seriesData := make(map[string]seriesResult)
	var errors []string

	for i := 0; i < len(seriesIDs); i++ {
		result := <-results
		if result.err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", result.seriesID, result.err))
			h.logger.Error("failed to fetch series", "series_id", result.seriesID, "error", result.err)
		} else {
			seriesData[result.seriesID] = result
		}
	}

	if len(errors) > 0 {
		http.Error(w, fmt.Sprintf("Failed to fetch some series: %s", strings.Join(errors, "; ")), http.StatusServiceUnavailable)
		return
	}

	// Build series with "date: value" format, skipping missing values
	formattedSeries := make(map[string]FREDSeriesInfo)
	for seriesID, result := range seriesData {
		// Get all dates and sort them
		dates := make([]string, 0, len(result.data))
		for date := range result.data {
			dates = append(dates, date)
		}
		sort.Strings(dates)

		// Format as "date: value", skipping missing/invalid values
		dataPoints := make([]string, 0, len(dates))
		for _, date := range dates {
			if val, ok := result.data[date]; ok && val != "." {
				dataPoints = append(dataPoints, fmt.Sprintf("%s: %s", date, val))
			}
		}

		formattedSeries[seriesID] = FREDSeriesInfo{
			Title: result.metadata.Title,
			Units: result.metadata.Units,
			Data:  dataPoints,
		}
	}

	// Build response
	response := FREDMultiSeriesResponse{
		Series: formattedSeries,
		Metadata: FREDMultiSeriesMetadata{
			SeriesCount: len(seriesIDs),
		},
	}

	h.logger.Info("successfully fetched multiple FRED series",
		"series_count", len(seriesIDs))

	// Cache the response
	h.setCached(cacheKey, response)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("X-Cache", "MISS")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (h *FREDHandler) HandleFREDSeries(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract series_id from URL path
	// URL pattern: /api/market/fred/:series_id
	// Example: /api/market/fred/DFF
	path := r.URL.Path
	seriesID := path[len("/api/market/fred/"):]

	if seriesID == "" {
		http.Error(w, "Series ID required", http.StatusBadRequest)
		return
	}

	// Get observation start date from query parameter (default to 6 months ago)
	observationStart := r.URL.Query().Get("start")
	if observationStart == "" {
		observationStart = time.Now().AddDate(0, -6, 0).Format("2006-01-02")
	}

	// Check cache
	cacheKey := fmt.Sprintf("single:%s:%s", seriesID, observationStart)
	if cached, found := h.getCached(cacheKey); found {
		h.logger.Info("returning cached FRED series data", "series_id", seriesID, "cache_key", cacheKey)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("X-Cache", "HIT")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(cached)
		return
	}

	h.logger.Info("fetching FRED series data", "series_id", seriesID)

	// Fetch series metadata
	metadata, err := h.fetchSeriesMetadata(seriesID)
	if err != nil {
		h.logger.Error("failed to fetch series metadata", "error", err, "series_id", seriesID)
		http.Error(w, fmt.Sprintf("Failed to fetch series metadata: %v", err), http.StatusServiceUnavailable)
		return
	}

	// Fetch observations data
	observations, err := h.fetchObservations(seriesID, observationStart)
	if err != nil {
		h.logger.Error("failed to fetch observations", "error", err, "series_id", seriesID)
		http.Error(w, fmt.Sprintf("Failed to fetch observations: %v", err), http.StatusServiceUnavailable)
		return
	}

	// Build clean response
	response := FREDSeriesResponse{
		Title:       metadata.Title,
		SeriesID:    metadata.ID,
		Units:       metadata.Units,
		Frequency:   metadata.Frequency,
		Description: truncateDescription(metadata.Notes),
		Data:        observations,
		Metadata: FREDSeriesMetadata{
			LastUpdated:      metadata.LastUpdated,
			ObservationStart: metadata.ObservationStart,
			ObservationEnd:   metadata.ObservationEnd,
			DataPoints:       len(observations),
		},
	}

	h.logger.Info("successfully fetched and cleaned FRED data",
		"series_id", seriesID,
		"data_points", len(observations),
		"title", metadata.Title)

	// Cache the response
	h.setCached(cacheKey, response)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("X-Cache", "MISS")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

type seriesMetadata struct {
	ID               string
	Title            string
	Units            string
	Frequency        string
	Notes            string
	LastUpdated      string
	ObservationStart string
	ObservationEnd   string
}

func (h *FREDHandler) fetchSeriesMetadata(seriesID string) (*seriesMetadata, error) {
	url := fmt.Sprintf("https://api.stlouisfed.org/fred/series?series_id=%s&api_key=%s&file_type=json",
		seriesID, h.apiKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("FRED API returned status %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var apiResp fredAPISeriesResponse
	if err := json.Unmarshal(bodyBytes, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(apiResp.Seriess) == 0 {
		return nil, fmt.Errorf("series not found")
	}

	series := apiResp.Seriess[0]
	return &seriesMetadata{
		ID:               series.ID,
		Title:            series.Title,
		Units:            series.Units,
		Frequency:        series.Frequency,
		Notes:            series.Notes,
		LastUpdated:      series.LastUpdated,
		ObservationStart: series.ObservationStart,
		ObservationEnd:   series.ObservationEnd,
	}, nil
}

func (h *FREDHandler) fetchObservations(seriesID, observationStart string) ([]string, error) {
	url := fmt.Sprintf("https://api.stlouisfed.org/fred/series/observations?series_id=%s&api_key=%s&file_type=json&observation_start=%s",
		seriesID, h.apiKey, observationStart)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("FRED API returned status %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var apiResp fredAPIObservationsResponse
	if err := json.Unmarshal(bodyBytes, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Format as compact strings: "date: value"
	observations := make([]string, 0, len(apiResp.Observations))
	for _, obs := range apiResp.Observations {
		observations = append(observations, fmt.Sprintf("%s: %s", obs.Date, obs.Value))
	}

	return observations, nil
}

// fetchObservationsRaw fetches observations but returns raw observation structs for date alignment
func (h *FREDHandler) fetchObservationsRaw(seriesID, observationStart string) ([]struct{ Date, Value string }, error) {
	url := fmt.Sprintf("https://api.stlouisfed.org/fred/series/observations?series_id=%s&api_key=%s&file_type=json&observation_start=%s",
		seriesID, h.apiKey, observationStart)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("FRED API returned status %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var apiResp fredAPIObservationsResponse
	if err := json.Unmarshal(bodyBytes, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Return raw observation structs
	observations := make([]struct{ Date, Value string }, 0, len(apiResp.Observations))
	for _, obs := range apiResp.Observations {
		observations = append(observations, struct{ Date, Value string }{
			Date:  obs.Date,
			Value: obs.Value,
		})
	}

	return observations, nil
}

// truncateDescription truncates the notes to a reasonable length for description
func truncateDescription(notes string) string {
	if len(notes) == 0 {
		return ""
	}

	// Find first sentence or first 200 characters
	maxLen := 200
	if len(notes) <= maxLen {
		return notes
	}

	// Try to truncate at sentence end
	for i := 0; i < maxLen && i < len(notes); i++ {
		if notes[i] == '.' && i+1 < len(notes) && notes[i+1] == ' ' {
			return notes[:i+1]
		}
	}

	// Otherwise truncate at space near maxLen
	truncated := notes[:maxLen]
	lastSpace := len(truncated)
	for i := len(truncated) - 1; i >= 0; i-- {
		if truncated[i] == ' ' {
			lastSpace = i
			break
		}
	}

	if lastSpace > maxLen/2 {
		return truncated[:lastSpace] + "..."
	}

	return truncated + "..."
}
