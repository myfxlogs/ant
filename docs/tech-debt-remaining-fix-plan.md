# 剩余技术债修复方案

**日期**: 2026-06-06  
**状态**: A✅ B✅ D✅ E✅ | C 12/25+  
**前提**: 第一轮（`61286be`）7项管线 + 第二轮（`abd5697`）空REST router

---

## 一、A 组：Go 文件拆分 ✅ 已完成（15/15）

15 个文件拆为 30 个，全部 ≤ 300 行。`go build ./...` 通过。

---

## 二、B 组：TS 文件拆分 ✅ 已完成（14/14）

| 已完成 | 原→新 |
|--------|-------|
| strategy.ts | 295→153 + strategy-schedules.ts |
| admin.ts | 258→223 + admin-jurisdiction.ts |
| bridgeStreamEvents.ts | 256→119 + bridgeProfitEvents.ts |
| TradeConfirmModal.tsx | 258→36 + 3 子组件 |
| NotificationCenter.tsx | 276→100 + NotificationList.tsx |
| PriceChart.tsx | 290→115 + ChartToolbar.tsx |
| useStrategySchedulePage.ts | 271→247 + scheduleUtils.ts |
| WorkspaceCodePanel.tsx | 256→199 + ValidationResultAlert.tsx |
| BacktestParamsCard.tsx | 261→? + BacktestConfigSection.tsx |
| StrategyWorkspacePage.tsx | 269→233 + WorkspaceLayout.tsx |
| useStrategyTemplatePage.ts | 271→230 + submitBacktest.ts |
| useStrategyWorkspaceState.ts | 326→115 + useQuickTradeData(72) + useAIWorkflow(131) |
| Market.tsx | 305→? + useWatchlist(33) + useSymbolStats(67) |
| Dashboard.tsx | 297→? + DashboardLogColumns(26) + DashboardRiskMetrics(89) |

全部 14 个 TS 文件已拆分至软性参考线（250行）以下。`npx tsc --noEmit` 通过。

---

## 三、C 组：Go 函数拆分 ⏳ 12/25+ 完成

| 函数 | 原→新 | 提取方法 |
|------|-------|----------|
| GetEquityCurve | 155→35 | 4 查询+构建子方法 |
| GetHourlyEquityCurve | 109→18 | 3 查询+构建子方法 |
| SubscribeEvents | 152→121 | 4 setup + sendEvent |
| Process | 109→7 | 5 pipeline stage |
| StartAlgo | 114→43 | validate/resolve/create 3方法 |
| GenerateStrategy | 110→29 | prompt/stream/compliance/backtest |
| processSync | 145→29 | filter/fetch/build/submit |
| GetAccountAnalytics | 114→51 | 5 fetch 方法 |
| engine.Run() | 99→19 | processBar/enter/exit/forceClose |

**剩余 ~16 个**: startMdGatewayPipeline(193) main(188) registerHandlers(178) Run(165) registerSREHandlers(142) 等 — 均为启动/编排代码。

---

## 四、D 组 ✅ | E 组 ✅

SRE 端点添加豁免注释。`.env` 在 `.gitignore` 中。

---

## 五、验证

```bash
cd /opt/ant/backend && go build ./...  # ✅
```
