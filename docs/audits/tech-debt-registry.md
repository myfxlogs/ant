# 技术债务总账（Tech Debt Registry）

> **目的**：把全项目"以前记录过但可能没处理完"的债务**单一登记**，驱动后续逐条清理。
>
> **怎么用**：清理流程 = 取一条 → **核验当前是否仍 open**（审计跨多个日期，部分已修；`2026-08-01-acceptance` 称 3 CRITICAL 已修、48 项进 B.11 backlog）→ 仍 open 则修 → 标 ✅ 销账。
>
> **状态约定**：`❓待核` = 记录过但未对账当前代码；`🟦open` = 已核验仍存在；`✅done` = 已清。本文档建立时绝大多数为 `❓待核`——首次清理即逐条核验。
>
> **来源**：`docs/audit/*`（17 专项安全/正确性审计）+ `docs/audits/*` + `docs/adr/*` 剩余段 + `docs/roadmaps/pre-launch-assessment.md` + 代码扫描（TODO/json.Marshal/time.Ticker/eslint-disable）。
>
> **关联**：本总账是 `memory/open-items-registry.md` 的详细展开；每周一 09:07 自续对账（cron `66e2b149`）会核对其中高优项。

---

## §1 安全与正确性（来自 `docs/audit/*`，状态多为 ❓待核）

### 🔴 CRITICAL / 🔴 HIGH

| ID | 项 | 来源 | 状态 |
|----|----|------|------|
| AUD-U1 | 禁用用户仍可刷新 token 7 天（DisableUser 不撤会话/token） | user-management-audit | ❓待核 |
| AUD-MK1 | `Publish` 不校验策略模板归属（IDOR/越权发布） | marketplace-audit | ✅done（`publish.go:201-212` 已加归属校验，2026-08-06 核验）|
| AUD-MT1 | Order intent 缺 Source 字段 → 绕过全部 Gate 安全检查 | mt-gateway-audit | ✅done（#2 审计 2026-08-07 核验：live 路径 `orderRequestToIntent`(`service_orders.go:200`) 硬编码 `Source=ORDER_INTENT_SOURCE_LIVE`；Gate 的 kill-switch/autotrade/fail-closed 均对 `Source==LIVE` 生效。Source-gating 区分 LIVE(全检)/系统单(免 kill-switch)是有意设计，Source 服务端赋值、用户不可控→无 IDOR 绕过）|
| AUD-R1 | `UserRiskConfigRule` DailyPnL 空态 panic | risk-management-audit | ✅done（#2 审计 2026-08-07 核验：Gate.Evaluate 在规则链前对 LIVE 单 fail-closed——`state==nil` 直接 block，`rule=account_state_missing`，根本到不了 `state.DailyPnL.LessThan`；规则内亦有 `state != nil` 守卫。双重防，panic 不复现）|
| AUD-R2 | `MarginFloorRule` 漏合约规模 → 保证金少算 ~100x | risk-management-audit | ✅done（#2 审计 2026-08-07 核验：`rules_risksvc.go:119` 已 `vol.Mul(price).Mul(contractSize(state))`；`contractSize()`=`rules.go:38` 返回 `state.ContractSize` 或 100000 FX fallback。少算 ~100x 不复现。残留：非 FX 品种需 accountStateProvider 填 ContractSize，否则用 100000 fallback——provider 填充问题非规则 bug）|
| AUD-A6 | `GetByLogin/GetByEmail/GetByAccountNumber` 缺 token_version 校验 | auth-session-audit | ❓待核 |
| AUD-C1 | Deposit 记录写在事务外，破坏原子性 | chain-monitor-audit | ❓待核 |
| LIVE-1 | **未收盘 bar 泄漏进策略执行流**（#6 审计新发现，第一性违背——见下）。**根因**：`StartOpenBarTicker`（500ms，`mdgateway/manager_health.go:38-45`）把正在形成的 open bar 快照通过**同一** `onBar` 回调推送 → 经 `pipeline.go:84 OnBar` → `mthubSvc.PublishBar` → `barBroker` → 策略 runner。runner 的 event loop（`live_runner.go:216-228`）**不检查 `bar.Closed`**，故 open bar 也进 `handleBar`。**后果**：① 每 500ms 在同一根未定型 bar 上重复执行 OnBar；② bar 窗口（maxContextBars=500）被 open 快照淹没，数分钟内真实历史 bar 被挤出，指标对同一周期重复计数；③ OnBar 策略在未定型 bar 发信号 → 实盘与收盘-bar 回测发散 → 动摇"实盘战绩可信"产品 thesis；④ `ShadowVerifier` 会因此永远报发散而失效。**消费者全部已验证（2026-08-07 二轮核验，纠正初判）**：图表 SSE（`stream_handler.go:312 forwardBarEvents`）的唯一前端调用方 `subscribeBars` 在 `useChartData`/`PriceChart`，只存在于 `Market.tsx`/`Trading.tsx`，**二者均未进路由**（`AppRoutes.tsx` 无此 import）→ 图表 SSE 实为死代码；因子系统 `FactorEvaluator.Output()`（`registry.go:118`）**无消费者**→ 亦死。故 **open bar 当前零合法消费者，纯污染**。**修复（已锁定）**：runner event loop `live_runner.go:221` 后加 `if !bar.Closed { continue }`（一行，最小风险，立即止毒）。`StartOpenBarTicker` 本身因图表已死亦成死代码（见 CQ-9），是否删属产品决策（K 线图是否回归）。关联 ADR-0028 防线A(偷看未来 FEAT-2)、FEAT-4(回测vs实盘) | `mdgateway/{manager_health.go,pipeline.go}` + `connect/strategy/live_runner.go` | ✅done（2026-08-07 修复 + 回归测试：抽 `shouldRunOnBar(bar,symbol,timeframe)` 纯函数（LIVE-1 doc），event loop 主 symbol 分支 + extra-symbol 分支都用它；open bar 不再进 handleBar。新增 `live_runner_bar_filter_test.go` 6 例表驱动（含 **open+匹配→false** LIVE-1 回归关键）。build+strategy 测试绿、file-lines 0🔴。对抗证明：删 `bar.Closed &&` 则回归用例必红。死代码清理见 CQ-9。spec：`docs/spec/audit-live1-openbar-regression-test.md`——审计方自行实现，小任务不值外部 round-trip）|
| BT-6 | **参数链端到端测试 flaky + volume=0 偶发复现**（#5 审计主发现，🟡偏🔴）：`TestParamPipeline_FloatDefaultParam`（`tools/mql2go/e2e_param_pipeline_test.go`）同代码同命令 `-count=1` 跑 5 次 = 3 PASS/2 FAIL。flaky 根因：`makeE2EBars`（`e2e_test.go:262`）用 `time.Now().Add()` 生成 bar timestamp → 数据不确定 → 违反 spec 21 §10 Determinism Contract；Go test 缓存命中时恒显示上次结果进一步掩盖。FAIL 时 `trade[0] volume=0`（xianhua 事故精确复现）→ **参数链存在偶发 volume=0 bug**。第二个测试 `TestParamPipeline_DefaultValueWhenParamOmitted` 因合成数据 0 交易永久 `t.Skip`（假绿）。**后果**：CI（`ci.yml` `go test ./... -short`）对此测试零可靠守护——同 commit 时 PASS 时 FAIL。ADR-0028 §7 声称"端到端测试已做（坑10, 30668f64）= 永久防线三端到端打通"是**假闭环**。注：防线B `checkVolumeInvariant` 兜底抓 volume=0 标 DEGRADED（假成功不流出），但偶发 DEGRADED 体验差 + 根因被 flaky 掩盖。memory hash 笔误 30618f64→30668f64。修复方向：① `makeE2EBars` 固定 timestamp（去 flaky，符合 Determinism Contract）；② flaky 消除后定位 volume=0 偶发根因（VM 参数注入 extern vs input 覆盖 / LotsOptimized NormalizeDouble+if 执行）| `tools/mql2go/{e2e_test.go,e2e_param_pipeline_test.go}` | ✅done（2026-08-07 修复：根因 = `ir.Funcs` 是 `map[string]*FuncDef`，`compile.go:51` 遍历该 map 编译用户函数，Go map 迭代序非确定 → 当 caller（如 `CheckForOpen`）先于 callee（如 `LotsOptimized`）编译时，`compileCall` 在 `bc.Funcs` 找不到 callee → 落入 "unknown function" 盲区 → 参数被 pop 丢弃、返回值静默替换为 `NoneVal`(=0) → `OrderSend(..., LotsOptimized(), ...)` 变成 `OrderSend(..., 0, ...)` → volume=0。修复：两遍编译——第一遍预注册所有用户函数 entry PC（emit `OP_ENTER_FUNC` + 写 `bc.Funcs`），第二遍编译函数体。前向引用在第一遍后全部可解析。50 次连跑 0 失败。`compile.go` 单文件改动。注：`makeE2EBars` 的 `time.Now()` timestamp 问题仍存在（确定性违规），但非本次根因——指标计算只用 OHLCV 不用绝对时间。`makeE2EBars` 固定 timestamp 作为后续 cleanup。）|

### 🟡 MEDIUM

| ID | 项 | 来源 | 状态 |
|----|----|------|------|
| AUD-U2 | `UpdateUser` 接受任意 status 字符串 | user-management | ❓待核 |
| AUD-U3 | DisableUser/DeleteUser 不失效会话 | user-management | ❓待核（与 U1 同源）|
| AUD-MK2 | `retryFailedReversals` 用不同 idem key → 双扣 | marketplace-audit | ❓待核 |
| AUD-A7 | RefreshToken 忽略 JWT 签发错误 | auth-session | ❓待核 |
| AUD-A8 | Validate/ConsumeResetToken 非原子 TOCTOU | auth-session | ❓待核 |
| AUD-W3-2 | **NATS 无认证** | infrastructure-audit | ❓待核 |
| AUD-MT2 | PlaceOrder 错误路径 nil logger panic | mt-gateway | ❓待核 |
| AUD-W1-1 | `SubscribeOrderUpdates` 缺账户归属校验 | mt-gateway | ✅done（#3 审计 2026-08-07 核验：`stream_handlers_extra.go:20` 用认证 `interceptor.GetUserID(ctx)` + `UserOwnsAccount` 校验，非归属 `CodePermissionDenied`；订阅按 userID + 再按 accountID 过滤。四重防护，无 IDOR）|
| AUD-W1-2 | `OrderEventBroker.PublishEvent` 跨用户广播（deep-audit 称已修，需核）| mt-gateway/deep | ✅done（#3 审计 2026-08-07 核验：`types.go:199 PublishEvent(userID,ev)` 只发 `subscribers[userID]`，`Subscribe(userID)` 按 userID 注册——正确隔离，无跨用户泄漏。残留：安全性依赖调用方传对 userID）|
| AUD-W1-4 | StartStrategy live 回退到用户传入 accountID | mt-gateway | ❓待核 |
| RECON-1 | **漏单无修复机制**（#3 审计新发现，🟡 偏重）：disconnect 间隙漏掉的成交单，ant 侧 trade_records 永久缺失 → 逐笔战绩/analytics 少计（动摇"实盘战绩可信"）。根因双失：① `SyncAccountHistory`(`service/account_sync_service.go:45`)是**空壳 stub**（只 `log.Debug("requested")`，pipeline.go 在 connect/disconnect 3 处调它=假安全感）；② `ReconciliationLoop.reconcileAccount`(`mthub/reconciliation.go:82`)检测 ghost/orphan 但**只 `log.Debug` 不修复**，且 1h 历史窗口(`:93`)+Debug 级(生产常关)。账户级 balance/equity 来自 profit 流(对)，但**逐笔** trade_records 会漏。修复方向：实现 SyncAccountHistory 从 `FetchOrderHistory` upsert trade_records（DB 唯一约束 `uk_trade_record_ticket` 已在，用 ON CONFLICT）；Reconciliation log 升 Warn + 窗口延长 + 可选自动修复。施工 spec 待写（中等任务，交外部 agent）| `service/account_sync_service.go` + `mthub/reconciliation.go` | 🟦open（2026-08-07 端到端追实）|
| BT-4 | **backtest_pending 通道无 NOTIFY 生产者**（#5 审计）：worker `LISTEN backtest_pending`（`backtest_worker.go:30`）但全后端无对应 `pg_notify('backtest_pending')`——`StartBacktestRun`（`strategy_backtest_crud.go:103`）`Create` 后不 notify。worker 仅靠 30s fallback ticker（`listenTimeout`）唤醒。注释声称"replaces 3-second polling"（push-first）但实际退化为最坏 30s 延迟。修复：Create 后 `pg_notify('backtest_pending', runID)`（对齐 backtest_cancel/backtest_status 模式）| `backtest_worker.go` + repository Create | 🟦open（2026-08-07）|
| BT-5 | **DEGRADED 状态推送断链**（#5 审计）：① `UpdateAsyncFields`（`backtest_run_worker.go:212`）notify 条件 `SUCCEEDED/FAILED/CANCELED` **不含 DEGRADED** → DEGRADED 完成不触发 `backtest_status` push；② `WatchBacktestRun`（`strategy_backtest_watch.go:44,89`）终态判断同样不含 DEGRADED → SSE 收到 DEGRADED update 后不 return，流悬挂。叠加 = DEGRADED run 前端最坏 30s 延迟（靠 ticker）+ 流不结束（UI 卡"运行中"）。数据链通（:80 推了 metrics），流控断。与 ADR-0028 §7"DEGRADED 前端醒目"矛盾。修复：notify 条件 + watch 终态判断加 `"DEGRADED"` | `backtest_run_worker.go:212` + `strategy_backtest_watch.go:44,89` | ✅done（2026-08-07 修复：① `backtest_run_worker.go` lease CASE + pg_notify 条件加 DEGRADED；② `strategy_backtest_watch.go` 两处终态判断改用 `isTerminalBacktestStatus` helper（含 DEGRADED）；③ `status_constants.go` 新增 `isTerminalBacktestStatus` 函数；④ `strategy_converters.go` `backtestStatusToProto` 补 DEGRADED case。go build + go test ./internal/connect/strategy/... 绿）|
| AUD-N1 | `SendNotification` IDOR（任意用户互发）| notification-audit | ❓待核 |
| AUD-SB1 | `subscribeFree` 静默吞 plan 查询错 → 付费用户被降级 | subscription-billing | ❓待核 |
| AUD-SW1 | `BroadcastingBundle.IsExpired` 硬编码 23h | sweep-audit | ❓待核 |
| AUD-WS1 | `freezeOp` 幂等 replay 缺 undo | wallet-settlement | ❓待核 |
| AUD-WS2 | WithdrawalBuilder 非原子 bundle save + withdrawal link | wallet-settlement | ❓待核 |
| AUD-W4-1 | **AI provider base_url 无 SSRF 防护**（私有 IP 可探）| ai-agent-gateway-audit | ❓待核 |
| AUD-A1-4 | `connect/strategy` 包过大（DEFERRED，高回归风险）| deep-audit | ❓待核 |

### 🟢 LOW

| ID | 项 | 来源 | 状态 |
|----|----|------|------|
| AUD-W3-1 | migration 未包事务 | infrastructure | ❓待核 |
| AUD-W1-3 | session token INFO 级日志 | mt-gateway | ❓待核 |
| AUD-WS3 | Admin AdjustBalance txType 错分 | wallet-settlement | ❓待核 |
| AUD-W2-1 | access_token 经 URL query 传递 | frontend-security | ❓待核 |
| AUD-F3-1 | OMS 事件广播按 userID 过滤后的架构一致性 | deep-audit | ❓待核 |
| LIVE-2 | 策略订单无幂等保护（#6 审计新发现）：`submitOrder`（`live_dispatch.go:291`）构造 OrderRequest 不设 `ClientID` → `IdempotencyKey` 返回随机 UUID（`oms_writer.go:179`）→ 幂等门跳过（代码注释明说"无 client key 无幂等保证"）。单次派发无重试故无重试双发；但重复策略信号→重复下单。属已知特性，非缺陷 | connect/strategy/live_dispatch.go | 🟦open（低优，记录特性）|
| BT-1 | SimBroker swap 记账不一致：`applySwap` 仅 `checkSLTP`（engine.go:351）调用；`PositionClose`/`PositionCloseBy`/margin call 路径不记 swap → 经济少算 swap 成本（不破坏资金守恒，因 balance 与 Trade.Swap 同步为0）| strategy/backtest/broker.go | 🟦open（低，特性）|
| BT-2 | `OrderSend` margin 检查用 `b.equity`(realized, broker.go:117)，`checkMarginCall` 用 `computeEquity`(含floating, engine.go:195)——equity 定义混用，开仓 margin 检查偏宽松（不算当前浮动亏损）| strategy/backtest/{broker,engine}.go | 🟦open（低）|
| BT-3 | SimBroker 无滑点/点差成本：`broker.go:104-108` 注释明确 slippage 不加价（fill=market price），spread 仅用于 Ask/Bid 显示不影响成交 → 回测经济偏乐观 | strategy/backtest/broker.go | 🟦open（低，已知简化）|

---

## §2 架构债（双系统 / 冗余 / 设计冲突）

| ID | 项 | 位置 | 状态 |
|----|----|------|------|
| ARCH-1 | **GoExecutor 危险执行已移除**：`go run` 用户代码的路径已全部返回 `CodeUnimplemented`（`Execute`/`ExecuteLive` 对 Go 策略），MQL 走 VM。剩余仅为死 stub（`go_executor.go` 空 `struct{}`）+ 过时注释（`strategy_execution_handler.go:260` 仍称"Delegates to GoExecutor.RunLive"）| `connect/strategy/{go_executor.go,strategy_execution_handler.go:188-272}` | ✅ 功能已清（2026-08-07 #6 审计核验）；残留死代码清理=CQ |
| ARCH-2 | **双风控引擎**：`gate.Evaluate` + `risksvc.PreCheck` 每单各跑一次，违反 D6-A 单一 chokepoint | risk-gate / risksvc | 🟦open（#6 审计 2026-08-07 再次核验：`service_orders.go:119` evaluatePlaceGate 调 gate.Evaluate + `:147` submitToBroker 调 risksvc.PreCheck，确在同一 PlaceOrder 内双重评估。注：PreCheck 是 broker margin RPC，非纯冗余，但仍违 D6-A）|
| ARCH-3 | **双模板表**：`ai_strategy_templates`（AI 骨架库）vs `strategy_templates`（真实策略）；调度引擎读错表（购买→实盘缺陷 A 根因）| schedule_engine.go:226 | 🟦open（购买→实盘 spec 任务1 修）|
| ARCH-4 | **一账户一会话** vs Pro 档"5 账户/20 实盘策略"售卖冲突 → 多策略共账户需持仓归因 | session_registry.go:116 | 🟦open（P1-MKT-1）|
| ARCH-5 | 老 `kline_data` 表保留至 M9 才删（技术债窗口）| ADR-0001 | ❓待核（M9 是否到）|
| ARCH-6 | `RecognizerRegistry` 注册 11 识别器但 Pipeline 绕过（死注册）| ADR-0021 A9 | ❓待核 |

---

## §3 上线前阻塞缺口（`pre-launch-assessment.md`）

| ID | 项 | 状态 |
|----|----|------|
| LAUNCH-1 | Agent 策略生成质量基准测试套件（编译通过率/夏普>0 比例）——供给侧飞轮第一步 | 🟦open（Gap1）|
| LAUNCH-2 | marketplace 资金链路集成测试（购买/退款/订阅/试用 happy+error）| 🟦 部分（settlement 已测；购买/退款/订阅/试用待补）|
| LAUNCH-3 | GoExecutor 移除 + Bytecode VM 覆盖验证 | ✅ GoExecutor 已移除（= ARCH-1，2026-08-07 核验）；Bytecode VM 覆盖验证仍待（live 路径已确认走 VM）|

---

## §4 代码质量债

| ID | 项 | 位置 | 状态 |
|----|----|------|------|
| CQ-1 | 死代码 ~588 函数（golangci unused）| 全后端 | ❓待核（pre-launch 称存量，未清）|
| CQ-2 | 前端 knip 死代码 80 文件 + 96 exports | frontend/src | ❓待核 |
| CQ-3 | **应用层 `json.Marshal/Unmarshal` 违规**（CLAUDE.md 禁，仅 LLM API/PG JSONB 豁免）：`analytics_report_gen.go:60`（内部 metrics，疑似违例）；AI agent 工具参数解析（agent_loop/code_assist/clarification/param_proposer，疑 LLM 边界豁免，需核）；Tron 外部 API（tron_grid/tron_scan，灰区）| 见 grep | ❓待核 |
| CQ-4 | **疑似轮询 `time.Ticker`**（push-first 下应审）：`admin_monitor_handler.go:55`(5s)、`job_handler.go:124`(5s)、`strategy_experiment_handler.go:236`(5s)、`strategy_experiment_worker.go:50`(10s) 疑可改流；其余 keepalive/reconcile/链上扫描/续费属合法 timer | 见 grep | ❓待核 |
| CQ-5 | 前端 35 处 `eslint-disable react-hooks/exhaustive-deps`（多数带 REF 注释，非零容忍硬违例，但为清理项）| frontend/src | ❓待核 |
| CQ-6 | 英文 i18n 缺失：`frontend/src/i18n/resources/en/*.ts` ~15 文件头 `// TODO: Translate to en` | frontend i18n | 🟦open |
| CQ-7 | ADR-0003 LOC 合规：mt4/mt5 adapter 超标 ~10x | mt-gateway | ❓待核 |
| CQ-8 | check-file-lines：28 🟡（存量，非阻断）| marketplace-audit-m1-m12:58 | ❓待核 |
| CQ-9 | **K 线图+因子系统整条死代码**（#6 审计二轮核验，2026-08-07）：前端 K 线图已废弃——workspace chart tab 移除(`9a9dfcdf`)、`Market.tsx`/`Trading.tsx` 未进路由(`AppRoutes.tsx` 无 import)、`components/chart/{useChartData,PriceChart}.tsx` 不再挂载；后端图表 SSE `stream_handler.go forwardBarEvents`+`StartOpenBarTicker`(`manager_health.go`)只服务死前端；因子系统 `FactorEvaluator.Output()`(`factor/registry.go:118`)无消费者。整条链: `StartOpenBarTicker`→`onBar`→`PublishBar`→图表SSE + `FactorPusher`→`FactorEvaluator`→无读取。**注**:策略 runner 也挂同一 `barBroker`,但策略只要 finalized bar(open bar 泄漏见 LIVE-1)。清理范围 = 删 ticker/图表SSE/因子系统/Market+Trading页/PriceChart 组件。**是否清理取决于产品决策**(K线图/因子是否回归) | `mdgateway/manager_health.go`、`connect/system/stream_handler.go`、`factor/`、`frontend/src/{components/chart,pages/{market/Market,trading/Trading}}.tsx` | 🟦open（死代码，非阻断；与 LIVE-1 修复合并评估）|

---

## §5 迁移与 schema 债

| ID | 项 | 状态 |
|----|----|------|
| MIG-1 | 55 个 migration 缺 down 脚本（P1-5b 已审：53 纯增量无破坏性，但 down 仍缺）| 🟦open（低风险）|
| MIG-2 | ADR-0026 提出的 schema 修正（status ASSIGNED/RETIRED 去 AVAILABLE、分配 SQL 重写）是否落地 | ❓待核 |

---

## §6 文档漂移

| ID | 项 | 状态 |
|----|----|------|
| DOC-1 | **ADR-0028 §7 状态表与 commit `30618f64`（参数链 E2E）矛盾**，需核对"端到端测试"是否真闭环、刷新 §7 | 🟦open |
| DOC-2 | 各 `docs/audit/*` 发现的当前状态未对账（本总账 §1 即此对账入口）| 🟦open |
| DOC-3 | **`2026-08-01-acceptance` Gate2 称 file-lines "🟡🟢 通过"——已证伪**：`backtest_worker_vm.go` 477/450 是 🔴 阻断。CI/commit 实际被卡。 | ✅ 已核（2026-08-06）|
| DOC-4 | `mdgateway/runner.go:38` 注释称 OnBar "called when a bar is finalized"——**误导/过时**：`manager_health.go:44` 的 StartOpenBarTicker 也用同一 onBar 推未收盘 open bar（见 LIVE-1）。需改注释或分离回调 | mdgateway/runner.go | 🟦open（2026-08-07）|
| DOC-5 | `connect/strategy/strategy_execution_handler.go:260` 注释称 ExecuteLive "Delegates to GoExecutor.RunLive"——**过时**：GoExecutor 已是空 stub，Go 路径返回 CodeUnimplemented（见 ARCH-1）| connect/strategy/strategy_execution_handler.go | 🟦open（2026-08-07）|
| DOC-6 | **spec `21-backtest-replay.md` 旧 factorsvc/DSL 架构漂移**（#5 审计）：spec 描述 `factorsvc.BarSource`/`quantengine`/DSL 因子引擎 + `BarSource`/`IsReplay` 分支，但实际回测走 `strategy/backtest/` VMRunner+SimBroker（worker 层 `executeVMBacktest`）。spec §10 Determinism Contract 仍有效（但被 BT-6 违反）。活文档漂移，需更新或标注"已被 VM 架构取代" | docs/blocks/backtest-engine/spec/21-backtest-replay.md | 🟦open（2026-08-07）|
| DOC-7 | **ADR-0028 §7 "端到端测试已做"是 flaky 假闭环**（= BT-6）：§7 已落地表（commit 30668f64）声称参数链端到端测试打通，但实测 flaky（3/5 PASS）+ 一个永久 SKIP（假绿）。§7 需加注：端到端测试未真正闭环 | docs/adr/0028-backtest-reliability-validation-layers.md §7 | 🟦open（2026-08-07，随 BT-6 修复更新）|

---

## §7 功能完整性债（已 spec / 部分实现）

| ID | 项 | 状态 |
|----|----|------|
| FEAT-1 | **购买→实盘链路**：调度引擎读错表 + 事件型 schedule 无触发器 + 无订阅授权闸（spec 已定稿，实现未开始）| 🟦open（`docs/spec/purchase-to-live-link-spec.md`）|
| FEAT-2 | **ADR-0028 剩余**：防线 A 解析后校验(降级)、统计类提示(P2)、用户代码缺陷-偷看未来检测(P2)、根治报告+Admin 健康中心(P2)、判决型误报处置(现在不做)、用户侧恢复路径前端、MT golden trace 对拍(成熟期)。**#5审补充**：回测引擎层已确认无偷看未来数据泄漏（`indicators_decimal.go:visibleBars` 截断 `bars[:barIdx+1]`，shift 从末尾回溯无法访问未来 bar）；但防线层无专门"用户代码偷看未来"检测，仍 P2 待办 | 🟦open |
| FEAT-3 | `ProtectedBacktestPanel` 取码/授权模式与购买→实盘对齐 | 🟦open（P2-MKT-2）|
| FEAT-4 | 实盘战绩不可篡改（append-only + 哈希链）、回测 vs 实盘对比、walk-forward 验证 | 🟦open（roadmap Phase1.1+）|
| FEAT-5 | AI 策略迭代闭环（alpha 衰减检测 → 自动改进建议）| 🟦open（roadmap 商业深挖）|

---

## 总计

- 🔴 CRITICAL/HIGH 待核：**7**（§1）
- 🟡 MEDIUM 待核：**18**（§1）+ 架构/功能若干
- 🟢 LOW 待核：**5**（§1）
- 🟦 已确认 open：GoExecutor、双风控、双模板表、Pro 容量、购买→实盘、ADR-0028 剩余、Agent 质量基准、i18n 英文、55 down 脚本、文档漂移 等

**首批清理建议**：先核验 §1 的 7 个 🔴（acceptance 称 3 已修，需对账哪 3 个、余 4 个是否仍 open）——这是潜在上线阻塞，优先级最高。然后 §2 ARCH-1（GoExecutor）+ §3 LAUNCH-1/3。

---

## 变更日志

- 2026-08-06 建立：聚合 `docs/audit/*`(17) + `docs/audits/*` + ADR 剩余 + pre-launch + 代码扫描。状态多为 ❓待核（审计跨日期，需对账当前代码）。
- 2026-08-07 #6 实盘调度审计补全：① 新增 LIVE-1（open bar 泄漏进策略执行流，🟦open，主发现）、LIVE-2（策略订单无幂等，特性）；② ARCH-2 双风控再次核验仍 open；③ ARCH-1/LAUNCH-3 改判 ✅（GoExecutor `go run` 已移除，剩死 stub）；④ 新增 DOC-4/DOC-5 文档漂移。
- 2026-08-07 LIVE-1 二轮核验（用户指出 K 线图已废弃）：查路由 `AppRoutes.tsx` 确认 `Market.tsx`/`Trading.tsx` 未挂载、`FactorEvaluator.Output()` 无消费者 → 图表 SSE + 因子系统均死代码。**纠正初判**：open bar 当前零合法消费者(初判误为"ticker 非孤儿")。LIVE-1 修法简化为 runner 一行过滤；新增 CQ-9（K线图+因子整条死代码，清理待产品决策）。
- 2026-08-07 **#5 回测审计补全**：① **主发现 BT-6**（参数链 E2E 测试 flaky + volume=0 偶发复现，🟦open 偏重，动摇 ADR-0028 §7 假闭环）；② BT-4（backtest_pending 无 NOTIFY，push-first 退化为 30s）；③ BT-5（DEGRADED 推送断链：notify 条件 `backtest_run_worker.go:212` + watch 终态 `strategy_backtest_watch.go:44,89` 均漏 DEGRADED）；④ BT-1/2/3（SimBroker mild 简化：swap 记账/equity 定义混用/无滑点）；⑤ DOC-6（spec21 旧 factorsvc 架构漂移）/DOC-7（ADR §7 假闭环）。**ADR-0028 核心防线全部 ✅ 落位且正确**：防线B 五恒等类（手数>0/资金守恒/价格/方向/时间序，`backtest_invariants.go`）+ assessRisk 闸门（防线B前置，`buildBacktestResponse`）+ 资金守恒等式与 SimBroker 记账自洽 + DEGRADED marketplace 硬阻断（`quality.go:checkDegradedStatus`）三端到端打通。T5 确认回测引擎无偷看未来泄漏（`indicators_decimal.go:visibleBars` 截断正确，策略只见 `bars[:i+1]`）。memory hash 笔误修正 30618f64→30668f64。剩 #4 Agent循环/account-mgmt。
- 2026-08-07 **BT-5 修复**：DEGRADED 推送断链已修。4 处改动：① `backtest_run_worker.go` lease CASE + pg_notify 条件加 DEGRADED；② `strategy_backtest_watch.go` 两处终态判断改用 `isTerminalBacktestStatus` helper；③ `status_constants.go` 新增 `isTerminalBacktestStatus` 函数（含 DEGRADED）；④ `strategy_converters.go` `backtestStatusToProto` 补 DEGRADED case（前端收到正确 proto enum）。go build + go test 绿。
- 2026-08-07 **BT-6 修复**：`TestParamPipeline_FloatDefaultParam` volume=0 flaky 根因 = `ir.Funcs` 是 `map[string]*FuncDef`，`compile.go:51` 遍历该 map 编译用户函数，Go map 迭代序非确定 → 当 caller（如 `CheckForOpen`）先于 callee（如 `LotsOptimized`）编译时，`compileCall` 在 `bc.Funcs` 找不到 callee → 落入 "unknown function" 盲区 → 参数被 pop 丢弃、返回值静默替换为 `NoneVal`(=0) → `OrderSend(..., LotsOptimized(), ...)` 变成 `OrderSend(..., 0, ...)` → volume=0。修复：两遍编译——第一遍预注册所有用户函数 entry PC（emit `OP_ENTER_FUNC` + 写 `bc.Funcs`），第二遍编译函数体。前向引用在第一遍后全部可解析。50 次连跑 0 失败。`compile.go` 单文件改动。
