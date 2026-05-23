# REST / fetch 调用清点基线（2026-05-01）

## 1. 文档目标

本文档记录当前浏览器到后端、浏览器到策略服务的 REST 调用现状，作为后续迁移到 ConnectRPC 的执行基线。

目标状态：

```text
浏览器
  → Connect-Web
  → 后端 ConnectRPC
  → 后端 service / repository / strategy runtime client
```

禁止目标状态：

```text
浏览器
  → /api/... REST 业务接口
```

更禁止：

```text
浏览器
  → strategy-service /api/... 直接调用
```

## 2. 清点范围

本次清点范围：

- 后端 `backend/internal/server` 中的 `mux.HandleFunc`。
- 前端 `frontend/src` 中的 `apiFetch(...)`。
- 前端硬编码 `/api/...`。
- 前端直接 `fetch(...)`。
- 前端 `axios` 使用。

## 3. 总体结论

当前发现：

- 后端注册了 **17 个** REST handler，其中 `/health` 为非业务健康检查，其余 **16 个** 属于业务或业务数据接口。
- 前端业务 REST 调用集中在 `frontend/src/client` 下。
- 前端没有直接散落大量原生 `fetch(...)`，实际由 `client/transport.ts` 中的 `apiFetch` 统一封装。
- 前端原存在一个高风险调用：`frontend/src/client/strategyService.ts` 直接请求策略服务 `/api/objective-score`；现已迁移到后端 `ObjectiveScoreService.CalculateObjectiveScore`。
- `frontend/src/utils/error.ts` 只引用 `AxiosError` 类型，未发现实际 axios 请求。

## 4. 后端 REST handler 清单

来源：`backend/internal/server/server.go`

| 路径 | Handler | 类型 | 迁移优先级 | 目标处理 |
|---|---|---|---|---|
| `/health` | inline handler | 健康检查 | 保留 | 可保留为 REST，后续扩展 `/health/live`、`/health/ready`、`/health/deps` |
| `/api/economic-calendar` | 已删除 | 业务数据/外部数据代理 | 中 | 已迁移到 `EconomicDataService.ListEconomicCalendarEvents` |
| `/api/economic-indicators` | 已删除 | 业务数据/外部数据代理 | 中 | 已迁移到 `EconomicDataService.ListEconomicIndicators` |
| `/api/strategy/indicator-catalog` | 已删除 | 策略业务数据 | 高 | 已迁移到 `IndicatorCatalogService.GetIndicatorCatalog` |
| `/api/debate/v2/start` | 已删除 | AI Debate 业务 | 高 | 已迁移到 `DebateV2Service.StartDebateV2` |
| `/api/debate/v2/chat` | 已删除 | AI Debate 业务 | 高 | 已迁移到 `DebateV2Service.ChatDebateV2` |
| `/api/debate/v2/advance` | 已删除 | AI Debate 业务 | 高 | 已迁移到 `DebateV2Service.AdvanceDebateV2` |
| `/api/debate/v2/back` | 已删除 | AI Debate 业务 | 高 | 已迁移到 `DebateV2Service.BackDebateV2` |
| `/api/debate/v2/params` | 已删除 | AI Debate 业务 | 高 | 已迁移到 `DebateV2Service.SetDebateV2Params` |
| `/api/debate/v2/code/reject` | 已删除 | AI Debate 业务 | 高 | 已迁移到 `DebateV2Service.RejectDebateV2Code` |
| `/api/debate/v2/sessions` | 已删除 | AI Debate 业务 | 高 | 已迁移到 `DebateV2Service.ListDebateV2Sessions` |
| `/api/debate/v2/sessions/` | 已删除 | AI Debate 业务 | 高 | 已迁移到 `DebateV2Service.GetDebateV2Session` / `DeleteDebateV2Session` |
| `/api/ai/code/revise` | 已删除 | AI 代码辅助 | 高 | 已迁移到 `CodeAssistService.ReviseCode` |
| `/api/ai/code/explain` | 已删除 | AI 代码辅助 | 高 | 已迁移到 `CodeAssistService.ExplainCode` |
| `/api/ai/primary` | 已删除 | AI 配置业务 | 高 | 已迁移到 `AIPrimaryService.GetAIPrimary` / `SetAIPrimary` |
| `/api/strategy/validate-extended` | 已删除 | 策略校验业务 | 高 | 已迁移到 `CodeAssistService.ValidateStrategyExtended` |
| `/api/backtest-runs/` | 已删除 | 回测结果查询 | 高 | 已迁移到 `BacktestTradesService.ListBacktestRunTrades` |

## 5. 后端 handler 文件分布

| 文件 | REST handler |
|---|---|
| `backend/internal/server/server.go` | `/health` |
| `backend/internal/server/debate_v2_handler.go` | 已删除，迁移到 `DebateV2Service` |
| `backend/internal/server/ai_code_handler.go` | 已删除，迁移到 `CodeAssistService` |
| `backend/internal/server/ai_primary_handler.go` | 已删除，迁移到 `AIPrimaryService` |
| `backend/internal/server/backtest_run_trades_handler.go` | 已删除，迁移到 `BacktestTradesService` |

## 6. 前端 REST 调用清单

### 6.1 `frontend/src/client/debateV2.ts`

| 前端方法 | 当前 REST | 目标 RPC | 优先级 |
|---|---|---|---|
| `start` | 已迁移 | `DebateV2Service.StartDebateV2` | 高 |
| `chat` | 已迁移 | `DebateV2Service.ChatDebateV2` | 高 |
| `advance` | 已迁移 | `DebateV2Service.AdvanceDebateV2` | 高 |
| `back` | 已迁移 | `DebateV2Service.BackDebateV2` | 高 |
| `listSessions` | 已迁移 | `DebateV2Service.ListDebateV2Sessions` | 高 |
| `getSession` | 已迁移 | `DebateV2Service.GetDebateV2Session` | 高 |
| `deleteSession` | 已迁移 | `DebateV2Service.DeleteDebateV2Session` | 高 |
| `rejectCode` | 已迁移 | `DebateV2Service.RejectDebateV2Code` | 高 |

### 6.2 `frontend/src/client/codeAssist.ts`

| 前端方法 | 当前 REST | 目标 RPC | 优先级 |
|---|---|---|---|
| `revise` | 已迁移 | `CodeAssistService.ReviseCode` | 高 |
| `explain` | 已迁移 | `CodeAssistService.ExplainCode` | 高 |
| `validateExtended` | 已迁移 | `CodeAssistService.ValidateStrategyExtended` | 高 |

### 6.3 `frontend/src/client/ai.ts`

| 前端方法 | 当前 REST | 目标 RPC | 优先级 |
|---|---|---|---|
| `getPrimary` | 已迁移 | `AIPrimaryService.GetAIPrimary` | 高 |
| `setPrimary` | 已迁移 | `AIPrimaryService.SetAIPrimary` | 高 |

### 6.4 `frontend/src/client/backtestRuns.ts`

| 前端方法 | 当前 REST | 目标 RPC | 优先级 |
|---|---|---|---|
| `getTrades` | 已迁移 | `BacktestTradesService.ListBacktestRunTrades` | 高 |

### 6.5 `frontend/src/client/analytics.ts`

| 前端方法 | 当前 REST | 目标 RPC | 优先级 |
|---|---|---|---|
| `getEconomicCalendar` | 已迁移 | `EconomicDataService.ListEconomicCalendarEvents` | 中 |
| `getEconomicIndicators` | 已迁移 | `EconomicDataService.ListEconomicIndicators` | 中 |

### 6.6 `frontend/src/client/strategyService.ts`

| 前端方法 | 当前调用 | 问题 | 目标 RPC | 优先级 |
|---|---|---|---|---|
| `postObjectiveScore` | 已迁移为 `ObjectiveScoreService.CalculateObjectiveScore` | 原浏览器直连策略服务 | 已完成 | 最高 |

## 7. 前端直接 fetch / axios 结论

### 7.1 原生 `fetch`

当前只发现：

| 文件 | 说明 |
|---|---|
| `frontend/src/client/transport.ts` | `apiFetch` 内部封装使用 `fetch` |

没有发现大量页面直接使用原生 `fetch`。

### 7.2 axios

当前只发现：

| 文件 | 说明 |
|---|---|
| `frontend/src/utils/error.ts` | 只引用 `AxiosError` 类型用于错误解析 |

未发现实际 axios 请求。

## 8. 迁移优先级

### 8.1 第一批：最高优先级

先处理浏览器直连策略服务：

1. `frontend/src/client/strategyService.ts` → `/api/objective-score`

原因：

- 直接绕过后端。
- 不符合浏览器只访问后端 ConnectRPC 的原则。
- 策略服务不应直接暴露给浏览器。

建议目标：

```text
Frontend
  → ObjectiveScoreService.CalculateObjectiveScore ConnectRPC
  → Go backend PythonStrategyService client
  → strategy-service /api/objective-score internal call
```

当前状态：已完成。

### 8.2 第二批：AI / Debate

迁移：

- `/api/debate/v2/*`：已完成
- `/api/ai/code/revise`：已完成
- `/api/ai/code/explain`：已完成
- `/api/ai/primary`：已完成

原因：

- 数量最多。
- 属于明确业务接口。
- 与当前 AI 体系重构强相关。

### 8.3 第三批：策略 / 回测

迁移：

- `/api/strategy/validate-extended`：已完成
- `/api/backtest-runs/{id}/trades`：已完成
- `/api/strategy/indicator-catalog`：已完成

原因：

- 与策略生命周期重构直接相关。
- 发布、回测、调度都依赖这些接口。

### 8.4 第四批：宏观数据

迁移或明确例外：

- `/api/economic-calendar`：已完成
- `/api/economic-indicators`：已完成

建议：

- 如果前端业务页面使用，应迁移到 ConnectRPC。
- 如果仅作为外部数据代理，也应优先由后端 ConnectRPC 包装，再由后端内部访问第三方。

## 9. ConnectRPC 迁移步骤模板

每个接口迁移按以下步骤执行：

```text
1. 定义或扩展 proto service / rpc
2. 生成 Go / TS 代码
3. 实现 backend/internal/connect 方法
4. 复用现有 internal/service 或 handler 内部逻辑
5. 前端 client 改为 Connect client
6. 页面调用保持语义不变
7. 删除对应 apiFetch 调用
8. 删除后端 REST handler 注册
9. 删除后端 REST handler 实现
10. 运行 make verify
11. 重建并重启容器
12. 更新迁移清单
```

## 10. 注意事项

迁移时必须遵守：

- 不把 REST handler 逻辑简单复制成更大的 Connect handler。
- 优先把业务逻辑下沉到 service，Connect handler 只做适配。
- proto 拆分必须考虑 800 行限制。
- 生成代码也必须不超过 800 行；如超限，继续拆 proto。
- 前端只渲染后端返回结果，不在迁移中新增前端计算。
- 每批迁移必须保证功能可回归验证。

## 11. 当前完成标准

本文档完成后，下一步进入：

```text
第二批迁移：AI / Debate 相关 REST
```

第一批已完成，浏览器不再直连策略服务，策略服务重新回到后端内部依赖位置。
