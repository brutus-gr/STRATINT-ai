package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// RunMigrations runs all pending SQL migrations from the migrations directory
func RunMigrations(db *sql.DB, migrationsDir string, logger *slog.Logger) error {
	logger.Info("checking for pending database migrations")

	// Create migrations tracking table if it doesn't exist
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMP NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create schema_migrations table: %w", err)
	}

	// Get list of applied migrations
	appliedMigrations := make(map[string]bool)
	rows, err := db.Query("SELECT version FROM schema_migrations")
	if err != nil {
		return fmt.Errorf("failed to query applied migrations: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return fmt.Errorf("failed to scan migration version: %w", err)
		}
		appliedMigrations[version] = true
	}

	// Get list of migration files
	files, err := filepath.Glob(filepath.Join(migrationsDir, "*.sql"))
	if err != nil {
		return fmt.Errorf("failed to list migration files: %w", err)
	}

	// Sort files to ensure they run in order
	sort.Strings(files)

	pendingCount := 0
	for _, file := range files {
		migrationName := filepath.Base(file)

		// Skip non-migration files
		if strings.HasPrefix(migrationName, "combined_") || strings.HasPrefix(migrationName, "apply-") {
			continue
		}

		// Skip if already applied
		if appliedMigrations[migrationName] {
			continue
		}

		pendingCount++
		logger.Info("applying migration", "file", migrationName)

		// Read migration file
		content, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", migrationName, err)
		}

		// Execute migration in a transaction
		ctx := context.Background()
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to begin transaction for %s: %w", migrationName, err)
		}

		// Execute the migration SQL
		if _, err := tx.Exec(string(content)); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to execute migration %s: %w", migrationName, err)
		}

		// Record migration as applied
		if _, err := tx.Exec("INSERT INTO schema_migrations (version) VALUES ($1)", migrationName); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to record migration %s: %w", migrationName, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration %s: %w", migrationName, err)
		}

		logger.Info("migration applied successfully", "file", migrationName)
	}

	if pendingCount == 0 {
		logger.Info("no pending migrations found")
	} else {
		logger.Info("migrations completed", "count", pendingCount)
	}

	return nil
}
