#!/usr/bin/env bash
set -euo pipefail

# =============================================================================
# Deployment Script for Subcults (Docker Compose)
# =============================================================================
# Builds images, optionally runs migrations, recreates services, and verifies
# health from inside containers (works even when service ports are not published).
#
# Usage:
#   ./scripts/deploy.sh                # Deploy latest from current branch
#   ./scripts/deploy.sh --redeploy     # Explicitly trigger redeploy
#   ./scripts/deploy.sh --status       # Show current active slot
#   ./scripts/deploy.sh --rollback     # Restart services with currently available images
#
# Prerequisites:
#   - Docker Compose v2+
#   - deploy/compose.yml configured
#   - deploy/.env populated (copy deploy/.env.example first)
#   - External Caddy proxy on 'web' network
# =============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
COMPOSE_DIR="$PROJECT_DIR/deploy"
COMPOSE_FILE="$COMPOSE_DIR/compose.yml"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log()  { echo -e "${GREEN}[deploy]${NC} $*"; }
warn() { echo -e "${YELLOW}[deploy]${NC} $*"; }
err()  { echo -e "${RED}[deploy]${NC} $*" >&2; }

compose() {
    docker compose -f "$COMPOSE_FILE" "$@"
}

assert_prerequisites() {
    [[ -f "$COMPOSE_FILE" ]] || {
        err "Compose file not found: $COMPOSE_FILE"
        exit 1
    }

    if [[ ! -f "$COMPOSE_DIR/.env" ]]; then
        err "Missing $COMPOSE_DIR/.env"
        warn "Create it with: cp $COMPOSE_DIR/.env.example $COMPOSE_DIR/.env"
        exit 1
    fi
}

load_env() {
    # shellcheck disable=SC1090
    set -a && source "$COMPOSE_DIR/.env" && set +a
}

wait_for_http() {
    local service="$1"
    local url="$2"
    local max_attempts="${3:-30}"
    local interval="${4:-2}"

    log "Waiting for $service to become healthy at $url..."
    for i in $(seq 1 "$max_attempts"); do
        if compose exec -T "$service" sh -c "wget --no-verbose --tries=1 --spider '$url'" >/dev/null 2>&1; then
            log "Health check passed (attempt $i/$max_attempts)"
            return 0
        fi
        sleep "$interval"
    done
    err "Health check failed for $service after $max_attempts attempts"
    return 1
}

wait_for_container_health() {
    local service="$1"
    local max_attempts="${2:-90}"
    local interval="${3:-2}"

    local cid
    cid="$(compose ps -q "$service" 2>/dev/null || true)"
    if [[ -z "$cid" ]]; then
        err "No container found for service '$service'"
        return 1
    fi

    log "Waiting for $service container health to become healthy..."
    for i in $(seq 1 "$max_attempts"); do
        local state health
        state="$(docker inspect --format '{{.State.Status}}' "$cid" 2>/dev/null || echo unknown)"
        health="$(docker inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}}' "$cid" 2>/dev/null || echo unknown)"

        if [[ "$health" == "healthy" ]]; then
            log "$service health is healthy (attempt $i/$max_attempts)"
            return 0
        fi

        # If no healthcheck is defined, fall back to running state.
        if [[ "$health" == "none" && "$state" == "running" ]]; then
            log "$service is running (no healthcheck) (attempt $i/$max_attempts)"
            return 0
        fi

        sleep "$interval"
    done

    warn "$service health did not become healthy within timeout"
    return 1
}

run_migrations() {
    load_env

    if [[ -z "${DATABASE_URL:-}" ]]; then
        warn "DATABASE_URL is empty in deploy/.env; skipping migrations"
        return 0
    fi

    log "Running database migrations..."
    if (
        cd "$PROJECT_DIR" && \
        DATABASE_URL="$DATABASE_URL" \
        MIGRATE_USE_DOCKER=1 \
        MIGRATE_DOCKER_NETWORK=subcults-internal \
        ./scripts/migrate.sh up
    ); then
        log "Migrations completed"
    else
        warn "Migration command failed — continuing deploy (run migrations manually if required)"
    fi
}

run_smoke_tests() {
    local failures=0

    log "Running smoke tests..."

    # Test liveness
    if ! compose exec -T api sh -c "wget --no-verbose --tries=1 --spider 'http://localhost:8080/health/live'" >/dev/null 2>&1; then
        err "  FAIL: /health/live"
        ((failures++))
    else
        log "  PASS: /health/live"
    fi

    # Test readiness
    if ! compose exec -T api sh -c "wget --no-verbose --tries=1 --spider 'http://localhost:8080/health/ready'" >/dev/null 2>&1; then
        warn "  WARN: /health/ready (may need dependencies)"
    else
        log "  PASS: /health/ready"
    fi

    # Frontend nginx health (probe from API container on shared web network)
    if ! compose exec -T api sh -c "wget --no-verbose --tries=1 --spider 'http://frontend/nginx-health'" >/dev/null 2>&1; then
        err "  FAIL: frontend /nginx-health"
        ((failures++))
    else
        log "  PASS: frontend /nginx-health"
    fi

    # Indexer health: prefer container health status to avoid transient startup noise.
    if wait_for_container_health "indexer" 1 1 >/dev/null 2>&1; then
        log "  PASS: indexer container health"
    elif compose exec -T indexer sh -c "wget --no-verbose --tries=1 --spider 'http://localhost:9090/health'" >/dev/null 2>&1; then
        log "  PASS: indexer /health"
    else
        warn "  WARN: indexer health check"
    fi

    if [[ "$failures" -gt 0 ]]; then
        err "Smoke tests failed ($failures failures)"
        return 1
    fi

    log "All smoke tests passed"
    return 0
}

# Show deployment status
cmd_status() {
    log "Compose file: $COMPOSE_FILE"

    echo ""
    compose ps 2>/dev/null || warn "No containers running"
}

# Main deploy flow
cmd_deploy() {
    assert_prerequisites

    log "=== Deploying Subcults ==="
    log ""

    # Step 1: Build new images
    log "[1/4] Building images..."
    compose build api indexer frontend

    # Step 2: Run database migrations (if needed)
    log "[2/4] Running database migrations..."
    run_migrations

    # Step 3: Recreate services
    log "[3/4] Recreating services..."
    compose up -d --force-recreate api indexer frontend

    # Step 4: Wait for health + smoke tests
    log "[4/4] Running health checks and smoke tests..."
    if ! wait_for_http "api" "http://localhost:8080/health/live" 30 2; then
        err "API failed health checks"
        exit 1
    fi

    if ! wait_for_http "api" "http://frontend/nginx-health" 30 2; then
        err "Frontend failed health checks"
        exit 1
    fi

    # Indexer startup can lag while establishing upstream connections.
    # Use container health with a longer window to reduce false warnings.
    if ! wait_for_container_health "indexer" 180 2; then
        warn "Indexer did not become healthy in time"
    fi

    if ! run_smoke_tests; then
        err "Smoke tests failed"
        exit 1
    fi

    log ""
    log "=== Deployment Complete ==="
}

# Best-effort rollback (service restart using currently available images)
cmd_rollback() {
    assert_prerequisites

    log "=== Rollback ==="
    warn "This is a best-effort restart, not a guaranteed image rollback."

    compose up -d --force-recreate api indexer frontend

    log "Rollback command completed"
}

# Main
case "${1:-}" in
    --redeploy)
        cmd_deploy
        ;;
    --status)
        cmd_status
        ;;
    --rollback)
        cmd_rollback
        ;;
    --help|-h)
        echo "Usage: $0 [--redeploy|--status|--rollback|--help]"
        echo ""
        echo "  (no args)    Build, migrate, recreate, and verify services"
        echo "  --redeploy   Explicit redeploy command (same as no args)"
        echo "  --status     Show current deployment status"
        echo "  --rollback   Best-effort service restart"
        echo "  --help       Show this help"
        ;;
    *)
        cmd_deploy
        ;;
esac
