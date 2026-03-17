#!/bin/bash
#
# Audit Log IP Anonymization Script
#
# This script anonymizes IP addresses in audit logs older than 90 days
# for compliance with privacy regulations (GDPR, CCPA).
#
# Usage:
#   ./scripts/anonymize_audit_ips.sh [--dry-run]
#
# Options:
#   --dry-run    Show what would be anonymized without making changes
#
# This script should be run via cron daily or weekly:
#   # Daily at 2 AM
#   0 2 * * * /path/to/subcults/scripts/anonymize_audit_ips.sh >> /var/log/subcults/anonymize.log 2>&1
#

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Parse arguments
DRY_RUN=0
if [[ "${1:-}" == "--dry-run" ]]; then
  DRY_RUN=1
fi

# Load environment if .env file exists
if [[ -f "$PROJECT_ROOT/configs/dev.env" ]]; then
  echo "Loading environment from configs/dev.env"
  set -a
  source "$PROJECT_ROOT/configs/dev.env"
  set +a
fi

# Require DATABASE_URL
if [[ -z "${DATABASE_URL:-}" ]]; then
  echo "ERROR: DATABASE_URL environment variable is required"
  echo "Export DATABASE_URL or create configs/dev.env"
  exit 1
fi

# Calculate cutoff date (90 days ago) - portable for both GNU and BSD date
if date --version >/dev/null 2>&1; then
  # GNU date (Linux)
  CUTOFF_DATE=$(date -u -d '90 days ago' '+%Y-%m-%d %H:%M:%S')
else
  # BSD/macOS date
  CUTOFF_DATE=$(date -u -v-90d '+%Y-%m-%d %H:%M:%S')
fi

echo "========================================="
echo "Audit Log IP Anonymization"
echo "========================================="
echo "Cutoff date: $CUTOFF_DATE UTC"
echo "Dry run: $DRY_RUN"
echo ""

# SQL to anonymize IPv4 addresses (replace last octet with 0)
# and IPv6 addresses (replace last 80 bits with zeros)
# Using parameterized query to prevent SQL injection
ANONYMIZE_SQL="
-- Anonymize IPv4 addresses (replace last octet with 0)
UPDATE audit_logs
SET 
  ip_address = CASE
    WHEN family(ip_address::inet) = 4 THEN 
      regexp_replace(ip_address, '\\.[0-9]+\$', '.0')
    ELSE
      -- IPv6: zero out last 80 bits (keep first 48 bits)
      host(set_masklen(ip_address::inet, 48))
  END,
  ip_anonymized_at = NOW()
WHERE 
  created_at < :'cutoff'::timestamptz
  AND ip_anonymized_at IS NULL
  AND ip_address IS NOT NULL
  AND ip_address != '';
"

if [[ $DRY_RUN -eq 1 ]]; then
  echo "DRY RUN - Would anonymize the following logs:"
  psql "$DATABASE_URL" -v cutoff="$CUTOFF_DATE" -c "
    SELECT 
      id,
      created_at,
      ip_address,
      CASE
        WHEN family(ip_address::inet) = 4 THEN 
          regexp_replace(ip_address, '\\.[0-9]+\$', '.0')
        ELSE
          host(set_masklen(ip_address::inet, 48))
      END as anonymized_ip
    FROM audit_logs
    WHERE 
      created_at < :'cutoff'::timestamptz
      AND ip_anonymized_at IS NULL
      AND ip_address IS NOT NULL
      AND ip_address != ''
    LIMIT 10;
  "
  
  COUNT=$(psql "$DATABASE_URL" -v cutoff="$CUTOFF_DATE" -t -c "
    SELECT COUNT(*)
    FROM audit_logs
    WHERE 
      created_at < :'cutoff'::timestamptz
      AND ip_anonymized_at IS NULL
      AND ip_address IS NOT NULL
      AND ip_address != '';
  " | tr -d ' ')
  
  echo ""
  echo "Total logs to anonymize: $COUNT"
else
  echo "Anonymizing IP addresses..."
  RESULT=$(psql "$DATABASE_URL" -v cutoff="$CUTOFF_DATE" -c "$ANONYMIZE_SQL")
  echo "$RESULT"
  
  echo ""
  echo "✓ IP anonymization completed"
fi

echo ""
echo "========================================="
