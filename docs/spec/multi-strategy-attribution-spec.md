# 多策略共账户持仓归因施工 Spec（ARCH-4 step⑥）

> **功能块**：strategy-runtime（执行侧归因）+ strategy-marketplace（按策略实盘战绩）+ market-data/oms（trade_records 写路径）
> **关联**：`docs/audits/tech-debt-registry.md` ARCH-4（决策 A/B + ①-⑤ 已落地）、`docs/adr/0029-purchase-to-live-execution.md`、`docs/spec/purchase-to-live-link-spec.md` §八 P1-MKT-1
> **状态**：🏗 待施工（审计方出 spec，施工方实现）。ARCH-4 ①-⑤ 已 ✅ 验收（2026-08-08），本 spec 只覆盖遗留的 step⑥ 归因闭环。
> **日期**：2026-08-08

---

## 1. 背景

ARCH-4「多策略共账户」已实现并发机制（commit `e47ea7bb`，①-⑤）：

- ① `LiveStrategyConfig` + `ActiveSession` 加 `ScheduleID uuid.UUID`
- ② `strategyMagic(scheduleID)`（FNV-1a 截 32 位，`live_helpers.go:29`）+ `submitOrder` 设 `req.Magic`（`live_dispatch.go:329`）
- ③ `dispatchCloseAll` 按 magic 过滤（magic=0 fallback symbol，`live_dispatch.go:152-159`）
- ④ `SessionRegistry.Register` 移除一账户一 session 限制（`session_registry.go:104`）
- ⑤ Gate 不改（account 级聚合，见决策 B）

**结果**：多个策略现在能同账户运行、互不平仓。**但归因闭环没做（step⑥ 缺失）**——审计方验收（2026-08-08）确认 `TradeRecord.ScheduleID` 全仓零赋值。

## 2. 问题（功能空心）

- `orderRecordToTradeRecord`（`mthub_service_orders.go:143` 与 `account_sync_service.go:157` 两份）只设 `MagicNumber = r.Magic`，**从不设 `ScheduleID`**（`.ScheduleID =` 全仓 grep 空）。
- `GetByStrategyID`（`trade_record_repository.go:120-129`）JOIN `trade_records.schedule_id → strategy_schedules.id WHERE ss.template_id = $1`。schedule_id 恒 NULL → **返回空集**。
- 后果：per-strategy 实盘战绩、回测 vs 实盘 divergence 报告（`GetByStrategyID` 是其数据源）、策略市场「实盘战绩公开」**全部拿不到 live 成交**——产品核心价值未交付。

magic 已打到订单、trade_records 也存了 `magic_number`，但**没有人把 magic 闭环回 schedule_id**。

## 3. 目标 / 非目标

**目标**：live 成交单能按策略归属 → `GetByStrategyID(strategyID)` 返回该策略的真实 live trades，divergence 报告与「实盘战绩公开」有数据。

**非目标**：
- 不改 ARCH-4 ①-⑤（并发机制已验）。
- 不改 Gate 风控聚合粒度（决策 B：account 级，见 §4）。
- 不处理 paper 模式成交（paper 走 `paperEngine`，不落 `trade_records`；如未来 paper 也要归因另立项）。
- 不回溯归因 ARCH-4 上线前、magic=0 的历史单（那些是非策略/手动单，归属空正确）。

## 4. 决策回顾（已定，施工方遵守，勿改）

- **决策 A**：允许多策略共账户，采 Magic Number 归因（否则 Pro 档「20 实盘策略」在 5 账户上限下无法交付）。
- **决策 B**：**Gate 风控按 account 级聚合（保持现状，不改）**。Magic **仅用于归因**（P&L 归属 / close_all 隔离 / schedule_id 回填），**不参与风控聚合**。理由：风控真实边界是账户（保证金/净值/强平均按整账户）；改 magic 级会削弱安全。per-magic 软上限是未来可选产品层，非本 spec 范围。

> ⚠️ 文档漂移已修正：旧文档（purchase-to-live spec §八 / ADR-0029 §5 / GLM-master-task-list P1-6b）原写「按策略风控聚合」，与决策 B 冲突，已改为 account 级。

## 5. 设计：magic → schedule_id 归因闭环

### 5.1 核心思路

`magic = strategyMagic(scheduleID)` 是 scheduleID 的确定性函数（已在 ② 落地）。成交单的 `magic_number` 来自 MT（开仓时由我们设的 magic 原样返回）。**只需在写 trade_record 时，按 `(account_id, magic_number)` 反查 schedule_id 回填**，`GetByStrategyID` 即可不变工作。

### 5.2 数据模型

`strategy_schedules` 加一列持久化 magic（O(1) 索引反查，覆盖 backfill / 跨重启 / session 结束后平仓）：

```sql
-- migration 266
ALTER TABLE strategy_schedules ADD COLUMN magic_number INTEGER;
CREATE INDEX idx_strategy_schedules_magic ON strategy_schedules(account_id, magic_number)
  WHERE magic_number IS NOT NULL;
```

**何时填 `magic_number`**：schedule 创建时（拿到生成的 `id` 后）计算 `strategyMagic(id)` 写入。三选一（施工方择优，推荐 a）：
- (a) `strategy_schedules` 创建 handler 里 INSERT 拿到 id → `UPDATE ... SET magic_number = strategyMagic(id)`（最稳，schedule 一存在就有 magic）；
- (b) session 启动（`schedule_event.go` 构造 cfg 处）懒填；
- (c) DB trigger `AFTER INSERT ... UPDATE magic_number`。

> `strategyMagic` 当前在 `internal/connect/strategy/live_helpers.go:29`（`connect/strategy` 包）。填 `magic_number` 的代码若在 `repository`/`service` 层，需把 `strategyMagic` 提到共享位置（如 `internal/strategy/` 或 `model`）或在 SQL 侧用等价 FNV-1a（不推荐，Go 侧更易测）。**保持 `strategyMagic` 单一实现，避免两份漂移。**

### 5.3 解析器（resolver）

新增 repo 方法（加在 `StrategyScheduleRepository` 或 `TradeRecordRepository`，施工方择一）：

```go
// ResolveScheduleIDByMagic maps a live trade's magic number back to its schedule.
// account-scoped: magic collisions across different accounts do not cross-attribute.
// Returns nil for magic=0 (manual/non-strategy trades) or unknown magic.
func (r *XxxRepository) ResolveScheduleIDByMagic(ctx context.Context, accountID uuid.UUID, magic int32) (*uuid.UUID, error)
// SQL: SELECT id FROM strategy_schedules WHERE account_id=$1 AND magic_number=$2 LIMIT 1
```

### 5.4 写路径回填（correctness-critical，两处都要改）

把两份 `orderRecordToTradeRecord` 都接到 resolver。**统一走一个 helper，禁止两份各写一遍**（DRY，否则漂移）：

```go
// 共享 helper（放 mthub 包或 strategy 共享包），两份 orderRecordToTradeRecord 都调：
func resolveSchedule(ctx context.Context, resolver ScheduleResolver, accountID uuid.UUID, magic int32) *uuid.UUID {
    if magic == 0 { return nil } // 手动单 / 非 strategy 单
    sid, err := resolver.ResolveScheduleIDByMagic(ctx, accountID, magic)
    if err != nil { /* log warn, 不阻断成交写入 */ return nil }
    return sid
}
```

- `mthub_service_orders.go:143 orderRecordToTradeRecord` —— 加 `ScheduleID` 入参或注入 resolver，设 `rec.ScheduleID = resolveSchedule(...)`。
- `account_sync_service.go:157 orderRecordToTradeRecord` —— 同上（backfill 路径同样需要归因）。

> `WriteClosedTrade`（`mthub_service_orders.go:116`）路径当前连 `MagicNumber` 都不设；若它接收的 `ClosedTradeParams` 带 magic 则一并接 resolver，否则保持 schedule_id=nil（属可接受的非策略单）。

### 5.5 读取路径

**`GetByStrategyID` 不改**——schedule_id 回填后，现有 JOIN `trade_records.schedule_id → strategy_schedules.id WHERE ss.template_id=$1` 自然工作。

### 5.6 接线（依赖注入）

`orderRecordToTradeRecord` 的两个宿主（`MtHubServer` / account_sync `Service`）需注入 resolver。按现有 `SetXxx` 注入模式（如 `SetTradeRecords`、`SetAccountProvider`）加 `SetScheduleResolver`，在 `cmd/server` wiring 处接上。

## 6. 实现任务（one task = one scope）

| # | 任务 | 锚点 |
|----|------|------|
| 1 | migration 266：`strategy_schedules.magic_number` + 索引 | `backend/migrations/266_*.up.sql` + down |
| 2 | 填 `magic_number`（schedule 创建时，选 §5.2 a）；如跨包则把 `strategyMagic` 提共享 | schedule 创建 handler + `live_helpers.go:29` |
| 3 | 新增 `ResolveScheduleIDByMagic` repo 方法 | `internal/repository/*schedule*.go` |
| 4 | 共享 `resolveSchedule` helper + 两份 `orderRecordToTradeRecord` 回填 `ScheduleID` | `mthub_service_orders.go:143`、`account_sync_service.go:157` |
| 5 | wiring：`SetScheduleResolver` 注入两宿主 | `cmd/server/handlers_strategy*.go` |
| 6 | 回归/集成测试（见 §8） | 新增 test 文件 |

> **复用核对（CLAUDE.md Reuse Preflight）**：`strategyMagic`@`live_helpers.go:29`（REUSE，勿重写）、`orderRecordToTradeRecord` 两份（合并而非新增第三份）、`GetByStrategyID`（不改）。NEW：`ResolveScheduleIDByMagic` + `resolveSchedule` helper + migration 266。施工方动工前跑 `bash scripts/cap.sh <动词>` 二次确认。

## 7. 边界情况

| 场景 | 处理 |
|----|------|
| magic=0（手动单 / ①-⑤ 前历史单 / 未设 ScheduleID 的 StartStrategy 手动启动） | resolver 返回 nil → schedule_id=NULL → 不归属任何策略（**正确**） |
| 同账户两 schedule magic 碰撞（FNV-1a 32 位，同账户 schedule 数 ≤20，概率可忽略） | resolver `LIMIT 1` 可能误归属。**建议**：schedule 创建写 magic_number 时 `UNIQUE(account_id, magic_number)` 约束 + 碰撞则 rehash（追加 account_id 入 hash 输入）或报错。施工方加约束即可，碰撞实际不会发生 |
| 跨账户同 magic | resolver 按 `account_id` scope 查询，**不跨账户误归属**（已在 §5.3 SQL 体现） |
| backfill（`SyncAccountHistory` 拉历史单） | 走 `account_sync_service.go:157` 那份 → 同样接 resolver → 历史 strategy 单可归因（前提：schedule 仍在 `strategy_schedules`，magic_number 已填） |
| schedule 已删除但 trade 仍在 | `strategy_schedules` 行不在 → resolver 返回 nil → schedule_id=NULL。可接受（策略已下架，归属空）；如需保留历史归属，`strategy_schedules` 删除改软删 |
| MT 返回 magic 与我们设的不一致（broker 改写） | 罕见；resolver 查不到 → nil，不阻断成交写入 |

## 8. 验收标准 + 对抗证明

**功能验收**：
1. `GetByStrategyID(strategyA)` 返回 strategyA 的 live trades，**不含** strategyB 的（同账户两策略同 symbol 场景）。
2. divergence 报告（`divergence_handler.go` 用 `GetByStrategyID`）live 侧非空。
3. backfill 后历史 strategy 单也归属正确。

**对抗证明（删关键行测试必红）**：
- 删 §5.4 的 `resolveSchedule` 回填（`rec.ScheduleID = ...`）→ `TestGetByStrategyID_ReturnsStrategyTrades` 必红（返回空）。
- resolver 对 magic=0 返回 nil → 手动单不进策略战绩的测试必绿。
- migration 266 的 `UNIQUE(account_id, magic_number)` → 碰撞插入测试必红。

**测试计划**：
- 单元：`ResolveScheduleIDByMagic`（hit/miss/magic=0/跨账户隔离）、`strategyMagic` 确定性 + 同 schedule 跨调用一致。
- 集成（`//go:build integration`）：同账户两策略同 symbol 各开平仓 → `GetByStrategyID(A)` 只含 A、`(B)` 只含 B；backfill 路径归因；magic=0 单不归属。
- 回归：现有 `trade_record_hash_integration_test.go`（FEAT-4V）仍绿（schedule_id 回填不能破坏 hash chain——`entry_hash` 不含 schedule_id，安全）。

**工程验收（CLAUDE.md 5 维）**：
- 可演进性：resolver 是接口/单一 helper，未来加「magic + comment 双键归因」或 paper 归因只扩一处。
- 克制：不改 `GetByStrategyID`、不改 Gate、不引新表（只加列）。

## 9. 完工回填（施工方必做）

1. `tech-debt-registry.md` ARCH-4：补 step⑥ 真实根因/修复/对抗证明/测试结果，状态 `🟦open → ✅done`（标日期）。**①-⑤ 已验，⑥ 补齐才算整条 ✅**。
2. `handover-audit-plan.md` 变更日志加一行。
3. 普遍 pitfall（若有）沉淀进 `CLAUDE.md`。
4. **不自行宣告完成**——等审计方核对状态 + 实测。`go build` + `go test` + `check-file-lines --strict` 0🔴 为底线。
