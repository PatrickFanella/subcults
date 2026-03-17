# Deployment Guide

## Overview

Subcults runs on a VPS with Docker Compose and an external Caddy reverse proxy.
This guide focuses on deployment readiness for that setup and intentionally keeps
monitoring/observability optional.

## Architecture

```
Internet → Caddy (TLS + routing) → Docker Compose services
                                     ├── subcults-api:8080
                                     ├── subcults-indexer
                                     └── subcults-frontend:80
```

## Prerequisites

1. **Docker & Docker Compose v2+** on the VPS
2. **Caddy** running at `~/projects/caddy` on the `web` Docker network
3. **Neon Postgres** (managed, external) with PostGIS
4. **Environment file**: `deploy/.env` with all secrets populated

## Environment Setup

```bash
# Create the shared Docker network (if not exists)
docker network create web 2>/dev/null || true

# Copy and fill environment file
cp deploy/.env.example deploy/.env
# Edit deploy/.env with production values
```

Required environment variables are documented in `deploy/.env.example`.

## Standard Deployment

### Using the deploy script (recommended)

```bash
# Deploy latest version
./scripts/deploy.sh

# Check status
./scripts/deploy.sh --status

# Rollback if needed
./scripts/deploy.sh --rollback
```

### Manual deployment

```bash
# Pull latest code
git pull origin main

# Ensure network and env
docker network create web 2>/dev/null || true
cp deploy/.env.example deploy/.env # first time only

# Build and start services
cd deploy
docker compose build
docker compose up -d --force-recreate

# (Optional but recommended) run migrations from repo root
cd ..
make migrate-up-prod

# Verify health (inside containers; no host port publishing required)
docker compose -f deploy/compose.yml exec -T api wget --no-verbose --tries=1 --spider http://localhost:8080/health/live
docker compose -f deploy/compose.yml exec -T frontend wget --no-verbose --tries=1 --spider http://localhost/nginx-health
docker compose -f deploy/compose.yml exec -T indexer wget --no-verbose --tries=1 --spider http://localhost:9090/health
```

## Reverse Proxy Wiring (Caddy)

Subcults expects Caddy to reach containers on the shared `web` Docker network.

Route recommendations:

- `/api/*` → `subcults-api:8080`
- `/health/*` (or chosen health path) → `subcults-api:8080`
- `/` and static assets → `subcults-frontend:80`

The indexer is internal-only and not intended for direct public routing.

## Pre-deployment Checklist

Before deploying:

- [ ] `deploy/.env` created from `deploy/.env.example`
- [ ] Required secrets and service URLs set
- [ ] `CORS_ALLOWED_ORIGINS` set to your public frontend origin(s)
- [ ] Docker `web` network exists and Caddy is attached
- [ ] Images build successfully (`docker compose -f deploy/compose.yml build`)
- [ ] Migrations run successfully (`make migrate-up-prod`)
- [ ] Health checks pass in running containers

## Docker Images

| Service  | Dockerfile            | Base Image                              | Port            |
| -------- | --------------------- | --------------------------------------- | --------------- |
| API      | `Dockerfile.api`      | golang:1.24-alpine → alpine:3.21        | 8080            |
| Indexer  | `Dockerfile.indexer`  | golang:1.24-alpine → distroless:nonroot | 9090 (internal) |
| Frontend | `Dockerfile.frontend` | node:22-alpine → nginx:1-alpine-slim    | 80              |

Build manually:

```bash
make docker-build              # Build all
make docker-build-api          # API only
make docker-build-frontend     # Frontend only
make docker-build-indexer      # Indexer only
```

## Networking

Subcults containers use:

- **`web`** (external): shared with Caddy reverse proxy.
- **`subcults-internal`** (bridge): internal service communication.

Indexer is attached only to the internal network in Compose, reducing unnecessary exposure.

## Rollback Procedure

If deployment causes issues:

1. **Quick service restart**:

   ```bash
   ./scripts/deploy.sh --rollback
   ```

2. **Rollback to a known-good commit**:

   ```bash
   git checkout <known-good-commit>
   ./scripts/deploy.sh
   ```

3. **Rollback the last migration** (if needed):
   ```bash
   make migrate-down-prod
   ```

## Kubernetes (Future)

K8s manifests are in `deploy/k8s/` and Helm chart in `deploy/helm/subcults/`.

## Troubleshooting

### Service won't start

```bash
docker compose -f deploy/compose.yml logs api
docker compose -f deploy/compose.yml logs indexer
docker compose -f deploy/compose.yml logs frontend
```

### Health checks failing

```bash
docker compose -f deploy/compose.yml exec -T api wget --no-verbose --tries=1 --spider http://localhost:8080/health/live
docker compose -f deploy/compose.yml exec -T frontend wget --no-verbose --tries=1 --spider http://localhost/nginx-health
docker compose -f deploy/compose.yml exec -T indexer wget --no-verbose --tries=1 --spider http://localhost:9090/health
```

### Database connection issues

```bash
set -a && source deploy/.env && set +a
psql "$DATABASE_URL" -c "SELECT 1"
./scripts/migrate.sh version
```

### Caddy not routing traffic

```bash
# Check Caddy logs
docker logs caddy --tail 50

# Check containers are on web network
docker network inspect web | grep subcults
```
