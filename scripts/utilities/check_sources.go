//go:build ignore

package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL not set")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	// Check sources count
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM sources").Scan(&count)
	if err != nil {
		log.Fatalf("failed to count sources: %v", err)
	}
	fmt.Printf("Total sources: %d\n", count)

	// Check recent sources
	rows, err := db.Query(`
		SELECT id, url, type, published_at, created_at
		FROM sources
		ORDER BY created_at DESC
		LIMIT 5
	`)
	if err != nil {
		log.Fatalf("failed to query sources: %v", err)
	}
	defer rows.Close()

	fmt.Println("\nRecent sources:")
	for rows.Next() {
		var id, url, sourceType string
		var publishedAt, createdAt time.Time
		if err := rows.Scan(&id, &url, &sourceType, &publishedAt, &createdAt); err != nil {
			log.Printf("error scanning row: %v", err)
			continue
		}
		urlPreview := url
		if len(url) > 50 {
			urlPreview = url[:50] + "..."
		}
		fmt.Printf("- %s: %s (type: %s, created: %s, age: %s)\n",
			id[:12], urlPreview, sourceType, createdAt.Format("15:04:05"), time.Since(createdAt).Round(time.Second))
	}

	// Check events count
	err = db.QueryRow("SELECT COUNT(*) FROM events").Scan(&count)
	if err != nil {
		log.Fatalf("failed to count events: %v", err)
	}
	fmt.Printf("\nTotal events: %d\n", count)
}
