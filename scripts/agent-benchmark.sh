#!/bin/bash
# agent-benchmark.sh — Agent strategy generation quality benchmark.
# Usage: bash scripts/agent-benchmark.sh [email] [password]
#
# Runs 10 strategy generation tasks × 5 iterations each against the production
# Agent Gateway API. Records compile success, backtest completion, and metrics.
# Results written to docs/benchmarks/agent-quality-YYYY-MM-DD.md.
#
# Pre-release checklist item (Gap 1). Not in CI — depends on live LLM.
# Re-run manually after any prompt or tool-chain change.

set -euo pipefail

EMAIL="${1:-admin@1.com}"
PASSWORD="${2:-12345678}"
BASE="${BASE_URL:-http://localhost:8022}"
OUTPUT="docs/benchmarks/agent-quality-$(date +%F).md"

# ── Login ──
TOKEN=$(curl -sf -X POST "$BASE/api/alphaforge.auth.v1.AuthService/Login" \
  -H 'Content-Type: application/json' \
  -d "{\"login\":\"$EMAIL\",\"password\":\"$PASSWORD\"}" | \
  python3 -c "import sys,json; print(json.load(sys.stdin).get('access_token',''))" 2>/dev/null)

if [ -z "$TOKEN" ]; then
  echo "ERROR: login failed for $EMAIL"
  exit 1
fi

AUTH="Authorization: Bearer $TOKEN"
RUNS=5
PASS=0; FAIL=0; COMPILE_OK=0; BT_OK=0; TOTAL_TRADES=0

# ── Test cases ──
declare -A CASES
CASES["simple_ma_cross"]="Write a Python trading strategy using a simple moving average crossover. Buy when 10-period MA crosses above 30-period MA, sell when it crosses below. Include stop-loss at 2%."
CASES["rsi_oversold"]="Create a mean-reversion strategy using RSI(14). Buy when RSI drops below 30, sell when RSI rises above 70. Use position sizing of 1% risk per trade."
CASES["bollinger_breakout"]="Write a breakout strategy using Bollinger Bands(20,2). Buy when price closes above the upper band, sell when it closes below the lower band. Include a volatility filter."
CASES["macd_trend"]="Create a trend-following strategy using MACD(12,26,9). Buy when MACD line crosses above signal line AND price is above 200-period MA. Sell on opposite conditions."
CASES["adx_filter"]="Write a strategy using ADX(14) as a trend filter. Only trade when ADX > 25. Use EMA crossover (5,20) for entry signals. Add ATR(14)-based stop-loss."
CASES["multi_tf"]="Create a multi-timeframe strategy. Use D1 for trend direction (EMA 50 vs 200) and H1 for entry timing (RSI oversold in uptrend, overbought in downtrend)."
CASES["grid_trading"]="Write a grid trading strategy. Place buy orders at every 1% drop and sell orders at every 1% rise from a center price. Maximum 5 levels. Reset grid daily."
CASES["volatility_adaptive"]="Create an adaptive strategy that switches between trend-following (high volatility, ATR > 1.5*avg) and mean-reversion (low volatility). Use 20-period ATR baseline."
CASES["event_driven"]="Write an event-driven strategy. Trade breakouts after news events: if the current bar range (high-low) is 2x the average range of the last 20 bars, enter in the direction of the breakout."
CASES["cross_pair"]="Create a correlated pair strategy. Track the spread between EURUSD and GBPUSD using a 20-period z-score. Buy when z-score < -2, sell when z-score > 2. Target z-score = 0."

echo "# Agent Quality Benchmark — $(date +%F)" > "$OUTPUT"
echo "" >> "$OUTPUT"
echo "| # | Case | Run | Compile | Backtest | Trades | Sharpe | Return |" >> "$OUTPUT"
echo "|---|------|-----|---------|----------|--------|--------|--------|" >> "$OUTPUT"

for case_name in "${!CASES[@]}"; do
  prompt="${CASES[$case_name]}"
  for run in $(seq 1 $RUNS); do
    echo -n "  $case_name run $run/$RUNS... "

    RESP=$(curl -sf -X POST "$BASE/ant.v1.AgentGatewayService/SubmitStrategy" \
      -H "$AUTH" -H 'Content-Type: application/json' \
      -d "{\"source_code\":\"\",\"language\":\"python\",\"prompt\":\"$prompt\",\"backtest_config\":{\"symbol\":\"EURUSD\",\"timeframe\":\"H1\",\"initial_capital\":\"10000\",\"start_date_ms\":1700000000000,\"end_date_ms\":1730000000000}}" 2>/dev/null || echo '{"compile_success":false}')

    COMPILE=$(echo "$RESP" | python3 -c "import sys,json; d=json.load(sys.stdin); print('true' if d.get('compile_success') else 'false')" 2>/dev/null || echo "false")
    TRADES=$(echo "$RESP" | python3 -c "import sys,json; d=json.load(sys.stdin); bt=d.get('backtest_result',{}); print(bt.get('total_trades',0))" 2>/dev/null || echo "0")
    SHARPE=$(echo "$RESP" | python3 -c "import sys,json; d=json.load(sys.stdin); bt=d.get('backtest_result',{}); print(bt.get('sharpe_ratio','N/A'))" 2>/dev/null || echo "N/A")
    RETURN=$(echo "$RESP" | python3 -c "import sys,json; d=json.load(sys.stdin); bt=d.get('backtest_result',{}); print(bt.get('total_return','N/A'))" 2>/dev/null || echo "N/A")
    BT_OK="false"; [ "$TRADES" != "0" ] && BT_OK="true"

    echo "| $((++PASS)) | $case_name | $run | $COMPILE | $BT_OK | $TRADES | $SHARPE | $RETURN |" >> "$OUTPUT"

    [ "$COMPILE" = "true" ] && ((COMPILE_OK++)) || true
    [ "$BT_OK" = "true" ] && ((BT_OK++)) || true
    TOTAL_TRADES=$((TOTAL_TRADES + TRADES))

    echo "$COMPILE (trades=$TRADES)"
  done
done

TOTAL=$((RUNS * ${#CASES[@]}))
echo "" >> "$OUTPUT"
echo "## Summary" >> "$OUTPUT"
echo "" >> "$OUTPUT"
echo "| Metric | Value | Target |" >> "$OUTPUT"
echo "|--------|-------|--------|" >> "$OUTPUT"
echo "| Total runs | $TOTAL | — |" >> "$OUTPUT"
echo "| Compile rate | $COMPILE_OK / $TOTAL ($(awk "BEGIN {printf \"%.0f\", $COMPILE_OK*100/$TOTAL}")%) | ≥ 90% |" >> "$OUTPUT"
echo "| Backtest rate | $BT_OK / $TOTAL ($(awk "BEGIN {printf \"%.0f\", $BT_OK*100/$TOTAL}")%) | ≥ 80% |" >> "$OUTPUT"

echo ""
echo "=== Benchmark complete ==="
echo "Results: $OUTPUT"
echo "Compile: $COMPILE_OK/$TOTAL  Backtest: $BT_OK/$TOTAL"
