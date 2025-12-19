package server

import (
	"net/http"
	"os"
	"strings"
)

// SPAMiddleware wraps an http.Handler to serve a Single Page Application
// It checks if the request was handled by API routes, and if not, serves the SPA
func SPAMiddleware(next http.Handler, staticPath, indexPath string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip SPA for API, MCP endpoints, healthz, or metrics - let them pass through
		if strings.HasPrefix(r.URL.Path, "/api/") ||
			strings.HasPrefix(r.URL.Path, "/mcp/") ||
			r.URL.Path == "/healthz" ||
			r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}

		// For everything else, try to serve as static file, or fall back to index.html
		path := staticPath + r.URL.Path

		// Serve index.html for root and SPA routes
		if r.URL.Path == "/" || r.URL.Path == "/admin" {
			http.ServeFile(w, r, indexPath)
			return
		}

		// Check if static file exists
		if _, err := os.Stat(path); os.IsNotExist(err) {
			// File doesn't exist, serve index.html for React Router
			http.ServeFile(w, r, indexPath)
			return
		}

		// Serve the static file
		http.FileServer(http.Dir(staticPath)).ServeHTTP(w, r)
	})
}
