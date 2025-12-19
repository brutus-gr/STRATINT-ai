//go:build ignore

package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	_ "github.com/lib/pq"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL not set")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	// Find all Reddit RSS feeds
	rows, err := db.Query(`
		SELECT id, account_identifier, display_name
		FROM tracked_accounts
		WHERE platform = 'rss'
		AND account_identifier LIKE '%reddit.com%'
	`)
	if err != nil {
		log.Fatalf("Failed to query tracked accounts: %v", err)
	}
	defer rows.Close()

	var updates []struct {
		ID     string
		OldURL string
		NewURL string
		Name   string
	}

	for rows.Next() {
		var id, url, name string
		if err := rows.Scan(&id, &url, &name); err != nil {
			log.Printf("Failed to scan row: %v", err)
			continue
		}

		// Check if URL needs updating
		newURL := convertToNewSortedURL(url)
		if newURL != url {
			updates = append(updates, struct {
				ID     string
				OldURL string
				NewURL string
				Name   string
			}{id, url, newURL, name})
		}
	}

	if len(updates) == 0 {
		fmt.Println("✅ No Reddit RSS feeds need updating - all are already using proper format")
		return
	}

	fmt.Printf("Found %d Reddit RSS feeds that need updating:\n\n", len(updates))

	// Show what will be updated
	for i, update := range updates {
		fmt.Printf("%d. %s\n", i+1, update.Name)
		fmt.Printf("   Old: %s\n", update.OldURL)
		fmt.Printf("   New: %s\n\n", update.NewURL)
	}

	// Update each feed
	fmt.Println("Updating feeds...")
	for _, update := range updates {
		_, err := db.Exec(`
			UPDATE tracked_accounts
			SET account_identifier = $1
			WHERE id = $2
		`, update.NewURL, update.ID)
		if err != nil {
			log.Printf("❌ Failed to update %s: %v", update.Name, err)
		} else {
			fmt.Printf("✅ Updated: %s\n", update.Name)
		}
	}

	fmt.Printf("\n✅ Successfully updated %d Reddit RSS feeds to sort by newest\n", len(updates))
}

// convertToNewSortedURL converts Reddit RSS URLs to use /new/.rss?sort=new format
func convertToNewSortedURL(url string) string {
	// Already has the correct format
	if strings.Contains(url, "/new/.rss?sort=new") {
		return url
	}

	// Remove existing .rss extension and query params
	url = strings.TrimSuffix(url, ".rss")
	if idx := strings.Index(url, "?"); idx > -1 {
		url = url[:idx]
	}

	// Remove trailing slash
	url = strings.TrimSuffix(url, "/")

	// Check if it already has /new, /hot, /top, etc.
	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		lastPart := parts[len(parts)-1]
		if lastPart == "new" || lastPart == "hot" || lastPart == "top" || lastPart == "rising" {
			// Remove the sort path since we're replacing it
			url = strings.TrimSuffix(url, "/"+lastPart)
		}
	}

	// Add /new/.rss?sort=new
	return url + "/new/.rss?sort=new"
}
