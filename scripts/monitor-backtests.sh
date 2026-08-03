#!/bin/bash
# monitor-backtests.sh — Backtest usage monitoring dashboard.
# Run: bash scripts/monitor-backtests.sh [--watch]
# --watch mode refreshes every 60s.

set -euo pipefail
cd "$(dirname "$0")/.."

run_report() {
  clear 2>/dev/null || true
  echo "=== AlphaForge Backtest Monitor — $(date '+%Y-%m-%d %H:%M:%S') ==="
  echo ""

  # Today's backtests
  echo "▸ Today's activity:"
  docker exec alphaforge-postgres psql -U ant -d ant -t -c "
    SELECT status, count(*)
    FROM backtest_runs
    WHERE created_at >= CURRENT_DATE
    GROUP BY status
    ORDER BY count(*) DESC
  " 2>/dev/null | sed '/^$/d'

  echo ""
  echo "▸ Recent 10 runs:"
  docker exec alphaforge-postgres psql -U ant -d ant -c "
    SELECT u.email, br.symbol, br.timeframe, br.status,
           br.error, br.created_at::timestamp(0)
    FROM backtest_runs br JOIN users u ON br.user_id = u.id
    WHERE br.created_at >= CURRENT_DATE
    ORDER BY br.created_at DESC LIMIT 10
  " 2>/dev/null

  echo ""
  echo "▸ 7-day summary:"
  docker exec alphaforge-postgres psql -U ant -d ant -t -c "
    SELECT status, count(*)
    FROM backtest_runs
    WHERE created_at >= NOW() - INTERVAL '7 days'
    GROUP BY status
    ORDER BY count(*) DESC
  " 2>/dev/null | sed '/^$/d'

  echo ""
  echo "▸ Users who ran backtests (7 days):"
  docker exec alphaforge-postgres psql -U ant -d ant -t -c "
    SELECT u.email, count(*) as runs
    FROM backtest_runs br JOIN users u ON br.user_id = u.id
    WHERE br.created_at >= NOW() - INTERVAL '7 days'
    GROUP BY u.email ORDER BY runs DESC
  " 2>/dev/null | sed '/^$/d'

  echo ""
  echo "▸ Active MT accounts (streaming data):"
  docker exec alphaforge-postgres psql -U ant -d ant -t -c "
    SELECT u.email, ma.login, ma.broker_company, ma.account_status
    FROM mt_accounts ma JOIN users u ON ma.user_id = u.id
    WHERE ma.deleted_at IS NULL AND ma.account_status = 'connected'
    ORDER BY u.email
  " 2>/dev/null | sed '/^$/d'

  echo ""
  echo "▸ Total:"
  docker exec alphaforge-postgres psql -U ant -d ant -t -c "
    SELECT
      COUNT(*) AS total,
      COUNT(*) FILTER (WHERE status = 'SUCCEEDED') AS succeeded,
      COUNT(*) FILTER (WHERE status = 'FAILED') AS failed,
      COUNT(*) FILTER (WHERE status = 'PENDING') AS pending
    FROM backtest_runs
  " 2>/dev/null | sed '/^$/d'

  echo ""
  echo "---"
  echo "Refresh: $(date '+%H:%M:%S') | Next: 60s | Ctrl+C to stop"
}

if [ "${1:-}" = "--watch" ]; then
  while true; do
    run_report
    sleep 60
  done
else
  run_report
fi
