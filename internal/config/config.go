// Package config loads the service configuration from environment variables.
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"
)

// Log formats supported by the service.
const (
	LogFormatJSON = "json"
	LogFormatText = "text"
)

// Config holds the runtime configuration of the service.
type Config struct {
	// Port is the TCP port the HTTP server listens on.
	Port int
	// LogLevel is the minimum level of emitted log records.
	LogLevel slog.Level
	// LogFormat is either LogFormatJSON or LogFormatText.
	LogFormat string
	// ShutdownTimeout bounds how long in-flight requests may take to finish
	// after a shutdown signal is received.
	ShutdownTimeout time.Duration
	// ShutdownDelay is how long the server keeps serving after it starts
	// reporting not ready, giving load balancers time to stop sending traffic.
	ShutdownDelay time.Duration
}

// Default returns the configuration used when no environment variables are set.
func Default() Config {
	return Config{
		Port:            8080,
		LogLevel:        slog.LevelInfo,
		LogFormat:       LogFormatJSON,
		ShutdownTimeout: 15 * time.Second,
		ShutdownDelay:   0,
	}
}

// Load builds a Config from the given lookup function, typically os.Getenv.
// Unset or empty variables fall back to their defaults.
func Load(getenv func(string) string) (Config, error) {
	cfg := Default()

	var errs []error
	if v := getenv("PORT"); v != "" {
		port, err := strconv.Atoi(v)
		if err != nil || port < 1 || port > 65535 {
			errs = append(errs, fmt.Errorf("PORT: invalid port %q", v))
		}
		cfg.Port = port
	}
	if v := getenv("LOG_LEVEL"); v != "" {
		if err := cfg.LogLevel.UnmarshalText([]byte(v)); err != nil {
			errs = append(errs, fmt.Errorf("LOG_LEVEL: %w", err))
		}
	}
	if v := getenv("LOG_FORMAT"); v != "" {
		switch v {
		case LogFormatJSON, LogFormatText:
			cfg.LogFormat = v
		default:
			errs = append(errs, fmt.Errorf("LOG_FORMAT: must be %q or %q, got %q", LogFormatJSON, LogFormatText, v))
		}
	}
	if v := getenv("SHUTDOWN_TIMEOUT"); v != "" {
		d, err := parseDuration(v)
		if err != nil {
			errs = append(errs, fmt.Errorf("SHUTDOWN_TIMEOUT: %w", err))
		}
		cfg.ShutdownTimeout = d
	}
	if v := getenv("SHUTDOWN_DELAY"); v != "" {
		d, err := parseDuration(v)
		if err != nil {
			errs = append(errs, fmt.Errorf("SHUTDOWN_DELAY: %w", err))
		}
		cfg.ShutdownDelay = d
	}

	if err := errors.Join(errs...); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func parseDuration(v string) (time.Duration, error) {
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, err
	}
	if d < 0 {
		return 0, fmt.Errorf("must not be negative, got %s", v)
	}
	return d, nil
}
