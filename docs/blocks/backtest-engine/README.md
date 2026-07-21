# backtest-engine — 回测引擎

> SimBroker 撮合、滑点/手续费、净值曲线、多品种回测。

## 代码位置

```
backend/internal/backtest/       ← defaults.go（佣金/杠杆/初始资金默认值）
backend/strategy/backtest/       ← 18 文件：Engine、SimBroker、撮合、净值、指标计算
```

## 关键设计

- 逐 bar 遍历 OHLC，在 bar 高低价处检查挂单/止损/止盈触发
- SimBroker 模拟真实 broker 行为：点差、滑点、手续费、隔夜利息
- 统一回测/实盘代码路径——同一份策略编译产物，回测和实盘无差异

## 依赖

```
mql-compiler(Bytecode) → backtest-engine(VM 执行)
market-data(历史K线) → backtest-engine
```

## 被依赖

```
backtest-engine → agent-engine(回测验证 + 反馈)
backtest-engine → strategy-marketplace(策略上架前质量门槛)
```
## 关联文档
- [spec/21-backtest-replay.md](spec/21-backtest-replay.md)
- [plans/real-commission-rates.md](plans/real-commission-rates.md) — 真实手续费替代固定费率（依赖 mt-gateway P1）
