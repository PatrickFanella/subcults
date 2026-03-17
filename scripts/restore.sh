#!/usr/bin/env bash
set -euo pipefail

# =============================================================================
# Database Restore Script for Subcults
# =============================================================================
# Restores a Subcults PostgreSQL database from a backup file.
#
# Usage:
#   ./scripts/restore.sh backups/subcults_20260221_120000.sql.gz
#
# CAUTION: This will REPLACE the target database contents.
# Prerequisites:
#   - pg_restore and psql available
#   - DATABASE_URL set (or source from deploy/.env)
# =============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log()  { echo -e "${GREEN}[restore]${NC} $*"; }
warn() { echo -e "${YELLOW}[restore]${NC} $*"; }
err()  { echo -e "${RED}[restore]${NC} $*" >&2; }

# Validate arguments
BACKUP_FILE="${1:-}"
if [[ -z "$BACKUP_FILE" ]]; then
    err "Usage: $0 <backup-file.sql.gz>"
    echo ""
    echo "Available backups:"
    if [[ -d "$PROJECT_DIR/backups" ]]; then
        ls -lh "$PROJECT_DIR/backups/"subcults_*.sql.gz 2>/dev/null || echo "  (none found)"
    fi
    exit 1
fi

if [[ ! -f "$BACKUP_FILE" ]]; then
    err "Backup file not found: $BACKUP_FILE"
    exit 1
fi

# Load DATABASE_URL if not set
if [[ -z "${DATABASE_URL:-}" ]]; then
    if [[ -f "$PROJECT_DIR/deploy/.env" ]]; then
        set -a
        # shellcheck source=/dev/null
        source "$PROJECT_DIR/deploy/.env"
        set +a
    elif [[ -f "$PROJECT_DIR/configs/dev.env" ]]; then
        set -a
        # shellcheck source=/dev/null
        source "$PROJECT_DIR/configs/dev.env"
        set +a
    fi
fi

if [[ -z "${DATABASE_URL:-}" ]]; then
    err "DATABASE_URL not set. Export it or create deploy/.env"
    exit 1
fi

# Safety confirmation
warn "=== DATABASE RESTORE ==="
warn "This will REPLACE the contents of the target database."
warn ""
warn "  Backup:   $BACKUP_FILE"
warn "  Database:  ${DATABASE_URL%%@*}@***"
warn ""
read -r -p "Type 'RESTORE' to confirm: " CONFIRMATION

if [[ "$CONFIRMATION" != "RESTORE" ]]; then
    log "Restore cancelled."
    exit 0
fi

# Verify backup integrity
log "Verifying backup integrity..."
if ! gzip -t "$BACKUP_FILE" 2>/dev/null; then
    err "Backup file is corrupt"
    exit 1
fi
log "Integrity check: PASSED"

# Restore
log "Starting restore..."

if gunzip -c "$BACKUP_FILE" | pg_restore \
    --dbname="$DATABASE_URL" \
    --no-owner \
    --no-privileges \
    --clean \
    --if-exists \
    --verbose \
    2>&1 | tail -5; then

    log "Restore complete."
else
    err "Restore failed — check output above"
    exit 1
fi

# Verify restore
log "Verifying restored data..."
TABLES=$(psql "$DATABASE_URL" -t -c "SELECT count(*) FROM information_schema.tables WHERE table_schema='public';" 2>/dev/null | tr -d ' ')
log "Tables found: $TABLES"

if [[ "$TABLES" -gt 0 ]]; then
    log "Restore verification: PASSED"
else
    err "Restore verification: FAILED — no tables found"
    exit 1
fi

log "Done. Run 'make migrate-up' if any pending migrations need to be applied."
