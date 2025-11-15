// Package database - Data models
//
// This file contains Go structs that map to PostgreSQL tables.
// All models include JSON tags for API responses and database column mapping.
package database

import (
	"database/sql"
	"time"

	"github.com/lib/pq"
)

// Service represents a backend microservice that the gateway proxies to.
//
// Maps to the 'services' table in PostgreSQL.
type Service struct {
	ID   string `json:"id" db:"id"`
	Name string `json:"name" db:"name"`

	// Connection details
	Protocol string         `json:"protocol" db:"protocol"` // http, https, grpc
	Host     string         `json:"host" db:"host"`
	Port     int            `json:"port" db:"port"`
	Path     sql.NullString `json:"path,omitempty" db:"path"`

	// Timeouts (milliseconds)
	ConnectTimeoutMs int `json:"connect_timeout_ms" db:"connect_timeout_ms"`
	ReadTimeoutMs    int `json:"read_timeout_ms" db:"read_timeout_ms"`
	WriteTimeoutMs   int `json:"write_timeout_ms" db:"write_timeout_ms"`
	Retries          int `json:"retries" db:"retries"`

	// Load balancing
	LoadBalancerType string `json:"load_balancer_type" db:"load_balancer_type"` // round-robin, least-connections, weighted, ip-hash

	Enabled   bool      `json:"enabled" db:"enabled"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// ServiceTarget represents a backend instance for load balancing.
//
// Maps to the 'service_targets' table in PostgreSQL.
// Multiple targets can be associated with one service.
type ServiceTarget struct {
	ID        string `json:"id" db:"id"`
	ServiceID string `json:"service_id" db:"service_id"`

	Target          string `json:"target" db:"target"`                       // Format: "host:port"
	Weight          int    `json:"weight" db:"weight"`                       // For weighted load balancing
	HealthCheckPath string `json:"health_check_path" db:"health_check_path"` // e.g., "/health"

	Enabled   bool      `json:"enabled" db:"enabled"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// Route maps incoming HTTP requests to services based on path, method, and host.
//
// Maps to the 'routes' table in PostgreSQL.
type Route struct {
	ID        string         `json:"id" db:"id"`
	ServiceID string         `json:"service_id" db:"service_id"`
	Name      sql.NullString `json:"name,omitempty" db:"name"`

	// Matching criteria
	Hosts   pq.StringArray `json:"hosts,omitempty" db:"hosts"` // e.g., ["api.example.com", "*.example.com"]
	Paths   pq.StringArray `json:"paths" db:"paths"`           // e.g., ["/api/users", "/api/users/:id"]
	Methods pq.StringArray `json:"methods" db:"methods"`       // e.g., ["GET", "POST"]

	// Path handling
	StripPath    bool `json:"strip_path" db:"strip_path"`       // Remove matched path before proxying
	PreserveHost bool `json:"preserve_host" db:"preserve_host"` // Keep original Host header

	Enabled   bool      `json:"enabled" db:"enabled"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// Consumer represents an API client (application or service) that calls the gateway.
//
// Maps to the 'consumers' table in PostgreSQL.
// Note: Consumer ≠ end user. Consumer = application/service making API requests.
type Consumer struct {
	ID       string         `json:"id" db:"id"`
	Username string         `json:"username" db:"username"`
	Email    sql.NullString `json:"email,omitempty" db:"email"`
	CustomID sql.NullString `json:"custom_id,omitempty" db:"custom_id"`

	// Metadata stores arbitrary JSON data about the consumer
	Metadata map[string]interface{} `json:"metadata,omitempty" db:"metadata"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// APIKey represents an authentication credential for a consumer.
//
// Maps to the 'api_keys' table in PostgreSQL.
//
// SECURITY: KeyHash stores SHA256 hash, NEVER plaintext!
// The actual key is only shown to the user once during creation.
type APIKey struct {
	ID         string         `json:"id" db:"id"`
	ConsumerID string         `json:"consumer_id" db:"consumer_id"`
	KeyHash    string         `json:"-" db:"key_hash"` // Never expose in JSON!
	Name       sql.NullString `json:"name,omitempty" db:"name"`

	Enabled    bool         `json:"enabled" db:"enabled"`
	CreatedAt  time.Time    `json:"created_at" db:"created_at"`
	LastUsedAt sql.NullTime `json:"last_used_at,omitempty" db:"last_used_at"`
	ExpiresAt  sql.NullTime `json:"expires_at,omitempty" db:"expires_at"`
}

// Plugin represents modular functionality (auth, rate limiting, caching, etc.).
//
// Maps to the 'plugins' table in PostgreSQL.
//
// Plugins can be scoped to:
//   - global: applies to all routes
//   - service: applies to all routes of a service
//   - route: applies to a specific route
//   - consumer: applies to a specific consumer
type Plugin struct {
	ID    string `json:"id" db:"id"`
	Name  string `json:"name" db:"name"`   // e.g., "rate-limit", "api-key-auth", "cache"
	Scope string `json:"scope" db:"scope"` // global, service, route, consumer

	// Foreign keys (only one should be set based on scope)
	ServiceID  sql.NullString `json:"service_id,omitempty" db:"service_id"`
	RouteID    sql.NullString `json:"route_id,omitempty" db:"route_id"`
	ConsumerID sql.NullString `json:"consumer_id,omitempty" db:"consumer_id"`

	// Config stores plugin-specific configuration as JSON
	Config map[string]interface{} `json:"config" db:"config"`

	Enabled   bool      `json:"enabled" db:"enabled"`
	Priority  int       `json:"priority" db:"priority"` // Lower = executes first
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// PluginScope constants define valid plugin scopes.
const (
	PluginScopeGlobal   = "global"
	PluginScopeService  = "service"
	PluginScopeRoute    = "route"
	PluginScopeConsumer = "consumer"
)

// ValidPluginScopes lists all valid plugin scopes.
var ValidPluginScopes = []string{
	PluginScopeGlobal,
	PluginScopeService,
	PluginScopeRoute,
	PluginScopeConsumer,
}
