// Package main is the entrypoint for the Switchboard API Gateway.
//
// The gateway is a high-performance reverse proxy that sits between clients
// and backend microservices, providing features like:
// - Request routing and load balancing
// - Authentication and authorization
// - Rate limiting and traffic control
// - Response caching
// - Circuit breaking and resilience
// - Observability and analytics
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"

	"github.com/saidutt46/switchboard-gateway/internal/config"
	"github.com/saidutt46/switchboard-gateway/internal/database"
	"github.com/saidutt46/switchboard-gateway/internal/health"
	"github.com/saidutt46/switchboard-gateway/internal/logging"
)

// Version information (set during build via ldflags)
var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

func main() {
	// Run the application and exit with appropriate code
	if err := run(); err != nil {
		log.Fatal().Err(err).Msg("Application failed to start")
		os.Exit(1)
	}
}

// run contains the main application logic.
// Separating this from main() makes it easier to test and handle errors.
func run() error {
	// Print banner
	printBanner()

	// Load .env file if it exists (optional - won't fail if missing)
	// This allows local development with .env file
	// Production should use actual environment variables
	if err := godotenv.Load(); err != nil {
		// Only log if file doesn't exist, don't fail
		// In production, .env won't exist and that's fine
		log.Debug().Msg("No .env file found, using environment variables")
	} else {
		log.Debug().Msg("Loaded configuration from .env file")
	}

	// Load configuration from environment variables
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Setup logging
	if err := logging.Setup(cfg.LogLevel, cfg.LogFormat); err != nil {
		return fmt.Errorf("failed to setup logging: %w", err)
	}

	log.Info().
		Str("version", Version).
		Str("build_time", BuildTime).
		Str("git_commit", GitCommit).
		Str("environment", cfg.Environment).
		Msg("Switchboard API Gateway starting...")

	// Connect to database
	db, err := database.NewDB(cfg.Database)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Error().Err(err).Msg("Error closing database connection")
		}
	}()

	// Create repository
	repo := database.NewRepository(db)

	log.Info().Msg("Database connection established")

	// Load initial configuration from database
	if err := loadGatewayConfig(repo); err != nil {
		return fmt.Errorf("failed to load gateway configuration: %w", err)
	}

	// Setup HTTP server
	mux := setupRoutes(db, repo)

	server := &http.Server{
		Addr:         cfg.ServerAddress(),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Channel to listen for errors from the server
	serverErrors := make(chan error, 1)

	// Start HTTP server in a goroutine
	go func() {
		log.Info().
			Str("address", cfg.ServerAddress()).
			Msg("HTTP server starting")

		serverErrors <- server.ListenAndServe()
	}()

	// Channel to listen for interrupt signals
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Block until we receive a signal or server error
	select {
	case err := <-serverErrors:
		return fmt.Errorf("server error: %w", err)

	case sig := <-shutdown:
		log.Info().
			Str("signal", sig.String()).
			Msg("Shutdown signal received, starting graceful shutdown...")

		// Create context with timeout for shutdown
		ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()

		// Attempt graceful shutdown
		if err := server.Shutdown(ctx); err != nil {
			log.Error().Err(err).Msg("Error during graceful shutdown, forcing shutdown")
			if err := server.Close(); err != nil {
				return fmt.Errorf("could not stop server gracefully: %w", err)
			}
		}

		log.Info().Msg("Server stopped gracefully")
	}

	return nil
}

// setupRoutes configures all HTTP routes for the gateway.
func setupRoutes(db *database.DB, repo *database.Repository) *http.ServeMux {
	mux := http.NewServeMux()

	// Health check endpoint
	healthHandler := health.NewHandler(db, repo)
	mux.HandleFunc("/health", healthHandler.Health)

	// Ready check endpoint (for Kubernetes)
	mux.HandleFunc("/ready", healthHandler.Ready)

	// Root endpoint
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// For now, return 404 for all other routes
		// In Phase 3, this will be the proxy handler
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"message":"Switchboard API Gateway","version":"%s","status":"running"}`, Version)
	})

	return mux
}

// loadGatewayConfig loads the gateway configuration from the database.
// This includes services, routes, plugins, etc.
func loadGatewayConfig(repo *database.Repository) error {
	ctx := context.Background()

	// Load services
	services, err := repo.GetServices(ctx, false) // Only enabled services
	if err != nil {
		return fmt.Errorf("failed to load services: %w", err)
	}

	log.Info().
		Int("count", len(services)).
		Msg("Services loaded from database")

	// Load routes
	routes, err := repo.GetRoutes(ctx, false) // Only enabled routes
	if err != nil {
		return fmt.Errorf("failed to load routes: %w", err)
	}

	log.Info().
		Int("count", len(routes)).
		Msg("Routes loaded from database")

	// Load plugins
	plugins, err := repo.GetPlugins(ctx, true) // Only enabled plugins
	if err != nil {
		return fmt.Errorf("failed to load plugins: %w", err)
	}

	log.Info().
		Int("count", len(plugins)).
		Msg("Plugins loaded from database")

	// TODO: Phase 3 - Build route tree from loaded routes
	// TODO: Phase 7 - Initialize plugin chain

	return nil
}

// printBanner prints the application banner with version information.
func printBanner() {
	banner := `
╔═══════════════════════════════════════════════════════════╗
║                                                           ║
║   ███████╗██╗    ██╗██╗████████╗ ██████╗██╗  ██╗         ║
║   ██╔════╝██║    ██║██║╚══██╔══╝██╔════╝██║  ██║         ║
║   ███████╗██║ █╗ ██║██║   ██║   ██║     ███████║         ║
║   ╚════██║██║███╗██║██║   ██║   ██║     ██╔══██║         ║
║   ███████║╚███╔███╔╝██║   ██║   ╚██████╗██║  ██║         ║
║   ╚══════╝ ╚══╝╚══╝ ╚═╝   ╚═╝    ╚═════╝╚═╝  ╚═╝         ║
║                                                           ║
║              API Gateway - High Performance               ║
║                                                           ║
╚═══════════════════════════════════════════════════════════╝
`
	fmt.Println(banner)
	fmt.Printf("Version: %s | Build: %s | Commit: %s\n\n", Version, BuildTime, GitCommit)
}
