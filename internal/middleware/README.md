# Middleware Package

This package provides HTTP middleware components for the Subcults API server.

## Available Middleware

### Request ID Middleware

The Request ID middleware (`RequestID`) injects a unique request identifier into every HTTP request, enabling log correlation and distributed tracing.

#### Features

- **Automatic ID Generation**: Generates UUIDv4 when `X-Request-ID` header is absent
- **ID Preservation**: Preserves valid existing request IDs from incoming requests
- **Security Validation**: Validates incoming IDs to prevent log injection attacks
- **Context Storage**: Stores request ID in context for downstream handlers
- **Response Header**: Always includes `X-Request-ID` in responses

#### Usage

```go
import (
    "net/http"
    "github.com/onnwee/subcults/internal/middleware"
)

// Wrap your handler
mux := http.NewServeMux()
handler := middleware.RequestID(mux)
```

#### Accessing Request ID

```go
// In your handlers
func myHandler(w http.ResponseWriter, r *http.Request) {
    requestID := middleware.GetRequestID(r.Context())
    // Use requestID for logging, tracing, etc.
}
```

#### Validation Rules

Incoming `X-Request-ID` headers must meet these criteria:
- Maximum length: 128 characters
- Allowed characters: alphanumeric (a-z, A-Z, 0-9), hyphens (-), underscores (_)
- Non-empty

Invalid request IDs are automatically replaced with generated UUIDs.

#### Security

The middleware validates all incoming request IDs to prevent:
- Log injection attacks (special characters, newlines)
- Excessively long IDs that could cause DoS
- Empty or malformed identifiers

See [docs/PRIVACY.md](../../docs/PRIVACY.md) for more security details.

### Logging Middleware

The Logging middleware (`Logging`) provides structured request logging with configurable log levels based on response status.

#### Features

- **Structured Logging**: Uses `slog` for structured JSON/text output
- **Automatic Fields**: Captures method, path, status, latency, size
- **Context Integration**: Includes request_id and user_did when present
- **Error Codes**: Logs error codes for 4xx/5xx responses
- **Configurable Output**: JSON for production, text for development

#### Usage

```go
import (
    "log/slog"
    "github.com/onnwee/subcults/internal/middleware"
)

// Create logger based on environment
logger := middleware.NewLogger("production") // or "development"

// Wrap your handler
handler := middleware.Logging(logger)(mux)
```

#### Middleware Ordering

**Important**: Request ID middleware must be applied before Logging middleware to ensure request IDs are available in logs.

```go
// Correct ordering
handler := middleware.RequestID(
    middleware.Logging(logger)(
        mux,
    ),
)
```

#### Log Fields

| Field | Type | Description |
|-------|------|-------------|
| `method` | string | HTTP method (GET, POST, etc.) |
| `path` | string | Request path |
| `status` | int | HTTP response status code |
| `latency_ms` | int64 | Request duration in milliseconds |
| `size` | int | Response body size in bytes |
| `request_id` | string | Request correlation ID (if present) |
| `user_did` | string | Authenticated user's DID (if present) |
| `error_code` | string | Application error code (for 4xx/5xx) |

#### Log Levels

- **5xx errors**: `ERROR` level
- **4xx errors**: `WARN` level  
- **2xx/3xx success**: `INFO` level

### Rate Limiting Middleware

The Rate Limiting middleware (`RateLimiter`) implements sliding window rate limiting per client.

#### Features

- **Sliding Window**: Accurate rate limiting with sliding window algorithm
- **Flexible Keys**: Rate limit by IP address or authenticated user
- **Configurable Limits**: Per-endpoint rate limit configuration
- **Standard Headers**: Returns `Retry-After` and `X-RateLimit-Reset` headers
- **In-Memory Store**: Built-in memory store with automatic cleanup

#### Usage

```go
import (
    "time"
    "github.com/onnwee/subcults/internal/middleware"
)

// Create rate limit configuration
config := middleware.RateLimitConfig{
    RequestsPerWindow: 100,
    WindowDuration:    time.Minute,
}

// Create store and middleware
store := middleware.NewInMemoryRateLimitStore()
rateLimiter := middleware.RateLimiter(store, config, middleware.IPKeyFunc())

// Apply to handler
handler := rateLimiter(mux)
```

#### Key Functions

- **`IPKeyFunc()`**: Returns a KeyFunc that rate limits by client IP (uses X-Forwarded-For, X-Real-IP, or RemoteAddr)
- **`UserKeyFunc()`**: Returns a KeyFunc that rate limits by authenticated user DID (falls back to IP)

#### Default Limits

```go
// Pre-configured rate limit functions
middleware.DefaultGlobalLimit()    // 100 req/min
middleware.DefaultAuthLimit()      // 10 req/min
middleware.DefaultSearchLimit()    // 30 req/min
```

## Context Helpers

### User DID

```go
// Set user DID in context (typically in auth middleware)
ctx = middleware.SetUserDID(ctx, "did:plc:...")

// Retrieve user DID
userDID := middleware.GetUserDID(ctx)
```

### Error Code

```go
// Set error code for logging
ctx = middleware.SetErrorCode(ctx, "auth_failed")

// Retrieve error code
errorCode := middleware.GetErrorCode(ctx)
```

### Request ID

```go
// Retrieve request ID (set by RequestID middleware)
requestID := middleware.GetRequestID(ctx)
```

## Testing

All middleware components have comprehensive test coverage. Run tests with:

```bash
go test -v -race -cover ./internal/middleware
```

## Example: Complete Middleware Stack

```go
package main

import (
    "log/slog"
    "net/http"
    "time"
    "github.com/onnwee/subcults/internal/middleware"
)

func main() {
    // Create logger
    logger := middleware.NewLogger("production")
    
    // Create rate limiter
    store := middleware.NewInMemoryRateLimitStore()
    rateLimiter := middleware.RateLimiter(
        store,
        middleware.DefaultGlobalLimit(),
        middleware.UserKeyFunc(),
    )
    
    // Create your routes
    mux := http.NewServeMux()
    mux.HandleFunc("/api/health", healthHandler)
    
    // Apply middleware in correct order
    handler := middleware.RequestID(          // First: Generate/validate request ID
        middleware.Logging(logger)(          // Second: Log requests with request ID
            rateLimiter(                     // Third: Rate limit requests
                mux,                         // Finally: Your routes
            ),
        ),
    )
    
    // Start server
    http.ListenAndServe(":8080", handler)
}
```

## Performance Considerations

- **Request ID**: Minimal overhead (UUID generation or validation)
- **Logging**: Negligible impact using structured logging
- **Rate Limiting**: O(1) operations with in-memory store; periodic cleanup runs in background

For production deployments with multiple instances, consider using a distributed rate limit store (Redis-based implementation planned).
