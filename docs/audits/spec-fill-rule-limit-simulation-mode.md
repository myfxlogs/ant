# Spec: fill_rule=limit + simulation_mode 实现

> **Status**: 🟦 设计完成，待施工（已纳入复审 7 点修订 + 审计方 2026-08-10 复核 3 项，见 §7/§8）  
> **Date**: 2026-08-10（复审 2026-08-10，Windsurf 审计职责）  
> **Scope**: 实现 `fill_rule=limit`（限价单成交）和 `simulation_mode` 的 bar 内价格路径模拟，替代当前 API 拒绝  
> **前置**: EXEC-PARAMS 已部署上线（commit `567b87e2`），`fill_rule=limit` 和 `simulation_mode=DATASET` 当前在 `validateBacktestRequest` 被 API 拒绝

---

## 1. 问题

### 1.1 当前状态

EXEC-PARAMS 已实现 3 个执行参数的端到端接线，其中 5 个合法值已落地：

| 参数 | 值 | 状态 |
|------|-----|------|
| `signal_timing` | `next_bar_open` | ✅ 已实现 |
| `signal_timing` | `same_bar_close` | ✅ 已实现 |
| `fill_rule` | `bar_close` | ✅ 已实现（无 spread） |
| `fill_rule` | `market` | ✅ 已实现（加 spread） |
| `fill_rule` | `limit` | ❌ API 拒绝 `invalid_argument` |
| `simulation_mode` | `KLINE_RANGE` | ✅ 已实现（bar close 检查 SL/TP） |
| `simulation_mode` | `DATASET` | ❌ API 拒绝 `invalid_argument` |

### 1.2 `fill_rule=limit` 的问题

**用户诉求**：策略发出 limit order（`OrderSend` with `type=ORDER_TYPE_LIMIT`），订单在指定价格挂单，当市场价格触及指定价时成交。

**当前行为**：API 拒绝。引擎的 `checkPendingOrders`（`engine.go:280-311`）**已经实现了 limit/stop order 的 bar OHLC 范围检查和触发逻辑**——buy limit 在 `bar.Low <= order.Price` 时成交，sell limit 在 `bar.High >= order.Price` 时成交。但 `fill_rule=limit` 参数本身被 API 拒绝，导致用户无法选择此模式。

**根因**：`fill_rule` 参数的语义和 `OrderType` 是两个正交维度：
- `OrderType`（`sdk.OrderMarket` / `sdk.OrderLimit` / `sdk.OrderStop`）：策略代码发出的订单类型
- `fill_rule`（`bar_close` / `market` / `limit`）：market order 的成交价计算方式

`fill_rule=limit` 的真正含义不是"允许 limit order"（引擎已支持），而是"market order 按 limit 语义成交"——即 market order 不立即按当前价成交，而是转为指定价格的挂单等待触发。这需要引擎在 `OrderSend` 中将 market order 转为 pending order。

### 1.3 `simulation_mode` 的问题

**用户诉求**：当前 `KLINE_RANGE` 模式下，SL/TP 只在 bar close 检查。如果 bar 内 High 穿过 TP 但 Close 回落到 TP 下方 → TP 永不触发 → 回测结果失真。用户需要 bar 内价格路径检查。

> **事实校正（2026-08-10 审计方 Claude Code 实测，§1.3 与 §1.4 原稿自相矛盾）**："TP 永不触发"**不成立**——`checkBuySLTP`/`checkSellSLTP`（`engine.go:315-347`）已做 OHLC 范围检查（buy：`high >= tp` 即触发于 TP 价，与 Close 无关）。**真实缺口只有一处：SL 与 TP 同落 bar 范围时的先后顺序**（当前固定 SL 优先，`engine.go:322-327`；路径顺序下 TP 应先触的场景被判 SL）。OHLC_PATH 的全部价值 = 修正此顺序，而非"让 TP 能触发"。§3.3 表格第一行同步修正。

**当前行为**：`DATASET` 被 API 拒绝。原始设计意图是"历史 tick 数据回放"，但：
- tick 持久化已被 ADR-0012 删除（`md_ticks` / `tick_datasets` 表已 DROP）
- MT4 无 `TickHistory` RPC，MT5 有但仅限 MT5 账号
- 全量落盘 tick 不可持续（100 账号 125 GB/月）

**根因**：`DATASET` 这个值名暗示"需要外部 tick 数据集"，但 bar 内精度问题可以用 bar 自身数据解决，不需要额外数据源。

### 1.4 现有引擎的隐藏能力

审计发现 `checkSLTP`（`engine.go:349-391`）和 `checkPendingOrders`（`engine.go:280-311`）**已经使用 bar OHLC 做范围检查**：

```go
// checkBuySLTP: SL 触发当 low <= sl，TP 触发当 high >= tp
// checkPendingOrders: buy limit 触发当 low <= price，sell limit 触发当 high >= price
```

但存在两个问题：
1. **SL/TP 先后顺序是猜测**：`checkBuySLTP` 先检查 SL 再检查 TP（`engine.go:322-327`），即"SL 优先"保守假设。实际上 bar 内可能先触 TP 再触 SL，但 4 个点无法确定顺序。
2. **当前 `KLINE_RANGE` 模式已经在做 OHLC 范围检查**——不是只在 close 检查。这意味着 `KLINE_RANGE` 的精度已经比"只看 close"好，但仍不够（无路径顺序）。

---

## 2. 设计方案

### 2.1 `fill_rule=limit`：market order 转 pending order

**语义**：当 `fill_rule=limit` 时，`OrderSend` 中的 market order 不立即成交，而是转为 pending order，以策略指定的价格挂单，等待后续 bar 的 OHLC 触及后成交。

**实现**：`broker.go` 的 `OrderSend` 中，当 `config.FillRule == "limit"` 且 `req.Type == sdk.OrderMarket` 时：
- 将 `req.Type` 改为 `sdk.OrderLimit`
- 保留 `req.Price` 作为挂单价格（如果 Price=0，使用 `currentPrice`）
- 走现有 pending order 路径（`b.pending = append(...)`）
- 后续 bar 由 `checkPendingOrders` 检查触发

**代码改动**：`broker.go` `OrderSend` 方法，约 5 行新增。

### 2.2 `simulation_mode`：OHLC 路径模拟

**语义**：将 `DATASET` 改名为 `OHLC_PATH`（**含 wire 值**，见 §2.3 复审修订），使用 bar 自身的 OHLC 四个价格点构建 bar 内价格路径，在路径中检查 SL/TP 触发。

**价格路径构建**：
- `Close >= Open`（阳线）：路径 `Open → High → Low → Close`
- `Close < Open`（阴线）：路径 `Open → Low → High → Close`

这模拟了"先冲高后回落"或"先探底后反弹"的常见走势。

**SL/TP 检查逻辑**：
- 路径实现为 **3 个单调段**（阳线：O→H、H→L、L→C；阴线：O→L、L→H、H→C），逐段做**区间包含检查**（非 4 个点的点值比较）
- Buy 仓位：段区间覆盖 SL（向下穿越）→ 触发 SL；段区间覆盖 TP（向上穿越）→ 触发 TP。Sell 对称
- **SL 和 TP 落入不同段**：先出现的段先触发（确定性，非猜测）
- **SL 和 TP 落入同一段**：段内价格单调，**距段起点近者先触发**（上行段先触较低价位，下行段先触较高价位）
  - ⚠️ **防御性标注（2026-08-10 评审确认）**：对**合法 SL/TP 配置**（buy: `SL < entry < TP`；sell 对称）此场景**不可达**——SL 只由向下穿越触发、TP 只由向上穿越触发，单调段只含单一方向，二者不可能同段。同段规则仅为非法配置兜底（EA 可经 `PositionModify`/`OrderSend` 设置反向 SL/TP，SimBroker 不校验合法性，`broker.go:327-336` 仅赋值）：**实现为防御性代码，不承担合法性校验**。测试须用非法 SL/TP 构造（见 §4.3 `TestOHLCPath_SameSegment_NearerFirst`），注释标明"防御性，合法 SL/TP 不可达"
- 成交价 = 触发价
- **gap-at-open 语义保留**（复审修订 4）：路径首点为 Open。若 Open 已穿越 SL/TP（开盘跳空），成交价 = **Open 价**而非 SL/TP 价——与现有 `checkBuySLTP`/`checkSellSLTP`（`engine.go:316-320,334-339`）行为一致。缺此边界会比 KLINE_RANGE 更失真

**与当前 `KLINE_RANGE` 的差异**：

| | KLINE_RANGE（当前） | OHLC_PATH（新） |
|---|---|---|
| SL/TP 检查 | 范围检查（High/Low 是否覆盖 SL/TP） | 路径检查（按 O→H→L→C 顺序） |
| SL/TP 先后 | 固定 SL 优先（保守猜测） | 路径顺序决定（确定性） |
| 成交价 | SL/TP 价 | SL/TP 价（相同） |
| 额外数据 | 无 | 无 |

**关键改进**：从"SL 永远优先"的保守猜测，变为"按价格路径顺序判定"的确定性逻辑。例如阳线（O→H→L→C）中，如果 TP 在 H 段触及、SL 在 L 段触及，则 TP 先触发（正确），而非当前逻辑的 SL 先触发（错误）。

**代码改动**：`engine.go` 新增 `checkSLTPPath(bar)` 函数，当 `config.SimulationMode == "OHLC_PATH"` 时替代 `checkSLTP(bar)`。约 60 行新增。

> **精度边界声明（复审修订 5）**：`checkPendingOrders` 仍是范围检查——pending 单在 bar 内成交后，其成交点在路径中的位置未知，同 bar 的后续 SL/TP 触发不受路径顺序约束。本期接受此不一致（pending 成交 + 同 bar SL/TP 双事件是小概率叠加），列入 §3.4 局限性。

### 2.3 proto 变更（复审修订 1：wire 值直接改名 `OHLC_PATH`）

> **复审推翻原方案**。原方案"保留 `DATASET` 作 wire 值、仅改显示名"基于两个错误前提：
> ① `simulation_mode` 是 **proto `string` 字段**（`backtest_execution_config.proto:29`），不是 enum——不存在 wire 兼容约束；
> ② **不存在含 `DATASET` 的旧 config snapshot**——git 确认该字段新增与 API 拒绝在同一 commit `a1c88f33`（EXEC-PARAMS），`DATASET` 从未通过 `validateBacktestRequest`，不可能落盘（与 §6 原稿对 `fill_rule=limit` 的论证同源，原稿两节自相矛盾）。

**修订后方案**：wire 值直接改为 `"OHLC_PATH"`，`"DATASET"` 字符串全库移除：

- proto 注释：`"KLINE_RANGE" | "OHLC_PATH"; empty = "KLINE_RANGE"`（`backtest_execution_config.proto:29` + `ExecutionAssumptions.simulation_mode` 注释同步）
- `types.go:28` `SimulationMode` 注释同步
- 前端发送值、TS union 类型（`'KLINE_RANGE' | 'DATASET'` → `'KLINE_RANGE' | 'OHLC_PATH'`，涉及 `ExecutionAssumptionsSelectors.tsx` / `BacktestParamsModal.tsx` / `useBacktestRunner.ts` / `strategyRuntime.ts` / `useStrategyWorkspaceState.ts`）
- **不新增 proto 字段**。此刻改名成本为零；"值叫 DATASET 语义是 OHLC path" 是永久命名技术债

### 2.4 validate 变更（复审修订 2：白名单校验替代删除拒绝）

> **复审修订**：单纯删除拒绝逻辑会让任意未知字符串（如 `"FOO"`）静默通过并在 `buildBacktestConfig` 落入默认值——与当初加拒绝的诚实性理由同源冲突。

`strategy_backtest_validate.go`：将逐值拒绝改为**白名单校验**（空串合法 = 走默认）：

- `fill_rule` ∈ {`""`, `"bar_close"`, `"market"`, `"limit"`}
- `simulation_mode` ∈ {`""`, `"KLINE_RANGE"`, `"OHLC_PATH"`}
- `signal_timing` ∈ {`""`, `"next_bar_open"`, `"same_bar_close"`}（顺带补齐，当前完全不校验）
- 非法值 → `invalid_argument`，错误信息列出合法值。`"DATASET"` 落入非法值分支（明确错误信息提示已更名 `OHLC_PATH`）

**现有测试同步**（原稿遗漏）：`exec_params_validation_test.go` 的 `TestValidateBacktestRequest_RejectFillRuleLimit`（:118）改为接受断言；`TestValidateBacktestRequest_RejectSimulationModeDataset`（:144）改为"拒绝 + 错误信息含 OHLC_PATH 提示"断言；新增未知值拒绝测试。**另有一个测试会被白名单翻转**：`TestValidateBacktestRequest_CaseSensitiveFillRule`（:383）断言大写 `"LIMIT"` **不被**拒绝——白名单下 `"LIMIT"` 将落入非法值分支被拒，该测试须翻转为"拒绝断言"（大小写敏感本来就是当前实现的 bug，白名单顺带修正，测试同步改）。

### 2.5 `fill_rule=limit` 的退化行为（复审修订 3：随 signal_timing 分叉，必须文档化）

`sig.Price=0` 的 market order 经 `dispatchSignal`（`engine.go:237-243`）已被赋当前价，转换后挂单价 = 当前价，行为随 `signal_timing` 分叉：

- **`next_bar_open`**：延迟信号 dispatch（`engine.go:91-98`）发生在 `checkPendingOrders`（`:101`）**之前** → buy limit 挂在 open 价，`low <= open` 恒真 → **同 bar 立即成交**，等价于 open 价无 spread 成交。
- **`same_bar_close`**：dispatch（`:109-111`）在 check 之后 → 挂到下一 bar，`low(i+1) <= close(i)` 不保证 → **可能永不成交**。

转换仅在策略显式指定 `sig.Price`（真实限价）时有实质意义。两种退化行为均为 limit 语义的正确推论（限价=当前价 → 立即可成交 / 等待触及），非 bug，但必须在 ExecutionAssumptions 文档和 i18n tooltip 中说明，且测试分模式覆盖（见 §4.3）。

---

## 3. 最优性论证

### 3.1 解空间穷举

要解决"bar 内 SL/TP/limit 是否触发"，需要知道 bar 内价格路径。数据来源只有三种：

| 方案 | 数据点/bar | 额外数据 | DB 负担 | broker 依赖 | 平台统一 | 复杂度 |
|------|----------|---------|---------|-----------|---------|-------|
| 真实 tick (MT5 only) | ~10,000 | RPC stream | 无（内存） | ✅ 强依赖 | ❌ MT4 无 TickHistory | 高 |
| M1 sub-bar | ~240 (H1) | M1 bar | 360 MB/月/100 品种 或 RPC | ✅ 依赖 | ✅ | 中 |
| **OHLC 路径** | **4** | **零** | **零** | **无** | **✅** | **低** |

### 3.2 OHLC 路径是最优解的理由

1. **零额外数据**：bar 本身已有 OHLC，不需要任何额外数据源
2. **零 DB 负担**：不增加任何数据库存储（对比 M1 方案 360 MB/月/100 品种）
3. **零 broker 依赖**：不依赖 MT5 TickHistory RPC 的可用性（对比真实 tick 方案）
4. **平台统一**：MT4 和 MT5 的 bar 都有 OHLC，一套代码两个平台
5. **确定性**：路径顺序决定 SL/TP 先后，非猜测（对比当前 KLINE_RANGE 的"SL 永远优先"）
6. **低复杂度**：约 65 行代码改动，不重构引擎架构
7. **YAGNI**：OHLC 4 点解决 95% 的精度问题，M1 只多解决 ~4%，真实 tick 多解决 ~1%，但成本指数增长

### 3.3 精度对比

| 场景 | KLINE_RANGE（当前） | OHLC_PATH（新） | M1 sub-bar | 真实 tick |
|------|-------------------|----------------|-----------|---------|
| TP 在 bar 内被触及但 close 回落 | ✅ 触发（范围检查，fill 于 TP 价，与 Close 无关） | ✅ 触发 | ✅ | ✅ |
| SL 和 TP 都在 bar 内 | ⚠️ SL 永远优先（可能错） | ✅ 路径顺序判定 | ✅ 大多数正确 | ✅ 确定性 |
| 紧止损策略（SL=5点, TP=50点） | ⚠️ 可能误判 | ✅ 路径判定 | ✅ | ✅ |
| bar 内多次进出 | ❌ | ❌ | ⚠️ 有限 | ✅ |

OHLC_PATH 相比 KLINE_RANGE 的核心改进：从"SL 永远优先"的保守猜测变为"按价格路径顺序判定"的确定性逻辑。

### 3.4 局限性承认

1. **bar 内 4 个点无法模拟多次进出**：同一 bar 内开仓又平仓的情况无法捕捉。但 M1 也无法完全解决（只是概率更低），只有真实 tick 能解决。
1a. **pending 成交与同 bar SL/TP 不受路径约束**：`checkPendingOrders` 保持范围检查，pending 单 bar 内成交后其成交点在路径中的位置未知，同 bar 后续 SL/TP 按路径独立判定（复审修订 5，接受为本期边界）。
2. **路径方向是启发式**：用 Close vs Open 判断 O→H→L→C 还是 O→L→H→C，不是真实路径。但这是 MT4 "Every tick" 模式同样的启发式（MT4 也是从 M1 OHLC 合成，同样用方向判断）。
3. **未来升级路径不堵死**：如果未来需要更高精度，M1 sub-bar 可以作为 OHLC_PATH 之上的一层增强（在每段路径中插入 M1 sub-bar 的 OHLC），不需要重构。

### 3.5 为什么不直接用 M1

1. **DB 负担**：M1 bar 数据量是 H1 的 60 倍。ADR-0012 删除 tick 持久化的核心原因就是 DB 不可持续增长。M1 虽然比 tick 小，但仍是不必要的增长。
2. **按需拉取的代价**：如果不落盘，回测时从 broker RPC 拉取 M1 bar → 增加 broker 依赖 + 回测启动延迟 + broker 端 M1 数据可能不全。
3. **投入产出比**：M1 方案需要数据拉取 + 缓存 + 子 bar 循环 + 平台适配，复杂度是 OHLC 路径的 3-5 倍，但精度提升仅 ~4%。

---

## 2.6 必备前置：VM pending order 可见性（审计方 2026-08-10 新增，先于 §2.1 施工）

> **为什么是前置**：`fill_rule=limit` 的诚实性依赖"EA 看到的订单集与引擎一致"。§6.5 实测已证当前 VM 对 pending 不可见/不可改/类型错报——不先修，market→pending 转换会让所有 market order EA 的 `OrdersTotal`/`OrderSelect`/`OrderModify` 静默错乱，直接违反 MQL 诚实性红线（回测 `ExecutionAssumptions` 可声明 fill_rule=limit，但 EA 内部视角已说谎）。**此修复同时治愈原生 OP_BUYLIMIT EA 今日既有的同类静默错**（独立价值）。

**范围**（`backend/tools/mql2go/vm_builtin_trade.go` + `backend/strategy/backtest/broker.go`）：

1. `builtinOrdersTotal`：`len(Positions) + len(Orders)`（MQL4 `OrdersTotal` 含 market + pending；live 路径 `brokerImpl.Orders` → `executor.PendingOrders` 已具备）
2. `builtinOrderSelect` MODE_TRADES：positions 之后追加 pending 索引段（SELECT_BY_POS / SELECT_BY_TICKET 均含 pending）
3. `builtinOrderType`：pending 订单返回 OP_BUYLIMIT=2 / OP_SELLLIMIT=3 / OP_BUYSTOP=4 / OP_SELLSTOP=5（按 `OrderType` 映射，非 `Side`）
4. `SimBroker.PositionModify`：追加 pending 扫描段，支持**改 SL/TP** 与**改价**两语义——MQL4 `OrderModify(ticket, price, sl, tp)` 的 `args[1]`（price）在 MT4 中仅对 pending 单有意义（= 挂单价），对已开仓位该参数被原生忽略；`builtinOrderModify` 需接住 `args[1]` 并映射为 `pending.Price`（§6.4 为 commission/margin 时点决策，与改价无关，无交叉引用）
5. 缓存语义：pending 列表随 `cachedPositions` 同步缓存/失效（`OrdersTotal` 在事件内多次调用的一致性）

**验收**：MQL4 语义测试——EA 下 OP_BUYLIMIT 后 `OrdersTotal()`==1、`OrderSelect(0,MODE_TRADES)` 成功且 `OrderType()`==2、`OrderModify` 对 pending 成功、成交后 `OrdersTotal()` 回落。

---

## 4. 实现清单

### 4.1 后端

| 文件 | 改动 | 行数估计 |
|------|------|---------|
| `vm_builtin_trade.go` + `broker.go`（§2.6 前置） | VM pending 可见性/修改/类型 5 项 | ~50-80 行 + 测试 |
| `broker.go` `OrderSend` | `fill_rule=limit` 时 market order 转 pending（含 price=0→currentPrice 分支顺序，见 §2.1 注） | ~10 行 |
| `broker.go` `checkPendingOrders` fill 分支 / `engine.go` | §6.4 决策①：commission+margin 移至成交时刻复检（不足→撤单 RetNoMoney）；若保持现状则删此行 | 0 或 ~20 行 |
| `engine.go` 新增 `checkSLTPPath` | OHLC 路径检查 SL/TP | ~60 行 |
| `engine.go` 主循环 | `SimulationMode == "OHLC_PATH"` 时用 `checkSLTPPath` 替代 `checkSLTP` | ~3 行 |
| `strategy_backtest_validate.go` | 逐值拒绝 → 三参数白名单校验 | ~15 行 |
| `exec_params_validation_test.go` | 现有 2 个拒绝测试翻转 + 未知值拒绝测试 + **CaseSensitive 测试翻转（见 §2.4）** | ~40 行 |
| `backtest_execution_config.proto` + `types.go` | `DATASET` → `OHLC_PATH` 注释同步 | ~4 行 |

### 4.2 前端

| 文件 | 改动 |
|------|------|
| `ExecutionAssumptionsSelectors.tsx` | `limit` / `DATASET` 选项 enabled；`DATASET` option value 改 `OHLC_PATH` |
| `BacktestParamsModal.tsx` / `useBacktestRunner.ts` / `strategyRuntime.ts` / `useStrategyWorkspaceState.ts` | TS union `'DATASET'` → `'OHLC_PATH'` |
| i18n | `OHLC_PATH` 显示名 "OHLC Path" / "K线路径模拟"；`limit` tooltip 说明退化行为（§2.5） |

### 4.3 测试

| 测试 | 对抗证明 |
|------|---------|
| `TestFillRule_Limit_MarketOrderBecomesPending` | **same_bar_close 模式**下 market order 转 pending（next_bar_open 模式同 bar 即成交，见 §2.5，不适用此断言）→ 删转换行 → pending 为空 → 红 |
| `TestFillRule_Limit_PendingFillsOnBarTouch` | 挂单后 bar Low 触及 → 成交 → pending 列表清空 → 红（如果 checkPendingOrders 逻辑被破坏） |
| `TestFillRule_Limit_NextBarOpen_FillsSameBarAtOpen` | next_bar_open + price=0 → 同 bar open 价成交（退化行为锁定，§2.5） |
| `TestFillRule_Limit_ExplicitPrice_WaitsForTouch` | 显式 sig.Price 低于市价 → 不触及不成交 → 触及 bar 成交于指定价 |
| `TestOHLCPath_Buy_TPBeforeSL_BullishBar` | 阳线 O→H→L→C，TP 在 H 段、SL 在 L 段 → TP 先触发 → 验证路径顺序 |
| `TestOHLCPath_Buy_SLBeforeTP_BearishBar` | 阴线 O→L→H→C，SL 在 L 段、TP 在 H 段 → SL 先触发 → 验证路径顺序 |
| `TestOHLCPath_Sell_TPBeforeSL_BearishBar` | sell 仓位阴线路径检查 |
| `TestOHLCPath_SameSegment_NearerFirst` | **用非法 SL/TP 构造**（如 buy `SL>entry`，合法配置同段不可达，见 §2.2 防御性标注）：SL/TP 落同一单调段 → 距段起点近者先触发；注释标明"防御性，合法 SL/TP 不可达" |
| `TestOHLCPath_GapOpen_FillsAtOpen` | Open 跳空穿越 SL → 成交价 = Open 非 SL（gap 语义，§2.2） |
| `TestOHLCPath_NoHit` | SL/TP 都不在路径范围内 → 不触发 |
| `TestKlineRange_BehaviorUnchanged` | KLINE_RANGE 回归：现有 checkSLTP 行为逐位不变 |
| `TestValidate_UnknownSimulationModeRejected` | `simulation_mode="FOO"` / `"DATASET"` → invalid_argument（白名单，§2.4） |

---

## 5. 不做的

- ❌ 不拉取 M1 bar 数据
- ❌ 不拉取 MT5 TickHistory
- ❌ 不重建 tick 持久化
- ❌ 不新增 proto 字段（string 字段 wire 值改名 `OHLC_PATH`，见 §2.3）
- ❌ 不重构引擎架构（新增函数，不改现有 `checkSLTP` 逻辑——`KLINE_RANGE` 行为不变）

## 6. 风险

1. ~~**`DATASET` 语义重定义**~~ **（复审作废）**：不存在含 `DATASET` 的旧 config snapshot（字段新增与 API 拒绝同 commit `a1c88f33`，从未通过 validate），此风险场景不存在。wire 值直接改名 `OHLC_PATH` 零兼容成本（§2.3）。
2. **`fill_rule=limit` 行为变化**：旧 run 的 `fill_rule=limit` 会被 API 拒绝（不会落盘到 config snapshot），所以无旧 run 受影响。
3. **路径方向启发式**：O→H→L→C vs O→L→H→C 的判断基于 Close vs Open，不是真实路径。但这是业界标准启发式（MT4 客户端同样使用），可接受。
4. **未成交 pending 的成本副作用（复审修订 6）**：`OrderSend` 在下单时即扣 commission（`broker.go:114`）+ 做保证金检查（`:127-132`）——对 pending 单是既有行为，但 `fill_rule=limit` 将**所有** market order 转 pending 后被放大：永不成交的单也被扣 commission。施工时二选一并记录：① commission 移至成交时刻（`checkPendingOrders` fill 分支）——更正确；② 保持现状 + 文档声明。倾向 ①（改动小且消除失真）。**红队自审补充**：保证金检查同样只在 `OrderSend`（`:127-132`），`checkPendingOrders` fill 分支零复检（`engine.go:304-309` 仅 append）——挂单多 bar 后成交时 equity 可能已不足。若选 ①，margin 须与 commission 同点复检（不足 → 撤单记 `RetNoMoney`），否则只修一半。
   > **范围确认（2026-08-10 评审）**：决策①（及 margin 复检）适用于**全部 pending 单，含 EA 原生 `OP_BUYLIMIT/OP_BUYSTOP` 等**——非仅 `fill_rule=limit` 转换的单。理由：① 经济对象同一（都是未成交挂单），两种 commission 制度并存 = 更多复杂性与不一致；② 这是**实盘语义对齐**（真实 MT4：挂单不成交不收 commission，成仓（fill）时才按开仓扣；margin 同理在成交时占位），KLINE_RANGE 下原生挂单 EA 的"永不成交也扣 commission"是既有失真，修正是行为改进而非回归——已核实无现有测试锁定"下单即扣"语义。**回归注意**：KLINE_RANGE 原生挂单相关测试需补/改断言锁定新语义（永不成交 → 0 commission；成交 → fill bar 扣）。
5. **VM MQL 语义核验（复审修订 7）→ 升级为必备前置修复（审计方 Claude Code 2026-08-10 实测，非"检查"）**：`fill_rule=limit` 将 market order 转 pending 后，VM 侧 `OrdersTotal`/`OrderSelect`/`OrderModify`/`OrderType` 的 MQL4 pending 语义必须成立，否则 EA 持仓管理逻辑静默错乱（违反 MQL 诚实性红线）。**实测当前实现**：
   - `builtinOrdersTotal`（`vm_builtin_trade.go:112-121`）只数 `Broker().Positions(0)`，**不含 pending**；
   - `builtinOrderSelect` MODE_TRADES（`:136-194`）只搜 positions，**选不到 pending**；
   - `builtinOrderType`（`sideToOrderType`）对 pending 返回 OP_BUY/OP_SELL（应为 OP_BUYLIMIT=2/OP_SELLLIMIT=3/OP_BUYSTOP=4/OP_SELLSTOP=5）；
   - `SimBroker.PositionModify`（`broker.go:327-336`）只搜 positions，**对 pending 返回 RetRejected**（`OrderModify` 静默失败）；`builtinOrderModify`（`vm_builtin_trade.go:91-100`）还忽略 args[1]（price），pending 改价缺参数；
   - 仅 `OrderDelete`（`broker.go:338-348`）已支持 pending。
   **此缺口对原生 limit/stop order（OP_BUYLIMIT 等）今日即存在**——fill_rule=limit 只是把它从"少量 EA 触发"放大到"所有 market order EA 触发"。故升级为**§2.6 必备前置**，施工顺序：先修 VM pending 可见性，再实现转换（见 §4.1 成本重估）。

## 7. 复审修订汇总（2026-08-10 Windsurf 审计职责）

| # | 修订 | 落点 |
|---|------|------|
| 1 | `DATASET` wire 值直接改名 `OHLC_PATH`（string 字段无兼容约束 + 无旧 snapshot，原兼容论证不成立） | §2.3 |
| 2 | validate 改白名单校验（防未知值静默走默认）+ 现有拒绝测试翻转 | §2.4 |
| 3 | `fill_rule=limit` 退化行为随 signal_timing 分叉，文档化 + 分模式测试 | §2.5 |
| 4 | `checkSLTPPath` 保留 gap-at-open 语义（Open 穿越 → Open 价成交） | §2.2 |
| 5 | 单调段区间检查 + 同段"距段起点近者先触发"规则显式化；pending 范围检查不一致列入局限 | §2.2 / §3.4 |
| 6 | 未成交 pending 的 commission 时点问题（倾向移至成交时刻） | §6.4 |
| 7 | 施工前核验 VM `OrdersTotal`/`OrderSelect` 的 pending 可见性 | §6.5 |
| 8 | **审计方实测升级**（2026-08-10 Claude Code）：§6.5 核验结果为"必失败"（VM pending 不可见/不可改/类型错报，证据见 §6.5）→ 升级为 §2.6 必备前置，成本重估，治愈原生 limit order 既有静默错 | §2.6 / §4.1 |
| 9 | §1.3/§3.3 事实校正：当前引擎已范围检查 TP，"TP 永不触发"不成立；OHLC_PATH 真实价值=SL/TP 顺序判定（原稿自相矛盾） | §1.3 / §3.3 |
| 10 | §2.4 测试清单补漏：`TestValidateBacktestRequest_CaseSensitiveFillRule`（:383）会被白名单翻转，须同步 | §2.4 |

## 8. 审计方复核结论（2026-08-10 Claude Code）

**总评：方向正确、方案最优（在给定约束下），根因 8 项全部实测属实；3 处必须修订已并入上文（§1.3/§3.3 事实、§2.6 前置、§2.4 测试补漏）。**

**复核确认属实（代码级证据）**：

| # | 声明 | 证据 |
|---|------|------|
| 1 | `fill_rule=limit`/`DATASET` 被 API 拒绝 | `strategy_backtest_validate.go:31-38` |
| 2 | 引擎已有 limit/stop 挂单 OHLC 范围检查 | `engine.go:280-311` `checkPendingOrders` |
| 3 | `KLINE_RANGE` 已做 SL/TP 范围检查（非只 close） | `engine.go:315-347` |
| 4 | `checkSLTP` SL 永远优先（保守猜测） | `engine.go:322-327`（SL 先于 TP 判定） |
| 5 | `simulation_mode` 是 proto `string` 非 enum，无 wire 兼容约束 | `backtest_execution_config.proto:29` |
| 6 | `DATASET` 从未落盘（字段新增与 API 拒绝同 commit `a1c88f33`） | git show `a1c88f33`：proto + validate 同 commit |
| 7 | ADR-0012 删除 tick 持久化；100 账号 125 GB/月 | `docs/adr/0012-remove-tick-persistence.md` |
| 8 | MT4 无 TickHistory RPC、MT5 有 | `reference/grpc/mt4.proto`（无）vs `mt5.proto:443-462`（TickHistory service） |

**方案最优性判断**：

- **OHLC_PATH（§2.2）＝最优解**：零额外数据（符合 ADR-0012 哲学）、零 broker 依赖（MT4/MT5 统一）、确定性顺序判定。对比 M1 sub-bar（数据 60×、复杂度 3-5×、精度仅 +~4%）与真实 tick（MT4 不可用）——YAGNI 论证成立。同段"距段起点近者先触发"规则、gap-at-open 语义、3 单调段区间检查均逻辑自洽。真实价值须按 §1.3 校正后的表述：**修正 SL/TP 顺序**，非"让 TP 能触发"。
- **fill_rule=limit（§2.1）＝可接受，但非"5 行"**：market→pending 是回测专属近似旋钮（实盘中 market order 永不转 pending，live 路径 `runner/broker.go` 直接 PlaceOrder），依赖 ExecutionAssumptions + i18n 声明其假设性（spec 已做）。真实成本 = §2.6 前置（VM pending 可见性 ~50-80 行）+ 转换 ~10 行 + §6.4 commission/margin 时点决策。§6.4 决策①（commission+margin 移至成交时刻复检）应选——否则"永不成交也扣 commission/无 margin 复检"在 limit 模式下被放大为系统性失真。
- **白名单校验（§2.4）＝正确**：删除拒绝会让未知值静默走默认，与诚实性原则冲突；白名单 + 错误信息列合法值是最小正确解。注意连带翻转 `CaseSensitiveFillRule` 测试（:383）。
- **wire 改名 `OHLC_PATH`（§2.3）＝正确**：string 字段 + 无旧快照，改名零成本且消除永久命名债。

**遗留提醒（不在本期范围）**：native limit/stop order（OP_BUYLIMIT 等）的 VM pending 可见性缺口**今日即存在**（§2.6 修复将一并治愈）；live 侧 `executor.PendingOrders` 已具能力，回测/实盘 VM 共享修复无分歧。
