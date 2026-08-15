#!/usr/bin/env bash
# check_coverage_per_block.sh — Per-block coverage gate (rd.md B.3.2 baseline).
#
# Enforces per-package coverage minimums from rd.md B.3.2 snapshot.
# Thresholds only go up (ratchet), never down (B.1.7).
#
# Usage: bash scripts/check_coverage_per_block.sh <coverage.out> [backend-dir]
#   coverage.out — path to go test -coverprofile output
#   backend-dir  — defaults to ./backend (for go tool cover resolution)

set -euo pipefail

COV_FILE="${1:?Usage: $0 <coverage.out> [backend-dir]}"
BACKEND_DIR="${2:-./backend}"

if [ ! -f "$COV_FILE" ]; then
  echo "ERROR: coverage file not found: $COV_FILE" >&2
  exit 1
fi

# B.3.2 baseline snapshot (2026-08-01) — per-block floor.
declare -A BASELINES=(
  ["alphaforge/internal/risk"]=83.7
  ["alphaforge/internal/risksvc"]=79.6
  ["alphaforge/internal/mthub"]=72.0
  ["alphaforge/internal/oms"]=81.8
  ["alphaforge/internal/mdgateway"]=46.7
  ["alphaforge/internal/marketplace"]=3.8
  ["alphaforge/internal/agent"]=20.4
)

cd "$BACKEND_DIR"

errors=0
total_checked=0

for pkg in "${!BASELINES[@]}"; do
  baseline="${BASELINES[$pkg]}"
  total_checked=$((total_checked + 1))

  # Calculate per-package statement coverage from the coverprofile.
  # coverprofile lines: mode/file:line.col,line.col numStmts count
  # A statement is covered if count > 0.
  current=$(awk -v pkg="$pkg" '
    /^[a-z]/ {
      split($1, parts, ":")
      if (index(parts[1], pkg "/") == 1) {
        total += $2
        if ($3 > 0) covered += $2
      }
    }
    END {
      if (total > 0) printf "%.1f", covered * 100.0 / total
      else print 0
    }
  ' "$COV_FILE")

  if [ -z "$current" ]; then
    current=0
  fi

  below=$(awk "BEGIN {print ($current < $baseline - 0.5) ? 1 : 0}")

  if [ "$below" -eq 1 ]; then
    echo "REGRESSION: $pkg — current ${current}% < baseline ${baseline}%"
    errors=$((errors + 1))
  else
    echo "OK: $pkg — ${current}% (baseline ${baseline}%)"
  fi
done

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
