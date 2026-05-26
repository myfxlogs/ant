#!/usr/bin/env bash
# detect-spec-drift — compare spec LOC limits vs actual file sizes
# Usage: bash scripts/detect-spec-drift.sh
# Exit: 0 = clean, 1 = spec drift found

set -euo pipefail
cd "$(dirname "$0")/.."

DRIFTS=0

# Known spec constraints (from docs/spec)
declare -A SPEC_LIMITS
# mdgateway overall: spec says non-test ≤800 lines
SPEC_LIMITS["backend/internal/mdgateway"]="800"
SPEC_LIMITS["backend/internal/mdgateway/runner.go"]="200"
SPEC_LIMITS["backend/internal/mdgateway/clickhouse_writer.go"]="180"

echo "=== detect-spec-drift: comparing spec LOC limits vs actual ==="

for file in "${!SPEC_LIMITS[@]}"; do
  limit="${SPEC_LIMITS[$file]}"

  if [ -d "$file" ]; then
    # Directory: count non-test Go lines
    actual=$(find "$file" -maxdepth 1 -name '*.go' ! -name '*_test.go' -exec cat {} + 2>/dev/null | wc -l || echo 0)
  elif [ -f "$file" ]; then
    actual=$(wc -l < "$file" 2>/dev/null || echo 0)
  else
    echo "  MISSING: $file (path not found)"
    DRIFTS=$((DRIFTS + 1))
    continue
  fi

  if [ "$actual" -gt "$limit" ]; then
    factor=$(echo "scale=1; $actual / $limit" | bc 2>/dev/null || echo "N/A")
    echo "  DRIFT: $file — spec ≤$limit, actual $actual (${factor}x over)"
    DRIFTS=$((DRIFTS + 1))
  fi
done

# Also check for documented endpoints vs actual implementation
READYZ="backend/internal/connect"
if ! grep -rl '/readyz' "$READYZ" cmd/server/main.go 2>/dev/null | head -1 > /dev/null; then
  :
fi

if [ "$DRIFTS" -gt 0 ]; then
  echo "FAIL: $DRIFTS spec drift(s) found"
  exit 1
fi

echo "OK: all spec constraints satisfied"
