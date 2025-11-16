// Package builtin provides built-in plugins that ship with the gateway.
//
// These plugins serve as both production-ready functionality and
// examples for developers creating custom plugins.
package builtin

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/saidutt46/switchboard-gateway/internal/plugin"
)

// RequestLoggerPlugin logs detailed information about each request.
//
// This plugin logs in two phases:
//  1. BeforeRequest: Logs incoming request details
//  2. AfterResponse: Logs response status, duration, and size
//
// It's useful for:
//   - Debugging and troubleshooting
//   - Performance monitoring
//   - Audit trails
//   - Traffic analysis
//   - Error tracking
//
// Configuration example:
//
//	{
//	  "critical": false,
//	  "log_headers": true,
//	  "log_query_params": true,
//	  "excluded_paths": ["/health", "/metrics"],
//	  "max_body_log_size": 1024
//	}
type RequestLoggerPlugin struct {
	config LoggerConfig
}

// LoggerConfig holds configuration for the request logger plugin.
type LoggerConfig struct {
	// Critical indicates if logging failure should stop the request.
	// Usually false - we don't want logging issues to break requests.
	Critical bool `json:"critical"`

	// LogHeaders enables logging of request/response headers.
	// Warning: May contain sensitive data (API keys, tokens).
	LogHeaders bool `json:"log_headers"`

	// LogQueryParams enables logging of URL query parameters.
	LogQueryParams bool `json:"log_query_params"`

	// ExcludedPaths is a list of paths to skip logging.
	// Useful for health checks and metrics endpoints.
	ExcludedPaths []string `json:"excluded_paths"`

	// MaxBodyLogSize limits how much of request/response body to log.
	// Set to 0 to disable body logging (recommended for production).
	MaxBodyLogSize int `json:"max_body_log_size"`
}

// DefaultLoggerConfig returns sensible defaults for production.
func DefaultLoggerConfig() LoggerConfig {
	return LoggerConfig{
		Critical:       false, // Don't break requests if logging fails
		LogHeaders:     false, // Don't log headers by default (sensitive data)
		LogQueryParams: true,  // Query params usually safe to log
		ExcludedPaths: []string{
			"/health",
			"/ready",
			"/metrics",
		},
		MaxBodyLogSize: 0, // Don't log bodies by default
	}
}

// NewRequestLogger creates a new request logger plugin.
//
// This is the factory function registered with the plugin registry.
func NewRequestLogger(configJSON json.RawMessage) (plugin.Plugin, error) {
	// Start with defaults
	config := DefaultLoggerConfig()

	// Override with user config if provided
	if len(configJSON) > 0 {
		if err := json.Unmarshal(configJSON, &config); err != nil {
			return nil, fmt.Errorf("invalid request-logger config: %w", err)
		}
	}

	return &RequestLoggerPlugin{
		config: config,
	}, nil
}

// Name returns the plugin identifier.
func (p *RequestLoggerPlugin) Name() string {
	return "request-logger"
}

// Execute runs the logging plugin.
func (p *RequestLoggerPlugin) Execute(ctx *plugin.Context) error {
	// Check if this path should be excluded from logging
	if p.shouldExclude(ctx.Request.URL.Path) {
		return nil
	}

	// Route to appropriate phase handler
	if ctx.Phase == plugin.PhaseBeforeRequest {
		return p.logRequest(ctx)
	} else {
		return p.logResponse(ctx)
	}
}

// logRequest logs incoming request details (BeforeRequest phase).
func (p *RequestLoggerPlugin) logRequest(ctx *plugin.Context) error {
	// Generate unique request ID for tracing
	requestID := fmt.Sprintf("req_%d", time.Now().UnixNano())

	// Store request ID in context for later phases and plugins
	ctx.Set("request_id", requestID)
	ctx.Set("request_start_time", time.Now())

	// Build log event
	event := log.Info().
		Str("component", "plugin").
		Str("plugin", "request-logger").
		Str("phase", "before_request").
		Str("request_id", requestID).
		Str("method", ctx.Request.Method).
		Str("path", ctx.Request.URL.Path).
		Str("remote_addr", ctx.Request.RemoteAddr).
		Str("user_agent", ctx.Request.UserAgent()).
		Str("route_id", ctx.Route.ID).
		Str("route_name", ctx.Route.Name.String).
		Str("service_id", ctx.Service.ID).
		Str("service_name", ctx.Service.Name)

	// Add query params if enabled
	if p.config.LogQueryParams && len(ctx.Request.URL.RawQuery) > 0 {
		event.Str("query", ctx.Request.URL.RawQuery)
	}

	// Add headers if enabled
	if p.config.LogHeaders {
		headers := make(map[string]string)
		for key, values := range ctx.Request.Header {
			// Don't log sensitive headers
			if p.isSensitiveHeader(key) {
				headers[key] = "[REDACTED]"
			} else {
				headers[key] = strings.Join(values, ", ")
			}
		}
		event.Interface("headers", headers)
	}

	event.Msg("Request received")

	return nil
}

// logResponse logs response details (AfterResponse phase).
func (p *RequestLoggerPlugin) logResponse(ctx *plugin.Context) error {
	// Retrieve request ID from context
	requestID := ctx.GetString("request_id")

	// Calculate request duration
	var duration time.Duration
	if startTime, exists := ctx.Get("request_start_time"); exists {
		if t, ok := startTime.(time.Time); ok {
			duration = time.Since(t)
		}
	}

	// Get response details
	statusCode := ctx.Response.StatusCode()
	bodySize := ctx.Response.BodySize()

	// Build log event
	event := log.Info().
		Str("component", "plugin").
		Str("plugin", "request-logger").
		Str("phase", "after_response").
		Str("request_id", requestID).
		Str("method", ctx.Request.Method).
		Str("path", ctx.Request.URL.Path).
		Int("status_code", statusCode).
		Int64("duration_ms", duration.Milliseconds()).
		Int("response_size", bodySize).
		Str("route_id", ctx.Route.ID).
		Str("service_id", ctx.Service.ID)

	// Add response headers if enabled
	if p.config.LogHeaders {
		headers := make(map[string]string)
		for key, values := range ctx.Response.Header() {
			headers[key] = strings.Join(values, ", ")
		}
		event.Interface("response_headers", headers)
	}

	// Determine log level based on status code
	var message string
	if statusCode >= 500 {
		event = log.Error().
			Str("component", "plugin").
			Str("plugin", "request-logger").
			Str("phase", "after_response").
			Str("request_id", requestID).
			Str("method", ctx.Request.Method).
			Str("path", ctx.Request.URL.Path).
			Int("status_code", statusCode).
			Int64("duration_ms", duration.Milliseconds()).
			Int("response_size", bodySize)
		message = "Request failed with 5xx error"
	} else if statusCode >= 400 {
		event = log.Warn().
			Str("component", "plugin").
			Str("plugin", "request-logger").
			Str("phase", "after_response").
			Str("request_id", requestID).
			Str("method", ctx.Request.Method).
			Str("path", ctx.Request.URL.Path).
			Int("status_code", statusCode).
			Int64("duration_ms", duration.Milliseconds()).
			Int("response_size", bodySize)
		message = "Request completed with client error"
	} else {
		message = "Request completed successfully"
	}

	event.Msg(message)

	return nil
}

// shouldExclude checks if a path should be excluded from logging.
func (p *RequestLoggerPlugin) shouldExclude(path string) bool {
	for _, excludedPath := range p.config.ExcludedPaths {
		if path == excludedPath {
			return true
		}
	}
	return false
}

// isSensitiveHeader checks if a header contains sensitive data.
//
// These headers are redacted in logs to prevent leaking credentials.
func (p *RequestLoggerPlugin) isSensitiveHeader(headerName string) bool {
	sensitiveHeaders := []string{
		"authorization",
		"x-api-key",
		"api-key",
		"apikey",
		"cookie",
		"set-cookie",
		"x-auth-token",
		"x-access-token",
		"proxy-authorization",
	}

	lowerHeader := strings.ToLower(headerName)
	for _, sensitive := range sensitiveHeaders {
		if lowerHeader == sensitive {
			return true
		}
	}

	return false
}
