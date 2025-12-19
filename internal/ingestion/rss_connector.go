package ingestion

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/STRATINT/stratint/internal/database"
	"github.com/STRATINT/stratint/internal/models"
	"log/slog"
)

// RSSConnector fetches articles from RSS feeds.
type RSSConnector struct {
	feeds        []string
	logger       *slog.Logger
	errorRepo    database.IngestionErrorRepository
	activityRepo *database.ActivityLogRepository
}

// NewRSSConnector creates a new RSS connector.
func NewRSSConnector(feeds []string, logger *slog.Logger, errorRepo database.IngestionErrorRepository, activityRepo *database.ActivityLogRepository) (*RSSConnector, error) {
	// Filter out feeds containing /video/ or /videos/
	filteredFeeds := make([]string, 0, len(feeds))
	for _, feed := range feeds {
		if strings.Contains(feed, "/video/") || strings.Contains(feed, "/videos/") {
			logger.Debug("ignoring video feed", "url", feed)
			continue
		}
		filteredFeeds = append(filteredFeeds, feed)
	}

	return &RSSConnector{
		feeds:        filteredFeeds,
		logger:       logger,
		errorRepo:    errorRepo,
		activityRepo: activityRepo,
	}, nil
}

// Close shuts down the RSS connector.
func (c *RSSConnector) Close() error {
	return nil
}

// RSS represents the RSS 2.0 feed structure.
type RSS struct {
	XMLName xml.Name `xml:"rss"`
	Channel struct {
		Title       string    `xml:"title"`
		Description string    `xml:"description"`
		Items       []RSSItem `xml:"item"`
	} `xml:"channel"`
}

// RSSItem represents a single RSS 2.0 item.
type RSSItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
	GUID        string `xml:"guid"`
	Category    string `xml:"category"`
	RedditURL   string // Original Reddit discussion URL (only set for Reddit feeds)
}

// AtomFeed represents the Atom feed structure (used by Reddit and others).
type AtomFeed struct {
	XMLName xml.Name    `xml:"feed"`
	Title   string      `xml:"title"`
	Entries []AtomEntry `xml:"entry"`
}

// AtomEntry represents a single Atom entry.
type AtomEntry struct {
	Title     string      `xml:"title"`
	Link      AtomLink    `xml:"link"`
	Content   AtomContent `xml:"content"`
	Published string      `xml:"published"`
	Updated   string      `xml:"updated"`
	ID        string      `xml:"id"`
	Author    AtomAuthor  `xml:"author"`
}

// AtomLink represents an Atom link element.
type AtomLink struct {
	Href string `xml:"href,attr"`
}

// AtomContent represents Atom content.
type AtomContent struct {
	Type  string `xml:"type,attr"`
	Value string `xml:",chardata"`
}

// AtomAuthor represents an Atom author.
type AtomAuthor struct {
	Name string `xml:"name"`
	URI  string `xml:"uri"`
}

// Fetch retrieves articles from all configured RSS feeds.
func (c *RSSConnector) Fetch() ([]models.Source, error) {
	var allSources []models.Source

	for _, feedURL := range c.feeds {
		c.logger.Info("fetching rss feed", "url", feedURL)
		startTime := time.Now()

		sources, err := c.fetchFeed(feedURL)
		if err != nil {
			c.logger.Error("failed to fetch feed", "url", feedURL, "error", err)

			// Log error to database
			if c.errorRepo != nil {
				c.logError(context.Background(), "rss", string(models.ErrorTypeRSSFetchFailed), feedURL, err.Error(), nil)
			}
			continue
		}

		duration := int(time.Since(startTime).Milliseconds())
		c.logger.Info("fetched rss articles", "url", feedURL, "count", len(sources))

		// Log successful fetch activity
		if c.activityRepo != nil {
			sourceCount := len(sources)
			c.activityRepo.Log(context.Background(), models.ActivityLog{
				ActivityType: models.ActivityTypeRSSFetch,
				Platform:     "rss",
				Message:      fmt.Sprintf("Successfully fetched %d articles from RSS feed", len(sources)),
				Details: map[string]interface{}{
					"feed_url": feedURL,
				},
				SourceCount: &sourceCount,
				DurationMs:  &duration,
			})
		}

		allSources = append(allSources, sources...)
	}

	return allSources, nil
}

// fetchFeed fetches and parses a single RSS feed.
func (c *RSSConnector) fetchFeed(feedURL string) ([]models.Source, error) {
	body, err := c.fetchFeedWithHTTP(feedURL)
	if err != nil {
		return nil, err
	}

	// Try to parse as RSS 2.0 first
	var rss RSS
	var items []RSSItem

	rssErr := xml.Unmarshal(body, &rss)
	if rssErr == nil && len(rss.Channel.Items) > 0 {
		// Successfully parsed as RSS 2.0
		items = rss.Channel.Items
		c.logger.Debug("parsed feed as RSS 2.0", "url", feedURL, "items", len(items))
	} else {
		// Log RSS parsing result for debugging
		if rssErr != nil {
			c.logger.Debug("RSS parsing failed, trying Atom", "url", feedURL, "rss_error", rssErr.Error())
		} else {
			c.logger.Debug("RSS parsed but no items found, trying Atom", "url", feedURL, "items", len(rss.Channel.Items))
		}

		// Try parsing as Atom format (used by Reddit)
		var atom AtomFeed
		atomErr := xml.Unmarshal(body, &atom)
		if atomErr == nil && len(atom.Entries) > 0 {
			// Successfully parsed as Atom, convert to RSS items for unified processing
			isRedditFeed := strings.Contains(feedURL, "reddit.com")

			for _, entry := range atom.Entries {
				articleLink := entry.Link.Href
				redditURL := ""

				// For Reddit feeds, extract the actual article URL from the content
				if isRedditFeed {
					if extractedURL, err := extractArticleURLFromReddit(entry.Content.Value); err == nil {
						c.logger.Debug("extracted article URL from reddit post",
							"reddit_url", entry.Link.Href,
							"article_url", extractedURL)
						redditURL = entry.Link.Href // Store original Reddit URL
						articleLink = extractedURL  // Use actual article URL
					} else {
						// If we can't extract an article URL, use the Reddit URL
						c.logger.Debug("no external URL found, using Reddit URL",
							"title", entry.Title,
							"reddit_url", entry.Link.Href)
						articleLink = entry.Link.Href
					}
				}

				items = append(items, RSSItem{
					Title:       entry.Title,
					Link:        articleLink,
					Description: entry.Content.Value,
					PubDate:     entry.Published,
					GUID:        entry.ID,
					RedditURL:   redditURL,
				})
			}
			c.logger.Debug("parsed feed as Atom", "url", feedURL, "items", len(items))
		} else {
			// Both RSS and Atom parsing failed
			if atomErr != nil {
				c.logger.Error("failed to parse feed as RSS or Atom",
					"url", feedURL,
					"rss_error", rssErr,
					"atom_error", atomErr.Error())
				return nil, fmt.Errorf("failed to parse as RSS (error: %v) or Atom (error: %v)", rssErr, atomErr)
			} else {
				c.logger.Error("feed parsed but no items found", "url", feedURL, "rss_items", len(rss.Channel.Items), "atom_entries", len(atom.Entries))
				return nil, fmt.Errorf("feed parsed successfully but contains no items")
			}
		}
	}

	// Sort items by publish date (newest first)
	sort.Slice(items, func(i, j int) bool {
		timeI := parsePubDate(items[i].PubDate)
		timeJ := parsePubDate(items[j].PubDate)
		return timeI.After(timeJ) // Descending order (newest first)
	})
	c.logger.Debug("sorted feed items by date", "url", feedURL, "count", len(items))

	// Convert to Source models using RSS description as content
	var sources []models.Source

	for _, item := range items {
		// Clean and validate URL, use GUID as fallback if Link is empty
		cleanURL := strings.TrimSpace(item.Link)
		if cleanURL == "" && item.GUID != "" {
			cleanURL = strings.TrimSpace(item.GUID)
			c.logger.Debug("using GUID as URL", "guid", item.GUID, "title", item.Title)
		}

		if cleanURL == "" || (!strings.HasPrefix(cleanURL, "http://") && !strings.HasPrefix(cleanURL, "https://")) {
			c.logger.Warn("invalid or empty URL in RSS item, skipping", "url", item.Link, "guid", item.GUID, "title", item.Title)
			continue
		}

		// Skip video URLs
		if strings.Contains(cleanURL, "/video/") || strings.Contains(cleanURL, "/videos/") {
			c.logger.Debug("skipping video URL", "url", cleanURL, "title", item.Title)
			continue
		}

		// Skip root domain URLs (e.g., https://washingtonpost.com, https://example.com/)
		// Parse URL to extract path
		if strings.HasSuffix(cleanURL, "/") {
			cleanURL = strings.TrimSuffix(cleanURL, "/")
		}
		// Count slashes after protocol (http://example.com has 2 slashes, http://example.com/article has 3+)
		slashCount := strings.Count(cleanURL, "/")
		if slashCount <= 2 {
			c.logger.Warn("skipping root domain URL without article path", "url", cleanURL, "title", item.Title)
			continue
		}

		// Generate unique ID and parse publish date
		sourceID := fmt.Sprintf("rss-%d-%s", time.Now().UnixNano(), hashString(cleanURL))
		pubDate := parsePubDate(item.PubDate)

		// Use RSS description as content
		content := cleanText(item.Description)

		// Skip if content is too short to be meaningful (lowered threshold for RSS descriptions)
		if len(content) < 20 {
			c.logger.Debug("skipping item with insufficient content",
				"url", cleanURL,
				"title", item.Title,
				"content_length", len(content))
			continue
		}

		// Create source with RSS content (no scraping needed)
		source := models.Source{
			ID:               sourceID,
			Type:             models.SourceTypeNewsMedia,
			URL:              cleanURL,
			Title:            cleanText(item.Title),
			RawContent:       content, // Use RSS description as final content
			ContentHash:      hashString(cleanURL + item.Title + content),
			PublishedAt:      pubDate,
			RetrievedAt:      time.Now(),
			Credibility:      0.85, // Default credibility for RSS sources
			CreatedAt:        time.Now(),
			ScrapeStatus:     models.ScrapeStatusCompleted,   // Mark as completed since we're using RSS content directly
			EnrichmentStatus: models.EnrichmentStatusPending, // Ready for enrichment
			Metadata: models.SourceMetadata{
				FeedURL:   feedURL,
				RedditURL: item.RedditURL,
			},
		}

		sources = append(sources, source)
	}

	c.logger.Info("created sources from RSS feed", "url", feedURL, "count", len(sources))
	return sources, nil
}

// Name returns the connector name.
func (c *RSSConnector) Name() string {
	return "RSS"
}

// parsePubDate attempts to parse RSS pubDate and Atom date formats.
func parsePubDate(dateStr string) time.Time {
	if dateStr == "" {
		return time.Now()
	}

	// Common date formats (RSS and Atom)
	formats := []string{
		time.RFC3339,                // Atom format (2025-10-09T04:02:26+00:00)
		"2006-01-02T15:04:05Z07:00", // Alternative Atom format
		time.RFC1123Z,               // RSS format (Mon, 02 Jan 2006 15:04:05 -0700)
		time.RFC1123,
		time.RFC822Z,
		time.RFC822,
		"Mon, 02 Jan 2006 15:04:05 -0700",
		"Mon, 02 Jan 2006 15:04:05 MST",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t
		}
	}

	// Try investing.com format (no timezone specified, assume UTC)
	// Format: "2025-10-11 01:36:29"
	if t, err := time.ParseInLocation("2006-01-02 15:04:05", dateStr, time.UTC); err == nil {
		return t
	}

	// Fallback to now
	return time.Now()
}

// cleanText removes HTML tags and extra whitespace.
func cleanText(text string) string {
	// Basic HTML tag removal
	text = strings.ReplaceAll(text, "<p>", "\n")
	text = strings.ReplaceAll(text, "</p>", "\n")
	text = strings.ReplaceAll(text, "<br>", "\n")
	text = strings.ReplaceAll(text, "<br/>", "\n")
	text = strings.ReplaceAll(text, "<br />", "\n")

	// Remove all other HTML tags (simple approach)
	for {
		start := strings.Index(text, "<")
		if start == -1 {
			break
		}
		end := strings.Index(text[start:], ">")
		if end == -1 {
			break
		}
		text = text[:start] + text[start+end+1:]
	}

	// Clean whitespace
	text = strings.TrimSpace(text)
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	// Remove multiple newlines
	for strings.Contains(text, "\n\n\n") {
		text = strings.ReplaceAll(text, "\n\n\n", "\n\n")
	}

	return text
}

// hashString creates a simple hash of a string for ID generation.
func hashString(s string) string {
	hash := uint32(0)
	for _, c := range s {
		hash = hash*31 + uint32(c)
	}
	return fmt.Sprintf("%x", hash)
}

// fetchFeedWithHTTP fetches RSS feed using standard HTTP client.
func (c *RSSConnector) fetchFeedWithHTTP(feedURL string) ([]byte, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("GET", feedURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http get failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read body: %w", err)
	}

	return body, nil
}

// extractArticleURLFromReddit extracts the actual article URL from Reddit post content.
// Reddit posts contain HTML like: <span><a href="https://example.com/article">[link]</a></span>
// We want to extract the first external link (not reddit.com).
func extractArticleURLFromReddit(htmlContent string) (string, error) {
	// Match href attributes in anchor tags
	// Pattern: <a href="URL">...</a> where URL is not reddit.com
	re := regexp.MustCompile(`<a\s+href="([^"]+)"`)
	matches := re.FindAllStringSubmatch(htmlContent, -1)

	for _, match := range matches {
		if len(match) > 1 {
			url := match[1]
			// Skip Reddit URLs (comments, user pages, etc.)
			if !strings.Contains(url, "reddit.com") && !strings.Contains(url, "redd.it") {
				// This is likely the actual article URL
				return url, nil
			}
		}
	}

	return "", fmt.Errorf("no external article URL found in Reddit content")
}

// logError logs an ingestion error to the database.
func (c *RSSConnector) logError(ctx context.Context, platform, errorType, url, errorMsg string, metadataMap map[string]interface{}) {
	metadata, err := database.CreateErrorMetadata(metadataMap)
	if err != nil {
		c.logger.Error("failed to create error metadata", "error", err)
		metadata = ""
	}

	ingestionErr := models.IngestionError{
		Platform:  platform,
		ErrorType: errorType,
		URL:       url,
		ErrorMsg:  errorMsg,
		Metadata:  metadata,
		CreatedAt: time.Now(),
		Resolved:  false,
	}

	if err := c.errorRepo.Store(ctx, ingestionErr); err != nil {
		c.logger.Error("failed to log ingestion error", "error", err)
	}
}
