#!/usr/bin/env bash
# Migration script using golang-migrate
# Supports up/down migrations with optional step count
#
# Usage:
#   ./scripts/migrate.sh up          # Apply all pending migrations
#   ./scripts/migrate.sh up 1        # Apply next 1 migration
#   ./scripts/migrate.sh down 1      # Rollback last 1 migration
#   ./scripts/migrate.sh version     # Show current migration version
#   ./scripts/migrate.sh force N     # Force set version (use with caution)
#   ./scripts/migrate.sh drop        # Drop everything in database (use with caution)
#
# Environment Variables:
#   DATABASE_URL - PostgreSQL connection string (required)
#   MIGRATIONS_PATH - Path to migrations directory (default: ./migrations)

set -euo pipefail

# Configuration
MIGRATIONS_PATH="${MIGRATIONS_PATH:-./migrations}"
MIGRATE_IMAGE="migrate/migrate:v4.17.0"
MIGRATE_DOCKER_NETWORK="${MIGRATE_DOCKER_NETWORK:-host}"
MIGRATE_USE_DOCKER="${MIGRATE_USE_DOCKER:-0}"

# Show usage
usage() {
    echo "Usage: $0 <command> [args]"
    echo ""
    echo "Commands:"
    echo "  up [N]        Apply all (or N) pending migrations"
    echo "  down N        Rollback N migrations (N is required)"
    echo "  version       Show current migration version"
    echo "  force N       Force set version to N (use with caution)"
    echo "  drop          Drop everything in the database (use with caution)"
    echo ""
    echo "Examples:"
    echo "  $0 up              # Apply all pending migrations"
    echo "  $0 up 1            # Apply next migration only"
    echo "  $0 down 1          # Rollback the last migration"
    echo "  $0 version         # Show current version"
    echo ""
    echo "Environment Variables:"
    echo "  DATABASE_URL      - PostgreSQL connection string (required)"
    echo "  MIGRATIONS_PATH   - Path to migrations directory (default: ./migrations)"
    echo "  MIGRATE_DOCKER_NETWORK - Docker network for docker fallback (default: host)"
    echo "  MIGRATE_USE_DOCKER - Force docker migration even if local migrate binary exists (0/1)"
    exit 0
}

# Show usage if no args or help requested
if [[ $# -lt 1 ]] || [[ "${1:-}" == "-h" ]] || [[ "${1:-}" == "--help" ]] || [[ "${1:-}" == "help" ]]; then
    usage
fi

# Validate required environment variable
if [[ -z "${DATABASE_URL:-}" ]]; then
    echo "Error: DATABASE_URL environment variable is required" >&2
    echo "Example: export DATABASE_URL='postgres://user:pass@host:5432/dbname?sslmode=disable'" >&2
    exit 1
fi

# Detect if we should use Docker or local binary
use_docker() {
    if [[ "${MIGRATE_USE_DOCKER}" == "1" ]] || [[ "${MIGRATE_USE_DOCKER}" == "true" ]]; then
        if command -v docker &>/dev/null; then
            return 0
        fi
        echo "Error: MIGRATE_USE_DOCKER is set but docker is not available" >&2
        exit 1
    fi

    if command -v migrate &>/dev/null; then
        return 1  # Use local binary
    fi
    if command -v docker &>/dev/null; then
        return 0  # Use Docker
    fi
    echo "Error: Neither 'migrate' binary nor 'docker' is available" >&2
    echo "Install golang-migrate: https://github.com/golang-migrate/migrate" >&2
    echo "Or install Docker: https://docs.docker.com/get-docker/" >&2
    exit 1
}

# Validate step count is a positive integer
validate_step_count() {
    local step="$1"
    if ! [[ "${step}" =~ ^[1-9][0-9]*$ ]]; then
        echo "Error: step count must be a positive integer" >&2
        exit 1
    fi
}

# Validate version number is a non-negative integer
validate_version() {
    local version="$1"
    if ! [[ "${version}" =~ ^[0-9]+$ ]]; then
        echo "Error: version must be a non-negative integer" >&2
        exit 1
    fi
}

# Run migrate command
run_migrate() {
    local args=("$@")

    if use_docker; then
        # Resolve absolute path for migrations directory
        local migrations_abs_path
        if command -v realpath &>/dev/null; then
            migrations_abs_path="$(realpath "${MIGRATIONS_PATH}")"
        else
            if [[ "${MIGRATIONS_PATH}" == /* ]]; then
                migrations_abs_path="${MIGRATIONS_PATH}"
            else
                migrations_abs_path="$(cd "$(dirname "${MIGRATIONS_PATH}")" && pwd)/$(basename "${MIGRATIONS_PATH}")"
            fi
        fi

        # Use Docker with volume mount for migrations.
        # Default network is "host" for localhost DBs; override with
        # MIGRATE_DOCKER_NETWORK for service-name DBs (e.g. subcults-postgres).
        docker run --rm \
            --network "${MIGRATE_DOCKER_NETWORK}" \
            -v "${migrations_abs_path}:/migrations:ro" \
            "${MIGRATE_IMAGE}" \
            -path=/migrations \
            -database "${DATABASE_URL}" \
            "${args[@]}"
    else
        # Use local binary
        migrate \
            -path "${MIGRATIONS_PATH}" \
            -database "${DATABASE_URL}" \
            "${args[@]}"
    fi
}

# Main
main() {
    local command="$1"
    shift

    case "${command}" in
        up)
            if [[ $# -gt 0 ]]; then
                validate_step_count "$1"
                echo "Applying ${1} migration(s)..."
                run_migrate up "$1"
            else
                echo "Applying all pending migrations..."
                run_migrate up
            fi
            echo "Done."
            ;;
        down)
            if [[ $# -lt 1 ]]; then
                echo "Error: 'down' command requires step count (e.g., down 1)" >&2
                exit 1
            fi
            validate_step_count "$1"
            echo "Rolling back ${1} migration(s)..."
            run_migrate down "$1"
            echo "Done."
            ;;
        version)
            echo "Current migration version:"
            run_migrate version
            ;;
        force)
            if [[ $# -lt 1 ]]; then
                echo "Error: 'force' command requires version number" >&2
                exit 1
            fi
            validate_version "$1"
            echo "Forcing version to ${1}..."
            run_migrate force "$1"
            echo "Done."
            ;;
        drop)
            echo "WARNING: This will drop all tables in the database!"
            read -r -p "Are you sure? (yes/no): " confirm
            if [[ "${confirm}" == "yes" ]]; then
                run_migrate drop
                echo "Done."
            else
                echo "Aborted."
                exit 1
            fi
            ;;
        *)
            echo "Error: Unknown command '${command}'" >&2
            echo "Run '$0 --help' for usage information." >&2
            exit 1
            ;;
    esac
}

main "$@"
