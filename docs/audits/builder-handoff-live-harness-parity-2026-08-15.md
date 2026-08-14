# Builder Handoff — LIVE-HARNESS-PARITY（实盘 harness 注入面补全）

> 批次名：LIVE-HARNESS-PARITY ｜ 日期：2026-08-15 ｜ 审计方：Claude Code ｜ 施工方：Windsurf
> 涉及功能块：`strategy-runtime`（live_runner / runner harness）+ `mql-compiler`（VM builtins）+ `api-gateway`（proto）
> 根因/证据全量见 `docs/audits/tech-debt-registry.md` 段「LIVE 语义一致性审计」+「实盘"无法开仓"调查」。
> **本批与「实盘无法开仓调查」批（CLOSE-ORDER-UUID / SUBMIT-STUCK-RACE / DEDUP-5S-THROTTLE）互相独立，可并行；两批全部完成 = 实盘执行语义闭环。**

---

## 0. 背景与架构原则

live 模式的 VM broker 是 harness 模式（`vm_live_session.go:100` runner.New 无 executor），**唯一数据入口** = 每个 bar/tick/trade 事件一次的 `r.UpdateLiveState(balance, equity, positions)`（`vm_live_handlers.go:16/82`）+ bar 窗口（从零攒起）。审计发现 3 个 P1 + 1 个 P2 导致策略在实盘静默不交易/交易错：

| # | 缺陷 | 对 MACD Sample（用户实测策略）的连锁卡点 |
|---|---|---|
| LIVE-NO-PRELOAD | bar 窗口（主+副 symbol）从零攒起 | `Bars<100` 干等 100 分钟（本次用户报告的元凶） |
| LIVE-NO-FREEMARGIN | `AccountFreeMargin/Margin` 恒 0 | 过了门槛后 `AccountFreeMargin()<(1000*Lots)` 恒真 → "We have no money" 永不交易 |
| LIVE-NO-EXIT | `OrderClose/Modify/Delete` 静默返回 false | 策略只能开、永远平不了仓（资金风险级） |
| LIVE-NO-SYMBOLINFO | `Point/Digits` 恒 0 | `MACDOpenLevel*Point` 阈值消失（入场条件错误放宽）；移动止损距离=0 |

**架构原则（本批的设计基准）**：**VM 能查询的一切（bars/account/symbolInfo/history）都必须有 live 权威源注入**，且注入源与回测同源（md_bars = 回测唯一源；PositionSnapshot = push 权威账户态；SymbolParam 缓存 = broker 权威 symbol 参数）。不做轮询，全走既有 push/cache 通道。

---

## 1. 任务分解

### Task 1 — LIVE-NO-PRELOAD：run 启动预载 bar 窗口（P1）

**改动文件**：`backend/internal/connect/strategy/live_runner.go`（+新 helper 可放 `live_runner.go` 或新文件 `live_seed_bars.go`）

1. `startLiveRun` 中 `bars := make([]liveBar, 0, maxContextBars)` 之后、事件循环之前，调用新 helper：
   ```go
   func (s *StrategyExecutionServer) seedBarWindows(ctx context.Context, cfg LiveStrategyConfig, bars *[]liveBar, extraBars map[string][]liveBar)
   ```
2. **REUSE `PgMarketDataStore.GetKlines`**（`repository/market_data_pg.go:59`，回测同源加载器，自带去重/时序/质量 tiebreaker）：`s.marketDataRepo` 字段已在 `handlers_strategy.go:113` 注入 ✓。
   - 主 symbol：`GetKlines(ctx, cfg.Symbol, <账户broker>, cfg.Timeframe, nil, nil, maxContextBars)`
   - 账户 broker 标识：从 `mt_accounts` 取该账户的 broker 字段（施工方先核对 `187_cleanup_account_schema` 后的实际列名，如 `broker_company`/`broker_id`），**broker 必传**（空 broker 的 legacy 路径 distinct key 含 broker 会混串账户）。查不到 → 跳过预载（降级为今天行为，不更糟）。
   - 副 symbol：`cfg.ExtraSymbols` 逐一同法预载进 `extraBars`。
3. **只种闭合 bar（LIVE-1 一致）**：丢弃 `openTime + PeriodMs(timeframe) > now` 的 bar（当前正在形成的周期不种）。
4. **KlineBar → liveBar** 转换：`open/high/low/close` 用 `decimal.String()`，`volume` 用 `strconv.FormatFloat`，`openTime` 直取。
5. **去重守卫**：`handleBar`（`live_runner_events.go:23` append 前）与 `handleExtraSymbolBar`（`live_runner.go:377`）加：`bar.OpenTime <= 窗口最后一根 openTime` 则跳过（流重投/预载重叠防重复）。种子期间形成的"边界 bar"由实时流权威覆盖。
6. `firstBar` 生命周期**保持不变**（首根闭合 bar 到达时建 session，此时窗口已满）——控制范围，不做"启动即建 session"。
7. paper 模式：账户无 md_bars 数据时查询自然为空 = 无操作，无需特判。

**对抗证明（必带）**：
- 集成测试（mock MarketDataStore 或 PG 集成，参照 `oms_writer_integration_test.go` 模式）：种子 120 根 → 首 bar 事件后 VM 内 `Bars >= 100`（GREEN）；删 `seedBarWindows` 调用 → 首 bar 窗口长度 1（RED）。
- 去重守卫：mock 流重投与最后一根同 openTime 的 bar → 窗口长度不变（GREEN）；删守卫 → 重复追加（RED）。

### Task 2 — LIVE-NO-FREEMARGIN：margin/free_margin 注入（P1）

**数据源（现成，无需新 RPC）**：`mthub.PositionSnapshot`（`broker_types.go:14`）已带 `Margin/FreeMargin/MarginLevel`，由 OnOrderUpdate 流 push 填充，`backfillContextStrings`（`live_context.go:65`）读的正是这份快照。

**改动文件**：
1. `proto/ant/v1/strategy_runtime.proto`：`LiveStrategyContext`（:210）/ `TickContext`（:285）/ `TradeContext`（:300）各加 `string free_margin = <next>; string margin = <next>;`（buf generate）。
2. `backend/internal/connect/strategy/live_context.go`：`backfillContextStrings` 扩展为同时填 `margin/free_margin`（`snap.Margin.String()` / `snap.FreeMargin.String()`），三个 builder（buildTickContext/buildTradeContext/buildLiveContext/buildDeltaContext）透传。**快照缺失时沿用现有 `-1` 语义**（fail-visible，勿静默 0 冒充真值）。
3. `backend/strategy/runner/runner.go:58`：`UpdateLiveState(balance, equity, margin, freeMargin string, positions)`（加参）；`contextImpl` 加 `liveMargin/liveFreeMargin` 字段；`brokerImpl.Account()`（broker.go:167 harness 分支）返回 `{Balance, Equity, Margin, FreeMargin}`。
4. `backend/internal/connect/strategy/vm_live_handlers.go`：vmHandleBar/vmHandleTick/vmHandleTrade 三个调用点传 `lctx/tctx.Margin, lctx/tctx.FreeMargin`。
5. **Leverage 本轮不做**（无 push 源；源待定=mt4 AccountInfo RPC 缓存），registry 记 P2 后补，`AccountLeverage()` 保持 0 并在注释标注。

**对抗证明**：mock tctx 带 margin=10/free_margin=9990 → VM 内 `AccountFreeMargin()>0` 且 `AccountMargin()>0`（GREEN）；删 backfill 字段赋值 → 恒 0（RED）。另：快照缺失路径 → 值 = -1 而非 0（防"0 冒充真值"）。

### Task 3 — LIVE-NO-EXIT：signalMode 补平仓/改单/删单信号（P1）

**背景**：`dispatchLiveSignal` 的 `close/modify/cancel/close_all` 分发分支（`live_dispatch.go:81-88`）与 `ExecutedTicket` proto 映射（`vm_live_handlers.go:236`）**全部现成**，只差 VM builtin 侧在 signalMode 下发信号。当前这 5 个内置函数直调 `Broker()`（executor=nil → 静默 false）。

**改动文件**：`backend/tools/mql2go/vm_builtin_trade.go`

| builtin | signalMode 行为 |
|---|---|
| `builtinOrderClose`（:110） | `vm.signal = &sdk.Signal{Action: sdk.ActionClose, Symbol: vm.ctx.Symbol(), OrderTicket: ticket, Volume: volume, Price: price}` → return true |
| `builtinOrderCloseBy`（:121） | `ActionClose` + `OrderTicket: ticket1`（ticket2 受限——单信号槽；MQL4 对冲锁仓罕见，注释标注限制，不做双信号扩容） |
| `builtinOrderModify`（:132） | `ActionModify` + `OrderTicket: ticket, StopLoss: sl, TakeProfit: tp, Price: price` → true |
| `builtinOrderDelete`（:157） | `ActionCancel` + `OrderTicket: ticket` → true |
| `ctradeOrder` 系列平仓/改单（`CTrade.PositionClose` :694 等） | 同上映射 |

非 signalMode（回测 SimBroker）路径**零改动**。注意 `vm.signal` 单信号槽的既有语义（同 tick 多次下单最后一次胜出）保持不变。

**验收链（对 MACD Sample）**：MQL `OrderClose(OrderTicket(),OrderLots(),Bid,3,Violet)` → 信号 `type=close, ticket=持仓ticket, volume=lots` → `dispatchCloseOrder`（`live_dispatch.go:191`）→ `mthub.CloseOrder`。**注意**：到达 mthub 后 broker 平仓会成功，但 OMS 平仓记录仍受「实盘无法开仓调查」批 CLOSE-ORDER-UUID（22P02）影响——两批独立，都完成后闭环。

**对抗证明**：
- VM 单测：编译含 `OrderClose(123, 0.01, 1.0, 3)` 的策略，signalMode 执行 → 断言产出 `ActionClose` + ticket=123（GREEN）；删 signalMode 分支 → 无信号且返回 false（RED）。Modify/Delete 同模式。
- 集成：live harness 全链 mock（参照 `deploy_live_test.go`）→ close 信号到达 dispatchCloseOrder。

### Task 4 — LIVE-NO-SYMBOLINFO：Point/Digits/ContractSize/StopsLevel 注入（P2，同批顺带）

**数据源**：`mthub.CachedSymbolParam(ctx, accountID, symbol)`（既有缓存+TTL，`evaluatePlaceGate` 已在用）→ `SymbolParam{PointValue, Digits, StopsLevel, ContractSize}`。

**改动文件**：
1. `proto/ant/v1/strategy_runtime.proto`：`LiveStrategyContext`/`TickContext` 加 `string point / int32 digits / string contract_size / int32 stops_level`（buf generate）。
2. `live_context.go`：新增 `backfillSymbolInfo(cfg, lctx/tctx)`——调 `s.mtHub.CachedSymbolParam`（5s 超时，失败时字段留空=0 语义并 log，不阻塞 bar 分发），四个 builder 调用。
3. `runner/broker.go`：`brokerImpl` 加 `symbolInfo sdk.SymbolInfo` 字段 + `setSymbolInfo`；`SymbolInfo(symbol)` harness 分支返回 `b.symbolInfo`（executor 非 nil 路径不动）。
4. `runner/runner.go`：`UpdateSymbolInfo(info sdk.SymbolInfo)`；`vm_live_handlers.go` 三处 `UpdateLiveState` 旁调 `r.UpdateSymbolInfo(...)`（幂等，重复设置无害）。
5. `contextImpl.Point()/Digits()`（context.go:135-147）自动受益，零改动。`Spread()` 已由 tick Ask-Bid 计算 ✓。

**对抗证明**：lctx 带 point=0.01/digits=2 → VM `Point()==0.01`（GREEN）；删 `UpdateSymbolInfo` 调用 → 0（RED）。

### 明确不做（本轮范围外，registry 已记）

- **LIVE-NO-HISTORY**（P2）：`HistoryOrders/Deals` 恒 nil——设计方向已定（run 启动从 PG `trade_records` 种历史 + trade 事件流增量追加），**单独后批**，不与本批混。
- cross-timeframe 指标（Phase B2 既有 shelfware）；forming bar 进窗口（与 LIVE-1 冲突，文档化不改）；AccountLeverage。

---

## 2. 复用核对（Reuse Preflight）

| 项 | 结论 |
|---|---|
| bar 历史加载 | **REUSE: `PgMarketDataStore.GetKlines`** @ `repository/market_data_pg.go:59`（回测同源，去重/时序/质量选优已内建） |
| 账户态注入点 | **REUSE: `backfillContextStrings`** @ `live_context.go:65`（push 快照读，Task 2 原地扩展） |
| margin 数据源 | **REUSE: `PositionSnapshot.Margin/FreeMargin`** @ `mthub/broker_types.go:21-22`（OnOrderUpdate push 已填，零新增采集） |
| close/modify/cancel 分发 | **REUSE: `dispatchLiveSignal` 分支** @ `live_dispatch.go:81-88`（现成，Task 3 只补 VM 侧） |
| ticket 传递 | **REUSE: `ExecutedTicket`** @ `vm_live_handlers.go:236`（已映射） |
| symbol 参数源 | **REUSE: `mthub.CachedSymbolParam`**（`service_orders.go:168` evaluatePlaceGate 已用） |
| 新能力 | **NEW**：proto 字段（margin/free_margin/point/digits/contract_size/stops_level）+ `seedBarWindows`/`UpdateSymbolInfo`——无现成能力（已搜：`seedBar`/`preload`/`UpdateSymbol` 全代码库零命中） |

## 3. 门禁（Before Commit）

```bash
go build ./...                                            # 必须过
cd backend && go run ./tools/check-file-lines --strict     # 0 新增 ERROR（live_context.go 当前 ~246 行，Task 2/4 扩展注意 375 红线；超则拆 helper 文件）
go test ./internal/connect/strategy/... ./strategy/runner/... ./tools/mql2go/...   # 全绿
bash scripts/verify-adversarial.sh <test> <pkg> <file> <sed-mutation>   # 对抗证明自检（删行必红）
bash scripts/gen_capability_map.sh                        # 刷新 CAPABILITIES.md
```

部署：`docker compose build backend && docker compose up -d backend`（唯一合法方式）。

## 4. 验收标准（审计方 5 维）

1. **意图**：每处修复解决"VM 查询面 vs live 权威源"的断裂，非字面补丁（如 margin 用 push 快照而非新 RPC）。
2. **可演进**：注入面统一走 proto 字段 + UpdateLiveState/UpdateSymbolInfo 模式——未来加 history 注入同构扩展，不加新通道。
3. **测试**：4 项对抗证明全为**断言级删行必红**（参照本批 §1 各 GREEN/RED 描述；禁用 log-only/直调绕过主链路的无效证明）。
4. **防御**：快照缺失 = -1（可见失败）不冒充 0；GetKlines 无数据 = 降级不阻塞；broker 查不到 = 跳过预载；CachedSymbolParam 失败 = 字段留空不阻塞 bar。
5. **克制**：不做 leverage、不做双信号槽、不做启动即建 session、不做 history（另批）——边界已在 §1"明确不做"。

**生产实测（部署后，由审计方执行）**：
1. 重启 schedule `599ddaa5`（MACD Sample）→ 日志 `bars less than 100` 刷屏**立即停止**（窗口已种 ≥100）；
2. 无 `We have no money`（FreeMargin 真实）；`Point=0.01`（MACDOpenLevel 阈值生效）；
3. MACD 条件满足时产生 buy 信号 → 订单到 broker；条件翻转时**出现 `type=close` 信号**（strategy_signals 可查）→ broker 平仓成功（OMS 平仓记录受 CLOSE-ORDER-UUID 批影响，另行验收）；
4. 回测同一策略同一参数 → 信号时序与实盘一致（live=backtest 同源验证）。

## 5. 回填纪律（完工必做，不做=判失败）

1. registry 各条目状态 → ✅done + 真实根因/修复方式/对抗证明结果（与审计方假设不同处如实写明）；
2. `handover-audit-plan.md` 变更日志追加一行（append-only）；
3. 若普遍 pitfall → 补 CLAUDE.md 对应 Pitfalls 段；
4. 不自行宣告完成——等审计方独立删行复测 + 生产实测。

## 6. 依赖与顺序

- Task 1→2→3→4 无强依赖，可任意顺序；建议 1 先行（用户可立即验证刷屏停止）。
- 与「实盘无法开仓调查」批（CLOSE-ORDER-UUID/SUBMIT-STUCK-RACE/DEDUP-5S-THROTTLE，registry 同段）并行施工零文件冲突（本批：live_runner*/live_context/vm_builtin_trade/runner*；彼批：mthub service_orders*/risk gate）。
