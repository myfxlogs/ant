# ADR-0021 · 策略运行时从 Python 迁移到 Go

- **状态**：Partially superseded by 0023
- **日期**：2026-06-26
- **决策者**：人类负责人
- **关联 ADR**：ADR-0020（被本 ADR 覆盖 D1/D4/D5/D7）、ADR-0012（统一回测/实盘路径）、ADR-0023（覆盖 G1/§3.1）
- ** superseded 注记**：G1（目标语言 = Go Strategy SDK，MQL → Go 源码编译）和 §3.1（代码生成方式：tree-sitter CST → IR → 字符串拼接）已被 ADR-0023 取代。ADR-0023 改为 MQL → AST → Bytecode VM 进程内执行，不再生成 Go 源码用于运行时。G2-G7（Python 退役、风控门、Decimal 一致性、行为对齐 harness）仍然有效。
- **关联文档**：`docs/audit/2026-06-22-策略EA执行系统-现状地图.md`、`docs/audit/2026-06-26-翻译器实施说明书落地审计-v2.md`

## 1. 背景

ADR-0020 确立了"Python Strategy SDK + Go 引擎编排"的架构。经过实际落地和审计（见 v2 审计报告），Python 路线暴露出多个不可修复的架构缺陷：

### 1.1 Python 路线的根本问题

| 问题 | 影响 |
|------|------|
| **安全沙箱不可靠** | RestrictedPython 可绕过、对 C 扩展无效；OS 级隔离（seccomp/断网/只读 FS）运维复杂且不彻底 |
| **跨语言 IPC 开销** | Go 引擎 ↔ Python worker 通过 proto/REST 通信，每 bar 序列化+反序列化+进程间通信，延迟显著 |
| **Decimal/float 精度管理割裂** | Python 用 `decimal.Decimal`，Go 用 `shopspring/decimal`，跨边界需 `str→float64→Decimal` 转换，精度丢失 |
| **IR 语言耦合** | 迁移引擎 IR 的 `Condition.expr` 字段在识别器阶段就被 `ExpressionGen` 填充为 Python 字符串，声称"语言无关"实则 Python 绑定（审计 A1） |
| **Registry 死代码** | `RecognizerRegistry` 注册 11 个识别器但 `Pipeline` 完全绕过（审计 A9） |
| **覆盖率虚高** | `meta`/`execution`/`sizing` 永远 +1，25% 基线分白送（审计 A10） |
| **JSON 违规** | `gap_kb.py` 用 JSON 做持久化，违反 CLAUDE.md 红线（审计 D7） |
| **import API 违规** | 前端直连 Python REST+JSON，违反 CLAUDE.md 红线（审计 A11） |
| **Python 服务维护负担** | `strategy-service/` 是独立的 Python FastAPI 服务，需要独立的依赖管理、部署、监控 |

### 1.2 Go 侧已落地的资产

在 Python 路线遇到上述问题的同时，Go 侧已经建设了完整的策略执行栈：

| 模块 | 路径 | 状态 |
|------|------|------|
| **Go Strategy SDK** | `backend/strategy/sdk/` | `Strategy`/`Context`/`Broker`/`IndicatorSet`/`BarSeries` 接口 + 全类型定义 |
| **Go 回测引擎** | `backend/strategy/backtest/` | `Engine` + `SimBroker`（订单/持仓/SL/TP/佣金/equity 曲线/metrics） |
| **Go 实盘 Runner** | `backend/strategy/runner/` | `Runner`/`LiveRunner` + `contextImpl`/`brokerImpl`/`indicatorSet` |
| **Go MQL 转译器** | `backend/tools/mql2go/` | tree-sitter cgo + CST 识别器 + IR + Go 代码生成器 + CLI |
| **Go 策略执行器** | `backend/internal/connect/strategy/go_runner.go` | `GoStrategyExecutor` |

## 2. 决策

**全面放弃 Python 策略运行时，转为 Go 语言实现。**

| # | 决策 |
|---|---|
| G1 | ~~**目标语言 = Go Strategy SDK**。MQL EA 经 `mql2go` 转译为 Go 源码，编译后与后端同二进制执行。~~ → **Superseded by ADR-0023**: MQL → AST → Bytecode VM 进程内执行，不再生成 Go 源码用于运行时。`mql2go` 的 Go 代码生成仅保留用于 CLI 开发调试。 |
| G2 | **`strategy-service/` Python 服务退役**。所有策略执行（回测+实盘）由 Go 后端直接处理。 |
| G3 | **`tools/mql_transpiler/` + `tools/mql_migration/` 退役**。MQL→Go 转译由 `backend/tools/mql2go/` 替代。 |
| G4 | **安全模型简化**：Go 编译产物无 RCE 风险，无需 OS 级沙箱（seccomp/断网/只读 FS/cgroup）。策略代码在编译期类型检查。 |
| G5 | **风控门不变**：ADR-0020 D6 保持——所有下单意图必经 Go 风控门。 |
| G6 | **Decimal 一致性**：全链路使用 `shopspring/decimal`，无跨语言精度转换。 |
| G7 | **行为对齐 harness 用 Go SimBroker 重写**。Python 侧 `behavioral_harness.py` 退役。 |

## 3. 备选方案

| 方案 | 优点 | 缺点 | 否决理由 |
|------|------|------|----------|
| 继续修补 Python 路线 | 已有大量 Python 代码 | 架构缺陷不可修复（A1/A9/A10/A11），安全沙箱不可靠 | 沉没成本谬误 |
| Python + Go 双轨并行 | 渐进迁移 | 维护两套 SDK/运行时/转译器，复杂度翻倍 | 违反"不造空壳"原则 |
| **全 Go（本决策）** | 架构统一、无 IPC、无 RCE、类型安全、与后端无缝 | 需重写行为对齐 harness、Go 生态无 ML 库 | 采纳；ML 不在 EA 场景需求内 |

### 3.1 代码生成方式决策

> **§3.1 已被 ADR-0023 取代**。运行时不再使用 Go 代码生成，改为 MQL → AST → Bytecode VM 进程内执行。以下内容保留仅供历史参考。

`mql2go` 代码生成采用 **tree-sitter CST → StrategyIntent IR → 字符串拼接（emitf）** 架构。

| 方案 | 评估 | 结论 |
|------|------|------|
| **字符串拼接（emitf）** | 简单直接，模式匹配式生成足够覆盖常见 MQL 模式 | **采纳**（仅 CLI 开发调试） |
| `go/ast` + `format.Node` | 保证语法正确性，但构造 AST 节点复杂度高，对当前模式匹配式生成属于过度工程 | **否决** — 2026-06-27 |

> **注意**：Python 侧的 `ast_transpiler.py`（MQL → Python AST）已废弃。`go/ast`（Go 标准库，用于 Go 代码生成端）是不同的东西，但同样不采纳。两者都不要重新提起。

## 4. 后果

- **正面**：
  - 消除跨语言 IPC 开销，策略执行延迟降低
  - 消除安全沙箱复杂度——Go 编译产物天然安全
  - 消除 Decimal 精度跨语言转换问题
  - 消除 ADR-0020 审计中的全部 P1 缺陷（A1/A9/A10/A11/D7）
  - 策略代码可 `go vet`/`go build` 编译期检查
  - 部署简化——少一个 Python 服务
  - 与现有 Go 后端共享类型系统（`sdk.Position`/`sdk.OrderRequest` 等）

- **负面**：
  - ~~Python `strategy-service/` 的回测引擎（fill/cost/margin/portfolio/market）功能丰富，Go 侧 `backtest/` 需要补齐~~ → ✅ 已补齐（T2）
  - ~~行为对齐 harness 需要用 Go SimBroker 重新实现~~ → ✅ 已实现（parity 框架：MT 报告解析 + 交易逐笔对比）
  - ~~Go `indicatorSet` 中部分指标是 stub（Stochastic/CCI/ADX/MFI/OBV/SAR/StdDev/WPR），需要补齐实现~~ → ✅ 已补齐（T1: 全部 38 个指标真实计算）
  - ~~`mql2go` 识别器覆盖率需评估——目前从 MQL4 CST 提取，MQL5（class/OOP）支持待验证~~ → ✅ 已完成（MQL5 CTrade/Position/事件回调全部支持）

- **中性**：
  - `strategy-service/` 代码保留作为参考，但不再维护
  - factor DSL / ONNX 继续保持货架状态（与 ADR-0020 一致）

## 5. 退役清单

| 模块 | 路径 | 处置 |
|------|------|------|
| Python 策略服务 | `strategy-service/` | 标记 deprecated，停止维护 |
| Python MQL 翻译器 | `tools/mql_transpiler/` | 标记 deprecated，由 `backend/tools/mql2go/` 替代 |
| Python 迁移引擎 | `tools/mql_migration/` | 标记 deprecated，由 `backend/tools/mql2go/` 替代 |
| Python Strategy SDK | `strategy-service/app/sdk/` | 标记 deprecated，由 `backend/strategy/sdk/` 替代 |
| Python SimBroker | `strategy-service/app/engine/sim_broker.py` | 标记 deprecated，由 `backend/strategy/backtest/broker.go` 替代 |
| Python LiveBroker | `strategy-service/app/engine/live_broker.py` | 标记 deprecated，由 `backend/strategy/runner/broker.go` 替代 |
| Python behavioral_harness | `tools/mql_migration/behavioral_harness.py` | 标记 deprecated，需在 Go 侧重写 |
| PythonStrategyService proto | `proto/ant/v1/python_strategy.proto` | 退役后删除 |

## 6. Go 侧待补齐

| 项目 | 当前状态 | 优先级 |
|------|---------|--------|
| `indicatorSet` stub 指标 | ✅ 已实现（T1: 24 个 stub 指标全部真实计算） | ~~P1~~ 完成 |
| Go SimBroker 功能补齐 | ✅ 已实现（T2: PositionClosePartial/PositionCloseBy/HistoryOrders/Deals/SymbolInfo） | ~~P1~~ 完成 |
| `GoStrategyExecutor` 完整实现 | ✅ 已实现（解释器路径替代，`interp.Interpreter` 实现 `sdk.Strategy` + 可选接口） | ~~P1~~ 完成 |
| `mql2go` MQL5 支持 | ✅ 已完成（CTrade/Position/OnTrade/OnTradeTransaction 事件回调） | ~~P2~~ 完成 |
| `mql2go` 表达式翻译 | ✅ 已完成（解释器路径不再需要表达式翻译，直接执行 IR） | ~~P2~~ 完成 |
| Go 回测 metrics | ✅ 已完成 | ~~P2~~ 完成 |
| SymbolInfo/MarketInfo | ✅ 已实现（T3: SymbolInfoDouble/Integer/String + MarketInfo） | ~~P2~~ 完成 |
| OrderCloseBy + OrdersHistoryTotal | ✅ 已实现（T4: 对冲平仓 + 历史订单遍历） | ~~P2~~ 完成 |
| MQL5 事件回调 | ✅ 已实现（T5: OnTrade/OnTradeTransaction） | ~~P2~~ 完成 |
| `*OnArray` 指标 | ✅ 已实现（T6a: iMAOnArray/iRSIOnArray 等 8 个函数） | ~~P3~~ 完成 |
| 永久盲区标记 | ✅ 已实现（T6c: File/Object/DLL/NativeOrderSend 标记为"永久盲区"） | ~~P3~~ 完成 |
| iCustom 自定义指标 | ⏳ 待规划（需 OnCalculate + buffer 模型 + 指标加载机制） | P3 |
| Go 行为对齐 harness | ✅ 已实现（`parity.go` + `parity_runner.go` + `mt_report.go`：MT4/MT5 报告解析 + 交易逐笔对比 + 容差配置） | ~~P2~~ 完成 |

## 7. 验证方式

- **编译通过**：`mql2go` 生成的 Go 代码能 `go build` 通过
- **SDK 接口实现**：生成的策略实现 `sdk.Strategy` 接口（`var _ sdk.Strategy = (*StrategyName)(nil)`）
- **回测一致性**：同一 MQL EA 在 Go SimBroker 和 MT Strategy Tester 的信号序列对齐
- **实盘一致性**：Go `LiveRunner` 执行策略信号与回测结果一致（同码不变量）
- **风控不可绕过**：Go 策略的所有 `Broker.OrderSend` 调用必经 `Gate.Evaluate`
