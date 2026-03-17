# Rate Limiting

The Subcults API implements comprehensive rate limiting to protect against abuse and ensure fair resource allocation.

## Features

- **Redis-backed distributed rate limiting** for multi-instance deployments
- **Per-endpoint limits** with different quotas for different operations
- **Per-user and per-IP limits** with automatic fallback
- **Burst allowance** – brief spikes above the base rate are permitted
- **Internal service bypass** – trusted services skip rate limiting via a shared secret
- **Pro-tier limits** – higher quotas for authenticated pro-tier users
- **Fail-open design** – if Redis is unavailable, requests are allowed to prevent outages
- **Prometheus metrics** for monitoring rate limit violations
- **Structured logging** for all rate limit events

---

## Configuration

### Redis Connection (Optional)

To enable distributed rate limiting with Redis, set the `REDIS_URL` environment variable:

```bash
REDIS_URL="redis://localhost:6379"
# or with password
REDIS_URL="redis://:password@localhost:6379/0"
```

If `REDIS_URL` is not set, the system uses an in-memory store (single-instance only).

### Internal Service Token (Optional)

Trusted internal services (e.g., other micro-services, background jobs) may bypass
per-endpoint and global rate limits by including the `X-Internal-Token` header with
the shared secret.

```bash
INTERNAL_SERVICE_TOKEN="<long-random-secret>"   # min 32 chars recommended
```

When the token is empty or unset, the bypass feature is disabled.

> ⚠️ **Security**: treat this token like a password.  Rotate it using the same
> dual-key rotation process described in [JWT_ROTATION_GUIDE.md](JWT_ROTATION_GUIDE.md).

---

## Rate Limits

### Per-Endpoint Limits

| Endpoint | Method | Limit | Window | Key Type |
|---|---|---|---|---|
| `POST /events` | write | 5 requests | 1 hour | per authenticated user |
| `POST /scenes` | write | 10 requests | 1 hour | per authenticated user |
| `POST /alliances` | write | 10 requests | 1 hour | per authenticated user |
| `GET /search/events` | read | 100 requests ¹ | 1 minute | per user / IP |
| `GET /search/scenes` | read | 100 requests ¹ | 1 minute | per user / IP |
| `GET /search/posts` | read | 100 requests ¹ | 1 minute | per user / IP |
| `POST /streams/{id}/join` | action | 10 requests | 1 minute | per authenticated user |
| `POST /telemetry/metrics` | write | 100 requests | 1 minute | per IP |
| All other endpoints | — | 1 000 requests ¹ | 1 minute | per IP |

¹ Burst allowance applies (see below).

### Pro-Tier Limits

Pro-tier users have higher limits on search and other read-heavy endpoints.
The tier is set in the request context via `middleware.SetUserTier(ctx, "pro")` by
authentication middleware that reads tier information from the JWT or user record.

Use `middleware.TieredRateLimiter` and `middleware.ProTierLimitSelector` to apply
different configs per tier:

```go
selector := middleware.ProTierLimitSelector(
    middleware.RateLimitConfig{RequestsPerWindow: 100, WindowDuration: time.Minute},  // free
    middleware.RateLimitConfig{RequestsPerWindow: 500, WindowDuration: time.Minute},  // pro
)
handler = middleware.TieredRateLimiter(store, selector, middleware.UserKeyFunc(), metrics)(handler)
```

---

## Rate Limit Headers

All responses include the following headers:

| Header | Presence | Description |
|---|---|---|
| `X-RateLimit-Limit` | All responses | Maximum requests allowed in the current window |
| `X-RateLimit-Remaining` | All responses | Requests remaining in the current window |
| `X-RateLimit-Reset` | 429 responses only | Unix timestamp when the limit resets |
| `Retry-After` | 429 responses only | Seconds to wait before retrying |

### Example: Normal Request

```http
HTTP/1.1 200 OK
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
Content-Type: application/json
```

### Example: Rate-Limited Request

```http
HTTP/1.1 429 Too Many Requests
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 0
X-RateLimit-Reset: 1706476800
Retry-After: 45
Content-Type: text/plain

Too Many Requests
```

---

## Burst Allowance

To absorb legitimate traffic spikes, endpoints with `BurstFactor > 1.0` allow
requests above the base rate for a short sub-window at the start of each main window.

| Endpoint group | Base limit | Burst factor | Burst limit | Burst window |
|---|---|---|---|---|
| Search endpoints | 100 req/min | 1.5× | 150 req/min | 10 s |
| General (all other) | 1 000 req/min | 1.5× | 1 500 req/min | 10 s |

**How it works (in-memory store)**:

1. At the start of each window, a *burst sub-window* begins (default 10 s).
2. During the burst sub-window, up to `base × BurstFactor` requests are allowed.
3. Once the burst sub-window expires, only the base rate applies for the remainder of
   the main window.
4. Burst requests count towards the main window total; they are not a separate pool.

> **Note**: When using the Redis store the burst limit is applied as a uniform cap on
> the sliding window (i.e., `base × BurstFactor` requests per window) because Redis
> does not track sub-windows.  Full burst sub-window semantics are only available with
> the in-memory store.

Configure burst in code:

```go
middleware.RateLimitConfig{
    RequestsPerWindow: 100,
    WindowDuration:    time.Minute,
    BurstFactor:       1.5,              // 1.5× burst (0 or omit to disable)
    BurstWindow:       10 * time.Second, // optional; defaults to 10 s
}
```

---

## Internal Service Bypass

Internal services send the `X-Internal-Token` header to skip rate limiting:

```http
GET /search/events HTTP/1.1
X-Internal-Token: <INTERNAL_SERVICE_TOKEN>
```

Create a bypass function:

```go
bypass := middleware.InternalServiceBypassFunc(cfg.InternalServiceToken)
handler = middleware.RateLimiterWithBypass(store, config, keyFunc, metrics, bypass)(handler)
```

Bypassed requests do **not** receive rate-limit headers.

---

## Monitoring

### Prometheus Metrics

Exposed on the `/metrics` endpoint:

| Metric | Labels | Description |
|---|---|---|
| `rate_limit_requests_total` | `endpoint`, `key_type` | Total rate limit checks |
| `rate_limit_blocked_total` | `endpoint`, `key_type` | Total blocked requests |

`key_type` is either `user` (authenticated) or `ip` (anonymous).

### Structured Log Fields

Rate limit violations are logged at `WARN` level with:

```json
{
  "level": "warn",
  "method": "GET",
  "path": "/search/events",
  "status": 429,
  "latency_ms": 2,
  "error_code": "rate_limit_exceeded",
  "rate_limit_key": "user:did:web:example.com:user123",
  "message": "request completed"
}
```

---

## Implementation Details

### Algorithm

**In-memory store**: fixed window counter with optional burst sub-window.

1. Each request increments a per-key counter.
2. When a new window starts, a burst sub-window is opened (if `BurstFactor > 1.0`).
3. During the burst sub-window the effective limit is `base × BurstFactor`.
4. After the burst sub-window only the base limit applies.

**Redis store**: sliding window counter (Lua script) with atomic operations.

### Fail-Open Design

If Redis becomes unavailable all requests are **allowed** to prevent cascading failures.
Full quota is returned in headers and errors are logged for monitoring.

### Key Types

- **User keys** (`user:{did}`): authenticated requests (DID from JWT)
- **IP keys** (`ip:{address}`): anonymous requests

IP addresses are extracted in order:
1. `X-Forwarded-For` (first IP in chain)
2. `X-Real-IP`
3. `RemoteAddr`

---

## Development

### Running Tests

```bash
# All rate limiting tests
go test ./internal/middleware -v -run "TestRateLimit|TestMetrics|TestBurst|TestBypass|TestTiered|TestPro"

# Redis integration tests (requires Redis on localhost:6379)
go test ./internal/middleware -v -run "TestRedis"
```

### In-Memory vs Redis

| Feature | In-Memory | Redis |
|---|---|---|
| Burst sub-window | ✅ exact | ⚠️ simplified (burst limit as cap) |
| Multi-instance | ❌ | ✅ |
| Restart-safe | ❌ | ✅ |
| Zero dependencies | ✅ | ❌ |
| Network latency | none | ~1 ms |

---

## Troubleshooting

### `rate_limit_exceeded` errors in logs

1. Check whether the limits are appropriate for the use case.
2. Identify if a specific user/IP is abusive.
3. Use `INTERNAL_SERVICE_TOKEN` for trusted callers instead of raising global limits.

### Redis connection errors

1. Verify Redis is running: `redis-cli ping`
2. Check `REDIS_URL` format.
3. Ensure firewall rules allow the connection.
4. Use in-memory mode for development (omit `REDIS_URL`).

### Headers missing on bypassed requests

Bypassed requests intentionally omit rate-limit headers.  If you need quota information
for bypassed callers, query the `/metrics` endpoint instead.
