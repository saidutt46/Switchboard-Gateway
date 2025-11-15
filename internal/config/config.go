// Package config provides application configuration management.
//
// Configuration is loaded from environment variables using the envconfig package.
// This follows the 12-factor app methodology for configuration management.
package config

import (
	"fmt"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/rs/zerolog/log"
)

// Config holds all application configuration.
//
// Configuration is loaded from environment variables with sensible defaults.
// Required fields will cause the application to fail if not provided.
type Config struct {
	// Environment
	Environment string `envconfig:"ENVIRONMENT" default:"development"`

	// Server
	ServerHost string `envconfig:"GATEWAY_HOST" default:"0.0.0.0"`
	ServerPort int    `envconfig:"GATEWAY_PORT" default:"8080"`

	// Database
	Database DatabaseConfig

	// Redis (Phase 8)
	RedisURL string `envconfig:"REDIS_URL" default:"redis://localhost:6379/0"`

	// Kafka (Phase 14)
	KafkaBrokers string `envconfig:"KAFKA_BROKERS" default:"localhost:9092"`

	// Logging
	LogLevel  string `envconfig:"LOG_LEVEL" default:"info"`
	LogFormat string `envconfig:"LOG_FORMAT" default:"json"` // json or console

	// Shutdown
	ShutdownTimeout time.Duration `envconfig:"SHUTDOWN_TIMEOUT" default:"30s"`
}

// DatabaseConfig holds database-specific configuration.
type DatabaseConfig struct {
	DSN string `envconfig:"POSTGRES_DSN" required:"true"`

	// Connection pool settings
	MaxOpenConns    int           `envconfig:"DB_MAX_OPEN_CONNS" default:"25"`
	MaxIdleConns    int           `envconfig:"DB_MAX_IDLE_CONNS" default:"5"`
	ConnMaxLifetime time.Duration `envconfig:"DB_CONN_MAX_LIFETIME" default:"5m"`
	ConnMaxIdleTime time.Duration `envconfig:"DB_CONN_MAX_IDLE_TIME" default:"5m"`

	// Connection timeout
	ConnectTimeout time.Duration `envconfig:"DB_CONNECT_TIMEOUT" default:"10s"`
}

// Load loads configuration from environment variables.
//
// It uses envconfig to parse environment variables into the Config struct.
// Returns an error if required variables are missing or invalid.
func Load() (*Config, error) {
	var cfg Config

	// Process environment variables
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	log.Info().
		Str("environment", cfg.Environment).
		Str("server_host", cfg.ServerHost).
		Int("server_port", cfg.ServerPort).
		Str("log_level", cfg.LogLevel).
		Str("log_format", cfg.LogFormat).
		Msg("Configuration loaded successfully")

	return &cfg, nil
}

// Validate validates the configuration.
//
// Returns an error if any configuration values are invalid.
func (c *Config) Validate() error {
	// Validate environment
	validEnvironments := map[string]bool{
		"development": true,
		"staging":     true,
		"production":  true,
		"test":        true,
	}

	if !validEnvironments[c.Environment] {
		return fmt.Errorf("invalid environment: %s (must be development, staging, production, or test)", c.Environment)
	}

	// Validate server port
	if c.ServerPort < 1 || c.ServerPort > 65535 {
		return fmt.Errorf("invalid server port: %d (must be between 1 and 65535)", c.ServerPort)
	}

	// Validate log level
	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}

	if !validLogLevels[c.LogLevel] {
		return fmt.Errorf("invalid log level: %s (must be debug, info, warn, or error)", c.LogLevel)
	}

	// Validate log format
	if c.LogFormat != "json" && c.LogFormat != "console" {
		return fmt.Errorf("invalid log format: %s (must be json or console)", c.LogFormat)
	}

	// Validate database DSN is not empty (envconfig already checks required)
	if c.Database.DSN == "" {
		return fmt.Errorf("database DSN is required")
	}

	// Validate connection pool settings
	if c.Database.MaxOpenConns < 1 {
		return fmt.Errorf("max_open_conns must be at least 1")
	}

	if c.Database.MaxIdleConns < 1 {
		return fmt.Errorf("max_idle_conns must be at least 1")
	}

	if c.Database.MaxIdleConns > c.Database.MaxOpenConns {
		return fmt.Errorf("max_idle_conns (%d) cannot be greater than max_open_conns (%d)",
			c.Database.MaxIdleConns, c.Database.MaxOpenConns)
	}

	return nil
}

// IsDevelopment returns true if running in development environment.
func (c *Config) IsDevelopment() bool {
	return c.Environment == "development"
}

// IsProduction returns true if running in production environment.
func (c *Config) IsProduction() bool {
	return c.Environment == "production"
}

// ServerAddress returns the server address in host:port format.
func (c *Config) ServerAddress() string {
	return fmt.Sprintf("%s:%d", c.ServerHost, c.ServerPort)
}
