package router

import (
	"net/http/httptest"
	"testing"

	"github.com/saidutt46/switchboard-gateway/internal/database"
)

func TestRouter_MatchRequest(t *testing.T) {
	// Setup test data
	service := &database.Service{
		ID:       "test-service-id",
		Name:     "test-service",
		Protocol: "http",
		Host:     "localhost",
		Port:     8081,
		Enabled:  true,
	}

	route := &database.Route{
		ID:        "test-route-id",
		ServiceID: service.ID,
		Paths:     []string{"/api/users", "/api/users/:id"},
		Methods:   []string{"GET", "POST"},
		Enabled:   true,
	}

	// Create router
	r := NewRouter([]*database.Route{route}, []*database.Service{service})

	tests := []struct {
		name       string
		method     string
		path       string
		wantMatch  bool
		wantParams map[string]string
	}{
		{
			name:      "exact match",
			method:    "GET",
			path:      "/api/users",
			wantMatch: true,
		},
		{
			name:      "parameter match",
			method:    "GET",
			path:      "/api/users/123",
			wantMatch: true,
			wantParams: map[string]string{
				"id": "123",
			},
		},
		{
			name:      "method not allowed",
			method:    "DELETE",
			path:      "/api/users",
			wantMatch: false,
		},
		{
			name:      "path not found",
			method:    "GET",
			path:      "/api/products",
			wantMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			result, err := r.Match(req)

			gotMatch := err == nil
			if gotMatch != tt.wantMatch {
				t.Errorf("Match() match = %v, want %v (error: %v)", gotMatch, tt.wantMatch, err)
				return
			}

			if tt.wantMatch && tt.wantParams != nil {
				for key, want := range tt.wantParams {
					if got := result.PathParams[key]; got != want {
						t.Errorf("PathParams[%s] = %v, want %v", key, got, want)
					}
				}
			}
		})
	}
}
