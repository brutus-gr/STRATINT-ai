package api

import (
	"encoding/xml"
	"html"
	"net/http"
	"time"

	"github.com/STRATINT/stratint/internal/models"
	"log/slog"
)

// RSSHandler handles RSS feed generation.
type RSSHandler struct {
	eventManager EventQueryInterface
	logger       *slog.Logger
}

// EventQueryInterface defines the minimal interface needed for querying events.
type EventQueryInterface interface {
	GetEvents(query models.EventQuery) ([]models.Event, error)
}

// NewRSSHandler creates a new RSS handler.
func NewRSSHandler(eventManager EventQueryInterface, logger *slog.Logger) *RSSHandler {
	return &RSSHandler{
		eventManager: eventManager,
		logger:       logger,
	}
}

// RSS 2.0 feed structures
type RSS struct {
	XMLName xml.Name `xml:"rss"`
	Version string   `xml:"version,attr"`
	Channel *Channel `xml:"channel"`
}

type Channel struct {
	Title         string  `xml:"title"`
	Link          string  `xml:"link"`
	Description   string  `xml:"description"`
	Language      string  `xml:"language,omitempty"`
	LastBuildDate string  `xml:"lastBuildDate,omitempty"`
	Items         []*Item `xml:"item"`
}

type Item struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
	GUID        string `xml:"guid"`
	Category    string `xml:"category,omitempty"`
}

// GetRSSFeedHandler returns an RSS feed of the 20 most recent published events.
// GET /api/feed.rss
func (h *RSSHandler) GetRSSFeedHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Query for the 20 most recent published events
	published := models.EventStatusPublished
	query := models.EventQuery{
		Status: &published,
		Limit:  20,
		Page:   1,
	}

	events, err := h.eventManager.GetEvents(query)
	if err != nil {
		h.logger.Error("failed to get events for RSS feed", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Determine base URL from request
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	baseURL := scheme + "://" + r.Host

	// Build RSS feed
	feed := &RSS{
		Version: "2.0",
		Channel: &Channel{
			Title:       "OSINTMCP Intelligence Feed",
			Link:        baseURL,
			Description: "Real-time OSINT intelligence events from OSINTMCP",
			Language:    "en-us",
			Items:       make([]*Item, 0, len(events)),
		},
	}

	// Set last build date to now
	feed.Channel.LastBuildDate = time.Now().Format(time.RFC1123Z)

	// Convert events to RSS items
	for _, event := range events {
		item := &Item{
			Title:       event.Title,
			Link:        baseURL + "/api/events/" + event.ID,
			Description: html.EscapeString(event.Summary),
			PubDate:     event.Timestamp.Format(time.RFC1123Z),
			GUID:        event.ID,
			Category:    string(event.Category),
		}
		feed.Channel.Items = append(feed.Channel.Items, item)
	}

	// Set content type to RSS XML
	w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)

	// Write XML declaration and encode feed
	w.Write([]byte(xml.Header))
	encoder := xml.NewEncoder(w)
	encoder.Indent("", "  ")
	if err := encoder.Encode(feed); err != nil {
		h.logger.Error("failed to encode RSS feed", "error", err)
	}
}
