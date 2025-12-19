package api

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/STRATINT/stratint/internal/models"
)

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidateOpenAIConfig validates OpenAI configuration
func ValidateOpenAIConfig(config *models.OpenAIConfig) error {
	if config.APIKey == "" {
		return ValidationError{Field: "api_key", Message: "API key is required"}
	}

	if len(config.APIKey) < 20 {
		return ValidationError{Field: "api_key", Message: "API key appears to be invalid (too short)"}
	}

	if !strings.HasPrefix(config.APIKey, "sk-") {
		return ValidationError{Field: "api_key", Message: "API key must start with 'sk-'"}
	}

	// Validate model - all models from https://platform.openai.com/docs/pricing
	validModels := []string{
		// GPT-4o models
		"gpt-4o",
		"gpt-4o-mini",
		"gpt-4o-2024-11-20",
		"gpt-4o-2024-08-06",
		"gpt-4o-2024-05-13",
		"gpt-4o-mini-2024-07-18",
		// GPT-4 Turbo models
		"gpt-4-turbo",
		"gpt-4-turbo-2024-04-09",
		"gpt-4-turbo-preview",
		"gpt-4-0125-preview",
		"gpt-4-1106-preview",
		// GPT-4 models
		"gpt-4",
		"gpt-4-0613",
		"gpt-4-0314",
		// GPT-3.5 Turbo models
		"gpt-3.5-turbo",
		"gpt-3.5-turbo-0125",
		"gpt-3.5-turbo-1106",
		"gpt-3.5-turbo-16k",
		// o1 models
		"o1-preview",
		"o1-preview-2024-09-12",
		"o1-mini",
		"o1-mini-2024-09-12",
		// o4 models
		"o4-mini",
		// gpt-5 models
		"gpt-5",
		"gpt-5-mini",
	}

	modelValid := false
	for _, validModel := range validModels {
		if config.Model == validModel {
			modelValid = true
			break
		}
	}

	if !modelValid {
		return ValidationError{Field: "model", Message: "Invalid model name"}
	}

	// Validate temperature (0.0 - 2.0)
	if config.Temperature < 0.0 || config.Temperature > 2.0 {
		return ValidationError{Field: "temperature", Message: "Temperature must be between 0.0 and 2.0"}
	}

	// Validate max tokens (1 - 128000)
	if config.MaxTokens < 1 || config.MaxTokens > 128000 {
		return ValidationError{Field: "max_tokens", Message: "Max tokens must be between 1 and 128000"}
	}

	// Validate timeout (1 - 300 seconds)
	if config.TimeoutSeconds < 1 || config.TimeoutSeconds > 300 {
		return ValidationError{Field: "timeout_seconds", Message: "Timeout must be between 1 and 300 seconds"}
	}

	return nil
}

// ValidateThresholdConfig validates threshold configuration
func ValidateThresholdConfig(config *models.ThresholdConfig) error {
	// Validate confidence (0.0 - 1.0)
	if config.MinConfidence < 0.0 || config.MinConfidence > 1.0 {
		return ValidationError{Field: "min_confidence", Message: "Confidence must be between 0.0 and 1.0"}
	}

	// Validate magnitude (0.0 - 10.0)
	if config.MinMagnitude < 0.0 || config.MinMagnitude > 10.0 {
		return ValidationError{Field: "min_magnitude", Message: "Magnitude must be between 0.0 and 10.0"}
	}

	// Validate max age hours (0 = disabled, or > 0)
	if config.MaxSourceAgeHours < 0 {
		return ValidationError{Field: "max_source_age_hours", Message: "Max age hours cannot be negative"}
	}

	return nil
}

// ValidateURL validates a URL string
func ValidateURL(urlStr string) error {
	if urlStr == "" {
		return ValidationError{Field: "url", Message: "URL is required"}
	}

	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return ValidationError{Field: "url", Message: "Invalid URL format"}
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return ValidationError{Field: "url", Message: "URL must use http or https scheme"}
	}

	if parsedURL.Host == "" {
		return ValidationError{Field: "url", Message: "URL must have a host"}
	}

	return nil
}

// ValidateTrackedAccount validates tracked account data
func ValidateTrackedAccount(platform, identifier string, fetchInterval int) error {
	if platform == "" {
		return ValidationError{Field: "platform", Message: "Platform is required"}
	}

	validPlatforms := []string{"twitter", "rss", "reddit"}
	platformValid := false
	for _, validPlatform := range validPlatforms {
		if platform == validPlatform {
			platformValid = true
			break
		}
	}

	if !platformValid {
		return ValidationError{Field: "platform", Message: "Invalid platform (must be twitter, rss, or reddit)"}
	}

	if identifier == "" {
		return ValidationError{Field: "account_identifier", Message: "Account identifier is required"}
	}

	// For RSS, validate URL
	if platform == "rss" {
		if err := ValidateURL(identifier); err != nil {
			return err
		}
	}

	// Validate fetch interval (1 minute to 1440 minutes/24 hours)
	if fetchInterval < 1 || fetchInterval > 1440 {
		return ValidationError{Field: "fetch_interval_minutes", Message: "Fetch interval must be between 1 and 1440 minutes"}
	}

	return nil
}

// Scraper configuration removed - using RSS content only

// ValidateTwitterConfig validates Twitter configuration
func ValidateTwitterConfig(config *models.TwitterConfigUpdate) error {
	// Validate magnitude threshold (0.0 - 10.0)
	if config.MinMagnitudeForTweet < 0.0 || config.MinMagnitudeForTweet > 10.0 {
		return ValidationError{Field: "min_magnitude_for_tweet", Message: "Magnitude threshold must be between 0.0 and 10.0"}
	}

	// Validate confidence threshold (0.0 - 1.0)
	if config.MinConfidenceForTweet < 0.0 || config.MinConfidenceForTweet > 1.0 {
		return ValidationError{Field: "min_confidence_for_tweet", Message: "Confidence threshold must be between 0.0 and 1.0"}
	}

	// If enabled, require API credentials
	if config.Enabled {
		if config.APIKey == "" {
			return ValidationError{Field: "api_key", Message: "API key is required when Twitter posting is enabled"}
		}
		if config.APISecret == "" {
			return ValidationError{Field: "api_secret", Message: "API secret is required when Twitter posting is enabled"}
		}
		if config.AccessToken == "" {
			return ValidationError{Field: "access_token", Message: "Access token is required when Twitter posting is enabled"}
		}
		if config.AccessTokenSecret == "" {
			return ValidationError{Field: "access_token_secret", Message: "Access token secret is required when Twitter posting is enabled"}
		}
	}

	// Validate prompt is not empty
	if config.TweetGenerationPrompt == "" {
		return ValidationError{Field: "tweet_generation_prompt", Message: "Tweet generation prompt is required"}
	}

	// Validate enabled_categories JSON
	if len(config.EnabledCategories) > 0 {
		var categories []string
		if err := json.Unmarshal(config.EnabledCategories, &categories); err != nil {
			return ValidationError{Field: "enabled_categories", Message: "Invalid JSON format for enabled categories"}
		}
	}

	return nil
}
