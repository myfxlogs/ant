#!/usr/bin/env bash
set -euo pipefail

# Usage:
#   scripts/risk_soak_snapshot.sh [hours]
# Example:
#   scripts/risk_soak_snapshot.sh 24

HOURS="${1:-24}"
if ! [[ "$HOURS" =~ ^[0-9]+$ ]]; then
  echo "hours must be integer, got: $HOURS" >&2
  exit 1
fi

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
OUT_DIR="$ROOT_DIR/docs/reports"
mkdir -p "$OUT_DIR"

TS="$(date -u +"%Y%m%dT%H%M%SZ")"
OUT_FILE="$OUT_DIR/risk-soak-snapshot-${TS}.md"

SQL_COMMON_FILTER="module = 'trading_risk' AND action = 'pre_trade_validate' AND created_at >= NOW() - INTERVAL '${HOURS} hours'"

run_sql() {
  local sql="$1"
  docker compose exec -T postgres psql -U antuser -d antrader -At -F $'\t' -c "$sql"
}

TOTAL="$(run_sql "SELECT COUNT(*) FROM system_operation_logs WHERE ${SQL_COMMON_FILTER};")"
PASS="$(run_sql "SELECT COUNT(*) FROM system_operation_logs WHERE ${SQL_COMMON_FILTER} AND COALESCE(new_value->>'result','')='pass';")"
REJECT="$(run_sql "SELECT COUNT(*) FROM system_operation_logs WHERE ${SQL_COMMON_FILTER} AND COALESCE(new_value->>'result','')='reject';")"
FAILED="$(run_sql "SELECT COUNT(*) FROM system_operation_logs WHERE ${SQL_COMMON_FILTER} AND status='failed';")"
ERR_CNT="$(run_sql "SELECT COUNT(*) FROM system_operation_logs WHERE ${SQL_COMMON_FILTER} AND COALESCE(new_value->>'result','')='error';")"

TOP_CODES="$(run_sql "
SELECT COALESCE(new_value->>'risk_code','(none)') AS risk_code, COUNT(*)
FROM system_operation_logs
WHERE ${SQL_COMMON_FILTER}
GROUP BY risk_code
ORDER BY COUNT(*) DESC
LIMIT 10;
")"

TOP_SOURCES="$(run_sql "
SELECT COALESCE(new_value->>'trigger_source','(none)') AS trigger_source, COUNT(*)
FROM system_operation_logs
WHERE ${SQL_COMMON_FILTER}
GROUP BY trigger_source
ORDER BY COUNT(*) DESC;
")"

{
  echo "# Risk Soak Snapshot (${HOURS}h)"
  echo
  echo "- Generated at (UTC): $(date -u +"%Y-%m-%d %H:%M:%S")"
  echo "- Window: last ${HOURS} hours"
  echo
  echo "## Summary"
  echo
  echo "- total pre-trade validations: ${TOTAL}"
  echo "- pass: ${PASS}"
  echo "- reject: ${REJECT}"
  echo "- failed status rows: ${FAILED}"
  echo "- explicit result=error rows: ${ERR_CNT}"
  echo
  echo "## Top Risk Codes"
  echo
  echo "| risk_code | count |"
  echo "|---|---:|"
  if [[ -n "${TOP_CODES}" ]]; then
    while IFS=$'\t' read -r code count; do
      [[ -z "${code}" ]] && continue
      echo "| ${code} | ${count} |"
    done <<< "${TOP_CODES}"
  fi
  echo
  echo "## Trigger Sources"
  echo
  echo "| trigger_source | count |"
  echo "|---|---:|"
  if [[ -n "${TOP_SOURCES}" ]]; then
    while IFS=$'\t' read -r src count; do
      [[ -z "${src}" ]] && continue
      echo "| ${src} | ${count} |"
    done <<< "${TOP_SOURCES}"
  fi
  echo
  echo "## Gate Checklist"
  echo
  echo "- [ ] result=error count == 0"
  echo "- [ ] no unexplained reject spikes"
  echo "- [ ] strategy trigger_source reject ratio within expected bound"
  echo "- [ ] top risk codes align with expected business rules"
} > "$OUT_FILE"

echo "wrote: $OUT_FILE"
