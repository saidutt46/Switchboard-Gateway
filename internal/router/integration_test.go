//go:build integration
// +build integration

package router

import (
	"context"
	"net/http"
	"os"
	"testing"

	"github.com/saidutt46/switchboard-gateway/internal/config"
	"github.com/saidutt46/switchboard-gateway/internal/database"
)

func TestRouter_Integration(t *testing.T) {
	// Only run if POSTGRES_DSN is set
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Skip("Skipping integration test: POSTGRES_DSN not set")
	}

	// Connect to database
	cfg := config.DatabaseConfig{
		DSN:          dsn,
		MaxOpenConns: 5,
		MaxIdleConns: 2,
	}

	db, err := database.NewDB(cfg)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	repo := database.NewRepository(db)
	ctx := context.Background()

	// Load routes and services
	routes, err := repo.GetRoutes(ctx, false)
	if err != nil {
		t.Fatalf("Failed to load routes: %v", err)
	}

	services, err := repo.GetServices(ctx, false)
	if err != nil {
		t.Fatalf("Failed to load services: %v", err)
	}

	// Create router
	r := NewRouter(routes, services)

	// Test matching with sample data
	req, _ := http.NewRequest("GET", "/api/users", nil)
	result, err := r.Match(req)

	if err != nil {
		t.Logf("No route matched (this is OK if no routes exist): %v", err)
		return
	}

	t.Logf("Matched route: %s -> service: %s", result.Route.Name.String, result.Service.Name)
}
