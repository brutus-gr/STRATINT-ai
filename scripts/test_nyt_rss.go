package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// RSS represents the RSS 2.0 feed structure.
type RSS struct {
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
}

func main() {
	feedURL := "https://rss.nytimes.com/services/xml/rss/nyt/World.xml"

	fmt.Printf("Testing RSS feed: %s\n\n", feedURL)

	// HTTP request with User-Agent
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("GET", feedURL, nil)
	if err != nil {
		fmt.Printf("ERROR creating request: %v\n", err)
		return
	}
	req.Header.Set("User-Agent", "OSINTMCP/1.0")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("ERROR fetching feed: %v\n", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("HTTP Status: %d\n", resp.StatusCode)
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("ERROR: unexpected status code: %d\n", resp.StatusCode)
		return
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("ERROR reading body: %v\n", err)
		return
	}

	fmt.Printf("Response body length: %d bytes\n\n", len(body))

	// Try to parse as RSS 2.0
	var rss RSS
	if err := xml.Unmarshal(body, &rss); err != nil {
		fmt.Printf("ERROR parsing as RSS 2.0: %v\n", err)
		return
	}

	fmt.Printf("✓ Successfully parsed as RSS 2.0\n")
	fmt.Printf("Channel Title: %s\n", rss.Channel.Title)
	fmt.Printf("Total Items: %d\n\n", len(rss.Channel.Items))

	if len(rss.Channel.Items) == 0 {
		fmt.Println("WARNING: No items found in feed!")
		return
	}

	// Count items after filtering
	validCount := 0
	skippedRootDomain := 0
	skippedInvalidURL := 0

	fmt.Println("Checking each item...")
	for i, item := range rss.Channel.Items {
		fmt.Printf("\n[Item %d]\n", i+1)
		fmt.Printf("  Title: %s\n", item.Title)
		fmt.Printf("  Link: %s\n", item.Link)
		fmt.Printf("  GUID: %s\n", item.GUID)

		// Use GUID if Link is empty (NYT sometimes has link in GUID)
		url := strings.TrimSpace(item.Link)
		if url == "" && item.GUID != "" {
			url = strings.TrimSpace(item.GUID)
			fmt.Printf("  → Using GUID as URL\n")
		}

		// Clean and validate URL
		cleanURL := url
		if cleanURL == "" || (!strings.HasPrefix(cleanURL, "http://") && !strings.HasPrefix(cleanURL, "https://")) {
			fmt.Printf("  ✗ SKIPPED: Invalid or empty URL\n")
			skippedInvalidURL++
			continue
		}

		// Skip root domain URLs
		if strings.HasSuffix(cleanURL, "/") {
			cleanURL = strings.TrimSuffix(cleanURL, "/")
		}
		slashCount := strings.Count(cleanURL, "/")
		fmt.Printf("  Slash count: %d\n", slashCount)

		if slashCount <= 2 {
			fmt.Printf("  ✗ SKIPPED: Root domain URL without article path\n")
			skippedRootDomain++
			continue
		}

		fmt.Printf("  ✓ VALID: Would be included\n")
		validCount++
	}

	fmt.Printf("\n=== SUMMARY ===\n")
	fmt.Printf("Total items in feed: %d\n", len(rss.Channel.Items))
	fmt.Printf("Valid items: %d\n", validCount)
	fmt.Printf("Skipped (root domain): %d\n", skippedRootDomain)
	fmt.Printf("Skipped (invalid URL): %d\n", skippedInvalidURL)
}
