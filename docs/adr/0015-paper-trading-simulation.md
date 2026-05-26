# ADR-0015 · 仿真交易（Paper Trading）

- **状态**：Accepted
- **日期**：2026-05-24
- **决策者**：架构组
- **关联 spec**：`docs/spec/24-paper-trading.md`、`docs/spec/21-backtest-replay.md`、`docs/spec/22-order-state-machine.md`
- **关联 ADR**：ADR-0012、ADR-0013

## 1. 背景

策略在接入真实资金之前，必须在模拟环境中验证。仿真交易（paper trading）使用历史或实时行情数据，模拟订单执行（包含滑点、延迟、部分成交），但不涉及真实资金。

当前 v2 架构中完全没有仿真交易层。策略只能"直接上线"或"不做验证"——两者对零售用户都不可接受。

## 2. 决策

### 2.1 PaperExecutor 实现同一接口

```go
type OrderExecutor interface {
    Place(ctx context.Context, req *PlaceOrderRequest) (*Order, error)
    Cancel(ctx context.Context, accountID string, ticket int64) (*Order, error)
    Modify(ctx context.Context, req *ModifyOrderRequest) (*Order, error)
}
```

- `LiveExecutor`：调用 mtapi OrderSend（实盘）
- `PaperExecutor`：模拟成交，不调用任何 broker

### 2.2 仿真成交模型

| 参数 | 默认值 | 说明 |
|------|--------|------|
| 滑点 | 0.5 pips | 市价单成交价与触发价的偏差 |
| 延迟 | 50ms | 订单确认延迟（模拟网络 RTT） |
| 部分成交概率 | 5% | 仅成交部分手数 |
| 限价单成交 | 价格穿越时 | 当 bid/ask 穿越限价时成交 |

模型简化（M11+ 可增强）：
- 无订单簿深度模拟（不模拟市场冲击）
- 无流动性枯竭模拟

### 2.3 数据隔离

- PG `paper_accounts`：仿真账户（无真实 broker 绑定）
- PG `paper_orders`：仿真订单（与 `orders` 表结构相同，独立存储）
- CH `paper_portfolio_snapshots`：仿真持仓时序

### 2.4 与回测/实盘的关系

```
回测（ReplaySource + PaperExecutor）
  → 验证策略历史表现
  → 指标达标 → 升级为仿真交易

仿真交易（LiveSource + PaperExecutor）
  → 实时行情、模拟成交、真实延迟
  → 7 天稳定 → 一键升级为实盘

实盘（LiveSource + LiveExecutor）
  → 真实资金、真实成交
```

## 3. 备选方案

| 方案 | 否决理由 |
|------|----------|
| 独立仿真交易服务 | 与实盘代码路径分叉；违反 ADR-0012 |
| 不建仿真交易，直接实盘 | 零售用户资金风险不可接受 |
| 使用 MT5 内置模拟账户 | 无法跨 broker 统一管理；无法编程控制成交模型 |

## 4. 后果

- **正面**：策略上线前有完整的验证链条（回测 → 仿真 → 实盘）
- **负面**：PaperExecutor 需维护成交模型，但与 LiveExecutor 共享 OrderExecutor 接口，额外代码 < 200 行
- **中性**：仿真账户和订单存储量随策略数量线性增长

## 5. 实施约束

1. `internal/paper/executor.go`：PaperExecutor 实现
2. `internal/paper/fill_model.go`：滑点/延迟/部分成交模型
3. PG migration：`paper_accounts` + `paper_orders` 表
4. Admin API：创建仿真账户、仿真→实盘升级
5. spec/24 详细规范

## 6. 验证方式

```bash
# 同一策略：回测 P&L vs 仿真 7 天 P&L vs 理论 P&L（CH bar 直接计算）
# 偏差 < 5%（仿真增加滑点和延迟）
go test -tags=integration ./tests/regression/ -run TestPaperParity -v
```
