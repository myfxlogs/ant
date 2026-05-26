# ADR-0013 · 订单状态机 + 崩溃恢复 + 幂等性

- **状态**：Accepted
- **日期**：2026-05-24
- **决策者**：架构组
- **关联 spec**：`docs/spec/22-order-state-machine.md`、`docs/spec/12-mthub.md`
- **关联 ADR**：ADR-0005

## 1. 背景

量化交易系统的一个基本事实：**进程随时可能崩溃**。如果在 `OrderSend` 已发出但响应未收到的瞬间崩溃，重启后系统不知道订单是否已提交、是否已成交。

当前 v2 架构中 `mthub.OrderExecutor` 是一个接口定义，没有：
- 订单幂等性（重复 OrderSend 检测）
- 崩溃恢复（重启后订单状态对账）
- 主动对账（定期 polling 验证 PG 状态与 broker 状态一致）

MT5 `OrderState` 有 10 个状态（Started/Placed/Partial/Filled/Rejected/Expired/...），ant 侧没有对应的状态机镜像。

## 2. 决策

### 2.1 ant 侧订单状态机

镜像 MT5 10-state，映射如下：

| MT5 State | ant Internal State | 终态？ |
|-----------|-------------------|--------|
| ORDER_STATE_STARTED | PENDING | 否 |
| ORDER_STATE_PLACED | SUBMITTED | 否 |
| ORDER_STATE_PARTIAL | PARTIALLY_FILLED | 否 |
| ORDER_STATE_FILLED | FILLED | 是 |
| ORDER_STATE_CANCELED | CANCELED | 是 |
| ORDER_STATE_REJECTED | REJECTED | 是 |
| ORDER_STATE_EXPIRED | EXPIRED | 是 |
| ORDER_STATE_REQUEST_ADD | (内部，映射到 PENDING) | 否 |
| ORDER_STATE_REQUEST_MODIFY | MODIFY_PENDING | 否 |
| ORDER_STATE_REQUEST_CANCEL | CANCEL_PENDING | 否 |

### 2.2 幂等性保护

```go
type IdempotencyGuard struct {
    store IdempotencyStore // Redis: key=(account_id, client_id), TTL=24h
}

func (g *IdempotencyGuard) Check(ctx context.Context, accountID, clientID string) (*Order, bool) {
    // 命中 → 返回已存储的 broker_ticket + 当前状态；记录 warning 日志
    // 未命中 → 返回 false，调用方正常执行 OrderSend
}
```

每个 `OrderSendRequest` 必须携带唯一 `ClientID`（UUID，前端/策略引擎生成）。重复 `ClientID` 在 24h 窗口内返回原 ticket。

### 2.3 崩溃恢复

启动时 ReconciliationLoop：

```
1. 对每个已注册的 account:
   a. FetchOpenedOrders(accountID) → broker 侧持仓列表
   b. FetchOrderHistory(accountID, last 5min) → broker 侧最近订单
   c. 与 PG orders 表对比:
      - broker 有、PG 无 → INSERT（崩溃期间错过的订单）
      - PG 有、broker 无 → 标记为 "ghost"（人工审查）
      - 状态不一致 → 以 broker 状态为准更新 PG
2. 对账完成 → 账户状态 = healthy
```

### 2.4 主动对账

每 30s 对所有活跃账户执行 `FetchOpenedOrders` → 对比 PG 中非终态订单。偏差告警：`mthub_state_mismatch_total`。

## 3. 备选方案

| 方案 | 否决理由 |
|------|----------|
| Event sourcing（订单事件日志回放）| MT5 集成场景下 broker 才是权威数据源，事件溯源过度设计 |
| 不建恢复机制，重启后人工核对 | 自动化交易不可接受 |
| 每次 OrderSend 前都查 broker 状态 | 延迟不可接受（+200ms per order） |

## 4. 后果

- **正面**：进程崩溃后 30s 内自动恢复订单状态一致性；重复 OrderSend 不会产生重复订单
- **负面**：mthub 复杂度增加（+~200 行状态机 + 对账逻辑）
- **中性**：Redis 依赖增加（幂等性 key），但 Redis 不可达时降级为无幂等保护（记录 critical 日志）

## 5. 实施约束

1. `internal/mthub/order_state.go`：状态机定义 + 转换规则
2. `internal/mthub/idempotency.go`：幂等性守卫
3. `internal/mthub/reconciliation.go`：对账循环
4. PG `orders` 表新增列：`client_id UUID UNIQUE`、`ant_state VARCHAR(32)`
5. spec/22 详细规范

## 6. 验证方式

```bash
# Chaos test: kill -9 mid-order → restart → verify reconciliation within 30s
go test -tags=chaos ./tests/chaos/ -run TestOrderCrashRecovery -v
```
