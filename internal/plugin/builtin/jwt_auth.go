// Package builtin - JWT Authentication Plugin
//
// This plugin validates JSON Web Tokens (JWT) from headers or cookies
// and extracts user information from token claims.
//
// Features:
//   - Extract tokens from Authorization Bearer header or cookies
//   - Validate JWT signature using HS256 algorithm
//   - Validate standard claims (exp, nbf, iss, aud)
//   - Redis caching for validated tokens
//   - User context injection
//   - Configurable bypass paths (health checks)
//
// Configuration Example:
//
//	{
//	  "critical": false,
//	  "secret_key": "your-secret-key-here",
//	  "algorithm": "HS256",
//	  "cookie_name": "jwt_token",
//	  "header_name": "Authorization",
//	  "issuer": "your-api-gateway",
//	  "audience": "your-api",
//	  "cache_ttl_seconds": 300,
//	  "bypass_paths": ["/health", "/ready", "/metrics"]
//	}
package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
	"github.com/saidutt46/switchboard-gateway/internal/database"
	"github.com/saidutt46/switchboard-gateway/internal/plugin"
)

// JWTAuthPlugin validates JWT tokens and loads user information.
type JWTAuthPlugin struct {
	config JWTAuthConfig
	redis  *redis.Client
}

// JWTAuthConfig holds configuration for JWT authentication.
type JWTAuthConfig struct {
	// Critical indicates if auth failure should stop the request
	Critical bool `json:"critical"`

	// SecretKey is the HMAC secret key for HS256 algorithm
	// IMPORTANT: Keep this secret and rotate it regularly
	SecretKey string `json:"secret_key"`

	// Algorithm specifies the signing algorithm (currently only HS256 supported)
	Algorithm string `json:"algorithm"`

	// CookieName is the name of the cookie containing the JWT
	// Default: "jwt_token"
	CookieName string `json:"cookie_name"`

	// HeaderName is the header containing the JWT
	// Default: "Authorization" (expects "Bearer <token>")
	HeaderName string `json:"header_name"`

	// Issuer is the expected "iss" claim value (optional)
	// If empty, issuer validation is skipped
	Issuer string `json:"issuer"`

	// Audience is the expected "aud" claim value (optional)
	// If empty, audience validation is skipped
	Audience string `json:"audience"`

	// CacheTTLSeconds is Redis cache duration for validated tokens
	// Default: 300 (5 minutes)
	// Note: Cache TTL should be <= token expiration
	CacheTTLSeconds int `json:"cache_ttl_seconds"`

	// BypassPaths are paths that skip authentication entirely
	// Default: ["/health", "/ready", "/metrics"]
	BypassPaths []string `json:"bypass_paths"`
}

// DefaultJWTAuthConfig returns sensible defaults.
func DefaultJWTAuthConfig() JWTAuthConfig {
	return JWTAuthConfig{
		Critical:        false,
		Algorithm:       "HS256",
		CookieName:      "jwt_token",
		HeaderName:      "Authorization",
		CacheTTLSeconds: 300, // 5 minutes
		BypassPaths:     defaultBypassPaths,
	}
}

// NewJWTAuthPlugin creates a new JWT authentication plugin.
//
// This is the factory function registered with the plugin registry.
func NewJWTAuthPlugin(configJSON json.RawMessage) (plugin.Plugin, error) {
	// Start with defaults
	config := DefaultJWTAuthConfig()

	// Override with user config if provided
	if len(configJSON) > 0 {
		if err := json.Unmarshal(configJSON, &config); err != nil {
			return nil, fmt.Errorf("invalid jwt-auth config: %w", err)
		}
	}

	// Validate configuration
	if config.SecretKey == "" {
		return nil, fmt.Errorf("secret_key is required")
	}

	if config.Algorithm != "HS256" {
		return nil, fmt.Errorf("only HS256 algorithm is currently supported")
	}

	log.Info().
		Str("component", "plugin").
		Str("plugin", "jwt-auth").
		Str("algorithm", config.Algorithm).
		Str("cookie_name", config.CookieName).
		Str("header_name", config.HeaderName).
		Bool("issuer_validation", config.Issuer != "").
		Bool("audience_validation", config.Audience != "").
		Int("cache_ttl", config.CacheTTLSeconds).
		Msg("Initializing JWT authentication plugin")

	return &JWTAuthPlugin{
		config: config,
		// redis will be injected by registry
	}, nil
}

// SetDependencies injects Redis client.
// Called by Registry after plugin creation.
// Note: JWT auth doesn't need repo, only Redis for caching
func (p *JWTAuthPlugin) SetDependencies(repo *database.Repository, redis *redis.Client) {
	// JWT auth doesn't need repo (no database lookup)
	// We only use Redis for caching
	p.redis = redis

	log.Debug().
		Str("component", "plugin").
		Str("plugin", "jwt-auth").
		Bool("redis_injected", redis != nil).
		Msg("Dependencies injected")
}

// Name returns the plugin identifier.
func (p *JWTAuthPlugin) Name() string {
	return "jwt-auth"
}

// Execute validates JWT token and loads user information.
func (p *JWTAuthPlugin) Execute(ctx *plugin.Context) error {
	// Only run in BeforeRequest phase
	if ctx.Phase != plugin.PhaseBeforeRequest {
		return nil
	}

	// Check if path should bypass authentication
	if p.shouldBypass(ctx.Request.URL.Path) {
		log.Debug().
			Str("path", ctx.Request.URL.Path).
			Msg("Bypassing JWT authentication for health check endpoint")
		return nil
	}

	// Extract JWT token from request
	tokenString := p.extractToken(ctx.Request)

	if tokenString == "" {
		log.Info().
			Str("path", ctx.Request.URL.Path).
			Str("method", ctx.Request.Method).
			Msg("JWT token required but not provided")

		ctx.Abort(401, "JWT token required")
		return nil
	}

	log.Debug().
		Str("token_prefix", tokenString[:min(20, len(tokenString))]).
		Msg("JWT token extracted")

	// Try to get cached claims
	claims, err := p.getCachedClaims(ctx.Context(), tokenString)
	if err != nil {
		log.Debug().
			Err(err).
			Msg("Cache lookup failed, will validate token")
	}

	if claims == nil {
		// Cache miss - validate token
		log.Debug().Msg("Cache miss - validating JWT token")

		claims, err = p.validateToken(tokenString)
		if err != nil {
			log.Warn().
				Err(err).
				Msg("Invalid JWT token")

			ctx.Abort(401, "Invalid or expired JWT token")
			return nil
		}

		// Cache the validated claims
		if cacheErr := p.cacheClaims(ctx.Context(), tokenString, claims); cacheErr != nil {
			log.Warn().
				Err(cacheErr).
				Msg("Failed to cache JWT claims (non-critical)")
		}
	} else {
		log.Debug().
			Str("user_id", getClaimString(claims, "user_id")).
			Msg("JWT claims found in cache")
	}

	// Extract user information from claims
	userID := getClaimString(claims, "user_id")
	email := getClaimString(claims, "email")
	username := getClaimString(claims, "username")

	if userID == "" {
		log.Warn().Msg("JWT token missing user_id claim")
		ctx.Abort(401, "Invalid token: missing user_id")
		return nil
	}

	// Add user information to context
	ctx.Set("user_id", userID)
	if email != "" {
		ctx.Set("user_email", email)
	}
	if username != "" {
		ctx.Set("user_username", username)
	}

	// Forward user info to backend via headers
	ctx.Request.Header.Set("X-User-ID", userID)
	if email != "" {
		ctx.Request.Header.Set("X-User-Email", email)
	}
	if username != "" {
		ctx.Request.Header.Set("X-User-Username", username)
	}

	log.Info().
		Str("user_id", userID).
		Str("username", username).
		Str("path", ctx.Request.URL.Path).
		Msg("JWT token validated successfully")

	return nil
}

// shouldBypass checks if the path should bypass authentication.
func (p *JWTAuthPlugin) shouldBypass(path string) bool {
	for _, bypassPath := range p.config.BypassPaths {
		if path == bypassPath || strings.HasPrefix(path, bypassPath+"/") {
			return true
		}
	}
	return false
}

// extractToken retrieves JWT token from Authorization header or cookie.
func (p *JWTAuthPlugin) extractToken(r *http.Request) string {
	// Try Authorization header first (more common)
	authHeader := r.Header.Get(p.config.HeaderName)
	if authHeader != "" {
		// Expected format: "Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			return parts[1]
		}
	}

	// Try cookie as fallback
	cookie, err := r.Cookie(p.config.CookieName)
	if err == nil && cookie.Value != "" {
		return cookie.Value
	}

	return ""
}

// validateToken validates JWT token and returns claims.
func (p *JWTAuthPlugin) validateToken(tokenString string) (jwt.MapClaims, error) {
	// Parse and validate token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validate signing algorithm
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// Return secret key for validation
		return []byte(p.config.SecretKey), nil
	})

	if err != nil {
		return nil, fmt.Errorf("token parsing failed: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("token is invalid")
	}

	// Extract claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("failed to extract claims")
	}

	// Validate standard claims
	if err := p.validateClaims(claims); err != nil {
		return nil, fmt.Errorf("claims validation failed: %w", err)
	}

	return claims, nil
}

// validateClaims validates JWT standard claims (exp, nbf, iss, aud).
func (p *JWTAuthPlugin) validateClaims(claims jwt.MapClaims) error {
	now := time.Now().Unix()

	// Validate expiration (exp)
	if exp, ok := claims["exp"].(float64); ok {
		if int64(exp) < now {
			return fmt.Errorf("token has expired")
		}
	} else {
		// exp claim is required
		return fmt.Errorf("missing exp claim")
	}

	// Validate not before (nbf) - optional
	if nbf, ok := claims["nbf"].(float64); ok {
		if int64(nbf) > now {
			return fmt.Errorf("token not yet valid")
		}
	}

	// Validate issuer (iss) - if configured
	if p.config.Issuer != "" {
		if iss, ok := claims["iss"].(string); !ok || iss != p.config.Issuer {
			return fmt.Errorf("invalid issuer")
		}
	}

	// Validate audience (aud) - if configured
	if p.config.Audience != "" {
		if aud, ok := claims["aud"].(string); !ok || aud != p.config.Audience {
			return fmt.Errorf("invalid audience")
		}
	}

	return nil
}

// getCachedClaims retrieves cached JWT claims from Redis.
func (p *JWTAuthPlugin) getCachedClaims(ctx context.Context, token string) (jwt.MapClaims, error) {
	if p.redis == nil {
		return nil, fmt.Errorf("redis client not available")
	}

	// Use token hash as cache key for privacy
	cacheKey := fmt.Sprintf("auth:jwt:%s", hashAPIKeyFull(token)[:32])

	// Get from Redis
	data, err := p.redis.Get(ctx, cacheKey).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Cache miss
		}
		return nil, fmt.Errorf("redis get failed: %w", err)
	}

	// Deserialize claims
	var claims jwt.MapClaims
	if err := json.Unmarshal(data, &claims); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cached claims: %w", err)
	}

	// Re-validate expiration (token might have expired since caching)
	if err := p.validateClaims(claims); err != nil {
		// Token expired, remove from cache
		p.redis.Del(ctx, cacheKey)
		return nil, err
	}

	return claims, nil
}

// cacheClaims stores JWT claims in Redis cache.
func (p *JWTAuthPlugin) cacheClaims(ctx context.Context, token string, claims jwt.MapClaims) error {
	if p.redis == nil {
		return fmt.Errorf("redis client not available")
	}

	cacheKey := fmt.Sprintf("auth:jwt:%s", hashAPIKeyFull(token)[:32])

	// Serialize claims
	data, err := json.Marshal(claims)
	if err != nil {
		return fmt.Errorf("failed to marshal claims: %w", err)
	}

	// Calculate TTL - use minimum of config TTL and token expiration
	ttl := time.Duration(p.config.CacheTTLSeconds) * time.Second

	// If token has exp claim, don't cache beyond expiration
	if exp, ok := claims["exp"].(float64); ok {
		tokenTTL := time.Until(time.Unix(int64(exp), 0))
		if tokenTTL < ttl {
			ttl = tokenTTL
		}
	}

	// Store in Redis with TTL
	if err := p.redis.Set(ctx, cacheKey, data, ttl).Err(); err != nil {
		return fmt.Errorf("redis set failed: %w", err)
	}

	log.Debug().
		Str("user_id", getClaimString(claims, "user_id")).
		Dur("ttl", ttl).
		Msg("JWT claims cached successfully")

	return nil
}

// Helper function to get string claim safely
func getClaimString(claims jwt.MapClaims, key string) string {
	if val, ok := claims[key].(string); ok {
		return val
	}
	return ""
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
