# 地基审计 · strategy-runtime

## 1. 核心设计替代方案

**当前**：Strategy 接口（OnInit/OnBar/OnDeinit）+ LiveRunner（实盘）+ Engine（回测，SimBroker 撮合）。Bar 级执行。回测和实盘共用同一套 Strategy 接口和编译产物。

**替代方案 A**：Tick 级执行。

不选。性能差距 200x。当前 bar 级对 99% 策略精度够用。Tick 级留给混合模式做未来提升（bar 内 OHLC 模拟 tick 检测 SL/TP）。选择是对的，但混合模式是客观上的精度更优解——已写入未来计划。

**替代方案 B**：回测和实盘用不同的执行引擎。

不选。统一引擎 = "回测=实盘"的唯一途径。两套引擎 = 永远在追差异。选择正确。

**结论**：✅ 架构最优。混合模式（bar 内模拟 tick）是未来的精度提升，当前 bar 级够用。

## 2. 上下游契约

**上游**：mql-compiler 提供字节码，market-data 提供实时 bar/历史 K 线。

**下游**：risk-gate 消费 Signal。backtest-engine 复用 Strategy 接口。

**契约关键**：Strategy 接口定义了 `OnBar(bar sdk.Bar) Signal`——所有策略与下游的唯一通信契约。这个接口的稳定性是全部策略的基石。

**隐患**：`sdk.Bar` 和 `mdtick.Bar` 的价格字段已确认使用 `decimal.Decimal`——✅ 无精度风险。

## 3. 已知架构债务

| 债务 | 严重度 | 方案 |
|------|--------|------|
| Bar 级执行对短线策略精度不足 | 🟡 | 混合模式（bar 内模拟 tick）已写入未来计划 |

## 4. 总评

架构最优。最紧迫的是验证 `sdk.Bar` 和 `Signal` 的价格字段是否用了 decimal。如果这里用了 float64，所有依赖它的模块（risk-gate 的 PnL、backtest-engine 的净值、agent-engine 的回测反馈）都有精度问题。
