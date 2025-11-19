// Package ratelimit - Sliding Window rate limiting algorithm
//
// Sliding Window Algorithm:
//   - Maintains a sorted set of request timestamps
//   - Window slides continuously with time
//   - Removes timestamps older than window duration
//   - Counts requests within current window
//   - More accurate than token bucket (no bursts)
//
// Use Cases:
//   - Strict API quotas (e.g., 1000 req/hour exactly)
//   - When bursts are not acceptable
//   - Compliance requirements (rate limit auditing)
//   - Fair usage enforcement
//
// Example:
//   - Limit: 100 requests/minute
//   - At 12:00:00 → 100 requests allowed
//   - At 12:00:30 → Still 100 requests allowed in [11:59:30 - 12:00:30]
//   - True sliding window, not fixed buckets
//
// Trade-offs:
//   - More accurate than token bucket
//   - Slightly more Redis memory (stores timestamps)
//   - Slightly more Redis CPU (sorted set operations)
package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
)

// SlidingWindow implements rate limiting using the sliding window algorithm.
//
// Algorithm Details:
//   - Each request adds a timestamp to Redis sorted set
//   - Score = Unix timestamp (for sorting)
//   - Member = Unique request ID (for deduplication)
//   - Old timestamps removed automatically
//   - Count requests in current window
//   - Atomic check + add using Lua script
type SlidingWindow struct {
	store  *RedisStore
	config SlidingWindowConfig
}

// SlidingWindowConfig holds configuration for sliding window rate limiter.
type SlidingWindowConfig struct {
	// Limit is the maximum number of requests allowed in the window
	// Example: 100 means max 100 requests per window
	Limit int

	// Window is the time duration for the sliding window
	// Example: 1 minute means 100 requests per minute
	Window time.Duration

	// KeyPrefix is prepended to all Redis keys
	// Example: "rate_limit:sw:" -> "rate_limit:sw:user123"
	KeyPrefix string

	// TTL is how long to keep window data in Redis after last access
	// Recommended: 2x window duration
	TTL time.Duration
}

// SlidingWindowResult holds the result of a rate limit check.
type SlidingWindowResult struct {
	// Allowed indicates if the request should be allowed
	Allowed bool

	// Remaining is how many requests are left in the window
	Remaining int

	// ResetTime is when the oldest request will expire
	ResetTime time.Time

	// RetryAfter is how long to wait before retrying (if not allowed)
	RetryAfter time.Duration

	// CurrentCount is the current number of requests in the window
	CurrentCount int
}

// NewSlidingWindow creates a new sliding window rate limiter.
//
// Example:
//
//	config := SlidingWindowConfig{
//	    Limit: 100,                    // 100 requests
//	    Window: time.Minute,           // per minute
//	    KeyPrefix: "rate_limit:sw:",
//	    TTL: 2 * time.Minute,
//	}
//	limiter := NewSlidingWindow(store, config)
func NewSlidingWindow(store *RedisStore, config SlidingWindowConfig) *SlidingWindow {
	log.Info().
		Str("component", "sliding_window").
		Int("limit", config.Limit).
		Dur("window", config.Window).
		Str("key_prefix", config.KeyPrefix).
		Dur("ttl", config.TTL).
		Msg("Sliding window rate limiter initialized")

	return &SlidingWindow{
		store:  store,
		config: config,
	}
}

// Allow checks if a request should be allowed and records it if so.
//
// This method is thread-safe and works correctly across multiple gateway instances
// because it uses a Lua script executed atomically on Redis.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - identifier: Unique identifier for the rate limit (consumer ID, IP, etc.)
//
// Returns:
//   - SlidingWindowResult with allow/deny decision and metadata
//   - Error if Redis operation fails
func (sw *SlidingWindow) Allow(ctx context.Context, identifier string) (*SlidingWindowResult, error) {
	key := sw.config.KeyPrefix + identifier
	now := time.Now()
	windowStart := now.Add(-sw.config.Window)

	// Generate unique request ID (timestamp + random component)
	requestID := fmt.Sprintf("%d", now.UnixNano())

	log.Debug().
		Str("component", "sliding_window").
		Str("identifier", identifier).
		Str("key", key).
		Time("window_start", windowStart).
		Msg("Checking rate limit")

	// Execute Lua script for atomic cleanup + count + add
	result, err := sw.store.EvalLua(
		ctx,
		slidingWindowLuaScript,
		[]string{key},
		windowStart.Unix(),              // ARGV[1] - window start timestamp
		now.Unix(),                      // ARGV[2] - current timestamp
		sw.config.Limit,                 // ARGV[3] - request limit
		requestID,                       // ARGV[4] - unique request ID
		int(sw.config.TTL.Seconds()),    // ARGV[5] - TTL
		int(sw.config.Window.Seconds()), // ARGV[6] - window duration
	)
	if err != nil {
		log.Error().
			Err(err).
			Str("component", "sliding_window").
			Str("identifier", identifier).
			Msg("Sliding window check failed")
		return nil, fmt.Errorf("sliding window check failed: %w", err)
	}

	// Parse Lua script result: {allowed, current_count, oldest_timestamp}
	resultArray, ok := result.([]interface{})
	if !ok || len(resultArray) != 3 {
		return nil, fmt.Errorf("unexpected lua script result format")
	}

	allowed := resultArray[0].(int64) == 1
	currentCount := int(resultArray[1].(int64))
	oldestTimestamp := resultArray[2].(int64)

	// Calculate remaining requests
	remaining := sw.config.Limit - currentCount
	if remaining < 0 {
		remaining = 0
	}

	// Calculate reset time (when oldest request expires)
	var resetTime time.Time
	if oldestTimestamp > 0 {
		resetTime = time.Unix(oldestTimestamp, 0).Add(sw.config.Window)
	} else {
		resetTime = now.Add(sw.config.Window)
	}

	// Calculate retry after duration
	var retryAfter time.Duration
	if !allowed && oldestTimestamp > 0 {
		// Time until oldest request expires
		retryAfter = time.Until(resetTime)
		if retryAfter < 0 {
			retryAfter = 0
		}
	}

	result2 := &SlidingWindowResult{
		Allowed:      allowed,
		Remaining:    remaining,
		ResetTime:    resetTime,
		RetryAfter:   retryAfter,
		CurrentCount: currentCount,
	}

	log.Debug().
		Str("component", "sliding_window").
		Str("identifier", identifier).
		Bool("allowed", allowed).
		Int("current_count", currentCount).
		Int("remaining", remaining).
		Time("reset_time", resetTime).
		Msg("Rate limit check completed")

	return result2, nil
}

// Reset clears the rate limit state for an identifier.
//
// This removes all request timestamps from the sliding window.
//
// Use cases:
//   - Admin override to unblock a user
//   - Testing
//   - Manual intervention
func (sw *SlidingWindow) Reset(ctx context.Context, identifier string) error {
	key := sw.config.KeyPrefix + identifier

	log.Info().
		Str("component", "sliding_window").
		Str("identifier", identifier).
		Str("key", key).
		Msg("Resetting rate limit")

	err := sw.store.Del(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to reset rate limit: %w", err)
	}

	return nil
}

// GetCount returns the current number of requests in the window.
//
// This is useful for:
//   - Monitoring
//   - Debugging
//   - Admin dashboards
//   - Metrics collection
func (sw *SlidingWindow) GetCount(ctx context.Context, identifier string) (int, error) {
	key := sw.config.KeyPrefix + identifier
	windowStart := time.Now().Add(-sw.config.Window)

	// Count requests in current window
	count, err := sw.store.ZCount(ctx, key, fmt.Sprintf("%d", windowStart.Unix()), "+inf")
	if err != nil {
		return 0, fmt.Errorf("failed to get count: %w", err)
	}

	return int(count), nil
}

// GetOldestTimestamp returns the timestamp of the oldest request in the window.
//
// Returns 0 if window is empty.
func (sw *SlidingWindow) GetOldestTimestamp(ctx context.Context, identifier string) (time.Time, error) {
	key := sw.config.KeyPrefix + identifier

	// Get oldest entry (lowest score)
	result, err := sw.store.client.ZRangeWithScores(ctx, key, 0, 0).Result()
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to get oldest timestamp: %w", err)
	}

	if len(result) == 0 {
		return time.Time{}, nil // Empty window
	}

	timestamp := int64(result[0].Score)
	return time.Unix(timestamp, 0), nil
}

// slidingWindowLuaScript implements atomic sliding window check + record.
//
// Algorithm:
//  1. Remove all timestamps older than window start (cleanup)
//  2. Count remaining requests in window
//  3. If count < limit, add new request timestamp and allow
//  4. If count >= limit, deny request
//  5. Get oldest timestamp for reset time calculation
//  6. Set TTL on key
//  7. Return: {allowed (0/1), current_count, oldest_timestamp}
//
// Keys:
//   - KEYS[1]: Redis sorted set key for this identifier
//
// Args:
//   - ARGV[1]: Window start timestamp (Unix seconds)
//   - ARGV[2]: Current timestamp (Unix seconds)
//   - ARGV[3]: Request limit
//   - ARGV[4]: Unique request ID
//   - ARGV[5]: TTL (seconds)
//   - ARGV[6]: Window duration (seconds)
//
// Returns:
//   - {1, current_count, oldest_timestamp} if allowed
//   - {0, current_count, oldest_timestamp} if denied
const slidingWindowLuaScript = `
-- Parse arguments
local window_start = tonumber(ARGV[1])
local current_time = tonumber(ARGV[2])
local limit = tonumber(ARGV[3])
local request_id = ARGV[4]
local ttl = tonumber(ARGV[5])
local window_duration = tonumber(ARGV[6])

-- Remove old timestamps (cleanup)
-- ZREMRANGEBYSCORE removes entries with score < window_start
redis.call('ZREMRANGEBYSCORE', KEYS[1], '-inf', window_start)

-- Count current requests in window
local current_count = redis.call('ZCARD', KEYS[1])

-- Check if request should be allowed
local allowed = 0
if current_count < limit then
    -- Add new request timestamp
    redis.call('ZADD', KEYS[1], current_time, request_id)
    current_count = current_count + 1
    allowed = 1
end

-- Get oldest timestamp in window (for reset time calculation)
local oldest_timestamp = 0
local oldest_entries = redis.call('ZRANGE', KEYS[1], 0, 0, 'WITHSCORES')
if #oldest_entries > 0 then
    oldest_timestamp = tonumber(oldest_entries[2])
end

-- Set TTL on key to prevent memory leaks
-- TTL should be longer than window to keep data for reset calculation
redis.call('EXPIRE', KEYS[1], ttl)

-- Return result: {allowed, current_count, oldest_timestamp}
return {allowed, current_count, oldest_timestamp}
`

// SlidingWindowStats holds statistics about the sliding window.
type SlidingWindowStats struct {
	// Identifier is the rate limit key
	Identifier string

	// CurrentCount is requests in current window
	CurrentCount int

	// Limit is the maximum allowed
	Limit int

	// Remaining is how many more are allowed
	Remaining int

	// OldestTimestamp is when the oldest request was made
	OldestTimestamp time.Time

	// ResetTime is when the window will reset
	ResetTime time.Time
}

// GetStats returns statistics for a rate limit identifier.
//
// This provides a comprehensive view of the current state.
func (sw *SlidingWindow) GetStats(ctx context.Context, identifier string) (*SlidingWindowStats, error) {
	count, err := sw.GetCount(ctx, identifier)
	if err != nil {
		return nil, err
	}

	oldest, err := sw.GetOldestTimestamp(ctx, identifier)
	if err != nil {
		return nil, err
	}

	remaining := sw.config.Limit - count
	if remaining < 0 {
		remaining = 0
	}

	var resetTime time.Time
	if !oldest.IsZero() {
		resetTime = oldest.Add(sw.config.Window)
	} else {
		resetTime = time.Now().Add(sw.config.Window)
	}

	return &SlidingWindowStats{
		Identifier:      identifier,
		CurrentCount:    count,
		Limit:           sw.config.Limit,
		Remaining:       remaining,
		OldestTimestamp: oldest,
		ResetTime:       resetTime,
	}, nil
}
