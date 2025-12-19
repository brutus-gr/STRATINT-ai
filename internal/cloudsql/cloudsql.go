package cloudsql

import (
	"fmt"
	"os"
	"strings"
)

// BuildDatabaseURL constructs a PostgreSQL connection string that works with both
// local development and Google Cloud SQL on Cloud Run.
//
// For Cloud Run with Cloud SQL:
// - Set INSTANCE_CONNECTION_NAME to your Cloud SQL instance (e.g., project:region:instance)
// - Set DB_USER, DB_PASSWORD, DB_NAME
// - The function will automatically use Unix socket connection
//
// For local development:
// - Set DATABASE_URL directly (e.g., postgresql://user:pass@localhost:5432/dbname)
func BuildDatabaseURL() (string, error) {
	// Check if DATABASE_URL is already set (local development)
	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		return dbURL, nil
	}

	// Cloud Run / Cloud SQL configuration
	instanceConnectionName := os.Getenv("INSTANCE_CONNECTION_NAME")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")

	// Check if we have Cloud SQL credentials
	if instanceConnectionName == "" {
		return "", fmt.Errorf("neither DATABASE_URL nor INSTANCE_CONNECTION_NAME is set")
	}

	if dbUser == "" || dbName == "" {
		return "", fmt.Errorf("DB_USER and DB_NAME must be set when using INSTANCE_CONNECTION_NAME")
	}

	// Build Unix socket connection string for Cloud SQL
	// Cloud Run mounts Cloud SQL instances at /cloudsql/[INSTANCE_CONNECTION_NAME]
	socketPath := fmt.Sprintf("/cloudsql/%s", instanceConnectionName)

	// Build connection string
	var connStr string
	if dbPassword != "" {
		connStr = fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=disable",
			socketPath, dbUser, dbPassword, dbName)
	} else {
		// For IAM authentication (no password needed)
		connStr = fmt.Sprintf("host=%s user=%s dbname=%s sslmode=disable",
			socketPath, dbUser, dbName)
	}

	return connStr, nil
}

// GetConnectionConfig returns connection configuration details for logging/debugging
func GetConnectionConfig() map[string]string {
	config := make(map[string]string)

	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		// Redact password from DATABASE_URL for logging
		redactedURL := redactPassword(dbURL)
		config["connection_type"] = "direct"
		config["database_url"] = redactedURL
	} else if instanceConnectionName := os.Getenv("INSTANCE_CONNECTION_NAME"); instanceConnectionName != "" {
		config["connection_type"] = "cloud_sql"
		config["instance"] = instanceConnectionName
		config["user"] = os.Getenv("DB_USER")
		config["database"] = os.Getenv("DB_NAME")
		config["socket_path"] = fmt.Sprintf("/cloudsql/%s", instanceConnectionName)
	} else {
		config["connection_type"] = "none"
		config["error"] = "no database configuration found"
	}

	return config
}

// redactPassword removes password from connection string for safe logging
func redactPassword(connStr string) string {
	// Handle postgresql:// URLs
	if strings.HasPrefix(connStr, "postgresql://") || strings.HasPrefix(connStr, "postgres://") {
		parts := strings.SplitN(connStr, "@", 2)
		if len(parts) == 2 {
			userParts := strings.Split(parts[0], ":")
			if len(userParts) >= 3 {
				return userParts[0] + "://" + userParts[1] + ":***@" + parts[1]
			}
		}
	}
	return connStr
}
