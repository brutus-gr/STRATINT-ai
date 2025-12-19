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
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}

	// Clear all tables
	queries := []string{
		"TRUNCATE TABLE sources CASCADE",
		"TRUNCATE TABLE events CASCADE",
		"TRUNCATE TABLE entities CASCADE",
	}

	for _, query := range queries {
		fmt.Printf("Executing: %s\n", query)
		if _, err := db.Exec(query); err != nil {
			log.Fatalf("failed to execute %s: %v", query, err)
		}
	}

	fmt.Println("✓ All sources, events, and entities cleared")
}
