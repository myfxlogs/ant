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
| 改 SL/TP | **OrderModify / ModifyOrder** / amend / update_stops / modify | ✅ mt4/mt5 | ✅ `ModifyOrder` | ✅ `ModifyOrder` | ❌ **未暴露** | ⚠️ **live 仍用 close+place 绕路（注释过时，待修）** | `backend/internal/mthub/service_orders_modify.go:18` |
| 查持仓 | OpenedOrders / positions / FetchOpenedOrders | ✅ mt4/mt5 | ✅ `FetchOpenedOrders` | ✅ `OpenedOrders` | ✅ | ✅ | `backend/internal/mthub/service.go:207` |
| 历史订单 | OrderHistory / FetchOrderHistory / history | ✅ | ✅ | ✅ `OrderHistory` | ✅ | ✅ | `backend/internal/mthub/service.go:218` |
| 品种参数 | SymbolParams / FetchSymbolParams | ✅ | ✅ | ✅ `SymbolParams` | ✅ | ✅ | `backend/internal/mthub/service.go:227` |
| 账户状态推送 | AccountStatus / SubscribeAccountStatus | — | — | ✅ `SubscribeAccountStatus` | ✅ (Stream) | ⚠️ live_runner 未消费（equity 仍硬编码 10000，见 T3.2b） | `backend/internal/mthub/service.go:153` |

> **教训实例**：`live_runner.go` 曾写"MtHubService doesn't expose a direct ModifyOrder, use close+place workaround"——**该注释已过时**：`MtHubService.ModifyOrder` 早已实现（`service_orders_modify.go:18`）。这正是本清单要消灭的失误。真正缺的只是 **public ConnectRPC 暴露** + **接进 live 路径**。

## 高风险域：策略 SDK / Broker（EA 替代）

| 能力 | 别名 | 状态 | 权威位置 |
|---|---|---|---|
| Broker 抽象接口 | order_send/position_modify/position_close/order_delete/positions/orders/account/symbol_info/server_time | ✅ 已冻结 | `strategy-service/app/sdk/broker.py` |
| 回测 Broker | SimBroker | ✅ 实现，⚠️ 未集成 gate（见 D6-A） | `strategy-service/app/engine/sim_broker.py` |
| 实盘 Broker | LiveBroker / OrderIntent / to_signal_dict | ✅ 实现，⚠️ 未端到端验证 | `strategy-service/app/engine/live_broker.py` |
| 策略生命周期 | StrategyRuntime / on_init/on_tick/on_bar/on_timer/on_trade/on_deinit | ✅ | `strategy-service/app/sdk/runtime.py` |
| 风控门 | Gate / Evaluate / risk rules / kill-switch | ✅ 实现，🔴 **shelf-ware：未接进 live_runner**（见 T3.2） | `backend/internal/risk/gate.go:109` |
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

_最后生成：2026-06-24 02:37 UTC。运行 `bash scripts/gen_capability_map.sh` 刷新。_

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
proto/ant/v1/admin_system.proto:10:  rpc ResolveAlert(ResolveAlertRequest) returns (ResolveAlertResponse);
proto/ant/v1/admin_system.proto:11:  rpc ClearCache(ClearCacheRequest) returns (ClearCacheResponse);
proto/ant/v1/admin_system.proto:12:  rpc InvalidateCache(InvalidateCacheRequest) returns (InvalidateCacheResponse);
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
proto/ant/v1/objective_score.proto:8:  rpc CalculateObjectiveScore(ObjectiveScoreRequest) returns (ObjectiveScoreResponse);
proto/ant/v1/paper_trading.proto:10:  rpc CreatePaperAccount(CreatePaperAccountRequest) returns (PaperAccount);
proto/ant/v1/paper_trading.proto:11:  rpc ListPaperAccounts(ListPaperAccountsRequest) returns (ListPaperAccountsResponse);
proto/ant/v1/paper_trading.proto:12:  rpc StartPaperStrategy(StartPaperStrategyRequest) returns (StartPaperStrategyResponse);
proto/ant/v1/paper_trading.proto:13:  rpc StopPaperStrategy(StopPaperStrategyRequest) returns (StopPaperStrategyResponse);
proto/ant/v1/paper_trading.proto:14:  rpc WatchPaperAccount(WatchPaperAccountRequest) returns (stream PaperAccountUpdate);
proto/ant/v1/python_strategy.proto:15:  rpc Execute(ExecuteStrategyRequest) returns (ExecuteStrategyResponse);
proto/ant/v1/python_strategy.proto:16:  rpc Validate(ValidateStrategyRequest) returns (ValidateStrategyResponse);
proto/ant/v1/python_strategy.proto:17:  rpc Backtest(BacktestStrategyRequest) returns (BacktestStrategyResponse);
proto/ant/v1/python_strategy.proto:18:  rpc StartBacktestRun(StartBacktestRunRequest) returns (StartBacktestRunResponse);
proto/ant/v1/python_strategy.proto:19:  rpc GetBacktestRun(GetBacktestRunRequest) returns (GetBacktestRunResponse);
proto/ant/v1/python_strategy.proto:20:  rpc ListBacktestRuns(ListBacktestRunsRequest) returns (ListBacktestRunsResponse);
proto/ant/v1/python_strategy.proto:21:  rpc WatchBacktestRun(WatchBacktestRunRequest) returns (stream BacktestRunUpdate);
proto/ant/v1/python_strategy.proto:22:  rpc CancelBacktestRun(CancelBacktestRunRequest) returns (CancelBacktestRunResponse);
proto/ant/v1/python_strategy.proto:23:  rpc DeleteBacktestRun(DeleteBacktestRunRequest) returns (DeleteBacktestRunResponse);
proto/ant/v1/python_strategy.proto:24:  rpc DeleteBacktestRuns(DeleteBacktestRunsRequest) returns (DeleteBacktestRunsResponse);
proto/ant/v1/python_strategy.proto:25:  rpc GetTemplates(google.protobuf.Empty) returns (GetPythonTemplatesResponse);
proto/ant/v1/python_strategy.proto:29:  rpc ExecuteLive(ExecuteLiveRequest) returns (ExecuteLiveResponse);
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
backend/internal/connect/strategy/backtest_trades_handler.go:53:func (s *BacktestTradesServer) ListBacktestRunTrades(ctx context.Context, req *connect.Request[antv1.ListBacktestRunTradesRequest]) (*connect.Response[antv1.ListBacktestRunTradesResponse], error) {
backend/internal/connect/strategy/backtest_worker.go:360:func (s *PythonStrategyServer) StartBacktestWorker(ctx context.Context) {
backend/internal/connect/strategy/data_source.go:54:func (s *LiveSource) Name() string { return "live" }
backend/internal/connect/strategy/data_source.go:56:func (s *LiveSource) Fetch(ctx context.Context, symbol, timeframe string, from, to *time.Time) ([]*antv1.ExecuteKlineBar, error) {
backend/internal/connect/strategy/data_source.go:76:func (s *LiveSource) Subscribe(accountID string) (<-chan *mthub.BarUpdate, func()) {
backend/internal/connect/strategy/live_runner.go:57:func (s *PythonStrategyServer) RunLiveStrategy(ctx context.Context, cfg LiveStrategyConfig) error {
backend/internal/connect/strategy/objective_score_handler.go:30:func (s *ObjectiveScoreServer) CalculateObjectiveScore(
backend/internal/connect/strategy/python_strategy_backtest_crud.go:126:func (s *PythonStrategyServer) WatchBacktestRun(ctx context.Context, req *connect.Request[antv1.WatchBacktestRunRequest], stream *connect.ServerStream[antv1.BacktestRunUpdate]) error {
backend/internal/connect/strategy/python_strategy_backtest_crud.go:198:func (s *PythonStrategyServer) CancelBacktestRun(ctx context.Context, req *connect.Request[antv1.CancelBacktestRunRequest]) (*connect.Response[antv1.CancelBacktestRunResponse], error) {
backend/internal/connect/strategy/python_strategy_backtest_crud.go:214:func (s *PythonStrategyServer) DeleteBacktestRun(ctx context.Context, req *connect.Request[antv1.DeleteBacktestRunRequest]) (*connect.Response[antv1.DeleteBacktestRunResponse], error) {
backend/internal/connect/strategy/python_strategy_backtest_crud.go:229:func (s *PythonStrategyServer) DeleteBacktestRuns(ctx context.Context, req *connect.Request[antv1.DeleteBacktestRunsRequest]) (*connect.Response[antv1.DeleteBacktestRunsResponse], error) {
backend/internal/connect/strategy/python_strategy_backtest_crud.go:26:func (s *PythonStrategyServer) StartBacktestRun(ctx context.Context, req *connect.Request[antv1.StartBacktestRunRequest]) (*connect.Response[antv1.StartBacktestRunResponse], error) {
backend/internal/connect/strategy/python_strategy_backtest_crud.go:74:func (s *PythonStrategyServer) GetBacktestRun(ctx context.Context, req *connect.Request[antv1.GetBacktestRunRequest]) (*connect.Response[antv1.GetBacktestRunResponse], error) {
backend/internal/connect/strategy/python_strategy_backtest_crud.go:93:func (s *PythonStrategyServer) ListBacktestRuns(ctx context.Context, req *connect.Request[antv1.ListBacktestRunsRequest]) (*connect.Response[antv1.ListBacktestRunsResponse], error) {
backend/internal/connect/strategy/python_strategy_handler.go:120:func (s *PythonStrategyServer) Validate(ctx context.Context, req *connect.Request[antv1.ValidateStrategyRequest]) (*connect.Response[antv1.ValidateStrategyResponse], error) {
backend/internal/connect/strategy/python_strategy_handler.go:142:func (s *PythonStrategyServer) Backtest(ctx context.Context, req *connect.Request[antv1.BacktestStrategyRequest]) (*connect.Response[antv1.BacktestStrategyResponse], error) {
backend/internal/connect/strategy/python_strategy_handler.go:195:func (s *PythonStrategyServer) GetTemplates(_ context.Context, _ *connect.Request[emptypb.Empty]) (*connect.Response[antv1.GetPythonTemplatesResponse], error) {
backend/internal/connect/strategy/python_strategy_handler.go:201:func (s *PythonStrategyServer) ExecuteLive(ctx context.Context, req *connect.Request[antv1.ExecuteLiveRequest]) (*connect.Response[antv1.ExecuteLiveResponse], error) {
backend/internal/connect/strategy/python_strategy_handler.go:216:func (s *PythonStrategyServer) SetPgListen(l *pglisten.Listener) { s.pgListen = l }
backend/internal/connect/strategy/python_strategy_handler.go:53:func (s *PythonStrategyServer) SetMarketDataRepo(r repository.MarketDataStore) {
backend/internal/connect/strategy/python_strategy_handler.go:64:func (s *PythonStrategyServer) SetBarSource(bs BarSource)                 { s.barSource = bs }
backend/internal/connect/strategy/python_strategy_handler.go:65:func (s *PythonStrategyServer) SetMtHub(h *mthub.MtHubService)            { s.mtHub = h }
backend/internal/connect/strategy/python_strategy_handler.go:66:func (s *PythonStrategyServer) SetPaperEngine(pe PaperOrderExecutor)      { s.paperEngine = pe }
backend/internal/connect/strategy/python_strategy_handler.go:70:func (s *PythonStrategyServer) SetGate(g *risk.Gate) { s.gate = g }
backend/internal/connect/strategy/python_strategy_handler.go:73:func (s *PythonStrategyServer) SetAccountProvider(p AccountStateProvider) { s.accountProvider = p }
backend/internal/connect/strategy/python_strategy_handler.go:81:func (s *PythonStrategyServer) SetConnectClient(c antv1c.PythonStrategyServiceClient) { s.connectClient = c }
backend/internal/connect/strategy/python_strategy_handler.go:82:func (s *PythonStrategyServer) SetBacktestClient(c antv1c.BacktestServiceClient)     { s.backtestClient = c }
backend/internal/connect/strategy/python_strategy_handler.go:83:func (s *PythonStrategyServer) SetNotificationSender(ns *notification.Sender)         { s.notifSender = ns }
backend/internal/connect/strategy/python_strategy_handler.go:84:func (s *PythonStrategyServer) SetOnBacktestComplete(fn func(context.Context, *repository.BacktestRun)) { s.onBacktestComplete = fn }
backend/internal/connect/strategy/python_strategy_handler.go:95:func (s *PythonStrategyServer) Execute(ctx context.Context, req *connect.Request[antv1.ExecuteStrategyRequest]) (*connect.Response[antv1.ExecuteStrategyResponse], error) {
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
backend/internal/connect/strategy/strategy_handler.go:41:func (s *StrategyServer) SetEngine(e *ScheduleEngine) { s.engine = e }
backend/internal/connect/strategy/strategy_handler.go:44:func (s *StrategyServer) SetBacktestClient(c antv1c.BacktestServiceClient) { s.backtestClient = c }
backend/internal/connect/strategy/strategy_handler.go:47:func (s *StrategyServer) SetMarketDataRepo(r repository.MarketDataStore) { s.marketDataRepo = r }
backend/internal/connect/strategy/strategy_handler.go:63:func (s *StrategyServer) SetPgListen(l *pglisten.Listener) { s.pgListen = l }
backend/internal/connect/strategy/strategy_schedules.go:133:func (s *StrategyServer) DeleteSchedule(ctx context.Context, req *connect.Request[antv1.DeleteScheduleRequest]) (*connect.Response[emptypb.Empty], error) {
backend/internal/connect/strategy/strategy_schedules.go:147:func (s *StrategyServer) ToggleSchedule(ctx context.Context, req *connect.Request[antv1.ToggleScheduleRequest]) (*connect.Response[antv1.StrategySchedule], error) {
backend/internal/connect/strategy/strategy_schedules.go:180:func (s *StrategyServer) WatchSchedules(ctx context.Context, req *connect.Request[antv1.WatchSchedulesRequest], stream *connect.ServerStream[antv1.WatchSchedulesEvent]) error {
backend/internal/connect/strategy/strategy_schedules.go:20:func (s *StrategyServer) ListSchedules(ctx context.Context, req *connect.Request[antv1.ListSchedulesRequest]) (*connect.Response[antv1.ListSchedulesResponse], error) {
backend/internal/connect/strategy/strategy_schedules.go:32:func (s *StrategyServer) GetSchedule(ctx context.Context, req *connect.Request[antv1.GetScheduleRequest]) (*connect.Response[antv1.StrategySchedule], error) {
backend/internal/connect/strategy/strategy_schedules.go:44:func (s *StrategyServer) CreateSchedule(ctx context.Context, req *connect.Request[antv1.CreateScheduleRequest]) (*connect.Response[antv1.StrategySchedule], error) {
backend/internal/connect/strategy/strategy_schedules.go:92:func (s *StrategyServer) UpdateSchedule(ctx context.Context, req *connect.Request[antv1.UpdateScheduleRequest]) (*connect.Response[antv1.StrategySchedule], error) {
backend/internal/connect/strategy/strategy_signals.go:147:func (s *StrategyServer) ListSignals(
backend/internal/connect/strategy/strategy_signals.go:164:func (s *StrategyServer) ExecuteSignal(
backend/internal/connect/strategy/strategy_signals.go:185:func (s *StrategyServer) ConfirmSignal(
backend/internal/connect/strategy/strategy_signals.go:198:func (s *StrategyServer) CancelSignal(
backend/internal/connect/strategy/strategy_signals.go:19:func (s *StrategyServer) RunBacktest(
backend/internal/connect/strategy/strategy_templates.go:105:func (s *StrategyServer) DeleteTemplate(ctx context.Context, req *connect.Request[antv1.DeleteTemplateRequest]) (*connect.Response[emptypb.Empty], error) {
backend/internal/connect/strategy/strategy_templates.go:116:func (s *StrategyServer) CreateTemplateDraft(ctx context.Context, req *connect.Request[antv1.CreateTemplateDraftRequest]) (*connect.Response[antv1.StrategyTemplate], error) {
backend/internal/connect/strategy/strategy_templates.go:131:func (s *StrategyServer) UpdateTemplateDraft(ctx context.Context, req *connect.Request[antv1.UpdateTemplateDraftRequest]) (*connect.Response[antv1.StrategyTemplate], error) {
backend/internal/connect/strategy/strategy_templates.go:162:func (s *StrategyServer) PublishTemplateDraft(ctx context.Context, req *connect.Request[antv1.PublishTemplateDraftRequest]) (*connect.Response[antv1.StrategyTemplate], error) {
backend/internal/connect/strategy/strategy_templates.go:17:func (s *StrategyServer) ListTemplates(ctx context.Context, req *connect.Request[antv1.ListTemplatesRequest]) (*connect.Response[antv1.ListTemplatesResponse], error) {
backend/internal/connect/strategy/strategy_templates.go:181:func (s *StrategyServer) CancelTemplateDraft(ctx context.Context, req *connect.Request[antv1.CancelTemplateDraftRequest]) (*connect.Response[emptypb.Empty], error) {
backend/internal/connect/strategy/strategy_templates.go:29:func (s *StrategyServer) GetTemplate(ctx context.Context, req *connect.Request[antv1.GetTemplateRequest]) (*connect.Response[antv1.StrategyTemplate], error) {
backend/internal/connect/strategy/strategy_templates.go:52:func (s *StrategyServer) CreateTemplate(ctx context.Context, req *connect.Request[antv1.CreateTemplateRequest]) (*connect.Response[antv1.StrategyTemplate], error) {
backend/internal/connect/strategy/strategy_templates.go:71:func (s *StrategyServer) UpdateTemplate(ctx context.Context, req *connect.Request[antv1.UpdateTemplateRequest]) (*connect.Response[antv1.StrategyTemplate], error) {
backend/internal/mthub/broker_types.go:126:func (b *BarBroker) Publish(ev *BarUpdate) {
backend/internal/mthub/broker_types.go:153:func (b *BarBroker) DroppedBars(accountID string) int64 {
backend/internal/mthub/broker_types.go:158:func (b *BarBroker) Subscribe(accountID string) (<-chan *BarUpdate, func()) {
backend/internal/mthub/broker_types.go:201:func (b *AccountStatusBroker) Publish(ev *AccountStatusEvent) {
backend/internal/mthub/broker_types.go:215:func (b *AccountStatusBroker) Subscribe(accountID string) (<-chan *AccountStatusEvent, func()) {
backend/internal/mthub/broker_types.go:63:func (b *PositionSnapshotBroker) Publish(ev *PositionSnapshot) {
backend/internal/mthub/broker_types.go:77:func (b *PositionSnapshotBroker) Subscribe(accountID string) (<-chan *PositionSnapshot, func()) {
backend/internal/mthub/derived_state.go:110:func (dc *DerivedComputer) Start() {
backend/internal/mthub/derived_state.go:115:func (dc *DerivedComputer) Stop() {
backend/internal/mthub/derived_state.go:120:func (dc *DerivedComputer) State() *DerivedState {
backend/internal/mthub/derived_state.go:61:func (d *DerivedState) Update(accounts map[string]*AccountDerivedState, totalExposure, totalMargin, grossPnL, netPnL, var95 float64) {
backend/internal/mthub/derived_state.go:75:func (d *DerivedState) Get() (accounts map[string]*AccountDerivedState, totalExposure, totalMargin, grossPnL, netPnL, var95 float64, lastUpdated time.Time) {
backend/internal/mthub/derived_state.go:82:func (d *DerivedState) GetAccount(accountID string) *AccountDerivedState {
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
backend/internal/mthub/service.go:100:func (s *MtHubService) SetOmsWriter(w *OmsWriter) { s.omsWriter = w }
backend/internal/mthub/service.go:103:func (s *MtHubService) SetKillSwitch(ks KillSwitchGate) { s.killSwitch = ks }
backend/internal/mthub/service.go:108:func (s *MtHubService) SetBrokerRegistry(r BrokerRegistry) { s.brokerRegistry = r }
backend/internal/mthub/service.go:111:func (s *MtHubService) BrokerRegistry() BrokerRegistry { return s.brokerRegistry }
backend/internal/mthub/service.go:117:func (s *MtHubService) SetAccountOwnerVerifier(v AccountOwnerVerifier) { s.accountOwnerVerifier = v }
backend/internal/mthub/service.go:120:func (s *MtHubService) SetLogger(l *zap.Logger) { s.logger = l }
backend/internal/mthub/service.go:123:func (s *MtHubService) SetBarBroker(b *BarBroker) { s.barBroker = b }
backend/internal/mthub/service.go:126:func (s *MtHubService) SetStatusBroker(b *AccountStatusBroker) { s.statusBroker = b }
backend/internal/mthub/service.go:129:func (s *MtHubService) PublishBar(ev *BarUpdate) {
backend/internal/mthub/service.go:136:func (s *MtHubService) SubscribeBarUpdates(accountID string) (<-chan *BarUpdate, func()) {
backend/internal/mthub/service.go:146:func (s *MtHubService) PublishAccountStatus(ev *AccountStatusEvent) {
backend/internal/mthub/service.go:153:func (s *MtHubService) SubscribeAccountStatus(accountID string) (<-chan *AccountStatusEvent, func()) {
backend/internal/mthub/service.go:166:func (s *MtHubService) Platform(accountID string) string {
backend/internal/mthub/service.go:176:func (s *MtHubService) SessionState(ctx context.Context, accountID string) string {
backend/internal/mthub/service.go:207:func (s *MtHubService) OpenedOrders(ctx context.Context, accountID string) ([]*OrderRecord, error) {
backend/internal/mthub/service.go:218:func (s *MtHubService) OrderHistory(ctx context.Context, accountID string, from, to time.Time) ([]*OrderRecord, error) {
backend/internal/mthub/service.go:227:func (s *MtHubService) SymbolParams(ctx context.Context, accountID string, canonicals []string) ([]*SymbolParam, error) {
backend/internal/mthub/service.go:236:func (s *MtHubService) PriceHistory(ctx context.Context, accountID, symbol, period string, from, to int64, count int) ([]*Bar, error) {
backend/internal/mthub/service.go:245:func (s *MtHubService) SymbolList(ctx context.Context, accountID string) ([]string, error) {
backend/internal/mthub/service.go:256:func (s *MtHubService) SubscribeSymbols(ctx context.Context, accountID string, symbols []string) error {
backend/internal/mthub/service.go:265:func (s *MtHubService) SubscribeUserOrderEvents(ctx context.Context, userID string) (<-chan *OrderEvent, func()) {
backend/internal/mthub/service.go:270:func (s *MtHubService) PublishAccountProfit(ev *AccountProfitEvent) {
backend/internal/mthub/service.go:275:func (s *MtHubService) SubscribeAccountProfit(ctx context.Context, accountID string) (<-chan *AccountProfitEvent, func()) {
backend/internal/mthub/service.go:280:func (s *MtHubService) PublishPositionSnapshot(ev *PositionSnapshot) {
backend/internal/mthub/service.go:285:func (s *MtHubService) SubscribePositionSnapshots(ctx context.Context, accountID string) (<-chan *PositionSnapshot, func()) {
backend/internal/mthub/service.go:74:func (s *MtHubService) SetUserLimiter(l *usermgr.UserLimiter) { s.userLimiter = l }
backend/internal/mthub/service.go:77:func (s *MtHubService) SetCostEstimator(e costsvc.CostEstimator) { s.costEstimator = e }
backend/internal/mthub/service.go:85:func (s *MtHubService) SetRiskPipeline(p RiskPipeline) { s.riskPipeline = p }
backend/internal/mthub/service.go:97:func (s *MtHubService) SetAccountStateProvider(p AccountStateProvider) { s.accountStateProvider = p }
backend/internal/mthub/service_orders.go:18:func (s *MtHubService) PlaceOrder(ctx context.Context, req *OrderRequest) (*OrderRecord, error) {
backend/internal/mthub/service_orders_close.go:15:func (s *MtHubService) CloseOrder(ctx context.Context, accountID string, ticket int64, lots decimal.Decimal) error {
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
backend/internal/paper/engine.go:121:func (e *PaperEngine) Subscribe(accountID string) (<-chan *repository.PaperAccount, func()) {
backend/internal/paper/engine.go:57:func (e *PaperEngine) PlacePaperOrder(ctx context.Context, accountID, symbol, side string,
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
backend/internal/risk/gate.go:226:func (e *AuditEntry) String() string {
backend/internal/risk/gate.go:94:func (g *Gate) SetKillSwitch(fn func() bool) {
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
backend/internal/service/template_svc.go:101:func (s *StrategySvc) SetTemplateStatus(ctx context.Context, id, userID uuid.UUID, status string) error {
backend/internal/service/template_svc.go:33:func (s *StrategySvc) ListTemplates(ctx context.Context, userID uuid.UUID) ([]TemplateRow, error) {
backend/internal/service/template_svc.go:44:func (s *StrategySvc) GetTemplate(ctx context.Context, id, userID uuid.UUID) (*TemplateRow, error) {
backend/internal/service/template_svc.go:59:func (s *StrategySvc) CreateTemplate(ctx context.Context, t *TemplateRow) error {
backend/internal/service/template_svc.go:79:func (s *StrategySvc) UpdateTemplate(ctx context.Context, t *TemplateRow) error {
backend/internal/service/template_svc.go:90:func (s *StrategySvc) DeleteTemplate(ctx context.Context, id, userID uuid.UUID) error {
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

## Python SDK / 引擎（已实现的类与函数）

```
strategy-service/app/engine/code_quality.py:182:def _check_missing_params(code: str) -> List[CodeHint]:
strategy-service/app/engine/code_quality.py:207:def _check_unread_params(code: str) -> List[CodeHint]:
strategy-service/app/engine/code_quality.py:250:def _check_ndarray_pandas_misuse(code: str) -> List[CodeHint]:
strategy-service/app/engine/code_quality.py:25:class CodeHint:
strategy-service/app/engine/code_quality.py:315:def _has_buy_sell_signal(code: str) -> bool:
strategy-service/app/engine/code_quality.py:325:def _check_no_stop_take_profit(code: str) -> List[CodeHint]:
strategy-service/app/engine/code_quality.py:357:def _check_no_entry_pct(code: str) -> List[CodeHint]:
strategy-service/app/engine/code_quality.py:379:def _line_of(pattern: str, code: str, start: int = 0) -> int:
strategy-service/app/engine/code_quality.py:37:def _strip_comments(code: str) -> str:
strategy-service/app/engine/code_quality.py:391:def analyze_code_quality(source: str) -> List[CodeHint]:
strategy-service/app/engine/code_quality.py:65:def _declared_param_names(code: str) -> List[str]:
strategy-service/app/engine/code_quality.py:77:def _param_get_calls(code: str) -> Set[str]:
strategy-service/app/engine/code_quality.py:92:def _check_future_data_leak(code: str) -> List[CodeHint]:
strategy-service/app/engine/compilation.py:21:def _compile_source(source: str) -> Any:
strategy-service/app/engine/compilation.py:29:def code_sha256(source: str) -> str:
strategy-service/app/engine/compilation.py:36:def build_sandbox_globals() -> dict:
strategy-service/app/engine/cost.py:105:    def apply_rollover_swaps(
strategy-service/app/engine/cost.py:22:def _ms_to_utc(ms: int) -> datetime:
strategy-service/app/engine/cost.py:26:def _utc_to_ms(dt: datetime) -> int:
strategy-service/app/engine/cost.py:32:class CostModel:
strategy-service/app/engine/cost.py:35:    def __init__(self, profile: CostProfile) -> None:
strategy-service/app/engine/cost.py:51:    def commission(self, volume: float) -> float:
strategy-service/app/engine/cost.py:57:    def apply_slippage(self, price: float, is_buy_side: bool) -> float:
strategy-service/app/engine/cost.py:76:    def _to_local(self, dt: datetime) -> datetime:
strategy-service/app/engine/cost.py:83:    def _next_rollover_utc(self, dt: datetime) -> datetime:
strategy-service/app/engine/fill.py:140:    def process_market_order(
strategy-service/app/engine/fill.py:180:    def _is_expired(order: Order, ts: int) -> bool:
strategy-service/app/engine/fill.py:188:    def _check_trigger(self, order: Order, tick: Tick) -> Tuple[bool, float]:
strategy-service/app/engine/fill.py:225:    def _fill_volume(self, requested: float) -> float:
strategy-service/app/engine/fill.py:45:def _is_buy_side(order_type: OrderType) -> bool:
strategy-service/app/engine/fill.py:49:class FillModel:
strategy-service/app/engine/fill.py:52:    def __init__(self, cost: CostModel, max_fill_volume: float = 0.0) -> None:
strategy-service/app/engine/fill.py:61:    def pending(self) -> List[Order]:
strategy-service/app/engine/fill.py:64:    def enqueue(self, order: Order, replace_same_type: bool = False) -> None:
strategy-service/app/engine/fill.py:76:    def cancel_all(self) -> int:
strategy-service/app/engine/fill.py:85:    def process_on_tick(self, tick: Tick) -> List[Tuple[Fill, Order]]:
strategy-service/app/engine/indicators.py:105:def iMACD(
strategy-service/app/engine/indicators.py:129:def iStochastic(
strategy-service/app/engine/indicators.py:154:def iATR(high, low, close, period: int = 14, shift: int = 0) -> float:
strategy-service/app/engine/indicators.py:173:def iCCI(high, low, close, period: int = 14, shift: int = 0) -> float:
strategy-service/app/engine/indicators.py:191:def iMomentum(prices, period: int = 14, shift: int = 0) -> float:
strategy-service/app/engine/indicators.py:200:def iWPR(high, low, close, period: int = 14, shift: int = 0) -> float:
strategy-service/app/engine/indicators.py:221:def OrdersTotal(context: dict) -> int:
strategy-service/app/engine/indicators.py:225:def AccountBalance(context: dict) -> float:
strategy-service/app/engine/indicators.py:229:def AccountEquity(context: dict) -> float:
strategy-service/app/engine/indicators.py:39:def _arr(prices) -> np.ndarray:
strategy-service/app/engine/indicators.py:43:def iMA(prices, period: int = 14, shift: int = 0, method: str = "sma") -> float:
strategy-service/app/engine/indicators.py:62:def iRSI(prices, period: int = 14, shift: int = 0) -> float:
strategy-service/app/engine/indicators.py:79:def iBands(
strategy-service/app/engine/indicators.py:97:def _ema_series(arr: np.ndarray, n: int) -> np.ndarray:
strategy-service/app/engine/live_broker.py:101:    def order_send(self, request: OrderRequest) -> OrderResult:
strategy-service/app/engine/live_broker.py:120:    def position_modify(
strategy-service/app/engine/live_broker.py:139:    def position_close(
strategy-service/app/engine/live_broker.py:162:    def order_delete(self, ticket: int) -> OrderResult:
strategy-service/app/engine/live_broker.py:168:    def positions(
strategy-service/app/engine/live_broker.py:179:    def orders(
strategy-service/app/engine/live_broker.py:190:    def account(self) -> AccountInfo:
strategy-service/app/engine/live_broker.py:194:    def symbol_info(self, symbol: str) -> SymbolInfo:
strategy-service/app/engine/live_broker.py:209:    def server_time(self) -> int:
strategy-service/app/engine/live_broker.py:216:    def order_intents(self) -> List[OrderIntent]:
strategy-service/app/engine/live_broker.py:220:    def close_intents(self) -> List[CloseIntent]:
strategy-service/app/engine/live_broker.py:224:    def modify_intents(self) -> List[ModifyIntent]:
strategy-service/app/engine/live_broker.py:228:    def delete_intents(self) -> List[int]:
strategy-service/app/engine/live_broker.py:231:    def export_intents(self) -> List[dict]:
strategy-service/app/engine/live_broker.py:253:class OrderIntent:
strategy-service/app/engine/live_broker.py:257:    def __init__(self, ticket: int, request: OrderRequest):
strategy-service/app/engine/live_broker.py:261:    def to_signal_dict(self) -> dict:
strategy-service/app/engine/live_broker.py:279:class CloseIntent:
strategy-service/app/engine/live_broker.py:283:    def __init__(self, ticket: int, volume: Optional[Decimal] = None):
strategy-service/app/engine/live_broker.py:287:    def to_signal_dict(self) -> dict:
strategy-service/app/engine/live_broker.py:295:class ModifyIntent:
strategy-service/app/engine/live_broker.py:299:    def __init__(self, ticket: int, sl: Optional[Decimal] = None, tp: Optional[Decimal] = None):
strategy-service/app/engine/live_broker.py:304:    def to_signal_dict(self) -> dict:
strategy-service/app/engine/live_broker.py:313:def build_live_broker_from_proto(lctx) -> LiveBroker:
strategy-service/app/engine/live_broker.py:38:class LiveBroker(Broker):
strategy-service/app/engine/live_broker.py:46:    def __init__(
strategy-service/app/engine/live_broker.py:75:    def update_state(
strategy-service/app/engine/live_broker.py:92:    def clear_intents(self) -> None:
strategy-service/app/engine/margin.py:20:class MarginModel:
strategy-service/app/engine/margin.py:29:    def __init__(self, leverage: float, contract_size: float = 100_000.0) -> None:
strategy-service/app/engine/margin.py:33:    def enabled(self) -> bool:
strategy-service/app/engine/margin.py:38:    def required_margin(self, position: Position, price: float) -> float:
strategy-service/app/engine/margin.py:43:    def used_margin(self, portfolio: "Portfolio", tick: Tick) -> float:
strategy-service/app/engine/margin.py:51:    def margin_level(self, portfolio: "Portfolio", tick: Tick) -> float:
strategy-service/app/engine/margin.py:60:    def is_margin_call(self, portfolio: "Portfolio", tick: Tick) -> bool:
strategy-service/app/engine/market.py:112:class MultiSymbolMarket:
strategy-service/app/engine/market.py:123:    def __init__(self, bars_by_symbol: Dict[str, List], primary: str) -> None:
strategy-service/app/engine/market.py:136:    def primary(self) -> str:
strategy-service/app/engine/market.py:139:    def primary_market(self) -> MarketSimulator:
strategy-service/app/engine/market.py:142:    def symbols(self) -> List[str]:
strategy-service/app/engine/market.py:145:    def market(self, symbol: str) -> MarketSimulator:
strategy-service/app/engine/market.py:151:    def __len__(self) -> int:
strategy-service/app/engine/market.py:154:    def bar_closed_at_or_before(self, ts: int) -> int:
strategy-service/app/engine/market.py:158:    def slice_until(self, primary_idx: int) -> Dict[str, dict]:
strategy-service/app/engine/market.py:15:class TickSimulator:
strategy-service/app/engine/market.py:24:    def __init__(self, bars: List[Bar], ticks: Optional[List[Tick]] = None) -> None:
strategy-service/app/engine/market.py:35:    def __iter__(self) -> Iterator[Tick]:
strategy-service/app/engine/market.py:38:    def __len__(self) -> int:
strategy-service/app/engine/market.py:42:    def synthetic(self) -> bool:
strategy-service/app/engine/market.py:49:class MarketSimulator:
strategy-service/app/engine/market.py:60:    def __init__(self, bars: List[Bar]) -> None:
strategy-service/app/engine/market.py:75:    def __len__(self) -> int:
strategy-service/app/engine/market.py:79:    def bars(self) -> List[Bar]:
strategy-service/app/engine/market.py:82:    def bar_closed_at_or_before(self, ts: int) -> int:
strategy-service/app/engine/market.py:94:    def slice_until(self, idx: int) -> dict:
strategy-service/app/engine/metrics.py:116:def _build_risk(max_dd: float, sharpe: float, total_trades: int) -> RiskAssessment:
strategy-service/app/engine/metrics.py:135:def build_metrics(
strategy-service/app/engine/metrics.py:31:def _total_return(eq: np.ndarray) -> float:
strategy-service/app/engine/metrics.py:37:def _max_drawdown(eq: np.ndarray) -> float:
strategy-service/app/engine/metrics.py:45:def _bars_per_year(bars: List[Bar]) -> int:
strategy-service/app/engine/metrics.py:55:def _sharpe_ratio(eq: np.ndarray, bars: List[Bar]) -> float:
strategy-service/app/engine/metrics.py:65:def _annual_return(total_return: float, bars: List[Bar]) -> float:
strategy-service/app/engine/metrics.py:79:def _trade_stats(trades: List[Trade]) -> dict:
strategy-service/app/engine/migrate_legacy_strategy.py:101:def convert_to_dsl(code: str) -> tuple[str, list[str], list[str]]:
strategy-service/app/engine/migrate_legacy_strategy.py:139:def _expr_to_dsl(node: ast.AST, depth: int = 0) -> str:
strategy-service/app/engine/migrate_legacy_strategy.py:221:def migrate_strategy(strategy_id: str, code: str) -> MigrationResult:
strategy-service/app/engine/migrate_legacy_strategy.py:251:def main() -> None:
strategy-service/app/engine/migrate_legacy_strategy.py:84:class MigrationResult:
strategy-service/app/engine/migrate_legacy_strategy.py:93:def is_convertible(code: str) -> tuple[bool, str]:
strategy-service/app/engine/params_extractor.py:111:def _is_params_alias(value: ast.AST) -> bool:
strategy-service/app/engine/params_extractor.py:128:def _collect_alias_names(tree: ast.AST) -> Set[str]:
strategy-service/app/engine/params_extractor.py:139:def extract_required_params(code: str) -> List[Dict[str, Any]]:
strategy-service/app/engine/params_extractor.py:150:    def remember(spec: ParamSpec) -> None:
strategy-service/app/engine/params_extractor.py:224:def _coerce_strategy_value(key: str, raw: str) -> Any:
strategy-service/app/engine/params_extractor.py:241:def extract_strategy_directives(code: str) -> Dict[str, Any]:
strategy-service/app/engine/params_extractor.py:30:class ParamSpec:
strategy-service/app/engine/params_extractor.py:37:    def to_dict(self) -> Dict[str, Any]:
strategy-service/app/engine/params_extractor.py:81:def _suggest_default(key: str) -> Optional[Dict[str, Any]]:
strategy-service/app/engine/params_extractor.py:90:def _literal_value(node: ast.AST) -> Any:
strategy-service/app/engine/params_extractor.py:99:def _type_of(value: Any) -> Optional[str]:
strategy-service/app/engine/portfolio.py:105:    def equity(self, tick: Tick) -> float:
strategy-service/app/engine/portfolio.py:111:    def adjust_cash(self, delta: float) -> None:
strategy-service/app/engine/portfolio.py:115:    def set_cash(self, new_cash: float) -> None:
strategy-service/app/engine/portfolio.py:119:    def apply_fill(self, fill: Fill, order: Order, tick: Tick) -> Position:
strategy-service/app/engine/portfolio.py:139:    def close_position(
strategy-service/app/engine/portfolio.py:170:    def check_sl_tp(self, tick: Tick) -> List[Trade]:
strategy-service/app/engine/portfolio.py:191:    def force_liquidate_all(self, tick: Tick, reason: CloseReason) -> List[Trade]:
strategy-service/app/engine/portfolio.py:199:    def force_liquidate_side(self, tick: Tick, side: Side, reason: CloseReason) -> List[Trade]:
strategy-service/app/engine/portfolio.py:37:def _order_side(order_type: OrderType) -> Side:
strategy-service/app/engine/portfolio.py:46:class Portfolio:
strategy-service/app/engine/portfolio.py:49:    def __init__(
strategy-service/app/engine/portfolio.py:65:    def cash(self) -> float:
strategy-service/app/engine/portfolio.py:70:    def positions(self) -> List[Position]:
strategy-service/app/engine/portfolio.py:74:    def closed_trades(self) -> List[Trade]:
strategy-service/app/engine/portfolio.py:78:    def legacy_pnl(self) -> bool:
strategy-service/app/engine/portfolio.py:81:    def has_open(self) -> bool:
strategy-service/app/engine/portfolio.py:84:    def positions_total(self) -> int:
strategy-service/app/engine/portfolio.py:89:    def _unit_pnl(self, side: Side, open_price: float, close_price: float) -> float:
strategy-service/app/engine/portfolio.py:95:    def unrealized(self, position: Position, tick: Tick) -> float:
strategy-service/app/engine/runner.py:102:    def slice(self, count: int) -> List[float]:
strategy-service/app/engine/runner.py:106:class _EngineBars(Bars):
strategy-service/app/engine/runner.py:107:    def __init__(self, market: MarketSimulator, timeframe: str = ""):
strategy-service/app/engine/runner.py:116:    def total(self) -> int:
strategy-service/app/engine/runner.py:122:def _is_sdk_strategy(code: str) -> bool:
strategy-service/app/engine/runner.py:140:class BacktestRunner:
strategy-service/app/engine/runner.py:149:    def __init__(self, req: BacktestRequest, sandbox=None) -> None:
strategy-service/app/engine/runner.py:210:    def _init_sdk_path(self, req: BacktestRequest) -> None:
strategy-service/app/engine/runner.py:227:        def bars_provider(timeframe=None):
strategy-service/app/engine/runner.py:242:    def run(self) -> BacktestResult:
strategy-service/app/engine/runner.py:269:    def _run_loop(self) -> None:
strategy-service/app/engine/runner.py:327:    def _dispatch_signal(self, signal, tick: Tick) -> None:
strategy-service/app/engine/runner.py:391:    def _open_event(self, pos: Position, fill: Fill) -> dict:
strategy-service/app/engine/runner.py:400:    def _close_event(self, trade) -> dict:
strategy-service/app/engine/runner.py:409:    def _build_snapshot(self):
strategy-service/app/engine/runner.py:420:    def _build_execution_assumptions(self):
strategy-service/app/engine/runner.py:434:def run_backtest(req: BacktestRequest) -> BacktestResult:
strategy-service/app/engine/runner.py:70:def _parse_expiration(raw: Any) -> Optional[int]:
strategy-service/app/engine/runner.py:89:class _EngineSeries(Series):
strategy-service/app/engine/runner.py:90:    def __init__(self, data: np.ndarray):
strategy-service/app/engine/runner.py:93:    def __getitem__(self, shift: int) -> float:
strategy-service/app/engine/runner.py:99:    def __len__(self) -> int:
strategy-service/app/engine/sandbox_os.py:122:def _build_seccomp_filter() -> bytes:
strategy-service/app/engine/sandbox_os.py:156:def seccomp_is_available() -> bool:
strategy-service/app/engine/sandbox_os.py:165:def _get_libc():
strategy-service/app/engine/sandbox_os.py:169:def apply_seccomp() -> bool:
strategy-service/app/engine/sandbox_os.py:203:def drop_root(uid: Optional[int] = None, gid: Optional[int] = None) -> bool:
strategy-service/app/engine/sandbox_os.py:220:def is_root() -> bool:
strategy-service/app/engine/sandbox_os.py:229:def _cgroup_v2_available() -> bool:
strategy-service/app/engine/sandbox_os.py:233:def apply_cgroup_limits(
strategy-service/app/engine/sandbox_os.py:265:def apply_os_sandbox(
strategy-service/app/engine/sandbox_os.py:294:def can_open_network() -> bool:
strategy-service/app/engine/sandbox_os.py:304:def can_write_file(path: str = "/tmp/sandbox_escape_test") -> bool:
strategy-service/app/engine/sandbox_os.py:314:def can_spawn_process() -> bool:
strategy-service/app/engine/sandbox_os.py:56:def _bpf_stmt(code: int, k: int) -> "sock_filter":
strategy-service/app/engine/sandbox_os.py:60:def _bpf_jump(code: int, k: int, jt: int, jf: int) -> "sock_filter":
strategy-service/app/engine/sandbox_os.py:64:class sock_filter(ctypes.Structure):
strategy-service/app/engine/sandbox_os.py:72:    def pack(self) -> bytes:
strategy-service/app/engine/sandbox_os.py:76:class sock_fprog(ctypes.Structure):
strategy-service/app/engine/sdk_indicators.py:12:class SDKIndicators:
strategy-service/app/engine/sdk_indicators.py:20:    def __init__(self, bars_provider):
strategy-service/app/engine/sdk_indicators.py:23:    def _get_close(self) -> np.ndarray:
strategy-service/app/engine/sdk_indicators.py:27:    def ma(self, period=14, shift=0, method="sma"):
strategy-service/app/engine/sdk_indicators.py:40:    def ema(self, period=14, shift=0):
strategy-service/app/engine/sdk_indicators.py:43:    def rsi(self, period=14, shift=0):
strategy-service/app/engine/sdk_indicators.py:55:    def bands(self, period=20, deviation=2.0, shift=0):
strategy-service/app/engine/sdk_indicators.py:64:    def macd(self, fast=12, slow=26, signal=9, shift=0):
strategy-service/app/engine/sdk_indicators.py:67:    def atr(self, period=14, shift=0):
strategy-service/app/engine/sdk_indicators.py:70:    def stochastic(self, k_period=5, d_period=3, shift=0):
strategy-service/app/engine/sdk_indicators.py:73:    def cci(self, period=14, shift=0):
strategy-service/app/engine/sdk_indicators.py:76:    def i_custom(self, name, params=(), buffer=0, shift=0):
strategy-service/app/engine/sdk_loader.py:13:def load_sdk_strategy(code: str) -> type[StrategyBase]:
strategy-service/app/engine/sdk_worker.py:102:def process_bar(code: str, bar_context: dict) -> Dict[str, Any]:
strategy-service/app/engine/sdk_worker.py:161:def reset_runtime():
strategy-service/app/engine/sdk_worker.py:35:def _get_or_create_runtime(code: str, bar_context: dict) -> StrategyRuntime:
strategy-service/app/engine/sdk_worker.py:65:def _build_bars_provider(bar_context: dict) -> callable:
strategy-service/app/engine/sdk_worker.py:74:    class _LiveSeries(Series):
strategy-service/app/engine/sdk_worker.py:75:        def __init__(self, data):
strategy-service/app/engine/sdk_worker.py:77:        def __getitem__(self, shift):
strategy-service/app/engine/sdk_worker.py:82:        def __len__(self):
strategy-service/app/engine/sdk_worker.py:84:        def slice(self, count):
strategy-service/app/engine/sdk_worker.py:87:    def provider(timeframe=None):
strategy-service/app/engine/sim_broker.py:103:def _is_market_order(ot: SDKOrderType) -> bool:
strategy-service/app/engine/sim_broker.py:107:def _is_pending_order(ot: SDKOrderType) -> bool:
strategy-service/app/engine/sim_broker.py:115:class _PositionMeta:
strategy-service/app/engine/sim_broker.py:122:class _OrderMeta:
strategy-service/app/engine/sim_broker.py:131:class SimBroker(Broker):
strategy-service/app/engine/sim_broker.py:149:    def __init__(
strategy-service/app/engine/sim_broker.py:185:    def order_send(self, request: OrderRequest) -> OrderResult:
strategy-service/app/engine/sim_broker.py:220:    def position_modify(
strategy-service/app/engine/sim_broker.py:235:    def position_close(
strategy-service/app/engine/sim_broker.py:289:    def order_delete(self, ticket: int) -> OrderResult:
strategy-service/app/engine/sim_broker.py:302:    def positions(
strategy-service/app/engine/sim_broker.py:319:    def orders(
strategy-service/app/engine/sim_broker.py:351:    def account(self) -> AccountInfo:
strategy-service/app/engine/sim_broker.py:379:    def symbol_info(self, symbol: str) -> SymbolInfo:
strategy-service/app/engine/sim_broker.py:401:    def server_time(self) -> int:
strategy-service/app/engine/sim_broker.py:408:    def advance_tick(self, tick: Tick) -> List[Fill]:
strategy-service/app/engine/sim_broker.py:432:    def sync_account_state(self) -> None:
strategy-service/app/engine/sim_broker.py:438:    def _execute_market(self, order: Order, tick: Tick) -> OrderResult:
strategy-service/app/engine/sim_broker.py:451:    def _enqueue_pending(self, order: Order, tick: Tick) -> OrderResult:
strategy-service/app/engine/sim_broker.py:468:    def _on_fill(self, fill: Fill, order: Order) -> OrderResult:
strategy-service/app/engine/sim_broker.py:493:    def _find_position(self, ticket: int) -> Optional[EnginePosition]:
strategy-service/app/engine/sim_broker.py:499:    def _to_sdk_position(
strategy-service/app/engine/sim_broker.py:60:def _to_float(d: Optional[Decimal]) -> float:
strategy-service/app/engine/sim_broker.py:66:def _to_decimal(f: float, precision: Decimal = _DECIMAL_PRECISION) -> Decimal:
strategy-service/app/engine/sim_broker.py:74:def _to_decimal_or_none(f: float) -> Optional[Decimal]:
strategy-service/app/engine/types.py:116:class Order:
strategy-service/app/engine/types.py:131:class Fill:
strategy-service/app/engine/types.py:141:class Position:
strategy-service/app/engine/types.py:152:class Trade:
strategy-service/app/engine/types.py:169:class Metrics:
strategy-service/app/engine/types.py:184:class RiskAssessment:
strategy-service/app/engine/types.py:18:class Side(str, Enum):
strategy-service/app/engine/types.py:193:class ExecutionAssumptions:
strategy-service/app/engine/types.py:206:class RunSnapshot:
strategy-service/app/engine/types.py:219:class BacktestRequest:
strategy-service/app/engine/types.py:23:class OrderType(str, Enum):
strategy-service/app/engine/types.py:258:class BacktestResult:
strategy-service/app/engine/types.py:274:class EngineError(Exception):
strategy-service/app/engine/types.py:278:class StrategyCompileError(EngineError):
strategy-service/app/engine/types.py:282:class StrategyRuntimeError(EngineError):
strategy-service/app/engine/types.py:286:class DataUnavailableError(EngineError):
strategy-service/app/engine/types.py:290:class DeadlineExceededError(EngineError):
strategy-service/app/engine/types.py:294:class MarginCallError(EngineError):
strategy-service/app/engine/types.py:36:class OrderStatus(str, Enum):
strategy-service/app/engine/types.py:45:class RunMode(str, Enum):
strategy-service/app/engine/types.py:52:class SlippageMode(str, Enum):
strategy-service/app/engine/types.py:57:class CloseReason(str, Enum):
strategy-service/app/engine/types.py:70:class Bar:
strategy-service/app/engine/types.py:83:class Tick:
strategy-service/app/engine/types.py:95:class CostProfile:
strategy-service/app/engine/validation.py:106:def scan_security(code: str) -> SecurityScanResult:
strategy-service/app/engine/validation.py:139:class StrategyValidationResult:
strategy-service/app/engine/validation.py:146:def _is_sdk_strategy(tree) -> bool:
strategy-service/app/engine/validation.py:158:def _validate_sdk_strategy(
strategy-service/app/engine/validation.py:193:def validate_strategy_code(code: str) -> StrategyValidationResult:
strategy-service/app/engine/validation.py:44:class SecurityScanResult:
strategy-service/app/engine/validation.py:53:class _SecurityVisitor(ast.NodeVisitor):
strategy-service/app/engine/validation.py:56:    def __init__(self, result: SecurityScanResult):
strategy-service/app/engine/validation.py:59:    def visit_Import(self, node):
strategy-service/app/engine/validation.py:70:    def visit_ImportFrom(self, node):
strategy-service/app/engine/validation.py:81:    def visit_Call(self, node):
strategy-service/app/engine/validation.py:89:def _scan_strings_for_danger(code: str, result: SecurityScanResult) -> None:
strategy-service/app/engine/vectorized_runner.py:106:    def __init__(self, source: str, timeout_ms: int = 30_000) -> None:
strategy-service/app/engine/vectorized_runner.py:123:    def source_sha256(self) -> str:
strategy-service/app/engine/vectorized_runner.py:126:    def call(self, ctx: dict) -> Optional[dict]:
strategy-service/app/engine/vectorized_runner.py:134:    def shutdown(self) -> None:
strategy-service/app/engine/vectorized_runner.py:137:    def call_dataframe(
strategy-service/app/engine/vectorized_runner.py:172:    def _build_globals(self) -> dict:
strategy-service/app/engine/vectorized_runner.py:176:def extract_signal_at(
strategy-service/app/engine/vectorized_runner.py:234:def _run_general_validation(source: str) -> StrategyValidationResult:
strategy-service/app/engine/vectorized_runner.py:243:def detect_strategy_type(code: str) -> str:
strategy-service/app/engine/vectorized_runner.py:44:class DataFrameValidationResult:
strategy-service/app/engine/vectorized_runner.py:51:def validate_dataframe_code(code: str) -> DataFrameValidationResult:
strategy-service/app/engine/vectorized_runner.py:98:class DataFrameStrategyRunner:
strategy-service/app/sdk/account.py:16:class AccountInfo:
strategy-service/app/sdk/broker.py:22:class Broker(ABC):
strategy-service/app/sdk/broker.py:26:    def order_send(self, request: OrderRequest) -> OrderResult:
strategy-service/app/sdk/broker.py:31:    def position_modify(
strategy-service/app/sdk/broker.py:38:    def position_close(self, ticket: int, volume: Optional[Decimal] = None) -> OrderResult:
strategy-service/app/sdk/broker.py:43:    def order_delete(self, ticket: int) -> OrderResult:
strategy-service/app/sdk/broker.py:48:    def positions(self, symbol: Optional[str] = None, magic: Optional[int] = None) -> List[Position]:
strategy-service/app/sdk/broker.py:53:    def orders(self, symbol: Optional[str] = None, magic: Optional[int] = None) -> List[PendingOrder]:
strategy-service/app/sdk/broker.py:58:    def account(self) -> AccountInfo:
strategy-service/app/sdk/broker.py:63:    def symbol_info(self, symbol: str) -> SymbolInfo:
strategy-service/app/sdk/broker.py:68:    def server_time(self) -> int:
strategy-service/app/sdk/context.py:16:class Context:
strategy-service/app/sdk/context.py:22:    def bars(self, timeframe: Optional[str] = None) -> Bars:
strategy-service/app/sdk/context.py:26:    def param(self, name: str, default: object = None) -> object:
strategy-service/app/sdk/context.py:30:    def set_timer(self, seconds: int) -> None:
strategy-service/app/sdk/context.py:34:    def kill_timer(self) -> None:
strategy-service/app/sdk/indicators.py:16:class Indicators:
strategy-service/app/sdk/indicators.py:19:    def ma(self, period: int = 14, shift: int = 0, method: str = "sma") -> float:
strategy-service/app/sdk/indicators.py:22:    def ema(self, period: int = 14, shift: int = 0) -> float:
strategy-service/app/sdk/indicators.py:25:    def rsi(self, period: int = 14, shift: int = 0) -> float:
strategy-service/app/sdk/indicators.py:28:    def bands(
strategy-service/app/sdk/indicators.py:34:    def macd(
strategy-service/app/sdk/indicators.py:40:    def atr(self, period: int = 14, shift: int = 0) -> float:
strategy-service/app/sdk/indicators.py:43:    def stochastic(
strategy-service/app/sdk/indicators.py:49:    def cci(self, period: int = 14, shift: int = 0) -> float:
strategy-service/app/sdk/indicators.py:52:    def i_custom(self, name: str, params: Sequence[object], buffer: int = 0, shift: int = 0) -> float:
strategy-service/app/sdk/runtime.py:119:    def on_tick(self) -> None:
strategy-service/app/sdk/runtime.py:124:    def on_bar(self, timeframe: str) -> None:
strategy-service/app/sdk/runtime.py:129:    def on_timer(self) -> None:
strategy-service/app/sdk/runtime.py:136:    def on_trade(self) -> None:
strategy-service/app/sdk/runtime.py:141:    def deinit(self, reason: str = "user_stop") -> None:
strategy-service/app/sdk/runtime.py:161:    def state(self) -> str:
strategy-service/app/sdk/runtime.py:165:    def strategy(self) -> Optional[StrategyBase]:
strategy-service/app/sdk/runtime.py:169:    def is_timer_active(self) -> bool:
strategy-service/app/sdk/runtime.py:174:    def export_intents(self) -> List[Dict[str, Any]]:
strategy-service/app/sdk/runtime.py:188:    def _require_ready(self) -> None:
strategy-service/app/sdk/runtime.py:194:    def _safe_call(self, method_name: str, *args: Any) -> Optional[Any]:
strategy-service/app/sdk/runtime.py:219:    def _register_timer(self, seconds: int) -> None:
strategy-service/app/sdk/runtime.py:225:    def _unregister_timer(self) -> None:
strategy-service/app/sdk/runtime.py:233:class RuntimeContext(Context):
strategy-service/app/sdk/runtime.py:241:    def __init__(
strategy-service/app/sdk/runtime.py:257:    def bars(self, timeframe: Optional[str] = None) -> Bars:
strategy-service/app/sdk/runtime.py:261:    def param(self, name: str, default: object = None) -> object:
strategy-service/app/sdk/runtime.py:265:    def set_timer(self, seconds: int) -> None:
strategy-service/app/sdk/runtime.py:269:    def kill_timer(self) -> None:
strategy-service/app/sdk/runtime.py:37:class StrategyRuntime:
strategy-service/app/sdk/runtime.py:59:    def __init__(
strategy-service/app/sdk/runtime.py:92:    def init(self) -> None:
strategy-service/app/sdk/series.py:13:class Series:
strategy-service/app/sdk/series.py:19:    def __getitem__(self, shift: int) -> float:
strategy-service/app/sdk/series.py:23:    def __len__(self) -> int:
strategy-service/app/sdk/series.py:26:    def slice(self, count: int) -> List[float]:
strategy-service/app/sdk/series.py:31:class Bars:
strategy-service/app/sdk/series.py:42:    def total(self) -> int:
strategy-service/app/sdk/strategy_base.py:18:class StrategyBase:
strategy-service/app/sdk/strategy_base.py:29:    def on_init(self) -> None:
strategy-service/app/sdk/strategy_base.py:32:    def on_tick(self) -> None:
strategy-service/app/sdk/strategy_base.py:35:    def on_bar(self, timeframe: str) -> None:
strategy-service/app/sdk/strategy_base.py:38:    def on_timer(self) -> None:
strategy-service/app/sdk/strategy_base.py:41:    def on_trade(self) -> None:
strategy-service/app/sdk/strategy_base.py:44:    def on_deinit(self, reason: str) -> None:
strategy-service/app/sdk/symbol.py:14:class SymbolInfo:
strategy-service/app/sdk/symbol.py:32:    def normalize_price(self, price: Decimal) -> Decimal:
strategy-service/app/sdk/symbol.py:36:    def normalize_volume(self, volume: Decimal) -> Decimal:
strategy-service/app/sdk/types.py:106:class Position:
strategy-service/app/sdk/types.py:124:class PendingOrder:
strategy-service/app/sdk/types.py:16:class PositionSide(Enum):
strategy-service/app/sdk/types.py:23:class OrderType(Enum):
strategy-service/app/sdk/types.py:36:class TypeFilling(Enum):
strategy-service/app/sdk/types.py:44:class AccountMode(Enum):
strategy-service/app/sdk/types.py:51:class DealType(Enum):
strategy-service/app/sdk/types.py:60:class Retcode(Enum):
strategy-service/app/sdk/types.py:75:class OrderRequest:
strategy-service/app/sdk/types.py:95:class OrderResult:
```

## Handler 注册（已接进生产路由 = 非 shelf-ware）

> 在此列表 = 真正可被调用；只在某 *_test.go 出现而不在此 = 货架闲置（shelf-ware）。

```
backend/cmd/server/handlers.go:103:	mux.Handle(antv1c.NewMtHubServiceHandler(mthubServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
backend/cmd/server/handlers.go:110:	mux.Handle(antv1c.NewAccountServiceHandler(accountServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
backend/cmd/server/handlers.go:113:	mux.Handle(antv1c.NewMarketServiceHandler(mktServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
backend/cmd/server/handlers.go:117:	mux.Handle(antv1c.NewMarketplaceServiceHandler(mktplaceHandler, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
backend/cmd/server/handlers.go:125:	mux.Handle(antv1c.NewExecutionAlgoServiceHandler(algoServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
backend/cmd/server/handlers.go:144:	mux.Handle(antv1c.NewAIServiceHandler(aiServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
backend/cmd/server/handlers.go:146:	mux.Handle(antv1c.NewAgentDefinitionServiceHandler(aiServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
backend/cmd/server/handlers.go:151:	mux.Handle(antv1c.NewAssetAnalysisServiceHandler(assetAnalysisServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
backend/cmd/server/handlers.go:158:	mux.Handle(antv1c.NewShareServiceHandler(shareServer, connectrpc.WithInterceptors(otelInterceptor, authInterceptor)))
backend/cmd/server/handlers.go:183:	mux.Handle(antv1c.NewAIGatewayServiceHandler(gatewayServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
backend/cmd/server/handlers.go:247:	mux.Handle(antv1c.NewStreamServiceHandler(streamServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
backend/cmd/server/handlers.go:256:	mux.Handle(antv1c.NewStrategyServiceHandler(strategyServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
backend/cmd/server/handlers.go:273:		mux.Handle(antv1c.NewObjectiveScoreServiceHandler(objectiveScoreServer,
backend/cmd/server/handlers.go:277:	mux.Handle(antv1c.NewPythonStrategyServiceHandler(pythonStrategyServer,
backend/cmd/server/handlers.go:286:	mux.Handle(antv1c.NewPaperTradingServiceHandler(paperHandler,
backend/cmd/server/handlers.go:292:	mux.Handle(antv1c.NewCodeAssistServiceHandler(codeAssistServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
backend/cmd/server/handlers.go:294:	mux.Handle(antv1c.NewSystemAIServiceHandler(systemAIServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
backend/cmd/server/handlers.go:296:	mux.Handle(antv1c.NewAIPrimaryServiceHandler(aiPrimaryServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
backend/cmd/server/handlers.go:298:	mux.Handle(antv1c.NewBacktestTradesServiceHandler(backtestTradesServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
backend/cmd/server/handlers.go:300:	mux.Handle(antv1c.NewGateServiceHandler(gateEvalServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
backend/cmd/server/handlers.go:302:	mux.Handle(antv1c.NewStrategyGenerationServiceHandler(strategyGenServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
backend/cmd/server/handlers.go:310:	mux.Handle(antv1c.NewStrategyPlanServiceHandler(strategyPlanServer, connectrpc.WithInterceptors(otelInterceptor, authInterceptor)))
backend/cmd/server/handlers.go:313:	mux.Handle(antv1c.NewEconomicDataServiceHandler(economicDataServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
backend/cmd/server/handlers.go:315:	mux.Handle(antv1c.NewJobServiceHandler(jobServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
backend/cmd/server/handlers.go:317:	mux.Handle(antv1c.NewLogServiceHandler(logServiceServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
backend/cmd/server/handlers.go:319:	mux.Handle(antv1c.NewNotificationServiceHandler(notifServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
backend/cmd/server/handlers.go:329:	mux.Handle(antv1c.NewAdminTradingServiceHandler(adminTradingServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor, adminInterceptor)))
backend/cmd/server/handlers.go:331:	mux.Handle(antv1c.NewAdminConfigServiceHandler(adminConfigServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor, adminInterceptor)))
backend/cmd/server/handlers.go:333:	mux.Handle(antv1c.NewAdminLogServiceHandler(adminLogServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor, adminInterceptor)))
backend/cmd/server/handlers.go:335:	mux.Handle(antv1c.NewAdminAccountServiceHandler(adminAccountServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor, adminInterceptor)))
backend/cmd/server/handlers.go:338:	mux.Handle(antv1c.NewAdminUserServiceHandler(adminUserServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor, adminInterceptor)))
backend/cmd/server/handlers.go:340:	mux.Handle(antv1c.NewAdminSystemServiceHandler(adminSystemServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor, adminInterceptor)))
backend/cmd/server/handlers.go:342:		mux.Handle(antv1c.NewAdminStrategyServiceHandler(adminStrategyServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor, adminInterceptor)))
backend/cmd/server/handlers.go:350:	mux.Handle(antv1c.NewAutoTradingServiceHandler(autoTradingServer,
backend/cmd/server/handlers.go:378:	mux.Handle(antv1c.NewAdminJurisdictionServiceHandler(adminJurisdictionServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor, adminInterceptor)))
backend/cmd/server/handlers.go:95:	mux.Handle(antv1c.NewAuthServiceHandler(authServer, connectrpc.WithInterceptors(otelInterceptor,rateLimitInterceptor, authInterceptor)))
backend/cmd/server/handlers.go:98:	mux.Handle(antv1c.NewWalletServiceHandler(walletServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
backend/cmd/server/handlers_sre.go:101:	mux.Handle(antv1c.NewIndicatorCatalogServiceHandler(indicatorCatalogServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
backend/cmd/server/handlers_sre.go:71:	mux.Handle(antv1c.NewAdminSREServiceHandler(sreHandler, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
backend/cmd/server/handlers_sre.go:75:	mux.Handle(antv1c.NewAnalyticsServiceHandler(analyticsServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
backend/cmd/server/handlers_sre.go:79:	mux.Handle(antv1c.NewMarketRegimeServiceHandler(marketRegimeServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
backend/cmd/server/handlers_sre.go:83:	mux.Handle(antv1c.NewStrategyExperimentServiceHandler(strategyExperimentServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
backend/cmd/server/handlers_sre.go:97:	mux.Handle(antv1c.NewStrategyAssetServiceHandler(strategyAssetServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
backend/cmd/server/handlers_sre.go:99:	mux.Handle(antv1c.NewScheduleHealthServiceHandler(scheduleHealthServer, connectrpc.WithInterceptors(otelInterceptor,authInterceptor)))
```
