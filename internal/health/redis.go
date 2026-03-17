// Package health provides health check implementations for external dependencies.
package health

import (
"context"

"github.com/redis/go-redis/v9"
)

// RedisChecker implements health checking for Redis.
type RedisChecker struct {
client *redis.Client
}

// NewRedisChecker creates a new Redis health checker.
func NewRedisChecker(client *redis.Client) *RedisChecker {
return &RedisChecker{
client: client,
}
}

// HealthCheck performs a health check on Redis by sending a PING command.
func (r *RedisChecker) HealthCheck(ctx context.Context) error {
return r.client.Ping(ctx).Err()
}
