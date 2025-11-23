// Package builtin - API Key Authentication Plugin
//
// This plugin validates API keys from headers or query parameters
// and loads consumer information from the database.
//
// Features:
//   - Extract keys from headers (X-API-Key) or query params (?apikey=...)
//   - SHA256 hashing for secure key storage
//   - Redis caching (5 min TTL) for performance
//   - Consumer context injection
//   - Configurable bypass paths (health checks)
//   - Anonymous access mode
//
// Configuration Example:
//
//	{
//	  "critical": false,
//	  "key_names": ["X-API-Key", "apikey"],
//	  "key_in_query": true,
//	  "key_in_header": true,
//	  "hide_credentials": true,
//	  "cache_ttl_seconds": 300,
//	  "anonymous_allowed": false,
//	  "bypass_paths": ["/health", "/ready", "/metrics"]
//	}
package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
	"github.com/saidutt46/switchboard-gateway/internal/database"
	"github.com/saidutt46/switchboard-gateway/internal/plugin"
)

// Default paths that bypass authentication
var defaultBypassPaths = []string{"/health", "/ready", "/metrics"}

// APIKeyAuthPlugin validates API keys and loads consumer information.
type APIKeyAuthPlugin struct {
	config APIKeyAuthConfig
	repo   *database.Repository
	redis  *redis.Client
}

// APIKeyAuthConfig holds configuration for API key authentication.
type APIKeyAuthConfig struct {
	// Critical indicates if auth failure should stop the request
	// false = allow requests if Redis/DB fails (default)
	// true = block all requests on any failure
	Critical bool `json:"critical"`

	// KeyNames are the header/query parameter names to check for API keys
	// Default: ["X-API-Key"]
	// Example: ["X-API-Key", "apikey", "api_key"]
	KeyNames []string `json:"key_names"`

	// KeyInQuery enables API key extraction from query parameters
	// Default: true
	// Example: ?apikey=xyz123
	KeyInQuery bool `json:"key_in_query"`

	// KeyInHeader enables API key extraction from headers
	// Default: true
	// Example: X-API-Key: xyz123
	KeyInHeader bool `json:"key_in_header"`

	// HideCredentials removes API key from backend request
	// Default: true (recommended for security)
	HideCredentials bool `json:"hide_credentials"`

	// CacheTTLSeconds is Redis cache duration for consumer lookups
	// Default: 300 (5 minutes)
	CacheTTLSeconds int `json:"cache_ttl_seconds"`

	// AnonymousAllowed permits requests without API keys
	// Default: false
	// true = missing key is OK, invalid key still rejected
	AnonymousAllowed bool `json:"anonymous_allowed"`

	// BypassPaths are paths that skip authentication entirely
	// Default: ["/health", "/ready", "/metrics"]
	// Set to [] to disable all bypasses
	BypassPaths []string `json:"bypass_paths"`
}

// DefaultAPIKeyAuthConfig returns sensible defaults.
func DefaultAPIKeyAuthConfig() APIKeyAuthConfig {
	return APIKeyAuthConfig{
		Critical:         false,
		KeyNames:         []string{"X-API-Key"},
		KeyInQuery:       true,
		KeyInHeader:      true,
		HideCredentials:  true,
		CacheTTLSeconds:  300, // 5 minutes
		AnonymousAllowed: false,
		BypassPaths:      defaultBypassPaths,
	}
}

// NewAPIKeyAuthPlugin creates a new API key authentication plugin.
//
// This is the factory function registered with the plugin registry.
func NewAPIKeyAuthPlugin(configJSON json.RawMessage) (plugin.Plugin, error) {
	// Start with defaults
	config := DefaultAPIKeyAuthConfig()

	// Override with user config if provided
	if len(configJSON) > 0 {
		if err := json.Unmarshal(configJSON, &config); err != nil {
			return nil, fmt.Errorf("invalid api-key-auth config: %w", err)
		}
	}

	// Validate configuration
	if len(config.KeyNames) == 0 {
		return nil, fmt.Errorf("key_names cannot be empty")
	}

	if !config.KeyInHeader && !config.KeyInQuery {
		return nil, fmt.Errorf("at least one of key_in_header or key_in_query must be true")
	}

	log.Info().
		Str("component", "plugin").
		Str("plugin", "api-key-auth").
		Interface("key_names", config.KeyNames).
		Bool("key_in_header", config.KeyInHeader).
		Bool("key_in_query", config.KeyInQuery).
		Bool("anonymous_allowed", config.AnonymousAllowed).
		Int("cache_ttl", config.CacheTTLSeconds).
		Msg("Initializing API key authentication plugin")

	return &APIKeyAuthPlugin{
		config: config,
		// repo and redis will be injected by registry
	}, nil
}

// SetDependencies injects repository and Redis client.
// Called by Registry after plugin creation.
func (p *APIKeyAuthPlugin) SetDependencies(repo *database.Repository, redis *redis.Client) {
	p.repo = repo
	p.redis = redis

	log.Debug().
		Str("component", "plugin").
		Str("plugin", "api-key-auth").
		Bool("repo_injected", repo != nil).
		Bool("redis_injected", redis != nil).
		Msg("Dependencies injected")
}

// Name returns the plugin identifier.
func (p *APIKeyAuthPlugin) Name() string {
	return "api-key-auth"
}

// Execute validates API key and loads consumer information.
func (p *APIKeyAuthPlugin) Execute(ctx *plugin.Context) error {
	// Only run in BeforeRequest phase
	if ctx.Phase != plugin.PhaseBeforeRequest {
		return nil
	}

	// Check if path should bypass authentication
	if p.shouldBypass(ctx.Request.URL.Path) {
		log.Debug().
			Str("path", ctx.Request.URL.Path).
			Msg("Bypassing authentication for health check endpoint")
		return nil
	}

	// Extract API key from request
	apiKey := p.extractAPIKey(ctx.Request)

	if apiKey == "" {
		// No API key provided
		if p.config.AnonymousAllowed {
			log.Debug().Msg("No API key provided, allowing anonymous access")
			return nil
		}

		log.Info().
			Str("path", ctx.Request.URL.Path).
			Str("method", ctx.Request.Method).
			Msg("API key required but not provided")

		ctx.Abort(401, "API key required")
		return nil
	}

	// Hash the key (for database lookup)
	keyHash := hashAPIKeyFull(apiKey)

	log.Debug().
		Str("key_hash_prefix", keyHash[:16]).
		Msg("API key extracted and hashed")

	// Try to get consumer from cache
	consumer, err := p.getCachedConsumer(ctx.Context(), keyHash)
	if err != nil {
		log.Debug().
			Err(err).
			Msg("Cache lookup failed, will query database")
	}

	if consumer == nil {
		// Cache miss - query database
		log.Debug().Msg("Cache miss - querying database")

		consumer, err = p.repo.GetConsumerByAPIKeyHash(ctx.Context(), keyHash)
		if err != nil {
			log.Warn().
				Err(err).
				Str("key_hash_prefix", keyHash[:16]).
				Msg("Invalid API key - consumer not found")

			ctx.Abort(401, "Invalid API key")
			return nil
		}

		// Cache the consumer for next time
		if cacheErr := p.cacheConsumer(ctx.Context(), keyHash, consumer); cacheErr != nil {
			log.Warn().
				Err(cacheErr).
				Msg("Failed to cache consumer (non-critical)")
		}
	} else {
		log.Debug().
			Str("consumer_id", consumer.ID).
			Msg("Consumer found in cache")
	}

	// Add consumer information to context
	ctx.Set("consumer_id", consumer.ID)
	ctx.Set("consumer_username", consumer.Username)
	if consumer.Email.Valid {
		ctx.Set("consumer_email", consumer.Email.String)
	}

	// Forward consumer info to backend via headers
	ctx.Request.Header.Set("X-Consumer-ID", consumer.ID)
	ctx.Request.Header.Set("X-Consumer-Username", consumer.Username)
	if consumer.Email.Valid {
		ctx.Request.Header.Set("X-Consumer-Email", consumer.Email.String)
	}

	// Hide credentials from backend request
	if p.config.HideCredentials {
		p.removeAPIKey(ctx.Request)
	}

	log.Info().
		Str("consumer_id", consumer.ID).
		Str("consumer_username", consumer.Username).
		Str("path", ctx.Request.URL.Path).
		Msg("API key validated successfully")

	return nil
}

// shouldBypass checks if the path should bypass authentication.
func (p *APIKeyAuthPlugin) shouldBypass(path string) bool {
	// Use configured bypass paths
	paths := p.config.BypassPaths

	for _, bypassPath := range paths {
		if path == bypassPath || strings.HasPrefix(path, bypassPath+"/") {
			return true
		}
	}

	return false
}

// extractAPIKey retrieves API key from headers or query parameters.
func (p *APIKeyAuthPlugin) extractAPIKey(r *http.Request) string {
	// Try headers first (more secure)
	if p.config.KeyInHeader {
		for _, keyName := range p.config.KeyNames {
			if key := r.Header.Get(keyName); key != "" {
				return key
			}
		}
	}

	// Try query parameters (less secure, but convenient)
	if p.config.KeyInQuery {
		for _, keyName := range p.config.KeyNames {
			if key := r.URL.Query().Get(keyName); key != "" {
				return key
			}
		}
	}

	return ""
}

// removeAPIKey removes API key from request (before forwarding to backend).
func (p *APIKeyAuthPlugin) removeAPIKey(r *http.Request) {
	// Remove from headers
	for _, keyName := range p.config.KeyNames {
		r.Header.Del(keyName)
	}

	// Remove from query parameters
	if p.config.KeyInQuery {
		q := r.URL.Query()
		modified := false

		for _, keyName := range p.config.KeyNames {
			if q.Has(keyName) {
				q.Del(keyName)
				modified = true
			}
		}

		if modified {
			r.URL.RawQuery = q.Encode()
		}
	}
}

// getCachedConsumer retrieves consumer from Redis cache.
func (p *APIKeyAuthPlugin) getCachedConsumer(ctx context.Context, keyHash string) (*database.Consumer, error) {
	if p.redis == nil {
		return nil, fmt.Errorf("redis client not available")
	}

	cacheKey := fmt.Sprintf("auth:apikey:%s", keyHash)

	// Get from Redis
	data, err := p.redis.Get(ctx, cacheKey).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Cache miss
		}
		return nil, fmt.Errorf("redis get failed: %w", err)
	}

	// Deserialize consumer
	var consumer database.Consumer
	if err := json.Unmarshal(data, &consumer); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cached consumer: %w", err)
	}

	return &consumer, nil
}

// cacheConsumer stores consumer in Redis cache.
func (p *APIKeyAuthPlugin) cacheConsumer(ctx context.Context, keyHash string, consumer *database.Consumer) error {
	if p.redis == nil {
		return fmt.Errorf("redis client not available")
	}

	cacheKey := fmt.Sprintf("auth:apikey:%s", keyHash)

	// Serialize consumer
	data, err := json.Marshal(consumer)
	if err != nil {
		return fmt.Errorf("failed to marshal consumer: %w", err)
	}

	// Store in Redis with TTL
	ttl := time.Duration(p.config.CacheTTLSeconds) * time.Second
	if err := p.redis.Set(ctx, cacheKey, data, ttl).Err(); err != nil {
		return fmt.Errorf("redis set failed: %w", err)
	}

	log.Debug().
		Str("consumer_id", consumer.ID).
		Dur("ttl", ttl).
		Msg("Consumer cached successfully")

	return nil
}
