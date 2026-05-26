#!/usr/bin/env bash
# detect-layering — scan connect/*.go for direct imports of pgxpool/sqlx/clickhouse
# Usage: bash scripts/detect-layering.sh
# Exit: 0 = clean, 1 = layering violations found

set -euo pipefail
cd "$(dirname "$0")/.."

CONNECT_DIR="backend/internal/connect"
VIOLATIONS=0

echo "=== detect-layering: scanning connect/ handlers for direct DB access ==="

for f in "$CONNECT_DIR"/*.go; do
  [ -f "$f" ] || continue
  [[ "$f" == *_test.go ]] && continue

  name=$(basename "$f")
  reasons=()

  # Check for direct pgxpool import/usage
  if grep -q 'pgxpool\.Pool' "$f" 2>/dev/null; then
    reasons+=("pgxpool.Pool")
  fi

  # Check for direct clickhouse.Conn import/usage
  if grep -q 'clickhouse\.Conn' "$f" 2>/dev/null; then
    reasons+=("clickhouse.Conn")
  fi

  # Check for direct sqlx import/usage
  if grep -qE 'sqlx\.(DB|Tx|Rows)' "$f" 2>/dev/null; then
    reasons+=("sqlx")
  fi

  if [ ${#reasons[@]} -gt 0 ]; then
    echo "  LAYER-VIOLATION: $name — $(IFS=, ; echo "${reasons[*]}")"
    VIOLATIONS=$((VIOLATIONS + 1))
  fi
done

if [ "$VIOLATIONS" -gt 0 ]; then
  echo "FAIL: $VIOLATIONS handler(s) with direct DB access — must route through service layer"
  exit 1
fi

echo "OK: all handlers clean (no direct DB access)"
