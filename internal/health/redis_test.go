package health

import (
"testing"

"github.com/redis/go-redis/v9"
)

// TestRedisChecker_Creation tests that the Redis health checker is created correctly.
func TestRedisChecker_Creation(t *testing.T) {
// Create a Redis client (doesn't connect immediately)
client := redis.NewClient(&redis.Options{
Addr: "localhost:6379",
})

checker := NewRedisChecker(client)
if checker == nil {
t.Fatal("expected checker to be non-nil")
}

if checker.client != client {
t.Error("expected checker client to match provided client")
}
}

// TestRedisChecker_NilClient tests handling of nil client.
func TestRedisChecker_NilClient(t *testing.T) {
checker := NewRedisChecker(nil)
if checker == nil {
t.Fatal("expected checker to be non-nil even with nil client")
}

if checker.client != nil {
t.Error("expected checker client to be nil")
}
}
