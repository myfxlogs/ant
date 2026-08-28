# FIX-2026-08-28-DATA-TRUTH-1 · Reconciliation 收敛方案

> **Status**: 🟦open（设计 SSOT，待业主决策 3 个架构问题后施工）
> **Priority**: P1（正确性——`orders` 表与 broker 双向不一致，reconciliation 只检测不收敛）
> **Author**: Devin CLI
> **Date**: 2026-08-28
> **关联 ADR**: ADR-0013（订单状态机 + 崩溃恢复 + 幂等性）

## 1. 问题陈述

### 1.1 症状（registry 实测）

| 方向 | 实测 | 含义 |
|------|------|------|
| ant 有 broker 无（orphan） | 账号 904d14e6 `orders` 有 9 条未平仓，broker `OpenedOrders` 连查 3 次为空 | ant 侧 9 条幽灵订单 |
| broker 有 ant 无（ghost） | 账号 3038ae9d/40a7655e/fcca3414 DB 0 条却持续收到 broker `positions:1~2` profit 帧 | broker 侧持仓 ant 完全不知道 |
| 假 orphan 噪声 | `reconcileAccount` 每轮 129 条 warn/账户 | 真问题被噪声淹没 |

### 1.2 根因（3 层）

**根因 A：ghost 仅 log.Warn 从不补写**

`reconciliation.go:189-196`：
```go
for ticket := range brokerTickets {
    if _, exists := antTickets[ticket]; !exists {
        r.log.Warn("reconciliation: ghost order (broker has, ant missing)", ...)
        ghosts++  // ← 仅计数，从不 INSERT
    }
}
```

ADR-0013 §2.3 明确要求"broker 有、PG 无 → INSERT（崩溃期间错过的订单）"，但实现只 log.Warn。ghost 订单永远不收敛。

**根因 B：orphan 仅在 SUBMITTED 时修复**

`reconciliation.go:174-177`：
```go
if antState == string(OMSStateSubmitted) && r.svc != nil {
    r.repairOrder(ctx, accountID, ticket, OMSStateFailed)
    repaired++
}
```

orphan 订单（ant 有 broker 无）只在 `state==SUBMITTED` 时标记为 FAILED。如果订单在 ant 侧是 `FILLED`/`WORKING` 等其他非终态，永远不会被修复 → 永久 orphan。

**根因 C：时间窗口不对称导致结构性假 orphan**

`reconciliation.go:125,143-147`：
```go
brokerHistory, err := exec.FetchOrderHistory(ctx, Clk.Now().Add(-24*time.Hour), Clk.Now())  // broker 24h
// ant 侧：
rows, err := r.pg.Query(ctx, `
    SELECT ticket, state FROM orders WHERE mt_account_id = $1::uuid
    UNION ALL
    SELECT ticket, 'CLOSED' FROM trade_records WHERE account_id = $1::uuid
`, accountID)  // ant 全量（无时间下界）
```

ant 侧查 `orders ∪ trade_records` **全量**（无时间下界），broker 侧只查 **24h 窗口**。任何 24h 之前的 ant 订单在 broker 24h 窗口里都"不存在" → 被误判为 orphan。实测每轮 129 条 warn/账户 = 历史订单全部被误报。

### 1.3 影响范围

| 文件 | 行 | 问题 |
|------|----|------|
| `reconciliation.go:125` | broker 24h 窗口 | 根因 C |
| `reconciliation.go:143-147` | ant 全量查询 | 根因 C |
| `reconciliation.go:174-177` | orphan 仅修 SUBMITTED | 根因 B |
| `reconciliation.go:189-196` | ghost 仅 log.Warn | 根因 A |

调用方：
- `reconciliation.go:46` `reconcileAll`（启动时全量）
- `pipeline.go:246` `ReconcileAccount`（OnBrokerInfo 连接/重连时）
- `service_setters.go:20` `SetReconcileTrigger`（TransitionOrderByTicket fallback）

## 2. 需业主决策的 3 个架构问题

### Q1：orphan 比对是否加 24h 下界？

**现状**：ant 全量 vs broker 24h → 结构性假 orphan（129 条/账户/轮）。

**方案 A（推荐）**：ant 侧查询加 `WHERE created_at >= NOW() - INTERVAL '24 hours'`（或 `close_time >= NOW() - INTERVAL '24 hours'` for trade_records），与 broker 窗口对齐。24h 前的 orphan 不再报告。

- ✅ 消除假 orphan 噪声
- ✅ 与 broker 窗口对称
- ⚠️ 24h 前的真 orphan 不再检测（但 24h 足够覆盖崩溃恢复窗口，ADR-0013 §2.4 的 30s 主动对账是实时层）

**方案 B**：ant 全量 + broker 全量（`FetchOrderHistory` 不传时间范围）。但 MT4/MT5 `OrderHistory` RPC 对全量查询可能超时或拒绝（mtapi 限制）。

**方案 C**：ant 全量 + broker 7d 窗口。折中，但仍有不对称。

**Devin CLI 建议**：**方案 A**。对称窗口消除假阳性，24h 足够覆盖崩溃恢复。

### Q2：ghost 是否自动补写入 OMS？

**现状**：ghost 仅 log.Warn，从不 INSERT。

**ADR-0013 §2.3 要求**："broker 有、PG 无 → INSERT（崩溃期间错过的订单）"。

**方案 A（推荐）**：ghost 自动补写入 `orders` 表，state 从 broker `OrderRecord.State` 映射（`brokerOrderStateToOMS`）。同时写入 `trade_records`（如果是已平仓订单）。

- ✅ 符合 ADR-0013
- ✅ ghost 收敛（下次对账该 ticket 不再是 ghost）
- ⚠️ 涉及资金账实——补写的订单会出现在用户持仓列表/历史中
- ⚠️ 需要完整的 OrderRecord 信息（symbol/volume/price/open_time/close_time/profit）。broker `OpenedOrders` + `OrderHistory` 是否返回完整字段需验证。

**方案 B**：ghost 写入 `orders` 表但 state=`RECONCILING`（ADR-0013 状态机有此状态），人工审查后确认。

- ✅ 不直接伪造终态
- ⚠️ 需要人工介入流程（目前无）

**方案 C**：ghost 仅 log.Warn（现状），但升级为 `log.Error` + 告警。

- ✅ 不碰资金账实
- ❌ 不收敛，违反 ADR-0013

**Devin CLI 建议**：**方案 A**，但需业主确认"自动补写 broker 侧订单到 OMS"是否可接受。如果业主担心资金账实风险，退方案 B。

### Q3：reconciliation 定位是"检测器"还是"修复器"？

**现状**：混合——orphan 修 SUBMITTED→FAILED（修复器），ghost 仅 log（检测器）。

**方案 A（推荐）**：reconciliation 是**修复器**（符合 ADR-0013）。ghost 补写、orphan 修复（扩展到所有非终态→对应终态）、状态不一致以 broker 为准。

- ✅ 符合 ADR-0013
- ✅ 收敛（每次对账减少偏差，最终趋于 0）
- ⚠️ 修复器需要更严格的幂等性 + 错误处理

**方案 B**：reconciliation 是**检测器**。只报告偏差，不修复。修复由人工或单独的 repair job 执行。

- ✅ 简单
- ❌ 不收敛
- ❌ 违反 ADR-0013

**Devin CLI 建议**：**方案 A**（修复器），符合 ADR-0013。

## 3. 修复方案（待业主决策后定稿）

### 3.1 前置条件

业主需对 Q1/Q2/Q3 做出决策。以下方案假设 **Q1=A / Q2=A / Q3=A**（Devin CLI 建议）。

### 3.2 S1：ant 侧查询加 24h 下界（根因 C）

**文件**: `reconciliation.go:143-147`

```go
// before
rows, err := r.pg.Query(ctx, `
    SELECT ticket, state FROM orders WHERE mt_account_id = $1::uuid
    UNION ALL
    SELECT ticket, 'CLOSED' FROM trade_records WHERE account_id = $1::uuid
`, accountID)

// after
cutoff := Clk.Now().Add(-24 * time.Hour)
rows, err := r.pg.Query(ctx, `
    SELECT ticket, state FROM orders WHERE mt_account_id = $1::uuid AND created_at >= $2
    UNION ALL
    SELECT ticket, 'CLOSED' FROM trade_records WHERE account_id = $1::uuid AND close_time >= $2
`, accountID, cutoff)
```

### 3.3 S2：ghost 自动补写（根因 A）

**文件**: `reconciliation.go:189-196`

```go
// before
for ticket := range brokerTickets {
    if _, exists := antTickets[ticket]; !exists {
        r.log.Warn("reconciliation: ghost order (broker has, ant missing)", ...)
        ghosts++
    }
}

// after
for ticket, br := range brokerTickets {
    if _, exists := antTickets[ticket]; !exists {
        if r.svc != nil {
            if err := r.svc.ImportBrokerOrder(ctx, accountID, br); err != nil {
                r.log.Error("reconciliation: ghost import failed", ...)
            } else {
                repaired++
            }
        }
        ghosts++
    }
}
```

**新增方法**: `MtHubService.ImportBrokerOrder(ctx, accountID, *OrderRecord) error`——将 broker OrderRecord 写入 `orders` 表（state 从 `brokerOrderStateToOMS` 映射），如果是已平仓订单同时写入 `trade_records`。

### 3.4 S3：orphan 修复扩展到所有非终态（根因 B）

**文件**: `reconciliation.go:174-177`

```go
// before
if antState == string(OMSStateSubmitted) && r.svc != nil {
    r.repairOrder(ctx, accountID, ticket, OMSStateFailed)
    repaired++
}

// after
if isNonTerminal(antState) && r.svc != nil {
    to := OMSStateFailed  // broker 确认不存在 → ant 侧标记失败
    r.repairOrder(ctx, accountID, ticket, to)
    repaired++
}
```

**新增 helper**: `isNonTerminal(state string) bool`——返回 true 如果 state 不是 `FILLED`/`CANCELLED`/`FAILED`/`EXPIRED` 等终态。

### 3.5 S4：对抗证明

**T1**: `TestReconciliation_AntQueryHasTimeBound`——断言 SQL 含 `created_at >=` / `close_time >=`。
**T2**: `TestReconciliation_GhostAutoImports`——mock broker 返回 ghost ticket → 断言 `ImportBrokerOrder` 被调用。
**T3**: `TestReconciliation_OrphanRepairsAllNonTerminal`——orphan state=`WORKING` → 断言 `repairOrder` 被调用（不再只修 SUBMITTED）。
**T4**: `TestReconciliation_TimeWindowSymmetric`——断言 ant cutoff 与 broker cutoff 一致（24h）。

### 3.6 S5：回填历史 ghost（一次性数据修复）

部署后需执行一次性 SQL/脚本，将当前存在的 ghost 订单补写入 OMS。具体脚本待 S2 实现后根据 `ImportBrokerOrder` 逻辑编写。

## 4. 风险

| 风险 | 严重度 | 缓解 |
|------|--------|------|
| ghost 补写涉及资金账实 | 🟡 中 | Q2 业主决策；方案 B 可降级为 RECONCILING 状态 |
| broker OrderRecord 字段不完整 | 🟡 中 | S2 前验证 `FetchOpenedOrders`/`FetchOrderHistory` 返回字段 |
| 24h 前的真 orphan 不再检测 | 🟢 低 | ADR-0013 的 30s 主动对账是实时层，24h 足够覆盖崩溃恢复 |
| 补写订单与 OnOrderUpdate 流竞争 | 🟡 中 | ImportBrokerOrder 用 INSERT ON CONFLICT DO NOTHING（幂等） |

## 5. 不做

- 不改 ADR-0013（如果业主选 Q2/Q3 非 A 方案，则更新 ADR-0013）
- 不改 reconciliation 的事件驱动架构（ADR-0013 已定）
- 不改 `FetchOpenedOrders`/`FetchOrderHistory` 的 RPC 调用
- 不部署（施工完成后停手等 Devin CLI 复审）

## 6. 验收标准

- 机检五件套全绿
- 对抗证明 T1-T4 RED→restore→GREEN
- 部署后实测：reconciliation warn 数从 129/账户/轮降至 <5（真实偏差）
- ghost 订单在下次对账中收敛（不再报 ghost）

## 7. 待业主决策清单

| # | 问题 | Devin CLI 建议 | 业主决策 |
|---|------|---------------|----------|
| Q1 | orphan 比对加 24h 下界？ | A（加下界） | ⬜ |
| Q2 | ghost 自动补写入 OMS？ | A（自动补写） | ⬜ |
| Q3 | reconciliation 是检测器还是修复器？ | A（修复器） | ⬜ |
