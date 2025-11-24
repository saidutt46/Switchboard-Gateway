// Package router - Path matching logic using radix tree
//
// This file implements path matching with support for:
//   - Exact paths: /api/users
//   - Parameters: /api/users/:id
//   - Wildcards: /api/users/*
//
// Uses a radix tree for O(log n) performance instead of O(n) linear search.
package router

import (
	"github.com/rs/zerolog/log"
	"github.com/saidutt46/switchboard-gateway/internal/database"
)

// PathMatch represents a matched path with extracted parameters.
type PathMatch struct {
	Route  *database.Route
	Params map[string]string // Extracted path parameters
}

// Matcher handles path matching for routes using a radix tree.
type Matcher struct {
	tree *RadixTree
}

// NewMatcher creates a new path matcher with an empty radix tree.
func NewMatcher() *Matcher {
	log.Debug().
		Str("component", "matcher").
		Msg("Creating new matcher with radix tree")

	return &Matcher{
		tree: NewRadixTree(),
	}
}

// AddRoute adds a route to the matcher.
//
// Each path in the route is inserted into the radix tree.
// Example:
//
//	route.Paths = ["/api/users", "/api/users/:id"]
//	Both paths will be inserted and point to the same route.
func (m *Matcher) AddRoute(route *database.Route) {
	if route == nil {
		log.Warn().
			Str("component", "matcher").
			Msg("Attempted to add nil route")
		return
	}

	if !route.Enabled {
		log.Debug().
			Str("component", "matcher").
			Str("route_id", route.ID).
			Msg("Skipping disabled route")
		return
	}

	// Insert each path pattern into the radix tree
	for _, pattern := range route.Paths {
		m.tree.Insert(pattern, route)

		log.Debug().
			Str("component", "matcher").
			Str("route_id", route.ID).
			Str("pattern", pattern).
			Int("tree_size", m.tree.Size()).
			Msg("Route path added to radix tree")
	}
}

// Match finds all routes that match the given path.
//
// With radix tree, we get the best match directly (O(log n)).
// Returns matches in priority order (most specific first).
//
// Example:
//
//	matches := matcher.Match("/api/users/123")
//	// Returns route for /api/users/:id with params={"id": "123"}
func (m *Matcher) Match(path string) []*PathMatch {
	log.Debug().
		Str("component", "matcher").
		Str("path", path).
		Msg("Matching path against radix tree")

	// Search the radix tree (O(log n))
	route, params := m.tree.Search(path)

	// No match found
	if route == nil {
		log.Debug().
			Str("component", "matcher").
			Str("path", path).
			Msg("No route matched in radix tree")
		return nil
	}

	// Check if route is still enabled (defensive check)
	if !route.Enabled {
		log.Debug().
			Str("component", "matcher").
			Str("path", path).
			Str("route_id", route.ID).
			Msg("Matched route is disabled")
		return nil
	}

	// Return single match (radix tree gives us the best match)
	match := &PathMatch{
		Route:  route,
		Params: params,
	}

	log.Debug().
		Str("component", "matcher").
		Str("path", path).
		Str("route_id", route.ID).
		Str("route_name", route.Name.String).
		Interface("params", params).
		Msg("Path matched successfully via radix tree")

	return []*PathMatch{match}
}

// Clear removes all routes from the matcher.
//
// This is useful when reloading all routes from the database.
func (m *Matcher) Clear() {
	log.Debug().
		Str("component", "matcher").
		Msg("Clearing all routes from radix tree")

	m.tree.Clear()
}

// Size returns the number of route paths in the tree.
func (m *Matcher) Size() int {
	return m.tree.Size()
}
