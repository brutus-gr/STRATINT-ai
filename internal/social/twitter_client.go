package social

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"log/slog"
)

// TwitterClient handles Twitter API v2 interactions
type TwitterClient struct {
	apiKey            string
	apiSecret         string
	accessToken       string
	accessTokenSecret string
	bearerToken       string
	httpClient        *http.Client
	logger            *slog.Logger
}

// NewTwitterClient creates a new Twitter API client
func NewTwitterClient(apiKey, apiSecret, accessToken, accessTokenSecret, bearerToken string, logger *slog.Logger) *TwitterClient {
	return &TwitterClient{
		apiKey:            apiKey,
		apiSecret:         apiSecret,
		accessToken:       accessToken,
		accessTokenSecret: accessTokenSecret,
		bearerToken:       bearerToken,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// TweetRequest represents the request to post a tweet
type TweetRequest struct {
	Text string `json:"text"`
}

// TweetResponse represents the response from Twitter API
type TweetResponse struct {
	Data struct {
		ID   string `json:"id"`
		Text string `json:"text"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"errors,omitempty"`
}

// PostTweet posts a tweet to Twitter using API v2 with OAuth 1.0a
func (c *TwitterClient) PostTweet(text string) (tweetID string, err error) {
	// Twitter API v2 endpoint
	apiURL := "https://api.twitter.com/2/tweets"

	// Create request body
	tweetReq := TweetRequest{
		Text: text,
	}

	bodyBytes, err := json.Marshal(tweetReq)
	if err != nil {
		return "", fmt.Errorf("failed to marshal tweet request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")

	// Generate OAuth 1.0a signature and authorization header
	authHeader, err := c.generateOAuthHeader("POST", apiURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate OAuth header: %w", err)
	}
	req.Header.Set("Authorization", authHeader)

	// Make request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to post tweet: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	bodyBytes, err = io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Parse response
	var tweetResp TweetResponse
	if err := json.Unmarshal(bodyBytes, &tweetResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for errors
	if resp.StatusCode != http.StatusCreated {
		if len(tweetResp.Errors) > 0 {
			return "", fmt.Errorf("twitter API error: %s", tweetResp.Errors[0].Message)
		}
		return "", fmt.Errorf("twitter API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	c.logger.Info("tweet posted successfully",
		"tweet_id", tweetResp.Data.ID,
		"text_length", len(text))

	return tweetResp.Data.ID, nil
}

// generateOAuthHeader generates OAuth 1.0a authorization header
func (c *TwitterClient) generateOAuthHeader(method, apiURL string, params map[string]string) (string, error) {
	// Generate nonce
	nonce := make([]byte, 32)
	rand.Read(nonce)
	nonceStr := base64.StdEncoding.EncodeToString(nonce)
	nonceStr = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return r
		}
		return -1
	}, nonceStr)

	// Generate timestamp
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	// OAuth parameters
	oauthParams := map[string]string{
		"oauth_consumer_key":     c.apiKey,
		"oauth_nonce":            nonceStr,
		"oauth_signature_method": "HMAC-SHA1",
		"oauth_timestamp":        timestamp,
		"oauth_token":            c.accessToken,
		"oauth_version":          "1.0",
	}

	// Combine OAuth params with request params
	allParams := make(map[string]string)
	for k, v := range oauthParams {
		allParams[k] = v
	}
	for k, v := range params {
		allParams[k] = v
	}

	// Create parameter string
	var paramPairs []string
	for k, v := range allParams {
		paramPairs = append(paramPairs, url.QueryEscape(k)+"="+url.QueryEscape(v))
	}
	sort.Strings(paramPairs)
	paramString := strings.Join(paramPairs, "&")

	// Create signature base string
	signatureBase := method + "&" + url.QueryEscape(apiURL) + "&" + url.QueryEscape(paramString)

	// Create signing key
	signingKey := url.QueryEscape(c.apiSecret) + "&" + url.QueryEscape(c.accessTokenSecret)

	// Generate signature
	mac := hmac.New(sha1.New, []byte(signingKey))
	mac.Write([]byte(signatureBase))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	// Add signature to OAuth params
	oauthParams["oauth_signature"] = signature

	// Build authorization header
	var authPairs []string
	for k, v := range oauthParams {
		authPairs = append(authPairs, url.QueryEscape(k)+"=\""+url.QueryEscape(v)+"\"")
	}
	sort.Strings(authPairs)

	return "OAuth " + strings.Join(authPairs, ", "), nil
}

// ValidateCredentials checks if the Twitter credentials are valid
func (c *TwitterClient) ValidateCredentials() error {
	// Use the /2/users/me endpoint to validate credentials
	url := "https://api.twitter.com/2/users/me"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.bearerToken))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to validate credentials: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("invalid credentials (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	c.logger.Info("twitter credentials validated successfully")
	return nil
}
