package logging

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/STRATINT/stratint/internal/config"
)

// New constructs a slog.Logger configured according to the provided settings.
func New(cfg config.LoggingConfig) (*slog.Logger, error) {
	handler, err := buildHandler(cfg)
	if err != nil {
		return nil, err
	}

	return slog.New(handler), nil
}

func buildHandler(cfg config.LoggingConfig) (slog.Handler, error) {
	opts := &slog.HandlerOptions{Level: cfg.Level}

	switch cfg.Format {
	case "json":
		return slog.NewJSONHandler(os.Stdout, opts), nil
	case "text":
		return slog.NewTextHandler(os.Stdout, opts), nil
	default:
		return nil, fmt.Errorf("unsupported log format: %s", cfg.Format)
	}
}
