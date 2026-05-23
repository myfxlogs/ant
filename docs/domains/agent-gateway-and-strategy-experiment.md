# Agent Gateway 与策略实验能力边界

## 1. 文档目标

本文档定义 AntTrader 后续借鉴 QuantDinger 的 Agent Gateway、通用 Job、MCP 工具面、市场状态识别与策略实验能力时的边界。

本文档不是立即实施清单，而是后续设计、proto、后端、前端和策略服务改造的约束入口。

核心原则：

```text
不绕过 ConnectRPC
不绕过 RiskEngine
不绕过 ExecutionGateway
不让前端承担业务计算
不让 Agent 获得隐式全库或隐式交易权限
```

---

## 2. 可借鉴方向

QuantDinger 中值得 AntTrader 学习的能力包括：

- Agent Gateway：机器身份、能力分级、审计、限流、幂等。
- Job + SSE：长任务提交与进度推送分离。
- MCP Server：面向 Cursor、Claude、Codex 等 AI 客户端的受控工具面。
- Market Regime：用市场状态作为策略生成、回测评分和调度建议上下文。
- Strategy Experiment：参数空间、批量回测、多因子评分与 AI 多轮优化。
- Strategy Sharing：策略或指标资产发布、复用、审核、同步与团队协作。
- Kline Cache：面向回测和外部行情源的 K 线缓存与按周期 TTL。
- Strategy Runtime DSL：策略脚本内嵌运行时、`on_bar` 风格状态机与轻量执行模型。
- Execution Adapter：交易执行适配器、工厂模式和统一订单接口，优先用于整理 MT4/MT5 内部边界。

AntTrader 不应照搬 QuantDinger 的 REST 路由模型。AntTrader 的业务入口仍应以 ConnectRPC 为主，SSE 仅用于浏览器 `EventSource` 必需的长任务事件流场景。

当前明确暂不纳入排期：

- 新增非 MT 市场的实盘接入：例如币圈交易所、美股券商、预测市场等，涉及风控、订单模型、账户模型、成交回报、资金口径和运维面过广；AntTrader 当前仍以 MT4/MT5 订单流为核心。
- 积分、计费、会员体系：属于 SaaS 商业化能力，暂不影响当前交易核心、策略调度与 AI 能力治理。

这些暂缓方向可保留为长期观察项，但不得进入近期实现计划。

需要区分的是，暂缓“新增非 MT 市场实盘接入”不代表暂缓交易适配器治理。QuantDinger 的多交易所实现中，工厂模式、基础客户端、统一下单接口、账户/订单适配边界对 AntTrader 的 MT4/MT5 同样有参考价值。近期应优先服务于：

- 统一 MT4/MT5 执行适配边界。
- 减少 MT4/MT5 分支在业务服务中扩散。
- 将交易前检查、执行、成交回报、错误归一、审计记录固定在一致接口后面。
- 为未来是否接入非 MT 市场预留模型边界，但不提前扩展业务范围。

---

## 3. Agent Gateway 边界

### 3.1 目标

Agent Gateway 面向外部 AI Agent、MCP Server、内部自动化任务提供稳定能力面。

它不是新的业务实现层，而是现有后端服务的受控入口。

```text
Agent / MCP Client
  → Agent Identity + Scope
  → ConnectRPC Agent Service
  → Existing Backend Services
  → RiskEngine / ExecutionGateway / Repositories
```

### 3.2 身份模型

Agent 身份应独立于普通用户 JWT。

建议模型：

| 字段 | 说明 |
|------|------|
| `id` | Agent token 记录 ID |
| `user_id` | 所属用户 |
| `name` | 可读名称 |
| `token_prefix` | 审计展示用前缀 |
| `token_hash` | token 哈希，禁止明文入库 |
| `scopes` | 能力集合 |
| `account_allowlist` | 可访问账户范围 |
| `symbol_allowlist` | 可访问交易品种范围 |
| `paper_only` | 是否仅允许模拟/纸面交易 |
| `rate_limit_per_min` | 单 token 限流 |
| `expires_at` | 过期时间 |
| `status` | `active` / `revoked` / `expired` |
| `last_used_at` | 最近使用时间 |

完整 token 只允许创建时展示一次。

### 3.3 能力分级

建议初始能力分级：

| Scope | 含义 | 默认 |
|-------|------|------|
| `R` | 只读：账户摘要、行情、模板、回测结果、调度健康 | 可启用 |
| `W` | 工作区写：策略草稿、参数配置、非交易型设置 | 显式启用 |
| `B` | 回测与实验：提交回测、参数实验、评分任务 | 显式启用 |
| `AI` | AI 辩论、代码生成、策略优化 | 显式启用 |
| `T_PAPER` | 模拟交易或纸面订单 | 显式启用 |
| `T_LIVE` | 真实交易 | 默认禁止 |
| `C` | 凭证、账户密码、API Key 相关操作 | 默认禁止 |

`T_LIVE` 必须同时满足：

1. token 具备 `T_LIVE`。
2. 服务端 live-agent 开关开启。
3. 账户、symbol、手数、方向在 allowlist 内。
4. RiskEngine 返回 allow。
5. ExecutionGateway 记录完整执行生命周期。

### 3.4 审计

所有 Agent 调用必须写审计日志。

建议字段：

| 字段 | 说明 |
|------|------|
| `user_id` | 所属用户 |
| `agent_token_id` | Agent token ID |
| `agent_name` | Agent 名称 |
| `rpc_service` | ConnectRPC service |
| `rpc_method` | ConnectRPC method |
| `scope` | 需要的 scope |
| `status_code` | 结果状态 |
| `idempotency_key` | 幂等键 |
| `risk_decision` | 如涉及交易或调度，记录 RiskDecision |
| `request_summary` | 脱敏后的请求摘要 |
| `response_summary` | 脱敏后的响应摘要 |
| `duration_ms` | 耗时 |
| `created_at` | 创建时间 |

脱敏字段至少包括：

```text
password
secret
token
api_key
authorization
trading_password
```

---

## 4. ConnectRPC 服务形态

### 4.1 Agent 管理服务

建议服务：

```text
AgentService
  IssueAgentToken
  ListAgentTokens
  RevokeAgentToken
  ListAgentAudit
  GetAgentCapabilities
```

管理类接口仅允许普通登录用户或管理员调用，不允许 Agent 自行提权。

### 4.2 Agent 业务服务

Agent 不应获得独立业务实现。可以采用两种方式：

1. 通过 Agent 拦截器校验 scope 后调用既有 ConnectRPC。
2. 暴露少量稳定的 Agent facade RPC，再委托既有 service。

优先建议第 2 种，用于避免直接暴露内部复杂接口。

示例：

```text
AgentMarketService
  ListAllowedAccounts
  ListAllowedSymbols
  GetQuotes

AgentStrategyService
  ListTemplates
  GetTemplate
  CreateTemplateDraft
  ValidateStrategy

AgentBacktestService
  SubmitBacktestJob
  GetBacktestJob
  SubscribeBacktestJob
```

---

## 5. 通用 Job 模型

### 5.1 目标

长耗时任务必须采用提交与等待分离模型。

适用场景：

- AI 辩论与代码生成。
- 策略校验。
- 回测。
- 批量参数实验。
- 策略评分。
- 报告生成。

### 5.2 状态

建议状态：

```text
queued
running
succeeded
failed
cancelled
expired
```

### 5.3 事件

建议事件：

```text
snapshot
queued
running
progress
chunk
result
failed
cancelled
ping
```

### 5.4 推送通道

优先级：

1. Connect server-stream。
2. SSE，仅用于浏览器 `EventSource` 和边缘网关友好的单向事件推送。
3. 单次 Unary 读取终态作为降级，不允许周期轮询模拟流式。

### 5.5 幂等

写类、回测类、AI 类和交易类 Agent 请求应支持 `idempotency_key`。

幂等范围建议：

```text
user_id + agent_token_id + rpc_method + idempotency_key
```

---

## 6. MCP 工具面

### 6.1 原则

MCP Server 只能是薄封装。

```text
MCP Tool
  → Agent token
  → ConnectRPC Agent facade
  → Backend service
```

禁止：

- MCP 直接访问数据库。
- MCP 直接读取 `.env` 中交易凭证。
- MCP 绕过 RiskEngine 下单。
- MCP 暴露未声明 scope 的隐式工具。

### 6.2 首批工具建议

只开放只读和回测类工具：

```text
whoami
list_accounts
list_symbols
get_quotes
list_templates
get_template
submit_backtest
get_backtest_job
detect_market_regime
get_schedule_health
```

后续再开放受控写能力：

```text
create_template_draft
validate_strategy
submit_strategy_experiment
```

真实交易工具必须最后开放，并默认关闭。

---

## 7. Market Regime 能力

### 7.1 目标

Market Regime 用于为策略生成、回测评分、调度建议提供市场上下文。

建议输出：

| 字段 | 说明 |
|------|------|
| `regime` | 市场状态，如趋势、震荡、高波动、过渡 |
| `confidence` | 置信度 |
| `features` | 特征指标 |
| `segments` | 分段市场状态 |
| `strategy_families` | 推荐策略族 |

### 7.2 初始特征

可从轻量规则开始：

- 价格涨跌幅。
- EMA 快慢线差距。
- ATR 百分比。
- 已实现波动率。
- 方向效率。
- 成交量比例，如数据可得。

### 7.3 使用场景

- AI 生成策略时作为 prompt 上下文。
- 回测评分时计算 regime fit。
- 调度启用前提供提示，不直接替代 RiskEngine。
- 调度健康分析中解释策略表现变化。

---

## 8. 策略实验能力

### 8.1 非 LLM 参数实验

第一阶段应先实现确定性实验：

```text
base template
  → parameter space
  → grid/random candidates
  → batch backtest
  → score
  → ranked result
```

### 8.2 LLM 多轮优化

第二阶段再引入 LLM：

```text
DetectRegime
  → BuildPrompt
  → GenerateCandidates
  → BatchBacktest
  → ScoreCandidates
  → NextRound
  → EarlyStop
  → BestDraft
```

建议参数：

| 参数 | 默认 |
|------|------|
| `max_rounds` | 3 |
| `candidates_per_round` | 5 |
| `early_stop_score` | 82 |

LLM 只生成候选参数或候选草稿，不能直接启用真实交易调度。

### 8.3 评分

策略评分应由后端或策略服务计算。

建议组件：

- 收益分。
- 年化收益分。
- Sharpe 分。
- 盈亏比分。
- 胜率分。
- 回撤分。
- 稳定性分。
- 样本量惩罚。
- regime fit 分。

输出：

```text
overall_score
grade
components
summary
recommendation
```

---

## 9. 策略共享与资产复用

### 9.1 缺口判断

QuantDinger 具备社区/市场能力，核心包括指标发布、审核、购买、本地副本、发布者更新后同步等流程。

AntTrader 当前有策略模板、AI 生成、回测、发布与调度，但缺少等价的策略共享/社区能力。该缺口会影响：

- 团队内复用稳定策略模板。
- 将 AI 生成或人工优化后的策略沉淀为可复用资产。
- 对策略模板进行审核、版本管理、复制和同步。
- 未来接入 Agent 后，让 Agent 在受控资产库中检索和复用策略。

因此，AntTrader 应补充“策略资产库”方向，但不应在近期引入公开市场、付费购买或会员体系。

### 9.2 推荐定位

第一阶段建议定位为内部策略资产库，而不是公开社区。

```text
StrategyTemplate
  → Review / Publish
  → SharedStrategyAsset
  → CloneToWorkspace
  → Version / Sync
  → Schedule / Backtest
```

### 9.3 最小能力

建议最小能力：

| 能力 | 说明 |
|------|------|
| `publish_to_library` | 将策略模板发布到内部资产库 |
| `review_status` | `draft` / `pending` / `approved` / `rejected` |
| `visibility` | `private` / `team` / `public_internal` |
| `source_template_id` | 克隆副本关联原始模板 |
| `source_version` | 记录克隆时的源版本 |
| `clone_count` | 复用次数 |
| `rating_summary` | 内部评分摘要，可后置 |
| `sync_available` | 源模板更新后提示副本可同步 |

### 9.4 版本与同步

策略共享必须避免“发布者修改导致使用者调度行为突变”。

推荐规则：

- 使用者克隆策略时创建独立副本。
- 调度只能引用自己的模板副本，不直接引用共享源模板。
- 源模板更新后只提示可同步，不自动覆盖。
- 同步前必须展示 diff 摘要、风险提示和回测建议。
- 已启用调度的模板同步后必须重新经过后端校验。

### 9.5 与 Agent / MCP 的关系

Agent 可读策略资产库，但写操作必须受 scope 限制。

建议工具：

```text
list_strategy_assets
get_strategy_asset
clone_strategy_asset
submit_asset_review
check_asset_update
sync_strategy_asset
```

其中 `clone`、`submit_asset_review`、`sync` 属于写能力，必须要求 `W` scope；如果同步后触发回测或实验，则还需要 `B` scope。

### 9.6 明确暂不做

- 不做公开策略市场。
- 不做付费购买。
- 不做积分、会员、佣金分成。
- 不做跨租户公开分发。
- 不允许共享资产绕过策略校验、回测或调度启用检查。

---

## 10. K 线缓存与外部数据源

### 10.1 定位

AntTrader 当前以 MT4/MT5 Gateway 行情与订单流为核心。K 线缓存只能作为回测、策略实验和外部数据源接入时的性能优化，不能替代交易实时行情权威。

### 10.2 建议规则

- 按 `account_id + symbol + timeframe + range` 建缓存键。
- 按周期设置 TTL，短周期更短，长周期更长。
- 缓存只服务回测、报告、实验，不直接驱动真实交易下单。
- 缓存命中、过期、数据源、数据时间范围应进入回测 metadata。

### 10.3 数据源工厂

如果未来接入非 MT 数据源，优先建立只读数据源工厂：

```text
MarketDataSource
  → MTGatewaySource
  → ExternalKlineSource
  → CachedSource
```

该方向不得扩展为多交易所实盘适配；交易执行仍以现有 MT4/MT5 核心为准。

---

## 11. 策略运行时 DSL 研究项

QuantDinger 的策略运行时 DSL 对降低进程冷启动有参考价值，但 AntTrader 不应直接替换现有策略服务。

### 11.1 先决条件

实施前必须先完成：

- 调度冷启动耗时剖析。
- 策略服务常驻化可行性评估。
- RestrictedPython 或等价沙箱边界评估。
- 回测与实盘执行语义一致性评估。
- 日志、审计、异常隔离和资源回收设计。

### 11.2 推荐方向

优先研究常驻 worker 或策略服务内 warm pool，而不是把策略脚本嵌入 Go 交易核心。

```text
Schedule Runner
  → Strategy Runtime Pool
  → Sandbox Execute
  → Signal Output
  → RiskEngine
  → ExecutionGateway
```

---

## 12. 交易执行适配器抽象

### 12.1 定位

QuantDinger 的多交易所实现不应被理解为 AntTrader 近期要扩展币圈或美股实盘，而应优先提炼其中的执行适配器思想，用于整理 AntTrader 当前 MT4/MT5 执行边界。

AntTrader 当前交易核心仍是 MT4/MT5，但 MT4 与 MT5 的差异不应持续向业务服务、调度、Agent 或 MCP 工具层扩散。

### 12.2 建议边界

建议将交易执行抽象为：

```text
ExecutionAdapter
  → MT4ExecutionAdapter
  → MT5ExecutionAdapter
```

统一接口至少覆盖：

- 账户交易能力检查。
- symbol 规格读取。
- 下单、平仓、撤单。
- 持仓与订单查询。
- 成交回报归一。
- 错误码与错误原因归一。
- 执行 metadata 与审计字段。

### 12.3 与 RiskEngine / ExecutionGateway 的关系

适配器只负责连接外部交易系统并归一执行语义，不负责做最终业务决策。

```text
TradingService / AutoTradingService / ScheduleRunner
  → RiskEngine
  → ExecutionGateway
  → ExecutionAdapter
  → MT4 / MT5
```

### 12.4 实施约束

- 不新增非 MT 市场实盘适配。
- 不改变现有 MT4/MT5 下单行为。
- 先以行为等价重构为目标。
- 所有交易入口仍必须经过 RiskEngine 与 ExecutionGateway。
- 适配器错误归一后必须保留原始错误信息，便于排障。

---

## 13. 与现有 AntTrader 模块关系

| 新能力 | 依赖现有模块 |
|--------|--------------|
| Agent token | 用户、API key、审计日志、动态配置 |
| Agent scope | Auth interceptor、ConnectRPC transport |
| Agent trading | RiskEngine、ExecutionGateway、TradingService |
| Agent schedule | StrategyScheduleService、RiskEngine、ScheduleHealthService |
| Agent backtest | PythonStrategyService、BacktestRunService |
| Market regime | MarketService、KlineService、strategy-service |
| Strategy experiment | CodeAssistService、PythonStrategyService、ObjectiveScoreService |
| Strategy sharing | StrategyTemplateService、BacktestRunService、审计日志 |
| Kline cache | MarketService、BacktestRunService、strategy-service |
| Runtime DSL | StrategyScheduleRunner、PythonStrategyService、RiskEngine |
| Execution adapter | TradingService、ExecutionGateway、RiskEngine、MT4/MT5 Gateway |
| MCP Server | Agent facade ConnectRPC |

---

## 14. 分阶段路线

### Phase 0：文档与 proto 设计

- 定义 Agent scope。
- 定义 Agent token / audit / job proto。
- 定义 MCP 工具边界。
- 定义不可开放能力清单。
- 定义策略资产库与同步边界。

### Phase 1：只读 Agent Gateway

- Agent token 管理。
- Agent audit。
- 只读 facade：账户、symbol、模板、回测结果、调度健康。

### Phase 2：通用 Job

- 后端 Job 表。
- Job service。
- Connect stream / SSE 事件推送。
- 将新增回测/实验任务接入 Job。

### Phase 3：策略实验

- 参数空间。
- 批量回测。
- 多因子评分。
- 结果排名。

### Phase 4：交易执行适配器治理

- 梳理 MT4/MT5 差异。
- 定义统一 ExecutionAdapter 接口。
- 行为等价迁移 MT4/MT5 执行路径。
- 错误码、成交回报与审计字段归一。

### Phase 5：策略资产库

- 内部发布与审核。
- 克隆到个人工作区。
- 源版本记录与同步提示。
- 启用调度前强制重新校验。

### Phase 6：MCP Server

- 只读与回测工具。
- 本地 stdio。
- 可选 streamable HTTP。

### Phase 7：运行时性能研究

- 调度冷启动剖析。
- 策略服务 warm pool。
- 内嵌 DSL 可行性评估。

### Phase 8：受控交易 Agent

- 仅 paper trading。
- live trading 默认关闭。
- allowlist、限额、RiskEngine、ExecutionGateway、审计全部完成后再开放。

---

## 15. 明确禁止

- 禁止新增业务 REST CRUD 作为 Agent 主入口。
- 禁止 MCP 直接访问数据库。
- 禁止 Agent 绕过 ConnectRPC 调内部 service。
- 禁止 Agent 绕过 RiskEngine 或 ExecutionGateway 下单。
- 禁止前端计算策略评分、风控结论或调度健康。
- 禁止用周期轮询模拟长任务流式输出。
- 禁止默认开放真实交易权限。
- 禁止近期引入非 MT 市场实盘接入。
- 禁止近期引入积分、计费、会员体系。
- 禁止共享策略绕过校验、回测建议和调度启用检查。
- 禁止以新增市场为理由绕过 MT4/MT5 执行适配器治理。

---

## 16. 当前结论

AntTrader 应吸收 QuantDinger 的 Agent 产品化、MCP 工具边界、Job 进度流、市场状态识别、策略实验闭环、策略资产复用、回测数据缓存和交易执行适配器思路，但实现方式必须符合 AntTrader 当前治理：

```text
ConnectRPC 契约优先
后端权威计算
交易核心最高优先级
RiskEngine 强制参与
ExecutionGateway 强制审计
前端只展示和交互
```
