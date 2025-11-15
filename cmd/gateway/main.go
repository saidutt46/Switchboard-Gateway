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
	"strings"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"

	"github.com/saidutt46/switchboard-gateway/internal/config"
	"github.com/saidutt46/switchboard-gateway/internal/database"
	"github.com/saidutt46/switchboard-gateway/internal/gateway"
	"github.com/saidutt46/switchboard-gateway/internal/health"
	"github.com/saidutt46/switchboard-gateway/internal/logging"
	"github.com/saidutt46/switchboard-gateway/internal/router"
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

	log.Info().
		Str("environment", cfg.Environment).
		Str("server_host", cfg.ServerHost).
		Int("server_port", cfg.ServerPort).
		Str("log_level", cfg.LogLevel).
		Str("log_format", cfg.LogFormat).
		Msg("Configuration loaded successfully")

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
	routes, err := repo.GetRoutes(context.Background(), false)
	if err != nil {
		return fmt.Errorf("failed to load routes: %w", err)
	}

	services, err := repo.GetServices(context.Background(), false)
	if err != nil {
		return fmt.Errorf("failed to load services: %w", err)
	}

	// Create router EARLY (before setupRoutes)
	rt := router.NewRouter(routes, services)

	log.Info().
		Int("routes", len(routes)).
		Int("services", len(services)).
		Msg("Router initialized")

	// Load plugins (for future phases)
	plugins, err := repo.GetPlugins(context.Background(), true)
	if err != nil {
		return fmt.Errorf("failed to load plugins: %w", err)
	}

	log.Info().
		Int("count", len(plugins)).
		Msg("Plugins loaded from database")

	// Initialize Redis for hot reload
	// Parse Redis URL or use direct address
	redisAddr := "localhost:6379"
	redisDB := 0

	// If RedisURL is set, use it
	if len(cfg.RedisURL) > 0 {
		if strings.HasPrefix(cfg.RedisURL, "redis://") {
			// Parse full Redis URL
			opt, err := redis.ParseURL(cfg.RedisURL)
			if err != nil {
				log.Warn().
					Err(err).
					Str("url", cfg.RedisURL).
					Msg("Failed to parse Redis URL, using default localhost:6379")
			} else {
				redisAddr = opt.Addr
				redisDB = opt.DB
				log.Debug().
					Str("addr", redisAddr).
					Int("db", redisDB).
					Msg("Parsed Redis URL")
			}
		} else {
			// Direct address format (host:port)
			redisAddr = cfg.RedisURL
		}
	}

	log.Debug().
		Str("redis_addr", redisAddr).
		Int("redis_db", redisDB).
		Msg("Connecting to Redis")

	redisClient := redis.NewClient(&redis.Options{
		Addr: redisAddr,
		DB:   redisDB,
	})
	defer redisClient.Close()

	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Warn().Err(err).Msg("Redis connection failed - hot reload disabled")
	} else {
		log.Info().Msg("Redis connection established")

		// Create gateway instance
		gw := gateway.New(rt, repo)

		// Start config watcher
		watcher := config.NewWatcher(redisClient, gw)
		go func() {
			if err := watcher.Start(ctx); err != nil {
				log.Error().Err(err).Msg("Config watcher stopped")
			}
		}()

		log.Info().Msg("Config watcher started - hot reload enabled! 🔥")
	}

	// Setup HTTP server
	mux := setupRoutes(db, repo, rt)

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
func setupRoutes(db *database.DB, repo *database.Repository, rt *router.Router) *http.ServeMux {
	mux := http.NewServeMux()

	// Health check endpoint
	healthHandler := health.NewHandler(db, repo)
	mux.HandleFunc("/health", healthHandler.Health)

	// Ready check endpoint (for Kubernetes)
	mux.HandleFunc("/ready", healthHandler.Ready)

	// Proxy handler - USE THE ROUTER!
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Skip health/ready checks
		if r.URL.Path == "/health" || r.URL.Path == "/ready" {
			return
		}

		// Match route using router
		result, err := rt.Match(r)
		if err != nil {
			log.Debug().
				Str("path", r.URL.Path).
				Str("method", r.Method).
				Msg("No route matched")
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}

		log.Info().
			Str("path", r.URL.Path).
			Str("method", r.Method).
			Str("route_id", result.Route.ID).
			Str("service_id", result.Service.ID).
			Msg("Route matched")

		// For now, return success (Phase 3 will add actual proxying)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"matched":true,"route":"%s","service":"%s","params":%v}`,
			result.Route.Name.String,
			result.Service.Name,
			result.PathParams)
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

	// TODO: Phase 3 onwards - Build route tree from loaded routes
	// TODO: Phase 7 onwards - Initialize plugin chain

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
