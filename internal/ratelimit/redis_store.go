// Package ratelimit provides rate limiting implementations using Redis.
//
// This package supports multiple rate limiting algorithms:
//   - Token Bucket: Allows controlled bursts, tokens refill over time
//   - Sliding Window: Most accurate, uses sorted sets for timestamp tracking
//
// All implementations use Redis for distributed state, allowing rate limits
// to work correctly across multiple gateway instances.
//
// Thread Safety:
// All operations use Lua scripts executed atomically on Redis, ensuring
// correct behavior under high concurrency.
package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

// RedisStore provides Redis connection and helper methods for rate limiting.
//
// This store is separate from the hot-reload Redis connection to:
//   - Isolate rate limiting failures from config updates
//   - Allow different connection pool settings
//   - Enable independent scaling
type RedisStore struct {
	client *redis.Client
	config RedisConfig
}

// RedisConfig holds configuration for Redis connection.
type RedisConfig struct {
	// URL is the Redis connection string
	// Format: redis://[:password@]host[:port][/db]
	// Example: redis://localhost:6379/1
	URL string

	// PoolSize is the maximum number of socket connections
	// Default: 10 * runtime.NumCPU()
	PoolSize int

	// MinIdleConns is the minimum number of idle connections
	// Default: 0 (connections created on demand)
	MinIdleConns int

	// MaxRetries is the maximum number of retries before giving up
	// Default: 3
	MaxRetries int

	// DialTimeout is the timeout for establishing new connections
	// Default: 5 seconds
	DialTimeout time.Duration

	// ReadTimeout is the timeout for socket reads
	// Default: 3 seconds
	ReadTimeout time.Duration

	// WriteTimeout is the timeout for socket writes
	// Default: 3 seconds
	WriteTimeout time.Duration
}

// DefaultRedisConfig returns sensible defaults for rate limiting.
func DefaultRedisConfig() RedisConfig {
	return RedisConfig{
		URL:          "redis://localhost:6379/0",
		PoolSize:     50, // Higher pool for rate limiting
		MinIdleConns: 10, // Keep connections warm
		MaxRetries:   3,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	}
}

// NewRedisStore creates a new Redis store for rate limiting.
//
// The store maintains its own connection pool separate from other Redis usage.
// Call Close() when done to release resources.
func NewRedisStore(config RedisConfig) (*RedisStore, error) {
	log.Info().
		Str("component", "ratelimit_store").
		Str("url", maskRedisURL(config.URL)).
		Int("pool_size", config.PoolSize).
		Msg("Initializing Redis store for rate limiting")

	// Parse Redis URL
	opt, err := redis.ParseURL(config.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid redis URL: %w", err)
	}

	// Apply custom settings
	opt.PoolSize = config.PoolSize
	opt.MinIdleConns = config.MinIdleConns
	opt.MaxRetries = config.MaxRetries
	opt.DialTimeout = config.DialTimeout
	opt.ReadTimeout = config.ReadTimeout
	opt.WriteTimeout = config.WriteTimeout

	// Create client
	client := redis.NewClient(opt)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	log.Info().
		Str("component", "ratelimit_store").
		Str("addr", opt.Addr).
		Int("db", opt.DB).
		Msg("Redis store initialized successfully")

	return &RedisStore{
		client: client,
		config: config,
	}, nil
}

// Close closes the Redis connection and releases resources.
func (s *RedisStore) Close() error {
	log.Info().
		Str("component", "ratelimit_store").
		Msg("Closing Redis store connection")

	return s.client.Close()
}

// Ping checks if the Redis connection is alive.
func (s *RedisStore) Ping(ctx context.Context) error {
	return s.client.Ping(ctx).Err()
}

// EvalLua executes a Lua script on Redis.
//
// This is used for atomic operations like token bucket refill + consume.
// Lua scripts execute atomically on Redis, preventing race conditions.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - script: Lua script code
//   - keys: Redis keys the script will access (KEYS[1], KEYS[2], ...)
//   - args: Arguments to the script (ARGV[1], ARGV[2], ...)
//
// Returns:
//   - Result of the script execution (type varies by script)
//   - Error if script execution fails
func (s *RedisStore) EvalLua(ctx context.Context, script string, keys []string, args ...interface{}) (interface{}, error) {
	result, err := s.client.Eval(ctx, script, keys, args...).Result()
	if err != nil {
		log.Error().
			Err(err).
			Str("component", "ratelimit_store").
			Int("num_keys", len(keys)).
			Int("num_args", len(args)).
			Msg("Lua script execution failed")
		return nil, fmt.Errorf("lua script failed: %w", err)
	}

	return result, nil
}

// Get retrieves a string value from Redis.
func (s *RedisStore) Get(ctx context.Context, key string) (string, error) {
	val, err := s.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil // Key doesn't exist
	}
	if err != nil {
		return "", fmt.Errorf("redis GET failed: %w", err)
	}
	return val, nil
}

// Set stores a string value in Redis with optional TTL.
//
// If ttl is 0, the key will not expire.
func (s *RedisStore) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	err := s.client.Set(ctx, key, value, ttl).Err()
	if err != nil {
		return fmt.Errorf("redis SET failed: %w", err)
	}
	return nil
}

// Del deletes one or more keys from Redis.
func (s *RedisStore) Del(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}

	err := s.client.Del(ctx, keys...).Err()
	if err != nil {
		return fmt.Errorf("redis DEL failed: %w", err)
	}
	return nil
}

// Exists checks if a key exists in Redis.
func (s *RedisStore) Exists(ctx context.Context, key string) (bool, error) {
	count, err := s.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("redis EXISTS failed: %w", err)
	}
	return count > 0, nil
}

// TTL returns the remaining time-to-live for a key.
//
// Returns:
//   - duration > 0: Key exists with TTL
//   - duration == -1: Key exists but has no TTL
//   - duration == -2: Key does not exist
func (s *RedisStore) TTL(ctx context.Context, key string) (time.Duration, error) {
	ttl, err := s.client.TTL(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("redis TTL failed: %w", err)
	}
	return ttl, nil
}

// HGetAll retrieves all fields and values from a Redis hash.
func (s *RedisStore) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	result, err := s.client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("redis HGETALL failed: %w", err)
	}
	return result, nil
}

// HSet sets field in a Redis hash to value.
func (s *RedisStore) HSet(ctx context.Context, key string, field string, value interface{}) error {
	err := s.client.HSet(ctx, key, field, value).Err()
	if err != nil {
		return fmt.Errorf("redis HSET failed: %w", err)
	}
	return nil
}

// ZAdd adds a member with score to a sorted set.
func (s *RedisStore) ZAdd(ctx context.Context, key string, score float64, member string) error {
	err := s.client.ZAdd(ctx, key, redis.Z{
		Score:  score,
		Member: member,
	}).Err()
	if err != nil {
		return fmt.Errorf("redis ZADD failed: %w", err)
	}
	return nil
}

// ZRemRangeByScore removes members from a sorted set by score range.
//
// This is used in sliding window to remove old timestamps.
func (s *RedisStore) ZRemRangeByScore(ctx context.Context, key string, min, max string) error {
	err := s.client.ZRemRangeByScore(ctx, key, min, max).Err()
	if err != nil {
		return fmt.Errorf("redis ZREMRANGEBYSCORE failed: %w", err)
	}
	return nil
}

// ZCount counts members in a sorted set within a score range.
func (s *RedisStore) ZCount(ctx context.Context, key string, min, max string) (int64, error) {
	count, err := s.client.ZCount(ctx, key, min, max).Result()
	if err != nil {
		return 0, fmt.Errorf("redis ZCOUNT failed: %w", err)
	}
	return count, nil
}

// Stats returns Redis connection pool statistics.
func (s *RedisStore) Stats() *redis.PoolStats {
	return s.client.PoolStats()
}

// maskRedisURL masks the password in a Redis URL for logging.
//
// Example: redis://:password@localhost:6379 -> redis://:***@localhost:6379
func maskRedisURL(url string) string {
	// Simple masking - just replace password section
	// For production, consider using a proper URL parser
	if len(url) > 0 {
		return "redis://***"
	}
	return url
}
