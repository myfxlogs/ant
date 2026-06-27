# AntTrader — 已实现功能管线全览

> 生成日期：2026-06-26
> 范围：`/opt/ant/backend` + `/opt/ant/frontend`

---

## 一、后端 ConnectRPC 服务（42 个）

| # | 服务 | Handler 文件 | 功能 |
|---|------|-------------|------|
| 1 | AuthService | `connect/user/auth_handler.go` | 登录/登出/JWT 刷新/注册 |
| 2 | WalletService | `connect/user/wallet_handler.go` | 钱包余额查询/充值/扣费 |
| 3 | AccountService | `connect/user/account_handler.go` | MT4/5 账户 CRUD、连接测试、经纪商搜索 |
| 4 | ShareService | `connect/user/share_handler.go` | 交易绩效分享链接生成 + REST 端点 |
| 5 | MtHubService | `connect/system/mthub_service.go` | 账户列表、持仓、订单、品种信息查询 |
| 6 | StreamService | `connect/system/stream_handler.go` | SSE 实时推送：订单事件/利润/持仓快照/Bar/账户状态 |
| 7 | AnalyticsService | `connect/system/analytics_handler.go` | 交易分析、收益曲线、Sharpe/回撤等指标 |
| 8 | JobService | `connect/system/job_handler.go` | 异步任务状态查询 |
| 9 | LogService | `connect/system/log_handler.go` | 用户日志查询 |
| 10 | EconomicDataService | `connect/system/economic_data_handler.go` | 经济日历（stub，未接入外部 API） |
| 11 | ScheduleHealthService | `connect/system/schedule_health_handler.go` | 策略调度健康检查 |
| 12 | NotificationService | `connect/notification/handler.go` | 通知列表/SSE 实时推送 |
| 13 | MarketService | `connect/marketplace/market_handler.go` | 品种搜索、K线数据、NATS 实时报价 |
| 14 | MarketplaceService | `connect/marketplace/marketplace_handler.go` | 策略市场 CRUD、订阅续费、代码访问控制 |
| 15 | MarketRegimeService | `connect/marketplace/market_regime_handler.go` | 市场状态识别（趋势/震荡） |
| 16 | IndicatorCatalogService | `connect/marketplace/indicator_catalog_handler.go` | 指标目录查询 |
| 17 | StrategyService | `connect/strategy/strategy_handler.go` | 策略 CRUD、模板草稿、信号列表/执行/确认/取消 |
| 18 | StrategyRuntimeService | `connect/strategy/strategy_execution_handler.go` | 策略执行/验证/回测/模板/Transpile（Go 原生执行） |
| 19 | BacktestTradesService | `connect/strategy/backtest_trades_handler.go` | 回测交易明细查询 |
| 20 | StrategyExperimentService | `connect/strategy/strategy_experiment_handler.go` | 策略参数实验 CRUD + SSE |
| 21 | StrategyAssetService | `connect/strategy/strategy_asset_handler.go` | 策略资产管理 |
| 22 | CodeAssistService | `connect/ai/code_assist_handler.go` | AI 代码辅助：修订/修复/解释/转换/验证（流式） |
| 23 | AIService | `connect/ai/ai_handler.go` | AI 对话、Agent 定义 CRUD |
| 24 | AIPrimaryService | `connect/ai/ai_primary_handler.go` | AI 主模型直接调用 |
| 25 | SystemAIService | `connect/ai/system_ai_handler.go` | AI Provider 配置 CRUD、密钥加密存储 |
| 26 | StrategyGenerationService | `connect/ai/strategy_gen_handler.go` | AI 策略生成（含澄清轮次 + 自动回测） |
| 27 | StrategyPlanService | `connect/ai/strategy_plan_handler.go` | AI 策略规划→执行两步管线 |
| 28 | GateService | `connect/ai/gate_eval_handler.go` | 风控 Gate 评估 SSE 流 |
| 29 | AIGatewayService | `connect/gateway/ai_gateway_handler.go` | AI 网关：Provider/Model/Token 计费管理 |
| 30 | AssetAnalysisService | `connect/asset_analysis/handler.go` | 多时间框架分析、支撑/阻力、波动率、AI 推荐 |
| 31 | ExecutionAlgoService | `connect/algo/execution_algo_handler.go` | TWAP/VWAP/POV/Shortfall 执行算法 |
| 32 | AutoTradingService | `connect/autotrading/server.go` | 自动交易配置、风控规则 CRUD |
| 33 | PaperTradingService | `connect/paper/handler.go` | 模拟交易：下单/平仓/持仓查询 |
| 34 | AdminTradingService | `connect/admin/admin_trading_handler.go` | 管理员交易监控 |
| 35 | AdminConfigService | `connect/admin/admin_config_handler.go` | 管理员系统配置 |
| 36 | AdminLogService | `connect/admin/admin_log_handler.go` | 管理员日志查看 |
| 37 | AdminAccountService | `connect/admin/admin_account_handler.go` | 管理员账户管理 |
| 38 | AdminUserService | `connect/admin/admin_user_handler.go` | 管理员用户管理/密码重置/删除 |
| 39 | AdminSystemService | `connect/admin/admin_system_handler.go` | 管理员系统信息 |
| 40 | AdminStrategyService | `connect/admin/admin_strategy_handler.go` | 管理员策略管理 |
| 41 | AdminJurisdictionService | `connect/admin/admin_jurisdiction_handler.go` | 管理员司法管辖区配置 |
| 42 | AdminSREService | `connect/admin/` | SRE Kill Switch、熔断器、金丝雀发布 |

---

## 二、后端后台 Worker / 循环（13 个）

| # | Worker | 文件 | 触发方式 | 功能 |
|---|--------|------|---------|------|
| 1 | MdGateway Pipeline | `cmd/server/pipeline.go` → `mdgateway/runner.go` | goroutine 启动 | MT4/5 tick → 归一化 → 聚合 → PG 写入 + NATS 发布 + 回调 |
| 2 | BacktestWorker (×3) | `strategy/backtest_worker.go` | PG LISTEN `backtest_pending` + 30s ticker | 认领 PENDING 回测 → Go 原生引擎执行 → 结果持久化 |
| 3 | ExperimentWorker | `strategy/strategy_experiment_worker.go` | PG LISTEN `experiment_status` + 10s ticker | 策略参数实验：网格搜索/AI 多轮提案 → 回测 → 评分 |
| 4 | ScheduleEngine | `strategy/schedule_engine.go` | 定时器循环 | 策略调度：到时触发 → RunLiveStrategy → Bar 流 → 信号 → OMS |
| 5 | ReconciliationLoop | `mthub/reconciliation.go` | 事件驱动（gateway 连接/重连） | 账户对账：比较 ant 端 vs broker 端持仓/订单状态 |
| 6 | ReflectionWorker | `ai/reflection_worker.go` | 定时循环 | AI 预测校准：验证历史预测 → 重新校准置信度 |
| 7 | NormalizerInvalidator | `mdgateway/normalizer_invalidator.go` | PG LISTEN + 30s ticker | 缓存失效：市场数据变更时清除归一化缓存 |
| 8 | PgWriter | `mdgateway/pg_writer.go` | 定时 flush | 批量写入 tick/bar 数据到 PG |
| 9 | UserMetricsFlusher | `mdgateway/user_metrics_flusher.go` | 定时 flush | 用户级指标聚合写入 |
| 10 | Marketplace Renewal Loop | `marketplace/` | 24h ticker | 订阅续费检查 |
| 11 | User Snapshot Cleanup | `cmd/server/main.go` | 24h ticker | 清理过期账户快照 |
| 12 | Hard-Delete Expired Users | `cmd/server/handlers.go` | 启动时 + 24h ticker | 30 天后硬删除已软删用户 |
| 13 | PlatformAggregator | `risksvc/` | 5s 刷新循环 | 平台级聚合指标刷新 |

---

## 三、后端数据管线（11 条）

| # | 管线 | 数据流 |
|---|------|--------|
| 1 | Tick → Bar 管线 | MT4/5 gRPC → adapter → normalizer → tick_dedup → bar_aggregator → pg_writer + publisher(NATS) |
| 2 | SSE 实时推送管线 | mthub brokers (Order/Profit/Snapshot/Bar/Status) → StreamServer.SubscribeEvents → 多路 select → StreamEvent proto → 前端 SSE |
| 3 | 策略回测管线 | StartBacktestRun → PG INSERT + NOTIFY → BacktestWorker 认领 → extractBacktestParams → K线拉取 → Go 原生引擎 → 结果持久化 → 通知 |
| 4 | AI 策略生成管线 | StrategyGenServer → param_extractor → strategy_prompt → LLM → code_compliance → ApplyOverrides → 自动回测 → 反馈迭代 |
| 5 | AI 代码辅助管线 | CodeAssistServer → prompt_context(分类意图) → LLM 流式 → extractCodeFromRepair → buildValidationPrompt |
| 6 | 风控 Gate 管线 | 策略信号 → risk.Gate (JurisdictionGate + CapabilityStore + UserRiskConfig) → 通过/拒绝 → OMS → MT4/5 下单 |
| 7 | Margin Call 管线 | OnAccountProfit 回调 → 3 级检测 (预警/警告/危急) → SSE 推送 + Email 通知 |
| 8 | 订单事件管线 | MT4/5 gRPC → OnOrderUpdate → TradeEventStore(NATS JetStream) → OrderEventBroker → SSE + TradeRecordRepository |
| 9 | 账户同步管线 | OnBrokerInfo/OnAccountDisconnect → AccountSyncService.SyncAccountHistory → 交易历史同步 → DB 更新 |
| 10 | AI 计费管线 | LLM 调用 → TokenRecorder(用量记录) + PostCallBiller(扣费) → AITokenUsageRepository + WalletService |
| 11 | MQL→Go 转换管线 | tools/mql2go (tree-sitter cgo) → MQL4/5 EA → Go SDK 策略代码 |

---

## 四、后端基础设施

| # | 组件 | 文件 | 功能 |
|---|------|------|------|
| 1 | PG LISTEN/NOTIFY | `pglisten/` | Push-first 事件分发（backtest/experiment/normalizer） |
| 2 | Secrets 加解密 | `secrets/` + `pkg/secretbox/` | 账户密码/AI API Key 加密存储 |
| 3 | OpenTelemetry | `trace/` + `otelconnect` | 全链路追踪，每个 RPC 自动 span |
| 4 | Prometheus Metrics | `mdgateway/metrics.go` | `/metrics` 端点，backpressure/quality 直方图 |
| 5 | Circuit Breaker | `mdgateway/circuit_breaker.go` + `controlplane/` | 熔断器：保护下游 MT4/5 gRPC |
| 6 | SSE Keepalive | `interceptor/` | 10s 心跳防 Cloudflare/proxy 超时 |
| 7 | Auth Interceptor | `interceptor/` | JWT 解析 + 用户上下文注入 |
| 8 | Rate Limiter | `interceptor/` | 登录限流（per-minute） |
| 9 | Admin Interceptor | `interceptor/` | 管理员权限校验 |
| 10 | DLQ Writer | `mdgateway/dlq_writer.go` | 死信队列：无法处理的 tick 写入 spill dir |
| 11 | Quote Stuffing 检测 | `mdgateway/quote_stuffing.go` | 异常报价频率检测 |
| 12 | Market State | `mdgateway/market_state.go` | 市场开收市状态管理 |
| 13 | Idempotency Guard | `mthub/` | Redis 幂等性保护（重复订单检测） |

---

## 五、后端策略 SDK / 回测引擎

| # | 组件 | 路径 | 功能 |
|---|------|------|------|
| 1 | Go Strategy SDK | `strategy/sdk/` | sdk.Strategy 接口、OnInit/OnBar/OnDeinit、ctx.Param/Bars/Broker/Indicators |
| 2 | SimBroker | `strategy/backtest/` | 模拟经纪商：订单匹配、滑点、佣金、保证金 |
| 3 | Backtest Engine | `strategy/backtest/` | 回测引擎：K线回放 → 策略执行 → 信号 → SimBroker 成交 |
| 4 | LiveRunner | `strategy/runner/` + `connect/strategy/live_runner.go` | 实时运行：Bar 流订阅 → 策略执行 → 信号 → OMS/paper |
| 5 | GoExecutor | `connect/strategy/go_executor.go` | go run 编译执行策略文件，proto binary IPC |
| 6 | ScheduleEngine | `connect/strategy/schedule_engine.go` | 定时调度：cron-like 触发 LiveRunner |
| 7 | Strategy Templates | `connect/strategy/strategy_templates.go` | 内置模板：MA Crossover / RSI / Bollinger（Go SDK） |

---

## 六、前端页面（按路由分组）

### 6.1 公共页面（无需登录）

| # | 路由 | 页面文件 | 功能 |
|---|------|---------|------|
| 1 | `/login` | `pages/auth/Login.tsx` | 登录 |
| 2 | `/register` | `pages/auth/Register.tsx` | 注册 |
| 3 | `/forgot-password` | `pages/auth/ForgotPassword.tsx` | 忘记密码 |
| 4 | `/share/:token` | `pages/share/SharePerformancePage.tsx` | 公开绩效分享页（独立，无 SSE） |

### 6.2 主应用页面（需登录）

| # | 路由 | 页面文件 | 功能 |
|---|------|---------|------|
| 5 | `/` | `pages/dashboard/Dashboard.tsx` | 仪表盘：账户概览、统计卡片、快捷操作 |
| 6 | `/accounts/:id` | `pages/accounts/AccountDetail.tsx` | 账户详情：持仓、订单、图表、交易历史 |
| 7 | `/accounts/:id/report` | `pages/accounts/AccountReport.tsx` | 账户报告：收益分析、风险指标 |
| 8 | `/accounts/bind` | `pages/accounts/BindAccount.tsx` | 绑定 MT4/5 账户（多步骤向导） |
| 9 | `/profile` | `pages/profile/ProfilePage.tsx` | 个人资料 |
| 10 | `/wallet` | `pages/wallet/WalletPage.tsx` | 钱包：余额、充值、交易记录 |
| 11 | `/strategy/library` | `pages/strategy/StrategyLibraryPage.tsx` | 策略库：模板列表、回测历史、调度管理 |
| 12 | `/strategy/workspace` | `pages/strategy/StrategyWorkspacePage.tsx` | 策略工作台：代码编辑器、AI 对话、回测面板、风控 Gate |
| 13 | `/strategy/schedules/:id/logs` | `pages/strategy/StrategyScheduleLogsPage.tsx` | 调度日志查看 |
| 14 | `/strategy/indicator-catalog` | `pages/strategy/IndicatorCatalogPage.tsx` | 指标目录浏览 |
| 15 | `/strategy/experiments` | `pages/strategy/StrategyExperimentPage.tsx` | 策略参数实验：网格搜索、AI 提案、对比分析 |
| 16 | `/strategy/market-tools` | `pages/strategy/MarketToolsPage.tsx` | 市场工具（Tab：品种分析 + 市场状态） |
| 17 | `/marketplace` | `pages/marketplace/MarketplacePage.tsx` | 策略市场：浏览/购买/发布策略 |
| 18 | `/logs` | `pages/logs/LogManagement.tsx` | 用户日志管理 |
| 19 | `/auto-trading` | `pages/auto-trading/AutoTradingSettings.tsx` | 自动交易设置：风控配置、交易日志 |
| 20 | `/trading/algos` | `pages/trading/AlgoDashboard.tsx` | 执行算法面板：TWAP/VWAP/POV 提交 |
| 21 | `/analytics` | `pages/analytics/Summary.tsx` | 交易分析汇总 |

### 6.3 管理员页面（需 Admin 权限）

| # | 路由 | 页面文件 | 功能 |
|---|------|---------|------|
| 22 | `/admin` | `pages/admin/Dashboard.tsx` | 管理员仪表盘 |
| 23 | `/admin/users` | `pages/admin/UserManagement.tsx` | 用户管理：CRUD、密码重置、软删除 |
| 24 | `/admin/wallet` | `pages/admin/WalletManagement.tsx` | 钱包管理：用户余额、交易记录 |
| 25 | `/admin/accounts` | `pages/admin/AccountManagement.tsx` | 账户管理 |
| 26 | `/admin/trading` | `pages/admin/TradingMonitor.tsx` | 交易监控 |
| 27 | `/admin/logs` | `pages/admin/OperationLogs.tsx` | 操作日志 |
| 28 | `/admin/config` | `pages/admin/SystemConfig.tsx` | 系统配置 |
| 29 | `/admin/jurisdiction` | `pages/admin/JurisdictionGate.tsx` | 司法管辖区配置 |
| 30 | `/admin/strategies` | `pages/admin/StrategyManagement.tsx` | 策略管理 |
| 31 | `/admin/shares` | `pages/admin/ShareManagement.tsx` | 分享链接管理 |
| 32 | `/admin/ai-gateway` | `pages/admin/AIGatewayManagement.tsx` | AI 网关管理：Provider/Model/定价 |
| 33 | `/admin/sre/killswitch` | `pages/admin/sre/KillSwitchPage.tsx` | SRE Kill Switch |
| 34 | `/admin/sre/breakers` | `pages/admin/sre/BreakersPage.tsx` | SRE 熔断器管理 |
| 35 | `/admin/sre/canary` | `pages/admin/sre/CanaryPage.tsx` | SRE 金丝雀发布 |

---

## 七、前端组件（按功能域）

### 7.1 图表组件 (`components/chart/`)

| # | 组件 | 功能 |
|---|------|------|
| 1 | PriceChart | K线图表（TradingView Lightweight Charts） |
| 2 | ChartToolbar | 图表工具栏：品种切换、周期选择 |
| 3 | SymbolPicker | 品种选择器 |
| 4 | IndicatorPicker | 指标选择器 |
| 5 | IndicatorSettingsModal | 指标参数配置弹窗 |
| 6 | ActiveIndicatorsBar | 当前激活指标条 |
| 7 | DrawingToolbar | 画图工具栏 |
| 8 | QuickTradePanel | 快速交易面板 |
| 9 | BidAskIndicator | 买卖价指示器 |
| 10 | QuoteBar | 实时报价条 |
| 11 | PositionSection | 持仓列表 |
| 12 | TradeHistorySection | 交易历史 |
| 13 | serverIndicators | 服务端指标计算 |
| 14 | useChartData | 图表数据 hook |

### 7.2 策略组件 (`components/strategy/`)

| # | 组件 | 功能 |
|---|------|------|
| 1 | StrategyChat | 策略 AI 对话主面板 |
| 2 | AIChatPanel | AI 聊天面板 |
| 3 | AIChatPanelSections | AI 聊天分区（分析/建议/代码） |
| 4 | AICodePanel | AI 生成代码展示 |
| 5 | AICodeReviseChat | AI 代码修订对话 |
| 6 | CodeAssist | 代码辅助面板 |
| 7 | StrategyCodeEditor | 策略代码编辑器（Monaco） |
| 8 | StrategyGenChat | 策略生成对话 |
| 9 | StrategyList | 策略列表 |
| 10 | BacktestRunDrawer | 回测运行抽屉 |
| 11 | DiffView | 代码差异对比 |
| 12 | ExecutionPanel | 策略执行面板 |
| 13 | ImportAnalysisReport | MQL 导入分析报告 |
| 14 | PlanPanel | AI 策略规划面板 |
| 15 | SaveTemplateModal | 保存模板弹窗 |
| 16 | StepProgress | 步骤进度条 |
| 17 | WorkflowBar | 工作流工具栏 |
| 18 | ChatMessageItem | 聊天消息项 |

### 7.3 回测组件 (`components/backtest/`)

| # | 组件 | 功能 |
|---|------|------|
| 1 | BacktestPanel | 回测参数面板 |
| 2 | BacktestResultsTab | 回测结果展示 |
| 3 | BacktestTradesTab | 回测交易明细 |
| 4 | StrategyParamsModal | 策略参数弹窗 |
| 5 | useBacktestRunner | 回测运行 hook |

### 7.4 交易组件 (`components/trade/`)

| # | 组件 | 功能 |
|---|------|------|
| 1 | AlgoSubmitForm | 执行算法提交表单 |
| 2 | AutoTradeConfirmModal | 自动交易确认弹窗 |
| 3 | RiskConfigConfirmModal | 风控配置确认弹窗 |
| 4 | StrategyExecuteConfirmModal | 策略执行确认弹窗 |
| 5 | TradeConfirmModal | 交易确认弹窗 |

### 7.5 工作台子组件 (`pages/strategy/components/workspace/`)

| # | 组件 | 功能 |
|---|------|------|
| 1 | WorkspaceCodePanel | 代码编辑面板 |
| 2 | WorkspaceToolbar | 工作台工具栏 |
| 3 | WorkspaceBacktestPanel | 工作台回测面板 |
| 4 | BacktestParamsCard | 回测参数卡片 |
| 5 | BacktestHistoryModal | 回测历史弹窗 |
| 6 | SmartTuningPanel | 智能调参面板 |
| 7 | GatePanel | 风控 Gate 面板 |
| 8 | ValidationResultAlert | 验证结果告警 |
| 9 | StrategyDirectivesCard | 策略指令卡片 |
| 10 | MiniPositionsTable | 迷你持仓表 |
| 11 | AISettingsModal | AI 设置弹窗 |
| 12 | WorkspaceTemplateManager | 模板管理器 |
| 13 | WorkspaceErrorBoundary | 错误边界 |
| 14 | MobileGuard | 移动端守卫 |

---

## 八、前端状态管理 (Zustand Stores)

| # | Store | 文件 | 管理状态 |
|---|-------|------|---------|
| 1 | authStore | `stores/authStore.ts` | 登录状态、JWT token、用户信息 |
| 2 | accountStore | `stores/accountStore.ts` | 账户列表、选中账户 |
| 3 | tradingStore | `stores/tradingStore.ts` | 交易面板状态：品种、周期、订单表单 |
| 4 | chartIndicatorsStore | `stores/chartIndicatorsStore.ts` | 图表指标配置 |
| 5 | notificationStore | `stores/notificationStore.ts` | 通知列表、未读计数 |
| 6 | workspaceStore | `stores/workspaceStore.ts` | 策略工作台状态 |
| 7 | uiStore | `stores/uiStore.ts` | 全局 UI 状态（侧边栏折叠等） |
| 8 | adminAccountStore | `stores/adminAccountStore.ts` | 管理员账户管理状态 |

---

## 九、前端 ConnectRPC 客户端 (`src/client/`)

| # | 客户端文件 | 对应后端服务 |
|---|-----------|-------------|
| 1 | `auth.ts` | AuthService |
| 2 | `account.ts` | AccountService |
| 3 | `wallet.ts` | WalletService |
| 4 | `stream.ts` | StreamService (SSE) |
| 5 | `sharedStream.ts` | 共享 SSE 订阅去重 |
| 6 | `trading.ts` | MtHubService (交易操作) |
| 7 | `market.ts` | MarketService |
| 8 | `strategy.ts` | StrategyService |
| 9 | `strategyRuntime.ts` | StrategyRuntimeService |
| 10 | `strategy-schedules.ts` | 策略调度 SSE |
| 11 | `strategyGen.ts` | StrategyGenerationService |
| 12 | `strategyPlan.ts` | StrategyPlanService |
| 13 | `codeAssist.ts` | CodeAssistService |
| 14 | `gate.ts` | GateService |
| 15 | `backtestRuns.ts` | BacktestTradesService |
| 16 | `strategyExperiment.ts` | StrategyExperimentService |
| 17 | `strategyAsset.ts` | StrategyAssetService |
| 18 | `analyticsApi.ts` | AnalyticsService |
| 19 | `analyticsMappers.ts` | Analytics 数据映射 |
| 20 | `aiGateway.ts` | AIGatewayService |
| 21 | `autoTrading.ts` | AutoTradingService |
| 22 | `admin.ts` | Admin 系列 |
| 23 | `admin-jurisdiction.ts` | AdminJurisdictionService |
| 24 | `log.ts` | LogService |
| 25 | `job.ts` | JobService |
| 26 | `scheduleHealth.ts` | ScheduleHealthService |
| 27 | `marketRegime.ts` | MarketRegimeService |
| 28 | `connect.ts` | ConnectRPC 客户端工厂 |
| 29 | `transport.ts` | HTTP 传输层（拦截器、token 刷新） |

---

## 十、前端 Hooks (`src/hooks/`)

| # | Hook | 功能 |
|---|------|------|
| 1 | useAuth | 登录状态、权限检查 |
| 2 | useAccount | 账户数据获取、缓存 |
| 3 | useTrading | 交易操作、持仓订阅 |
| 4 | useRealtimeUpdates | SSE 实时更新入口 |
| 5 | useNotificationListener | 通知 SSE 监听 |
| 6 | useServerIndicators | 服务端指标加载 |
| 7 | useWatchBacktestRun | 回测运行状态轮询 |
| 8 | useRpcQuery | ConnectRPC 查询封装 |
| 9 | useRpcMutation | ConnectRPC 变更封装 |
| 10 | useThrottle | 节流 hook |

---

## 十一、前端策略 Hooks (`pages/strategy/hooks/`)

| # | Hook | 功能 |
|---|------|------|
| 1 | useAIWorkflow | AI 策略生成全流程 |
| 2 | useStrategyWorkspaceState | 工作台状态管理 |
| 3 | useStrategyCode | 策略代码管理 |
| 4 | useBacktestModal | 回测弹窗控制 |
| 5 | useBacktestParams | 回测参数管理 |
| 6 | useBacktestDefaults | 回测默认值 |
| 7 | useGateEvaluation | 风控 Gate 评估 |
| 8 | useTuning | 参数调优 |
| 9 | useLibraryTemplates | 模板库管理 |
| 10 | useLibraryRuns | 回测运行历史 |
| 11 | useLibrarySchedules | 调度管理 |
| 12 | useStrategyTemplateRuns | 模板回测运行 |
| 13 | useAccountsAndSymbols | 账户和品种选择 |
| 14 | useAssetAnalysis | 资产分析 |
| 15 | useMarketRegimeForm | 市场状态表单 |
| 16 | useQuickTradeData | 快速交易数据 |
| 17 | useStrategyLibrary | 策略库列表 |
| 18 | useWorkspaceSession | 工作台会话 |
| 19 | backtestParamHelpers | 回测参数工具函数 |
| 20 | libraryTypes | 库类型定义 |

---

## 十二、前端 Providers

| # | Provider | 功能 |
|---|---------|------|
| 1 | StreamProvider | SSE 连接管理：自动订阅用户账户事件 |
| 2 | QueryProvider | TanStack Query 全局配置 |
| 3 | LocaleProvider | i18n 多语言（中/英） |
| 4 | connectContext | ConnectRPC 客户端上下文 |
| 5 | useConnect | ConnectRPC 客户端 hook |

---

## 十三、未实现 / Stub

| # | 组件 | 状态 |
|---|------|------|
| 1 | EconomicDataService (后端) | 返回空结果，未接入外部 API（前端已接入，功能空缺非技术债） |
| 2 | Factor Subscriber (后端) | 代码存在但未启动，prerequisites 未满足 |
