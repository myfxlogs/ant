# Builder Handoff — LIVE-HARNESS-PARITY（实盘 harness 注入面补全）

> 批次名：LIVE-HARNESS-PARITY ｜ 日期：2026-08-15 ｜ 审计方：Claude Code ｜ 施工方：Windsurf
> 涉及功能块：`strategy-runtime`（live_runner / runner harness）+ `mql-compiler`（VM builtins）+ `api-gateway`（proto）
> 根因/证据全量见 `docs/audits/tech-debt-registry.md` 段「LIVE 语义一致性审计」+「实盘"无法开仓"调查」。
> **本批与「实盘无法开仓调查」批（CLOSE-ORDER-UUID / SUBMIT-STUCK-RACE / DEDUP-5S-THROTTLE）互相独立，可并行；两批全部完成 = 实盘执行语义闭环。**
> **v1.1（2026-08-15 自我审计修订）**：7 项修订见 §7（种子查询方向性错误、Task 3 零量守卫、758 行红线、TimerContext 遗漏等）。
> **v1.2（2026-08-15 第三轮第一性原则审计，推翻 v1.1 Task 1 架构）**：发现 **LIVE-DELTA-DROPS-WINDOW（P0）**——delta 协议实现把 VM bar 窗口**每事件替换成 1 根**（`vmHandleBar:19` delta 分支 + `runner.go:88` `setBars` 替换语义），VM `Bars` 实盘**恒 = 1**（首 bar 后），任何读 bar 的策略永不工作。v1.1 的"种子修复"无效（只救首 bar 事件，第二个 bar 塌缩回 1）。Task 1 架构改为**每 bar 全量窗口（方案 B）+ 服务端种子**，删除 delta 路径。详见 §1 Task 1 与 §7 R8。

---

## 0. 背景与架构原则

live 模式的 VM broker 是 harness 模式（`vm_live_session.go:100` runner.New 无 executor），**唯一数据入口** = 每个 bar/tick/trade/timer 事件一次的 `r.UpdateLiveState(...)`（`vm_live_handlers.go:16/82/100/147`，**4 个调用点**）+ bar 窗口（从零攒起）。审计发现 3 个 P1 + 1 个 P2 导致策略在实盘静默不交易/交易错：

| # | 缺陷 | 对 MACD Sample（用户实测策略）的连锁卡点 |
|---|---|---|
| **LIVE-DELTA-DROPS-WINDOW（P0，v1.2 新定论）** | delta 协议实现把 VM bar 窗口每事件**替换**成 1 根（proto 注释承诺 "harness appends"，实现从未落地——git 全历史无 append 侧，ADR-0023 引入即替换语义） | VM `Bars` 实盘**恒 = 1**——`Bars<100` **永远**不过（不是干等 100 分钟），一切读 bar/指标的策略实盘永不工作 |
| LIVE-NO-PRELOAD | 服务端 bar 窗口（主+副 symbol）从零攒起（数据起点问题；与上行结构问题叠加） | 即使修好 delta，从零攒起仍要 N 分钟 |
| LIVE-NO-FREEMARGIN | `AccountFreeMargin/Margin` 恒 0 | 过了门槛后 `AccountFreeMargin()<(1000*Lots)` 恒真 → "We have no money" 永不交易 |
| LIVE-NO-EXIT | `OrderClose/Modify/Delete` 静默返回 false | 策略只能开、永远平不了仓（资金风险级） |
| LIVE-NO-SYMBOLINFO | `Point/Digits` 恒 0 | `MACDOpenLevel*Point` 阈值消失（入场条件错误放宽）；移动止损距离=0 |

**架构原则（本批的设计基准）**：**VM 能查询的一切（bars/account/symbolInfo/history）都必须有 live 权威源注入**，且注入源与回测同源（md_bars = 回测唯一源；PositionSnapshot = push 权威账户态；SymbolParam 缓存 = broker 权威 symbol 参数）。不做轮询，全走既有 push/cache 通道。

---

## 1. 任务分解

### Task 1 — LIVE-DELTA-DROPS-WINDOW + LIVE-NO-PRELOAD：全量窗口架构 + 启动种子（P0+P1）

**根因（第三轮审计定论，两个叠加缺陷）**：
1. **结构缺陷（P0）**：`vm_live_handlers.go:19-29` delta 分支把 `barWindow` 构建成**仅含 delta bars（1 根）**，`runner.go:88` `r.ctx.setBars(bars)` 是**替换语义** → 首个 bar 事件（走全量数组路径）之后，每个后续 bar 事件 VM 的 `Bars()` 恒 = 1。proto 注释（`strategy_runtime.proto:236-239`）承诺 "the harness **appends** these to its local bar window"——**append 侧从未实现**（git 全历史 `vm_live_handlers.go` 6 commits 无 append 逻辑；ADR-0023 引入即替换）。**回测对照**：`backtest/engine.go:61/85` btCtx 持全量 `e.bars` + `barIndex` 递增，策略每根 bar 看到完整窗口 → live/backtest 结构性断裂。
2. **数据起点缺陷（P1）**：服务端窗口 `bars := make([]liveBar, 0, maxContextBars)`（`live_runner.go:218`）从零攒起，主+副 symbol 均无历史预载。

**架构决策（方案 B：每 bar 全量窗口——第一性论证）**：

| 维度 | 方案 A：会话侧滚动窗口（补 append） | **方案 B：每 bar 全量窗口（采纳）** |
|---|---|---|
| 状态 | VMLiveSession 新增窗口状态（append/去重/替换/cap），**双窗口两份真相**（服务端 `bars` + 会话窗口）可漂移 | **零新状态**——handlers 保持无状态，服务端窗口是唯一事实源 |
| 与回测同构 | 需自行保证窗口语义与回测逐 bar 全量一致 | **天然同构**——回测就是每事件全量窗口（`btCtx.bars` + barIndex），live 改为同一模式，bar 语义一致性问题**从架构上消失** |
| 代码量 | 增（append 逻辑 + 守卫 + 测试） | **减**——删 `buildDeltaContext`（live_context.go:185-200）+ `vmHandleBar` delta 分支（:19-29），负代码量 |
| 带宽/成本 | ~40B/bar 事件 | ~30KB/bar 事件（500 根×6 字段）——**1m 周期 = 每策略每分钟一次，proto 二进制约百微秒级 marshal**，成本可忽略；tick 策略不走 bar 通道不受影响 |
| 副symbol 一致性 | — | **副 symbol 本来就是每次全量**（`buildSymbolSeries` 无 delta）——delta 是主 symbol 独有的坏特例，删除后两者一致 |
| 本质判断 | 为不存在的带宽问题保留坏优化 | delta 协议是**过度设计**（1 事件/分钟场景的带宽优化），且实现是错的——删除而非修补 |

**改动清单**：
1. **删 delta 路径**：`live_runner_events.go:50` `buildDeltaContext` 调用改为恒 `buildLiveContext`；删 `buildDeltaContext`（live_context.go:185-200）；删 `vm_live_handlers.go:19-29` delta 分支（`vmHandleBar` 只留全量数组路径）。`firstBar` 标志**保留**（仍决定 session Start 时机），只是不再决定全量/delta。proto `DeltaBar`/`delta_bars` 字段**保留**（字段号不可复用，注释标 deprecated，防旧消息兼容问题）。
2. **启动种子**：`startLiveRun` 中 `bars := make(...)` 之后调新 helper（新文件 `live_seed_bars.go`）：
   ```go
   func (s *StrategyExecutionServer) seedBarWindows(ctx context.Context, cfg LiveStrategyConfig, bars *[]liveBar, extraBars map[string][]liveBar)
   ```
   - **REUSE `MarketDataStore.MaxCloseTs` + `GetKlines`**（`repository/market_data_store.go:36/:22`）两步：`maxTs := MaxCloseTs(broker, symbol, tf)`（0=无数据→跳过）；`GetKlines(symbol, broker, tf, from=maxTs-500*periodMs+periodMs, nil, maxContextBars)`。⚠️ **GetKlines 是升序 `ORDER BY open_ts ASC LIMIT N`（`market_data_pg.go:105`）——不给 `from` 会拿到 6 个月窗口里最旧的 500 根（方向性错误）**。
   - **broker 标识** = `mt_accounts.broker_company`（本账户实证对齐：BTCUSDm bars "Exness Technologies Ltd"=23,930 根 = 账户 broker_company ✓；"mt4"/"Exness (VG) Ltd" 属其他来源必须过滤）。取不到/MaxCloseTs=0/klines 空 → `log.Warn` + 跳过（降级=今天行为，不阻塞 run）。
   - 只种闭合 bar（LIVE-1 一致）：丢 `openTime+periodMs > now`。
   - `KlineBar→liveBar`：`OpenTsUnixMs` uint64→`int64()` 显式转换；`Volume` float64→`strconv.FormatFloat(f,-1,64)`；价格字段 `decimal.String()`；`IsReplay` 忽略。
   - 副 symbol：`cfg.ExtraSymbols` 逐一同法预载进 `extraBars`。
3. **服务端去重守卫（三态）**：`handleBar`（live_runner_events.go:23 append 前）与 `handleExtraSymbolBar`（live_runner.go:377）：`<`末根 → 跳过（乱序/重投）；`==` → **替换末根**（实时流权威于 backfill 快照）；`>` → append。**方案 B 下守卫只在服务端这一处**（无会话侧第二份窗口）。
4. `firstBar` 生命周期/`Start` 时机不变；session 错误重置路径（`*firstBar = true`）自然复用全量重发 ✓。paper 模式：md_bars 无其数据 → 自然空种，无需特判。

**对抗证明（必带，断言级）**：
- **结构缺陷证明**：集成测试——服务端窗口预置 120 根 + 发**两个**连续 bar 事件（模拟 v1.2 场景）→ 第二个事件后 VM `Bars >= 120`（GREEN）；还原 delta 路径（或删全量发送改动）→ 第二个事件后 `Bars == 1`（RED）。⚠️ 测试必须发**两个**事件——只发一个测不出替换塌缩（v1.1 方案正是漏在这）。
- **种子证明**：mock repo 种 120 根 → 首事件 VM `Bars >= 100`（GREEN）；删 `seedBarWindows` 调用 → 首事件窗口长度 1（RED）。
- **去重守卫**：同 openTime 重投 → 长度不变且末根被替换（GREEN）；删 `==` 分支 → 重复追加（RED）。
- **回归护栏**：`go test ./tools/mql2go/... ./internal/connect/strategy/...` 全绿（delta 删除不伤回测——回测不经此路径）。

### Task 2 — LIVE-NO-FREEMARGIN：margin/free_margin 注入（P1）

**数据源（现成，无需新 RPC）**：`mthub.PositionSnapshot`（`broker_types.go:21-22`）已带 `Margin/FreeMargin`，由 OnOrderUpdate 流 push 填充，`backfillContextStrings`（`live_context.go:65`）读的正是这份快照。

**改动文件**：
1. `proto/ant/v1/strategy_runtime.proto`——**字段号精确指定（已核对现有占用）**：
   - `LiveStrategyContext`：`string margin = 19; string free_margin = 20;`（18 已用）
   - `TickContext`：`string margin = 11; string free_margin = 12;`（10 已用）
   - `TradeContext`：`string margin = 15; string free_margin = 16;`（14 已用）
   - `TimerContext`：`string margin = 8; string free_margin = 9;`（7 已用）——**vmHandleTimer（`vm_live_handlers.go:147`）也调 UpdateLiveState，不遗漏**。
   - `buf generate` 只生成 Go（TS 生成工具未装，changelog 先例）；新字段纯增量，前端 TS 客户端无感知，**不需要** TS 再生成。
2. `backend/internal/connect/strategy/live_context.go`：`backfillContextStrings` 扩展同时填 margin/free_margin。**缺失语义与 equity/balance 统一 = "-1"**（fail-visible：`AccountFreeMargin()=-1` 使 `-1<1000*Lots` 恒真 → 快照缺失时策略 fail-closed 不交易，且值可辨非真值）。三个 builder（buildTickContext/buildTradeContext/buildLiveContext——**buildDeltaContext 已随 Task 1 删除**，若施工时仍存在说明 Task 1 未完成）透传。
3. `backend/strategy/runner/runner.go:58`：`UpdateLiveState(balance, equity, margin, freeMargin string, positions)`（加参）；`contextImpl` 加 `liveMargin/liveFreeMargin`；`brokerImpl.Account()`（broker.go:167 harness 分支）返回 `{Balance, Equity, Margin, FreeMargin}`。
4. `backend/internal/connect/strategy/vm_live_handlers.go`：**4 个调用点**（vmHandleBar/vmHandleTick/vmHandleTrade/vmHandleTimer）传 `lctx/tctx.Margin, lctx/tctx.FreeMargin`。
5. **签名变更波及面**：`grep -rn "UpdateLiveState"` 全代码库更新（含 runner 测试/deploy_live_test 等 mock）。
6. **Leverage 本轮不做**（无 push 源；源待定=mt4 AccountInfo RPC 缓存），registry 记 P2 后补，`AccountLeverage()` 保持 0 并注释标注。

**对抗证明**：
- mock tctx 带 margin=10/free_margin=9990 → VM `AccountFreeMargin()>0` 且 `AccountMargin()>0`（GREEN）；删 backfill 字段赋值 → 全 0（RED）。
- 快照缺失路径 → `AccountFreeMargin()==-1` 而非 0（防"0 冒充真值"，断言 -1）。

### Task 3 — LIVE-NO-EXIT：signalMode 补平仓/改单/删单信号（P1）

**背景**：`dispatchLiveSignal` 的 `close/modify/cancel/close_all` 分发分支（`live_dispatch.go:81-88`）与 `ExecutedTicket` proto 映射（`vm_live_handlers.go:236`）**全部现成**，只差 VM builtin 侧在 signalMode 下发信号。当前这些内置函数直调 `Broker()`（executor=nil → 静默 false）。

**⚠️ 文件红线与搬移方式（v1.2 精确化）**：`vm_builtin_trade.go` 现 758 行，**已超 450 硬红线**——该文件**一行都不加**。做法：把受影响的 builtin 函数（builtinOrderClose/builtinOrderCloseBy/builtinOrderModify/builtinOrderDelete + CTrade 平仓/改单系列）**整体搬移**到同包新文件 `vm_builtin_trade_signals.go`，signalMode 分支在新文件里加。**函数名不变、同包 → `vm_builtin_wiring.go` 注册行零改动**（wiring 按标识符引用）。副效应：`vm_builtin_trade.go` 净减 100+ 行，脱离红线区。

**改动文件**：`backend/tools/mql2go/vm_builtin_trade_signals.go`（新）+ 注册行

| builtin | signalMode 行为 |
|---|---|
| `builtinOrderClose` | `vm.signal = &sdk.Signal{Action: sdk.ActionClose, Symbol: vm.ctx.Symbol(), OrderTicket: ticket, Volume: volume, Price: price}` → return true |
| `builtinOrderCloseBy` | `ActionClose` + `OrderTicket: ticket1` + **`Volume: 0`（语义=全仓平）**——配套放宽 `dispatchCloseOrder`（`live_dispatch.go:201`）零量守卫：`volume <= 0` 时**放行**（mthub.CloseOrder 对 lots=0 语义=全仓平，UI 平仓路径已实证 broker 成功）。ticket2 受限（单信号槽；MQL4 对冲锁仓罕见），注释标注 |
| `builtinOrderModify` | `ActionModify` + `OrderTicket: ticket, StopLoss: sl, TakeProfit: tp, Price: price` → true |
| `builtinOrderDelete` | `ActionCancel` + `OrderTicket: ticket` → true |
| `CTrade.PositionClose`（:694 附近）等 CTrade 系列平仓/改单 | 同上映射 |

非 signalMode（回测 SimBroker）路径**零改动**。`vm.signal` 单信号槽既有语义（同 tick 多次下单最后一次胜出）保持不变。

**语义约定（必须写进代码注释）**：signalMode 下返回 true = **"信号已发出"**，非"已平仓/已改单"——dispatch 走异步 goroutine（`dispatchCloseOrder` 等），与 `builtinOrderSend` 返回 ticket=1 同构。MQL 代码拿 true 继续走，真实结果由 broker 事件回流。

**正面效应（v1.1 补）**：paper 模式同样走 signalMode VM + `dispatchPaperSignal` 已有 close/modify/cancel 分支（`live_dispatch.go:296-327`）→ **本任务同时修复 paper 模式的平仓/改单**。

**验收链（对 MACD Sample）**：MQL `OrderClose(OrderTicket(),OrderLots(),Bid,3,Violet)` → 信号 `type=close, ticket=持仓ticket, volume=lots` → `dispatchCloseOrder` → `mthub.CloseOrder` → broker 平仓成功（OMS 平仓记录受 CLOSE-ORDER-UUID 批影响，另行验收）。

**对抗证明**：
- VM 单测：编译含 `OrderClose(123, 0.01, 1.0, 3)` 的策略，signalMode 执行 → 断言产出 `ActionClose` + `OrderTicket==123`（GREEN）；删 signalMode 分支 → 无信号且返回 false（RED）。Modify/Delete/OrderCloseBy 同模式各一。
- 集成：live harness 全链 mock（参照 `deploy_live_test.go`）→ close 信号到达 dispatchCloseOrder；零量 close 信号不再被守卫丢弃。

### Task 4 — LIVE-NO-SYMBOLINFO：Point/Digits/ContractSize/StopsLevel 注入（P2，同批顺带）

**数据源**：`mthub.CachedSymbolParam(ctx, accountID, symbol)`（既有缓存+TTL，`evaluatePlaceGate` 已在用）→ `SymbolParam{PointValue, Digits, StopsLevel, ContractSize}`。

**取数时机（v1.1 修订：run 启动取一次，不进热路径）**：在 `startLiveRun` 中调一次 `CachedSymbolParam`（每 run 一次 RPC，TTL 缓存内零开销），结果存 runner-scope（经 `cfg` 旁结构或 server 字段按 runID 存）；`buildLiveContext/buildTickContext`（delta builder 已随 Task 1 删除）只做字符串填充。**禁止**在每 tick 的 builder 里调 CachedSymbolParam（TTL 过期时会把 broker RPC 引进 tick 热路径）。

**改动文件**：
1. `proto/ant/v1/strategy_runtime.proto`——字段号精确：`LiveStrategyContext`：`string point = 21; int32 digits = 22; string contract_size = 23; int32 stops_level = 24;`；`TickContext`：`string point = 13; int32 digits = 14; string contract_size = 15; int32 stops_level = 16;`（TradeContext/TimerContext 不加——OnTrade/OnTimer 语义不依赖 symbol 常量，克制）。
2. `live_context.go`：`backfillSymbolInfo(cfg, lctx/tctx)` 填充（取 run 启动时缓存值）；失败/缺失 → 字段留空 = 0 语义 + log，不阻塞分发。
3. `runner/broker.go`：`brokerImpl` 加 `symbolInfo sdk.SymbolInfo` + `setSymbolInfo`；`SymbolInfo(symbol)` harness 分支返回 `b.symbolInfo`（executor 非 nil 路径不动）。字段映射：`Point←PointValue, Digits←Digits, StopsLevel←StopLevel, ContractSize←ContractSize`（`sdk.SymbolInfo` 四字段已核对存在，broker.go:58-72）。
4. `runner/runner.go`：`UpdateSymbolInfo(info sdk.SymbolInfo)`；`vm_live_handlers.go` 各事件 `UpdateLiveState` 旁调 `r.UpdateSymbolInfo(...)`（幂等）。
5. `contextImpl.Point()/Digits()`（context.go:135-147）自动受益，零改动。`Spread()` 已由 tick Ask-Bid 计算 ✓。

**对抗证明**：lctx 带 point=0.01/digits=2 → VM `Point()==0.01`（GREEN）；删 `UpdateSymbolInfo` 调用 → 0（RED）。本账户实测值：BTCUSDm Point=0.01、Digits=2（快照价格 63098.02 佐证）。

### 明确不做（本轮范围外，registry 已记）

- **LIVE-NO-HISTORY**（P2）：`HistoryOrders/Deals` 恒 nil——设计方向已定（run 启动从 PG `trade_records` 种历史 + trade 事件流增量追加），**单独后批**，不与本批混。
- cross-timeframe 指标（Phase B2 既有 shelfware）；forming bar 进窗口（与 LIVE-1 冲突，文档化不改）；AccountLeverage。

---

## 2. 红队自审清单（施工方交付前必逐项核对，铁律要求）

> 与对抗证明是两件事：对抗证明证"测试非空跑"；本清单抓"测试没覆盖的坑"。逐项打 ✅ 才准提交。

**Task 1（全量窗口 + 种子）**
- [ ] **结构测试必须发两个连续 bar 事件**——只发一个测不出 delta 替换塌缩（v1.1 正是漏在这，测试若只发一个事件会假绿）
- [ ] `GetKlines` 调用必带 `from` 窗口（升序 LIMIT 坑）——漏 `from` = 种进半年前数据，测试必须断言种子的**最新根时间接近 now**（而非只断数量）
- [ ] `MaxCloseTs==0`（账户无数据）/broker_company 为空/klines 空 → 三路径全 log+跳过，run 不阻塞
- [ ] uint64→int64 溢出：OpenTsUnixMs 是毫秒时间戳，int64 足够；但**编译期要显式转换**，隐式转换在 Go 是编译错误
- [ ] 周末缺口：种子后窗口最后一根可能是周五收盘（距 now 2 天）——确认 VM/指标不因"时间跳跃"崩（回测同源数据同样有洞，行为一致即可）
- [ ] 全量窗口内存：500 根 × strings 每 bar 事件重建——确认无每事件泄漏（barWindow 为局部变量，GC 回收；勿引入包级缓存）
- [ ] delta 删除后 grep `DeltaBars`/`buildDeltaContext` 源码零残留（gen/ 生成代码除外）；`DeltaBar` proto 字段保留 + deprecated 注释
- [ ] 副 symbol 循环里单个 symbol 失败不中断其余（continue + log）
- [ ] `live_context.go` 行数：删 buildDeltaContext（-16 行）抵消 Task 2/4 增量，仍超 300 软限则拆 `live_context_backfill.go`

**Task 2（margin 注入）**
- [ ] 快照缺失 → margin/free_margin = "-1"（与 equity 同语义）；**不是 "" 不是 "0"**——"" 会被 parseDecimal 成 0，`AccountFreeMargin()=0` 与"真实 0 margin"不可区分
- [ ] `UpdateLiveState` 签名变更后 `grep UpdateLiveState` 零遗漏（含 runner 包测试、deploy_live_test mock）
- [ ] proto 字段号按 §Task 2 指定值（19/20、11/12、15/16、8/9）——写错号 = 静默错位
- [ ] TimerContext 不漏（OnTimer 策略同样需要真实 FreeMargin）

**Task 3（close/modify 信号）**
- [ ] 新代码不进 `vm_builtin_trade.go`（758 行红线文件）；新文件后 `check-file-lines --strict` 无新增 ERROR
- [ ] `OrderCloseBy` 配套的 `dispatchCloseOrder` 零量守卫放宽——只放宽 close 信号路径，**不影响** modify/cancel 的 ticket==0 守卫
- [ ] MQL 常见写法 `OrderClose(OrderTicket(), OrderLots(), Bid, 3, Violet)` 的 OrderTicket 来自 OrderSelect 缓存——live 下 cachedPositions 每事件重置（`vm.go:173-179`），确认 close 信号 ticket 与 broker 真实 ticket 一致（用本账户持仓实证一次）
- [ ] 回测路径零改动验证：`go test ./tools/mql2go/...` 全绿（signalMode=false 分支不变）
- [ ] 零量放行后 `mthub.CloseOrder` 的 `evaluateCloseGate` 不因 volume=0 报错（gate 只读 intent，实证 UI 路径 lots=0 已通）

**Task 4（symbol info）**
- [ ] `CachedSymbolParam` 只在 `startLiveRun` 调一次——diff 里搜 `CachedSymbolParam` 确认未出现在 tick 路径
- [ ] `SymbolParam.Digits` 是 `int32`（proto 字段用 int32 对齐）；PointValue 用 `decimal.String()` 传（勿 `InexactFloat64` 往返丢精度）
- [ ] 缓存未命中（run 启动时 gateway 未就绪）→ 字段留空 + 不阻塞；**但**启动竞态后不重试会整 run 无 symbol info——考虑在首 bar 时若为空再补取一次（一次重试上限）

**通用**
- [ ] 本批改动不进「无法开仓调查」批的 mthub 文件（service_orders*/oms_writer/reconciliation）——冲突区零越界
- [ ] `go vet ./...` 干净；无 `//nolint`/`// @ts-ignore`；无 JSON 序列化引入
- [ ] 每项对抗证明用 `scripts/verify-adversarial.sh` 自跑一遍（删行必红→自动还原）

---

## 3. 复用核对（Reuse Preflight）

| 项 | 结论 |
|---|---|
| bar 历史加载 | **REUSE: `MarketDataStore.GetKlines` + `MaxCloseTs`** @ `repository/market_data_store.go:22/:36`（回测同源，去重/时序/质量选优已内建） |
| 账户态注入点 | **REUSE: `backfillContextStrings`** @ `live_context.go:65`（push 快照读，Task 2 原地扩展） |
| margin 数据源 | **REUSE: `PositionSnapshot.Margin/FreeMargin`** @ `mthub/broker_types.go:21-22`（OnOrderUpdate push 已填，零新增采集） |
| close/modify/cancel 分发 | **REUSE: `dispatchLiveSignal` 分支** @ `live_dispatch.go:81-88`（现成，Task 3 只补 VM 侧） |
| ticket 传递 | **REUSE: `ExecutedTicket`** @ `vm_live_handlers.go:236`（已映射） |
| symbol 参数源 | **REUSE: `mthub.CachedSymbolParam`**（`service_orders.go:168` evaluatePlaceGate 已用） |
| 新能力 | **NEW**：proto 字段（margin/free_margin/point/digits/contract_size/stops_level）+ `seedBarWindows`/`UpdateSymbolInfo`/`vm_builtin_trade_signals.go`——无现成能力（已搜：`seedBar`/`preload`/`UpdateSymbol` 全代码库零命中） |

## 4. 门禁（Before Commit）

```bash
go build ./...                                            # 必须过
cd backend && go run ./tools/check-file-lines --strict     # 0 新增 ERROR（vm_builtin_trade.go 758 行为预存红线，本批不得增行）
go test ./internal/connect/strategy/... ./strategy/runner/... ./tools/mql2go/...   # 全绿
bash scripts/verify-adversarial.sh <test> <pkg> <file> <sed-mutation>   # 对抗证明自检（删行必红）
bash scripts/gen_capability_map.sh                        # 刷新 CAPABILITIES.md
```

部署：`docker compose build backend && docker compose up -d backend`（唯一合法方式；**禁宿主机 go build/go run 后端**——16:52:21 CLEANUP-MISFIRE 事件同源嫌疑）。

## 5. 验收标准（审计方 5 维）

1. **意图**：每处修复解决"VM 查询面 vs live 权威源"的断裂，非字面补丁（如 margin 用 push 快照而非新 RPC；预载用回测同源而非 broker RPC）。
2. **可演进**：注入面统一走 proto 字段 + UpdateLiveState/UpdateSymbolInfo 模式——未来加 history 注入同构扩展，不加新通道。
3. **测试**：4 项对抗证明全为**断言级删行必红**（§1 各 GREEN/RED 描述；禁用 log-only/直调绕过主链路的无效证明）+ §2 红队清单全 ✅。
4. **防御**：快照缺失 = -1（可见失败）不冒充 0；MaxCloseTs=0/空 broker/空 klines = 降级不阻塞；CachedSymbolParam 失败 = 字段留空不阻塞；副 symbol 单点失败不影响其余。
5. **克制**：不做 leverage、不做双信号槽、不做启动即建 session、不做 history（另批）、symbol info 不进 TradeContext/TimerContext——边界已在 §1"明确不做"。

**生产实测（部署后，由审计方执行）**：
1. 重启 schedule `599ddaa5`（MACD Sample）→ 日志 `bars less than 100` 刷屏**立即停止**（窗口已种 ≥100，且最新根 ≈ now）；
2. 无 `We have no money`（FreeMargin 真实 ≈ 55094）；`Point=0.01`（MACDOpenLevel 阈值生效）；
3. MACD 条件满足时产生 buy 信号 → 订单到 broker；条件翻转时**出现 `type=close` 信号**（strategy_signals 可查）→ broker 平仓成功（OMS 平仓记录受 CLOSE-ORDER-UUID 批影响，另行验收）；
4. 回测同一策略同一参数 → 信号时序与实盘一致（live=backtest 同源验证）。

## 6. 回填纪律（完工必做，不做=判失败）

1. registry 各条目状态 → ✅done + 真实根因/修复方式/对抗证明结果（与审计方假设不同处如实写明）；
2. `handover-audit-plan.md` 变更日志追加一行（append-only）；
3. 若普遍 pitfall → 补 CLAUDE.md 对应 Pitfalls 段；
4. 不自行宣告完成——等审计方独立删行复测 + 生产实测。

**任务顺序**：Task 1→2→3→4 无强依赖，可任意顺序；建议 1 先行（用户可立即验证 `bars less than 100` 刷屏停止）。

## 7. 修订记录

**v1.1 自我审计（7 项）**

| # | 修订 | 原 v1.0 问题 |
|---|---|---|
| R1 | Task 1 种子查询改两步：`MaxCloseTs` + 带 `from` 窗口的 `GetKlines` | **方向性错误**：GetKlines 升序 LIMIT，裸调会种进 6 个月窗口里最旧的 500 根 |
| R2 | Task 2 调用点 3→4（补 vmHandleTimer）+ TimerContext 字段号 8/9 + 全量 grep 波及 | Timer 调用点遗漏 → 签名变更编译失败 |
| R3 | Task 3 新增代码**不进 vm_builtin_trade.go**（新文件 + 注册行） | 该文件已 758 行超 450 硬红线，再增行违规 |
| R4 | OrderCloseBy 补 Volume=0 语义 + 放宽 dispatchCloseOrder 零量守卫 | 原映射会被 volume>0 守卫丢弃 → 信号死路 |
| R5 | 去重守卫三态化：`<`跳过 / `==`替换 / `>`追加 | 原"<=跳过"会用低质量 backfill 边界 bar 压掉实时权威版本 |
| R6 | Task 4 取数时机：run 启动一次，禁入 tick 热路径 | 每 tick CachedSymbolParam 在 TTL 过期时会把 broker RPC 引上 tick 热路径 |
| R7 | 补 §2 红队自审清单 + §Task 2/4 精确 proto 字段号 + 语义约定（true=信号已发出）+ paper 正面效应 + buf 只生成 Go | 交接铁律要求显式红队自审段（不只对抗证明）；字段号留白易错位 |

**v1.2 第三轮第一性原则审计（1 项架构级推翻 + 教训）**

| # | 修订 | v1.1 问题 |
|---|---|---|
| R8 | **Task 1 架构推翻重设计**：发现 LIVE-DELTA-DROPS-WINDOW（P0）——delta 分支（vm_live_handlers.go:19）+ setBars 替换语义（runner.go:88）→ VM `Bars` 实盘恒 1；v1.1 的"种子"只救首 bar 事件，第二个 bar 即塌缩。架构改**方案 B（每 bar 全量窗口）**：删 delta 路径（负代码量）、零新状态、单一事实源、与回测逐 bar 全量天然同构、副 symbol 本就全量（delta 是主 symbol 坏特例）；带宽 ~30KB/bar 事件（1 事件/分钟）可忽略。结构对抗测试**必须发两个连续 bar 事件** | v1.1 只修数据起点（种子）没修结构（替换语义）——**修复无效**。教训：审计到"注入面缺数据"层就停了，没有继续下钻到"注入的数据结构本身是否正确"（窗口替换 vs 追加）；且第一轮对用户宣布的"攒 100 根要 100 分钟/18:22 过门槛"预测错误——实际**永远不过**（该错误预测已同步修正 registry） |

**v1.2 合规复查（第一性原则 + 禁令清单，全项过）**

| 检查项 | 结论 |
|---|---|
| 第一性：数据权威源唯一 | ✓ bars=PG md_bars（回测同源）；账户态=push 快照；symbol 参数=broker 权威缓存——无第二真相源 |
| 第一性：状态最小化 | ✓ 方案 B 删状态（delta 会话窗口）而非增；Task 2/4 走既有注入模式不加新通道 |
| 第一性：live/backtest 同构 | ✓ 全量窗口模式 = 回测引擎模式（btCtx.bars+barIndex），从架构上消除 bar 语义分叉 |
| Push-First | ✓ 种子=run 启动一次性读（非轮询）；每 bar 全量=流式推送；无 Ticker/setInterval |
| JSON 禁令 / decimal 纪律 | ✓ proto only；价格全 decimal-string，无 float64 价格计算 |
| 硬编码禁令 | ✓ broker 取 mt_accounts 权威列；无外部状态写死 |
| 文件红线 | ✓ vm_builtin_trade.go（758）零增行；live_context.go 删 16 行抵消增量 + 拆分预案 |
| Root-Cause-First | ✓ git 全历史核过 delta-append 从未实现（非回归），新写合规；v1.2 推翻 v1.1 依据的是代码+回测对照+git 三重证据 |
| one task = one scope | ✓ 批次="live harness 注入面补全"单一内聚域，与「无法开仓调查」批零文件交集 |

**v1.2.1 开工提示词核验轮（3 项小修）**

| # | 修订 | 问题 |
|---|---|---|
| R9 | ① Task 2/Task 4 两处仍引用已删除的 buildDeltaContext（v1.1 残留）→ 改"三个 builder（若仍存在说明 Task 1 未完成）"兼作施工自检点；② Task 3 搬移方式精确化（函数整体搬移、函数名不变 → wiring.go 零改动，原文件净减 100+ 行脱离红线）；③ §6 补回"任务无强依赖"说明 | 提示词逐句对文档+代码核验时抓出：内部矛盾会误导施工方；"新增代码放新文件"歧义可能诱导往 758 行文件加行 |

## 8. 审计方验收裁决（2026-08-15）——实现大体质忠实，**小返工后通过**

**已验证通过项**：
- 门禁全绿实测：`go build ./...` ✓ / `go test` runner+mql2go+connect/strategy ✓ / `check-file-lines --strict` 0 ERROR ✓
- **对抗复测审计方独立删行 5/5 RED 有效**：dedupe `==` 分支（TestAppendDedupBar_ThreeStates）/ margin backfill（TestBackfillContextStrings_MarginFreeMargin）/ OrderClose signalMode（TestSignalMode_OrderClose）/ UpdateSymbolInfo Point / liveMargin 注入——均断言级真红
- **结构验证 PASS（审计方临时测试，未入库）**：两个连续 bar 事件后 VM 窗口=121（无 delta 塌缩）——v1.2 架构在真实代码路径生效
- 文件红线：`vm_builtin_trade.go` 758→636（净 -122，一行未加 ✓）；`live_context.go` 269 / `live_runner.go` 409（<450 ✓）
- proto 字段号与规格逐一对上；Timer 第 4 调用点补齐；"-1" fail-visible 语义统一；三态去重统一单 helper（克制）；CloseAll 主动覆盖（同类合理扩展）

**返工项（W1-W3 必做，W4 可选）**：

| # | 级别 | 项 | 说明 |
|---|---|---|---|
| W1 | 🔴 | **dispatchCloseOrder 零量守卫放宽（原规格 R4，漏做）** | `live_dispatch.go:202` 仍丢弃 `volume<=0` 的 close 信号 → `OrderCloseBy` 与 `CTrade.PositionClose`（MQL5 全仓平语义，Volume=0）发出的信号被静默丢弃。修法：close 路径 volume<=0 **放行**（mthub.CloseOrder 对 lots=0=全仓平，UI 路径已实证）；modify/cancel 的 ticket==0 守卫不动。对抗测试：零量 close 信号到达 CloseOrder mock（删放行 → RED） |
| W2 | 🟡 | **Task 4 取数移出事件热路径（原规格 R6，偏离）** | `backfillSymbolInfo`/`backfillTickSymbolInfo` 在**每 bar/tick** 的 builder 里调 `CachedSymbolParam` 且 `context.Background()` **无超时**——TTL 5min 过期时一个 tick 事件同步做无超时 broker RPC；网关断连时（cache 无条目）**每 tick 都发起失败 RPC**。修法：run 启动取一次存 server 字段（按 runID），builder 只做字符串填充；启动失败 → 首 bar 重试一次上限；任何兜底调用必须 5s 超时 |
| W3 | 🟡 | **补 2 项缺失对抗测试** | ① `seedBarWindows` 零测试——mock MarketDataStore：种子 120 根 → 首事件窗口≥100（GREEN）/ 删 seedBarWindows 调用 → RED；② 两连续 bar 事件结构测试入库（审计方已独立验证 PASS，可用探针策略模式：stub sdk.Strategy 在 OnBar 记录 `ctx.Bars().Len()`，断言第二事件后 ≥121） |
| W4 | ℹ️ | gen churn（可选） | protoc-gen-go v1.36.11→v1.36.12 重生 150+ 无关 pb.go（仅版本注释行变化，无害）——建议 pin 插件版本防再次全量 churn |
| W5 | ⚠️ | **部署**（返工完成后） | 容器仍是 2026-08-14 16:42 旧镜像（未部署）。`docker compose build backend && docker compose up -d backend` 后**重启 MACD 调度 `599ddaa5`**，审计方跑生产验收链（§5：刷屏立停 / FreeMargin 真实 / buy+close 信号出现） |

