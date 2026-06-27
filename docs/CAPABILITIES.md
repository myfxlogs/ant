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

_最后生成：2026-06-27 09:26 UTC。运行 `bash scripts/gen_capability_map.sh` 刷新。_

## 符号索引（扁平 symbol → file:line，grep 友好）

> 查询方式：`bash scripts/cap.sh <动词/别名/符号>`（只返回命中行，token 有上界）。**禁止整篇 Read 本文件。**

```
AcceptDisclaimer	backend/internal/risksvc/jurisdiction_store.go:88
AcceptDisclaimerAt	backend/internal/risksvc/jurisdiction_store.go:137
Account	reference/grpc/mt5.proto:58
AccountSummary	reference/grpc/mt4.proto:60
AccountSummary	reference/grpc/mt5.proto:64
ActivateCanary	backend/internal/risk/canary.go:172
ActiveAccountIDs	backend/internal/mthub/types.go:101
AddAccount	backend/internal/risk/canary.go:127
AddGateRule	backend/internal/connect/strategy/strategy_execution_handler.go:83
AddRule	backend/internal/risk/gate.go:179
AddSanctionedCountry	proto/ant/v1/admin_jurisdiction.proto:13
AdjustBalance	backend/internal/service/wallet_service.go:46
AdjustBalance	proto/ant/v1/wallet.proto:11
Allocate	backend/internal/risksvc/block_allocator.go:103
Allocate	backend/internal/risksvc/block_allocator.go:27
Allocate	backend/internal/risksvc/block_allocator.go:67
AllowedLotSize	backend/internal/risk/canary.go:153
AnalyzeAsset	proto/ant/v1/asset_analysis.proto:14
AnalyzeImportCode	backend/internal/connect/strategy/strategy_execution_handler.go:228
AnalyzeImportCode	proto/ant/v1/strategy_runtime.proto:33
AnalyzePlan	proto/ant/v1/strategy_execution.proto:11
ApplyEvent	backend/internal/mthub/state_cache.go:117
ArchiveStrategy	proto/ant/v1/admin_strategy.proto:29
ArchiveTemplate	backend/internal/service/template_svc_admin.go:330
AssignAccountNumber	backend/internal/service/account_number.go:215
Backtest	backend/internal/connect/strategy/strategy_execution_handler.go:162
Backtest	proto/ant/v1/strategy_runtime.proto:17
BatchSetAgents	proto/ant/v1/ai.proto:20
BatchSetAgents	proto/ant/v1/ai_agent.proto:9
BeginTx	backend/internal/service/account_service.go:132
BrokerRegistry	backend/internal/mthub/service.go:102
CalculatePositionSize	proto/ant/v1/auto_trading.proto:21
CanAccept	backend/internal/mthub/reconcile_gate.go:38
CancelAlgo	proto/ant/v1/execution_algo.proto:20
CancelBacktestRun	backend/internal/connect/strategy/strategy_backtest_crud.go:198
CancelBacktestRun	proto/ant/v1/strategy_runtime.proto:22
CancelJob	proto/ant/v1/job.proto:12
CancelPaperOrder	backend/internal/paper/engine.go:163
CancelSignal	backend/internal/connect/strategy/strategy_signals.go:86
CancelSignal	backend/internal/service/signal_svc.go:95
CancelSignal	proto/ant/v1/strategy.proto:34
CancelStrategyExperiment	backend/internal/connect/strategy/strategy_experiment_handler.go:164
CancelStrategyExperiment	proto/ant/v1/strategy_experiment.proto:14
CancelTemplateDraft	backend/internal/connect/strategy/strategy_handler.go:42
CancelTemplateDraft	proto/ant/v1/strategy.proto:20
ChangePassword	reference/grpc/mt5.proto:231
Chat	proto/ant/v1/ai.proto:11
ChatCompletion	backend/internal/connect/strategy/ai_proposer_adapter.go:22
ChatCompletion	backend/internal/service/systemai/chat.go:117
ChatCompletionStream	backend/internal/service/systemai/chat_stream.go:15
ChatCompletionWithUsage	backend/internal/service/systemai/chat.go:130
ChatStream	proto/ant/v1/ai.proto:13
Check	backend/internal/risk/rules.go:114
Check	backend/internal/risk/rules.go:135
Check	backend/internal/risk/rules.go:157
Check	backend/internal/risk/rules.go:181
Check	backend/internal/risk/rules.go:205
Check	backend/internal/risk/rules.go:241
Check	backend/internal/risk/rules.go:284
Check	backend/internal/risk/rules.go:44
Check	backend/internal/risk/rules.go:64
Check	backend/internal/risk/rules.go:85
Check	backend/internal/risk/rules_risksvc.go:104
Check	backend/internal/risk/rules_risksvc.go:174
Check	backend/internal/risk/rules_risksvc.go:34
Check	backend/internal/risk/rules_risksvc.go:70
Check	backend/internal/risk/rule_user_config.go:33
Check	backend/internal/risksvc/hardlimit.go:109
Check	backend/internal/risksvc/hardlimit.go:132
Check	backend/internal/risksvc/hardlimit.go:56
Check	backend/internal/risksvc/hardlimit.go:74
Check	backend/internal/risksvc/jurisdiction.go:91
Check	backend/internal/risksvc/platform_limits.go:35
Check	backend/internal/risksvc/rules.go:103
Check	backend/internal/risksvc/rules.go:14
Check	backend/internal/risksvc/rules.go:31
Check	backend/internal/risksvc/rules.go:52
Check	backend/internal/risksvc/rules.go:71
Check	backend/internal/risksvc/rules.go:84
CheckAndSet	backend/internal/mthub/idempotency.go:151
CheckAndSet	backend/internal/mthub/idempotency.go:53
CheckAssetUpdate	backend/internal/connect/strategy/strategy_asset_handler.go:172
CheckAssetUpdate	proto/ant/v1/strategy_asset.proto:15
CheckConnect	reference/grpc/mt4.proto:45
CheckConnect	reference/grpc/mt5.proto:43
CheckMarginCall	backend/internal/service/account_sync.go:31
CheckRiskLimits	proto/ant/v1/auto_trading.proto:20
Cleanup	backend/internal/connect/strategy/go_executor.go:146
CleanupOldSnapshots	backend/internal/service/account_lifecycle.go:205
ClearAccount	backend/internal/risksvc/platform_aggregator.go:85
CloneStrategyAsset	backend/internal/connect/strategy/strategy_asset_handler.go:146
CloneStrategyAsset	proto/ant/v1/strategy_asset.proto:14
ClosedOrders	reference/grpc/mt4.proto:151
CloseOrder	backend/internal/mthub/service_orders_close.go:17
CloseOrder	proto/ant/v1/mthub_service.proto:8
ClosePaperOrder	backend/internal/paper/engine.go:121
CloseSession	backend/internal/mthub/types.go:93
CommentOnStrategy	proto/ant/v1/marketplace_service.proto:18
CompileCheck	backend/internal/connect/strategy/go_executor.go:123
Confirm	backend/internal/mthub/idempotency.go:87
ConfirmSignal	backend/internal/connect/strategy/strategy_signals.go:73
ConfirmSignal	backend/internal/service/signal_svc.go:81
ConfirmSignal	proto/ant/v1/strategy.proto:33
Connect	reference/grpc/mt4.proto:17
Connect	reference/grpc/mt5.proto:15
ConnectAccount	backend/internal/service/account_lifecycle.go:15
ConnectAccount	proto/ant/v1/account.proto:19
ConnectEx	reference/grpc/mt4.proto:25
ConnectEx	reference/grpc/mt5.proto:22
ConnectProxy	reference/grpc/mt4.proto:39
ConnectProxy	reference/grpc/mt5.proto:37
Conversate	proto/ant/v1/strategy_execution.proto:14
Count	backend/internal/risksvc/capability.go:155
CountryCode	backend/internal/risksvc/jurisdiction.go:185
CreateAccount	backend/internal/service/account_service.go:155
CreateAccount	proto/ant/v1/account.proto:16
CreateAccountTx	backend/internal/service/account_service.go:137
CreateConversation	proto/ant/v1/ai.proto:16
CreateFrozenBacktestDataset	proto/ant/v1/backtest_dataset.proto:12
CreatePaperAccount	proto/ant/v1/paper_trading.proto:10
CreateSchedule	backend/internal/connect/strategy/strategy_schedules.go:46
CreateSchedule	backend/internal/service/schedule_svc.go:75
CreateSchedule	proto/ant/v1/strategy.proto:24
CreateShareToken	proto/ant/v1/share.proto:9
CreateSystemStrategy	backend/internal/service/template_svc_admin.go:70
CreateSystemStrategy	proto/ant/v1/admin_strategy.proto:14
CreateTemplate	backend/internal/connect/strategy/strategy_template_handlers.go:77
CreateTemplate	backend/internal/service/template_svc.go:60
CreateTemplate	proto/ant/v1/strategy.proto:13
CreateTemplateDraft	backend/internal/connect/strategy/strategy_template_handlers.go:155
CreateTemplateDraft	proto/ant/v1/strategy.proto:17
CreateUser	proto/ant/v1/admin_user.proto:12
CreateWallet	backend/internal/service/wallet_service.go:40
CurrentStage	backend/internal/risk/canary.go:163
DeleteAccount	backend/internal/service/account_service.go:199
DeleteAccount	proto/ant/v1/account.proto:18
DeleteBacktestDataset	proto/ant/v1/backtest_dataset.proto:13
DeleteBacktestRun	backend/internal/connect/strategy/strategy_backtest_crud.go:214
DeleteBacktestRun	proto/ant/v1/strategy_runtime.proto:23
DeleteBacktestRuns	backend/internal/connect/strategy/strategy_backtest_crud.go:232
DeleteBacktestRuns	proto/ant/v1/strategy_runtime.proto:24
DeleteConversation	proto/ant/v1/ai.proto:17
DeleteKey	backend/internal/mthub/idempotency.go:172
DeleteModel	proto/ant/v1/ai_gateway.proto:28
DeleteProvider	proto/ant/v1/ai_gateway.proto:22
DeleteSchedule	backend/internal/connect/strategy/strategy_schedules.go:135
DeleteSchedule	backend/internal/service/schedule_svc.go:136
DeleteSchedule	proto/ant/v1/strategy.proto:26
DeleteSystemStrategy	backend/internal/service/template_svc_admin.go:133
DeleteSystemStrategy	proto/ant/v1/admin_strategy.proto:16
DeleteTemplate	backend/internal/connect/strategy/strategy_template_handlers.go:140
DeleteTemplate	backend/internal/service/template_svc.go:91
DeleteTemplate	proto/ant/v1/strategy.proto:15
DeleteUser	proto/ant/v1/admin_user.proto:14
DeleteUsers	proto/ant/v1/admin_user.proto:15
DetectMarketRegime	proto/ant/v1/market_regime.proto:10
Diagnose	proto/ant/v1/strategy_execution.proto:12
DisableStrategy	proto/ant/v1/admin_strategy.proto:27
DisableTemplate	backend/internal/service/template_svc_admin.go:300
DisableUser	proto/ant/v1/admin_user.proto:16
Disconnect	reference/grpc/mt4.proto:51
Disconnect	reference/grpc/mt5.proto:49
DisconnectAccount	backend/internal/service/account_lifecycle.go:29
DisconnectAccount	proto/ant/v1/account.proto:20
DisconnectAccountByID	backend/internal/service/account_lifecycle.go:42
DiscoverModels	backend/internal/service/systemai/service.go:256
DiscoverSystemAIModels	proto/ant/v1/system_ai.proto:15
DisengageKillSwitch	backend/internal/risk/canary.go:266
DroppedBars	backend/internal/mthub/broker_types.go:153
EnableStrategy	proto/ant/v1/admin_strategy.proto:28
EnableTemplate	backend/internal/service/template_svc_admin.go:318
EnableUser	proto/ant/v1/admin_user.proto:17
EngageKillSwitch	backend/internal/risk/canary.go:251
EnsureSeed	backend/internal/service/systemai/service.go:140
EnsureSession	backend/internal/mthub/types.go:71
EnterAll	backend/internal/mthub/reconcile_gate.go:59
EnterReconciling	backend/internal/mthub/reconcile_gate.go:24
Error	backend/internal/mthub/types.go:115
Error	backend/internal/risksvc/hardlimit.go:181
Error	backend/internal/service/registration_service.go:109
Error	backend/internal/service/systemai/chat_failover.go:83
Error	backend/internal/service/wallet_service.go:86
Error	backend/internal/service/wallet_service.go:94
Estimate	backend/internal/mthub/hub_estimator.go:46
Evaluate	backend/internal/risk/gate.go:114
Evaluate	backend/internal/risksvc/engine.go:27
Evaluate	backend/internal/risksvc/hardlimit.go:165
Events	reference/grpc/mt5.proto:512
Execute	backend/internal/connect/strategy/strategy_execution_handler.go:110
Execute	proto/ant/v1/strategy_runtime.proto:15
ExecuteLive	backend/internal/connect/strategy/strategy_execution_handler.go:183
ExecuteLive	proto/ant/v1/strategy_runtime.proto:28
ExecutePlan	proto/ant/v1/strategy_execution.proto:13
ExecuteSignal	backend/internal/connect/strategy/strategy_signals.go:52
ExecuteSignal	backend/internal/service/signal_svc.go:66
ExecuteSignal	proto/ant/v1/strategy.proto:32
ExplainCode	proto/ant/v1/code_assist.proto:12
Fetch	backend/internal/connect/strategy/data_source.go:57
FlagStrategy	proto/ant/v1/admin_strategy.proto:23
FlagTemplate	backend/internal/service/template_svc_admin.go:249
FreezeAccount	proto/ant/v1/admin_account.proto:11
GenerateAccountNumber	backend/internal/service/account_number.go:45
GenerateImportCode	backend/internal/connect/strategy/strategy_execution_handler.go:277
GenerateImportCode	proto/ant/v1/strategy_runtime.proto:35
GenerateReport	proto/ant/v1/analytics.proto:15
GenerateStrategy	proto/ant/v1/strategy_generation.proto:12
Get	backend/internal/mthub/derived_state.go:74
Get	backend/internal/mthub/types.go:65
Get	backend/internal/risksvc/capability.go:97
Get	backend/internal/service/analytics_cache.go:39
Get	backend/internal/service/systemai/service.go:177
GetAccount	backend/internal/mthub/derived_state.go:81
GetAccount	backend/internal/service/account_service.go:118
GetAccount	backend/internal/service/platform_service.go:123
GetAccount	proto/ant/v1/account.proto:15
GetAccountAnalytics	proto/ant/v1/analytics.proto:8
GetAccountBroker	backend/internal/service/platform_service.go:128
GetAccountCredentials	backend/internal/service/account_lifecycle.go:75
GetAccountState	backend/internal/connect/strategy/account_provider.go:53
GetAccountStatus	proto/ant/v1/mthub_service.proto:14
GetAgentCapabilities	proto/ant/v1/agent_gateway.proto:14
GetAIPrimary	backend/internal/service/systemai/service.go:188
GetAIPrimary	proto/ant/v1/ai_primary.proto:8
GetAlgoStatus	proto/ant/v1/execution_algo.proto:17
GetAllLogs	backend/internal/service/log_service.go:69
GetAttribution	backend/internal/service/analytics_cache.go:63
GetAttributionAnalysis	proto/ant/v1/analytics.proto:13
GetAutoTradingStatus	proto/ant/v1/auto_trading.proto:22
GetBacktestRun	backend/internal/connect/strategy/strategy_backtest_crud.go:74
GetBacktestRun	proto/ant/v1/strategy_runtime.proto:19
GetCanary	proto/ant/v1/admin_sre.proto:10
GetClients	reference/grpc/mt4.proto:217
GetClients	reference/grpc/mt5.proto:367
GetConnectionLogs	backend/internal/service/log_service.go:25
GetConnectionLogs	proto/ant/v1/log.proto:14
GetConversation	proto/ant/v1/ai.proto:15
GetDashboard	proto/ant/v1/admin_user.proto:10
GetDemo	reference/grpc/mt5.proto:374
GetExecutionLogs	backend/internal/service/log_service.go:37
GetExecutionLogs	proto/ant/v1/log.proto:15
GetExperimentCandidate	backend/internal/connect/strategy/strategy_experiment_handler.go:194
GetExperimentCandidate	proto/ant/v1/strategy_experiment.proto:16
GetGlobalSettings	proto/ant/v1/auto_trading.proto:15
GetIndicatorCatalog	proto/ant/v1/indicator_catalog.proto:10
GetJob	proto/ant/v1/job.proto:11
GetJurisdictionStatus	proto/ant/v1/admin_jurisdiction.proto:10
GetKillSwitch	proto/ant/v1/admin_sre.proto:6
GetKlines	proto/ant/v1/market_service.proto:8
GetLogs	reference/grpc/mt4.proto:200
GetLogsByUser	reference/grpc/mt4.proto:207
GetMarketRegime	proto/ant/v1/market_regime.proto:11
GetMe	proto/ant/v1/auth.proto:14
GetMetrics	proto/ant/v1/admin_system.proto:9
GetMonthlyAnalysis	proto/ant/v1/analytics.proto:11
GetMonthlyDetail	backend/internal/service/analytics_cache.go:111
GetMonthlyDetail	proto/ant/v1/analytics.proto:12
GetMonthlyPnL	proto/ant/v1/analytics.proto:10
GetOperationLogs	backend/internal/service/log_service.go:61
GetOperationLogs	proto/ant/v1/log.proto:17
GetOrCreateWallet	backend/internal/service/wallet_service.go:28
GetOrder	backend/internal/mthub/state_cache.go:76
GetOrderHistory	backend/internal/service/log_service.go:53
GetOrderLogHistory	proto/ant/v1/log.proto:16
GetOrdersByAccount	backend/internal/mthub/state_cache.go:83
GetPeakEquity	backend/internal/connect/strategy/account_provider.go:144
GetPosition	backend/internal/mthub/state_cache.go:96
GetPositionsByAccount	backend/internal/mthub/state_cache.go:103
GetPublisherStats	proto/ant/v1/marketplace_service.proto:25
GetQuote	reference/grpc/mt5.proto:138
GetQuoteMany	reference/grpc/mt4.proto:80
GetQuoteMany	reference/grpc/mt5.proto:145
GetRecentTrades	proto/ant/v1/analytics.proto:9
GetRecentTradingLogs	proto/ant/v1/auto_trading.proto:24
GetRiskConfig	proto/ant/v1/auto_trading.proto:18
GetRolling	backend/internal/service/analytics_cache.go:87
GetRollingMetrics	proto/ant/v1/analytics.proto:14
GetSchedule	backend/internal/connect/strategy/strategy_schedules.go:34
GetSchedule	backend/internal/service/schedule_svc.go:55
GetSchedule	proto/ant/v1/strategy.proto:23
GetScheduleHealth	proto/ant/v1/schedule_health.proto:10
GetScheduleRunLogs	backend/internal/service/log_service.go:65
GetScheduleRunLogs	proto/ant/v1/log.proto:18
GetSecret	backend/internal/service/systemai/service.go:235
GetSharedPerformance	proto/ant/v1/share.proto:11
GetSignal	backend/internal/service/signal_svc.go:50
GetSnapshot	backend/internal/risksvc/platform_aggregator.go:138
GetStatus	backend/internal/risksvc/jurisdiction_store.go:22
GetStrategyAsset	backend/internal/connect/strategy/strategy_asset_handler.go:97
GetStrategyAsset	proto/ant/v1/strategy_asset.proto:11
GetStrategyDetail	proto/ant/v1/admin_strategy.proto:20
GetStrategyExperiment	backend/internal/connect/strategy/strategy_experiment_handler.go:137
GetStrategyExperiment	proto/ant/v1/strategy_experiment.proto:12
GetSymbolStats	proto/ant/v1/market_service.proto:9
GetSystemAIConfig	proto/ant/v1/system_ai.proto:12
GetTemplate	backend/internal/connect/strategy/strategy_template_handlers.go:61
GetTemplate	backend/internal/service/template_svc.go:45
GetTemplate	proto/ant/v1/strategy.proto:12
GetTemplateDetail	backend/internal/service/template_svc_admin.go:14
GetTemplates	backend/internal/connect/strategy/strategy_execution_handler.go:177
GetTemplates	proto/ant/v1/strategy_runtime.proto:25
GetTickValueMany	reference/grpc/mt5.proto:216
GetTier	backend/internal/risk/rules_risksvc.go:159
GetTokenUsage	proto/ant/v1/ai_gateway.proto:15
GetTradingLogs	proto/ant/v1/auto_trading.proto:23
GetTradingSummary	proto/ant/v1/admin_trading.proto:8
GetUserAccountIDs	backend/internal/service/account_snapshot.go:59
GetUserAccountIDs	backend/internal/service/platform_service.go:164
GetUserAccountSnapshots	backend/internal/service/account_snapshot.go:77
GetUserAccountSnapshots	backend/internal/service/platform_service.go:169
GetUserAccountsSummary	backend/internal/service/account_snapshot.go:149
GetUserAccountsSummary	backend/internal/service/platform_service.go:174
GetWallet	proto/ant/v1/wallet.proto:9
Groups	reference/grpc/mt4.proto:66
HasOrderType	backend/internal/risksvc/capability.go:61
Health	reference/grpc/mt5.proto:343
HealthCheck	proto/ant/v1/admin_system.proto:8
History	backend/internal/risk/canary.go:308
ImportStrategy	backend/internal/connect/strategy/strategy_execution_handler.go:325
ImportStrategy	proto/ant/v1/strategy_runtime.proto:37
InsertOrder	backend/internal/mthub/oms_writer.go:116
Invalidate	backend/internal/service/analytics_cache.go:138
InvalidateSummaryCache	backend/internal/service/account_snapshot.go:140
IsAccountNumberAvailable	backend/internal/service/account_number.go:87
IsAccountNumberAvailableExcluding	backend/internal/service/account_number.go:93
IsAdmin	backend/internal/service/platform_service.go:109
IsCanaryAccount	backend/internal/risk/canary.go:141
IsDisclaimerAccepted	backend/internal/risksvc/jurisdiction_store.go:76
IsExpired	backend/internal/mthub/types.go:15
IsInvestor	reference/grpc/mt4.proto:174
IsKillSwitchActive	backend/internal/risk/canary.go:281
IsQuestionnaireCompleted	backend/internal/risksvc/jurisdiction_store.go:100
IsQuoteSession	reference/grpc/mt5.proto:204
IsQuoteSessionMany	reference/grpc/mt5.proto:210
IsReconciling	backend/internal/mthub/reconcile_gate.go:45
IsSanctioned	backend/internal/risksvc/jurisdiction_store.go:125
IssueAgentToken	proto/ant/v1/agent_gateway.proto:10
IsTradeSession	reference/grpc/mt5.proto:191
IsTradeSessionMany	reference/grpc/mt5.proto:197
List	backend/internal/service/systemai/service.go:170
ListAccounts	backend/internal/service/account_service.go:105
ListAccounts	proto/ant/v1/account.proto:14
ListAccountsAdmin	proto/ant/v1/admin_account.proto:10
ListAgentAudit	proto/ant/v1/agent_gateway.proto:13
ListAgentDefs	proto/ant/v1/ai_agent.proto:10
ListAgents	proto/ant/v1/ai.proto:19
ListAgentTokens	proto/ant/v1/agent_gateway.proto:11
ListAlgos	proto/ant/v1/execution_algo.proto:23
ListAllStrategies	backend/internal/service/template_svc_admin.go:175
ListAllStrategies	proto/ant/v1/admin_strategy.proto:19
ListAssetClones	backend/internal/connect/strategy/strategy_asset_handler.go:203
ListAssetClones	proto/ant/v1/strategy_asset.proto:17
ListBacktestDatasets	proto/ant/v1/backtest_dataset.proto:11
ListBacktestRuns	backend/internal/connect/strategy/strategy_backtest_crud.go:93
ListBacktestRuns	proto/ant/v1/strategy_runtime.proto:20
ListBacktestRunTrades	backend/internal/connect/strategy/backtest_trades_handler.go:55
ListBacktestRunTrades	proto/ant/v1/backtest_trades.proto:8
ListBreakers	proto/ant/v1/admin_sre.proto:8
ListComments	proto/ant/v1/marketplace_service.proto:19
ListConfigs	proto/ant/v1/admin_config.proto:10
ListConversations	proto/ant/v1/ai.proto:14
ListEconomicCalendarEvents	proto/ant/v1/economic_data.proto:10
ListEconomicIndicators	proto/ant/v1/economic_data.proto:11
ListExperimentCandidates	backend/internal/connect/strategy/strategy_experiment_handler.go:177
ListExperimentCandidates	proto/ant/v1/strategy_experiment.proto:15
ListLogs	proto/ant/v1/admin_log.proto:10
ListModels	proto/ant/v1/ai_gateway.proto:24
ListNotifications	proto/ant/v1/notification_service.proto:10
ListPaperAccounts	proto/ant/v1/paper_trading.proto:11
ListProviders	proto/ant/v1/ai_gateway.proto:18
ListPublished	proto/ant/v1/marketplace_service.proto:13
ListRatings	proto/ant/v1/marketplace_service.proto:17
ListSanctionedCountries	proto/ant/v1/admin_jurisdiction.proto:12
ListSchedules	backend/internal/connect/strategy/strategy_schedules.go:22
ListSchedules	backend/internal/service/schedule_svc.go:42
ListSchedules	proto/ant/v1/strategy.proto:22
ListSignals	backend/internal/connect/strategy/strategy_signals.go:35
ListSignals	backend/internal/service/signal_svc.go:30
ListSignals	proto/ant/v1/strategy.proto:31
ListStrategies	backend/internal/service/platform_service.go:42
ListStrategyAssets	backend/internal/connect/strategy/strategy_asset_handler.go:83
ListStrategyAssets	proto/ant/v1/strategy_asset.proto:10
ListStrategyExperiments	backend/internal/connect/strategy/strategy_experiment_handler.go:150
ListStrategyExperiments	proto/ant/v1/strategy_experiment.proto:13
ListSubscriptions	backend/internal/service/platform_service.go:79
ListSubscriptions	proto/ant/v1/marketplace_service.proto:14
ListSystemAIConfigs	proto/ant/v1/system_ai.proto:11
ListSystemModels	proto/ant/v1/ai_gateway.proto:13
ListSystemStrategies	backend/internal/service/template_svc_admin.go:51
ListSystemStrategies	proto/ant/v1/admin_strategy.proto:13
ListTemplates	backend/internal/connect/strategy/strategy_template_handlers.go:48
ListTemplates	backend/internal/service/template_svc.go:34
ListTemplates	proto/ant/v1/strategy.proto:11
ListTransactions	backend/internal/service/wallet_service.go:77
ListTransactions	proto/ant/v1/wallet.proto:10
ListUsers	proto/ant/v1/admin_user.proto:11
ListUsersByKYCStatus	proto/ant/v1/admin_jurisdiction.proto:15
LoadFromPG	backend/internal/risksvc/capability.go:115
LoadFromRedis	backend/internal/mthub/state_cache.go:186
LogConnection	backend/internal/service/log_service.go:21
LogExecution	backend/internal/service/log_service.go:29
Login	proto/ant/v1/auth.proto:11
LogOperation	backend/internal/service/log_service.go:57
LogOrder	backend/internal/service/log_service.go:41
Logout	proto/ant/v1/auth.proto:12
Mails	reference/grpc/mt5.proto:237
MarkAccountNeedsRebind	backend/internal/service/account_lifecycle.go:64
MarkAllRead	proto/ant/v1/notification_service.proto:12
MarketWatchMany	reference/grpc/mt5.proto:151
MarkRead	proto/ant/v1/notification_service.proto:11
MarkReconciled	backend/internal/mthub/reconcile_gate.go:31
MemorySnapshot	reference/grpc/mt4.proto:209
MemorySnapshot	reference/grpc/mt5.proto:359
MemoryUsage	reference/grpc/mt4.proto:219
ModifyOrder	backend/internal/mthub/service_orders_modify.go:18
ModifyPaperOrder	backend/internal/paper/engine.go:143
Name	backend/internal/connect/strategy/data_source.go:55
Name	backend/internal/risk/rules.go:112
Name	backend/internal/risk/rules.go:133
Name	backend/internal/risk/rules.go:155
Name	backend/internal/risk/rules.go:179
Name	backend/internal/risk/rules.go:203
Name	backend/internal/risk/rules.go:239
Name	backend/internal/risk/rules.go:282
Name	backend/internal/risk/rules.go:42
Name	backend/internal/risk/rules.go:62
Name	backend/internal/risk/rules.go:83
Name	backend/internal/risk/rules_risksvc.go:102
Name	backend/internal/risk/rules_risksvc.go:172
Name	backend/internal/risk/rules_risksvc.go:32
Name	backend/internal/risk/rules_risksvc.go:68
Name	backend/internal/risk/rule_user_config.go:31
Name	backend/internal/risksvc/block_allocator.go:101
Name	backend/internal/risksvc/block_allocator.go:25
Name	backend/internal/risksvc/block_allocator.go:65
Name	backend/internal/risksvc/hardlimit.go:107
Name	backend/internal/risksvc/hardlimit.go:130
Name	backend/internal/risksvc/hardlimit.go:54
Name	backend/internal/risksvc/hardlimit.go:72
Name	backend/internal/risksvc/kelly_sizer.go:42
Name	backend/internal/risksvc/rules.go:102
Name	backend/internal/risksvc/rules.go:13
Name	backend/internal/risksvc/rules.go:30
Name	backend/internal/risksvc/rules.go:50
Name	backend/internal/risksvc/rules.go:70
Name	backend/internal/risksvc/rules.go:83
Name	backend/internal/risksvc/vol_target_sizer.go:43
NetExposureForSymbol	backend/internal/risksvc/platform_aggregator.go:143
Notify	backend/internal/connect/strategy/schedule_engine.go:102
OnBar	backend/internal/connect/strategy/strategy_templates.go:141
OnBar	backend/internal/connect/strategy/strategy_templates.go:234
OnBar	backend/internal/connect/strategy/strategy_templates.go:42
OnDeinit	backend/internal/connect/strategy/strategy_templates.go:104
OnDeinit	backend/internal/connect/strategy/strategy_templates.go:199
OnDeinit	backend/internal/connect/strategy/strategy_templates.go:292
OnInit	backend/internal/connect/strategy/strategy_templates.go:131
OnInit	backend/internal/connect/strategy/strategy_templates.go:225
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
OpenedOrders	backend/internal/mthub/service.go:198
OpenedOrders	proto/ant/v1/mthub_service.proto:9
OpenedOrders	reference/grpc/mt4.proto:86
OpenedOrders	reference/grpc/mt5.proto:72
OpenedOrdersTickets	reference/grpc/mt5.proto:85
OrderClose	reference/grpc/mt4.proto:329
OrderClose	reference/grpc/mt5.proto:503
OrderCloseBy	reference/grpc/mt4.proto:312
OrderDelete	reference/grpc/mt4.proto:319
OrderHistory	backend/internal/mthub/service.go:209
OrderHistory	proto/ant/v1/mthub_service.proto:10
OrderHistory	reference/grpc/mt4.proto:126
OrderHistory	reference/grpc/mt5.proto:95
OrderHistoryPagination	reference/grpc/mt5.proto:118
OrderModify	reference/grpc/mt4.proto:304
OrderModify	reference/grpc/mt5.proto:493
OrderSend	reference/grpc/mt4.proto:294
OrderSend	reference/grpc/mt5.proto:482
PendingOrderHistory	reference/grpc/mt5.proto:104
Ping	reference/grpc/mt4.proto:185
Ping	reference/grpc/mt5.proto:342
PingHost	reference/grpc/mt4.proto:192
PingHost	reference/grpc/mt5.proto:350
PingHostMany	reference/grpc/mt4.proto:194
PingHostMany	reference/grpc/mt5.proto:357
PlaceOrder	backend/internal/mthub/service_orders.go:19
PlaceOrder	proto/ant/v1/mthub_service.proto:7
PlacePaperOrder	backend/internal/paper/engine.go:60
Platform	backend/internal/mthub/service.go:157
PriceHistory	backend/internal/mthub/service.go:227
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
Process	backend/internal/risksvc/pipeline.go:94
PromoteCandidateToDraft	backend/internal/connect/strategy/strategy_experiment_handler.go:207
PromoteCandidateToDraft	proto/ant/v1/strategy_experiment.proto:17
PromoteToFull	backend/internal/risk/canary.go:234
Publish	backend/internal/mthub/broker_types.go:126
Publish	backend/internal/mthub/broker_types.go:201
Publish	backend/internal/mthub/broker_types.go:63
Publish	backend/internal/mthub/trade_event_store.go:95
Publish	backend/internal/mthub/types.go:148
PublishAccountProfit	backend/internal/mthub/service.go:261
PublishAccountStatus	backend/internal/mthub/service.go:137
PublishBar	backend/internal/mthub/service.go:120
PublishEvent	backend/internal/mthub/types.go:190
PublishPositionSnapshot	backend/internal/mthub/service.go:271
PublishStrategy	proto/ant/v1/admin_strategy.proto:26
PublishStrategy	proto/ant/v1/marketplace_service.proto:9
PublishTemplate	backend/internal/service/template_svc_admin.go:288
PublishTemplateDraft	backend/internal/connect/strategy/strategy_template_handlers.go:204
PublishTemplateDraft	proto/ant/v1/strategy.proto:19
PurchaseStrategy	proto/ant/v1/marketplace_service.proto:12
QuestionnaireCompletedAt	backend/internal/risksvc/jurisdiction_store.go:149
Quote	reference/grpc/mt4.proto:73
QuoteHistory	reference/grpc/mt4.proto:136
QuoteHistoryMany	reference/grpc/mt4.proto:145
RateStrategy	proto/ant/v1/marketplace_service.proto:16
Recalculate	backend/internal/risksvc/platform_aggregator.go:102
ReconcileAccount	backend/internal/mthub/reconciliation.go:47
ReconcilingCount	backend/internal/mthub/reconcile_gate.go:52
ReconnectAccount	backend/internal/service/account_lifecycle.go:53
ReconnectAccount	proto/ant/v1/account.proto:21
RecordBalanceSnapshot	backend/internal/service/account_snapshot.go:15
RecordCountry	backend/internal/risksvc/jurisdiction_store.go:64
RecordSuccessfulTrade	backend/internal/risk/canary.go:188
reference/grpc/mt4.proto:362://  rpc OnQuoteHistory (OnQuoteHistoryRequest) returns (OnQuoteHistoryReply);
reference/grpc/mt4.proto:368://  rpc OnDisconnect (OnDisconnectRequest) returns (OnDisconnectReply);
reference/grpc/mt5.proto:229://  rpc ClusterDetails (ClusterDetailsRequest) returns (ClusterDetailsReply);
Refresh	backend/internal/mthub/hub_estimator.go:152
RefreshToken	proto/ant/v1/auth.proto:13
Register	backend/internal/mthub/types.go:38
Register	proto/ant/v1/auth.proto:15
RegisterUser	backend/internal/service/registration_service.go:49
RemoveAccount	backend/internal/risk/canary.go:134
RemoveSanctionedCountry	proto/ant/v1/admin_jurisdiction.proto:14
RemoveSession	backend/internal/mthub/types.go:86
RequestQuoteHistory	reference/grpc/mt4.proto:161
RequiredMargin	reference/grpc/mt5.proto:247
ResetBreaker	proto/ant/v1/admin_sre.proto:9
ResetPeakEquity	backend/internal/connect/strategy/account_provider.go:151
ResetUserPassword	proto/ant/v1/admin_user.proto:18
ResolveSession	proto/ant/v1/ai.proto:21
ResolveSymbol	backend/internal/service/platform_service.go:139
RestoreUser	backend/internal/service/user_deletion_service.go:167
RestoreUser	proto/ant/v1/admin_user.proto:19
ReviewStrategyAsset	backend/internal/connect/strategy/strategy_asset_handler.go:134
ReviewStrategyAsset	proto/ant/v1/strategy_asset.proto:13
ReviseCode	proto/ant/v1/code_assist.proto:10
ReviseCodeStream	proto/ant/v1/code_assist.proto:11
RevokeAgentToken	proto/ant/v1/agent_gateway.proto:12
Rollback	backend/internal/risk/canary.go:290
Rules	backend/internal/risk/gate.go:186
Rules	backend/internal/risksvc/engine.go:49
Run	backend/internal/connect/strategy/go_executor.go:37
RunBacktest	backend/internal/connect/strategy/go_executor.go:75
RunBacktest	backend/internal/connect/strategy/strategy_signals.go:16
RunBacktest	proto/ant/v1/backtest_service.proto:12
RunBacktest	proto/ant/v1/strategy.proto:29
RunEvaluation	proto/ant/v1/ai_gate.proto:11
RunLive	backend/internal/connect/strategy/go_executor.go:152
RunLiveStrategy	backend/internal/connect/strategy/live_runner.go:58
RunMarketBacktest	proto/ant/v1/marketplace_service.proto:29
RunStrategy	proto/ant/v1/backtest_service.proto:14
Search	reference/grpc/mt4.proto:215
Search	reference/grpc/mt5.proto:365
SearchBroker	proto/ant/v1/account.proto:22
SendNotification	proto/ant/v1/notification_service.proto:14
ServerTimezone	reference/grpc/mt4.proto:105
ServerTimezone	reference/grpc/mt5.proto:184
SessionState	backend/internal/mthub/service.go:167
Set	backend/internal/risksvc/capability.go:107
Set	backend/internal/service/analytics_cache.go:54
SetAccountNumber	backend/internal/service/account_number.go:230
SetAccountOwnerVerifier	backend/internal/mthub/service.go:108
SetAccountProvider	backend/internal/connect/strategy/strategy_execution_handler.go:90
SetAccountStateProvider	backend/internal/mthub/service.go:85
SetAIPrimary	backend/internal/service/systemai/service.go:195
SetAIPrimary	proto/ant/v1/ai_primary.proto:9
SetAIService	backend/internal/connect/strategy/strategy_experiment_worker.go:80
SetAttribution	backend/internal/service/analytics_cache.go:78
SetAutotradeEnabled	backend/internal/risk/gate.go:101
SetBarBroker	backend/internal/mthub/service.go:114
SetBarSource	backend/internal/connect/strategy/strategy_execution_handler.go:73
SetBrokerLimits	backend/internal/risksvc/platform_aggregator.go:93
SetBrokerRegistry	backend/internal/mthub/service.go:99
SetCanary	proto/ant/v1/admin_sre.proto:11
SetCircuitBreakerDB	backend/internal/service/systemai/chat_failover.go:27
SetCodeAccessChecker	backend/internal/connect/strategy/strategy_handler.go:33
SetConfig	proto/ant/v1/admin_config.proto:11
SetCostEstimator	backend/internal/mthub/service.go:78
SetEngine	backend/internal/connect/strategy/strategy_handler.go:53
SetGate	backend/internal/connect/strategy/strategy_execution_handler.go:80
SetGate	backend/internal/mthub/service.go:88
SetGatewayProviderRepo	backend/internal/service/systemai/service.go:123
SetGoExecutor	backend/internal/connect/strategy/strategy_execution_handler.go:76
SetKillSwitch	backend/internal/mthub/service.go:94
SetKillSwitch	backend/internal/risk/gate.go:94
SetKillSwitch	proto/ant/v1/admin_sre.proto:7
SetKYCStatus	backend/internal/risksvc/jurisdiction_store.go:51
SetKYCStatus	proto/ant/v1/admin_jurisdiction.proto:11
SetLogger	backend/internal/mthub/service.go:111
SetLogger	backend/internal/service/account_service.go:60
SetLogger	backend/internal/service/platform_service.go:29
SetMarketDataRepo	backend/internal/connect/strategy/strategy_execution_handler.go:59
SetMonthlyDetail	backend/internal/service/analytics_cache.go:127
SetMtHub	backend/internal/connect/strategy/strategy_execution_handler.go:74
SetNotificationSender	backend/internal/connect/strategy/strategy_execution_handler.go:98
SetNotificationSender	backend/internal/service/account_sync.go:127
SetOmsWriter	backend/internal/mthub/service.go:91
SetOnBacktestComplete	backend/internal/connect/strategy/strategy_execution_handler.go:99
SetOrderEventBroker	backend/internal/mthub/oms_writer.go:98
SetPaperEngine	backend/internal/connect/strategy/strategy_execution_handler.go:75
SetPgListen	backend/internal/connect/strategy/strategy_execution_handler.go:397
SetPgListen	backend/internal/connect/strategy/strategy_experiment_handler.go:273
SetPgListen	backend/internal/connect/strategy/strategy_experiment_worker.go:62
SetPgListen	backend/internal/connect/strategy/strategy_handler.go:69
SetPlacedType	reference/grpc/mt4.proto:168
SetPostCallBiller	backend/internal/service/systemai/service.go:118
SetRolling	backend/internal/service/analytics_cache.go:102
SetSanctionedOverride	proto/ant/v1/admin_jurisdiction.proto:16
SetScheduleActive	backend/internal/service/schedule_svc.go:148
SetStatusBroker	backend/internal/mthub/service.go:117
SetStrategyPricing	proto/ant/v1/marketplace_service.proto:21
SetTemplateStatus	backend/internal/service/template_svc.go:102
SetTicket	backend/internal/mthub/idempotency.go:167
SetTokenRecorder	backend/internal/service/systemai/service.go:106
SetUserLimiter	backend/internal/mthub/service.go:75
SetUserLimiter	backend/internal/risksvc/engine.go:22
SetUserRepo	backend/internal/service/systemai/service.go:91
SetWalletChecker	backend/internal/service/systemai/service.go:112
Shutdown	backend/internal/risksvc/platform_aggregator.go:174
Size	backend/internal/risksvc/kelly_sizer.go:44
Size	backend/internal/risksvc/vol_target_sizer.go:45
SoftDeleteUser	backend/internal/service/user_deletion_service.go:34
SoftDeleteUsers	backend/internal/service/user_deletion_service.go:85
Start	backend/internal/connect/strategy/schedule_engine.go:66
Start	backend/internal/connect/strategy/strategy_experiment_worker.go:44
Start	backend/internal/mthub/derived_state.go:109
Start	backend/internal/mthub/reconciliation.go:31
StartAlgo	proto/ant/v1/execution_algo.proto:14
StartBacktestRun	backend/internal/connect/strategy/strategy_backtest_crud.go:26
StartBacktestRun	proto/ant/v1/strategy_runtime.proto:18
StartBacktestWorker	backend/internal/connect/strategy/backtest_worker.go:425
StartPaperStrategy	proto/ant/v1/paper_trading.proto:12
StartRefreshLoop	backend/internal/risksvc/platform_aggregator.go:153
State	backend/internal/mthub/derived_state.go:119
Stats	backend/internal/mthub/state_cache.go:207
StepUpLotSize	backend/internal/risk/canary.go:205
Stop	backend/internal/connect/strategy/schedule_engine.go:278
Stop	backend/internal/connect/strategy/strategy_experiment_worker.go:77
Stop	backend/internal/mthub/derived_state.go:114
StopPaperStrategy	proto/ant/v1/paper_trading.proto:13
StopSchedule	backend/internal/connect/strategy/schedule_engine.go:266
StreamNotifications	proto/ant/v1/notification_service.proto:13
StreamOrderEvents	proto/ant/v1/mthub_service.proto:15
StreamTicks	proto/ant/v1/market_service.proto:10
String	backend/internal/risk/gate.go:228
SubmitAssetReview	backend/internal/connect/strategy/strategy_asset_handler.go:110
SubmitAssetReview	proto/ant/v1/strategy_asset.proto:12
SubmitQuestionnaire	backend/internal/risksvc/jurisdiction_store.go:112
SubmitStrategyExperiment	backend/internal/connect/strategy/strategy_experiment_handler.go:101
SubmitStrategyExperiment	proto/ant/v1/strategy_experiment.proto:11
Subscribe	backend/internal/connect/strategy/data_source.go:77
Subscribe	backend/internal/mthub/broker_types.go:158
Subscribe	backend/internal/mthub/broker_types.go:215
Subscribe	backend/internal/mthub/broker_types.go:77
Subscribe	backend/internal/mthub/types.go:162
Subscribe	backend/internal/mthub/types.go:203
Subscribe	backend/internal/paper/engine.go:184
Subscribe	proto/ant/v1/marketplace_service.proto:10
Subscribe	reference/grpc/mt4.proto:230
Subscribe	reference/grpc/mt5.proto:387
SubscribeAccountProfit	backend/internal/mthub/service.go:266
SubscribeAccountStatus	backend/internal/mthub/service.go:144
SubscribeBars	proto/ant/v1/mthub_service.proto:17
SubscribeBarUpdates	backend/internal/mthub/service.go:127
SubscribeEvents	proto/ant/v1/stream.proto:75
SubscribeHistory	proto/ant/v1/stream.proto:76
SubscribeIndicators	proto/ant/v1/stream.proto:80
SubscribeJob	proto/ant/v1/job.proto:13
SubscribeMany	reference/grpc/mt4.proto:237
SubscribeMany	reference/grpc/mt5.proto:394
SubscribeMarketWatch	reference/grpc/mt5.proto:433
SubscribeOpenedOrdersTickets	reference/grpc/mt5.proto:440
SubscribeOrderProfit	reference/grpc/mt4.proto:256
SubscribeOrderProfit	reference/grpc/mt5.proto:413
SubscribeOrderUpdate	reference/grpc/mt4.proto:270
SubscribeOrderUpdate	reference/grpc/mt5.proto:427
SubscribeOrderUpdates	proto/ant/v1/stream.proto:77
SubscribePositionSnapshots	backend/internal/mthub/service.go:276
SubscribeProfitUpdates	proto/ant/v1/stream.proto:78
SubscribeQuoteHistory	reference/grpc/mt4.proto:276
SubscribeSymbols	backend/internal/mthub/service.go:247
SubscribeTickValue	reference/grpc/mt4.proto:264
SubscribeTickValue	reference/grpc/mt5.proto:421
SubscribeUserOrderEvents	backend/internal/mthub/service.go:256
SubscribeUserSummary	proto/ant/v1/stream.proto:79
SymbolList	backend/internal/mthub/service.go:236
SymbolList	proto/ant/v1/mthub_service.proto:12
SymbolList	reference/grpc/mt5.proto:130
SymbolParams	backend/internal/mthub/service.go:218
SymbolParams	proto/ant/v1/mthub_service.proto:11
SymbolParams	reference/grpc/mt4.proto:99
SymbolParams	reference/grpc/mt5.proto:158
SymbolParamsMany	reference/grpc/mt4.proto:111
SymbolParamsMany	reference/grpc/mt5.proto:165
Symbols	reference/grpc/mt4.proto:92
Symbols	reference/grpc/mt5.proto:124
SymbolSessionsEx	reference/grpc/mt5.proto:172
SymbolSessionsExMany	reference/grpc/mt5.proto:178
SyncAccountHistory	backend/internal/service/account_sync.go:133
SyncOrderHistory	proto/ant/v1/mthub_service.proto:16
SyncStrategyAsset	backend/internal/connect/strategy/strategy_asset_handler.go:185
SyncStrategyAsset	proto/ant/v1/strategy_asset.proto:16
TickHistoryRequest	reference/grpc/mt5.proto:453
TickHistoryStop	reference/grpc/mt5.proto:460
TickValueWithSize	reference/grpc/mt4.proto:180
TickValueWithSize	reference/grpc/mt5.proto:222
TierCheck	backend/internal/risksvc/capability.go:74
ToggleAutoTrade	proto/ant/v1/auto_trading.proto:17
ToggleConfigEnabled	proto/ant/v1/admin_config.proto:12
ToggleSchedule	backend/internal/connect/strategy/strategy_schedules.go:149
ToggleSchedule	proto/ant/v1/strategy.proto:27
TransformCode	proto/ant/v1/code_assist.proto:15
Transition	backend/internal/mthub/oms_writer.go:133
TranslateParamLabels	proto/ant/v1/code_assist.proto:17
TranspileCode	backend/internal/connect/strategy/strategy_execution_handler.go:198
TranspileCode	proto/ant/v1/strategy_runtime.proto:31
TriggerReconcile	backend/internal/mthub/reconciliation.go:55
UnflagStrategy	proto/ant/v1/admin_strategy.proto:24
UnflagTemplate	backend/internal/service/template_svc_admin.go:263
UnfreezeAccount	proto/ant/v1/admin_account.proto:12
UnpublishStrategy	proto/ant/v1/admin_strategy.proto:25
UnpublishStrategy	proto/ant/v1/marketplace_service.proto:23
UnpublishTemplate	backend/internal/service/template_svc_admin.go:276
Unsubscribe	proto/ant/v1/marketplace_service.proto:11
UnSubscribe	reference/grpc/mt4.proto:244
UnSubscribe	reference/grpc/mt5.proto:401
UnSubscribeMany	reference/grpc/mt4.proto:250
UnSubscribeMany	reference/grpc/mt5.proto:407
Update	backend/internal/mthub/derived_state.go:60
UpdateAccount	backend/internal/service/account_service.go:173
UpdateAccount	proto/ant/v1/account.proto:17
UpdateAccountInfo	backend/internal/service/account_lifecycle.go:115
UpdateAccountInfoTx	backend/internal/service/account_lifecycle.go:93
UpdateAccountMetrics	backend/internal/service/account_lifecycle.go:136
UpdateBalanceFromProfitEvent	backend/internal/connect/strategy/account_provider.go:159
UpdateBrokerThresholds	backend/internal/service/account_lifecycle.go:178
UpdateConfig	backend/internal/service/systemai/service.go:181
UpdateConversationTitle	proto/ant/v1/ai.proto:18
UpdateExecution	backend/internal/service/log_service.go:33
UpdateGlobalSettings	proto/ant/v1/auto_trading.proto:16
UpdateOrderHistoryClose	backend/internal/service/log_service.go:46
UpdatePosition	backend/internal/risksvc/platform_aggregator.go:73
UpdateProvider	proto/ant/v1/ai_gateway.proto:20
UpdateRiskConfig	proto/ant/v1/auto_trading.proto:19
UpdateSchedule	backend/internal/connect/strategy/strategy_schedules.go:94
UpdateSchedule	backend/internal/service/schedule_svc.go:115
UpdateSchedule	proto/ant/v1/strategy.proto:25
UpdateSecret	backend/internal/service/systemai/service.go:205
UpdateSessionStrategyKey	proto/ant/v1/ai.proto:22
UpdateSummaryCache	backend/internal/service/account_snapshot.go:104
UpdateSystemAIConfig	proto/ant/v1/system_ai.proto:13
UpdateSystemAISecret	proto/ant/v1/system_ai.proto:14
UpdateSystemStrategy	backend/internal/service/template_svc_admin.go:98
UpdateSystemStrategy	proto/ant/v1/admin_strategy.proto:15
UpdateTemplate	backend/internal/connect/strategy/strategy_template_handlers.go:99
UpdateTemplate	backend/internal/service/template_svc.go:80
UpdateTemplate	proto/ant/v1/strategy.proto:14
UpdateTemplateDraft	backend/internal/connect/strategy/strategy_template_handlers.go:169
UpdateTemplateDraft	proto/ant/v1/strategy.proto:18
UpdateTradingPassword	backend/internal/service/account_lifecycle.go:189
UpdateTradingPassword	proto/ant/v1/account.proto:24
UpdateUser	proto/ant/v1/admin_user.proto:13
UpsertModel	proto/ant/v1/ai_gateway.proto:26
UserOwnsAccount	backend/internal/service/account_snapshot.go:46
UserOwnsAccount	backend/internal/service/platform_service.go:118
Validate	backend/internal/connect/strategy/strategy_execution_handler.go:133
Validate	proto/ant/v1/strategy_runtime.proto:16
ValidateStrategy	proto/ant/v1/backtest_service.proto:13
ValidateStrategyExtended	proto/ant/v1/code_assist.proto:13
ValidateSystemAIConnection	proto/ant/v1/system_ai.proto:16
VerifyAccount	proto/ant/v1/account.proto:25
VerifyTradePermission	proto/ant/v1/account.proto:23
Version	reference/grpc/mt5.proto:376
WaitSession	backend/internal/mthub/types.go:52
WatchBacktestRun	backend/internal/connect/strategy/strategy_backtest_crud.go:126
WatchBacktestRun	proto/ant/v1/strategy_runtime.proto:21
WatchExperiment	backend/internal/connect/strategy/strategy_experiment_handler.go:225
WatchExperiment	proto/ant/v1/strategy_experiment.proto:20
WatchPaperAccount	proto/ant/v1/paper_trading.proto:14
WatchSchedules	backend/internal/connect/strategy/strategy_schedules.go:182
WatchSchedules	proto/ant/v1/strategy.proto:28
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
proto/ant/v1/agent_gateway.proto:10:  rpc IssueAgentToken(IssueAgentTokenRequest) returns (IssueAgentTokenResponse);
proto/ant/v1/agent_gateway.proto:11:  rpc ListAgentTokens(ListAgentTokensRequest) returns (ListAgentTokensResponse);
proto/ant/v1/agent_gateway.proto:12:  rpc RevokeAgentToken(RevokeAgentTokenRequest) returns (AgentToken);
proto/ant/v1/agent_gateway.proto:13:  rpc ListAgentAudit(ListAgentAuditRequest) returns (ListAgentAuditResponse);
proto/ant/v1/agent_gateway.proto:14:  rpc GetAgentCapabilities(GetAgentCapabilitiesRequest) returns (AgentCapabilities);
proto/ant/v1/ai.proto:11:  rpc Chat(ChatRequest) returns (ChatResponse);
proto/ant/v1/ai.proto:13:  rpc ChatStream(ChatRequest) returns (stream ChatStreamChunk);
proto/ant/v1/ai.proto:14:  rpc ListConversations(ListConversationsRequest) returns (ListConversationsResponse);
proto/ant/v1/ai.proto:15:  rpc GetConversation(GetConversationRequest) returns (GetConversationResponse);
proto/ant/v1/ai.proto:16:  rpc CreateConversation(CreateConversationRequest) returns (CreateConversationResponse);
proto/ant/v1/ai.proto:17:  rpc DeleteConversation(DeleteConversationRequest) returns (DeleteConversationResponse);
proto/ant/v1/ai.proto:18:  rpc UpdateConversationTitle(UpdateConversationTitleRequest) returns (UpdateConversationTitleResponse);
proto/ant/v1/ai.proto:19:  rpc ListAgents(ListAgentsRequest) returns (ListAgentsResponse);
proto/ant/v1/ai.proto:20:  rpc BatchSetAgents(BatchSetAgentsRequest) returns (BatchSetAgentsResponse);
proto/ant/v1/ai.proto:21:  rpc ResolveSession(ResolveSessionRequest) returns (ResolveSessionResponse);
proto/ant/v1/ai.proto:22:  rpc UpdateSessionStrategyKey(UpdateSessionStrategyKeyRequest) returns (UpdateSessionStrategyKeyResponse);
proto/ant/v1/ai_agent.proto:10:  rpc ListAgentDefs(ListAgentDefsRequest) returns (ListAgentDefsResponse);
proto/ant/v1/ai_agent.proto:9:  rpc BatchSetAgents(BatchSetAgentsRequest) returns (BatchSetAgentsResponse);
proto/ant/v1/ai_gate.proto:11:  rpc RunEvaluation(RunGateEvaluationRequest) returns (stream GateEvaluationUpdate);
proto/ant/v1/ai_gateway.proto:13:  rpc ListSystemModels(ListSystemModelsRequest) returns (ListSystemModelsResponse);
proto/ant/v1/ai_gateway.proto:15:  rpc GetTokenUsage(GetTokenUsageRequest) returns (GetTokenUsageResponse);
proto/ant/v1/ai_gateway.proto:18:  rpc ListProviders(ListProvidersRequest) returns (ListProvidersResponse);
proto/ant/v1/ai_gateway.proto:20:  rpc UpdateProvider(UpdateProviderRequest) returns (UpdateProviderResponse);
proto/ant/v1/ai_gateway.proto:22:  rpc DeleteProvider(DeleteProviderRequest) returns (DeleteProviderResponse);
proto/ant/v1/ai_gateway.proto:24:  rpc ListModels(ListModelsRequest) returns (ListModelsResponse);
proto/ant/v1/ai_gateway.proto:26:  rpc UpsertModel(UpsertModelRequest) returns (UpsertModelResponse);
proto/ant/v1/ai_gateway.proto:28:  rpc DeleteModel(DeleteModelRequest) returns (DeleteModelResponse);
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
proto/ant/v1/auth.proto:14:  rpc GetMe(google.protobuf.Empty) returns (GetMeResponse);
proto/ant/v1/auth.proto:15:  rpc Register(RegisterRequest) returns (RegisterResponse);
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
proto/ant/v1/backtest_service.proto:12:  rpc RunBacktest(ExecuteBacktestRequest) returns (ExecuteBacktestResponse);
proto/ant/v1/backtest_service.proto:13:  rpc ValidateStrategy(EngineValidateRequest) returns (EngineValidateResponse);
proto/ant/v1/backtest_service.proto:14:  rpc RunStrategy(EngineRunStrategyRequest) returns (EngineRunStrategyResponse);
proto/ant/v1/backtest_trades.proto:8:  rpc ListBacktestRunTrades(ListBacktestRunTradesRequest) returns (ListBacktestRunTradesResponse);
proto/ant/v1/code_assist.proto:10:  rpc ReviseCode(ReviseCodeRequest) returns (ReviseCodeResponse);
proto/ant/v1/code_assist.proto:11:  rpc ReviseCodeStream(ReviseCodeRequest) returns (stream ReviseCodeStreamChunk);
proto/ant/v1/code_assist.proto:12:  rpc ExplainCode(ExplainCodeRequest) returns (ExplainCodeResponse);
proto/ant/v1/code_assist.proto:13:  rpc ValidateStrategyExtended(ValidateStrategyExtendedRequest) returns (ValidateStrategyExtendedResponse);
proto/ant/v1/code_assist.proto:15:  rpc TransformCode(TransformCodeRequest) returns (TransformCodeResponse);
proto/ant/v1/code_assist.proto:17:  rpc TranslateParamLabels(TranslateParamLabelsRequest) returns (TranslateParamLabelsResponse);
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
proto/ant/v1/log.proto:14:  rpc GetConnectionLogs(GetConnectionLogsRequest) returns (GetConnectionLogsResponse);
proto/ant/v1/log.proto:15:  rpc GetExecutionLogs(GetExecutionLogsRequest) returns (GetExecutionLogsResponse);
proto/ant/v1/log.proto:16:  rpc GetOrderLogHistory(GetOrderLogHistoryRequest) returns (GetOrderLogHistoryResponse);
proto/ant/v1/log.proto:17:  rpc GetOperationLogs(GetOperationLogsRequest) returns (GetOperationLogsResponse);
proto/ant/v1/log.proto:18:  rpc GetScheduleRunLogs(GetScheduleRunLogsRequest) returns (GetScheduleRunLogsResponse);
proto/ant/v1/market_regime.proto:10:  rpc DetectMarketRegime(DetectMarketRegimeRequest) returns (DetectMarketRegimeResponse);
proto/ant/v1/market_regime.proto:11:  rpc GetMarketRegime(GetMarketRegimeRequest) returns (GetMarketRegimeResponse);
proto/ant/v1/market_service.proto:10:  rpc StreamTicks(StreamTicksRequest) returns (stream TickMsg);
proto/ant/v1/market_service.proto:8:  rpc GetKlines(GetKlinesRequest) returns (GetKlinesResponse);
proto/ant/v1/market_service.proto:9:  rpc GetSymbolStats(GetSymbolStatsRequest) returns (GetSymbolStatsResponse);
proto/ant/v1/marketplace_service.proto:10:  rpc Subscribe(SubscribeRequest) returns (SubscribeResponse);
proto/ant/v1/marketplace_service.proto:11:  rpc Unsubscribe(UnsubscribeRequest) returns (UnsubscribeResponse);
proto/ant/v1/marketplace_service.proto:12:  rpc PurchaseStrategy(PurchaseStrategyRequest) returns (PurchaseStrategyResponse);
proto/ant/v1/marketplace_service.proto:13:  rpc ListPublished(ListPublishedRequest) returns (ListPublishedResponse);
proto/ant/v1/marketplace_service.proto:14:  rpc ListSubscriptions(ListSubscriptionsRequest) returns (ListSubscriptionsResponse);
proto/ant/v1/marketplace_service.proto:16:  rpc RateStrategy(RateStrategyRequest) returns (RateStrategyResponse);
proto/ant/v1/marketplace_service.proto:17:  rpc ListRatings(ListRatingsRequest) returns (ListRatingsResponse);
proto/ant/v1/marketplace_service.proto:18:  rpc CommentOnStrategy(CommentOnStrategyRequest) returns (CommentOnStrategyResponse);
proto/ant/v1/marketplace_service.proto:19:  rpc ListComments(ListCommentsRequest) returns (ListCommentsResponse);
proto/ant/v1/marketplace_service.proto:21:  rpc SetStrategyPricing(SetStrategyPricingRequest) returns (SetStrategyPricingResponse);
proto/ant/v1/marketplace_service.proto:23:  rpc UnpublishStrategy(UnpublishMarketStrategyRequest) returns (UnpublishMarketStrategyResponse);
proto/ant/v1/marketplace_service.proto:25:  rpc GetPublisherStats(GetPublisherStatsRequest) returns (GetPublisherStatsResponse);
proto/ant/v1/marketplace_service.proto:29:  rpc RunMarketBacktest(RunMarketBacktestRequest) returns (stream BacktestRunUpdate);
proto/ant/v1/marketplace_service.proto:9:  rpc PublishStrategy(PublishStrategyRequest) returns (PublishStrategyResponse);
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
proto/ant/v1/notification_service.proto:10:  rpc ListNotifications(ListNotificationsRequest) returns (ListNotificationsResponse);
proto/ant/v1/notification_service.proto:11:  rpc MarkRead(MarkReadRequest) returns (MarkReadResponse);
proto/ant/v1/notification_service.proto:12:  rpc MarkAllRead(MarkAllReadRequest) returns (MarkAllReadResponse);
proto/ant/v1/notification_service.proto:13:  rpc StreamNotifications(StreamNotificationsRequest) returns (stream Notification);
proto/ant/v1/notification_service.proto:14:  rpc SendNotification(SendNotificationRequest) returns (SendNotificationResponse);
proto/ant/v1/paper_trading.proto:10:  rpc CreatePaperAccount(CreatePaperAccountRequest) returns (PaperAccount);
proto/ant/v1/paper_trading.proto:11:  rpc ListPaperAccounts(ListPaperAccountsRequest) returns (ListPaperAccountsResponse);
proto/ant/v1/paper_trading.proto:12:  rpc StartPaperStrategy(StartPaperStrategyRequest) returns (StartPaperStrategyResponse);
proto/ant/v1/paper_trading.proto:13:  rpc StopPaperStrategy(StopPaperStrategyRequest) returns (StopPaperStrategyResponse);
proto/ant/v1/paper_trading.proto:14:  rpc WatchPaperAccount(WatchPaperAccountRequest) returns (stream PaperAccountUpdate);
proto/ant/v1/schedule_health.proto:10:  rpc GetScheduleHealth(GetScheduleHealthRequest) returns (GetScheduleHealthResponse);
proto/ant/v1/share.proto:11:  rpc GetSharedPerformance(GetSharedPerformanceRequest) returns (GetSharedPerformanceResponse);
proto/ant/v1/share.proto:9:  rpc CreateShareToken(CreateShareTokenRequest) returns (CreateShareTokenResponse);
proto/ant/v1/strategy.proto:11:  rpc ListTemplates(ListTemplatesRequest) returns (ListTemplatesResponse);
proto/ant/v1/strategy.proto:12:  rpc GetTemplate(GetTemplateRequest) returns (StrategyTemplate);
proto/ant/v1/strategy.proto:13:  rpc CreateTemplate(CreateTemplateRequest) returns (StrategyTemplate);
proto/ant/v1/strategy.proto:14:  rpc UpdateTemplate(UpdateTemplateRequest) returns (StrategyTemplate);
proto/ant/v1/strategy.proto:15:  rpc DeleteTemplate(DeleteTemplateRequest) returns (google.protobuf.Empty);
proto/ant/v1/strategy.proto:17:  rpc CreateTemplateDraft(CreateTemplateDraftRequest) returns (StrategyTemplate);
proto/ant/v1/strategy.proto:18:  rpc UpdateTemplateDraft(UpdateTemplateDraftRequest) returns (StrategyTemplate);
proto/ant/v1/strategy.proto:19:  rpc PublishTemplateDraft(PublishTemplateDraftRequest) returns (StrategyTemplate);
proto/ant/v1/strategy.proto:20:  rpc CancelTemplateDraft(CancelTemplateDraftRequest) returns (google.protobuf.Empty);
proto/ant/v1/strategy.proto:22:  rpc ListSchedules(ListSchedulesRequest) returns (ListSchedulesResponse);
proto/ant/v1/strategy.proto:23:  rpc GetSchedule(GetScheduleRequest) returns (StrategySchedule);
proto/ant/v1/strategy.proto:24:  rpc CreateSchedule(CreateScheduleRequest) returns (StrategySchedule);
proto/ant/v1/strategy.proto:25:  rpc UpdateSchedule(UpdateScheduleRequest) returns (StrategySchedule);
proto/ant/v1/strategy.proto:26:  rpc DeleteSchedule(DeleteScheduleRequest) returns (google.protobuf.Empty);
proto/ant/v1/strategy.proto:27:  rpc ToggleSchedule(ToggleScheduleRequest) returns (StrategySchedule);
proto/ant/v1/strategy.proto:28:  rpc WatchSchedules(WatchSchedulesRequest) returns (stream WatchSchedulesEvent);
proto/ant/v1/strategy.proto:29:  rpc RunBacktest(RunBacktestRequest) returns (RunBacktestResponse);
proto/ant/v1/strategy.proto:31:  rpc ListSignals(ListSignalsRequest) returns (ListSignalsResponse);
proto/ant/v1/strategy.proto:32:  rpc ExecuteSignal(ExecuteSignalRequest) returns (ExecuteSignalResponse);
proto/ant/v1/strategy.proto:33:  rpc ConfirmSignal(ConfirmSignalRequest) returns (google.protobuf.Empty);
proto/ant/v1/strategy.proto:34:  rpc CancelSignal(CancelSignalRequest) returns (google.protobuf.Empty);
proto/ant/v1/strategy_asset.proto:10:  rpc ListStrategyAssets(ListStrategyAssetsRequest) returns (ListStrategyAssetsResponse);
proto/ant/v1/strategy_asset.proto:11:  rpc GetStrategyAsset(GetStrategyAssetRequest) returns (StrategyAsset);
proto/ant/v1/strategy_asset.proto:12:  rpc SubmitAssetReview(SubmitAssetReviewRequest) returns (StrategyAsset);
proto/ant/v1/strategy_asset.proto:13:  rpc ReviewStrategyAsset(ReviewStrategyAssetRequest) returns (StrategyAsset);
proto/ant/v1/strategy_asset.proto:14:  rpc CloneStrategyAsset(CloneStrategyAssetRequest) returns (CloneStrategyAssetResponse);
proto/ant/v1/strategy_asset.proto:15:  rpc CheckAssetUpdate(CheckAssetUpdateRequest) returns (StrategyAssetClone);
proto/ant/v1/strategy_asset.proto:16:  rpc SyncStrategyAsset(SyncStrategyAssetRequest) returns (StrategyAssetClone);
proto/ant/v1/strategy_asset.proto:17:  rpc ListAssetClones(ListAssetClonesRequest) returns (ListAssetClonesResponse);
proto/ant/v1/strategy_execution.proto:11:  rpc AnalyzePlan(AnalyzePlanRequest) returns (stream AnalyzePlanChunk);
proto/ant/v1/strategy_execution.proto:12:  rpc Diagnose(DiagnoseRequest) returns (stream AnalyzePlanChunk);
proto/ant/v1/strategy_execution.proto:13:  rpc ExecutePlan(ExecutePlanRequest) returns (stream ExecutePlanChunk);
proto/ant/v1/strategy_execution.proto:14:  rpc Conversate(ConversateRequest) returns (stream ConversateChunk);
proto/ant/v1/strategy_experiment.proto:11:  rpc SubmitStrategyExperiment(SubmitStrategyExperimentRequest) returns (SubmitStrategyExperimentResponse);
proto/ant/v1/strategy_experiment.proto:12:  rpc GetStrategyExperiment(GetStrategyExperimentRequest) returns (StrategyExperiment);
proto/ant/v1/strategy_experiment.proto:13:  rpc ListStrategyExperiments(ListStrategyExperimentsRequest) returns (ListStrategyExperimentsResponse);
proto/ant/v1/strategy_experiment.proto:14:  rpc CancelStrategyExperiment(CancelStrategyExperimentRequest) returns (StrategyExperiment);
proto/ant/v1/strategy_experiment.proto:15:  rpc ListExperimentCandidates(ListExperimentCandidatesRequest) returns (ListExperimentCandidatesResponse);
proto/ant/v1/strategy_experiment.proto:16:  rpc GetExperimentCandidate(GetExperimentCandidateRequest) returns (StrategyExperimentCandidate);
proto/ant/v1/strategy_experiment.proto:17:  rpc PromoteCandidateToDraft(PromoteCandidateToDraftRequest) returns (PromoteCandidateToDraftResponse);
proto/ant/v1/strategy_experiment.proto:20:  rpc WatchExperiment(WatchExperimentRequest) returns (stream WatchExperimentEvent);
proto/ant/v1/strategy_generation.proto:12:  rpc GenerateStrategy(GenerateStrategyRequest) returns (stream GenerateStrategyChunk);
proto/ant/v1/strategy_runtime.proto:15:  rpc Execute(ExecuteStrategyRequest) returns (ExecuteStrategyResponse);
proto/ant/v1/strategy_runtime.proto:16:  rpc Validate(ValidateStrategyRequest) returns (ValidateStrategyResponse);
proto/ant/v1/strategy_runtime.proto:17:  rpc Backtest(BacktestStrategyRequest) returns (BacktestStrategyResponse);
proto/ant/v1/strategy_runtime.proto:18:  rpc StartBacktestRun(StartBacktestRunRequest) returns (StartBacktestRunResponse);
proto/ant/v1/strategy_runtime.proto:19:  rpc GetBacktestRun(GetBacktestRunRequest) returns (GetBacktestRunResponse);
proto/ant/v1/strategy_runtime.proto:20:  rpc ListBacktestRuns(ListBacktestRunsRequest) returns (ListBacktestRunsResponse);
proto/ant/v1/strategy_runtime.proto:21:  rpc WatchBacktestRun(WatchBacktestRunRequest) returns (stream BacktestRunUpdate);
proto/ant/v1/strategy_runtime.proto:22:  rpc CancelBacktestRun(CancelBacktestRunRequest) returns (CancelBacktestRunResponse);
proto/ant/v1/strategy_runtime.proto:23:  rpc DeleteBacktestRun(DeleteBacktestRunRequest) returns (DeleteBacktestRunResponse);
proto/ant/v1/strategy_runtime.proto:24:  rpc DeleteBacktestRuns(DeleteBacktestRunsRequest) returns (DeleteBacktestRunsResponse);
proto/ant/v1/strategy_runtime.proto:25:  rpc GetTemplates(google.protobuf.Empty) returns (GetStrategyTemplatesResponse);
proto/ant/v1/strategy_runtime.proto:28:  rpc ExecuteLive(ExecuteLiveRequest) returns (ExecuteLiveResponse);
proto/ant/v1/strategy_runtime.proto:31:  rpc TranspileCode(TranspileCodeRequest) returns (TranspileCodeResponse);
proto/ant/v1/strategy_runtime.proto:33:  rpc AnalyzeImportCode(AnalyzeImportCodeRequest) returns (AnalyzeImportCodeResponse);
proto/ant/v1/strategy_runtime.proto:35:  rpc GenerateImportCode(GenerateImportCodeRequest) returns (GenerateImportCodeResponse);
proto/ant/v1/strategy_runtime.proto:37:  rpc ImportStrategy(ImportStrategyRequest) returns (ImportStrategyResponse);
proto/ant/v1/stream.proto:75:  rpc SubscribeEvents(SubscribeEventsRequest) returns (stream StreamEvent);
proto/ant/v1/stream.proto:76:  rpc SubscribeHistory(SubscribeHistoryRequest) returns (stream StreamEvent);
proto/ant/v1/stream.proto:77:  rpc SubscribeOrderUpdates(SubscribeOrderUpdatesRequest) returns (stream OrderUpdateEvent);
proto/ant/v1/stream.proto:78:  rpc SubscribeProfitUpdates(SubscribeProfitUpdatesRequest) returns (stream ProfitUpdateEvent);
proto/ant/v1/stream.proto:79:  rpc SubscribeUserSummary(google.protobuf.Empty) returns (stream UserSummaryEvent);
proto/ant/v1/stream.proto:80:  rpc SubscribeIndicators(SubscribeIndicatorsRequest) returns (stream IndicatorUpdateEvent);
proto/ant/v1/system_ai.proto:11:  rpc ListSystemAIConfigs(ListSystemAIConfigsRequest) returns (ListSystemAIConfigsResponse);
proto/ant/v1/system_ai.proto:12:  rpc GetSystemAIConfig(GetSystemAIConfigRequest) returns (GetSystemAIConfigResponse);
proto/ant/v1/system_ai.proto:13:  rpc UpdateSystemAIConfig(UpdateSystemAIConfigRequest) returns (UpdateSystemAIConfigResponse);
proto/ant/v1/system_ai.proto:14:  rpc UpdateSystemAISecret(UpdateSystemAISecretRequest) returns (UpdateSystemAISecretResponse);
proto/ant/v1/system_ai.proto:15:  rpc DiscoverSystemAIModels(DiscoverSystemAIModelsRequest) returns (DiscoverSystemAIModelsResponse);
proto/ant/v1/system_ai.proto:16:  rpc ValidateSystemAIConnection(ValidateSystemAIConnectionRequest) returns (ValidateSystemAIConnectionResponse);
proto/ant/v1/wallet.proto:10:  rpc ListTransactions(ListWalletTransactionsRequest) returns (ListWalletTransactionsResponse);
proto/ant/v1/wallet.proto:11:  rpc AdjustBalance(AdjustBalanceRequest) returns (AdjustBalanceResponse);  // admin
proto/ant/v1/wallet.proto:9:  rpc GetWallet(GetWalletRequest) returns (GetWalletResponse);
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
backend/internal/connect/strategy/account_provider.go:144:func (p *MTAccountStateProvider) GetPeakEquity(accountID string) decimal.Decimal {
backend/internal/connect/strategy/account_provider.go:151:func (p *MTAccountStateProvider) ResetPeakEquity(accountID string) {
backend/internal/connect/strategy/account_provider.go:159:func (p *MTAccountStateProvider) UpdateBalanceFromProfitEvent(accountID string, balance, equity, margin, freeMargin float64) {
backend/internal/connect/strategy/account_provider.go:53:func (p *MTAccountStateProvider) GetAccountState(ctx context.Context, accountID string) (*risk.AccountState, error) {
backend/internal/connect/strategy/ai_proposer_adapter.go:22:func (a *systemAIAdapter) ChatCompletion(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
backend/internal/connect/strategy/backtest_trades_handler.go:55:func (s *BacktestTradesServer) ListBacktestRunTrades(ctx context.Context, req *connect.Request[antv1.ListBacktestRunTradesRequest]) (*connect.Response[antv1.ListBacktestRunTradesResponse], error) {
backend/internal/connect/strategy/backtest_worker.go:425:func (s *StrategyExecutionServer) StartBacktestWorker(ctx context.Context) {
backend/internal/connect/strategy/data_source.go:55:func (s *LiveSource) Name() string { return "live" }
backend/internal/connect/strategy/data_source.go:57:func (s *LiveSource) Fetch(ctx context.Context, symbol, timeframe string, from, to *time.Time) ([]*antv1.ExecuteKlineBar, error) {
backend/internal/connect/strategy/data_source.go:77:func (s *LiveSource) Subscribe(accountID string) (<-chan *mthub.BarUpdate, func()) {
backend/internal/connect/strategy/go_executor.go:123:func (e *GoExecutor) CompileCheck(ctx context.Context, code string) (bool, string) {
backend/internal/connect/strategy/go_executor.go:146:func (e *GoExecutor) Cleanup() {
backend/internal/connect/strategy/go_executor.go:152:func (e *GoExecutor) RunLive(ctx context.Context, code string, req *antv1.ExecuteLiveRequest) (*antv1.ExecuteLiveResponse, error) {
backend/internal/connect/strategy/go_executor.go:37:func (e *GoExecutor) Run(ctx context.Context, code string, req *antv1.ExecuteStrategyRequest) (*antv1.ExecuteStrategyResponse, error) {
backend/internal/connect/strategy/go_executor.go:75:func (e *GoExecutor) RunBacktest(ctx context.Context, code string, req *antv1.ExecuteBacktestRequest) (*antv1.ExecuteBacktestResponse, error) {
backend/internal/connect/strategy/live_runner.go:58:func (s *StrategyExecutionServer) RunLiveStrategy(ctx context.Context, cfg LiveStrategyConfig) error {
backend/internal/connect/strategy/schedule_engine.go:102:func (e *ScheduleEngine) Notify() {
backend/internal/connect/strategy/schedule_engine.go:266:func (e *ScheduleEngine) StopSchedule(id uuid.UUID) {
backend/internal/connect/strategy/schedule_engine.go:278:func (e *ScheduleEngine) Stop() {
backend/internal/connect/strategy/schedule_engine.go:66:func (e *ScheduleEngine) Start(ctx context.Context) error {
backend/internal/connect/strategy/strategy_asset_handler.go:110:func (s *StrategyAssetServer) SubmitAssetReview(ctx context.Context, req *connect.Request[antv1.SubmitAssetReviewRequest]) (*connect.Response[antv1.StrategyAsset], error) {
backend/internal/connect/strategy/strategy_asset_handler.go:134:func (s *StrategyAssetServer) ReviewStrategyAsset(ctx context.Context, req *connect.Request[antv1.ReviewStrategyAssetRequest]) (*connect.Response[antv1.StrategyAsset], error) {
backend/internal/connect/strategy/strategy_asset_handler.go:146:func (s *StrategyAssetServer) CloneStrategyAsset(ctx context.Context, req *connect.Request[antv1.CloneStrategyAssetRequest]) (*connect.Response[antv1.CloneStrategyAssetResponse], error) {
backend/internal/connect/strategy/strategy_asset_handler.go:172:func (s *StrategyAssetServer) CheckAssetUpdate(ctx context.Context, req *connect.Request[antv1.CheckAssetUpdateRequest]) (*connect.Response[antv1.StrategyAssetClone], error) {
backend/internal/connect/strategy/strategy_asset_handler.go:185:func (s *StrategyAssetServer) SyncStrategyAsset(ctx context.Context, req *connect.Request[antv1.SyncStrategyAssetRequest]) (*connect.Response[antv1.StrategyAssetClone], error) {
backend/internal/connect/strategy/strategy_asset_handler.go:203:func (s *StrategyAssetServer) ListAssetClones(ctx context.Context, req *connect.Request[antv1.ListAssetClonesRequest]) (*connect.Response[antv1.ListAssetClonesResponse], error) {
backend/internal/connect/strategy/strategy_asset_handler.go:83:func (s *StrategyAssetServer) ListStrategyAssets(ctx context.Context, req *connect.Request[antv1.ListStrategyAssetsRequest]) (*connect.Response[antv1.ListStrategyAssetsResponse], error) {
backend/internal/connect/strategy/strategy_asset_handler.go:97:func (s *StrategyAssetServer) GetStrategyAsset(ctx context.Context, req *connect.Request[antv1.GetStrategyAssetRequest]) (*connect.Response[antv1.StrategyAsset], error) {
backend/internal/connect/strategy/strategy_backtest_crud.go:126:func (s *StrategyExecutionServer) WatchBacktestRun(ctx context.Context, req *connect.Request[antv1.WatchBacktestRunRequest], stream *connect.ServerStream[antv1.BacktestRunUpdate]) error {
backend/internal/connect/strategy/strategy_backtest_crud.go:198:func (s *StrategyExecutionServer) CancelBacktestRun(ctx context.Context, req *connect.Request[antv1.CancelBacktestRunRequest]) (*connect.Response[antv1.CancelBacktestRunResponse], error) {
backend/internal/connect/strategy/strategy_backtest_crud.go:214:func (s *StrategyExecutionServer) DeleteBacktestRun(ctx context.Context, req *connect.Request[antv1.DeleteBacktestRunRequest]) (*connect.Response[antv1.DeleteBacktestRunResponse], error) {
backend/internal/connect/strategy/strategy_backtest_crud.go:232:func (s *StrategyExecutionServer) DeleteBacktestRuns(ctx context.Context, req *connect.Request[antv1.DeleteBacktestRunsRequest]) (*connect.Response[antv1.DeleteBacktestRunsResponse], error) {
backend/internal/connect/strategy/strategy_backtest_crud.go:26:func (s *StrategyExecutionServer) StartBacktestRun(ctx context.Context, req *connect.Request[antv1.StartBacktestRunRequest]) (*connect.Response[antv1.StartBacktestRunResponse], error) {
backend/internal/connect/strategy/strategy_backtest_crud.go:74:func (s *StrategyExecutionServer) GetBacktestRun(ctx context.Context, req *connect.Request[antv1.GetBacktestRunRequest]) (*connect.Response[antv1.GetBacktestRunResponse], error) {
backend/internal/connect/strategy/strategy_backtest_crud.go:93:func (s *StrategyExecutionServer) ListBacktestRuns(ctx context.Context, req *connect.Request[antv1.ListBacktestRunsRequest]) (*connect.Response[antv1.ListBacktestRunsResponse], error) {
backend/internal/connect/strategy/strategy_execution_handler.go:110:func (s *StrategyExecutionServer) Execute(ctx context.Context, req *connect.Request[antv1.ExecuteStrategyRequest]) (*connect.Response[antv1.ExecuteStrategyResponse], error) {
backend/internal/connect/strategy/strategy_execution_handler.go:133:func (s *StrategyExecutionServer) Validate(ctx context.Context, req *connect.Request[antv1.ValidateStrategyRequest]) (*connect.Response[antv1.ValidateStrategyResponse], error) {
backend/internal/connect/strategy/strategy_execution_handler.go:162:func (s *StrategyExecutionServer) Backtest(ctx context.Context, req *connect.Request[antv1.BacktestStrategyRequest]) (*connect.Response[antv1.BacktestStrategyResponse], error) {
backend/internal/connect/strategy/strategy_execution_handler.go:177:func (s *StrategyExecutionServer) GetTemplates(_ context.Context, _ *connect.Request[emptypb.Empty]) (*connect.Response[antv1.GetStrategyTemplatesResponse], error) {
backend/internal/connect/strategy/strategy_execution_handler.go:183:func (s *StrategyExecutionServer) ExecuteLive(ctx context.Context, req *connect.Request[antv1.ExecuteLiveRequest]) (*connect.Response[antv1.ExecuteLiveResponse], error) {
backend/internal/connect/strategy/strategy_execution_handler.go:198:func (s *StrategyExecutionServer) TranspileCode(ctx context.Context, req *connect.Request[antv1.TranspileCodeRequest]) (*connect.Response[antv1.TranspileCodeResponse], error) {
backend/internal/connect/strategy/strategy_execution_handler.go:228:func (s *StrategyExecutionServer) AnalyzeImportCode(ctx context.Context, req *connect.Request[antv1.AnalyzeImportCodeRequest]) (*connect.Response[antv1.AnalyzeImportCodeResponse], error) {
backend/internal/connect/strategy/strategy_execution_handler.go:277:func (s *StrategyExecutionServer) GenerateImportCode(ctx context.Context, req *connect.Request[antv1.GenerateImportCodeRequest]) (*connect.Response[antv1.GenerateImportCodeResponse], error) {
backend/internal/connect/strategy/strategy_execution_handler.go:325:func (s *StrategyExecutionServer) ImportStrategy(ctx context.Context, req *connect.Request[antv1.ImportStrategyRequest]) (*connect.Response[antv1.ImportStrategyResponse], error) {
backend/internal/connect/strategy/strategy_execution_handler.go:397:func (s *StrategyExecutionServer) SetPgListen(l *pglisten.Listener) { s.pgListen = l }
backend/internal/connect/strategy/strategy_execution_handler.go:59:func (s *StrategyExecutionServer) SetMarketDataRepo(r repository.MarketDataStore) {
backend/internal/connect/strategy/strategy_execution_handler.go:73:func (s *StrategyExecutionServer) SetBarSource(bs BarSource)                 { s.barSource = bs }
backend/internal/connect/strategy/strategy_execution_handler.go:74:func (s *StrategyExecutionServer) SetMtHub(h *mthub.MtHubService)            { s.mtHub = h }
backend/internal/connect/strategy/strategy_execution_handler.go:75:func (s *StrategyExecutionServer) SetPaperEngine(pe PaperOrderExecutor)      { s.paperEngine = pe }
backend/internal/connect/strategy/strategy_execution_handler.go:76:func (s *StrategyExecutionServer) SetGoExecutor(ge *GoExecutor)              { s.goExecutor = ge }
backend/internal/connect/strategy/strategy_execution_handler.go:80:func (s *StrategyExecutionServer) SetGate(g *risk.Gate) { s.gate = g }
backend/internal/connect/strategy/strategy_execution_handler.go:83:func (s *StrategyExecutionServer) AddGateRule(r risk.Rule) {
backend/internal/connect/strategy/strategy_execution_handler.go:90:func (s *StrategyExecutionServer) SetAccountProvider(p AccountStateProvider) { s.accountProvider = p }
backend/internal/connect/strategy/strategy_execution_handler.go:98:func (s *StrategyExecutionServer) SetNotificationSender(ns *notification.Sender) { s.notifSender = ns }
backend/internal/connect/strategy/strategy_execution_handler.go:99:func (s *StrategyExecutionServer) SetOnBacktestComplete(fn func(context.Context, *repository.BacktestRun)) { s.onBacktestComplete = fn }
backend/internal/connect/strategy/strategy_experiment_handler.go:101:func (s *StrategyExperimentServer) SubmitStrategyExperiment(ctx context.Context, req *connect.Request[antv1.SubmitStrategyExperimentRequest]) (*connect.Response[antv1.SubmitStrategyExperimentResponse], error) {
backend/internal/connect/strategy/strategy_experiment_handler.go:137:func (s *StrategyExperimentServer) GetStrategyExperiment(ctx context.Context, req *connect.Request[antv1.GetStrategyExperimentRequest]) (*connect.Response[antv1.StrategyExperiment], error) {
backend/internal/connect/strategy/strategy_experiment_handler.go:150:func (s *StrategyExperimentServer) ListStrategyExperiments(ctx context.Context, req *connect.Request[antv1.ListStrategyExperimentsRequest]) (*connect.Response[antv1.ListStrategyExperimentsResponse], error) {
backend/internal/connect/strategy/strategy_experiment_handler.go:164:func (s *StrategyExperimentServer) CancelStrategyExperiment(ctx context.Context, req *connect.Request[antv1.CancelStrategyExperimentRequest]) (*connect.Response[antv1.StrategyExperiment], error) {
backend/internal/connect/strategy/strategy_experiment_handler.go:177:func (s *StrategyExperimentServer) ListExperimentCandidates(ctx context.Context, req *connect.Request[antv1.ListExperimentCandidatesRequest]) (*connect.Response[antv1.ListExperimentCandidatesResponse], error) {
backend/internal/connect/strategy/strategy_experiment_handler.go:194:func (s *StrategyExperimentServer) GetExperimentCandidate(ctx context.Context, req *connect.Request[antv1.GetExperimentCandidateRequest]) (*connect.Response[antv1.StrategyExperimentCandidate], error) {
backend/internal/connect/strategy/strategy_experiment_handler.go:207:func (s *StrategyExperimentServer) PromoteCandidateToDraft(ctx context.Context, req *connect.Request[antv1.PromoteCandidateToDraftRequest]) (*connect.Response[antv1.PromoteCandidateToDraftResponse], error) {
backend/internal/connect/strategy/strategy_experiment_handler.go:225:func (s *StrategyExperimentServer) WatchExperiment(ctx context.Context, req *connect.Request[antv1.WatchExperimentRequest], stream *connect.ServerStream[antv1.WatchExperimentEvent]) error {
backend/internal/connect/strategy/strategy_experiment_handler.go:273:func (s *StrategyExperimentServer) SetPgListen(l *pglisten.Listener) { s.pgListen = l }
backend/internal/connect/strategy/strategy_experiment_worker.go:44:func (w *ExperimentWorker) Start(ctx context.Context) {
backend/internal/connect/strategy/strategy_experiment_worker.go:62:func (w *ExperimentWorker) SetPgListen(l *pglisten.Listener) { w.pgListen = l }
backend/internal/connect/strategy/strategy_experiment_worker.go:77:func (w *ExperimentWorker) Stop() { close(w.stopCh) }
backend/internal/connect/strategy/strategy_experiment_worker.go:80:func (w *ExperimentWorker) SetAIService(svc *systemai.Service) { w.systemAISvc = svc }
backend/internal/connect/strategy/strategy_handler.go:33:func (s *StrategyServer) SetCodeAccessChecker(c CodeAccessChecker) { s.codeAccess = c }
backend/internal/connect/strategy/strategy_handler.go:42:func (s *StrategyServer) CancelTemplateDraft(ctx context.Context, req *connect.Request[antv1.CancelTemplateDraftRequest]) (*connect.Response[emptypb.Empty], error) {
backend/internal/connect/strategy/strategy_handler.go:53:func (s *StrategyServer) SetEngine(e *ScheduleEngine) { s.engine = e }
backend/internal/connect/strategy/strategy_handler.go:69:func (s *StrategyServer) SetPgListen(l *pglisten.Listener) { s.pgListen = l }
backend/internal/connect/strategy/strategy_schedules.go:135:func (s *StrategyServer) DeleteSchedule(ctx context.Context, req *connect.Request[antv1.DeleteScheduleRequest]) (*connect.Response[emptypb.Empty], error) {
backend/internal/connect/strategy/strategy_schedules.go:149:func (s *StrategyServer) ToggleSchedule(ctx context.Context, req *connect.Request[antv1.ToggleScheduleRequest]) (*connect.Response[antv1.StrategySchedule], error) {
backend/internal/connect/strategy/strategy_schedules.go:182:func (s *StrategyServer) WatchSchedules(ctx context.Context, req *connect.Request[antv1.WatchSchedulesRequest], stream *connect.ServerStream[antv1.WatchSchedulesEvent]) error {
backend/internal/connect/strategy/strategy_schedules.go:22:func (s *StrategyServer) ListSchedules(ctx context.Context, req *connect.Request[antv1.ListSchedulesRequest]) (*connect.Response[antv1.ListSchedulesResponse], error) {
backend/internal/connect/strategy/strategy_schedules.go:34:func (s *StrategyServer) GetSchedule(ctx context.Context, req *connect.Request[antv1.GetScheduleRequest]) (*connect.Response[antv1.StrategySchedule], error) {
backend/internal/connect/strategy/strategy_schedules.go:46:func (s *StrategyServer) CreateSchedule(ctx context.Context, req *connect.Request[antv1.CreateScheduleRequest]) (*connect.Response[antv1.StrategySchedule], error) {
backend/internal/connect/strategy/strategy_schedules.go:94:func (s *StrategyServer) UpdateSchedule(ctx context.Context, req *connect.Request[antv1.UpdateScheduleRequest]) (*connect.Response[antv1.StrategySchedule], error) {
backend/internal/connect/strategy/strategy_signals.go:16:func (s *StrategyServer) RunBacktest(
backend/internal/connect/strategy/strategy_signals.go:35:func (s *StrategyServer) ListSignals(
backend/internal/connect/strategy/strategy_signals.go:52:func (s *StrategyServer) ExecuteSignal(
backend/internal/connect/strategy/strategy_signals.go:73:func (s *StrategyServer) ConfirmSignal(
backend/internal/connect/strategy/strategy_signals.go:86:func (s *StrategyServer) CancelSignal(
backend/internal/connect/strategy/strategy_template_handlers.go:140:func (s *StrategyServer) DeleteTemplate(ctx context.Context, req *connect.Request[antv1.DeleteTemplateRequest]) (*connect.Response[emptypb.Empty], error) {
backend/internal/connect/strategy/strategy_template_handlers.go:155:func (s *StrategyServer) CreateTemplateDraft(ctx context.Context, req *connect.Request[antv1.CreateTemplateDraftRequest]) (*connect.Response[antv1.StrategyTemplate], error) {
backend/internal/connect/strategy/strategy_template_handlers.go:169:func (s *StrategyServer) UpdateTemplateDraft(ctx context.Context, req *connect.Request[antv1.UpdateTemplateDraftRequest]) (*connect.Response[antv1.StrategyTemplate], error) {
backend/internal/connect/strategy/strategy_template_handlers.go:204:func (s *StrategyServer) PublishTemplateDraft(ctx context.Context, req *connect.Request[antv1.PublishTemplateDraftRequest]) (*connect.Response[antv1.StrategyTemplate], error) {
backend/internal/connect/strategy/strategy_template_handlers.go:48:func (s *StrategyServer) ListTemplates(ctx context.Context, req *connect.Request[antv1.ListTemplatesRequest]) (*connect.Response[antv1.ListTemplatesResponse], error) {
backend/internal/connect/strategy/strategy_template_handlers.go:61:func (s *StrategyServer) GetTemplate(ctx context.Context, req *connect.Request[antv1.GetTemplateRequest]) (*connect.Response[antv1.StrategyTemplate], error) {
backend/internal/connect/strategy/strategy_template_handlers.go:77:func (s *StrategyServer) CreateTemplate(ctx context.Context, req *connect.Request[antv1.CreateTemplateRequest]) (*connect.Response[antv1.StrategyTemplate], error) {
backend/internal/connect/strategy/strategy_template_handlers.go:99:func (s *StrategyServer) UpdateTemplate(ctx context.Context, req *connect.Request[antv1.UpdateTemplateRequest]) (*connect.Response[antv1.StrategyTemplate], error) {
backend/internal/connect/strategy/strategy_templates.go:104:func (s *MACrossStrategy) OnDeinit(ctx sdk.Context, reason string) error {
backend/internal/connect/strategy/strategy_templates.go:131:func (s *RSIStrategy) OnInit(ctx sdk.Context) error {
backend/internal/connect/strategy/strategy_templates.go:141:func (s *RSIStrategy) OnBar(ctx sdk.Context, timeframe string) (*sdk.Signal, error) {
backend/internal/connect/strategy/strategy_templates.go:199:func (s *RSIStrategy) OnDeinit(ctx sdk.Context, reason string) error {
backend/internal/connect/strategy/strategy_templates.go:225:func (s *BollingerStrategy) OnInit(ctx sdk.Context) error {
backend/internal/connect/strategy/strategy_templates.go:234:func (s *BollingerStrategy) OnBar(ctx sdk.Context, timeframe string) (*sdk.Signal, error) {
backend/internal/connect/strategy/strategy_templates.go:292:func (s *BollingerStrategy) OnDeinit(ctx sdk.Context, reason string) error {
backend/internal/connect/strategy/strategy_templates.go:33:func (s *MACrossStrategy) OnInit(ctx sdk.Context) error {
backend/internal/connect/strategy/strategy_templates.go:42:func (s *MACrossStrategy) OnBar(ctx sdk.Context, timeframe string) (*sdk.Signal, error) {
backend/internal/mthub/broker_types.go:126:func (b *BarBroker) Publish(ev *BarUpdate) {
backend/internal/mthub/broker_types.go:153:func (b *BarBroker) DroppedBars(accountID string) int64 {
backend/internal/mthub/broker_types.go:158:func (b *BarBroker) Subscribe(accountID string) (<-chan *BarUpdate, func()) {
backend/internal/mthub/broker_types.go:201:func (b *AccountStatusBroker) Publish(ev *AccountStatusEvent) {
backend/internal/mthub/broker_types.go:215:func (b *AccountStatusBroker) Subscribe(accountID string) (<-chan *AccountStatusEvent, func()) {
backend/internal/mthub/broker_types.go:63:func (b *PositionSnapshotBroker) Publish(ev *PositionSnapshot) {
backend/internal/mthub/broker_types.go:77:func (b *PositionSnapshotBroker) Subscribe(accountID string) (<-chan *PositionSnapshot, func()) {
backend/internal/mthub/derived_state.go:109:func (dc *DerivedComputer) Start() {
backend/internal/mthub/derived_state.go:114:func (dc *DerivedComputer) Stop() {
backend/internal/mthub/derived_state.go:119:func (dc *DerivedComputer) State() *DerivedState {
backend/internal/mthub/derived_state.go:60:func (d *DerivedState) Update(accounts map[string]*AccountDerivedState, totalExposure, totalMargin, grossPnL, netPnL, var95 float64) {
backend/internal/mthub/derived_state.go:74:func (d *DerivedState) Get() (accounts map[string]*AccountDerivedState, totalExposure, totalMargin, grossPnL, netPnL, var95 float64, lastUpdated time.Time) {
backend/internal/mthub/derived_state.go:81:func (d *DerivedState) GetAccount(accountID string) *AccountDerivedState {
backend/internal/mthub/hub_estimator.go:152:func (e *HubCostEstimator) Refresh(symbol string) {
backend/internal/mthub/hub_estimator.go:46:func (e *HubCostEstimator) Estimate(ctx context.Context, params costsvc.EstimateParams) costsvc.CostBreakdown {
backend/internal/mthub/idempotency.go:151:func (g *IdempotencyGuard) CheckAndSet(ctx context.Context, accountID, clientID string, ticket int64) (isDup bool, existingTicket int64, err error) {
backend/internal/mthub/idempotency.go:167:func (g *IdempotencyGuard) SetTicket(ctx context.Context, accountID, clientID string, ticket int64) error {
backend/internal/mthub/idempotency.go:172:func (g *IdempotencyGuard) DeleteKey(ctx context.Context, accountID, clientID string) {
backend/internal/mthub/idempotency.go:53:func (g *ThreeLayerGuard) CheckAndSet(ctx context.Context, accountID, clientID string, ticket int64) (isDup bool, existingTicket int64, err error) {
backend/internal/mthub/idempotency.go:87:func (g *ThreeLayerGuard) Confirm(ctx context.Context, accountID, clientID string, ticket int64) error {
backend/internal/mthub/oms_writer.go:116:func (w *OmsWriter) InsertOrder(ctx context.Context, orderID, accountID, platform, symbol string, orderType int16, volume, price, stopLoss, takeProfit decimal.Decimal) error {
backend/internal/mthub/oms_writer.go:133:func (w *OmsWriter) Transition(ctx context.Context, orderID, accountID string, current, next OMSState) error {
backend/internal/mthub/oms_writer.go:98:func (w *OmsWriter) SetOrderEventBroker(b *OrderEventBroker) {
backend/internal/mthub/reconcile_gate.go:24:func (g *ReconcileGate) EnterReconciling(accountID string) {
backend/internal/mthub/reconcile_gate.go:31:func (g *ReconcileGate) MarkReconciled(accountID string) {
backend/internal/mthub/reconcile_gate.go:38:func (g *ReconcileGate) CanAccept(accountID string) bool {
backend/internal/mthub/reconcile_gate.go:45:func (g *ReconcileGate) IsReconciling(accountID string) bool {
backend/internal/mthub/reconcile_gate.go:52:func (g *ReconcileGate) ReconcilingCount() int {
backend/internal/mthub/reconcile_gate.go:59:func (g *ReconcileGate) EnterAll(accountIDs []string) {
backend/internal/mthub/reconciliation.go:31:func (r *ReconciliationLoop) Start(ctx context.Context) {
backend/internal/mthub/reconciliation.go:47:func (r *ReconciliationLoop) ReconcileAccount(ctx context.Context, accountID string) {
backend/internal/mthub/reconciliation.go:55:func (r *ReconciliationLoop) TriggerReconcile(accountID string) {
backend/internal/mthub/service.go:102:func (s *MtHubService) BrokerRegistry() BrokerRegistry { return s.brokerRegistry }
backend/internal/mthub/service.go:108:func (s *MtHubService) SetAccountOwnerVerifier(v AccountOwnerVerifier) { s.accountOwnerVerifier = v }
backend/internal/mthub/service.go:111:func (s *MtHubService) SetLogger(l *zap.Logger) { s.logger = l }
backend/internal/mthub/service.go:114:func (s *MtHubService) SetBarBroker(b *BarBroker) { s.barBroker = b }
backend/internal/mthub/service.go:117:func (s *MtHubService) SetStatusBroker(b *AccountStatusBroker) { s.statusBroker = b }
backend/internal/mthub/service.go:120:func (s *MtHubService) PublishBar(ev *BarUpdate) {
backend/internal/mthub/service.go:127:func (s *MtHubService) SubscribeBarUpdates(accountID string) (<-chan *BarUpdate, func()) {
backend/internal/mthub/service.go:137:func (s *MtHubService) PublishAccountStatus(ev *AccountStatusEvent) {
backend/internal/mthub/service.go:144:func (s *MtHubService) SubscribeAccountStatus(accountID string) (<-chan *AccountStatusEvent, func()) {
backend/internal/mthub/service.go:157:func (s *MtHubService) Platform(accountID string) string {
backend/internal/mthub/service.go:167:func (s *MtHubService) SessionState(ctx context.Context, accountID string) string {
backend/internal/mthub/service.go:198:func (s *MtHubService) OpenedOrders(ctx context.Context, accountID string) ([]*OrderRecord, error) {
backend/internal/mthub/service.go:209:func (s *MtHubService) OrderHistory(ctx context.Context, accountID string, from, to time.Time) ([]*OrderRecord, error) {
backend/internal/mthub/service.go:218:func (s *MtHubService) SymbolParams(ctx context.Context, accountID string, canonicals []string) ([]*SymbolParam, error) {
backend/internal/mthub/service.go:227:func (s *MtHubService) PriceHistory(ctx context.Context, accountID, symbol, period string, from, to int64, count int) ([]*Bar, error) {
backend/internal/mthub/service.go:236:func (s *MtHubService) SymbolList(ctx context.Context, accountID string) ([]string, error) {
backend/internal/mthub/service.go:247:func (s *MtHubService) SubscribeSymbols(ctx context.Context, accountID string, symbols []string) error {
backend/internal/mthub/service.go:256:func (s *MtHubService) SubscribeUserOrderEvents(ctx context.Context, userID string) (<-chan *OrderEvent, func()) {
backend/internal/mthub/service.go:261:func (s *MtHubService) PublishAccountProfit(ev *AccountProfitEvent) {
backend/internal/mthub/service.go:266:func (s *MtHubService) SubscribeAccountProfit(ctx context.Context, accountID string) (<-chan *AccountProfitEvent, func()) {
backend/internal/mthub/service.go:271:func (s *MtHubService) PublishPositionSnapshot(ev *PositionSnapshot) {
backend/internal/mthub/service.go:276:func (s *MtHubService) SubscribePositionSnapshots(ctx context.Context, accountID string) (<-chan *PositionSnapshot, func()) {
backend/internal/mthub/service.go:75:func (s *MtHubService) SetUserLimiter(l *usermgr.UserLimiter) { s.userLimiter = l }
backend/internal/mthub/service.go:78:func (s *MtHubService) SetCostEstimator(e costsvc.CostEstimator) { s.costEstimator = e }
backend/internal/mthub/service.go:85:func (s *MtHubService) SetAccountStateProvider(p AccountStateProvider) { s.accountStateProvider = p }
backend/internal/mthub/service.go:88:func (s *MtHubService) SetGate(g *risk.Gate) { s.gate = g }
backend/internal/mthub/service.go:91:func (s *MtHubService) SetOmsWriter(w *OmsWriter) { s.omsWriter = w }
backend/internal/mthub/service.go:94:func (s *MtHubService) SetKillSwitch(ks KillSwitchGate) { s.killSwitch = ks }
backend/internal/mthub/service.go:99:func (s *MtHubService) SetBrokerRegistry(r BrokerRegistry) { s.brokerRegistry = r }
backend/internal/mthub/service_orders.go:19:func (s *MtHubService) PlaceOrder(ctx context.Context, req *OrderRequest) (*OrderRecord, error) {
backend/internal/mthub/service_orders_close.go:17:func (s *MtHubService) CloseOrder(ctx context.Context, accountID string, ticket int64, lots decimal.Decimal) error {
backend/internal/mthub/service_orders_modify.go:18:func (s *MtHubService) ModifyOrder(ctx context.Context, accountID string, ticket int64, sl, tp, price decimal.Decimal) error {
backend/internal/mthub/state_cache.go:103:func (c *StateCache) GetPositionsByAccount(accountID string) []*PositionCacheEntry {
backend/internal/mthub/state_cache.go:117:func (c *StateCache) ApplyEvent(ev *TradeEvent) {
backend/internal/mthub/state_cache.go:186:func (c *StateCache) LoadFromRedis(ctx context.Context) error {
backend/internal/mthub/state_cache.go:207:func (c *StateCache) Stats() (orders, positions int) {
backend/internal/mthub/state_cache.go:76:func (c *StateCache) GetOrder(ticket int64) *OrderStateCacheEntry {
backend/internal/mthub/state_cache.go:83:func (c *StateCache) GetOrdersByAccount(accountID string) []*OrderStateCacheEntry {
backend/internal/mthub/state_cache.go:96:func (c *StateCache) GetPosition(accountID, canonical string) *PositionCacheEntry {
backend/internal/mthub/trade_event_store.go:95:func (s *TradeEventStore) Publish(ctx context.Context, ev *TradeEvent) error {
backend/internal/mthub/types.go:101:func (h *Hub) ActiveAccountIDs() []string {
backend/internal/mthub/types.go:115:func (e *HubError) Error() string { return "mthub: " + e.Msg }
backend/internal/mthub/types.go:148:func (b *AccountProfitBroker) Publish(ev *AccountProfitEvent) {
backend/internal/mthub/types.go:15:func (s *Session) IsExpired() bool {
backend/internal/mthub/types.go:162:func (b *AccountProfitBroker) Subscribe(accountID string) (<-chan *AccountProfitEvent, func()) {
backend/internal/mthub/types.go:190:func (b *OrderEventBroker) PublishEvent(userID string, ev *OrderEvent) {
backend/internal/mthub/types.go:203:func (b *OrderEventBroker) Subscribe(userID string) (<-chan *OrderEvent, func()) {
backend/internal/mthub/types.go:38:func (h *Hub) Register(id string, s *Session, e OrderExecutor) {
backend/internal/mthub/types.go:52:func (h *Hub) WaitSession(id string) <-chan struct{} {
backend/internal/mthub/types.go:65:func (h *Hub) Get(id string) OrderExecutor {
backend/internal/mthub/types.go:71:func (h *Hub) EnsureSession(ctx context.Context, id string) (*Session, error) {
backend/internal/mthub/types.go:86:func (h *Hub) RemoveSession(id string) {
backend/internal/mthub/types.go:93:func (h *Hub) CloseSession(ctx context.Context, id string) error {
backend/internal/paper/engine.go:121:func (e *PaperEngine) ClosePaperOrder(ctx context.Context, accountID, symbol string) error {
backend/internal/paper/engine.go:143:func (e *PaperEngine) ModifyPaperOrder(ctx context.Context, accountID, symbol string, sl, tp decimal.Decimal) error {
backend/internal/paper/engine.go:163:func (e *PaperEngine) CancelPaperOrder(ctx context.Context, accountID, symbol string) error {
backend/internal/paper/engine.go:184:func (e *PaperEngine) Subscribe(accountID string) (<-chan *repository.PaperAccount, func()) {
backend/internal/paper/engine.go:60:func (e *PaperEngine) PlacePaperOrder(ctx context.Context, accountID, symbol, side string,
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
backend/internal/risk/gate.go:101:func (g *Gate) SetAutotradeEnabled(fn func(userID string) bool) {
backend/internal/risk/gate.go:114:func (g *Gate) Evaluate(ctx context.Context, intent *antv1.OrderIntent, state *AccountState) *antv1.RiskDecision {
backend/internal/risk/gate.go:179:func (g *Gate) AddRule(rule Rule) {
backend/internal/risk/gate.go:186:func (g *Gate) Rules() []string {
backend/internal/risk/gate.go:228:func (e *AuditEntry) String() string {
backend/internal/risk/gate.go:94:func (g *Gate) SetKillSwitch(fn func() bool) {
backend/internal/risk/rule_user_config.go:31:func (r *UserRiskConfigRule) Name() string { return "user_risk_config" }
backend/internal/risk/rule_user_config.go:33:func (r *UserRiskConfigRule) Check(ctx context.Context, intent *antv1.OrderIntent, state *AccountState) *RuleResult {
backend/internal/risk/rules.go:112:func (r *DailyLossBreaker) Name() string { return "daily_loss" }
backend/internal/risk/rules.go:114:func (r *DailyLossBreaker) Check(_ context.Context, intent *antv1.OrderIntent, state *AccountState) *RuleResult {
backend/internal/risk/rules.go:133:func (r *DrawdownBreaker) Name() string { return "drawdown" }
backend/internal/risk/rules.go:135:func (r *DrawdownBreaker) Check(_ context.Context, intent *antv1.OrderIntent, state *AccountState) *RuleResult {
backend/internal/risk/rules.go:155:func (r *SymbolWhitelist) Name() string { return "symbol_whitelist" }
backend/internal/risk/rules.go:157:func (r *SymbolWhitelist) Check(_ context.Context, intent *antv1.OrderIntent, _ *AccountState) *RuleResult {
backend/internal/risk/rules.go:179:func (r *LeverageCap) Name() string { return "leverage_cap" }
backend/internal/risk/rules.go:181:func (r *LeverageCap) Check(_ context.Context, intent *antv1.OrderIntent, state *AccountState) *RuleResult {
backend/internal/risk/rules.go:203:func (r *OrderFrequencyLimit) Name() string { return "order_frequency" }
backend/internal/risk/rules.go:205:func (r *OrderFrequencyLimit) Check(_ context.Context, intent *antv1.OrderIntent, _ *AccountState) *RuleResult {
backend/internal/risk/rules.go:239:func (r *DuplicateProtection) Name() string { return "duplicate_protection" }
backend/internal/risk/rules.go:241:func (r *DuplicateProtection) Check(_ context.Context, intent *antv1.OrderIntent, _ *AccountState) *RuleResult {
backend/internal/risk/rules.go:282:func (r *MarginPreCheck) Name() string { return "margin_pre_check" }
backend/internal/risk/rules.go:284:func (r *MarginPreCheck) Check(_ context.Context, intent *antv1.OrderIntent, state *AccountState) *RuleResult {
backend/internal/risk/rules.go:42:func (r *MaxLotSize) Name() string { return "max_lot_size" }
backend/internal/risk/rules.go:44:func (r *MaxLotSize) Check(_ context.Context, intent *antv1.OrderIntent, _ *AccountState) *RuleResult {
backend/internal/risk/rules.go:62:func (r *MaxPositionCount) Name() string { return "max_position_count" }
backend/internal/risk/rules.go:64:func (r *MaxPositionCount) Check(_ context.Context, intent *antv1.OrderIntent, state *AccountState) *RuleResult {
backend/internal/risk/rules.go:83:func (r *MaxExposure) Name() string { return "max_exposure" }
backend/internal/risk/rules.go:85:func (r *MaxExposure) Check(_ context.Context, intent *antv1.OrderIntent, state *AccountState) *RuleResult {
backend/internal/risk/rules_risksvc.go:102:func (r *MarginFloorRule) Name() string { return "margin_floor" }
backend/internal/risk/rules_risksvc.go:104:func (r *MarginFloorRule) Check(_ context.Context, intent *antv1.OrderIntent, state *AccountState) *RuleResult {
backend/internal/risk/rules_risksvc.go:159:func (a *risksvcCapStoreAdapter) GetTier(_ context.Context, userID string) (CapabilityTier, error) {
backend/internal/risk/rules_risksvc.go:172:func (r *CapabilityTierRule) Name() string { return "capability_tier" }
backend/internal/risk/rules_risksvc.go:174:func (r *CapabilityTierRule) Check(ctx context.Context, intent *antv1.OrderIntent, _ *AccountState) *RuleResult {
backend/internal/risk/rules_risksvc.go:32:func (r *KycJurisdictionGateRule) Name() string { return "kyc_jurisdiction" }
backend/internal/risk/rules_risksvc.go:34:func (r *KycJurisdictionGateRule) Check(ctx context.Context, intent *antv1.OrderIntent, _ *AccountState) *RuleResult {
backend/internal/risk/rules_risksvc.go:68:func (r *ContractExpiryRule) Name() string { return "contract_expiry" }
backend/internal/risk/rules_risksvc.go:70:func (r *ContractExpiryRule) Check(ctx context.Context, intent *antv1.OrderIntent, _ *AccountState) *RuleResult {
backend/internal/risksvc/block_allocator.go:101:func (a *VWAPAllocator) Name() string { return "vwap" }
backend/internal/risksvc/block_allocator.go:103:func (a *VWAPAllocator) Allocate(_ context.Context, totalVolume float64, accounts []AllocAccount) map[string]float64 {
backend/internal/risksvc/block_allocator.go:25:func (a *ProRataAllocator) Name() string { return "pro_rata" }
backend/internal/risksvc/block_allocator.go:27:func (a *ProRataAllocator) Allocate(_ context.Context, totalVolume float64, accounts []AllocAccount) map[string]float64 {
backend/internal/risksvc/block_allocator.go:65:func (a *FIFOAllocator) Name() string { return "fifo" }
backend/internal/risksvc/block_allocator.go:67:func (a *FIFOAllocator) Allocate(_ context.Context, totalVolume float64, accounts []AllocAccount) map[string]float64 {
backend/internal/risksvc/capability.go:107:func (s *CapabilityStore) Set(c *Capability) {
backend/internal/risksvc/capability.go:115:func (s *CapabilityStore) LoadFromPG(ctx context.Context, rows interface{ Scan(dest ...interface{}) error; Next() bool; Close() }) error {
backend/internal/risksvc/capability.go:155:func (s *CapabilityStore) Count() int {
backend/internal/risksvc/capability.go:61:func (c *Capability) HasOrderType(ot string) bool {
backend/internal/risksvc/capability.go:74:func (c *Capability) TierCheck() *PreCheckResult {
backend/internal/risksvc/capability.go:97:func (s *CapabilityStore) Get(userID string) *Capability {
backend/internal/risksvc/engine.go:22:func (e *Engine) SetUserLimiter(l *usermgr.UserLimiter) { e.userLimiter = l }
backend/internal/risksvc/engine.go:27:func (e *Engine) Evaluate(ctx context.Context, req *CheckRequest) *CheckResult {
backend/internal/risksvc/engine.go:49:func (e *Engine) Rules() []string {
backend/internal/risksvc/hardlimit.go:107:func (r *KillSwitchRule) Name() string { return "kill_switch" }
backend/internal/risksvc/hardlimit.go:109:func (r *KillSwitchRule) Check(_ context.Context, req *HardLimitRequest) error {
backend/internal/risksvc/hardlimit.go:130:func (r *ContractExpiryRule) Name() string { return "contract_expiry" }
backend/internal/risksvc/hardlimit.go:132:func (r *ContractExpiryRule) Check(_ context.Context, req *HardLimitRequest) error {
backend/internal/risksvc/hardlimit.go:165:func (e *HardLimitEvaluator) Evaluate(ctx context.Context, req *HardLimitRequest) error {
backend/internal/risksvc/hardlimit.go:181:func (e *HardLimitError) Error() string {
backend/internal/risksvc/hardlimit.go:54:func (r *KycJurisdictionRule) Name() string { return "kyc_jurisdiction" }
backend/internal/risksvc/hardlimit.go:56:func (r *KycJurisdictionRule) Check(ctx context.Context, req *HardLimitRequest) error {
backend/internal/risksvc/hardlimit.go:72:func (r *MarginFloorRule) Name() string { return "margin_floor" }
backend/internal/risksvc/hardlimit.go:74:func (r *MarginFloorRule) Check(_ context.Context, req *HardLimitRequest) error {
backend/internal/risksvc/jurisdiction.go:185:func (r *MaxMindGeoIPResolver) CountryCode(ipStr string) (string, error) {
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
backend/internal/risksvc/kelly_sizer.go:42:func (s *KellyFractionSizer) Name() string { return "kelly_fraction" }
backend/internal/risksvc/kelly_sizer.go:44:func (s *KellyFractionSizer) Size(_ context.Context, req *SizerRequest) (*SizerResult, error) {
backend/internal/risksvc/pipeline.go:94:func (p *SignalPipeline) Process(ctx context.Context, sig *SignalRequest) *SignalResult {
backend/internal/risksvc/platform_aggregator.go:102:func (a *PlatformAggregator) Recalculate() *PlatformExposure {
backend/internal/risksvc/platform_aggregator.go:138:func (a *PlatformAggregator) GetSnapshot() *PlatformExposure {
backend/internal/risksvc/platform_aggregator.go:143:func (a *PlatformAggregator) NetExposureForSymbol(canonical string) float64 {
backend/internal/risksvc/platform_aggregator.go:153:func (a *PlatformAggregator) StartRefreshLoop(interval time.Duration) {
backend/internal/risksvc/platform_aggregator.go:174:func (a *PlatformAggregator) Shutdown() {
backend/internal/risksvc/platform_aggregator.go:73:func (a *PlatformAggregator) UpdatePosition(accountID string, pos *AggregatorPosition) {
backend/internal/risksvc/platform_aggregator.go:85:func (a *PlatformAggregator) ClearAccount(accountID string) {
backend/internal/risksvc/platform_aggregator.go:93:func (a *PlatformAggregator) SetBrokerLimits(limits map[string]float64) {
backend/internal/risksvc/platform_limits.go:35:func (l *PlatformLimits) Check(exposure *PlatformExposure) *PlatformLimitResult {
backend/internal/risksvc/rules.go:102:func (r *CanonicalAuth) Name() string { return "canonical_auth" }
backend/internal/risksvc/rules.go:103:func (r *CanonicalAuth) Check(_ context.Context, req *CheckRequest) *CheckResult {
backend/internal/risksvc/rules.go:13:func (r *MaxPosition) Name() string { return "max_position" }
backend/internal/risksvc/rules.go:14:func (r *MaxPosition) Check(_ context.Context, req *CheckRequest) *CheckResult {
backend/internal/risksvc/rules.go:30:func (r *DailyLoss) Name() string { return "daily_loss" }
backend/internal/risksvc/rules.go:31:func (r *DailyLoss) Check(_ context.Context, req *CheckRequest) *CheckResult {
backend/internal/risksvc/rules.go:50:func (r *Drawdown) Name() string { return "drawdown" }
backend/internal/risksvc/rules.go:52:func (r *Drawdown) Check(_ context.Context, req *CheckRequest) *CheckResult {
backend/internal/risksvc/rules.go:70:func (r *Session) Name() string { return "session" }
backend/internal/risksvc/rules.go:71:func (r *Session) Check(_ context.Context, req *CheckRequest) *CheckResult {
backend/internal/risksvc/rules.go:83:func (r *Margin) Name() string { return "margin" }
backend/internal/risksvc/rules.go:84:func (r *Margin) Check(_ context.Context, req *CheckRequest) *CheckResult {
backend/internal/risksvc/vol_target_sizer.go:43:func (s *VolTargetSizer) Name() string { return "vol_target" }
backend/internal/risksvc/vol_target_sizer.go:45:func (s *VolTargetSizer) Size(_ context.Context, req *SizerRequest) (*SizerResult, error) {
backend/internal/service/account_lifecycle.go:115:func (s *AccountService) UpdateAccountInfo(ctx context.Context, userID uuid.UUID, id string, balance, equity, credit, margin, freeMargin float64, leverage int64, currency string) error {
backend/internal/service/account_lifecycle.go:136:func (s *AccountService) UpdateAccountMetrics(ctx context.Context, userID uuid.UUID, id string, balance, equity, credit, margin, freeMargin, marginLevel float64) error {
backend/internal/service/account_lifecycle.go:15:func (s *AccountService) ConnectAccount(ctx context.Context, userID uuid.UUID, accountID string) error {
backend/internal/service/account_lifecycle.go:178:func (s *AccountService) UpdateBrokerThresholds(ctx context.Context, id string, marginCallPct, stopOutPct float64) error {
backend/internal/service/account_lifecycle.go:189:func (s *AccountService) UpdateTradingPassword(ctx context.Context, userID uuid.UUID, id, oldPassword, newPassword string) error {
backend/internal/service/account_lifecycle.go:205:func (s *AccountService) CleanupOldSnapshots(ctx context.Context, log *zap.Logger) {
backend/internal/service/account_lifecycle.go:29:func (s *AccountService) DisconnectAccount(ctx context.Context, userID uuid.UUID, id string) error {
backend/internal/service/account_lifecycle.go:42:func (s *AccountService) DisconnectAccountByID(ctx context.Context, id string) error {
backend/internal/service/account_lifecycle.go:53:func (s *AccountService) ReconnectAccount(ctx context.Context, userID uuid.UUID, id string) error {
backend/internal/service/account_lifecycle.go:64:func (s *AccountService) MarkAccountNeedsRebind(ctx context.Context, userID uuid.UUID, id string) error {
backend/internal/service/account_lifecycle.go:75:func (s *AccountService) GetAccountCredentials(ctx context.Context, userID uuid.UUID, id string) (*AccountCredentials, error) {
backend/internal/service/account_lifecycle.go:93:func (s *AccountService) UpdateAccountInfoTx(ctx context.Context, tx pgx.Tx, userID uuid.UUID, id string, balance, equity, credit, margin, freeMargin float64, leverage int64, currency string, isInvestor bool) error {
backend/internal/service/account_number.go:215:func (s *AccountNumberService) AssignAccountNumber(ctx context.Context, num string) error {
backend/internal/service/account_number.go:230:func (s *AccountNumberService) SetAccountNumber(ctx context.Context, userID, num string) error {
backend/internal/service/account_number.go:45:func (s *AccountNumberService) GenerateAccountNumber(ctx context.Context) (string, error) {
backend/internal/service/account_number.go:87:func (s *AccountNumberService) IsAccountNumberAvailable(ctx context.Context, num string) (bool, error) {
backend/internal/service/account_number.go:93:func (s *AccountNumberService) IsAccountNumberAvailableExcluding(ctx context.Context, num, excludeUserID string) (bool, error) {
backend/internal/service/account_service.go:105:func (s *AccountService) ListAccounts(ctx context.Context, userID uuid.UUID) ([]AccountDTO, error) {
backend/internal/service/account_service.go:118:func (s *AccountService) GetAccount(ctx context.Context, userID uuid.UUID, accountID string) (*AccountDTO, error) {
backend/internal/service/account_service.go:132:func (s *AccountService) BeginTx(ctx context.Context) (pgx.Tx, error) {
backend/internal/service/account_service.go:137:func (s *AccountService) CreateAccountTx(ctx context.Context, tx pgx.Tx, userID uuid.UUID, login, password, mtType, brokerCompany, brokerServer, brokerHost string) (string, error) {
backend/internal/service/account_service.go:155:func (s *AccountService) CreateAccount(ctx context.Context, userID uuid.UUID, login, password, mtType, brokerCompany, brokerServer, brokerHost string) (string, error) {
backend/internal/service/account_service.go:173:func (s *AccountService) UpdateAccount(ctx context.Context, userID uuid.UUID, id, brokerCompany, brokerServer, brokerHost string, isDisabled *bool) error {
backend/internal/service/account_service.go:199:func (s *AccountService) DeleteAccount(ctx context.Context, userID uuid.UUID, id string) error {
backend/internal/service/account_service.go:60:func (s *AccountService) SetLogger(log *zap.Logger) { s.log = log }
backend/internal/service/account_snapshot.go:104:func (s *AccountService) UpdateSummaryCache(userID, accountID string, balance, equity float64, status string) {
backend/internal/service/account_snapshot.go:140:func (s *AccountService) InvalidateSummaryCache(userID string) {
backend/internal/service/account_snapshot.go:149:func (s *AccountService) GetUserAccountsSummary(ctx context.Context, userID string) (*UserAccountsSummary, error) {
backend/internal/service/account_snapshot.go:15:func (s *AccountService) RecordBalanceSnapshot(ctx context.Context, id string, userID string, balance, equity, margin, freeMargin float64) error {
backend/internal/service/account_snapshot.go:46:func (s *AccountService) UserOwnsAccount(ctx context.Context, userID, accountID string) (bool, error) {
backend/internal/service/account_snapshot.go:59:func (s *AccountService) GetUserAccountIDs(ctx context.Context, userID string) ([]string, error) {
backend/internal/service/account_snapshot.go:77:func (s *AccountService) GetUserAccountSnapshots(ctx context.Context, userID string) ([]AccountSnapshot, error) {
backend/internal/service/account_sync.go:127:func (s *AccountSyncService) SetNotificationSender(ns *notification.Sender) { s.notifSender = ns }
backend/internal/service/account_sync.go:133:func (s *AccountSyncService) SyncAccountHistory(accountID, userID string) {
backend/internal/service/account_sync.go:31:func (s *AccountSyncService) CheckMarginCall(
backend/internal/service/analytics_cache.go:102:func (c *AnalyticsCache) SetRolling(ctx context.Context, accountID string, resp *antv1.GetRollingMetricsResponse) error {
backend/internal/service/analytics_cache.go:111:func (c *AnalyticsCache) GetMonthlyDetail(ctx context.Context, accountID string, year, month int32) (*antv1.GetMonthlyDetailResponse, error) {
backend/internal/service/analytics_cache.go:127:func (c *AnalyticsCache) SetMonthlyDetail(ctx context.Context, accountID string, year, month int32, resp *antv1.GetMonthlyDetailResponse) error {
backend/internal/service/analytics_cache.go:138:func (c *AnalyticsCache) Invalidate(ctx context.Context, accountID string) {
backend/internal/service/analytics_cache.go:39:func (c *AnalyticsCache) Get(ctx context.Context, accountID string) (*antv1.AccountAnalyticsResponse, error) {
backend/internal/service/analytics_cache.go:54:func (c *AnalyticsCache) Set(ctx context.Context, accountID string, resp *antv1.AccountAnalyticsResponse) error {
backend/internal/service/analytics_cache.go:63:func (c *AnalyticsCache) GetAttribution(ctx context.Context, accountID string) (*antv1.GetAttributionAnalysisResponse, error) {
backend/internal/service/analytics_cache.go:78:func (c *AnalyticsCache) SetAttribution(ctx context.Context, accountID string, resp *antv1.GetAttributionAnalysisResponse) error {
backend/internal/service/analytics_cache.go:87:func (c *AnalyticsCache) GetRolling(ctx context.Context, accountID string) (*antv1.GetRollingMetricsResponse, error) {
backend/internal/service/log_service.go:21:func (s *LogService) LogConnection(ctx context.Context, log *model.AccountConnectionLog) error {
backend/internal/service/log_service.go:25:func (s *LogService) GetConnectionLogs(ctx context.Context, userID uuid.UUID, params *model.LogQueryParams) ([]*model.AccountConnectionLog, int, error) {
backend/internal/service/log_service.go:29:func (s *LogService) LogExecution(ctx context.Context, log *model.StrategyExecutionLog) error {
backend/internal/service/log_service.go:33:func (s *LogService) UpdateExecution(ctx context.Context, log *model.StrategyExecutionLog) error {
backend/internal/service/log_service.go:37:func (s *LogService) GetExecutionLogs(ctx context.Context, userID uuid.UUID, params *model.LogQueryParams) ([]*model.StrategyExecutionLog, int, error) {
backend/internal/service/log_service.go:41:func (s *LogService) LogOrder(ctx context.Context, order *model.OrderHistory) error {
backend/internal/service/log_service.go:46:func (s *LogService) UpdateOrderHistoryClose(ctx context.Context, userID, accountID, scheduleID uuid.UUID, ticket int64, closePrice, profit, swap, commission float64, closeTime time.Time) (int64, error) {
backend/internal/service/log_service.go:53:func (s *LogService) GetOrderHistory(ctx context.Context, userID uuid.UUID, params *model.LogQueryParams) ([]*model.OrderHistory, int, error) {
backend/internal/service/log_service.go:57:func (s *LogService) LogOperation(ctx context.Context, log *model.SystemOperationLog) error {
backend/internal/service/log_service.go:61:func (s *LogService) GetOperationLogs(ctx context.Context, userID uuid.UUID, params *model.LogQueryParams) ([]*model.SystemOperationLog, int, error) {
backend/internal/service/log_service.go:65:func (s *LogService) GetScheduleRunLogs(ctx context.Context, userID uuid.UUID, scheduleID uuid.UUID, page, pageSize int) ([]*repository.ScheduleRunLogRow, int, error) {
backend/internal/service/log_service.go:69:func (s *LogService) GetAllLogs(ctx context.Context, userID uuid.UUID, params *model.LogQueryParams) (map[string]interface{}, error) {
backend/internal/service/platform_service.go:109:func (s *PlatformService) IsAdmin(ctx context.Context, userID uuid.UUID) (bool, error) {
backend/internal/service/platform_service.go:118:func (s *PlatformService) UserOwnsAccount(ctx context.Context, userID, accountID string) (bool, error) {
backend/internal/service/platform_service.go:123:func (s *PlatformService) GetAccount(ctx context.Context, userID uuid.UUID, accountID string) (*AccountDTO, error) {
backend/internal/service/platform_service.go:128:func (s *PlatformService) GetAccountBroker(ctx context.Context, accountID string) (string, error) {
backend/internal/service/platform_service.go:139:func (s *PlatformService) ResolveSymbol(ctx context.Context, accountID, rawSymbol string) string {
backend/internal/service/platform_service.go:164:func (s *PlatformService) GetUserAccountIDs(ctx context.Context, userID string) ([]string, error) {
backend/internal/service/platform_service.go:169:func (s *PlatformService) GetUserAccountSnapshots(ctx context.Context, userID string) ([]AccountSnapshot, error) {
backend/internal/service/platform_service.go:174:func (s *PlatformService) GetUserAccountsSummary(ctx context.Context, userID string) (*UserAccountsSummary, error) {
backend/internal/service/platform_service.go:29:func (s *PlatformService) SetLogger(log *zap.Logger) { s.log = log }
backend/internal/service/platform_service.go:42:func (s *PlatformService) ListStrategies(ctx context.Context, userID string) ([]Strategy, error) {
backend/internal/service/platform_service.go:79:func (s *PlatformService) ListSubscriptions(ctx context.Context, userID string) ([]UserSubscription, error) {
backend/internal/service/registration_service.go:109:func (e *errEmailAlreadyRegistered) Error() string { return "email already registered" }
backend/internal/service/registration_service.go:49:func (s *RegistrationService) RegisterUser(ctx context.Context, email, password, nickname string) (*model.User, string, error) {
backend/internal/service/schedule_svc.go:115:func (s *StrategySvc) UpdateSchedule(ctx context.Context, r *ScheduleRow) error {
backend/internal/service/schedule_svc.go:136:func (s *StrategySvc) DeleteSchedule(ctx context.Context, id, userID uuid.UUID) error {
backend/internal/service/schedule_svc.go:148:func (s *StrategySvc) SetScheduleActive(ctx context.Context, id, userID uuid.UUID, active bool) error {
backend/internal/service/schedule_svc.go:42:func (s *StrategySvc) ListSchedules(ctx context.Context, userID uuid.UUID) ([]ScheduleRow, error) {
backend/internal/service/schedule_svc.go:55:func (s *StrategySvc) GetSchedule(ctx context.Context, id, userID uuid.UUID) (*ScheduleRow, error) {
backend/internal/service/schedule_svc.go:75:func (s *StrategySvc) CreateSchedule(ctx context.Context, r *ScheduleRow) error {
backend/internal/service/signal_svc.go:30:func (s *StrategySvc) ListSignals(ctx context.Context, userID, accountID uuid.UUID, status string) ([]SignalRow, error) {
backend/internal/service/signal_svc.go:50:func (s *StrategySvc) GetSignal(ctx context.Context, id, userID uuid.UUID) (*SignalRow, error) {
backend/internal/service/signal_svc.go:66:func (s *StrategySvc) ExecuteSignal(ctx context.Context, signalID, userID uuid.UUID) (*SignalRow, error) {
backend/internal/service/signal_svc.go:81:func (s *StrategySvc) ConfirmSignal(ctx context.Context, signalID, userID uuid.UUID) error {
backend/internal/service/signal_svc.go:95:func (s *StrategySvc) CancelSignal(ctx context.Context, signalID, userID uuid.UUID) error {
backend/internal/service/systemai/chat.go:117:func (s *Service) ChatCompletion(
backend/internal/service/systemai/chat.go:130:func (s *Service) ChatCompletionWithUsage(
backend/internal/service/systemai/chat_failover.go:27:func (s *Service) SetCircuitBreakerDB(db cbExecutor) {
backend/internal/service/systemai/chat_failover.go:83:func (e *failoverErr) Error() string { return e.msg }
backend/internal/service/systemai/chat_stream.go:15:func (s *Service) ChatCompletionStream(
backend/internal/service/systemai/service.go:106:func (s *Service) SetTokenRecorder(fn TokenRecorder) {
backend/internal/service/systemai/service.go:112:func (s *Service) SetWalletChecker(fn func(ctx context.Context, userID uuid.UUID) error) {
backend/internal/service/systemai/service.go:118:func (s *Service) SetPostCallBiller(fn PostCallBiller) {
backend/internal/service/systemai/service.go:123:func (s *Service) SetGatewayProviderRepo(repo *repository.SystemAIProviderRepository) {
backend/internal/service/systemai/service.go:140:func (s *Service) EnsureSeed(ctx context.Context, userID uuid.UUID) error {
backend/internal/service/systemai/service.go:170:func (s *Service) List(ctx context.Context, userID uuid.UUID) ([]*repository.SystemAIConfigRow, error) {
backend/internal/service/systemai/service.go:177:func (s *Service) Get(ctx context.Context, userID uuid.UUID, providerID string) (*repository.SystemAIConfigRow, error) {
backend/internal/service/systemai/service.go:181:func (s *Service) UpdateConfig(ctx context.Context, row *repository.SystemAIConfigRow, updatedBy string) error {
backend/internal/service/systemai/service.go:188:func (s *Service) GetAIPrimary(ctx context.Context, userID uuid.UUID) (providerID, model string, err error) {
backend/internal/service/systemai/service.go:195:func (s *Service) SetAIPrimary(ctx context.Context, userID uuid.UUID, providerID, defaultModel string) error {
backend/internal/service/systemai/service.go:205:func (s *Service) UpdateSecret(ctx context.Context, userID uuid.UUID, providerID, secret, updatedBy string) error {
backend/internal/service/systemai/service.go:235:func (s *Service) GetSecret(ctx context.Context, userID uuid.UUID, providerID string) (string, error) {
backend/internal/service/systemai/service.go:256:func (s *Service) DiscoverModels(ctx context.Context, userID uuid.UUID, providerID string) ([]string, error) {
backend/internal/service/systemai/service.go:91:func (s *Service) SetUserRepo(r *repository.UserRepository) {
backend/internal/service/template_svc.go:102:func (s *StrategySvc) SetTemplateStatus(ctx context.Context, id, userID uuid.UUID, status string) error {
backend/internal/service/template_svc.go:34:func (s *StrategySvc) ListTemplates(ctx context.Context, userID uuid.UUID) ([]TemplateRow, error) {
backend/internal/service/template_svc.go:45:func (s *StrategySvc) GetTemplate(ctx context.Context, id, userID uuid.UUID) (*TemplateRow, error) {
backend/internal/service/template_svc.go:60:func (s *StrategySvc) CreateTemplate(ctx context.Context, t *TemplateRow) error {
backend/internal/service/template_svc.go:80:func (s *StrategySvc) UpdateTemplate(ctx context.Context, t *TemplateRow) error {
backend/internal/service/template_svc.go:91:func (s *StrategySvc) DeleteTemplate(ctx context.Context, id, userID uuid.UUID) error {
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
backend/internal/service/user_deletion_service.go:167:func (s *UserDeletionService) RestoreUser(ctx context.Context, actorID, targetID uuid.UUID) error {
backend/internal/service/user_deletion_service.go:34:func (s *UserDeletionService) SoftDeleteUser(ctx context.Context, actorID, targetID uuid.UUID) error {
backend/internal/service/user_deletion_service.go:85:func (s *UserDeletionService) SoftDeleteUsers(ctx context.Context, actorID uuid.UUID, ids []string) (deleted int64, failed int, errors []string) {
backend/internal/service/wallet_service.go:28:func (s *WalletService) GetOrCreateWallet(ctx context.Context, userID uuid.UUID) (*model.Wallet, error) {
backend/internal/service/wallet_service.go:40:func (s *WalletService) CreateWallet(ctx context.Context, userID uuid.UUID) (*model.Wallet, error) {
backend/internal/service/wallet_service.go:46:func (s *WalletService) AdjustBalance(ctx context.Context, userID uuid.UUID, amount, txType, description string, operatorID *uuid.UUID) (*model.Wallet, error) {
backend/internal/service/wallet_service.go:77:func (s *WalletService) ListTransactions(ctx context.Context, userID uuid.UUID, page, pageSize int) ([]model.WalletTransaction, int64, error) {
backend/internal/service/wallet_service.go:86:func (e *walletNotFoundError) Error() string { return "wallet not found" }
backend/internal/service/wallet_service.go:94:func (e *InsufficientBalanceError) Error() string {
```

## Handler 注册（已接进生产路由 = 非 shelf-ware）

> 在此列表 = 真正可被调用；只在某 *_test.go 出现而不在此 = 货架闲置（shelf-ware）。

```
backend/cmd/server/handlers.go:100:	mux.Handle(antv1c.NewWalletServiceHandler(walletServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
backend/cmd/server/handlers.go:105:	mux.Handle(antv1c.NewMtHubServiceHandler(mthubServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
backend/cmd/server/handlers.go:112:	mux.Handle(antv1c.NewAccountServiceHandler(accountServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
backend/cmd/server/handlers.go:115:	mux.Handle(antv1c.NewMarketServiceHandler(mktServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
backend/cmd/server/handlers.go:119:	mux.Handle(antv1c.NewMarketplaceServiceHandler(mktplaceHandler, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
backend/cmd/server/handlers.go:127:	mux.Handle(antv1c.NewExecutionAlgoServiceHandler(algoServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
backend/cmd/server/handlers.go:146:	mux.Handle(antv1c.NewAIServiceHandler(aiServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
backend/cmd/server/handlers.go:148:	mux.Handle(antv1c.NewAgentDefinitionServiceHandler(aiServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
backend/cmd/server/handlers.go:153:	mux.Handle(antv1c.NewAssetAnalysisServiceHandler(assetAnalysisServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
backend/cmd/server/handlers.go:160:	mux.Handle(antv1c.NewShareServiceHandler(shareServer, connectrpc.WithInterceptors(otelInterceptor, authInterceptor)))
backend/cmd/server/handlers.go:185:	mux.Handle(antv1c.NewAIGatewayServiceHandler(gatewayServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
backend/cmd/server/handlers.go:249:	mux.Handle(antv1c.NewStreamServiceHandler(streamServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
backend/cmd/server/handlers.go:257:	mux.Handle(antv1c.NewStrategyServiceHandler(strategyServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
backend/cmd/server/handlers.go:269:	mux.Handle(antv1c.NewStrategyRuntimeServiceHandler(strategyExecServer,
backend/cmd/server/handlers.go:285:	mux.Handle(antv1c.NewPaperTradingServiceHandler(paperHandler,
backend/cmd/server/handlers.go:289:	mux.Handle(antv1c.NewCodeAssistServiceHandler(codeAssistServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
backend/cmd/server/handlers.go:291:	mux.Handle(antv1c.NewSystemAIServiceHandler(systemAIServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
backend/cmd/server/handlers.go:293:	mux.Handle(antv1c.NewAIPrimaryServiceHandler(aiPrimaryServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
backend/cmd/server/handlers.go:295:	mux.Handle(antv1c.NewBacktestTradesServiceHandler(backtestTradesServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
backend/cmd/server/handlers.go:297:	mux.Handle(antv1c.NewGateServiceHandler(gateEvalServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
backend/cmd/server/handlers.go:299:	mux.Handle(antv1c.NewStrategyGenerationServiceHandler(strategyGenServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
backend/cmd/server/handlers.go:307:	mux.Handle(antv1c.NewStrategyPlanServiceHandler(strategyPlanServer, connectrpc.WithInterceptors(otelInterceptor, authInterceptor)))
backend/cmd/server/handlers.go:310:	mux.Handle(antv1c.NewEconomicDataServiceHandler(economicDataServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
backend/cmd/server/handlers.go:313:	mux.Handle(antv1c.NewJobServiceHandler(jobServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
backend/cmd/server/handlers.go:315:	mux.Handle(antv1c.NewLogServiceHandler(logServiceServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
backend/cmd/server/handlers.go:317:	mux.Handle(antv1c.NewNotificationServiceHandler(notifServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
backend/cmd/server/handlers.go:327:	mux.Handle(antv1c.NewAdminTradingServiceHandler(adminTradingServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor, adminInterceptor)))
backend/cmd/server/handlers.go:329:	mux.Handle(antv1c.NewAdminConfigServiceHandler(adminConfigServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor, adminInterceptor)))
backend/cmd/server/handlers.go:331:	mux.Handle(antv1c.NewAdminLogServiceHandler(adminLogServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor, adminInterceptor)))
backend/cmd/server/handlers.go:333:	mux.Handle(antv1c.NewAdminAccountServiceHandler(adminAccountServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor, adminInterceptor)))
backend/cmd/server/handlers.go:336:	mux.Handle(antv1c.NewAdminUserServiceHandler(adminUserServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor, adminInterceptor)))
backend/cmd/server/handlers.go:338:	mux.Handle(antv1c.NewAdminSystemServiceHandler(adminSystemServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor, adminInterceptor)))
backend/cmd/server/handlers.go:340:		mux.Handle(antv1c.NewAdminStrategyServiceHandler(adminStrategyServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor, adminInterceptor)))
backend/cmd/server/handlers.go:378:	mux.Handle(antv1c.NewAutoTradingServiceHandler(autoTradingServer,
backend/cmd/server/handlers.go:397:	mux.Handle(antv1c.NewAdminJurisdictionServiceHandler(adminJurisdictionServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor, adminInterceptor)))
backend/cmd/server/handlers.go:97:	mux.Handle(antv1c.NewAuthServiceHandler(authServer, connectrpc.WithInterceptors(otelInterceptor,rateLimitInterceptor, authInterceptor)))
backend/cmd/server/handlers_sre.go:100:	mux.Handle(antv1c.NewScheduleHealthServiceHandler(scheduleHealthServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
backend/cmd/server/handlers_sre.go:102:	mux.Handle(antv1c.NewIndicatorCatalogServiceHandler(indicatorCatalogServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
backend/cmd/server/handlers_sre.go:72:	mux.Handle(antv1c.NewAdminSREServiceHandler(sreHandler, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
backend/cmd/server/handlers_sre.go:76:	mux.Handle(antv1c.NewAnalyticsServiceHandler(analyticsServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
backend/cmd/server/handlers_sre.go:80:	mux.Handle(antv1c.NewMarketRegimeServiceHandler(marketRegimeServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
backend/cmd/server/handlers_sre.go:84:	mux.Handle(antv1c.NewStrategyExperimentServiceHandler(strategyExperimentServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
backend/cmd/server/handlers_sre.go:98:	mux.Handle(antv1c.NewStrategyAssetServiceHandler(strategyAssetServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
```
