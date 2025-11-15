// Package router - Path matching logic
//
// This file implements path matching with support for:
//   - Exact paths: /api/users
//   - Parameters: /api/users/:id
//   - Wildcards: /api/users/*
package router

import (
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/saidutt46/switchboard-gateway/internal/database"
)

// PathMatch represents a matched path with extracted parameters.
type PathMatch struct {
	Route  *database.Route
	Params map[string]string // Extracted path parameters
}

// Matcher handles path matching for routes.
type Matcher struct {
	routes []*database.Route
}

// NewMatcher creates a new path matcher.
func NewMatcher() *Matcher {
	return &Matcher{
		routes: make([]*database.Route, 0),
	}
}

// AddRoute adds a route to the matcher.
func (m *Matcher) AddRoute(route *database.Route) {
	m.routes = append(m.routes, route)
}

// Match finds all routes that match the given path.
//
// Returns matches in priority order:
//  1. Exact matches
//  2. Parameter matches (/:id)
//  3. Wildcard matches (/*)
//
// Within each priority level, longer paths take precedence.
func (m *Matcher) Match(path string) []*PathMatch {
	var exactMatches []*PathMatch
	var paramMatches []*PathMatch
	var wildcardMatches []*PathMatch

	log.Debug().
		Str("component", "matcher").
		Str("path", path).
		Int("candidates", len(m.routes)).
		Msg("Starting path match")

	// Try to match against each route
	for _, route := range m.routes {
		// Skip disabled routes
		if !route.Enabled {
			continue
		}

		// Try each path pattern in the route
		for _, pattern := range route.Paths {
			if match := m.matchPattern(path, pattern); match != nil {
				match.Route = route

				// Categorize by match type
				if m.isExactMatch(pattern) {
					log.Debug().
						Str("component", "matcher").
						Str("path", path).
						Str("pattern", pattern).
						Msg("Path matched with parameters")
					exactMatches = append(exactMatches, match)
				} else if m.hasParameters(pattern) {
					paramMatches = append(paramMatches, match)
				} else {
					wildcardMatches = append(wildcardMatches, match)
				}
			}
		}
	}

	// Return in priority order
	result := make([]*PathMatch, 0)
	result = append(result, exactMatches...)
	result = append(result, paramMatches...)
	result = append(result, wildcardMatches...)

	return result
}

// matchPattern checks if a path matches a pattern and extracts parameters.
//
// Pattern syntax:
//   - Exact: /api/users
//   - Parameter: /api/users/:id
//   - Wildcard: /api/users/* (matches /api/users/anything but NOT /api/users)
func (m *Matcher) matchPattern(path, pattern string) *PathMatch {
	// Normalize paths (remove trailing slash)
	path = strings.TrimSuffix(path, "/")
	pattern = strings.TrimSuffix(pattern, "/")

	// Empty paths match
	if path == "" && pattern == "" {
		return &PathMatch{Params: make(map[string]string)}
	}

	pathSegments := strings.Split(strings.Trim(path, "/"), "/")
	patternSegments := strings.Split(strings.Trim(pattern, "/"), "/")

	params := make(map[string]string)

	// Check if pattern ends with wildcard
	hasWildcard := len(patternSegments) > 0 && patternSegments[len(patternSegments)-1] == "*"

	// Without wildcard, lengths must match exactly
	if !hasWildcard && len(pathSegments) != len(patternSegments) {
		return nil
	}

	// With wildcard, path must have at least as many segments as pattern
	// (the wildcard needs at least one segment to match)
	if hasWildcard && len(pathSegments) < len(patternSegments) {
		return nil
	}

	// Match each segment
	maxSegments := len(patternSegments)
	if hasWildcard {
		maxSegments-- // Don't process the wildcard itself
	}

	for i := 0; i < maxSegments; i++ {
		patternSeg := patternSegments[i]

		// Path ended but pattern continues (shouldn't happen due to check above)
		if i >= len(pathSegments) {
			return nil
		}

		pathSeg := pathSegments[i]

		// Parameter segment (e.g., :id)
		if strings.HasPrefix(patternSeg, ":") {
			paramName := patternSeg[1:] // Remove ":"
			params[paramName] = pathSeg
			continue
		}

		// Exact match required
		if pathSeg != patternSeg {
			return nil
		}
	}

	// If pattern has wildcard, we matched successfully
	// (we already verified path has enough segments above)
	if hasWildcard {
		return &PathMatch{Params: params}
	}

	// For non-wildcard patterns, all segments must be consumed
	return &PathMatch{Params: params}
}

// isExactMatch returns true if the pattern is an exact match (no params or wildcards).
func (m *Matcher) isExactMatch(pattern string) bool {
	return !strings.Contains(pattern, ":") && !strings.Contains(pattern, "*")
}

// hasParameters returns true if the pattern has path parameters.
func (m *Matcher) hasParameters(pattern string) bool {
	return strings.Contains(pattern, ":")
}

// hasWildcard returns true if the pattern has a wildcard.
func (m *Matcher) hasWildcard(pattern string) bool {
	return strings.Contains(pattern, "*")
}
