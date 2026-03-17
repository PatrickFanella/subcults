# Troubleshooting Guide

Quick reference for common issues and solutions when developing Subcults.

## Table of Contents

- [Build Problems](#build-problems)
- [Dependency Issues](#dependency-issues)
- [Database Connection Issues](#database-connection-issues)
- [Runtime Issues](#runtime-issues)
- [Network Problems](#network-problems)
- [Docker Issues](#docker-issues)
- [Performance Issues](#performance-issues)
- [Logging and Debug Mode](#logging-and-debug-mode)

---

## Build Problems

### Go Build Fails With CGO Errors

**Symptoms:** `cgo: C compiler "gcc" not found` or linker errors referencing `vips`.

**Cause:** The API binary requires `libvips` via the `bimg` package (CGO enabled).

**Solution:**

```bash
# macOS
brew install vips pkg-config

# Ubuntu/Debian
sudo apt install libvips-dev gcc pkg-config

# Alpine (inside Docker)
apk add --no-cache gcc musl-dev pkgconfig vips-dev
```

Verify with `vips --version` (requires 8.15+).

### Frontend Build Fails: Missing Dependencies

**Symptoms:** `Module not found` errors when running `make build-frontend` or `npm run build`.

**Solution:**

```bash
cd web
rm -rf node_modules
npm ci
```

If `package-lock.json` is missing, run `npm install` instead of `npm ci`.

### Binary Not Found After Build

**Symptoms:** `make dev-api` or `./bin/api` fails with "no such file or directory".

**Solution:**

```bash
make build       # builds all binaries to bin/
ls -la bin/      # verify api, indexer, backfill exist
```

---

## Dependency Issues

### `go mod download` Fails

**Symptoms:** Timeout or authentication errors pulling Go modules.

**Cause:** Network issues, proxy misconfiguration, or private repo access.

**Solution:**

```bash
# Check proxy settings
go env GOPROXY    # should be "https://proxy.golang.org,direct"

# Reset module cache
go clean -modcache
go mod download
```

### Mismatched Go Version

**Symptoms:** `go: go.mod requires go >= 1.22` or unexpected syntax errors.

**Solution:**

```bash
go version        # check current version
# Install Go 1.22+ from https://go.dev/dl/
```

### Node Version Mismatch

**Symptoms:** Frontend build errors about unsupported syntax or engine requirements.

**Solution:** Use Node.js 22+ (matches Dockerfile.frontend).

```bash
node --version    # check current
nvm use 22        # if using nvm
```

---

## Database Connection Issues

### Cannot Connect to PostgreSQL

**Symptoms:** `connection refused` or `FATAL: role "subcults" does not exist`.

**Investigation:**

```bash
# 1. Check if PostgreSQL is running
make compose-up
docker ps | grep subcults-postgres

# 2. Verify health
docker exec subcults-postgres pg_isready -U subcults

# 3. Check port binding (default: 5439)
ss -tlnp | grep 5439
```

**Common Causes:**
- Docker Compose not started: run `make compose-up`
- Port conflict: another service on port 5439
- Missing `.env` file: `cp configs/dev.env.example .env` and set `POSTGRES_PASSWORD`

### Migration Fails

**Symptoms:** `error: Dirty database version X. Fix and force version.`

**Cause:** A previous migration was interrupted, leaving the schema in a dirty state.

**Solution:**

```bash
# Check current version
./scripts/migrate.sh version

# Force to the last known good version (replace N with the version)
./scripts/migrate.sh force N

# Re-run migrations
make migrate-up
```

### `DATABASE_URL` Not Set

**Symptoms:** `required key DATABASE_URL missing` on startup.

**Solution:**

```bash
# Option 1: Export from env file
export $(grep -v '^#' configs/dev.env | xargs)

# Option 2: Set directly (local Docker Compose)
export DATABASE_URL="postgres://subcults:yourpassword@localhost:5439/subcults?sslmode=disable"

# Verify
echo $DATABASE_URL
make migrate-up
```

### PostGIS Extension Missing

**Symptoms:** `ERROR: type "geography" does not exist` during migrations.

**Cause:** Using plain PostgreSQL instead of PostGIS.

**Solution:** Use the PostGIS image: `postgis/postgis:16-3.4-alpine` (used by docker-compose.yml by default). If connecting to a remote database, ensure PostGIS is installed:

```sql
CREATE EXTENSION IF NOT EXISTS postgis;
```

---

## Runtime Issues

### API Fails to Start

**Symptoms:** Exits immediately or panics on startup.

**Investigation:**

```bash
# 1. Check required environment variables
env | grep -E "DATABASE_URL|JWT_SECRET|SUBCULT"

# 2. Run with verbose output
SUBCULT_ENV=development ./bin/api

# 3. Verify database connectivity
curl http://localhost:8080/health/ready
```

**Common Causes:**
- Missing `JWT_SECRET` or `JWT_SECRET_CURRENT` (min 32 characters)
- Invalid `DATABASE_URL` format
- Port already in use: check `SUBCULT_PORT` (default 8080)

### JWT Authentication Errors

**Symptoms:** `401 Unauthorized` on all authenticated endpoints.

**Common Causes:**
- `JWT_SECRET` shorter than 32 characters
- Clock skew between client and server (tokens expire after 15 min)
- Using expired refresh token (7-day expiry)

**Solution:**

```bash
# Generate a proper secret
openssl rand -base64 32

# Set it
export JWT_SECRET_CURRENT="<generated-secret>"
```

### Indexer Disconnects From Jetstream

**Symptoms:** Indexer stops processing events, logs show WebSocket disconnect.

**Cause:** Network interruption or Jetstream server maintenance.

**Solution:** The indexer has automatic reconnection with exponential backoff. Check logs for reconnection attempts:

```bash
# View indexer logs
make dev-indexer 2>&1 | grep -i "reconnect\|connect\|error"
```

If the indexer is stuck, restart it. Backfill missed records with:

```bash
./bin/backfill --since "2024-01-01T00:00:00Z"
```

### Stripe Webhook Verification Fails

**Symptoms:** `400 Bad Request` on `/webhooks/stripe` endpoint.

**Common Causes:**
- `STRIPE_WEBHOOK_SECRET` mismatch (must start with `whsec_`)
- Request body was modified by a proxy
- Clock skew > 5 minutes

**Solution:**

```bash
# For local testing, use Stripe CLI
stripe listen --forward-to localhost:8080/webhooks/stripe

# Copy the webhook signing secret from the CLI output
export STRIPE_WEBHOOK_SECRET="whsec_..."
```

---

## Network Problems

### CORS Errors in Browser

**Symptoms:** Console shows `Access-Control-Allow-Origin` errors.

**Cause:** Frontend origin not in the API's allowed origins list.

**Solution:** Check `CORS_ALLOWED_ORIGINS` in your config. For local dev:

```bash
export CORS_ALLOWED_ORIGINS="http://localhost:5173,http://localhost:3000"
```

### MapTiler Tiles Not Loading

**Symptoms:** Map shows grey/blank tiles or 403 errors in network tab.

**Common Causes:**
- `VITE_MAPTILER_API_KEY` not set or invalid
- API key domain restrictions don't include `localhost`

**Solution:**

```bash
# Set in web/.env.local or configs/dev.env
VITE_MAPTILER_API_KEY="your-key-here"
```

Verify at https://cloud.maptiler.com/account/keys/ that `localhost` is in the allowed domains.

### API Health Check Fails

**Symptoms:** `curl http://localhost:8080/health/live` returns connection refused.

**Investigation:**

```bash
# Check if process is running
pgrep -f "bin/api"

# Check port
ss -tlnp | grep 8080

# Try alternate port
SUBCULT_PORT=9000 ./bin/api
curl http://localhost:9000/health/live
```

---

## Docker Issues

### Docker Build Context Too Large

**Symptoms:** `Sending build context to Docker daemon` takes a long time.

**Cause:** Missing or incomplete `.dockerignore`.

**Solution:** The project includes a `.dockerignore` file. Verify it exists and includes `node_modules/`, `.git/`, `docs/`, `e2e/`, `perf/`.

### Container Fails Health Check

**Symptoms:** Container restarts repeatedly, `docker ps` shows `(unhealthy)`.

**Investigation:**

```bash
# Check container logs
docker logs subcults-api --tail 50

# Check health check result
docker inspect --format='{{json .State.Health}}' subcults-api | python3 -m json.tool
```

### Image Size Too Large

**Symptoms:** Docker images exceed target size budgets.

**Investigation:**

```bash
make docker-build
make docker-size

# Analyze layers
docker history subcults-api:latest
```

**Target sizes:** API <30MB (alpine+vips), Indexer <10MB (distroless), Frontend <20MB (nginx).

---

## Performance Issues

### API Latency Exceeds p95 Budget (>300ms)

**Investigation:**

```bash
# Check database query performance
curl http://localhost:8080/health/ready   # includes DB check timing

# Monitor metrics endpoint
curl http://localhost:9090/metrics | grep api_request_duration
```

**Common Causes:**
- Missing database indexes (check `EXPLAIN ANALYZE` on slow queries)
- N+1 queries in handlers (use batch fetching)
- Large result sets without pagination

### High Memory Usage

**Investigation:**

```bash
# Go runtime metrics
curl http://localhost:9090/metrics | grep go_memstats

# Profile (if pprof enabled)
go tool pprof http://localhost:8080/debug/pprof/heap
```

**Common Causes:**
- In-memory repositories growing unbounded (dev only; production uses Postgres)
- Goroutine leaks from unclosed WebSocket connections

---

## Logging and Debug Mode

### Enable Development Logging

Set the environment to `development` for human-readable log output:

```bash
export SUBCULT_ENV=development
```

- **Development:** text format, `DEBUG` level
- **Production:** JSON format, `INFO` level

### Structured Log Fields

All HTTP requests include these fields automatically:

| Field | Description |
|-------|-------------|
| `request_id` | Unique ID per request (from `X-Request-ID` header or auto-generated) |
| `method` | HTTP method |
| `path` | Request path |
| `status` | Response status code |
| `latency_ms` | Request duration |
| `user_did` | Authenticated user's DID (if present) |
| `error_code` | Application error code (for 4xx/5xx) |

### Filter Logs by Component

```bash
# API errors only
make dev-api 2>&1 | grep '"level":"ERROR"'

# Specific request
make dev-api 2>&1 | grep "request_id.*abc123"

# Database issues
make dev-api 2>&1 | grep -i "database\|postgres\|sql"
```

### Health Check Endpoints

```bash
# Liveness (is the process running?)
curl http://localhost:8080/health/live
# {"status":"ok"}

# Readiness (can it serve traffic?)
curl http://localhost:8080/health/ready
# {"status":"ok","checks":{"database":"ok"}}
```

### Metrics Endpoint

```bash
# Prometheus metrics (API)
curl http://localhost:9090/metrics

# Key metrics to watch
curl -s http://localhost:9090/metrics | grep -E "api_request_duration|go_goroutines|go_memstats_alloc"
```

---

## Quick Diagnostic Checklist

When something isn't working, run through this checklist:

1. **Environment loaded?** `env | grep SUBCULT`
2. **Database running?** `docker ps | grep postgres`
3. **Database accessible?** `curl localhost:8080/health/ready`
4. **Migrations current?** `./scripts/migrate.sh version`
5. **Ports free?** `ss -tlnp | grep -E "8080|5439|5173"`
6. **Dependencies installed?** `go version && node --version && vips --version`
7. **Clean build?** `make clean && make build`
