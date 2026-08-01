#!/usr/bin/env bash
# check_coverage_per_block.sh — Per-block coverage gate (rd.md B.3.2 baseline).
#
# Enforces per-package coverage minimums from rd.md B.3.2 snapshot.
# Thresholds only go up (ratchet), never down (B.1.7).
#
# Usage: bash scripts/check_coverage_per_block.sh [backend-dir]
#   backend-dir defaults to ./backend

set -euo pipefail

BACKEND_DIR="${1:-./backend}"

# B.3.2 baseline snapshot (2026-08-01) — per-block floor.
# Format: package_path:minimum_coverage_percent
declare -A BASELINES=(
  ["alphaforge/internal/risk"]=83.7
  ["alphaforge/internal/risksvc"]=70.8
  ["alphaforge/internal/mthub"]=21.1
  ["alphaforge/internal/oms"]=58.2
  ["alphaforge/internal/mdgateway"]=45.2
  ["alphaforge/internal/marketplace"]=0.1
  ["alphaforge/internal/agent"]=5.7
)

cd "$BACKEND_DIR"

echo "Running go test -short -coverprofile..."
COV_FILE=$(mktemp)
go test -short -count=1 -coverprofile="$COV_FILE" -covermode=atomic ./... > /dev/null 2>&1

errors=0
total_checked=0

for pkg in "${!BASELINES[@]}"; do
  baseline="${BASELINES[$pkg]}"
  total_checked=$((total_checked + 1))

  # Calculate per-package coverage from the coverprofile
  # go tool cover -func outputs lines like: pkg/file.go:line: func  XX.X%
  # We average all functions in the package
  current=$(go tool cover -func="$COV_FILE" 2>/dev/null | grep "^${pkg}/" | awk '{print $NF}' | sed 's/%//' | awk '{sum+=$1; n++} END {if(n>0) print sum/n; else print 0}')

  if [ -z "$current" ]; then
    current=0
  fi

  # Compare with 1 decimal precision tolerance
  below=$(awk "BEGIN {print ($current < $baseline - 0.5) ? 1 : 0}")

  if [ "$below" -eq 1 ]; then
    echo "REGRESSION: $pkg — current ${current}% < baseline ${baseline}%"
    errors=$((errors + 1))
  else
    echo "OK: $pkg — ${current}% (baseline ${baseline}%)"
  fi
done

rm -f "$COV_FILE"

echo ""
echo "=== Per-Block Coverage Gate ==="
echo "Packages checked: $total_checked"
echo "Regressions:      $errors"
echo ""

if [ "$errors" -gt 0 ]; then
  echo "FAIL: $errors package(s) below B.3.2 baseline"
  exit 1
fi

echo "PASS: All packages at or above B.3.2 baseline"
exit 0
