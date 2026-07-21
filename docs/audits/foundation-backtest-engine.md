# 地基审计 · backtest-engine

## 1. 核心设计替代方案

**当前**：SimBroker 在 OHLC bar 上撮合。逐 bar 遍历，检查挂单/止损/止盈触发。滑点/手续费/隔夜利息。多品种支持。统一回测/实盘代码路径。

**替代方案**：用真实 MT Strategy Tester。

不选。MT Tester 无法 CI/CD、无法批量回测、无法和 AI 迭代管线集成。自建是对的。

**结论**：✅ 架构最优。SimBroker 的撮合逻辑正确，API 层（`backtest/defaults.go`）提供合理默认值。不需要改。

## 2. 上下游契约

**上游**：market-data 提供历史 K 线，mql-compiler 提供字节码。

**下游**：agent-engine 消费回测结果（`BacktestMetrics`），strategy-marketplace 消费回测结果做质量门槛。

**隐患 A**：SimBroker 的撮合价格是 bar 的收盘价还是高低价范围？如果是收盘价=所有订单在同一价格成交——不真实。已确认：当前在 bar 高低价处检查 SL/TP/挂单触发——这是正确的。

**隐患 B**：手续费/隔夜利息的计算是否和 MT 原生一致？当前使用固定费率，但真实 broker 的手续费因品种和账户类型而异。**SimBroker 的回测结果和 MT Strategy Tester 的回测结果可能不一致**——如果用户发现差异，会质疑平台回测的可信度。

## 3. 已知架构债务

| 债务 | 严重度 | 方案 |
|------|--------|------|
| 手续费和 MT 原生不一致 | 🟡 | 从 mt-gateway 获取真实 broker 手续费配置（利用未来 `SymbolParams` RPC 封装），替代固定值 |
| Bar 级撮合对短线精度不足 | 🟡 | 同 strategy-runtime——混合模式未来提升 |

## 4. 总评

架构最优。唯一值得关注的是 SimBroker 和 MT 原生的手续费差异——影响回测可信度。建议在 `SymbolParams` RPC 封装后（Phase 3），从 broker 获取真实费率替换固定值。
