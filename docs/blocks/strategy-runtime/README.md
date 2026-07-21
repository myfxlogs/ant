# strategy-runtime — 策略运行时

> Strategy 接口定义 + Bar 重放 + LiveRunner + 技术指标库。

## 代码位置

```
backend/strategy/sdk/          ← Strategy 接口、Signal、Broker、BarSeries、IndicatorSet
backend/strategy/runner/       ← LiveRunner、Context、BarSource（实盘）
backend/strategy/backtest/     ← Engine、SimBroker、撮合、净值曲线（回测）
backend/strategy/indicators/   ← 50+ 技术指标（SMA/EMA/RSI/MACD/Bollinger/Ichimoku…）
```

## 关键设计

- 回测和实盘共用同一套 Strategy 接口和编译产物——"回测=实盘"的保证
- Bar 级执行（非 tick 级）——性能优势 200x，对 99% 策略精度够用
- SimBroker 在 OHLC 价格上撮合，含滑点/手续费/隔夜利息

## 依赖

```
mql-compiler(编译产物) → strategy-runtime(执行)
market-data(实时bar) → strategy-runtime(LiveRunner)
```

## 被依赖

```
strategy-runtime → risk-gate(信号管线) → mt-gateway(下单)
```
