// Package main is the entrypoint for the Switchboard API Gateway.
//
// The gateway is a high-performance reverse proxy that sits between clients
// and backend microservices, providing features like:
// - Request routing with O(log n) radix tree matching
// - Hot configuration reload via Redis pub/sub
// - Health checks and monitoring
// - Graceful shutdown handling
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
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"

	"github.com/saidutt46/switchboard-gateway/internal/config"
	"github.com/saidutt46/switchboard-gateway/internal/database"
	"github.com/saidutt46/switchboard-gateway/internal/gateway"
	"github.com/saidutt46/switchboard-gateway/internal/health"
	"github.com/saidutt46/switchboard-gateway/internal/logging"
	"github.com/saidutt46/switchboard-gateway/internal/plugin"
	"github.com/saidutt46/switchboard-gateway/internal/plugin/builtin"
	"github.com/saidutt46/switchboard-gateway/internal/proxy"
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
	if err := godotenv.Load(); err != nil {
		log.Debug().Msg("No .env file found, using environment variables")
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
			log.Error().
				Err(err).
				Str("component", "database").
				Msg("Error closing database connection")
		}
	}()

	// Create repository
	repo := database.NewRepository(db)

	log.Info().
		Str("component", "database").
		Msg("Database connection established successfully")

	// Load initial configuration from database
	routes, err := repo.GetRoutes(context.Background(), false)
	if err != nil {
		return fmt.Errorf("failed to load routes: %w", err)
	}

	services, err := repo.GetServices(context.Background(), false)
	if err != nil {
		return fmt.Errorf("failed to load services: %w", err)
	}

	// Initialize plugin system
	pluginRegistry, pluginInstances, err := initializePlugins(context.Background(), repo)
	if err != nil {
		log.Warn().
			Err(err).
			Msg("Failed to initialize plugins - continuing without plugins")
		pluginRegistry = nil
		pluginInstances = []plugin.PluginInstance{} // Empty plugins
	}

	// Create router with radix tree and plugins
	rt := router.NewRouter(routes, services, pluginInstances)

	// Log router statistics
	stats := rt.Stats()
	log.Info().
		Str("component", "router").
		Int("routes", len(routes)).
		Int("services", len(services)).
		Int("plugins", len(pluginInstances)).
		Interface("stats", stats).
		Msg("Router initialized with radix tree and plugins")

	// Create reverse proxy with HTTP transport configuration
	transportConfig := &proxy.TransportConfig{
		// Connection pool settings
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		MaxConnsPerHost:     100,

		// Timeouts
		DialTimeout:           30 * time.Second,
		KeepAlive:             30 * time.Second,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,

		// TLS
		InsecureSkipVerify: false, // Verify TLS certificates in production
	}

	px := proxy.NewProxy(rt, proxy.NewTransport(transportConfig))

	log.Info().
		Str("component", "proxy").
		Int("max_idle_conns", transportConfig.MaxIdleConns).
		Int("max_idle_per_host", transportConfig.MaxIdleConnsPerHost).
		Dur("idle_timeout", transportConfig.IdleConnTimeout).
		Msg("Reverse proxy initialized with connection pooling")

	log.Info().
		Str("component", "proxy").
		Msg("Reverse proxy initialized")

	// Load plugins (for future phases)
	plugins, err := repo.GetPlugins(context.Background(), true)
	if err != nil {
		return fmt.Errorf("failed to load plugins: %w", err)
	}

	log.Info().
		Str("component", "plugins").
		Int("count", len(plugins)).
		Msg("Plugins loaded from database")

	// Initialize Redis for hot reload
	redisClient, err := initializeRedis(cfg)
	if err != nil {
		log.Warn().
			Err(err).
			Msg("Redis setup failed - hot reload disabled")
	} else {
		// Create gateway instance for config changes (with plugin registry for hot reload)
		gw := gateway.New(rt, repo, pluginRegistry)

		// Start config watcher in background
		watcher := config.NewWatcher(redisClient, gw)
		go func() {
			if err := watcher.Start(context.Background()); err != nil {
				log.Error().
					Err(err).
					Str("component", "watcher").
					Msg("Config watcher stopped")
			}
		}()

		log.Info().
			Str("component", "hot_reload").
			Msg("Config watcher started - hot reload enabled 🔥")
	}

	// Setup HTTP server
	mux := setupRoutes(db, repo, rt, px)

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

// initializePlugins sets up the plugin registry and loads plugins.
// Returns the registry and loaded plugin instances.
func initializePlugins(ctx context.Context, repo *database.Repository) (*plugin.Registry, []plugin.PluginInstance, error) {
	log.Info().
		Str("component", "plugins").
		Msg("Initializing plugin system")

	// Create plugin registry
	registry := plugin.NewRegistry()

	// Register built-in plugins
	registry.Register("request-logger", builtin.NewRequestLogger)
	registry.Register("cors", builtin.NewCORSPlugin)

	log.Info().
		Str("component", "plugins").
		Interface("registered", registry.GetRegisteredPlugins()).
		Msg("Built-in plugins registered")

	// Load plugin configurations from database
	instances, err := registry.LoadFromDatabase(ctx, repo)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load plugins from database: %w", err)
	}

	// Log statistics
	stats := registry.Stats()
	log.Info().
		Str("component", "plugins").
		Interface("stats", stats).
		Msg("Plugin system initialized successfully")

	return registry, instances, nil
}

// initializeRedis creates and tests Redis connection for hot reload.
func initializeRedis(cfg *config.Config) (*redis.Client, error) {
	log.Debug().
		Str("component", "redis").
		Msg("Initializing Redis connection")

	// Parse Redis URL
	opt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		// Fallback to default if URL parsing fails
		opt = &redis.Options{
			Addr: "localhost:6379",
			DB:   0,
		}
		log.Debug().
			Err(err).
			Str("fallback", opt.Addr).
			Msg("Using default Redis address")
	}

	// Create client
	client := redis.NewClient(opt)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	log.Info().
		Str("component", "redis").
		Str("addr", opt.Addr).
		Msg("Redis connection established")

	return client, nil
}

// setupRoutes configures all HTTP routes for the gateway.
func setupRoutes(db *database.DB, repo *database.Repository, rt *router.Router, px *proxy.Proxy) *http.ServeMux {
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

		// Generate request ID
		requestID := fmt.Sprintf("req_%d", time.Now().UnixNano())

		// Match route using router
		result, err := rt.Match(r)
		if err != nil {
			log.Debug().
				Str("component", "proxy").
				Str("request_id", requestID).
				Str("path", r.URL.Path).
				Str("method", r.Method).
				Msg("No route matched")

			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}

		// Log successful match
		log.Info().
			Str("component", "proxy").
			Str("request_id", requestID).
			Str("path", r.URL.Path).
			Str("method", r.Method).
			Str("route_id", result.Route.ID).
			Str("route_name", result.Route.Name.String).
			Str("service_id", result.Service.ID).
			Str("service_name", result.Service.Name).
			Interface("path_params", result.PathParams).
			Int("plugin_count", result.Chain.Count()).
			Msg("Route matched successfully")

		// Create plugin context
		ctx := plugin.NewContext(
			r,
			w,
			result.Route,
			result.Service,
			plugin.PhaseBeforeRequest,
		)

		// Execute plugin chain - BEFORE request
		if err := result.Chain.Execute(ctx); err != nil {
			log.Error().
				Err(err).
				Str("request_id", requestID).
				Msg("Critical plugin failure - aborting request")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Check if plugin aborted the request
		if ctx.IsAborted() {
			log.Info().
				Str("request_id", requestID).
				Int("status_code", ctx.AbortStatusCode()).
				Str("message", ctx.AbortMessage()).
				Msg("Request aborted by plugin")

			// Plugin already wrote response (e.g., preflight CORS)
			// Just return
			return
		}

		// Proxy request to backend service
		log.Debug().
			Str("request_id", requestID).
			Str("route", result.Route.Name.String).
			Str("service", result.Service.Name).
			Msg("Proxying request to backend")

		// Proxy to backend (use plugin's ResponseWriter to track size)
		px.ServeHTTP(ctx.Response, r)

		// Execute plugin chain - AFTER response
		ctx.Phase = plugin.PhaseAfterResponse
		if err := result.Chain.Execute(ctx); err != nil {
			log.Warn().
				Err(err).
				Str("request_id", requestID).
				Msg("Plugin error in AfterResponse phase")
			// Don't fail the request - response already sent
		}

		// Execute plugin chain - AFTER response (for logging, etc.)
		ctx.Phase = plugin.PhaseAfterResponse
		if err := result.Chain.Execute(ctx); err != nil {
			log.Warn().
				Err(err).
				Str("request_id", requestID).
				Msg("Plugin error in AfterResponse phase")
			// Don't fail the request - response already sent
		}
	})

	return mux
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
