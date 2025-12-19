package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/STRATINT/stratint/internal/config"
	"log/slog"
)

// Server represents the HTTP server hosting the MCP endpoints.
type Server struct {
	cfg    config.ServerConfig
	logger *slog.Logger
	http   *http.Server
}

// New constructs a Server with sane defaults.
func New(cfg config.ServerConfig, logger *slog.Logger, handler http.Handler) *Server {
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      handler,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}

	return &Server{
		cfg:    cfg,
		logger: logger,
		http:   srv,
	}
}

// Start begins serving HTTP traffic.
func (s *Server) Start() error {
	s.logger.Info("starting server", "addr", s.http.Addr)
	if err := s.http.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("http listen: %w", err)
	}
	return nil
}

// Shutdown gracefully terminates the server.
func (s *Server) Shutdown(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, s.cfg.ShutdownTimeout)
	defer cancel()

	s.logger.Info("shutting down server")
	if err := s.http.Shutdown(ctx); err != nil {
		return fmt.Errorf("http shutdown: %w", err)
	}
	return nil
}
