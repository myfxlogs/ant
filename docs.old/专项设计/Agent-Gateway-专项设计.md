# Agent Gateway 专项设计

本文档定义 AntTrader 借鉴 QuantDinger Agent Gateway 能力时的身份、权限、审计、幂等与 ConnectRPC facade 边界。

## 1. 目标与非目标

目标：

- 建立独立于普通用户 JWT 的 Agent token 身份。
- 用 scope、账户白名单、品种白名单和服务端开关限制 Agent 能力。
- 为 MCP、外部 AI Agent 和内部自动化任务提供受控入口。
- 所有 Agent 调用可审计、可限流、可追踪、可撤销。

非目标：

- 不新增业务 REST API 作为 Agent 主入口。
- 不允许 Agent 直接访问数据库或内部 repository。
- 不允许 Agent 绕过 RiskEngine 或 ExecutionGateway 触发交易。
- 不在第一阶段开放真实交易能力。
- 当前业务范围内暂不实现第三方 Agent 或外部工具提交真实交易指令；该能力仅作为后期评估方向记录。

## 2. 现有依赖模块

- 用户与认证模块：绑定 Agent token 所属用户。
- ConnectRPC transport 与 auth interceptor：承载 Agent 调用入口。
- 审计日志与操作日志：记录 Agent 请求生命周期。
- RiskEngine：交易、调度与高风险操作的统一风控入口。
- ExecutionGateway：交易执行生命周期与幂等记录入口。
- StrategyTemplateService、BacktestRunService、ScheduleHealthService：首批 facade 委托目标。

## 3. QuantDinger 可借鉴点

- Agent Gateway 的机器身份与能力分级。
- token scope 与工具级权限控制。
- 请求审计、限流与幂等键。
- 对 AI 客户端暴露稳定 facade，而不是暴露内部复杂接口。

## 4. AntTrader 实施边界

Agent Gateway 只做受控入口，不做新的业务实现层：

```text
Agent / MCP Client
  -> Agent token + scope
  -> ConnectRPC Agent facade
  -> Existing Backend Services
  -> RiskEngine / ExecutionGateway / Repositories
```

第一阶段只允许只读与回测类能力；写能力、AI 生成与交易能力必须分阶段启用。

## 5. 数据模型草案

Agent token 记录建议字段：

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
| `paper_only` | 是否仅允许模拟或纸面交易 |
| `rate_limit_per_min` | 单 token 每分钟限流 |
| `expires_at` | 过期时间 |
| `status` | `active` / `revoked` / `expired` |
| `last_used_at` | 最近使用时间 |
| `created_at` | 创建时间 |
| `updated_at` | 更新时间 |

完整 token 只允许创建时展示一次，数据库只保存哈希与前缀。

## 6. Scope 草案

| Scope | 含义 | 默认 |
|-------|------|------|
| `R` | 只读：账户摘要、行情、模板、回测结果、调度健康 | 可启用 |
| `W` | 工作区写：策略草稿、参数配置、非交易型设置 | 显式启用 |
| `B` | 回测与实验：提交回测、参数实验、评分任务 | 显式启用 |
| `AI` | AI 辩论、代码生成、策略优化 | 显式启用 |
| `T_PAPER` | 模拟交易或纸面订单 | 显式启用 |
| `T_LIVE` | 真实交易 | 默认禁止 |
| `C` | 凭证、账户密码、API Key 相关操作 | 默认禁止 |

`T_LIVE` 必须同时满足 token scope、服务端 live-agent 开关、allowlist、RiskEngine allow 与 ExecutionGateway 完整记录。

## 7. ConnectRPC 接口草案

管理服务仅允许普通登录用户或管理员调用：

```text
AgentService
  IssueAgentToken
  ListAgentTokens
  RevokeAgentToken
  ListAgentAudit
  GetAgentCapabilities
```

Agent facade 首批建议：

```text
AgentMarketService
  ListAllowedAccounts
  ListAllowedSymbols
  GetQuotes

AgentStrategyService
  ListTemplates
  GetTemplate

AgentBacktestService
  ListBacktestRuns
  GetBacktestRun
```

写类 facade 后置：

```text
AgentStrategyService
  CreateTemplateDraft
  ValidateStrategy

AgentExperimentService
  SubmitStrategyExperiment
  GetStrategyExperiment
```

## 8. 审计、幂等与脱敏

所有 Agent 调用必须写审计日志。建议字段：

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
| `risk_decision` | 涉及交易或调度时记录 RiskDecision |
| `request_summary` | 脱敏请求摘要 |
| `response_summary` | 脱敏响应摘要 |
| `duration_ms` | 耗时 |
| `created_at` | 创建时间 |

脱敏字段至少包括 `password`、`secret`、`token`、`api_key`、`authorization`、`trading_password`。

写类、回测类、AI 类和交易类 Agent 请求应支持 `idempotency_key`，幂等范围为：

```text
user_id + agent_token_id + rpc_method + idempotency_key
```

## 9. 分阶段实施

1. 建立 Agent token / audit proto 与数据库模型。
2. 实现 Agent 管理服务与审计写入。
3. 实现只读 facade：账户、symbol、模板、回测、调度健康。
4. 接入 MCP 只读工具。
5. 后续开放写类与实验类能力。
6. 最后评估 paper trading；live trading 默认关闭。

## 10. 验收标准与禁止事项

验收标准：

- token 创建时只展示一次明文。
- 已撤销、过期或 scope 不足的 token 被拒绝。
- 每次 Agent 调用都有审计记录。
- facade 不直接访问前端或绕过既有后端服务。

禁止事项：

- 禁止新增业务 REST CRUD 作为 Agent 主入口。
- 禁止 Agent 直接访问数据库。
- 禁止 Agent 自行提权或管理自身 scope。
- 禁止默认开放真实交易权限。

## 11. 第三方 Agent 交易能力的后期评估方向

Agent Gateway 的长期方向可以支持第三方 Agent、MCP 工具或外部自动化系统调用 AntTrader 后端能力。该能力的本质不是开放后门，而是提供受控入口：

```text
Third-party Agent / External Tool
  -> Agent token
  -> scope 权限
  -> account_allowlist / symbol_allowlist
  -> ConnectRPC Agent Gateway
  -> AntTrader backend services
  -> RiskEngine
  -> ExecutionGateway
  -> Broker / MT4 / MT5
```

该方向可支持的能力分层如下：

| 能力层级 | 示例 | 建议状态 |
|----------|------|----------|
| 只读能力 | 账户摘要、行情、K 线、模板、回测结果、市场状态 | 可作为早期能力 |
| 分析能力 | 行情上下文、策略草稿、参数建议、实验计划 | 可作为中期能力 |
| 任务能力 | 提交回测、提交实验、查询 Job 进度 | 可作为中期能力 |
| 调度草案 | 生成调度草案、上线前检查建议 | 后期谨慎开放 |
| 模拟交易 | paper order、模拟执行 | 后期评估 |
| 真实交易 | 实盘订单意图 | 当前业务范围内暂不实现 |

如果未来评估真实交易能力，外部 Agent 不应直接提交底层订单，而应提交交易意图：

```text
OrderIntent
  account_id
  symbol
  side
  order_type
  volume
  price
  stop_loss
  take_profit
  reason
  strategy_ref
  confidence
  idempotency_key
```

后端必须将交易意图转入风控与执行网关：

```text
OrderIntent
  -> Auth / Scope / Allowlist
  -> RiskEngine.CheckOrderIntent
  -> ExecutionGateway.SubmitOrder
  -> Audit / Order Result
```

真实交易能力若未来启用，必须同时满足：

- Agent token 具备 `T_LIVE` scope。
- 服务端 live-agent 开关显式开启。
- 账户在 `account_allowlist` 内。
- 品种在 `symbol_allowlist` 内。
- 请求携带 `idempotency_key`。
- 通过 RiskEngine。
- 通过 ExecutionGateway。
- 全量写入审计日志。
- 用户可随时撤销 token。
- 支持最大手数、最大日亏损、最大频率、交易时段和熔断限制。

禁止事项：

- 禁止第三方 Agent 直接访问数据库。
- 禁止第三方 Agent 直接访问 MT4 / MT5 连接。
- 禁止第三方 Agent 读取账户密码或交易凭证。
- 禁止第三方 Agent 调用内部 repository。
- 禁止绕过 RiskEngine。
- 禁止绕过 ExecutionGateway。
- 禁止绕过审计日志。
- 禁止绕过用户授权。
- 禁止默认拥有实盘交易权限。

当前决策：

```text
第三方 Agent / 外部工具提交真实交易指令不纳入当前业务范围。
当前阶段只保留只读、分析、策略草稿、实验和 Job 查询等受控能力设计。
真实交易能力仅作为后期产品、安全、风控和合规共同评估的候选方向。
```
