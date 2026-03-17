#!/usr/bin/env bash
# bootstrap.sh — Idempotent developer environment setup for Subcults.
# Safe to re-run; skips completed steps.
#
# Usage:
#   ./scripts/bootstrap.sh          # Full setup
#   ./scripts/bootstrap.sh --check  # Verify environment only (no changes)

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
CHECK_ONLY=false

if [[ "${1:-}" == "--check" ]]; then
  CHECK_ONLY=true
fi

ok()   { printf "${GREEN}✓${NC} %s\n" "$1"; }
warn() { printf "${YELLOW}!${NC} %s\n" "$1"; }
fail() { printf "${RED}✗${NC} %s\n" "$1"; }
info() { printf "${CYAN}→${NC} %s\n" "$1"; }

errors=0

# ─── Step 1: Check required tools ───────────────────────────────────────
echo
info "Checking required tools..."

check_tool() {
  local name="$1" min_version="$2" cmd="$3"
  if command -v "$name" &>/dev/null; then
    local version
    version=$(eval "$cmd" 2>/dev/null || echo "unknown")
    ok "$name installed ($version)"
  else
    fail "$name not found (need $min_version+)"
    errors=$((errors + 1))
  fi
}

check_tool "go"     "1.24"  "go version | grep -oP 'go\d+\.\d+(\.\d+)?'"
check_tool "node"   "20"    "node --version"
check_tool "docker" "24"    "docker --version | grep -oP '\d+\.\d+\.\d+'"
check_tool "make"   "any"   "make --version | head -1"

# libvips check (optional but needed for image processing)
if pkg-config --exists vips 2>/dev/null; then
  vips_ver=$(pkg-config --modversion vips 2>/dev/null || echo "unknown")
  ok "libvips installed ($vips_ver)"
else
  warn "libvips not found — image processing features will be limited"
fi

if [[ $errors -gt 0 ]]; then
  echo
  fail "Missing $errors required tool(s). Install them and re-run."
  exit 1
fi

if $CHECK_ONLY; then
  echo
  ok "Environment check passed."
  exit 0
fi

# ─── Step 2: Environment file ───────────────────────────────────────────
echo
info "Setting up environment file..."

ENV_SRC="$REPO_ROOT/configs/dev.env.example"
ENV_DST="$REPO_ROOT/configs/dev.env"

if [[ -f "$ENV_DST" ]]; then
  ok "configs/dev.env already exists (skipped)"
else
  cp "$ENV_SRC" "$ENV_DST"
  ok "Copied configs/dev.env.example → configs/dev.env"
  warn "Edit configs/dev.env and fill in your secrets before proceeding."
  warn "  Required: DATABASE_URL, JWT_SECRET_CURRENT"
  warn "  Optional: LIVEKIT_*, STRIPE_*, R2_*, MAPTILER_API_KEY"
fi

# ─── Step 3: Go modules ────────────────────────────────────────────────
echo
info "Downloading Go modules..."

cd "$REPO_ROOT"
if go mod download 2>/dev/null; then
  ok "Go modules downloaded"
else
  fail "go mod download failed"
  exit 1
fi

# ─── Step 4: Frontend dependencies ──────────────────────────────────────
echo
info "Installing frontend dependencies..."

if [[ -d "$REPO_ROOT/web/node_modules" ]]; then
  ok "web/node_modules exists (run 'cd web && npm ci' to refresh)"
else
  cd "$REPO_ROOT/web"
  if npm ci --silent 2>/dev/null; then
    ok "Frontend dependencies installed"
  else
    fail "npm ci failed in web/"
    exit 1
  fi
  cd "$REPO_ROOT"
fi

# ─── Step 5: Docker services ───────────────────────────────────────────
echo
info "Starting Docker services (Postgres + PostGIS)..."

cd "$REPO_ROOT"
if docker compose ps --format '{{.Name}}' 2>/dev/null | grep -q "subcults-postgres"; then
  ok "Postgres container already running"
else
  if make compose-up 2>/dev/null; then
    # Wait for Postgres to be ready
    info "Waiting for Postgres to accept connections..."
    retries=0
    until docker compose exec -T postgres pg_isready -U subcults &>/dev/null || [[ $retries -ge 30 ]]; do
      sleep 1
      retries=$((retries + 1))
    done
    if [[ $retries -lt 30 ]]; then
      ok "Postgres is ready"
    else
      fail "Postgres did not become ready within 30s"
      exit 1
    fi
  else
    warn "Docker Compose failed — ensure Docker is running"
  fi
fi

# ─── Step 6: Database migrations ────────────────────────────────────────
echo
info "Running database migrations..."

if [[ -z "${DATABASE_URL:-}" ]]; then
  # Try loading from dev.env
  if [[ -f "$ENV_DST" ]]; then
    DB_URL=$(grep -E '^DATABASE_URL=' "$ENV_DST" 2>/dev/null | cut -d'=' -f2- | tr -d '"' || true)
    if [[ -n "$DB_URL" && "$DB_URL" != *"your_"* && "$DB_URL" != *"example"* ]]; then
      export DATABASE_URL="$DB_URL"
    fi
  fi
fi

if [[ -n "${DATABASE_URL:-}" ]]; then
  if make migrate-up 2>/dev/null; then
    ok "Migrations applied"
  else
    warn "Migration failed — check DATABASE_URL in configs/dev.env"
  fi
else
  warn "DATABASE_URL not set — skipping migrations"
  warn "  Set it in configs/dev.env, then run: make migrate-up"
fi

# ─── Step 7: Verification ──────────────────────────────────────────────
echo
info "Verifying setup..."

# Check Go builds
if go build ./cmd/api/... 2>/dev/null; then
  ok "Go API builds successfully"
else
  fail "Go API build failed"
fi

# Check frontend builds
cd "$REPO_ROOT/web"
if npx tsc --noEmit 2>/dev/null; then
  ok "Frontend TypeScript checks pass"
else
  warn "Frontend TypeScript has errors (non-blocking)"
fi
cd "$REPO_ROOT"

# ─── Summary ────────────────────────────────────────────────────────────
echo
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
ok "Bootstrap complete!"
echo
echo "  Next steps:"
echo "    1. Edit configs/dev.env with your secrets"
echo "    2. make dev          — Start API + frontend"
echo "    3. make test         — Run test suite"
echo
echo "  Useful commands:"
echo "    make dev-api         — API only (port 8080)"
echo "    make dev-frontend    — Frontend only (port 5173)"
echo "    make compose-down    — Stop Docker services"
echo
echo "  Docs:"
echo "    docs/DEVELOPER_ONBOARDING.md  — Full setup guide"
echo "    docs/BACKEND_DEVELOPMENT_GUIDE.md"
echo "    docs/TESTING_GUIDE.md"
echo "    CONTRIBUTING.md"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
