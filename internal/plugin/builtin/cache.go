// Package builtin - Response Cache Plugin
//
// This plugin caches HTTP responses in Redis to reduce backend load
// and improve response times.
package builtin

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
	"github.com/saidutt46/switchboard-gateway/internal/cache"
	"github.com/saidutt46/switchboard-gateway/internal/database"
	"github.com/saidutt46/switchboard-gateway/internal/plugin"
)

// CachePlugin provides response caching functionality.
type CachePlugin struct {
	config       CacheConfig
	cacheManager *cache.Manager
	redis        *redis.Client
}

// CacheConfig holds configuration for the cache plugin.
type CacheConfig struct {
	// Critical indicates if cache failures should stop the request
	Critical bool `json:"critical"`

	// TTLSeconds is the default cache duration in seconds
	TTLSeconds int `json:"ttl_seconds"`

	// MaxTTLSeconds caps the maximum cache duration
	MaxTTLSeconds int `json:"max_ttl_seconds"`

	// CacheMethods lists HTTP methods to cache
	// Default: ["GET", "HEAD"]
	CacheMethods []string `json:"cache_methods"`

	// CacheStatusCodes lists status codes to cache
	// Default: [200, 301, 302]
	CacheStatusCodes []int `json:"cache_status_codes"`

	// MaxBodySizeBytes is the maximum response body size to cache
	// Default: 1048576 (1MB)
	MaxBodySizeBytes int `json:"max_body_size_bytes"`

	// BypassPaths are paths that skip caching entirely
	// Default: ["/health", "/ready", "/metrics"]
	BypassPaths []string `json:"bypass_paths"`

	// VaryByConsumer includes consumer ID in cache key
	// When true, each consumer gets their own cached response
	VaryByConsumer bool `json:"vary_by_consumer"`

	// VaryByQuery includes query parameters in cache key
	// Default: true
	VaryByQuery bool `json:"vary_by_query"`

	// VaryByHeaders includes specific header values in cache key
	VaryByHeaders []string `json:"vary_by_headers"`

	// IgnoreQueryParams lists query params to exclude from cache key
	// Useful for tracking params like utm_source
	IgnoreQueryParams []string `json:"ignore_query_params"`

	// RespectCacheControl honors backend Cache-Control headers
	// When true, uses max-age from backend if present
	RespectCacheControl bool `json:"respect_cache_control"`

	// RespectNoCache skips caching when backend sends no-store/no-cache
	RespectNoCache bool `json:"respect_no_cache"`

	// CacheErrors enables caching of error responses (4xx, 5xx)
	// Default: false
	CacheErrors bool `json:"cache_errors"`

	// ErrorTTLSeconds is the TTL for cached error responses
	// Default: 30
	ErrorTTLSeconds int `json:"error_ttl_seconds"`

	// AddDebugHeaders adds X-Cache-Key header for debugging
	AddDebugHeaders bool `json:"add_debug_headers"`

	// KeyPrefix is the Redis key prefix
	// Default: "cache:"
	KeyPrefix string `json:"key_prefix"`
}

// DefaultCacheConfig returns sensible defaults.
func DefaultCacheConfig() CacheConfig {
	return CacheConfig{
		Critical:            false,
		TTLSeconds:          60,
		MaxTTLSeconds:       3600,
		CacheMethods:        []string{"GET", "HEAD"},
		CacheStatusCodes:    []int{200, 301, 302},
		MaxBodySizeBytes:    1024 * 1024, // 1MB
		BypassPaths:         []string{"/health", "/ready", "/metrics"},
		VaryByConsumer:      false,
		VaryByQuery:         true,
		VaryByHeaders:       []string{},
		IgnoreQueryParams:   []string{"utm_source", "utm_medium", "utm_campaign"},
		RespectCacheControl: true,
		RespectNoCache:      true,
		CacheErrors:         false,
		ErrorTTLSeconds:     30,
		AddDebugHeaders:     false,
		KeyPrefix:           "cache:",
	}
}

// Context keys for passing data between phases
const (
	cacheKeyCtxKey       = "_cache_key"
	cacheSkipCtxKey      = "_cache_skip"
	cacheStartTimeCtxKey = "_cache_start_time"
)

// NewCachePlugin creates a new cache plugin.
func NewCachePlugin(configJSON json.RawMessage) (plugin.Plugin, error) {
	config := DefaultCacheConfig()

	if len(configJSON) > 0 {
		if err := json.Unmarshal(configJSON, &config); err != nil {
			return nil, err
		}
	}

	// Apply defaults for empty slices
	if len(config.CacheMethods) == 0 {
		config.CacheMethods = DefaultCacheConfig().CacheMethods
	}
	if len(config.CacheStatusCodes) == 0 {
		config.CacheStatusCodes = DefaultCacheConfig().CacheStatusCodes
	}
	if len(config.BypassPaths) == 0 {
		config.BypassPaths = DefaultCacheConfig().BypassPaths
	}

	log.Info().
		Str("component", "plugin").
		Str("plugin", "cache").
		Int("ttl_seconds", config.TTLSeconds).
		Int("max_body_size", config.MaxBodySizeBytes).
		Bool("vary_by_consumer", config.VaryByConsumer).
		Bool("vary_by_query", config.VaryByQuery).
		Strs("cache_methods", config.CacheMethods).
		Ints("cache_status_codes", config.CacheStatusCodes).
		Bool("respect_cache_control", config.RespectCacheControl).
		Bool("cache_errors", config.CacheErrors).
		Msg("Initializing response cache plugin")

	return &CachePlugin{
		config: config,
	}, nil
}

// SetDependencies injects repository and Redis client.
// Called by Registry after plugin creation.
// Note: Cache plugin only needs Redis, not the repository.
func (p *CachePlugin) SetDependencies(repo *database.Repository, redisClient *redis.Client) {
	p.redis = redisClient

	if redisClient == nil {
		log.Warn().
			Str("component", "plugin").
			Str("plugin", "cache").
			Msg("Redis client is nil - caching disabled")
		return
	}

	// Create cache manager with config
	managerConfig := cache.ManagerConfig{
		DefaultTTL:        time.Duration(p.config.TTLSeconds) * time.Second,
		MaxTTL:            time.Duration(p.config.MaxTTLSeconds) * time.Second,
		MaxBodySize:       p.config.MaxBodySizeBytes,
		KeyPrefix:         p.config.KeyPrefix,
		VaryByConsumer:    p.config.VaryByConsumer,
		VaryByQuery:       p.config.VaryByQuery,
		VaryByHeaders:     p.config.VaryByHeaders,
		IgnoreQueryParams: p.config.IgnoreQueryParams,
	}

	p.cacheManager = cache.NewManager(redisClient, managerConfig)

	log.Debug().
		Str("component", "plugin").
		Str("plugin", "cache").
		Bool("cache_manager_ready", p.cacheManager != nil).
		Msg("Dependencies injected, cache manager created")
}

// Name returns the plugin identifier.
func (p *CachePlugin) Name() string {
	return "cache"
}

// Execute handles cache logic for both request and response phases.
func (p *CachePlugin) Execute(ctx *plugin.Context) error {
	switch ctx.Phase {
	case plugin.PhaseBeforeRequest:
		return p.handleBeforeRequest(ctx)
	case plugin.PhaseAfterResponse:
		return p.handleAfterResponse(ctx)
	}
	return nil
}

// handleBeforeRequest checks the cache and returns cached response if available.
func (p *CachePlugin) handleBeforeRequest(ctx *plugin.Context) error {
	// Check if caching is enabled
	if p.cacheManager == nil || !p.cacheManager.IsEnabled() {
		return nil
	}

	// Check if request method is cacheable
	if !p.isCacheableMethod(ctx.Request.Method) {
		log.Debug().
			Str("method", ctx.Request.Method).
			Str("path", ctx.Request.URL.Path).
			Msg("Request method not cacheable")
		ctx.Set(cacheSkipCtxKey, true)
		return nil
	}

	// Check if path should bypass cache
	if p.shouldBypass(ctx.Request.URL.Path) {
		log.Debug().
			Str("path", ctx.Request.URL.Path).
			Msg("Path bypasses cache")
		ctx.Set(cacheSkipCtxKey, true)
		return nil
	}

	// Get consumer ID from context (set by auth plugin)
	consumerID := ctx.GetString("consumer_id")

	// Get route ID from Route object or context
	routeID := ""
	if ctx.Route != nil {
		routeID = ctx.Route.ID
	}
	if routeID == "" {
		routeID = ctx.GetString("route_id")
	}
	if routeID == "" {
		routeID = "unknown"
	}

	// Store start time for latency calculation
	ctx.Set(cacheStartTimeCtxKey, time.Now())

	// Generate cache key
	cacheKey, keyComponents := p.cacheManager.BuildCacheKey(ctx.Request, routeID, consumerID)

	// Store key in context for AfterResponse phase
	ctx.Set(cacheKeyCtxKey, cacheKey)

	log.Debug().
		Str("cache_key", cacheKey).
		Str("route_id", routeID).
		Str("consumer_id", consumerID).
		Str("path", ctx.Request.URL.Path).
		Msg("Checking cache")

	// Look up cached response
	cached, err := p.cacheManager.Get(ctx.Request.Context(), cacheKey)
	if err != nil {
		log.Warn().
			Err(err).
			Str("cache_key", cacheKey).
			Msg("Cache lookup failed")
		// Continue to backend on cache error
		return nil
	}

	// Cache miss
	if cached == nil {
		log.Debug().
			Str("cache_key", cacheKey).
			Msg("Cache miss - request will be proxied")

		// Set X-Cache: MISS header NOW (before proxy runs)
		// This must be done in BeforeRequest because proxy writes headers during ServeHTTP
		ctx.Response.Header().Set("X-Cache", "MISS")

		// Add debug headers if enabled
		if p.config.AddDebugHeaders {
			ctx.Response.Header().Set("X-Cache-Key", cacheKey)
		}

		return nil
	}

	// Cache hit! Return cached response
	log.Info().
		Str("component", "plugin").
		Str("plugin", "cache").
		Str("cache_key", cacheKey).
		Int("status_code", cached.StatusCode).
		Int64("age_seconds", int64(cached.Age().Seconds())).
		Int("remaining_ttl", int(cached.RemainingTTL().Seconds())).
		Msg("Cache hit - serving cached response")

	// Decode body
	body, err := cached.DecodeBody()
	if err != nil {
		log.Warn().
			Err(err).
			Str("cache_key", cacheKey).
			Msg("Failed to decode cached body")
		// Continue to backend on decode error
		return nil
	}

	// Write cached response headers
	for key, value := range cached.Headers {
		ctx.Response.Header().Set(key, value)
	}

	// Add cache headers
	ctx.Response.Header().Set("X-Cache", "HIT")
	ctx.Response.Header().Set("X-Cache-TTL", strconv.Itoa(int(cached.RemainingTTL().Seconds())))
	ctx.Response.Header().Set("Age", strconv.Itoa(int(cached.Age().Seconds())))

	// Add debug headers if enabled
	if p.config.AddDebugHeaders && keyComponents != nil {
		ctx.Response.Header().Set("X-Cache-Key", cacheKey)
	}

	// Write status code and body
	ctx.Response.WriteHeader(cached.StatusCode)
	_, _ = ctx.Response.Write(body)

	// Abort the chain - we've served the cached response
	// Use a special status to indicate cache hit (won't send error response)
	ctx.Abort(0, "cache_hit")

	return nil
}

// handleAfterResponse stores cacheable responses in the cache.
func (p *CachePlugin) handleAfterResponse(ctx *plugin.Context) error {
	// Check if caching is enabled
	if p.cacheManager == nil || !p.cacheManager.IsEnabled() {
		return nil
	}

	// Check if we should skip caching this response
	if val, exists := ctx.Get(cacheSkipCtxKey); exists {
		if skip, ok := val.(bool); ok && skip {
			return nil
		}
	}

	// Get cache key from context
	cacheKey := ctx.GetString(cacheKeyCtxKey)
	if cacheKey == "" {
		return nil
	}

	// Get response data
	statusCode := ctx.Response.StatusCode()
	responseBody := ctx.Response.Body()
	responseHeaders := ctx.Response.Header()

	// Check if status code is cacheable
	if !p.isCacheableStatus(statusCode) {
		// Check if we should cache errors
		if p.config.CacheErrors && statusCode >= 400 {
			return p.cacheResponse(ctx, cacheKey, statusCode, responseBody, responseHeaders,
				time.Duration(p.config.ErrorTTLSeconds)*time.Second)
		}

		log.Debug().
			Int("status_code", statusCode).
			Str("cache_key", cacheKey).
			Msg("Status code not cacheable")

		ctx.Response.Header().Set("X-Cache", "BYPASS")
		return nil
	}

	// Check Cache-Control from backend
	if p.config.RespectNoCache {
		cacheControl := responseHeaders.Get("Cache-Control")
		if strings.Contains(cacheControl, "no-store") || strings.Contains(cacheControl, "no-cache") {
			log.Debug().
				Str("cache_control", cacheControl).
				Str("cache_key", cacheKey).
				Msg("Backend requested no caching")

			ctx.Response.Header().Set("X-Cache", "BYPASS")
			return nil
		}
	}

	// Determine TTL
	ttl := p.determineTTL(responseHeaders)

	// Cache the response
	return p.cacheResponse(ctx, cacheKey, statusCode, responseBody, responseHeaders, ttl)
}

// cacheResponse stores a response in the cache.
func (p *CachePlugin) cacheResponse(
	ctx *plugin.Context,
	cacheKey string,
	statusCode int,
	body []byte,
	headers http.Header,
	ttl time.Duration,
) error {
	// Check body size
	if len(body) > p.config.MaxBodySizeBytes {
		log.Debug().
			Int("body_size", len(body)).
			Int("max_size", p.config.MaxBodySizeBytes).
			Str("cache_key", cacheKey).
			Msg("Response body too large to cache")

		ctx.Response.Header().Set("X-Cache", "BYPASS")
		return nil
	}

	// Get metadata
	consumerID := ctx.GetString("consumer_id")

	routeID := ""
	if ctx.Route != nil {
		routeID = ctx.Route.ID
	}
	if routeID == "" {
		routeID = ctx.GetString("route_id")
	}

	// Calculate backend latency
	var backendLatency int64
	if val, exists := ctx.Get(cacheStartTimeCtxKey); exists {
		if startTime, ok := val.(time.Time); ok {
			backendLatency = time.Since(startTime).Milliseconds()
		}
	}

	// Create cached response
	cachedResponse := p.cacheManager.CreateCachedResponse(
		statusCode,
		headers,
		body,
		cache.CacheMetadata{
			RouteID:          routeID,
			BackendLatencyMs: backendLatency,
			ContentLength:    len(body),
			ConsumerID:       consumerID,
		},
	)

	if cachedResponse == nil {
		ctx.Response.Header().Set("X-Cache", "BYPASS")
		return nil
	}

	// Store in cache
	if err := p.cacheManager.Set(ctx.Request.Context(), cacheKey, cachedResponse, ttl); err != nil {
		log.Warn().
			Err(err).
			Str("cache_key", cacheKey).
			Msg("Failed to cache response")

		ctx.Response.Header().Set("X-Cache", "MISS")
		return nil
	}

	log.Info().
		Str("component", "plugin").
		Str("plugin", "cache").
		Str("cache_key", cacheKey).
		Int("status_code", statusCode).
		Int("body_size", len(body)).
		Dur("ttl", ttl).
		Int64("backend_latency_ms", backendLatency).
		Msg("Response cached successfully")

	// Add cache headers
	ctx.Response.Header().Set("X-Cache", "MISS")
	ctx.Response.Header().Set("X-Cache-TTL", strconv.Itoa(int(ttl.Seconds())))

	return nil
}

// determineTTL calculates the TTL for a response.
func (p *CachePlugin) determineTTL(headers http.Header) time.Duration {
	defaultTTL := time.Duration(p.config.TTLSeconds) * time.Second
	maxTTL := time.Duration(p.config.MaxTTLSeconds) * time.Second

	if !p.config.RespectCacheControl {
		return defaultTTL
	}

	cacheControl := headers.Get("Cache-Control")
	if cacheControl == "" {
		return defaultTTL
	}

	// Parse max-age directive
	for _, directive := range strings.Split(cacheControl, ",") {
		directive = strings.TrimSpace(directive)
		if strings.HasPrefix(directive, "max-age=") {
			ageStr := strings.TrimPrefix(directive, "max-age=")
			if age, err := strconv.Atoi(ageStr); err == nil && age > 0 {
				ttl := time.Duration(age) * time.Second
				if ttl > maxTTL {
					ttl = maxTTL
				}
				log.Debug().
					Int("max_age", age).
					Dur("ttl", ttl).
					Msg("Using TTL from Cache-Control header")
				return ttl
			}
		}
	}

	return defaultTTL
}

// isCacheableMethod checks if the HTTP method should be cached.
func (p *CachePlugin) isCacheableMethod(method string) bool {
	for _, m := range p.config.CacheMethods {
		if strings.EqualFold(m, method) {
			return true
		}
	}
	return false
}

// isCacheableStatus checks if the status code should be cached.
func (p *CachePlugin) isCacheableStatus(status int) bool {
	for _, s := range p.config.CacheStatusCodes {
		if s == status {
			return true
		}
	}
	return false
}

// shouldBypass checks if the path should bypass caching.
func (p *CachePlugin) shouldBypass(path string) bool {
	for _, bypassPath := range p.config.BypassPaths {
		if path == bypassPath || strings.HasPrefix(path, bypassPath+"/") {
			return true
		}
	}
	return false
}
