# MQL 源码转换系统 — 覆盖率补齐计划

- **创建日期**：2026-06-29
- **关联 ADR**：ADR-0023（AST 解释器 + MQL 源码为唯一真实来源）、ADR-0021（Python→Go 策略迁移，G1/§3.1 已被 0023 覆盖）
- **目标**：将 MQL EA 导入系统的实际执行覆盖率从当前水平提升到生产可用

## 1. 当前状态快照

### 1.1 已完成 ✅

| 模块 | 文件 | 说明 |
|------|------|------|
| tree-sitter 解析 | `tools/mql2go/analyze.go` | MQL4/MQL5 → CST |
| IR 编译 | `tools/mql2go/compile_interp.go` | CST → `interp.IR`（纯 Go） |
| Bytecode 编译 | `tools/mql2go/compile.go` | AST → 线性字节码（一次性，~300ms） |
| VM 执行 | `tools/mql2go/vm.go` + `interp/exec.go` | 指令计数器 + 显式数据栈 + panic recovery |
| 回测路径 | `internal/connect/strategy/backtest_worker.go` | MQL → AST → Bytecode → VM 执行（进程内） |
| 实盘路径 | `internal/connect/strategy/vm_live_session.go` | MQL → AST → Bytecode → VMLiveSession（进程内） |
| Go 代码导出 | `tools/mql2go/gen.go` / `gen_ir.go` | IR → Go 源码（仅 CLI 开发调试，不参与运行时） |
| 前端导入 | `frontend/src/pages/strategy/components/editor/CodeEditorPanel.tsx` | Analyze → Generate → Import |
| 14 个核心指标 | `strategy/backtest/engine.go`, `strategy/runner/indicators.go` | iMA/iRSI/iATR/iMACD/iBands/iStochastic/iCCI/iADX/iMFI/iOBV/iSAR/iStdDev/iWPR/iMomentum |
| MQL4 交易 | `tools/mql2go/interp/builtin_trade.go` | OrderSend/OrderClose/OrderModify/OrderDelete + 16 Order* 属性函数 |
| MQL5 CTrade | `tools/mql2go/interp/builtin_ctrade.go` | Buy/Sell/BuyLimit/SellLimit/BuyStop/SellStop/PositionClose/PositionModify/OrderDelete |
| MQL5 Position 查询 | `tools/mql2go/interp/builtin_trade.go` | PositionsTotal/PositionGetTicket/PositionGetDouble/Integer/String |
| 控制流 | `tools/mql2go/interp/exec.go` | if/else, for, while, do-while, switch, break, continue, return |
| 工具函数 | `tools/mql2go/interp/builtin_tools.go` | Math*, String*, Array*, Time*, Print, Sleep, NormalizeDouble |
| 定时器 | `tools/mql2go/interp/builtins.go` | EventSetTimer/EventKillTimer |

### 1.2 Stub 实现 ✅ 已全部实现（T1）

~~24 个指标在 `interp/builtin_indicators.go` 有 case 分支，但 `backtest/engine.go` 和 `runner/indicators.go` 中 `return decimal.Zero`~~

全部 24 个指标已实现真实计算算法。

### 1.3 未实现 → 已完成/永久盲区

| 原缺失项 | 原严重度 | 新状态 |
|----------|---------|--------|
| iCustom | 致命 | ⏳ 待规划（T6b，可行但需额外设计） |
| `*OnArray` 变体 | 致命 | ✅ 已实现（T6a: 8 个 OnArray 函数） |
| MQL5 原生 OrderSend | 致命 | 🔒 永久盲区（CTrade 封装已覆盖） |
| MQL5 事件回调 | 致命 | ✅ 已实现（T5: OnTrade/OnTradeTransaction） |
| File 系列 | 致命 | 🔒 永久盲区（系统有自己的日志/参数管理） |
| Object/Chart 系列 | 致命 | 🔒 永久盲区（MT 客户端 UI，服务端无意义） |
| SymbolInfo 系列 | 致命 | ✅ 已实现（T3） |
| MarketInfo | 致命 | ✅ 已实现（T3） |
| OrderCloseBy | 致命 | ✅ 已实现（T4） |
| OrdersHistoryTotal | 致命 | ✅ 已实现（T4） |
| DLL (#import) | 致命 | 🔒 永久盲区（安全风险，不支持） |
| SimBroker 缺口 | 警告 | ✅ 已实现（T2: PositionClosePartial/CloseBy/HistoryOrders/Deals/SymbolInfo） |

## 2. 实施计划

### 阶段一：执行正确性（P1）

#### T1: 24 个 stub 指标真实计算

- **文件**：`backend/strategy/backtest/engine.go` + `backend/strategy/runner/indicators.go`
- **方式**：每个指标在 SDK 接口已有方法定义，把 `return decimal.Zero` 替换为真实算法
- **数据源**：`BarSeries`（OHLCV），已有 14 个指标的实现可参考
- **算法参考**：

| 指标 | 算法要点 |
|------|---------|
| iAlligator | 3 条 SMMA（jaw=13偏移8, teeth=8偏移5, lips=5偏移3） |
| iIchimoku | Tenkan=(9期高+低)/2, Kijun=(26期高+低)/2, SenkouA=(T+K)/2偏移26, SenkouB=(52期高+低)/2偏移26 |
| iEnvelopes | 上下轨 = MA ± (MA × devPercent) |
| iDeMarker | (当前High-前High) / (|当前Low-前Low| + |当前High-前High|) 的 N 期均值 |
| iOsMA | MACD - Signal（MACD 柱状图） |
| iRVI | 类 RSI 但用 (Close-Open)/(High-Low) |
| iForce | (Close[i]-Close[i+1]) × Volume 的 MA |
| iFractals | 5 根 K 线分形：中间 High 最高 → 上分形 |
| iGator | Alligator 三线差的绝对值 |
| iAC | AO - AO 的 5 期 SMA |
| iAD | 累积/派发线：((Close-Low)-(High-Close))/(High-Low) × Volume 累加 |
| iAO | 5 期 SMA(median) - 34 期 SMA(median) |
| iBearsPower | Low - EMA(period) |
| iBullsPower | High - EMA(period) |
| iBWMFI | (High-Low)/Volume |
| iAMA | AMA = 价格 + SC × (价格 - AMA[prev])，SC = (ER×(2/3-1/3)+1/3)² |
| iDEMA | 2×EMA - EMA(EMA) |
| iTEMA | 3×EMA - 3×EMA(EMA) + EMA(EMA(EMA)) |
| iFrAMA | FRAMA = EMA(EMA, α)，α 由 FR 决定 |
| iVIDyA | 类 AMA 但用 CMO 做 SC |
| iTriX | TEMA(ROC) 的百分数变化 |
| iADXWilder | ADX 的 Wilder 平滑版 |
| iChaikin | (3 期 EMA(MFV)) 的 ROC，MFV = ((C-L)-(H-C))/(H-L)×V |
| iVolumes | Volume 的 MA 或原始 Volume |

- **验收**：单元测试验证每个指标在有数据时返回非零值；`go build ./...` 通过
- **工作量**：约 2 天

#### T2: SimBroker 功能补齐

- **文件**：`backend/strategy/backtest/broker.go`
- **任务**：

| 功能 | 实现方式 |
|------|---------|
| `PositionClosePartial` | 改造 `PositionClose`，当 volume < position volume 时部分平仓 |
| `PositionCloseBy` | 找反向持仓，净额结算，差额记入 balance |
| `HistoryOrders` | 维护 `closedPositions []*OrderRecord` 列表，平仓时追加 |
| `Deals` | 同上，平仓时生成 Deal 记录 |
| `SymbolInfo` | 从 Config 读取品种属性（digits/point/contract_size），替代硬编码 |

- **验收**：单元测试覆盖部分平仓、对冲平仓、历史查询；`go build ./...` 通过
- **工作量**：约 1 天

### 阶段二：EA 兼容性（P2）

#### T3: SymbolInfo / MarketInfo

- **文件**：`tools/mql2go/interp/builtins.go`（callMarketData switch）+ `tools/mql2go/interp/builtin_registry.go`
- **方式**：
  - `SymbolInfoDouble(symbol, prop)` → 从 `ctx.Broker().SymbolInfo()` 读取
  - `SymbolInfoInteger(symbol, prop)` → 同上
  - `SymbolInfoString(symbol, prop)` → 同上
  - `MarketInfo(symbol, mode)` → MQL4 等价映射
  - 在 `implementedMarketData` 列表注册新 case
- **验收**：解释器测试调用 SymbolInfoDouble 返回非零值；blind spot 报告不再标记
- **工作量**：约半天

#### T4: OrderCloseBy + OrdersHistoryTotal

- **文件**：`tools/mql2go/interp/builtin_trade.go` + `tools/mql2go/interp/builtin_registry.go` + `tools/mql2go/interp/pool.go`
- **方式**：
  - `OrderCloseBy(ticket1, ticket2)` → callTrade 增加 case，调用 `ctx.Broker().PositionCloseBy`
  - `OrdersHistoryTotal()` → MQL4OrderPool 维护历史列表
  - `OrderSelect(idx, SELECT_BY_POS, MODE_HISTORY)` → pool 支持 history 模式
- **验收**：单元测试覆盖对冲平仓 + 历史遍历
- **工作量**：约 1 天

### 阶段三：扩展支持（P3）

#### T5: MQL5 事件回调

- **文件**：
  - `tools/mql2go/interp/ir.go` — IR 增加 `OnTrade`/`OnTradeTransaction` 字段
  - `tools/mql2go/interp/exec.go` — 增加 `OnTrade`/`OnTradeTransaction` 方法
  - `tools/mql2go/compile_interp.go` — 编译器收集新函数
  - `internal/connect/strategy/vm_live_handlers.go` — trade 事件分发到 VM `OnTrade`
  - `backend/strategy/sdk/strategy.go` — SDK 增加可选 `TradeStrategy` 接口
- **验收**：MQL5 EA 含 OnTrade 可正确编译并执行
- **工作量**：约 1 天

#### T6: 永久盲区标记（拆分为 T6a/T6b/T6c）

经分析，原计划中的部分"永久盲区"实际可在服务端实现，因此拆分为三个子任务：

**T6a: `*OnArray` 指标支持** ✅ 已完成

`*OnArray` 变体通过 `arrayBarSource` 适配器实现——将用户数组映射为 `BarSource` 接口，复用已有指标算法。支持 8 个函数：iMAOnArray/iRSIOnArray/iATROnArray/iBandsOnArray/iStdDevOnArray/iMomentumOnArray/iCCIOnArray/iMACDOnArray。

**T6b: iCustom 自定义指标** ⏳ 待规划

MT 客户端自定义指标可通过解释执行 `OnCalculate()` 实现，需额外设计 buffer 模型和指标加载机制。不标记为永久盲区。

**T6c: 永久盲区标记** ✅ 已完成

在 `tools/mql2go/interp/analyze.go` 的 `classifySeverity` 中标记以下为永久盲区（severity: "永久盲区"）：

| 盲区 | 原因 |
|------|------|
| File I/O | EA 文件操作用于日志/参数，系统有自己的日志和参数管理 |
| Object/Chart | MT 客户端 UI 功能，服务端无意义 |
| DLL (#import) | 安全风险，不支持 |
| MQL5 原生 OrderSend | CTrade 封装已覆盖，底层原生调用不单独支持 |

- **工作量**：T6a 约 0.5 天，T6c 约 1 小时，T6b 待规划

## 3. 更新 ADR-0021 第 6 节

ADR-0021 第 6 节的待补齐清单已同步更新：

| 原条目 | 新状态 |
|--------|--------|
| `indicatorSet` stub 指标: Stochastic/CCI/ADX/MFI/OBV/SAR/StdDev/WPR | ✅ 已实现（T1: 全部 38 个指标真实计算） |
| Go SimBroker 功能补齐 | ✅ 已实现（T2: PositionClosePartial/CloseBy/HistoryOrders/Deals/SymbolInfo） |
| `GoStrategyExecutor` 完整实现 | ✅ 已实现（Bytecode VM 路径替代，`VMRunner` 实现 `sdk.Strategy`） |
| `mql2go` MQL5 支持 | ✅ 已完成（CTrade/Position/OnTrade/OnTradeTransaction） |
| Go 行为对齐 harness | ✅ 已实现（`parity.go` + `parity_runner.go` + `mt_report.go`） |
| `mql2go` 表达式翻译 | ✅ 已完成（VM 路径直接执行 AST，不再需要表达式翻译） |
| Go 回测 metrics | ✅ 已完成 |
| SymbolInfo/MarketInfo | ✅ 已实现（T3） |
| OrderCloseBy + OrdersHistoryTotal | ✅ 已实现（T4） |
| MQL5 事件回调 | ✅ 已实现（T5） |
| `*OnArray` 指标 | ✅ 已实现（T6a: 8 个 OnArray 函数） |
| 永久盲区标记 | ✅ 已实现（T6c: File/Object/DLL/NativeOrderSend） |
| iCustom 自定义指标 | ⏳ 待规划（T6b） |

## 4. 时间线

| 阶段 | 任务 | 工作量 | 依赖 |
|------|------|--------|------|
| 一 | T1: 24 个 stub 指标 | 2 天 | 无 |
| 一 | T2: SimBroker 补齐 | 1 天 | 无 |
| 二 | T3: SymbolInfo/MarketInfo | 0.5 天 | T2（SymbolInfo 从 Config 读取） |
| 二 | T4: OrderCloseBy + History | 1 天 | T2（SimBroker PositionCloseBy） |
| 三 | T5: MQL5 事件回调 | 1 天 | 无 |
| 三 | T6a: `*OnArray` 指标支持 | 0.5 天 | 无 |
| 三 | T6b: iCustom 自定义指标 | 待规划 | 无 |
| 三 | T6c: 永久盲区标记 | 0.1 天 | 无 |
| **合计** | | **6.1 天** | |

## 5. 验收标准

- [x] `go build ./...` 通过
- [x] `go test ./tools/mql2go/... ./strategy/backtest/... ./strategy/runner/...` 通过
- [x] 24 个 stub 指标在有数据时返回非零值
- [x] SimBroker 支持部分平仓、对冲平仓、历史查询
- [x] SymbolInfoDouble/Integer/String 返回正确值
- [x] OrderCloseBy 在解释器中可调用
- [x] MQL5 EA 含 OnTrade 可正确编译并执行
- [x] 永久盲区在分析报告中标记为"永久盲区"而非"致命"
- [x] ADR-0021 第 6 节状态更新
