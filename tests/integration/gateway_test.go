//go:build integration
// +build integration

package integration

import (
	"net/http"
	"testing"
	"time"
)

func TestGatewayHealthEndpoint(t *testing.T) {
	// Wait for gateway to be ready
	time.Sleep(2 * time.Second)

	resp, err := http.Get("http://localhost:8080/health")
	if err != nil {
		t.Fatalf("Failed to reach gateway: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestAdminAPIHealthEndpoint(t *testing.T) {
	resp, err := http.Get("http://localhost:8000/health")
	if err != nil {
		t.Fatalf("Failed to reach admin API: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}
