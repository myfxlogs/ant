# 主业产品（策略市场）Launch-Readiness 地图（2026-08-13 审计方）

> 3 agent 并行只读审计：需求侧用户环 / 实盘执行链深度 / 供给侧+AI。目标回答"距上线还差什么"。
>
> **核心结论：产品比预期接近完成。** 需求侧 6 环代码层**全通、无 mock/占位/假数据**；实盘执行链核心路径（signal→风控→下单→equity push→战绩）**真实且连通**；MQL 手写作者侧**上线就绪**；AI 迭代循环**真闭环**。距上线是一个**有限、集中的 launch-blocking 清单**，集中在 live 执行 + AI 部署这条最难的路径上。

---

## Tier 0 — 核心价值断裂（launch 前必修）

| # | 缺陷 | 位置 | 影响 | 修复量 |
|---|---|---|---|---|
| **LIVE-PRICE-4** | 硬编码 symbol 原子订阅失败 → 实盘零报价 → 无法开仓 | `runner_gateway.go:127` defaultQuoteSymbols + SubscribeMany | 实盘核心价值断 | 中（FetchAllSymbols 过滤 + 逐 symbol） |
| **SUPPLY-1** | **AI(Python)策略能生成/上架，但不能被买家部署实盘、也不能经 worker 重测**——运行时重入层只有 MQL 分支 | `vm_live_session.go:32`/`backtest_worker.go:184`/`strategy_execution_handler.go:307`/`backtest_worker_vm.go:61`（`executePythonVMBacktest` 从未实现） | 打在"AI 迭代策略"核心差异化上 | 中（4 个 dispatch 点加 Python 分支；编译器/VM 已语言无关） |
| **EXEC-1** | 实时 `trade_records` 写入**必失败**：`writeClosedTradeRecord`/`WriteClosedTrade` 不设 UserID（uuid.Nil）→ 违反 NOT NULL+FK | `cmd/server/pipeline_callbacks.go:64` + `mthub/service_orders.go:122` | trade 笔数/胜率延迟到重连才更新（equity/PnL/回撤不受影响） | **一行**（签名加 userID，buildOnOrderUpdate 已有） |
| **EXEC-2** | OMS orders **永远卡 SUBMITTED**：OnOrderUpdate 回调从不调 `omsTransition`，broker fill/reject 不转状态 | `mthub/service_orders.go:183` + `pipeline_callbacks.go:27` | 订单生命周期追踪断裂，9 个终态不可达，状态不可信 | 中（callback 解析 UpdateType → omsTransition） |
| **EXEC-3** | **OnTrade 执行模型生产环境死亡**：`PublishTradeEvent` 零调用方，subscribe 端通但 publish 端死 | `mthub/service.go:221` | 依赖 OnTrade 的策略（移动止损/martingale）静默失效 | 中（OnOrderUpdate 检测 trade event → Publish） |

## Tier 1 — 信任护城河（"实盘战绩公开可信"是核心差异化）

| # | 缺陷 | 位置 | 影响 | 决策方 |
|---|---|---|---|---|
| **TRUST-1** | **Demo 账户（虚拟金、真实执行）与真实金账户战绩混为一体公开展示**，无标注/无过滤 | `marketplace/live_performance.go:42` LinkLiveAccount 不查 account_type | 用户发现"实盘战绩"来自 demo → 信任护城河崩 | **业务决策**：强制 real-only？或允许 demo 但显式标注？ |
| **TRUST-2** | OOS-at-publish 门是死代码（快照从不填 OOS）+ 质量阈值近乎 no-op（min_sharpe=-1/min_win_rate=0） | `quality.go:301` + `backtest_persistence.go:219` + migration 260 | 纯样本内过拟合/垃圾策略可上架 | = 已知 TUNING-OVERFIT-2；**业务决策**：上线前收紧？ |
| **G1** | `/strategy/:strategyId` 公开详情页用**已认证 client**（`marketplaceClient.getStrategyPublicInfo`），但路由公开无 auth guard → 未登录访客 401 | `AppRoutes.tsx:188` + `StrategySharePage.tsx:7` | 公开发现→转化漏斗断 | 改 public transport（参考 sharePublicClient） |

## Tier 2 — Happy-path 摩擦（基础用户认可）

| # | 缺陷 | 位置 | 影响 |
|---|---|---|---|
| **G2** | 购买后无"去绑账户/去启用实盘"引导；DeployScheduleModal 空账户态无"绑定 MT 账户"入口 → 新买家走到部署即卡死 | `useMarketplace.ts:168` + `DeployScheduleModal.tsx:117` | 新用户 happy-path 最实际卡点 |
| **G3-G5** | 普通用户侧边栏无"账户"入口；`/credits`/`/subscription` 不在导航；手动触发硬编码 paper | `MainLayout.tsx:40` 等 | UX 摩擦，非阻断 |

## 延后（不阻碍 launch，用户已定）

- TRON-GRID-1 / TRON-SECURITY-1 / BROKER-SEARCH-1（资金/搜索边界硬化，等收费上线后）
- risksvc / state_cache / canary / 三层幂等：**实现了但零生产接线**的死代码（按需接入或标 future）
- timer-type schedule 只触发一次（若产品只需 event-type 流式调度则无影响——**需产品确认**）

## 需用户（业务/产品）决策的点

1. **TRUST-1**：demo 战绩处理（强制 real-only / 标注 / 允许）——信任底线。
2. **费率**：种子 3-10% vs CLAUDE.md "15-30%"——确认实际抽成。
3. **TRUST-2**：上线前是否收紧质量门（OOS + 阈值）。
4. **调度语义**：需不需要 cron/timer-type 递归调度，还是只 event-type 流式（影响是否修 timer-type bug）。
5. **SUPPLY-1 范围**：AI 策略实盘部署是 launch 必须，还是 MQL 先上、AI 部署随后？（产品节奏）

## 已核实扎实（不用动，给信心）

- 需求侧 6 环全通、零 mock；权益闸（CanAccessCode）真生效（调度+回测双闸）；MT 密码加密落库；钱包充值真实（HD 地址）。
- 实盘核心链真实：SimBroker 真撮合、风控 gate fail-closed（7 规则）、hash chain 防篡改、equity/回撤后端算（零信任合规）、paper/live 分离。
- MQL 作者侧上线就绪；AI 迭代真闭环（1000 轮 LLM↔工具↔回测，I1-I4 代码强制）。
- 信任机制：防篡改快照、所有权校验、DEGRADED 硬阻断。
