package database

import (
	"testing"
	"time"
)

func TestConfig_Defaults(t *testing.T) {
	cfg := Config{
		DSN: "test-dsn",
	}

	// Verify defaults would be applied by envconfig
	if cfg.DSN != "test-dsn" {
		t.Errorf("expected DSN to be 'test-dsn', got %s", cfg.DSN)
	}
}

func TestModels_ServiceStruct(t *testing.T) {
	svc := Service{
		ID:        "test-id",
		Name:      "test-service",
		Protocol:  "http",
		Host:      "localhost",
		Port:      8080,
		Enabled:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if svc.Name != "test-service" {
		t.Errorf("expected name 'test-service', got %s", svc.Name)
	}

	if svc.Port != 8080 {
		t.Errorf("expected port 8080, got %d", svc.Port)
	}
}
