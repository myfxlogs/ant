#!/usr/bin/env bash
# detect-deadcode — scan backend/internal/ for packages with 0 imports
# Usage: bash scripts/detect-deadcode.sh [--apply]
# Exit: 0 = clean, 1 = dead packages found

set -euo pipefail
cd "$(dirname "$0")/.."

PACKAGES=(
  "internal/controlplane"
  "internal/quantengine"
  "internal/tenant"
  "internal/dataquality"
  "internal/marketstate"
  "internal/qualitygate"
)

HITS=()
for pkg in "${PACKAGES[@]}"; do
  path="backend/${pkg}"
  if [ -d "$path" ]; then
    refs=$(grep -rln "$pkg" backend/cmd backend/internal --include='*.go' 2>/dev/null | grep -v '_test.go' || true)
    if [ -z "$refs" ]; then
      HITS+=("$pkg")
    fi
  fi
done

# Also check for stub_handlers.go in connect/
STUB_FILE="backend/internal/connect/stub_handlers.go"
if [ -f "$STUB_FILE" ]; then
  unimplemented_count=$(grep -c 'CodeUnimplemented' "$STUB_FILE" 2>/dev/null || echo 0)
  if [ "$unimplemented_count" -gt 0 ]; then
    HITS+=("internal/connect/stub_handlers.go ($unimplemented_count CodeUnimplemented)")
  fi
fi

# Check for go.mod dependencies only used by scripts
if grep -rln 'gorm\.io' backend/cmd backend/internal --include='*.go' 2>/dev/null | grep -v '_test.go' | grep -v 'scripts/' > /dev/null 2>&1; then
  : # gorm used in production code — OK
else
  if grep -q 'gorm.io' backend/go.mod 2>/dev/null; then
    HITS+=("gorm.io/gorm (in go.mod, not imported by any production code)")
  fi
fi

if [ ${#HITS[@]} -gt 0 ]; then
  echo "=== detect-deadcode: ${#HITS[@]} dead packages/deps found ==="
  for hit in "${HITS[@]}"; do
    echo "  DEAD: $hit"
  done
  echo "FAIL: ${#HITS[@]} dead code items found"
  exit 1
fi

echo "OK: 0 dead packages found"
