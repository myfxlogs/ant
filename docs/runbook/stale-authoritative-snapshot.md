# Runbook · Stale Authoritative Account Snapshot

> 关联 registry：`DATA-TRUTH-10`（`docs/audits/tech-debt-registry.md`）
> 关联 pitfall 段：`AGENTS.md` → `## Broker Snapshot & Stream Pitfalls`

## 症状

`schedule_run_logs` 持续写入：

```text
authoritative account snapshot unavailable or stale: account=<uuid>
```

策略 VM 与 Risk Gate 因缺权威快照而 fail-closed，实盘信号不执行。`account_balance_history` 该账户 `recorded_at` 远滞后于当前时间，`free_margin` 可能停在旧值或 0。

## 影响

**严重**——策略产生信号但无法下单。若持续，用户看到"策略在跑但从不交易"，与回测预期严重偏离。MT4 空账户（有余额无持仓）最易触发，因 `OnOrderProfit` 不推有效金融帧。

## 诊断步骤

```bash
# 1. 确认 stale 是否持续（替换 <uuid>）
docker exec alphaforge-postgres psql -U alphaforge -d alphaforge -c \
  "SELECT created_at, error_message FROM schedule_run_logs
   WHERE error_message LIKE '%authoritative account snapshot%<uuid>%'
   ORDER BY created_at DESC LIMIT 20;"

# 2. 查该账户最新快照是否在续期
docker exec alphaforge-postgres psql -U alphaforge -d alphaforge -c \
  "SELECT account_id, recorded_at, free_margin, margin, equity
   FROM account_balance_history
   WHERE account_id = '<uuid>'
   ORDER BY recorded_at DESC LIMIT 10;"

# 3. 查 broker AccountSummary 是否被周期调用
docker logs alphaforge-backend --since 5m 2>&1 | grep -iE "AccountSummary|profit.*refresh|snapshot.*replay"

# 4. 查 PositionSnapshotBroker 是否 replay latest 给 late subscriber
docker logs alphaforge-backend --since 5m 2>&1 | grep -iE "replayed.*snapshot|late.*subscriber|PositionSnapshotBroker"
```

## 应急处置

1. **`recorded_at` 滞后 > 90s** → AccountSummary 续期链断了：
   - 确认 MT4/MT5 profit stream 是否 active（`docker logs ... | grep -iE "profit.*stream|OnOrderProfit"`）
   - 确认 45s 周期 refresh goroutine 是否在跑（应有间隔 45s 的 `AccountSummary` 调用日志）
   - 临时处置：重启 backend（`docker compose restart backend`）让 gateway 重连并触发首份 AccountSummary
2. **`recorded_at` 正常但策略仍 stale** → PositionSnapshotBroker 未 replay：
   - 策略启动应在订阅后立即收到 latest snapshot，日志应有 "replayed latest snapshot"
   - 若无 → broker retained latest 逻辑回归了，需代码级修复（见 registry DATA-TRUTH-10）
3. **MT4 空账户特有** → `OnOrderProfit` 不推有效帧是已知行为，45s 独立 `AccountSummary` refresh 必须生效。若不生效，查 `profit.go` 的 `fetchAndPublish` 路径是否被误删。
4. **financial-only refresh 清空了持仓** → 查 `PositionsAuthoritative` 字段是否被正确分离，financial-only refresh 不得清空 positions。

## 常见根因

- **PositionSnapshotBroker 不保留 latest**（DATA-TRUTH-10 根因①）：纯瞬时 pub/sub，策略晚订阅错过 gateway 初始快照。已修复为 retained latest + replay。
- **nil-result stream 帧被当 stream 活跃**（DATA-TRUTH-10 根因②）：MT4/MT5 空账户 `OnOrderProfit` 持续发 nil-result heartbeat，旧代码不触发 silence timeout → `AccountSummary` 只在 connect 时调一次 → 90s 后 stale。已修复为 45s 独立 refresh。
- **financial-only refresh 清空持仓**（DATA-TRUTH-10 连带）：周期 `AccountSummary` 只刷新金融字段时误清 positions。已修复为 `PositionsAuthoritative` 字段分离。
- **gateway 重连后 sessionID 空/过期**（LIVE-PRICE-8 连带）：重连无 single-flight 导致 `SubscribeMany` 被拒，profit stream 无法 active → AccountSummary 续期链断。见 `docs/runbook/mthub-session-disconnect.md`。

## 验证修复是否生效

部署修复后，90s 内应观察到：

```bash
# stale 错误新增 = 0
docker exec alphaforge-postgres psql -U alphaforge -d alphaforge -c \
  "SELECT count(*) FROM schedule_run_logs
   WHERE error_message LIKE '%authoritative account snapshot%<uuid>%'
   AND created_at > NOW() - INTERVAL '90 seconds';"
# 期望 0

# account_balance_history 持续续期
docker exec alphaforge-postgres psql -U alphaforge -d alphaforge -c \
  "SELECT count(*) FROM account_balance_history
   WHERE account_id = '<uuid>' AND recorded_at > NOW() - INTERVAL '90 seconds';"
# 期望 >= 1（45s 周期 → 90s 内至少 1 条）
```
