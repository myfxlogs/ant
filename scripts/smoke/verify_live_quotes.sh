#!/usr/bin/env bash
# verify_live_quotes.sh — LIVE-PRICE-4 部署后验收：OnQuote 报价流是否真活。
#
# 用法：LIVE-PRICE-4（删硬编码 symbol + FetchAllSymbols 过滤 + 逐 symbol 订阅）部署后运行。
# 这是审计方的"网"——之前用同样信号证明 OnQuote 已死，现在反过来证它已活。
#
# 判定：
#   1. md_e2e_latency_seconds_count > 0     （HandleTick 真在触发；bug 时 = 0）
#   2. 近期日志无 "subscribe symbols rejected by mtapi" for 有效 symbol（BTCUSDm/XAUUSDm）
#      （bug 时：code 257 "XAUJPYm not exist" 整批失败；修复后：不存在的逐个跳过，存在的订阅成功）
#   3. NATS md.tick/md.bar 消息数在增长      （bug 时：全流 40 条死水）
#   4. md_bars 有近 10 分钟内的新 bar        （bug 时：31h 无新 bar）
#
# 全绿 = LIVE-PRICE-4 真生效 = 实盘报价恢复。任一红 = 未修好。
set -u

BACKEND=alphaforge-backend
NATS=alphaforge-nats
DB=alphaforge-postgres

say() { printf '\n=== %s ===\n' "$1"; }

say "1. md_e2e_latency_seconds_count（HandleTick 是否触发，bug 时=0）"
LAT=$(docker exec "$BACKEND" wget -qO- http://localhost:8080/metrics 2>/dev/null | grep '^md_e2e_latency_seconds_count' | awk '{print $2}')
echo "md_e2e_latency_seconds_count = ${LAT:-N/A}"
if [ "${LAT:-0}" -gt 0 ] 2>/dev/null; then echo "✅ GREEN (>0，HandleTick 在触发)"; else echo "❌ RED (=0，OnQuote 仍无交付)"; fi

say "2. 近期 subscribe 被拒日志（应为空，或仅不存在的 symbol 单独跳过）"
docker logs --since 5m "$BACKEND" 2>&1 | grep -E "subscribe symbols rejected by mtapi|subscribed symbols" | tail -8
echo "--- 判定：若仍有 'rejected ... BTCUSDm/XAUUSDm not exist' = ❌ RED（有效 symbol 仍订阅不上）；仅不存在的 symbol（XAUJPYm 等）被跳过 = ✅"

say "3. NATS md.tick/md.bar 消息数（bug 时全流~40 死水）"
docker exec "$NATS" sh -c 'wget -qO- http://localhost:8222/jsz' 2>/dev/null | tr ',' '\n' | grep -E '"messages"'
echo "--- 判定：messages 数应明显 > 40 且随时间增长 = ✅；仍 ~40 = ❌"

say "4. md_bars 近 10 分钟新 bar（bug 时 31h 无新）"
U=$(docker exec "$DB" printenv POSTGRES_USER); DBNAME=$(docker exec "$DB" printenv POSTGRES_DB)
NOW_MS=$(($(date +%s)*1000))
TEN_AGO_MS=$((NOW_MS - 600000))
docker exec "$DB" psql -U "$U" -d "$DBNAME" -c "SELECT canonical, period, open_ts_unix_ms, tick_count FROM md_bars WHERE open_ts_unix_ms > $TEN_AGO_MS ORDER BY open_ts_unix_ms DESC LIMIT 5;" 2>/dev/null
echo "--- 判定：近 10 分钟有 bar 且 tick_count>1（实时聚合）= ✅；空 = ❌"

say "5. Active Runs 价格列（需 auth，手动在 UI 看 或 跑 watchActive SSE smoke）"
echo "UI 验证：Live Strategy Monitor → Active Runs → BTCUSDm/有效 symbol 的 Price 列应显示 bid/ask 且实时刷新。空/'-' = ❌"

printf '\n验收结论：1-4 全绿 + UI 价格列显示 = LIVE-PRICE-4 生效，实盘报价恢复。\n'
