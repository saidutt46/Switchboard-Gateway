// Package plugin - Chain executor for running plugins in priority order
//
// The chain executor is responsible for:
//   - Loading applicable plugins for a request
//   - Sorting plugins by priority (lower = runs first)
//   - Executing plugins in the correct phase
//   - Handling plugin errors gracefully
//   - Short-circuiting when needed (abort or critical error)
//
// Chain Execution Order:
//
//	BeforeRequest Phase (ascending priority):
//	  1. Global plugins (priority 1, 2, 3...)
//	  2. Service plugins (priority 10, 20, 30...)
//	  3. Route plugins (priority 40, 50, 60...)
//
//	AfterResponse Phase (descending priority):
//	  1. Route plugins (priority 60, 50, 40...)
//	  2. Service plugins (priority 30, 20, 10...)
//	  3. Global plugins (priority 3, 2, 1...)
package plugin

import (
	"sort"

	"github.com/rs/zerolog/log"
	"github.com/saidutt46/switchboard-gateway/internal/database"
)

// Chain represents a collection of plugins to execute.
type Chain struct {
	plugins []PluginInstance
}

// PluginInstance combines a plugin with its configuration and metadata.
type PluginInstance struct {
	// Plugin is the actual plugin implementation
	Plugin Plugin

	// Config from database (plugin-specific configuration)
	Config *database.Plugin

	// Scope indicates where this plugin applies
	Scope string

	// Priority determines execution order (lower = earlier)
	Priority int

	// Critical indicates if plugin failure should stop the chain
	// Read from plugin config JSON: {"critical": true}
	Critical bool
}

// NewChain creates a new empty plugin chain.
func NewChain() *Chain {
	return &Chain{
		plugins: make([]PluginInstance, 0),
	}
}

// Add adds a plugin instance to the chain.
func (c *Chain) Add(instance PluginInstance) {
	c.plugins = append(c.plugins, instance)

	log.Debug().
		Str("component", "plugin_chain").
		Str("plugin", instance.Plugin.Name()).
		Str("scope", instance.Scope).
		Int("priority", instance.Priority).
		Bool("critical", instance.Critical).
		Msg("Plugin added to chain")
}

// Sort sorts plugins by priority (ascending order).
// Lower priority numbers execute first.
func (c *Chain) Sort() {
	sort.Slice(c.plugins, func(i, j int) bool {
		return c.plugins[i].Priority < c.plugins[j].Priority
	})

	log.Debug().
		Str("component", "plugin_chain").
		Int("count", len(c.plugins)).
		Msg("Plugin chain sorted by priority")
}

// Execute runs all plugins in the chain for the given phase.
//
// Execution rules:
//   - BeforeRequest: Execute in ascending priority order (1, 2, 3...)
//   - AfterResponse: Execute in descending priority order (3, 2, 1...)
//   - Stop on ctx.Abort() (e.g., auth failure)
//   - Stop on critical plugin error
//   - Continue on non-critical plugin error (just log it)
//
// Returns error if a critical plugin fails.
func (c *Chain) Execute(ctx *Context) error {
	if len(c.plugins) == 0 {
		log.Debug().
			Str("component", "plugin_chain").
			Str("phase", string(ctx.Phase)).
			Msg("No plugins to execute")
		return nil
	}

	log.Info().
		Str("component", "plugin_chain").
		Str("phase", string(ctx.Phase)).
		Int("plugin_count", len(c.plugins)).
		Str("route_id", ctx.Route.ID).
		Msg("Starting plugin chain execution")

	// Determine execution order based on phase
	plugins := c.getExecutionOrder(ctx.Phase)

	// Execute each plugin
	for _, instance := range plugins {
		// Check if chain was aborted by previous plugin
		if ctx.IsAborted() {
			log.Info().
				Str("component", "plugin_chain").
				Str("phase", string(ctx.Phase)).
				Str("aborted_by", "previous_plugin").
				Int("status_code", ctx.AbortStatusCode()).
				Msg("Chain execution stopped - request aborted")
			return nil
		}

		// Execute plugin
		if err := c.executePlugin(instance, ctx); err != nil {
			// Check if this is a critical error
			if instance.Critical {
				log.Error().
					Err(err).
					Str("component", "plugin_chain").
					Str("plugin", instance.Plugin.Name()).
					Str("phase", string(ctx.Phase)).
					Bool("critical", true).
					Msg("Critical plugin failed - stopping chain")

				return NewPluginError(
					instance.Plugin.Name(),
					ctx.Phase,
					err,
					true,
				)
			}

			// Non-critical error - log and continue
			log.Warn().
				Err(err).
				Str("component", "plugin_chain").
				Str("plugin", instance.Plugin.Name()).
				Str("phase", string(ctx.Phase)).
				Bool("critical", false).
				Msg("Plugin failed - continuing chain execution")
		}
	}

	log.Info().
		Str("component", "plugin_chain").
		Str("phase", string(ctx.Phase)).
		Int("executed", len(plugins)).
		Msg("Plugin chain execution completed")

	return nil
}

// getExecutionOrder returns plugins in the correct order for the phase.
//
// BeforeRequest: Ascending priority (1, 2, 3...)
// AfterResponse: Descending priority (3, 2, 1...)
func (c *Chain) getExecutionOrder(phase Phase) []PluginInstance {
	plugins := make([]PluginInstance, len(c.plugins))
	copy(plugins, c.plugins)

	if phase == PhaseAfterResponse {
		// Reverse order for after-response phase
		for i, j := 0, len(plugins)-1; i < j; i, j = i+1, j-1 {
			plugins[i], plugins[j] = plugins[j], plugins[i]
		}
	}

	return plugins
}

// executePlugin executes a single plugin and handles errors.
func (c *Chain) executePlugin(instance PluginInstance, ctx *Context) error {
	pluginName := instance.Plugin.Name()

	log.Debug().
		Str("component", "plugin_chain").
		Str("plugin", pluginName).
		Str("phase", string(ctx.Phase)).
		Int("priority", instance.Priority).
		Msg("Executing plugin")

	// Execute the plugin
	err := instance.Plugin.Execute(ctx)

	if err != nil {
		ctx.LogError(pluginName, err, "Plugin execution failed")
		return err
	}

	// Check if plugin aborted the request
	if ctx.IsAborted() {
		log.Info().
			Str("component", "plugin_chain").
			Str("plugin", pluginName).
			Int("status_code", ctx.AbortStatusCode()).
			Str("message", ctx.AbortMessage()).
			Msg("Plugin aborted the request")
	} else {
		log.Debug().
			Str("component", "plugin_chain").
			Str("plugin", pluginName).
			Msg("Plugin executed successfully")
	}

	return nil
}

// Count returns the number of plugins in the chain.
func (c *Chain) Count() int {
	return len(c.plugins)
}

// GetPlugins returns all plugin instances in the chain.
func (c *Chain) GetPlugins() []PluginInstance {
	return c.plugins
}

// Clear removes all plugins from the chain.
func (c *Chain) Clear() {
	c.plugins = make([]PluginInstance, 0)
	log.Debug().
		Str("component", "plugin_chain").
		Msg("Plugin chain cleared")
}

// ChainBuilder helps build plugin chains for specific requests.
type ChainBuilder struct {
	allPlugins []PluginInstance
}

// NewChainBuilder creates a new chain builder.
func NewChainBuilder(plugins []PluginInstance) *ChainBuilder {
	return &ChainBuilder{
		allPlugins: plugins,
	}
}

// BuildForRoute builds a plugin chain for a specific route.
//
// Includes plugins with scope:
//   - global (apply to all requests)
//   - service (match route's service)
//   - route (match this specific route)
func (cb *ChainBuilder) BuildForRoute(route *database.Route, service *database.Service) *Chain {
	chain := NewChain()

	for _, instance := range cb.allPlugins {
		// Check if plugin applies to this request
		if cb.shouldInclude(instance, route, service) {
			chain.Add(instance)
		}
	}

	// Sort by priority
	chain.Sort()

	log.Debug().
		Str("component", "chain_builder").
		Str("route_id", route.ID).
		Str("service_id", service.ID).
		Int("plugin_count", chain.Count()).
		Msg("Plugin chain built for route")

	return chain
}

// shouldInclude determines if a plugin should be included in the chain.
func (cb *ChainBuilder) shouldInclude(
	instance PluginInstance,
	route *database.Route,
	service *database.Service,
) bool {
	switch instance.Scope {
	case database.PluginScopeGlobal:
		// Global plugins apply to all requests
		return true

	case database.PluginScopeService:
		// Service plugins apply to requests for that service
		if instance.Config.ServiceID.Valid {
			return instance.Config.ServiceID.String == service.ID
		}
		return false

	case database.PluginScopeRoute:
		// Route plugins apply to that specific route
		if instance.Config.RouteID.Valid {
			return instance.Config.RouteID.String == route.ID
		}
		return false

	case database.PluginScopeConsumer:
		// Consumer plugins - will implement in future phase
		// For now, skip consumer-scoped plugins
		return false

	default:
		log.Warn().
			Str("component", "chain_builder").
			Str("scope", instance.Scope).
			Str("plugin", instance.Plugin.Name()).
			Msg("Unknown plugin scope - excluding from chain")
		return false
	}
}

// Stats returns statistics about the chain builder.
func (cb *ChainBuilder) Stats() map[string]interface{} {
	globalCount := 0
	serviceCount := 0
	routeCount := 0
	consumerCount := 0

	for _, instance := range cb.allPlugins {
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
	}

	return map[string]interface{}{
		"total_plugins":    len(cb.allPlugins),
		"global_plugins":   globalCount,
		"service_plugins":  serviceCount,
		"route_plugins":    routeCount,
		"consumer_plugins": consumerCount,
	}
}
