package ratelimit

import (
	"context"
	"testing"
	"time"
)

// TestTokenBucket_Allow tests basic token consumption.
func TestTokenBucket_Allow(t *testing.T) {
	// Setup Redis store (skip if Redis not available)
	config := DefaultRedisConfig()
	config.URL = "redis://localhost:6379/15" // Use test DB
	store, err := NewRedisStore(config)
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	defer store.Close()

	// Create token bucket: 10 tokens, refill 2/second
	tb := NewTokenBucket(store, TokenBucketConfig{
		Capacity:   10,
		RefillRate: 2.0,
		KeyPrefix:  "test:tb:",
		TTL:        1 * time.Minute,
	})

	ctx := context.Background()
	identifier := "test-user-1"

	// Clean up before test
	tb.Reset(ctx, identifier)

	// Test 1: First 10 requests should succeed (burst)
	for i := 0; i < 10; i++ {
		result, err := tb.Allow(ctx, identifier)
		if err != nil {
			t.Fatalf("Allow failed: %v", err)
		}
		if !result.Allowed {
			t.Errorf("Request %d should be allowed (burst)", i+1)
		}
	}

	// Test 2: 11th request should fail (bucket empty)
	result, err := tb.Allow(ctx, identifier)
	if err != nil {
		t.Fatalf("Allow failed: %v", err)
	}
	if result.Allowed {
		t.Error("Request 11 should be denied (bucket empty)")
	}
	if result.Remaining != 0 {
		t.Errorf("Expected 0 remaining, got %d", result.Remaining)
	}

	// Test 3: Wait for refill (0.5 seconds = 1 token)
	time.Sleep(500 * time.Millisecond)
	result, err = tb.Allow(ctx, identifier)
	if err != nil {
		t.Fatalf("Allow failed: %v", err)
	}
	if !result.Allowed {
		t.Error("Request should be allowed after refill")
	}

	// Clean up
	tb.Reset(ctx, identifier)
}

// TestTokenBucket_Concurrent tests concurrent access.
func TestTokenBucket_Concurrent(t *testing.T) {
	config := DefaultRedisConfig()
	config.URL = "redis://localhost:6379/15"
	store, err := NewRedisStore(config)
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	defer store.Close()

	tb := NewTokenBucket(store, TokenBucketConfig{
		Capacity:   100,
		RefillRate: 10.0,
		KeyPrefix:  "test:tb:",
		TTL:        1 * time.Minute,
	})

	ctx := context.Background()
	identifier := "test-user-2"
	tb.Reset(ctx, identifier)

	// Make 100 concurrent requests
	results := make(chan bool, 100)
	for i := 0; i < 100; i++ {
		go func() {
			result, err := tb.Allow(ctx, identifier)
			if err != nil {
				results <- false
				return
			}
			results <- result.Allowed
		}()
	}

	// Count allowed requests
	allowed := 0
	for i := 0; i < 100; i++ {
		if <-results {
			allowed++
		}
	}

	// Exactly 100 should be allowed (bucket capacity)
	if allowed != 100 {
		t.Errorf("Expected exactly 100 allowed, got %d", allowed)
	}

	// Clean up
	tb.Reset(ctx, identifier)
}

// TestCalculateRefillRate tests the helper function.
func TestCalculateRefillRate(t *testing.T) {
	tests := []struct {
		limit    int
		window   time.Duration
		expected float64
	}{
		{100, time.Minute, 1.6667},
		{1000, time.Minute, 16.6667},
		{10, time.Second, 10.0},
	}

	for _, tt := range tests {
		result := CalculateRefillRate(tt.limit, tt.window)
		if result < tt.expected-0.001 || result > tt.expected+0.001 {
			t.Errorf("CalculateRefillRate(%d, %v) = %f, want %f",
				tt.limit, tt.window, result, tt.expected)
		}
	}
}
