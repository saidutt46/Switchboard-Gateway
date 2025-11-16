// Package router - Radix tree implementation for efficient path matching
//
// A radix tree (compressed trie) provides O(log n) path lookups.
// It's specifically designed for routing with support for:
//   - Exact paths: /api/users
//   - Parameters: /api/users/:id
//   - Wildcards: /api/users/*
package router

import (
	"strings"
	"sync"

	"github.com/rs/zerolog/log"
	"github.com/saidutt46/switchboard-gateway/internal/database"
)

// nodeType represents the type of radix tree node
type nodeType uint8

const (
	static   nodeType = iota // Normal path segment: /api/users
	param                    // Parameter segment: /:id
	wildcard                 // Wildcard segment: /*
)

// node represents a single node in the radix tree
type node struct {
	// Node properties
	nType    nodeType
	label    string          // Path segment label
	prefix   string          // Common prefix for this node
	children []*node         // Child nodes
	route    *database.Route // Route if this is a leaf node
	priority uint32          // Priority for sorting (higher = checked first)

	// Parameter handling
	paramName string // Name of parameter if nType == param (e.g., "id" from ":id")
}

// RadixTree is a thread-safe radix tree for route matching
type RadixTree struct {
	root *node
	mu   sync.RWMutex
	size int
}

// NewRadixTree creates a new empty radix tree
func NewRadixTree() *RadixTree {
	return &RadixTree{
		root: &node{
			nType:    static,
			children: make([]*node, 0),
		},
		size: 0,
	}
}

// Insert adds a route to the radix tree
//
// Example:
//
//	tree.Insert("/api/users", route)
//	tree.Insert("/api/users/:id", route)
//	tree.Insert("/api/products/*", route)
func (t *RadixTree) Insert(path string, route *database.Route) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Normalize path
	path = normalizePath(path)

	log.Debug().
		Str("component", "radix_tree").
		Str("path", path).
		Str("route_id", route.ID).
		Msg("Inserting route into radix tree")

	// Split path into segments
	segments := splitPath(path)

	// Insert from root
	current := t.root

	for i, segment := range segments {
		// Determine segment type
		segType, paramName := getSegmentType(segment)

		// Look for existing child with matching prefix
		child := t.findChild(current, segment, segType)

		if child != nil {
			// Child exists, continue down the tree
			current = child
		} else {
			// Create new child
			newNode := &node{
				nType:     segType,
				label:     segment,
				prefix:    segment,
				children:  make([]*node, 0),
				paramName: paramName,
				priority:  uint32(len(segments) - i), // Longer paths have higher priority
			}

			current.children = append(current.children, newNode)

			// Sort children by priority (static > param > wildcard)
			t.sortChildren(current)

			current = newNode
		}
	}

	// Set route at leaf node
	current.route = route
	t.size++

	log.Debug().
		Str("component", "radix_tree").
		Str("path", path).
		Int("tree_size", t.size).
		Msg("Route inserted successfully")
}

// Search finds a route matching the given path
//
// Returns the route and extracted parameters.
// Example:
//
//	route, params := tree.Search("/api/users/123")
//	// params = {"id": "123"}
func (t *RadixTree) Search(path string) (*database.Route, map[string]string) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	// Normalize path
	path = normalizePath(path)

	log.Debug().
		Str("component", "radix_tree").
		Str("path", path).
		Msg("Searching for route")

	// Split path into segments
	segments := splitPath(path)
	params := make(map[string]string)

	// Search from root
	route := t.search(t.root, segments, 0, params)

	if route != nil {
		log.Debug().
			Str("component", "radix_tree").
			Str("path", path).
			Str("route_id", route.ID).
			Interface("params", params).
			Msg("Route found")
	} else {
		log.Debug().
			Str("component", "radix_tree").
			Str("path", path).
			Msg("No route found")
	}

	return route, params
}

// search recursively searches the tree
func (t *RadixTree) search(n *node, segments []string, index int, params map[string]string) *database.Route {
	// Reached end of path
	if index >= len(segments) {
		return n.route
	}

	segment := segments[index]

	// Try children in priority order (static > param > wildcard)
	for _, child := range n.children {
		switch child.nType {
		case static:
			// Exact match required
			if child.label == segment {
				if route := t.search(child, segments, index+1, params); route != nil {
					return route
				}
			}

		case param:
			// Parameter matches any segment
			params[child.paramName] = segment
			if route := t.search(child, segments, index+1, params); route != nil {
				return route
			}
			// Backtrack: remove param if this path didn't work
			delete(params, child.paramName)

		case wildcard:
			// Wildcard matches remaining path
			if child.route != nil {
				// Capture remaining path
				remaining := strings.Join(segments[index:], "/")
				params["*"] = remaining
				return child.route
			}
		}
	}

	return nil
}

// findChild looks for a child node matching the segment
func (t *RadixTree) findChild(n *node, segment string, segType nodeType) *node {
	for _, child := range n.children {
		if child.nType == segType && child.label == segment {
			return child
		}
	}
	return nil
}

// sortChildren sorts children by priority (static > param > wildcard)
func (t *RadixTree) sortChildren(n *node) {
	// Bubble sort (small arrays, simple is fine)
	for i := 0; i < len(n.children); i++ {
		for j := i + 1; j < len(n.children); j++ {
			if t.nodePriority(n.children[j]) > t.nodePriority(n.children[i]) {
				n.children[i], n.children[j] = n.children[j], n.children[i]
			}
		}
	}
}

// nodePriority returns priority value for sorting
// Static (100) > Param (50) > Wildcard (1)
func (t *RadixTree) nodePriority(n *node) int {
	switch n.nType {
	case static:
		return 100 + int(n.priority)
	case param:
		return 50 + int(n.priority)
	case wildcard:
		return 1
	default:
		return 0
	}
}

// Size returns the number of routes in the tree
func (t *RadixTree) Size() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.size
}

// Clear removes all routes from the tree
func (t *RadixTree) Clear() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.root = &node{
		nType:    static,
		children: make([]*node, 0),
	}
	t.size = 0

	log.Debug().
		Str("component", "radix_tree").
		Msg("Radix tree cleared")
}

// Helper functions

// normalizePath normalizes a URL path
func normalizePath(path string) string {
	// Remove trailing slash (except for root)
	if len(path) > 1 && strings.HasSuffix(path, "/") {
		path = path[:len(path)-1]
	}

	// Ensure leading slash
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	return path
}

// splitPath splits a path into segments
func splitPath(path string) []string {
	// Remove leading/trailing slashes
	path = strings.Trim(path, "/")

	// Empty path = root
	if path == "" {
		return []string{}
	}

	return strings.Split(path, "/")
}

// getSegmentType determines the type of a path segment
// Returns (type, paramName)
func getSegmentType(segment string) (nodeType, string) {
	if len(segment) == 0 {
		return static, ""
	}

	// Wildcard: *
	if segment == "*" {
		return wildcard, ""
	}

	// Parameter: :name
	if segment[0] == ':' {
		return param, segment[1:] // Remove ':' prefix
	}

	// Static segment
	return static, ""
}
