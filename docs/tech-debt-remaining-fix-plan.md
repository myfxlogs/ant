# 剩余技术债修复方案

**日期**: 2026-06-06  
**状态**: 待审核  
**前提**: 第一轮（commit `61286be`）已修7项管线 + 第二轮（commit `abd5697`）已清空 REST router 残留

---

## 一、问题清单总览

| 组 | 类型 | 数量 | 严重度 | 说明 |
|----|------|------|--------|------|
| A | Go 文件 > 300 行 | 15 | 高 | 违反非协商规则，CI 拦截 |
| B | TS 文件 > 250 行 | 14 | 高 | 同上 |
| C | Go 函数 > 50 行 | 25+ | 中 | 违反非协商规则 |
| D | SRE `/api/*` 端点 | 7 | 低 | **接受现状**：内部运维端点，kill switch 需 curl 可达 |
| E | `.env` 密钥 | 1 | 低 | 安全面，未入 git |

---

## 二、A 组：Go 文件拆分（15 个 → 目标 ≤ 300 行）

### 方法论（通用模式）

每个超标文件的拆分遵循同一套决策树：

```
1. 按功能域分组：将方法/函数按职责归类
2. 检查是否有天然边界：CRUD (Create/Read/Update/Delete)、生命周期 (start/run/stop)、不同实体
3. 若分类后每块 ≤ 300 行 → 直接拆
4. 若某块仍超标 → 再按子功能或阶段拆分
5. 公共类型/接口保留在原文件或抽到 types.go
```

### A1: `strategy_experiment_worker.go`（475 行）

**当前结构分析**：
- 实验 worker 主循环（~120 行）
- 稳定性计算 `computeStability`（~40 行）
- 参数空间探索 `exploreParamSpace`（~60 行）
- 实验执行 `runExperiment`（~80 行）
- 评分桥接 `calculateScore` / `scoreComponentsToProto`（~60 行）
- 辅助函数（~40 行）
- 类型定义 + 转换（~75 行）

**拆分方案**：
```
strategy_experiment_worker.go       → 保留 worker 主循环 + 类型定义
strategy_experiment_explore.go      → exploreParamSpace + 参数变异逻辑
strategy_experiment_score.go        → calculateScore + scoreComponentsToProto + computeStability
strategy_experiment_run.go          → runExperiment + 实验状态管理
```

**拆分后预估**：worker ~180行, explore ~145行, score ~130行, run ~140行

### A2: `pipeline.go`（355 行）

**当前结构分析**：
- `startMdGatewayPipeline`（306 行，主函数）
- 包级类型定义

**拆分方案**：
```
pipeline.go                         → 保留 pipeline 编排骨架 + 类型定义
pipeline_mdgateway.go               → mdgateway 启动相关子阶段
pipeline_risk.go                    → 风控管线初始化
pipeline_subscriptions.go           → 订阅注册
```

**关键判断**：`startMdGatewayPipeline` 本身 306 行也超标，需要连带拆分。拆法：将内部 "step" 注释块提取为独立函数。

### A3: `calibration.go`（353 行）

**当前结构分析**：AI 模型校准逻辑，多为统计计算函数。

**拆分方案**：
```
calibration.go                      → 保留 Calibrate 主入口 + 类型
calibration_metrics.go              → 校准指标计算（precision/recall/F1等）
calibration_curve.go                → 校准曲线拟合
```

### A4: `systemai/chat.go`（331 行）

**当前结构分析**：AI 聊天服务，含多 provider 路由、流式/非流式、上下文管理。

**拆分方案**：
```
chat.go                             → 保留 Chat/ChatStream 入口 + 类型
chat_provider.go                    → provider 路由 + 多 provider 管理
chat_context.go                     → 上下文窗口管理 + token 计数
```

### A5: `executor.go`（323 行）

**分析**：执行算法执行器。按生命周期拆分。

**拆分方案**：
```
executor.go                         → 保留主循环 + 类型
executor_order.go                   → 订单拆分/执行逻辑
executor_risk.go                    → 执行风控检查
```

### A6: `auth_handler.go`（320 行）

**分析**：前轮已拆分 account_handler.go，auth 模块尚待拆。

**拆分方案**：
```
auth_handler.go                     → 保留接口/结构体 + New + 公共辅助
auth_login.go                       → Login + 凭证验证
auth_token.go                       → TokenRefresh + TokenRevoke + 令牌管理
auth_register.go                    → Register + 注册验证
```

### A7: `manager.go`（311 行）

**分析**：mdgateway 连接管理器。按操作类型拆分。

**拆分方案**：
```
manager.go                          → 保留 Manager 结构体 + New + 生命周期
manager_connect.go                  → 连接建立/断开/重连
manager_health.go                   → 健康检查 + 心跳
```

### A8-A15：其余 8 个文件

采用同样方法论，按功能域自然边界拆分：

| 文件 | 行数 | 拆分键 |
|------|------|--------|
| `simulated.go` | 309 | simulated_clock.go + simulated_scheduler.go |
| `analytics_repository_equity.go` | 309 | 已是按主题拆分的 repo（equity 专精），按查询复杂度拆：equity_daily.go + equity_hourly.go |
| `marketplace/service.go` | 308 | service_copytrade.go + service_marketdata.go |
| `jurisdiction.go` | 307 | jurisdiction_sanctions.go + jurisdiction_kyc.go |
| `runner.go` | 307 | runner_gateway.go + runner_subscription.go |
| `strategy_gen_handler.go` | 307 | 前轮已拆 ai_handler.go，此文件为 StrategyGen 专用。按 RPC 拆：strategy_gen_create.go + strategy_gen_list.go |
| `handlers.go` | 305 | 前轮从 ~400 降到 305。进一步拆：handlers_strategy.go + handlers_ai.go |
| `service_orders.go` | 303 | service_orders_open.go + service_orders_close.go |

---

## 三、B 组：TS 文件拆分（14 个 → 目标 ≤ 250 行）

### 方法论

TS/React 拆分模式：
```
1. 大 hooks 文件 → 拆出子 hooks
2. 大页面 → 拆出 Section 组件
3. 大客户端 → 按资源域拆
4. 大组件 → 拆子组件 + 逻辑 hooks
```

### B1-B3: Hooks 拆分

| 文件 | 行数 | 拆分方案 |
|------|------|----------|
| `useStrategyWorkspaceState.ts` (326) | → `useWorkspaceEditor.ts` + `useWorkspaceBacktest.ts` + `useWorkspaceSignals.ts` |
| `useStrategyTemplatePage.ts` (271) | → `useTemplateList.ts` + `useTemplateForm.ts` |
| `useStrategySchedulePage.ts` (271) | → `useScheduleList.ts` + `useScheduleForm.ts` |

### B4-B9: 页面拆分

| 文件 | 行数 | 拆分方案 |
|------|------|----------|
| `Market.tsx` (305) | → `MarketWatchlist.tsx` + `MarketChart.tsx` + `MarketOrderPanel.tsx` |
| `Dashboard.tsx` (297) | → `DashboardOverview.tsx` + `DashboardAccounts.tsx` + `DashboardAlerts.tsx` |
| `StrategyWorkspacePage.tsx` (269) | → 抽出 `WorkspaceToolbar.tsx` + 逻辑到 `useWorkspaceLayout.ts` |
| `PriceChart.tsx` (290) | → `ChartToolbar.tsx` + `ChartIndicators.tsx` |
| `NotificationCenter.tsx` (276) | → `NotificationList.tsx` + `NotificationItem.tsx` + `NotificationFilter.tsx` |
| `BacktestParamsCard.tsx` (261) | → `ParamsForm.tsx` + `ParamsValidation.ts` |

### B10-B14: 客户端/桥接拆分

| 文件 | 行数 | 拆分方案 |
|------|------|----------|
| `client/strategy.ts` (295) | → `client/strategy-templates.ts` + `client/strategy-backtest.ts` |
| `client/admin.ts` (258) | → `client/admin-users.ts` + `client/admin-system.ts` |
| `TradeConfirmModal.tsx` (258) | → `TradeRiskSummary.tsx` + `TradeConfirmForm.tsx` |
| `WorkspaceCodePanel.tsx` (256) | → `CodeEditor.tsx` + `CodeToolbar.tsx` |
| `bridgeStreamEvents.ts` (256) | → `bridgeTradeEvents.ts` + `bridgeAccountEvents.ts` |

---

## 四、C 组：函数超标（25+ 个 → 目标 ≤ 50 行）

### 方法论

函数拆分的标准模式：

```go
// Before: 一个大函数做所有事
func bigFunction(...) {
    // 验证 (10行)
    // 查询A (20行)
    // 查询B (20行)
    // 计算 (30行)
    // 组装响应 (20行)
}

// After: 每个语义块提取为独立函数
func bigFunction(...) {
    params := validateAndParse(...)
    dataA := fetchA(...)
    dataB := fetchB(...)
    result := compute(dataA, dataB)
    return assemble(result)
}
```

### Top 10 超标函数及拆分方案

| 函数 | 行数 | 提取为 |
|------|------|--------|
| `startMdGatewayPipeline` | 306 | `initBrokerConnections()` + `initStreamSubscriptions()` + `initRiskPipeline()` + `initBackfiller()` + `startGatewayLoop()` |
| `registerHandlers` | 260 | `registerStrategyHandlers()` + `registerAIHandlers()` + `registerAccountHandlers()` + `registerPlatformHandlers()` |
| `main` | 188 | `loadConfig()` + `initDB()` + `initNATS()` + `startServer()` |
| `Run` (runner.go) | 165 | `initGatewayForAllAccounts()` + `startReconciliationLoop()` + `startHealthCheck()` |
| `GetEquityCurve` | 157 | `queryDaily()` + `queryRawTrades()` + `aggregateToCurve()` |
| `SubscribeEvents` | 152 | `handleOrderUpdate()` + `handleProfitUpdate()` + `handleStatus()` + `handleBarUpdate()` |
| `processSync` (copytrade) | 145 | `validateSignal()` + `calculateCopySize()` + `executeCopyOrder()` |
| `registerSREHandlers` | 142 | `registerAuthEndpoints()` + `registerKillSwitchEndpoints()` + `registerBreakerEndpoints()` |
| `GetHourlyEquityCurve` | 120 | `queryHourlyBuckets()` + `fillGaps()` + `smoothCurve()` |
| `GetAccountAnalytics` | 114 | `computeReturns()` + `computeRiskMetrics()` + `computeActivityStats()` |

---

## 五、D 组：SRE `/api/*` → **接受现状**（审计修正）

### 分析

```handlers_sre.go
/api/auth/refresh                 → cookie 操作，无法 ConnectRPC
/api/auth/logout                  → 同上
/api/admin/sre/killswitch/*       → 紧急熔断，必须curl可达（3端点）
/api/admin/sre/breakers/*         → 运维工具需简单HTTP（2端点）
/api/admin/sre/canary/*           → 灰度管理（3端点）
/metrics                           → Prometheus 标准端点
```

### 判断：不迁移。接受现状。

**原因**：
- Kill switch 是**紧急操作**，系统降级时必须能 `curl -X POST` 直接触发。走 ConnectRPC（proto 序列化 + 生成客户端依赖）反而增加故障面。这是 **最优解反例** — over-engineering 违背可靠性优先原则
- SRE 端点是**内部运维端点**，不是 "External API"——平台规则 `ConnectRPC + SSE ONLY` 针对的是外部业务 API
- Auth cookie 操作本身就是 HTTP 原生语义，ConnectRPC 无等价物
- `/metrics` 是 Prometheus 行业标准端点，不可替换

**处理**：在 `handlers_sre.go` 头部加注释标注 `// Internal SRE endpoints — exempt from ConnectRPC requirement (operational necessity).`

---

## 六、E 组：`.env` 密钥

### 当前状态
- 文件存在于 `/opt/ant/.env`，不在 git 跟踪中
- 含 `JWT_SECRET` + `DB_PASSWORD` 真实值

### 方法
1. 确认 `.env` 在 `.gitignore` 中
2. 若密钥曾提交过 → 轮换密钥 + 清理 git 历史
3. 建议：迁移到环境变量注入（systemd/k8s secret）或 `.env.example` 模板

---

## 七、执行顺序

```
Phase 1 (并行): A组 Go 拆分 (15 文件) + B组 TS 拆分 (14 文件)
Phase 2 (独立): C组 函数拆分 (25+ 函数)
Phase 3 (独立): D组 SRE 标注合规例外 (1 条注释)
Phase 4 (独立): E组 .env 确认清理
```

每 Phase 完成后 `make check-size` + `go build ./...` + `npm run build` 验证。

---

## 八、验证清单

```bash
# Go
cd /opt/ant/backend && go build ./... && make check-size

# TS
cd /opt/ant/frontend && npm run build

# Python
cd /opt/ant/strategy-service && python3 -c "from app.main import app" 2>/dev/null || python3 -m py_compile app/main.py
```
