# Runbook · Equity Curve Data Stale / Missing

> 关联 registry：`DATA-TRUTH-4`（`docs/audits/tech-debt-registry.md`）
> 关联 pitfall 段：`AGENTS.md` → `## Broker Snapshot & Stream Pitfalls` → "双实现写不同表"

## 症状

前端净值曲线/小时净值/月度明细/起始余额派生指标（ReturnPercent）显示为空或停留在旧日期。`account_balance_history` 表 `max(recorded_at)` 远滞后于当前时间。

## 影响

**严重**——净值曲线是策略市场的核心展示数据。断供后用户无法看到策略实盘表现，AI 迭代无数据基础，策略市场战绩页空白。DATA-TRUTH-4 曾导致 28 天静默断供。

## 诊断步骤

```bash
# 1. 查 account_balance_history 是否在续写
docker exec alphaforge-postgres psql -U alphaforge -d alphaforge -c \
  "SELECT account_id, max(recorded_at) AS latest, count(*) AS total
   FROM account_balance_history
   GROUP BY account_id ORDER BY latest DESC LIMIT 20;"

# 2. 查日志是否有 snapshot 写入失败（注意：旧版用 log.Debug 吞，生产 level=info 看不到）
docker logs alphaforge-backend --since 10m 2>&1 | grep -iE "snapshot.*insert.*fail|RecordBalanceSnapshot|account_balance"

# 3. 确认写入目标表存在
docker exec alphaforge-postgres psql -U alphaforge -d alphaforge -c \
  "SELECT to_regclass('account_balance_history') AS history_exists,
          to_regclass('account_balance_snapshots') AS snapshots_exists;"
# 期望：history_exists = 'account_balance_history', snapshots_exists = NULL

# 4. 查清理任务是否操作正确表
docker logs alphaforge-backend --since 24h 2>&1 | grep -iE "cleaned up old balance snapshots"
```

## 应急处置

1. **`account_balance_history` 无新写入** → 查写入方是否指向正确表：
   - `AccountService.RecordBalanceSnapshot` 应 INSERT 进 `account_balance_history`
   - 若写进 `account_balance_snapshots`（不存在）→ 100% 失败，需代码修复（DATA-TRUTH-4）
2. **写入失败被 `log.Debug` 吞** → 生产日志级别 info 看不到 Debug。临时改 `log.Warn` 或调低日志级别取证。
3. **清理任务操作旧表** → `CleanupOldSnapshots` 应 DELETE `account_balance_history`，不是 `account_balance_snapshots`（DATA-TRUTH-8）。
4. **双实现竞争** → 确认只有一个 `RecordBalanceSnapshot` 实现，死代码已删除（DATA-TRUTH-4 根因②）。

## 常见根因

- **写入方指向不存在的表**（DATA-TRUTH-4）：`RecordBalanceSnapshot` INSERT 进 `account_balance_snapshots`（schema 中不存在）→ 100% 失败。已修复为 `account_balance_history`。
- **写入失败被 `log.Debug` 吞**（DATA-TRUTH-4）：生产日志级别 info，Debug 不输出 → 零告警。已修复为 `log.Warn`。
- **双实现竞争**（DATA-TRUTH-4）：两个同名 `RecordBalanceSnapshot`，生产路径漂移到错误实现。已删除死代码。
- **清理任务操作旧表**（DATA-TRUTH-8）：writer 已修但 cleanup 未同步 → 真实历史表无限增长。已修复 cleanup 表名。
- **broker 快照断供**（DATA-TRUTH-10 连带）：AccountSummary 续期链断 → 无新快照可写入。见 `docs/runbook/stale-authoritative-snapshot.md`。

## 验证修复是否生效

部署修复后，应观察到：

```bash
# account_balance_history 持续续写
docker exec alphaforge-postgres psql -U alphaforge -d alphaforge -c \
  "SELECT count(*) FROM account_balance_history
   WHERE recorded_at > NOW() - INTERVAL '5 minutes';"
# 期望 >= 1（有连接的账户应有新快照）

# 无 snapshot insert failed 日志
docker logs alphaforge-backend --since 10m 2>&1 | grep -iE "snapshot.*insert.*fail"
# 期望空
```
