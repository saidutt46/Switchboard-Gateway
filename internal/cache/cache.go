// Package cache provides response caching functionality for the API Gateway.
//
// The CacheManager handles storing and retrieving cached HTTP responses
// from Redis with support for TTL, serialization, and cache invalidation.
package cache

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

// CachedResponse represents a cached HTTP response.
//
// This structure is serialized to JSON and stored in Redis.
// The body is base64 encoded to handle binary content safely.
type CachedResponse struct {
	// StatusCode is the HTTP status code of the response
	StatusCode int `json:"status_code"`

	// Headers contains selected response headers to cache
	Headers map[string]string `json:"headers"`

	// Body is the base64-encoded response body
	Body string `json:"body"`

	// CachedAt is the Unix timestamp when the response was cached
	CachedAt int64 `json:"cached_at"`

	// TTL is the original TTL in seconds
	TTL int `json:"ttl"`

	// Metadata contains additional information about the cached response
	Metadata CacheMetadata `json:"metadata"`
}

// CacheMetadata contains additional information about the cached response.
type CacheMetadata struct {
	// RouteID is the route that generated this response
	RouteID string `json:"route_id"`

	// BackendLatencyMs is how long the backend took to respond
	BackendLatencyMs int64 `json:"backend_latency_ms"`

	// ContentLength is the size of the response body
	ContentLength int `json:"content_length"`

	// ConsumerID is the consumer who made the original request (if any)
	ConsumerID string `json:"consumer_id,omitempty"`
}

// Manager handles cache operations with Redis.
//
// It provides methods for storing, retrieving, and invalidating
// cached HTTP responses.
type Manager struct {
	redis      *redis.Client
	keyBuilder *KeyBuilder

	// Configuration
	defaultTTL    time.Duration
	maxTTL        time.Duration
	maxBodySize   int
	headersToKeep []string
}

// ManagerConfig holds configuration for the cache manager.
type ManagerConfig struct {
	// DefaultTTL is the default cache duration
	DefaultTTL time.Duration

	// MaxTTL caps the maximum cache duration
	MaxTTL time.Duration

	// MaxBodySize is the maximum response body size to cache (bytes)
	MaxBodySize int

	// HeadersToKeep lists response headers to cache
	// If empty, defaults to common headers
	HeadersToKeep []string

	// KeyBuilder configuration
	KeyPrefix         string
	VaryByConsumer    bool
	VaryByQuery       bool
	VaryByHeaders     []string
	IgnoreQueryParams []string
}

// DefaultManagerConfig returns sensible defaults.
func DefaultManagerConfig() ManagerConfig {
	return ManagerConfig{
		DefaultTTL:  60 * time.Second,
		MaxTTL:      1 * time.Hour,
		MaxBodySize: 1024 * 1024, // 1MB
		HeadersToKeep: []string{
			"Content-Type",
			"Content-Length",
			"Content-Encoding",
			"Cache-Control",
			"ETag",
			"Last-Modified",
		},
		KeyPrefix:         "cache:",
		VaryByConsumer:    false,
		VaryByQuery:       true,
		VaryByHeaders:     []string{},
		IgnoreQueryParams: []string{"utm_source", "utm_medium", "utm_campaign"},
	}
}

// NewManager creates a new cache manager.
func NewManager(redisClient *redis.Client, config ManagerConfig) *Manager {
	if redisClient == nil {
		log.Warn().
			Str("component", "cache").
			Msg("Redis client is nil - caching will be disabled")
		return nil
	}

	keyBuilder := &KeyBuilder{
		Prefix:            config.KeyPrefix,
		VaryByConsumer:    config.VaryByConsumer,
		VaryByQuery:       config.VaryByQuery,
		VaryByHeaders:     config.VaryByHeaders,
		IgnoreQueryParams: config.IgnoreQueryParams,
	}

	// Apply defaults if not set
	if config.DefaultTTL == 0 {
		config.DefaultTTL = 60 * time.Second
	}
	if config.MaxTTL == 0 {
		config.MaxTTL = 1 * time.Hour
	}
	if config.MaxBodySize == 0 {
		config.MaxBodySize = 1024 * 1024
	}
	if len(config.HeadersToKeep) == 0 {
		config.HeadersToKeep = DefaultManagerConfig().HeadersToKeep
	}

	log.Info().
		Str("component", "cache").
		Dur("default_ttl", config.DefaultTTL).
		Dur("max_ttl", config.MaxTTL).
		Int("max_body_size", config.MaxBodySize).
		Bool("vary_by_consumer", config.VaryByConsumer).
		Bool("vary_by_query", config.VaryByQuery).
		Msg("Cache manager initialized")

	return &Manager{
		redis:         redisClient,
		keyBuilder:    keyBuilder,
		defaultTTL:    config.DefaultTTL,
		maxTTL:        config.MaxTTL,
		maxBodySize:   config.MaxBodySize,
		headersToKeep: config.HeadersToKeep,
	}
}

// GetKeyBuilder returns the key builder for external use.
func (m *Manager) GetKeyBuilder() *KeyBuilder {
	return m.keyBuilder
}

// Get retrieves a cached response by key.
//
// Returns:
//   - *CachedResponse if found and valid
//   - nil if not found or expired
//   - error if Redis operation fails
func (m *Manager) Get(ctx context.Context, key string) (*CachedResponse, error) {
	if m == nil || m.redis == nil {
		return nil, nil
	}

	data, err := m.redis.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			// Cache miss - not an error
			return nil, nil
		}
		return nil, fmt.Errorf("redis get failed: %w", err)
	}

	var cached CachedResponse
	if err := json.Unmarshal(data, &cached); err != nil {
		// Corrupted cache entry - delete it
		log.Warn().
			Str("component", "cache").
			Str("key", key).
			Err(err).
			Msg("Failed to unmarshal cached response - deleting corrupted entry")
		m.redis.Del(ctx, key)
		return nil, nil
	}

	log.Debug().
		Str("component", "cache").
		Str("key", key).
		Int("status_code", cached.StatusCode).
		Int("body_size", len(cached.Body)).
		Int64("age_seconds", time.Now().Unix()-cached.CachedAt).
		Msg("Cache hit")

	return &cached, nil
}

// Set stores a response in the cache.
//
// Parameters:
//   - ctx: Context for cancellation
//   - key: Cache key
//   - response: The response to cache
//   - ttl: Time-to-live for this entry (capped by MaxTTL)
//
// Returns error if serialization or Redis operation fails.
func (m *Manager) Set(ctx context.Context, key string, response *CachedResponse, ttl time.Duration) error {
	if m == nil || m.redis == nil {
		return nil
	}

	// Cap TTL to max
	if ttl > m.maxTTL {
		ttl = m.maxTTL
	}

	// Set cache timestamp and TTL
	response.CachedAt = time.Now().Unix()
	response.TTL = int(ttl.Seconds())

	data, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal cached response: %w", err)
	}

	if err := m.redis.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("redis set failed: %w", err)
	}

	log.Debug().
		Str("component", "cache").
		Str("key", key).
		Int("status_code", response.StatusCode).
		Int("body_size", len(response.Body)).
		Dur("ttl", ttl).
		Msg("Response cached successfully")

	return nil
}

// Delete removes a cache entry by key.
func (m *Manager) Delete(ctx context.Context, key string) error {
	if m == nil || m.redis == nil {
		return nil
	}

	if err := m.redis.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("redis del failed: %w", err)
	}

	log.Debug().
		Str("component", "cache").
		Str("key", key).
		Msg("Cache entry deleted")

	return nil
}

// InvalidatePattern deletes all cache entries matching a pattern.
//
// This is useful for cache invalidation when a resource is updated.
// Uses Redis SCAN to avoid blocking on large datasets.
//
// Example patterns:
//   - "cache:route-123:*" - All entries for a route
//   - "cache:route-123:GET:*" - All GET entries for a route
func (m *Manager) InvalidatePattern(ctx context.Context, pattern string) (int64, error) {
	if m == nil || m.redis == nil {
		return 0, nil
	}

	var deleted int64
	var cursor uint64
	var err error

	for {
		var keys []string
		keys, cursor, err = m.redis.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return deleted, fmt.Errorf("redis scan failed: %w", err)
		}

		if len(keys) > 0 {
			count, err := m.redis.Del(ctx, keys...).Result()
			if err != nil {
				log.Warn().
					Str("component", "cache").
					Err(err).
					Int("keys", len(keys)).
					Msg("Failed to delete some cache keys")
			}
			deleted += count
		}

		if cursor == 0 {
			break
		}
	}

	log.Info().
		Str("component", "cache").
		Str("pattern", pattern).
		Int64("deleted", deleted).
		Msg("Cache invalidation completed")

	return deleted, nil
}

// BuildCacheKey generates a cache key for the given request.
func (m *Manager) BuildCacheKey(r *http.Request, routeID, consumerID string) (string, *KeyComponents) {
	if m == nil {
		return "", nil
	}
	return m.keyBuilder.BuildKey(r, routeID, consumerID)
}

// CreateCachedResponse creates a CachedResponse from an HTTP response.
//
// Parameters:
//   - statusCode: HTTP status code
//   - headers: Response headers
//   - body: Response body bytes
//   - metadata: Additional metadata
//
// Returns nil if body exceeds MaxBodySize.
func (m *Manager) CreateCachedResponse(
	statusCode int,
	headers http.Header,
	body []byte,
	metadata CacheMetadata,
) *CachedResponse {
	if m == nil {
		return nil
	}

	// Check body size limit
	if len(body) > m.maxBodySize {
		log.Debug().
			Str("component", "cache").
			Int("body_size", len(body)).
			Int("max_size", m.maxBodySize).
			Msg("Response body too large to cache")
		return nil
	}

	// Extract headers to keep
	cachedHeaders := make(map[string]string)
	for _, h := range m.headersToKeep {
		if v := headers.Get(h); v != "" {
			cachedHeaders[h] = v
		}
	}

	return &CachedResponse{
		StatusCode: statusCode,
		Headers:    cachedHeaders,
		Body:       base64.StdEncoding.EncodeToString(body),
		Metadata:   metadata,
	}
}

// DecodeBody decodes the base64-encoded response body.
func (cr *CachedResponse) DecodeBody() ([]byte, error) {
	return base64.StdEncoding.DecodeString(cr.Body)
}

// Age returns how long ago this response was cached.
func (cr *CachedResponse) Age() time.Duration {
	return time.Since(time.Unix(cr.CachedAt, 0))
}

// RemainingTTL returns the remaining time before this entry expires.
func (cr *CachedResponse) RemainingTTL() time.Duration {
	expiresAt := time.Unix(cr.CachedAt+int64(cr.TTL), 0)
	remaining := time.Until(expiresAt)
	if remaining < 0 {
		return 0
	}
	return remaining
}

// GetDefaultTTL returns the default TTL for the manager.
func (m *Manager) GetDefaultTTL() time.Duration {
	if m == nil {
		return 60 * time.Second
	}
	return m.defaultTTL
}

// GetMaxBodySize returns the maximum cacheable body size.
func (m *Manager) GetMaxBodySize() int {
	if m == nil {
		return 0
	}
	return m.maxBodySize
}

// IsEnabled returns true if caching is enabled (Redis available).
func (m *Manager) IsEnabled() bool {
	return m != nil && m.redis != nil
}
