// Package builtin - Rate Limit plugin for request throttling
//
// This plugin enforces rate limits on incoming requests to protect
// backend services from overload and ensure fair usage.
//
// Features:
//   - Multiple algorithms: Token Bucket (burst-friendly), Sliding Window (strict)
//   - Identifier hierarchy: consumer_id > api_key > ip_address
//   - Standard rate limit headers (X-RateLimit-*)
//   - 429 Too Many Requests response
//   - Distributed state using Redis
//   - Hot reload support
//
// Configuration Example:
//
//	{
//	  "critical": false,
//	  "algorithm": "token-bucket",
//	  "limit": 1000,
//	  "window": "1m",
//	  "identifier": "consumer_id",
//	  "redis_url": "redis://localhost:6379/0",
//	  "key_prefix": "rate_limit:",
//	  "headers": true,
//	  "response_code": 429,
//	  "response_message": "Rate limit exceeded"
//	}
package builtin

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/saidutt46/switchboard-gateway/internal/plugin"
	"github.com/saidutt46/switchboard-gateway/internal/ratelimit"
)

// RateLimitPlugin implements rate limiting for the gateway.
type RateLimitPlugin struct {
	config        RateLimitConfig
	store         *ratelimit.RedisStore
	tokenBucket   *ratelimit.TokenBucket
	slidingWindow *ratelimit.SlidingWindow
}

// RateLimitConfig holds configuration for the rate limit plugin.
type RateLimitConfig struct {
	// Critical indicates if rate limit failure should stop the request
	// Usually false - we want to allow requests if Redis is down
	Critical bool `json:"critical"`

	// Algorithm selects the rate limiting algorithm
	// Options: "token-bucket", "sliding-window"
	// Default: "token-bucket"
	Algorithm string `json:"algorithm"`

	// Limit is the maximum number of requests allowed
	// Example: 1000 means 1000 requests per window
	Limit int `json:"limit"`

	// Window is the time duration for rate limiting
	// Format: "1s", "1m", "1h", "24h"
	// Examples: "1m" = 1 minute, "1h" = 1 hour
	Window string `json:"window"`

	// Identifier determines how to identify rate limit buckets
	// Options: "consumer_id", "api_key", "ip", "auto"
	// Default: "auto" (tries consumer_id > api_key > ip)
	Identifier string `json:"identifier"`

	// RedisURL is the Redis connection string
	// Default: "redis://localhost:6379/0"
	RedisURL string `json:"redis_url"`

	// KeyPrefix is prepended to all Redis keys
	// Default: "rate_limit:"
	KeyPrefix string `json:"key_prefix"`

	// Headers indicates if rate limit headers should be added
	// Default: true
	Headers bool `json:"headers"`

	// ResponseCode is the HTTP status code when rate limit is exceeded
	// Default: 429 (Too Many Requests)
	ResponseCode int `json:"response_code"`

	// ResponseMessage is the error message when rate limit is exceeded
	// Default: "Rate limit exceeded"
	ResponseMessage string `json:"response_message"`
}

// DefaultRateLimitConfig returns sensible defaults.
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		Critical:        false,
		Algorithm:       "token-bucket",
		Limit:           1000,
		Window:          "1m",
		Identifier:      "auto",
		RedisURL:        "redis://localhost:6379/0",
		KeyPrefix:       "rate_limit:",
		Headers:         true,
		ResponseCode:    429,
		ResponseMessage: "Rate limit exceeded",
	}
}

// NewRateLimitPlugin creates a new rate limit plugin.
//
// This is the factory function registered with the plugin registry.
func NewRateLimitPlugin(configJSON json.RawMessage) (plugin.Plugin, error) {
	// Start with defaults
	config := DefaultRateLimitConfig()

	// Override with user config if provided
	if len(configJSON) > 0 {
		if err := json.Unmarshal(configJSON, &config); err != nil {
			return nil, fmt.Errorf("invalid rate-limit config: %w", err)
		}
	}

	// Validate configuration
	if err := validateRateLimitConfig(config); err != nil {
		return nil, fmt.Errorf("invalid rate limit configuration: %w", err)
	}

	// Parse window duration
	windowDuration, err := parseWindowDuration(config.Window)
	if err != nil {
		return nil, fmt.Errorf("invalid window duration: %w", err)
	}

	log.Info().
		Str("component", "plugin").
		Str("plugin", "rate-limit").
		Str("algorithm", config.Algorithm).
		Int("limit", config.Limit).
		Str("window", config.Window).
		Str("identifier", config.Identifier).
		Msg("Initializing rate limit plugin")

	// Create Redis store
	redisConfig := ratelimit.DefaultRedisConfig()
	redisConfig.URL = config.RedisURL
	store, err := ratelimit.NewRedisStore(redisConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create redis store: %w", err)
	}

	// Create rate limiters based on algorithm
	var tokenBucket *ratelimit.TokenBucket
	var slidingWindow *ratelimit.SlidingWindow

	keyPrefix := config.KeyPrefix + config.Algorithm + ":"

	switch config.Algorithm {
	case "token-bucket":
		refillRate := ratelimit.CalculateRefillRate(config.Limit, windowDuration)
		tokenBucket = ratelimit.NewTokenBucket(store, ratelimit.TokenBucketConfig{
			Capacity:   config.Limit,
			RefillRate: refillRate,
			KeyPrefix:  keyPrefix,
			TTL:        windowDuration * 2,
		})

	case "sliding-window":
		slidingWindow = ratelimit.NewSlidingWindow(store, ratelimit.SlidingWindowConfig{
			Limit:     config.Limit,
			Window:    windowDuration,
			KeyPrefix: keyPrefix,
			TTL:       windowDuration * 2,
		})

	default:
		return nil, fmt.Errorf("unknown algorithm: %s", config.Algorithm)
	}

	log.Info().
		Str("component", "plugin").
		Str("plugin", "rate-limit").
		Msg("Rate limit plugin initialized successfully")

	return &RateLimitPlugin{
		config:        config,
		store:         store,
		tokenBucket:   tokenBucket,
		slidingWindow: slidingWindow,
	}, nil
}

// validateRateLimitConfig validates the plugin configuration.
func validateRateLimitConfig(config RateLimitConfig) error {
	// Validate algorithm
	validAlgorithms := []string{"token-bucket", "sliding-window"}
	valid := false
	for _, alg := range validAlgorithms {
		if config.Algorithm == alg {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("invalid algorithm '%s' (must be one of: %v)", config.Algorithm, validAlgorithms)
	}

	// Validate limit
	if config.Limit <= 0 {
		return fmt.Errorf("limit must be positive")
	}

	// Validate window format
	if _, err := parseWindowDuration(config.Window); err != nil {
		return fmt.Errorf("invalid window format: %w", err)
	}

	// Validate identifier
	validIdentifiers := []string{"consumer_id", "api_key", "ip", "auto"}
	valid = false
	for _, id := range validIdentifiers {
		if config.Identifier == id {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("invalid identifier '%s' (must be one of: %v)", config.Identifier, validIdentifiers)
	}

	// Validate response code
	if config.ResponseCode < 400 || config.ResponseCode >= 600 {
		return fmt.Errorf("response_code must be 4xx or 5xx")
	}

	return nil
}

// parseWindowDuration parses window string to time.Duration.
//
// Supports: "1s", "10s", "1m", "10m", "1h", "24h"
func parseWindowDuration(window string) (time.Duration, error) {
	duration, err := time.ParseDuration(window)
	if err != nil {
		return 0, fmt.Errorf("failed to parse duration: %w", err)
	}
	if duration <= 0 {
		return 0, fmt.Errorf("window must be positive")
	}
	return duration, nil
}

// Name returns the plugin identifier.
func (p *RateLimitPlugin) Name() string {
	return "rate-limit"
}

// Execute runs the rate limit plugin.
func (p *RateLimitPlugin) Execute(ctx *plugin.Context) error {
	// Only run in BeforeRequest phase
	if ctx.Phase != plugin.PhaseBeforeRequest {
		return nil
	}

	// Extract identifier for rate limiting
	identifier := p.getIdentifier(ctx)

	log.Debug().
		Str("component", "plugin").
		Str("plugin", "rate-limit").
		Str("identifier", identifier).
		Str("algorithm", p.config.Algorithm).
		Msg("Checking rate limit")

	// Check rate limit based on algorithm
	var allowed bool
	var remaining int
	var resetTime time.Time
	var retryAfter time.Duration

	switch p.config.Algorithm {
	case "token-bucket":
		result, err := p.tokenBucket.Allow(ctx.Context(), identifier)
		if err != nil {
			return p.handleError(ctx, err)
		}
		allowed = result.Allowed
		remaining = result.Remaining
		resetTime = result.ResetTime
		retryAfter = result.RetryAfter

	case "sliding-window":
		result, err := p.slidingWindow.Allow(ctx.Context(), identifier)
		if err != nil {
			return p.handleError(ctx, err)
		}
		allowed = result.Allowed
		remaining = result.Remaining
		resetTime = result.ResetTime
		retryAfter = result.RetryAfter
	}

	// Add rate limit headers if enabled
	if p.config.Headers {
		p.addRateLimitHeaders(ctx, remaining, resetTime, retryAfter)
	}

	// Check if request should be denied
	if !allowed {
		log.Warn().
			Str("component", "plugin").
			Str("plugin", "rate-limit").
			Str("identifier", identifier).
			Int("limit", p.config.Limit).
			Dur("retry_after", retryAfter).
			Msg("Rate limit exceeded")

		// Add Retry-After header
		if retryAfter > 0 {
			ctx.Response.Header().Set("Retry-After", fmt.Sprintf("%d", int(retryAfter.Seconds())))
		}

		// Abort request with 429
		ctx.Abort(p.config.ResponseCode, p.config.ResponseMessage)
		return nil
	}

	log.Debug().
		Str("component", "plugin").
		Str("plugin", "rate-limit").
		Str("identifier", identifier).
		Int("remaining", remaining).
		Msg("Rate limit check passed")

	return nil
}

// getIdentifier extracts the identifier for rate limiting.
//
// Hierarchy (configurable via config.Identifier):
//  1. consumer_id (from authentication plugin)
//  2. api_key (from X-API-Key header, hashed)
//  3. ip (from X-Forwarded-For or RemoteAddr)
func (p *RateLimitPlugin) getIdentifier(ctx *plugin.Context) string {
	// If specific identifier is requested, try that first
	if p.config.Identifier != "auto" {
		if id := p.tryGetIdentifier(ctx, p.config.Identifier); id != "" {
			return id
		}
	}

	// Auto mode: try in priority order
	// Priority 1: Consumer ID (from auth plugin)
	if consumerID := ctx.GetString("consumer_id"); consumerID != "" {
		return "consumer:" + consumerID
	}

	// Priority 2: API Key (from header, hashed for privacy)
	if apiKey := ctx.Request.Header.Get("X-API-Key"); apiKey != "" {
		hashedKey := hashAPIKey(apiKey)
		return "apikey:" + hashedKey
	}

	// Priority 3: IP Address (fallback)
	ip := getClientIP(ctx.Request)
	return "ip:" + ip
}

// tryGetIdentifier attempts to get a specific identifier type.
func (p *RateLimitPlugin) tryGetIdentifier(ctx *plugin.Context, identifierType string) string {
	switch identifierType {
	case "consumer_id":
		if consumerID := ctx.GetString("consumer_id"); consumerID != "" {
			return "consumer:" + consumerID
		}

	case "api_key":
		if apiKey := ctx.Request.Header.Get("X-API-Key"); apiKey != "" {
			hashedKey := hashAPIKey(apiKey)
			return "apikey:" + hashedKey
		}

	case "ip":
		ip := getClientIP(ctx.Request)
		return "ip:" + ip
	}

	return ""
}

// hashAPIKey hashes an API key for privacy.
//
// We don't store raw API keys in Redis - we hash them first.
func hashAPIKey(apiKey string) string {
	hash := sha256.Sum256([]byte(apiKey))
	return fmt.Sprintf("%x", hash[:8]) // Use first 8 bytes (16 hex chars)
}

// getClientIP extracts the client IP address from the request.
//
// Checks in order:
//  1. X-Forwarded-For header (proxy/load balancer)
//  2. X-Real-IP header (nginx)
//  3. RemoteAddr (direct connection)
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For (most common with proxies)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For can be a list: "client, proxy1, proxy2"
		// Take the first IP (original client)
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP (nginx)
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fallback to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr // Return as-is if can't parse
	}
	return ip
}

// addRateLimitHeaders adds standard rate limit headers to the response.
//
// Headers:
//   - X-RateLimit-Limit: Maximum requests allowed
//   - X-RateLimit-Remaining: Requests remaining in window
//   - X-RateLimit-Reset: Unix timestamp when limit resets
func (p *RateLimitPlugin) addRateLimitHeaders(
	ctx *plugin.Context,
	remaining int,
	resetTime time.Time,
	retryAfter time.Duration,
) {
	ctx.Response.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", p.config.Limit))
	ctx.Response.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
	ctx.Response.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", resetTime.Unix()))

	log.Debug().
		Str("component", "plugin").
		Str("plugin", "rate-limit").
		Int("limit", p.config.Limit).
		Int("remaining", remaining).
		Time("reset", resetTime).
		Msg("Rate limit headers added")
}

// handleError handles rate limiting errors.
//
// If critical=false (default), we allow the request through if Redis fails.
// If critical=true, we deny the request.
func (p *RateLimitPlugin) handleError(ctx *plugin.Context, err error) error {
	log.Error().
		Err(err).
		Str("component", "plugin").
		Str("plugin", "rate-limit").
		Bool("critical", p.config.Critical).
		Msg("Rate limit check failed")

	if p.config.Critical {
		// Critical: Deny request
		ctx.Abort(503, "Rate limiting service unavailable")
		return fmt.Errorf("rate limit check failed: %w", err)
	}

	// Non-critical: Allow request through
	log.Warn().
		Str("component", "plugin").
		Str("plugin", "rate-limit").
		Msg("Rate limit check failed but allowing request (non-critical)")

	return nil
}
