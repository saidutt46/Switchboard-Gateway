// Package cache provides response caching functionality for the API Gateway.
//
// Cache keys are generated based on request attributes and configuration,
// allowing for flexible caching strategies including:
//   - Public caching (shared across all consumers)
//   - Per-consumer caching (private responses)
//   - Vary by query parameters
//   - Vary by request headers
//
// Key Format:
//
//	cache:{route_id}:{method}:{path}:{query_hash}:{consumer_id}
package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
)

// KeyBuilder generates cache keys from HTTP requests.
//
// Cache keys uniquely identify cacheable responses based on
// request attributes and caching configuration.
type KeyBuilder struct {
	// Prefix for all cache keys (default: "cache:")
	Prefix string

	// VaryByConsumer includes consumer ID in cache key
	// When true, each consumer gets their own cached response
	VaryByConsumer bool

	// VaryByQuery includes query parameters in cache key
	// When true, different query params = different cache entries
	VaryByQuery bool

	// VaryByHeaders includes specific header values in cache key
	// Example: ["Accept-Language", "Accept-Encoding"]
	VaryByHeaders []string

	// IgnoreQueryParams lists query params to exclude from key
	// Useful for tracking params like utm_source
	IgnoreQueryParams []string
}

// DefaultKeyBuilder returns a KeyBuilder with sensible defaults.
func DefaultKeyBuilder() *KeyBuilder {
	return &KeyBuilder{
		Prefix:            "cache:",
		VaryByConsumer:    false,
		VaryByQuery:       true,
		VaryByHeaders:     []string{},
		IgnoreQueryParams: []string{},
	}
}

// KeyComponents holds the components used to build a cache key.
// Exposed for debugging and cache invalidation purposes.
type KeyComponents struct {
	Prefix     string
	RouteID    string
	Method     string
	Path       string
	QueryHash  string
	ConsumerID string
	HeaderHash string
}

// String returns the full cache key.
func (kc *KeyComponents) String() string {
	parts := []string{
		kc.Prefix + kc.RouteID,
		kc.Method,
		kc.Path,
	}

	if kc.QueryHash != "" {
		parts = append(parts, kc.QueryHash)
	}

	if kc.ConsumerID != "" {
		parts = append(parts, kc.ConsumerID)
	}

	if kc.HeaderHash != "" {
		parts = append(parts, kc.HeaderHash)
	}

	return strings.Join(parts, ":")
}

// BuildKey generates a cache key for the given request.
//
// Parameters:
//   - r: The HTTP request
//   - routeID: The matched route's ID
//   - consumerID: The authenticated consumer's ID (empty if anonymous)
//
// Returns the cache key string and its components.
func (kb *KeyBuilder) BuildKey(r *http.Request, routeID string, consumerID string) (string, *KeyComponents) {
	components := &KeyComponents{
		Prefix:  kb.Prefix,
		RouteID: routeID,
		Method:  r.Method,
		Path:    normalizePath(r.URL.Path),
	}

	// Include query parameters hash if configured
	if kb.VaryByQuery && r.URL.RawQuery != "" {
		components.QueryHash = kb.hashQueryParams(r.URL.Query())
	}

	// Include consumer ID if configured
	if kb.VaryByConsumer {
		if consumerID != "" {
			components.ConsumerID = consumerID
		} else {
			// Anonymous/public consumer
			components.ConsumerID = "public"
		}
	}

	// Include header hash if configured
	if len(kb.VaryByHeaders) > 0 {
		components.HeaderHash = kb.hashHeaders(r, kb.VaryByHeaders)
	}

	return components.String(), components
}

// BuildKeySimple generates a simple cache key without all options.
// Useful for cache invalidation by route.
func (kb *KeyBuilder) BuildKeySimple(routeID, method, path string) string {
	return fmt.Sprintf("%s%s:%s:%s", kb.Prefix, routeID, method, normalizePath(path))
}

// BuildInvalidationPattern generates a pattern for cache invalidation.
// Returns a Redis SCAN pattern that matches all keys for a route.
//
// Examples:
//   - InvalidateRoute("route-123") → "cache:route-123:*"
//   - InvalidateRouteMethod("route-123", "GET") → "cache:route-123:GET:*"
func (kb *KeyBuilder) BuildInvalidationPattern(routeID string, method string) string {
	if method == "" {
		return fmt.Sprintf("%s%s:*", kb.Prefix, routeID)
	}
	return fmt.Sprintf("%s%s:%s:*", kb.Prefix, routeID, method)
}

// hashQueryParams creates a deterministic hash of query parameters.
// Parameters are sorted to ensure consistent hashing regardless of order.
func (kb *KeyBuilder) hashQueryParams(params url.Values) string {
	if len(params) == 0 {
		return ""
	}

	// Get sorted keys (excluding ignored params)
	keys := make([]string, 0, len(params))
	for key := range params {
		if !kb.isIgnoredParam(key) {
			keys = append(keys, key)
		}
	}

	if len(keys) == 0 {
		return ""
	}

	sort.Strings(keys)

	// Build canonical query string
	var sb strings.Builder
	for i, key := range keys {
		if i > 0 {
			sb.WriteByte('&')
		}
		values := params[key]
		sort.Strings(values) // Sort values for consistency
		for j, val := range values {
			if j > 0 {
				sb.WriteByte('&')
			}
			sb.WriteString(url.QueryEscape(key))
			sb.WriteByte('=')
			sb.WriteString(url.QueryEscape(val))
		}
	}

	// Return short hash (first 16 chars of SHA256)
	return hashString(sb.String())[:16]
}

// hashHeaders creates a deterministic hash of specified header values.
func (kb *KeyBuilder) hashHeaders(r *http.Request, headers []string) string {
	if len(headers) == 0 {
		return ""
	}

	var sb strings.Builder
	for i, header := range headers {
		if i > 0 {
			sb.WriteByte('|')
		}
		sb.WriteString(header)
		sb.WriteByte(':')
		sb.WriteString(r.Header.Get(header))
	}

	// Return short hash
	return hashString(sb.String())[:16]
}

// isIgnoredParam checks if a query parameter should be ignored.
func (kb *KeyBuilder) isIgnoredParam(param string) bool {
	for _, ignored := range kb.IgnoreQueryParams {
		if strings.EqualFold(param, ignored) {
			return true
		}
	}
	return false
}

// normalizePath ensures consistent path formatting.
func normalizePath(path string) string {
	// Remove trailing slash (except for root)
	if len(path) > 1 && strings.HasSuffix(path, "/") {
		path = strings.TrimSuffix(path, "/")
	}

	// Ensure leading slash
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	return path
}

// hashString returns SHA256 hash of a string.
func hashString(s string) string {
	hash := sha256.Sum256([]byte(s))
	return hex.EncodeToString(hash[:])
}
