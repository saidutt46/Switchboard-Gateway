// Package database - Repository layer
//
// This file implements the Repository pattern for database operations.
// All CRUD operations for gateway configuration (services, routes, consumers, etc.)
// are centralized here.
package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/rs/zerolog/log"
)

// Repository provides data access methods for all gateway entities.
//
// It encapsulates all database operations and provides a clean interface
// for the rest of the application.
type Repository struct {
	db *DB
}

// NewRepository creates a new repository instance.
func NewRepository(db *DB) *Repository {
	return &Repository{db: db}
}

// ============================================================================
// Services
// ============================================================================

// GetServices retrieves all services from the database.
//
// Only returns enabled services unless includeDisabled is true.
func (r *Repository) GetServices(ctx context.Context, includeDisabled bool) ([]*Service, error) {
	query := `
		SELECT id, name, protocol, host, port, path,
		       connect_timeout_ms, read_timeout_ms, write_timeout_ms, retries,
		       load_balancer_type, enabled, created_at, updated_at
		FROM services
		WHERE enabled = true OR $1 = true
		ORDER BY created_at DESC
	`

	rows, err := r.db.pool.QueryContext(ctx, query, includeDisabled)
	if err != nil {
		return nil, fmt.Errorf("failed to query services: %w", err)
	}
	defer rows.Close()

	var services []*Service
	for rows.Next() {
		var svc Service
		err := rows.Scan(
			&svc.ID, &svc.Name, &svc.Protocol, &svc.Host, &svc.Port, &svc.Path,
			&svc.ConnectTimeoutMs, &svc.ReadTimeoutMs, &svc.WriteTimeoutMs, &svc.Retries,
			&svc.LoadBalancerType, &svc.Enabled, &svc.CreatedAt, &svc.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan service: %w", err)
		}
		services = append(services, &svc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating services: %w", err)
	}

	log.Debug().
		Str("component", "repository").
		Int("count", len(services)).
		Bool("include_disabled", includeDisabled).
		Msg("Retrieved services")

	return services, nil
}

// GetServiceByID retrieves a service by its ID.
//
// Returns sql.ErrNoRows if the service doesn't exist.
func (r *Repository) GetServiceByID(ctx context.Context, id string) (*Service, error) {
	query := `
		SELECT id, name, protocol, host, port, path,
		       connect_timeout_ms, read_timeout_ms, write_timeout_ms, retries,
		       load_balancer_type, enabled, created_at, updated_at
		FROM services
		WHERE id = $1
	`

	var svc Service
	err := r.db.pool.QueryRowContext(ctx, query, id).Scan(
		&svc.ID, &svc.Name, &svc.Protocol, &svc.Host, &svc.Port, &svc.Path,
		&svc.ConnectTimeoutMs, &svc.ReadTimeoutMs, &svc.WriteTimeoutMs, &svc.Retries,
		&svc.LoadBalancerType, &svc.Enabled, &svc.CreatedAt, &svc.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("service not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get service: %w", err)
	}

	return &svc, nil
}

// GetServiceByName retrieves a service by its name.
//
// Returns sql.ErrNoRows if the service doesn't exist.
func (r *Repository) GetServiceByName(ctx context.Context, name string) (*Service, error) {
	query := `
		SELECT id, name, protocol, host, port, path,
		       connect_timeout_ms, read_timeout_ms, write_timeout_ms, retries,
		       load_balancer_type, enabled, created_at, updated_at
		FROM services
		WHERE name = $1
	`

	var svc Service
	err := r.db.pool.QueryRowContext(ctx, query, name).Scan(
		&svc.ID, &svc.Name, &svc.Protocol, &svc.Host, &svc.Port, &svc.Path,
		&svc.ConnectTimeoutMs, &svc.ReadTimeoutMs, &svc.WriteTimeoutMs, &svc.Retries,
		&svc.LoadBalancerType, &svc.Enabled, &svc.CreatedAt, &svc.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("service not found: %s", name)
		}
		return nil, fmt.Errorf("failed to get service by name: %w", err)
	}

	return &svc, nil
}

// ============================================================================
// Routes
// ============================================================================

// GetRoutes retrieves all routes from the database.
//
// Only returns enabled routes unless includeDisabled is true.
func (r *Repository) GetRoutes(ctx context.Context, includeDisabled bool) ([]*Route, error) {
	query := `
		SELECT id, service_id, name, hosts, paths, methods,
		       strip_path, preserve_host, enabled, created_at, updated_at
		FROM routes
		WHERE enabled = true OR $1 = true
		ORDER BY created_at DESC
	`

	rows, err := r.db.pool.QueryContext(ctx, query, includeDisabled)
	if err != nil {
		return nil, fmt.Errorf("failed to query routes: %w", err)
	}
	defer rows.Close()

	var routes []*Route
	for rows.Next() {
		var route Route
		err := rows.Scan(
			&route.ID, &route.ServiceID, &route.Name, &route.Hosts, &route.Paths, &route.Methods,
			&route.StripPath, &route.PreserveHost, &route.Enabled, &route.CreatedAt, &route.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan route: %w", err)
		}
		routes = append(routes, &route)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating routes: %w", err)
	}

	log.Debug().
		Str("component", "repository").
		Int("count", len(routes)).
		Bool("include_disabled", includeDisabled).
		Msg("Retrieved routes")

	return routes, nil
}

// GetRouteByID retrieves a route by its ID.
//
// Returns sql.ErrNoRows if the route doesn't exist.
func (r *Repository) GetRouteByID(ctx context.Context, id string) (*Route, error) {
	query := `
		SELECT id, service_id, name, hosts, paths, methods,
		       strip_path, preserve_host, enabled, created_at, updated_at
		FROM routes
		WHERE id = $1
	`

	var route Route
	err := r.db.pool.QueryRowContext(ctx, query, id).Scan(
		&route.ID, &route.ServiceID, &route.Name, &route.Hosts, &route.Paths, &route.Methods,
		&route.StripPath, &route.PreserveHost, &route.Enabled, &route.CreatedAt, &route.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("route not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get route: %w", err)
	}

	return &route, nil
}

// GetRoutesByServiceID retrieves all routes for a specific service.
func (r *Repository) GetRoutesByServiceID(ctx context.Context, serviceID string) ([]*Route, error) {
	query := `
		SELECT id, service_id, name, hosts, paths, methods,
		       strip_path, preserve_host, enabled, created_at, updated_at
		FROM routes
		WHERE service_id = $1 AND enabled = true
		ORDER BY created_at DESC
	`

	rows, err := r.db.pool.QueryContext(ctx, query, serviceID)
	if err != nil {
		return nil, fmt.Errorf("failed to query routes by service: %w", err)
	}
	defer rows.Close()

	var routes []*Route
	for rows.Next() {
		var route Route
		err := rows.Scan(
			&route.ID, &route.ServiceID, &route.Name, &route.Hosts, &route.Paths, &route.Methods,
			&route.StripPath, &route.PreserveHost, &route.Enabled, &route.CreatedAt, &route.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan route: %w", err)
		}
		routes = append(routes, &route)
	}

	return routes, nil
}

// ============================================================================
// Consumers
// ============================================================================

// GetConsumerByID retrieves a consumer by its ID.
func (r *Repository) GetConsumerByID(ctx context.Context, id string) (*Consumer, error) {
	query := `
		SELECT id, username, email, custom_id, metadata, created_at, updated_at
		FROM consumers
		WHERE id = $1
	`

	var consumer Consumer
	var metadataJSON []byte

	err := r.db.pool.QueryRowContext(ctx, query, id).Scan(
		&consumer.ID, &consumer.Username, &consumer.Email, &consumer.CustomID,
		&metadataJSON, &consumer.CreatedAt, &consumer.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("consumer not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get consumer: %w", err)
	}

	// Parse metadata JSON
	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &consumer.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal consumer metadata: %w", err)
		}
	}

	return &consumer, nil
}

// GetConsumerByUsername retrieves a consumer by username.
func (r *Repository) GetConsumerByUsername(ctx context.Context, username string) (*Consumer, error) {
	query := `
		SELECT id, username, email, custom_id, metadata, created_at, updated_at
		FROM consumers
		WHERE username = $1
	`

	var consumer Consumer
	var metadataJSON []byte

	err := r.db.pool.QueryRowContext(ctx, query, username).Scan(
		&consumer.ID, &consumer.Username, &consumer.Email, &consumer.CustomID,
		&metadataJSON, &consumer.CreatedAt, &consumer.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("consumer not found: %s", username)
		}
		return nil, fmt.Errorf("failed to get consumer by username: %w", err)
	}

	// Parse metadata JSON
	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &consumer.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal consumer metadata: %w", err)
		}
	}

	return &consumer, nil
}

// GetConsumerByAPIKeyHash retrieves a consumer by API key hash.
//
// This is the critical path for API key authentication.
// Returns the consumer associated with the given key hash.
func (r *Repository) GetConsumerByAPIKeyHash(ctx context.Context, keyHash string) (*Consumer, error) {
	query := `
		SELECT c.id, c.username, c.email, c.custom_id, c.metadata, c.created_at, c.updated_at
		FROM consumers c
		INNER JOIN api_keys k ON c.id = k.consumer_id
		WHERE k.key_hash = $1 AND k.enabled = true
	`

	var consumer Consumer
	var metadataJSON []byte

	err := r.db.pool.QueryRowContext(ctx, query, keyHash).Scan(
		&consumer.ID, &consumer.Username, &consumer.Email, &consumer.CustomID,
		&metadataJSON, &consumer.CreatedAt, &consumer.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no consumer found for API key")
		}
		return nil, fmt.Errorf("failed to get consumer by API key: %w", err)
	}

	// Parse metadata JSON
	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &consumer.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal consumer metadata: %w", err)
		}
	}

	log.Debug().
		Str("component", "repository").
		Str("consumer_id", consumer.ID).
		Str("username", consumer.Username).
		Msg("Retrieved consumer by API key")

	return &consumer, nil
}

// ============================================================================
// Plugins
// ============================================================================

// GetPlugins retrieves all plugins from the database.
//
// Returns plugins ordered by priority (lower = executes first).
func (r *Repository) GetPlugins(ctx context.Context, enabledOnly bool) ([]*Plugin, error) {
	query := `
		SELECT id, name, scope, service_id, route_id, consumer_id,
		       config, enabled, priority, created_at, updated_at
		FROM plugins
		WHERE enabled = true OR $1 = false
		ORDER BY priority ASC, created_at ASC
	`

	rows, err := r.db.pool.QueryContext(ctx, query, enabledOnly)
	if err != nil {
		return nil, fmt.Errorf("failed to query plugins: %w", err)
	}
	defer rows.Close()

	var plugins []*Plugin
	for rows.Next() {
		var plugin Plugin
		var configJSON []byte

		err := rows.Scan(
			&plugin.ID, &plugin.Name, &plugin.Scope, &plugin.ServiceID, &plugin.RouteID, &plugin.ConsumerID,
			&configJSON, &plugin.Enabled, &plugin.Priority, &plugin.CreatedAt, &plugin.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan plugin: %w", err)
		}

		// Parse config JSON
		if len(configJSON) > 0 {
			if err := json.Unmarshal(configJSON, &plugin.Config); err != nil {
				return nil, fmt.Errorf("failed to unmarshal plugin config: %w", err)
			}
		}

		plugins = append(plugins, &plugin)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating plugins: %w", err)
	}

	log.Debug().
		Str("component", "repository").
		Int("count", len(plugins)).
		Bool("enabled_only", enabledOnly).
		Msg("Retrieved plugins")

	return plugins, nil
}

// GetPluginsByRouteID retrieves all plugins for a specific route.
//
// This includes:
//   - Global plugins (scope = 'global')
//   - Service-level plugins (for the route's service)
//   - Route-specific plugins
//
// Returns plugins ordered by priority.
func (r *Repository) GetPluginsByRouteID(ctx context.Context, routeID string) ([]*Plugin, error) {
	// First, get the route to find its service_id
	route, err := r.GetRouteByID(ctx, routeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get route: %w", err)
	}

	query := `
		SELECT id, name, scope, service_id, route_id, consumer_id,
		       config, enabled, priority, created_at, updated_at
		FROM plugins
		WHERE enabled = true
		  AND (
		      scope = 'global'
		      OR (scope = 'service' AND service_id = $1)
		      OR (scope = 'route' AND route_id = $2)
		  )
		ORDER BY priority ASC, created_at ASC
	`

	rows, err := r.db.pool.QueryContext(ctx, query, route.ServiceID, routeID)
	if err != nil {
		return nil, fmt.Errorf("failed to query plugins for route: %w", err)
	}
	defer rows.Close()

	var plugins []*Plugin
	for rows.Next() {
		var plugin Plugin
		var configJSON []byte

		err := rows.Scan(
			&plugin.ID, &plugin.Name, &plugin.Scope, &plugin.ServiceID, &plugin.RouteID, &plugin.ConsumerID,
			&configJSON, &plugin.Enabled, &plugin.Priority, &plugin.CreatedAt, &plugin.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan plugin: %w", err)
		}

		// Parse config JSON
		if len(configJSON) > 0 {
			if err := json.Unmarshal(configJSON, &plugin.Config); err != nil {
				return nil, fmt.Errorf("failed to unmarshal plugin config: %w", err)
			}
		}

		plugins = append(plugins, &plugin)
	}

	return plugins, nil
}

// GetServiceTargets retrieves all targets for a specific service.
func (r *Repository) GetServiceTargets(ctx context.Context, serviceID string) ([]*ServiceTarget, error) {
	query := `
		SELECT id, service_id, target, weight, health_check_path, enabled, created_at
		FROM service_targets
		WHERE service_id = $1 AND enabled = true
		ORDER BY created_at ASC
	`

	rows, err := r.db.pool.QueryContext(ctx, query, serviceID)
	if err != nil {
		return nil, fmt.Errorf("failed to query service targets: %w", err)
	}
	defer rows.Close()

	var targets []*ServiceTarget
	for rows.Next() {
		var target ServiceTarget
		err := rows.Scan(
			&target.ID, &target.ServiceID, &target.Target, &target.Weight,
			&target.HealthCheckPath, &target.Enabled, &target.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan service target: %w", err)
		}
		targets = append(targets, &target)
	}

	return targets, nil
}
