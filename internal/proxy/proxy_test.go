package proxy

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/saidutt46/switchboard-gateway/internal/database"
	"github.com/saidutt46/switchboard-gateway/internal/router"
)

func TestProxy_GetClientIP(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		xff        string
		xri        string
		expectedIP string
	}{
		{
			name:       "from RemoteAddr",
			remoteAddr: "192.168.1.100:12345",
			expectedIP: "192.168.1.100",
		},
		{
			name:       "from X-Forwarded-For",
			remoteAddr: "10.0.0.1:12345",
			xff:        "203.0.113.1, 198.51.100.1",
			expectedIP: "203.0.113.1",
		},
		{
			name:       "from X-Real-IP",
			remoteAddr: "10.0.0.1:12345",
			xri:        "203.0.113.1",
			expectedIP: "203.0.113.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = tt.remoteAddr

			if tt.xff != "" {
				req.Header.Set("X-Forwarded-For", tt.xff)
			}
			if tt.xri != "" {
				req.Header.Set("X-Real-IP", tt.xri)
			}

			ip := getClientIP(req)
			if ip != tt.expectedIP {
				t.Errorf("getClientIP() = %v, want %v", ip, tt.expectedIP)
			}
		})
	}
}

func TestProxy_IsHopByHopHeader(t *testing.T) {
	tests := []struct {
		header string
		want   bool
	}{
		{"Connection", true},
		{"Keep-Alive", true},
		{"Transfer-Encoding", true},
		{"Content-Type", false},
		{"Authorization", false},
		{"X-Custom-Header", false},
	}

	for _, tt := range tests {
		t.Run(tt.header, func(t *testing.T) {
			if got := isHopByHopHeader(tt.header); got != tt.want {
				t.Errorf("isHopByHopHeader(%s) = %v, want %v", tt.header, got, tt.want)
			}
		})
	}
}

func TestProxy_GenerateRequestID(t *testing.T) {
	id1 := generateRequestID()
	time.Sleep(1 * time.Millisecond)
	id2 := generateRequestID()

	// IDs should be different
	if id1 == id2 {
		t.Error("Expected different request IDs")
	}

	// IDs should start with "req_"
	if id1[:4] != "req_" {
		t.Errorf("Expected ID to start with 'req_', got %s", id1)
	}
}

func TestProxy_GetTargetURL(t *testing.T) {
	p := &Proxy{}

	tests := []struct {
		name    string
		service *database.Service
		want    string
	}{
		{
			name: "http with default port",
			service: &database.Service{
				Protocol: "http",
				Host:     "localhost",
				Port:     80,
			},
			want: "http://localhost",
		},
		{
			name: "https with default port",
			service: &database.Service{
				Protocol: "https",
				Host:     "api.example.com",
				Port:     443,
			},
			want: "https://api.example.com",
		},
		{
			name: "http with custom port",
			service: &database.Service{
				Protocol: "http",
				Host:     "localhost",
				Port:     8081,
			},
			want: "http://localhost:8081",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := p.getTargetURL(tt.service)
			if err != nil {
				t.Fatalf("getTargetURL() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("getTargetURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProxy_BuildUpstreamURL(t *testing.T) {
	p := &Proxy{}

	tests := []struct {
		name      string
		targetURL string
		path      string
		query     string
		stripPath bool
		routePath string
		want      string
	}{
		{
			name:      "simple path",
			targetURL: "http://backend",
			path:      "/api/users",
			want:      "http://backend/api/users",
		},
		{
			name:      "with query string",
			targetURL: "http://backend",
			path:      "/api/users",
			query:     "page=1&limit=10",
			want:      "http://backend/api/users?page=1&limit=10",
		},
		{
			name:      "strip path",
			targetURL: "http://backend",
			path:      "/api/users/123",
			stripPath: true,
			routePath: "/api",
			want:      "http://backend/users/123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			if tt.query != "" {
				req.URL.RawQuery = tt.query
			}

			match := &router.MatchResult{
				Route: &database.Route{
					StripPath: tt.stripPath,
					Paths:     []string{tt.routePath},
				},
			}

			got := p.buildUpstreamURL(tt.targetURL, req, match)
			if got != tt.want {
				t.Errorf("buildUpstreamURL() = %v, want %v", got, tt.want)
			}
		})
	}
}
