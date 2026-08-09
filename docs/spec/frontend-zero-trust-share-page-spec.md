# Spec: 分享业绩页——后端正确回撤 + 衍生统计迁移（零信任对齐 + 修真 bug）

> 涉及功能块：frontend / api-gateway / account-mgmt（share 子系统）
> 类型：**正确性修复（公开面 bug）+ 架构原则修复**（中优，非上线阻断）
> 来源审计：2026-08-09「智能调优管线 + 前端零信任全局排查」（审计方 Claude Code）
> 关联原则：`CLAUDE.md`「所有计算留后端，前端零信任」（公开面尤其刚性）
>
> **⚠️ 本 spec 在初版基础上修订**：审计验证 `BuildSharePerformance` 后端计算时发现**潜在正确性 bug**——后端 `MaxDrawdown` 实为"最差单笔交易"非"最大回撤"。故本 spec 不只是"前端迁移到后端"，而是**后端必须先算对回撤**，否则会把错误值当作权威值固化。

---

## 一、问题（两个相互关联的缺陷）

产品第一负责人原则：**所有计算留后端，前端零信任**——前端可被篡改，公开面数字必须后端权威。全局排查发现 `/share/:token` 分享页（`AppRoutes.tsx:173` live 公开路由，面向潜在买家）违反此原则，且核验时连带发现后端 bug：

### 缺陷 A（正确性 bug，根因）：后端 `MaxDrawdown` 是"最差单笔"非"回撤"

`share_service.go:summarizeTrades` 的 `maxDD`（`:177-179`）：
```go
if t.Profit.LessThan(s.maxDD) { s.maxDD = t.Profit }  // maxDD 初值=0
```
这只记录**单笔最负 Profit**（最差单笔交易），根本不是最大回撤（回撤=净值 peak-to-trough 下降）。该误标值经 `maxDrawdownStr()`(:196) → `perf.MaxDrawdown` → 两条公开出口：
1. **OG 社交预览图**（`share_og_image.go:97 renderOGImagePNG(..., perf.MaxDrawdown, ...)`）——链接被分享时的社交平台预览，**广泛可见**，显示的是"最差单笔美元值"却标成"最大回撤"。
2. 分享页 proto `MaxDrawdown` 字段（`share_handler.go:109`）。

### 缺陷 B（零信任违背）：前端用 equity curve 重算回撤 + 算交易级统计

`SharePerformancePageStats.ts:computeTradeStats` + `SharePerformancePageHelpers.tsx` 在前端 TS 从 raw trades/equityCurve 算：
- `computeMaxDrawdownPct(equity, fallback)`（`:60-71`）：从 equityCurve 逐点算**真正的** peak-to-trough 回撤%（前端算的是对的，恰好在补后端缺陷 A 的错）。
- `computeTradeStats`（`Stats:19-37`）：win/loss 计数、best/worst trade、avgWin/avgLoss、winPct。
- `aggregateBySymbol`（`:73-83`）：按品种聚合 count + net。

### 两个缺陷叠加的后果（同一指标三个不同数）

| 出口 | 显示的"最大回撤" | 实际语义 | 单位 |
|---|---|---|---|
| OG 社交预览图 | `perf.MaxDrawdown` | 最差单笔交易 | 美元绝对值 |
| 分享页 KPI（`Stats:46`） | `computeMaxDrawdownPct` | 真回撤 | 百分比 |
| 分享页 fallback（无 equity curve 时） | 后端 `data.maxDrawdown` | 最差单笔交易 | 美元绝对值 |

→ 同一"最大回撤"指标在社交预览图（美元、最差单笔）和分享页 KPI（百分比、真回撤）显示**完全不同的数字**。买家混淆。

### 严重性

**中（非上线阻断）**：
- 原始 trades/equityCurve 后端可信，前端只是重算聚合（非凭空伪造）。
- 买家最终信任锚点是**实盘战绩**（`trade_records` 哈希链）+ FEAT-5 衰减，非分享页 KPI。
- **但**：① 公开面（OG 预览广泛可见）显示错误语义的"回撤"；② 三处数字不一致；③ 违背确立原则 → 应修。且修缺陷 A 让后端值变正确，是迁移动零信任的**前提**（否则把错误值当权威固化=更糟）。

### 审计确认合规（不在本 spec 范围）

- **Smart Tuning**：瘦客户端，全部计算（优化器/评分/regime/OOS/过拟合）后端。✅
- **analytics/summary/accounts**：用后端预算 rolling 指标（`roll.drawdownCurve`/`monthlyWinRates`/`drawdownEvents`），前端只 chart shaping。✅
- **marketplace cards/tabs**：渲染后端值（`summary.maxDrawdown`/`row.winRate`），仅 `.toFixed`。✅
- **admin 页面**：后端值的 `.toFixed` 格式化（非公开信任面）。✅
- `Summary.helpers.ts:getProfitPieData`：用后端 winRate+total 拆饼=展示变换。✅

---

## 二、决策（最优解）

**后端算对 + 前端只渲染**：
1. **先修缺陷 A**：后端 `BuildSharePerformance` 用 equity curve 算**真正的 peak-to-trough 最大回撤**（decimal，百分比），复用现有算法（见 REUSE）。
2. **补齐缺陷 B 的迁移**：后端 `tradeSummary` 增算交易级统计（best/worst/avgWin/avgLoss/品种聚合），proto 加字段。
3. **前端删全部推导**（`computeMaxDrawdownPct`/`computeTradeStats`/`aggregateBySymbol`），纯渲染后端值。
4. 结果：OG 预览图 + 分享页 KPI **同一正确回撤值**，单一后端真相源，前端零计算。

不做"前端也校验一遍"——那是双真相源反模式。

---

## 三、实现任务（施工方）

### T1 — 后端修真回撤 + 补交易级统计（核心）

文件：`backend/internal/connect/user/share_service.go`（`summarizeTrades` + `tradeSummary` + `BuildSharePerformance`）

**① 修回撤（缺陷 A 根因）**：
- 删 `summarizeTrades` 里 `maxDD = 单笔最负 Profit` 的错误逻辑。
- 在 `BuildSharePerformance`（已有 `equityPoints []*model.EquityPoint`，`:62`）算**真正 peak-to-trough 回撤**：遍历 equity，维护 running peak，`drawdown = (peak - current) / peak`，取 max，×100 成百分比，**用 `decimal.Decimal`**（非 float64，遵 CLAUDE.md 数据精度）。
- **REUSE（动工前必须跑 `bash scripts/cap.sh drawdown` 等查）**：后端已有两处正确算法可直接复用/抽公共：
  - `backend/internal/connect/system/analytics_rolling.go:103-124 computeDrawdownEvents` —— `runningMax` + `(runningMax-eq)/runningMax*100`（float64 版，回撤事件）。
  - `backend/internal/marketplace/live_performance.go:208-222` —— decimal 版 `peak.Sub(equity).Div(peak)`。
  - **优先抽一个共享 `computeMaxDrawdownPct(equityPoints []*model.EquityPoint) decimal.Decimal`**，share_service 与 analytics 共用，消除重复（DRY + 单一真相源）。注意单位口径统一为**百分比**（与分享页现有显示 + marketplace `quality.go` `MaxDrawdownPct` 闸一致）。
- `maxDrawdownStr()` 返回真回撤百分比字符串。

**② 补交易级统计（缺陷 B 迁移）**：`tradeSummary` 已有 wins/losses/grossProfit/grossLoss，补：
- `bestTrade`/`worstTrade`（Profit max/min，decimal）
- `avgWin`（grossProfit/wins）/ `avgLoss`（grossLoss/losses）（decimal）
- `symbolStats`（按 `t.Symbol` 聚合 `{symbol, count, net}`，decimal 求和）——**REUSE**：`backend/internal/connect/system/analytics_compute.go:228/258` 已有按品种聚合，优先复用/抽公共。

### T2 — proto 加字段

文件：`proto/ant/v1/share.proto`（`GetSharedPerformanceResponse`）

加嵌套 message（保持整洁，decimal 全用 string 传输）：
```protobuf
message ShareTradeStats {
  int32 winning_trades = 1;
  int32 losing_trades = 2;
  string best_trade = 3;     // decimal as string
  string worst_trade = 4;
  string avg_win = 5;
  string avg_loss = 6;
}
message ShareSymbolStat {
  string symbol = 1;
  int32 count = 2;
  string net = 3;            // decimal as string
}
```
`GetSharedPerformanceResponse` 加 `ShareTradeStats trade_stats = N;` + `repeated ShareSymbolStat symbol_stats = N;`。
（`MaxDrawdown` 字段复用——T1 已让它变正确，语义=真回撤百分比，无需新字段。）

`share_handler.go:107-115` 的 `connect.NewResponse` 填充 `trade_stats`/`symbol_stats`。

### T3 — 前端删推导，改纯渲染

文件：`frontend/src/pages/share/SharePerformancePageStats.ts` + `SharePerformancePageHelpers.tsx`

- 删 `computeTradeStats`、`computeMaxDrawdownPct`、`aggregateBySymbol` 三个推导函数。
- `buildKpiCards`/分享页主体改读后端：胜率 `data.winRate`、回撤 `data.maxDrawdown`（现已正确）、avg win/loss/best/worst 来自 `data.tradeStats`、品种表来自 `data.symbolStats`。
- 保留纯格式化 helper（`toNum`/`fmt`/`avgHoldingText`）。
- `ShareData` 接口同步加 `tradeStats`/`symbolStats` 字段（对齐 proto gen 类型）。

### T4 — 对抗证明

- **回撤正确性（缺陷 A）**：构造 equity curve `[100, 120, 90, 110]`（peak=120, trough=90, 回撤=25%）→ 后端 `maxDrawdownStr()` 返回 "25"（或与现有口径一致的 25.x）。**对抗**：T1 故意退回旧逻辑（`maxDD=单笔最负`）→ 测试必红（证明修了真 bug，非空操作）。
- **零信任迁移（缺陷 B）**：后端给定 10 条 trades（6 盈 4 亏，已知 best/worst/avg）→ 分享页 KPI 与后端一致。**对抗**：删前端推导函数后 `grep computeMaxDrawdownPct|computeTradeStats|aggregateBySymbol src/` 零命中 + `tsc` 无残留引用。
- **OG 一致性**：OG 预览图（`share_og_image.go`）与分享页 KPI 用同一 `perf.MaxDrawdown` → 数字一致。

---

## 四、验收标准（审计方实测）

1. `/share/:token` 所有 KPI（胜率/回撤/avgWin/avgLoss/best/worst/win-loss 数/品种 net）来自后端 proto；前端 grep 三个推导函数零命中。
2. **回撤语义正确**：后端 `MaxDrawdown` = equity peak-to-trough 回撤%（非单笔最差）；OG 预览图与分享页 KPI 同值。
3. `go build` + `go test ./internal/connect/user/...` + `npm run build` + `npm test` 全绿；`check-file-lines --strict` 0🔴。
4. 后端用 `decimal.Decimal`（非 float64）算回撤/best/worst/avg/net。
5. 对抗证明成立（退回旧回撤逻辑 → 测试红）。

---

## 五、完工回填纪律（施工方，不做 = 失败）

1. `tech-debt-registry.md` FE-TRUST-1 `🟦open → ✅done`（标日期）+ 追加**真实根因**（重点写清缺陷 A：后端 maxDD=单笔最负、被误标为回撤、OG 预览公开显示）+ 修复方式 + 对抗证明。**如实写**——若真根因与本 spec 假设不同，高价值纠偏。**不自宣告完成**，等审计方实测。
2. `handover-audit-plan.md` 变更日志加一行。
3. commit（doc + code 一并，commit message 写清"修分享页回撤 bug + 零信任迁移"）。
4. 普遍经验 → CLAUDE.md「零信任」段补一句"公开面衍生统计必须后端算；回撤等指标须后端用 equity 算真值，勿用单笔最差冒充"。

---

## 六、范围边界

- **不**改 analytics/summary/accounts（已合规）。
- **不**改 admin 页面 `.toFixed` 格式化（合规）。
- **不**改 Smart Tuning（合规）。
- **不**顺手重构 share_service——只修回撤 + 补交易级统计 + 删前端推导。
- 回撤单位口径统一为**百分比**（与分享页现有显示 + marketplace quality gate 一致）；如发现 quality.go 的 `MaxDrawdown` 期望的是别的单位，以分享页现有百分比显示为准并在回填里说明。
