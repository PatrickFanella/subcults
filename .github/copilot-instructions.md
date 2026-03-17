# Subcults - AI Agent Development Guide

## Design System

This project uses a unified design system. See [design-system.md](../design-system.md).

All UI development must follow design-system.md for:
- Color specifications (dark-first, vermillion accent)
- Typography rules (Inter Tight display, Inter body, JetBrains Mono data)
- Component styles (sharp corners, no shadows, underline-based interactions)
- Spacing system and layout strategy

## Project Overview

**Subcults** is a privacy-first platform for mapping underground music communities, combining real-time location awareness, live audio streaming, and trust-based discovery. The mission is to empower grassroots music scenes while respecting user privacy and enabling direct artist support.

**Core Principle**: Presence over popularity with scene sovereignty and explicit location consent.

### Architecture

Three main services orchestrated via Docker Compose:
- **API** (`cmd/api`): Go/chi REST API handling scenes, events, payments (Stripe), auth (JWT), media (R2)
- **Indexer** (`cmd/indexer`): Jetstream consumer ingesting AT Protocol records into Postgres
- **Frontend** (`web/`): Vite + React + TypeScript + MapLibre for map-based discovery

**Critical Data Flow**: AT Protocol → Jetstream → Indexer → Postgres ← API ← Frontend

## Core Principles

### 1. Privacy First

**ALL** scene/event location data must enforce consent via `allow_precise` flag:

```go
// Always call before persisting
scene.EnforceLocationConsent()  // Clears PrecisePoint if allow_precise=false
```

**Privacy Requirements**:
- **Geographic Privacy**: Public coordinates use geohash-based jitter (deterministic noise) to prevent precise location tracking
- **Consent Management**: Users explicitly opt-in for precise location sharing; default is always jittered
- **Minimal Data Collection**: Only collect what's necessary; avoid tracking user movements
- **Repository enforcement**: All repository methods automatically enforce consent ([internal/scene/repository.go](../internal/scene/repository.go))
- **Database constraints**: Migrations default `allow_precise=FALSE` with CHECK constraints ([migrations/000000_initial_schema.up.sql](../migrations/000000_initial_schema.up.sql))
- **Test coverage**: Privacy invariants verified in both models and repositories ([internal/scene/scene_test.go](../internal/scene/scene_test.go))

### 2. Trust-Based Discovery

- **Alliance System**: Users form alliances with role multipliers (organizer, artist, promoter, etc.)
- **Trust Scores**: Composite scores based on alliance strength and role weights
- **Ranking Integration**: Search results weighted by trust graph (feature-flagged for safe rollout)
- **No Centralized Curation**: Discovery driven by peer trust, not algorithmic feeds

**Search Ranking Formula**:
```text
composite_score = (text_relevance * 0.4) + (proximity_score * 0.3) + (recency * 0.2) + (trust_weight * 0.1)
Feature flag controls trust_weight inclusion; fallback to 0.0 when disabled
```

### 3. Direct Artist Support

- **Stripe Connect**: Artists onboard with Express accounts for direct payments
- **Application Fees**: Transparent platform fee on transactions
- **Zero Platform Lock-in**: Artists own their payment relationships

### 4. Real-Time & Live

- **AT Protocol Ingestion**: Real-time commit streaming via Jetstream WebSocket
- **LiveKit Streaming**: Low-latency audio streaming for live performances
- **Immediate Availability**: Content appears on map within seconds of posting

## Tech Stack

### Backend
- **Language**: Go 1.22+
- **Router**: chi
- **Database**: Neon Postgres 16 with PostGIS
- **Config**: koanf (YAML-based configuration)
- **Logging**: structured logging with `slog` (JSON in production, text in dev)
- **Auth**: JWT access + refresh tokens with dual-key rotation

### Frontend
- **Framework**: Vite + React + TypeScript
- **Build**: SWC for fast compilation
- **Styling**: Tailwind CSS with dark mode support
- **Maps**: MapLibre + MapTiler tiles
- **State**: Zustand or Redux
- **Routing**: react-router
- **i18n**: i18next

### Infrastructure
- **Containerization**: Docker Compose for local dev
- **Reverse Proxy**: Caddy
- **Streaming**: LiveKit Cloud (WebRTC SFU, TURN, token issuance)
- **Storage**: Cloudflare R2 for media uploads (S3-compatible)
- **Payments**: Stripe Connect Express
- **Deployment**: VPS with zero-downtime rollout

## Development Workflows

### Build & Test

```bash
make build          # All binaries to bin/
make build-api      # Just bin/api
make test           # Go tests (race detector + coverage)
make lint           # Go vet + frontend linters
```

**Testing Requirements**:
- **Backend**: >80% coverage; table-driven tests with descriptive names
- **Frontend**: >70% coverage; React Testing Library
- **E2E**: Playwright smoke tests for critical paths
- **Load**: k6 scenarios for API and streaming
- **Privacy**: Always test consent enforcement and jitter application

### Database Migrations

Uses [golang-migrate](https://github.com/golang-migrate/migrate). Requires `DATABASE_URL` env var.

```bash
make migrate-up     # Apply all pending
make migrate-down   # Rollback last migration
./scripts/migrate.sh version  # Check current version
```

Auto-detects local `migrate` binary or falls back to Docker. See [scripts/migrate.sh](../scripts/migrate.sh) for advanced usage.

### Docker Compose

```bash
make compose-up     # Start all services
make compose-down   # Stop all services
```

Override compose file: `make compose-up DOCKER_COMPOSE_FILE=docker-compose.dev.yml`

### Environment Setup

1. Copy `configs/dev.env.example` to `configs/dev.env`
2. Fill required secrets:
   - `DATABASE_URL`: Neon Postgres with PostGIS
   - `JWT_SECRET`: Min 32 chars (generate: `openssl rand -base64 32`)
   - `LIVEKIT_*`: WebRTC credentials for live audio
   - `STRIPE_*`: Payment integration
   - `R2_*`: Cloudflare R2 for media storage
   - `MAPTILER_API_KEY`: Map tiles

## Code Conventions

### Go Style

- `internal/` for private app code (auth, middleware, domain models)
- `pkg/` for reusable packages (use sparingly)
- `cmd/` for entry points (minimal logic, just flag parsing + initialization)
- Structured logging with `slog` (JSON in production, text in dev)
- Context values use private types (e.g., `type userDIDKey struct{}`)

### Error Handling

```go
// Use structured errors with context
return fmt.Errorf("failed to create scene: %w", err)

// Package-level sentinel errors
var ErrInvalidToken = errors.New("invalid token")

// Context-aware error codes in responses
SetErrorCode(ctx, "auth_failed")

// Log errors with relevant fields
slog.Error("scene creation failed", "user_id", userID, "error", err)

// Middleware logs errors at appropriate levels (5xx=error, 4xx=warn)
```

### Geographic Operations

```go
// Always apply jitter for public display
jitteredCoords := geo.ApplyJitter(coords, consentLevel)

// Check consent before returning precise coordinates
if !user.HasPreciseLocationConsent(requesterID) {
    coords = geo.ApplyJitter(coords, geo.DefaultPrecision)
}
```

### Feature Flags

```go
// Gate experimental features
if config.Features.TrustRanking {
    score += trustWeight
}
```

### Metrics & Observability

```go
// Instrument critical paths
metrics.APILatency.Observe(elapsed.Seconds())
metrics.StreamJoinCount.Inc()
```

### Authentication

JWT-based auth with two token types ([internal/auth/jwt.go](../internal/auth/jwt.go)):
- **Access tokens**: 15min expiry, includes `did` claim (Decentralized Identifier)
- **Refresh tokens**: 7 day expiry, no DID
- Middleware injects `user_did` into context via `SetUserDID(ctx, did)`

### Request Tracing

All HTTP requests use structured logging with required fields ([internal/middleware/logging.go](../internal/middleware/logging.go)):
- `request_id` (via `X-Request-ID` header or auto-generated UUID)
- `method`, `path`, `status`, `latency_ms`, `size`
- `user_did` (if authenticated)
- `error_code` (for 4xx/5xx responses)

### Frontend Patterns

- Use hooks for component-local state
- Zustand/Redux for global app state
- React Query for server state caching
- Minimize prop drilling; prefer context for cross-cutting concerns
- Use Tailwind utilities for all styling
- Dark mode via `dark:` prefix
- Implement accessibility (ARIA, keyboard nav)
- Add i18n for user-facing text

### Dockerfiles

All services use multi-stage builds:
- **Build stage**: `golang:1.22-alpine` with cross-compilation support
- **Runtime stage**: `gcr.io/distroless/static-debian12:nonroot` for minimal attack surface
- CGO disabled for static binaries
- Target: <60MB images (current: ~3.4MB ✓)
- No secrets baked in - runtime env vars only

## Project Structure

```
cmd/             # Entry points (api, indexer, backfill)
internal/        # Private application code
  ├── api/       # HTTP handlers
  ├── auth/      # JWT token management
  ├── audit/     # Privacy-compliant access logging
  ├── config/    # koanf-based configuration
  ├── db/        # Database access layer
  ├── geo/       # Geohash, jitter utilities
  ├── indexer/   # Jetstream WebSocket client
  ├── middleware/# HTTP middleware (logging, request ID)
  ├── scene/     # Domain models + repositories
  ├── trust/     # Trust graph computation
  └── validate/  # Input validation
pkg/             # Reusable packages
web/             # Frontend React app
  ├── src/
  │   ├── components/
  │   ├── hooks/
  │   ├── services/
  │   └── stores/
migrations/      # Database schema changes
scripts/         # Build/automation (migrate.sh)
docs/            # Documentation
configs/         # Environment templates
perf/            # Performance baselines
```

## Performance Budgets

- **API Latency**: p95 <300ms
- **Stream Join**: <2s
- **Map Render**: <1.2s
- **FCP**: <1.0s
- **Trust Recompute**: <5m

## Security Practices

- **Input Validation**: All user input sanitized via validation layer
- **Rate Limiting**: Per-endpoint buckets with Redis backend
- **CORS**: Strict allowlist; no wildcard origins
- **CSP**: Report-only → enforce progression
- **Audit Logging**: Hash chain for tamper detection
- **Secret Rotation**: Dual JWT key support for zero-downtime rotation
- **Dependency Scanning**: govulncheck + npm audit in CI
- **SSRF Protection**: URL allowlist for external fetches

## Common Tasks

### Adding a New API Endpoint

1. Define handler in `internal/api/`
2. Add route in router setup
3. Implement validation schema
4. Add database queries if needed
5. Write unit tests
6. Update OpenAPI spec
7. Add metrics instrumentation

### Adding a New Database Table

1. Create migration in `migrations/`
2. Update schema documentation in `migrations/README.md`
3. Add model struct in appropriate `internal/` package
4. Implement query functions
5. Add indexes for performance
6. Write migration tests
7. Consider privacy implications (location data, PII)

### Adding a New Frontend Component

1. Create component in `web/src/components/`
2. Use Tailwind for styling
3. Implement accessibility (ARIA, keyboard nav)
4. Add i18n for user-facing text
5. Write unit tests with React Testing Library
6. Add to Storybook if needed

## Roadmap & Current Status

**Phase**: Foundation & Initial Scaffolding (Phase 0)  
**Active Development**: 200+ GitHub issues across 24 epics

**Major Epics**:
- ✓ **Containerized stack** - Docker Compose setup complete
- ✓ **Core schema** - Scenes/events with location consent
- ✓ **Auth scaffolding** - JWT implementation complete
- ✓ **Migration tooling** - golang-migrate setup
- **Backend Core** (#3): Chi router setup, handlers, config management
- **Jetstream Indexer** (#5): AT Protocol real-time ingestion via WebSocket
- **Scene/Event/Post Management** (#10, #15, #17): CRUD APIs for core entities
- **Trust Graph** (#24): Alliance system + ranking algorithm (feature flagged)
- **LiveKit Streaming** (#23): WebRTC audio rooms with participant controls
- **Mapping Frontend** (#21): MapLibre + clustering with privacy jitter visualization
- **Stripe Connect** (#22): Direct scene payouts with platform fees
- **Security & Hardening** (#20): Rate limiting, CORS, secret rotation
- **Observability** (#19): Prometheus metrics, OpenTelemetry tracing, dashboards
- **Performance** (#17): Query optimization, caching strategies, latency budgets
- **Documentation** (#12): Architecture docs, API reference, onboarding guides

See issues for detailed acceptance criteria and step-by-step implementation plans.

## Common Pitfalls

1. **Never persist location data without calling `EnforceLocationConsent()`** - repositories enforce this but models allow mutation
2. **Middleware ordering matters**: `RequestID` → `Logging` (logging needs request ID in context)
3. **Migration scripts need DATABASE_URL** - export before running `make migrate-*`
4. **In-memory repos return deep copies** - prevents external mutation, but wastes memory in production (use Postgres repo)
5. **Issue templates guide implementation** - check GitHub issues for detailed steps and acceptance criteria before starting work
6. **Feature flags gate experimental features** - always provide fallbacks when features are disabled

## Git Workflow

### Branch Naming
- `feature/issue-123-short-description`
- `fix/issue-456-bug-description`
- `docs/issue-789-update-readme`

### Commit Messages
```text
type(scope): brief description

Longer explanation if needed.

Fixes #123
```

Types: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`

### Pull Requests
- Link to issue: "Closes #123"
- Follow PR template
- Ensure CI passes
- Request Copilot review for automated feedback
- Minimum one human review for production code

## External Dependencies

- **Neon Postgres 16**: Primary database with PostGIS for geo queries
- **AT Protocol/Jetstream**: Decentralized identity and data ingestion
- **LiveKit Cloud**: WebRTC SFU for live audio rooms
- **Stripe Connect**: Direct scene payouts with platform fee
- **Cloudflare R2**: S3-compatible media storage
- **MapTiler**: Map tiles for MapLibre frontend

## Agent-Specific Guidance

When working on this codebase:

1. **Always prioritize privacy**: Check consent flags, apply jitter, avoid logging PII
2. **Follow security best practices**: Validate input, parameterize queries, check CORS
3. **Instrument everything**: Add metrics, structured logs, tracing spans
4. **Test thoroughly**: Unit, integration, and E2E coverage for all new features
5. **Document decisions**: Use ADR process for architectural choices
6. **Check dependencies**: Reference linked issues before starting work
7. **Maintain performance**: Verify against budgets, capture EXPLAIN plans
8. **Respect feature flags**: Gate experimental features, provide fallbacks
9. **Privacy-aware queries**: All location-based queries must respect consent flags
10. **Context propagation**: Always pass context through call chains for tracing and cancellation

**Question everything that might compromise user privacy or trust.**
