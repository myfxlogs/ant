# Runbook · MdGatewayNormalizerFallbackHigh

> 关联告警：`MdGatewayNormalizerFallbackHigh`（fallback rate >5% of tick rate，持续 10m，severity=warning）

## 症状

`rate(md_canonical_fallback_total[10m]) > 0.05 * rate(md_tick_total[10m])` 持续 10 分钟。Normalizer 的 PG LISTEN 缓存失效，退化到 ticker 轮询模式（30s 间隔）。

## 影响

Symbol 解析延迟增大（从缓存命中变为 PG 查询或 raw=canonical fallback）。若 `broker_symbols` 表不完整，raw symbol 直接作为 canonical，可能导致跨 broker 品种映射不一致。

## 诊断步骤

```bash
# 1. 确认 fallback 比率
curl -s http://localhost:8080/metrics | grep -E 'md_canonical_fallback_total|md_tick_total'

# 2. 查 NormalizerInvalidator 日志，确认是 LISTEN 模式还是 ticker fallback
docker logs alphaforge-backend --since 10m | grep -iE "normalizer_invalidator|LISTEN.*lost|ticker.*fallback"

# 3. 检查 PG LISTEN 连接是否存活
docker exec alphaforge-postgres psql -U ant -d ant -c "SELECT pid, query, state FROM pg_stat_activity WHERE query LIKE '%LISTEN%'"

# 4. 检查 broker_symbols 表完整性
docker exec alphaforge-postgres psql -U ant -d ant -c "SELECT count(*) FROM broker_symbols"
```

## 应急处置

1. **LISTEN 连接断开** → `normalizer_invalidator.go:126` 会自动降级到 ticker 模式（30s 轮询）。重启 backend 可恢复 LISTEN 连接：
   ```bash
   docker compose restart backend
   ```
2. **PG 连接池耗尽导致 LISTEN 获取失败** → 参考 `pg-pool-exhausted.md` runbook。`normalizer_invalidator.go:40` 的 `pool.Acquire` 失败会降级到 ticker。
3. **broker_symbols 表不完整** → 补充缺失的 symbol 映射：
   ```bash
   docker exec alphaforge-postgres psql -U ant -d ant -c \
     "INSERT INTO broker_symbols (broker_id, symbol_raw, canonical) VALUES (...) ON CONFLICT DO NOTHING"
   ```
   插入后 PG NOTIFY 自动触发缓存失效。
4. **ticker fallback 模式下运行** → 不影响正确性，仅延迟增大（30s vs 实时）。修复 PG LISTEN 连接即可。

## 常见根因

- **PG LISTEN 连接断开**：`normalizer_invalidator.go:124` 的 `WaitForNotification` 返回错误，自动降级到 `tickerLoop`（30s 轮询）
- **PG 连接池耗尽**：`newPGListener` 中 `pool.Acquire` 失败返回 nil，走 ticker fallback
- **PG 重启**：PG 重启后所有 LISTEN 连接断开，需 backend 重启恢复
- **broker_symbols 表不完整**：新 broker/symbol 未录入映射表，`normalizer.go:78` fallback 到 raw=canonical
