# Structured Logging Standard

This document describes the structured logging standard for Subcults.

## Overview

Subcults uses Go's standard `log/slog` package for all structured logging. Logs are output in JSON format in production and human-readable text format in development.

## Standard Fields

All log entries automatically include:

- **timestamp** (`time`): ISO 8601 timestamp of when the log was written
- **level**: One of `DEBUG`, `INFO`, `WARN`, `ERROR`
- **message** (`msg`): Human-readable description of the event

## Context Fields

Additional context fields are included when available:

- **request_id**: Unique identifier for the HTTP request (from `X-Request-ID` header or auto-generated)
- **user_did**: Decentralized identifier of the authenticated user (if authenticated)
- **error**: Error message (for error-level logs)

## HTTP Request Logs

HTTP request logs (from `middleware.Logging`) include:

- `method`: HTTP method (GET, POST, etc.)
- `path`: Request path
- `status`: HTTP status code
- `latency_ms`: Request duration in milliseconds
- `size`: Response size in bytes
- `request_id`: Request identifier
- `user_did`: User DID (if authenticated)
- `error_code`: Application error code (for 4xx and 5xx responses)

### Log Levels by Status Code

- **2xx**: `INFO` level
- **4xx**: `WARN` level
- **5xx**: `ERROR` level

## Configuration

Logger initialization is centralized in `internal/middleware/logging.go`:

```go
logger := middleware.NewLogger(env)
slog.SetDefault(logger)
```

### Environment Modes

- **production**: JSON handler with `INFO` level
- **development**: Text handler with `DEBUG` level

Set via `SUBCULT_ENV` environment variable (defaults to `development`).

## Usage Examples

### Basic Logging

```go
import "log/slog"

// Info level
slog.Info("server started", "port", 8080)

// With context
slog.InfoContext(ctx, "processing request", "user_id", userID)

// Error with structured fields
slog.Error("database connection failed", 
    "error", err,
    "retry_count", retryCount,
    "database", dbName)
```

### Stream Events

```go
slog.Info("stream started",
    "stream_id", streamID,
    "organizer_did", organizerDID,
    "participant_count", 42,
)
```

### Error Handling

```go
// Log errors with context
slog.ErrorContext(r.Context(), "failed to create scene",
    "error", err,
    "scene_id", sceneID,
)
```

### Debug Logging

```go
// Only logged in development mode
slog.Debug("cache hit",
    "key", cacheKey,
    "ttl_seconds", ttl,
)
```

## Best Practices

### DO

✅ Use structured fields instead of string interpolation:
```go
// Good
slog.Info("user created", "user_id", userID, "email", email)

// Bad
slog.Info(fmt.Sprintf("user %s created with email %s", userID, email))
```

✅ Use `*Context` methods to include request context:
```go
slog.InfoContext(ctx, "operation completed", "duration_ms", elapsed)
```

✅ Include relevant fields for debugging:
```go
slog.Error("payment failed",
    "error", err,
    "payment_id", paymentID,
    "amount", amount,
    "currency", currency,
)
```

✅ Use consistent field names across the codebase:
- `error` for error values
- `*_id` for identifiers (e.g., `scene_id`, `event_id`, `stream_id`)
- `*_did` for decentralized identifiers
- `*_count` for counts
- `*_ms` for millisecond durations

### DON'T

❌ Don't log sensitive data (passwords, API keys, tokens):
```go
// Bad - leaks credentials
slog.Info("authenticating", "password", password, "api_key", apiKey)
```

❌ Don't use string concatenation for log messages:
```go
// Bad
slog.Info("user " + userID + " logged in")

// Good
slog.Info("user logged in", "user_id", userID)
```

❌ Don't log excessive debug information in production:
```go
// Debug logs should use slog.Debug() - they won't appear in production
slog.Debug("cache details", "key", key, "value", value)
```

## Testing

### JSON Format Verification

Tests verify that production logs are valid JSON:

```go
func TestProductionLogJSON(t *testing.T) {
    logger := middleware.NewLogger("production")
    // ... test that output is parseable JSON
}
```

### Standard Fields Verification

Tests verify that all required fields are present:

```go
func TestLogging_StandardFields(t *testing.T) {
    // Verifies: time, level, msg, method, path, status, latency_ms, etc.
}
```

See `internal/middleware/logging_test.go` for comprehensive test examples.

## Migration Guide

### From Custom Logger

If you're using a custom logger, migrate to `slog`:

```go
// Before
logger.Printf("user %s created", userID)

// After
slog.Info("user created", "user_id", userID)
```

### From fmt.Printf

Replace print statements with structured logs:

```go
// Before
fmt.Printf("Error: %v\n", err)

// After
slog.Error("operation failed", "error", err)
```

## Related

- Epic: [subculture-collective/subcults#307](https://github.com/subculture-collective/subcults/issues/307) - Observability, Monitoring & Operations (canonical)
- Issue: [subculture-collective/subcults#121](https://github.com/subculture-collective/subcults/issues/121) - Logging standardization
- Roadmap: [subculture-collective/subcults#416](https://github.com/subculture-collective/subcults/issues/416) - Canonical Roadmap
