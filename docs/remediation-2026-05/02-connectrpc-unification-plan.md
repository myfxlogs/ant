# ConnectRPC 统一回归专项方案

## 1. 目标

AntTrader 的浏览器到后端业务通信统一回归 ConnectRPC。

目标状态：

```text
Frontend
  → Connect-Web Client
  → Backend ConnectRPC Handler
  → Internal Service
  → Repository / External Adapter
```

禁止目标状态：

```text
Frontend
  → fetch('/api/xxx')
  → Backend REST Handler
  → Internal Service
```

## 2. 基本原则

### 2.1 所有业务能力先定义 proto

新增、迁移、重构业务接口时，必须先在 `proto/*.proto` 中定义：

- service。
- rpc。
- request message。
- response message。
- enum。
- 分页结构。
- 错误语义。

### 2.2 前端不得直接访问 REST 业务端点

前端只允许：

- 使用 `frontend/src/client/connect.ts` 中的 Connect client。
- 使用业务封装 client。
- 接收后端已计算好的结果。

前端不得：

- 直接 `fetch('/api/...')` 调业务。
- 直接访问 `strategy-service`。
- 自行拼接后端业务 URL。
- 自行解释宽松 JSON 状态。

### 2.3 健康检查例外

以下端点可保留 REST：

- `/health`
- `/health/live`
- `/health/ready`
- `/health/deps`

这些端点不得承载用户业务能力。

## 3. 当前待迁移 REST 清单

根据当前后端注册情况，以下端点应纳入迁移。

| 当前 REST | 目标 Service | 目标 RPC 建议 | 优先级 |
|---|---|---|---|
| `/api/economic-calendar` | `EconomicDataService` | `ListEconomicCalendarEvents` | 已完成 |
| `/api/economic-indicators` | `EconomicDataService` | `ListEconomicIndicators` | 已完成 |
| `/api/strategy/indicator-catalog` | `IndicatorCatalogService` | `GetIndicatorCatalog` | 已完成 |
| `/api/debate/v2/start` | `DebateV2Service` | `StartDebateV2` | 已完成 |
| `/api/debate/v2/chat` | `DebateV2Service` | `ChatDebateV2` | 已完成 |
| `/api/debate/v2/advance` | `DebateV2Service` | `AdvanceDebateV2` | 已完成 |
| `/api/debate/v2/back` | `DebateV2Service` | `BackDebateV2` | 已完成 |
| `/api/debate/v2/params` | `DebateV2Service` | `SetDebateV2Params` | 已完成 |
| `/api/debate/v2/code/reject` | `DebateV2Service` | `RejectDebateV2Code` | 已完成 |
| `/api/debate/v2/sessions` | `DebateV2Service` | `ListDebateV2Sessions` | 已完成 |
| `/api/debate/v2/sessions/{id}` | `DebateV2Service` | `GetDebateV2Session` / `DeleteDebateV2Session` | 已完成 |
| `/api/ai/code/revise` | `AIService` | `ReviseStrategyCode` | 高 |
| `/api/ai/code/explain` | `AIService` | `ExplainStrategyCode` | 高 |
| `/api/ai/primary` | `SystemAIService` 或 `AIService` | `GetPrimaryModel` / `SetPrimaryModel` | 高 |
| `/api/strategy/validate-extended` | `StrategyService` 或 `PythonStrategyService` | `ValidateStrategyExtended` | 高 |
| `/api/backtest-runs/{id}/trades` | `PythonStrategyService` 或 `BacktestService` | `ListBacktestRunTrades` | 高 |

## 4. 迁移步骤

每个 REST 端点按以下步骤迁移：

```text
1. 查找前端调用点
2. 查找后端 REST handler
3. 确认内部 service 方法
4. 设计 proto request/response
5. 实现 internal/connect 方法
6. 前端新增 Connect client 封装
7. 替换页面调用
8. 删除旧 fetch 调用
9. 删除 REST handler 注册
10. 删除 REST handler 实现
11. 跑生成、编译、测试、容器重启
```

## 5. proto 拆分规则

为了满足单文件 800 行限制，proto 必须按领域拆分。

建议：

```text
proto/auth.proto
proto/account.proto
proto/trading.proto
proto/market.proto
proto/stream.proto
proto/strategy_template.proto
proto/strategy_schedule.proto
proto/backtest.proto
proto/ai_chat.proto
proto/ai_debate.proto
proto/system_ai.proto
proto/admin_user.proto
proto/admin_system.proto
proto/log.proto
proto/common.proto
proto/pagination.proto
proto/risk.proto
```

注意：

- 不要把所有 AI 能力继续塞进一个巨大的 `ai.proto`。
- 不要把策略模板、调度、回测全部塞进一个 `strategy.proto`。
- 如果生成文件超过 800 行，继续拆 proto。

## 6. 生成代码行数控制

必须增加检查：

```text
find backend frontend proto strategy-service scripts -type f
  → 排除 *.md
  → 统计行数
  → 任一文件 > 800 失败
```

检查必须覆盖：

- `backend/gen/proto/**`
- `frontend/src/gen/**`
- `proto/**`

如果生成代码超限，不允许提交。

## 7. 过渡期策略

迁移期允许短时间保留 REST，但必须满足：

- 有迁移表记录。
- 有目标 proto。
- 有删除计划。
- 前端新代码不得继续调用旧 REST。
- 同一能力不能长期同时维护 REST 和 ConnectRPC 两套入口。

## 8. 完成标准

ConnectRPC 回归完成时应满足：

- 前端无业务 `fetch('/api/...')`。
- 后端 `server.go` 不注册业务 `/api/...`。
- 业务能力全部在 proto 中可见。
- ConnectRPC 反射服务列表完整。
- 生成代码全部不超过 800 行。
- 文档更新为 ConnectRPC-only 业务通信。
