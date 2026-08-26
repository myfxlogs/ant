# Pitfalls 坑库（T1 知识层）

> 已确认的静默失败模式 + 调试路径。agent 开工前按需读取相关章节。≤ 450 行。

## MQL2GO VM Pitfalls (必读)

> 回测不开单 / volume=0 / 指标全零但 MT4/MT5 客户端正常？先查 [`docs/runbook/mql2go-known-pitfalls.md`](runbook/mql2go-known-pitfalls.md)

### 已确认的静默失败模式

- **未知常量 → 0** — `interp/constants.go` 缺常量 → 编译器 push 0。例如 `MODE_SIGNAL` 缺失时 `iMACD` 返回主线而非信号线 → `MacdCurrent == SignalCurrent` → 永不开单。
- **`builtinOrderType` 映射错误** — 必须返回 `OP_BUY=0 / OP_SELL=1`，不能返回 `PositionSide` (`SideBuy=1 / SideSell=-1`)，否则持仓管理/平仓逻辑失效。
- **Go map 迭代非确定 → 用户函数前向引用返回 0** — `ir.Funcs` 是 map，编译器必须两遍编译：Pass 1 预注册所有 entry PC，Pass 2 编译体。**通用规则：任何有序 pipeline 禁止裸遍历 map 处理有序依赖**。
- **Pass 2 中 user→user 前向引用仍可能使用 stale marker PC（BT-FUNC-ENTRYPC-FWD 🟦open）** — 当前修复只在逐个编译函数 body 时更新当前 EntryPC；尚未编译的 callee 在 `c.bc.Funcs` 中仍是 Pass 1 marker PC，caller 若先编译会把 stale PC 写进 `OP_CALL_USER`，之后 callee 更新也不会回补。现有 T3 `ForwardReference` 只覆盖 event→function（事件在所有 user body 之后编译），不是 user→user。**通用规则：所有 user function body entry 地址必须在发出任何 `OP_CALL_USER` 前最终确定，或先发符号目标再统一 patch；必须测试 caller→callee 且 callee 后定义。**
- **两遍编译的 EntryPC 指向 marker 而非 body → OP_CALL_USER 跳到错误 PC → 被调函数静默不执行（BT-FUNC-ENTRYPC ✅done）** — Pass 1 为每个用户函数 emit `OP_ENTER_FUNC` marker 并记录 `EntryPC = marker PC`；Pass 2 编译所有 body（body 在所有 marker 之后连续排列）。`executeCallUser` 执行 `vm.pc = entryPC + 1` 跳过 marker——但当有 ≥2 个用户函数时，`entryPC+1` 是下一个函数的 marker，不是本函数 body。后果：`if(res==0) CheckForOpen()` 中 `CheckForOpen()` 的 `OP_CALL_USER` 跳到错误 PC → body 静默不执行 → 永不下单。**修复**：`compileUserFuncBody` 在编译 body 前更新 `EntryPC = len(c.bc.Code)`（body 实际起始）；`executeCallUser` 改为 `vm.pc = entryPC`（不再 +1）。**通用规则：两遍编译中 Pass 1 预注册的 PC 值必须在 Pass 2 编译 body 前更新为 body 实际起始位置，不能保留 marker 位置——marker 是占位符，不是跳转目标**。
- **固定长度滚动窗口 + append-only 指标缓存 → 指标永久冻结（LIVE-INDICATOR-1 ✅done）** — live 启动 seed 恰好 `maxContextBars=500`，之后每根新 bar 都 drop oldest + append newest，窗口长度恒为 500；`SeriesCache.EnsureUpdated()` 只比较 `Len()`，看到 `n==c.n==500` 就认为无更新，导致 EMA/MACD/RSI/ATR/ADX 等所有 cache-backed 指标永远停在启动首帧。生产证据：4817 次 VM eval / 46 bar eval，但 MACD/EMA 与 00:44 首帧逐位一致，00:51 SELL 条件成立却 0 signal。**修复**：indicators 层新增可选 `RevisionedBarSource`（`Revision() uint64`），`EnsureUpdated()` 对 revisioned source 检测 revision 变化并 reset+lazy rebuild；runner 层 `runnerBarSource` 实现 `RevisionedBarSource`，`Runner.OnBar` 用 `atomic.Uint64` 推进 barRev（OnTick/OnTrade/OnTimer 不推进）；backtest `btBarSource` 不实现 revision，零开销。**通用规则：增量缓存 freshness 不能只靠长度；revisioned source 的任何 revision 变化都必须 reset+lazy rebuild，不能因长度增长就猜测只是 append**。对抗测试必须复用同一个 source+cache 做 500→500 mutation（新建 cache 会假绿），并精确覆盖 legacy `start()` 的 BAR→TICK 信号路径。完整证据/修复/对抗证明见 registry `LIVE-INDICATOR-1`。
- **broker → proto → SDK 字段丢失 → MQL OrderSelect/OrderMagicNumber 返回错误值（LIVE-MQL-ORDER-CONTEXT-1 ✅done）** — `vmPositionsToSdk` 曾只保留 Ticket/Side/Volume/OpenPrice，丢弃 Symbol/Magic/SL/TP/Swap/Commission/Profit/Comment/OpenTime；`LivePosition` proto 只有 8 字段；挂单（buy_limit/sell_stop）没有独立 proto，全塞进 `Positions`；harness `Orders(magic)` 直接返回 nil。后果：MQL `OrderMagicNumber()` 返回 0、`OrderSymbol()` 返回空、`OrdersTotal()` 漏算挂单。**通用规则：跨层字段映射必须全字段透传，禁止"只保留策略当前需要的字段"——MQL 策略可通过 OrderSelect 访问任意字段，任何字段丢失都是静默 bug**。修复：proto 补齐全字段 + 新增 `LivePendingOrder`；pipeline 按 `IsPendingOrderType` 拆分 market vs pending；`UpdateLiveState` 接收 pendingOrders；harness `Orders` 返回 `livePendingOrders`。对抗测试必须验证 magic 端到端 + 删任一层映射必 RED。完整证据见 registry `LIVE-MQL-ORDER-CONTEXT-1`。**返工教训（审计复审阻断）**：Go 层断言 `broker.Positions/Orders` 和手工 `len()` 不够——必须编译 MQL 源码 → `Runner.OnTick` → VM 实际执行 `OrdersTotal`/`OrderSelect`/`OrderMagicNumber` → 通过 `VMRunner.GetGlobal` 读取 MQL 全局变量验证值。独立改 `builtinOrderMagicNumber` pending Magic 为 0 后 Go 层测试仍 GREEN，只有 VM 层测试 RED。**通用规则：MQL builtin 对抗测试必须穿透到 VM 执行层，不能只检查 Go 层 slice**。
- **bar-based 回测 `Volume[0]` 返回整根 bar tick volume 而非 1 → `if(Volume[0]>1) return;` 新 bar 检测恒 true → 策略永不下单（BT-VOLUME-NEWBAR ✅done）** — 回测引擎每根 bar 调一次 OnTick（等价 MT4 "Open prices only" 模式），但 `Volume[0]` 返回 DB 整根 bar 的 tick volume（几百到上千）而非 MT4 该模式的 Volume=1。`Volume[0]>1` 是 MetaQuotes 官方 Moving Average sample 的新 bar 检测模式——MT4 文档明确 "Open prices only" 模式下 "bar is opened (Open = High = Low = Close, Volume=1)"。**修复**：`bt_bar_series.go` 包装 `sdk.BarSeries`，`Volume(0)=1`（当前 bar），`Volume(>0)` 保持实际值（历史 bar）；`backtestContext.Bars()/BarsTF()/BarsForSymbol()` 全部走包装。`btBarSource`（指标路径）不改——`iVolume` 等指标函数应返回实际 volume，只有 `Volume[0]` series 访问需 MT4 语义。**通用规则：bar-based 回测模拟 MT4 "Open prices only" 模式时，当前 bar 的 `Volume[0]` 必须返回 1（新 bar 第一个 tick），历史 bar 保持实际值**。

### 调试路径

1. 先查 `CoverageReport.BlindSpots` 中的 "unknown constant" / "unknown function" — 这是静默失败的首要信号。
2. 连跑验证：`for i in $(seq 1 50); do go test -run <TestName> -count=1 -v 2>&1 | grep -E 'PASS|FAIL'; done`。
3. 对比 PASS vs FAIL 的 debug 输出，确定哪些函数被调用、哪些被静默跳过。
4. 在 builtin 入口加 `fmt.Fprintf(os.Stderr, ...)` 临时日志，确认参数值与调用次数。
5. 检查 `compileCall` 路径：user func → builtin → API registry → unknown（每层 fallback 都可能静默吞掉调用）。

## Strategy Runner Rules (实盘执行 — 强制)

**Open bar 过滤（LIVE-1）**：
- ❌ 策略 runner 禁止处理未收盘 bar（`bar.Closed == false`）—— open bar 是行情快照，非策略事件
- ✅ 用 `shouldRunOnBar(bar, symbol, timeframe)` 纯函数过滤（`live_runner.go:214`）
- ✅ extra-symbol context window 也只用 finalized bar（`live_runner.go:231`）
- 后果：open bar 进 handleBar → 同一根 bar 重复执行 → 指标重复计数 → 实盘与回测发散

**Order submission 串行语义（LIVE-ORDER-REENTRY-1）**：
- ❌ 禁止 fire-and-forget goroutine 下单——`submitOrder`/`dispatchCloseOrder`/`dispatchModifyOrder`/`dispatchCancelOrder`/`dispatchCloseAll` 全部同步，事件循环阻塞到 broker mutation 确定性 outcome
- ✅ 每 `ActiveSession` 持有 `TradeBarrier`，CAS Acquire 保证最多一个未确认 broker mutation（I1）
- ✅ broker ticket ≠ positions caught up——须等权威 `OnOrderUpdate` push 或单次 read-after-write `OpenedOrders` 确认（I3/I4/I6）
- ✅ transport timeout（DeadlineExceeded）= outcome unknown → barrier 锁定 fail-closed，不重下（I5）
- ✅ `PositionCache` freshness 拆分：`GetFreshTradingSnapshot`（financials+positions 都须 fresh）给 VM/Risk Gate；`GetFreshFinancialSnapshot`/`GetFreshPositionSnapshot` 给 display；financial-only refresh 不让 stale positions 显 fresh
- 后果：fire-and-forget → VM 在 broker 确认前继续 → `OrdersTotal()==0` → 下一 tick 重复开仓 → 数秒内多个同方向订单

**Broker 应用层拒绝 vs 传输层错误（LIVE-ORDER-REENTRY-1-BROKER-REJECT）**：
- ❌ MT4/MT5 adapter 在 `resp.GetError().GetCode() != 0` 时返回裸 `fmt.Errorf("mt4 OrderSend: code=%d msg=%s", ...)` 无 sentinel → `brokerError()` 包装为 `PhaseBroker` → `ClassifyMutationError` 先检查 phase → `outcome_unknown` → barrier 永久锁定 → 策略无法再下单（生产实测 MT4 code=130 Invalid S/L or T/P → 策略永久阻塞）
- ✅ MT4/MT5 adapter 4 个订单操作（send/close/delete/modify）的应用层拒绝必须用 `fmt.Errorf("%w: ...", mthub.ErrBrokerRejected, ...)` 包装 sentinel
- ✅ `ClassifyMutationError` **sentinel 检查必须在 MutationError phase 检查之前**——sentinel 是权威的，无论 phase 包装如何，`ErrBrokerRejected` 永远是 `deterministic_rejected`
- **关键语义区分**：`resp.GetError()` 是 broker **应用层响应**（订单已到达 broker 并被明确拒绝，如 code=130 SL/TP 无效、code=134 余额不足、code=135 价格变化），不是传输层错误（gRPC `err != nil`，不知道订单有没有到达）。前者 = 确定性拒绝 → 释放 barrier → 策略可重试；后者 = outcome unknown → barrier 锁定 → fail-closed
- **通用规则**：任何从 broker 响应体（`resp.GetError()`）提取的错误都是应用层拒绝，必须标 sentinel；只有 gRPC 传输层错误（`err != nil`）才是 outcome unknown

**账户快照 freshness 续期（DATA-TRUTH-10 / DATA-TRUTH-10-FIX2）**：
- ❌ `refreshAccountSummary`（45s ticker）调 `fetchAndPublish(ctx, sid, nil, handler)` 时 `p==nil` → `PositionsAuthoritative: false` → `PositionCache.put()` 不更新 `positionsReceivedAt` → 0 持仓账户无 `OnOrderUpdate` 事件 → 90s 后 `GetFreshTradingSnapshot` 全阻塞（生产实测 92 分钟 7,730 次错误）
- ✅ `fetchAndPublish` 在 `p==nil` 时必须额外调 `OpenedOrders` RPC 获取权威持仓（即使为空），设 `PositionsAuthoritative: true`——0 持仓也是权威的
- ❌ **测试禁止把 bug 行为编码为期望行为**：原 `TestAccountSummaryRefreshContinuesWithoutProfitFrames` 断言 `PositionsAuthoritative: false`（即 bug），删修复行测试仍 GREEN = 测试穿透。修复后断言改为 `true` + mock 加 `openedOrdersRes`，删 `positionsAuth = true` → RED
- **通用规则**：任何 `PositionsAuthoritative` / `FinancialsAuthoritative` 的续期路径必须同时更新 `positionsReceivedAt` 和 `financialsReceivedAt`；只更新一个会导致 `GetFreshTradingSnapshot`（要求两者都 fresh）在 90s 后 fail-closed

## Strategy Schedule Engine Pitfalls (SCHEDULE-HOTLOOP-1 代码验收通过，待生产部署验收)

- **due timer occurrence 必须在所有 skip/deny/dispatch 分支前被持久化消费**：过期 `next_run_at` 若在 `isRunning`、`autoTrade=false`、entitlement/quota deny 等分支直接 `continue`，`GetEarliestNextRunAt` 会持续返回过去时间，timer delay=0 → CPU/DB/日志热循环。正确语义：timer schedule 每次 due 先推进 `next_run_at > now`，再决定是否 dispatch；autoTrade 关闭期间不补跑历史次数，恢复后从未来周期继续。
- **禁止在 live run 返回后才推进 next_run_at**：实盘 run 可以永久运行，`runOne` 完成路径不是 timer occurrence 的收敛点。event schedule 必须保持 `next_run_at=NULL`，timer repository 查询只选 interval/cron，startup 清理 event 脏 next 值。
- **持久化失败必须有界退避**：GetDue/ComputeNext/UpdateNext 失败时不 dispatch，ScheduleEngine 用 context-aware backoff timer 等待，`Notify` 可提前唤醒；invalid config 记录错误并 clear next 隔离。只降日志级别不能修复热循环。
- **autoTrade cache 必须由所有写入口主动失效，且 check/query/write 与 invalidate 必须线性化**：`ToggleAutoTrade` 与 `UpdateGlobalSettings` 成功后都必须 callback invalidate+Notify；但仅 delete 不够——cache miss 解锁查 DB 后再回写存在 TOCTOU，旧查询可在 invalidate 后把旧 true 写回 30s。**修复（SCHEDULE-HOTLOOP-1a）**：per-user `autoTradeGeneration` map（与 cache 共用 `autoTradeCacheMu`），`InvalidateAutoTradeCache` 临界区内 `generation[userID]++`+delete；`isAutoTradeEnabled` miss 时记录 gen → DB query 锁外执行 → 回写时 generation 不匹配则丢弃旧结果并重查。对抗测试用 channel 精确控制"旧查询开始→更新+invalidate→旧查询返回"时序，删 generation retry → 4 断言确定性 RED。
- **ClearNextRunAt 必须用 SQL NULL 而非零时间**：`GetEarliestNextRunAt`/`GetDueSchedules` 过滤 `IS NOT NULL`，零时间 `time.Time{}` 会被当作"现在到期"重新进入热循环。
- 对抗测试 11 项全 PASS（含 race -race）：autoTrade=false pre-advance/no dispatch、already-running pre-advance、eligible UpdateNextRunAt 先于 dispatch、UpdateNext 失败不 dispatch、GetDue 失败 backoff 有界≤5/200ms、Notify 可提前唤醒 backoff、event SQL 排除+startup 清脏、invalid config 隔离、cache invalidation、callback wiring（ToggleAutoTrade + UpdateGlobalSettings onChange）、runOne 不重写 next。删 pre-consume 行 → Test_AutoTradeDisabledConsumesDue RED。完整证据与方案见 registry `SCHEDULE-HOTLOOP-1`。

## Backtest Status Management (回测状态 — 强制)

**新增回测终态时的检查清单**（BT-5 教训：DEGRADED 曾漏 4 处）：
新增或修改回测终态状态时，**必须同步更新以下所有位置**，缺任一 = 状态推送断链：

1. `status_constants.go` — 状态常量 + `isTerminalBacktestStatus()` 函数
2. `backtest_run_worker.go` — lease CASE 语句 + `pg_notify` 条件
3. `strategy_backtest_watch.go` — SSE watch 终态判断（用 `isTerminalBacktestStatus` helper）
4. `strategy_converters.go` — `backtestStatusToProto()` switch case + `IsTerminal` 字段
5. proto enum — `antv1.BacktestRunStatus` 枚举值

## Frontend Auth & Stream Error Pitfalls (FE-AUTH-1 / FE-STREAM-1)

- **Auth-free endpoint 必须前后端同步放行**：`backend/internal/interceptor/auth.go` 的 `WrapUnary` 排除列表与 `frontend/src/client/transport.ts` 的 `isAuthFree` 必须同时包含 `/refreshtoken`、`/refreshtokenfromcookie`、`/verifyemail`、`/resendverification`。任一遗漏 → 刷新后 401 或 token preflight 死锁。
- **`procedureHint` 必须包含 method name**：格式 `${service.typeName}.${method.name}`（小写）。仅用 service name 会导致 `isAuthFree` 的 `includes('refreshtoken')` 失效。
- **`"missing request message"` 不是 auth 失败**：页面刷新时 server-stream 被浏览器 abort 会产生该字符串错误，应归类为 `isLikelyStreamTransportFailure`（传输层中断），而非 `isStreamAuthFailure`。误归类会触发 "Session expired" toast。
- **前端 assets 清理**：`alphaforge-frontend` 以非 root 运行，`docker cp frontend/dist/.` 只叠加不清理，旧 chunk 会堆积。大版本部署前以 root 清理：`docker exec -u root alphaforge-frontend rm -rf /usr/share/nginx/html/assets`。

## Broker Snapshot & Stream Pitfalls (DATA-TRUTH-2~10 / LOG-UX-1)

> 实盘策略 stale / 数据断流 / 前端列空白 / 风控失明？先查 [`docs/runbook/stale-authoritative-snapshot.md`](runbook/stale-authoritative-snapshot.md)
> 完整根因 + 对抗证明见 `docs/audits/tech-debt-registry.md` DATA-TRUTH-2~10 / LOG-UX-1
> **核心原则（用户 2026-08-19 确立）**：服务器有的数据一律以服务器为准，本地只做透传与持久化，不做重算；缺失/过期必须 fail-closed，不静默转零。

### 已确认的静默失败模式

**数据源归属类（DATA-TRUTH-2/7）**：

- **MT4 `OnOrderProfit` 不填 margin/free_margin/margin_level，却覆盖权威值（DATA-TRUTH-2）** — MT4 profit stream 帧 `margin=0 / free_margin=equity / margin_level=0`（字段根本未填，`free_margin==equity` 是铁证）。旧代码用这些帧覆盖 `mt_accounts` → `MarginLevel > 0` 门槛永不成立 → MT4 爆仓预警完全不触发。**MT4 的 `AccountSummary` 含 margin（实测 718 行 margin>0），只是 profit 流不含**——错的是数据源归属，不是平台能力。修复：MT4 profit handler 改用 `fetchAndPublish` 模式，每帧调 `AccountSummary` 取权威金融值，stream 帧仅取 positions。
- **权威 RPC 失败后回退到已知不完整数据（DATA-TRUTH-7）** — MT4 `fetchAndPublish` 在 `AccountSummary` 失败时曾回退到 margin=0 的 `OnOrderProfit`；订单流用 `equity-balance` 本地重算 Profit。短暂 RPC 故障会把正确快照覆盖成假值。**通用规则：权威 RPC 失败时拒绝发布快照，不降级为假数据**。修复：RPC/app error/空结果均拒绝发布；订单流 Profit 改用 broker `GetProfit()`。

**快照生命周期类（DATA-TRUTH-5/6/10）**：

- **瞬时 pub/sub 不保留 latest → late subscriber 错过初始快照（DATA-TRUTH-10 根因①）** — `PositionSnapshotBroker` 曾是纯瞬时 pub/sub：gateway 在策略订阅前就 publish 了初始 AccountSummary，策略晚订阅 → 永远收不到 → 90s 后 `AccountSnapshotMaxAge` 判 stale → `authoritative account snapshot unavailable`。**通用规则：任何"先 publish 后 subscribe"的 broker 必须保留 latest + replay 给 late subscriber**。修复：broker retained latest + `Subscribe` 时立即 replay。
- **nil-result stream 帧被当作 stream 活跃 → 静默超时永不触发 → 权威快照不续期（DATA-TRUTH-10 根因②）** — MT4/MT5 空账户的 `OnOrderProfit` 持续发 nil-result heartbeat，旧代码把它当作 stream 活跃 → 既无金融更新又不触发 silence timeout → `AccountSummary` 只在 connect 时调一次 → `CapturedAt` 永不刷新 → 90s 后 stale。**通用规则：stream 续期必须基于有效数据帧或独立定时器，nil-result/heartbeat 帧不能算 stream 活跃**。修复：MT4/MT5 profit stream 每 45s 独立调 broker `AccountSummary` 续期。
- **quote stream 纯 Recv-error 驱动无 silence timeout → mtapi 代理层"gRPC 连接活着但不推数据"时 `Recv()` 永久阻塞 → 策略无 tick 饿死（STREAM-KEEPALIVE-1-FIX）** — STREAM-KEEPALIVE-1 "移除 quote stream no-data 超时反模式，改为纯 Recv-error 驱动"是过度修正。gRPC keepalive (30s/20s) 只检测 TCP/HTTP2 连接活着，不检测应用层有数据；mtapi 代理层可以保持 gRPC 连接活着（keepalive ping 成功）但不推 quote 数据（broker 端连接断开但 mtapi 未检测到）→ `stream.Recv()` 永久阻塞 → 策略无 tick。**通用规则：所有数据流（quote/profit/order-update）必须有 silence timeout——gRPC keepalive 不够，应用层无数据 N 秒必须 cancel+retry（不 Disconnect，避免 tear down 共享连接）**。修复：MT4/MT5 quote stream 加 45s silence timeout（同 profit stream 模式）。**测试穿透反例**：旧测试 `TestQuoteStream_RecvError_DoesNotFireOnIdle` 把 bug 行为（idle 不重连）编码为期望行为，已反转为 `TestQuoteStream_SilenceTimeout_FiresReconnect`。
- **financial-only refresh 清空持仓（DATA-TRUTH-10 连带）** — 周期 `AccountSummary` 只刷新金融字段时，若把持仓也一起覆盖会清空已有 positions。**通用规则：金融快照与持仓快照 authority 分离**——financial-only refresh 用 `PositionsAuthoritative=false` 标记，consumer 不得用它清空持仓。修复：新增 `PositionsAuthoritative` 字段。
- **Risk Gate 本地重算 equity/margin/free_margin + 固定 leverage=100（DATA-TRUTH-5）** — `MTAccountStateProvider` 曾固定 `leverage=100`，本地算 `equity=balance+profit`、`margin=notional/100`、`free_margin=equity-margin`，并在 `PositionCache` 与私有 balance/equity cache 间回退。不同 broker 的合约大小、分层杠杆、品种杠杆和保证金货币都可能不同，固定 leverage 没有正确性基础。**通用规则：Risk Gate 必须消费同一份 broker 权威快照，禁止本地重算金融字段**。修复：删除 legacy poll、balance/equity cache、固定 leverage 和本地重算；直接从 `PositionCache.GetFreshSnapshot()` 读取。
- **快照无来源/采集时间/接收时间，旧快照可无限使用（DATA-TRUTH-6）** — `PositionSnapshot` 曾无 provenance 字段，broker 断开后策略和风控可无限期使用最后一份旧快照。**通用规则：权威快照必须带 `FinancialsSource`/`CapturedAt`/`ReceivedAt`，过期（>90s）fail-closed**。修复：`PositionSnapshot` 增加 leverage/source/authoritative/captured_at；`PositionCache` 90s freshness 检查。

**事实完整性类（DATA-TRUTH-4/6/8）**：

- **双实现写不同表 + `log.Debug` 吞错误 → 净值曲线静默断供 28 天（DATA-TRUTH-4）** — 两个同名 `RecordBalanceSnapshot`：生产那份写进**不存在**的表 `account_balance_snapshots`（`to_regclass` 返回 NULL）→ 100% 失败；错误被 `log.Debug` 吞（生产 level=info）→ 零告警。分析栈全读另一张表 `account_balance_history` → 28 天零写入。**通用规则：一个事实只允许一个写入方 + 一张真相表；写入失败不得用 `log.Debug` 吞**。修复：写入方指向 `account_balance_history` + `log.Warn`；删死代码消除双实现。
- **无 session 查询返回空仓成功（DATA-TRUTH-6）** — `OpenedOrders`/`OrderHistory` 在 `exec==nil` 时曾返回 `empty,nil`，把"无法查询 broker"伪装成"broker 确认 0 订单"，污染 `OrdersTotal()`、UI、风控、CloseAll 与对账。**通用规则：无 session 必须返回明确 error，不能伪装为空仓**。修复：返回 `ErrSessionNotFound`。
- **历史快照清理操作不存在的旧表（DATA-TRUTH-8）** — writer 已修到 `account_balance_history`，但 `CleanupOldSnapshots` 仍 DELETE `account_balance_snapshots` → 清理 100% 失败、真实历史表无限增长。**通用规则：修复写入路径时必须同步检查清理/归档/迁移路径**。修复：`CleanupOldSnapshots` 改为清理 `account_balance_history`。

**保证金权威化类（DATA-TRUTH-9）**：

- **MT5 已有 `RequiredMargin` RPC 但风险链本地估算（DATA-TRUTH-9）** — `adapter/mt5.RequiredMargin` 已实现，但 `risk/rules.go` 仍按 contract size/price/leverage 本地估算 required margin；无法覆盖 broker 分层杠杆、品种保证金货币与动态规则。**通用规则：存在 broker 权威 RPC 时必须使用，禁止本地公式 fallback**。修复：MT5 `evaluatePlaceGate` 调 broker `RequiredMargin`；MT4 无等价 RPC → 显式 `Platform==mt4` capability boundary，交由 OrderSend 服务器校验，禁止套固定公式。

**前端字段对账类（LOG-UX-1）**：

- **proto 无 `message` 字段，前端却读 `dataIndex: 'message'` → 列永远空白（LOG-UX-1）** — `ScheduleRunLog` proto 只有 `error_message`/`signal_type`/`kind`/`action`/`status`，无通用 `message`。前端列 `dataIndex: 'message'` → 字段不存在 → 永远空白。**通用规则：前端列 dataIndex 必须与 proto 字段对账**，缺字段时显式 fallback 语义（错误显示 `error_message`，普通行显示 `kind/action/signal_type` 上下文，真空显示 `-`），禁止靠不存在的字段名静默空白。

**诊断真实性类（LIVE-DIAG-TRUTH-1 / SRD-1）**：

- **`RecordIndicators` 空值 early return 阻断 `ordersTotalSeen` 更新（LIVE-DIAG-TRUTH-1）** — `session_diag.go` 的 `RecordIndicators` 在 `len(values)==0` 时直接 return，导致 OnTick-only 策略的 bar 事件（空指标 map）永远不更新 `ordersTotalSeen` → 诊断页 OrdersTotal 永远停在首次非空值。**通用规则：诊断计数器（ordersTotal/evalCount 等）的更新不得被无关字段（indicator values）的空值阻断**——空值只跳过 ring buffer 写入，不跳过计数器更新。修复：`ordersTotalSeen` 始终更新，空值不烧节流窗口。
- **诊断页只显示单一 OrdersTotal，无法区分 VM vs broker vs magic（LIVE-DIAG-TRUTH-1）** — 旧诊断只暴露 `orders_total`（VM 内部值），mixed magic 场景（broker=3, magic=1, VM=0）无法展示三者差异。**通用规则：诊断必须区分 VM 视角（策略执行时看到的）、broker 视角（账户级真实持仓）、strategy magic 视角（本策略归属的）**——三者不一致即 warning，不能显示为 active 绿色。修复：proto 加 L3 字段 + 后端从 PositionCache+TradeBarrier 计算 + 前端只渲染不推断。
- **signal_generated 被显示为成交（LIVE-DIAG-TRUTH-1 rule 1）** — 信号不是成交。诊断页必须区分 `signal_generated`/`order_submitting`/`order_submitted`/`order_confirmed`/`order_rejected`/`order_outcome_unknown` 六态，signal_generated 用 default 色非绿色。
- **从瞬时 barrier state 推导 lifecycle，Release() 后真相丢失（LIVE-DIAG-TRUTH-1 返工）** — `TradeBarrier.Release()` 清空 state/ticket 到 idle/0，若 lifecycle 从 barrier state 推导，confirmed/rejected 后下一次诊断退化为 signal_generated/ticket=0。**通用规则：lifecycle 是历史事实，必须持久化在诊断状态中（`sessionDiag.lastLifecycle`/`lastBrokerTicket`），不能从瞬态状态机推导**。修复：`logOrderLifecycle` 每次过渡调 `RecordLifecycle`，`SnapshotDiag` 返回持久化值，`enrichDiagSnapshot` 只从 barrier 取 transient `ExecutionState`。
- **server-owned shared cache 放进 ActiveSession 字段导致 Register→notify 竞态（LIVE-DIAG-TRUTH-1 返工）** — `SessionRegistry.Register()` 插入 + notify watcher 后，调用点才写入 `sess.posCache`；watcher 启动时 `activeSessionToProto` 无锁读取 → data race。**通用规则：server-owned shared cache 不得作为 ActiveSession 字段，必须由 server converter 注入参数**。修复：删 `ActiveSession.posCache`，`activeSessionToProto` 加 `posCache` 参数，三处调用点传 `s.posCache`。
- **posCache=nil（paper/未接入）被前端渲染为 Stale + Warning（LIVE-DIAG-TRUTH-1 返工）** — 无数据源 ≠ 数据过期。paper mode 无 broker 数据，但前端显示 Stale/Warning 混淆语义。**通用规则：诊断必须区分 unavailable（无数据源）与 stale（有源但过期）**——unavailable 显示 N/A，不触发 warning。修复：proto 加 `data_available` 字段，前端 `!dataAvailable` 显示 N/A，warning 只在 `dataAvailable` 时检查。

### 调试路径

1. 看到 `authoritative account snapshot unavailable or stale: account=<uuid>` → 先查 `schedule_run_logs` 确认是持续还是偶发。
2. 查 `account_balance_history` 该账户最新 `recorded_at` 与 `free_margin` —— 若 `recorded_at` 远滞后 → AccountSummary 续期链断了（DATA-TRUTH-10 根因②）。
3. 查 `PositionSnapshotBroker` 是否 replay latest 给 late subscriber —— 策略启动日志应有 "replayed latest snapshot" 字样（DATA-TRUTH-10 根因①）。
4. MT4 账户 `margin=0 / free_margin=equity / margin_level=0` → 查是否用了 profit stream 帧而非 `AccountSummary`（DATA-TRUTH-2）。
5. 净值曲线断供 → 查 `account_balance_history` 是否有新写入 + 查日志是否有 `snapshot insert failed` 被 `log.Debug` 吞（DATA-TRUTH-4）。
6. 策略 `OrdersTotal()=0` 但 broker 有持仓 → 查 `OpenedOrders` 是否因无 session 返回了空仓成功（DATA-TRUTH-6）。
7. 前端日志/订单列空白 → 核对 `dataIndex` 与 proto 字段名是否一致（LOG-UX-1）。

## PG Connection Pool & Push-First LISTEN (PG-POOL-1)

- **主 pgxpool 必须配置 MaxConns**：env `DB_MAX_CONNS` 默认 25。默认 `max(4, NumCPU)` 在 4 核主机上 = 4，而 push-first refactor 后有 4 个永久 LISTEN holder（`normalizer_invalidator`、`backfiller`、`strategy_experiment_worker`、`backtest_worker`）各占 1 conn，启动即占满池。
- **每个 SSE stream 再占 1 conn**：`pgListen.Listen` per stream 会进一步耗尽池。规模上去后应使用独立 LISTEN pool 或单 listener 按 channel fan-out。
- **症状**：Login、`/healthz` 的 `pool.Ping` 在 `pool.Acquire()` 上阻塞，524/504，`/readyz` 正常（无 pool），容器 unhealthy。
