// Package database provides PostgreSQL database connectivity and operations
// for the Switchboard API Gateway.
//
// This package handles:
//   - Database connection pool management
//   - Health checks and connection verification
//   - Graceful shutdown
//   - Connection retry logic
package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	// PostgreSQL driver
	"github.com/rs/zerolog/log"
)

// DB wraps the sql.DB connection pool and provides additional functionality.
type DB struct {
	pool *sql.DB
	dsn  string
}

// Config holds database connection configuration.
type Config struct {
	DSN string `envconfig:"POSTGRES_DSN" required:"true"`

	// Connection pool settings
	MaxOpenConns    int           `envconfig:"DB_MAX_OPEN_CONNS" default:"25"`
	MaxIdleConns    int           `envconfig:"DB_MAX_IDLE_CONNS" default:"5"`
	ConnMaxLifetime time.Duration `envconfig:"DB_CONN_MAX_LIFETIME" default:"5m"`
	ConnMaxIdleTime time.Duration `envconfig:"DB_CONN_MAX_IDLE_TIME" default:"5m"`

	// Connection timeout
	ConnectTimeout time.Duration `envconfig:"DB_CONNECT_TIMEOUT" default:"10s"`
}

// NewDB creates a new database connection pool with the provided configuration.
//
// It establishes a connection, configures the pool, and verifies connectivity.
// Returns an error if connection fails or ping times out.
func NewDB(cfg Config) (*DB, error) {
	log.Info().
		Str("component", "database").
		Msg("Connecting to PostgreSQL...")

	// Create connection pool
	pool, err := sql.Open("postgres", cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Configure connection pool
	pool.SetMaxOpenConns(cfg.MaxOpenConns)
	pool.SetMaxIdleConns(cfg.MaxIdleConns)
	pool.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	pool.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	db := &DB{
		pool: pool,
		dsn:  cfg.DSN,
	}

	// Verify connection with timeout
	ctx, cancel := context.WithTimeout(context.Background(), cfg.ConnectTimeout)
	defer cancel()

	if err := db.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Info().
		Str("component", "database").
		Int("max_open_conns", cfg.MaxOpenConns).
		Int("max_idle_conns", cfg.MaxIdleConns).
		Dur("conn_max_lifetime", cfg.ConnMaxLifetime).
		Msg("Database connection established")

	return db, nil
}

// Pool returns the underlying *sql.DB connection pool.
//
// This allows other packages to execute queries directly when needed.
func (db *DB) Pool() *sql.DB {
	return db.pool
}

// Ping verifies the database connection is alive.
//
// It attempts to ping the database with the provided context.
// Returns an error if the ping fails or context times out.
func (db *DB) Ping(ctx context.Context) error {
	if err := db.pool.PingContext(ctx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}
	return nil
}

// Stats returns database connection pool statistics.
//
// Useful for monitoring and debugging connection pool health.
func (db *DB) Stats() sql.DBStats {
	return db.pool.Stats()
}

// Health checks the database health and returns status information.
//
// Returns a map with health metrics including:
//   - status: "healthy" or "unhealthy"
//   - open_connections: current open connections
//   - in_use: connections currently in use
//   - idle: idle connections
//   - wait_count: total number of connections waited for
//   - wait_duration: total time blocked waiting for connections
//   - max_idle_closed: connections closed due to max idle
//   - max_lifetime_closed: connections closed due to max lifetime
func (db *DB) Health(ctx context.Context) map[string]interface{} {
	health := make(map[string]interface{})

	// Try to ping
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	if err := db.Ping(ctx); err != nil {
		health["status"] = "unhealthy"
		health["error"] = err.Error()
		return health
	}

	// Get connection pool stats
	stats := db.Stats()

	health["status"] = "healthy"
	health["open_connections"] = stats.OpenConnections
	health["in_use"] = stats.InUse
	health["idle"] = stats.Idle
	health["wait_count"] = stats.WaitCount
	health["wait_duration_ms"] = stats.WaitDuration.Milliseconds()
	health["max_idle_closed"] = stats.MaxIdleClosed
	health["max_lifetime_closed"] = stats.MaxLifetimeClosed

	return health
}

// Close gracefully closes the database connection pool.
//
// It waits for all active connections to finish before closing.
// Should be called during application shutdown.
func (db *DB) Close() error {
	log.Info().
		Str("component", "database").
		Msg("Closing database connection pool...")

	if err := db.pool.Close(); err != nil {
		return fmt.Errorf("failed to close database pool: %w", err)
	}

	log.Info().
		Str("component", "database").
		Msg("Database connection pool closed")

	return nil
}

// Begin starts a new database transaction.
//
// The transaction must be committed or rolled back.
// Use defer to ensure rollback on error:
//
//	tx, err := db.Begin(ctx)
//	if err != nil {
//	    return err
//	}
//	defer tx.Rollback()
//
//	// ... do work ...
//
//	return tx.Commit()
func (db *DB) Begin(ctx context.Context) (*sql.Tx, error) {
	tx, err := db.pool.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	return tx, nil
}
