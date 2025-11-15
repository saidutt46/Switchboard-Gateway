package router

import (
	"testing"

	"github.com/saidutt46/switchboard-gateway/internal/database"
)

func TestMatcher_ExactMatch(t *testing.T) {
	matcher := NewMatcher()
	route := &database.Route{
		ID:      "route-1",
		Paths:   []string{"/api/users"},
		Enabled: true,
	}
	matcher.AddRoute(route)

	tests := []struct {
		name      string
		path      string
		wantMatch bool
	}{
		{
			name:      "exact match",
			path:      "/api/users",
			wantMatch: true,
		},
		{
			name:      "exact match with trailing slash",
			path:      "/api/users/",
			wantMatch: true,
		},
		{
			name:      "no match - different path",
			path:      "/api/products",
			wantMatch: false,
		},
		{
			name:      "no match - longer path",
			path:      "/api/users/123",
			wantMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := matcher.Match(tt.path)
			gotMatch := len(matches) > 0

			if gotMatch != tt.wantMatch {
				t.Errorf("Match() = %v, want %v", gotMatch, tt.wantMatch)
			}
		})
	}
}

func TestMatcher_ParameterMatch(t *testing.T) {
	matcher := NewMatcher()
	route := &database.Route{
		ID:      "route-1",
		Paths:   []string{"/api/users/:id"},
		Enabled: true,
	}
	matcher.AddRoute(route)

	tests := []struct {
		name       string
		path       string
		wantMatch  bool
		wantParams map[string]string
	}{
		{
			name:      "parameter match",
			path:      "/api/users/123",
			wantMatch: true,
			wantParams: map[string]string{
				"id": "123",
			},
		},
		{
			name:      "parameter match - uuid",
			path:      "/api/users/550e8400-e29b-41d4-a716-446655440000",
			wantMatch: true,
			wantParams: map[string]string{
				"id": "550e8400-e29b-41d4-a716-446655440000",
			},
		},
		{
			name:      "no match - too short",
			path:      "/api/users",
			wantMatch: false,
		},
		{
			name:      "no match - too long",
			path:      "/api/users/123/posts",
			wantMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := matcher.Match(tt.path)
			gotMatch := len(matches) > 0

			if gotMatch != tt.wantMatch {
				t.Errorf("Match() = %v, want %v", gotMatch, tt.wantMatch)
				return
			}

			if tt.wantMatch && len(matches) > 0 {
				params := matches[0].Params
				for key, expectedValue := range tt.wantParams {
					if params[key] != expectedValue {
						t.Errorf("Param %s = %v, want %v", key, params[key], expectedValue)
					}
				}
			}
		})
	}
}

func TestMatcher_WildcardMatch(t *testing.T) {
	matcher := NewMatcher()
	route := &database.Route{
		ID:      "route-1",
		Paths:   []string{"/api/users/*"},
		Enabled: true,
	}
	matcher.AddRoute(route)

	tests := []struct {
		name      string
		path      string
		wantMatch bool
	}{
		{
			name:      "wildcard match - one segment",
			path:      "/api/users/123",
			wantMatch: true,
		},
		{
			name:      "wildcard match - multiple segments",
			path:      "/api/users/123/posts/456",
			wantMatch: true,
		},
		{
			name:      "no match - too short",
			path:      "/api/users",
			wantMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := matcher.Match(tt.path)
			gotMatch := len(matches) > 0

			if gotMatch != tt.wantMatch {
				t.Errorf("Match() = %v, want %v", gotMatch, tt.wantMatch)
			}
		})
	}
}

func TestMatcher_Priority(t *testing.T) {
	matcher := NewMatcher()

	// Add routes in random order
	exactRoute := &database.Route{
		ID:      "exact",
		Paths:   []string{"/api/users/profile"},
		Enabled: true,
	}
	paramRoute := &database.Route{
		ID:      "param",
		Paths:   []string{"/api/users/:id"},
		Enabled: true,
	}
	wildcardRoute := &database.Route{
		ID:      "wildcard",
		Paths:   []string{"/api/users/*"},
		Enabled: true,
	}

	matcher.AddRoute(wildcardRoute)
	matcher.AddRoute(paramRoute)
	matcher.AddRoute(exactRoute)

	// Test that exact match has priority
	matches := matcher.Match("/api/users/profile")
	if len(matches) == 0 {
		t.Fatal("Expected matches")
	}

	// First match should be exact
	if matches[0].Route.ID != "exact" {
		t.Errorf("Expected exact route first, got %s", matches[0].Route.ID)
	}
}

func TestMatcher_MultipleParameters(t *testing.T) {
	matcher := NewMatcher()
	route := &database.Route{
		ID:      "route-1",
		Paths:   []string{"/api/users/:userId/posts/:postId"},
		Enabled: true,
	}
	matcher.AddRoute(route)

	matches := matcher.Match("/api/users/123/posts/456")
	if len(matches) == 0 {
		t.Fatal("Expected match")
	}

	params := matches[0].Params
	if params["userId"] != "123" {
		t.Errorf("userId = %v, want 123", params["userId"])
	}
	if params["postId"] != "456" {
		t.Errorf("postId = %v, want 456", params["postId"])
	}
}
