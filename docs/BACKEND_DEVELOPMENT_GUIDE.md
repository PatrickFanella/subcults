# Backend Development Guide

Patterns, conventions, and practical examples for working on the Subcults Go backend.

## Architecture Overview

The backend is a Go HTTP API serving the Subcults platform. The main entry point is `cmd/api/main.go`, which wires up configuration, database connections, repositories, handlers, middleware, and the HTTP server.

```
cmd/api/main.go          # Server initialization + routing
internal/api/            # HTTP handlers (scene, event, stream, payment, search, auth)
internal/auth/           # JWT token generation + validation
internal/config/         # koanf-based config loading (env vars + files)
internal/db/             # Database connection pool + slow query metrics
internal/geo/            # Geohash, jitter, proximity calculations
internal/middleware/      # RequestID, logging, CORS, rate limiting, security headers
internal/scene/          # Domain models, repositories, ranking, search
internal/validate/       # Input validation schemas
internal/payment/        # Stripe Connect integration
internal/upload/         # R2 signed URLs for media
internal/livekit/        # WebRTC token generation
internal/audit/          # Access logging with hash chains
internal/indexer/        # Jetstream WebSocket consumer
internal/telemetry/      # Error metrics + client telemetry
internal/tracing/        # OpenTelemetry setup
```

## Adding a New Endpoint

Follow this pattern when adding API endpoints:

### 1. Define the Handler

```go
// internal/api/thing_handlers.go

func HandleCreateThing(repo ThingRepository) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // 1. Extract authenticated user
        userDID := middleware.GetUserDID(r.Context())
        if userDID == "" {
            ctx := middleware.SetErrorCode(r.Context(), ErrCodeAuthFailed)
            WriteError(w, ctx, http.StatusUnauthorized, ErrCodeAuthFailed, "authentication required")
            return
        }

        // 2. Parse and validate request body
        var req CreateThingRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            ctx := middleware.SetErrorCode(r.Context(), ErrCodeBadRequest)
            WriteError(w, ctx, http.StatusBadRequest, ErrCodeBadRequest, "invalid request body")
            return
        }

        if errs := validate.CreateThing(req); len(errs) > 0 {
            ctx := middleware.SetErrorCode(r.Context(), ErrCodeValidation)
            WriteError(w, ctx, http.StatusBadRequest, ErrCodeValidation, errs[0])
            return
        }

        // 3. Business logic
        thing, err := repo.Create(r.Context(), req)
        if err != nil {
            slog.Error("failed to create thing", "error", err, "user_did", userDID)
            ctx := middleware.SetErrorCode(r.Context(), ErrCodeInternal)
            WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "failed to create thing")
            return
        }

        // 4. Return response
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusCreated)
        json.NewEncoder(w).Encode(thing)
    }
}
```

### 2. Register the Route

In `cmd/api/main.go`, register the handler on the router:

```go
mux.HandleFunc("POST /api/things", authMiddleware(api.HandleCreateThing(thingRepo)))
```

Rate-limited endpoints wrap with the rate limiter:

```go
mux.HandleFunc("POST /api/things",
    authMiddleware(
        rateLimiter.PerUser(10, time.Hour)(
            api.HandleCreateThing(thingRepo),
        ),
    ),
)
```

### 3. Add Validation

```go
// internal/validate/thing.go

func CreateThing(req api.CreateThingRequest) []string {
    var errs []string
    if strings.TrimSpace(req.Name) == "" {
        errs = append(errs, "name is required")
    }
    if len(req.Name) > 200 {
        errs = append(errs, "name must be 200 characters or less")
    }
    return errs
}
```

### 4. Write Tests

```go
// internal/api/thing_handlers_test.go

func TestHandleCreateThing_Success(t *testing.T) {
    repo := NewInMemoryThingRepository()
    handler := HandleCreateThing(repo)

    body := `{"name": "Test Thing"}`
    req := httptest.NewRequest("POST", "/api/things", strings.NewReader(body))
    req.Header.Set("Content-Type", "application/json")

    // Inject authenticated user into context
    ctx := middleware.SetUserDID(req.Context(), "did:plc:test123")
    req = req.WithContext(ctx)

    w := httptest.NewRecorder()
    handler.ServeHTTP(w, req)

    if w.Code != http.StatusCreated {
        t.Errorf("expected 201, got %d", w.Code)
    }
}
```

## Repository Pattern

All data access goes through repository interfaces. This allows in-memory implementations for testing and Postgres implementations for production.

### Interface Definition

```go
// internal/scene/repository.go

type SceneRepository interface {
    Create(ctx context.Context, scene *Scene) error
    GetByID(ctx context.Context, id string) (*Scene, error)
    Update(ctx context.Context, scene *Scene) error
    Delete(ctx context.Context, id string) error
    Search(ctx context.Context, query SearchQuery) ([]Scene, error)
}
```

### In-Memory Implementation (Testing)

```go
type InMemorySceneRepository struct {
    mu     sync.RWMutex
    scenes map[string]*Scene
}

func NewInMemorySceneRepository() *InMemorySceneRepository {
    return &InMemorySceneRepository{scenes: make(map[string]*Scene)}
}

func (r *InMemorySceneRepository) Create(ctx context.Context, s *Scene) error {
    r.mu.Lock()
    defer r.mu.Unlock()
    s.EnforceLocationConsent() // Privacy enforcement
    r.scenes[s.ID] = s.DeepCopy()
    return nil
}
```

### Privacy Enforcement

All repository methods that store location data **must** call `EnforceLocationConsent()`:

```go
func (r *PostgresSceneRepository) Create(ctx context.Context, s *Scene) error {
    s.EnforceLocationConsent() // Clears PrecisePoint if allow_precise=false
    // ... insert into database
}
```

This is the most critical privacy invariant in the codebase. Tests verify it:

```go
func TestScene_EnforceLocationConsent(t *testing.T) {
    s := &Scene{
        AllowPrecise: false,
        PrecisePoint: &Point{Lat: 40.7128, Lng: -74.0060},
    }
    s.EnforceLocationConsent()
    if s.PrecisePoint != nil {
        t.Error("PrecisePoint should be nil when allow_precise is false")
    }
}
```

## Error Handling

### Standardized Error Responses

All errors use a consistent JSON format:

```json
{
  "error": {
    "code": "validation_error",
    "message": "name is required"
  }
}
```

Use the `api.WriteError()` helper:

```go
api.WriteError(w, ctx, http.StatusBadRequest, api.ErrCodeValidation, "name is required")
```

### Error Codes

| Code | HTTP Status | Use |
|------|------------|-----|
| `validation_error` | 400 | Input validation failure |
| `bad_request` | 400 | Malformed request body |
| `invalid_time_range` | 400 | Event time constraints |
| `auth_failed` | 401 | Missing or invalid auth |
| `forbidden` | 403 | Insufficient permissions |
| `not_found` | 404 | Resource doesn't exist |
| `conflict` | 409 | Duplicate or state conflict |
| `rate_limited` | 429 | Too many requests |
| `internal_error` | 500 | Unexpected server error |

### Error Wrapping

Use `%w` for error chains that callers might need to inspect:

```go
return fmt.Errorf("failed to create scene: %w", err)
```

Use sentinel errors for expected conditions:

```go
var ErrSceneNotFound = errors.New("scene not found")

// In handler
if errors.Is(err, scene.ErrSceneNotFound) {
    api.WriteError(w, ctx, http.StatusNotFound, api.ErrCodeNotFound, "scene not found")
}
```

### Context Error Codes

Set error codes in context for the logging middleware to capture:

```go
ctx := middleware.SetErrorCode(r.Context(), api.ErrCodeNotFound)
api.WriteError(w, ctx, http.StatusNotFound, api.ErrCodeNotFound, "scene not found")
```

The logging middleware automatically logs `error_code` for 4xx/5xx responses.

## Configuration

Configuration is loaded via koanf from environment variables and optional YAML files. Env vars take precedence.

### Accessing Config

```go
cfg, warnings := config.Load()
for _, w := range warnings {
    slog.Warn("config warning", "warning", w)
}
```

### Required Variables

| Variable | Description |
|----------|-------------|
| `DATABASE_URL` | Postgres connection string (PostGIS enabled) |
| `JWT_SECRET_CURRENT` | Current JWT signing key (min 32 chars) |
| `LIVEKIT_URL` | LiveKit server URL |
| `STRIPE_API_KEY` | Stripe API key |

### Feature Flags

```go
if cfg.RankTrustEnabled {
    score += trustWeight
}
```

Always provide a fallback when the feature flag is disabled.

### Adding a New Config Key

1. Add field to the config struct in `internal/config/config.go`
2. Set default value if applicable
3. Add env var binding in the koanf setup
4. Add validation if required
5. Update `configs/dev.env.example`

## Logging

Use `slog` for all logging. Never use `fmt.Println` or `log.Println`.

```go
// Info level for normal operations
slog.Info("scene created", "scene_id", scene.ID, "user_did", userDID)

// Warn for recoverable issues
slog.Warn("rate limit approaching", "user_did", userDID, "requests", count)

// Error for failures requiring attention
slog.Error("payment failed", "error", err, "scene_id", sceneID, "amount", amount)
```

### Logging Rules

- **JSON format** in production, text in development (`SUBCULT_ENV` controls this)
- **Required fields** for HTTP logs: `request_id`, `method`, `path`, `status`, `latency_ms`
- **Never log**: passwords, full JWTs, precise coordinates (when consent is false), email addresses
- **Always log**: `user_did` (when authenticated), `error_code` (for failures), duration metrics

## Database Queries

### Parameterized Queries

Always use parameterized queries. Never concatenate user input into SQL:

```go
// Correct
row := db.QueryRowContext(ctx,
    "SELECT id, name FROM scenes WHERE id = $1 AND owner_did = $2",
    sceneID, userDID,
)

// NEVER do this
query := fmt.Sprintf("SELECT * FROM scenes WHERE id = '%s'", sceneID) // SQL injection!
```

### PostGIS Queries

For geospatial operations:

```go
rows, err := db.QueryContext(ctx,
    `SELECT id, name, ST_AsGeoJSON(point) as geojson
     FROM scenes
     WHERE ST_DWithin(point, ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography, $3)
     ORDER BY point <-> ST_SetSRID(ST_MakePoint($1, $2), 4326)
     LIMIT $4`,
    lng, lat, radiusMeters, limit,
)
```

### Migrations

Create new migrations with sequential numbering:

```bash
# Create migration files
touch migrations/000031_add_thing_table.up.sql
touch migrations/000031_add_thing_table.down.sql
```

Up migration:
```sql
CREATE TABLE things (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    owner_did TEXT NOT NULL REFERENCES users(did),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_things_owner ON things(owner_did);
```

Down migration:
```sql
DROP TABLE IF EXISTS things;
```

Apply with `make migrate-up`. Rollback with `make migrate-down`.

## Middleware Stack

The middleware chain in `cmd/api/main.go` executes in this order:

1. **Tracing** — OpenTelemetry spans (optional, `TRACING_ENABLED`)
2. **Profiling** — pprof endpoints (dev only, `PROFILING_ENABLED`)
3. **CORS** — Cross-origin handling (`CORS_ALLOWED_ORIGINS`)
4. **Canary** — Traffic splitting for deployments (`CANARY_ENABLED`)
5. **Rate Limiting** — Global: 1000 req/min per IP
6. **Security Headers** — HSTS, CSP, X-Frame-Options, etc.
7. **MaxBodySize** — 1MB JSON, 15MB file uploads
8. **HTTP Metrics** — Prometheus instrumentation
9. **RequestID** — UUID from `X-Request-ID` header or auto-generated
10. **Logging** — Structured slog with request_id, method, path, status, latency

### Adding Middleware

Wrap the handler chain in `cmd/api/main.go`:

```go
handler = yourMiddleware(handler)
```

Place it at the appropriate position in the chain. Logging should be innermost (closest to handler), tracing outermost.

## Rate Limiting

### Global Limits

Applied to all routes: 1000 requests/minute per IP.

### Per-Endpoint Limits

| Endpoint | Limit | Window |
|----------|-------|--------|
| Scene creation | 10/hour | Per user |
| Event creation | 5/hour | Per user |
| Alliance creation | 10/hour | Per user |
| Stream join | 10/minute | Per user |
| Search | 100/minute | Per user |
| Telemetry | 100/minute | Per IP |

### Implementation

```go
// Per-user rate limiting
mux.HandleFunc("POST /api/scenes",
    authMiddleware(
        rateLimiter.PerUser(10, time.Hour)(
            api.HandleCreateScene(sceneRepo),
        ),
    ),
)
```

Rate limiting uses Redis when `REDIS_URL` is set, otherwise falls back to in-memory with a 5-minute cleanup interval.

## Authentication

### JWT Tokens

- **Access tokens**: 15-minute expiry, contains `did` claim
- **Refresh tokens**: 7-day expiry, no DID claim

### Extracting User Identity

```go
userDID := middleware.GetUserDID(r.Context())
if userDID == "" {
    // Not authenticated
}
```

### Key Rotation

The system supports dual-key rotation for zero-downtime secret changes:

- `JWT_SECRET_CURRENT` — Signs new tokens
- `JWT_SECRET_PREVIOUS` — Still validates existing tokens

Rotate using `./scripts/rotate-jwt-secret.sh`.

## Performance Considerations

### Budgets

| Metric | Target |
|--------|--------|
| API latency (p95) | <300ms |
| Stream join | <2s |
| Map render | <1.2s |
| First contentful paint | <1.0s |

### Query Optimization

- Add indexes for columns used in WHERE/ORDER BY clauses
- Use `EXPLAIN ANALYZE` to verify query plans
- Slow query logging is enabled via `SLOW_QUERY_THRESHOLD_MS`
- Avoid N+1 queries — use JOINs or batch fetches

### Connection Pooling

The database connection pool is configured in `internal/db/`. Default settings are tuned for production workloads. Adjust via environment variables if needed.

## Security Checklist

Before merging any backend PR:

- [ ] All user input validated via `internal/validate`
- [ ] SQL queries use parameterized placeholders (`$1`, `$2`)
- [ ] Location data calls `EnforceLocationConsent()` before storage
- [ ] No PII in log messages
- [ ] Error responses don't leak internal details
- [ ] New endpoints have rate limiting
- [ ] Auth-required endpoints check `GetUserDID()`
- [ ] No hardcoded secrets or credentials
