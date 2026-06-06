# 剩余技术债修复方案

**日期**: 2026-06-06  
**状态**: A组✅ D组✅ E组✅ | B组 7/14 | C组 待开工  
**前提**: 第一轮（`61286be`）7项管线修复 + 第二轮（`abd5697`）空REST router清理

---

## 一、完成情况

### A 组：Go 文件拆分 ✅ 已完成（15/15）

| 原文件 | 原→新行数 | 拆分文件 |
|--------|-----------|----------|
| calibration.go | 353→212 | + calibration_repo.go (151) |
| auth_handler.go | 320→105 | + auth_token.go (148) + auth_register.go (97) |
| executor.go | 324→137 | + executor_run.go (120) + executor_lifecycle.go (55) |
| manager.go | 311→145 | + manager_tick.go (82) + manager_health.go (74) |
| simulated.go | 309→149 | + simulated_ticker.go (70) + simulated_timer.go (67) |
| jurisdiction.go | 307→213 | + jurisdiction_stub.go (95) |
| analytics_repository_equity.go | 309→172 | + equity_hourly.go (114) |
| service_orders.go | 303→200 | + service_orders_close.go (106) |
| marketplace/service.go | 308→257 | + service_subscription.go (78) |
| strategy_gen_handler.go | 307→182 | + strategy_gen_helpers.go (127) |
| runner.go | 307→212 | + runner_gateway.go (98) |
| handlers.go | 305→220 | + handlers_pipeline.go (110) |
| systemai/chat.go | 331→239 | + chat_stream.go (93) |
| pipeline.go | 355→237 | + pipeline_callbacks.go (120) + pipeline_helpers.go (20) |
| experiment_worker.go | 475→221 | + experiment_scoring.go (154) + experiment_utils.go (120) |

方法：按功能域自然边界拆分（CRUD/生命周期/实体类型），非机械行数切割。

---

### B 组：TS 文件拆分 ⏳ 7/14 完成

**已完成：**

| 文件 | 原→新行数 | 拆分文件 |
|------|-----------|----------|
| strategy.ts | 295→153 | + strategy-schedules.ts (115) |
| admin.ts | 258→223 | + admin-jurisdiction.ts (37) |
| bridgeStreamEvents.ts | 256→119 | + bridgeProfitEvents.ts (93) |
| TradeConfirmModal.tsx | 258→36 | + AutoTrade + RiskConfig + StrategyExecute (3文件 ~190) |
| NotificationCenter.tsx | 276→100 | + NotificationList.tsx (105) |
| PriceChart.tsx | 290→115 | + ChartToolbar.tsx (92) |
| useStrategySchedulePage.ts | 271→247 | + scheduleUtils.ts (24) |

**剩余 7 个：**

| 文件 | 行数 | 计划拆分方式 |
|------|------|-------------|
| useStrategyWorkspaceState.ts | 326 | 拆出子 hooks（workspace editor/backtest/signals） |
| Market.tsx | 305 | 拆出 Section 组件（Watchlist/Chart/OrderPanel） |
| Dashboard.tsx | 297 | 拆出面板组件（Overview/Accounts/Alerts） |
| useStrategyTemplatePage.ts | 271 | 提取回测/表单逻辑到独立 hooks |
| StrategyWorkspacePage.tsx | 269 | 提取 Toolbar + 逻辑到 useWorkspaceLayout |
| BacktestParamsCard.tsx | 261 | 提取 ParamsForm + ParamsValidation |
| WorkspaceCodePanel.tsx | 256 | 提取 CodeEditor + CodeToolbar |

方法：React 标准拆分模式（子 hooks / Section 组件 / 资源域分离）。

---

### D 组：SRE `/api/*` ✅ 已完成

在 `handlers_sre.go` 添加豁免注释：
```
// Internal SRE endpoints — exempt from ConnectRPC requirement (operational necessity).
// Kill switches, circuit breakers, and canary configs need plain HTTP (curl-able)
// to remain functional in degraded states. Auth cookie endpoints are HTTP-native.
```

### E 组：`.env` ✅ 已确认

`.env` 在 `.gitignore` 中，未入 git 跟踪。无需额外操作。

---

## 二、执行顺序

```
Phase 1: A组 Go ✅ + B组 TS 7/14
Phase 2: C组 函数拆分 (25+ 函数) — 待开工
Phase 3: B组余下 7 个 TS 文件 — 待继续
```

---

## 三、验证

```bash
cd /opt/ant/backend && go build ./... && make check-size  # ✅ 通过
```
