# Developer Onboarding Guide

Get from zero to running all three services in under 30 minutes.

## Prerequisites

Install these before starting:

| Tool | Version | Install |
|------|---------|---------|
| Go | 1.24+ | [go.dev/dl](https://go.dev/dl/) |
| Node.js | 20+ LTS | [nodejs.org](https://nodejs.org/) |
| Docker | 24+ | [docs.docker.com](https://docs.docker.com/get-docker/) |
| Make | Any | Included on macOS/Linux |
| libvips | 8.15+ | `brew install vips` / `apt install libvips-dev` |

Verify installations:

```bash
go version       # go1.24+
node --version   # v20+
docker --version # Docker 24+
make --version
vips --version   # libvips 8.15+
```

## Step 1: Clone and Configure (~5 minutes)

```bash
git clone https://github.com/subculture-collective/subcults.git
cd subcults
```

Copy the environment template:

```bash
cp configs/dev.env.example configs/dev.env
```

Edit `configs/dev.env` and fill in the required values:

| Variable | How to Get It |
|----------|---------------|
| `DATABASE_URL` | From your Neon Postgres dashboard or local Postgres |
| `JWT_SECRET_CURRENT` | Generate: `openssl rand -base64 32` |
| `LIVEKIT_URL` | From [LiveKit Cloud](https://cloud.livekit.io/) or local instance |
| `LIVEKIT_API_KEY` | From LiveKit dashboard |
| `LIVEKIT_API_SECRET` | From LiveKit dashboard |
| `STRIPE_API_KEY` | From [Stripe Dashboard](https://dashboard.stripe.com/test/apikeys) (use test key) |
| `STRIPE_WEBHOOK_SECRET` | From Stripe webhook settings |
| `MAPTILER_API_KEY` | From [MapTiler](https://cloud.maptiler.com/) (free tier) |
| `JETSTREAM_URL` | `wss://jetstream2.us-east.bsky.network/subscribe` |
| `R2_BUCKET_NAME` | From Cloudflare R2 dashboard |
| `R2_ACCESS_KEY_ID` | From R2 API tokens |
| `R2_SECRET_ACCESS_KEY` | From R2 API tokens |
| `R2_ENDPOINT` | From R2 bucket settings |

For local development without external services, at minimum you need `DATABASE_URL` and `JWT_SECRET_CURRENT`.

## Step 2: Start Infrastructure (~5 minutes)

Start Postgres with PostGIS:

```bash
make compose-up
```

This starts:
- **PostgreSQL 16** with PostGIS on `localhost:5439`

Verify it's running:

```bash
docker compose ps
# postgres should show "healthy"
```

## Step 3: Run Database Migrations (~2 minutes)

```bash
# Source environment variables
export $(grep -v '^#' configs/dev.env | xargs)

# Apply all migrations
make migrate-up
```

This creates 30+ tables including scenes, events, posts, users, alliances, payments, streams, and audit logs.

Verify:

```bash
./scripts/migrate.sh version
# Should show the latest migration number (e.g., 30)
```

## Step 4: Install Frontend Dependencies (~3 minutes)

```bash
cd web && npm ci && cd ..
```

## Step 5: Run Everything (~1 minute)

```bash
make dev
```

This starts:
- **API server** at `http://localhost:8080`
- **Frontend dev server** at `http://localhost:5173`

Alternatively, run services individually:

```bash
make dev-api        # API only (Go with hot reload)
make dev-frontend   # Frontend only (Vite HMR)
make dev-indexer    # Jetstream indexer (AT Protocol consumer)
```

## Step 6: Verify (~2 minutes)

Check the API health endpoint:

```bash
curl http://localhost:8080/health/live
# {"status":"ok"}

curl http://localhost:8080/health/ready
# {"status":"ok","checks":{"database":"ok"}}
```

Open the frontend:

```bash
open http://localhost:5173
```

Run the test suite:

```bash
make test
```

## Common Tasks

### Adding an API Endpoint

1. Create handler in `internal/api/thing_handlers.go`
2. Add validation in `internal/validate/thing.go`
3. Register route in `cmd/api/main.go`
4. Write tests in `internal/api/thing_handlers_test.go`
5. See `docs/BACKEND_DEVELOPMENT_GUIDE.md` for patterns

### Creating a Frontend Page

1. Create page component in `web/src/pages/ThingPage.tsx`
2. Add route in `web/src/routes/`
3. Use Tailwind CSS for styling
4. Add i18n keys in translation files
5. Write tests with React Testing Library
6. Add accessibility test (`.a11y.test.tsx`)

### Working with the Database

```bash
# Apply pending migrations
make migrate-up

# Rollback last migration
make migrate-down

# Check migration version
./scripts/migrate.sh version

# Connect to database directly
psql "$DATABASE_URL"
```

### Running Tests

```bash
make test             # All tests
make test-coverage    # With HTML coverage reports
make test-integration # Integration tests (Docker required)
make test-e2e         # Playwright E2E
```

See `docs/TESTING_GUIDE.md` for writing tests.

### Building Docker Images

```bash
make docker-build     # Build all images
make docker-size      # Show image sizes
```

### Viewing Logs

```bash
make logs             # All Docker Compose logs
make logs-api         # API service logs only
make logs-postgres    # Database logs only
```

## CLI Commands Cheat Sheet

| Command | What It Does |
|---------|-------------|
| `make dev` | Start API + frontend |
| `make test` | Run all tests |
| `make lint` | Go vet + ESLint |
| `make fmt` | Format Go code |
| `make migrate-up` | Apply database migrations |
| `make compose-up` | Start Docker services |
| `make compose-down` | Stop Docker services |
| `make build` | Build all Go binaries |
| `make clean` | Remove build artifacts |

## Project Structure at a Glance

```
cmd/api/           → API server entry point
cmd/indexer/       → AT Protocol consumer
internal/api/      → HTTP handlers
internal/auth/     → JWT management
internal/config/   → Configuration loading
internal/geo/      → Location privacy + geohash
internal/middleware/ → HTTP middleware stack
internal/scene/    → Core domain (scenes, events, ranking)
internal/validate/ → Input validation
web/src/           → React frontend
  components/      → Reusable UI components
  pages/           → Route pages
  hooks/           → Custom React hooks
  stores/          → Zustand state stores
  lib/             → API client, auth, telemetry
migrations/        → Database schema files
configs/           → Environment templates
docs/              → Documentation
scripts/           → Automation scripts
```

## Key Concepts

### Privacy-First Location

All location data respects user consent. The `allow_precise` flag controls whether precise coordinates are stored. When false, coordinates are jittered (deterministic noise) before being returned. See `internal/geo/` and `internal/scene/privacy_test.go`.

### AT Protocol Integration

Subcults uses AT Protocol (Bluesky's decentralized protocol) for user identity. Users authenticate with their DID (Decentralized Identifier). The indexer consumes real-time events from Jetstream.

### Trust Graph

Users form alliances with role-based trust weighting. Search ranking optionally incorporates trust scores (feature-flagged via `RANK_TRUST_ENABLED`).

### Live Streaming

LiveKit provides WebRTC audio streaming. The API generates room tokens and manages participant state. See `internal/livekit/` and `internal/stream/`.

## Troubleshooting

### Port Already in Use

```bash
# Check what's using port 8080
lsof -i :8080

# Check what's using port 5439 (Postgres)
lsof -i :5439

# Kill Docker containers and retry
make compose-down && make compose-up
```

### Migration Failures

```bash
# Check current version
./scripts/migrate.sh version

# Force to a specific version (use with caution)
./scripts/migrate.sh force <version>

# Ensure DATABASE_URL is set
echo $DATABASE_URL
```

### Docker Memory Issues

Docker Desktop defaults may be too low. Allocate at least 4GB RAM in Docker Desktop settings.

### libvips Not Found

The API requires libvips for image processing (CGO enabled):

```bash
# macOS
brew install vips

# Ubuntu/Debian
sudo apt install libvips-dev

# Verify
vips --version
```

### Tests Fail with Race Conditions

```bash
# Run with race detector to find the issue
go test -race -v ./internal/package/...

# Run multiple times to catch intermittent failures
go test -race -count=5 ./internal/package/...
```

### Frontend Build Errors

```bash
# Clear node_modules and reinstall
cd web && rm -rf node_modules && npm ci

# Clear Vite cache
cd web && rm -rf node_modules/.vite
```

## Getting Help

- Browse `docs/` for documentation on specific topics
- Check closed GitHub issues for past solutions
- Read `CONTRIBUTING.md` for workflow conventions
- Read `docs/BACKEND_DEVELOPMENT_GUIDE.md` for backend patterns
- Read `docs/TESTING_GUIDE.md` for testing guidance
