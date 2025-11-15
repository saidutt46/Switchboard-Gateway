// Package health provides health check handlers for the gateway.
//
// Health checks are essential for:
//   - Load balancer health checks
//   - Kubernetes liveness/readiness probes
//   - Monitoring and alerting
package health

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/saidutt46/switchboard-gateway/internal/database"
)

// Handler provides HTTP handlers for health checks.
type Handler struct {
	db   *database.DB
	repo *database.Repository
}

// NewHandler creates a new health check handler.
func NewHandler(db *database.DB, repo *database.Repository) *Handler {
	return &Handler{
		db:   db,
		repo: repo,
	}
}

// HealthResponse represents the health check response.
type HealthResponse struct {
	Status   string                 `json:"status"` // "healthy" or "unhealthy"
	Version  string                 `json:"version,omitempty"`
	Uptime   string                 `json:"uptime,omitempty"`
	Database map[string]interface{} `json:"database"`
	Checks   map[string]CheckResult `json:"checks,omitempty"`
}

// CheckResult represents the result of an individual health check.
type CheckResult struct {
	Status  string `json:"status"` // "pass" or "fail"
	Message string `json:"message,omitempty"`
}

var startTime = time.Now()

// Health handles the /health endpoint.
//
// Returns detailed health information including:
//   - Overall status
//   - Database health
//   - Uptime
//
// Returns 200 if healthy, 503 if unhealthy.
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Check database health
	dbHealth := h.db.Health(ctx)

	// Determine overall status
	overallStatus := "healthy"
	statusCode := http.StatusOK

	if dbHealth["status"] != "healthy" {
		overallStatus = "unhealthy"
		statusCode = http.StatusServiceUnavailable
	}

	// Calculate uptime
	uptime := time.Since(startTime)

	// Build response
	response := HealthResponse{
		Status:   overallStatus,
		Uptime:   formatDuration(uptime),
		Database: dbHealth,
		Checks: map[string]CheckResult{
			"database": {
				Status:  getCheckStatus(dbHealth["status"]),
				Message: getCheckMessage(dbHealth),
			},
		},
	}

	// Log health check
	log.Debug().
		Str("component", "health").
		Str("status", overallStatus).
		Str("remote_addr", r.RemoteAddr).
		Msg("Health check requested")

	// Send response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Error().Err(err).Msg("Failed to encode health response")
	}
}

// Ready handles the /ready endpoint.
//
// This is specifically for Kubernetes readiness probes.
// Returns 200 if the gateway is ready to accept traffic, 503 otherwise.
//
// Currently checks:
//   - Database connectivity
//
// In future phases will check:
//   - Configuration loaded
//   - Routes initialized
//   - Plugins loaded
func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	// Check database connectivity
	if err := h.db.Ping(ctx); err != nil {
		log.Warn().
			Err(err).
			Str("component", "health").
			Msg("Readiness check failed: database not reachable")

		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"status":"not ready","reason":"database unavailable"}`))
		return
	}

	// TODO: Phase 3 - Check if routes are loaded
	// TODO: Phase 7 - Check if plugins are initialized

	log.Debug().
		Str("component", "health").
		Str("remote_addr", r.RemoteAddr).
		Msg("Readiness check passed")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ready"}`))
}

// getCheckStatus converts a health status to a check status.
func getCheckStatus(status interface{}) string {
	if s, ok := status.(string); ok && s == "healthy" {
		return "pass"
	}
	return "fail"
}

// getCheckMessage extracts a message from health check results.
func getCheckMessage(health map[string]interface{}) string {
	if err, ok := health["error"].(string); ok {
		return err
	}
	return "operational"
}

// formatDuration formats a duration in a human-readable way.
func formatDuration(d time.Duration) string {
	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm %ds", days, hours, minutes, seconds)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
	}
	if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}
