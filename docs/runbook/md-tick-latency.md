# Runbook · MdGatewayTickLatencyP99High

> 关联告警：`MdGatewayTickLatencyP99High`（`md_e2e_latency_seconds` p99 >500ms，持续 5m，severity=warning）

## 症状

`histogram_quantile(0.99, rate(md_e2e_latency_seconds_bucket[5m])) > 0.5` 持续触发。从 MT tick 到达到 bar 聚合完成的端到端延迟 P99 超过 500ms。违反 SLO-MD-2。

## 影响

策略收到 bar 的延迟增大 → 信号延迟 → 滑点增大。高频策略受影响最大。

## 诊断步骤

```bash
# 1. 查看 e2e 延迟分布
curl -s http://localhost:8080/metrics | grep 'md_e2e_latency_seconds_bucket'

# 2. 查各阶段耗时（Normalizer/Quality/Dedup/Aggregator/PgWriter/Publisher）
docker logs alphaforge-backend --since 10m | grep -iE "normalizer.*took|aggregator.*took|pgwriter.*took|publisher.*took|pipeline.*stage"

# 3. 检查 PG 写入延迟（PgWriter 通常是瓶颈）
curl -s http://localhost:8080/metrics | grep -E 'md_pg_writer_queue_depth|md_pg_write_duration'

# 4. 检查 CPU 负载
docker stats --no-stream alphaforge-backend
```

## 应急处置

1. **PgWriter 队列积压** → PG 写入是瓶颈：
   ```bash
   docker exec alphaforge-postgres psql -U ant -d ant -c "SELECT count(*) FROM pg_stat_activity WHERE state='active'"
   ```
   检查是否有慢查询阻塞写入。临时：重启 backend 释放积压。
2. **Normalizer 慢** → 可能 PG `broker_symbols` 查询慢：
   ```bash
   docker exec alphaforge-postgres psql -U ant -d ant -c "EXPLAIN ANALYZE SELECT canonical FROM broker_symbols WHERE broker='...' AND symbol_raw='...' LIMIT 1"
   ```
   确认有索引：`broker_symbols(broker, symbol_raw)`
3. **CPU 瓶颈** → 减少并发订阅 symbol 数或扩容
4. **单 broker 延迟高** → 该 broker 推送频率过高或 mtapi.io 代理慢

## 常见根因

- **PgWriter 瓶颈**：PG 写入速度跟不上 tick 速率，队列积压导致 e2e 延迟
- **Normalizer PG 查询慢**：`broker_symbols` 表缺少索引或表过大，`normalizer.go:63` 的 PG 查询耗时
- **CPU 饱和**：大量并发 tick 处理 + 策略执行占满 CPU，管线各阶段调度延迟
- **mtapi.io 推送频率突增**：高波动时段 tick 量暴增，管线处理跟不上
