package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/STRATINT/stratint/internal/auth"
	"log/slog"
)

// AuthHandler handles authentication requests
type AuthHandler struct {
	config auth.Config
	logger *slog.Logger
}

// NewAuthHandler creates a new authentication handler
func NewAuthHandler(config auth.Config, logger *slog.Logger) *AuthHandler {
	return &AuthHandler{
		config: config,
		logger: logger,
	}
}

// LoginRequest represents a login request
type LoginRequest struct {
	Password string `json:"password"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

// Login handles POST /api/auth/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers for all responses
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	// Handle CORS preflight
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate password
	if req.Password != h.config.AdminPassword {
		h.logger.Warn("failed login attempt", "ip", r.RemoteAddr)
		// Use a generic error message to prevent username enumeration
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Generate JWT token
	token, err := auth.GenerateToken("admin", h.config.JWTSecret, h.config.TokenDuration)
	if err != nil {
		h.logger.Error("failed to generate token", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	h.logger.Info("successful login", "ip", r.RemoteAddr)

	// Return token
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := LoginResponse{
		Token:     token,
		ExpiresAt: time.Now().Add(h.config.TokenDuration),
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("failed to encode response", "error", err)
	}
}

// ValidateToken handles GET /api/auth/validate
func (h *AuthHandler) ValidateToken(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers for all responses
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	// Handle CORS preflight
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Token validation is handled by the middleware
	// If we reach here, the token is valid
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"valid":  true,
		"userID": "admin",
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("failed to encode response", "error", err)
	}
}
