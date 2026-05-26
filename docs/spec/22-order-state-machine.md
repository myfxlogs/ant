# 22 · 订单状态机 + 崩溃恢复 + 幂等性规范

> **关联 ADR**：ADR-0013
> **关联 spec**：`docs/spec/12-mthub.md`、`docs/spec/24-paper-trading.md`

## 1. ant 侧订单状态机

镜像 MT5 10-state，增加 ant 内部状态：

```
                         ┌──────────┐
                         │ PENDING  │
                         └────┬─────┘
                              │ OrderSend 请求发出
                         ┌────▼─────┐
                         │SUBMITTED │
                         └────┬─────┘
                    ┌─────┬───┴───┬──────┐
                    │     │       │      │
               ┌────▼──┐ ┌▼──┐ ┌──▼──┐ ┌▼──────┐
               │PARTIAL│ │OK │ │REJ. │ │EXPIRED│
               └───┬───┘ └───┘ └─────┘ └───────┘
                   │
              ┌────▼────┐
              │ FILLED   │
              └──────────┘

非终态: PENDING, SUBMITTED, PARTIALLY_FILLED, MODIFY_PENDING, CANCEL_PENDING
终态:   FILLED, CANCELED, REJECTED, EXPIRED
```

## 2. 状态转换规则

| 当前状态 | 允许转换到 | 触发条件 |
|----------|-----------|----------|
| PENDING | SUBMITTED | OrderSend RPC 调用 |
| PENDING | REJECTED | 风控拒绝 / 参数校验失败 |
| SUBMITTED | PARTIALLY_FILLED | broker 回调部分成交 |
| SUBMITTED | FILLED | broker 回调完全成交 |
| SUBMITTED | REJECTED | broker 拒绝 |
| SUBMITTED | CANCELED | 用户取消（需 broker 确认） |
| PARTIALLY_FILLED | FILLED | 剩余部分成交 |
| MODIFY_PENDING | SUBMITTED | 修改完成 |
| CANCEL_PENDING | CANCELED | 取消确认 |

## 3. 幂等性守卫

```go
// internal/mthub/idempotency.go

type IdempotencyGuard struct {
    store IdempotencyStore
}

type IdempotencyStore interface {
    // SetNX: 仅在 key 不存在时写入。返回是否写入成功。
    SetNX(ctx context.Context, key string, value []byte, ttl time.Duration) (bool, error)
    Get(ctx context.Context, key string) ([]byte, error)
}

func (g *IdempotencyGuard) Check(ctx context.Context, accountID, clientID string) (*Order, bool) {
    key := fmt.Sprintf("idem:%s:%s", accountID, clientID)
    data, err := g.store.Get(ctx, key)
    if err != nil {
        return nil, false  // Redis 不可达 → 降级为无保护（记录 CRITICAL 日志）
    }
    if data != nil {
        var order Order
        json.Unmarshal(data, &order)
        return &order, true  // 命中 → 返回已存储订单
    }
    return nil, false  // 未命中 → 正常执行
}
```

Redis TTL: 24h。Key 格式: `idem:{account_id}:{client_id}`。

## 4. 崩溃恢复协议

### 4.1 启动对账

```go
// internal/mthub/reconciliation.go

func (h *Hub) ReconcileOnStartup(ctx context.Context) error {
    for _, accountID := range h.GetRegisteredAccounts() {
        // 1. 拉取 broker 侧当前持仓
        brokerOrders, err := h.executor.FetchOpenedOrders(ctx, accountID)
        if err != nil { /* retry with backoff */ }

        // 2. 拉取 broker 侧最近 5min 历史（捕获崩溃瞬间的成交）
        recentHistory, err := h.executor.FetchOrderHistory(ctx, accountID,
            time.Now().Add(-5*time.Minute), time.Now())
        if err != nil { /* retry */ }

        // 3. 拉取 PG 侧非终态订单
        pgOrders, _ := h.orderRepo.FindNonTerminal(ctx, accountID)

        // 4. 三方对比
        h.reconcile(ctx, accountID, brokerOrders, recentHistory, pgOrders)
    }
    return nil
}
```

### 4.2 对账决策矩阵

| Broker 侧 | PG 侧 | 动作 |
|-----------|-------|------|
| 存在 | 不存在 | INSERT PG（崩溃期间错过的订单） |
| 不存在 | 存在（非终态）| 标记 PG 为 GHOST（人工审查） |
| 存在（state=A）| 存在（state=B, A!=B）| 以 broker state 为准，UPDATE PG |
| 存在 | 存在（一致）| 无动作 |
| 不存在 | 存在（终态）| 无动作（已正常结束） |

## 5. 主动对账

每 30s 对所有活跃账户执行 `FetchOpenedOrders` → 与 PG 非终态订单对比。偏差超过 2 个连续周期 → PagerDuty 告警。

```go
func (h *Hub) ActiveReconciliationLoop(ctx context.Context) {
    ticker := time.NewTicker(30 * time.Second)
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            for _, accountID := range h.GetActiveAccounts() {
                h.reconcileAccount(ctx, accountID)
            }
        }
    }
}
```

## 6. Ghost 订单处理

Ghost 订单（PG 有但 broker 无）的可能原因：
- OrderSend RPC 调用时网络超时，订单实际未到达 broker
- Broker 侧订单被外部操作删除

处理方式：
1. 标记 PG `ant_state = 'GHOST'`
2. metric `mthub_ghost_order_total` 计数
3. 写入 audit log
4. 不自动操作——人工在 admin 面板确认后手动处理

## 7. 指标

| 指标 | 说明 |
|------|------|
| `mthub_state_transition_total{from,to}` | 状态转换计数 |
| `mthub_reconciliation_total{result}` | 对账结果 (ok/mismatch/ghost/missing) |
| `mthub_idempotent_hit_total` | 幂等命中次数 |
| `mthub_ghost_order_total` | Ghost 订单数 |

## 8. PG 表变更

```sql
ALTER TABLE orders ADD COLUMN client_id UUID UNIQUE;
ALTER TABLE orders ADD COLUMN ant_state VARCHAR(32) NOT NULL DEFAULT 'PENDING';
ALTER TABLE orders ADD COLUMN reconciled_at TIMESTAMPTZ;
```

## 9. 验收命令

```bash
# 1. 状态机完整性（10 个状态全部定义）
grep -c "ORDER_STATE_" backend/internal/mthub/order_state.go | grep -q "10"

# 2. 幂等性测试
go test -race ./internal/mthub/ -run TestIdempotency -v

# 3. Chaos 测试：kill -9 → 重启 → 30s 内对账恢复
go test -tags=chaos ./tests/chaos/ -run TestOrderCrashRecovery -v
```
