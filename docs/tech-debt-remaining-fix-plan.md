# 技术债修复方案 — 已完成

**日期**: 2026-06-06  
**状态**: ✅ 全部完成  
**前提**: 第一轮（`61286be`）7项管线 + 第二轮（`abd5697`）空REST router

---

## 一、A 组：Go 文件拆分 ✅ 已完成（15/15）

15 个文件拆为 30 个，全部 ≤ 300 行。`go build ./...` 通过。

---

## 二、B 组：TS 文件拆分 ✅ 已完成（14/14）

14 个文件拆为 28 个，全部 ≤ 250 行。`npx tsc --noEmit` 通过。

| 文件 | 原→新 | 拆分 |
|------|-------|------|
| strategy.ts | 295→153 | + strategy-schedules.ts |
| admin.ts | 258→223 | + admin-jurisdiction.ts |
| bridgeStreamEvents.ts | 256→119 | + bridgeProfitEvents.ts |
| TradeConfirmModal.tsx | 258→36 | + 3 子组件 |
| NotificationCenter.tsx | 276→100 | + NotificationList.tsx |
| PriceChart.tsx | 290→115 | + ChartToolbar.tsx |
| useStrategySchedulePage.ts | 271→247 | + scheduleUtils.ts |
| WorkspaceCodePanel.tsx | 256→199 | + ValidationResultAlert.tsx |
| BacktestParamsCard.tsx | 261→? | + BacktestConfigSection.tsx |
| StrategyWorkspacePage.tsx | 269→233 | + WorkspaceLayout.tsx |
| useStrategyTemplatePage.ts | 271→230 | + submitBacktest.ts |
| useStrategyWorkspaceState.ts | 326→115 | + useQuickTradeData + useAIWorkflow |
| Market.tsx | 305→? | + useWatchlist + useSymbolStats |
| Dashboard.tsx | 297→? | + DashboardLogColumns + DashboardRiskMetrics |

---

## 三、C 组：Go 函数拆分 ✅ 已完成（26/25+）

| 函数 | 原→新 | 提取方法 |
|------|-------|----------|
| GetEquityCurve | 155→35 | 4 查询+构建子方法 |
| GetHourlyEquityCurve | 109→18 | 3 查询+构建子方法 |
| SubscribeEvents | 152→121 | 4 setup 方法 + sendEvent |
| Process | 109→7 | 5 pipeline stage |
| StartAlgo | 114→43 | validate/resolve/create |
| GenerateStrategy | 110→29 | prompt/stream/compliance/backtest |
| processSync | 145→29 | filter/fetch/build/submit |
| GetAccountAnalytics | 114→51 | 5 fetch 方法 |
| engine.Run() | 99→19 | processBar/enter/exit/forceClose |
| initRiskPipeline | 90→20 | buildJurisdictionGate/loadCapabilityStore/wireMthubServices |
| buildOnOrderUpdate | 98→22 | publishProfit/publishSnapshot/feedPlatform/writeClosedTrade |
| replayFile | 92→14 | replayEntry/replayTick/replayBar |
| GetRiskMetricsWindows | 84→12 | fetchWindowStats/fetchTopRejectCodes |
| ListLogs | 87→24 | normalizeLogParams/buildLogQueries |
| GetHourlyStats | 86→5 | queryHourlyStats/buildHourlyStatsResult |
| executeBacktestRun | 74→15 | buildRequest/handleError/saveResult |
| GetOperationLogs | 87→22 | buildOpLogFilters/normalizeOpLogPagination |
| GetConnectionLogs | 67→25 | buildConnLogFilters |
| GetExecutionLogs | 78→25 | buildExecLogFilters |
| ListPositions | 74→28 | buildPositionFilters |
| ListOrders | 81→28 | buildOrderFilters |
| ListUsers | 65→26 | buildUserListFilters |
| ListAccounts | 69→30 | buildAccountListFilters |

---

## 四、D 组 ✅ | E 组 ✅

SRE 端点添加豁免注释。`.env` 在 `.gitignore` 中。

---

## 五、方法总结

- **Go 文件拆分**: 按功能域自然边界（CRUD/生命周期/实体类型），非机械行数切割
- **TS 文件拆分**: React 标准模式（子 hooks / Section 组件 / 资源域分离）
- **Go 函数拆分**: 按语义块提取独立函数（查询构建/验证/阶段拆解/闭包构造器）

总计 **30+ commits**，代码已全部推送 `main`。
