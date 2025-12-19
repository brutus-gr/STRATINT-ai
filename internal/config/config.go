package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"
)

// Config represents runtime configuration derived from environment variables.
type Config struct {
	Server  ServerConfig
	Logging LoggingConfig
}

// ServerConfig holds HTTP server runtime parameters.
type ServerConfig struct {
	Port            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
}

// LoggingConfig represents structured logging configuration.
type LoggingConfig struct {
	Level  slog.Level
	Format string
}

const (
	defaultPort            = "8080"
	defaultReadTimeout     = 10 * time.Second
	defaultWriteTimeout    = 10 * time.Second
	defaultShutdownTimeout = 5 * time.Second

	defaultLogFormat = "json"
)

// Load reads configuration from environment variables, applying defaults when
// values are not provided or invalid.
func Load() (Config, error) {
	// Cloud Run sets PORT, but allow SERVER_PORT override for local dev
	port := getEnv("PORT", "")
	if port == "" {
		port = getEnv("SERVER_PORT", defaultPort)
	}

	cfg := Config{
		Server: ServerConfig{
			Port:            port,
			ReadTimeout:     defaultReadTimeout,
			WriteTimeout:    defaultWriteTimeout,
			ShutdownTimeout: defaultShutdownTimeout,
		},
		Logging: LoggingConfig{
			Level:  slog.LevelInfo,
			Format: defaultLogFormat,
		},
	}

	if v := os.Getenv("SERVER_READ_TIMEOUT_SECONDS"); v != "" {
		d, err := parseSeconds(v)
		if err != nil {
			return Config{}, fmt.Errorf("invalid SERVER_READ_TIMEOUT_SECONDS: %w", err)
		}
		cfg.Server.ReadTimeout = d
	}

	if v := os.Getenv("SERVER_WRITE_TIMEOUT_SECONDS"); v != "" {
		d, err := parseSeconds(v)
		if err != nil {
			return Config{}, fmt.Errorf("invalid SERVER_WRITE_TIMEOUT_SECONDS: %w", err)
		}
		cfg.Server.WriteTimeout = d
	}

	if v := os.Getenv("SERVER_SHUTDOWN_TIMEOUT_SECONDS"); v != "" {
		d, err := parseSeconds(v)
		if err != nil {
			return Config{}, fmt.Errorf("invalid SERVER_SHUTDOWN_TIMEOUT_SECONDS: %w", err)
		}
		cfg.Server.ShutdownTimeout = d
	}

	if v := os.Getenv("LOG_LEVEL"); v != "" {
		level, err := parseLogLevel(v)
		if err != nil {
			return Config{}, fmt.Errorf("invalid LOG_LEVEL: %w", err)
		}
		cfg.Logging.Level = level
	}

	if v := os.Getenv("LOG_FORMAT"); v != "" {
		switch v {
		case "json", "text":
			cfg.Logging.Format = v
		default:
			return Config{}, fmt.Errorf("invalid LOG_FORMAT: must be 'json' or 'text'")
		}
	}

	return cfg, nil
}

func parseSeconds(raw string) (time.Duration, error) {
	seconds, err := strconv.Atoi(raw)
	if err != nil || seconds < 0 {
		return 0, fmt.Errorf("must be a non-negative integer")
	}
	return time.Duration(seconds) * time.Second, nil
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func parseLogLevel(raw string) (slog.Level, error) {
	switch raw {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("must be one of debug, info, warn, error")
	}
}
