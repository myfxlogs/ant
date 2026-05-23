#!/bin/bash
# M6-5: Simple load-test / smoke-bench for ant health endpoints.
# Usage: ./scripts/bench-health.sh [--rps 10] [--duration 30]
# Requires: curl

set -euo pipefail

RPS="${RPS:-10}"
DURATION="${DURATION:-30}"
BASE="${BASE:-http://localhost:8080}"
INTERVAL=$(bc <<< "scale=6; 1/$RPS")

while [[ $# -gt 0 ]]; do
    case $1 in
        --rps) RPS="$2"; shift 2 ;;
        --duration) DURATION="$2"; shift 2 ;;
        *) echo "Unknown arg: $1"; exit 1 ;;
    esac
done

echo "=== ant health benchmark ==="
echo "Target: $BASE"
echo "RPS: $RPS | Duration: ${DURATION}s"
echo ""

TOTAL=200
SUCCESS=0
FAIL=0
declare -a RESPONSE_TIMES

for ((i=1; i<=TOTAL; i++)); do
    START=$(date +%s%N)
    if curl -sf -o /dev/null "$BASE/healthz" 2>/dev/null; then
        ((SUCCESS++))
    else
        ((FAIL++))
    fi
    END=$(date +%s%N)
    ELAPSED=$(bc <<< "scale=3; ($END-$START)/1000000")
    RESPONSE_TIMES+=("$ELAPSED")

    # Respect RPS rate
    SLEEP_TIME=$(bc <<< "scale=6; $INTERVAL - $ELAPSED/1000")
    if (( $(bc <<< "$SLEEP_TIME > 0") )); then
        sleep "$SLEEP_TIME"
    fi

    # Print progress every 10 requests
    if (( i % 10 == 0 )); then
        echo "  [$i/$TOTAL] ok=$SUCCESS fail=$FAIL"
    fi
done

# Calculate p50/p95/p99
IFS=$'\n' sorted=($(printf '%s\n' "${RESPONSE_TIMES[@]}" | sort -n))
p50=${sorted[$((TOTAL*50/100 - 1))]:-0}
p95=${sorted[$((TOTAL*95/100 - 1))]:-0}
p99=${sorted[$((TOTAL*99/100 - 1))]:-0}

echo ""
echo "=== Results ==="
echo "Requests:  $TOTAL"
echo "Success:   $SUCCESS"
echo "Failed:    $FAIL"
echo "P50:       ${p50}ms"
echo "P95:       ${p95}ms"
echo "P99:       ${p99}ms"

if (( SUCCESS < TOTAL * 99 / 100 )); then
    echo "WARNING: success rate < 99%"
    exit 1
fi
echo "Benchmark passed ✅"
