# Runbook · PgConnectionPoolExhausted

> 关联告警：`PgConnectionPoolExhausted`（`pgxpool_pool_total_conns >= 24`，持续 1m，severity=critical）

## 症状

`pgxpool_pool_total_conns{job="alphaforge-backend"} >= 24` 持续 1 分钟。PG 连接池（max=25，`DB_MAX_CONNS` 环境变量）接近耗尽。新请求阻塞在 `pool.Acquire()`。

## 影响

新请求阻塞 → API 超时 → 级联故障。`/healthz` 的 `pool.Ping` 也阻塞 → 容器被标记 unhealthy → 可能触发 `BackendDown`。DB 是真相源，连接池耗尽影响全站。

## 诊断步骤

```bash
# 1. 确认连接池状态
curl -s http://localhost:8080/metrics | grep -E 'pgxpool_pool_total_conns|pgxpool_pool_acquired_conns'

# 2. 查 PG 端活跃连接
docker exec alphaforge-postgres psql -U ant -d ant -c \
  "SELECT pid, state, wait_event_type, wait_event, query, query_start, state_change FROM pg_stat_activity WHERE datname='ant' ORDER BY query_start"

# 3. 查长事务
docker exec alphaforge-postgres psql -U ant -d ant -c \
  "SELECT pid, age(now(), xact_start) AS xact_duration, query FROM pg_stat_activity WHERE state='active' AND xact_start IS NOT NULL ORDER BY xact_duration DESC"

# 4. 查后端日志中的连接获取失败
docker logs alphaforge-backend --since 10m | grep -iE "pool.*acquire|conn.*exhausted|too.*many.*conn"
```

## 应急处置

1. **长事务占连接** → 终止长事务：
   ```bash
   docker exec alphaforge-postgres psql -U ant -d ant -c \
     "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE state='idle in transaction' AND xact_start < now() - interval '5 minutes'"
   ```
2. **永久 LISTEN 占用过多连接** → 4 个永久 LISTEN holder（`normalizer_invalidator.go:40`、`wiring.go:124`、`strategy_experiment_worker.go:46`、`backtest_worker.go:29`）各占 1 conn。SSE stream 每个再占 1 conn。若并发 SSE 多，连接池被 LISTEN 耗尽。临时：调大 `DB_MAX_CONNS`。
3. **连接泄漏** → 查后端代码中未关闭的 `pool.Acquire` / `tx`：
   ```bash
   docker logs alphaforge-backend --since 30m | grep -iE "conn.*leak|tx.*not.*closed|rows.*not.*closed"
   ```
4. **紧急恢复** → 重启 backend 释放所有连接：
   ```bash
   docker compose restart backend
   ```

## 常见根因

- **永久 LISTEN holder 占用**：4 个永久 PG LISTEN 各占 1 conn（`normalizer_invalidator`、`backfiller`、`strategy_experiment_worker`、`backtest_worker`），加上每个 SSE stream 的 LISTEN，并发高时耗尽池
- **长事务/慢查询**：`pg_stat_activity` 中 `idle in transaction` 状态的连接持有不释放
- **连接泄漏**：代码中 `pool.Acquire` 后未 `Release`，或 `rows.Close()` / `tx.Commit()` 缺失
- **并发暴涨**：大量用户同时操作（如开盘时段），API 并发请求突增
