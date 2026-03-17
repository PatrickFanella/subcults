# Backend Core - Go API Service Implementation

## Overview

This document describes the Backend Core implementation (Epic #3), which provides the foundational Go API service with `http.ServeMux`-based routing, middleware stack, configuration management, JWT authentication, and standardized error handling.

## Architecture

### Components

1. **API Server** (`cmd/api/main.go`)
   - Entry point with graceful shutdown
   - Server initialization and configuration
   - Middleware stack composition
   - Route registration

2. **Configuration** (`internal/config/`)
   - Environment variable loading with `koanf`
   - YAML file support with env override
   - Validation and error reporting
   - Secrets masking for logs

3. **Authentication** (`internal/auth/`)
   - JWT token generation (access & refresh)
   - Token validation with leeway
   - Standard claims (sub, iat, exp)
   - DID claim for access tokens

4. **Middleware Stack** (`internal/middleware/`)
   - Request ID generation/extraction
   - Structured logging with `slog`
   - Rate limiting (in-memory & Redis)
   - HTTP metrics (Prometheus)
   - OpenTelemetry tracing

5. **Error Handling** (`internal/api/errors.go`)
   - Standard JSON error format
   - Error code catalog
   - Context propagation for logging

6. **Health Checks** (`internal/api/health_handlers.go`)
   - Liveness probe (`/health`)
   - Readiness probe (`/ready`)
   - Dependency health checks

## Request Flow

Requests pass through middleware in this order:

1. **Tracing** - OpenTelemetry instrumentation (if enabled)
2. **RateLimiter** - Global rate limit (1000 req/min per IP)
3. **HTTPMetrics** - Prometheus metrics collection
4. **RequestID** - Generate/extract request IDs
5. **Logging** - Structured request/response logging

## Configuration

### Environment Variables

Configuration follows a precedence order: **Environment > YAML File > Defaults**

#### Core Settings
- `SUBCULT_ENV` / `ENV` / `GO_ENV` - Environment mode (development/production)
- `SUBCULT_PORT` / `PORT` - Server port (default: 8080)

#### Required Variables
- `DATABASE_URL` - Neon Postgres connection string
- `JWT_SECRET` - JWT signing secret (min 32 chars recommended)
- `LIVEKIT_URL` - LiveKit WebSocket URL
- `LIVEKIT_API_KEY` - LiveKit API key
- `LIVEKIT_API_SECRET` - LiveKit API secret
- `STRIPE_API_KEY` - Stripe secret API key
- `STRIPE_WEBHOOK_SECRET` - Stripe webhook signing secret
- `STRIPE_ONBOARDING_RETURN_URL` - Stripe onboarding return URL
- `STRIPE_ONBOARDING_REFRESH_URL` - Stripe onboarding refresh URL
- `MAPTILER_API_KEY` - MapTiler API key
- `JETSTREAM_URL` - Jetstream WebSocket URL

#### Optional Variables
- `REDIS_URL` - Redis connection for distributed rate limiting
- `R2_BUCKET_NAME` - Cloudflare R2 bucket for media storage
- `R2_ACCESS_KEY_ID` - R2 access key
- `R2_SECRET_ACCESS_KEY` - R2 secret key
- `R2_ENDPOINT` - R2 endpoint URL
- `R2_MAX_UPLOAD_SIZE_MB` - Max upload size (default: 15MB)

#### Feature Flags
- `RANK_TRUST_ENABLED` - Enable trust-weighted ranking (default: false)
- `TRACING_ENABLED` - Enable OpenTelemetry tracing (default: false)
- `TRACING_EXPORTER_TYPE` - Exporter type: otlp-http, otlp-grpc (default: otlp-http)
- `TRACING_OTLP_ENDPOINT` - OTLP endpoint URL
- `TRACING_SAMPLE_RATE` - Sampling rate 0.0-1.0 (default: 0.1)
- `TRACING_INSECURE` - Disable TLS for OTLP (dev only, default: false)

### Loading Configuration

```go
import "github.com/onnwee/subcults/internal/config"

// Load configuration from environment (no file)
cfg, errs := config.Load("")

// Load with optional YAML file (env variables take precedence)
cfg, errs := config.Load("config.yaml")

// Check for validation errors
if len(errs) > 0 {
    for _, err := range errs {
        log.Printf("Config error: %v", err)
    }
    os.Exit(1)
}

// Log configuration summary (secrets masked)
log.Printf("Config: %+v", cfg.LogSummary())
```

## Authentication

### JWT Tokens

Two token types with different expiration times:

1. **Access Token** (15 minutes)
   - Contains user ID and DID claim
   - Used for API authentication
   - Short-lived for security

2. **Refresh Token** (7 days)
   - Contains only user ID
   - Used to obtain new access tokens
   - No DID claim for separation of concerns

### Usage

```go
import "github.com/onnwee/subcults/internal/auth"

// Create JWT service
jwtService := auth.NewJWTService(jwtSecret)

// Generate tokens
accessToken, err := jwtService.GenerateAccessToken(userID, did)
refreshToken, err := jwtService.GenerateRefreshToken(userID)

// Validate token
claims, err := jwtService.ValidateToken(tokenString)
if err == auth.ErrExpiredToken {
    // Token expired, request refresh
}
if err == auth.ErrInvalidToken {
    // Invalid token
}

// Access claims
userID := claims.Subject
did := claims.DID
tokenType := claims.Type  // "access" or "refresh"
```

## Middleware

### Request ID

Generates a unique request ID (UUIDv4) or uses existing `X-Request-ID` header if valid.

```go
import "github.com/onnwee/subcults/internal/middleware"

// Apply middleware
handler = middleware.RequestID(handler)

// Access request ID in handlers
requestID := middleware.GetRequestID(ctx)
```

**Security**: Request IDs from headers are validated to prevent injection attacks. Invalid IDs are rejected and new UUIDs are generated.

### Logging

Structured logging with `log/slog` capturing request/response metadata.

**Logged Fields**:
- `method` - HTTP method
- `path` - Request path
- `status` - Response status code
- `latency_ms` - Request duration in milliseconds
- `size` - Response size in bytes
- `request_id` - Request ID (if present)
- `user_did` - Authenticated user DID (if present)
- `error_code` - Error code for 4xx/5xx responses

**Log Levels**:
- 5xx responses → ERROR
- 4xx responses → WARN
- 2xx/3xx responses → INFO

```go
logger := middleware.NewLogger(env)  // "production" = JSON, else text
handler = middleware.Logging(logger)(handler)
```

### Rate Limiting

Token bucket algorithm with configurable limits per endpoint.

**Backends**:
- **In-Memory** - Single instance deployments (automatic cleanup)
- **Redis** - Distributed deployments with sliding window

**Key Functions**:
- `IPKeyFunc()` - Rate limit by IP address
- `UserKeyFunc()` - Rate limit by authenticated user (fallback to IP)

**Headers Set**:
- `X-RateLimit-Limit` - Maximum requests per window
- `X-RateLimit-Remaining` - Requests remaining
- `Retry-After` - Seconds until limit resets (on 429)
- `X-RateLimit-Reset` - Unix timestamp of reset time

```go
// Define rate limit configuration
config := middleware.RateLimitConfig{
    RequestsPerWindow: 100,
    WindowDuration:    time.Minute,
}

// Create store (in-memory or Redis)
store := middleware.NewInMemoryRateLimitStore()
// OR
store := middleware.NewRedisRateLimitStore(redisClient)

// Apply middleware
handler = middleware.RateLimiter(store, config, middleware.IPKeyFunc(), metrics)(handler)
```

### HTTP Metrics

Prometheus metrics for HTTP requests.

**Metrics Collected**:
- `http_requests_total` - Total requests by method, path, status
- `http_request_duration_seconds` - Request duration histogram
- `http_request_size_bytes` - Request size histogram
- `http_response_size_bytes` - Response size histogram
- `rate_limit_requests_total` - Rate limit checks by path, key type
- `rate_limit_blocked_total` - Rate limit violations by path, key type
- `rate_limit_redis_errors_total` - Redis backend errors

```go
metrics := middleware.NewMetrics()
if err := metrics.Register(promRegistry); err != nil {
    log.Fatal(err)
}

handler = middleware.HTTPMetrics(metrics)(handler)
```

### Tracing

OpenTelemetry distributed tracing with configurable sampling.

**Exporters**:
- OTLP HTTP (default)
- OTLP gRPC

```go
import "github.com/onnwee/subcults/internal/tracing"

config := tracing.Config{
    ServiceName:  "subcults-api",
    Enabled:      true,
    Environment:  "production",
    ExporterType: "otlp-http",
    OTLPEndpoint: "http://collector:4318",
    SamplingRate: 0.1,  // 10% sampling
    InsecureMode: false,
}

provider, err := tracing.NewProvider(config)
defer provider.Shutdown(ctx)

// Apply middleware
handler = middleware.Tracing("subcults-api")(handler)
```

## Error Handling

### Standard Error Format

All API errors return JSON in this structure:

```json
{
  "error": {
    "code": "error_code",
    "message": "Human-readable error message"
  }
}
```

### Error Codes

Common error codes defined in `internal/api/errors.go`:

- `validation_error` - Input validation failure
- `auth_failed` - Authentication failure
- `not_found` - Resource not found
- `rate_limited` - Rate limit exceeded
- `forbidden` - Access forbidden
- `conflict` - Resource conflict
- `bad_request` - Malformed request
- `internal_error` - Internal server error

### Writing Errors

```go
import "github.com/onnwee/subcults/internal/api"
import "github.com/onnwee/subcults/internal/middleware"

func handler(w http.ResponseWriter, r *http.Request) {
    // Set error code for logging
    ctx := middleware.SetErrorCode(r.Context(), api.ErrCodeNotFound)
    
    // Write error response
    api.WriteError(w, ctx, http.StatusNotFound, api.ErrCodeNotFound, "Resource not found")
}
```

The error code is automatically logged by the logging middleware for observability.

## Health Endpoints

### Liveness Probe

**Endpoint**: `GET /health`

Returns 200 if the process is alive and can handle requests.

```json
{
  "status": "healthy",
  "checks": {
    "runtime": "ok"
  },
  "timestamp": "2026-01-30T22:00:00Z"
}
```

### Readiness Probe

**Endpoint**: `GET /ready`

Returns 200 if ready to serve traffic, 503 if dependencies are unavailable.

Checks:
- Database connectivity (if configured)
- LiveKit availability (if configured)
- Stripe availability (if configured)
- Metrics availability (always ok)

```json
{
  "status": "healthy",
  "checks": {
    "database": "ok",
    "livekit": "ok",
    "stripe": "ok",
    "metrics": "ok"
  },
  "timestamp": "2026-01-30T22:00:00Z"
}
```

## Graceful Shutdown

The API server implements graceful shutdown on SIGINT/SIGTERM:

1. Stop accepting new connections
2. Wait for in-flight requests (up to 10s timeout)
3. Flush tracing spans
4. Close database connections
5. Close Redis connections (if used)
6. Exit cleanly

```go
// Graceful shutdown triggered by OS signals
quit := make(chan os.Signal, 1)
signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
<-quit

logger.Info("shutting down server...")

// Create context with timeout
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

// Shutdown server
if err := server.Shutdown(ctx); err != nil {
    logger.Error("server forced to shutdown", "error", err)
}
```

## Testing

All core components have comprehensive test coverage:

### Config Tests
```bash
go test -v ./internal/config
```
- Environment variable loading
- YAML file loading
- Default values
- Validation errors
- Secrets masking

### Auth Tests
```bash
go test -v ./internal/auth
```
- Token generation (access & refresh)
- Token validation
- Expiration handling
- Claims extraction

### Middleware Tests
```bash
go test -v ./internal/middleware
```
- Request ID generation/validation
- Logging output format
- Rate limiting (in-memory & Redis)
- Metrics collection
- Tracing span creation

## Performance Budgets

- **API Latency**: p95 < 300ms
- **Stream Join**: < 2s
- **Trust Recompute**: < 5m

## Security Best Practices

1. **Secrets Management**
   - Never log JWT secrets or API keys
   - Use environment variables, not config files
   - Mask secrets in log output

2. **Input Validation**
   - Validate all user input
   - Reject malformed request IDs
   - Use prepared statements for database queries

3. **Rate Limiting**
   - Apply rate limits to all endpoints
   - Use Redis for distributed deployments
   - Return appropriate headers

4. **CORS**
   - Strict allowlist (no wildcard origins)
   - Credentials support where needed

5. **Error Messages**
   - No sensitive data in error messages
   - Generic errors for authentication failures
   - Detailed logs server-side only

## Dependencies

### Core Dependencies
- `github.com/knadh/koanf/v2` - Configuration management
- `github.com/golang-jwt/jwt/v5` - JWT token handling
- `log/slog` - Structured logging (stdlib)
- `github.com/google/uuid` - Request ID generation

### Optional Dependencies
- `github.com/redis/go-redis/v9` - Redis rate limiting backend
- `github.com/prometheus/client_golang` - Metrics collection
- `go.opentelemetry.io/otel` - Distributed tracing

## Known Issues

### libvips Dependency

The `github.com/h2non/bimg` package (used in `internal/image` and `internal/attachment`) requires libvips 8.x+, a C library for image processing. This is needed for EXIF stripping and image optimization.

**Docker Deployment**: libvips is installed in the build container automatically.

**Local Development**: Install libvips manually:
```bash
# macOS
brew install vips

# Ubuntu/Debian
apt-get install libvips-dev

# Alpine
apk add vips-dev
```

**Without libvips**: The API will fail to build. Use Docker for development if you don't need to work on image processing features.

## Next Steps

1. ✅ **Backend Core Complete** - All foundational components implemented
2. ⏭️ **Feature Integration** - Connect scenes, events, payments endpoints
3. ⏭️ **Production Deployment** - Deploy to staging environment
4. ⏭️ **Monitoring Setup** - Configure Prometheus/Grafana dashboards
5. ⏭️ **Load Testing** - Verify performance under load

## References

- Issue #3: Epic: Backend Core (Go API Service)
- Issue #25: Task: JWT Auth Module
- Issue #29: Task: Structured Logging Middleware
- Issue #34: Task: Rate Limiting Middleware
- Issue #37: Task: Graceful Shutdown Handling
- Issue #46: Task: API Config Loader (koanf)
- Issue #52: Task: Standard Error Response Format
- Issue #53: Task: Request ID Middleware
