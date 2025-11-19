// Package ratelimit - Token Bucket rate limiting algorithm
//
// Token Bucket Algorithm:
//   - Bucket holds tokens (capacity = max tokens)
//   - Tokens refill at a constant rate
//   - Each request consumes 1 token
//   - Request allowed if tokens available
//   - Allows controlled bursts
//
// Use Cases:
//   - APIs that allow occasional bursts
//   - Systems with bursty traffic patterns
//   - When gradual recovery is desired
//
// Example:
//   - Limit: 100 tokens/minute
//   - Refill: ~1.67 tokens/second
//   - User can burst 100 requests immediately
//   - Then limited to 1.67 req/s until bucket refills
package ratelimit

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/rs/zerolog/log"
)

// TokenBucket implements rate limiting using the token bucket algorithm.
//
// Algorithm Details:
//   - Each identifier (consumer, IP, etc.) has their own bucket
//   - Buckets stored in Redis as hash: {tokens, last_refill}
//   - Tokens refill continuously based on elapsed time
//   - Atomic refill + consume using Lua script
type TokenBucket struct {
	store  *RedisStore
	config TokenBucketConfig
}

// TokenBucketConfig holds configuration for token bucket rate limiter.
type TokenBucketConfig struct {
	// Capacity is the maximum number of tokens in the bucket
	// Example: 100 means burst of 100 requests allowed
	Capacity int

	// RefillRate is tokens added per second
	// Example: 10 means 10 requests/second sustained
	RefillRate float64

	// KeyPrefix is prepended to all Redis keys
	// Example: "rate_limit:tb:" -> "rate_limit:tb:user123"
	KeyPrefix string

	// TTL is how long to keep bucket state in Redis after last access
	// This prevents memory leaks for inactive users
	// Recommended: 2x window duration
	TTL time.Duration
}

// TokenBucketResult holds the result of a rate limit check.
type TokenBucketResult struct {
	// Allowed indicates if the request should be allowed
	Allowed bool

	// Remaining is how many tokens are left in the bucket
	Remaining int

	// ResetTime is when the bucket will be full again
	ResetTime time.Time

	// RetryAfter is how long to wait before retrying (if not allowed)
	RetryAfter time.Duration
}

// NewTokenBucket creates a new token bucket rate limiter.
//
// Example:
//
//	config := TokenBucketConfig{
//	    Capacity: 100,           // Allow burst of 100
//	    RefillRate: 10,          // 10 requests/second sustained
//	    KeyPrefix: "rate_limit:tb:",
//	    TTL: 2 * time.Minute,
//	}
//	limiter := NewTokenBucket(store, config)
func NewTokenBucket(store *RedisStore, config TokenBucketConfig) *TokenBucket {
	log.Info().
		Str("component", "token_bucket").
		Int("capacity", config.Capacity).
		Float64("refill_rate", config.RefillRate).
		Str("key_prefix", config.KeyPrefix).
		Dur("ttl", config.TTL).
		Msg("Token bucket rate limiter initialized")

	return &TokenBucket{
		store:  store,
		config: config,
	}
}

// Allow checks if a request should be allowed and consumes a token if so.
//
// This method is thread-safe and works correctly across multiple gateway instances
// because it uses a Lua script executed atomically on Redis.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - identifier: Unique identifier for the rate limit bucket (consumer ID, IP, etc.)
//
// Returns:
//   - TokenBucketResult with allow/deny decision and metadata
//   - Error if Redis operation fails
func (tb *TokenBucket) Allow(ctx context.Context, identifier string) (*TokenBucketResult, error) {
	key := tb.config.KeyPrefix + identifier

	log.Debug().
		Str("component", "token_bucket").
		Str("identifier", identifier).
		Str("key", key).
		Msg("Checking rate limit")

	// Execute Lua script for atomic refill + consume
	// NEW (FIXED)
	now := time.Now()
	nowMs := now.UnixMilli() // Use milliseconds for precision

	result, err := tb.store.EvalLua(
		ctx,
		tokenBucketLuaScript,
		[]string{key},
		tb.config.Capacity,           // ARGV[1]
		tb.config.RefillRate,         // ARGV[2]
		nowMs,                        // ARGV[3] ← FIX: Milliseconds
		int(tb.config.TTL.Seconds()), // ARGV[4]
	)
	if err != nil {
		log.Error().
			Err(err).
			Str("component", "token_bucket").
			Str("identifier", identifier).
			Msg("Token bucket check failed")
		return nil, fmt.Errorf("token bucket check failed: %w", err)
	}

	// Parse Lua script result: {allowed, tokens_remaining, reset_time}
	resultArray, ok := result.([]interface{})
	if !ok || len(resultArray) != 3 {
		return nil, fmt.Errorf("unexpected lua script result format")
	}

	allowed := resultArray[0].(int64) == 1
	remaining := int(resultArray[1].(int64))
	resetTime := time.Unix(resultArray[2].(int64), 0)

	// Calculate retry after duration
	var retryAfter time.Duration
	if !allowed {
		// Time until one token is refilled
		retryAfter = time.Duration(1.0 / tb.config.RefillRate * float64(time.Second))
	}

	result2 := &TokenBucketResult{
		Allowed:    allowed,
		Remaining:  remaining,
		ResetTime:  resetTime,
		RetryAfter: retryAfter,
	}

	log.Debug().
		Str("component", "token_bucket").
		Str("identifier", identifier).
		Bool("allowed", allowed).
		Int("remaining", remaining).
		Time("reset_time", resetTime).
		Msg("Rate limit check completed")

	return result2, nil
}

// Reset clears the rate limit state for an identifier.
//
// This can be used for:
//   - Admin override to unblock a user
//   - Testing
//   - Manual intervention
func (tb *TokenBucket) Reset(ctx context.Context, identifier string) error {
	key := tb.config.KeyPrefix + identifier

	log.Info().
		Str("component", "token_bucket").
		Str("identifier", identifier).
		Str("key", key).
		Msg("Resetting rate limit")

	err := tb.store.Del(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to reset rate limit: %w", err)
	}

	return nil
}

// GetState retrieves the current state of a rate limit bucket.
//
// This is useful for:
//   - Monitoring
//   - Debugging
//   - Admin dashboards
//
// Returns nil if bucket doesn't exist (no requests yet).
func (tb *TokenBucket) GetState(ctx context.Context, identifier string) (map[string]string, error) {
	key := tb.config.KeyPrefix + identifier

	state, err := tb.store.HGetAll(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get rate limit state: %w", err)
	}

	return state, nil
}

// tokenBucketLuaScript implements atomic token bucket refill + consume.
//
// Algorithm:
//  1. Get current tokens and last refill time from Redis
//  2. Calculate tokens to add based on elapsed time
//  3. Add tokens up to capacity
//  4. If tokens >= 1, consume one token and allow request
//  5. Update state in Redis
//  6. Return: {allowed (0/1), remaining_tokens, reset_time}
//
// Keys:
//   - KEYS[1]: Redis hash key for this bucket
//
// -- Args:
// --   - ARGV[1]: Capacity (max tokens)
// --   - ARGV[2]: Refill rate (tokens per second)
// --   - ARGV[3]: Current timestamp (Unix milliseconds)  ← FIXED
// --   - ARGV[4]: TTL (seconds)
// Returns:
//   - {1, remaining_tokens, reset_time} if allowed
//   - {0, remaining_tokens, reset_time} if denied
const tokenBucketLuaScript = `
-- Get current state
local tokens = tonumber(redis.call('HGET', KEYS[1], 'tokens'))
local last_refill = tonumber(redis.call('HGET', KEYS[1], 'last_refill'))

-- Parse arguments
local capacity = tonumber(ARGV[1])
local refill_rate = tonumber(ARGV[2])
local now = tonumber(ARGV[3])
local ttl = tonumber(ARGV[4])

-- Initialize if bucket doesn't exist
if tokens == nil then
    tokens = capacity
    last_refill = now
end

-- Calculate elapsed time since last refill (in seconds)
local elapsed_ms = math.max(0, now - last_refill)
local elapsed_sec = elapsed_ms / 1000.0  -- Convert ms to seconds

-- Calculate tokens to add
local tokens_to_add = elapsed_sec * refill_rate

-- Refill tokens up to capacity
tokens = math.min(capacity, tokens + tokens_to_add)

-- Update last refill time
last_refill = now

-- Try to consume one token
local allowed = 0
if tokens >= 1 then
    tokens = tokens - 1
    allowed = 1
end

-- Calculate reset time (when bucket will be full)
local tokens_needed = capacity - tokens
local seconds_to_full = 0
if tokens_needed > 0 then
    seconds_to_full = math.ceil(tokens_needed / refill_rate)
end
local reset_time_ms = now + (seconds_to_full * 1000)  -- Convert to ms
local reset_time = math.floor(reset_time_ms / 1000)   -- Convert back to Unix seconds for return

-- Save state to Redis
redis.call('HSET', KEYS[1], 'tokens', tostring(tokens))
redis.call('HSET', KEYS[1], 'last_refill', tostring(last_refill))
redis.call('EXPIRE', KEYS[1], ttl)

-- Return result: {allowed, remaining_tokens, reset_time}
return {allowed, math.floor(tokens), reset_time}
`

// CalculateRefillRate is a helper to calculate refill rate from limit and window.
//
// Example:
//
//	// 100 requests per minute
//	rate := CalculateRefillRate(100, time.Minute)
//	// rate = 1.6667 (tokens per second)
func CalculateRefillRate(limit int, window time.Duration) float64 {
	return float64(limit) / window.Seconds()
}

// CalculateResetTime calculates when the bucket will be full again.
func CalculateResetTime(tokensRemaining int, capacity int, refillRate float64) time.Time {
	tokensNeeded := capacity - tokensRemaining
	if tokensNeeded <= 0 {
		return time.Now()
	}

	secondsToFull := float64(tokensNeeded) / refillRate
	return time.Now().Add(time.Duration(secondsToFull) * time.Second)
}

// FormatDuration formats a duration for rate limit headers.
//
// Example: 1m30s -> "90"
func FormatDuration(d time.Duration) string {
	return fmt.Sprintf("%d", int(math.Ceil(d.Seconds())))
}
