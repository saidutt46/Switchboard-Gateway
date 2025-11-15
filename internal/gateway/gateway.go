// Package gateway provides the main gateway logic and config change handling.
package gateway

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/saidutt46/switchboard-gateway/internal/config"
	"github.com/saidutt46/switchboard-gateway/internal/database"
	"github.com/saidutt46/switchboard-gateway/internal/router"
)

// Gateway handles HTTP proxying and config changes.
type Gateway struct {
	router *router.Router
	repo   *database.Repository
}

// New creates a new Gateway instance.
func New(router *router.Router, repo *database.Repository) *Gateway {
	return &Gateway{
		router: router,
		repo:   repo,
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

	// Use the router's existing Reload method
	// It loads routes AND services from DB and does atomic swap
	ctx := context.Background()
	if err := g.router.Reload(ctx, g.repo); err != nil {
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

	// Reload router (includes services)
	ctx := context.Background()
	if err := g.router.Reload(ctx, g.repo); err != nil {
		log.Error().
			Err(err).
			Msg("Failed to reload services")
		return err
	}

	log.Info().Msg("Service configuration reloaded successfully")

	return nil
}

func (g *Gateway) handlePluginChange(event config.ConfigChangeEvent) error {
	// For now, just log it
	// In Phase 7, we'll reload plugin configurations
	log.Info().
		Str("action", event.Action).
		Str("plugin_id", event.EntityID).
		Msg("Plugin change detected (no action needed yet)")
	return nil
}
