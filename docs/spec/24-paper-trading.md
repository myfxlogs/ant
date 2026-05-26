# 24 · 仿真交易（Paper Trading）规范

> **关联 ADR**：ADR-0015
> **关联 spec**：`docs/spec/21-backtest-replay.md`、`docs/spec/22-order-state-machine.md`、`docs/spec/23-risk-management.md`

## 1. 定位

仿真交易是策略从回测到实盘的中间环节。使用**实时行情**（LiveSource），但以**模拟成交**（PaperExecutor）代替真实 broker 下单。

## 2. PaperExecutor

```go
// internal/paper/executor.go

type PaperExecutor struct {
    fillModel  *FillModel
    orderStore *PaperOrderStore
    state      *OrderStateMachine  // 复用 spec/22 的状态机
    nextTicket int64               // 模拟 ticket 生成
}

func (e *PaperExecutor) Place(ctx context.Context, req *PlaceOrderRequest) (*Order, error) {
    // 与 LiveExecutor 完全相同的签名和返回类型
    e.mu.Lock()
    ticket := e.nextTicket
    e.nextTicket++
    e.mu.Unlock()

    order := &Order{
        Ticket:       ticket,
        ClientID:     req.ClientID,
        AccountID:    req.AccountID,
        Symbol:       req.Symbol,
        Volume:       req.Volume,
        OrderType:    req.OrderType,
        OpenPrice:    req.Price,
        State:        antState_SUBMITTED,
        IsPaper:      true,
        CreatedAt:    time.Now(),
    }

    // 模拟 broker 延迟
    time.Sleep(e.fillModel.AckDelay)

    // 模拟成交
    e.fillModel.SimulateFill(order)

    e.orderStore.Save(ctx, order)
    return order, nil
}
```

## 3. 成交模型

```go
// internal/paper/fill_model.go

type FillModel struct {
    SlippagePips     float64       // 默认 0.5 pips
    AckDelay         time.Duration // 默认 50ms
    PartialFillProb  float64       // 默认 0.05 (5%)
    PartialFillRatio float64       // 默认 0.5 (成交一半)
}

func (m *FillModel) SimulateFill(order *Order) {
    // 市价单：当前价格 + 滑点
    if order.IsMarketOrder() {
        slippagePrice := m.applySlippage(order.Symbol, order.Direction)
        order.FillPrice = slippagePrice
    }

    // 限价单和挂单：价格穿越时成交
    if order.IsLimitOrder() {
        // 在 recvLoop 中检查价格是否穿越限价
        // 若穿越 → FillPrice = order.LimitPrice
    }

    // 部分成交
    if rand.Float64() < m.PartialFillProb {
        order.FilledVolume = order.Volume * m.PartialFillRatio
        order.State = antState_PARTIALLY_FILLED
    } else {
        order.FilledVolume = order.Volume
        order.State = antState_FILLED
    }
}
```

## 4. 数据隔离

```sql
CREATE TABLE paper_accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    name VARCHAR(128) NOT NULL,
    initial_balance NUMERIC(20,8) NOT NULL,
    currency VARCHAR(8) DEFAULT 'USD',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE paper_orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    paper_account_id UUID NOT NULL REFERENCES paper_accounts(id),
    client_id UUID UNIQUE,
    ticket BIGINT NOT NULL,
    symbol VARCHAR(32) NOT NULL,
    order_type VARCHAR(16) NOT NULL,
    volume NUMERIC(20,8) NOT NULL,
    filled_volume NUMERIC(20,8) DEFAULT 0,
    open_price NUMERIC(20,8),
    fill_price NUMERIC(20,8),
    ant_state VARCHAR(32) NOT NULL DEFAULT 'PENDING',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE paper_portfolio_snapshots (
    paper_account_id UUID,
    ts_unix_ms Int64,
    balance NUMERIC(20,8),
    equity NUMERIC(20,8),
    realized_pnl NUMERIC(20,8),
    unrealized_pnl NUMERIC(20,8),
    margin_used NUMERIC(20,8)
) ENGINE = MergeTree()
ORDER BY (paper_account_id, ts_unix_ms);
```

## 5. 与实盘的升级路径

```
回测（ReplaySource + PaperExecutor）
  │  验证历史表现
  │
  ▼
仿真交易（LiveSource + PaperExecutor）
  │  实时行情 + 模拟成交 + 真实延迟
  │  至少 7 天稳定运行
  │
  ▼
一键升级为实盘
  │  Admin API: POST /api/paper/{id}/promote
  │  → 复制策略配置到真实 mt_account
  │  → 切换 PaperExecutor → LiveExecutor
  │  → 用户确认（前端弹窗：即将使用真实资金）
```

## 6. 性能指标

仿真交易自动计算并写入 CH：

| 指标 | 说明 |
|------|------|
| 总盈亏 (P&L) | realized + unrealized |
| Sharpe 比率 | (日均收益 / 日收益标准差) * sqrt(252) |
| 最大回撤 | (峰值净值 - 最低净值) / 峰值净值 |
| 胜率 | 盈利交易数 / 总交易数 |
| 盈亏比 | 平均盈利 / 平均亏损 |

## 7. 后端 API（Admin）

| RPC | 说明 |
|-----|------|
| `CreatePaperAccount` | 创建仿真账户（指定初始余额） |
| `ListPaperAccounts` | 列表 |
| `GetPaperAccount` | 详情 + 当前持仓 + 权益曲线 |
| `PromotePaperToLive` | 升级为实盘（需用户二次确认） |

## 8. 配置

```bash
ANT_PAPER_ENABLED=true
ANT_PAPER_DEFAULT_BALANCE=100000   # 默认初始余额
ANT_PAPER_SLIPPAGE_PIPS=0.5
ANT_PAPER_ACK_DELAY_MS=50
ANT_PAPER_PARTIAL_FILL_PROB=0.05
```

## 9. 限制（M11+ 增强）

- 无订单簿深度模拟（不影响无市场冲击的小资金策略）
- 无流动性枯竭模拟
- 无市场冲击模型

## 10. 验收命令

```bash
# 1. PaperExecutor 实现 OrderExecutor 接口
grep -q "OrderExecutor" backend/internal/paper/executor.go

# 2. 仿真数据隔离
docker exec ant-postgres psql -U ant -c "\dt paper_*"

# 3. 回归测试：仿真 7 天 P&L vs 理论 P&L 偏差 < 5%
go test -tags=regression ./tests/regression/ -run TestPaperParity -v
```
