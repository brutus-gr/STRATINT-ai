package config

import (
	"os"
	"testing"
	"time"

	"log/slog"
)

func TestLoadDefaults(t *testing.T) {
	clearConfigEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if cfg.Server.Port != defaultPort {
		t.Errorf("expected default port %q, got %q", defaultPort, cfg.Server.Port)
	}
	if cfg.Server.ReadTimeout != defaultReadTimeout {
		t.Errorf("expected default read timeout %v, got %v", defaultReadTimeout, cfg.Server.ReadTimeout)
	}
	if cfg.Server.WriteTimeout != defaultWriteTimeout {
		t.Errorf("expected default write timeout %v, got %v", defaultWriteTimeout, cfg.Server.WriteTimeout)
	}
	if cfg.Server.ShutdownTimeout != defaultShutdownTimeout {
		t.Errorf("expected default shutdown timeout %v, got %v", defaultShutdownTimeout, cfg.Server.ShutdownTimeout)
	}
	if cfg.Logging.Level != slog.LevelInfo {
		t.Errorf("expected default log level %v, got %v", slog.LevelInfo, cfg.Logging.Level)
	}
	if cfg.Logging.Format != defaultLogFormat {
		t.Errorf("expected default log format %q, got %q", defaultLogFormat, cfg.Logging.Format)
	}
}

func TestLoadWithOverrides(t *testing.T) {
	clearConfigEnv(t)

	overrides := map[string]string{
		"SERVER_PORT":                     "9090",
		"SERVER_READ_TIMEOUT_SECONDS":     "30",
		"SERVER_WRITE_TIMEOUT_SECONDS":    "45",
		"SERVER_SHUTDOWN_TIMEOUT_SECONDS": "15",
		"LOG_LEVEL":                       "debug",
		"LOG_FORMAT":                      "text",
	}
	for key, value := range overrides {
		t.Setenv(key, value)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if cfg.Server.Port != overrides["SERVER_PORT"] {
		t.Errorf("expected overridden port %q, got %q", overrides["SERVER_PORT"], cfg.Server.Port)
	}
	if cfg.Server.ReadTimeout != 30*time.Second {
		t.Errorf("expected read timeout %v, got %v", 30*time.Second, cfg.Server.ReadTimeout)
	}
	if cfg.Server.WriteTimeout != 45*time.Second {
		t.Errorf("expected write timeout %v, got %v", 45*time.Second, cfg.Server.WriteTimeout)
	}
	if cfg.Server.ShutdownTimeout != 15*time.Second {
		t.Errorf("expected shutdown timeout %v, got %v", 15*time.Second, cfg.Server.ShutdownTimeout)
	}
	if cfg.Logging.Level != slog.LevelDebug {
		t.Errorf("expected log level %v, got %v", slog.LevelDebug, cfg.Logging.Level)
	}
	if cfg.Logging.Format != overrides["LOG_FORMAT"] {
		t.Errorf("expected log format %q, got %q", overrides["LOG_FORMAT"], cfg.Logging.Format)
	}
}

func TestLoadPartialOverrides(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("SERVER_READ_TIMEOUT_SECONDS", "5")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if cfg.Server.ReadTimeout != 5*time.Second {
		t.Errorf("expected overridden read timeout %v, got %v", 5*time.Second, cfg.Server.ReadTimeout)
	}
	if cfg.Server.WriteTimeout != defaultWriteTimeout {
		t.Errorf("expected default write timeout %v, got %v", defaultWriteTimeout, cfg.Server.WriteTimeout)
	}
}

func TestLoadWithInvalidValues(t *testing.T) {
	tests := map[string]string{
		"SERVER_READ_TIMEOUT_SECONDS":     "-1",
		"SERVER_WRITE_TIMEOUT_SECONDS":    "abc",
		"SERVER_SHUTDOWN_TIMEOUT_SECONDS": "3.5",
		"LOG_LEVEL":                       "verbose",
		"LOG_FORMAT":                      "xml",
	}

	for key, value := range tests {
		t.Run(key, func(t *testing.T) {
			clearConfigEnv(t)
			t.Setenv(key, value)

			if _, err := Load(); err == nil {
				t.Fatalf("expected error when %s=%q", key, value)
			}
		})
	}
}

func TestParseLogLevelAliases(t *testing.T) {
	tests := map[string]slog.Level{
		"warn":    slog.LevelWarn,
		"warning": slog.LevelWarn,
	}

	for input, expected := range tests {
		level, err := parseLogLevel(input)
		if err != nil {
			t.Fatalf("parseLogLevel(%q) returned error: %v", input, err)
		}

		if level != expected {
			t.Errorf("parseLogLevel(%q) = %v, want %v", input, level, expected)
		}
	}
}

func TestParseSecondsRejectsInvalidInput(t *testing.T) {
	cases := []string{"-1", "abc"}

	for _, input := range cases {
		if _, err := parseSeconds(input); err == nil {
			t.Fatalf("expected error for input %q", input)
		}
	}
}

func TestLoadDoesNotPersistEnvBetweenRuns(t *testing.T) {
	clearConfigEnv(t)

	t.Setenv("SERVER_READ_TIMEOUT_SECONDS", "5")
	if _, err := Load(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := os.Unsetenv("SERVER_READ_TIMEOUT_SECONDS"); err != nil {
		t.Fatalf("failed to unset env: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.Server.ReadTimeout != defaultReadTimeout {
		t.Errorf("expected default read timeout after reset, got %v", cfg.Server.ReadTimeout)
	}
}

func clearConfigEnv(t *testing.T) {
	t.Helper()
	keys := []string{
		"SERVER_PORT",
		"SERVER_READ_TIMEOUT_SECONDS",
		"SERVER_WRITE_TIMEOUT_SECONDS",
		"SERVER_SHUTDOWN_TIMEOUT_SECONDS",
		"LOG_LEVEL",
		"LOG_FORMAT",
	}

	for _, key := range keys {
		t.Setenv(key, "")
	}
}
