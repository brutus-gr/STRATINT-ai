package logging

import (
	"context"
	"strings"
	"testing"

	"log/slog"

	"github.com/STRATINT/stratint/internal/config"
)

func TestNewConfiguresSupportedFormats(t *testing.T) {
	tests := []struct {
		name   string
		format string
		level  slog.Level
	}{
		{name: "json", format: "json", level: slog.LevelWarn},
		{name: "text", format: "text", level: slog.LevelDebug},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := New(config.LoggingConfig{Level: tt.level, Format: tt.format})
			if err != nil {
				t.Fatalf("New returned error: %v", err)
			}

			if logger == nil {
				t.Fatal("expected non-nil logger")
			}

			levels := []slog.Level{slog.LevelDebug, slog.LevelInfo, slog.LevelWarn, slog.LevelError}
			ctx := context.Background()
			for _, lvl := range levels {
				enabled := logger.Enabled(ctx, lvl)
				expected := lvl >= tt.level
				if enabled != expected {
					t.Fatalf("logger level %v enabled(%v)=%t, want %t", tt.level, lvl, enabled, expected)
				}
			}

			if logger.Handler() == nil {
				t.Fatal("expected handler to be configured")
			}
		})
	}
}

func TestNewWithUnsupportedFormat(t *testing.T) {
	_, err := New(config.LoggingConfig{Level: slog.LevelInfo, Format: "pretty"})
	if err == nil {
		t.Fatal("expected error for unsupported format, got nil")
	}

	if !strings.Contains(err.Error(), "unsupported log format") {
		t.Fatalf("unexpected error: %v", err)
	}
}
