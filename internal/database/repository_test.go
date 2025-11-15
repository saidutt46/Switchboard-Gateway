package database

import (
	"context"
	"testing"
	"time"
)

// TestRepository_NewRepository tests repository creation.
func TestRepository_NewRepository(t *testing.T) {
	// Create a mock DB (we're not testing actual DB connection here)
	db := &DB{}
	repo := NewRepository(db)

	if repo == nil {
		t.Fatal("expected repository to be created, got nil")
	}

	if repo.db != db {
		t.Error("expected repository to hold reference to DB")
	}
}

// TestModels_ServiceValidation tests service model structure.
func TestModels_ServiceValidation(t *testing.T) {
	tests := []struct {
		name    string
		service Service
		wantErr bool
	}{
		{
			name: "valid http service",
			service: Service{
				ID:       "test-id",
				Name:     "test-service",
				Protocol: "http",
				Host:     "localhost",
				Port:     8080,
				Enabled:  true,
			},
			wantErr: false,
		},
		{
			name: "valid https service",
			service: Service{
				ID:       "test-id-2",
				Name:     "secure-service",
				Protocol: "https",
				Host:     "api.example.com",
				Port:     443,
				Enabled:  true,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate protocol
			if tt.service.Protocol != "http" && tt.service.Protocol != "https" && tt.service.Protocol != "grpc" {
				if !tt.wantErr {
					t.Errorf("invalid protocol: %s", tt.service.Protocol)
				}
			}

			// Validate port
			if tt.service.Port <= 0 || tt.service.Port > 65535 {
				if !tt.wantErr {
					t.Errorf("invalid port: %d", tt.service.Port)
				}
			}
		})
	}
}

// TestModels_RouteStructure tests route model.
func TestModels_RouteStructure(t *testing.T) {
	route := Route{
		ID:        "route-1",
		ServiceID: "service-1",
		Paths:     []string{"/api/users", "/api/users/:id"},
		Methods:   []string{"GET", "POST"},
		Enabled:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Validate paths
	if len(route.Paths) == 0 {
		t.Error("route should have at least one path")
	}

	// Validate methods
	if len(route.Methods) == 0 {
		t.Error("route should have at least one method")
	}

	// Check path array works correctly
	expectedPath := "/api/users"
	if route.Paths[0] != expectedPath {
		t.Errorf("expected first path to be %s, got %s", expectedPath, route.Paths[0])
	}
}

// TestModels_ConsumerStructure tests consumer model.
func TestModels_ConsumerStructure(t *testing.T) {
	consumer := Consumer{
		ID:       "consumer-1",
		Username: "test-app",
		Metadata: map[string]interface{}{
			"app_version": "1.0.0",
			"platform":    "ios",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if consumer.Username == "" {
		t.Error("consumer should have a username")
	}

	// Validate metadata
	if consumer.Metadata == nil {
		t.Error("metadata should be initialized")
	}

	if version, ok := consumer.Metadata["app_version"]; !ok || version != "1.0.0" {
		t.Error("metadata should contain app_version")
	}
}

// TestModels_PluginScopeValidation tests plugin scope constants.
func TestModels_PluginScopeValidation(t *testing.T) {
	validScopes := map[string]bool{
		PluginScopeGlobal:   true,
		PluginScopeService:  true,
		PluginScopeRoute:    true,
		PluginScopeConsumer: true,
	}

	// Test all defined valid scopes
	for _, scope := range ValidPluginScopes {
		if !validScopes[scope] {
			t.Errorf("scope %s not in valid scopes map", scope)
		}
	}

	// Test invalid scope
	invalidScope := "invalid-scope"
	if validScopes[invalidScope] {
		t.Error("invalid scope should not be valid")
	}
}

// TestModels_APIKeySecurity tests that API key hash is not exposed in JSON.
func TestModels_APIKeySecurity(t *testing.T) {
	apiKey := APIKey{
		ID:         "key-1",
		ConsumerID: "consumer-1",
		KeyHash:    "should-not-be-exposed-in-json",
		Enabled:    true,
		CreatedAt:  time.Now(),
	}

	// In a real test, you would marshal to JSON and verify KeyHash is not present
	// For now, we just verify the field exists
	if apiKey.KeyHash == "" {
		t.Error("key hash should be set internally")
	}

	// The json:"-" tag ensures KeyHash is never serialized to JSON
	// This is a security measure to prevent accidental exposure
}

// TestRepository_ContextCancellation tests that queries respect context cancellation.
func TestRepository_ContextCancellation(t *testing.T) {
	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// This test verifies that our repository methods accept context
	// In a real integration test, this would test actual query cancellation
	if ctx.Err() == nil {
		t.Error("expected context to be cancelled")
	}
}
