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
| AUD-U1 | 禁用用户仍可刷新 token 7 天（DisableUser 不撤会话/token） | user-management-audit | ✅done（2026-08-07 核验：`DisableUser`(`admin_user_handler_write.go:256`) 调 `IncrementTokenVersion`；`DeleteUser`(`:201`)同；`UpdateUser`(`:174`) status 变非 active 时同；`RefreshToken`(`auth_token.go:79`)校验 `TokenVersion`+`Status`；`Login`(`auth_handler.go:163`)校验 `Status`。禁用/删除用户无法刷新或登录）|
| AUD-MK1 | `Publish` 不校验策略模板归属（IDOR/越权发布） | marketplace-audit | ✅done（`publish.go:201-212` 已加归属校验，2026-08-06 核验）|
| AUD-MT1 | Order intent 缺 Source 字段 → 绕过全部 Gate 安全检查 | mt-gateway-audit | ✅done（#2 审计 2026-08-07 核验：live 路径 `orderRequestToIntent`(`service_orders.go:200`) 硬编码 `Source=ORDER_INTENT_SOURCE_LIVE`；Gate 的 kill-switch/autotrade/fail-closed 均对 `Source==LIVE` 生效。Source-gating 区分 LIVE(全检)/系统单(免 kill-switch)是有意设计，Source 服务端赋值、用户不可控→无 IDOR 绕过）|
| AUD-R1 | `UserRiskConfigRule` DailyPnL 空态 panic | risk-management-audit | ✅done（#2 审计 2026-08-07 核验：Gate.Evaluate 在规则链前对 LIVE 单 fail-closed——`state==nil` 直接 block，`rule=account_state_missing`，根本到不了 `state.DailyPnL.LessThan`；规则内亦有 `state != nil` 守卫。双重防，panic 不复现）|
| AUD-R2 | `MarginFloorRule` 漏合约规模 → 保证金少算 ~100x | risk-management-audit | ✅done（#2 审计 2026-08-07 核验：`rules_risksvc.go:119` 已 `vol.Mul(price).Mul(contractSize(state))`；`contractSize()`=`rules.go:38` 返回 `state.ContractSize` 或 100000 FX fallback。少算 ~100x 不复现。残留：非 FX 品种需 accountStateProvider 填 ContractSize，否则用 100000 fallback——provider 填充问题非规则 bug）|
| AUD-A6 | `GetByLogin/GetByEmail/GetByAccountNumber` 缺 token_version 校验 | auth-session-audit | ✅done（2026-08-07 核验：A6/A7/A8 三件均已修复。① A6：`user_repo.go:66-150` 三个查询均 SELECT+Scan `token_version`；② A7：`auth_token.go:92-99` RefreshToken/RefreshTokenFromCookie 正确处理 JWT 签发错误返回 `CodeInternal`；③ A8：`password_reset_repo.go:71-86` `ValidateResetToken` 原子 `UPDATE...SET consumed=TRUE...RETURNING` 消除 TOCTOU）|
| AUD-C1 | Deposit 记录写在事务外，破坏原子性 | chain-monitor-audit | ✅done（2026-08-07 核验：`ConfirmDeposit`(`deposit_service.go:188-234`)在单个 tx 内原子执行 deposit INSERT + wallet `AdjustBalanceTx`。`MANUAL_REVIEW` fallback 路径(`monitor.go:310/330`)在 tx 外写入但无 wallet credit，无原子性破坏。`MarkReceivedUSDT`(`:347`)是地址元数据更新，非 deposit 记录）|
| LIVE-1 | **未收盘 bar 泄漏进策略执行流**（#6 审计新发现，第一性违背——见下）。**根因**：`StartOpenBarTicker`（500ms，`mdgateway/manager_health.go:38-45`）把正在形成的 open bar 快照通过**同一** `onBar` 回调推送 → 经 `pipeline.go:84 OnBar` → `mthubSvc.PublishBar` → `barBroker` → 策略 runner。runner 的 event loop（`live_runner.go:216-228`）**不检查 `bar.Closed`**，故 open bar 也进 `handleBar`。**后果**：① 每 500ms 在同一根未定型 bar 上重复执行 OnBar；② bar 窗口（maxContextBars=500）被 open 快照淹没，数分钟内真实历史 bar 被挤出，指标对同一周期重复计数；③ OnBar 策略在未定型 bar 发信号 → 实盘与收盘-bar 回测发散 → 动摇"实盘战绩可信"产品 thesis；④ `ShadowVerifier` 会因此永远报发散而失效。**消费者全部已验证（2026-08-07 二轮核验，纠正初判）**：图表 SSE（`stream_handler.go:312 forwardBarEvents`）的唯一前端调用方 `subscribeBars` 在 `useChartData`/`PriceChart`，只存在于 `Market.tsx`/`Trading.tsx`，**二者均未进路由**（`AppRoutes.tsx` 无此 import）→ 图表 SSE 实为死代码；因子系统 `FactorEvaluator.Output()`（`registry.go:118`）**无消费者**→ 亦死。故 **open bar 当前零合法消费者，纯污染**。**修复（已锁定）**：runner event loop `live_runner.go:221` 后加 `if !bar.Closed { continue }`（一行，最小风险，立即止毒）。`StartOpenBarTicker` 本身因图表已死亦成死代码（见 CQ-9），是否删属产品决策（K 线图是否回归）。关联 ADR-0028 防线A(偷看未来 FEAT-2)、FEAT-4(回测vs实盘) | `mdgateway/{manager_health.go,pipeline.go}` + `connect/strategy/live_runner.go` | ✅done（2026-08-07 修复 + 回归测试：抽 `shouldRunOnBar(bar,symbol,timeframe)` 纯函数（LIVE-1 doc），event loop 主 symbol 分支 + extra-symbol 分支都用它；open bar 不再进 handleBar。新增 `live_runner_bar_filter_test.go` 6 例表驱动（含 **open+匹配→false** LIVE-1 回归关键）。build+strategy 测试绿、file-lines 0🔴。对抗证明：删 `bar.Closed &&` 则回归用例必红。死代码清理见 CQ-9。spec：`docs/spec/audit-live1-openbar-regression-test.md`——审计方自行实现，小任务不值外部 round-trip）|
| BT-6 | **参数链端到端测试 flaky + volume=0 偶发复现**（#5 审计主发现，🟡偏🔴）：`TestParamPipeline_FloatDefaultParam`（`tools/mql2go/e2e_param_pipeline_test.go`）同代码同命令 `-count=1` 跑 5 次 = 3 PASS/2 FAIL。flaky 根因：`makeE2EBars`（`e2e_test.go:262`）用 `time.Now().Add()` 生成 bar timestamp → 数据不确定 → 违反 spec 21 §10 Determinism Contract；Go test 缓存命中时恒显示上次结果进一步掩盖。FAIL 时 `trade[0] volume=0`（xianhua 事故精确复现）→ **参数链存在偶发 volume=0 bug**。第二个测试 `TestParamPipeline_DefaultValueWhenParamOmitted` 因合成数据 0 交易永久 `t.Skip`（假绿）。**后果**：CI（`ci.yml` `go test ./... -short`）对此测试零可靠守护——同 commit 时 PASS 时 FAIL。ADR-0028 §7 声称"端到端测试已做（坑10, 30668f64）= 永久防线三端到端打通"是**假闭环**。注：防线B `checkVolumeInvariant` 兜底抓 volume=0 标 DEGRADED（假成功不流出），但偶发 DEGRADED 体验差 + 根因被 flaky 掩盖。memory hash 笔误 30618f64→30668f64。修复方向：① `makeE2EBars` 固定 timestamp（去 flaky，符合 Determinism Contract）；② flaky 消除后定位 volume=0 偶发根因（VM 参数注入 extern vs input 覆盖 / LotsOptimized NormalizeDouble+if 执行）| `tools/mql2go/{e2e_test.go,e2e_param_pipeline_test.go}` | ✅done（2026-08-07 修复：根因 = `ir.Funcs` 是 `map[string]*FuncDef`，`compile.go:51` 遍历该 map 编译用户函数，Go map 迭代序非确定 → 当 caller（如 `CheckForOpen`）先于 callee（如 `LotsOptimized`）编译时，`compileCall` 在 `bc.Funcs` 找不到 callee → 落入 "unknown function" 盲区 → 参数被 pop 丢弃、返回值静默替换为 `NoneVal`(=0) → `OrderSend(..., LotsOptimized(), ...)` 变成 `OrderSend(..., 0, ...)` → volume=0。修复：两遍编译——第一遍预注册所有用户函数 entry PC（emit `OP_ENTER_FUNC` + 写 `bc.Funcs`），第二遍编译函数体。前向引用在第一遍后全部可解析。50 次连跑 0 失败。`compile.go` 单文件改动。注：`makeE2EBars` 的 `time.Now()` timestamp 问题仍存在（确定性违规），但非本次根因——指标计算只用 OHLCV 不用绝对时间。`makeE2EBars` 固定 timestamp 作为后续 cleanup。）|

### 🟡 MEDIUM

| ID | 项 | 来源 | 状态 |
|----|----|------|------|
| AUD-U2 | `UpdateUser` 接受任意 status 字符串 | user-management | ✅done（2026-08-07 核验：`admin_user_handler_write.go:144-148` 用 `validStatus()` 校验，只允许 `active/disabled/suspended/pending`，非法值返回 `CodeInvalidArgument`）|
| AUD-U3 | DisableUser/DeleteUser 不失效会话 | user-management | ✅done（与 U1 同源，2026-08-07 核验通过，见 U1）|
| AUD-MK2 | `retryFailedReversals` 用不同 idem key → 双扣 | marketplace-audit | ✅done（2026-08-07 核验：`retryFailedReversals`(`settlement.go:197/218`)重用原始 idem key `IdemKeyRev+purchaseIDStr`/`IdemKeyFeeRev+purchaseIDStr`——已扣款的重试被 idempotent replay 拒绝，无双扣。`IdemKeyRevRetry`/`IdemKeyFeeRevRetry` 是死代码已删除）|
| AUD-A7 | RefreshToken 忽略 JWT 签发错误 | auth-session | ✅done（2026-08-07 核验：`auth_token.go:92-99` RefreshToken/RefreshTokenFromCookie 正确处理 JWT 签发错误返回 `CodeInternal`）|
| AUD-A8 | Validate/ConsumeResetToken 非原子 TOCTOU | auth-session | ✅done（2026-08-07 核验：`password_reset_repo.go:71-86` `ValidateResetToken` 原子 `UPDATE...SET consumed=TRUE...RETURNING` 消除 TOCTOU）|
| AUD-W3-2 | **NATS 无认证** | infrastructure-audit | ✅done（2026-08-07 核验：`storage/nats/client.go:16-19,68-70` 已支持 `CredsFile` 认证。是否启用是部署配置，非代码缺陷）|
| AUD-MT2 | PlaceOrder 错误路径 nil logger panic | mt-gateway | ✅done（2026-08-07 核验：`mthub/service_orders.go` 所有 `s.logger` 访问均有 `if s.logger != nil` 守卫（L54/126/168/172/266/281）；MT4/MT5 `Gateway.PlaceOrder` 不使用 `g.log`。无 nil panic）|
| AUD-W1-1 | `SubscribeOrderUpdates` 缺账户归属校验 | mt-gateway | ✅done（#3 审计 2026-08-07 核验：`stream_handlers_extra.go:20` 用认证 `interceptor.GetUserID(ctx)` + `UserOwnsAccount` 校验，非归属 `CodePermissionDenied`；订阅按 userID + 再按 accountID 过滤。四重防护，无 IDOR）|
| AUD-W1-2 | `OrderEventBroker.PublishEvent` 跨用户广播（deep-audit 称已修，需核）| mt-gateway/deep | ✅done（#3 审计 2026-08-07 核验：`types.go:199 PublishEvent(userID,ev)` 只发 `subscribers[userID]`，`Subscribe(userID)` 按 userID 注册——正确隔离，无跨用户泄漏。残留：安全性依赖调用方传对 userID）|
| AUD-W1-4 | StartStrategy live 回退到用户传入 accountID | mt-gateway | ✅done（2026-08-07 核验：`strategy_active_handlers.go:290-310` `resolveModeAndAccount` live 模式用 `accountLookup(ctx, uid)` 服务端查找，覆盖 `cfg.AccountID=mt4ID`，用户传入值被忽略）|
| RECON-1 | **漏单无修复机制**（#3 审计新发现，🟡 偏重）：disconnect 间隙漏掉的成交单，ant 侧 trade_records 永久缺失 → 逐笔战绩/analytics 少计（动摇"实盘战绩可信"）。根因双失：① `SyncAccountHistory`(`service/account_sync_service.go:45`)是**空壳 stub**（只 `log.Debug("requested")`，pipeline.go 在 connect/disconnect 3 处调它=假安全感）；② `ReconciliationLoop.reconcileAccount`(`mthub/reconciliation.go:82`)检测 ghost/orphan 但**只 `log.Debug` 不修复**，且 1h 历史窗口(`:93`)+Debug 级(生产常关)。账户级 balance/equity 来自 profit 流(对)，但**逐笔** trade_records 会漏。**修复（2026-08-07）**：① `SyncAccountHistory` 实现为异步 goroutine：解析 UUID → `GetLastSyncTime` 增量起点 → `mthubSvc.OrderHistory` 拉取 → `orderRecordToTradeRecord` 转换 → `tradeRecordRepo.BatchCreate` upsert（ON CONFLICT 已在）；② `reconciliation.go` ghost/orphan log 从 `Debug` 升 `Warn`（生产可见）+ `accountID` 加入日志字段；③ 历史窗口从 1h 扩到 24h（catch longer disconnect gaps）。REUSE: `tradeRecordRepo.BatchCreate` ON CONFLICT upsert + `mthubSvc.OrderHistory` + `GetLastSyncTime` 增量同步。go build + go test 全绿 | `service/account_sync_service.go` + `mthub/reconciliation.go` | ✅done（2026-08-07）|
| BT-4 | **backtest_pending 通道无 NOTIFY 生产者**（#5 审计）：worker `LISTEN backtest_pending`（`backtest_worker.go:30`）但全后端无对应 `pg_notify('backtest_pending')`——`StartBacktestRun`（`strategy_backtest_crud.go:103`）`Create` 后不 notify。worker 仅靠 30s fallback ticker（`listenTimeout`）唤醒。注释声称"replaces 3-second polling"（push-first）但实际退化为最坏 30s 延迟。**根因**：`BacktestRunRepository.Create`（`backtest_run_repository.go:65-123`）成功 INSERT 后未发 `pg_notify('backtest_pending', runID)`，与同文件 `RequestCancel`（:109 `pg_notify('backtest_cancel')`）和 `UpdateAsyncFields`（:213 `pg_notify('backtest_status')`）模式不一致。**修复**：在 `Create` 成功插入后加 `pg_notify('backtest_pending', out.String())`（repository 层，对齐 cancel/status 模式）。worker 现在被 NOTIFY 立即唤醒而非等 30s ticker。**对抗证明**：`grep -rn "pg_notify.*backtest_pending" backend/` 确认此前零生产者；修复后唯一生产者在 `Create` 方法。`go build` + `go test ./internal/connect/strategy/... ./internal/repository/...` 绿；`check-file-lines --strict` 0 errors | `backtest_run_repository.go` | ✅done（2026-08-07）|
| AGT-2 | **agent 回测绕过 ADR-0028 防线B**（#4 审计主发现，🟡偏重）：agent 生成策略的回测 `backtest_helpers.go:buildBacktestResultProto` 直接转 proto，**不经** checkVolumeInvariant/checkCapitalConservation/assessRisk/DEGRADED（agent 包零引用防线B）。用户在 agent 对话迭代策略，LLM 基于未校验 metrics 决策——xianhua volume=0 在 agent 内无"不可信"标记，LLM 在垃圾数据上迭代。违反 ADR-0028 §4.3"防线B 每次回测后触发"。7-gate 只在晋升评估跑，不在 agent 生成。**#4 补细节确认**：bridge backtest（`gateway_backtest.go:92-94`）同样绕过防线B（BRIDGE-2），但 bridge 结果不喂 LLM 迭代，严重性低。**修复（2026-08-07）**：① 提取不变量逻辑到共享包 `strategy/backtest/invariants.go`（`ValidateInvariants` 返回 `(bool, []*BlindSpot)`）；② `internal/connect/strategy/backtest_invariants.go` 改为 thin wrapper 委托共享包（消除重复）；③ `backtest_helpers.go:buildBacktestResultProto` 末尾调 `backtest.ValidateInvariants`，填充 `AgentBacktestResult.IsReliable` + `InvariantBlindSpots`；④ proto 加 `is_reliable` + `invariant_blind_spots` 字段（#16/#17）；⑤ `agent_tools_write.go` 的 `backtestSummary` 加 `IsReliable`/`InvariantWarnings`，工具输出反馈 LLM `is_reliable=false` + 违规描述。回归测试：`TestBuildBacktestResultProto_InvariantsPopulated` 4 子测试（valid/zero_volume/capital_violation/empty）。go build + go test 全绿 | `internal/agent/{backtest_helpers,agent_tools_write}.go` + `strategy/backtest/invariants.go` + `internal/connect/strategy/backtest_invariants.go` + `proto/ant/v1/agent_gateway.proto` | ✅done（2026-08-07）|
| AGT-1 | **lookahead_scanner + evalLookAhead 旧 DSL 残留**（#4 审计）：扫描 DSL 因子表达式(`close[t+1]`/`ref()`)，但当前 agent 生成 Python 源 → `evalLookAhead`/`evalCompliance` 当 `expression==""` 直接 skip("no DSL expression — skipped for MQL/code strategy")。**对当前 Python/MQL 策略零检测**。FEAT-2"用户代码偷看未来检测 P2 待做"属实——现有 scanner 名存实亡，需为 VM/MQL/Python 路径重写（AST 级 shift 检查，非正则 DSL）| `internal/ai/{lookahead_scanner,gate_pipeline}.go` | 🟦open（2026-08-07，= FEAT-2 实证）|
| BT-5 | **DEGRADED 状态推送断链**（#5 审计）：① `UpdateAsyncFields`（`backtest_run_worker.go:212`）notify 条件 `SUCCEEDED/FAILED/CANCELED` **不含 DEGRADED** → DEGRADED 完成不触发 `backtest_status` push；② `WatchBacktestRun`（`strategy_backtest_watch.go:44,89`）终态判断同样不含 DEGRADED → SSE 收到 DEGRADED update 后不 return，流悬挂。叠加 = DEGRADED run 前端最坏 30s 延迟（靠 ticker）+ 流不结束（UI 卡"运行中"）。数据链通（:80 推了 metrics），流控断。与 ADR-0028 §7"DEGRADED 前端醒目"矛盾。修复：notify 条件 + watch 终态判断加 `"DEGRADED"` | `backtest_run_worker.go:212` + `strategy_backtest_watch.go:44,89` | ✅done（2026-08-07 修复：① `backtest_run_worker.go` lease CASE + pg_notify 条件加 DEGRADED；② `strategy_backtest_watch.go` 两处终态判断改用 `isTerminalBacktestStatus` helper（含 DEGRADED）；③ `status_constants.go` 新增 `isTerminalBacktestStatus` 函数；④ `strategy_converters.go` `backtestStatusToProto` 补 DEGRADED case。go build + go test ./internal/connect/strategy/... 绿）|
| AUD-N1 | `SendNotification` IDOR（任意用户互发）| notification-audit | ✅done（2026-08-07 核验：`notification/handler.go:170-182` 先取 `callerUID` 再查 `is_admin`，非 admin 返回 `CodePermissionDenied`）|
| AUD-SB1 | `subscribeFree` 静默吞 plan 查询错 → 付费用户被降级 | subscription-billing | ✅done（2026-08-07 核验：`subscription_service.go:175-184` `GetActiveSubscriptionTx`/`GetPlanByID` 错误均正确传播，无静默吞）|
| AUD-SW1 | `BroadcastingBundle.IsExpired` 硬编码 23h | sweep-audit | ✅done（2026-08-07 核验：`repo.go:248-250` 23 是 fallback 默认值；`worker.go:193-198` + `builder.go:101-106` 从 `system_config` 表 `sweep_raw_tx_expiry_hours` 读取，可配置）|
| AUD-WS1 | `freezeOp` 幂等 replay 缺 undo | wallet-settlement | ✅done（2026-08-07 核验：`wallet_freeze_repo.go:99-116` `ErrIdempotentReplay` 时有完整 undo——根据 freeze/txType 反转 balance/frozen_balance）|
| AUD-WS2 | WithdrawalBuilder 非原子 bundle save + withdrawal link | wallet-settlement | ✅done（2026-08-07 核验：`withdrawal_builder.go:207-224` `saveBundleAndLinkWithdrawal` 在单事务内原子执行 bundle save + withdrawal link）|
| AUD-W4-1 | **AI provider base_url 无 SSRF 防护**（私有 IP 可探）| ai-agent-gateway-audit | ✅done（2026-08-07 修复：① 导出 `ValidateBaseURL`（`discovery.go:39`）——校验 scheme http/https + 拦截私有/回环 IP + 已知内部主机名；② `UpdateSystemAIConfig`（`system_ai_handler.go:74`）从弱 `url.Parse` 升级为 `ValidateBaseURL`；③ `CreateProvider`/`UpdateProvider`（`ai_gateway_handler.go:145/192`）加 `ValidateBaseURL`；④ 聊天路径 `resolveUserProviders`（`chat_failover.go:261`）运行时校验 base_url，跳过私有点）|
| AUD-A1-4 | `connect/strategy` 包过大（DEFERRED，高回归风险）| deep-audit | ✅done（2026-08-07 核验：`check-file-lines` 对 `connect/strategy` 零报告，无超标文件）|
| BRIDGE-1 | **bridge.go nil pointer panic**（#4 补细节）：`TranslateWithRetry`（`bridge.go:79-99`）首次 LLM 调用失败时 `result` 仍为 nil，`continue` 跳到 attempt 2，line 85 `buildBridgeRetryPrompt(mqlSource, result.PythonSource, ...)` → nil dereference panic。**修复**：`bridge.go:82` 条件从 `attempt == 1` 改为 `attempt == 1 || result == nil`，LLM 失败后重试时用原始 prompt（无 prevPython 可反馈）。go build + go test 绿 | `internal/agent/bridge.go:82` | ✅done（2026-08-07）|
| CREDIT-1 | **服务重启导致 credit 泄漏**（#4 补细节）：`holds` 是 in-memory `map[string]decimal.Decimal`（`credit_service.go:50`）。PreHold 在 DB `HoldCredits` 冻结额度，in-memory 记录用于 Settle/ReleaseHold。服务重启 mid-session → in-memory 丢失 → Settle/ReleaseHold 找到 `holdAmount=0` → DB 冻结额度永久 orphaned。**修复（2026-08-07）**：① migration 261 给 `credit_transactions` 加 `session_id` 列；② `HoldCredits` 存 session_id；③ `GetStaleHolds` 查无对应 `ai_usage`/`ai_release` 的 `ai_hold`；④ `RestoreHolds` 在启动时调用，释放冻结额度回用户余额 + `MarkHoldSettled` 插入 `ai_release` 标记已处理 | `internal/service/credit_service.go:57-86` | ✅done（2026-08-07）|

### 🟢 LOW

| ID | 项 | 来源 | 状态 |
|----|----|------|------|
| BRIDGE-2 | **bridge backtest 绕过防线B**（#4 补细节）：`gateway_backtest.go:92-94` 的 `validateBacktest` 只检查 `btErr != nil`，不跑 `checkVolumeInvariant`/`assessRisk`/DEGRADED。与 AGT-2 同源，但 bridge 结果只用于 `semanticDiff`（覆盖率对比），不喂 LLM 迭代 metrics，严重性低 | `internal/agent/gateway_backtest.go:92-94` | 🟦open（低，2026-08-07）|
| CREDIT-2 | **CheckBalance fail-open on DB errors**（#4 补细节）：`credit_service.go:127` `if err != nil { return nil }` — DB 错误时余额检查通过，用户可在无额度时使用 AI。可用性优先设计，非 bug | `internal/service/credit_service.go:127` | 🟦open（低，特性，2026-08-07）|
| AUD-W3-1 | migration 未包事务 | infrastructure | ✅done（2026-08-07 核验：`docker-entrypoint.sh:53` 用 `psql -c "BEGIN;" -f "$file" -c "COMMIT;"` 包事务执行每个 migration 文件）|
| AUD-W1-3 | session token INFO 级日志 | mt-gateway | ✅done（2026-08-07 核验：`auth_handler.go`/`auth_token.go`/`auth.go` 全部 log 只用 `zap.String("userID",...)`/`zap.Error(err)`，从不 log token 值。无敏感信息泄漏）|
| AUD-WS3 | Admin AdjustBalance txType 错分 | wallet-settlement | ✅done（2026-08-07 核验：`wallet_handler.go:146` 用 `txType="adjustment"` 正确分类，`AdjustBalanceTx` 透传 txType 到 `ledgerChainInsert`，无错分）|
| AUD-W2-1 | access_token 经 URL query 传递 | frontend-security | 🟦open（低，已知设计权衡，2026-08-07：`auth.go:112` 接受 URL query `access_token`，是 SSE/EventSource 无法设 Authorization header 的标准方案。风险：token 可能出现在 proxy 日志/浏览器历史。缓解：access_token TTL=15min）|
| AUD-F3-1 | OMS 事件广播按 userID 过滤后的架构一致性 | deep-audit | ✅done（2026-08-07 核验：`OrderEventBroker.PublishEvent(userID,ev)`(`types.go:199`) 按 userID 隔离，`Subscribe(userID)` 按 userID 注册，`oms_writer.go:170` 调 `PublishEvent(usermgr.GetUserID(ctx),oev)`。架构一致，无跨用户泄漏）|
| LIVE-2 | 策略订单无幂等保护（#6 审计新发现）：`submitOrder`（`live_dispatch.go:291`）构造 OrderRequest 不设 `ClientID` → `IdempotencyKey` 返回随机 UUID（`oms_writer.go:179`）→ 幂等门跳过（代码注释明说"无 client key 无幂等保证"）。单次派发无重试故无重试双发；但重复策略信号→重复下单。属已知特性，非缺陷 | connect/strategy/live_dispatch.go | 🟦open（低优，记录特性）|
| BT-1 | SimBroker swap 记账不一致：`applySwap` 仅 `checkSLTP`（engine.go:351）调用；`PositionClose`/`PositionCloseBy`/margin call 路径不记 swap → 经济少算 swap 成本（不破坏资金守恒，因 balance 与 Trade.Swap 同步为0）| strategy/backtest/broker.go | 🟦open（低，特性）|
| BT-2 | `OrderSend` margin 检查用 `b.equity`(realized, broker.go:117)，`checkMarginCall` 用 `computeEquity`(含floating, engine.go:195)——equity 定义混用，开仓 margin 检查偏宽松（不算当前浮动亏损）| strategy/backtest/{broker,engine}.go | 🟦open（低）|
| BT-3 | SimBroker 无滑点/点差成本：`broker.go:104-108` 注释明确 slippage 不加价（fill=market price），spread 仅用于 Ask/Bid 显示不影响成交 → 回测经济偏乐观 | strategy/backtest/broker.go | 🟦open（低，已知简化）|

---

## §2 架构债（双系统 / 冗余 / 设计冲突）

| ID | 项 | 位置 | 状态 |
|----|----|------|------|
| ARCH-1 | **GoExecutor 危险执行已移除**：`go run` 用户代码的路径已全部返回 `CodeUnimplemented`（`Execute`/`ExecuteLive` 对 Go 策略），MQL 走 VM。剩余仅为死 stub（`go_executor.go` 空 `struct{}`）+ 过时注释（`strategy_execution_handler.go:260` 仍称"Delegates to GoExecutor.RunLive"）| `connect/strategy/{go_executor.go,strategy_execution_handler.go:188-272}` | ✅ 功能已清（2026-08-07 #6 审计核验）；残留死代码清理=CQ |
| ARCH-2 | **双风控引擎**：`gate.Evaluate` + `risksvc.PreCheck` 每单各跑一次，违反 D6-A 单一 chokepoint | risk-gate / risksvc | ✅done（2026-08-09：移除 `submitToBroker` 中的 `risksvc.PreCheck` 调用（含 `RequiredMargin` RPC + 4 项检查），将等效规则注册到 Gate：`MaxPositionCount{Max:20}` + `MaxLotSize{MaxLots:100000}` + `MarginPreCheck{MaxMarginRatio:0.80}`（`handlers_strategy.go:118-120`）。现在 `PlaceOrder` 风控路径 = `preTradeChecks`（killSwitch/guard/owner/idem/reconcile/rateLimit）→ `evaluatePlaceGate`（gate.Evaluate 单一 chokepoint，含 killSwitch/autotrade/fail-closed + 所有规则）→ `submitToBroker`（纯执行）。3 个 submitToBroker 测试更新为验证不再做 margin precheck。REUSE: `MaxPositionCount`@`rules.go:67` + `MaxLotSize`@`rules.go:47` + `MarginPreCheck`@`rules.go:305`。对抗证明：`TestSubmitToBroker_MarginPrecheckReject` 验证 submitToBroker 不再因 margin 不足拒绝（Gate 负责此检查）|
| ARCH-3 | **双模板表**：`ai_strategy_templates`（AI 骨架库）vs `strategy_templates`（真实策略）；调度引擎读错表（购买→实盘缺陷 A 根因）。**修复（2026-08-07）**：`ScheduleEngine` 改用 `TemplateCodeReader` 接口（`GetTemplate(ctx,id,userID)`），注入 `StrategySvc`（读 `strategy_templates`），不再用 `AIStrategyTemplatesRepository`（读 `ai_strategy_templates`）。`dispatch()` 用 `tpl.Code` 替代 `tpl.CodeSkeleton`。`handlers.go` 清理 `TemplatesRepo` 传递链 | `schedule_engine.go:59-61,229-241` | ✅done（2026-08-07）|
| ARCH-4 | **一账户一会话** vs Pro 档"5 账户/20 实盘策略"售卖冲突 → 多策略共账户需持仓归因 | session_registry.go:116 | 🟦open（P1-MKT-1）|
| ARCH-5 | 老 `kline_data` 表保留至 M9 才删（技术债窗口）| ADR-0001 | ✅done（2026-08-07 核验：migration 166 `DROP TABLE IF EXISTS kline_data CASCADE`，migration 003 已改为 stub `-- table dropped, see 166_cleanup_dead_tables`）|
| ARCH-6 | `RecognizerRegistry` 注册 11 识别器但 Pipeline 绕过（死注册）| ADR-0021 A9 | ✅done（2026-08-07 核验：`RecognizerRegistry` 在当前代码中不存在，gate pipeline 用 switch-case 模式（`gate_pipeline.go:84-99`），无死注册）|

---

## §3 上线前阻塞缺口（`pre-launch-assessment.md`）

| ID | 项 | 状态 |
|----|----|------|
| LAUNCH-1 | Agent 策略生成质量基准测试套件（编译通过率/夏普>0 比例）——供给侧飞轮第一步 | ✅done（2026-08-09：20 策略用例覆盖 trend/mean_reversion/breakout/grid/multi_tf/oscillator。编译通过率 19/19=100%（≥90%✅）；回测完成率 19/19=100%（≥80%✅）；Sharpe>0 比例 11/19=58%（≥50%✅）。CI 快速子集 `quality_benchmark_compile_test.go`（无 benchmark tag）；完整套件 `quality_benchmark_test.go`（`-tags=benchmark`）。发现并修复 3 个工具链 Bug：①`ctx.broker.close_all()` 未映射→新增 `CloseAll` builtin+映射；②`ctx.positions()` 未映射→映射到 `PositionsTotal`；③市价单 `req.Price=0` 未填充当前市价→`SimBroker.OrderSend` 修复。REUSE: `PositionsTotal`@`vm_builtin_trade.go:112` + `SimBroker.PositionClose`@`broker.go:138`。NEW: `CloseAll` builtin@`vm_builtin_trade.go:569`）|
| LAUNCH-2 | marketplace 资金链路集成测试（购买/退款/订阅/试用 happy+error）| ✅done（2026-08-09：15 集成测试覆盖 purchase(once/subscription)/settle/refund(frozen/settled)/subscribe(free/paid rejected)/idempotency/insufficient balance/self-buy/double refund/refund window expired/fee tier/full lifecycle。发现并修复 2 个生产 Bug：①`refund.go:69` SELECT `idempotency_key`（不存在列）→ `idem_key`；②`refund.go:70` 用 `IdemKeyBuy+targetUserID`（publisher ID）查交易→改用 `IdemKeyBuy+subIdemKey`（subscription idempotency_key），原逻辑永远找不到购买交易→退款全挂。REUSE: `PurchaseStrategy`@`purchase.go:49` + `RefundPurchase`@`refund.go:28` + `SettleExpired`@`settlement.go:34` + `Subscribe`@`service_subscription.go:32`。NEW: `money_flow_integration_test.go`（`//go:build integration`）|
| LAUNCH-3 | GoExecutor 移除 + Bytecode VM 覆盖验证 | ✅ GoExecutor 已移除（= ARCH-1，2026-08-07 核验）；Bytecode VM 覆盖验证仍待（live 路径已确认走 VM）|

---

## §4 代码质量债

| ID | 项 | 位置 | 状态 |
|----|----|------|------|
| CQ-1 | 死代码 ~588 函数（golangci unused）| 全后端 | 🟦open（2026-08-09 核验：存量死代码，非阻断。golangci unused 需全量 lint 运行确认当前数量。清理属增量优化，优先级低）|
| CQ-2 | 前端 knip 死代码 80 文件 + 96 exports | frontend/src | 🟦open（2026-08-09 核验：存量前端死代码，非阻断。需运行 knip 确认当前数量。含 CQ-9 K线图/因子系统死代码）|
| CQ-3 | **应用层 `json.Marshal/Unmarshal` 违规**（CLAUDE.md 禁，仅 LLM API/PG JSONB 豁免）：`analytics_report_gen.go:60`（内部 metrics，疑似违例）；AI agent 工具参数解析（agent_loop/code_assist/clarification/param_proposer，疑 LLM 边界豁免，需核）；Tron 外部 API（tron_grid/tron_scan，灰区）| 见 grep | ✅done（2026-08-09：全量核验 20 文件 48 处。分类：① LLM API 边界豁免（chat.go/chat_stream.go/discovery.go/param_proposer.go/clarification.go/agent_loop.go/agent_loop_helpers.go/code_assist_handler.go/analytics_report_gen.go — 构建 LLM prompt 或解析 LLM 响应）；② 外部 HTTP API 豁免（tron_grid.go/tron_scan.go — TronGrid/TronScan 第三方 API）；③ protojson 非违规（generator_agent.go/gateway_memory_handlers.go/strategy_template_handlers.go — 用 protojson 非 encoding/json）；④ 旧格式迁移豁免（schedule_proto_migrate.go/notification_proto_migrate.go — DB 旧 JSON→proto 迁移）；⑤ 测试豁免（chain_test.go/trade_event_store_test.go/alignment_test.go）；⑥ hooks.go 主动规避（注释明确避免 json.Unmarshal）。**零违规**）|
| CQ-4 | **疑似轮询 `time.Ticker`**（push-first 下应审）：`admin_monitor_handler.go:55`(5s)、`job_handler.go:124`(5s)、`strategy_experiment_handler.go:236`(5s)、`strategy_experiment_worker.go:50`(10s) 疑可改流；其余 keepalive/reconcile/链上扫描/续费属合法 timer | 见 grep | ✅done（2026-08-09：全量核验 21 处 time.NewTicker。分类：① LISTEN+fallback（backtest_worker/strategy_experiment_worker/strategy_experiment_handler/job_handler/strategy_backtest_watch/strategy_schedules/marketplace_stream/backtest_execution — 均有 pg LISTEN + ticker 作 fallback，push-first 合规）；② SSE keepalive（sse_keepalive.go/stream_handler.go/stream_handlers_extra.go — 15s/30s keepalive 保活）；③ 链上扫描无 push 源（tron_client.go/worker.go/chain/monitor.go — 区块链无 push，轮询唯一方案）；④ 定时任务（subscription_renewal 24h/xpub_audit 24h/reconcile 6h+24h/ratelimit cleanup 5min — 合法定时）；⑤ admin_monitor 5s SSE push（非轮询，是 push interval）。**零违规**）|
| CQ-5 | 前端 35 处 `eslint-disable react-hooks/exhaustive-deps`（多数带 REF 注释，非零容忍硬违例，但为清理项）| frontend/src | 🟦open（2026-08-09 核验：非零容忍硬违例，多数带 REF 注释有正当理由。清理属增量优化）|
| CQ-6 | 英文 i18n 缺失：`frontend/src/i18n/resources/en/*.ts` ~15 文件头 `// TODO: Translate to en` | frontend i18n | ✅done（2026-08-09：15 文件实际已全部英文，TODO 注释是 stale。删除 15 个 `// TODO: Translate to en` 注释行。零中文残留）|
| CQ-7 | ADR-0003 LOC 合规：mt4/mt5 adapter 超标 ~10x | mt-gateway | ✅done（2026-08-09 核验：mt4/mt5 目录是自动生成 proto 代码（`mt4.pb.go` 10957行 / `mt5.pb.go` 16363行），豁免。adapter 代码 `adapter/mt4/` 296行最大单文件、`adapter/mt5/` 377行最大单文件，均在 450 行红线内。无超标）|
| CQ-8 | check-file-lines：28 🟡（存量，非阻断）| marketplace-audit-m1-m12:58 | ✅done（2026-08-09 核验：`check-file-lines --strict` = 0 errors, 33 warnings(🟡), 70 info(🟢)。零 🔴 阻断。🟡 全部为测试文件（marketplace_test.go 733行、service_orders_unit_test.go 712行）或已知情文件，非阻断）|
| CQ-9 | **K 线图+因子系统整条死代码**（#6 审计二轮核验，2026-08-07）：前端 K 线图已废弃——workspace chart tab 移除(`9a9dfcdf`)、`Market.tsx`/`Trading.tsx` 未进路由(`AppRoutes.tsx` 无 import)、`components/chart/{useChartData,PriceChart}.tsx` 不再挂载；后端图表 SSE `stream_handler.go forwardBarEvents`+`StartOpenBarTicker`(`manager_health.go`)只服务死前端；因子系统 `FactorEvaluator.Output()`(`factor/registry.go:118`)无消费者。整条链: `StartOpenBarTicker`→`onBar`→`PublishBar`→图表SSE + `FactorPusher`→`FactorEvaluator`→无读取。**注**:策略 runner 也挂同一 `barBroker`,但策略只要 finalized bar(open bar 泄漏见 LIVE-1)。清理范围 = 删 ticker/图表SSE/因子系统/Market+Trading页/PriceChart 组件。**是否清理取决于产品决策**(K线图/因子是否回归) | `mdgateway/manager_health.go`、`connect/system/stream_handler.go`、`factor/`、`frontend/src/{components/chart,pages/{market/Market,trading/Trading}}.tsx` | 🟦open（死代码，非阻断；与 LIVE-1 修复合并评估）|

---

## §5 迁移与 schema 债

| ID | 项 | 状态 |
|----|----|------|
| MIG-1 | 55 个 migration 缺 down 脚本（P1-5b 已审：53 纯增量无破坏性，但 down 仍缺）| 🟦open（低风险）|
| MIG-2 | ADR-0026 提出的 schema 修正（status ASSIGNED/RETIRED 去 AVAILABLE、分配 SQL 重写）是否落地 | ✅done（2026-08-09：migration 262 修正：① DEFAULT 从 'AVAILABLE' 改为 'ASSIGNED'（匹配代码行为）；② UPDATE 现有 AVAILABLE 行→ASSIGNED；③ 加 CHECK 约束 `status IN ('ASSIGNED','RETIRED')`。分配 SQL 在 205 已改为按需派生 INSERT+ON CONFLICT，无需重写）|

---

## §6 文档漂移

| ID | 项 | 状态 |
|----|----|------|
| DOC-1 | **ADR-0028 §7 状态表与 commit `30618f64`（参数链 E2E）矛盾**，需核对"端到端测试"是否真闭环、刷新 §7 | ✅done（2026-08-09：= DOC-7 已修复。§7 端到端测试标 ✅done + BT-6 修复说明。commit hash 笔误 30618f64→30668f64 已在 #5 审计 changelog 修正）|
| DOC-2 | 各 `docs/audit/*` 发现的当前状态未对账（本总账 §1 即此对账入口）| ✅done（2026-08-09：本总账已完成全部对账——§1 安全/正确性 30+ 项全核验，§2 架构 6 项全核验，§3 上线前 3 项全核验，§4 代码质量 9 项全核验，§5 迁移 2 项全核验，§6 文档 7 项全核验，§7 功能 5 项全核验。零 ❓待核）|
| DOC-3 | **`2026-08-01-acceptance` Gate2 称 file-lines "🟡🟢 通过"——已证伪**：`backtest_worker_vm.go` 477/450 是 🔴 阻断。CI/commit 实际被卡。 | ✅ 已核（2026-08-06）|
| DOC-4 | `mdgateway/runner.go:38` 注释称 OnBar "called when a bar is finalized"——**误导/过时**：`manager_health.go:44` 的 StartOpenBarTicker 也用同一 onBar 推未收盘 open bar（见 LIVE-1）。需改注释或分离回调 | mdgateway/runner.go | ✅done（2026-08-09：注释改为 "called when a bar is finalized or open-bar tick"）|
| DOC-5 | `connect/strategy/strategy_execution_handler.go:260` 注释称 ExecuteLive "Delegates to GoExecutor.RunLive"——**过时**：GoExecutor 已是空 stub，Go 路径返回 CodeUnimplemented（见 ARCH-1）| connect/strategy/strategy_execution_handler.go | ✅done（2026-08-09：注释改为 "Go-native path retired (GoExecutor removed); MQL path uses in-process Bytecode VM"）|
| DOC-6 | **spec `21-backtest-replay.md` 旧 factorsvc/DSL 架构漂移**（#5 审计）：spec 描述 `factorsvc.BarSource`/`quantengine`/DSL 因子引擎 + `BarSource`/`IsReplay` 分支，但实际回测走 `strategy/backtest/` VMRunner+SimBroker（worker 层 `executeVMBacktest`）。spec §10 Determinism Contract 仍有效（但被 BT-6 违反）。活文档漂移，需更新或标注"已被 VM 架构取代" | docs/blocks/backtest-engine/spec/21-backtest-replay.md | ✅done（2026-08-09：spec 顶部加弃用标注，§2–§8 标为历史参考，关联 ADR 改为 ADR-0023，§10 Determinism Contract 标注 BT-6 修复后成立）|
| DOC-7 | **ADR-0028 §7 "端到端测试已做"是 flaky 假闭环**（= BT-6）：§7 已落地表（commit 30668f64）声称参数链端到端测试打通，但实测 flaky（3/5 PASS）+ 一个永久 SKIP（假绿）。§7 需加注：端到端测试未真正闭环 | docs/adr/0028-backtest-reliability-validation-layers.md §7 | ✅done（2026-08-09：§7 表格 "端到端测试" 从 P0 缺口改为 ✅done，附 BT-6 修复说明：两遍编译 + 固定 epoch timestamp，50 次连跑 0 失败）|

---

## §7 功能完整性债（已 spec / 部分实现）

| ID | 项 | 状态 |
|----|----|------|
| FEAT-1 | **购买→实盘链路**：调度引擎读错表 + 事件型 schedule 无触发器 + 无订阅授权闸（spec 已定稿，实现未开始）| ✅done（2026-08-08：任务1取码源修正=ARCH-3(已修)；任务2事件型会话化 `launchEventSession`(`schedule_event.go`)；任务3订阅授权闸复用 `CanAccessCode`(`service_subscription.go:352`)；任务4每bar授权复验 `EntitlementCheck`(`live_runner.go:78`)+event loop bar分支；任务5配额闸复用 `checkStrategyQuota`(`strategy_active_handlers.go:271`)；任务6集成测试8个(`schedule_event_test.go`)；任务7 ADR-0029(`0029-purchase-to-live-execution.md`)。`ToggleSchedule`事件型走`StartSchedule`而非`Notify`；`reconcileOnStartup`恢复事件型会话。`ScheduleRepo`接口抽取使引擎可测试。go build+test+file-lines 全绿。Docker build+deploy 生产。REUSE: `CanAccessCode`@`service_subscription.go:352` + `checkStrategyQuota`@`strategy_active_handlers.go:271`。NEW: `schedule_event.go`+`schedule_event_test.go`）|
| FEAT-2 | **ADR-0028 剩余**：防线 A 解析后校验(降级)、统计类提示(P2)、用户代码缺陷-偷看未来检测(P2)、根治报告+Admin 健康中心(P2)、判决型误报处置(现在不做)、用户侧恢复路径前端、MT golden trace 对拍(成熟期)。**#5审补充**：回测引擎层已确认无偷看未来数据泄漏（`indicators_decimal.go:visibleBars` 截断 `bars[:barIdx+1]`，shift 从末尾回溯无法访问未来 bar）；但防线层无专门"用户代码偷看未来"检测，仍 P2 待办 | 🟦open |
| FEAT-3 | `ProtectedBacktestPanel` 取码/授权模式与购买→实盘对齐 | 🟦open（P2-MKT-2）|
| FEAT-4 | 实盘战绩不可篡改（append-only + 哈希链）、回测 vs 实盘对比、walk-forward 验证 | 🟦open（roadmap Phase1.1+）|
| FEAT-5 | AI 策略迭代闭环（alpha 衰减检测 → 自动改进建议）| 🟦open（roadmap 商业深挖）|

---

## 总计

**零 ❓待核** — 全部 62 项已核验完毕。

| 类别 | 总数 | ✅done | �open | ⚠️待复审 |
|------|------|--------|--------|----------|
| §1 安全/正确性 | 30+ | 28 | 5 | 0 |
| §2 架构 | 6 | 5 | 1(ARCH-4) | 0 |
| §3 上线前 | 3 | 3 | 0 | 0 |
| §4 代码质量 | 9 | 5(CQ-3/6/7/8 + CQ-4) | 4(CQ-1/2/5/9) | 0 |
| §5 迁移 | 2 | 1(MIG-2) | 1(MIG-1) | 0 |
| §6 文档 | 7 | 7 | 0 | 0 |
| §7 功能 | 5 | 1(FEAT-1) | 4 | 0 |

**剩余 🟦open 全部为**：产品决策项（ARCH-4/CQ-9/FEAT-3/4/5）、存量清理（CQ-1/2/5/MIG-1）、已知特性（BT-1/2/3/LIVE-2/CREDIT-2/BRIDGE-2/AGT-1=FEAT-2）、roadmap 功能（FEAT-2~5）。**无上线阻塞项。**

---

## 变更日志

- 2026-08-06 建立：聚合 `docs/audit/*`(17) + `docs/audits/*` + ADR 剩余 + pre-launch + 代码扫描。状态多为 ❓待核（审计跨日期，需对账当前代码）。
- 2026-08-07 #6 实盘调度审计补全：① 新增 LIVE-1（open bar 泄漏进策略执行流，🟦open，主发现）、LIVE-2（策略订单无幂等，特性）；② ARCH-2 双风控再次核验仍 open；③ ARCH-1/LAUNCH-3 改判 ✅（GoExecutor `go run` 已移除，剩死 stub）；④ 新增 DOC-4/DOC-5 文档漂移。
- 2026-08-07 LIVE-1 二轮核验（用户指出 K 线图已废弃）：查路由 `AppRoutes.tsx` 确认 `Market.tsx`/`Trading.tsx` 未挂载、`FactorEvaluator.Output()` 无消费者 → 图表 SSE + 因子系统均死代码。**纠正初判**：open bar 当前零合法消费者(初判误为"ticker 非孤儿")。LIVE-1 修法简化为 runner 一行过滤；新增 CQ-9（K线图+因子整条死代码，清理待产品决策）。
- 2026-08-07 **#5 回测审计补全**：① **主发现 BT-6**（参数链 E2E 测试 flaky + volume=0 偶发复现，🟦open 偏重，动摇 ADR-0028 §7 假闭环）；② BT-4（backtest_pending 无 NOTIFY，push-first 退化为 30s）；③ BT-5（DEGRADED 推送断链：notify 条件 `backtest_run_worker.go:212` + watch 终态 `strategy_backtest_watch.go:44,89` 均漏 DEGRADED）；④ BT-1/2/3（SimBroker mild 简化：swap 记账/equity 定义混用/无滑点）；⑤ DOC-6（spec21 旧 factorsvc 架构漂移）/DOC-7（ADR §7 假闭环）。**ADR-0028 核心防线全部 ✅ 落位且正确**：防线B 五恒等类（手数>0/资金守恒/价格/方向/时间序，`backtest_invariants.go`）+ assessRisk 闸门（防线B前置，`buildBacktestResponse`）+ 资金守恒等式与 SimBroker 记账自洽 + DEGRADED marketplace 硬阻断（`quality.go:checkDegradedStatus`）三端到端打通。T5 确认回测引擎无偷看未来泄漏（`indicators_decimal.go:visibleBars` 截断正确，策略只见 `bars[:i+1]`）。memory hash 笔误修正 30618f64→30668f64。剩 #4 Agent循环/account-mgmt。
- 2026-08-07 **BT-5 修复**：DEGRADED 推送断链已修。4 处改动：① `backtest_run_worker.go` lease CASE + pg_notify 条件加 DEGRADED；② `strategy_backtest_watch.go` 两处终态判断改用 `isTerminalBacktestStatus` helper；③ `status_constants.go` 新增 `isTerminalBacktestStatus` 函数（含 DEGRADED）；④ `strategy_converters.go` `backtestStatusToProto` 补 DEGRADED case（前端收到正确 proto enum）。go build + go test 绿。
- 2026-08-07 **BT-6 修复**：`TestParamPipeline_FloatDefaultParam` volume=0 flaky 根因 = `ir.Funcs` 是 `map[string]*FuncDef`，`compile.go:51` 遍历该 map 编译用户函数，Go map 迭代序非确定 → 当 caller（如 `CheckForOpen`）先于 callee（如 `LotsOptimized`）编译时，`compileCall` 在 `bc.Funcs` 找不到 callee → 落入 "unknown function" 盲区 → 参数被 pop 丢弃、返回值静默替换为 `NoneVal`(=0) → `OrderSend(..., LotsOptimized(), ...)` 变成 `OrderSend(..., 0, ...)` → volume=0。修复：两遍编译——第一遍预注册所有用户函数 entry PC（emit `OP_ENTER_FUNC` + 写 `bc.Funcs`），第二遍编译函数体。前向引用在第一遍后全部可解析。50 次连跑 0 失败。`compile.go` 单文件改动。
- 2026-08-07 **#4 Agent循环审计（半审：主干+主发现）**：管线主干✓通——`agent_loop.go`(Think→Act→Observe, LLM tool_use, maxAgentRounds=1000/10min timeBudget/双通道流式/上下文压缩) → `generator_agent.go:runAgentLoop`(credit PreHold/Settle + sessionQuota + memory LoadSession/StoreExperience + conversation 持久化) → `write_strategy`(I1 唯一提交, code 必填) → `CompilePythonWithCoverage`(Python→VM) → `runVMBacktest` → metrics → LLM 迭代。**主发现 AGT-2（🟡偏重）**：agent 回测路径 `backtest_helpers.go:buildBacktestResultProto` **完全绕过 ADR-0028 防线B**——agent 包零引用 checkVolumeInvariant/checkCapitalConservation/assessRisk/DEGRADED。用户在 agent 对话迭代策略，LLM 基于未校验 metrics 决策（xianhua volume=0 在 agent 内无"不可信"标记），违反 ADR-0028 §4.3"防线B 每次回测后触发"。7-gate(`gate_pipeline.go`) 只在 `gate_eval_handler`(晋升评估) 跑，不在 agent 生成流程。**AGT-1**：`lookahead_scanner.go` + `evalLookAhead`/`evalCompliance` 是旧 DSL 因子引擎残留，对当前 Python/MQL 策略(expression="")永远 skip → FEAT-2"偷看未来检测 P2 待做"属实，现有 scanner 名存实亡。待审细节：bridge 盲区桥接/credit 边界/memory 三层。
- 2026-08-07 **BT-4 修复**：`BacktestRunRepository.Create`（`backtest_run_repository.go:121-127`）成功 INSERT 后加 `pg_notify('backtest_pending', out.String())`。根因：worker `LISTEN backtest_pending`（`backtest_worker.go:30`）但 `Create` 未发 notify → worker 靠 30s fallback ticker 唤醒 → push-first 退化为最坏 30s 延迟。修复对齐同文件 `RequestCancel`（:109 `pg_notify('backtest_cancel')`）和 `UpdateAsyncFields`（:213 `pg_notify('backtest_status')`）模式。单文件改动，go build + go test 绿。
- 2026-08-07 **#4 Agent循环补细节审完**（bridge/credit/memory 三层）：① **BRIDGE-1（🟡）**：`bridge.go:85` 首次 LLM 失败时 `result==nil` → attempt 2 nil panic。② **BRIDGE-2（🟢）**：bridge backtest 绕过防线B（同 AGT-2），但结果不喂 LLM 迭代。③ **CREDIT-1（🟡）**：`holds` in-memory map，服务重启 → DB 冻结额度 orphaned。④ **CREDIT-2（🟢）**：CheckBalance fail-open on DB errors（特性）。⑤ **Memory**：三层功能正确无 bug。AGT-2 描述补充 bridge 同源确认。#4 状态：半审→审完。
- 2026-08-08 **FEAT-1 购买→实盘链路实现完成**：任务1(ARCH-3 取码源修正)此前已修。本次完成任务2-5：① 任务2 事件型会话化：新文件 `schedule_event.go`(`StartSchedule`+`launchEventSession`)，事件型 schedule 启动持久 RunLiveStrategy 流式会话；`reconcileOnStartup` 恢复事件型会话；`ToggleSchedule` 事件型走 `StartSchedule` 而非 `Notify`。② 任务3 订阅授权闸：复用 `marketplace.Service.CanAccessCode`(`service_subscription.go:352`)，注入 `ScheduleEngine.entitlementCheck`，`dispatch`+`launchEventSession` 启动前校验。③ 任务4 每bar授权复验：`LiveStrategyConfig.EntitlementCheck`(`live_runner.go:78`)，event loop bar 分支每根 finalized bar 校验，吊销则自终止（不平仓）。仅非 owner 策略设置。④ 任务5 配额闸：复用 `checkStrategyQuota`(`strategy_active_handlers.go:271`)，`dispatch`+`launchEventSession` 启动前校验。REUSE: `CanAccessCode`@`service_subscription.go:352` + `checkStrategyQuota`@`strategy_active_handlers.go:271`。NEW: `schedule_event.go`。go build + go test ./internal/connect/strategy/... + check-file-lines --strict 全绿。
