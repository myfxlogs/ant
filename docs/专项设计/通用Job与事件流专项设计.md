# 通用 Job 与事件流专项设计

本文档定义 AntTrader 借鉴 QuantDinger Job + SSE 能力时的长任务提交、状态机、事件流、幂等与降级读取边界。

## 1. 目标与非目标

目标：

- 将 AI、回测、策略实验、报告生成等长耗时任务统一为提交与等待分离模型。
- 用后端 Job 记录任务状态、进度、结果摘要和错误原因。
- 优先使用 Connect server-stream，必要时提供 SSE 给浏览器 `EventSource`。
- 避免前端周期轮询模拟长任务流式输出。

非目标：

- 不把 Job 设计成绕过业务服务的通用执行器。
- 不用 REST CRUD 承载新增业务任务。
- 不允许前端自行推导终态或成功态。
- 不把 crontab 或定时轮询作为默认任务推进方式。

## 2. 现有依赖模块

- BacktestRunService：回测任务状态与结果。
- CodeAssistService：AI 生成、校验、辩论等长任务。
- PythonStrategyService：策略校验、回测与评分计算。
- ScheduleHealthService：调度健康分析。
- ConnectRPC transport：Unary 提交、server-stream 订阅。
- SSE 网关：仅用于浏览器 `EventSource` 必需场景。

## 3. QuantDinger 可借鉴点

- 长任务提交与进度推送分离。
- 任务状态、日志片段、阶段进度和最终结果统一建模。
- 前端只订阅事件与展示，不阻塞等待长请求。

## 4. AntTrader 实施边界

通用 Job 是任务状态与事件承载层，不是业务计算层：

```text
Client
  -> SubmitXxxJob ConnectRPC
  -> Backend Job Store
  -> Domain Service / Strategy Service
  -> Job Event Stream
```

任务实际计算仍由后端或策略服务完成，前端只展示后端返回状态。

## 5. Job 数据模型草案

| 字段 | 说明 |
|------|------|
| `id` | Job ID |
| `user_id` | 所属用户 |
| `kind` | 任务类型，如 `backtest`、`strategy_experiment`、`ai_generation` |
| `status` | 状态机状态 |
| `progress` | 0 到 100 的进度，允许为空 |
| `stage` | 当前阶段 |
| `request_summary` | 脱敏请求摘要 |
| `result_ref` | 结果对象引用，如 backtest_run_id |
| `result_summary` | 脱敏结果摘要 |
| `error_code` | 稳定错误码 |
| `error_message` | 可展示错误说明 |
| `idempotency_key` | 幂等键 |
| `created_at` | 创建时间 |
| `started_at` | 开始时间 |
| `finished_at` | 结束时间 |
| `expires_at` | 事件保留截止时间 |

## 6. 状态机

建议状态：

```text
queued
running
succeeded
failed
cancelled
expired
```

状态约束：

- `queued` 只能进入 `running`、`cancelled` 或 `expired`。
- `running` 只能进入 `succeeded`、`failed`、`cancelled` 或 `expired`。
- 终态不可回退。
- 终态判断以后端字段为准，前端不得自行根据字符串推导。

## 7. 事件模型

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

事件字段建议：

| 字段 | 说明 |
|------|------|
| `job_id` | Job ID |
| `seq` | 单 Job 内递增序号 |
| `type` | 事件类型 |
| `status` | 当前状态 |
| `progress` | 当前进度 |
| `stage` | 当前阶段 |
| `message` | 展示文案 |
| `payload` | 类型化 payload 或 JSON 摘要 |
| `created_at` | 事件时间 |

## 8. ConnectRPC / SSE 接口草案

通用 Job 服务建议：

```text
JobService
  GetJob
  CancelJob
  SubscribeJob
```

业务服务保留明确提交入口：

```text
BacktestService
  SubmitBacktestJob

StrategyExperimentService
  SubmitStrategyExperiment

CodeAssistService
  SubmitGenerationJob
```

推送优先级：

1. Connect server-stream。
2. SSE，仅用于浏览器 `EventSource` 与边缘网关友好的单向事件推送。
3. Unary `GetJob` 读取终态作为降级，不允许周期轮询模拟流式。

## 9. 幂等与权限

写类、回测类、AI 类和交易类 Agent 请求必须支持 `idempotency_key`。

幂等范围建议：

```text
user_id + agent_token_id + rpc_method + idempotency_key
```

普通用户请求可使用：

```text
user_id + rpc_method + idempotency_key
```

订阅 Job 时必须校验：

- Job 属于当前用户，或调用方具备管理员权限。
- Agent token 具备对应 scope。
- Job 事件摘要已脱敏。

## 10. 分阶段实施

1. 建立 Job proto 与通用状态模型。
2. 为新增策略实验接入 Job，而不是改造所有存量任务。
3. 将 AI、回测、报告生成逐步迁入 Job 模型。
4. 提供 Connect stream 订阅。
5. 仅在浏览器需要时补充 SSE bridge。

## 11. 验收标准与禁止事项

验收标准：

- 长任务提交立即返回 Job ID。
- Job 终态由后端持久化记录。
- 事件序号单调递增，断线后可读取 snapshot。
- 错误码与错误信息可用于前端展示和排障。

禁止事项：

- 禁止用周期轮询模拟长任务流式输出。
- 禁止把 Job 当成通用远程代码执行器。
- 禁止前端自行计算任务成功、评分或风控结论。
- 禁止新增业务 REST API 作为 Job 主入口。
