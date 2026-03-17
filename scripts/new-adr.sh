#!/usr/bin/env bash
# new-adr.sh — Create a new Architecture Decision Record from template.
#
# Usage:
#   ./scripts/new-adr.sh "Short title of decision"

set -euo pipefail

ADR_DIR="$(cd "$(dirname "$0")/../docs/adr" && pwd)"
TEMPLATE="$ADR_DIR/adr-template.md"

if [[ $# -lt 1 ]]; then
  echo "Usage: $0 \"Short title of decision\""
  exit 1
fi

TITLE="$1"

# Determine next number from existing ADRs
LAST=$(find "$ADR_DIR" -maxdepth 1 -name '[0-9][0-9][0-9][0-9]-*.md' -printf '%f\n' 2>/dev/null \
  | sort -r | head -1 | grep -oP '^\d+' || echo "0000")
NEXT=$(printf "%04d" $((10#$LAST + 1)))

# Slugify title
SLUG=$(echo "$TITLE" | tr '[:upper:]' '[:lower:]' | tr ' ' '-' | tr -cd 'a-z0-9-')

FILENAME="${NEXT}-${SLUG}.md"
FILEPATH="$ADR_DIR/$FILENAME"

if [[ -f "$FILEPATH" ]]; then
  echo "Error: $FILEPATH already exists"
  exit 1
fi

# Create from template
sed "s/ADR-NNNN/ADR-${NEXT}/;s/Title/${TITLE}/;s/YYYY-MM-DD/$(date +%Y-%m-%d)/" \
  "$TEMPLATE" > "$FILEPATH"

echo "Created: docs/adr/$FILENAME"
echo "Edit the file, then add it to docs/adr/README.md index."
