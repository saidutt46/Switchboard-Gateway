// Package router provides HTTP request routing functionality.
//
// The router matches incoming HTTP requests to configured routes based on:
//   - Request path (with support for parameters and wildcards)
//   - HTTP method
//   - Host header (optional)
//
// Routes are loaded from the database into memory at startup for
// fast lookups (< 0.1ms per request).
package router

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/rs/zerolog/log"

	"github.com/saidutt46/switchboard-gateway/internal/database"
)

// Router handles request routing to backend services.
type Router struct {
	routes   []*database.Route
	services map[string]*database.Service // service_id -> Service
	matcher  *Matcher
	mu       sync.RWMutex // Protects routes, services, and matcher during reload
}

// MatchResult contains the result of a route match.
type MatchResult struct {
	Route      *database.Route
	Service    *database.Service
	PathParams map[string]string // Extracted path parameters (e.g., {"id": "123"})
}

// NewRouter creates a new router from database routes and services.
//
// Routes and services are loaded into memory for fast matching.
// Uses a radix tree for O(log n) route lookups.
// This should be called once at startup.
func NewRouter(routes []*database.Route, services []*database.Service) *Router {
	// Build service map for fast lookups
	serviceMap := make(map[string]*database.Service)
	for _, svc := range services {
		serviceMap[svc.ID] = svc
	}

	// Create matcher with radix tree
	matcher := NewMatcher()

	// Insert all routes into radix tree
	enabledCount := 0
	for _, route := range routes {
		if route.Enabled {
			matcher.AddRoute(route)
			enabledCount++
		}
	}

	log.Info().
		Str("component", "router").
		Int("routes", len(routes)).
		Int("enabled_routes", enabledCount).
		Int("services", len(services)).
		Int("tree_size", matcher.Size()).
		Msg("Router initialized with radix tree")

	return &Router{
		routes:   routes,
		services: serviceMap,
		matcher:  matcher,
	}
}

// Match finds a route that matches the given HTTP request.
//
// Matching is done based on:
//  1. Path matching (exact, parameter, wildcard)
//  2. HTTP method
//  3. Host header (if route specifies hosts)
//
// Returns the matched route, service, and extracted path parameters.
// Returns nil if no route matches.
// Match finds a route that matches the given HTTP request.
func (r *Router) Match(req *http.Request) (*MatchResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	path := req.URL.Path
	method := req.Method
	host := req.Host

	log.Debug().
		Str("component", "router").
		Str("path", path).
		Str("method", method).
		Str("host", host).
		Msg("Matching request")

	// Find matching routes by path
	matches := r.matcher.Match(path)
	if len(matches) == 0 {
		log.Debug().
			Str("component", "router").
			Str("path", path).
			Msg("No routes matched path")
		return nil, fmt.Errorf("no route found for path: %s", path)
	}

	// Filter by method and host
	for _, match := range matches {
		route := match.Route

		// Check if method is allowed
		if !r.methodAllowed(route, method) {
			continue
		}

		// Check if host matches (if route specifies hosts)
		if !r.hostMatches(route, host) {
			continue
		}

		// Get the service for this route
		service, ok := r.services[route.ServiceID]
		if !ok {
			log.Warn().
				Str("component", "router").
				Str("route_id", route.ID).
				Str("service_id", route.ServiceID).
				Msg("Service not found for route")
			continue
		}

		// Check if service is enabled
		if !service.Enabled {
			log.Debug().
				Str("component", "router").
				Str("service_id", service.ID).
				Msg("Service is disabled")
			continue
		}

		log.Info().
			Str("component", "router").
			Str("route_id", route.ID).
			Str("route_name", route.Name.String).
			Str("service_id", service.ID).
			Str("service_name", service.Name).
			Str("path", path).
			Msg("Route matched")

		return &MatchResult{
			Route:      route,
			Service:    service,
			PathParams: match.Params,
		}, nil
	}

	log.Debug().
		Str("component", "router").
		Str("path", path).
		Str("method", method).
		Msg("No routes matched after filtering")

	return nil, fmt.Errorf("no route found for %s %s", method, path)
}

// methodAllowed checks if the HTTP method is allowed for the route.
func (r *Router) methodAllowed(route *database.Route, method string) bool {
	// If no methods specified, allow all
	if len(route.Methods) == 0 {
		return true
	}

	// Check if method is in the allowed list
	for _, m := range route.Methods {
		if m == method {
			return true
		}
	}

	return false
}

// hostMatches checks if the request host matches the route's host requirements.
func (r *Router) hostMatches(route *database.Route, requestHost string) bool {
	// If no hosts specified, match any host
	if len(route.Hosts) == 0 {
		return true
	}

	// Strip port from request host if present
	host := requestHost
	if colonIdx := strings.Index(host, ":"); colonIdx != -1 {
		host = host[:colonIdx]
	}

	// Check each host pattern
	for _, pattern := range route.Hosts {
		if r.hostMatchesPattern(host, pattern) {
			return true
		}
	}

	return false
}

// hostMatchesPattern checks if a host matches a pattern.
// Supports wildcard patterns like "*.example.com"
func (r *Router) hostMatchesPattern(host, pattern string) bool {
	// Exact match
	if host == pattern {
		return true
	}

	// Wildcard match (e.g., "*.example.com")
	if strings.HasPrefix(pattern, "*.") {
		suffix := pattern[2:] // Remove "*."
		return strings.HasSuffix(host, "."+suffix) || host == suffix
	}

	return false
}

// Reload reloads routes from the database.
//
// This is called when routes are updated via the Admin API.
// Rebuilds the radix tree with the new routes.
// It's safe to call concurrently - uses write lock for atomic swap.
func (r *Router) Reload(ctx context.Context, repo *database.Repository) error {
	log.Info().
		Str("component", "router").
		Msg("Reloading routes from database")

	// Load routes from database
	routes, err := repo.GetRoutes(ctx, false) // Only enabled routes
	if err != nil {
		return fmt.Errorf("failed to load routes: %w", err)
	}

	// Load services
	services, err := repo.GetServices(ctx, false)
	if err != nil {
		return fmt.Errorf("failed to load services: %w", err)
	}

	// Build new service map
	serviceMap := make(map[string]*database.Service)
	for _, svc := range services {
		serviceMap[svc.ID] = svc
	}

	// Create new matcher with radix tree
	matcher := NewMatcher()

	// Build radix tree from routes
	enabledCount := 0
	totalPaths := 0
	for _, route := range routes {
		if route.Enabled {
			matcher.AddRoute(route)
			enabledCount++
			totalPaths += len(route.Paths)
		}
	}

	// Atomic swap (write lock in router)
	r.mu.Lock()
	r.routes = routes
	r.services = serviceMap
	r.matcher = matcher
	r.mu.Unlock()

	log.Info().
		Str("component", "router").
		Int("routes", len(routes)).
		Int("enabled_routes", enabledCount).
		Int("total_paths", totalPaths).
		Int("services", len(services)).
		Int("tree_size", matcher.Size()).
		Msg("Routes reloaded successfully - radix tree rebuilt")

	return nil
}

// Stats returns router statistics including radix tree metrics.
func (r *Router) Stats() map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return map[string]interface{}{
		"routes":        len(r.routes),
		"services":      len(r.services),
		"tree_size":     r.matcher.Size(),
		"lookup_method": "radix_tree",
		"complexity":    "O(log n)",
	}
}
