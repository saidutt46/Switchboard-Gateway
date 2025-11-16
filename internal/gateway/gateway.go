// Package gateway provides the main gateway logic and config change handling.
package gateway

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/saidutt46/switchboard-gateway/internal/config"
	"github.com/saidutt46/switchboard-gateway/internal/database"
	"github.com/saidutt46/switchboard-gateway/internal/plugin" // ADD THIS
	"github.com/saidutt46/switchboard-gateway/internal/router"
)

// Gateway handles HTTP proxying and config changes.
type Gateway struct {
	router   *router.Router
	repo     *database.Repository
	registry *plugin.Registry
}

// New creates a new Gateway instance.
func New(router *router.Router, repo *database.Repository, registry *plugin.Registry) *Gateway {
	return &Gateway{
		router:   router,
		repo:     repo,
		registry: registry,
	}
}

// HandleConfigChange handles configuration change events from Admin API.
// This implements the config.ConfigChangeHandler interface.
func (g *Gateway) HandleConfigChange(event config.ConfigChangeEvent) error {
	log.Info().
		Str("entity_type", event.EntityType).
		Str("entity_id", event.EntityID).
		Str("action", event.Action).
		Msg("Handling config change")

	switch event.EntityType {
	case "route":
		return g.handleRouteChange(event)
	case "service":
		return g.handleServiceChange(event)
	case "plugin":
		return g.handlePluginChange(event)
	default:
		log.Warn().
			Str("entity_type", event.EntityType).
			Msg("Unknown entity type")
		return nil
	}
}

func (g *Gateway) handleRouteChange(event config.ConfigChangeEvent) error {
	log.Info().
		Str("action", event.Action).
		Str("route_id", event.EntityID).
		Msg("Route change detected - reloading configuration")

	ctx := context.Background()

	// Reload plugins first
	var pluginInstances []plugin.PluginInstance
	if g.registry != nil {
		if err := g.registry.Reload(ctx, g.repo); err != nil {
			log.Error().
				Err(err).
				Msg("Failed to reload plugins - continuing with empty plugins")
			pluginInstances = []plugin.PluginInstance{}
		} else {
			pluginInstances = g.registry.GetInstances()
		}
	} else {
		pluginInstances = []plugin.PluginInstance{}
	}

	// Reload router with new plugins
	if err := g.router.Reload(ctx, g.repo, pluginInstances); err != nil {
		log.Error().
			Err(err).
			Msg("Failed to reload routes")
		return err
	}

	log.Info().Msg("Route configuration reloaded successfully")

	return nil
}

func (g *Gateway) handleServiceChange(event config.ConfigChangeEvent) error {
	log.Info().
		Str("action", event.Action).
		Str("service_id", event.EntityID).
		Msg("Service change detected - reloading configuration")

	ctx := context.Background()

	// Reload plugins first
	var pluginInstances []plugin.PluginInstance
	if g.registry != nil {
		if err := g.registry.Reload(ctx, g.repo); err != nil {
			log.Error().
				Err(err).
				Msg("Failed to reload plugins - continuing with empty plugins")
			pluginInstances = []plugin.PluginInstance{}
		} else {
			pluginInstances = g.registry.GetInstances()
		}
	} else {
		pluginInstances = []plugin.PluginInstance{}
	}

	// Reload router with new plugins
	if err := g.router.Reload(ctx, g.repo, pluginInstances); err != nil {
		log.Error().
			Err(err).
			Msg("Failed to reload services")
		return err
	}

	log.Info().Msg("Service configuration reloaded successfully")

	return nil
}

func (g *Gateway) handlePluginChange(event config.ConfigChangeEvent) error {
	log.Info().
		Str("action", event.Action).
		Str("plugin_id", event.EntityID).
		Msg("Plugin change detected - reloading configuration")

	ctx := context.Background()

	// Reload plugins
	var pluginInstances []plugin.PluginInstance
	if g.registry != nil {
		if err := g.registry.Reload(ctx, g.repo); err != nil {
			log.Error().
				Err(err).
				Msg("Failed to reload plugins")
			return err
		}
		pluginInstances = g.registry.GetInstances()

		log.Info().
			Int("plugin_count", len(pluginInstances)).
			Msg("Plugins reloaded successfully")
	} else {
		log.Warn().Msg("Plugin registry not available")
		pluginInstances = []plugin.PluginInstance{}
	}

	// Reload router with new plugins
	if err := g.router.Reload(ctx, g.repo, pluginInstances); err != nil {
		log.Error().
			Err(err).
			Msg("Failed to reload configuration after plugin change")
		return err
	}

	log.Info().Msg("Plugin configuration reloaded successfully")

	return nil
}
