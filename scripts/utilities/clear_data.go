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
	// Get database URL from environment
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable not set")
	}

	// Connect to database
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	fmt.Println("🗑️  Clearing logs, events, and sources...")
	fmt.Println("✅ Keeping: tracked_accounts, openai_config, threshold_config, connector_config")
	fmt.Println()

	// Start transaction
	tx, err := db.Begin()
	if err != nil {
		log.Fatalf("Failed to start transaction: %v", err)
	}
	defer tx.Rollback()

	// Clear activity logs
	fmt.Print("Clearing activity_logs... ")
	result, err := tx.Exec("DELETE FROM activity_logs")
	if err != nil {
		log.Fatalf("Failed to clear activity_logs: %v", err)
	}
	activityCount, _ := result.RowsAffected()
	fmt.Printf("✅ %d rows deleted\n", activityCount)

	// Clear ingestion errors
	fmt.Print("Clearing ingestion_errors... ")
	result, err = tx.Exec("DELETE FROM ingestion_errors")
	if err != nil {
		log.Fatalf("Failed to clear ingestion_errors: %v", err)
	}
	errorCount, _ := result.RowsAffected()
	fmt.Printf("✅ %d rows deleted\n", errorCount)

	// Clear events (this will cascade to event_sources and event_entities)
	fmt.Print("Clearing events... ")
	result, err = tx.Exec("DELETE FROM events")
	if err != nil {
		log.Fatalf("Failed to clear events: %v", err)
	}
	eventCount, _ := result.RowsAffected()
	fmt.Printf("✅ %d rows deleted\n", eventCount)

	// Clear sources
	fmt.Print("Clearing sources... ")
	result, err = tx.Exec("DELETE FROM sources")
	if err != nil {
		log.Fatalf("Failed to clear sources: %v", err)
	}
	sourceCount, _ := result.RowsAffected()
	fmt.Printf("✅ %d rows deleted\n", sourceCount)

	// Clear entities
	fmt.Print("Clearing entities... ")
	result, err = tx.Exec("DELETE FROM entities")
	if err != nil {
		log.Fatalf("Failed to clear entities: %v", err)
	}
	entityCount, _ := result.RowsAffected()
	fmt.Printf("✅ %d rows deleted\n", entityCount)

	// Commit transaction
	if err := tx.Commit(); err != nil {
		log.Fatalf("Failed to commit transaction: %v", err)
	}

	fmt.Println()
	fmt.Println("✅ Database cleared successfully!")
	fmt.Println()
	fmt.Printf("Summary:\n")
	fmt.Printf("  - Activity logs: %d deleted\n", activityCount)
	fmt.Printf("  - Ingestion errors: %d deleted\n", errorCount)
	fmt.Printf("  - Events: %d deleted\n", eventCount)
	fmt.Printf("  - Sources: %d deleted\n", sourceCount)
	fmt.Printf("  - Entities: %d deleted\n", entityCount)
	fmt.Printf("  - Total: %d rows deleted\n", activityCount+errorCount+eventCount+sourceCount+entityCount)
	fmt.Println()
	fmt.Println("✅ Preserved: tracked_accounts, openai_config, threshold_config, connector_config")
}
