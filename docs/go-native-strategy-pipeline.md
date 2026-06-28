# Go-Native Strategy Pipeline

> **技术文档** — 供 AI 助手（DeepSeek 等）接手开发时参考。读完本文即可理解整条管线的架构、模块边界、数据流和已知缺口。
>
> **版本**: v3 — 2026-06-28 更新。新增 MQL 策略层（解释器路径 + 翻译器双路径）、WASM 运行时迁移、delta-bar 协议、回测对齐校验闭环、LiveRunner 删除等架构变更。

## 1. 概述

Go-Native Strategy Pipeline 是 AntTrader 平台中 **从 MQL 源码到实盘执行** 的完整管线。它取代 MetaTrader EA 运行时，用 Go 编译产物替代 MQL 解释执行，实现：

- MQL4/MQL5 EA → Go 代码转译（构建时）
- Go 策略代码在统一 SDK 接口上运行（运行时）
- 回测与实盘共用同一份策略代码（差异仅在注入的 Broker 实现）
- 全链路 `decimal.Decimal` 精度，无 `float64` 金融计算
- 所有下单意图必经风控 Gate（不可绕过）
- **WASM 沙箱执行**（wazero）— 替代子进程模型，~200ms 编译 vs ~5-10s `go run`
- **Delta-bar 协议** — 首次全量 OHLCV，后续仅传增量 bar，减少 IPC 开销
- **回测对齐校验闭环** — Go SimBroker 交易序列 vs MT Strategy Tester 报告比对

**关联 ADR**：ADR-0021（Python→Go 迁移）、ADR-0020（EA 替代架构，D1/D4/D5/D7 已被 0021 覆盖）、ADR-0012（统一回测/实盘路径）。

### 1.1 层级结构

```
┌─ 市场数据层 ──  mtapi gRPC → mdtick DTO → BarBroker / TickBroker
│
├─ MQL 策略层 ──  输入 MQL 源码，输出 sdk.Strategy
│   ├── 编译路径:  mql2go/ 转译器（构建时）
│   └── 解释路径:  mql2go/interp/ 解释器（运行时）
│
├─ 策略执行层 ──  LiveStrategyRunner / WASM / Runner
│
├─ 信号派发层 ──  dispatchLiveSignal → Gate
│
├─ 订单执行层 ──  MtHubService / PaperEngine
│
└─ 持久化层  ──  PG / ClickHouse / NATS JetStream
```

### 1.2 MQL 策略层

> 输入 MQL 源码，输出 `sdk.Strategy`。包含编译路径（`mql2go/` 转译器，构建时）和解释路径（`mql2go/interp/` 解释器，运行时），共享 tree-sitter CST → IR 前端。

| 路径 | 输入 | 输出 | 方式 | 适用场景 |
|------|------|------|------|---------|
| 编译路径 | MQL 源码 | Go 源码 → WASM 原生执行 | `mql2go.Analyze` + `mql2go.Generate` | 翻译器已识别模式（~85%+ EA） |
| 解释路径 | MQL 源码 | 纯 Go IR → WASM 解释执行 | `mql2go.CompileToIR` → `interp.Interpreter` | 翻译器盲点兜底（100% 覆盖率） |

两条路径共享 tree-sitter 解析器（`mql_lang.go`），输出统一的 `sdk.Strategy` 接口，对下游执行层透明。

## 2. 管线全景

```
┌─ MQL 策略层 ─────────────────────────────────────────────────────────┐
│                                                                    │
│  MQL 源码 (.mq4 / .mq5)                                            │
│    │                                                               │
│    ▼ tree-sitter parse (mql_lang.go)                               │
│  CST                                                               │
│    │                                                               │
│    ├─── 编译路径 ────  recognizer.go → StrategyIntent IR            │
│    │                  gen.go → Go 源码 → WASM 原生执行               │
│    │                                                               │
│    └─── 解释路径 ────  compile_interp.go → 纯 Go IR                  │
│                       interp.Interpreter → WASM 解释执行            │
│                                                                    │
│  输出: sdk.Strategy (两条路径统一接口)                                │
└────────────────────────┬───────────────────────────────────────────┘
                         │ 生成的 Go 代码
                         ▼
┌─ 运行时 ──────────────────────────────────────────────────────────┐
│                                                                    │
│  Layer 1: 市场数据接入                                              │
│  ┌──────────────────────────────────────────────────────────┐     │
│  │ mtapi gRPC (float64)                                      │     │
│  │   → MT4/MT5 adapter (decimal.NewFromFloat 边界转换)        │     │
│  │   → mdtick DTO (decimal.Decimal)                          │     │
│  │     Tick / Bar / ProfitUpdate / OrderUpdate                │     │
│  └──────────────────────────────────────────────────────────┘     │
│                           │                                        │
│                           ▼                                        │
│  Layer 2: 数据分发 (pipeline.go)                                    │
│  ┌──────────────────────────────────────────────────────────┐     │
│  │ mdtick → mthub DTO                                        │     │
│  │   BarUpdate / AccountProfitEvent / PositionSnapshot       │     │
│  │   (全部 decimal.Decimal)                                   │     │
│  │                                                            │     │
│  │ 分发目标:                                                  │     │
│  │   → BarBroker.Publish     (bar 流广播)                     │     │
│  │   → AccountProfitBroker    (盈亏事件广播)                   │     │
│  │   → PositionSnapshotBroker (持仓快照广播)                   │     │
│  │   → PG/CH 持久化 (pg_writer / ch_market_data_store)        │     │
│  │   → SSE → 前端 (stream_handler)                            │     │
│  └──────────────────────────────────────────────────────────┘     │
│                           │                                        │
│                           ▼ bar 流                                  │
│  Layer 3: 策略执行 (WASM 沙箱)                                       │
│  ┌──────────────────────────────────────────────────────────┐     │
│  │ LiveStrategyRunner                                        │     │
│  │   订阅 BarBroker → 滚动窗口 (max 500 bars)                 │     │
│  │   首次: buildLiveContext (全量 OHLCV string arrays)        │     │
│  │   后续: buildDeltaContext (仅 DeltaBar 增量)               │     │
│  │   backfillContextStrings (持仓/权益从 MT 网关回填)          │     │
│  │                                                            │     │
│  │ LiveSession (WASM/wazero)                                  │     │
│  │   编译: GOOS=wasip1 GOARCH=wasm go build                  │     │
│  │   执行: wazero runtime + WASI stdio (in-memory pipes)     │     │
│  │   通信: length-prefixed protobuf via io.Pipe              │     │
│  │   首次: Start() → OnInit + 第一个 OnBar → Response         │     │
│  │   后续: SendBar() → OnBar → Response (无重编译)            │     │
│  │   多事件: BAR / TICK / TRADE / TIMER 分发                  │     │
│  │                                                            │     │
│  │ ExecuteLiveResponse.Signals[]                              │     │
│  └──────────────────────────────────────────────────────────┘     │
│                           │                                        │
│                           ▼ signals                                 │
│  Layer 4: 信号派发 + 风控                                            │
│  ┌──────────────────────────────────────────────────────────┐     │
│  │ dispatchLiveSignal                                        │     │
│  │   switch action:                                           │     │
│  │     buy/sell           → dispatchMarketOrder               │     │
│  │     buy_limit/sell_..  → dispatchPendingOrder              │     │
│  │     close/close_all    → dispatchCloseOrder                │     │
│  │     modify             → dispatchModifyOrder               │     │
│  │     cancel             → dispatchCancelOrder               │     │
│  │                                                            │     │
│  │ submitOrder:                                               │     │
│  │   → risk.Gate.Evaluate(OrderIntent, AccountState)          │     │
│  │     ALLOW → 继续 | DENY → 审计日志 + metric                  │     │
│  │   → mthub.PlaceOrder (实盘) 或 PaperEngine (模拟)            │     │
│  └──────────────────────────────────────────────────────────┘     │
│                           │                                        │
│                           ▼                                        │
│  Layer 5: 订单执行                                                   │
│  ┌──────────────────────────────────────────────────────────┐     │
│  │ 实盘: mthub.MtHubService.PlaceOrder                       │     │
│  │   → 幂等检查 → OMS 状态机 → broker executor → MT 网关       │     │
│  │   → 异步发布 OrderCreated 事件                              │     │
│  │                                                            │     │
│  │ 模拟: PaperEngine.PlacePaperOrder                          │     │
│  │   → decimal 精度计算成交价                                   │     │
│  │   → 更新模拟账户余额                                         │     │
│  │   → SSE 广播更新                                             │     │
│  └──────────────────────────────────────────────────────────┘     │
│                                                                    │
└────────────────────────────────────────────────────────────────────┘

┌─ 校验闭环 ────────────────────────────────────────────────────────┐
│                                                                    │
│  MT Strategy Tester HTML 报告                                      │
│    │                                                               │
│    ▼ ParseMTReport (mt_report.go)                                  │
│  MTReport { Trades[]: MTReportTrade }                              │
│    │                                                               │
│  Go SimBroker 回测结果                                              │
│    │  Result { Trades[]: Trade }                                   │
│    │                                                               │
│    ▼ CompareParity (parity.go)                                     │
│  ParityReport { Passed, Mismatches[], Summary }                    │
│    │  按开盘时间贪心对齐 → 比较 side/volume/price/profit           │
│    │  容差: 时间 2min, 价格 2 pips, 手数 0.01                      │
│    ▼                                                               │
│  PASS / FAIL + 差异详情                                             │
│                                                                    │
└────────────────────────────────────────────────────────────────────┘
```

## 3. 模块清单

### 3.1 构建时 — mql2go 转译器

**路径**: `backend/tools/mql2go/`

| 文件 | 职责 |
|------|------|
| `analyze.go` | tree-sitter parse → CST；`detectMQLVersion` 识别 MQL4 vs MQL5 |
| `recognizer.go` | CST 模式匹配 → `StrategyIntent` IR；版本感知提取（MQL4 `OrderSend` vs MQL5 `CTrade`） |
| `ir.go` | `StrategyIntent` 及所有子类型定义（`EntryRule`/`ExitRule`/`OrderLoopRule`/`PositionLoopRule`/`IndicatorSpec`/`ModifyRule`/`RiskCheck`/`SizingRule`/`BlindSpot`） |
| `gen.go` | IR → Go 源码（string emit）；表达式转换（`Close[1]` → `ctx.Bars().Close(1).InexactFloat64()`） |

**转译策略**: 基于模式识别的语义级转译（Pattern-Recognition Semantic Transpilation）。不是逐行语法翻译，不是功能理解后重写。识别 MQL 交易模式 → 映射到 Go SDK 调用。

**已识别模式**:
- MQL4: `OrderSend` / `OrderClose` / `OrderModify` / `OrderDelete` / `OrderCloseBy` / `OrderSelect`+`OrdersTotal` 循环 / 追踪止损 / 16 个 `Order*` 属性函数 / `extern` 参数
- MQL5: `CTrade.Buy/Sell/BuyLimit/...` / `PositionClose/Modify` / `PositionsTotal`+`PositionGetTicket` 循环 / `PositionGetDouble/Integer/String` / `input` 参数 / `_Point` / `_Symbol`
- 指标: 30+ 指标全部生成 `ctx.Indicators().*()` 调用

**Blind spots** (识别不到 → 注释标记):
- `iCustom` (自定义指标)
- MQL5 原生 `OrderSend(MqlTradeRequest, MqlTradeResult)`
- `*OnArray` 指标变体
- MQL5 事件: `OnTrade`/`OnTradeTransaction`/`OnBookEvent`/`OnTesterInit/Deinit/Pass`

### 3.2 SDK 接口层

**路径**: `backend/strategy/sdk/`

| 文件 | 接口 | 关键方法 |
|------|------|---------|
| `strategy.go` | `Strategy` | `OnInit(Context) error` / `OnBar(Context, string) (*Signal, error)` / `OnDeinit(Context, string) error` |
| `strategy.go` | `TickStrategy` (可选) | `OnTick(Context, decimal.Decimal, decimal.Decimal) (*Signal, error)` |
| `strategy.go` | `TradeStrategy` (可选) | `OnTrade(Context, TradeEvent) (*Signal, error)` |
| `strategy.go` | `TimerStrategy` (可选) | `OnTimer(Context) (*Signal, error)` |
| `context.go` | `Context` | `Bars() BarSeries` / `BarsTF(string) BarSeries` / `Broker() Broker` / `Indicators() IndicatorSet` / `Param*(...)` / `Ask() decimal.Decimal` / `Bid() decimal.Decimal` / `Point() decimal.Decimal` / `Account() AccountInfo` |
| `broker.go` | `Broker` | `OrderSend(OrderRequest) (OrderResult, error)` / `PositionClose(int64, decimal.Decimal) (OrderResult, error)` / `PositionModify(int64, decimal.Decimal, decimal.Decimal) (OrderResult, error)` / `OrderDelete(int64) (OrderResult, error)` / `Positions(int32) []Position` / `Orders(int32) []PendingOrder` / `SymbolInfo(string) (SymbolInfo, error)` / `Account() AccountInfo` |
| `indicators.go` | `IndicatorSet` | `MA/EMA/RSI/MACD/MACDSignal/ATR/Bollinger/Stochastic/CCI/ADX/MFI/OBV/SAR/StdDev` + 15 共享 stub + 9 MQL5-only stub |
| `series.go` | `BarSeries` | `Open(int) decimal.Decimal` / `High(int) decimal.Decimal` / `Low(int) decimal.Decimal` / `Close(int) decimal.Decimal` / `Volume(int) int64` / `Time(int) int64` / `Len() int` — MQL 逆序索引: `[0]`=当前, `[1]`=前一根 |
| `types.go` | 核心类型 | `OrderRequest` / `OrderResult` / `Signal` / `Position` / `PendingOrder` / `Deal` / `AccountInfo` / `SymbolInfo` / `RetCode` / `TradeEvent` / `TradeEventType` — 全部使用 `decimal.Decimal` |

**SignalAction 枚举**: `ActionNone` / `ActionBuy` / `ActionSell` / `ActionBuyLimit` / `ActionSellLimit` / `ActionBuyStop` / `ActionSellStop` / `ActionClose` / `ActionModify` / `ActionCancel` / `ActionCloseAll` / `ActionCancelAll`

### 3.3 运行时 — 策略执行层

**路径**: `backend/internal/connect/strategy/`

| 文件 | 职责 |
|------|------|
| `strategy_execution_handler.go` | `StrategyExecutionServer` — 依赖注入中心；注入 `backtestRepo` / `marketDataRepo` / `barSource` / `mtHub` / `paperEngine` / `gate` / `accountProvider` / `goExecutor` / `wasmExecutor` |
| `wasm_executor.go` | `WasmExecutor` — wazero 运行时管理；`CompileStrategy` (Go→WASM) / `Run` (实例化执行) / `CompilationCache` (跨 session 复用) |
| `live_runner.go` | `RunLiveStrategy` — 多事件循环 (BAR/TICK/TRADE/TIMER)；`handleBar` / `handleTick` / `handleTrade`；`buildLiveContext` (全量) / `buildDeltaContext` (增量) |
| `live_session.go` | `LiveSession` — WASM session 生命周期；`Start` (编译+首次执行) / `SendBar` (增量 bar) / `Close` (EOF→OnDeinit)；in-memory `io.Pipe` 通信 |
| `live_dispatch.go` | `dispatchLiveSignal` — 信号路由；`dispatchMarketOrder` / `dispatchPendingOrder` / `dispatchCloseOrder` / `dispatchModifyOrder` / `dispatchCancelOrder` / `dispatchPaperSignal` |
| `backtest_harness.go` | `generateBacktestHarness` (回测 harness) / `generateLiveHarness` (WASM live harness)；harness 内含 delta-bar 窗口管理、多事件分发、`mustDecimal`/`mustInt64` |
| `go_executor.go` | `GoExecutor` — 回测用 `go run` 子进程执行；`Run` (单次执行) / `RunBacktest` (回测) |
| `backtest_worker.go` | `StartBacktestWorker` — 异步回测工作池 (SKIP LOCKED)；`executeGoBacktest` |
| `data_source.go` | `BarSource` 接口 / `LiveBarSubscriber` 接口 / `klineBarsToProto` — 历史数据 + 实时订阅统一抽象 |
| `account_provider.go` | `AccountStateProvider` 接口 — 实时账户状态供给 Gate 评估 |

**WASM 执行架构** (替代子进程模型):

```
Host goroutine              WASM goroutine (harness)
─────────────              ──────────────────────
InstantiateModule ──→     main():
                             readRequest(stdin) ─── blocks on pipe
write(initialReq) ──→       unblocks → OnInit + first OnBar
                             writeResponse(stdout)
readResponse() ←────        readRequest(stdin) ─── blocks on pipe
write(barReq) ─────→        unblocks → OnBar
                             writeResponse(stdout)
readResponse() ←────        readRequest(stdin) ─── blocks
... (loop per bar)
```

- 编译: `GOOS=wasip1 GOARCH=wasm go build -o strategy.wasm strategy.go harness.go`
- 运行: wazero `InstantiateModule` + WASI stdio piped through `io.Pipe`
- 缓存: `wazero.CompilationCache` — 相同 wasm 二进制不重编译
- 通信: 4 字节 big-endian length prefix + protobuf body (与子进程模型相同)

**Delta-bar 协议**:

首次 bar 请求 (full context):
```
LiveStrategyContext {
    Close[]: ["1.1000", "1.1010", ...]     // 全量 OHLCV string arrays
    Open[]:  ["1.0995", "1.1005", ...]
    High[]:  ["1.1005", "1.1015", ...]
    Low[]:   ["1.0990", "1.1000", ...]
    Volume[]: ["1000", "1200", ...]
    BarTimesMs[]: [1718035200000, ...]
    DeltaBars: []                            // 空
}
```

后续 bar 请求 (delta only):
```
LiveStrategyContext {
    DeltaBars: [{                            // 仅增量 bar
        Open: "1.1020", High: "1.1025",
        Low: "1.1015", Close: "1.1022",
        Volume: "800", BarTimeMs: 1718035260000
    }]
    Close[]: []                              // 空 — harness 自己维护窗口
}
```

Harness 侧窗口管理 (`backtest_harness.go` `handleBar`):
- `DeltaBars` 非空 → append 到 `barWindow` (max 500)
- `DeltaBars` 空 + `Close[]` 非空 → 全量重建 `barWindow`

**多事件支持**:

`ExecuteLiveRequest.RequestType` 分发:
- `REQUEST_TYPE_BAR` → `handleBar` → `runner.OnBar`
- `REQUEST_TYPE_TICK` → `handleTick` → `runner.OnTick` (需 `TickStrategy`)
- `REQUEST_TYPE_TRADE` → `handleTrade` → `runner.OnTrade` (需 `TradeStrategy`)
- `REQUEST_TYPE_TIMER` → `handleTimer` → `runner.OnTimerTick` (需 `TimerStrategy`)

**LiveStrategyContext proto 字段**:
- `Close[]` / `Open[]` / `High[]` / `Low[]` / `Volume[]` — string 数组 (decimal.String())
- `BarTimesMs[]` — int64 数组
- `DeltaBars[]` — `DeltaBar` proto (增量 bar)
- `Symbol` / `Timeframe` / `Mode` / `CurrentPrice` — string
- `Equity` / `Balance` — string (decimal.String()，`"-1"` = provider 未连接，fail-closed)
- `Positions[]` — `LivePosition` proto
- `Params[]` — `LiveParam` key-value

### 3.4 运行时 — Runner (WASM 内执行器)

**路径**: `backend/strategy/runner/`

| 文件 | 职责 |
|------|------|
| `runner.go` | `Runner` — 策略生命周期管理；`Init` / `OnBar` / `OnTick` / `OnTrade` / `OnTimerTick` / `Deinit`；`UpdateLiveState` / `UpdateTickState` (从 host 传入状态) |
| `broker.go` | `brokerImpl` — 实现 `sdk.Broker`；harness 模式 (无 executor，返回 `RetRejected`) |
| `context.go` | `contextImpl` — 实现 `sdk.Context`；持有 `liveBalance` / `liveEquity` / `livePositions` (由 `UpdateLiveState` 更新) |
| `indicators.go` | `indicatorSet` — 实现 `sdk.IndicatorSet`；14 完整实现 + 24 stub |
| `indicators_decimal.go` | 指标计算辅助 (decimal 精度) |

> **注意**: `runner.LiveRunner` 已删除 (v2)。原 `LiveRunner` 绕过风控 Gate 直接下单，是安全隐患。实盘执行现在通过 `LiveStrategyRunner` → `LiveSession` (WASM) → `dispatchLiveSignal` → `risk.Gate` 路径，风控不可绕过。

### 3.5 运行时 — 回测引擎 + 对齐校验

**路径**: `backend/strategy/backtest/`

| 文件 | 职责 |
|------|------|
| `engine.go` | `Engine` — 回测执行器；`Run` 遍历 bars → `checkPendingOrders` / `checkSLTP` / `OnBar` / `dispatchSignal`；`backtestContext` 实现 `sdk.Context` |
| `broker.go` | `SimBroker` — 实现 `sdk.Broker`；模拟订单成交 (market=即时, pending=价格触发)；margin 检查、commission、swap |
| `types.go` | `Config` / `Result` / `Trade` / `EquityPoint` / `OrderRecord` / `OrderState` — 全部 `decimal.Decimal` |
| `metrics.go` | `CalculateMetrics` — TotalReturn / AnnualReturn / MaxDrawdown / SharpeRatio / WinRate / ProfitFactor |
| `indicators_decimal.go` | 回测指标实现 (14 完整 + 24 stub) |
| `mt_report.go` | **MT Strategy Tester 报告解析器** — `ParseMT4Report` / `ParseMT5Report` / `ParseMTReport` (自动检测)；解析 HTML 表格 → `MTReportTrade[]` |
| `parity.go` | **交易对齐比较器** — `CompareParity` 按开盘时间贪心匹配；比较 side (fatal) / volume / price / profit (warning)；`ParityConfig` 容差配置；`ParityReport` 差异报告 |
| `parity_runner.go` | **校验闭环编排器** — `RunParityTest` (Go 回测 → 解析 MT 报告 → 比较 → 报告)；`QuickParityCheck` / `MakeSyntheticMTReport` (CI 用) |
| `parity_test.go` | 8 个测试: MT 时间解析 / MT4 报告解析 / 精确匹配 / side 不匹配 / 缺失交易 / 价格容差 / 报告格式化 / 自动检测 |

**回测对齐校验闭环使用方式**:

```go
report, err := backtest.RunParityTest(ctx, backtest.ParityTestInput{
    Strategy:     goStrategy,       // 实现 sdk.Strategy 的 Go 策略
    Bars:         historicalBars,   // 与 MT Strategy Tester 相同数据范围
    Config:       backtest.Config{...},
    MTReportHTML: mtTesterHTML,     // MT4/MT5 Strategy Tester 导出的 HTML
})
fmt.Println(report.FormatReport())
// === Backtest Parity Report: PASS ===
// Go trades: 30 | MT trades: 30 | Matched: 30
```

**容差默认值** (`DefaultParityConfig`):
- 时间: ±2 分钟 (bar 时间戳解释差异)
- 价格: ±2 pips (滑点差异)
- 手数: ±0.01 lot
- side 不匹配 = fatal；其他 = warning

### 3.6 运行时 — OMS + 数据分发

**路径**: `backend/internal/mthub/`

| 文件 | 职责 |
|------|------|
| `service.go` | `MtHubService` — 会话管理；`OpenedOrders` / `SubscribeSymbols` |
| `service_orders.go` | `PlaceOrder` — 幂等检查 → 风控 → OMS 状态机 → broker executor；`decimalToFloat64` 在 MT 网关边界转换 |
| `types.go` | `Hub` / `Session` / `AccountProfitEvent` / `AccountProfitPosition` / `AccountProfitBroker` / `OrderEventBroker` — 全部 `decimal.Decimal` |
| `broker_types.go` | `BarUpdate` / `BarBroker` / `PositionSnapshot` / `PositionSnapshotItem` / `PositionSnapshotBroker` / `TickUpdate` / `BrokerTradeEvent` — 全部 `decimal.Decimal` (Volume 除外，`float64`) |
| `order_types.go` | `OrderRecord` / `SymbolParam` / `Bar` — 全部 `decimal.Decimal` |

**关键 DTO 定义**:

```go
// mthub.BarUpdate — 实时 bar 推送
type BarUpdate struct {
    AccountID string
    Symbol    string
    Period    string
    OpenTime  int64
    Open      decimal.Decimal
    High      decimal.Decimal
    Low       decimal.Decimal
    Close     decimal.Decimal
    Bid       decimal.Decimal
    Ask       decimal.Decimal
    Volume    float64          // 统计聚合用途，非金融计算
    Closed    bool
}

// mthub.AccountProfitEvent — 实时盈亏推送
type AccountProfitEvent struct {
    AccountID                               string
    Balance, Credit, Equity                 decimal.Decimal
    Margin, FreeMargin, MarginLevel, Profit decimal.Decimal
    ProfitPercent                           float64
    Positions                               []AccountProfitPosition
}

// mthub.PositionSnapshot — 完整持仓快照
type PositionSnapshot struct {
    AccountID   string
    Balance     decimal.Decimal
    Equity      decimal.Decimal
    Margin      decimal.Decimal
    FreeMargin  decimal.Decimal
    MarginLevel decimal.Decimal
    Profit      decimal.Decimal
    Positions   []PositionSnapshotItem
}
```

### 3.7 运行时 — 风控 Gate

**路径**: `backend/internal/risk/gate.go`

```go
type AccountState struct {
    Balance        decimal.Decimal
    Equity         decimal.Decimal
    FreeMargin     decimal.Decimal
    UsedMargin     decimal.Decimal
    OpenPositions  int
    DailyPnL       decimal.Decimal
    PeakEquity     decimal.Decimal
    SymbolLeverage int
}

type Gate struct {
    rules []Rule
    // ...
}

func (g *Gate) Evaluate(ctx context.Context, intent *antv1.OrderIntent, state *AccountState) *RiskDecision
```

**语义**: fail-closed — `AccountState` 为 nil 或 equity 负数时，equity 相关规则拒绝。支持全局 kill-switch 和 per-user autotrade switch。

### 3.8 运行时 — 模拟交易引擎

**路径**: `backend/internal/paper/engine.go`

```go
func (e *PaperEngine) PlacePaperOrder(
    ctx context.Context,
    accountID, symbol, side string,
    volume decimal.Decimal,
    bid, ask decimal.Decimal,  // decimal 精度
) error
```

使用 `decimal` 算术计算成交价: `fillPrice = bid.Add(ask).Div(decimal.NewFromInt(2))`。

### 3.9 运行时 — 市场数据网关

**路径**: `backend/internal/mdgateway/`

| 目录/文件 | 职责 |
|----------|------|
| `adapter/mdtick/mdtick.go` | 统一 DTO: `Tick` / `Bar` / `ProfitUpdate` / `OrderUpdate` / `AccountConfig` — 全部 `decimal.Decimal` (Volume/BidVolume/AskVolume 为 `float64`) |
| `adapter/mt4/` | MT4 gRPC adapter — `decimal.NewFromFloat()` 在 gRPC 边界转换 |
| `adapter/mt5/` | MT5 gRPC adapter — 同上 |
| `runner.go` | Gateway 生命周期管理；回调注入: `OnAccountProfit` / `OnOrderUpdate` / `OnAccountDisconnect` / `OnBrokerInfo` |
| `pg_writer.go` | bar 批量写入 PG (decimal.Decimal 直传) + 异步双写 CH |
| `pipeline.go` (cmd/server) | `OnBar` / `OnAccountProfit` / `OnOrderUpdate` 回调实现 — mdtick → mthub DTO 转换 + 分发 |

### 3.10 运行时 — 持久化

**路径**: `backend/internal/repository/`

| 文件 | 职责 |
|------|------|
| `market_data_types.go` | `KlineBar` struct — OHLC 为 `decimal.Decimal`，Volume 为 `float64` |
| `pg_market_data_store.go` | PG bar 读写 — `decimal.Decimal` 直传 |
| `ch_market_data_store.go` | CH bar 读写 — `decimal.Decimal` 直传 |
| `backtest_run_trades.go` | `BacktestRunTrade` — 回测交易持久化 (PG `backtest_run_trades` 表) |

### 3.11 运行时 — SSE 流处理

**路径**: `backend/internal/connect/system/`

| 文件 | 职责 |
|------|------|
| `stream_handler.go` | `formatPrice(decimal.Decimal) string` — 动态精度 `.StringFixed()`；bar_update SSE 事件 |
| `stream_handler_events.go` | `emitPositionSnapshot` — `decimal.String()` 格式化持仓字段 |
| `stream_helpers.go` | `profitEventToProto` — `decimal.String()` 格式化盈亏事件 |
| `mthub_service_extra.go` | `PriceHistory` — `decimal.String()` 格式化 OHLCV |
| `mthub_service_backfill.go` | broker fallback — `b.Volume.InexactFloat64()` 在 `KlineBar` 边界转换 |

## 4. 数据类型链 — float64 → decimal.Decimal 边界

### 4.1 边界转换规则

| 边界 | 方向 | 方法 | 原因 |
|------|------|------|------|
| mtapi gRPC → adapter | `float64` → `decimal.Decimal` | `decimal.NewFromFloat()` | mtapi proto 返回 `float64` |
| adapter → pipeline | `decimal.Decimal` 直传 | — | mdtick DTO 全部 decimal |
| pipeline → mthub DTO | `decimal.Decimal` 直传 | — | mthub DTO 全部 decimal |
| mthub DTO → SSE proto | `decimal.Decimal` → `string` | `.String()` / `.StringFixed(N)` | proto wire 用 string |
| mthub DTO → LiveStrategyContext | `decimal.Decimal` → `string` | `.String()` | proto IPC 用 string |
| KlineBar → 统计分析 (ATR/EMA/regime) | `decimal.Decimal` → `float64` | `.InexactFloat64()` | 统计聚合，非金融计算 |
| KlineBar → PG/CH 持久化 | `decimal.Decimal` 直传 | — | DB 支持 NUMERIC/Decimal |
| KlineBar.Volume | `float64` (保留) | — | 成交量非价格，统计用途 |
| AccountService API | `decimal.Decimal` → `float64` | `.InexactFloat64()` | 服务层尚未迁移 (future task) |
| risk.Gate.OrderIntent | `decimal.Decimal` → `string` | `.String()` | proto wire 用 string |

### 4.2 绝对禁止

- `float64` 用于价格计算（OHLC、Bid/Ask、Balance/Equity/Margin/Profit）
- `json.Marshal` / `json.Unmarshal` 用于数据交换（用 proto）
- 跨 `decimal.Decimal` 比较用 `==`（用 `.Equal()` / `.GreaterThan()` / `.LessThanOrEqual()`）

## 5. mql2go IR 结构

```go
type StrategyIntent struct {
    Meta           StrategyMeta       // 策略名 + MQL 版本
    Params         []ParamSpec        // extern/input 参数
    State          []StateVar         // 策略全局变量
    Entry          []EntryRule        // 入场规则 (OrderSend → Action + Conditions)
    Exit           []ExitRule         // 出场规则 (reverse_signal / magic_close / close_all)
    Modifies       []ModifyRule       // OrderModify (trailing_stop / manual_modify)
    OrderLoops     []OrderLoopRule    // MQL4 OrdersTotal 循环
    PositionLoops  []PositionLoopRule // MQL5 PositionsTotal 循环
    Indicators     []IndicatorSpec    // 指标调用 → SDK method 映射
    Sizing         *SizingRule        // 仓位管理 (fixed / martingale / percent_balance)
    Risk           []RiskCheck        // 风控条件 (closeAll / return / log)
    Execution      ExecutionModel     // on_tick / on_bar / on_init_grid
    Timer          *TimerRule         // EventSetTimer
    BlindSpots     []BlindSpot        // 识别不到的模式
}
```

**代码生成顺序** (gen.go `genOnBar`):
1. Indicators — `ctx.Indicators().*()`
2. Risk — `if condition { closeAll/return/log }`
3. Exits — `ctx.Broker().PositionClose() / OrderDelete()`
4. OrderLoops — `for _, pos := range ctx.Broker().Positions(magic)`
5. PositionLoops — 同上 (MQL5)
6. Modifies — `ctx.Broker().PositionModify()`
7. Entries — `ctx.Broker().OrderSend()`

## 6. 运行时调用链 — 从 bar 到达到订单执行

```
1. mtapi OnQuote stream → mt4/mt5 adapter
2. adapter → mdtick.Tick → mdgateway bar_aggregator → mdtick.Bar
3. pipeline.go OnBar(bar *mdtick.Bar):
   a. pg_writer.Flush → PG + CH 持久化
   b. mthub.BarBroker.Publish(&mthub.BarUpdate{...decimal.Decimal...})
   c. stream_handler SSE → 前端

4. BarBroker → barCh → LiveStrategyRunner.RunLiveStrategy:
   a. append to rolling window (max 500 bars)
   b. first bar: buildLiveContext → full LiveStrategyContext proto
      subsequent: buildDeltaContext → DeltaBar only
   c. backfillContextStrings → AccountStateProvider.GetAccountState + mtHub.OpenedOrders
   d. first bar: LiveSession.Start(reqBytes) → WASM compile + OnInit + OnBar
      subsequent: LiveSession.SendBar(reqBytes) → OnBar (no recompile)
   e. ExecuteLiveResponse.Signals[]

5. ExecuteLiveResponse.Signals[] → dispatchLiveSignal:
   a. paper mode → PaperEngine.PlacePaperOrder (decimal bid/ask)
   b. live mode → submitOrder:
      i.  risk.Gate.Evaluate(OrderIntent, AccountState) → RiskDecision
      ii. ALLOW → mthub.PlaceOrder → OMS → broker executor → MT 网关
      iii. DENY → audit log + metric

6. MT 网关 → OnOrderUpdate stream → adapter → mdtick.OrderUpdate
7. pipeline.go OnOrderUpdate:
   a. writeClosedTradeRecord → PG trade_records
   b. PositionSnapshotBroker.Publish → SSE
   c. AccountProfitBroker.Publish → SSE
   d. platformAgg → 统计聚合
```

## 7. 回测对齐校验闭环

### 7.1 流程

```
同一 MQL EA
    │
    ├──→ MT Strategy Tester (MT4/MT5 原生回测)
    │       │
    │       ▼ 导出 HTML 报告
    │       ParseMTReport (mt_report.go)
    │       │
    │       ▼
    │       MTReport.Trades[]: MTReportTrade
    │
    └──→ mql2go 转译 → Go 策略代码
            │
            ▼ backtest.Engine + SimBroker
            Result.Trades[]: Trade
            │
            ▼
    CompareParity (parity.go)
            │ 按开盘时间贪心对齐
            │ 比较: side (fatal) / volume / price / profit (warning)
            │ 容差: 时间 ±2min, 价格 ±2 pips, 手数 ±0.01
            ▼
    ParityReport
    ├── Passed: true/false
    ├── MatchedCount / GoTradeCount / MTTradeCount
    ├── Mismatches[]: { Type, Severity, GoValue, MTValue, Diff }
    └── FormatReport() → 人类可读输出
```

### 7.2 关键类型

```go
// MTReportTrade — 从 MT 报告解析的单笔交易
type MTReportTrade struct {
    Ticket     int64
    Side       string          // "buy" | "sell"
    Volume     decimal.Decimal
    OpenTime   time.Time
    OpenPrice  decimal.Decimal
    CloseTime  time.Time
    ClosePrice decimal.Decimal
    Profit     decimal.Decimal
    Swap       decimal.Decimal
    Commission decimal.Decimal
    Symbol     string
}

// ParityMismatch — 单个差异
type ParityMismatch struct {
    Type     ParityMismatchType  // missing_in_go / missing_in_mt / side_mismatch / ...
    GoTrade  *Trade
    MTTrade  *MTReportTrade
    Severity string              // "fatal" | "warning" | "info"
    GoValue  string
    MTValue  string
    Diff     string
}
```

### 7.3 CI 集成

`MakeSyntheticMTReport` 可构造合成 MT 报告用于 CI 测试（无需 MT Strategy Tester）:

```go
mtReport := backtest.MakeSyntheticMTReport([]MTReportTrade{
    MakeMTReportTrade("buy", decimal.NewFromFloat(0.1), ...),
}, "EURUSD", "H1")
report := backtest.CompareParity(goTrades, mtReport.Trades, DefaultParityConfig())
assert.True(t, report.Passed)
```

## 8. 已知缺口

| 缺口 | 描述 | 优先级 |
|------|------|--------|
| **runner.OrderExecutor 接口遗留** | `runner.go` 中 `OrderExecutor` 接口仍存在但无使用方 (LiveRunner 已删除)，`broker.go` 中 `executor` 字段待清理 | P1 |
| **risksvc.SignalPipeline 死代码** | `risksvc.SignalPipeline` + `SignalRequest` (float64) 为死代码，待删除 | P1 |
| **mustDecimal 静默零值** | `backtest_harness.go` 中 `mustDecimal` 解析失败返回 `decimal.Zero` 而非 panic，可能掩盖数据问题 | P1 |
| **barsDropped 通知** | 策略无法感知被丢弃的 bar（网络抖动等），需添加 `barsDropped` 字段 | P2 |
| **per-bar OpenedOrders 查询** | `backfillContextStrings` 每 bar 调用 `mtHub.OpenedOrders`，应改为 `PositionSnapshotBroker` 订阅 | P2 |
| **blind spot 启动检查** | `RunLiveStrategy` 启动时未检查策略是否有 blind spot，可能导致未识别模式在实盘静默跳过 | P2 |
| **AccountService 仍用 float64** | `UpdateAccountMetrics` / `RecordBalanceSnapshot` / `UpdateSummaryCache` 接受 `float64`，pipeline 层需 `.InexactFloat64()` 转换 | P2 |
| **indicator stub 未实现** | 15 共享 stub + 9 MQL5-only stub 返回 0，需补齐真实计算 | P1 |
| **Go SimBroker 功能补齐** | 对比 Python SimBroker 的 fill/cost/margin/portfolio 功能缺失 | P1 |
| **mql2go 表达式翻译** | `pyToGoExpr()` 是简单字符串替换，复杂表达式可能出错 | P2 |
| **Go 回测 metrics** | `CalculateMetrics` 需对齐 Python 侧完整 metrics 计算 | P2 |
| **MTAccountInfo 仍用 float64** | `mdtick.MTAccountInfo` 的 Balance/Equity 等仍为 `float64` | P2 |

## 9. 验证方式

| 验证 | 方法 | 状态 |
|------|------|------|
| 编译通过 | `go build ./...` | ✅ |
| 全部测试通过 | `go test ./...` | ✅ |
| 文件行数检查 | `go run ./tools/check-file-lines --strict` | ✅ (无新增违规) |
| mql2go 生成代码编译 | `go vet` on generated MQL4+MQL5 EA | ✅ |
| 精度一致性 | 全链路 `decimal.Decimal`，边界 `.InexactFloat64()` 仅用于统计/服务层 | ✅ |
| 回测对齐 | `backtest.RunParityTest` — Go SimBroker vs MT Strategy Tester 信号序列比对 | ✅ 已实现 (8 个测试通过) |
| 实盘一致性 | Go LiveRunner 信号与回测结果一致 (同码不变量) | ❌ 未验证 |
| 风控不可绕过 | Go 策略所有 `Broker.OrderSend` 必经 `Gate.Evaluate` | ✅ (编译期保证) |

## 10. 关键文件索引

```
backend/
├── tools/mql2go/                          # [构建时] 转译器
│   ├── analyze.go                         #   tree-sitter parse + 版本检测
│   ├── recognizer.go                      #   CST 模式匹配 → IR
│   ├── ir.go                              #   StrategyIntent 类型定义
│   ├── gen.go                             #   IR → Go 源码生成
│   ├── mql2go_test.go                     #   33 个测试
│   └── behavioral_test.go                 #   行为对齐测试 (SimBroker + 生成代码结构验证)
│
├── strategy/
│   ├── sdk/                               # [接口层] SDK 定义
│   │   ├── strategy.go                    #   Strategy / TickStrategy / TradeStrategy / TimerStrategy 接口 + Signal
│   │   ├── context.go                     #   Context 接口
│   │   ├── broker.go                      #   Broker 接口 + AccountInfo
│   │   ├── indicators.go                  #   IndicatorSet 接口 (30+ 指标)
│   │   ├── series.go                      #   BarSeries 接口 (MQL 逆序索引)
│   │   └── types.go                       #   OrderRequest/Result/Position/Deal/TradeEvent
│   ├── runner/                            # [运行时] WASM 内执行器
│   │   ├── runner.go                      #   Runner (生命周期: Init/OnBar/OnTick/OnTrade/OnTimerTick/Deinit)
│   │   ├── broker.go                      #   brokerImpl (harness 模式)
│   │   ├── context.go                     #   contextImpl (liveBalance/liveEquity/livePositions)
│   │   ├── indicators.go                  #   指标实现 (14 完整 + 24 stub)
│   │   └── indicators_decimal.go          #   指标计算辅助
│   └── backtest/                          # [运行时] 回测引擎 + 对齐校验
│       ├── engine.go                      #   Engine + SimBroker + backtestContext
│       ├── broker.go                      #   SimBroker (sdk.Broker 实现)
│       ├── types.go                       #   Config/Result/Trade/EquityPoint/OrderRecord
│       ├── metrics.go                     #   CalculateMetrics
│       ├── indicators_decimal.go          #   回测指标实现
│       ├── mt_report.go                   #   MT4/MT5 Strategy Tester HTML 报告解析器
│       ├── parity.go                      #   交易对齐比较器 (CompareParity)
│       ├── parity_runner.go               #   校验闭环编排器 (RunParityTest)
│       └── parity_test.go                 #   8 个对齐测试
│
├── internal/
│   ├── connect/strategy/                  # [运行时] 策略服务层
│   │   ├── strategy_execution_handler.go  #   StrategyExecutionServer + DI
│   │   ├── wasm_executor.go               #   WasmExecutor (wazero 运行时 + 编译缓存)
│   │   ├── live_runner.go                 #   LiveStrategyRunner (多事件循环 + delta-bar)
│   │   ├── live_session.go                #   LiveSession (WASM session 生命周期 + io.Pipe)
│   │   ├── live_dispatch.go               #   信号派发 (market/pending/close/modify/cancel)
│   │   ├── backtest_harness.go            #   generateBacktestHarness / generateLiveHarness
│   │   ├── go_executor.go                 #   GoExecutor (回测用 go run 子进程)
│   │   ├── backtest_worker.go             #   异步回测工作池 (SKIP LOCKED)
│   │   ├── data_source.go                 #   BarSource + klineBarsToProto
│   │   └── account_provider.go            #   AccountStateProvider 接口
│   │
│   ├── mthub/                             # [运行时] OMS + 数据分发
│   │   ├── service.go                     #   MtHubService
│   │   ├── service_orders.go              #   PlaceOrder (风控→OMS→broker)
│   │   ├── types.go                       #   AccountProfitEvent + Brokers
│   │   ├── broker_types.go                #   BarUpdate/PositionSnapshot/TickUpdate/BrokerTradeEvent
│   │   └── order_types.go                 #   OrderRecord/SymbolParam/Bar
│   │
│   ├── risk/                              # [运行时] 风控
│   │   └── gate.go                        #   Gate.Evaluate (fail-closed)
│   │
│   ├── paper/                             # [运行时] 模拟交易
│   │   └── engine.go                      #   PaperEngine (decimal 精度)
│   │
│   ├── mdgateway/                         # [运行时] 市场数据网关
│   │   ├── adapter/mdtick/mdtick.go       #   统一 DTO (decimal.Decimal)
│   │   ├── adapter/mt4/quotes.go          #   MT4 gRPC → decimal 转换
│   │   ├── adapter/mt5/quotes.go          #   MT5 gRPC → decimal 转换
│   │   ├── runner.go                      #   Gateway 生命周期
│   │   └── pg_writer.go                   #   bar 持久化 (PG + CH)
│   │
│   ├── repository/                        # [持久化]
│   │   ├── market_data_types.go           #   KlineBar (decimal OHLC)
│   │   ├── pg_market_data_store.go        #   PG bar 读写
│   │   ├── ch_market_data_store.go        #   CH bar 读写
│   │   └── backtest_run_trades.go         #   BacktestRunTrade (回测交易持久化)
│   │
│   └── connect/system/                    # [SSE] 流处理
│       ├── stream_handler.go              #   formatPrice + bar_update SSE
│       ├── stream_handler_events.go       #   position snapshot SSE
│       ├── stream_helpers.go              #   profit event → proto
│       └── mthub_service_extra.go         #   PriceHistory API
│
└── cmd/server/
    ├── pipeline.go                        # [运行时] OnBar/OnAccountProfit/OnOrderUpdate 回调
    ├── pipeline_callbacks.go              # [运行时] 回调实现细节
    └── pipeline_helpers.go                # [运行时] convertProfitPositions
```
