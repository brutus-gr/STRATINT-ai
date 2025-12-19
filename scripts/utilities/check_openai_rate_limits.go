package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/STRATINT/stratint/internal/database"
	_ "github.com/lib/pq"
)

// This script checks the current OpenAI rate limit status by making a minimal API call
// and inspecting the response headers. Useful for checking when rate limits will reset.

func main() {
	// Get API key from database
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		connStr = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			getEnvOrDefault("DB_HOST", "localhost"),
			getEnvOrDefault("DB_PORT", "5432"),
			getEnvOrDefault("DB_USER", "osintmcp"),
			getEnvOrDefault("DB_PASSWORD", "osintmcp_dev_password"),
			getEnvOrDefault("DB_NAME", "osintmcp"))
	}

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	configRepo := database.NewOpenAIConfigRepository(db)
	ctx := context.Background()

	config, err := configRepo.Get(ctx)
	if err != nil {
		log.Fatalf("Failed to get OpenAI config: %v", err)
	}

	if config.APIKey == "" {
		log.Fatal("OpenAI API key not configured")
	}

	// Make a minimal API call to get rate limit headers
	fmt.Println("Checking OpenAI rate limit status...")
	fmt.Println("Model:", config.Model)
	fmt.Println()

	headers, err := checkRateLimits(config.APIKey, config.Model)
	if err != nil {
		log.Fatalf("Failed to check rate limits: %v", err)
	}

	fmt.Println("Rate Limit Status:")
	fmt.Println("==================")

	// Display request limits
	if val := headers["X-Ratelimit-Limit-Requests"]; val != "" {
		fmt.Printf("Request Limit: %s requests\n", val)
	}
	if val := headers["X-Ratelimit-Remaining-Requests"]; val != "" {
		fmt.Printf("Requests Remaining: %s\n", val)
	}
	if val := headers["X-Ratelimit-Reset-Requests"]; val != "" {
		fmt.Printf("Request Limit Resets In: %s\n", val)
	}

	fmt.Println()

	// Display token limits
	if val := headers["X-Ratelimit-Limit-Tokens"]; val != "" {
		fmt.Printf("Token Limit: %s tokens\n", val)
	}
	if val := headers["X-Ratelimit-Remaining-Tokens"]; val != "" {
		fmt.Printf("Tokens Remaining: %s\n", val)
	}
	if val := headers["X-Ratelimit-Reset-Tokens"]; val != "" {
		fmt.Printf("Token Limit Resets In: %s\n", val)
	}

	fmt.Println()
	fmt.Println("Note: Reset times are shown as duration strings (e.g., '1h30m' = 1 hour 30 minutes)")
}

func checkRateLimits(apiKey, model string) (map[string]string, error) {
	// Make a minimal completion request (single token)
	reqBody := map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{"role": "user", "content": "Hi"},
		},
		"max_tokens": 1,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Extract rate limit headers
	headers := make(map[string]string)

	// Capture all x-ratelimit headers (case-insensitive)
	for key, values := range resp.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}

	// Read body to check for errors
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return headers, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	return headers, nil
}

func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
