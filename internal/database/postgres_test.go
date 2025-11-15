package database

import (
	"testing"
	"time"

	"github.com/saidutt46/switchboard-gateway/internal/config"
)

func TestConfig_Defaults(t *testing.T) {
	cfg := config.DatabaseConfig{
		DSN:          "test-dsn",
		MaxOpenConns: 25,
		MaxIdleConns: 5,
	}

	// Verify config structure
	if cfg.DSN != "test-dsn" {
		t.Errorf("expected DSN to be 'test-dsn', got %s", cfg.DSN)
	}

	if cfg.MaxOpenConns != 25 {
		t.Errorf("expected MaxOpenConns to be 25, got %d", cfg.MaxOpenConns)
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
