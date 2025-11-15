// Package proxy provides HTTP reverse proxy functionality.
//
// The proxy forwards requests from clients to backend services,
// handling connection pooling, timeouts, and error handling.
package proxy

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
)

// TransportConfig holds configuration for the HTTP transport.
type TransportConfig struct {
	// Connection pool settings
	MaxIdleConns        int
	MaxIdleConnsPerHost int
	MaxConnsPerHost     int

	// Timeouts
	DialTimeout           time.Duration
	KeepAlive             time.Duration
	IdleConnTimeout       time.Duration
	TLSHandshakeTimeout   time.Duration
	ResponseHeaderTimeout time.Duration
	ExpectContinueTimeout time.Duration

	// TLS
	InsecureSkipVerify bool
}

// DefaultTransportConfig returns a production-ready transport configuration.
func DefaultTransportConfig() *TransportConfig {
	return &TransportConfig{
		// Connection pool
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		MaxConnsPerHost:     100,

		// Timeouts
		DialTimeout:           10 * time.Second,
		KeepAlive:             30 * time.Second,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,

		// TLS - verify certificates by default
		InsecureSkipVerify: false,
	}
}

// NewTransport creates a new HTTP transport with the given configuration.
//
// The transport handles:
//   - Connection pooling (reuses connections to reduce latency)
//   - Timeouts (prevents hanging requests)
//   - Keep-alive (maintains persistent connections)
//   - TLS configuration (for HTTPS backends)
func NewTransport(cfg *TransportConfig) *http.Transport {
	if cfg == nil {
		cfg = DefaultTransportConfig()
	}

	transport := &http.Transport{
		// Connection pool settings
		MaxIdleConns:        cfg.MaxIdleConns,
		MaxIdleConnsPerHost: cfg.MaxIdleConnsPerHost,
		MaxConnsPerHost:     cfg.MaxConnsPerHost,

		// Idle connection timeout
		IdleConnTimeout: cfg.IdleConnTimeout,

		// Dialer for establishing connections
		DialContext: (&net.Dialer{
			Timeout:   cfg.DialTimeout,
			KeepAlive: cfg.KeepAlive,
		}).DialContext,

		// TLS configuration
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: cfg.InsecureSkipVerify,
			MinVersion:         tls.VersionTLS12,
		},
		TLSHandshakeTimeout: cfg.TLSHandshakeTimeout,

		// Response timeouts
		ResponseHeaderTimeout: cfg.ResponseHeaderTimeout,
		ExpectContinueTimeout: cfg.ExpectContinueTimeout,

		// Disable compression (let client/server handle it)
		DisableCompression: true,

		// Force HTTP/2 (if backend supports it)
		ForceAttemptHTTP2: true,
	}

	log.Info().
		Str("component", "proxy").
		Int("max_idle_conns", cfg.MaxIdleConns).
		Int("max_idle_conns_per_host", cfg.MaxIdleConnsPerHost).
		Dur("idle_conn_timeout", cfg.IdleConnTimeout).
		Msg("HTTP transport configured")

	return transport
}
