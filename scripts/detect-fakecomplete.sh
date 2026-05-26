#!/usr/bin/env bash
# detect-fakecomplete — scan ROADMAP ☑ cards for missing handover logs
# Usage: bash scripts/detect-fakecomplete.sh
# Exit: 0 = clean, 1 = fake completions found

set -euo pipefail
cd "$(dirname "$0")/.."

ROADMAP="docs/plan/ROADMAP.md"
HANDOVER_DIR="docs/handover"
MIN_LINES=30

MISSING=()

# Extract all ☑ card IDs from ROADMAP
# Card IDs look like: M7.0-1, M10-BASE-A1, M10.5-3, M11-13, etc.
while IFS= read -r card_id; do
  log_file="${HANDOVER_DIR}/verify-${card_id}.log"
  if [ ! -f "$log_file" ]; then
    MISSING+=("$card_id (log missing)")
  else
    lines=$(wc -l < "$log_file" 2>/dev/null || echo 0)
    if [ "$lines" -lt "$MIN_LINES" ]; then
      MISSING+=("$card_id (log only $lines/$MIN_LINES lines)")
    fi
  fi
done < <(grep -oE '[M][0-9]+(\.[0-9Z]+-[0-9]+|-BASE-[A-Z][0-9]+)' "$ROADMAP" | sort -u | while read id; do
  # Check if this card is marked ☑ in its row
  if grep -qP "^\|.*${id}.*\|.*☑" "$ROADMAP" 2>/dev/null; then
    echo "$id"
  fi
done)

if [ ${#MISSING[@]} -gt 0 ]; then
  echo "=== detect-fakecomplete: ${#MISSING[@]} ☑ cards with missing/inadequate handover ==="
  for m in "${MISSING[@]}"; do
    echo "  FAKE: $m"
  done
  echo "FAIL: ${#MISSING[@]} fake completions found"
  exit 1
fi

echo "OK: all ☑ cards have ≥${MIN_LINES}-line handover logs"
