// Package plugin provides a flexible, extensible plugin system for the API Gateway.
//
// The plugin system allows developers to extend gateway functionality without
// modifying core code. Plugins can:
//   - Authenticate requests
//   - Rate limit requests
//   - Transform requests/responses
//   - Add logging and metrics
//   - Cache responses
//   - And much more...
//
// Plugin Lifecycle:
//
//	Request comes in
//	    ↓
//	[BeforeRequest Phase]
//	    → Global plugins (priority order)
//	    → Service plugins (priority order)
//	    → Route plugins (priority order)
//	    ↓
//	[Proxy to Backend]
//	    ↓
//	[AfterResponse Phase]
//	    → Route plugins (reverse priority)
//	    → Service plugins (reverse priority)
//	    → Global plugins (reverse priority)
//	    ↓
//	Response sent to client
//
// Creating a Plugin:
//
//	type MyPlugin struct {
//	    config MyConfig
//	}
//
//	func (p *MyPlugin) Name() string {
//	    return "my-plugin"
//	}
//
//	func (p *MyPlugin) Execute(ctx *Context) error {
//	    // Your plugin logic here
//	    if ctx.Phase == PhaseBeforeRequest {
//	        // Modify request
//	        ctx.Request.Header.Set("X-Custom", "value")
//	    } else {
//	        // Modify response
//	        ctx.Response.Header().Set("X-Processed", "true")
//	    }
//	    return nil
//	}
package plugin

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/saidutt46/switchboard-gateway/internal/database"
)

// Phase represents the execution phase of a plugin.
type Phase string

const (
	// PhaseBeforeRequest - Plugin executes before proxying to backend.
	// Use this phase for:
	//   - Authentication
	//   - Rate limiting
	//   - Request validation
	//   - Request transformation
	//   - Cache lookups
	PhaseBeforeRequest Phase = "before_request"

	// PhaseAfterResponse - Plugin executes after receiving backend response.
	// Use this phase for:
	//   - Response transformation
	//   - Response caching
	//   - Adding headers (CORS, security headers)
	//   - Logging response metrics
	//   - Compression
	PhaseAfterResponse Phase = "after_response"
)

// Plugin is the interface that all plugins must implement.
//
// Example implementation:
//
//	type AuthPlugin struct {
//	    secretKey string
//	}
//
//	func (p *AuthPlugin) Name() string {
//	    return "auth"
//	}
//
//	func (p *AuthPlugin) Execute(ctx *Context) error {
//	    token := ctx.Request.Header.Get("Authorization")
//	    if !p.validateToken(token) {
//	        ctx.Abort(401, "Unauthorized")
//	        return nil
//	    }
//	    ctx.Set("user_id", "123") // Pass data to next plugins
//	    return nil
//	}
type Plugin interface {
	// Name returns the unique identifier for this plugin.
	// Must match the plugin name in the database.
	Name() string

	// Execute runs the plugin logic.
	// Return nil if successful, error if plugin failed.
	//
	// The plugin can:
	//   - Read/modify ctx.Request (in BeforeRequest phase)
	//   - Read/modify ctx.Response (in AfterResponse phase)
	//   - Store data in ctx.Metadata for other plugins
	//   - Call ctx.Abort() to stop the chain
	Execute(ctx *Context) error
}

// Context holds all data available to plugins during execution.
//
// This is the primary way plugins interact with the gateway and each other.
type Context struct {
	// Request is the incoming HTTP request.
	// Plugins can read and modify this in BeforeRequest phase.
	Request *http.Request

	// Response is the response writer.
	// Plugins can write headers and body in AfterResponse phase.
	Response *ResponseWriter

	// Route is the matched route from the database.
	Route *database.Route

	// Service is the target backend service.
	Service *database.Service

	// Phase indicates whether this is before or after proxying.
	Phase Phase

	// StartTime is when the request started processing.
	StartTime time.Time

	// Metadata is a map for plugins to communicate with each other.
	// Example:
	//   ctx.Set("user_id", "123")
	//   userID := ctx.Get("user_id").(string)
	Metadata map[string]interface{}

	// aborted indicates if the chain should stop.
	aborted bool

	// abortStatusCode is the HTTP status code if aborted.
	abortStatusCode int

	// abortMessage is the error message if aborted.
	abortMessage string

	// Context for cancellation and timeouts
	ctx context.Context
}

// ResponseWriter wraps http.ResponseWriter to capture response data.
//
// This allows plugins to:
//   - Read the response status code
//   - Read/modify response headers
//   - Access response body (if buffered)
type ResponseWriter struct {
	http.ResponseWriter
	statusCode  int
	written     bool
	bodySize    int
	headersSent bool
}

// NewResponseWriter creates a new ResponseWriter wrapper.
func NewResponseWriter(w http.ResponseWriter) *ResponseWriter {
	return &ResponseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK, // Default to 200
		written:        false,
		bodySize:       0,
		headersSent:    false,
	}
}

// WriteHeader captures the status code and writes it.
func (w *ResponseWriter) WriteHeader(statusCode int) {
	if w.written {
		log.Warn().
			Str("component", "response_writer").
			Msg("WriteHeader called multiple times")
		return
	}

	w.statusCode = statusCode
	w.written = true
	w.headersSent = true
	w.ResponseWriter.WriteHeader(statusCode)
}

// Write writes the response body and captures the size.
func (w *ResponseWriter) Write(b []byte) (int, error) {
	if !w.written {
		w.WriteHeader(http.StatusOK)
	}

	n, err := w.ResponseWriter.Write(b)
	w.bodySize += n
	return n, err
}

// StatusCode returns the HTTP status code that was written.
func (w *ResponseWriter) StatusCode() int {
	return w.statusCode
}

// BodySize returns the number of bytes written to the response body.
func (w *ResponseWriter) BodySize() int {
	return w.bodySize
}

// Written returns true if WriteHeader has been called.
func (w *ResponseWriter) Written() bool {
	return w.written
}

// NewContext creates a new plugin context for a request.
func NewContext(
	r *http.Request,
	w http.ResponseWriter,
	route *database.Route,
	service *database.Service,
	phase Phase,
) *Context {
	return &Context{
		Request:   r,
		Response:  NewResponseWriter(w),
		Route:     route,
		Service:   service,
		Phase:     phase,
		StartTime: time.Now(),
		Metadata:  make(map[string]interface{}),
		aborted:   false,
		ctx:       r.Context(),
	}
}

// Set stores a value in the context metadata.
// This allows plugins to pass data to other plugins in the chain.
//
// Example:
//
//	ctx.Set("user_id", "123")
//	ctx.Set("rate_limit_remaining", 99)
func (c *Context) Set(key string, value interface{}) {
	c.Metadata[key] = value

	log.Debug().
		Str("component", "plugin_context").
		Str("key", key).
		Interface("value", value).
		Msg("Context value set")
}

// Get retrieves a value from the context metadata.
//
// Example:
//
//	userID, exists := ctx.Get("user_id")
//	if exists {
//	    id := userID.(string)
//	}
func (c *Context) Get(key string) (interface{}, bool) {
	value, exists := c.Metadata[key]
	return value, exists
}

// GetString is a type-safe helper to get a string value.
//
// Returns empty string if key doesn't exist or value is not a string.
func (c *Context) GetString(key string) string {
	if value, exists := c.Metadata[key]; exists {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}

// GetInt is a type-safe helper to get an int value.
//
// Returns 0 if key doesn't exist or value is not an int.
func (c *Context) GetInt(key string) int {
	if value, exists := c.Metadata[key]; exists {
		if num, ok := value.(int); ok {
			return num
		}
	}
	return 0
}

// GetBool is a type-safe helper to get a bool value.
//
// Returns false if key doesn't exist or value is not a bool.
func (c *Context) GetBool(key string) bool {
	if value, exists := c.Metadata[key]; exists {
		if b, ok := value.(bool); ok {
			return b
		}
	}
	return false
}

// Abort stops the plugin chain execution and returns an error response.
//
// This is used when a plugin determines the request should not continue:
//   - Authentication failure
//   - Rate limit exceeded
//   - Validation error
//
// Example:
//
//	if !authenticated {
//	    ctx.Abort(401, "Unauthorized - Invalid API key")
//	    return nil
//	}
func (c *Context) Abort(statusCode int, message string) {
	c.aborted = true
	c.abortStatusCode = statusCode
	c.abortMessage = message

	log.Info().
		Str("component", "plugin_context").
		Int("status_code", statusCode).
		Str("message", message).
		Msg("Request aborted by plugin")
}

// IsAborted returns true if the plugin chain has been aborted.
func (c *Context) IsAborted() bool {
	return c.aborted
}

// AbortStatusCode returns the HTTP status code set by Abort().
func (c *Context) AbortStatusCode() int {
	return c.abortStatusCode
}

// AbortMessage returns the error message set by Abort().
func (c *Context) AbortMessage() string {
	return c.abortMessage
}

// Context returns the underlying Go context for cancellation/timeouts.
func (c *Context) Context() context.Context {
	return c.ctx
}

// Elapsed returns the time elapsed since request started.
func (c *Context) Elapsed() time.Duration {
	return time.Since(c.StartTime)
}

// LogInfo logs an info message with plugin context.
func (c *Context) LogInfo(pluginName string, message string) {
	log.Info().
		Str("component", "plugin").
		Str("plugin", pluginName).
		Str("phase", string(c.Phase)).
		Str("route_id", c.Route.ID).
		Str("service_id", c.Service.ID).
		Dur("elapsed_ms", c.Elapsed()).
		Msg(message)
}

// LogError logs an error message with plugin context.
func (c *Context) LogError(pluginName string, err error, message string) {
	log.Error().
		Err(err).
		Str("component", "plugin").
		Str("plugin", pluginName).
		Str("phase", string(c.Phase)).
		Str("route_id", c.Route.ID).
		Str("service_id", c.Service.ID).
		Dur("elapsed_ms", c.Elapsed()).
		Msg(message)
}

// LogDebug logs a debug message with plugin context.
func (c *Context) LogDebug(pluginName string, message string) {
	log.Debug().
		Str("component", "plugin").
		Str("plugin", pluginName).
		Str("phase", string(c.Phase)).
		Str("route_id", c.Route.ID).
		Str("service_id", c.Service.ID).
		Msg(message)
}

// PluginError represents an error that occurred during plugin execution.
type PluginError struct {
	PluginName string
	Phase      Phase
	Err        error
	Critical   bool // If true, stop the entire chain
}

// Error implements the error interface.
func (e *PluginError) Error() string {
	return fmt.Sprintf("plugin '%s' failed in %s phase: %v", e.PluginName, e.Phase, e.Err)
}

// Unwrap returns the underlying error.
func (e *PluginError) Unwrap() error {
	return e.Err
}

// IsCritical returns true if this error should stop the chain.
func (e *PluginError) IsCritical() bool {
	return e.Critical
}

// NewPluginError creates a new PluginError.
func NewPluginError(pluginName string, phase Phase, err error, critical bool) *PluginError {
	return &PluginError{
		PluginName: pluginName,
		Phase:      phase,
		Err:        err,
		Critical:   critical,
	}
}
