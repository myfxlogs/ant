#!/bin/bash
# agent-benchmark.sh — Strategy compile + backtest tool chain benchmark.
# Usage: bash scripts/agent-benchmark.sh
#
# Tests the full API pipeline: SubmitStrategy → compile MQL → VM backtest.
# Uses pre-written Python subset strategy snippets.
# Results written to docs/benchmarks/toolchain-quality-YYYY-MM-DD.md.

set -euo pipefail
cd "$(dirname "$0")/.."

BASE="${BASE_URL:-http://localhost:8022}"
OUTPUT="docs/benchmarks/toolchain-quality-$(date +%F).md"
PAYLOAD="/tmp/bench_payload.json"
PASS=0; FAIL=0

# ── Login ──
TOKEN=$(curl -sf -X POST "$BASE/ant.v1.AuthService/Login" \
  -H 'Content-Type: application/json' \
  -d '{"login":"admin@1.com","password":"12345678"}' | \
  python3 -c "import sys,json; print(json.load(sys.stdin).get('accessToken',''))" 2>/dev/null)

if [ -z "$TOKEN" ]; then echo "ERROR: login failed"; exit 1; fi
AUTH="Authorization: Bearer $TOKEN"

echo "# Strategy Tool Chain Benchmark — $(date +%F)" > "$OUTPUT"
echo "" >> "$OUTPUT"
echo "| Strategy | Compile | Trades |" >> "$OUTPUT"
echo "|----------|---------|--------|" >> "$OUTPUT"

run_case() {
  local name="$1"
  echo -n "  $name... "

  local resp
  resp=$(curl -sf -X POST "$BASE/ant.v1.AgentGatewayService/SubmitStrategy" \
    -H "$AUTH" -H 'Content-Type: application/json' \
    -d "@$PAYLOAD" 2>/dev/null)

  local compile_ok trades
  compile_ok=$(echo "$resp" | python3 -c "import sys,json; d=json.load(sys.stdin); print('true' if d.get('compileSuccess') else 'false')" 2>/dev/null || echo "false")
  trades=$(echo "$resp" | python3 -c "import sys,json; d=json.load(sys.stdin); bt=d.get('backtestResult',{}); print(bt.get('totalTrades',0))" 2>/dev/null || echo "0")

  if [ "$compile_ok" = "true" ]; then
    echo "compile=OK trades=$trades"
    echo "| $name | ✅ | $trades |" >> "$OUTPUT"
    ((PASS++)) || true
  else
    echo "compile=FAIL"
    echo "| $name | ❌ | 0 |" >> "$OUTPUT"
    ((FAIL++)) || true
  fi
}

# ── ma_cross ──
python3 -c "
import json; payload={'source_code':'class MyStrategy:\n    price_fast: float = 0.0\n    price_slow: float = 0.0\n    def on_bar(self, ctx) -> None:\n        price_fast = ctx.indicators.ima(\"EURUSD\", \"H1\", 10, 0, 0, \"close\")\n        price_slow = ctx.indicators.ima(\"EURUSD\", \"H1\", 30, 0, 0, \"close\")\n        if price_fast > price_slow and price_slow > 0:\n            if not ctx.positions():\n                ctx.broker.buy(0.1, 0, 0)\n        elif price_fast < price_slow and price_slow > 0:\n            ctx.broker.close_all()\n','language':'python','backtest_config':{'symbol':'EURUSD','timeframe':'H1','initial_capital':'10000','start_date_ms':1700000000000,'end_date_ms':1730000000000}}; json.dump(payload, open('$PAYLOAD','w'))
" && run_case "ma_cross"

# ── rsi_simple ──
python3 -c "
import json; payload={'source_code':'class MyStrategy:\n    rsi_val: float = 0.0\n    def on_bar(self, ctx) -> None:\n        rsi_val = ctx.indicators.rsi(\"EURUSD\", \"H1\", 14, 0)\n        if rsi_val < 30 and rsi_val > 0:\n            if not ctx.positions():\n                ctx.broker.buy(0.1, 0, 0)\n        elif rsi_val > 70:\n            ctx.broker.close_all()\n','language':'python','backtest_config':{'symbol':'EURUSD','timeframe':'H1','initial_capital':'10000','start_date_ms':1700000000000,'end_date_ms':1730000000000}}; json.dump(payload, open('$PAYLOAD','w'))
" && run_case "rsi_simple"

# ── bollinger ──
python3 -c "
import json; payload={'source_code':'class MyStrategy:\n    upper: float = 0.0\n    lower: float = 0.0\n    price: float = 0.0\n    def on_bar(self, ctx) -> None:\n        upper = ctx.indicators.ibands(\"EURUSD\", \"H1\", 20, 2, 0, 0, \"upper\")\n        lower = ctx.indicators.ibands(\"EURUSD\", \"H1\", 20, 2, 0, 0, \"lower\")\n        price = ctx.bars().close(0)\n        if price > upper and upper > 0:\n            if not ctx.positions():\n                ctx.broker.buy(0.1, lower, upper)\n        elif price < lower and lower > 0:\n            ctx.broker.close_all()\n','language':'python','backtest_config':{'symbol':'EURUSD','timeframe':'H1','initial_capital':'10000','start_date_ms':1700000000000,'end_date_ms':1730000000000}}; json.dump(payload, open('$PAYLOAD','w'))
" && run_case "bollinger"

# ── adx_trend ──
python3 -c "
import json; payload={'source_code':'class MyStrategy:\n    adx_val: float = 0.0\n    ema_fast: float = 0.0\n    ema_slow: float = 0.0\n    def on_bar(self, ctx) -> None:\n        adx_val = ctx.indicators.adx(\"EURUSD\", \"H1\", 14, 0)\n        ema_fast = ctx.indicators.ima(\"EURUSD\", \"H1\", 5, 0, 0, \"close\")\n        ema_slow = ctx.indicators.ima(\"EURUSD\", \"H1\", 20, 0, 0, \"close\")\n        if adx_val > 25 and ema_fast > ema_slow and ema_slow > 0:\n            if not ctx.positions():\n                ctx.broker.buy(0.1, 0, 0)\n        elif ema_fast < ema_slow or adx_val < 20:\n            ctx.broker.close_all()\n','language':'python','backtest_config':{'symbol':'EURUSD','timeframe':'H1','initial_capital':'10000','start_date_ms':1700000000000,'end_date_ms':1730000000000}}; json.dump(payload, open('$PAYLOAD','w'))
" && run_case "adx_trend"

# ── syntax_error ──
python3 -c "
import json; payload={'source_code':'class MyStrategy\n    def on_bar(ctx)\n        if ctx.bars().close(0) > 0\n            ctx.broker.buy(0.1)\n','language':'python','backtest_config':{'symbol':'EURUSD','timeframe':'H1','initial_capital':'10000','start_date_ms':1700000000000,'end_date_ms':1730000000000}}; json.dump(payload, open('$PAYLOAD','w'))
" && run_case "syntax_error"

echo "" >> "$OUTPUT"
echo "## Results" >> "$OUTPUT"
echo "PASS=$PASS FAIL=$FAIL" >> "$OUTPUT"

echo ""
echo "=== Benchmark complete ==="
echo "PASS=$PASS FAIL=$FAIL"
echo "Results: $OUTPUT"
rm -f "$PAYLOAD"
