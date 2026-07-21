# risk-gate — 实盘风控 + OMS

> 6 门信号管线 + 11+ 风控规则 + 16 状态机 OMS + 仿真交易。

## 代码位置

```
backend/internal/risk/           ← Gate 主入口、规则定义（MaxLotSize/Drawdown/DailyLoss…）
backend/internal/risksvc/        ← 25 文件：信号管线(Capability→HardLimit→PreCheck→Sizer→BlockAlloc)
backend/internal/oms/            ← 订单状态机(16 状态)、成交模型、PnL 计算、broker adapter
backend/internal/paper/          ← 仿真交易引擎
```

## 关键设计

- 6 门管线串行执行，全部通过才下发订单
- 16 状态机覆盖订单全生命周期（NEW→VALIDATED→RISK_APPROVED→SUBMITTED→WORKING→FILLED/REJECTED/…）
- 风控规则可按用户配置（`rule_user_config.go`）
- Guard（强安全网）+ Canary（比例放行新规则试验）

## 依赖

```
strategy-runtime(信号) → risk-gate
mt-gateway(订单事件) → oms(状态更新)
```

## 被依赖

```
risk-gate → mt-gateway(核准下单)
risk-gate → market-data(NATS 实时 PnL)
```

## 关联文档

- [spec/22-order-state-machine.md](spec/22-order-state-machine.md)
- [spec/23-risk-management.md](spec/23-risk-management.md)
- [spec/24-paper-trading.md](spec/24-paper-trading.md)
- [spec/31-risk-gate.md](spec/31-risk-gate.md)
