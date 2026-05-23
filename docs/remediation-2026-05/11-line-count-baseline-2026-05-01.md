# 单文件 800 行限制整改基线（2026-05-01）

## 1. 检查范围

本基线只统计项目源码、契约与生成代码范围，不统计 Markdown 文档、二进制产物、日志、测试数据、第三方仿真器数据。

检查范围：

```text
backend/internal
backend/cmd
backend/gen
backend/mt4
backend/mt5
frontend/src
proto
strategy-service/app
scripts
.windsurf
```

规则：

- 排除 `.md`。
- 排除明显二进制文件。
- 行数大于 800 记为超限。

## 2. 当前超限结果

当前基线仍允许的源码/契约/生成代码超限文件：**4 个**。

| 行数 | 文件 |
|---:|---|
| 16363 | `backend/mt5/mt5.pb.go` |
| 11511 | `backend/mt4/mt4.pb.go` |
| 4402 | `backend/mt5/mt5_grpc.pb.go` |
| 2879 | `backend/mt4/mt4_grpc.pb.go` |

## 3. 风险分组

### 3.1 proto 生成代码超限

最主要问题来自 proto 生成代码。

整改方向：

- 拆分过大的 proto。
- 拆分 service。
- 拆分 message。
- 按领域输出生成文件。
- 禁止继续把大量业务塞进单个 proto 或单个生成包。

重点对象已完成：

- `admin.pb.go` / `frontend/src/gen/admin_pb.ts` / `admin.connect.go`
- `analytics.pb.go` / `frontend/src/gen/analytics_pb.ts`
- `auto_trading_risk.pb.go`

已完成拆分并从基线移除：

- `backtest_run.pb.go`
- `strategy_messages.pb.go` / `frontend/src/gen/strategy_messages_pb.ts`
- `ai.pb.go` / `frontend/src/gen/ai_pb.ts`
- `system_ai.pb.go`
- `stream_events.pb.go` / `frontend/src/gen/stream_events_pb.ts`
- `auto_trading.pb.go` / `frontend/src/gen/auto_trading_pb.ts`
- `log.pb.go`
- `account.pb.go`
- `trading.pb.go`
- `market.pb.go`

当前状态：

- `make proto` 已改为本地插件生成，不再依赖 Buf 远端插件。
- 项目自有 proto 与 ConnectRPC 生成物均已低于 800 行。

### 3.2 MT4/MT5 生成代码超限

`backend/mt4` 与 `backend/mt5` 生成文件严重超限。

治理结论：

- `mtproto/mt4.proto` 与 `mtproto/mt5.proto` 属于外部上游协议契约。
- 不拆分、不改写这两个外部 proto。
- 对应生成物保留在 baseline，按外部协议隔离例外管理。
- 项目自有 proto 与 ConnectRPC 生成物不得新增超 800 行例外。

### 3.3 前端页面超限

当前超限页面已完成拆分整改。

整改方向：

```text
Page container
  → hooks
  → table components
  → modal components
  → drawer components
  → api client
  → types
```

### 3.4 前端 i18n 文件超限

当前超限已完成拆分整改，`zh-cn`、`en`、`ja`、`zh-tw` 的 AI 资源均已按 core/debate/wizard/settings/store 拆分。

后续同类整改方向：

```text
ai/common.ts
ai/settings.ts
ai/agents.ts
ai/debate.ts
ai/codeAssist.ts
ai/errors.ts
```

### 3.5 后端业务源码超限

当前超限已完成拆分整改，`strategy_schedule_runner.go` 与 `analytics_service.go` 均已拆分到多个职责文件。

后续同类整改方向：

`strategy_schedule_runner.go` 拆分建议：

```text
strategy_schedule_runner.go
strategy_schedule_runner_loop.go
strategy_schedule_runner_eval.go
strategy_schedule_runner_quote.go
strategy_schedule_runner_runtime.go
strategy_schedule_runner_logging.go
```

`analytics_service.go` 拆分建议：

```text
analytics_service.go
analytics_account.go
analytics_summary.go
analytics_risk.go
analytics_chart.go
```

## 4. 建议整改顺序

第一批：工程检查能力

- 增加行数检查脚本。
- 接入 `make verify`。
- 接入 CI。

第二批：手写代码

- `strategy_schedule_runner.go` 已拆分。
- `analytics_service.go` 已拆分。
- 两个策略页面已拆分。
- AI i18n 文件已拆分。

第三批：proto 体系

- 设计 proto 拆分方案。
- 分批迁移 service 和 message。
- 每批迁移后重新生成并检查行数。

第四批：MT4/MT5 生成代码

- 单独评估来源。
- 确认是否能拆外部 proto。
- 建立可接受的最终生成结构。

## 5. 验收标准

最终必须满足：

```text
find project source files excluding markdown
  → every file line count <= 800
```

任何新增超限文件都不允许进入主分支。
