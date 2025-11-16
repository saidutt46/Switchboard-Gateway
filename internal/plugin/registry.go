// Package plugin - Registry for managing plugin lifecycle and configuration
//
// The registry is responsible for:
//   - Registering plugin implementations (factory functions)
//   - Loading plugin configurations from database
//   - Creating plugin instances with their config
//   - Validating plugin configurations
//   - Managing plugin lifecycle
//
// Plugin Registration:
//
//	registry := NewRegistry()
//	registry.Register("auth", NewAuthPlugin)
//	registry.Register("rate-limit", NewRateLimitPlugin)
//
// Loading Plugins:
//
//	instances, err := registry.LoadFromDatabase(ctx, repo)
//	// Returns configured plugin instances ready to use
package plugin

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/saidutt46/switchboard-gateway/internal/database"
)

// PluginFactory is a function that creates a new plugin instance.
//
// The factory receives the raw JSON configuration from the database
// and returns a configured plugin instance.
//
// Example:
//
//	func NewAuthPlugin(config json.RawMessage) (Plugin, error) {
//	    var cfg AuthConfig
//	    if err := json.Unmarshal(config, &cfg); err != nil {
//	        return nil, err
//	    }
//	    return &AuthPlugin{config: cfg}, nil
//	}
type PluginFactory func(config json.RawMessage) (Plugin, error)

// Registry manages plugin registration and instantiation.
type Registry struct {
	// factories maps plugin names to their factory functions
	factories map[string]PluginFactory

	// instances holds all loaded plugin instances
	instances []PluginInstance
}

// NewRegistry creates a new plugin registry.
func NewRegistry() *Registry {
	return &Registry{
		factories: make(map[string]PluginFactory),
		instances: make([]PluginInstance, 0),
	}
}

// Register registers a plugin factory function.
//
// The name must match the plugin name in the database.
// The factory will be called to create plugin instances.
//
// Example:
//
//	registry.Register("auth", NewAuthPlugin)
//	registry.Register("rate-limit", NewRateLimitPlugin)
//	registry.Register("cors", NewCORSPlugin)
func (r *Registry) Register(name string, factory PluginFactory) {
	if _, exists := r.factories[name]; exists {
		log.Warn().
			Str("component", "plugin_registry").
			Str("plugin", name).
			Msg("Plugin factory already registered - overwriting")
	}

	r.factories[name] = factory

	log.Debug().
		Str("component", "plugin_registry").
		Str("plugin", name).
		Msg("Plugin factory registered")
}

// IsRegistered checks if a plugin factory is registered.
func (r *Registry) IsRegistered(name string) bool {
	_, exists := r.factories[name]
	return exists
}

// GetRegisteredPlugins returns all registered plugin names.
func (r *Registry) GetRegisteredPlugins() []string {
	names := make([]string, 0, len(r.factories))
	for name := range r.factories {
		names = append(names, name)
	}
	return names
}

// LoadFromDatabase loads all enabled plugins from the database.
//
// This method:
//  1. Queries database for enabled plugins
//  2. Creates plugin instances using registered factories
//  3. Validates plugin configurations
//  4. Returns plugin instances ready for chain execution
//
// Plugins without registered factories are skipped with a warning.
func (r *Registry) LoadFromDatabase(ctx context.Context, repo *database.Repository) ([]PluginInstance, error) {
	log.Info().
		Str("component", "plugin_registry").
		Msg("Loading plugins from database")

	// Get all enabled plugins from database
	pluginConfigs, err := repo.GetPlugins(ctx, true) // true = enabled only
	if err != nil {
		return nil, fmt.Errorf("failed to query plugins: %w", err)
	}

	if len(pluginConfigs) == 0 {
		log.Info().
			Str("component", "plugin_registry").
			Msg("No enabled plugins found in database")
		return []PluginInstance{}, nil
	}

	log.Info().
		Str("component", "plugin_registry").
		Int("count", len(pluginConfigs)).
		Msg("Found enabled plugins in database")

	// Create plugin instances
	instances := make([]PluginInstance, 0, len(pluginConfigs))

	for _, config := range pluginConfigs {
		instance, err := r.createInstance(config)
		if err != nil {
			// Log error but continue loading other plugins
			log.Error().
				Err(err).
				Str("component", "plugin_registry").
				Str("plugin", config.Name).
				Str("plugin_id", config.ID).
				Msg("Failed to create plugin instance - skipping")
			continue
		}

		instances = append(instances, instance)

		log.Info().
			Str("component", "plugin_registry").
			Str("plugin", config.Name).
			Str("scope", config.Scope).
			Int("priority", config.Priority).
			Bool("critical", instance.Critical).
			Msg("Plugin instance created successfully")
	}

	// Store instances
	r.instances = instances

	log.Info().
		Str("component", "plugin_registry").
		Int("total_configs", len(pluginConfigs)).
		Int("loaded", len(instances)).
		Int("failed", len(pluginConfigs)-len(instances)).
		Msg("Plugin loading completed")

	return instances, nil
}

// createInstance creates a plugin instance from database configuration.
func (r *Registry) createInstance(config *database.Plugin) (PluginInstance, error) {
	// Check if factory is registered
	factory, exists := r.factories[config.Name]
	if !exists {
		return PluginInstance{}, fmt.Errorf(
			"no factory registered for plugin '%s' (available: %v)",
			config.Name,
			r.GetRegisteredPlugins(),
		)
	}

	// Parse plugin config JSON
	var configJSON json.RawMessage
	if config.Config != nil {
		// Marshal JSONB (map[string]interface{}) to JSON bytes
		configBytes, err := json.Marshal(config.Config)
		if err != nil {
			return PluginInstance{}, fmt.Errorf("failed to marshal plugin config to JSON: %w", err)
		}
		configJSON = json.RawMessage(configBytes)
	} else {
		// Empty config
		configJSON = json.RawMessage("{}")
	}

	// Create plugin instance using factory
	plugin, err := factory(configJSON)
	if err != nil {
		return PluginInstance{}, fmt.Errorf("factory failed to create plugin: %w", err)
	}

	// Verify plugin name matches
	if plugin.Name() != config.Name {
		log.Warn().
			Str("component", "plugin_registry").
			Str("expected", config.Name).
			Str("actual", plugin.Name()).
			Msg("Plugin name mismatch")
	}

	// Parse critical flag from config JSON
	critical := r.parseCriticalFlag(configJSON)

	// Create plugin instance
	instance := PluginInstance{
		Plugin:   plugin,
		Config:   config,
		Scope:    config.Scope,
		Priority: config.Priority,
		Critical: critical,
	}

	// Validate instance
	if err := r.validateInstance(instance); err != nil {
		return PluginInstance{}, fmt.Errorf("plugin validation failed: %w", err)
	}

	return instance, nil
}

// parseCriticalFlag extracts the "critical" flag from plugin config JSON.
//
// Config example:
//
//	{
//	  "critical": true,
//	  "api_key": "secret"
//	}
//
// If "critical" is not specified, defaults to false (non-critical).
func (r *Registry) parseCriticalFlag(configJSON json.RawMessage) bool {
	var config struct {
		Critical bool `json:"critical"`
	}

	if err := json.Unmarshal(configJSON, &config); err != nil {
		log.Debug().
			Err(err).
			Str("component", "plugin_registry").
			Msg("Failed to parse critical flag - defaulting to false")
		return false
	}

	return config.Critical
}

// validateInstance validates a plugin instance configuration.
func (r *Registry) validateInstance(instance PluginInstance) error {
	// Validate plugin name
	if instance.Plugin.Name() == "" {
		return fmt.Errorf("plugin name cannot be empty")
	}

	// Validate scope
	validScopes := []string{
		database.PluginScopeGlobal,
		database.PluginScopeService,
		database.PluginScopeRoute,
		database.PluginScopeConsumer,
	}

	validScope := false
	for _, scope := range validScopes {
		if instance.Scope == scope {
			validScope = true
			break
		}
	}

	if !validScope {
		return fmt.Errorf("invalid plugin scope '%s' (must be one of: %v)", instance.Scope, validScopes)
	}

	// Validate priority (should be positive)
	if instance.Priority < 0 {
		log.Warn().
			Str("component", "plugin_registry").
			Str("plugin", instance.Plugin.Name()).
			Int("priority", instance.Priority).
			Msg("Plugin has negative priority - this may cause unexpected ordering")
	}

	// Validate scope-specific requirements
	switch instance.Scope {
	case database.PluginScopeService:
		if !instance.Config.ServiceID.Valid {
			return fmt.Errorf("service-scoped plugin must have a service_id")
		}

	case database.PluginScopeRoute:
		if !instance.Config.RouteID.Valid {
			return fmt.Errorf("route-scoped plugin must have a route_id")
		}

	case database.PluginScopeConsumer:
		if !instance.Config.ConsumerID.Valid {
			return fmt.Errorf("consumer-scoped plugin must have a consumer_id")
		}
	}

	return nil
}

// GetInstances returns all loaded plugin instances.
func (r *Registry) GetInstances() []PluginInstance {
	return r.instances
}

// GetInstancesByScope returns plugin instances filtered by scope.
func (r *Registry) GetInstancesByScope(scope string) []PluginInstance {
	instances := make([]PluginInstance, 0)

	for _, instance := range r.instances {
		if instance.Scope == scope {
			instances = append(instances, instance)
		}
	}

	return instances
}

// Count returns the number of loaded plugin instances.
func (r *Registry) Count() int {
	return len(r.instances)
}

// Stats returns statistics about the registry.
func (r *Registry) Stats() map[string]interface{} {
	globalCount := 0
	serviceCount := 0
	routeCount := 0
	consumerCount := 0
	criticalCount := 0

	for _, instance := range r.instances {
		switch instance.Scope {
		case database.PluginScopeGlobal:
			globalCount++
		case database.PluginScopeService:
			serviceCount++
		case database.PluginScopeRoute:
			routeCount++
		case database.PluginScopeConsumer:
			consumerCount++
		}

		if instance.Critical {
			criticalCount++
		}
	}

	return map[string]interface{}{
		"registered_factories": len(r.factories),
		"loaded_instances":     len(r.instances),
		"global_plugins":       globalCount,
		"service_plugins":      serviceCount,
		"route_plugins":        routeCount,
		"consumer_plugins":     consumerCount,
		"critical_plugins":     criticalCount,
	}
}

// Reload reloads all plugins from the database.
//
// This clears existing instances and loads fresh configurations.
// Used during hot reload when plugin configurations change.
func (r *Registry) Reload(ctx context.Context, repo *database.Repository) error {
	log.Info().
		Str("component", "plugin_registry").
		Msg("Reloading plugins from database")

	// Clear existing instances
	r.instances = make([]PluginInstance, 0)

	// Load fresh instances
	instances, err := r.LoadFromDatabase(ctx, repo)
	if err != nil {
		return fmt.Errorf("failed to reload plugins: %w", err)
	}

	r.instances = instances

	log.Info().
		Str("component", "plugin_registry").
		Int("loaded", len(instances)).
		Msg("Plugins reloaded successfully")

	return nil
}

// Clear removes all plugin instances (keeps factories registered).
func (r *Registry) Clear() {
	r.instances = make([]PluginInstance, 0)

	log.Debug().
		Str("component", "plugin_registry").
		Msg("Plugin instances cleared")
}

// ValidatePluginConfig validates a plugin configuration before saving to database.
//
// This is useful for Admin API to validate plugin configs before insertion.
func (r *Registry) ValidatePluginConfig(pluginName string, configJSON json.RawMessage) error {
	// Check if plugin is registered
	factory, exists := r.factories[pluginName]
	if !exists {
		return fmt.Errorf(
			"unknown plugin '%s' (registered plugins: %v)",
			pluginName,
			r.GetRegisteredPlugins(),
		)
	}

	// Try to create instance with the config
	_, err := factory(configJSON)
	if err != nil {
		return fmt.Errorf("invalid plugin configuration: %w", err)
	}

	log.Debug().
		Str("component", "plugin_registry").
		Str("plugin", pluginName).
		Msg("Plugin configuration validated successfully")

	return nil
}
