//go:build ignore

package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

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

	// Find all Reddit RSS feeds
	rows, err := db.Query(`
		SELECT display_name, account_identifier, enabled
		FROM tracked_accounts
		WHERE platform = 'rss'
		AND account_identifier LIKE '%reddit.com%'
		ORDER BY display_name
	`)
	if err != nil {
		log.Fatalf("Failed to query tracked accounts: %v", err)
	}
	defer rows.Close()

	fmt.Println("Current Reddit RSS Feeds:")
	fmt.Println("=" + string(make([]byte, 80)))

	count := 0
	for rows.Next() {
		var name, url string
		var enabled bool
		if err := rows.Scan(&name, &url, &enabled); err != nil {
			log.Printf("Failed to scan row: %v", err)
			continue
		}

		count++
		status := "✅"
		if !enabled {
			status = "⏸️"
		}

		fmt.Printf("\n%s %s\n", status, name)
		fmt.Printf("   URL: %s\n", url)
		fmt.Printf("   Enabled: %v\n", enabled)
	}

	fmt.Printf("\n" + string(make([]byte, 80)) + "\n")
	fmt.Printf("Total: %d Reddit RSS feeds\n", count)
}
