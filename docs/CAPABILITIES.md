# 能力清单（Capability Map）

> **唯一目的**：动工前查"什么已实现"，**避免重复造轮子**，并区分"真未实现"与"已实现但没接线（shelf-ware）"。
> **维护方式**：本文件分两部分——
> 1. **人工策展区（本段到哨兵之前）**：高风险域的分层状态表 + 别名索引，人工维护。
> 2. **自动生成区（哨兵以下）**：由 `bash scripts/gen_capability_map.sh` 从代码 grep 重生成，**不会过时、不会说谎**。

## 执行者必读：复用核对（Reuse Preflight）

写任何**新 file/function** 前，必须：

1. 先跑 `bash scripts/gen_capability_map.sh` 刷新本文件。
2. 用你要做的**动词 + 别名**（见下方索引）在本文件搜索。
3. 在 PR 描述里逐条给结论，二选一：
   - `REUSE: <symbol> @ <file:line>` —— 复用现成，不另写。
   - `NEW: 无现成能力（已搜：<关键词列表>）` —— 确认是真空白。
4. 若发现某能力的**状态标记过时**（例如代码注释说"不存在"但其实已实现），**必须同步修正注释与本表**。

**缺少这段 `REUSE:`/`NEW:` 引用 = 该 Task 直接判失败。**

## 分层状态图例

一个能力可能在不同层"存在"，**不可笼统说"已实现"**：

- `gateway-rpc` — mtapi 网关原始 RPC（`reference/grpc/mt4.proto` / `mt5.proto`）
- `executor` — `OrderExecutor` / `BrokerExecutor` 接口方法
- `mthub-method` — `MtHubService` 的 Go 方法（业务层，带 killSwitch/风控/幂等门）
- `connect-rpc` — `proto/ant/v1` 公开 ConnectRPC（前端/跨服务可直接调用）
- `wired-live` — 是否已接进实盘/调度执行路径

## 高风险域：交易操作（重复造轮子重灾区）

| 能力 | 动词/别名 | gateway-rpc | executor | mthub-method | connect-rpc | wired-live | 权威位置 |
|---|---|---|---|---|---|---|---|
| 下单 | OrderSend / PlaceOrder / open / submit | ✅ mt4/mt5 | ✅ `PlaceOrder`/`SubmitOrder` | ✅ `PlaceOrder` | ✅ | ✅ | `backend/internal/mthub/service_orders.go:18` |
| 平仓 | OrderClose / CloseOrder / close / close_position | ✅ mt4/mt5 | ✅ `CloseOrder` | ✅ `CloseOrder` | ✅ | ✅ | `backend/internal/mthub/service_orders_close.go:15` |
| 改 SL/TP | **OrderModify / ModifyOrder** / amend / update_stops / modify | ✅ mt4/mt5 | ✅ `ModifyOrder` | ✅ `ModifyOrder` | ❌ **未暴露 ConnectRPC** | ✅ **已接进 live 路径**（`dispatchModifyOrder` 调用 `MtHubService.ModifyOrder`） | `backend/internal/mthub/service_orders_modify.go:18` |
| 查持仓 | OpenedOrders / positions / FetchOpenedOrders | ✅ mt4/mt5 | ✅ `FetchOpenedOrders` | ✅ `OpenedOrders` | ✅ | ✅ | `backend/internal/mthub/service.go:207` |
| 历史订单 | OrderHistory / FetchOrderHistory / history | ✅ | ✅ | ✅ `OrderHistory` | ✅ | ✅ | `backend/internal/mthub/service.go:218` |
| 品种参数 | SymbolParams / FetchSymbolParams | ✅ | ✅ | ✅ `SymbolParams` | ✅ | ✅ | `backend/internal/mthub/service.go:227` |
| 账户状态推送 | AccountStatus / SubscribeAccountStatus | — | — | ✅ `SubscribeAccountStatus` | ✅ (Stream) | ✅ 已接 AccountStateProvider（balance 默认 10000，ProfitUpdate 事件到达后覆盖） | `backend/internal/mthub/service.go:153` |

> **教训实例**：`live_runner.go` 曾写"MtHubService doesn't expose a direct ModifyOrder, use close+place workaround"——**该注释已过时**：`MtHubService.ModifyOrder` 早已实现（`service_orders_modify.go:18`），live 路径也已接入。唯一剩余缺口：**ConnectRPC 未暴露 ModifyOrder**（前端/跨服务无法直接调用）。

## 高风险域：策略 SDK / Broker（EA 替代）

| 能力 | 别名 | 状态 | 权威位置 |
|---|---|---|---|
| Broker 抽象接口 | order_send/position_modify/position_close/order_delete/positions/orders/account/symbol_info/server_time | ✅ 已冻结 | `backend/strategy/sdk/broker.go` |
| 回测 Broker | SimBroker | ✅ 实现，⚠️ 未集成 gate（见 D6-A） | `backend/strategy/backtest/sim_broker.go` |
| 实盘 Broker | LiveBroker / OrderIntent / to_signal_dict | ✅ 实现，⚠️ 未端到端验证 | `backend/strategy/sdk/live_broker.go` |
| 策略生命周期 | StrategyRuntime / OnInit/OnBar/OnDeinit | ✅ | `backend/strategy/sdk/runtime.go` |
| 风控门 | Gate / Evaluate / risk rules / kill-switch | ✅ 实现，✅ **已接进 live_runner**（dispatchCloseOrder + submitOrder 均经 Gate.Evaluate 过滤，D6-A） | `backend/internal/risk/gate.go:109` |
| 旧风控管线 | risksvc / SignalPipeline | ✅ 已接线（勿与新 EA gate 混用） | `backend/internal/risksvc/pipeline.go` |

## 动词/别名索引（搜不到先查这里换个词）

- **改单/止损止盈**：modify, amend, update, OrderModify, ModifyOrder, set_sl, set_tp, update_stops → 用 `MtHubService.ModifyOrder`
- **平仓**：close, exit, OrderClose, CloseOrder, close_position, close_all → 用 `MtHubService.CloseOrder`
- **下单**：send, place, open, submit, OrderSend, PlaceOrder, SubmitOrder → 用 `MtHubService.PlaceOrder`
- **持仓/订单查询**：positions, opened, orders, fetch, OpenedOrders → 用 `MtHubService.OpenedOrders`
- **账户状态/权益**：equity, balance, margin, account state, AccountStatus → 用 `SubscribeAccountStatus` / `AccountStateProvider`
- **风控/拦单**：risk, gate, block, kill-switch, drawdown, max lot → 用 `risk.Gate`（新 EA）/ `risksvc`（旧管线）

---
<!-- AUTOGEN-BELOW: 由 scripts/gen_capability_map.sh 重生成，勿手工编辑以下内容 -->

_最后生成：2026-08-19 23:05 UTC。运行 `bash scripts/gen_capability_map.sh` 刷新。_

## 符号索引（扁平 symbol → file:line，grep 友好）

> 查询方式：`bash scripts/cap.sh <动词/别名/符号>`（只返回命中行，token 有上界）。**禁止整篇 Read 本文件。**

```
AcceptDisclaimer	backend/internal/risksvc/jurisdiction_store.go:88
AcceptDisclaimerAt	backend/internal/risksvc/jurisdiction_store.go:137
Account	reference/grpc/mt5.proto:58
AccountSummary	reference/grpc/mt4.proto:60
AccountSummary	reference/grpc/mt5.proto:64
ActivateCanary	backend/internal/risk/canary.go:172
ActiveAccountIDs	backend/internal/mthub/service.go:318
ActiveAccountIDs	backend/internal/mthub/types.go:116
AddAccount	backend/internal/risk/canary.go:127
AddCredits	proto/ant/v1/credit.proto:16
AddGateRule	backend/internal/connect/strategy/strategy_execution_handler.go:208
AddRule	backend/internal/risk/gate.go:184
AddSanctionedCountry	proto/ant/v1/admin_jurisdiction.proto:13
AddTokens	backend/internal/service/systemai/session_quota.go:24
AddWhitelistAddress	proto/ant/v1/webauthn.proto:31
AdjustBalance	backend/internal/service/wallet_service.go:51
AdjustBalance	proto/ant/v1/wallet.proto:11
AdminFeatureStrategy	proto/ant/v1/marketplace_service.proto:62
AdminListRefundRequests	proto/ant/v1/marketplace_service.proto:65
AdminListStrategies	proto/ant/v1/marketplace_service.proto:61
AdminProcessRefund	proto/ant/v1/marketplace_service.proto:66
AdminProcessVerification	proto/ant/v1/marketplace_service.proto:59
Allocate	backend/internal/risksvc/block_allocator.go:105
Allocate	backend/internal/risksvc/block_allocator.go:29
Allocate	backend/internal/risksvc/block_allocator.go:69
AllowedLotSize	backend/internal/risk/canary.go:153
AnalyzeAsset	proto/ant/v1/asset_analysis.proto:14
AnalyzeImportCode	backend/internal/connect/strategy/strategy_import_handler.go:40
AnalyzeImportCode	proto/ant/v1/strategy_runtime.proto:33
AnalyzePlan	proto/ant/v1/strategy_execution.proto:13
ApplyEvent	backend/internal/mthub/state_cache.go:117
ApproveAutoGenTask	proto/ant/v1/marketplace_service.proto:44
ArchiveStrategy	proto/ant/v1/admin_strategy.proto:29
ArchiveTemplate	backend/internal/service/template_svc_admin.go:330
AssignAccountNumber	backend/internal/service/user/account_number.go:221
BackfillPlaintextCredentials	backend/internal/service/account_sync.go:41
Backtest	backend/internal/connect/strategy/strategy_execution_handler.go:292
Backtest	proto/ant/v1/strategy_runtime.proto:19
BatchSetAgents	proto/ant/v1/ai.proto:21
BatchSetAgents	proto/ant/v1/ai_agent.proto:9
BeginRegistration	proto/ant/v1/webauthn.proto:15
BeginTx	backend/internal/service/account_crud.go:46
BeginWithdrawal	proto/ant/v1/webauthn.proto:23
BrokerRegistry	backend/internal/mthub/service_setters.go:32
BuildPendingWithdrawals	backend/internal/service/withdrawal_builder.go:69
BuildUndelegateOnlyBundle	proto/ant/v1/deposit.proto:33
CachedSymbolParam	backend/internal/mthub/service_symbol_params.go:17
CalculatePositionSize	proto/ant/v1/auto_trading.proto:21
CanAccept	backend/internal/mthub/reconcile_gate.go:38
CancelAlgo	proto/ant/v1/execution_algo.proto:20
CancelBacktestRun	backend/internal/connect/strategy/strategy_backtest_crud.go:224
CancelBacktestRun	proto/ant/v1/strategy_runtime.proto:24
CancelJob	proto/ant/v1/job.proto:12
CancelPaperOrder	backend/internal/paper/engine.go:221
CancelSignal	backend/internal/connect/strategy/strategy_signals.go:85
CancelSignal	backend/internal/service/signal_svc.go:96
CancelSignal	proto/ant/v1/strategy.proto:36
CancelStrategyExperiment	backend/internal/connect/strategy/strategy_experiment_handler.go:184
CancelStrategyExperiment	proto/ant/v1/strategy_experiment.proto:14
CancelSubscription	backend/internal/service/subscription_service.go:217
CancelSubscription	proto/ant/v1/subscription.proto:17
CancelTemplateDraft	backend/internal/connect/strategy/strategy_handler.go:50
CancelTemplateDraft	proto/ant/v1/strategy.proto:21
CancelWithdrawal	backend/internal/service/wallet_service.go:101
CancelWithdrawal	proto/ant/v1/webauthn.proto:29
ChangePassword	reference/grpc/mt5.proto:231
ChangePlan	backend/internal/service/subscription_service.go:229
ChangePlan	proto/ant/v1/subscription.proto:19
Chat	proto/ant/v1/ai.proto:12
ChatCompletion	backend/internal/connect/strategy/ai_proposer_adapter.go:22
ChatCompletion	backend/internal/service/systemai/chat.go:183
ChatCompletionStream	backend/internal/service/systemai/chat_stream.go:17
ChatCompletionStreamWithTools	backend/internal/service/systemai/chat_stream.go:29
ChatCompletionWithUsage	backend/internal/service/systemai/chat.go:196
ChatStream	proto/ant/v1/ai.proto:14
Check	backend/internal/risk/guard.go:58
Check	backend/internal/risk/rules.go:132
Check	backend/internal/risk/rules.go:153
Check	backend/internal/risk/rules.go:175
Check	backend/internal/risk/rules.go:199
Check	backend/internal/risk/rules.go:223
Check	backend/internal/risk/rules.go:277
Check	backend/internal/risk/rules.go:320
Check	backend/internal/risk/rules.go:55
Check	backend/internal/risk/rules.go:75
Check	backend/internal/risk/rules.go:96
Check	backend/internal/risk/rules_risksvc.go:106
Check	backend/internal/risk/rules_risksvc.go:182
Check	backend/internal/risk/rules_risksvc.go:36
Check	backend/internal/risk/rules_risksvc.go:72
Check	backend/internal/risk/rule_user_config.go:35
Check	backend/internal/risksvc/hardlimit.go:116
Check	backend/internal/risksvc/hardlimit.go:139
Check	backend/internal/risksvc/hardlimit.go:59
Check	backend/internal/risksvc/hardlimit.go:77
Check	backend/internal/risksvc/jurisdiction.go:91
Check	backend/internal/risksvc/platform_limits.go:39
Check	backend/internal/risksvc/rules.go:112
Check	backend/internal/risksvc/rules.go:17
Check	backend/internal/risksvc/rules.go:35
Check	backend/internal/risksvc/rules.go:59
Check	backend/internal/risksvc/rules.go:80
Check	backend/internal/risksvc/rules.go:93
CheckAccountLimit	backend/internal/service/quota_checker.go:143
CheckAITokenQuota	backend/internal/service/quota_checker.go:94
CheckAndSet	backend/internal/mthub/idempotency.go:151
CheckAndSet	backend/internal/mthub/idempotency.go:53
CheckAssetUpdate	backend/internal/connect/strategy/strategy_asset_handler.go:158
CheckAssetUpdate	proto/ant/v1/strategy_asset.proto:15
CheckBacktestDailyLimit	backend/internal/service/quota_checker.go:116
CheckBalance	backend/internal/service/credit_service.go:158
CheckCode	backend/internal/connect/strategy/code_check_handler.go:16
CheckCode	proto/ant/v1/strategy_runtime.proto:65
CheckConnect	reference/grpc/mt4.proto:45
CheckConnect	reference/grpc/mt5.proto:43
CheckLiveStrategyLimit	backend/internal/service/quota_checker.go:125
CheckMarginCall	backend/internal/service/account_sync_service.go:115
CheckQuota	backend/internal/service/daily_quota.go:107
CheckRiskLimits	proto/ant/v1/auto_trading.proto:20
CheckStrategyLimit	backend/internal/service/quota_checker.go:107
CheckSymbolLimit	backend/internal/service/quota_checker.go:134
CleanupOldSnapshots	backend/internal/service/account_lifecycle.go:189
ClearAccount	backend/internal/risksvc/platform_aggregator.go:88
CloneStrategyAsset	backend/internal/connect/strategy/strategy_asset_handler.go:132
CloneStrategyAsset	proto/ant/v1/strategy_asset.proto:14
Close	backend/internal/connect/strategy/vm_live_session.go:133
ClosedOrders	reference/grpc/mt4.proto:151
CloseOrder	backend/internal/mthub/service_orders_close.go:27
CloseOrder	proto/ant/v1/mthub_service.proto:8
ClosePaperOrder	backend/internal/paper/engine.go:167
CloseSession	backend/internal/mthub/types.go:108
CommentOnStrategy	proto/ant/v1/marketplace_service.proto:19
CompareStrategies	proto/ant/v1/marketplace_service.proto:54
CompleteWithdrawal	backend/internal/service/wallet_service.go:96
Config	backend/internal/service/daily_quota.go:130
Confirm	backend/internal/mthub/idempotency.go:87
ConfirmDeposit	backend/internal/service/deposit_service.go:188
ConfirmSignal	backend/internal/connect/strategy/strategy_signals.go:72
ConfirmSignal	backend/internal/service/signal_svc.go:82
ConfirmSignal	proto/ant/v1/strategy.proto:35
Connect	reference/grpc/mt4.proto:17
Connect	reference/grpc/mt5.proto:15
ConnectAccount	proto/ant/v1/account.proto:19
ConnectEx	reference/grpc/mt4.proto:25
ConnectEx	reference/grpc/mt5.proto:22
ConnectProxy	reference/grpc/mt4.proto:39
ConnectProxy	reference/grpc/mt5.proto:37
Conversate	proto/ant/v1/strategy_execution.proto:16
Count	backend/internal/risksvc/capability.go:157
CountryCode	backend/internal/risksvc/jurisdiction.go:188
CreateAccount	backend/internal/service/account_crud.go:77
CreateAccount	proto/ant/v1/account.proto:16
CreateAccountTx	backend/internal/service/account_crud.go:52
CreateBundle	proto/ant/v1/marketplace_service.proto:89
CreateConversation	proto/ant/v1/ai.proto:17
CreateCoupon	proto/ant/v1/marketplace_service.proto:73
CreateFrozenBacktestDataset	proto/ant/v1/backtest_dataset.proto:12
CreatePaperAccount	proto/ant/v1/paper_trading.proto:10
CreateProvider	proto/ant/v1/ai_gateway.proto:20
CreateSchedule	backend/internal/connect/strategy/strategy_schedules.go:49
CreateSchedule	backend/internal/service/schedule_svc.go:78
CreateSchedule	proto/ant/v1/strategy.proto:26
CreateShareToken	proto/ant/v1/share.proto:8
CreateSystemStrategy	backend/internal/service/template_svc_admin.go:70
CreateSystemStrategy	proto/ant/v1/admin_strategy.proto:14
CreateTemplate	backend/internal/connect/strategy/strategy_template_handlers.go:126
CreateTemplate	backend/internal/service/template_svc.go:66
CreateTemplate	proto/ant/v1/strategy.proto:13
CreateTemplateDraft	backend/internal/connect/strategy/strategy_template_handlers.go:217
CreateTemplateDraft	proto/ant/v1/strategy.proto:18
CreateUser	proto/ant/v1/admin_user.proto:12
CreateWallet	backend/internal/service/wallet_service.go:44
CurrentStage	backend/internal/risk/canary.go:163
DB	backend/internal/service/strategy_svc.go:25
DeleteAccount	backend/internal/service/account_crud.go:114
DeleteAccount	proto/ant/v1/account.proto:18
DeleteAgentExperience	proto/ant/v1/agent_gateway.proto:42
DeleteBacktestDataset	proto/ant/v1/backtest_dataset.proto:13
DeleteBacktestRun	backend/internal/connect/strategy/strategy_backtest_crud.go:246
DeleteBacktestRun	proto/ant/v1/strategy_runtime.proto:25
DeleteBacktestRuns	backend/internal/connect/strategy/strategy_backtest_crud.go:265
DeleteBacktestRuns	proto/ant/v1/strategy_runtime.proto:26
DeleteBundle	proto/ant/v1/marketplace_service.proto:93
DeleteConversation	proto/ant/v1/ai.proto:18
DeleteHookConfig	proto/ant/v1/agent_hooks.proto:16
DeleteKey	backend/internal/mthub/idempotency.go:172
DeleteManagedSetting	proto/ant/v1/admin_settings.proto:17
DeleteModel	proto/ant/v1/ai_gateway.proto:30
DeleteOrder	backend/internal/mthub/service_orders_delete.go:15
DeleteProvider	proto/ant/v1/ai_gateway.proto:24
DeleteSchedule	backend/internal/connect/strategy/strategy_schedules.go:245
DeleteSchedule	backend/internal/service/schedule_svc.go:144
DeleteSchedule	proto/ant/v1/strategy.proto:28
DeleteShareToken	proto/ant/v1/share.proto:12
DeleteSystemStrategy	backend/internal/service/template_svc_admin.go:133
DeleteSystemStrategy	proto/ant/v1/admin_strategy.proto:16
DeleteTemplate	backend/internal/connect/strategy/strategy_template_handlers.go:202
DeleteTemplate	backend/internal/service/template_svc.go:105
DeleteTemplate	proto/ant/v1/strategy.proto:15
DeleteUser	proto/ant/v1/admin_user.proto:14
DeleteUsers	proto/ant/v1/admin_user.proto:15
DeleteUserSetting	proto/ant/v1/agent_gateway.proto:47
DeleteUserTemplate	proto/ant/v1/agent_gateway.proto:41
Deregister	backend/internal/connect/strategy/session_registry.go:179
DetectMarketRegime	proto/ant/v1/market_regime.proto:11
DetectStrategyDecay	proto/ant/v1/marketplace_service.proto:80
Diagnose	proto/ant/v1/strategy_execution.proto:14
DiffStrategyVersions	backend/internal/connect/strategy/strategy_version_handler.go:103
DiffStrategyVersions	proto/ant/v1/strategy_runtime.proto:60
DisableCoupon	proto/ant/v1/marketplace_service.proto:75
DisableStrategy	proto/ant/v1/admin_strategy.proto:27
DisableTemplate	backend/internal/service/template_svc_admin.go:300
DisableUser	proto/ant/v1/admin_user.proto:16
Disconnect	reference/grpc/mt4.proto:51
Disconnect	reference/grpc/mt5.proto:49
DisconnectAccount	proto/ant/v1/account.proto:20
DisconnectAccountByID	backend/internal/service/account_lifecycle.go:162
DiscoverGatewayModels	proto/ant/v1/ai_gateway.proto:32
DiscoverModels	backend/internal/service/systemai/service.go:272
DiscoverSystemAIModels	proto/ant/v1/system_ai.proto:15
DisengageKillSwitch	backend/internal/risk/canary.go:266
DroppedBars	backend/internal/mthub/broker_types.go:180
EnableStrategy	proto/ant/v1/admin_strategy.proto:28
EnableTemplate	backend/internal/service/template_svc_admin.go:318
EnableUser	proto/ant/v1/admin_user.proto:17
EngageKillSwitch	backend/internal/risk/canary.go:251
EnsureBoundAccount	backend/internal/service/bound_account_svc.go:35
EnsureFreeSubscription	backend/internal/service/subscription_service_proto.go:106
EnsureSeed	backend/internal/service/systemai/service.go:156
EnsureSession	backend/internal/mthub/types.go:83
EnterAll	backend/internal/mthub/reconcile_gate.go:59
EnterReconciling	backend/internal/mthub/reconcile_gate.go:24
Error	backend/internal/mthub/types.go:130
Error	backend/internal/risksvc/hardlimit.go:188
Error	backend/internal/service/registration_service.go:152
Error	backend/internal/service/systemai/chat_failover.go:82
Error	backend/internal/service/wallet_service.go:145
Error	backend/internal/service/wallet_service.go:153
Estimate	backend/internal/mthub/hub_estimator.go:46
Evaluate	backend/internal/risk/gate.go:119
Evaluate	backend/internal/risksvc/engine.go:27
Evaluate	backend/internal/risksvc/hardlimit.go:172
Events	reference/grpc/mt5.proto:512
Execute	backend/internal/connect/strategy/strategy_execution_handler.go:240
Execute	proto/ant/v1/strategy_runtime.proto:17
ExecuteBacktestDirect	backend/internal/connect/strategy/backtest_execution.go:29
ExecuteLive	backend/internal/connect/strategy/strategy_execution_handler.go:313
ExecuteLive	proto/ant/v1/strategy_runtime.proto:31
ExecutePlan	proto/ant/v1/strategy_execution.proto:15
ExecuteSignal	backend/internal/connect/strategy/strategy_signals.go:51
ExecuteSignal	backend/internal/service/signal_svc.go:67
ExecuteSignal	proto/ant/v1/strategy.proto:34
ExplainCode	proto/ant/v1/code_assist.proto:13
ExportBatchUnsignedSweepBundle	proto/ant/v1/deposit.proto:27
ExportCredentialList	proto/ant/v1/webauthn.proto:37
ExportUnsignedSweepBundle	proto/ant/v1/deposit.proto:25
ExportWhitelist	proto/ant/v1/webauthn.proto:39
Fetch	backend/internal/connect/strategy/data_source.go:57
FinishRegistration	proto/ant/v1/webauthn.proto:17
FinishWithdrawal	proto/ant/v1/webauthn.proto:25
FlagStrategy	proto/ant/v1/admin_strategy.proto:23
FlagTemplate	backend/internal/service/template_svc_admin.go:249
ForgotPassword	proto/ant/v1/auth.proto:19
FreezeAccount	proto/ant/v1/admin_account.proto:11
FreezeForWithdrawal	backend/internal/service/wallet_service.go:91
GenerateAccountNumber	backend/internal/service/user/account_number.go:43
GenerateAndPublish	proto/ant/v1/marketplace_service.proto:36
GenerateAndSend	backend/internal/service/email_verification.go:35
GenerateFromTemplate	proto/ant/v1/marketplace_service.proto:38
GenerateReport	proto/ant/v1/analytics.proto:15
GenerateStrategy	proto/ant/v1/agent_gateway.proto:30
Get	backend/internal/connect/strategy/session_registry.go:199
Get	backend/internal/mthub/derived_state.go:76
Get	backend/internal/mthub/types.go:77
Get	backend/internal/risksvc/capability.go:99
Get	backend/internal/service/analytics_cache.go:28
Get	backend/internal/service/systemai/service.go:193
GetAccount	backend/internal/mthub/derived_state.go:83
GetAccount	backend/internal/service/account_crud.go:32
GetAccount	backend/internal/service/platform_service.go:123
GetAccount	proto/ant/v1/account.proto:15
GetAccountAnalytics	proto/ant/v1/analytics.proto:8
GetAccountAuditLogs	proto/ant/v1/admin_account.proto:13
GetAccountBroker	backend/internal/service/platform_service.go:128
GetAccountCredentials	backend/internal/service/account_lifecycle.go:19
GetAccountLimit	backend/internal/service/bound_account_svc.go:90
GetAccountState	backend/internal/connect/strategy/account_provider.go:49
GetAccountStatus	proto/ant/v1/mthub_service.proto:14
GetActiveStrategy	backend/internal/connect/strategy/strategy_active_watch.go:59
GetActiveStrategy	proto/ant/v1/strategy_runtime.proto:43
GetAgentCapabilities	proto/ant/v1/agent_gateway.proto:18
GetAgentSettings	proto/ant/v1/agent_gateway.proto:45
GetAIPrimary	backend/internal/service/systemai/service.go:204
GetAIPrimary	proto/ant/v1/ai_primary.proto:8
GetAlgoStatus	proto/ant/v1/execution_algo.proto:17
GetAllLogs	backend/internal/service/log_service.go:58
GetAttribution	backend/internal/service/analytics_cache.go:52
GetAttributionAnalysis	proto/ant/v1/analytics.proto:13
GetAutoTradingStatus	proto/ant/v1/auto_trading.proto:22
GetBacktestRun	backend/internal/connect/strategy/strategy_backtest_crud.go:160
GetBacktestRun	proto/ant/v1/strategy_runtime.proto:21
GetBundle	proto/ant/v1/marketplace_service.proto:91
GetByScheduleID	backend/internal/connect/strategy/session_registry.go:220
GetCanary	proto/ant/v1/admin_sre.proto:10
GetCapabilities	proto/ant/v1/agent_gateway.proto:50
GetCapabilityTier	backend/internal/service/quota_checker.go:152
GetClients	reference/grpc/mt4.proto:217
GetClients	reference/grpc/mt5.proto:367
GetConnectionLogs	backend/internal/service/log_service.go:26
GetConnectionLogs	proto/ant/v1/log.proto:13
GetConversation	proto/ant/v1/ai.proto:16
GetCreditBalance	proto/ant/v1/credit.proto:9
GetDashboard	proto/ant/v1/admin_user.proto:10
GetDecryptedPassword	backend/internal/service/account_sync.go:17
GetDemo	reference/grpc/mt5.proto:374
GetDepositAddress	proto/ant/v1/deposit.proto:13
GetDivergenceReport	backend/internal/connect/strategy/divergence_handler.go:42
GetDivergenceReport	proto/ant/v1/live_backtest_divergence.proto:12
GetExperimentCandidate	backend/internal/connect/strategy/strategy_experiment_handler.go:214
GetExperimentCandidate	proto/ant/v1/strategy_experiment.proto:16
GetFreshSnapshot	backend/internal/connect/strategy/position_cache.go:104
GetGlobalSettings	proto/ant/v1/auto_trading.proto:15
GetImportedStrategy	backend/internal/connect/strategy/strategy_import_handler.go:113
GetImportedStrategy	proto/ant/v1/strategy_runtime.proto:35
GetIndicatorCatalog	proto/ant/v1/indicator_catalog.proto:10
GetJob	proto/ant/v1/job.proto:11
GetJurisdictionStatus	proto/ant/v1/admin_jurisdiction.proto:10
GetKillSwitch	proto/ant/v1/admin_sre.proto:6
GetKlines	proto/ant/v1/market_service.proto:8
GetLedgerSummary	proto/ant/v1/admin_billing.proto:17
GetLivePerformance	proto/ant/v1/marketplace_service.proto:32
GetLogs	reference/grpc/mt4.proto:200
GetLogsByUser	reference/grpc/mt4.proto:207
GetManagedSettings	proto/ant/v1/admin_settings.proto:13
GetMarketplaceAnalytics	proto/ant/v1/marketplace_service.proto:68
GetMarketRegime	proto/ant/v1/market_regime.proto:12
GetMe	proto/ant/v1/auth.proto:15
GetMetrics	proto/ant/v1/admin_system.proto:9
GetMonthlyAnalysis	proto/ant/v1/analytics.proto:11
GetMonthlyDetail	backend/internal/service/analytics_cache.go:100
GetMonthlyDetail	proto/ant/v1/analytics.proto:12
GetMonthlyPnL	proto/ant/v1/analytics.proto:10
GetMySubscription	backend/internal/service/subscription_service.go:369
GetMySubscription	proto/ant/v1/subscription.proto:13
GetMySubscriptionProto	backend/internal/service/subscription_service_proto.go:38
GetNotificationPrefs	proto/ant/v1/notification_service.proto:18
GetOperationLogs	backend/internal/service/log_service.go:50
GetOperationLogs	proto/ant/v1/log.proto:15
GetOptimizationTask	proto/ant/v1/marketplace_service.proto:82
GetOrCreateWallet	backend/internal/service/wallet_service.go:32
GetOrder	backend/internal/mthub/state_cache.go:76
GetOrderHistory	backend/internal/service/log_service.go:42
GetOrDeriveAddress	backend/internal/service/deposit_service.go:118
GetOrderLogHistory	proto/ant/v1/log.proto:14
GetOrdersByAccount	backend/internal/mthub/state_cache.go:83
GetPeakEquity	backend/internal/connect/strategy/account_provider.go:87
GetPlan	backend/internal/service/quota_checker.go:83
GetPosition	backend/internal/mthub/state_cache.go:96
GetPositionsByAccount	backend/internal/mthub/state_cache.go:103
GetProviderEarnings	proto/ant/v1/marketplace_service.proto:77
GetProviderFeeTier	proto/ant/v1/marketplace_service.proto:97
GetPublisherStats	proto/ant/v1/marketplace_service.proto:26
GetQuote	reference/grpc/mt5.proto:138
GetQuoteMany	reference/grpc/mt4.proto:80
GetQuoteMany	reference/grpc/mt5.proto:145
GetRecentTrades	proto/ant/v1/analytics.proto:9
GetRecentTradingLogs	proto/ant/v1/auto_trading.proto:24
GetRevenueSummary	proto/ant/v1/admin_billing.proto:13
GetRiskConfig	proto/ant/v1/auto_trading.proto:18
GetRolling	backend/internal/service/analytics_cache.go:76
GetRollingMetrics	proto/ant/v1/analytics.proto:14
GetRootCauseReport	proto/ant/v1/platform_health.proto:14
GetSchedule	backend/internal/connect/strategy/strategy_schedules.go:37
GetSchedule	backend/internal/service/schedule_svc.go:58
GetSchedule	proto/ant/v1/strategy.proto:24
GetScheduleHealth	proto/ant/v1/schedule_health.proto:10
GetSchedulePositions	backend/internal/connect/strategy/strategy_schedule_positions.go:22
GetSchedulePositions	proto/ant/v1/strategy.proto:25
GetScheduleRunLogs	backend/internal/service/log_service.go:54
GetScheduleRunLogs	proto/ant/v1/log.proto:16
GetSecret	backend/internal/service/systemai/service.go:251
GetSessionFeedback	proto/ant/v1/ai.proto:25
GetSharedPerformance	proto/ant/v1/share.proto:9
GetSignal	backend/internal/service/signal_svc.go:51
GetSnapshot	backend/internal/connect/strategy/position_cache.go:97
GetSnapshot	backend/internal/mthub/snapshot_persister.go:99
GetSnapshot	backend/internal/risksvc/platform_aggregator.go:140
GetStatus	backend/internal/risksvc/jurisdiction_store.go:22
GetStrategyAsset	backend/internal/connect/strategy/strategy_asset_handler.go:83
GetStrategyAsset	proto/ant/v1/strategy_asset.proto:11
GetStrategyDetail	proto/ant/v1/admin_strategy.proto:20
GetStrategyExperiment	backend/internal/connect/strategy/strategy_experiment_handler.go:157
GetStrategyExperiment	proto/ant/v1/strategy_experiment.proto:12
GetStrategyPublicInfo	proto/ant/v1/marketplace_service.proto:56
GetStrategyRun	backend/internal/connect/strategy/strategy_execution_runs.go:46
GetStrategyRun	proto/ant/v1/strategy_runtime.proto:39
GetStrategyVersion	backend/internal/connect/strategy/strategy_version_handler.go:47
GetStrategyVersion	proto/ant/v1/strategy_runtime.proto:56
GetSweepDashboard	proto/ant/v1/deposit.proto:31
GetSymbolStats	proto/ant/v1/market_service.proto:9
GetSystemAIConfig	proto/ant/v1/system_ai.proto:12
GetTemplate	backend/internal/connect/strategy/strategy_template_handlers.go:65
GetTemplate	backend/internal/service/template_svc.go:51
GetTemplate	proto/ant/v1/strategy.proto:12
GetTemplateDetail	backend/internal/service/template_svc_admin.go:14
GetTemplates	backend/internal/connect/strategy/strategy_execution_handler.go:307
GetTemplates	proto/ant/v1/strategy_runtime.proto:28
GetTickValueMany	reference/grpc/mt5.proto:216
GetTier	backend/internal/risk/rules_risksvc.go:167
GetTokenUsage	proto/ant/v1/ai_gateway.proto:15
GetTopProviders	proto/ant/v1/marketplace_service.proto:70
GetTopStrategies	proto/ant/v1/marketplace_service.proto:69
GetTradingLogs	proto/ant/v1/auto_trading.proto:23
GetTradingSummary	proto/ant/v1/admin_trading.proto:8
GetUsageSummary	proto/ant/v1/subscription.proto:21
GetUsageSummaryProto	backend/internal/service/subscription_service_proto.go:119
GetUserAccountIDs	backend/internal/service/account_sync.go:102
GetUserAccountIDs	backend/internal/service/platform_service.go:164
GetUserAccountSnapshots	backend/internal/service/account_sync.go:209
GetUserAccountSnapshots	backend/internal/service/platform_service.go:169
GetUserAccountsSummary	backend/internal/service/account_sync.go:160
GetUserAccountsSummary	backend/internal/service/platform_service.go:174
GetUserEmail	backend/internal/service/platform_service.go:179
GetWalkForwardReport	backend/internal/connect/strategy/walk_forward_handler.go:39
GetWalkForwardReport	proto/ant/v1/walk_forward.proto:11
GetWallet	proto/ant/v1/wallet.proto:9
Groups	reference/grpc/mt4.proto:66
HasBacktestData	backend/internal/connect/strategy/strategy_backtest_validate.go:123
HasOrderType	backend/internal/risksvc/capability.go:63
Health	reference/grpc/mt5.proto:343
HealthCheck	proto/ant/v1/admin_system.proto:8
History	backend/internal/risk/canary.go:308
ImportDepositAddresses	proto/ant/v1/deposit.proto:21
ImportSignedSweepBundle	proto/ant/v1/deposit.proto:29
ImportXpub	proto/ant/v1/deposit.proto:35
InitiateStrategyIteration	proto/ant/v1/marketplace_service.proto:87
InsertOrder	backend/internal/mthub/oms_writer.go:119
InsertScheduleRunLog	backend/internal/connect/strategy/session_registry.go:354
Invalidate	backend/internal/service/analytics_cache.go:127
InvalidateSummaryCache	backend/internal/service/account_sync.go:133
IsAccountNumberAvailable	backend/internal/service/user/account_number.go:85
IsAccountNumberAvailableExcluding	backend/internal/service/user/account_number.go:91
IsAdmin	backend/internal/service/platform_service.go:109
IsCanaryAccount	backend/internal/risk/canary.go:141
IsCircuitOpen	backend/internal/connect/strategy/session_registry.go:394
IsDisclaimerAccepted	backend/internal/risksvc/jurisdiction_store.go:76
IsExpired	backend/internal/mthub/types.go:17
IsInvestor	reference/grpc/mt4.proto:174
IsKillSwitchActive	backend/internal/risk/canary.go:281
IsQuestionnaireCompleted	backend/internal/risksvc/jurisdiction_store.go:100
IsQuoteSession	reference/grpc/mt5.proto:204
IsQuoteSessionMany	reference/grpc/mt5.proto:210
IsReconciling	backend/internal/mthub/reconcile_gate.go:45
IsSanctioned	backend/internal/risksvc/jurisdiction_store.go:125
IssueAgentToken	proto/ant/v1/agent_gateway.proto:14
IsTradeSession	reference/grpc/mt5.proto:191
IsTradeSessionMany	reference/grpc/mt5.proto:197
IsTripped	backend/internal/service/daily_quota.go:173
LatestTick	backend/internal/mthub/service.go:152
LatestTick	backend/internal/mthub/tick_broker.go:49
LinkLiveAccount	proto/ant/v1/marketplace_service.proto:33
List	backend/internal/service/systemai/service.go:186
ListAccounts	backend/internal/service/account_crud.go:19
ListAccounts	proto/ant/v1/account.proto:14
ListAccountsAdmin	proto/ant/v1/admin_account.proto:10
ListActiveStrategies	backend/internal/connect/strategy/strategy_active_watch.go:20
ListActiveStrategies	proto/ant/v1/strategy_runtime.proto:41
ListAdminWalletTransactions	proto/ant/v1/admin_billing.proto:15
ListAgentAudit	proto/ant/v1/agent_gateway.proto:17
ListAgentDefs	proto/ant/v1/ai_agent.proto:10
ListAgents	proto/ant/v1/ai.proto:20
ListAgentTokens	proto/ant/v1/agent_gateway.proto:15
ListAlgos	proto/ant/v1/execution_algo.proto:23
ListAll	backend/internal/connect/strategy/session_registry.go:282
ListAllCreditTransactions	proto/ant/v1/credit.proto:20
ListAllShareTokens	proto/ant/v1/share.proto:13
ListAllStrategies	backend/internal/service/template_svc_admin.go:175
ListAllStrategies	proto/ant/v1/admin_strategy.proto:19
ListAssetClones	backend/internal/connect/strategy/strategy_asset_handler.go:189
ListAssetClones	proto/ant/v1/strategy_asset.proto:17
ListAutoGenTasks	proto/ant/v1/marketplace_service.proto:42
ListBacktestDatasets	proto/ant/v1/backtest_dataset.proto:11
ListBacktestRuns	backend/internal/connect/strategy/strategy_backtest_crud.go:188
ListBacktestRuns	proto/ant/v1/strategy_runtime.proto:22
ListBacktestRunTrades	backend/internal/connect/strategy/backtest_trades_handler.go:53
ListBacktestRunTrades	proto/ant/v1/backtest_trades.proto:8
ListBoundAccounts	backend/internal/service/bound_account_svc.go:80
ListBoundAccounts	proto/ant/v1/subscription.proto:23
ListBreakers	proto/ant/v1/admin_sre.proto:8
ListBundles	proto/ant/v1/marketplace_service.proto:90
ListByAccount	backend/internal/connect/strategy/session_registry.go:269
ListByUser	backend/internal/connect/strategy/session_registry.go:207
ListComments	proto/ant/v1/marketplace_service.proto:20
ListConfigs	proto/ant/v1/admin_config.proto:10
ListConversations	proto/ant/v1/ai.proto:15
ListCoupons	proto/ant/v1/marketplace_service.proto:74
ListCredentials	proto/ant/v1/webauthn.proto:19
ListCreditTransactions	proto/ant/v1/credit.proto:10
ListDepositAddresses	backend/internal/service/deposit_service.go:175
ListDepositAddresses	proto/ant/v1/deposit.proto:19
ListEconomicCalendarEvents	proto/ant/v1/economic_data.proto:10
ListEconomicIndicators	proto/ant/v1/economic_data.proto:11
ListExperimentCandidates	backend/internal/connect/strategy/strategy_experiment_handler.go:197
ListExperimentCandidates	proto/ant/v1/strategy_experiment.proto:15
ListFeeTiers	proto/ant/v1/marketplace_service.proto:95
ListHookConfigs	proto/ant/v1/agent_hooks.proto:12
ListLeaderboard	proto/ant/v1/marketplace_service.proto:50
ListLogs	proto/ant/v1/admin_log.proto:10
ListManualReviewDeposits	backend/internal/service/deposit_service.go:170
ListManualReviewDeposits	proto/ant/v1/deposit.proto:17
ListMemory	proto/ant/v1/agent_gateway.proto:39
ListModels	proto/ant/v1/ai_gateway.proto:26
ListMyDeposits	backend/internal/service/deposit_service.go:165
ListMyDeposits	proto/ant/v1/deposit.proto:15
ListNotifications	proto/ant/v1/notification_service.proto:12
ListOptimizationTasks	proto/ant/v1/marketplace_service.proto:81
ListPaperAccounts	proto/ant/v1/paper_trading.proto:11
ListPendingSignBundles	proto/ant/v1/deposit.proto:23
ListPlans	proto/ant/v1/subscription.proto:11
ListPlansProto	backend/internal/service/subscription_service_proto.go:25
ListProviders	proto/ant/v1/ai_gateway.proto:18
ListProviderTransactions	proto/ant/v1/marketplace_service.proto:78
ListPublished	proto/ant/v1/marketplace_service.proto:14
ListRatings	proto/ant/v1/marketplace_service.proto:18
ListSanctionedCountries	proto/ant/v1/admin_jurisdiction.proto:12
ListSchedules	backend/internal/connect/strategy/strategy_schedules.go:25
ListSchedules	backend/internal/service/schedule_svc.go:45
ListSchedules	proto/ant/v1/strategy.proto:23
ListShareTokens	proto/ant/v1/share.proto:11
ListSignals	backend/internal/connect/strategy/strategy_signals.go:34
ListSignals	backend/internal/service/signal_svc.go:31
ListSignals	proto/ant/v1/strategy.proto:33
ListStrategies	backend/internal/service/platform_service.go:42
ListStrategyAssets	backend/internal/connect/strategy/strategy_asset_handler.go:69
ListStrategyAssets	proto/ant/v1/strategy_asset.proto:10
ListStrategyCards	backend/internal/connect/strategy/strategy_template_handlers.go:87
ListStrategyCards	backend/internal/service/template_svc.go:172
ListStrategyCards	proto/ant/v1/strategy.proto:16
ListStrategyExperiments	backend/internal/connect/strategy/strategy_experiment_handler.go:170
ListStrategyExperiments	proto/ant/v1/strategy_experiment.proto:13
ListStrategyRuns	backend/internal/connect/strategy/strategy_execution_runs.go:16
ListStrategyRuns	proto/ant/v1/strategy_runtime.proto:37
ListStrategyTemplates	proto/ant/v1/marketplace_service.proto:40
ListStrategyVersions	backend/internal/connect/strategy/strategy_version_handler.go:17
ListStrategyVersions	proto/ant/v1/strategy_runtime.proto:54
ListSubscriptions	backend/internal/service/platform_service.go:79
ListSubscriptions	proto/ant/v1/admin_billing.proto:11
ListSubscriptions	proto/ant/v1/marketplace_service.proto:15
ListSystemAIConfigs	proto/ant/v1/system_ai.proto:11
ListSystemModels	proto/ant/v1/ai_gateway.proto:13
ListSystemStrategies	backend/internal/service/template_svc_admin.go:51
ListSystemStrategies	proto/ant/v1/admin_strategy.proto:13
ListTemplates	backend/internal/connect/strategy/strategy_template_handlers.go:52
ListTemplates	backend/internal/service/template_svc.go:40
ListTemplates	proto/ant/v1/strategy.proto:11
ListTransactions	backend/internal/service/wallet_service.go:85
ListTransactions	proto/ant/v1/wallet.proto:10
ListUsers	proto/ant/v1/admin_user.proto:11
ListUsersByKYCStatus	proto/ant/v1/admin_jurisdiction.proto:15
ListWhitelistAddresses	proto/ant/v1/webauthn.proto:33
ListWithdrawals	proto/ant/v1/webauthn.proto:27
LoadAll	backend/internal/service/quota_checker.go:40
LoadFromPG	backend/internal/risksvc/capability.go:117
LoadFromRedis	backend/internal/mthub/state_cache.go:197
LogAudit	backend/internal/service/account_lifecycle.go:139
LogConnection	backend/internal/service/log_service.go:22
Login	proto/ant/v1/auth.proto:11
LogOperation	backend/internal/service/log_service.go:46
LogOrder	backend/internal/service/log_service.go:30
Logout	proto/ant/v1/auth.proto:12
Mails	reference/grpc/mt5.proto:237
MarkAddressReceived	backend/internal/service/deposit_service.go:237
MarkAllRead	proto/ant/v1/notification_service.proto:14
MarketWatchMany	reference/grpc/mt5.proto:151
MarkRead	proto/ant/v1/notification_service.proto:13
MarkReconciled	backend/internal/mthub/reconcile_gate.go:31
MemorySnapshot	reference/grpc/mt4.proto:209
MemorySnapshot	reference/grpc/mt5.proto:359
MemoryUsage	reference/grpc/mt4.proto:219
ModifyOrder	backend/internal/mthub/service_orders_modify.go:18
ModifyPaperOrder	backend/internal/paper/engine.go:195
Name	backend/internal/connect/strategy/data_source.go:55
Name	backend/internal/risk/rules.go:130
Name	backend/internal/risk/rules.go:151
Name	backend/internal/risk/rules.go:173
Name	backend/internal/risk/rules.go:197
Name	backend/internal/risk/rules.go:221
Name	backend/internal/risk/rules.go:275
Name	backend/internal/risk/rules.go:318
Name	backend/internal/risk/rules.go:53
Name	backend/internal/risk/rules.go:73
Name	backend/internal/risk/rules.go:94
Name	backend/internal/risk/rules_risksvc.go:104
Name	backend/internal/risk/rules_risksvc.go:180
Name	backend/internal/risk/rules_risksvc.go:34
Name	backend/internal/risk/rules_risksvc.go:70
Name	backend/internal/risk/rule_user_config.go:33
Name	backend/internal/risksvc/block_allocator.go:103
Name	backend/internal/risksvc/block_allocator.go:27
Name	backend/internal/risksvc/block_allocator.go:67
Name	backend/internal/risksvc/hardlimit.go:114
Name	backend/internal/risksvc/hardlimit.go:137
Name	backend/internal/risksvc/hardlimit.go:57
Name	backend/internal/risksvc/hardlimit.go:75
Name	backend/internal/risksvc/kelly_sizer.go:44
Name	backend/internal/risksvc/rules.go:111
Name	backend/internal/risksvc/rules.go:16
Name	backend/internal/risksvc/rules.go:34
Name	backend/internal/risksvc/rules.go:57
Name	backend/internal/risksvc/rules.go:79
Name	backend/internal/risksvc/rules.go:92
Name	backend/internal/risksvc/vol_target_sizer.go:45
NetExposureForSymbol	backend/internal/risksvc/platform_aggregator.go:145
Notify	backend/internal/connect/strategy/schedule_engine.go:139
OnBar	backend/internal/connect/strategy/strategy_templates.go:142
OnBar	backend/internal/connect/strategy/strategy_templates.go:236
OnBar	backend/internal/connect/strategy/strategy_templates.go:42
OnDeinit	backend/internal/connect/strategy/strategy_templates.go:105
OnDeinit	backend/internal/connect/strategy/strategy_templates.go:201
OnDeinit	backend/internal/connect/strategy/strategy_templates.go:295
OnInit	backend/internal/connect/strategy/strategy_templates.go:132
OnInit	backend/internal/connect/strategy/strategy_templates.go:227
OnInit	backend/internal/connect/strategy/strategy_templates.go:33
OnMail	reference/grpc/mt5.proto:554
OnMarketWatch	reference/grpc/mt5.proto:542
OnOpenedOrdersTickets	reference/grpc/mt5.proto:561
OnOrderProfit	reference/grpc/mt4.proto:356
OnOrderProfit	reference/grpc/mt5.proto:536
OnOrderUpdate	reference/grpc/mt4.proto:338
OnOrderUpdate	reference/grpc/mt5.proto:518
OnQuote	reference/grpc/mt4.proto:344
OnQuote	reference/grpc/mt5.proto:524
OnTickHistory	reference/grpc/mt5.proto:462
OnTickHistory	reference/grpc/mt5.proto:548
OnTickValue	reference/grpc/mt4.proto:350
OnTickValue	reference/grpc/mt5.proto:530
OpenedOrder	reference/grpc/mt4.proto:118
OpenedOrder	reference/grpc/mt5.proto:79
OpenedOrders	backend/internal/mthub/service.go:251
OpenedOrders	proto/ant/v1/mthub_service.proto:9
OpenedOrders	reference/grpc/mt4.proto:86
OpenedOrders	reference/grpc/mt5.proto:72
OpenedOrdersTickets	reference/grpc/mt5.proto:85
OrderClose	reference/grpc/mt4.proto:329
OrderClose	reference/grpc/mt5.proto:503
OrderCloseBy	reference/grpc/mt4.proto:312
OrderDelete	reference/grpc/mt4.proto:319
OrderHistory	backend/internal/mthub/service.go:273
OrderHistory	proto/ant/v1/mthub_service.proto:10
OrderHistory	reference/grpc/mt4.proto:126
OrderHistory	reference/grpc/mt5.proto:95
OrderHistoryPagination	reference/grpc/mt5.proto:118
OrderIDByTicket	backend/internal/mthub/oms_writer.go:149
OrderModify	reference/grpc/mt4.proto:304
OrderModify	reference/grpc/mt5.proto:493
OrderSend	reference/grpc/mt4.proto:294
OrderSend	reference/grpc/mt5.proto:482
OrderTypeString	backend/internal/mthub/order_types.go:87
PaperPnl	backend/internal/paper/engine.go:146
PendingOrderHistory	reference/grpc/mt5.proto:104
Ping	reference/grpc/mt4.proto:185
Ping	reference/grpc/mt5.proto:342
PingHost	reference/grpc/mt4.proto:192
PingHost	reference/grpc/mt5.proto:350
PingHostMany	reference/grpc/mt4.proto:194
PingHostMany	reference/grpc/mt5.proto:357
PlaceOrder	backend/internal/mthub/service_orders.go:21
PlaceOrder	proto/ant/v1/mthub_service.proto:7
PlacePaperOrder	backend/internal/paper/engine.go:65
Platform	backend/internal/mthub/service.go:208
PreHold	backend/internal/service/credit_service.go:91
PreviewOptimization	proto/ant/v1/marketplace_service.proto:85
PriceHistory	backend/internal/mthub/service.go:323
PriceHistory	proto/ant/v1/mthub_service.proto:13
PriceHistory	reference/grpc/mt5.proto:299
PriceHistoryEx	reference/grpc/mt5.proto:328
PriceHistoryExMany	reference/grpc/mt5.proto:337
PriceHistoryHighLow	reference/grpc/mt5.proto:318
PriceHistoryMany	reference/grpc/mt5.proto:308
PriceHistoryMonth	reference/grpc/mt5.proto:261
PriceHistoryMonthMany	reference/grpc/mt5.proto:271
PriceHistoryToday	reference/grpc/mt5.proto:281
PriceHistoryTodayMany	reference/grpc/mt5.proto:288
Process	backend/internal/risksvc/pipeline.go:96
PromoteCandidateToDraft	backend/internal/connect/strategy/strategy_experiment_handler.go:227
PromoteCandidateToDraft	proto/ant/v1/strategy_experiment.proto:17
PromoteToFull	backend/internal/risk/canary.go:234
Publish	backend/internal/mthub/broker_types.go:153
Publish	backend/internal/mthub/broker_types.go:227
Publish	backend/internal/mthub/broker_types.go:60
Publish	backend/internal/mthub/tick_broker.go:79
Publish	backend/internal/mthub/trade_broker.go:82
Publish	backend/internal/mthub/trade_event_store.go:95
Publish	backend/internal/mthub/types.go:165
PublishAccountProfit	backend/internal/mthub/service_account_profit.go:6
PublishAccountStatus	backend/internal/mthub/service.go:188
PublishBar	backend/internal/mthub/service.go:117
PublishEvent	backend/internal/mthub/types.go:234
PublishOptimization	proto/ant/v1/marketplace_service.proto:84
PublishPositionSnapshot	backend/internal/mthub/service.go:383
PublishStrategy	proto/ant/v1/admin_strategy.proto:26
PublishStrategy	proto/ant/v1/marketplace_service.proto:10
PublishTemplate	backend/internal/service/template_svc_admin.go:288
PublishTemplateDraft	backend/internal/connect/strategy/strategy_template_handlers.go:274
PublishTemplateDraft	proto/ant/v1/strategy.proto:20
PublishTick	backend/internal/mthub/service.go:134
PublishTradeEvent	backend/internal/mthub/service.go:171
PublishTradeEventFromUpdate	backend/internal/mthub/service_orders.go:390
PurchaseBundle	proto/ant/v1/marketplace_service.proto:92
PurchaseStrategy	proto/ant/v1/marketplace_service.proto:13
PutSnapshot	backend/internal/connect/strategy/position_cache.go:84
QuestionnaireCompletedAt	backend/internal/risksvc/jurisdiction_store.go:149
Quote	reference/grpc/mt4.proto:73
QuoteHistory	reference/grpc/mt4.proto:136
QuoteHistoryMany	reference/grpc/mt4.proto:145
RateStrategy	proto/ant/v1/marketplace_service.proto:17
Recalculate	backend/internal/risksvc/platform_aggregator.go:105
ReconcileAccount	backend/internal/mthub/reconciliation.go:53
ReconcilingCount	backend/internal/mthub/reconcile_gate.go:52
ReconnectAccount	proto/ant/v1/account.proto:21
RecordBalanceSnapshot	backend/internal/service/account_sync.go:247
RecordBar	backend/internal/connect/strategy/shadow_verifier.go:75
RecordCountry	backend/internal/risksvc/jurisdiction_store.go:64
RecordError	backend/internal/connect/strategy/session_registry.go:367
RecordEval	backend/internal/connect/strategy/session_diag.go:76
RecordIndicators	backend/internal/connect/strategy/session_diag.go:102
RecordLiveSignal	backend/internal/connect/strategy/shadow_verifier.go:92
RecordSignal	backend/internal/connect/strategy/session_registry.go:326
RecordSuccessfulTrade	backend/internal/risk/canary.go:188
RecordTick	backend/internal/connect/strategy/session_registry.go:305
RecordWindow	backend/internal/connect/strategy/session_diag.go:90
reference/grpc/mt4.proto:362://  rpc OnQuoteHistory (OnQuoteHistoryRequest) returns (OnQuoteHistoryReply);
reference/grpc/mt4.proto:368://  rpc OnDisconnect (OnDisconnectRequest) returns (OnDisconnectReply);
reference/grpc/mt5.proto:229://  rpc ClusterDetails (ClusterDetailsRequest) returns (ClusterDetailsReply);
Refresh	backend/internal/mthub/hub_estimator.go:149
RefreshToken	proto/ant/v1/auth.proto:13
RefreshTokenFromCookie	proto/ant/v1/auth.proto:14
RefundCredits	proto/ant/v1/credit.proto:18
Register	backend/internal/connect/strategy/session_registry.go:155
Register	backend/internal/mthub/types.go:47
Register	proto/ant/v1/auth.proto:16
RegisterUser	backend/internal/service/registration_service.go:72
RejectAutoGenTask	proto/ant/v1/marketplace_service.proto:46
RejectOptimizationTask	proto/ant/v1/marketplace_service.proto:83
ReleaseHold	backend/internal/service/credit_service.go:235
RemoveAccount	backend/internal/risk/canary.go:134
RemoveCredential	proto/ant/v1/webauthn.proto:21
RemoveSanctionedCountry	proto/ant/v1/admin_jurisdiction.proto:14
RemoveSession	backend/internal/mthub/types.go:98
RemoveWhitelistAddress	proto/ant/v1/webauthn.proto:35
Repo	backend/internal/service/wallet_service.go:29
RequestQuoteHistory	reference/grpc/mt4.proto:161
RequestRefund	proto/ant/v1/marketplace_service.proto:64
RequestVerification	proto/ant/v1/marketplace_service.proto:58
RequiredMargin	reference/grpc/mt5.proto:247
ResendVerification	proto/ant/v1/auth.proto:18
ResetBreaker	proto/ant/v1/admin_sre.proto:9
ResetPassword	proto/ant/v1/auth.proto:20
ResetPeakEquity	backend/internal/connect/strategy/account_provider.go:94
ResetUserPassword	proto/ant/v1/admin_user.proto:18
ResolveSession	proto/ant/v1/ai.proto:22
ResolveSymbol	backend/internal/service/platform_service.go:139
RestoreHolds	backend/internal/service/credit_service.go:61
RestoreUser	backend/internal/service/user_deletion_service.go:168
RestoreUser	proto/ant/v1/admin_user.proto:19
ReviewStrategyAsset	backend/internal/connect/strategy/strategy_asset_handler.go:120
ReviewStrategyAsset	proto/ant/v1/strategy_asset.proto:13
ReviseCode	proto/ant/v1/code_assist.proto:11
ReviseCodeStream	proto/ant/v1/code_assist.proto:12
RevokeAgentToken	proto/ant/v1/agent_gateway.proto:16
Rollback	backend/internal/risk/canary.go:290
RollbackStrategyVersion	backend/internal/connect/strategy/strategy_version_handler.go:75
RollbackStrategyVersion	proto/ant/v1/strategy_runtime.proto:58
Rules	backend/internal/risk/gate.go:191
Rules	backend/internal/risksvc/engine.go:49
Run	backend/internal/service/ledger_shipper.go:46
RunBacktest	backend/internal/connect/strategy/strategy_signals.go:15
RunBacktest	proto/ant/v1/backtest_service.proto:14
RunBacktest	proto/ant/v1/strategy.proto:31
RunEvaluation	proto/ant/v1/ai_gate.proto:11
RunLiveStrategy	backend/internal/connect/strategy/live_runner.go:138
RunMarketBacktest	proto/ant/v1/marketplace_service.proto:30
RunStrategy	proto/ant/v1/backtest_service.proto:16
SaveUserTemplate	proto/ant/v1/agent_gateway.proto:40
Search	reference/grpc/mt4.proto:215
Search	reference/grpc/mt5.proto:365
SearchBroker	proto/ant/v1/account.proto:22
SearchExperience	proto/ant/v1/agent_gateway.proto:33
SendEvent	backend/internal/connect/strategy/vm_live_session.go:119
SendNotification	proto/ant/v1/notification_service.proto:16
ServerTimezone	reference/grpc/mt4.proto:105
ServerTimezone	reference/grpc/mt5.proto:184
SessionState	backend/internal/mthub/service.go:218
Set	backend/internal/risksvc/capability.go:109
Set	backend/internal/service/analytics_cache.go:43
SetAccountLookup	backend/internal/connect/strategy/strategy_execution_handler.go:169
SetAccountNumber	backend/internal/service/user/account_number.go:236
SetAccountOwnerVerifier	backend/internal/mthub/service_setters.go:38
SetAccountProvider	backend/internal/connect/strategy/strategy_execution_handler.go:215
SetAccountStateProvider	backend/internal/mthub/service_setters.go:71
SetAIPrimary	backend/internal/service/systemai/service.go:211
SetAIPrimary	proto/ant/v1/ai_primary.proto:9
SetAIService	backend/internal/connect/strategy/strategy_experiment_worker.go:86
SetAttribution	backend/internal/service/analytics_cache.go:67
SetAutotradeEnabled	backend/internal/risk/gate.go:106
SetBarBroker	backend/internal/mthub/service_setters.go:44
SetBarSource	backend/internal/connect/strategy/strategy_execution_handler.go:157
SetBoundSvc	backend/internal/connect/strategy/strategy_execution_handler.go:191
SetBoundSvc	backend/internal/connect/strategy/strategy_handler.go:66
SetBrokerCompanyLookup	backend/internal/connect/strategy/strategy_execution_handler.go:176
SetBrokerLimits	backend/internal/risksvc/platform_aggregator.go:96
SetBrokerRegistry	backend/internal/mthub/service_setters.go:29
SetCanary	proto/ant/v1/admin_sre.proto:11
SetCircuitBreakerDB	backend/internal/service/systemai/chat_failover.go:26
SetCircuitOpen	backend/internal/connect/strategy/session_registry.go:387
SetCodeAccessChecker	backend/internal/connect/strategy/strategy_handler.go:41
SetCompromisedChecker	backend/internal/service/deposit_service.go:110
SetConfig	proto/ant/v1/admin_config.proto:11
SetCostBreaker	backend/internal/service/systemai/service.go:124
SetCostEstimator	backend/internal/mthub/service_setters.go:68
SetCoverageChecker	backend/internal/connect/strategy/strategy_execution_handler.go:126
SetDiag	backend/internal/connect/strategy/vm_live_session.go:185
SetEmailVerification	backend/internal/service/registration_service.go:61
SetEngine	backend/internal/connect/strategy/strategy_handler.go:64
SetExecutor	backend/internal/connect/strategy/strategy_experiment_worker.go:89
SetFailureSignatureRepo	backend/internal/connect/strategy/strategy_execution_handler.go:224
SetGate	backend/internal/connect/strategy/strategy_execution_handler.go:205
SetGate	backend/internal/mthub/service_setters.go:62
SetGateEvalRepo	backend/internal/connect/strategy/strategy_execution_handler.go:132
SetGatewayProviderRepo	backend/internal/service/systemai/service.go:118
SetGoExecutor	backend/internal/connect/strategy/strategy_execution_handler.go:160
SetGuard	backend/internal/mthub/service_setters.go:56
SetGuard	backend/internal/paper/engine.go:51
SetHookConfig	proto/ant/v1/agent_hooks.proto:14
SetImportedRepo	backend/internal/connect/strategy/strategy_execution_handler.go:162
SetKillSwitch	backend/internal/mthub/service_setters.go:15
SetKillSwitch	backend/internal/risk/gate.go:99
SetKillSwitch	proto/ant/v1/admin_sre.proto:7
SetKYCStatus	backend/internal/risksvc/jurisdiction_store.go:51
SetKYCStatus	proto/ant/v1/admin_jurisdiction.proto:11
SetLogger	backend/internal/connect/strategy/session_registry.go:82
SetLogger	backend/internal/mthub/service_setters.go:41
SetLogger	backend/internal/service/account_service.go:59
SetLogger	backend/internal/service/platform_service.go:29
SetLogger	backend/internal/service/systemai/service.go:87
SetLogRepository	backend/internal/connect/strategy/session_registry.go:85
SetManagedSetting	proto/ant/v1/admin_settings.proto:15
SetManagedSettingProvider	backend/internal/service/daily_quota.go:155
SetManagedSettingProvider	backend/internal/service/daily_quota.go:84
SetMarketDataRepo	backend/internal/connect/strategy/strategy_execution_handler.go:142
SetModelFilter	backend/internal/service/systemai/service.go:131
SetMonthlyDetail	backend/internal/service/analytics_cache.go:116
SetMtHub	backend/internal/connect/strategy/strategy_execution_handler.go:158
SetMtHubService	backend/internal/mthub/reconciliation.go:30
SetNotificationPrefs	proto/ant/v1/notification_service.proto:19
SetNotificationSender	backend/internal/connect/strategy/strategy_execution_handler.go:223
SetNotificationSender	backend/internal/service/account_sync_service.go:48
SetOmsWriter	backend/internal/mthub/service_setters.go:59
SetOnBacktestComplete	backend/internal/connect/strategy/strategy_execution_handler.go:227
SetOrderEventBroker	backend/internal/mthub/oms_writer.go:101
SetPaperEngine	backend/internal/connect/strategy/strategy_execution_handler.go:159
SetPgListen	backend/internal/connect/strategy/divergence_handler.go:38
SetPgListen	backend/internal/connect/strategy/strategy_execution_handler.go:364
SetPgListen	backend/internal/connect/strategy/strategy_experiment_handler.go:304
SetPgListen	backend/internal/connect/strategy/strategy_experiment_worker.go:68
SetPgListen	backend/internal/connect/strategy/strategy_handler.go:86
SetPgListen	backend/internal/connect/strategy/walk_forward_handler.go:36
SetPgListen	backend/internal/service/quota_checker.go:158
SetPlacedType	reference/grpc/mt4.proto:168
SetPnL	backend/internal/connect/strategy/session_registry.go:315
SetPositionCache	backend/internal/connect/strategy/account_provider.go:46
SetPositionCache	backend/internal/connect/strategy/strategy_execution_handler.go:172
SetPositionCache	backend/internal/connect/strategy/strategy_handler.go:70
SetPostCallBiller	backend/internal/service/systemai/service.go:113
SetQualityValidator	backend/internal/connect/strategy/strategy_execution_handler.go:129
SetQuotaChecker	backend/internal/connect/strategy/strategy_execution_handler.go:188
SetReconcileTrigger	backend/internal/mthub/service_setters.go:20
SetRolling	backend/internal/service/analytics_cache.go:91
SetRunRepo	backend/internal/connect/strategy/strategy_execution_handler.go:161
SetSanctionedOverride	proto/ant/v1/admin_jurisdiction.proto:16
SetScheduleActive	backend/internal/service/schedule_svc.go:156
SetScheduleNameLookup	backend/internal/connect/strategy/strategy_execution_handler.go:173
SetScheduleResolver	backend/internal/service/account_sync_service.go:51
SetSessionRegistry	backend/internal/connect/strategy/strategy_execution_handler.go:168
SetSessionRegistry	backend/internal/connect/strategy/strategy_handler.go:68
SetStatus	backend/internal/service/account_lifecycle.go:173
SetStatusBroker	backend/internal/mthub/service_setters.go:53
SetStderrTail	backend/internal/connect/strategy/session_registry.go:401
SetStrategyPricing	proto/ant/v1/marketplace_service.proto:22
SetSubscriptionEnsurer	backend/internal/service/registration_service.go:66
SetTemplateStatus	backend/internal/service/template_svc.go:129
SetTickBroker	backend/internal/mthub/service_setters.go:47
SetTicket	backend/internal/mthub/idempotency.go:167
Settle	backend/internal/service/credit_service.go:126
SetTradeBroker	backend/internal/mthub/service_setters.go:50
SetUsageRepos	backend/internal/service/subscription_service.go:42
SetUserLimiter	backend/internal/mthub/service_setters.go:65
SetUserLimiter	backend/internal/risksvc/engine.go:22
SetUserRepo	backend/internal/service/systemai/service.go:90
SetUserSetting	proto/ant/v1/agent_gateway.proto:46
SetVersionRepo	backend/internal/connect/strategy/strategy_execution_handler.go:165
SetWalletChecker	backend/internal/service/systemai/service.go:107
Shutdown	backend/internal/risksvc/platform_aggregator.go:181
Size	backend/internal/risksvc/kelly_sizer.go:46
Size	backend/internal/risksvc/vol_target_sizer.go:47
SnapshotDiag	backend/internal/connect/strategy/session_diag.go:145
SoftDeleteUser	backend/internal/service/user_deletion_service.go:35
SoftDeleteUsers	backend/internal/service/user_deletion_service.go:86
Start	backend/internal/connect/strategy/schedule_engine.go:99
Start	backend/internal/connect/strategy/shadow_verifier.go:60
Start	backend/internal/connect/strategy/strategy_experiment_worker.go:44
Start	backend/internal/connect/strategy/vm_live_session.go:81
Start	backend/internal/mthub/derived_state.go:111
Start	backend/internal/mthub/reconciliation.go:37
Start	backend/internal/mthub/snapshot_persister.go:45
StartAlgo	proto/ant/v1/execution_algo.proto:14
StartBacktestRun	backend/internal/connect/strategy/strategy_backtest_crud.go:25
StartBacktestRun	proto/ant/v1/strategy_runtime.proto:20
StartBacktestWorker	backend/internal/connect/strategy/backtest_worker.go:203
StartPaperStrategy	proto/ant/v1/paper_trading.proto:13
StartPlatformRenewalLoop	backend/internal/service/subscription_renewal.go:18
StartRefreshLoop	backend/internal/risksvc/platform_aggregator.go:155
StartRefreshLoop	backend/internal/service/quota_checker.go:165
StartSchedule	backend/internal/connect/strategy/schedule_event.go:17
StartStrategy	backend/internal/connect/strategy/strategy_active_control.go:58
StartStrategy	proto/ant/v1/strategy_runtime.proto:49
StartTrial	proto/ant/v1/marketplace_service.proto:52
State	backend/internal/mthub/derived_state.go:121
Stats	backend/internal/mthub/state_cache.go:220
StepUpLotSize	backend/internal/risk/canary.go:205
Stop	backend/internal/connect/strategy/schedule_engine.go:435
Stop	backend/internal/connect/strategy/session_registry.go:293
Stop	backend/internal/connect/strategy/shadow_verifier.go:65
Stop	backend/internal/connect/strategy/strategy_experiment_worker.go:83
Stop	backend/internal/mthub/derived_state.go:116
StopPaperStrategy	proto/ant/v1/paper_trading.proto:15
StopSchedule	backend/internal/connect/strategy/schedule_engine.go:423
StopStrategy	backend/internal/connect/strategy/strategy_active_control.go:20
StopStrategy	proto/ant/v1/strategy_runtime.proto:45
StoreExperience	proto/ant/v1/agent_gateway.proto:36
StreamNotifications	proto/ant/v1/notification_service.proto:15
StreamOrderEvents	proto/ant/v1/mthub_service.proto:15
StreamTicks	proto/ant/v1/market_service.proto:10
String	backend/internal/risk/gate.go:233
SubmitAssetReview	backend/internal/connect/strategy/strategy_asset_handler.go:96
SubmitAssetReview	proto/ant/v1/strategy_asset.proto:12
SubmitQuestionnaire	backend/internal/risksvc/jurisdiction_store.go:112
SubmitSessionFeedback	proto/ant/v1/ai.proto:24
SubmitStrategy	proto/ant/v1/agent_gateway.proto:26
SubmitStrategyExperiment	backend/internal/connect/strategy/strategy_experiment_handler.go:113
SubmitStrategyExperiment	proto/ant/v1/strategy_experiment.proto:11
Subscribe	backend/internal/connect/strategy/data_source.go:68
Subscribe	backend/internal/connect/strategy/position_cache.go:36
Subscribe	backend/internal/mthub/broker_types.go:185
Subscribe	backend/internal/mthub/broker_types.go:241
Subscribe	backend/internal/mthub/broker_types.go:84
Subscribe	backend/internal/mthub/tick_broker.go:57
Subscribe	backend/internal/mthub/trade_broker.go:57
Subscribe	backend/internal/mthub/types.go:206
Subscribe	backend/internal/mthub/types.go:245
Subscribe	backend/internal/paper/engine.go:246
Subscribe	backend/internal/service/subscription_service.go:57
Subscribe	proto/ant/v1/marketplace_service.proto:11
Subscribe	proto/ant/v1/subscription.proto:15
Subscribe	reference/grpc/mt4.proto:230
Subscribe	reference/grpc/mt5.proto:387
SubscribeAccountProfit	backend/internal/mthub/service_account_profit.go:11
SubscribeAccountProfitAll	backend/internal/mthub/service_account_profit.go:16
SubscribeAccountStatus	backend/internal/mthub/service.go:195
SubscribeAll	backend/internal/mthub/broker_types.go:104
SubscribeBars	proto/ant/v1/mthub_service.proto:17
SubscribeBarUpdates	backend/internal/mthub/service.go:124
SubscribeEvents	proto/ant/v1/stream.proto:75
SubscribeHistory	proto/ant/v1/stream.proto:76
SubscribeIndicators	proto/ant/v1/stream.proto:80
SubscribeJob	proto/ant/v1/job.proto:13
SubscribeMany	reference/grpc/mt4.proto:237
SubscribeMany	reference/grpc/mt5.proto:394
SubscribeMarketWatch	reference/grpc/mt5.proto:433
SubscribeMetrics	proto/ant/v1/admin_monitor.proto:12
SubscribeOpenedOrdersTickets	reference/grpc/mt5.proto:440
SubscribeOrderProfit	reference/grpc/mt4.proto:256
SubscribeOrderProfit	reference/grpc/mt5.proto:413
SubscribeOrderUpdate	reference/grpc/mt4.proto:270
SubscribeOrderUpdate	reference/grpc/mt5.proto:427
SubscribeOrderUpdates	proto/ant/v1/stream.proto:77
SubscribePositionSnapshots	backend/internal/mthub/service.go:388
SubscribeProfitUpdates	proto/ant/v1/stream.proto:78
SubscribeQuoteHistory	reference/grpc/mt4.proto:276
SubscribeSignals	backend/internal/connect/strategy/session_registry.go:344
SubscribeSymbols	backend/internal/mthub/service.go:369
SubscribeTickUpdates	backend/internal/mthub/service.go:141
SubscribeTickValue	reference/grpc/mt4.proto:264
SubscribeTickValue	reference/grpc/mt5.proto:421
SubscribeToMthub	backend/internal/connect/strategy/session_registry.go:89
SubscribeTradeEvents	backend/internal/mthub/service.go:178
SubscribeUserOrderEvents	backend/internal/mthub/service.go:378
SubscribeUserSummary	proto/ant/v1/stream.proto:79
SymbolList	backend/internal/mthub/service.go:345
SymbolList	proto/ant/v1/mthub_service.proto:12
SymbolList	reference/grpc/mt5.proto:130
SymbolParams	backend/internal/mthub/service.go:295
SymbolParams	proto/ant/v1/mthub_service.proto:11
SymbolParams	reference/grpc/mt4.proto:99
SymbolParams	reference/grpc/mt5.proto:158
SymbolParamsMany	reference/grpc/mt4.proto:111
SymbolParamsMany	reference/grpc/mt5.proto:165
Symbols	reference/grpc/mt4.proto:92
Symbols	reference/grpc/mt5.proto:124
SymbolSessionsEx	reference/grpc/mt5.proto:172
SymbolSessionsExMany	reference/grpc/mt5.proto:178
SyncAccountHistory	backend/internal/service/account_sync_service.go:56
SyncOrderHistory	proto/ant/v1/mthub_service.proto:16
SyncStrategyAsset	backend/internal/connect/strategy/strategy_asset_handler.go:171
SyncStrategyAsset	proto/ant/v1/strategy_asset.proto:16
Threshold	backend/internal/service/daily_quota.go:226
TickHistoryRequest	reference/grpc/mt5.proto:453
TickHistoryStop	reference/grpc/mt5.proto:460
TickValueWithSize	reference/grpc/mt4.proto:180
TickValueWithSize	reference/grpc/mt5.proto:222
TierCheck	backend/internal/risksvc/capability.go:76
ToggleAutoTrade	proto/ant/v1/auto_trading.proto:17
ToggleConfigEnabled	proto/ant/v1/admin_config.proto:12
ToggleSchedule	backend/internal/connect/strategy/strategy_schedules.go:260
ToggleSchedule	proto/ant/v1/strategy.proto:29
Total	backend/internal/service/systemai/session_quota.go:31
TransformCode	proto/ant/v1/code_assist.proto:16
Transition	backend/internal/mthub/oms_writer.go:161
TransitionOrderByTicket	backend/internal/mthub/service_orders.go:347
TranslateParamLabels	proto/ant/v1/code_assist.proto:18
TriggerBatchGeneration	proto/ant/v1/marketplace_service.proto:48
TriggerReconcile	backend/internal/mthub/reconciliation.go:62
UnbindAccount	backend/internal/service/bound_account_svc.go:85
UnbindAccount	proto/ant/v1/subscription.proto:25
UnflagStrategy	proto/ant/v1/admin_strategy.proto:24
UnflagTemplate	backend/internal/service/template_svc_admin.go:263
UnfreezeAccount	proto/ant/v1/admin_account.proto:12
UnpublishStrategy	proto/ant/v1/admin_strategy.proto:25
UnpublishStrategy	proto/ant/v1/marketplace_service.proto:24
UnpublishTemplate	backend/internal/service/template_svc_admin.go:276
UnpublishUserTemplate	backend/internal/service/template_svc.go:116
Unsubscribe	backend/internal/connect/strategy/position_cache.go:89
Unsubscribe	proto/ant/v1/marketplace_service.proto:12
UnSubscribe	reference/grpc/mt4.proto:244
UnSubscribe	reference/grpc/mt5.proto:401
UnSubscribeMany	reference/grpc/mt4.proto:250
UnSubscribeMany	reference/grpc/mt5.proto:407
Update	backend/internal/mthub/derived_state.go:62
UpdateAccount	backend/internal/service/account_crud.go:95
UpdateAccount	proto/ant/v1/account.proto:17
UpdateAccountInfo	backend/internal/service/account_lifecycle.go:69
UpdateAccountInfoTx	backend/internal/service/account_lifecycle.go:54
UpdateAccountMetrics	backend/internal/service/account_lifecycle.go:84
UpdateBacktestRun	backend/internal/connect/strategy/strategy_backtest_crud.go:294
UpdateBacktestRun	proto/ant/v1/strategy_runtime.proto:27
UpdateBrokerThresholds	backend/internal/service/account_lifecycle.go:128
UpdateConfig	backend/internal/service/systemai/service.go:197
UpdateConversationTitle	proto/ant/v1/ai.proto:19
UpdateFeeTier	proto/ant/v1/marketplace_service.proto:96
UpdateGlobalSettings	proto/ant/v1/auto_trading.proto:16
UpdateOrderHistoryClose	backend/internal/service/log_service.go:35
UpdatePnlFromPositions	backend/internal/connect/strategy/session_registry.go:234
UpdatePosition	backend/internal/risksvc/platform_aggregator.go:76
UpdateProvider	proto/ant/v1/ai_gateway.proto:22
UpdateRiskConfig	proto/ant/v1/auto_trading.proto:19
UpdateSchedule	backend/internal/connect/strategy/strategy_schedules.go:118
UpdateSchedule	backend/internal/service/schedule_svc.go:122
UpdateSchedule	proto/ant/v1/strategy.proto:27
UpdateSecret	backend/internal/service/systemai/service.go:221
UpdateSessionStrategyKey	proto/ant/v1/ai.proto:23
UpdateShareToken	proto/ant/v1/share.proto:10
UpdateStrategyCode	backend/internal/connect/strategy/strategy_version_handler.go:134
UpdateStrategyCode	proto/ant/v1/strategy_runtime.proto:62
UpdateSummaryCache	backend/internal/service/account_sync.go:120
UpdateSystemAIConfig	proto/ant/v1/system_ai.proto:13
UpdateSystemAISecret	proto/ant/v1/system_ai.proto:14
UpdateSystemStrategy	backend/internal/service/template_svc_admin.go:98
UpdateSystemStrategy	proto/ant/v1/admin_strategy.proto:15
UpdateTemplate	backend/internal/connect/strategy/strategy_template_handlers.go:153
UpdateTemplate	backend/internal/service/template_svc.go:90
UpdateTemplate	proto/ant/v1/strategy.proto:14
UpdateTemplateDraft	backend/internal/connect/strategy/strategy_template_handlers.go:231
UpdateTemplateDraft	proto/ant/v1/strategy.proto:19
UpdateTicket	backend/internal/mthub/oms_writer.go:137
UpdateTradingPassword	proto/ant/v1/account.proto:24
UpdateUser	proto/ant/v1/admin_user.proto:13
UpdateXpub	backend/internal/service/deposit_service.go:93
UpsertModel	proto/ant/v1/ai_gateway.proto:28
UserOwnsAccount	backend/internal/service/account_lifecycle.go:148
UserOwnsAccount	backend/internal/service/platform_service.go:118
Validate	backend/internal/connect/strategy/strategy_execution_handler.go:264
Validate	proto/ant/v1/strategy_runtime.proto:18
ValidateCoupon	proto/ant/v1/marketplace_service.proto:72
ValidateStrategy	proto/ant/v1/backtest_service.proto:15
ValidateStrategyExtended	proto/ant/v1/code_assist.proto:14
ValidateSystemAIConnection	proto/ant/v1/system_ai.proto:16
VerifyAccount	proto/ant/v1/account.proto:25
VerifyEmail	proto/ant/v1/auth.proto:17
VerifyMTIdentity	proto/ant/v1/auth.proto:21
VerifyToken	backend/internal/service/email_verification.go:67
VerifyTradePermission	proto/ant/v1/account.proto:23
Version	reference/grpc/mt5.proto:376
WaitSession	backend/internal/mthub/types.go:64
Watch	backend/internal/connect/strategy/session_registry.go:131
WatchActiveStrategies	backend/internal/connect/strategy/strategy_active_watch.go:176
WatchActiveStrategies	proto/ant/v1/strategy_runtime.proto:52
WatchAll	backend/internal/mthub/tick_broker.go:108
WatchAll	backend/internal/mthub/types.go:187
WatchAllTicks	backend/internal/mthub/service.go:161
WatchBacktestRun	backend/internal/connect/strategy/strategy_backtest_watch.go:15
WatchBacktestRun	proto/ant/v1/strategy_runtime.proto:23
WatchDivergenceReport	backend/internal/connect/strategy/divergence_handler.go:60
WatchDivergenceReport	proto/ant/v1/live_backtest_divergence.proto:16
WatchExperiment	backend/internal/connect/strategy/strategy_experiment_handler.go:245
WatchExperiment	proto/ant/v1/strategy_experiment.proto:20
WatchHealthAlerts	proto/ant/v1/platform_health.proto:18
WatchPaperAccount	proto/ant/v1/paper_trading.proto:16
WatchSchedules	backend/internal/connect/strategy/strategy_schedules.go:306
WatchSchedules	proto/ant/v1/strategy.proto:30
WatchStrategySignals	backend/internal/connect/strategy/strategy_active_watch.go:87
WatchStrategySignals	proto/ant/v1/strategy_runtime.proto:47
WatchWalkForwardReport	backend/internal/connect/strategy/walk_forward_handler.go:57
WatchWalkForwardReport	proto/ant/v1/walk_forward.proto:15
Xpub	backend/internal/service/deposit_service.go:77
XpubKey	backend/internal/service/deposit_service.go:85
```

## 公开 ConnectRPC（前端/跨服务可直接调用）

```
proto/ant/v1/account.proto:14:  rpc ListAccounts(ListAccountsRequest) returns (ListAccountsResponse);
proto/ant/v1/account.proto:15:  rpc GetAccount(GetAccountRequest) returns (Account);
proto/ant/v1/account.proto:16:  rpc CreateAccount(CreateAccountRequest) returns (Account);
proto/ant/v1/account.proto:17:  rpc UpdateAccount(UpdateAccountRequest) returns (Account);
proto/ant/v1/account.proto:18:  rpc DeleteAccount(DeleteAccountRequest) returns (google.protobuf.Empty);
proto/ant/v1/account.proto:19:  rpc ConnectAccount(ConnectAccountRequest) returns (ConnectAccountResponse);
proto/ant/v1/account.proto:20:  rpc DisconnectAccount(DisconnectAccountRequest) returns (google.protobuf.Empty);
proto/ant/v1/account.proto:21:  rpc ReconnectAccount(ReconnectAccountRequest) returns (google.protobuf.Empty);
proto/ant/v1/account.proto:22:  rpc SearchBroker(SearchBrokerRequest) returns (SearchBrokerResponse);
proto/ant/v1/account.proto:23:  rpc VerifyTradePermission(VerifyTradePermissionRequest) returns (VerifyTradePermissionResponse);
proto/ant/v1/account.proto:24:  rpc UpdateTradingPassword(UpdateTradingPasswordRequest) returns (UpdateTradingPasswordResponse);
proto/ant/v1/account.proto:25:  rpc VerifyAccount(VerifyAccountRequest) returns (VerifyAccountResponse);
proto/ant/v1/admin_account.proto:10:  rpc ListAccountsAdmin(ListAccountsAdminRequest) returns (ListAccountsAdminResponse);
proto/ant/v1/admin_account.proto:11:  rpc FreezeAccount(FreezeAccountRequest) returns (FreezeAccountResponse);
proto/ant/v1/admin_account.proto:12:  rpc UnfreezeAccount(UnfreezeAccountRequest) returns (UnfreezeAccountResponse);
proto/ant/v1/admin_account.proto:13:  rpc GetAccountAuditLogs(GetAccountAuditLogsRequest) returns (GetAccountAuditLogsResponse);
proto/ant/v1/admin_billing.proto:11:  rpc ListSubscriptions(ListAdminSubscriptionsRequest) returns (ListAdminSubscriptionsResponse);
proto/ant/v1/admin_billing.proto:13:  rpc GetRevenueSummary(GetRevenueSummaryRequest) returns (GetRevenueSummaryResponse);
proto/ant/v1/admin_billing.proto:15:  rpc ListAdminWalletTransactions(ListAdminWalletTransactionsRequest) returns (ListAdminWalletTransactionsResponse);
proto/ant/v1/admin_billing.proto:17:  rpc GetLedgerSummary(GetLedgerSummaryRequest) returns (GetLedgerSummaryResponse);
proto/ant/v1/admin_config.proto:10:  rpc ListConfigs(ListConfigsRequest) returns (ListConfigsResponse);
proto/ant/v1/admin_config.proto:11:  rpc SetConfig(SetConfigRequest) returns (SetConfigResponse);
proto/ant/v1/admin_config.proto:12:  rpc ToggleConfigEnabled(ToggleConfigEnabledRequest) returns (ToggleConfigEnabledResponse);
proto/ant/v1/admin_jurisdiction.proto:10:  rpc GetJurisdictionStatus(GetJurisdictionStatusRequest) returns (GetJurisdictionStatusResponse);
proto/ant/v1/admin_jurisdiction.proto:11:  rpc SetKYCStatus(SetKYCStatusRequest) returns (SetKYCStatusResponse);
proto/ant/v1/admin_jurisdiction.proto:12:  rpc ListSanctionedCountries(ListSanctionedCountriesRequest) returns (ListSanctionedCountriesResponse);
proto/ant/v1/admin_jurisdiction.proto:13:  rpc AddSanctionedCountry(AddSanctionedCountryRequest) returns (AddSanctionedCountryResponse);
proto/ant/v1/admin_jurisdiction.proto:14:  rpc RemoveSanctionedCountry(RemoveSanctionedCountryRequest) returns (RemoveSanctionedCountryResponse);
proto/ant/v1/admin_jurisdiction.proto:15:  rpc ListUsersByKYCStatus(ListUsersByKYCStatusRequest) returns (ListUsersByKYCStatusResponse);
proto/ant/v1/admin_jurisdiction.proto:16:  rpc SetSanctionedOverride(SetSanctionedOverrideRequest) returns (SetSanctionedOverrideResponse);
proto/ant/v1/admin_log.proto:10:  rpc ListLogs(ListLogsRequest) returns (ListLogsResponse);
proto/ant/v1/admin_monitor.proto:12:  rpc SubscribeMetrics(SubscribeMetricsRequest) returns (stream MonitorSnapshot);
proto/ant/v1/admin_settings.proto:13:  rpc GetManagedSettings(GetManagedSettingsRequest) returns (GetManagedSettingsResponse);
proto/ant/v1/admin_settings.proto:15:  rpc SetManagedSetting(SetManagedSettingRequest) returns (SetManagedSettingResponse);
proto/ant/v1/admin_settings.proto:17:  rpc DeleteManagedSetting(DeleteManagedSettingRequest) returns (DeleteManagedSettingResponse);
proto/ant/v1/admin_sre.proto:10:  rpc GetCanary(GetCanaryRequest) returns (CanaryStatus);
proto/ant/v1/admin_sre.proto:11:  rpc SetCanary(SetCanaryRequest) returns (CanaryStatus);
proto/ant/v1/admin_sre.proto:6:  rpc GetKillSwitch(GetKillSwitchRequest) returns (KillSwitchStatus);
proto/ant/v1/admin_sre.proto:7:  rpc SetKillSwitch(SetKillSwitchRequest) returns (KillSwitchStatus);
proto/ant/v1/admin_sre.proto:8:  rpc ListBreakers(ListBreakersRequest) returns (ListBreakersResponse);
proto/ant/v1/admin_sre.proto:9:  rpc ResetBreaker(ResetBreakerRequest) returns (BreakerStatus);
proto/ant/v1/admin_strategy.proto:13:  rpc ListSystemStrategies(ListSystemStrategiesRequest) returns (ListSystemStrategiesResponse);
proto/ant/v1/admin_strategy.proto:14:  rpc CreateSystemStrategy(CreateSystemStrategyRequest) returns (SystemStrategy);
proto/ant/v1/admin_strategy.proto:15:  rpc UpdateSystemStrategy(UpdateSystemStrategyRequest) returns (SystemStrategy);
proto/ant/v1/admin_strategy.proto:16:  rpc DeleteSystemStrategy(DeleteSystemStrategyRequest) returns (DeleteSystemStrategyResponse);
proto/ant/v1/admin_strategy.proto:19:  rpc ListAllStrategies(ListAllStrategiesRequest) returns (ListAllStrategiesResponse);
proto/ant/v1/admin_strategy.proto:20:  rpc GetStrategyDetail(GetStrategyDetailRequest) returns (StrategyDetail);
proto/ant/v1/admin_strategy.proto:23:  rpc FlagStrategy(FlagStrategyRequest) returns (FlagStrategyResponse);
proto/ant/v1/admin_strategy.proto:24:  rpc UnflagStrategy(UnflagStrategyRequest) returns (UnflagStrategyResponse);
proto/ant/v1/admin_strategy.proto:25:  rpc UnpublishStrategy(UnpublishStrategyRequest) returns (UnpublishStrategyResponse);
proto/ant/v1/admin_strategy.proto:26:  rpc PublishStrategy(AdminPublishStrategyRequest) returns (AdminPublishStrategyResponse);
proto/ant/v1/admin_strategy.proto:27:  rpc DisableStrategy(DisableStrategyRequest) returns (DisableStrategyResponse);
proto/ant/v1/admin_strategy.proto:28:  rpc EnableStrategy(EnableStrategyRequest) returns (EnableStrategyResponse);
proto/ant/v1/admin_strategy.proto:29:  rpc ArchiveStrategy(ArchiveStrategyRequest) returns (ArchiveStrategyResponse);
proto/ant/v1/admin_system.proto:8:  rpc HealthCheck(HealthCheckRequest) returns (HealthCheckResponse);
proto/ant/v1/admin_system.proto:9:  rpc GetMetrics(GetMetricsRequest) returns (GetMetricsResponse);
proto/ant/v1/admin_trading.proto:8:  rpc GetTradingSummary(GetTradingSummaryRequest) returns (TradingSummary);
proto/ant/v1/admin_user.proto:10:  rpc GetDashboard(GetDashboardRequest) returns (GetDashboardResponse);
proto/ant/v1/admin_user.proto:11:  rpc ListUsers(ListUsersRequest) returns (ListUsersResponse);
proto/ant/v1/admin_user.proto:12:  rpc CreateUser(CreateUserRequest) returns (CreateUserResponse);
proto/ant/v1/admin_user.proto:13:  rpc UpdateUser(UpdateUserRequest) returns (UpdateUserResponse);
proto/ant/v1/admin_user.proto:14:  rpc DeleteUser(DeleteUserRequest) returns (DeleteUserResponse);
proto/ant/v1/admin_user.proto:15:  rpc DeleteUsers(DeleteUsersRequest) returns (DeleteUsersResponse);
proto/ant/v1/admin_user.proto:16:  rpc DisableUser(DisableUserRequest) returns (DisableUserResponse);
proto/ant/v1/admin_user.proto:17:  rpc EnableUser(EnableUserRequest) returns (EnableUserResponse);
proto/ant/v1/admin_user.proto:18:  rpc ResetUserPassword(ResetUserPasswordRequest) returns (ResetUserPasswordResponse);
proto/ant/v1/admin_user.proto:19:  rpc RestoreUser(RestoreUserRequest) returns (RestoreUserResponse);
proto/ant/v1/agent_gateway.proto:14:  rpc IssueAgentToken(IssueAgentTokenRequest) returns (IssueAgentTokenResponse);
proto/ant/v1/agent_gateway.proto:15:  rpc ListAgentTokens(ListAgentTokensRequest) returns (ListAgentTokensResponse);
proto/ant/v1/agent_gateway.proto:16:  rpc RevokeAgentToken(RevokeAgentTokenRequest) returns (AgentToken);
proto/ant/v1/agent_gateway.proto:17:  rpc ListAgentAudit(ListAgentAuditRequest) returns (ListAgentAuditResponse);
proto/ant/v1/agent_gateway.proto:18:  rpc GetAgentCapabilities(GetAgentCapabilitiesRequest) returns (AgentCapabilities);
proto/ant/v1/agent_gateway.proto:26:  rpc SubmitStrategy(SubmitStrategyRequest) returns (SubmitStrategyResponse);
proto/ant/v1/agent_gateway.proto:30:  rpc GenerateStrategy(AgentGenerateStrategyRequest) returns (stream AgentGenerateStrategyChunk);
proto/ant/v1/agent_gateway.proto:33:  rpc SearchExperience(SearchExperienceRequest) returns (SearchExperienceResponse);
proto/ant/v1/agent_gateway.proto:36:  rpc StoreExperience(StoreExperienceRequest) returns (StoreExperienceResponse);
proto/ant/v1/agent_gateway.proto:39:  rpc ListMemory(ListMemoryRequest) returns (ListMemoryResponse);
proto/ant/v1/agent_gateway.proto:40:  rpc SaveUserTemplate(SaveUserTemplateRequest) returns (SaveUserTemplateResponse);
proto/ant/v1/agent_gateway.proto:41:  rpc DeleteUserTemplate(DeleteUserTemplateRequest) returns (DeleteUserTemplateResponse);
proto/ant/v1/agent_gateway.proto:42:  rpc DeleteAgentExperience(DeleteAgentExperienceRequest) returns (DeleteAgentExperienceResponse);
proto/ant/v1/agent_gateway.proto:45:  rpc GetAgentSettings(GetAgentSettingsRequest) returns (GetAgentSettingsResponse);
proto/ant/v1/agent_gateway.proto:46:  rpc SetUserSetting(SetUserSettingRequest) returns (SetUserSettingResponse);
proto/ant/v1/agent_gateway.proto:47:  rpc DeleteUserSetting(DeleteUserSettingRequest) returns (DeleteUserSettingResponse);
proto/ant/v1/agent_gateway.proto:50:  rpc GetCapabilities(GetCapabilitiesRequest) returns (GetCapabilitiesResponse);
proto/ant/v1/agent_hooks.proto:12:  rpc ListHookConfigs(ListHookConfigsRequest) returns (ListHookConfigsResponse);
proto/ant/v1/agent_hooks.proto:14:  rpc SetHookConfig(SetHookConfigRequest) returns (SetHookConfigResponse);
proto/ant/v1/agent_hooks.proto:16:  rpc DeleteHookConfig(DeleteHookConfigRequest) returns (DeleteHookConfigResponse);
proto/ant/v1/ai.proto:12:  rpc Chat(ChatRequest) returns (ChatResponse);
proto/ant/v1/ai.proto:14:  rpc ChatStream(ChatRequest) returns (stream ChatStreamChunk);
proto/ant/v1/ai.proto:15:  rpc ListConversations(ListConversationsRequest) returns (ListConversationsResponse);
proto/ant/v1/ai.proto:16:  rpc GetConversation(GetConversationRequest) returns (GetConversationResponse);
proto/ant/v1/ai.proto:17:  rpc CreateConversation(CreateConversationRequest) returns (CreateConversationResponse);
proto/ant/v1/ai.proto:18:  rpc DeleteConversation(DeleteConversationRequest) returns (DeleteConversationResponse);
proto/ant/v1/ai.proto:19:  rpc UpdateConversationTitle(UpdateConversationTitleRequest) returns (UpdateConversationTitleResponse);
proto/ant/v1/ai.proto:20:  rpc ListAgents(ListAgentsRequest) returns (ListAgentsResponse);
proto/ant/v1/ai.proto:21:  rpc BatchSetAgents(BatchSetAgentsRequest) returns (BatchSetAgentsResponse);
proto/ant/v1/ai.proto:22:  rpc ResolveSession(ResolveSessionRequest) returns (ResolveSessionResponse);
proto/ant/v1/ai.proto:23:  rpc UpdateSessionStrategyKey(UpdateSessionStrategyKeyRequest) returns (UpdateSessionStrategyKeyResponse);
proto/ant/v1/ai.proto:24:  rpc SubmitSessionFeedback(SubmitSessionFeedbackRequest) returns (SubmitSessionFeedbackResponse);
proto/ant/v1/ai.proto:25:  rpc GetSessionFeedback(GetSessionFeedbackRequest) returns (GetSessionFeedbackResponse);
proto/ant/v1/ai_agent.proto:10:  rpc ListAgentDefs(ListAgentDefsRequest) returns (ListAgentDefsResponse);
proto/ant/v1/ai_agent.proto:9:  rpc BatchSetAgents(BatchSetAgentsRequest) returns (BatchSetAgentsResponse);
proto/ant/v1/ai_gate.proto:11:  rpc RunEvaluation(RunGateEvaluationRequest) returns (stream GateEvaluationUpdate);
proto/ant/v1/ai_gateway.proto:13:  rpc ListSystemModels(ListSystemModelsRequest) returns (ListSystemModelsResponse);
proto/ant/v1/ai_gateway.proto:15:  rpc GetTokenUsage(GetTokenUsageRequest) returns (GetTokenUsageResponse);
proto/ant/v1/ai_gateway.proto:18:  rpc ListProviders(ListProvidersRequest) returns (ListProvidersResponse);
proto/ant/v1/ai_gateway.proto:20:  rpc CreateProvider(CreateProviderRequest) returns (CreateProviderResponse);
proto/ant/v1/ai_gateway.proto:22:  rpc UpdateProvider(UpdateProviderRequest) returns (UpdateProviderResponse);
proto/ant/v1/ai_gateway.proto:24:  rpc DeleteProvider(DeleteProviderRequest) returns (DeleteProviderResponse);
proto/ant/v1/ai_gateway.proto:26:  rpc ListModels(ListModelsRequest) returns (ListModelsResponse);
proto/ant/v1/ai_gateway.proto:28:  rpc UpsertModel(UpsertModelRequest) returns (UpsertModelResponse);
proto/ant/v1/ai_gateway.proto:30:  rpc DeleteModel(DeleteModelRequest) returns (DeleteModelResponse);
proto/ant/v1/ai_gateway.proto:32:  rpc DiscoverGatewayModels(DiscoverGatewayModelsRequest) returns (DiscoverGatewayModelsResponse);
proto/ant/v1/ai_primary.proto:8:  rpc GetAIPrimary(GetAIPrimaryRequest) returns (AIPrimaryResponse);
proto/ant/v1/ai_primary.proto:9:  rpc SetAIPrimary(SetAIPrimaryRequest) returns (AIPrimaryResponse);
proto/ant/v1/analytics.proto:10:  rpc GetMonthlyPnL(GetMonthlyPnLRequest) returns (GetMonthlyPnLResponse);
proto/ant/v1/analytics.proto:11:  rpc GetMonthlyAnalysis(GetMonthlyAnalysisRequest) returns (GetMonthlyAnalysisResponse);
proto/ant/v1/analytics.proto:12:  rpc GetMonthlyDetail(GetMonthlyDetailRequest) returns (GetMonthlyDetailResponse);
proto/ant/v1/analytics.proto:13:  rpc GetAttributionAnalysis(GetAttributionAnalysisRequest) returns (GetAttributionAnalysisResponse);
proto/ant/v1/analytics.proto:14:  rpc GetRollingMetrics(GetRollingMetricsRequest) returns (GetRollingMetricsResponse);
proto/ant/v1/analytics.proto:15:  rpc GenerateReport(GenerateReportRequest) returns (stream GenerateReportChunk);
proto/ant/v1/analytics.proto:8:  rpc GetAccountAnalytics(GetAccountAnalyticsRequest) returns (AccountAnalyticsResponse);
proto/ant/v1/analytics.proto:9:  rpc GetRecentTrades(GetRecentTradesRequest) returns (GetRecentTradesResponse);
proto/ant/v1/asset_analysis.proto:14:  rpc AnalyzeAsset(AnalyzeAssetRequest) returns (stream AnalyzeAssetResponse);
proto/ant/v1/auth.proto:11:  rpc Login(LoginRequest) returns (LoginResponse);
proto/ant/v1/auth.proto:12:  rpc Logout(google.protobuf.Empty) returns (google.protobuf.Empty);
proto/ant/v1/auth.proto:13:  rpc RefreshToken(RefreshTokenRequest) returns (RefreshTokenResponse);
proto/ant/v1/auth.proto:14:  rpc RefreshTokenFromCookie(google.protobuf.Empty) returns (RefreshTokenResponse);
proto/ant/v1/auth.proto:15:  rpc GetMe(google.protobuf.Empty) returns (GetMeResponse);
proto/ant/v1/auth.proto:16:  rpc Register(RegisterRequest) returns (RegisterResponse);
proto/ant/v1/auth.proto:17:  rpc VerifyEmail(VerifyEmailRequest) returns (VerifyEmailResponse);
proto/ant/v1/auth.proto:18:  rpc ResendVerification(ResendVerificationRequest) returns (ResendVerificationResponse);
proto/ant/v1/auth.proto:19:  rpc ForgotPassword(ForgotPasswordRequest) returns (ForgotPasswordResponse);
proto/ant/v1/auth.proto:20:  rpc ResetPassword(ResetPasswordRequest) returns (ResetPasswordResponse);
proto/ant/v1/auth.proto:21:  rpc VerifyMTIdentity(VerifyMTIdentityRequest) returns (VerifyMTIdentityResponse);
proto/ant/v1/auto_trading.proto:15:  rpc GetGlobalSettings(GetGlobalSettingsRequest) returns (GlobalSettings);
proto/ant/v1/auto_trading.proto:16:  rpc UpdateGlobalSettings(UpdateGlobalSettingsRequest) returns (GlobalSettings);
proto/ant/v1/auto_trading.proto:17:  rpc ToggleAutoTrade(ToggleAutoTradeRequest) returns (ToggleAutoTradeResponse);
proto/ant/v1/auto_trading.proto:18:  rpc GetRiskConfig(GetRiskConfigRequest) returns (RiskConfig);
proto/ant/v1/auto_trading.proto:19:  rpc UpdateRiskConfig(UpdateRiskConfigRequest) returns (RiskConfig);
proto/ant/v1/auto_trading.proto:20:  rpc CheckRiskLimits(CheckRiskLimitsRequest) returns (CheckRiskLimitsResponse);
proto/ant/v1/auto_trading.proto:21:  rpc CalculatePositionSize(CalculatePositionSizeRequest) returns (CalculatePositionSizeResponse);
proto/ant/v1/auto_trading.proto:22:  rpc GetAutoTradingStatus(GetAutoTradingStatusRequest) returns (AutoTradingStatus);
proto/ant/v1/auto_trading.proto:23:  rpc GetTradingLogs(GetTradingLogsRequest) returns (GetTradingLogsResponse);
proto/ant/v1/auto_trading.proto:24:  rpc GetRecentTradingLogs(GetRecentTradingLogsRequest) returns (GetRecentTradingLogsResponse);
proto/ant/v1/backtest_dataset.proto:11:  rpc ListBacktestDatasets(ListBacktestDatasetsRequest) returns (ListBacktestDatasetsResponse);
proto/ant/v1/backtest_dataset.proto:12:  rpc CreateFrozenBacktestDataset(CreateFrozenBacktestDatasetRequest) returns (CreateFrozenBacktestDatasetResponse);
proto/ant/v1/backtest_dataset.proto:13:  rpc DeleteBacktestDataset(DeleteBacktestDatasetRequest) returns (google.protobuf.Empty);
proto/ant/v1/backtest_service.proto:14:  rpc RunBacktest(ExecuteBacktestRequest) returns (ExecuteBacktestResponse);
proto/ant/v1/backtest_service.proto:15:  rpc ValidateStrategy(EngineValidateRequest) returns (EngineValidateResponse);
proto/ant/v1/backtest_service.proto:16:  rpc RunStrategy(EngineRunStrategyRequest) returns (EngineRunStrategyResponse);
proto/ant/v1/backtest_trades.proto:8:  rpc ListBacktestRunTrades(ListBacktestRunTradesRequest) returns (ListBacktestRunTradesResponse);
proto/ant/v1/code_assist.proto:11:  rpc ReviseCode(ReviseCodeRequest) returns (ReviseCodeResponse);
proto/ant/v1/code_assist.proto:12:  rpc ReviseCodeStream(ReviseCodeRequest) returns (stream ReviseCodeStreamChunk);
proto/ant/v1/code_assist.proto:13:  rpc ExplainCode(ExplainCodeRequest) returns (ExplainCodeResponse);
proto/ant/v1/code_assist.proto:14:  rpc ValidateStrategyExtended(ValidateStrategyExtendedRequest) returns (ValidateStrategyExtendedResponse);
proto/ant/v1/code_assist.proto:16:  rpc TransformCode(TransformCodeRequest) returns (TransformCodeResponse);
proto/ant/v1/code_assist.proto:18:  rpc TranslateParamLabels(TranslateParamLabelsRequest) returns (TranslateParamLabelsResponse);
proto/ant/v1/credit.proto:10:  rpc ListCreditTransactions(ListCreditTransactionsRequest) returns (ListCreditTransactionsResponse);
proto/ant/v1/credit.proto:16:  rpc AddCredits(AddCreditsRequest) returns (AddCreditsResponse);
proto/ant/v1/credit.proto:18:  rpc RefundCredits(RefundCreditsRequest) returns (RefundCreditsResponse);
proto/ant/v1/credit.proto:20:  rpc ListAllCreditTransactions(ListAllCreditTransactionsRequest) returns (ListAllCreditTransactionsResponse);
proto/ant/v1/credit.proto:9:  rpc GetCreditBalance(GetCreditBalanceRequest) returns (GetCreditBalanceResponse);
proto/ant/v1/deposit.proto:13:  rpc GetDepositAddress(GetDepositAddressRequest) returns (GetDepositAddressResponse);
proto/ant/v1/deposit.proto:15:  rpc ListMyDeposits(ListMyDepositsRequest) returns (ListMyDepositsResponse);
proto/ant/v1/deposit.proto:17:  rpc ListManualReviewDeposits(ListManualReviewDepositsRequest) returns (ListManualReviewDepositsResponse);
proto/ant/v1/deposit.proto:19:  rpc ListDepositAddresses(ListDepositAddressesRequest) returns (ListDepositAddressesResponse);
proto/ant/v1/deposit.proto:21:  rpc ImportDepositAddresses(ImportDepositAddressesRequest) returns (ImportDepositAddressesResponse);
proto/ant/v1/deposit.proto:23:  rpc ListPendingSignBundles(ListPendingSignBundlesRequest) returns (ListPendingSignBundlesResponse);
proto/ant/v1/deposit.proto:25:  rpc ExportUnsignedSweepBundle(ExportUnsignedSweepBundleRequest) returns (ExportUnsignedSweepBundleResponse);
proto/ant/v1/deposit.proto:27:  rpc ExportBatchUnsignedSweepBundle(ExportBatchUnsignedSweepBundleRequest) returns (ExportBatchUnsignedSweepBundleResponse);
proto/ant/v1/deposit.proto:29:  rpc ImportSignedSweepBundle(ImportSignedSweepBundleRequest) returns (ImportSignedSweepBundleResponse);
proto/ant/v1/deposit.proto:31:  rpc GetSweepDashboard(GetSweepDashboardRequest) returns (GetSweepDashboardResponse);
proto/ant/v1/deposit.proto:33:  rpc BuildUndelegateOnlyBundle(BuildUndelegateOnlyBundleRequest) returns (BuildUndelegateOnlyBundleResponse);
proto/ant/v1/deposit.proto:35:  rpc ImportXpub(ImportXpubRequest) returns (ImportXpubResponse);
proto/ant/v1/economic_data.proto:10:  rpc ListEconomicCalendarEvents(ListEconomicCalendarEventsRequest) returns (ListEconomicCalendarEventsResponse);
proto/ant/v1/economic_data.proto:11:  rpc ListEconomicIndicators(google.protobuf.Empty) returns (ListEconomicIndicatorsResponse);
proto/ant/v1/execution_algo.proto:14:  rpc StartAlgo(StartAlgoRequest) returns (StartAlgoResponse);
proto/ant/v1/execution_algo.proto:17:  rpc GetAlgoStatus(GetAlgoStatusRequest) returns (GetAlgoStatusResponse);
proto/ant/v1/execution_algo.proto:20:  rpc CancelAlgo(CancelAlgoRequest) returns (CancelAlgoResponse);
proto/ant/v1/execution_algo.proto:23:  rpc ListAlgos(ListAlgosRequest) returns (ListAlgosResponse);
proto/ant/v1/indicator_catalog.proto:10:  rpc GetIndicatorCatalog(google.protobuf.Empty) returns (IndicatorCatalogResponse);
proto/ant/v1/job.proto:11:  rpc GetJob(GetJobRequest) returns (Job);
proto/ant/v1/job.proto:12:  rpc CancelJob(CancelJobRequest) returns (Job);
proto/ant/v1/job.proto:13:  rpc SubscribeJob(SubscribeJobRequest) returns (stream JobEvent);
proto/ant/v1/live_backtest_divergence.proto:12:  rpc GetDivergenceReport(GetDivergenceReportRequest) returns (GetDivergenceReportResponse);
proto/ant/v1/live_backtest_divergence.proto:16:  rpc WatchDivergenceReport(WatchDivergenceReportRequest) returns (stream DivergenceUpdate);
proto/ant/v1/log.proto:13:  rpc GetConnectionLogs(GetConnectionLogsRequest) returns (GetConnectionLogsResponse);
proto/ant/v1/log.proto:14:  rpc GetOrderLogHistory(GetOrderLogHistoryRequest) returns (GetOrderLogHistoryResponse);
proto/ant/v1/log.proto:15:  rpc GetOperationLogs(GetOperationLogsRequest) returns (GetOperationLogsResponse);
proto/ant/v1/log.proto:16:  rpc GetScheduleRunLogs(GetScheduleRunLogsRequest) returns (GetScheduleRunLogsResponse);
proto/ant/v1/market_regime.proto:11:  rpc DetectMarketRegime(DetectMarketRegimeRequest) returns (DetectMarketRegimeResponse);
proto/ant/v1/market_regime.proto:12:  rpc GetMarketRegime(GetMarketRegimeRequest) returns (GetMarketRegimeResponse);
proto/ant/v1/market_service.proto:10:  rpc StreamTicks(StreamTicksRequest) returns (stream TickMsg);
proto/ant/v1/market_service.proto:8:  rpc GetKlines(GetKlinesRequest) returns (GetKlinesResponse);
proto/ant/v1/market_service.proto:9:  rpc GetSymbolStats(GetSymbolStatsRequest) returns (GetSymbolStatsResponse);
proto/ant/v1/marketplace_service.proto:10:  rpc PublishStrategy(PublishStrategyRequest) returns (PublishStrategyResponse);
proto/ant/v1/marketplace_service.proto:11:  rpc Subscribe(SubscribeRequest) returns (SubscribeResponse);
proto/ant/v1/marketplace_service.proto:12:  rpc Unsubscribe(UnsubscribeRequest) returns (UnsubscribeResponse);
proto/ant/v1/marketplace_service.proto:13:  rpc PurchaseStrategy(PurchaseStrategyRequest) returns (PurchaseStrategyResponse);
proto/ant/v1/marketplace_service.proto:14:  rpc ListPublished(ListPublishedRequest) returns (ListPublishedResponse);
proto/ant/v1/marketplace_service.proto:15:  rpc ListSubscriptions(ListSubscriptionsRequest) returns (ListSubscriptionsResponse);
proto/ant/v1/marketplace_service.proto:17:  rpc RateStrategy(RateStrategyRequest) returns (RateStrategyResponse);
proto/ant/v1/marketplace_service.proto:18:  rpc ListRatings(ListRatingsRequest) returns (ListRatingsResponse);
proto/ant/v1/marketplace_service.proto:19:  rpc CommentOnStrategy(CommentOnStrategyRequest) returns (CommentOnStrategyResponse);
proto/ant/v1/marketplace_service.proto:20:  rpc ListComments(ListCommentsRequest) returns (ListCommentsResponse);
proto/ant/v1/marketplace_service.proto:22:  rpc SetStrategyPricing(SetStrategyPricingRequest) returns (SetStrategyPricingResponse);
proto/ant/v1/marketplace_service.proto:24:  rpc UnpublishStrategy(UnpublishMarketStrategyRequest) returns (UnpublishMarketStrategyResponse);
proto/ant/v1/marketplace_service.proto:26:  rpc GetPublisherStats(GetPublisherStatsRequest) returns (GetPublisherStatsResponse);
proto/ant/v1/marketplace_service.proto:30:  rpc RunMarketBacktest(RunMarketBacktestRequest) returns (stream BacktestRunUpdate);
proto/ant/v1/marketplace_service.proto:32:  rpc GetLivePerformance(GetLivePerformanceRequest) returns (GetLivePerformanceResponse);
proto/ant/v1/marketplace_service.proto:33:  rpc LinkLiveAccount(LinkLiveAccountRequest) returns (LinkLiveAccountResponse);
proto/ant/v1/marketplace_service.proto:36:  rpc GenerateAndPublish(GenerateAndPublishRequest) returns (stream GenerateAndPublishEvent);
proto/ant/v1/marketplace_service.proto:38:  rpc GenerateFromTemplate(GenerateFromTemplateRequest) returns (stream GenerateAndPublishEvent);
proto/ant/v1/marketplace_service.proto:40:  rpc ListStrategyTemplates(ListStrategyTemplatesRequest) returns (ListStrategyTemplatesResponse);
proto/ant/v1/marketplace_service.proto:42:  rpc ListAutoGenTasks(ListAutoGenTasksRequest) returns (ListAutoGenTasksResponse);
proto/ant/v1/marketplace_service.proto:44:  rpc ApproveAutoGenTask(ApproveAutoGenTaskRequest) returns (ApproveAutoGenTaskResponse);
proto/ant/v1/marketplace_service.proto:46:  rpc RejectAutoGenTask(RejectAutoGenTaskRequest) returns (RejectAutoGenTaskResponse);
proto/ant/v1/marketplace_service.proto:48:  rpc TriggerBatchGeneration(TriggerBatchGenerationRequest) returns (TriggerBatchGenerationResponse);
proto/ant/v1/marketplace_service.proto:50:  rpc ListLeaderboard(ListLeaderboardRequest) returns (ListLeaderboardResponse);
proto/ant/v1/marketplace_service.proto:52:  rpc StartTrial(StartTrialRequest) returns (StartTrialResponse);
proto/ant/v1/marketplace_service.proto:54:  rpc CompareStrategies(CompareStrategiesRequest) returns (CompareStrategiesResponse);
proto/ant/v1/marketplace_service.proto:56:  rpc GetStrategyPublicInfo(GetStrategyPublicInfoRequest) returns (GetStrategyPublicInfoResponse);
proto/ant/v1/marketplace_service.proto:58:  rpc RequestVerification(RequestVerificationRequest) returns (RequestVerificationResponse);
proto/ant/v1/marketplace_service.proto:59:  rpc AdminProcessVerification(AdminProcessVerificationRequest) returns (AdminProcessVerificationResponse);
proto/ant/v1/marketplace_service.proto:61:  rpc AdminListStrategies(AdminListStrategiesRequest) returns (AdminListStrategiesResponse);
proto/ant/v1/marketplace_service.proto:62:  rpc AdminFeatureStrategy(AdminFeatureStrategyRequest) returns (AdminFeatureStrategyResponse);
proto/ant/v1/marketplace_service.proto:64:  rpc RequestRefund(RequestRefundRequest) returns (RequestRefundResponse);
proto/ant/v1/marketplace_service.proto:65:  rpc AdminListRefundRequests(AdminListRefundRequestsRequest) returns (AdminListRefundRequestsResponse);
proto/ant/v1/marketplace_service.proto:66:  rpc AdminProcessRefund(AdminProcessRefundRequest) returns (AdminProcessRefundResponse);
proto/ant/v1/marketplace_service.proto:68:  rpc GetMarketplaceAnalytics(GetMarketplaceAnalyticsRequest) returns (MarketplaceAnalytics);
proto/ant/v1/marketplace_service.proto:69:  rpc GetTopStrategies(google.protobuf.Empty) returns (TopStrategiesResponse);
proto/ant/v1/marketplace_service.proto:70:  rpc GetTopProviders(google.protobuf.Empty) returns (TopProvidersResponse);
proto/ant/v1/marketplace_service.proto:72:  rpc ValidateCoupon(ValidateCouponRequest) returns (ValidateCouponResponse);
proto/ant/v1/marketplace_service.proto:73:  rpc CreateCoupon(CreateCouponRequest) returns (CreateCouponResponse);
proto/ant/v1/marketplace_service.proto:74:  rpc ListCoupons(ListCouponsRequest) returns (ListCouponsResponse);
proto/ant/v1/marketplace_service.proto:75:  rpc DisableCoupon(DisableCouponRequest) returns (DisableCouponResponse);
proto/ant/v1/marketplace_service.proto:77:  rpc GetProviderEarnings(google.protobuf.Empty) returns (ProviderEarnings);
proto/ant/v1/marketplace_service.proto:78:  rpc ListProviderTransactions(ListProviderTransactionsRequest) returns (ListProviderTransactionsResponse);
proto/ant/v1/marketplace_service.proto:80:  rpc DetectStrategyDecay(DetectStrategyDecayRequest) returns (DetectStrategyDecayResponse);
proto/ant/v1/marketplace_service.proto:81:  rpc ListOptimizationTasks(ListOptimizationTasksRequest) returns (ListOptimizationTasksResponse);
proto/ant/v1/marketplace_service.proto:82:  rpc GetOptimizationTask(GetOptimizationTaskRequest) returns (GetOptimizationTaskResponse);
proto/ant/v1/marketplace_service.proto:83:  rpc RejectOptimizationTask(RejectOptimizationTaskRequest) returns (RejectOptimizationTaskResponse);
proto/ant/v1/marketplace_service.proto:84:  rpc PublishOptimization(PublishOptimizationRequest) returns (PublishOptimizationResponse);
proto/ant/v1/marketplace_service.proto:85:  rpc PreviewOptimization(PreviewOptimizationRequest) returns (PreviewOptimizationResponse);
proto/ant/v1/marketplace_service.proto:87:  rpc InitiateStrategyIteration(InitiateStrategyIterationRequest) returns (InitiateStrategyIterationResponse);
proto/ant/v1/marketplace_service.proto:89:  rpc CreateBundle(CreateBundleRequest) returns (CreateBundleResponse);
proto/ant/v1/marketplace_service.proto:90:  rpc ListBundles(ListBundlesRequest) returns (ListBundlesResponse);
proto/ant/v1/marketplace_service.proto:91:  rpc GetBundle(GetBundleRequest) returns (GetBundleResponse);
proto/ant/v1/marketplace_service.proto:92:  rpc PurchaseBundle(PurchaseBundleRequest) returns (PurchaseBundleResponse);
proto/ant/v1/marketplace_service.proto:93:  rpc DeleteBundle(DeleteBundleRequest) returns (DeleteBundleResponse);
proto/ant/v1/marketplace_service.proto:95:  rpc ListFeeTiers(google.protobuf.Empty) returns (ListFeeTiersResponse);
proto/ant/v1/marketplace_service.proto:96:  rpc UpdateFeeTier(UpdateFeeTierRequest) returns (UpdateFeeTierResponse);
proto/ant/v1/marketplace_service.proto:97:  rpc GetProviderFeeTier(GetProviderFeeTierRequest) returns (GetProviderFeeTierResponse);
proto/ant/v1/mthub_service.proto:10:  rpc OrderHistory(OrderHistoryRequest) returns (OrderHistoryResponse);
proto/ant/v1/mthub_service.proto:11:  rpc SymbolParams(SymbolParamsRequest) returns (SymbolParamsResponse);
proto/ant/v1/mthub_service.proto:12:  rpc SymbolList(SymbolListRequest) returns (SymbolListResponse);
proto/ant/v1/mthub_service.proto:13:  rpc PriceHistory(PriceHistoryRequest) returns (PriceHistoryResponse);
proto/ant/v1/mthub_service.proto:14:  rpc GetAccountStatus(GetAccountStatusRequest) returns (AccountStatus);
proto/ant/v1/mthub_service.proto:15:  rpc StreamOrderEvents(StreamOrderEventsRequest) returns (stream OrderEvent);
proto/ant/v1/mthub_service.proto:16:  rpc SyncOrderHistory(SyncOrderHistoryRequest) returns (SyncOrderHistoryResponse);
proto/ant/v1/mthub_service.proto:17:  rpc SubscribeBars(SubscribeBarsRequest) returns (SubscribeBarsResponse);
proto/ant/v1/mthub_service.proto:7:  rpc PlaceOrder(PlaceOrderRequest) returns (PlaceOrderResponse);
proto/ant/v1/mthub_service.proto:8:  rpc CloseOrder(CloseOrderRequest) returns (CloseOrderResponse);
proto/ant/v1/mthub_service.proto:9:  rpc OpenedOrders(OpenedOrdersRequest) returns (OpenedOrdersResponse);
proto/ant/v1/notification_service.proto:12:  rpc ListNotifications(ListNotificationsRequest) returns (ListNotificationsResponse);
proto/ant/v1/notification_service.proto:13:  rpc MarkRead(MarkReadRequest) returns (MarkReadResponse);
proto/ant/v1/notification_service.proto:14:  rpc MarkAllRead(MarkAllReadRequest) returns (MarkAllReadResponse);
proto/ant/v1/notification_service.proto:15:  rpc StreamNotifications(StreamNotificationsRequest) returns (stream Notification);
proto/ant/v1/notification_service.proto:16:  rpc SendNotification(SendNotificationRequest) returns (SendNotificationResponse);
proto/ant/v1/notification_service.proto:18:  rpc GetNotificationPrefs(GetNotificationPrefsRequest) returns (GetNotificationPrefsResponse);
proto/ant/v1/notification_service.proto:19:  rpc SetNotificationPrefs(SetNotificationPrefsRequest) returns (SetNotificationPrefsResponse);
proto/ant/v1/paper_trading.proto:10:  rpc CreatePaperAccount(CreatePaperAccountRequest) returns (PaperAccount);
proto/ant/v1/paper_trading.proto:11:  rpc ListPaperAccounts(ListPaperAccountsRequest) returns (ListPaperAccountsResponse);
proto/ant/v1/paper_trading.proto:13:  rpc StartPaperStrategy(StartPaperStrategyRequest) returns (StartPaperStrategyResponse);
proto/ant/v1/paper_trading.proto:15:  rpc StopPaperStrategy(StopPaperStrategyRequest) returns (StopPaperStrategyResponse);
proto/ant/v1/paper_trading.proto:16:  rpc WatchPaperAccount(WatchPaperAccountRequest) returns (stream PaperAccountUpdate);
proto/ant/v1/platform_health.proto:14:  rpc GetRootCauseReport(GetRootCauseReportRequest) returns (RootCauseReport);
proto/ant/v1/platform_health.proto:18:  rpc WatchHealthAlerts(WatchHealthAlertsRequest) returns (stream HealthAlert);
proto/ant/v1/schedule_health.proto:10:  rpc GetScheduleHealth(GetScheduleHealthRequest) returns (GetScheduleHealthResponse);
proto/ant/v1/share.proto:10:  rpc UpdateShareToken(UpdateShareTokenRequest) returns (UpdateShareTokenResponse);
proto/ant/v1/share.proto:11:  rpc ListShareTokens(ListShareTokensRequest) returns (ListShareTokensResponse);
proto/ant/v1/share.proto:12:  rpc DeleteShareToken(DeleteShareTokenRequest) returns (DeleteShareTokenResponse);
proto/ant/v1/share.proto:13:  rpc ListAllShareTokens(ListAllShareTokensRequest) returns (ListAllShareTokensResponse);
proto/ant/v1/share.proto:8:  rpc CreateShareToken(CreateShareTokenRequest) returns (CreateShareTokenResponse);
proto/ant/v1/share.proto:9:  rpc GetSharedPerformance(GetSharedPerformanceRequest) returns (GetSharedPerformanceResponse);
proto/ant/v1/strategy.proto:11:  rpc ListTemplates(ListTemplatesRequest) returns (ListTemplatesResponse);
proto/ant/v1/strategy.proto:12:  rpc GetTemplate(GetTemplateRequest) returns (StrategyTemplate);
proto/ant/v1/strategy.proto:13:  rpc CreateTemplate(CreateTemplateRequest) returns (StrategyTemplate);
proto/ant/v1/strategy.proto:14:  rpc UpdateTemplate(UpdateTemplateRequest) returns (StrategyTemplate);
proto/ant/v1/strategy.proto:15:  rpc DeleteTemplate(DeleteTemplateRequest) returns (google.protobuf.Empty);
proto/ant/v1/strategy.proto:16:  rpc ListStrategyCards(ListStrategyCardsRequest) returns (ListStrategyCardsResponse);
proto/ant/v1/strategy.proto:18:  rpc CreateTemplateDraft(CreateTemplateDraftRequest) returns (StrategyTemplate);
proto/ant/v1/strategy.proto:19:  rpc UpdateTemplateDraft(UpdateTemplateDraftRequest) returns (StrategyTemplate);
proto/ant/v1/strategy.proto:20:  rpc PublishTemplateDraft(PublishTemplateDraftRequest) returns (StrategyTemplate);
proto/ant/v1/strategy.proto:21:  rpc CancelTemplateDraft(CancelTemplateDraftRequest) returns (google.protobuf.Empty);
proto/ant/v1/strategy.proto:23:  rpc ListSchedules(ListSchedulesRequest) returns (ListSchedulesResponse);
proto/ant/v1/strategy.proto:24:  rpc GetSchedule(GetScheduleRequest) returns (StrategySchedule);
proto/ant/v1/strategy.proto:25:  rpc GetSchedulePositions(GetSchedulePositionsRequest) returns (GetSchedulePositionsResponse);
proto/ant/v1/strategy.proto:26:  rpc CreateSchedule(CreateScheduleRequest) returns (StrategySchedule);
proto/ant/v1/strategy.proto:27:  rpc UpdateSchedule(UpdateScheduleRequest) returns (StrategySchedule);
proto/ant/v1/strategy.proto:28:  rpc DeleteSchedule(DeleteScheduleRequest) returns (google.protobuf.Empty);
proto/ant/v1/strategy.proto:29:  rpc ToggleSchedule(ToggleScheduleRequest) returns (StrategySchedule);
proto/ant/v1/strategy.proto:30:  rpc WatchSchedules(WatchSchedulesRequest) returns (stream WatchSchedulesEvent);
proto/ant/v1/strategy.proto:31:  rpc RunBacktest(RunBacktestRequest) returns (RunBacktestResponse);
proto/ant/v1/strategy.proto:33:  rpc ListSignals(ListSignalsRequest) returns (ListSignalsResponse);
proto/ant/v1/strategy.proto:34:  rpc ExecuteSignal(ExecuteSignalRequest) returns (ExecuteSignalResponse);
proto/ant/v1/strategy.proto:35:  rpc ConfirmSignal(ConfirmSignalRequest) returns (google.protobuf.Empty);
proto/ant/v1/strategy.proto:36:  rpc CancelSignal(CancelSignalRequest) returns (google.protobuf.Empty);
proto/ant/v1/strategy_asset.proto:10:  rpc ListStrategyAssets(ListStrategyAssetsRequest) returns (ListStrategyAssetsResponse);
proto/ant/v1/strategy_asset.proto:11:  rpc GetStrategyAsset(GetStrategyAssetRequest) returns (StrategyAsset);
proto/ant/v1/strategy_asset.proto:12:  rpc SubmitAssetReview(SubmitAssetReviewRequest) returns (StrategyAsset);
proto/ant/v1/strategy_asset.proto:13:  rpc ReviewStrategyAsset(ReviewStrategyAssetRequest) returns (StrategyAsset);
proto/ant/v1/strategy_asset.proto:14:  rpc CloneStrategyAsset(CloneStrategyAssetRequest) returns (CloneStrategyAssetResponse);
proto/ant/v1/strategy_asset.proto:15:  rpc CheckAssetUpdate(CheckAssetUpdateRequest) returns (StrategyAssetClone);
proto/ant/v1/strategy_asset.proto:16:  rpc SyncStrategyAsset(SyncStrategyAssetRequest) returns (StrategyAssetClone);
proto/ant/v1/strategy_asset.proto:17:  rpc ListAssetClones(ListAssetClonesRequest) returns (ListAssetClonesResponse);
proto/ant/v1/strategy_execution.proto:13:  rpc AnalyzePlan(AnalyzePlanRequest) returns (stream AnalyzePlanChunk);
proto/ant/v1/strategy_execution.proto:14:  rpc Diagnose(DiagnoseRequest) returns (stream AnalyzePlanChunk);
proto/ant/v1/strategy_execution.proto:15:  rpc ExecutePlan(ExecutePlanRequest) returns (stream ExecutePlanChunk);
proto/ant/v1/strategy_execution.proto:16:  rpc Conversate(ConversateRequest) returns (stream ConversateChunk);
proto/ant/v1/strategy_experiment.proto:11:  rpc SubmitStrategyExperiment(SubmitStrategyExperimentRequest) returns (SubmitStrategyExperimentResponse);
proto/ant/v1/strategy_experiment.proto:12:  rpc GetStrategyExperiment(GetStrategyExperimentRequest) returns (StrategyExperiment);
proto/ant/v1/strategy_experiment.proto:13:  rpc ListStrategyExperiments(ListStrategyExperimentsRequest) returns (ListStrategyExperimentsResponse);
proto/ant/v1/strategy_experiment.proto:14:  rpc CancelStrategyExperiment(CancelStrategyExperimentRequest) returns (StrategyExperiment);
proto/ant/v1/strategy_experiment.proto:15:  rpc ListExperimentCandidates(ListExperimentCandidatesRequest) returns (ListExperimentCandidatesResponse);
proto/ant/v1/strategy_experiment.proto:16:  rpc GetExperimentCandidate(GetExperimentCandidateRequest) returns (StrategyExperimentCandidate);
proto/ant/v1/strategy_experiment.proto:17:  rpc PromoteCandidateToDraft(PromoteCandidateToDraftRequest) returns (PromoteCandidateToDraftResponse);
proto/ant/v1/strategy_experiment.proto:20:  rpc WatchExperiment(WatchExperimentRequest) returns (stream WatchExperimentEvent);
proto/ant/v1/strategy_runtime.proto:17:  rpc Execute(ExecuteStrategyRequest) returns (ExecuteStrategyResponse);
proto/ant/v1/strategy_runtime.proto:18:  rpc Validate(ValidateStrategyRequest) returns (ValidateStrategyResponse);
proto/ant/v1/strategy_runtime.proto:19:  rpc Backtest(BacktestStrategyRequest) returns (BacktestStrategyResponse);
proto/ant/v1/strategy_runtime.proto:20:  rpc StartBacktestRun(StartBacktestRunRequest) returns (StartBacktestRunResponse);
proto/ant/v1/strategy_runtime.proto:21:  rpc GetBacktestRun(GetBacktestRunRequest) returns (GetBacktestRunResponse);
proto/ant/v1/strategy_runtime.proto:22:  rpc ListBacktestRuns(ListBacktestRunsRequest) returns (ListBacktestRunsResponse);
proto/ant/v1/strategy_runtime.proto:23:  rpc WatchBacktestRun(WatchBacktestRunRequest) returns (stream BacktestRunUpdate);
proto/ant/v1/strategy_runtime.proto:24:  rpc CancelBacktestRun(CancelBacktestRunRequest) returns (CancelBacktestRunResponse);
proto/ant/v1/strategy_runtime.proto:25:  rpc DeleteBacktestRun(DeleteBacktestRunRequest) returns (DeleteBacktestRunResponse);
proto/ant/v1/strategy_runtime.proto:26:  rpc DeleteBacktestRuns(DeleteBacktestRunsRequest) returns (DeleteBacktestRunsResponse);
proto/ant/v1/strategy_runtime.proto:27:  rpc UpdateBacktestRun(UpdateBacktestRunRequest) returns (UpdateBacktestRunResponse);
proto/ant/v1/strategy_runtime.proto:28:  rpc GetTemplates(google.protobuf.Empty) returns (GetStrategyTemplatesResponse);
proto/ant/v1/strategy_runtime.proto:31:  rpc ExecuteLive(ExecuteLiveRequest) returns (ExecuteLiveResponse);
proto/ant/v1/strategy_runtime.proto:33:  rpc AnalyzeImportCode(AnalyzeImportCodeRequest) returns (AnalyzeImportCodeResponse);
proto/ant/v1/strategy_runtime.proto:35:  rpc GetImportedStrategy(GetImportedStrategyRequest) returns (GetImportedStrategyResponse);
proto/ant/v1/strategy_runtime.proto:37:  rpc ListStrategyRuns(ListStrategyRunsRequest) returns (ListStrategyRunsResponse);
proto/ant/v1/strategy_runtime.proto:39:  rpc GetStrategyRun(GetStrategyRunRequest) returns (GetStrategyRunResponse);
proto/ant/v1/strategy_runtime.proto:41:  rpc ListActiveStrategies(ListActiveStrategiesRequest) returns (ListActiveStrategiesResponse);
proto/ant/v1/strategy_runtime.proto:43:  rpc GetActiveStrategy(GetActiveStrategyRequest) returns (GetActiveStrategyResponse);
proto/ant/v1/strategy_runtime.proto:45:  rpc StopStrategy(StopStrategyRequest) returns (StopStrategyResponse);
proto/ant/v1/strategy_runtime.proto:47:  rpc WatchStrategySignals(WatchStrategySignalsRequest) returns (stream StrategySignalEvent);
proto/ant/v1/strategy_runtime.proto:49:  rpc StartStrategy(StartStrategyRequest) returns (StartStrategyResponse);
proto/ant/v1/strategy_runtime.proto:52:  rpc WatchActiveStrategies(WatchActiveStrategiesRequest) returns (stream WatchActiveStrategiesEvent);
proto/ant/v1/strategy_runtime.proto:54:  rpc ListStrategyVersions(ListStrategyVersionsRequest) returns (ListStrategyVersionsResponse);
proto/ant/v1/strategy_runtime.proto:56:  rpc GetStrategyVersion(GetStrategyVersionRequest) returns (GetStrategyVersionResponse);
proto/ant/v1/strategy_runtime.proto:58:  rpc RollbackStrategyVersion(RollbackStrategyVersionRequest) returns (RollbackStrategyVersionResponse);
proto/ant/v1/strategy_runtime.proto:60:  rpc DiffStrategyVersions(DiffStrategyVersionsRequest) returns (DiffStrategyVersionsResponse);
proto/ant/v1/strategy_runtime.proto:62:  rpc UpdateStrategyCode(UpdateStrategyCodeRequest) returns (UpdateStrategyCodeResponse);
proto/ant/v1/strategy_runtime.proto:65:  rpc CheckCode(CheckCodeRequest) returns (CheckCodeResponse);
proto/ant/v1/stream.proto:75:  rpc SubscribeEvents(SubscribeEventsRequest) returns (stream StreamEvent);
proto/ant/v1/stream.proto:76:  rpc SubscribeHistory(SubscribeHistoryRequest) returns (stream StreamEvent);
proto/ant/v1/stream.proto:77:  rpc SubscribeOrderUpdates(SubscribeOrderUpdatesRequest) returns (stream OrderUpdateEvent);
proto/ant/v1/stream.proto:78:  rpc SubscribeProfitUpdates(SubscribeProfitUpdatesRequest) returns (stream ProfitUpdateEvent);
proto/ant/v1/stream.proto:79:  rpc SubscribeUserSummary(google.protobuf.Empty) returns (stream UserSummaryEvent);
proto/ant/v1/stream.proto:80:  rpc SubscribeIndicators(SubscribeIndicatorsRequest) returns (stream IndicatorUpdateEvent);
proto/ant/v1/subscription.proto:11:  rpc ListPlans(ListPlansRequest) returns (ListPlansResponse);
proto/ant/v1/subscription.proto:13:  rpc GetMySubscription(GetMySubscriptionRequest) returns (GetMySubscriptionResponse);
proto/ant/v1/subscription.proto:15:  rpc Subscribe(SubscribePlanRequest) returns (SubscribePlanResponse);
proto/ant/v1/subscription.proto:17:  rpc CancelSubscription(CancelSubscriptionRequest) returns (CancelSubscriptionResponse);
proto/ant/v1/subscription.proto:19:  rpc ChangePlan(ChangePlanRequest) returns (ChangePlanResponse);
proto/ant/v1/subscription.proto:21:  rpc GetUsageSummary(GetUsageSummaryRequest) returns (GetUsageSummaryResponse);
proto/ant/v1/subscription.proto:23:  rpc ListBoundAccounts(ListBoundAccountsRequest) returns (ListBoundAccountsResponse);
proto/ant/v1/subscription.proto:25:  rpc UnbindAccount(UnbindAccountRequest) returns (UnbindAccountResponse);
proto/ant/v1/system_ai.proto:11:  rpc ListSystemAIConfigs(ListSystemAIConfigsRequest) returns (ListSystemAIConfigsResponse);
proto/ant/v1/system_ai.proto:12:  rpc GetSystemAIConfig(GetSystemAIConfigRequest) returns (GetSystemAIConfigResponse);
proto/ant/v1/system_ai.proto:13:  rpc UpdateSystemAIConfig(UpdateSystemAIConfigRequest) returns (UpdateSystemAIConfigResponse);
proto/ant/v1/system_ai.proto:14:  rpc UpdateSystemAISecret(UpdateSystemAISecretRequest) returns (UpdateSystemAISecretResponse);
proto/ant/v1/system_ai.proto:15:  rpc DiscoverSystemAIModels(DiscoverSystemAIModelsRequest) returns (DiscoverSystemAIModelsResponse);
proto/ant/v1/system_ai.proto:16:  rpc ValidateSystemAIConnection(ValidateSystemAIConnectionRequest) returns (ValidateSystemAIConnectionResponse);
proto/ant/v1/walk_forward.proto:11:  rpc GetWalkForwardReport(GetWalkForwardReportRequest) returns (WalkForwardReport);
proto/ant/v1/walk_forward.proto:15:  rpc WatchWalkForwardReport(WatchWalkForwardReportRequest) returns (stream WalkForwardReport);
proto/ant/v1/wallet.proto:10:  rpc ListTransactions(ListWalletTransactionsRequest) returns (ListWalletTransactionsResponse);
proto/ant/v1/wallet.proto:11:  rpc AdjustBalance(AdjustBalanceRequest) returns (AdjustBalanceResponse);  // admin
proto/ant/v1/wallet.proto:9:  rpc GetWallet(GetWalletRequest) returns (GetWalletResponse);
proto/ant/v1/webauthn.proto:15:  rpc BeginRegistration(BeginRegistrationRequest) returns (BeginRegistrationResponse);
proto/ant/v1/webauthn.proto:17:  rpc FinishRegistration(FinishRegistrationRequest) returns (FinishRegistrationResponse);
proto/ant/v1/webauthn.proto:19:  rpc ListCredentials(ListCredentialsRequest) returns (ListCredentialsResponse);
proto/ant/v1/webauthn.proto:21:  rpc RemoveCredential(RemoveCredentialRequest) returns (RemoveCredentialResponse);
proto/ant/v1/webauthn.proto:23:  rpc BeginWithdrawal(BeginWithdrawalRequest) returns (BeginWithdrawalResponse);
proto/ant/v1/webauthn.proto:25:  rpc FinishWithdrawal(FinishWithdrawalRequest) returns (FinishWithdrawalResponse);
proto/ant/v1/webauthn.proto:27:  rpc ListWithdrawals(ListWithdrawalsRequest) returns (ListWithdrawalsResponse);
proto/ant/v1/webauthn.proto:29:  rpc CancelWithdrawal(CancelWithdrawalRequest) returns (CancelWithdrawalResponse);
proto/ant/v1/webauthn.proto:31:  rpc AddWhitelistAddress(AddWhitelistAddressRequest) returns (AddWhitelistAddressResponse);
proto/ant/v1/webauthn.proto:33:  rpc ListWhitelistAddresses(ListWhitelistAddressesRequest) returns (ListWhitelistAddressesResponse);
proto/ant/v1/webauthn.proto:35:  rpc RemoveWhitelistAddress(RemoveWhitelistAddressRequest) returns (RemoveWhitelistAddressResponse);
proto/ant/v1/webauthn.proto:37:  rpc ExportCredentialList(ExportCredentialListRequest) returns (ExportCredentialListResponse);
proto/ant/v1/webauthn.proto:39:  rpc ExportWhitelist(ExportWhitelistRequest) returns (ExportWhitelistResponse);
```

## MT 网关原始 RPC（mtapi 层，经 executor/adapter 暴露）

```
reference/grpc/mt4.proto:105:  rpc ServerTimezone (ServerTimezoneRequest) returns (ServerTimezoneReply);
reference/grpc/mt4.proto:111:  rpc SymbolParamsMany (SymbolParamsManyRequest) returns (SymbolParamsManyReply);
reference/grpc/mt4.proto:118:  rpc OpenedOrder (OpenedOrderRequest) returns (OpenedOrderReply);
reference/grpc/mt4.proto:126:  rpc OrderHistory (OrderHistoryRequest) returns (OrderHistoryReply);
reference/grpc/mt4.proto:136:  rpc QuoteHistory (QuoteHistoryRequest) returns (QuoteHistoryReply);
reference/grpc/mt4.proto:145:  rpc QuoteHistoryMany (QuoteHistoryManyRequest) returns (QuoteHistoryManyReply);
reference/grpc/mt4.proto:151:  rpc ClosedOrders (ClosedOrdersRequest) returns (ClosedOrdersReply);
reference/grpc/mt4.proto:161:  rpc RequestQuoteHistory (RequestQuoteHistoryRequest) returns (RequestQuoteHistoryReply);
reference/grpc/mt4.proto:168:  rpc SetPlacedType (SetPlacedTypeRequest) returns (SetPlacedTypeReply);
reference/grpc/mt4.proto:174:  rpc IsInvestor (IsInvestorRequest) returns (IsInvestorReply);
reference/grpc/mt4.proto:17:  rpc Connect (ConnectRequest) returns (ConnectReply);
reference/grpc/mt4.proto:180:rpc TickValueWithSize (TickValueWithSizeRequest) returns (TickValueWithSizeReply);
reference/grpc/mt4.proto:185:  rpc Ping (PingRequest) returns (PingReply);
reference/grpc/mt4.proto:192:  rpc PingHost (PingHostRequest) returns (PingHostReply);
reference/grpc/mt4.proto:194:  rpc PingHostMany (PingHostManyRequest) returns (PingHostManyReply);
reference/grpc/mt4.proto:200:  rpc GetLogs (GetLogsRequest) returns (GetLogsReply);
reference/grpc/mt4.proto:207:  rpc GetLogsByUser (GetLogsByUserRequest) returns (GetLogsByUserReply);
reference/grpc/mt4.proto:209:  rpc MemorySnapshot (MemorySnapshotRequest) returns (MemorySnapshotReply);
reference/grpc/mt4.proto:215:  rpc Search (SearchRequest) returns (SearchReply);
reference/grpc/mt4.proto:217:  rpc GetClients (GetClientsRequest) returns (GetClientsReply);
reference/grpc/mt4.proto:219:  rpc MemoryUsage (MemoryUsageRequest) returns (MemoryUsageReply);
reference/grpc/mt4.proto:230:  rpc Subscribe (SubscribeRequest) returns (SubscribeReply);
reference/grpc/mt4.proto:237:  rpc SubscribeMany (SubscribeManyRequest) returns (SubscribeManyReply);
reference/grpc/mt4.proto:244:  rpc UnSubscribe (UnSubscribeRequest) returns (UnSubscribeReply);
reference/grpc/mt4.proto:250:  rpc UnSubscribeMany (UnSubscribeManyRequest) returns (UnSubscribeManyReply);
reference/grpc/mt4.proto:256:  rpc SubscribeOrderProfit (SubscribeOrderProfitRequest) returns (SubscribeOrderProfitReply);
reference/grpc/mt4.proto:25:  rpc ConnectEx (ConnectExRequest) returns (ConnectExReply);
reference/grpc/mt4.proto:264:  rpc SubscribeTickValue (SubscribeTickValueRequest) returns (SubscribeTickValueReply);
reference/grpc/mt4.proto:270:  rpc SubscribeOrderUpdate (SubscribeOrderUpdateRequest) returns (SubscribeOrderUpdateReply);
reference/grpc/mt4.proto:276:  rpc SubscribeQuoteHistory (SubscribeQuoteHistoryRequest) returns (SubscribeQuoteHistoryReply);
reference/grpc/mt4.proto:294:  rpc OrderSend (OrderSendRequest) returns (OrderSendReply);
reference/grpc/mt4.proto:304:  rpc OrderModify (OrderModifyRequest) returns (OrderModifyReply);
reference/grpc/mt4.proto:312:  rpc OrderCloseBy (OrderCloseByRequest) returns (OrderCloseByReply);
reference/grpc/mt4.proto:319:  rpc OrderDelete (OrderDeleteRequest) returns (OrderDeleteReply);
reference/grpc/mt4.proto:329:  rpc OrderClose (OrderCloseRequest) returns (OrderCloseReply);
reference/grpc/mt4.proto:338:  rpc OnOrderUpdate (OnOrderUpdateRequest) returns (stream OnOrderUpdateReply);
reference/grpc/mt4.proto:344:  rpc OnQuote (OnQuoteRequest) returns (stream OnQuoteReply);
reference/grpc/mt4.proto:350:  rpc OnTickValue (OnTickValueRequest) returns (stream OnTickValueReply);
reference/grpc/mt4.proto:356:  rpc OnOrderProfit (OnOrderProfitRequest) returns (stream OnOrderProfitReply);
reference/grpc/mt4.proto:362://  rpc OnQuoteHistory (OnQuoteHistoryRequest) returns (OnQuoteHistoryReply);
reference/grpc/mt4.proto:368://  rpc OnDisconnect (OnDisconnectRequest) returns (OnDisconnectReply);
reference/grpc/mt4.proto:39:  rpc ConnectProxy (ConnectProxyRequest) returns (ConnectProxyReply);
reference/grpc/mt4.proto:45:  rpc CheckConnect (CheckConnectRequest) returns (CheckConnectReply);
reference/grpc/mt4.proto:51:  rpc Disconnect (DisconnectRequest) returns (DisconnectReply);
reference/grpc/mt4.proto:60:  rpc AccountSummary (AccountSummaryRequest) returns (AccountSummaryReply);
reference/grpc/mt4.proto:66:  rpc Groups (GroupsRequest) returns (GroupsReply);
reference/grpc/mt4.proto:73:  rpc Quote (QuoteRequest) returns (QuoteReply);
reference/grpc/mt4.proto:80:  rpc GetQuoteMany (GetQuoteManyRequest) returns (GetQuoteManyReply);
reference/grpc/mt4.proto:86:  rpc OpenedOrders (OpenedOrdersRequest) returns (OpenedOrdersReply);
reference/grpc/mt4.proto:92:  rpc Symbols (SymbolsRequest) returns (SymbolsReply);
reference/grpc/mt4.proto:99:  rpc SymbolParams (SymbolParamsRequest) returns (SymbolParamsReply);
reference/grpc/mt5.proto:104:  rpc PendingOrderHistory (PendingOrderHistoryRequest) returns (PendingOrderHistoryReply);
reference/grpc/mt5.proto:118:  rpc OrderHistoryPagination (OrderHistoryPaginationRequest) returns (OrderHistoryPaginationReply);
reference/grpc/mt5.proto:124:  rpc Symbols (SymbolsRequest) returns (SymbolsReply);
reference/grpc/mt5.proto:130:  rpc SymbolList (SymbolListRequest) returns (SymbolListReply);
reference/grpc/mt5.proto:138:  rpc GetQuote (GetQuoteRequest) returns (GetQuoteReply);
reference/grpc/mt5.proto:145:  rpc GetQuoteMany (GetQuoteManyRequest) returns (GetQuoteManyReply);
reference/grpc/mt5.proto:151:  rpc MarketWatchMany (MarketWatchManyRequest) returns (MarketWatchManyReply);
reference/grpc/mt5.proto:158:  rpc SymbolParams (SymbolParamsRequest) returns (SymbolParamsReply);
reference/grpc/mt5.proto:15:  rpc Connect (ConnectRequest) returns (ConnectReply);
reference/grpc/mt5.proto:165:  rpc SymbolParamsMany (SymbolParamsManyRequest) returns (SymbolParamsManyReply);
reference/grpc/mt5.proto:172:rpc SymbolSessionsEx (SymbolSessionsExRequest) returns (SymbolSessionsExReply);
reference/grpc/mt5.proto:178:rpc SymbolSessionsExMany (SymbolSessionsExManyRequest) returns (SymbolSessionsExManyReply);
reference/grpc/mt5.proto:184:  rpc ServerTimezone (ServerTimezoneRequest) returns (ServerTimezoneReply);
reference/grpc/mt5.proto:191:  rpc IsTradeSession (IsTradeSessionRequest) returns (IsTradeSessionReply);
reference/grpc/mt5.proto:197:  rpc IsTradeSessionMany (IsTradeSessionManyRequest) returns (IsTradeSessionManyReply);
reference/grpc/mt5.proto:204:  rpc IsQuoteSession (IsQuoteSessionRequest) returns (IsQuoteSessionReply);
reference/grpc/mt5.proto:210:  rpc IsQuoteSessionMany (IsQuoteSessionManyRequest) returns (IsQuoteSessionManyReply);
reference/grpc/mt5.proto:216:  rpc GetTickValueMany (GetTickValueManyRequest) returns (GetTickValueManyReply);
reference/grpc/mt5.proto:222:rpc TickValueWithSize (TickValueWithSizeRequest) returns (TickValueWithSizeReply);
reference/grpc/mt5.proto:229://  rpc ClusterDetails (ClusterDetailsRequest) returns (ClusterDetailsReply);
reference/grpc/mt5.proto:22:  rpc ConnectEx (ConnectExRequest) returns (ConnectExReply);
reference/grpc/mt5.proto:231:  rpc ChangePassword (ChangePasswordRequest) returns (ChangePasswordReply);
reference/grpc/mt5.proto:237:  rpc Mails (MailsRequest) returns (MailsReply);
reference/grpc/mt5.proto:247:  rpc RequiredMargin (RequiredMarginRequest) returns (RequiredMarginReply);
reference/grpc/mt5.proto:261:  rpc PriceHistoryMonth (PriceHistoryMonthRequest) returns (PriceHistoryMonthReply);
reference/grpc/mt5.proto:271:rpc PriceHistoryMonthMany (PriceHistoryMonthManyRequest) returns (PriceHistoryMonthManyReply);
reference/grpc/mt5.proto:281:  rpc PriceHistoryToday (PriceHistoryTodayRequest) returns (PriceHistoryTodayReply);
reference/grpc/mt5.proto:288:rpc PriceHistoryTodayMany (PriceHistoryTodayManyRequest) returns (PriceHistoryTodayManyReply);
reference/grpc/mt5.proto:299:  rpc PriceHistory (PriceHistoryRequest) returns (PriceHistoryReply);
reference/grpc/mt5.proto:308:  rpc PriceHistoryMany (PriceHistoryManyRequest) returns (PriceHistoryManyReply);
reference/grpc/mt5.proto:318:  rpc PriceHistoryHighLow (PriceHistoryHighLowRequest) returns (PriceHistoryHighLowReply);
reference/grpc/mt5.proto:328:  rpc PriceHistoryEx (PriceHistoryExRequest) returns (PriceHistoryExReply);
reference/grpc/mt5.proto:337:  rpc PriceHistoryExMany (PriceHistoryExManyRequest) returns (PriceHistoryExManyReply);
reference/grpc/mt5.proto:342:  rpc Ping (PingRequest) returns (PingReply);
reference/grpc/mt5.proto:343:  rpc Health (HealthRequest) returns (HealthReply);
reference/grpc/mt5.proto:350:  rpc PingHost (PingHostRequest) returns (PingHostReply);
reference/grpc/mt5.proto:357:  rpc PingHostMany (PingHostManyRequest) returns (PingHostManyReply);
reference/grpc/mt5.proto:359:  rpc MemorySnapshot (MemorySnapshotRequest) returns (MemorySnapshotReply);
reference/grpc/mt5.proto:365:  rpc Search (SearchRequest) returns (SearchReply);
reference/grpc/mt5.proto:367:  rpc GetClients (GetClientsRequest) returns (GetClientsReply);
reference/grpc/mt5.proto:374:  rpc GetDemo (GetDemoRequest) returns (GetDemoReply);
reference/grpc/mt5.proto:376:  rpc Version (VersionRequest) returns (VersionReply);
reference/grpc/mt5.proto:37:  rpc ConnectProxy (ConnectProxyRequest) returns (ConnectProxyReply);
reference/grpc/mt5.proto:387:  rpc Subscribe (SubscribeRequest) returns (SubscribeReply);
reference/grpc/mt5.proto:394:  rpc SubscribeMany (SubscribeManyRequest) returns (SubscribeManyReply);
reference/grpc/mt5.proto:401:  rpc UnSubscribe (UnSubscribeRequest) returns (UnSubscribeReply);
reference/grpc/mt5.proto:407:  rpc UnSubscribeMany (UnSubscribeManyRequest) returns (UnSubscribeManyReply);
reference/grpc/mt5.proto:413:  rpc SubscribeOrderProfit (SubscribeOrderProfitRequest) returns (SubscribeOrderProfitReply);
reference/grpc/mt5.proto:421:  rpc SubscribeTickValue (SubscribeTickValueRequest) returns (SubscribeTickValueReply);
reference/grpc/mt5.proto:427:  rpc SubscribeOrderUpdate (SubscribeOrderUpdateRequest) returns (SubscribeOrderUpdateReply);
reference/grpc/mt5.proto:433:  rpc SubscribeMarketWatch (SubscribeMarketWatchRequest) returns (SubscribeMarketWatchReply);
reference/grpc/mt5.proto:43:  rpc CheckConnect (CheckConnectRequest) returns (CheckConnectReply);
reference/grpc/mt5.proto:440:  rpc SubscribeOpenedOrdersTickets (SubscribeOpenedOrdersTicketsRequest) returns (SubscribeOpenedOrdersTicketsReply);
reference/grpc/mt5.proto:453:  rpc TickHistoryRequest (TickHistoryRequestRequest) returns (TickHistoryRequestReply);
reference/grpc/mt5.proto:460:  rpc TickHistoryStop (TickHistoryStopRequest) returns (TickHistoryStopReply);
reference/grpc/mt5.proto:462:  rpc OnTickHistory (OnTickHistoryRequest) returns (stream OnTickHistoryReply);
reference/grpc/mt5.proto:482:  rpc OrderSend (OrderSendRequest) returns (OrderSendReply);
reference/grpc/mt5.proto:493:  rpc OrderModify (OrderModifyRequest) returns (OrderModifyReply);
reference/grpc/mt5.proto:49:  rpc Disconnect (DisconnectRequest) returns (DisconnectReply);
reference/grpc/mt5.proto:503:  rpc OrderClose (OrderCloseRequest) returns (OrderCloseReply);
reference/grpc/mt5.proto:512:  rpc Events (EventsRequest) returns (stream EventsReply);
reference/grpc/mt5.proto:518:  rpc OnOrderUpdate (OnOrderUpdateRequest) returns (stream OnOrderUpdateReply);
reference/grpc/mt5.proto:524:  rpc OnQuote (OnQuoteRequest) returns (stream OnQuoteReply);
reference/grpc/mt5.proto:530:  rpc OnTickValue (OnTickValueRequest) returns (stream OnTickValueReply);
reference/grpc/mt5.proto:536:  rpc OnOrderProfit (OnOrderProfitRequest) returns (stream OnOrderProfitReply);
reference/grpc/mt5.proto:542:  rpc OnMarketWatch (OnMarketWatchRequest) returns (stream OnMarketWatchReply);
reference/grpc/mt5.proto:548:  rpc OnTickHistory (OnTickHistoryRequest) returns (stream OnTickHistoryReply);
reference/grpc/mt5.proto:554:  rpc OnMail (OnMailRequest) returns (stream OnMailReply);
reference/grpc/mt5.proto:561:  rpc OnOpenedOrdersTickets (OnOpenedOrdersTicketsRequest) returns (stream OnOpenedOrdersTicketsReply);
reference/grpc/mt5.proto:58:  rpc Account (AccountRequest) returns (AccountReply);
reference/grpc/mt5.proto:64:  rpc AccountSummary (AccountSummaryRequest) returns (AccountSummaryReply);
reference/grpc/mt5.proto:72:  rpc OpenedOrders (OpenedOrdersRequest) returns (OpenedOrdersReply);
reference/grpc/mt5.proto:79:  rpc OpenedOrder (OpenedOrderRequest) returns (OpenedOrderReply);
reference/grpc/mt5.proto:85:  rpc OpenedOrdersTickets (OpenedOrdersTicketsRequest) returns (OpenedOrdersTicketsReply);
reference/grpc/mt5.proto:95:  rpc OrderHistory (OrderHistoryRequest) returns (OrderHistoryReply);
```

## Go 服务方法（已实现的后端能力）

```
backend/internal/connect/strategy/account_provider.go:46:func (p *MTAccountStateProvider) SetPositionCache(pc *PositionCache) { p.posCache = pc }
backend/internal/connect/strategy/account_provider.go:49:func (p *MTAccountStateProvider) GetAccountState(_ context.Context, accountID string) (*risk.AccountState, error) {
backend/internal/connect/strategy/account_provider.go:87:func (p *MTAccountStateProvider) GetPeakEquity(accountID string) decimal.Decimal {
backend/internal/connect/strategy/account_provider.go:94:func (p *MTAccountStateProvider) ResetPeakEquity(accountID string) {
backend/internal/connect/strategy/ai_proposer_adapter.go:22:func (a *systemAIAdapter) ChatCompletion(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
backend/internal/connect/strategy/backtest_execution.go:29:func (s *StrategyExecutionServer) ExecuteBacktestDirect(
backend/internal/connect/strategy/backtest_trades_handler.go:53:func (s *BacktestTradesServer) ListBacktestRunTrades(ctx context.Context, req *connect.Request[antv1.ListBacktestRunTradesRequest]) (*connect.Response[antv1.ListBacktestRunTradesResponse], error) {
backend/internal/connect/strategy/backtest_worker.go:203:func (s *StrategyExecutionServer) StartBacktestWorker(ctx context.Context) {
backend/internal/connect/strategy/code_check_handler.go:16:func (s *StrategyExecutionServer) CheckCode(ctx context.Context, req *connect.Request[antv1.CheckCodeRequest]) (*connect.Response[antv1.CheckCodeResponse], error) {
backend/internal/connect/strategy/data_source.go:55:func (s *LiveSource) Name() string { return modeLive }
backend/internal/connect/strategy/data_source.go:57:func (s *LiveSource) Fetch(ctx context.Context, symbol, timeframe string, from, to *time.Time) ([]*antv1.ExecuteKlineBar, error) {
backend/internal/connect/strategy/data_source.go:68:func (s *LiveSource) Subscribe(accountID string) (<-chan *mthub.BarUpdate, func()) {
backend/internal/connect/strategy/divergence_handler.go:38:func (s *DivergenceServer) SetPgListen(l *pglisten.Listener) { s.pgListen = l }
backend/internal/connect/strategy/divergence_handler.go:42:func (s *DivergenceServer) GetDivergenceReport(
backend/internal/connect/strategy/divergence_handler.go:60:func (s *DivergenceServer) WatchDivergenceReport(
backend/internal/connect/strategy/live_runner.go:138:func (s *StrategyExecutionServer) RunLiveStrategy(ctx context.Context, cfg LiveStrategyConfig) error {
backend/internal/connect/strategy/position_cache.go:104:func (c *PositionCache) GetFreshSnapshot(accountID string, now time.Time) (*mthub.PositionSnapshot, bool) {
backend/internal/connect/strategy/position_cache.go:36:func (c *PositionCache) Subscribe(ctx context.Context, hub *mthub.MtHubService, accountID string) {
backend/internal/connect/strategy/position_cache.go:84:func (c *PositionCache) PutSnapshot(snap *mthub.PositionSnapshot, receivedAt time.Time) {
backend/internal/connect/strategy/position_cache.go:89:func (c *PositionCache) Unsubscribe(accountID string) {
backend/internal/connect/strategy/position_cache.go:97:func (c *PositionCache) GetSnapshot(accountID string) *mthub.PositionSnapshot {
backend/internal/connect/strategy/schedule_engine.go:139:func (e *ScheduleEngine) Notify() {
backend/internal/connect/strategy/schedule_engine.go:423:func (e *ScheduleEngine) StopSchedule(id uuid.UUID) {
backend/internal/connect/strategy/schedule_engine.go:435:func (e *ScheduleEngine) Stop() {
backend/internal/connect/strategy/schedule_engine.go:99:func (e *ScheduleEngine) Start(ctx context.Context) error {
backend/internal/connect/strategy/schedule_event.go:17:func (e *ScheduleEngine) StartSchedule(ctx context.Context, id uuid.UUID) error {
backend/internal/connect/strategy/session_diag.go:102:func (d *sessionDiag) RecordIndicators(values map[string]decimal.Decimal, ordersTotal int) {
backend/internal/connect/strategy/session_diag.go:145:func (d *sessionDiag) SnapshotDiag() DiagSnapshot {
backend/internal/connect/strategy/session_diag.go:76:func (d *sessionDiag) RecordEval(kind int) {
backend/internal/connect/strategy/session_diag.go:90:func (d *sessionDiag) RecordWindow(n int) {
backend/internal/connect/strategy/session_registry.go:131:func (r *SessionRegistry) Watch() (<-chan struct{}, func()) {
backend/internal/connect/strategy/session_registry.go:155:func (r *SessionRegistry) Register(runID uuid.UUID, userID uuid.UUID, accountID, symbol, timeframe, mode string, scheduleID uuid.UUID, cancel context.CancelFunc) *ActiveSession {
backend/internal/connect/strategy/session_registry.go:179:func (r *SessionRegistry) Deregister(runID uuid.UUID) *ActiveSession {
backend/internal/connect/strategy/session_registry.go:199:func (r *SessionRegistry) Get(runID uuid.UUID) (*ActiveSession, bool) {
backend/internal/connect/strategy/session_registry.go:207:func (r *SessionRegistry) ListByUser(userID uuid.UUID) []*ActiveSession {
backend/internal/connect/strategy/session_registry.go:220:func (r *SessionRegistry) GetByScheduleID(scheduleID uuid.UUID) (*ActiveSession, bool) {
backend/internal/connect/strategy/session_registry.go:234:func (r *SessionRegistry) UpdatePnlFromPositions(accountID string, positions []mdtick.ProfitPosition) {
backend/internal/connect/strategy/session_registry.go:269:func (r *SessionRegistry) ListByAccount(accountID string) []*ActiveSession {
backend/internal/connect/strategy/session_registry.go:282:func (r *SessionRegistry) ListAll() []*ActiveSession {
backend/internal/connect/strategy/session_registry.go:293:func (r *SessionRegistry) Stop(runID uuid.UUID) error {
backend/internal/connect/strategy/session_registry.go:305:func (s *ActiveSession) RecordTick(t time.Time) {
backend/internal/connect/strategy/session_registry.go:315:func (s *ActiveSession) SetPnL(pnl string) {
backend/internal/connect/strategy/session_registry.go:326:func (s *ActiveSession) RecordSignal(event *SignalEvent) {
backend/internal/connect/strategy/session_registry.go:344:func (s *ActiveSession) SubscribeSignals() <-chan *SignalEvent {
backend/internal/connect/strategy/session_registry.go:354:func (r *SessionRegistry) InsertScheduleRunLog(ctx context.Context, userID, scheduleID uuid.UUID, kind, action, status, errorMessage, signalType string, signalVolume decimal.Decimal) {
backend/internal/connect/strategy/session_registry.go:367:func (s *ActiveSession) RecordError(err string) {
backend/internal/connect/strategy/session_registry.go:387:func (s *ActiveSession) SetCircuitOpen(open bool) {
backend/internal/connect/strategy/session_registry.go:394:func (s *ActiveSession) IsCircuitOpen() bool {
backend/internal/connect/strategy/session_registry.go:401:func (s *ActiveSession) SetStderrTail(tail string) {
backend/internal/connect/strategy/session_registry.go:82:func (r *SessionRegistry) SetLogger(log *zap.Logger) { r.log = log }
backend/internal/connect/strategy/session_registry.go:85:func (r *SessionRegistry) SetLogRepository(repo *repository.LogRepository) { r.logRepo = repo }
backend/internal/connect/strategy/session_registry.go:89:func (r *SessionRegistry) SubscribeToMthub(mthubSvc *mthub.MtHubService) {
backend/internal/connect/strategy/shadow_verifier.go:60:func (sv *ShadowVerifier) Start(ctx context.Context) {
backend/internal/connect/strategy/shadow_verifier.go:65:func (sv *ShadowVerifier) Stop() {
backend/internal/connect/strategy/shadow_verifier.go:75:func (sv *ShadowVerifier) RecordBar(bar sdk.Bar) {
backend/internal/connect/strategy/shadow_verifier.go:92:func (sv *ShadowVerifier) RecordLiveSignal(barTime int64, action, volume, price string) {
backend/internal/connect/strategy/strategy_active_control.go:20:func (s *StrategyExecutionServer) StopStrategy(ctx context.Context, req *connect.Request[antv1.StopStrategyRequest]) (*connect.Response[antv1.StopStrategyResponse], error) {
backend/internal/connect/strategy/strategy_active_control.go:58:func (s *StrategyExecutionServer) StartStrategy(ctx context.Context, req *connect.Request[antv1.StartStrategyRequest]) (*connect.Response[antv1.StartStrategyResponse], error) {
backend/internal/connect/strategy/strategy_active_watch.go:176:func (s *StrategyExecutionServer) WatchActiveStrategies(
backend/internal/connect/strategy/strategy_active_watch.go:20:func (s *StrategyExecutionServer) ListActiveStrategies(ctx context.Context, req *connect.Request[antv1.ListActiveStrategiesRequest]) (*connect.Response[antv1.ListActiveStrategiesResponse], error) {
backend/internal/connect/strategy/strategy_active_watch.go:59:func (s *StrategyExecutionServer) GetActiveStrategy(ctx context.Context, req *connect.Request[antv1.GetActiveStrategyRequest]) (*connect.Response[antv1.GetActiveStrategyResponse], error) {
backend/internal/connect/strategy/strategy_active_watch.go:87:func (s *StrategyExecutionServer) WatchStrategySignals(ctx context.Context, req *connect.Request[antv1.WatchStrategySignalsRequest], stream *connect.ServerStream[antv1.StrategySignalEvent]) error {
backend/internal/connect/strategy/strategy_asset_handler.go:120:func (s *StrategyAssetServer) ReviewStrategyAsset(ctx context.Context, req *connect.Request[antv1.ReviewStrategyAssetRequest]) (*connect.Response[antv1.StrategyAsset], error) {
backend/internal/connect/strategy/strategy_asset_handler.go:132:func (s *StrategyAssetServer) CloneStrategyAsset(ctx context.Context, req *connect.Request[antv1.CloneStrategyAssetRequest]) (*connect.Response[antv1.CloneStrategyAssetResponse], error) {
backend/internal/connect/strategy/strategy_asset_handler.go:158:func (s *StrategyAssetServer) CheckAssetUpdate(ctx context.Context, req *connect.Request[antv1.CheckAssetUpdateRequest]) (*connect.Response[antv1.StrategyAssetClone], error) {
backend/internal/connect/strategy/strategy_asset_handler.go:171:func (s *StrategyAssetServer) SyncStrategyAsset(ctx context.Context, req *connect.Request[antv1.SyncStrategyAssetRequest]) (*connect.Response[antv1.StrategyAssetClone], error) {
backend/internal/connect/strategy/strategy_asset_handler.go:189:func (s *StrategyAssetServer) ListAssetClones(ctx context.Context, req *connect.Request[antv1.ListAssetClonesRequest]) (*connect.Response[antv1.ListAssetClonesResponse], error) {
backend/internal/connect/strategy/strategy_asset_handler.go:69:func (s *StrategyAssetServer) ListStrategyAssets(ctx context.Context, req *connect.Request[antv1.ListStrategyAssetsRequest]) (*connect.Response[antv1.ListStrategyAssetsResponse], error) {
backend/internal/connect/strategy/strategy_asset_handler.go:83:func (s *StrategyAssetServer) GetStrategyAsset(ctx context.Context, req *connect.Request[antv1.GetStrategyAssetRequest]) (*connect.Response[antv1.StrategyAsset], error) {
backend/internal/connect/strategy/strategy_asset_handler.go:96:func (s *StrategyAssetServer) SubmitAssetReview(ctx context.Context, req *connect.Request[antv1.SubmitAssetReviewRequest]) (*connect.Response[antv1.StrategyAsset], error) {
backend/internal/connect/strategy/strategy_backtest_crud.go:160:func (s *StrategyExecutionServer) GetBacktestRun(ctx context.Context, req *connect.Request[antv1.GetBacktestRunRequest]) (*connect.Response[antv1.GetBacktestRunResponse], error) {
backend/internal/connect/strategy/strategy_backtest_crud.go:188:func (s *StrategyExecutionServer) ListBacktestRuns(ctx context.Context, req *connect.Request[antv1.ListBacktestRunsRequest]) (*connect.Response[antv1.ListBacktestRunsResponse], error) {
backend/internal/connect/strategy/strategy_backtest_crud.go:224:func (s *StrategyExecutionServer) CancelBacktestRun(ctx context.Context, req *connect.Request[antv1.CancelBacktestRunRequest]) (*connect.Response[antv1.CancelBacktestRunResponse], error) {
backend/internal/connect/strategy/strategy_backtest_crud.go:246:func (s *StrategyExecutionServer) DeleteBacktestRun(ctx context.Context, req *connect.Request[antv1.DeleteBacktestRunRequest]) (*connect.Response[antv1.DeleteBacktestRunResponse], error) {
backend/internal/connect/strategy/strategy_backtest_crud.go:25:func (s *StrategyExecutionServer) StartBacktestRun(ctx context.Context, req *connect.Request[antv1.StartBacktestRunRequest]) (*connect.Response[antv1.StartBacktestRunResponse], error) {
backend/internal/connect/strategy/strategy_backtest_crud.go:265:func (s *StrategyExecutionServer) DeleteBacktestRuns(ctx context.Context, req *connect.Request[antv1.DeleteBacktestRunsRequest]) (*connect.Response[antv1.DeleteBacktestRunsResponse], error) {
backend/internal/connect/strategy/strategy_backtest_crud.go:294:func (s *StrategyExecutionServer) UpdateBacktestRun(ctx context.Context, req *connect.Request[antv1.UpdateBacktestRunRequest]) (*connect.Response[antv1.UpdateBacktestRunResponse], error) {
backend/internal/connect/strategy/strategy_backtest_validate.go:123:func (s *StrategyExecutionServer) HasBacktestData(ctx context.Context, symbol, timeframe string) bool {
backend/internal/connect/strategy/strategy_backtest_watch.go:15:func (s *StrategyExecutionServer) WatchBacktestRun(ctx context.Context, req *connect.Request[antv1.WatchBacktestRunRequest], stream *connect.ServerStream[antv1.BacktestRunUpdate]) error {
backend/internal/connect/strategy/strategy_execution_handler.go:126:func (s *StrategyExecutionServer) SetCoverageChecker(c CoverageChecker) { s.coverageChecker = c }
backend/internal/connect/strategy/strategy_execution_handler.go:129:func (s *StrategyExecutionServer) SetQualityValidator(v QualityValidator) { s.qualityValidator = v }
backend/internal/connect/strategy/strategy_execution_handler.go:132:func (s *StrategyExecutionServer) SetGateEvalRepo(r *repository.GateEvaluationRepository) {
backend/internal/connect/strategy/strategy_execution_handler.go:142:func (s *StrategyExecutionServer) SetMarketDataRepo(r repository.MarketDataStore) {
backend/internal/connect/strategy/strategy_execution_handler.go:157:func (s *StrategyExecutionServer) SetBarSource(bs BarSource)                      { s.barSource = bs }
backend/internal/connect/strategy/strategy_execution_handler.go:158:func (s *StrategyExecutionServer) SetMtHub(h *mthub.MtHubService)                 { s.mtHub = h }
backend/internal/connect/strategy/strategy_execution_handler.go:159:func (s *StrategyExecutionServer) SetPaperEngine(pe PaperOrderExecutor)           { s.paperEngine = pe }
backend/internal/connect/strategy/strategy_execution_handler.go:160:func (s *StrategyExecutionServer) SetGoExecutor(ge *GoExecutor)                   { s.goExecutor = ge }
backend/internal/connect/strategy/strategy_execution_handler.go:161:func (s *StrategyExecutionServer) SetRunRepo(r *repository.StrategyRunRepository) { s.runRepo = r }
backend/internal/connect/strategy/strategy_execution_handler.go:162:func (s *StrategyExecutionServer) SetImportedRepo(r *repository.ImportedStrategyRepository) {
backend/internal/connect/strategy/strategy_execution_handler.go:165:func (s *StrategyExecutionServer) SetVersionRepo(r *repository.StrategyVersionRepository) {
backend/internal/connect/strategy/strategy_execution_handler.go:168:func (s *StrategyExecutionServer) SetSessionRegistry(r *SessionRegistry) { s.sessionRegistry = r }
backend/internal/connect/strategy/strategy_execution_handler.go:169:func (s *StrategyExecutionServer) SetAccountLookup(f func(ctx context.Context, userID string) string) {
backend/internal/connect/strategy/strategy_execution_handler.go:172:func (s *StrategyExecutionServer) SetPositionCache(pc *PositionCache) { s.posCache = pc }
backend/internal/connect/strategy/strategy_execution_handler.go:173:func (s *StrategyExecutionServer) SetScheduleNameLookup(f func(ctx context.Context, scheduleID uuid.UUID) string) {
backend/internal/connect/strategy/strategy_execution_handler.go:176:func (s *StrategyExecutionServer) SetBrokerCompanyLookup(f func(ctx context.Context, accountID string) string) {
backend/internal/connect/strategy/strategy_execution_handler.go:188:func (s *StrategyExecutionServer) SetQuotaChecker(qc QuotaChecker) { s.quotaChecker = qc }
backend/internal/connect/strategy/strategy_execution_handler.go:191:func (s *StrategyExecutionServer) SetBoundSvc(b BoundAccountChecker) { s.boundSvc = b }
backend/internal/connect/strategy/strategy_execution_handler.go:205:func (s *StrategyExecutionServer) SetGate(g *risk.Gate) { s.gate = g }
backend/internal/connect/strategy/strategy_execution_handler.go:208:func (s *StrategyExecutionServer) AddGateRule(r risk.Rule) {
backend/internal/connect/strategy/strategy_execution_handler.go:215:func (s *StrategyExecutionServer) SetAccountProvider(p AccountStateProvider) { s.accountProvider = p }
backend/internal/connect/strategy/strategy_execution_handler.go:223:func (s *StrategyExecutionServer) SetNotificationSender(ns *notification.Sender) { s.notifSender = ns }
backend/internal/connect/strategy/strategy_execution_handler.go:224:func (s *StrategyExecutionServer) SetFailureSignatureRepo(repo *repository.FailureSignatureRepository) {
backend/internal/connect/strategy/strategy_execution_handler.go:227:func (s *StrategyExecutionServer) SetOnBacktestComplete(fn func(context.Context, *repository.BacktestRun)) {
backend/internal/connect/strategy/strategy_execution_handler.go:240:func (s *StrategyExecutionServer) Execute(ctx context.Context, req *connect.Request[antv1.ExecuteStrategyRequest]) (*connect.Response[antv1.ExecuteStrategyResponse], error) {
backend/internal/connect/strategy/strategy_execution_handler.go:264:func (s *StrategyExecutionServer) Validate(ctx context.Context, req *connect.Request[antv1.ValidateStrategyRequest]) (*connect.Response[antv1.ValidateStrategyResponse], error) {
backend/internal/connect/strategy/strategy_execution_handler.go:292:func (s *StrategyExecutionServer) Backtest(ctx context.Context, req *connect.Request[antv1.BacktestStrategyRequest]) (*connect.Response[antv1.BacktestStrategyResponse], error) {
backend/internal/connect/strategy/strategy_execution_handler.go:307:func (s *StrategyExecutionServer) GetTemplates(_ context.Context, _ *connect.Request[emptypb.Empty]) (*connect.Response[antv1.GetStrategyTemplatesResponse], error) {
backend/internal/connect/strategy/strategy_execution_handler.go:313:func (s *StrategyExecutionServer) ExecuteLive(ctx context.Context, req *connect.Request[antv1.ExecuteLiveRequest]) (*connect.Response[antv1.ExecuteLiveResponse], error) {
backend/internal/connect/strategy/strategy_execution_handler.go:364:func (s *StrategyExecutionServer) SetPgListen(l *pglisten.Listener) { s.pgListen = l }
backend/internal/connect/strategy/strategy_execution_runs.go:16:func (s *StrategyExecutionServer) ListStrategyRuns(ctx context.Context, req *connect.Request[antv1.ListStrategyRunsRequest]) (*connect.Response[antv1.ListStrategyRunsResponse], error) {
backend/internal/connect/strategy/strategy_execution_runs.go:46:func (s *StrategyExecutionServer) GetStrategyRun(ctx context.Context, req *connect.Request[antv1.GetStrategyRunRequest]) (*connect.Response[antv1.GetStrategyRunResponse], error) {
backend/internal/connect/strategy/strategy_experiment_handler.go:113:func (s *StrategyExperimentServer) SubmitStrategyExperiment(ctx context.Context, req *connect.Request[antv1.SubmitStrategyExperimentRequest]) (*connect.Response[antv1.SubmitStrategyExperimentResponse], error) {
backend/internal/connect/strategy/strategy_experiment_handler.go:157:func (s *StrategyExperimentServer) GetStrategyExperiment(ctx context.Context, req *connect.Request[antv1.GetStrategyExperimentRequest]) (*connect.Response[antv1.StrategyExperiment], error) {
backend/internal/connect/strategy/strategy_experiment_handler.go:170:func (s *StrategyExperimentServer) ListStrategyExperiments(ctx context.Context, req *connect.Request[antv1.ListStrategyExperimentsRequest]) (*connect.Response[antv1.ListStrategyExperimentsResponse], error) {
backend/internal/connect/strategy/strategy_experiment_handler.go:184:func (s *StrategyExperimentServer) CancelStrategyExperiment(ctx context.Context, req *connect.Request[antv1.CancelStrategyExperimentRequest]) (*connect.Response[antv1.StrategyExperiment], error) {
backend/internal/connect/strategy/strategy_experiment_handler.go:197:func (s *StrategyExperimentServer) ListExperimentCandidates(ctx context.Context, req *connect.Request[antv1.ListExperimentCandidatesRequest]) (*connect.Response[antv1.ListExperimentCandidatesResponse], error) {
backend/internal/connect/strategy/strategy_experiment_handler.go:214:func (s *StrategyExperimentServer) GetExperimentCandidate(ctx context.Context, req *connect.Request[antv1.GetExperimentCandidateRequest]) (*connect.Response[antv1.StrategyExperimentCandidate], error) {
backend/internal/connect/strategy/strategy_experiment_handler.go:227:func (s *StrategyExperimentServer) PromoteCandidateToDraft(ctx context.Context, req *connect.Request[antv1.PromoteCandidateToDraftRequest]) (*connect.Response[antv1.PromoteCandidateToDraftResponse], error) {
backend/internal/connect/strategy/strategy_experiment_handler.go:245:func (s *StrategyExperimentServer) WatchExperiment(ctx context.Context, req *connect.Request[antv1.WatchExperimentRequest], stream *connect.ServerStream[antv1.WatchExperimentEvent]) error {
backend/internal/connect/strategy/strategy_experiment_handler.go:304:func (s *StrategyExperimentServer) SetPgListen(l *pglisten.Listener) { s.pgListen = l }
backend/internal/connect/strategy/strategy_experiment_worker.go:44:func (w *ExperimentWorker) Start(ctx context.Context) {
backend/internal/connect/strategy/strategy_experiment_worker.go:68:func (w *ExperimentWorker) SetPgListen(l *pglisten.Listener) { w.pgListen = l }
backend/internal/connect/strategy/strategy_experiment_worker.go:83:func (w *ExperimentWorker) Stop() { close(w.stopCh) }
backend/internal/connect/strategy/strategy_experiment_worker.go:86:func (w *ExperimentWorker) SetAIService(svc *systemai.Service) { w.systemAISvc = svc }
backend/internal/connect/strategy/strategy_experiment_worker.go:89:func (w *ExperimentWorker) SetExecutor(e BacktestExecutor) { w.executor = e }
backend/internal/connect/strategy/strategy_handler.go:41:func (s *StrategyServer) SetCodeAccessChecker(c CodeAccessChecker) { s.codeAccess = c }
backend/internal/connect/strategy/strategy_handler.go:50:func (s *StrategyServer) CancelTemplateDraft(ctx context.Context, req *connect.Request[antv1.CancelTemplateDraftRequest]) (*connect.Response[emptypb.Empty], error) {
backend/internal/connect/strategy/strategy_handler.go:64:func (s *StrategyServer) SetEngine(e *ScheduleEngine) { s.engine = e }
backend/internal/connect/strategy/strategy_handler.go:66:func (s *StrategyServer) SetBoundSvc(b BoundAccountChecker) { s.boundSvc = b }
backend/internal/connect/strategy/strategy_handler.go:68:func (s *StrategyServer) SetSessionRegistry(r *SessionRegistry) { s.sessionRegistry = r }
backend/internal/connect/strategy/strategy_handler.go:70:func (s *StrategyServer) SetPositionCache(pc *PositionCache) { s.posCache = pc }
backend/internal/connect/strategy/strategy_handler.go:86:func (s *StrategyServer) SetPgListen(l *pglisten.Listener) { s.pgListen = l }
backend/internal/connect/strategy/strategy_import_handler.go:113:func (s *StrategyExecutionServer) GetImportedStrategy(ctx context.Context, req *connect.Request[antv1.GetImportedStrategyRequest]) (*connect.Response[antv1.GetImportedStrategyResponse], error) {
backend/internal/connect/strategy/strategy_import_handler.go:40:func (s *StrategyExecutionServer) AnalyzeImportCode(ctx context.Context, req *connect.Request[antv1.AnalyzeImportCodeRequest]) (*connect.Response[antv1.AnalyzeImportCodeResponse], error) {
backend/internal/connect/strategy/strategy_schedule_positions.go:22:func (s *StrategyServer) GetSchedulePositions(ctx context.Context, req *connect.Request[antv1.GetSchedulePositionsRequest]) (*connect.Response[antv1.GetSchedulePositionsResponse], error) {
backend/internal/connect/strategy/strategy_schedules.go:118:func (s *StrategyServer) UpdateSchedule(ctx context.Context, req *connect.Request[antv1.UpdateScheduleRequest]) (*connect.Response[antv1.StrategySchedule], error) {
backend/internal/connect/strategy/strategy_schedules.go:245:func (s *StrategyServer) DeleteSchedule(ctx context.Context, req *connect.Request[antv1.DeleteScheduleRequest]) (*connect.Response[emptypb.Empty], error) {
backend/internal/connect/strategy/strategy_schedules.go:25:func (s *StrategyServer) ListSchedules(ctx context.Context, req *connect.Request[antv1.ListSchedulesRequest]) (*connect.Response[antv1.ListSchedulesResponse], error) {
backend/internal/connect/strategy/strategy_schedules.go:260:func (s *StrategyServer) ToggleSchedule(ctx context.Context, req *connect.Request[antv1.ToggleScheduleRequest]) (*connect.Response[antv1.StrategySchedule], error) {
backend/internal/connect/strategy/strategy_schedules.go:306:func (s *StrategyServer) WatchSchedules(ctx context.Context, req *connect.Request[antv1.WatchSchedulesRequest], stream *connect.ServerStream[antv1.WatchSchedulesEvent]) error {
backend/internal/connect/strategy/strategy_schedules.go:37:func (s *StrategyServer) GetSchedule(ctx context.Context, req *connect.Request[antv1.GetScheduleRequest]) (*connect.Response[antv1.StrategySchedule], error) {
backend/internal/connect/strategy/strategy_schedules.go:49:func (s *StrategyServer) CreateSchedule(ctx context.Context, req *connect.Request[antv1.CreateScheduleRequest]) (*connect.Response[antv1.StrategySchedule], error) {
backend/internal/connect/strategy/strategy_signals.go:15:func (s *StrategyServer) RunBacktest(
backend/internal/connect/strategy/strategy_signals.go:34:func (s *StrategyServer) ListSignals(
backend/internal/connect/strategy/strategy_signals.go:51:func (s *StrategyServer) ExecuteSignal(
backend/internal/connect/strategy/strategy_signals.go:72:func (s *StrategyServer) ConfirmSignal(
backend/internal/connect/strategy/strategy_signals.go:85:func (s *StrategyServer) CancelSignal(
backend/internal/connect/strategy/strategy_template_handlers.go:126:func (s *StrategyServer) CreateTemplate(ctx context.Context, req *connect.Request[antv1.CreateTemplateRequest]) (*connect.Response[antv1.StrategyTemplate], error) {
backend/internal/connect/strategy/strategy_template_handlers.go:153:func (s *StrategyServer) UpdateTemplate(ctx context.Context, req *connect.Request[antv1.UpdateTemplateRequest]) (*connect.Response[antv1.StrategyTemplate], error) {
backend/internal/connect/strategy/strategy_template_handlers.go:202:func (s *StrategyServer) DeleteTemplate(ctx context.Context, req *connect.Request[antv1.DeleteTemplateRequest]) (*connect.Response[emptypb.Empty], error) {
backend/internal/connect/strategy/strategy_template_handlers.go:217:func (s *StrategyServer) CreateTemplateDraft(ctx context.Context, req *connect.Request[antv1.CreateTemplateDraftRequest]) (*connect.Response[antv1.StrategyTemplate], error) {
backend/internal/connect/strategy/strategy_template_handlers.go:231:func (s *StrategyServer) UpdateTemplateDraft(ctx context.Context, req *connect.Request[antv1.UpdateTemplateDraftRequest]) (*connect.Response[antv1.StrategyTemplate], error) {
backend/internal/connect/strategy/strategy_template_handlers.go:274:func (s *StrategyServer) PublishTemplateDraft(ctx context.Context, req *connect.Request[antv1.PublishTemplateDraftRequest]) (*connect.Response[antv1.StrategyTemplate], error) {
backend/internal/connect/strategy/strategy_template_handlers.go:52:func (s *StrategyServer) ListTemplates(ctx context.Context, req *connect.Request[antv1.ListTemplatesRequest]) (*connect.Response[antv1.ListTemplatesResponse], error) {
backend/internal/connect/strategy/strategy_template_handlers.go:65:func (s *StrategyServer) GetTemplate(ctx context.Context, req *connect.Request[antv1.GetTemplateRequest]) (*connect.Response[antv1.StrategyTemplate], error) {
backend/internal/connect/strategy/strategy_template_handlers.go:87:func (s *StrategyServer) ListStrategyCards(ctx context.Context, req *connect.Request[antv1.ListStrategyCardsRequest]) (*connect.Response[antv1.ListStrategyCardsResponse], error) {
backend/internal/connect/strategy/strategy_templates.go:105:func (s *MACrossStrategy) OnDeinit(ctx sdk.Context, reason string) error {
backend/internal/connect/strategy/strategy_templates.go:132:func (s *RSIStrategy) OnInit(ctx sdk.Context) error {
backend/internal/connect/strategy/strategy_templates.go:142:func (s *RSIStrategy) OnBar(ctx sdk.Context, timeframe string) (*sdk.Signal, error) {
backend/internal/connect/strategy/strategy_templates.go:201:func (s *RSIStrategy) OnDeinit(ctx sdk.Context, reason string) error {
backend/internal/connect/strategy/strategy_templates.go:227:func (s *BollingerStrategy) OnInit(ctx sdk.Context) error {
backend/internal/connect/strategy/strategy_templates.go:236:func (s *BollingerStrategy) OnBar(ctx sdk.Context, timeframe string) (*sdk.Signal, error) {
backend/internal/connect/strategy/strategy_templates.go:295:func (s *BollingerStrategy) OnDeinit(ctx sdk.Context, reason string) error {
backend/internal/connect/strategy/strategy_templates.go:33:func (s *MACrossStrategy) OnInit(ctx sdk.Context) error {
backend/internal/connect/strategy/strategy_templates.go:42:func (s *MACrossStrategy) OnBar(ctx sdk.Context, timeframe string) (*sdk.Signal, error) {
backend/internal/connect/strategy/strategy_version_handler.go:103:func (s *StrategyExecutionServer) DiffStrategyVersions(ctx context.Context, req *connect.Request[antv1.DiffStrategyVersionsRequest]) (*connect.Response[antv1.DiffStrategyVersionsResponse], error) {
backend/internal/connect/strategy/strategy_version_handler.go:134:func (s *StrategyExecutionServer) UpdateStrategyCode(ctx context.Context, req *connect.Request[antv1.UpdateStrategyCodeRequest]) (*connect.Response[antv1.UpdateStrategyCodeResponse], error) {
backend/internal/connect/strategy/strategy_version_handler.go:17:func (s *StrategyExecutionServer) ListStrategyVersions(ctx context.Context, req *connect.Request[antv1.ListStrategyVersionsRequest]) (*connect.Response[antv1.ListStrategyVersionsResponse], error) {
backend/internal/connect/strategy/strategy_version_handler.go:47:func (s *StrategyExecutionServer) GetStrategyVersion(ctx context.Context, req *connect.Request[antv1.GetStrategyVersionRequest]) (*connect.Response[antv1.GetStrategyVersionResponse], error) {
backend/internal/connect/strategy/strategy_version_handler.go:75:func (s *StrategyExecutionServer) RollbackStrategyVersion(ctx context.Context, req *connect.Request[antv1.RollbackStrategyVersionRequest]) (*connect.Response[antv1.RollbackStrategyVersionResponse], error) {
backend/internal/connect/strategy/vm_live_session.go:119:func (s *VMLiveSession) SendEvent(ctx context.Context, reqBytes []byte) ([]byte, error) {
backend/internal/connect/strategy/vm_live_session.go:133:func (s *VMLiveSession) Close() error {
backend/internal/connect/strategy/vm_live_session.go:185:func (s *VMLiveSession) SetDiag(d *sessionDiag) {
backend/internal/connect/strategy/vm_live_session.go:81:func (s *VMLiveSession) Start(ctx context.Context, reqBytes []byte) ([]byte, error) {
backend/internal/connect/strategy/walk_forward_handler.go:36:func (s *WalkForwardServer) SetPgListen(l *pglisten.Listener) { s.pgListen = l }
backend/internal/connect/strategy/walk_forward_handler.go:39:func (s *WalkForwardServer) GetWalkForwardReport(
backend/internal/connect/strategy/walk_forward_handler.go:57:func (s *WalkForwardServer) WatchWalkForwardReport(
backend/internal/mthub/broker_types.go:104:func (b *PositionSnapshotBroker) SubscribeAll() (<-chan *PositionSnapshot, func()) {
backend/internal/mthub/broker_types.go:153:func (b *BarBroker) Publish(ev *BarUpdate) {
backend/internal/mthub/broker_types.go:180:func (b *BarBroker) DroppedBars(accountID string) int64 {
backend/internal/mthub/broker_types.go:185:func (b *BarBroker) Subscribe(accountID string) (<-chan *BarUpdate, func()) {
backend/internal/mthub/broker_types.go:227:func (b *AccountStatusBroker) Publish(ev *AccountStatusEvent) {
backend/internal/mthub/broker_types.go:241:func (b *AccountStatusBroker) Subscribe(accountID string) (<-chan *AccountStatusEvent, func()) {
backend/internal/mthub/broker_types.go:60:func (b *PositionSnapshotBroker) Publish(ev *PositionSnapshot) {
backend/internal/mthub/broker_types.go:84:func (b *PositionSnapshotBroker) Subscribe(accountID string) (<-chan *PositionSnapshot, func()) {
backend/internal/mthub/derived_state.go:111:func (dc *DerivedComputer) Start() {
backend/internal/mthub/derived_state.go:116:func (dc *DerivedComputer) Stop() {
backend/internal/mthub/derived_state.go:121:func (dc *DerivedComputer) State() *DerivedState {
backend/internal/mthub/derived_state.go:62:func (d *DerivedState) Update(accounts map[string]*AccountDerivedState, totalExposure, totalMargin, grossPnL, netPnL decimal.Decimal, var95 float64) {
backend/internal/mthub/derived_state.go:76:func (d *DerivedState) Get() (accounts map[string]*AccountDerivedState, totalExposure, totalMargin, grossPnL, netPnL decimal.Decimal, var95 float64, lastUpdated time.Time) {
backend/internal/mthub/derived_state.go:83:func (d *DerivedState) GetAccount(accountID string) *AccountDerivedState {
backend/internal/mthub/hub_estimator.go:149:func (e *HubCostEstimator) Refresh(symbol string) {
backend/internal/mthub/hub_estimator.go:46:func (e *HubCostEstimator) Estimate(ctx context.Context, params costsvc.EstimateParams) costsvc.CostBreakdown {
backend/internal/mthub/idempotency.go:151:func (g *IdempotencyGuard) CheckAndSet(ctx context.Context, accountID, clientID string, ticket int64) (isDup bool, existingTicket int64, err error) {
backend/internal/mthub/idempotency.go:167:func (g *IdempotencyGuard) SetTicket(ctx context.Context, accountID, clientID string, ticket int64) error {
backend/internal/mthub/idempotency.go:172:func (g *IdempotencyGuard) DeleteKey(ctx context.Context, accountID, clientID string) {
backend/internal/mthub/idempotency.go:53:func (g *ThreeLayerGuard) CheckAndSet(ctx context.Context, accountID, clientID string, ticket int64) (isDup bool, existingTicket int64, err error) {
backend/internal/mthub/idempotency.go:87:func (g *ThreeLayerGuard) Confirm(ctx context.Context, accountID, clientID string, ticket int64) error {
backend/internal/mthub/oms_writer.go:101:func (w *OmsWriter) SetOrderEventBroker(b *OrderEventBroker) {
backend/internal/mthub/oms_writer.go:119:func (w *OmsWriter) InsertOrder(ctx context.Context, orderID, accountID, platform, symbol string, orderType int16, volume, price, stopLoss, takeProfit decimal.Decimal, magic int32) error {
backend/internal/mthub/oms_writer.go:137:func (w *OmsWriter) UpdateTicket(ctx context.Context, orderID string, ticket int64) error {
backend/internal/mthub/oms_writer.go:149:func (w *OmsWriter) OrderIDByTicket(ctx context.Context, accountID string, ticket int64) (orderID, state string, err error) {
backend/internal/mthub/oms_writer.go:161:func (w *OmsWriter) Transition(ctx context.Context, orderID, accountID string, current, next OMSState) error {
backend/internal/mthub/order_types.go:87:func (r *OrderRecord) OrderTypeString() string {
backend/internal/mthub/reconcile_gate.go:24:func (g *ReconcileGate) EnterReconciling(accountID string) {
backend/internal/mthub/reconcile_gate.go:31:func (g *ReconcileGate) MarkReconciled(accountID string) {
backend/internal/mthub/reconcile_gate.go:38:func (g *ReconcileGate) CanAccept(accountID string) bool {
backend/internal/mthub/reconcile_gate.go:45:func (g *ReconcileGate) IsReconciling(accountID string) bool {
backend/internal/mthub/reconcile_gate.go:52:func (g *ReconcileGate) ReconcilingCount() int {
backend/internal/mthub/reconcile_gate.go:59:func (g *ReconcileGate) EnterAll(accountIDs []string) {
backend/internal/mthub/reconciliation.go:30:func (r *ReconciliationLoop) SetMtHubService(svc *MtHubService) {
backend/internal/mthub/reconciliation.go:37:func (r *ReconciliationLoop) Start(ctx context.Context) {
backend/internal/mthub/reconciliation.go:53:func (r *ReconciliationLoop) ReconcileAccount(ctx context.Context, accountID string) {
backend/internal/mthub/reconciliation.go:62:func (r *ReconciliationLoop) TriggerReconcile(accountID string) {
backend/internal/mthub/service.go:117:func (s *MtHubService) PublishBar(ev *BarUpdate) {
backend/internal/mthub/service.go:124:func (s *MtHubService) SubscribeBarUpdates(accountID string) (<-chan *BarUpdate, func()) {
backend/internal/mthub/service.go:134:func (s *MtHubService) PublishTick(ev *TickUpdate) {
backend/internal/mthub/service.go:141:func (s *MtHubService) SubscribeTickUpdates(accountID string) (<-chan *TickUpdate, func()) {
backend/internal/mthub/service.go:152:func (s *MtHubService) LatestTick(accountID, symbol string) *TickUpdate {
backend/internal/mthub/service.go:161:func (s *MtHubService) WatchAllTicks() (<-chan *TickUpdate, func()) {
backend/internal/mthub/service.go:171:func (s *MtHubService) PublishTradeEvent(ev *BrokerTradeEvent) {
backend/internal/mthub/service.go:178:func (s *MtHubService) SubscribeTradeEvents(accountID string) (<-chan *BrokerTradeEvent, func()) {
backend/internal/mthub/service.go:188:func (s *MtHubService) PublishAccountStatus(ev *AccountStatusEvent) {
backend/internal/mthub/service.go:195:func (s *MtHubService) SubscribeAccountStatus(accountID string) (<-chan *AccountStatusEvent, func()) {
backend/internal/mthub/service.go:208:func (s *MtHubService) Platform(accountID string) string {
backend/internal/mthub/service.go:218:func (s *MtHubService) SessionState(ctx context.Context, accountID string) string {
backend/internal/mthub/service.go:251:func (s *MtHubService) OpenedOrders(ctx context.Context, accountID string) ([]*OrderRecord, error) {
backend/internal/mthub/service.go:273:func (s *MtHubService) OrderHistory(ctx context.Context, accountID string, from, to time.Time) ([]*OrderRecord, error) {
backend/internal/mthub/service.go:295:func (s *MtHubService) SymbolParams(ctx context.Context, accountID string, canonicals []string) ([]*SymbolParam, error) {
backend/internal/mthub/service.go:318:func (s *MtHubService) ActiveAccountIDs() []string {
backend/internal/mthub/service.go:323:func (s *MtHubService) PriceHistory(ctx context.Context, accountID, symbol, period string, from, to int64, count int) ([]*Bar, error) {
backend/internal/mthub/service.go:345:func (s *MtHubService) SymbolList(ctx context.Context, accountID string) ([]string, error) {
backend/internal/mthub/service.go:369:func (s *MtHubService) SubscribeSymbols(ctx context.Context, accountID string, symbols []string) error {
backend/internal/mthub/service.go:378:func (s *MtHubService) SubscribeUserOrderEvents(ctx context.Context, userID string) (<-chan *OrderEvent, func()) {
backend/internal/mthub/service.go:383:func (s *MtHubService) PublishPositionSnapshot(ev *PositionSnapshot) {
backend/internal/mthub/service.go:388:func (s *MtHubService) SubscribePositionSnapshots(ctx context.Context, accountID string) (<-chan *PositionSnapshot, func()) {
backend/internal/mthub/service_account_profit.go:11:func (s *MtHubService) SubscribeAccountProfit(ctx context.Context, accountID string) (<-chan *AccountProfitEvent, func()) {
backend/internal/mthub/service_account_profit.go:16:func (s *MtHubService) SubscribeAccountProfitAll() (<-chan *AccountProfitEvent, func()) {
backend/internal/mthub/service_account_profit.go:6:func (s *MtHubService) PublishAccountProfit(ev *AccountProfitEvent) {
backend/internal/mthub/service_orders.go:21:func (s *MtHubService) PlaceOrder(ctx context.Context, req *OrderRequest) (*OrderRecord, error) {
backend/internal/mthub/service_orders.go:347:func (s *MtHubService) TransitionOrderByTicket(ctx context.Context, accountID string, ticket int64, to OMSState) {
backend/internal/mthub/service_orders.go:390:func (s *MtHubService) PublishTradeEventFromUpdate(
backend/internal/mthub/service_orders_close.go:27:func (s *MtHubService) CloseOrder(ctx context.Context, accountID string, ticket int64, lots decimal.Decimal) error {
backend/internal/mthub/service_orders_delete.go:15:func (s *MtHubService) DeleteOrder(ctx context.Context, accountID string, ticket int64) error {
backend/internal/mthub/service_orders_modify.go:18:func (s *MtHubService) ModifyOrder(ctx context.Context, accountID string, ticket int64, sl, tp, price decimal.Decimal) error {
backend/internal/mthub/service_setters.go:15:func (s *MtHubService) SetKillSwitch(ks KillSwitchGate) { s.killSwitch = ks }
backend/internal/mthub/service_setters.go:20:func (s *MtHubService) SetReconcileTrigger(f func(accountID string)) { s.reconcileTrigger = f }
backend/internal/mthub/service_setters.go:29:func (s *MtHubService) SetBrokerRegistry(r BrokerRegistry) { s.brokerRegistry = r }
backend/internal/mthub/service_setters.go:32:func (s *MtHubService) BrokerRegistry() BrokerRegistry { return s.brokerRegistry }
backend/internal/mthub/service_setters.go:38:func (s *MtHubService) SetAccountOwnerVerifier(v AccountOwnerVerifier) { s.accountOwnerVerifier = v }
backend/internal/mthub/service_setters.go:41:func (s *MtHubService) SetLogger(l *zap.Logger) { s.logger = l }
backend/internal/mthub/service_setters.go:44:func (s *MtHubService) SetBarBroker(b *BarBroker) { s.barBroker = b }
backend/internal/mthub/service_setters.go:47:func (s *MtHubService) SetTickBroker(b *TickBroker) { s.tickBroker = b }
backend/internal/mthub/service_setters.go:50:func (s *MtHubService) SetTradeBroker(b *TradeBroker) { s.tradeBroker = b }
backend/internal/mthub/service_setters.go:53:func (s *MtHubService) SetStatusBroker(b *AccountStatusBroker) { s.statusBroker = b }
backend/internal/mthub/service_setters.go:56:func (s *MtHubService) SetGuard(g *risk.Guard) { s.guard = g }
backend/internal/mthub/service_setters.go:59:func (s *MtHubService) SetOmsWriter(w *OmsWriter) { s.omsWriter = w }
backend/internal/mthub/service_setters.go:62:func (s *MtHubService) SetGate(g *risk.Gate) { s.gate = g }
backend/internal/mthub/service_setters.go:65:func (s *MtHubService) SetUserLimiter(l *usermgr.UserLimiter) { s.userLimiter = l }
backend/internal/mthub/service_setters.go:68:func (s *MtHubService) SetCostEstimator(e costsvc.CostEstimator) { s.costEstimator = e }
backend/internal/mthub/service_setters.go:71:func (s *MtHubService) SetAccountStateProvider(p AccountStateProvider) { s.accountStateProvider = p }
backend/internal/mthub/service_symbol_params.go:17:func (s *MtHubService) CachedSymbolParam(ctx context.Context, accountID, canonical string) (*SymbolParam, error) {
backend/internal/mthub/snapshot_persister.go:45:func (p *SnapshotPersister) Start(ctx context.Context) {
backend/internal/mthub/snapshot_persister.go:99:func (p *SnapshotPersister) GetSnapshot(ctx context.Context, accountID string) (*antv1.MtPositionSnapshotRecord, error) {
backend/internal/mthub/state_cache.go:103:func (c *StateCache) GetPositionsByAccount(accountID string) []*PositionCacheEntry {
backend/internal/mthub/state_cache.go:117:func (c *StateCache) ApplyEvent(ev *TradeEvent) {
backend/internal/mthub/state_cache.go:197:func (c *StateCache) LoadFromRedis(ctx context.Context) error {
backend/internal/mthub/state_cache.go:220:func (c *StateCache) Stats() (orders, positions int) {
backend/internal/mthub/state_cache.go:76:func (c *StateCache) GetOrder(ticket int64) *OrderStateCacheEntry {
backend/internal/mthub/state_cache.go:83:func (c *StateCache) GetOrdersByAccount(accountID string) []*OrderStateCacheEntry {
backend/internal/mthub/state_cache.go:96:func (c *StateCache) GetPosition(accountID, canonical string) *PositionCacheEntry {
backend/internal/mthub/tick_broker.go:108:func (b *TickBroker) WatchAll() (<-chan *TickUpdate, func()) {
backend/internal/mthub/tick_broker.go:49:func (b *TickBroker) LatestTick(accountID, symbol string) *TickUpdate {
backend/internal/mthub/tick_broker.go:57:func (b *TickBroker) Subscribe(accountID string) (<-chan *TickUpdate, func()) {
backend/internal/mthub/tick_broker.go:79:func (b *TickBroker) Publish(u *TickUpdate) {
backend/internal/mthub/trade_broker.go:57:func (b *TradeBroker) Subscribe(accountID string) (<-chan *BrokerTradeEvent, func()) {
backend/internal/mthub/trade_broker.go:82:func (b *TradeBroker) Publish(evt *BrokerTradeEvent) {
backend/internal/mthub/trade_event_store.go:95:func (s *TradeEventStore) Publish(ctx context.Context, ev *TradeEvent) error {
backend/internal/mthub/types.go:108:func (h *Hub) CloseSession(ctx context.Context, id string) error {
backend/internal/mthub/types.go:116:func (h *Hub) ActiveAccountIDs() []string {
backend/internal/mthub/types.go:130:func (e *HubError) Error() string { return "mthub: " + e.Msg }
backend/internal/mthub/types.go:165:func (b *AccountProfitBroker) Publish(ev *AccountProfitEvent) {
backend/internal/mthub/types.go:17:func (s *Session) IsExpired() bool {
backend/internal/mthub/types.go:187:func (b *AccountProfitBroker) WatchAll() (<-chan *AccountProfitEvent, func()) {
backend/internal/mthub/types.go:206:func (b *AccountProfitBroker) Subscribe(accountID string) (<-chan *AccountProfitEvent, func()) {
backend/internal/mthub/types.go:234:func (b *OrderEventBroker) PublishEvent(userID string, ev *OrderEvent) {
backend/internal/mthub/types.go:245:func (b *OrderEventBroker) Subscribe(userID string) (<-chan *OrderEvent, func()) {
backend/internal/mthub/types.go:47:func (h *Hub) Register(id string, s *Session, e OrderExecutor) {
backend/internal/mthub/types.go:64:func (h *Hub) WaitSession(id string) <-chan struct{} {
backend/internal/mthub/types.go:77:func (h *Hub) Get(id string) OrderExecutor {
backend/internal/mthub/types.go:83:func (h *Hub) EnsureSession(ctx context.Context, id string) (*Session, error) {
backend/internal/mthub/types.go:98:func (h *Hub) RemoveSession(id string) {
backend/internal/paper/engine.go:146:func (e *PaperEngine) PaperPnl(ctx context.Context, accountID, symbol string, bid, ask decimal.Decimal) (decimal.Decimal, error) {
backend/internal/paper/engine.go:167:func (e *PaperEngine) ClosePaperOrder(ctx context.Context, accountID, symbol string) error {
backend/internal/paper/engine.go:195:func (e *PaperEngine) ModifyPaperOrder(ctx context.Context, accountID, symbol string, sl, tp decimal.Decimal) error {
backend/internal/paper/engine.go:221:func (e *PaperEngine) CancelPaperOrder(ctx context.Context, accountID, symbol string) error {
backend/internal/paper/engine.go:246:func (e *PaperEngine) Subscribe(accountID string) (<-chan *repository.PaperAccount, func()) {
backend/internal/paper/engine.go:51:func (e *PaperEngine) SetGuard(g *risk.Guard) { e.guard = g }
backend/internal/paper/engine.go:65:func (e *PaperEngine) PlacePaperOrder(ctx context.Context, accountID, symbol, side string,
backend/internal/risk/canary.go:127:func (c *CanaryController) AddAccount(accountID string) {
backend/internal/risk/canary.go:134:func (c *CanaryController) RemoveAccount(accountID string) {
backend/internal/risk/canary.go:141:func (c *CanaryController) IsCanaryAccount(accountID string) bool {
backend/internal/risk/canary.go:153:func (c *CanaryController) AllowedLotSize() decimal.Decimal {
backend/internal/risk/canary.go:163:func (c *CanaryController) CurrentStage() CanaryStage {
backend/internal/risk/canary.go:172:func (c *CanaryController) ActivateCanary() error {
backend/internal/risk/canary.go:188:func (c *CanaryController) RecordSuccessfulTrade() {
backend/internal/risk/canary.go:205:func (c *CanaryController) StepUpLotSize() error {
backend/internal/risk/canary.go:234:func (c *CanaryController) PromoteToFull() error {
backend/internal/risk/canary.go:251:func (c *CanaryController) EngageKillSwitch(reason string) {
backend/internal/risk/canary.go:266:func (c *CanaryController) DisengageKillSwitch() {
backend/internal/risk/canary.go:281:func (c *CanaryController) IsKillSwitchActive() bool {
backend/internal/risk/canary.go:290:func (c *CanaryController) Rollback() error {
backend/internal/risk/canary.go:308:func (c *CanaryController) History() []StageTransition {
backend/internal/risk/gate.go:106:func (g *Gate) SetAutotradeEnabled(fn func(userID string) bool) {
backend/internal/risk/gate.go:119:func (g *Gate) Evaluate(ctx context.Context, intent *antv1.OrderIntent, state *AccountState) *antv1.RiskDecision {
backend/internal/risk/gate.go:184:func (g *Gate) AddRule(rule Rule) {
backend/internal/risk/gate.go:191:func (g *Gate) Rules() []string {
backend/internal/risk/gate.go:233:func (e *AuditEntry) String() string {
backend/internal/risk/gate.go:99:func (g *Gate) SetKillSwitch(fn func() bool) {
backend/internal/risk/guard.go:58:func (g *Guard) Check(ctx context.Context, req *GuardRequest) *GuardResult {
backend/internal/risk/rule_user_config.go:33:func (r *UserRiskConfigRule) Name() string { return "user_risk_config" }
backend/internal/risk/rule_user_config.go:35:func (r *UserRiskConfigRule) Check(ctx context.Context, intent *antv1.OrderIntent, state *AccountState) *RuleResult {
backend/internal/risk/rules.go:130:func (r *DailyLossBreaker) Name() string { return "daily_loss" }
backend/internal/risk/rules.go:132:func (r *DailyLossBreaker) Check(_ context.Context, intent *antv1.OrderIntent, state *AccountState) *RuleResult {
backend/internal/risk/rules.go:151:func (r *DrawdownBreaker) Name() string { return "drawdown" }
backend/internal/risk/rules.go:153:func (r *DrawdownBreaker) Check(_ context.Context, intent *antv1.OrderIntent, state *AccountState) *RuleResult {
backend/internal/risk/rules.go:173:func (r *SymbolWhitelist) Name() string { return "symbol_whitelist" }
backend/internal/risk/rules.go:175:func (r *SymbolWhitelist) Check(_ context.Context, intent *antv1.OrderIntent, _ *AccountState) *RuleResult {
backend/internal/risk/rules.go:197:func (r *LeverageCap) Name() string { return "leverage_cap" }
backend/internal/risk/rules.go:199:func (r *LeverageCap) Check(_ context.Context, intent *antv1.OrderIntent, state *AccountState) *RuleResult {
backend/internal/risk/rules.go:221:func (r *OrderFrequencyLimit) Name() string { return "order_frequency" }
backend/internal/risk/rules.go:223:func (r *OrderFrequencyLimit) Check(_ context.Context, intent *antv1.OrderIntent, _ *AccountState) *RuleResult {
backend/internal/risk/rules.go:275:func (r *DuplicateProtection) Name() string { return "duplicate_protection" }
backend/internal/risk/rules.go:277:func (r *DuplicateProtection) Check(_ context.Context, intent *antv1.OrderIntent, _ *AccountState) *RuleResult {
backend/internal/risk/rules.go:318:func (r *MarginPreCheck) Name() string { return "margin_pre_check" }
backend/internal/risk/rules.go:320:func (r *MarginPreCheck) Check(_ context.Context, intent *antv1.OrderIntent, state *AccountState) *RuleResult {
backend/internal/risk/rules.go:53:func (r *MaxLotSize) Name() string { return "max_lot_size" }
backend/internal/risk/rules.go:55:func (r *MaxLotSize) Check(_ context.Context, intent *antv1.OrderIntent, _ *AccountState) *RuleResult {
backend/internal/risk/rules.go:73:func (r *MaxPositionCount) Name() string { return "max_position_count" }
backend/internal/risk/rules.go:75:func (r *MaxPositionCount) Check(_ context.Context, intent *antv1.OrderIntent, state *AccountState) *RuleResult {
backend/internal/risk/rules.go:94:func (r *MaxExposure) Name() string { return "max_exposure" }
backend/internal/risk/rules.go:96:func (r *MaxExposure) Check(_ context.Context, intent *antv1.OrderIntent, state *AccountState) *RuleResult {
backend/internal/risk/rules_risksvc.go:104:func (r *MarginFloorRule) Name() string { return "margin_floor" }
backend/internal/risk/rules_risksvc.go:106:func (r *MarginFloorRule) Check(_ context.Context, intent *antv1.OrderIntent, state *AccountState) *RuleResult {
backend/internal/risk/rules_risksvc.go:167:func (a *risksvcCapStoreAdapter) GetTier(_ context.Context, userID string) (CapabilityTier, error) {
backend/internal/risk/rules_risksvc.go:180:func (r *CapabilityTierRule) Name() string { return "capability_tier" }
backend/internal/risk/rules_risksvc.go:182:func (r *CapabilityTierRule) Check(ctx context.Context, intent *antv1.OrderIntent, _ *AccountState) *RuleResult {
backend/internal/risk/rules_risksvc.go:34:func (r *KycJurisdictionGateRule) Name() string { return "kyc_jurisdiction" }
backend/internal/risk/rules_risksvc.go:36:func (r *KycJurisdictionGateRule) Check(ctx context.Context, intent *antv1.OrderIntent, _ *AccountState) *RuleResult {
backend/internal/risk/rules_risksvc.go:70:func (r *ContractExpiryRule) Name() string { return "contract_expiry" }
backend/internal/risk/rules_risksvc.go:72:func (r *ContractExpiryRule) Check(ctx context.Context, intent *antv1.OrderIntent, _ *AccountState) *RuleResult {
backend/internal/risksvc/block_allocator.go:103:func (a *VWAPAllocator) Name() string { return "vwap" }
backend/internal/risksvc/block_allocator.go:105:func (a *VWAPAllocator) Allocate(_ context.Context, totalVolume decimal.Decimal, accounts []AllocAccount) map[string]decimal.Decimal {
backend/internal/risksvc/block_allocator.go:27:func (a *ProRataAllocator) Name() string { return "pro_rata" }
backend/internal/risksvc/block_allocator.go:29:func (a *ProRataAllocator) Allocate(_ context.Context, totalVolume decimal.Decimal, accounts []AllocAccount) map[string]decimal.Decimal {
backend/internal/risksvc/block_allocator.go:67:func (a *FIFOAllocator) Name() string { return "fifo" }
backend/internal/risksvc/block_allocator.go:69:func (a *FIFOAllocator) Allocate(_ context.Context, totalVolume decimal.Decimal, accounts []AllocAccount) map[string]decimal.Decimal {
backend/internal/risksvc/capability.go:109:func (s *CapabilityStore) Set(c *Capability) {
backend/internal/risksvc/capability.go:117:func (s *CapabilityStore) LoadFromPG(ctx context.Context, rows interface{ Scan(dest ...interface{}) error; Next() bool; Close() }) error {
backend/internal/risksvc/capability.go:157:func (s *CapabilityStore) Count() int {
backend/internal/risksvc/capability.go:63:func (c *Capability) HasOrderType(ot string) bool {
backend/internal/risksvc/capability.go:76:func (c *Capability) TierCheck() *PreCheckResult {
backend/internal/risksvc/capability.go:99:func (s *CapabilityStore) Get(userID string) *Capability {
backend/internal/risksvc/engine.go:22:func (e *Engine) SetUserLimiter(l *usermgr.UserLimiter) { e.userLimiter = l }
backend/internal/risksvc/engine.go:27:func (e *Engine) Evaluate(ctx context.Context, req *CheckRequest) *CheckResult {
backend/internal/risksvc/engine.go:49:func (e *Engine) Rules() []string {
backend/internal/risksvc/hardlimit.go:114:func (r *KillSwitchRule) Name() string { return "kill_switch" }
backend/internal/risksvc/hardlimit.go:116:func (r *KillSwitchRule) Check(_ context.Context, req *HardLimitRequest) error {
backend/internal/risksvc/hardlimit.go:137:func (r *ContractExpiryRule) Name() string { return "contract_expiry" }
backend/internal/risksvc/hardlimit.go:139:func (r *ContractExpiryRule) Check(_ context.Context, req *HardLimitRequest) error {
backend/internal/risksvc/hardlimit.go:172:func (e *HardLimitEvaluator) Evaluate(ctx context.Context, req *HardLimitRequest) error {
backend/internal/risksvc/hardlimit.go:188:func (e *HardLimitError) Error() string {
backend/internal/risksvc/hardlimit.go:57:func (r *KycJurisdictionRule) Name() string { return "kyc_jurisdiction" }
backend/internal/risksvc/hardlimit.go:59:func (r *KycJurisdictionRule) Check(ctx context.Context, req *HardLimitRequest) error {
backend/internal/risksvc/hardlimit.go:75:func (r *MarginFloorRule) Name() string { return "margin_floor" }
backend/internal/risksvc/hardlimit.go:77:func (r *MarginFloorRule) Check(_ context.Context, req *HardLimitRequest) error {
backend/internal/risksvc/jurisdiction.go:188:func (r *MaxMindGeoIPResolver) CountryCode(ipStr string) (string, error) {
backend/internal/risksvc/jurisdiction.go:91:func (g *JurisdictionGate) Check(ctx context.Context, userID, clientIP string) error {
backend/internal/risksvc/jurisdiction_store.go:100:func (s *PgJurisdictionStore) IsQuestionnaireCompleted(ctx context.Context, userID string) (bool, error) {
backend/internal/risksvc/jurisdiction_store.go:112:func (s *PgJurisdictionStore) SubmitQuestionnaire(ctx context.Context, userID, version string, riskScore int) error {
backend/internal/risksvc/jurisdiction_store.go:125:func (s *PgJurisdictionStore) IsSanctioned(ctx context.Context, countryCode string) (bool, error) {
backend/internal/risksvc/jurisdiction_store.go:137:func (s *PgJurisdictionStore) AcceptDisclaimerAt(ctx context.Context, userID string) (*time.Time, error) {
backend/internal/risksvc/jurisdiction_store.go:149:func (s *PgJurisdictionStore) QuestionnaireCompletedAt(ctx context.Context, userID string) (*time.Time, error) {
backend/internal/risksvc/jurisdiction_store.go:22:func (s *PgJurisdictionStore) GetStatus(ctx context.Context, userID string) (*JurisdictionStatus, error) {
backend/internal/risksvc/jurisdiction_store.go:51:func (s *PgJurisdictionStore) SetKYCStatus(ctx context.Context, userID, status, verifiedBy string) error {
backend/internal/risksvc/jurisdiction_store.go:64:func (s *PgJurisdictionStore) RecordCountry(ctx context.Context, userID, countryCode, source string) error {
backend/internal/risksvc/jurisdiction_store.go:76:func (s *PgJurisdictionStore) IsDisclaimerAccepted(ctx context.Context, userID string) (bool, error) {
backend/internal/risksvc/jurisdiction_store.go:88:func (s *PgJurisdictionStore) AcceptDisclaimer(ctx context.Context, userID, version string) error {
backend/internal/risksvc/kelly_sizer.go:44:func (s *KellyFractionSizer) Name() string { return "kelly_fraction" }
backend/internal/risksvc/kelly_sizer.go:46:func (s *KellyFractionSizer) Size(_ context.Context, req *SizerRequest) (*SizerResult, error) {
backend/internal/risksvc/pipeline.go:96:func (p *SignalPipeline) Process(ctx context.Context, sig *SignalRequest) *SignalResult {
backend/internal/risksvc/platform_aggregator.go:105:func (a *PlatformAggregator) Recalculate() *PlatformExposure {
backend/internal/risksvc/platform_aggregator.go:140:func (a *PlatformAggregator) GetSnapshot() *PlatformExposure {
backend/internal/risksvc/platform_aggregator.go:145:func (a *PlatformAggregator) NetExposureForSymbol(canonical string) decimal.Decimal {
backend/internal/risksvc/platform_aggregator.go:155:func (a *PlatformAggregator) StartRefreshLoop() {
backend/internal/risksvc/platform_aggregator.go:181:func (a *PlatformAggregator) Shutdown() {
backend/internal/risksvc/platform_aggregator.go:76:func (a *PlatformAggregator) UpdatePosition(accountID string, pos *AggregatorPosition) {
backend/internal/risksvc/platform_aggregator.go:88:func (a *PlatformAggregator) ClearAccount(accountID string) {
backend/internal/risksvc/platform_aggregator.go:96:func (a *PlatformAggregator) SetBrokerLimits(limits map[string]decimal.Decimal) {
backend/internal/risksvc/platform_limits.go:39:func (l *PlatformLimits) Check(exposure *PlatformExposure) *PlatformLimitResult {
backend/internal/risksvc/rules.go:111:func (r *CanonicalAuth) Name() string { return "canonical_auth" }
backend/internal/risksvc/rules.go:112:func (r *CanonicalAuth) Check(_ context.Context, req *CheckRequest) *CheckResult {
backend/internal/risksvc/rules.go:16:func (r *MaxPosition) Name() string { return "max_position" }
backend/internal/risksvc/rules.go:17:func (r *MaxPosition) Check(_ context.Context, req *CheckRequest) *CheckResult {
backend/internal/risksvc/rules.go:34:func (r *DailyLoss) Name() string { return "daily_loss" }
backend/internal/risksvc/rules.go:35:func (r *DailyLoss) Check(_ context.Context, req *CheckRequest) *CheckResult {
backend/internal/risksvc/rules.go:57:func (r *Drawdown) Name() string { return "drawdown" }
backend/internal/risksvc/rules.go:59:func (r *Drawdown) Check(_ context.Context, req *CheckRequest) *CheckResult {
backend/internal/risksvc/rules.go:79:func (r *Session) Name() string { return "session" }
backend/internal/risksvc/rules.go:80:func (r *Session) Check(_ context.Context, req *CheckRequest) *CheckResult {
backend/internal/risksvc/rules.go:92:func (r *Margin) Name() string { return "margin" }
backend/internal/risksvc/rules.go:93:func (r *Margin) Check(_ context.Context, req *CheckRequest) *CheckResult {
backend/internal/risksvc/vol_target_sizer.go:45:func (s *VolTargetSizer) Name() string { return "vol_target" }
backend/internal/risksvc/vol_target_sizer.go:47:func (s *VolTargetSizer) Size(_ context.Context, req *SizerRequest) (*SizerResult, error) {
backend/internal/service/account_crud.go:114:func (s *AccountService) DeleteAccount(ctx context.Context, userID uuid.UUID, id string) error {
backend/internal/service/account_crud.go:19:func (s *AccountService) ListAccounts(ctx context.Context, userID uuid.UUID) ([]AccountDTO, error) {
backend/internal/service/account_crud.go:32:func (s *AccountService) GetAccount(ctx context.Context, userID uuid.UUID, accountID string) (*AccountDTO, error) {
backend/internal/service/account_crud.go:46:func (s *AccountService) BeginTx(ctx context.Context) (pgx.Tx, error) {
backend/internal/service/account_crud.go:52:func (s *AccountService) CreateAccountTx(ctx context.Context, tx pgx.Tx, userID uuid.UUID, login, password, mtType, brokerCompany, brokerServer, brokerHost string) (string, error) {
backend/internal/service/account_crud.go:77:func (s *AccountService) CreateAccount(ctx context.Context, userID uuid.UUID, login, password, mtType, brokerCompany, brokerServer, brokerHost string) (string, error) {
backend/internal/service/account_crud.go:95:func (s *AccountService) UpdateAccount(ctx context.Context, userID uuid.UUID, id, brokerCompany, brokerServer, brokerHost string) error {
backend/internal/service/account_lifecycle.go:128:func (s *AccountService) UpdateBrokerThresholds(ctx context.Context, id string, marginCallPct, stopOutPct decimal.Decimal) error {
backend/internal/service/account_lifecycle.go:139:func (s *AccountService) LogAudit(ctx context.Context, accountID, userID uuid.UUID, action, detail string) {
backend/internal/service/account_lifecycle.go:148:func (s *AccountService) UserOwnsAccount(ctx context.Context, userID, accountID string) (bool, error) {
backend/internal/service/account_lifecycle.go:162:func (s *AccountService) DisconnectAccountByID(ctx context.Context, accountID string) error {
backend/internal/service/account_lifecycle.go:173:func (s *AccountService) SetStatus(ctx context.Context, userID uuid.UUID, id string, status AccountStatus) error {
backend/internal/service/account_lifecycle.go:189:func (s *AccountService) CleanupOldSnapshots(ctx context.Context, log *zap.Logger) error {
backend/internal/service/account_lifecycle.go:19:func (s *AccountService) GetAccountCredentials(ctx context.Context, userID uuid.UUID, id string) (*AccountCredentials, error) {
backend/internal/service/account_lifecycle.go:54:func (s *AccountService) UpdateAccountInfoTx(ctx context.Context, p AccountInfoUpdate) error {
backend/internal/service/account_lifecycle.go:69:func (s *AccountService) UpdateAccountInfo(ctx context.Context, p AccountInfoUpdate) error {
backend/internal/service/account_lifecycle.go:84:func (s *AccountService) UpdateAccountMetrics(ctx context.Context, userID uuid.UUID, id string, balance, equity, credit, margin, freeMargin, marginLevel decimal.Decimal) error {
backend/internal/service/account_service.go:59:func (s *AccountService) SetLogger(log *zap.Logger) { s.log = log }
backend/internal/service/account_sync.go:102:func (s *AccountService) GetUserAccountIDs(ctx context.Context, userID string) ([]string, error) {
backend/internal/service/account_sync.go:120:func (s *AccountService) UpdateSummaryCache(userID, accountID string, balance, equity decimal.Decimal, status string) {
backend/internal/service/account_sync.go:133:func (s *AccountService) InvalidateSummaryCache(userID string) {
backend/internal/service/account_sync.go:160:func (s *AccountService) GetUserAccountsSummary(ctx context.Context, userID string) (*UserAccountsSummary, error) {
backend/internal/service/account_sync.go:17:func (s *AccountService) GetDecryptedPassword(ctx context.Context, accountID string) (string, error) {
backend/internal/service/account_sync.go:209:func (s *AccountService) GetUserAccountSnapshots(ctx context.Context, userID string) ([]AccountSnapshot, error) {
backend/internal/service/account_sync.go:247:func (s *AccountService) RecordBalanceSnapshot(ctx context.Context, accountID, userID string, balance, equity, margin, freeMargin decimal.Decimal) error {
backend/internal/service/account_sync.go:41:func (s *AccountService) BackfillPlaintextCredentials(ctx context.Context) (int, error) {
backend/internal/service/account_sync_service.go:115:func (s *AccountSyncService) CheckMarginCall(
backend/internal/service/account_sync_service.go:48:func (s *AccountSyncService) SetNotificationSender(ns *notifpubsub.Sender) { s.notifSender = ns }
backend/internal/service/account_sync_service.go:51:func (s *AccountSyncService) SetScheduleResolver(r mthub.ScheduleResolver) { s.scheduleResolver = r }
backend/internal/service/account_sync_service.go:56:func (s *AccountSyncService) SyncAccountHistory(accountID, userID string) {
backend/internal/service/analytics_cache.go:100:func (c *AnalyticsCache) GetMonthlyDetail(ctx context.Context, accountID string, year, month int32) (*antv1.GetMonthlyDetailResponse, error) {
backend/internal/service/analytics_cache.go:116:func (c *AnalyticsCache) SetMonthlyDetail(ctx context.Context, accountID string, year, month int32, resp *antv1.GetMonthlyDetailResponse) error {
backend/internal/service/analytics_cache.go:127:func (c *AnalyticsCache) Invalidate(ctx context.Context, accountID string) {
backend/internal/service/analytics_cache.go:28:func (c *AnalyticsCache) Get(ctx context.Context, accountID string) (*antv1.AccountAnalyticsResponse, error) {
backend/internal/service/analytics_cache.go:43:func (c *AnalyticsCache) Set(ctx context.Context, accountID string, resp *antv1.AccountAnalyticsResponse) error {
backend/internal/service/analytics_cache.go:52:func (c *AnalyticsCache) GetAttribution(ctx context.Context, accountID string) (*antv1.GetAttributionAnalysisResponse, error) {
backend/internal/service/analytics_cache.go:67:func (c *AnalyticsCache) SetAttribution(ctx context.Context, accountID string, resp *antv1.GetAttributionAnalysisResponse) error {
backend/internal/service/analytics_cache.go:76:func (c *AnalyticsCache) GetRolling(ctx context.Context, accountID string) (*antv1.GetRollingMetricsResponse, error) {
backend/internal/service/analytics_cache.go:91:func (c *AnalyticsCache) SetRolling(ctx context.Context, accountID string, resp *antv1.GetRollingMetricsResponse) error {
backend/internal/service/bound_account_svc.go:35:func (s *BoundAccountService) EnsureBoundAccount(ctx context.Context, userID, accountID uuid.UUID) error {
backend/internal/service/bound_account_svc.go:80:func (s *BoundAccountService) ListBoundAccounts(ctx context.Context, userID uuid.UUID) ([]repository.BoundAccountRow, error) {
backend/internal/service/bound_account_svc.go:85:func (s *BoundAccountService) UnbindAccount(ctx context.Context, userID, accountID uuid.UUID) error {
backend/internal/service/bound_account_svc.go:90:func (s *BoundAccountService) GetAccountLimit(ctx context.Context, userID uuid.UUID) (int, error) {
backend/internal/service/credit_service.go:126:func (s *CreditService) Settle(ctx context.Context, userID uuid.UUID, sessionID, providerID, modelName string, inputTokens, outputTokens int) error {
backend/internal/service/credit_service.go:158:func (s *CreditService) CheckBalance(ctx context.Context, userID uuid.UUID, minCredits decimal.Decimal) error {
backend/internal/service/credit_service.go:235:func (s *CreditService) ReleaseHold(ctx context.Context, userID uuid.UUID, sessionID string) error {
backend/internal/service/credit_service.go:61:func (s *CreditService) RestoreHolds(ctx context.Context) error {
backend/internal/service/credit_service.go:91:func (s *CreditService) PreHold(ctx context.Context, userID uuid.UUID, sessionID string, providerID, modelName string) error {
backend/internal/service/daily_quota.go:107:func (q *DailyQuotaChecker) CheckQuota(ctx context.Context, userID uuid.UUID) error {
backend/internal/service/daily_quota.go:130:func (q *DailyQuotaChecker) Config() DailyQuotaConfig {
backend/internal/service/daily_quota.go:155:func (b *PlatformCostBreaker) SetManagedSettingProvider(p ManagedSettingProvider) {
backend/internal/service/daily_quota.go:173:func (b *PlatformCostBreaker) IsTripped(ctx context.Context) bool {
backend/internal/service/daily_quota.go:226:func (b *PlatformCostBreaker) Threshold() decimal.Decimal {
backend/internal/service/daily_quota.go:84:func (q *DailyQuotaChecker) SetManagedSettingProvider(p ManagedSettingProvider) {
backend/internal/service/deposit_service.go:110:func (s *DepositService) SetCompromisedChecker(c CompromisedChecker) {
backend/internal/service/deposit_service.go:118:func (s *DepositService) GetOrDeriveAddress(ctx context.Context, userID uuid.UUID) (addr, network string, err error) {
backend/internal/service/deposit_service.go:165:func (s *DepositService) ListMyDeposits(ctx context.Context, userID uuid.UUID, page, pageSize int) ([]model.Deposit, int64, error) {
backend/internal/service/deposit_service.go:170:func (s *DepositService) ListManualReviewDeposits(ctx context.Context, page, pageSize int) ([]model.Deposit, int64, error) {
backend/internal/service/deposit_service.go:175:func (s *DepositService) ListDepositAddresses(ctx context.Context, status string, page, pageSize int) ([]model.DepositAddress, int64, int, error) {
backend/internal/service/deposit_service.go:188:func (s *DepositService) ConfirmDeposit(ctx context.Context, userID, addrID uuid.UUID, txHash, amount string, blockNumber int64, confirmations int) error {
backend/internal/service/deposit_service.go:237:func (s *DepositService) MarkAddressReceived(ctx context.Context, addrID uuid.UUID) error {
backend/internal/service/deposit_service.go:77:func (s *DepositService) Xpub() string {
backend/internal/service/deposit_service.go:85:func (s *DepositService) XpubKey() *hdkeychain.ExtendedKey {
backend/internal/service/deposit_service.go:93:func (s *DepositService) UpdateXpub(xpub string) error {
backend/internal/service/email_verification.go:35:func (s *EmailVerificationService) GenerateAndSend(ctx context.Context, userID uuid.UUID, userEmail string) error {
backend/internal/service/email_verification.go:67:func (s *EmailVerificationService) VerifyToken(ctx context.Context, token string) (uuid.UUID, error) {
backend/internal/service/ledger_shipper.go:46:func (s *LedgerShipper) Run(ctx context.Context) {
backend/internal/service/log_service.go:22:func (s *LogService) LogConnection(ctx context.Context, log *model.AccountConnectionLog) error {
backend/internal/service/log_service.go:26:func (s *LogService) GetConnectionLogs(ctx context.Context, userID uuid.UUID, params *model.LogQueryParams) ([]*model.AccountConnectionLog, int, error) {
backend/internal/service/log_service.go:30:func (s *LogService) LogOrder(ctx context.Context, order *model.OrderHistory) error {
backend/internal/service/log_service.go:35:func (s *LogService) UpdateOrderHistoryClose(ctx context.Context, userID, accountID, scheduleID uuid.UUID, ticket int64, closePrice, profit, swap, commission decimal.Decimal, closeTime time.Time) (int64, error) {
backend/internal/service/log_service.go:42:func (s *LogService) GetOrderHistory(ctx context.Context, userID uuid.UUID, params *model.LogQueryParams) ([]*model.OrderHistory, int, error) {
backend/internal/service/log_service.go:46:func (s *LogService) LogOperation(ctx context.Context, log *model.SystemOperationLog) error {
backend/internal/service/log_service.go:50:func (s *LogService) GetOperationLogs(ctx context.Context, userID uuid.UUID, params *model.LogQueryParams) ([]*model.SystemOperationLog, int, error) {
backend/internal/service/log_service.go:54:func (s *LogService) GetScheduleRunLogs(ctx context.Context, userID uuid.UUID, scheduleID uuid.UUID, page, pageSize int) ([]*repository.ScheduleRunLogRow, int, error) {
backend/internal/service/log_service.go:58:func (s *LogService) GetAllLogs(ctx context.Context, userID uuid.UUID, params *model.LogQueryParams) (map[string]interface{}, error) {
backend/internal/service/platform_service.go:109:func (s *PlatformService) IsAdmin(ctx context.Context, userID uuid.UUID) (bool, error) {
backend/internal/service/platform_service.go:118:func (s *PlatformService) UserOwnsAccount(ctx context.Context, userID, accountID string) (bool, error) {
backend/internal/service/platform_service.go:123:func (s *PlatformService) GetAccount(ctx context.Context, userID uuid.UUID, accountID string) (*AccountDTO, error) {
backend/internal/service/platform_service.go:128:func (s *PlatformService) GetAccountBroker(ctx context.Context, accountID string) (string, error) {
backend/internal/service/platform_service.go:139:func (s *PlatformService) ResolveSymbol(ctx context.Context, accountID, rawSymbol string) string {
backend/internal/service/platform_service.go:164:func (s *PlatformService) GetUserAccountIDs(ctx context.Context, userID string) ([]string, error) {
backend/internal/service/platform_service.go:169:func (s *PlatformService) GetUserAccountSnapshots(ctx context.Context, userID string) ([]AccountSnapshot, error) {
backend/internal/service/platform_service.go:174:func (s *PlatformService) GetUserAccountsSummary(ctx context.Context, userID string) (*UserAccountsSummary, error) {
backend/internal/service/platform_service.go:179:func (s *PlatformService) GetUserEmail(ctx context.Context, userID string) (string, error) {
backend/internal/service/platform_service.go:29:func (s *PlatformService) SetLogger(log *zap.Logger) { s.log = log }
backend/internal/service/platform_service.go:42:func (s *PlatformService) ListStrategies(ctx context.Context, userID string) ([]Strategy, error) {
backend/internal/service/platform_service.go:79:func (s *PlatformService) ListSubscriptions(ctx context.Context, userID string) ([]UserSubscription, error) {
backend/internal/service/quota_checker.go:107:func (q *QuotaChecker) CheckStrategyLimit(userID uuid.UUID, currentCount int) bool {
backend/internal/service/quota_checker.go:116:func (q *QuotaChecker) CheckBacktestDailyLimit(userID uuid.UUID, todayCount int) bool {
backend/internal/service/quota_checker.go:125:func (q *QuotaChecker) CheckLiveStrategyLimit(userID uuid.UUID, currentLive int) bool {
backend/internal/service/quota_checker.go:134:func (q *QuotaChecker) CheckSymbolLimit(userID uuid.UUID, currentSymbols int) bool {
backend/internal/service/quota_checker.go:143:func (q *QuotaChecker) CheckAccountLimit(userID uuid.UUID, currentCount int) bool {
backend/internal/service/quota_checker.go:152:func (q *QuotaChecker) GetCapabilityTier(userID uuid.UUID) int {
backend/internal/service/quota_checker.go:158:func (q *QuotaChecker) SetPgListen(l *pglisten.Listener) {
backend/internal/service/quota_checker.go:165:func (q *QuotaChecker) StartRefreshLoop(ctx context.Context) {
backend/internal/service/quota_checker.go:40:func (q *QuotaChecker) LoadAll(ctx context.Context) error {
backend/internal/service/quota_checker.go:83:func (q *QuotaChecker) GetPlan(userID uuid.UUID) *model.SubscriptionPlan {
backend/internal/service/quota_checker.go:94:func (q *QuotaChecker) CheckAITokenQuota(userID uuid.UUID, usedThisMonth int) int {
backend/internal/service/registration_service.go:152:func (e *errEmailAlreadyRegistered) Error() string { return "email already registered" }
backend/internal/service/registration_service.go:61:func (s *RegistrationService) SetEmailVerification(ev EmailVerificationSender) {
backend/internal/service/registration_service.go:66:func (s *RegistrationService) SetSubscriptionEnsurer(se SubscriptionEnsurer) {
backend/internal/service/registration_service.go:72:func (s *RegistrationService) RegisterUser(ctx context.Context, email, password, nickname string) (*model.User, string, error) {
backend/internal/service/schedule_svc.go:122:func (s *StrategySvc) UpdateSchedule(ctx context.Context, r *ScheduleRow) error {
backend/internal/service/schedule_svc.go:144:func (s *StrategySvc) DeleteSchedule(ctx context.Context, id, userID uuid.UUID) error {
backend/internal/service/schedule_svc.go:156:func (s *StrategySvc) SetScheduleActive(ctx context.Context, id, userID uuid.UUID, active bool) error {
backend/internal/service/schedule_svc.go:45:func (s *StrategySvc) ListSchedules(ctx context.Context, userID uuid.UUID) ([]ScheduleRow, error) {
backend/internal/service/schedule_svc.go:58:func (s *StrategySvc) GetSchedule(ctx context.Context, id, userID uuid.UUID) (*ScheduleRow, error) {
backend/internal/service/schedule_svc.go:78:func (s *StrategySvc) CreateSchedule(ctx context.Context, r *ScheduleRow) error {
backend/internal/service/signal_svc.go:31:func (s *StrategySvc) ListSignals(ctx context.Context, userID, accountID uuid.UUID, status string) ([]SignalRow, error) {
backend/internal/service/signal_svc.go:51:func (s *StrategySvc) GetSignal(ctx context.Context, id, userID uuid.UUID) (*SignalRow, error) {
backend/internal/service/signal_svc.go:67:func (s *StrategySvc) ExecuteSignal(ctx context.Context, signalID, userID uuid.UUID) (*SignalRow, error) {
backend/internal/service/signal_svc.go:82:func (s *StrategySvc) ConfirmSignal(ctx context.Context, signalID, userID uuid.UUID) error {
backend/internal/service/signal_svc.go:96:func (s *StrategySvc) CancelSignal(ctx context.Context, signalID, userID uuid.UUID) error {
backend/internal/service/strategy_svc.go:25:func (s *StrategySvc) DB() *pgxpool.Pool { return s.pg }
backend/internal/service/subscription_renewal.go:18:func (s *SubscriptionService) StartPlatformRenewalLoop(ctx context.Context) {
backend/internal/service/subscription_service.go:217:func (s *SubscriptionService) CancelSubscription(ctx context.Context, userID uuid.UUID) error {
backend/internal/service/subscription_service.go:229:func (s *SubscriptionService) ChangePlan(ctx context.Context, userID uuid.UUID, newPlanName, billingCycle string) (*SubscribeResult, error) {
backend/internal/service/subscription_service.go:369:func (s *SubscriptionService) GetMySubscription(ctx context.Context, userID uuid.UUID) (*model.UserPlatformSubscription, *model.SubscriptionPlan, error) {
backend/internal/service/subscription_service.go:42:func (s *SubscriptionService) SetUsageRepos(tokenUsageRepo *repository.AITokenUsageRepository, runRepo *repository.StrategyRunRepository) {
backend/internal/service/subscription_service.go:57:func (s *SubscriptionService) Subscribe(ctx context.Context, userID uuid.UUID, planName, billingCycle string, autoRenew bool) (*SubscribeResult, error) {
backend/internal/service/subscription_service_proto.go:106:func (s *SubscriptionService) EnsureFreeSubscription(ctx context.Context, userID uuid.UUID) error {
backend/internal/service/subscription_service_proto.go:119:func (s *SubscriptionService) GetUsageSummaryProto(ctx context.Context, userID uuid.UUID) (*antv1.UsageSummary, *antv1.Plan, error) {
backend/internal/service/subscription_service_proto.go:25:func (s *SubscriptionService) ListPlansProto(ctx context.Context) ([]*antv1.Plan, error) {
backend/internal/service/subscription_service_proto.go:38:func (s *SubscriptionService) GetMySubscriptionProto(ctx context.Context, userID uuid.UUID) (*UserSubscriptionInfo, error) {
backend/internal/service/systemai/chat.go:183:func (s *Service) ChatCompletion(
backend/internal/service/systemai/chat.go:196:func (s *Service) ChatCompletionWithUsage(
backend/internal/service/systemai/chat_failover.go:26:func (s *Service) SetCircuitBreakerDB(db cbExecutor) {
backend/internal/service/systemai/chat_failover.go:82:func (e *failoverErr) Error() string { return e.msg }
backend/internal/service/systemai/chat_stream.go:17:func (s *Service) ChatCompletionStream(
backend/internal/service/systemai/chat_stream.go:29:func (s *Service) ChatCompletionStreamWithTools(
backend/internal/service/systemai/service.go:107:func (s *Service) SetWalletChecker(fn func(ctx context.Context, userID uuid.UUID) (int, error)) {
backend/internal/service/systemai/service.go:113:func (s *Service) SetPostCallBiller(fn PostCallBiller) {
backend/internal/service/systemai/service.go:118:func (s *Service) SetGatewayProviderRepo(repo *repository.SystemAIProviderRepository) {
backend/internal/service/systemai/service.go:124:func (s *Service) SetCostBreaker(cb CostBreakerChecker) {
backend/internal/service/systemai/service.go:131:func (s *Service) SetModelFilter(fn func(ctx context.Context, userID uuid.UUID, model string) bool) {
backend/internal/service/systemai/service.go:156:func (s *Service) EnsureSeed(ctx context.Context, userID uuid.UUID) error {
backend/internal/service/systemai/service.go:186:func (s *Service) List(ctx context.Context, userID uuid.UUID) ([]*repository.SystemAIConfigRow, error) {
backend/internal/service/systemai/service.go:193:func (s *Service) Get(ctx context.Context, userID uuid.UUID, providerID string) (*repository.SystemAIConfigRow, error) {
backend/internal/service/systemai/service.go:197:func (s *Service) UpdateConfig(ctx context.Context, row *repository.SystemAIConfigRow, updatedBy string) error {
backend/internal/service/systemai/service.go:204:func (s *Service) GetAIPrimary(ctx context.Context, userID uuid.UUID) (providerID, model string, err error) {
backend/internal/service/systemai/service.go:211:func (s *Service) SetAIPrimary(ctx context.Context, userID uuid.UUID, providerID, defaultModel string) error {
backend/internal/service/systemai/service.go:221:func (s *Service) UpdateSecret(ctx context.Context, userID uuid.UUID, providerID, secret, updatedBy string) error {
backend/internal/service/systemai/service.go:251:func (s *Service) GetSecret(ctx context.Context, userID uuid.UUID, providerID string) (string, error) {
backend/internal/service/systemai/service.go:272:func (s *Service) DiscoverModels(ctx context.Context, userID uuid.UUID, providerID string) ([]string, error) {
backend/internal/service/systemai/service.go:87:func (s *Service) SetLogger(l *zap.Logger) { s.log = l }
backend/internal/service/systemai/service.go:90:func (s *Service) SetUserRepo(r *repository.UserRepository) {
backend/internal/service/systemai/session_quota.go:24:func (sc *sessionCounter) AddTokens(n int) {
backend/internal/service/systemai/session_quota.go:31:func (sc *sessionCounter) Total() int {
backend/internal/service/template_svc.go:105:func (s *StrategySvc) DeleteTemplate(ctx context.Context, id, userID uuid.UUID) error {
backend/internal/service/template_svc.go:116:func (s *StrategySvc) UnpublishUserTemplate(ctx context.Context, id, userID uuid.UUID) error {
backend/internal/service/template_svc.go:129:func (s *StrategySvc) SetTemplateStatus(ctx context.Context, id, userID uuid.UUID, status string) error {
backend/internal/service/template_svc.go:172:func (s *StrategySvc) ListStrategyCards(ctx context.Context, userID uuid.UUID, params ListStrategyCardsParams) ([]StrategyCardRow, int, error) {
backend/internal/service/template_svc.go:40:func (s *StrategySvc) ListTemplates(ctx context.Context, userID uuid.UUID) ([]TemplateRow, error) {
backend/internal/service/template_svc.go:51:func (s *StrategySvc) GetTemplate(ctx context.Context, id, userID uuid.UUID) (*TemplateRow, error) {
backend/internal/service/template_svc.go:66:func (s *StrategySvc) CreateTemplate(ctx context.Context, t *TemplateRow) error {
backend/internal/service/template_svc.go:90:func (s *StrategySvc) UpdateTemplate(ctx context.Context, t *TemplateRow) error {
backend/internal/service/template_svc_admin.go:133:func (s *StrategySvc) DeleteSystemStrategy(ctx context.Context, id uuid.UUID) error {
backend/internal/service/template_svc_admin.go:14:func (s *StrategySvc) GetTemplateDetail(ctx context.Context, id uuid.UUID) (*TemplateRow, string, error) {
backend/internal/service/template_svc_admin.go:175:func (s *StrategySvc) ListAllStrategies(ctx context.Context, params ListAllStrategiesParams) ([]AllStrategyRow, int32, error) {
backend/internal/service/template_svc_admin.go:249:func (s *StrategySvc) FlagTemplate(ctx context.Context, id uuid.UUID, reason string, adminID uuid.UUID) error {
backend/internal/service/template_svc_admin.go:263:func (s *StrategySvc) UnflagTemplate(ctx context.Context, id uuid.UUID) error {
backend/internal/service/template_svc_admin.go:276:func (s *StrategySvc) UnpublishTemplate(ctx context.Context, id uuid.UUID) error {
backend/internal/service/template_svc_admin.go:288:func (s *StrategySvc) PublishTemplate(ctx context.Context, id uuid.UUID) error {
backend/internal/service/template_svc_admin.go:300:func (s *StrategySvc) DisableTemplate(ctx context.Context, id uuid.UUID) error {
backend/internal/service/template_svc_admin.go:318:func (s *StrategySvc) EnableTemplate(ctx context.Context, id uuid.UUID) error {
backend/internal/service/template_svc_admin.go:330:func (s *StrategySvc) ArchiveTemplate(ctx context.Context, id uuid.UUID) error {
backend/internal/service/template_svc_admin.go:51:func (s *StrategySvc) ListSystemStrategies(ctx context.Context) ([]SystemStrategyRow, error) {
backend/internal/service/template_svc_admin.go:70:func (s *StrategySvc) CreateSystemStrategy(ctx context.Context, name, description, code string, tags []string) (*TemplateRow, error) {
backend/internal/service/template_svc_admin.go:98:func (s *StrategySvc) UpdateSystemStrategy(ctx context.Context, id uuid.UUID, name, description, code *string, tags []string) (*TemplateRow, error) {
backend/internal/service/user/account_number.go:221:func (s *AccountNumberService) AssignAccountNumber(ctx context.Context, num string) error {
backend/internal/service/user/account_number.go:236:func (s *AccountNumberService) SetAccountNumber(ctx context.Context, userID, num string) error {
backend/internal/service/user/account_number.go:43:func (s *AccountNumberService) GenerateAccountNumber(ctx context.Context) (string, error) {
backend/internal/service/user/account_number.go:85:func (s *AccountNumberService) IsAccountNumberAvailable(ctx context.Context, num string) (bool, error) {
backend/internal/service/user/account_number.go:91:func (s *AccountNumberService) IsAccountNumberAvailableExcluding(ctx context.Context, num, excludeUserID string) (bool, error) {
backend/internal/service/user_deletion_service.go:168:func (s *UserDeletionService) RestoreUser(ctx context.Context, actorID, targetID uuid.UUID) error {
backend/internal/service/user_deletion_service.go:35:func (s *UserDeletionService) SoftDeleteUser(ctx context.Context, actorID, targetID uuid.UUID) error {
backend/internal/service/user_deletion_service.go:86:func (s *UserDeletionService) SoftDeleteUsers(ctx context.Context, actorID uuid.UUID, ids []string) (deleted int64, failed int, errors []string) {
backend/internal/service/wallet_service.go:101:func (s *WalletService) CancelWithdrawal(ctx context.Context, userID uuid.UUID, amount, idemKey string) (*model.Wallet, error) {
backend/internal/service/wallet_service.go:145:func (e *walletNotFoundError) Error() string { return "wallet not found" }
backend/internal/service/wallet_service.go:153:func (e *InsufficientBalanceError) Error() string {
backend/internal/service/wallet_service.go:29:func (s *WalletService) Repo() *repository.WalletRepository { return s.repo }
backend/internal/service/wallet_service.go:32:func (s *WalletService) GetOrCreateWallet(ctx context.Context, userID uuid.UUID) (*model.Wallet, error) {
backend/internal/service/wallet_service.go:44:func (s *WalletService) CreateWallet(ctx context.Context, userID uuid.UUID) (*model.Wallet, error) {
backend/internal/service/wallet_service.go:51:func (s *WalletService) AdjustBalance(ctx context.Context, userID uuid.UUID, amount, txType, description string, operatorID *uuid.UUID, idemKey string) (*model.Wallet, error) {
backend/internal/service/wallet_service.go:85:func (s *WalletService) ListTransactions(ctx context.Context, userID uuid.UUID, page, pageSize int) ([]model.WalletTransaction, int64, error) {
backend/internal/service/wallet_service.go:91:func (s *WalletService) FreezeForWithdrawal(ctx context.Context, userID uuid.UUID, amount, idemKey string) (*model.Wallet, error) {
backend/internal/service/wallet_service.go:96:func (s *WalletService) CompleteWithdrawal(ctx context.Context, userID uuid.UUID, amount, idemKey string) (*model.Wallet, error) {
backend/internal/service/withdrawal_builder.go:69:func (b *WithdrawalBuilder) BuildPendingWithdrawals(ctx context.Context) error {
```

## Handler 注册（已接进生产路由 = 非 shelf-ware）

> 在此列表 = 真正可被调用；只在某 *_test.go 出现而不在此 = 货架闲置（shelf-ware）。

```
backend/cmd/server/handlers.go:167:	mux.Handle(antv1c.NewExecutionAlgoServiceHandler(algoServer, withSency(d.OtelInterceptor, d.AuthInterceptor)))
backend/cmd/server/handlers.go:189:	mux.Handle(antv1c.NewStreamServiceHandler(streamServer, withSency(d.OtelInterceptor, d.AuthInterceptor)))
backend/cmd/server/handlers.go:253:	mux.Handle(antv1c.NewSubscriptionServiceHandler(subscriptionServer, withSency(otel, auth)))
backend/cmd/server/handlers.go:261:	mux.Handle(antv1c.NewEconomicDataServiceHandler(economicDataServer, withSency(otel, auth)))
backend/cmd/server/handlers.go:264:	mux.Handle(antv1c.NewJobServiceHandler(jobServer, withSency(otel, auth)))
backend/cmd/server/handlers.go:266:	mux.Handle(antv1c.NewLogServiceHandler(logServiceServer, withSency(otel, auth)))
backend/cmd/server/handlers.go:269:	mux.Handle(antv1c.NewNotificationServiceHandler(notifServer, withSency(otel, auth)))
backend/cmd/server/handlers.go:96:	mux.Handle(antv1c.NewMtHubServiceHandler(mthubServer, withSency(d.OtelInterceptor, d.AuthInterceptor)))
backend/cmd/server/handlers_admin.go:49:	mux.Handle(antv1c.NewAdminTradingServiceHandler(adminTradingServer, withSency(ic.otel, ic.auth, ic.admin)))
backend/cmd/server/handlers_admin.go:52:	mux.Handle(antv1c.NewAdminConfigServiceHandler(adminConfigServer, withSency(ic.otel, ic.auth, ic.admin)))
backend/cmd/server/handlers_admin.go:55:	mux.Handle(antv1c.NewAdminLogServiceHandler(adminLogServer, withSency(ic.otel, ic.auth, ic.admin)))
backend/cmd/server/handlers_admin.go:58:	mux.Handle(antv1c.NewAdminAccountServiceHandler(adminAccountServer, withSency(ic.otel, ic.auth, ic.admin)))
backend/cmd/server/handlers_admin.go:62:	mux.Handle(antv1c.NewAdminUserServiceHandler(adminUserServer, withSency(ic.otel, ic.auth, ic.admin)))
backend/cmd/server/handlers_admin.go:65:	mux.Handle(antv1c.NewAdminSystemServiceHandler(adminSystemServer, withSency(ic.otel, ic.auth, ic.admin)))
backend/cmd/server/handlers_admin.go:68:	mux.Handle(antv1c.NewAdminStrategyServiceHandler(adminStrategyServer, withSency(ic.otel, ic.auth, ic.admin)))
backend/cmd/server/handlers_admin.go:71:	mux.Handle(antv1c.NewAdminJurisdictionServiceHandler(adminJurisdictionServer, withSency(ic.otel, ic.auth, ic.admin)))
backend/cmd/server/handlers_admin.go:75:	mux.Handle(antv1c.NewAdminBillingServiceHandler(adminBillingServer, withSency(ic.otel, ic.auth, ic.admin)))
backend/cmd/server/handlers_admin.go:80:		mux.Handle(antv1c.NewAdminAgentSettingsServiceHandler(adminAgentSettingsServer, withSency(ic.otel, ic.auth, ic.admin)))
backend/cmd/server/handlers_admin.go:84:		mux.Handle(antv1c.NewAgentHooksServiceHandler(agentHooksServer, withSency(ic.otel, ic.auth, ic.admin)))
backend/cmd/server/handlers_admin.go:89:	mux.Handle(antv1c.NewAdminMonitorServiceHandler(adminMonitorServer, withSency(ic.otel, ic.auth, ic.admin)))
backend/cmd/server/handlers_admin.go:93:	mux.Handle(antv1c.NewPlatformHealthServiceHandler(platformHealthServer, withSency(ic.otel, ic.auth, ic.admin)))
backend/cmd/server/handlers_ai.go:130:	mux.Handle(antv1c.NewCodeAssistServiceHandler(codeAssistServer, withSency(p.OtelInterceptor, p.AuthInterceptor)))
backend/cmd/server/handlers_ai.go:132:	mux.Handle(antv1c.NewSystemAIServiceHandler(systemAIServer, withSency(p.OtelInterceptor, p.AuthInterceptor)))
backend/cmd/server/handlers_ai.go:134:	mux.Handle(antv1c.NewAIPrimaryServiceHandler(aiPrimaryServer, withSency(p.OtelInterceptor, p.AuthInterceptor)))
backend/cmd/server/handlers_ai.go:136:	mux.Handle(antv1c.NewBacktestTradesServiceHandler(backtestTradesServer, withSency(p.OtelInterceptor, p.AuthInterceptor)))
backend/cmd/server/handlers_ai.go:138:	mux.Handle(antv1c.NewGateServiceHandler(gateEvalServer, withSency(p.OtelInterceptor, p.AuthInterceptor)))
backend/cmd/server/handlers_ai.go:152:	mux.Handle(antv1c.NewStrategyPlanServiceHandler(strategyPlanServer, withSency(p.OtelInterceptor, p.AuthInterceptor)))
backend/cmd/server/handlers_ai.go:174:	mux.Handle(antv1c.NewCreditServiceHandler(creditServer, withSency(p.OtelInterceptor, p.AuthInterceptor)))
backend/cmd/server/handlers_ai.go:176:	mux.Handle(antv1c.NewAdminCreditServiceHandler(adminCreditServer, withSency(p.OtelInterceptor, p.AuthInterceptor)))
backend/cmd/server/handlers_ai.go:183:	mux.Handle(antv1c.NewAgentGatewayServiceHandler(agentGateway, withSency(p.OtelInterceptor, p.AuthInterceptor)))
backend/cmd/server/handlers_ai.go:83:	mux.Handle(antv1c.NewAIServiceHandler(aiServer, withSency(p.OtelInterceptor, p.AuthInterceptor)))
backend/cmd/server/handlers_ai.go:84:	mux.Handle(antv1c.NewAgentDefinitionServiceHandler(aiServer, withSency(p.OtelInterceptor, p.AuthInterceptor)))
backend/cmd/server/handlers_ai.go:88:	mux.Handle(antv1c.NewAssetAnalysisServiceHandler(assetAnalysisServer, withSency(p.OtelInterceptor, p.AuthInterceptor)))
backend/cmd/server/handlers_ai.go:94:	mux.Handle(antv1c.NewAIGatewayServiceHandler(gatewayServer, withSency(p.OtelInterceptor, p.AuthInterceptor)))
backend/cmd/server/handlers_marketplace.go:33:	mux.Handle(antv1c.NewMarketServiceHandler(mktServer, withSency(otelInterceptor, authInterceptor)))
backend/cmd/server/handlers_marketplace.go:37:	mux.Handle(antv1c.NewMarketplaceServiceHandler(mktplaceHandler, withSency(otelInterceptor, authInterceptor)))
backend/cmd/server/handlers_share.go:31:	mux.Handle(antv1c.NewShareServiceHandler(shareServer, withSency(otelInterceptor, authInterceptor)))
backend/cmd/server/handlers_sre.go:103:	mux.Handle(antv1c.NewStrategyAssetServiceHandler(strategyAssetServer, withSency(p.OtelInterceptor, p.AuthInterceptor)))
backend/cmd/server/handlers_sre.go:105:	mux.Handle(antv1c.NewScheduleHealthServiceHandler(scheduleHealthServer, withSency(p.OtelInterceptor, p.AuthInterceptor)))
backend/cmd/server/handlers_sre.go:107:	mux.Handle(antv1c.NewIndicatorCatalogServiceHandler(indicatorCatalogServer, withSency(p.OtelInterceptor, p.AuthInterceptor)))
backend/cmd/server/handlers_sre.go:74:	mux.Handle(antv1c.NewAdminSREServiceHandler(sreHandler, withSency(p.OtelInterceptor, p.AuthInterceptor)))
backend/cmd/server/handlers_sre.go:78:	mux.Handle(antv1c.NewAnalyticsServiceHandler(analyticsServer, withSency(p.OtelInterceptor, p.AuthInterceptor)))
backend/cmd/server/handlers_sre.go:82:	mux.Handle(antv1c.NewMarketRegimeServiceHandler(marketRegimeServer, withSency(p.OtelInterceptor, p.AuthInterceptor)))
backend/cmd/server/handlers_sre.go:86:	mux.Handle(antv1c.NewStrategyExperimentServiceHandler(strategyExperimentServer, withSency(p.OtelInterceptor, p.AuthInterceptor)))
backend/cmd/server/handlers_strategy_runtime.go:129:	mux.Handle(antv1c.NewStrategyRuntimeServiceHandler(strategyExecServer,
backend/cmd/server/handlers_strategy_runtime.go:145:	mux.Handle(antv1c.NewPaperTradingServiceHandler(paperHandler,
backend/cmd/server/handlers_strategy_runtime.go:192:	mux.Handle(antv1c.NewLiveBacktestDivergenceServiceHandler(divergenceServer,
backend/cmd/server/handlers_strategy_runtime.go:197:	mux.Handle(antv1c.NewWalkForwardServiceHandler(walkForwardServer,
backend/cmd/server/handlers_strategy_runtime.go:241:	mux.Handle(antv1c.NewAutoTradingServiceHandler(autoTradingServer,
backend/cmd/server/handlers_strategy_runtime.go:97:	mux.Handle(antv1c.NewStrategyServiceHandler(strategyServer, withSency(otelInterceptor, authInterceptor)))
backend/cmd/server/handlers_user.go:58:	mux.Handle(antv1c.NewAuthServiceHandler(authServer, withSency(p.OtelInterceptor, p.RateLimitInterceptor, p.AuthInterceptor)))
backend/cmd/server/handlers_user.go:71:	mux.Handle(antv1c.NewWalletServiceHandler(walletServer, withSency(otelInterceptor, authInterceptor)))
backend/cmd/server/handlers_user.go:86:	mux.Handle(antv1c.NewDepositServiceHandler(depositServer, withSency(otelInterceptor, authInterceptor)))
```
