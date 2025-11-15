// Package proxy - Reverse proxy implementation
package proxy

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/saidutt46/switchboard-gateway/internal/database"
	"github.com/saidutt46/switchboard-gateway/internal/router"
)

// Proxy handles reverse proxying requests to backend services.
type Proxy struct {
	router    *router.Router
	transport *http.Transport
}

// NewProxy creates a new reverse proxy with the given router and transport.
func NewProxy(r *router.Router, transport *http.Transport) *Proxy {
	if transport == nil {
		transport = NewTransport(nil)
	}

	return &Proxy{
		router:    r,
		transport: transport,
	}
}

// ServeHTTP implements http.Handler.
//
// This is the main entry point for all proxied requests.
func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Generate request ID
	requestID := generateRequestID()

	// Add request ID to response header
	w.Header().Set("X-Request-ID", requestID)

	// Match the request to a route
	match, err := p.router.Match(r)
	if err != nil {
		// No route found
		log.Debug().
			Str("component", "proxy").
			Str("request_id", requestID).
			Str("path", r.URL.Path).
			Str("method", r.Method).
			Msg("No route matched")

		http.Error(w, `{"error":"not found","message":"No route configured for this path"}`, http.StatusNotFound)
		return
	}

	// Log the matched route
	log.Info().
		Str("component", "proxy").
		Str("request_id", requestID).
		Str("method", r.Method).
		Str("path", r.URL.Path).
		Str("route_id", match.Route.ID).
		Str("service_id", match.Service.ID).
		Str("service_name", match.Service.Name).
		Msg("Request matched to route")

	// Get the first target from the service
	// TODO: Phase 11 - Use load balancer to select target
	targetURL, err := p.getTargetURL(match.Service)
	if err != nil {
		log.Error().
			Err(err).
			Str("component", "proxy").
			Str("request_id", requestID).
			Str("service_id", match.Service.ID).
			Msg("Failed to get target URL")

		http.Error(w, `{"error":"service unavailable","message":"Backend service not available"}`, http.StatusServiceUnavailable)
		return
	}

	// Build the upstream URL
	upstreamURL := p.buildUpstreamURL(targetURL, r, match)

	log.Debug().
		Str("component", "proxy").
		Str("request_id", requestID).
		Str("upstream_url", upstreamURL).
		Msg("Proxying request to upstream")

	// Proxy the request
	if err := p.proxyRequest(w, r, upstreamURL, match, requestID); err != nil {
		log.Error().
			Err(err).
			Str("component", "proxy").
			Str("request_id", requestID).
			Str("upstream_url", upstreamURL).
			Msg("Proxy request failed")

		// Only write error if headers haven't been sent
		if !isHeadersSent(w) {
			http.Error(w, `{"error":"bad gateway","message":"Failed to proxy request to backend"}`, http.StatusBadGateway)
		}
		return
	}

	// Log successful proxy
	latency := time.Since(start)
	log.Info().
		Str("component", "proxy").
		Str("request_id", requestID).
		Dur("latency_ms", latency).
		Str("upstream_url", upstreamURL).
		Msg("Request proxied successfully")
}

// getTargetURL gets the target URL for a service.
//
// For now, we construct it from the service host/port.
// In Phase 11, we'll use service_targets table for load balancing.
func (p *Proxy) getTargetURL(service *database.Service) (string, error) {
	// Build target URL from service
	scheme := service.Protocol
	if scheme == "" {
		scheme = "http"
	}

	host := service.Host
	port := service.Port

	// Build URL
	var targetURL string
	if port == 80 && scheme == "http" {
		targetURL = fmt.Sprintf("%s://%s", scheme, host)
	} else if port == 443 && scheme == "https" {
		targetURL = fmt.Sprintf("%s://%s", scheme, host)
	} else {
		targetURL = fmt.Sprintf("%s://%s:%d", scheme, host, port)
	}

	// Add service path if present
	if service.Path.Valid && service.Path.String != "" {
		targetURL += service.Path.String
	}

	return targetURL, nil
}

// buildUpstreamURL builds the full upstream URL for the request.
func (p *Proxy) buildUpstreamURL(targetURL string, r *http.Request, match *router.MatchResult) string {
	path := r.URL.Path

	// Handle strip_path
	if match.Route.StripPath {
		// Remove the matched route path from the request path
		for _, routePath := range match.Route.Paths {
			// Simple strip - just remove the prefix
			// TODO: More sophisticated stripping for parameters
			if strings.HasPrefix(path, routePath) {
				path = strings.TrimPrefix(path, routePath)
				break
			}
		}
	}

	// Ensure path starts with /
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	// Build full URL
	upstreamURL := targetURL + path

	// Add query string if present
	if r.URL.RawQuery != "" {
		upstreamURL += "?" + r.URL.RawQuery
	}

	return upstreamURL
}

// proxyRequest performs the actual HTTP request to the upstream service.
func (p *Proxy) proxyRequest(w http.ResponseWriter, r *http.Request, upstreamURL string, match *router.MatchResult, requestID string) error {
	// Parse upstream URL
	targetURL, err := url.Parse(upstreamURL)
	if err != nil {
		return fmt.Errorf("invalid upstream URL: %w", err)
	}

	// Create upstream request
	upstreamReq, err := http.NewRequestWithContext(r.Context(), r.Method, targetURL.String(), r.Body)
	if err != nil {
		return fmt.Errorf("failed to create upstream request: %w", err)
	}

	// Copy headers from original request
	p.copyHeaders(upstreamReq.Header, r.Header)

	// Add/modify proxy headers
	p.setProxyHeaders(upstreamReq, r, match, requestID)

	// Create HTTP client with our transport
	client := &http.Client{
		Transport: p.transport,
		Timeout:   time.Duration(match.Service.ReadTimeoutMs) * time.Millisecond,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Don't follow redirects - return them to client
			return http.ErrUseLastResponse
		},
	}

	// Perform the request
	upstreamStart := time.Now()
	resp, err := client.Do(upstreamReq)
	if err != nil {
		return fmt.Errorf("upstream request failed: %w", err)
	}
	defer resp.Body.Close()

	upstreamLatency := time.Since(upstreamStart)

	log.Debug().
		Str("component", "proxy").
		Str("request_id", requestID).
		Int("status_code", resp.StatusCode).
		Dur("upstream_latency_ms", upstreamLatency).
		Msg("Received response from upstream")

	// Copy response headers
	p.copyHeaders(w.Header(), resp.Header)

	// Add custom headers
	w.Header().Set("X-Upstream-Latency", fmt.Sprintf("%dms", upstreamLatency.Milliseconds()))

	// Write status code
	w.WriteHeader(resp.StatusCode)

	// Copy response body
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to copy response body: %w", err)
	}

	return nil
}

// copyHeaders copies HTTP headers from src to dst.
func (p *Proxy) copyHeaders(dst, src http.Header) {
	for key, values := range src {
		// Skip hop-by-hop headers
		if isHopByHopHeader(key) {
			continue
		}

		for _, value := range values {
			dst.Add(key, value)
		}
	}
}

// setProxyHeaders sets/modifies headers for the upstream request.
func (p *Proxy) setProxyHeaders(upstreamReq *http.Request, originalReq *http.Request, match *router.MatchResult, requestID string) {
	// X-Forwarded-For
	if clientIP := getClientIP(originalReq); clientIP != "" {
		if prior := upstreamReq.Header.Get("X-Forwarded-For"); prior != "" {
			upstreamReq.Header.Set("X-Forwarded-For", prior+", "+clientIP)
		} else {
			upstreamReq.Header.Set("X-Forwarded-For", clientIP)
		}
	}

	// X-Forwarded-Proto
	proto := "http"
	if originalReq.TLS != nil {
		proto = "https"
	}
	upstreamReq.Header.Set("X-Forwarded-Proto", proto)

	// X-Forwarded-Host
	upstreamReq.Header.Set("X-Forwarded-Host", originalReq.Host)

	// X-Real-IP
	if clientIP := getClientIP(originalReq); clientIP != "" {
		upstreamReq.Header.Set("X-Real-IP", clientIP)
	}

	// X-Request-ID
	upstreamReq.Header.Set("X-Request-ID", requestID)

	// Host header
	if !match.Route.PreserveHost {
		// Use upstream host
		upstreamReq.Host = upstreamReq.URL.Host
	} else {
		// Keep original host
		upstreamReq.Host = originalReq.Host
	}
}

// isHopByHopHeader checks if a header is hop-by-hop.
//
// Hop-by-hop headers should not be forwarded.
func isHopByHopHeader(header string) bool {
	hopByHopHeaders := map[string]bool{
		"Connection":          true,
		"Keep-Alive":          true,
		"Proxy-Authenticate":  true,
		"Proxy-Authorization": true,
		"Te":                  true,
		"Trailers":            true,
		"Transfer-Encoding":   true,
		"Upgrade":             true,
	}

	return hopByHopHeaders[http.CanonicalHeaderKey(header)]
}

// getClientIP extracts the client IP from the request.
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP
		if idx := strings.Index(xff, ","); idx > 0 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}

	// Check X-Real-IP
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}

	// Fall back to RemoteAddr
	if idx := strings.LastIndex(r.RemoteAddr, ":"); idx > 0 {
		return r.RemoteAddr[:idx]
	}

	return r.RemoteAddr
}

// generateRequestID generates a unique request ID.
//
// Format: req_<timestamp>_<random>
func generateRequestID() string {
	return fmt.Sprintf("req_%d_%d", time.Now().UnixNano(), time.Now().UnixNano()%1000000)
}

// isHeadersSent checks if response headers have been sent.
func isHeadersSent(w http.ResponseWriter) bool {
	// This is a simple check - in reality, once WriteHeader is called,
	// headers are sent. We can't reliably detect this without wrapping
	// the ResponseWriter, but this is good enough for now.
	return false
}
