package ratelimit

import (
	"context"
	"testing"
	"time"
)

// TestSlidingWindow_Allow tests basic request counting.
func TestSlidingWindow_Allow(t *testing.T) {
	// Setup Redis store
	config := DefaultRedisConfig()
	config.URL = "redis://localhost:6379/15" // Use test DB
	store, err := NewRedisStore(config)
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	defer store.Close()

	// Create sliding window: 10 requests per 5 seconds
	sw := NewSlidingWindow(store, SlidingWindowConfig{
		Limit:     10,
		Window:    5 * time.Second,
		KeyPrefix: "test:sw:",
		TTL:       10 * time.Second,
	})

	ctx := context.Background()
	identifier := "test-user-1"

	// Clean up before test
	sw.Reset(ctx, identifier)

	// Test 1: First 10 requests should succeed
	for i := 0; i < 10; i++ {
		result, err := sw.Allow(ctx, identifier)
		if err != nil {
			t.Fatalf("Allow failed: %v", err)
		}
		if !result.Allowed {
			t.Errorf("Request %d should be allowed", i+1)
		}
		if result.CurrentCount != i+1 {
			t.Errorf("Expected count %d, got %d", i+1, result.CurrentCount)
		}
		if result.Remaining != 10-i-1 {
			t.Errorf("Expected remaining %d, got %d", 10-i-1, result.Remaining)
		}
	}

	// Test 2: 11th request should fail (limit reached)
	result, err := sw.Allow(ctx, identifier)
	if err != nil {
		t.Fatalf("Allow failed: %v", err)
	}
	if result.Allowed {
		t.Error("Request 11 should be denied (limit reached)")
	}
	if result.CurrentCount != 10 {
		t.Errorf("Expected count 10, got %d", result.CurrentCount)
	}
	if result.Remaining != 0 {
		t.Errorf("Expected remaining 0, got %d", result.Remaining)
	}

	// Test 3: Wait for window to slide (5 seconds)
	time.Sleep(5100 * time.Millisecond)

	// All requests should have expired, new request allowed
	result, err = sw.Allow(ctx, identifier)
	if err != nil {
		t.Fatalf("Allow failed: %v", err)
	}
	if !result.Allowed {
		t.Error("Request should be allowed after window expires")
	}
	if result.CurrentCount != 1 {
		t.Errorf("Expected count 1 after window reset, got %d", result.CurrentCount)
	}

	// Clean up
	sw.Reset(ctx, identifier)
}

// TestSlidingWindow_SlidingBehavior tests that window truly slides.
func TestSlidingWindow_SlidingBehavior(t *testing.T) {
	config := DefaultRedisConfig()
	config.URL = "redis://localhost:6379/15"
	store, err := NewRedisStore(config)
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	defer store.Close()

	// Create sliding window: 5 requests per 2 seconds
	sw := NewSlidingWindow(store, SlidingWindowConfig{
		Limit:     5,
		Window:    2 * time.Second,
		KeyPrefix: "test:sw:",
		TTL:       5 * time.Second,
	})

	ctx := context.Background()
	identifier := "test-user-2"
	sw.Reset(ctx, identifier)

	// T=0: Make 5 requests (fills limit)
	for i := 0; i < 5; i++ {
		result, err := sw.Allow(ctx, identifier)
		if err != nil {
			t.Fatalf("Allow failed: %v", err)
		}
		if !result.Allowed {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}

	// T=0: 6th request should fail
	result, err := sw.Allow(ctx, identifier)
	if err != nil {
		t.Fatalf("Allow failed: %v", err)
	}
	if result.Allowed {
		t.Error("6th request should be denied immediately")
	}

	// T=1: Wait 1 second (window is [T-2, T-1], still full)
	time.Sleep(1 * time.Second)
	result, err = sw.Allow(ctx, identifier)
	if err != nil {
		t.Fatalf("Allow failed: %v", err)
	}
	if result.Allowed {
		t.Error("Request should still be denied at T=1 (window still full)")
	}

	// T=2.1: Wait until window slides past first requests
	time.Sleep(1100 * time.Millisecond)

	// Now all 5 initial requests should have expired
	result, err = sw.Allow(ctx, identifier)
	if err != nil {
		t.Fatalf("Allow failed: %v", err)
	}
	if !result.Allowed {
		t.Error("Request should be allowed after window slides")
	}

	// Clean up
	sw.Reset(ctx, identifier)
}

// TestSlidingWindow_Concurrent tests concurrent access.
func TestSlidingWindow_Concurrent(t *testing.T) {
	config := DefaultRedisConfig()
	config.URL = "redis://localhost:6379/15"
	store, err := NewRedisStore(config)
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	defer store.Close()

	sw := NewSlidingWindow(store, SlidingWindowConfig{
		Limit:     100,
		Window:    1 * time.Minute,
		KeyPrefix: "test:sw:",
		TTL:       2 * time.Minute,
	})

	ctx := context.Background()
	identifier := "test-user-3"
	sw.Reset(ctx, identifier)

	// Make 150 concurrent requests
	results := make(chan bool, 150)
	for i := 0; i < 150; i++ {
		go func() {
			result, err := sw.Allow(ctx, identifier)
			if err != nil {
				results <- false
				return
			}
			results <- result.Allowed
		}()
	}

	// Count allowed requests
	allowed := 0
	for i := 0; i < 150; i++ {
		if <-results {
			allowed++
		}
	}

	// Exactly 100 should be allowed (limit)
	if allowed != 100 {
		t.Errorf("Expected exactly 100 allowed, got %d", allowed)
	}

	// Verify count
	count, err := sw.GetCount(ctx, identifier)
	if err != nil {
		t.Fatalf("GetCount failed: %v", err)
	}
	if count != 100 {
		t.Errorf("Expected count 100, got %d", count)
	}

	// Clean up
	sw.Reset(ctx, identifier)
}

// TestSlidingWindow_GetStats tests statistics retrieval.
func TestSlidingWindow_GetStats(t *testing.T) {
	config := DefaultRedisConfig()
	config.URL = "redis://localhost:6379/15"
	store, err := NewRedisStore(config)
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	defer store.Close()

	sw := NewSlidingWindow(store, SlidingWindowConfig{
		Limit:     10,
		Window:    1 * time.Minute,
		KeyPrefix: "test:sw:",
		TTL:       2 * time.Minute,
	})

	ctx := context.Background()
	identifier := "test-user-4"
	sw.Reset(ctx, identifier)

	// Make 3 requests
	for i := 0; i < 3; i++ {
		_, err := sw.Allow(ctx, identifier)
		if err != nil {
			t.Fatalf("Allow failed: %v", err)
		}
	}

	// Get stats
	stats, err := sw.GetStats(ctx, identifier)
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}

	// Verify stats
	if stats.CurrentCount != 3 {
		t.Errorf("Expected count 3, got %d", stats.CurrentCount)
	}
	if stats.Remaining != 7 {
		t.Errorf("Expected remaining 7, got %d", stats.Remaining)
	}
	if stats.Limit != 10 {
		t.Errorf("Expected limit 10, got %d", stats.Limit)
	}
	if stats.OldestTimestamp.IsZero() {
		t.Error("Expected oldest timestamp to be set")
	}

	// Clean up
	sw.Reset(ctx, identifier)
}

// TestSlidingWindow_Reset tests resetting rate limit.
func TestSlidingWindow_Reset(t *testing.T) {
	config := DefaultRedisConfig()
	config.URL = "redis://localhost:6379/15"
	store, err := NewRedisStore(config)
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	defer store.Close()

	sw := NewSlidingWindow(store, SlidingWindowConfig{
		Limit:     5,
		Window:    1 * time.Minute,
		KeyPrefix: "test:sw:",
		TTL:       2 * time.Minute,
	})

	ctx := context.Background()
	identifier := "test-user-5"

	// Fill up the limit
	for i := 0; i < 5; i++ {
		sw.Allow(ctx, identifier)
	}

	// Verify limit is reached
	result, _ := sw.Allow(ctx, identifier)
	if result.Allowed {
		t.Error("Request should be denied (limit reached)")
	}

	// Reset
	err = sw.Reset(ctx, identifier)
	if err != nil {
		t.Fatalf("Reset failed: %v", err)
	}

	// Verify requests allowed again
	result, err = sw.Allow(ctx, identifier)
	if err != nil {
		t.Fatalf("Allow failed: %v", err)
	}
	if !result.Allowed {
		t.Error("Request should be allowed after reset")
	}
	if result.CurrentCount != 1 {
		t.Errorf("Expected count 1 after reset, got %d", result.CurrentCount)
	}

	// Clean up
	sw.Reset(ctx, identifier)
}
