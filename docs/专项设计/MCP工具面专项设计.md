# MCP 工具面专项设计

本文档定义 AntTrader 借鉴 QuantDinger MCP Server 能力时，面向 Cursor、Claude、Codex 等 AI 客户端的受控工具边界。

## 1. 目标与非目标

目标：

- 为 AI 客户端提供受控、可审计、可限流的工具面。
- MCP Server 作为 Agent facade 的薄封装。
- 首批只开放只读和回测类工具。
- 所有能力通过 ConnectRPC Agent facade 进入后端。

非目标：

- MCP 不直接访问数据库。
- MCP 不读取 `.env` 中交易凭证。
- MCP 不绕过 RiskEngine 或 ExecutionGateway。
- MCP 不暴露未声明 scope 的隐式工具。
- 不在第一阶段开放真实交易工具。

## 2. 现有依赖模块

- Agent Gateway：token、scope、allowlist、审计与限流。
- ConnectRPC Agent facade：MCP 的唯一业务入口。
- BacktestRunService、StrategyTemplateService、MarketService：首批只读工具委托目标。
- RiskEngine 与 ExecutionGateway：未来交易工具的强制依赖。

## 3. QuantDinger 可借鉴点

- 面向 AI 客户端暴露 MCP 工具。
- 工具按能力分组。
- 工具请求进入后端受控能力面，而不是直接操作底层资源。

AntTrader 必须用 ConnectRPC facade 重设接口，不照搬 QuantDinger REST 形态。

## 4. AntTrader 实施边界

MCP Server 只能是薄封装：

```text
MCP Tool
  -> Agent token
  -> ConnectRPC Agent facade
  -> Backend service
```

MCP 不持有用户交易凭证，不直接连接数据库，不承担业务计算。

## 5. 首批工具清单

只读工具：

```text
whoami
list_accounts
list_symbols
get_quotes
list_templates
get_template
get_schedule_health
```

回测与实验相关工具：

```text
submit_backtest
get_backtest_job
detect_market_regime
```

后续写类工具：

```text
create_template_draft
validate_strategy
submit_strategy_experiment
```

策略资产库工具：

```text
list_strategy_assets
get_strategy_asset
clone_strategy_asset
check_asset_update
```

真实交易工具必须最后开放，并默认关闭。

## 6. 工具到 scope 映射

| 工具 | Scope | 阶段 |
|------|-------|------|
| `whoami` | `R` | 第一阶段 |
| `list_accounts` | `R` | 第一阶段 |
| `list_symbols` | `R` | 第一阶段 |
| `get_quotes` | `R` | 第一阶段 |
| `list_templates` | `R` | 第一阶段 |
| `get_template` | `R` | 第一阶段 |
| `submit_backtest` | `B` | 第一阶段后 |
| `get_backtest_job` | `B` | 第一阶段后 |
| `detect_market_regime` | `B` | 第二阶段 |
| `create_template_draft` | `W` | 后续 |
| `validate_strategy` | `W` | 后续 |
| `submit_strategy_experiment` | `B` | 后续 |
| `clone_strategy_asset` | `W` | 后续 |

如果工具组合触发 AI 生成，还必须具备 `AI` scope。

## 7. ConnectRPC facade 映射草案

```text
MCP whoami
  -> AgentService.GetAgentCapabilities

MCP list_accounts
  -> AgentMarketService.ListAllowedAccounts

MCP list_symbols
  -> AgentMarketService.ListAllowedSymbols

MCP get_template
  -> AgentStrategyService.GetTemplate

MCP submit_backtest
  -> AgentBacktestService.SubmitBacktestJob

MCP get_backtest_job
  -> AgentBacktestService.GetBacktestJob
```

MCP 工具不得直接调用内部 repository 或非 facade service。

## 8. 安全、审计与脱敏

每次工具调用必须记录 Agent audit：

- Agent token ID 与 token 前缀。
- 工具名称与映射 RPC。
- scope 校验结果。
- 请求摘要和响应摘要。
- 耗时、状态码和错误码。
- idempotency key。

请求和响应摘要必须脱敏，至少覆盖 `password`、`secret`、`token`、`api_key`、`authorization`、`trading_password`。

## 9. 部署形态

第一阶段优先本地 stdio：

```text
AI Client
  -> local MCP stdio
  -> Backend ConnectRPC
```

后续可评估 streamable HTTP，但必须保留 Agent token、TLS、限流和审计要求。

## 10. 错误处理

MCP 工具错误应返回稳定错误结构：

| 字段 | 说明 |
|------|------|
| `code` | 稳定错误码 |
| `message` | 可展示错误 |
| `retryable` | 是否可重试 |
| `trace_id` | 追踪 ID |

不得把后端敏感堆栈、凭证、原始 token 暴露给 AI 客户端。

## 11. 分阶段实施

1. Agent Gateway token、scope、audit 完成后再启用 MCP。
2. 建立本地 stdio MCP Server。
3. 接入只读工具。
4. 接入回测工具与 Job 查询。
5. 接入策略资产库只读工具。
6. 写类工具在审计与幂等完善后开放。
7. 真实交易工具最后评估，默认关闭。

## 12. 验收标准与禁止事项

验收标准：

- MCP 工具调用都能追溯到 Agent audit。
- scope 不足时工具被拒绝。
- MCP 无数据库连接配置和交易凭证读取能力。
- 工具清单与后端 facade 能力一致。

禁止事项：

- 禁止 MCP 直接访问数据库。
- 禁止 MCP 直接读取 `.env` 交易凭证。
- 禁止 MCP 绕过 ConnectRPC 调内部 service。
- 禁止 MCP 绕过 RiskEngine 或 ExecutionGateway 下单。
- 禁止暴露未声明 scope 的隐式工具。
