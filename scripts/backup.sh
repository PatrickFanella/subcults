#!/usr/bin/env bash
set -euo pipefail

# =============================================================================
# Database Backup Script for Subcults
# =============================================================================
# Creates a backup of the Subcults PostgreSQL database.
# For Neon Postgres: Neon provides automatic branching and PITR natively.
# This script is for manual/on-demand backups via pg_dump.
#
# Usage:
#   ./scripts/backup.sh                         # Backup to default location
#   ./scripts/backup.sh /path/to/backup.sql.gz  # Backup to specific file
#
# Prerequisites:
#   - pg_dump available
#   - DATABASE_URL set (or source from deploy/.env)
# =============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BACKUP_DIR="${BACKUP_DIR:-$PROJECT_DIR/backups}"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log()  { echo -e "${GREEN}[backup]${NC} $*"; }
warn() { echo -e "${YELLOW}[backup]${NC} $*"; }
err()  { echo -e "${RED}[backup]${NC} $*" >&2; }

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

# Verify pg_dump is available
if ! command -v pg_dump > /dev/null 2>&1; then
    err "pg_dump not found. Install postgresql-client."
    exit 1
fi

# Determine backup path
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="${1:-$BACKUP_DIR/subcults_${TIMESTAMP}.sql.gz}"

# Create backup directory
mkdir -p "$(dirname "$BACKUP_FILE")"

log "Starting backup..."
log "  Database: ${DATABASE_URL%%@*}@***"
log "  Output:   $BACKUP_FILE"

# Run pg_dump with compression
if pg_dump "$DATABASE_URL" \
    --format=custom \
    --no-owner \
    --no-privileges \
    --verbose \
    2>/dev/null \
    | gzip > "$BACKUP_FILE"; then

    BACKUP_SIZE=$(du -h "$BACKUP_FILE" | cut -f1)
    log "Backup complete: $BACKUP_FILE ($BACKUP_SIZE)"

    # Verify backup integrity
    if gzip -t "$BACKUP_FILE" 2>/dev/null; then
        log "Integrity check: PASSED"
    else
        err "Integrity check: FAILED — backup may be corrupt"
        exit 1
    fi
else
    err "Backup failed"
    rm -f "$BACKUP_FILE"
    exit 1
fi

# Clean up old backups (keep last 30)
if [[ -d "$BACKUP_DIR" ]]; then
    BACKUP_COUNT=$(find "$BACKUP_DIR" -name "subcults_*.sql.gz" -type f | wc -l)
    if [[ "$BACKUP_COUNT" -gt 30 ]]; then
        log "Cleaning old backups (keeping last 30)..."
        find "$BACKUP_DIR" -name "subcults_*.sql.gz" -type f \
            | sort \
            | head -n -30 \
            | xargs rm -f
    fi
fi

log "Done."
