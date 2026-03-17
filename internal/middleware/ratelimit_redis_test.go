package middleware

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// TestRedisRateLimitStore_Allow tests the Redis rate limiter with a real Redis instance.
// This test requires a Redis instance running on localhost:6379.
// Skip this test if Redis is not available.
func TestRedisRateLimitStore_Allow(t *testing.T) {
	// Try to connect to Redis
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Skip test if Redis is not available
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available, skipping integration test")
	}
	defer client.Close()

	store := NewRedisRateLimitStore(client)
	config := RateLimitConfig{
		RequestsPerWindow: 5,
		WindowDuration:    time.Minute,
	}

	testKey := "test-redis-key-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	ctx = context.Background()

	// Test that requests are allowed up to the limit
	for i := 0; i < 5; i++ {
		allowed, remaining, _ := store.Allow(ctx, testKey, config)
		if !allowed {
			t.Errorf("request %d should be allowed", i+1)
		}
		expectedRemaining := 4 - i
		if remaining != expectedRemaining {
			t.Errorf("request %d: expected remaining=%d, got %d", i+1, expectedRemaining, remaining)
		}
	}

	// Test that the 6th request is blocked
	allowed, remaining, retryAfter := store.Allow(ctx, testKey, config)
	if allowed {
		t.Error("6th request should be blocked")
	}
	if remaining != 0 {
		t.Errorf("expected remaining=0 when blocked, got %d", remaining)
	}
	if retryAfter <= 0 || retryAfter > 60 {
		t.Errorf("expected retryAfter between 1 and 60, got %d", retryAfter)
	}

	// Cleanup
	client.Del(ctx, testKey)
}

// TestRedisRateLimitStore_DifferentKeys tests that different keys have independent limits.
func TestRedisRateLimitStore_DifferentKeys(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available, skipping integration test")
	}
	defer client.Close()

	store := NewRedisRateLimitStore(client)
	config := RateLimitConfig{
		RequestsPerWindow: 1,
		WindowDuration:    time.Minute,
	}

	key1 := "test-redis-key1-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	key2 := "test-redis-key2-" + strconv.FormatInt(time.Now().UnixNano()+1, 10)
	ctx = context.Background()

	// Each key should have its own limit
	allowed1, _, _ := store.Allow(ctx, key1, config)
	allowed2, _, _ := store.Allow(ctx, key2, config)

	if !allowed1 || !allowed2 {
		t.Error("both keys should be allowed their first request")
	}

	// Both should now be at their limit
	blocked1, _, _ := store.Allow(ctx, key1, config)
	blocked2, _, _ := store.Allow(ctx, key2, config)

	if blocked1 || blocked2 {
		t.Error("both keys should be blocked after reaching limit")
	}

	// Cleanup
	client.Del(ctx, key1, key2)
}

// TestRedisRateLimitStore_WindowExpiry tests that limits reset after the window expires.
func TestRedisRateLimitStore_WindowExpiry(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available, skipping integration test")
	}
	defer client.Close()

	store := NewRedisRateLimitStore(client)
	config := RateLimitConfig{
		RequestsPerWindow: 1,
		WindowDuration:    100 * time.Millisecond,
	}

	testKey := "test-redis-expiry-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	ctx = context.Background()

	// Use up the limit
	allowed, _, _ := store.Allow(ctx, testKey, config)
	if !allowed {
		t.Error("first request should be allowed")
	}

	// Should be blocked
	allowed, _, _ = store.Allow(ctx, testKey, config)
	if allowed {
		t.Error("second request should be blocked")
	}

	// Wait for window to expire
	time.Sleep(150 * time.Millisecond)

	// Should be allowed again
	allowed, _, _ = store.Allow(ctx, testKey, config)
	if !allowed {
		t.Error("request after window expiry should be allowed")
	}

	// Cleanup
	client.Del(ctx, testKey)
}

// TestRedisRateLimitStore_FailOpen tests that the store fails open on Redis errors.
func TestRedisRateLimitStore_FailOpen(t *testing.T) {
	// Create a client with invalid address to simulate connection failure
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:9999", // Invalid port
	})
	defer client.Close()

	store := NewRedisRateLimitStore(client)
	config := RateLimitConfig{
		RequestsPerWindow: 5,
		WindowDuration:    time.Minute,
	}

	ctx := context.Background()

	// Should fail open and allow the request despite Redis being unavailable
	allowed, remaining, _ := store.Allow(ctx, "test-key", config)
	if !allowed {
		t.Error("should fail open and allow request when Redis is unavailable")
	}
	if remaining != config.RequestsPerWindow {
		t.Errorf("should return full quota on error, got %d", remaining)
	}
}
