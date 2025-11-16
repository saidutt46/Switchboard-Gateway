// Package builtin - CORS plugin for handling Cross-Origin Resource Sharing
//
// CORS allows web applications from one domain to access resources from another domain.
// This plugin adds the necessary headers and handles preflight OPTIONS requests.
package builtin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/saidutt46/switchboard-gateway/internal/plugin"
)

// CORSPlugin handles Cross-Origin Resource Sharing (CORS) for the gateway.
//
// CORS is a security feature that allows you to control which domains
// can access your API from web browsers.
//
// This plugin:
//   - Adds CORS headers to responses
//   - Handles preflight OPTIONS requests
//   - Supports wildcards and specific origins
//   - Configurable headers and methods
//
// Configuration example:
//
//	{
//	  "critical": false,
//	  "allowed_origins": ["https://example.com", "https://app.example.com"],
//	  "allowed_methods": ["GET", "POST", "PUT", "DELETE"],
//	  "allowed_headers": ["Content-Type", "Authorization"],
//	  "exposed_headers": ["X-Request-ID"],
//	  "allow_credentials": true,
//	  "max_age": 86400
//	}
type CORSPlugin struct {
	config CORSConfig
}

// CORSConfig holds configuration for CORS handling.
type CORSConfig struct {
	// Critical indicates if CORS failure should stop the request.
	// Usually false - CORS is for browser security, not API security.
	Critical bool `json:"critical"`

	// AllowedOrigins is a list of allowed origin domains.
	// Use ["*"] to allow all origins (not recommended for production).
	// Examples: ["https://example.com", "https://app.example.com"]
	AllowedOrigins []string `json:"allowed_origins"`

	// AllowedMethods is a list of allowed HTTP methods.
	// Default: ["GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"]
	AllowedMethods []string `json:"allowed_methods"`

	// AllowedHeaders is a list of allowed request headers.
	// Default: ["Content-Type", "Authorization"]
	AllowedHeaders []string `json:"allowed_headers"`

	// ExposedHeaders is a list of headers exposed to the client.
	// These headers can be read by JavaScript in the browser.
	ExposedHeaders []string `json:"exposed_headers"`

	// AllowCredentials indicates if credentials (cookies, auth) are allowed.
	// Cannot be true when AllowedOrigins is ["*"].
	AllowCredentials bool `json:"allow_credentials"`

	// MaxAge is how long (in seconds) preflight results can be cached.
	// Default: 86400 (24 hours)
	MaxAge int `json:"max_age"`
}

// DefaultCORSConfig returns secure defaults for CORS.
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		Critical: false,
		AllowedOrigins: []string{
			"*", // Allow all origins by default (can be restricted per route)
		},
		AllowedMethods: []string{
			"GET",
			"POST",
			"PUT",
			"DELETE",
			"PATCH",
			"OPTIONS",
		},
		AllowedHeaders: []string{
			"Content-Type",
			"Authorization",
			"X-Requested-With",
			"Accept",
		},
		ExposedHeaders: []string{
			"X-Request-ID",
		},
		AllowCredentials: false,
		MaxAge:           86400, // 24 hours
	}
}

// NewCORSPlugin creates a new CORS plugin.
//
// This is the factory function registered with the plugin registry.
func NewCORSPlugin(configJSON json.RawMessage) (plugin.Plugin, error) {
	// Start with defaults
	config := DefaultCORSConfig()

	// Override with user config if provided
	if len(configJSON) > 0 {
		if err := json.Unmarshal(configJSON, &config); err != nil {
			return nil, fmt.Errorf("invalid cors config: %w", err)
		}
	}

	// Validate configuration
	if err := validateCORSConfig(config); err != nil {
		return nil, fmt.Errorf("invalid cors configuration: %w", err)
	}

	log.Debug().
		Str("component", "plugin").
		Str("plugin", "cors").
		Interface("config", config).
		Msg("CORS plugin initialized")

	return &CORSPlugin{
		config: config,
	}, nil
}

// validateCORSConfig validates CORS configuration.
func validateCORSConfig(config CORSConfig) error {
	// Check for credentials with wildcard origin
	if config.AllowCredentials {
		for _, origin := range config.AllowedOrigins {
			if origin == "*" {
				return fmt.Errorf("allow_credentials cannot be true when allowed_origins contains '*'")
			}
		}
	}

	// Validate max age
	if config.MaxAge < 0 {
		return fmt.Errorf("max_age must be positive")
	}

	return nil
}

// Name returns the plugin identifier.
func (p *CORSPlugin) Name() string {
	return "cors"
}

// Execute runs the CORS plugin.
func (p *CORSPlugin) Execute(ctx *plugin.Context) error {
	// CORS primarily works in BeforeRequest phase
	if ctx.Phase == plugin.PhaseBeforeRequest {
		return p.handleBeforeRequest(ctx)
	} else {
		return p.handleAfterResponse(ctx)
	}
}

// handleBeforeRequest handles the BeforeRequest phase.
//
// This is where we:
//   - Handle preflight OPTIONS requests
//   - Check if origin is allowed
//   - Prepare CORS headers
func (p *CORSPlugin) handleBeforeRequest(ctx *plugin.Context) error {
	origin := ctx.Request.Header.Get("Origin")

	// No Origin header = not a CORS request
	if origin == "" {
		ctx.LogDebug("cors", "No Origin header - not a CORS request")
		return nil
	}

	// Check if origin is allowed
	if !p.isOriginAllowed(origin) {
		ctx.LogInfo("cors", fmt.Sprintf("Origin not allowed: %s", origin))
		// Don't abort - just don't add CORS headers
		// Browser will block it
		return nil
	}

	// Store that this is a valid CORS request
	ctx.Set("cors_origin_allowed", true)
	ctx.Set("cors_origin", origin)

	// Handle preflight OPTIONS request
	if ctx.Request.Method == "OPTIONS" {
		return p.handlePreflight(ctx, origin)
	}

	ctx.LogDebug("cors", fmt.Sprintf("CORS request from allowed origin: %s", origin))
	return nil
}

// handleAfterResponse handles the AfterResponse phase.
//
// This is where we add CORS headers to the actual response.
func (p *CORSPlugin) handleAfterResponse(ctx *plugin.Context) error {
	// Check if this was a valid CORS request
	if !ctx.GetBool("cors_origin_allowed") {
		return nil
	}

	origin := ctx.GetString("cors_origin")

	// Add CORS headers to response
	p.addCORSHeaders(ctx.Response, origin)

	ctx.LogDebug("cors", "CORS headers added to response")
	return nil
}

// handlePreflight handles CORS preflight OPTIONS requests.
//
// Preflight requests are sent by browsers before the actual request
// to check if the CORS request is safe to send.
func (p *CORSPlugin) handlePreflight(ctx *plugin.Context, origin string) error {
	ctx.LogInfo("cors", "Handling CORS preflight request")

	// Add CORS headers
	p.addCORSHeaders(ctx.Response, origin)

	// Add preflight-specific headers
	ctx.Response.Header().Set(
		"Access-Control-Max-Age",
		fmt.Sprintf("%d", p.config.MaxAge),
	)

	// Respond with 204 No Content for preflight
	ctx.Response.WriteHeader(http.StatusNoContent)

	// Abort chain - preflight is complete
	ctx.Abort(http.StatusNoContent, "CORS preflight successful")

	ctx.LogInfo("cors", "CORS preflight completed successfully")
	return nil
}

// addCORSHeaders adds CORS headers to the response.
func (p *CORSPlugin) addCORSHeaders(w *plugin.ResponseWriter, origin string) {
	// Access-Control-Allow-Origin
	if p.hasWildcardOrigin() {
		w.Header().Set("Access-Control-Allow-Origin", "*")
	} else {
		w.Header().Set("Access-Control-Allow-Origin", origin)
	}

	// Access-Control-Allow-Methods
	if len(p.config.AllowedMethods) > 0 {
		w.Header().Set(
			"Access-Control-Allow-Methods",
			strings.Join(p.config.AllowedMethods, ", "),
		)
	}

	// Access-Control-Allow-Headers
	if len(p.config.AllowedHeaders) > 0 {
		w.Header().Set(
			"Access-Control-Allow-Headers",
			strings.Join(p.config.AllowedHeaders, ", "),
		)
	}

	// Access-Control-Expose-Headers
	if len(p.config.ExposedHeaders) > 0 {
		w.Header().Set(
			"Access-Control-Expose-Headers",
			strings.Join(p.config.ExposedHeaders, ", "),
		)
	}

	// Access-Control-Allow-Credentials
	if p.config.AllowCredentials {
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	}

	// Vary header for caching
	w.Header().Add("Vary", "Origin")
}

// isOriginAllowed checks if an origin is in the allowed list.
func (p *CORSPlugin) isOriginAllowed(origin string) bool {
	// Check for wildcard
	if p.hasWildcardOrigin() {
		return true
	}

	// Check exact match
	for _, allowed := range p.config.AllowedOrigins {
		if allowed == origin {
			return true
		}

		// Support subdomain wildcards: *.example.com
		if strings.HasPrefix(allowed, "*.") {
			domain := allowed[2:] // Remove "*."
			if strings.HasSuffix(origin, domain) {
				return true
			}
		}
	}

	return false
}

// hasWildcardOrigin checks if wildcard origin is configured.
func (p *CORSPlugin) hasWildcardOrigin() bool {
	for _, origin := range p.config.AllowedOrigins {
		if origin == "*" {
			return true
		}
	}
	return false
}
