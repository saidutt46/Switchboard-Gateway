// Package logging provides structured logging using zerolog.
//
// It configures zerolog based on the application configuration
// and provides helpers for common logging patterns.
package logging

import (
	"io"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Setup configures the global logger based on the provided configuration.
//
// It sets the log level, output format, and other logging preferences.
// Should be called once during application initialization.
func Setup(level string, format string) error {
	// Set log level
	logLevel, err := parseLogLevel(level)
	if err != nil {
		return err
	}
	zerolog.SetGlobalLevel(logLevel)

	// Configure output format
	var output io.Writer = os.Stdout

	if format == "console" {
		// Console output with colors (for development)
		output = zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
			NoColor:    false,
		}
	} else {
		// JSON output (for production)
		// Already defaults to JSON, no special configuration needed
	}

	// Create logger with timestamp
	log.Logger = zerolog.New(output).With().Timestamp().Logger()

	// Add caller information in development
	if format == "console" {
		log.Logger = log.Logger.With().Caller().Logger()
	}

	log.Info().
		Str("level", level).
		Str("format", format).
		Msg("Logger initialized")

	return nil
}

// parseLogLevel converts a string log level to zerolog.Level.
func parseLogLevel(level string) (zerolog.Level, error) {
	switch strings.ToLower(level) {
	case "debug":
		return zerolog.DebugLevel, nil
	case "info":
		return zerolog.InfoLevel, nil
	case "warn", "warning":
		return zerolog.WarnLevel, nil
	case "error":
		return zerolog.ErrorLevel, nil
	default:
		return zerolog.InfoLevel, nil
	}
}

// WithRequestID adds a request ID to the logger context.
//
// Example usage:
//
//	logger := logging.WithRequestID(r.Context(), requestID)
//	logger.Info().Msg("Processing request")
func WithRequestID(requestID string) zerolog.Logger {
	return log.With().Str("request_id", requestID).Logger()
}

// WithComponent adds a component name to the logger context.
//
// Useful for identifying which part of the application is logging.
//
// Example usage:
//
//	logger := logging.WithComponent("proxy")
//	logger.Info().Msg("Proxying request to backend")
func WithComponent(component string) zerolog.Logger {
	return log.With().Str("component", component).Logger()
}

// WithError adds an error to the logger context.
//
// This is a convenience wrapper around zerolog's Err() method.
func WithError(err error) *zerolog.Event {
	return log.Error().Err(err)
}

// LogRequest logs an HTTP request with common fields.
//
// This is a helper for consistent request logging across the application.
func LogRequest(method, path string, statusCode int, latencyMs int64) {
	log.Info().
		Str("method", method).
		Str("path", path).
		Int("status", statusCode).
		Int64("latency_ms", latencyMs).
		Msg("Request completed")
}

// LogError logs an error with context.
//
// Includes the error message and any additional context fields.
func LogError(err error, msg string, fields map[string]interface{}) {
	event := log.Error().Err(err)

	// Add additional fields
	for key, value := range fields {
		switch v := value.(type) {
		case string:
			event.Str(key, v)
		case int:
			event.Int(key, v)
		case int64:
			event.Int64(key, v)
		case bool:
			event.Bool(key, v)
		case time.Duration:
			event.Dur(key, v)
		default:
			event.Interface(key, v)
		}
	}

	event.Msg(msg)
}

// LogPanic logs a panic with stack trace.
//
// Should be used in defer recover() blocks.
func LogPanic(recovered interface{}) {
	log.Error().
		Interface("panic", recovered).
		Stack().
		Msg("Panic recovered")
}
