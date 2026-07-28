# ADR-0027 v2 · 策略模块前端重构（修订版 — Cascade 提案）

- **状态**：**Implemented** — 2026-07-27。Phase A-D + E1/E2 全部完成部署。定稿决议见 §14；本文取代 `0027-strategy-gallery-redesign.md`（GLM 版）为实施依据。
- **日期**：2026-07-23
- **决策者**：Team（Cascade / Claude / GLM 三方讨论）
- **关联**：本文件是 `0027-strategy-gallery-redesign.md` 的修订提案，保留其宏观骨架，修正三处设计并补齐其未覆盖的架构层。
- **前置声明（决策者态度）**：Owner 已明确「只要方案最优，模块级推倒重做也接受」。因此本 ADR **以最优性为唯一约束**，不做风险规避式保守。凡本文仍建议「重构而非重写」之处，均基于**最优终点判断**（见 §5），而非风险考量。

---

## 1. 背景与本修订的定位

策略模块现有四个二级页：Workspace（编辑+回测，一体化 IDE）、Library（模板表格）、Live（实盘）、Market Tools。GLM 版 0027 提出 Gallery + Detail + Guided Create 三层架构，并精简 Workspace。

本修订**认同 GLM 的宏观拆分**（浏览与创作分离），但基于对已实现代码的完整核查（见 §2），提出三点更优设计与一项架构补强：

| 项 | GLM 0027 | 本修订 v2 | 关系 |
|----|----------|-----------|------|
| 浏览层 Gallery | 卡片网格 | 卡片网格 **+ 多选 Compare** | 保留并增强 |
| 深入层 Detail | 4 Tab（概览/代码/回测/部署） | Detail 为**资源枢纽**，回测/部署做**轻量默认档**，深度迭代跳 Workspace | 修正（去重实现） |
| 创建 | **4 步 Stepper 向导页** | **砍掉向导**，改「起点选择 Modal → 非线性 Workspace + 首访引导浮层」 | 否决并替代 |
| 路由 | `/strategy/:id` | **避让 `/strategy/:strategyId`（已被占用）**，改资源化层级 | 修正（含硬约束） |
| Workspace 状态架构 | 未覆盖 | **283 行上帝 hook → feature-slice 分解** | 补齐（核心） |
| 卡片数据 | 「GetTemplateStats 或前端聚合」（悬而未决） | 明确 **`ListStrategyCards` 反范式化 RPC** | 定案 |

---

## 2. 已验证的现状约束（决策依据）

以下均经代码核查确认，是本 ADR 一切决策的事实前提：

### 2.1 数据/RPC 层已就绪

- 模板 CRUD：`strategyApi.{listTemplates, getTemplate, createTemplate, updateTemplate, deleteTemplate, createTemplateDraft, updateTemplateDraft}`（ConnectRPC）。
- 回测：`strategyRuntimeApi.{listBacktestRuns, getBacktestRun, watchBacktestRun(SSE), runBacktest, cancelBacktestRun}`。
- 调度：`listSchedules / watchSchedules(SSE)`。
- 实体 `StrategyTemplate`：`name/description/code/isSystem/isPublic/parameters/tags/i18n`；12 个内置模板种子（`pages/strategy/templates/*.ts`）。

**含义**：Gallery/Detail 所需数据基本齐备，唯一缺口是「每模板回测摘要」（见 2.4）。

**2026-07-23 新增**：`StrategyLibraryPage.tsx`（187 行 Table 列表页）和 MQL Import Drawer（`ImportEAPanel` 接入 Workspace Code Tab）已于同日完成。Library 的搜索/筛选/分页逻辑可作为 Gallery 骨架复用。

### 2.2 硬约束：路由 `/strategy/:strategyId` 已被占用

`routes/AppRoutes.tsx` 末尾：`<Route path="/strategy/:strategyId" element={<StrategySharePage />} />`（公开分享落地页）。
**GLM 版 `/strategy/:id` 作为 Detail 会与之冲突**。本修订路由方案必须避让（见 §4.2）。

### 2.3 现存 latent bug：模板打开参数不一致

`StrategyLibraryPage.tsx` 跳转 `?template=${id}`，而 `useStrategyWorkspaceState` 读 `searchParams.get('templateId')`。二者参数名不一致 → **「Open in Workspace」当前不会加载模板**。重构 L1 时顺带修复，并统一为 `templateId`。

### 2.4 缺口：缺少卡片摘要聚合 RPC

结果优先的卡片需要 sparkline + KPI（胜率/回撤/PF/在跑数）。现状只能对每模板 N+1 调 `listBacktestRuns`。需新增聚合 RPC（见 §4.7）。

### 2.5 三层价值分布（决定重做边界）

| 层 | 组成 | 成熟度 | 最优操作 |
|----|------|--------|----------|
| **L1 消费/CRUD** | `StrategyLibraryPage`（187 行纯表格）+ `StrategyTemplateEditModal` + `StrategyTemplateColumns` | 弱，薄（~30KB） | **推倒重做** → Gallery + Detail |
| **L2 创作** | Workspace + `components/`(37) + `hooks/`(17) + `workspaceStore` | UX 成熟，仅状态编排差（283 行上帝 hook） | **内部重构**（feature-slice），保 UI |
| **L3 部署/实盘/回测引擎** | `ScheduleLaunchForm`(13KB)、`LiveStrategyPage`、`useBacktestRunner`、SSE 管线 | 成熟、多处复用 | **原样复用** |

---

## 3. 决策总览

采用 **浏览/创作分离** 的资源化架构：

1. **L1 全新重做**：表格 → Gallery（卡片+对比）+ Detail（资源枢纽）。
2. **L2 内部重构**：上帝 hook + 单体 store → feature-slice 组合；UI/交互不变（保留一体化 IDE 心流）。
3. **L3 复用**：回测引擎、调度、实盘、SSE 管线不动。
4. **创建去向导化**：起点 Modal + 非线性 Workspace。
5. **新增** `ListStrategyCards` 聚合 RPC。

核心原则：**Integration ≠ Monolith**。页面对用户可以是「一体的」（友好），代码对维护者必须是「分片的」（最优）。

---

## 4. 详细设计

### 4.1 重做边界（L1 重写 / L2 重构 / L3 复用）

- **L1（重写）**：删除 `StrategyLibraryPage`、`StrategyTemplateColumns`、`StrategyTemplateEditModal`；新建 `StrategyGalleryPage`、`StrategyDetailPage`、`StrategyCard`、`StartPointModal`。
- **L2（重构）**：拆分 `useStrategyWorkspaceState`（见 §4.6）；`StrategyWorkspacePage` 退化为「组合根」。UI 组件（`WorkspaceCenterColumn`、`RightPanel`、`BacktestPanel` 等）基本保留。
- **L3（复用）**：`useBacktestRunner`、`StrategyTemplateScheduleLaunchModal/Form`、`LiveStrategyPage`、`watch*` SSE、`strategyRuntimeApi` 直接复用。

### 4.2 路由结构（避让 `/strategy/:strategyId`）

```
/strategy                → Gallery（卡片网格，替代 /strategy/library）
/strategy/view/:id       → Detail（资源枢纽；避开被占用的 /strategy/:strategyId）
/strategy/:id/edit       → Workspace（该策略的「编辑模式」）
/strategy/new            → 进入 Workspace 并弹出 StartPointModal（起点选择）
/strategy/live           → Live Monitor（不变）
/strategy/market-tools   → Market Tools（不变）
/strategy/:strategyId    → StrategySharePage（保持不动，放在最后匹配）
```

- 旧 `/strategy/library`、`/strategy/workspace`、`/strategy/templates` 等以 `<Navigate replace>` 重定向到新路径，保证外链与书签不断裂。
- `:id/edit` 与 `:strategyId` 通过「已知静态段优先 + 分享页兜底」的顺序消歧（沿用现有 AppRoutes 的排序策略）。

### 4.3 Gallery 页（结果优先 + 对比）

- **卡片**：名称+标签、描述、迷你权益曲线 sparkline、KPI（胜率/回撤/PF/夏普）、在跑调度数、[详情]/[部署] 按钮。
- **工具栏**：搜索、筛选（全部/我的/预设/公开）、排序（收益/回撤/使用/时间）、[创建]。
- **对比模式**：多选卡片 → 侧栏「Compare」并排看 KPI/权益曲线（复用 marketplace `CompareModal` 模式）。交易员靠横向对比决策。
- **响应式**：桌面 4 列 / 平板 2 列 / 手机 1 列。
- **数据**：单次 `ListStrategyCards`（§4.7），无 N+1；可选 SSE 增量刷新在跑数。

### 4.4 Detail 页（资源枢纽，轻量回测/部署）

- **概览**：描述、完整权益曲线、交易统计、参数说明表。
- **代码**：语法高亮只读；系统模板纯只读；用户模板 [Fork & Edit] → `/strategy/:id/edit`；含 AI 代码解释。
- **回测（轻量）**：选品种/周期/时间 → 用**默认参数**跑一次看结果；**深度调参/迭代**按钮跳 Workspace。**复用 `useBacktestRunner`（唯一实现）**，不重造。
- **部署（轻量）**：选账户 → 复用 `StrategyTemplateScheduleLaunchModal` → 上线。
- **反重复原则**：Detail 只做「一键默认档」；任何需要反复试错的操作一律引导到 Workspace。杜绝 GLM 版「三处各实现一遍回测」。

### 4.5 创建：起点 Modal 取代 4 步向导

**否决向导的理由**：策略创作是**非线性迭代循环**（写→测→改→问 AI→再测）。线性 Stepper（尤其「必须回测通过才能保存」）与该循环对抗，且其 Step2 会克隆一个残缺版编辑器 → 双份编辑器实现分叉。

**替代**：`/strategy/new` 打开 Workspace，首屏弹 `StartPointModal`（1 屏，4 选 1）：
- 空白模板
- 导入 EA（复用 `ImportEAPanel`）
- AI 生成（自然语言描述 → 生成代码，复用 `StrategyChat`/agent）
- Fork 现有（选一个模板复制）

选定后进入**全功能 Workspace**，配合可跳过的**首访引导浮层**（渐进式提示）实现新手友好。友好靠「渐进披露」，不靠「锁步」。

### 4.6 L2 架构：feature-slice 分解（核心补强）

**现状病灶**：`useStrategyWorkspaceState()` 单 hook 返回 `{account, code, templates, backtest, tuning, gate, quickTrade, layout, history, ai}` 十域大对象；域间用一堆 `useEffect` 手动 rewire；任一域变动触发大范围 re-render；无法单测、无法懒加载、无法被 Detail 复用。

**目标**：页面退化为「组合根」，每个功能域是自治 slice（独立 hook + 独立 store 切片 + 自己的 RPC + 自己的组件），跨 slice 通过选择器/事件协调而非传大对象。

```
StrategyWorkspacePage (组合根, ~50 行)
  ├─ <CodeEditorSlice/>    useCodeSlice()        // code/save/validate/draft
  ├─ <BacktestSlice/>      useBacktestRunner()   // L3 复用，设为唯一权威
  ├─ <TuningSlice/>        useTuningSlice()
  ├─ <AIAssistSlice/>      useAISlice()          // StrategyChat/CodeAssist
  ├─ <MarketChartSlice/>   useChartSlice()       // PriceChart
  ├─ <QuickTradeSlice/>    useQuickTradeData()   // 已存在
  └─ <HistoryDrawerSlice/> useHistorySlice()
```

**Store 切分**（Zustand slice-creator 模式，取代单体 `workspaceStore`）：

```ts
// stores/workspace/types.ts
export type CodeSlice = {
  code: string; codeName: string; strategyId: string; dirty: boolean;
  setCode(v: string): void; loadTemplate(id: string): Promise<void>;
};
export type LayoutSlice = {
  centerTab: 'design'|'code'|'backtest'; rightPanelWidth: number;
  setCenterTab(t: LayoutSlice['centerTab']): void; setRightPanelWidth(n: number): void;
};
// ...BacktestSlice / TuningSlice / AiSlice / ...

// stores/workspace/index.ts
export const useWorkspace = create<CodeSlice & LayoutSlice /* & ... */>()(
  persist(
    (...a) => ({ ...createCodeSlice(...a), ...createLayoutSlice(...a) /* ... */ }),
    { name: 'ant-workspace-v6',
      partialize: s => ({ /* 仅布局/导航，不存 code/结果，沿用现规则 */ }) },
  ),
);
// 用法：组件按需订阅单一 slice，避免全量 re-render
// const code = useWorkspace(s => s.code)
```

**跨 slice 协调**：`code` 变化 → backtest slice 通过 `selectCode` 选择器读取并 `resetStatus()`（用订阅/中间件，取代现在散落的 `useEffect` 同步）。

**懒加载**：Gallery/Detail 走 `React.lazy` 且**绝不 import** Monaco/回测引擎；仅 `/strategy/:id/edit` 才加载 Workspace 重包。交易员首屏（Gallery）体积骤降。

> 论证「即使可全推倒，L2 仍应重构而非重写」：上述 feature-slice 组合就是 L2 的**最优终点**；从零重写也只会收敛到同一形态，却要重新推导 37 个组件 + 17 个 hook + 已跑通的 SSE/回测联动，并有概率重新引入已修复缺陷（如 §2.3）。因此在**最优性维度**上，重构支配重写——这不是保守，是更短的最优路径。

### 4.7 后端新增：`ListStrategyCards` 聚合 RPC

> **已实现** — 以下 proto 为实际实现版本（2026-07-27），取代原始设计草稿。实现修正了原草稿中 `repeated double equity_sparkline` 违反项目精度红线的问题，改为 `repeated string`（decimal string）。

```proto
// proto/ant/v1/strategy_template_entity.proto (已实现)
message StrategyCard {
  string id = 1;                    // template id
  string user_id = 2;               // template owner user id
  string name = 3;
  string description = 4;
  repeated string tags = 5;
  bool is_system = 6;
  bool is_public = 7;
  int32 use_count = 8;
  google.protobuf.Timestamp created_at = 9;
  repeated string sparkline = 10;   // equity curve decimal strings (≤32 points)
  string win_rate = 11;             // decimal string
  string max_drawdown = 12;
  string profit_factor = 13;
  string sharpe_ratio = 14;
  int32 running_schedules = 15;
  string backtest_run_id = 16;      // empty if no successful backtest
  bool is_marketplace_published = 17;
}
// proto/ant/v1/strategy_template_requests.proto (已实现)
message ListStrategyCardsRequest {
  string filter = 1;  // "all" | "mine" | "preset"
  string sort = 2;    // "recent" | "return" | "risk" | "usage"
  string search = 3;
  int32 limit = 4;
  int32 offset = 5;
}
message ListStrategyCardsResponse { repeated StrategyCard cards = 1; int32 total = 2; }
// proto/ant/v1/strategy.proto (已实现)
service StrategyService {
  rpc ListStrategyCards(ListStrategyCardsRequest) returns (ListStrategyCardsResponse);
}
```

- 后端在查询侧 join 模板 + 最近回测摘要 + 在跑调度计数，一次返回，消除 N+1。
- Decimal 一律走 string（遵循项目精度红线，禁止 float64 参与金额/指标运算）。
- proto-only：无 REST。

---

## 5. 正面回答「是否全模块推倒」（回应 Owner 态度）

Owner 态度是「最优则推倒亦可」。诚实的最优答案是**分层不同**：

- **L1**：推倒。表格模型本质不适合策略浏览，重做收益高、且数据层已就绪，风险低。**采纳推倒。**
- **L2**：**不重写**——但这是最优性结论，不是风险妥协。理由见 §4.6 论证：重构与重写的**终点相同**（feature-slice），重构路径更短且不丢弃可用资产。若三方讨论中有人能提出「重写才能达到、重构达不到」的更优 L2 终点形态，本决策应被推翻——这是留给 §10 的头号靶点。
- **L3**：复用。回测引擎/SSE/调度是跨模块共享的地基，推倒会外溢到 Live、Marketplace 等，破坏「一任务一作用域」，非局部最优亦非全局最优。

结论：**「全模块推倒」在 L1 是最优，在 L2/L3 不是最优。** 本 ADR 因此选择「L1 重写 + L2 重构 + L3 复用」的**组合最优**，而非「全推倒」的一刀切。

---

## 6. 备选方案对比

| 方案 | 优点 | 缺点 | 结论 |
|------|------|------|------|
| **本 v2（L1重写+L2重构+L3复用）** | 组合最优；用户价值集中在 L1；L2 达最优终点且路径最短 | 需协调三层节奏 | **采纳** |
| GLM 0027 | 宏观拆分正确 | 向导反迭代、Detail 重造回测、路由撞车、未解决上帝 hook | 部分采纳（宏观） |
| 全模块推倒重写 | 「干净」 | L2/L3 收益为负、风险外溢、重复造轮子、重引缺陷 | 否决（非最优） |
| 仅优化表格 | 改动小 | 治标不治本 | 否决 |
| 保持单页 Tab 全整合 | 路由简单 | 消费路径被迫加载创作重包、不可分享、状态耦合 | 否决 |

---

## 7. 后果

### 正面
- 职责清晰：Gallery 浏览 / Detail 深入 / Workspace 创作，且三者依赖单向。
- 交易员结果优先 + 可对比 + 一键部署；链接可分享（`/strategy/view/:id`）。
- Workspace 保留一体化心流，同时代码可测、可懒加载、可被 Detail 复用 slice。
- 修复 §2.3 参数 bug；消除三处回测重复实现。

### 负面
- 新增 3 页/组件（Gallery/Detail/StartPointModal）+ 1 个后端 RPC。
- L2 重构需一次性梳理 slice 边界（可增量、可回退）。

### 中性
- i18n 新增 `strategy.gallery.*`、`strategy.detail.*`；`strategy.library.*` 逐步迁移。
- `workspaceStore` 升版 `v6`（旧 `v5` 持久化键平滑失效，仅影响布局偏好）。

---

## 8. 实施计划（分阶段，每阶段可独立上线）

**Phase A · L1 浏览层（高收益优先）**
1. 后端 `ListStrategyCards` RPC + 查询侧聚合。
2. `StrategyGalleryPage` + `StrategyCard` + Compare。
3. `StrategyDetailPage`（概览/代码/轻量回测/轻量部署，复用 `useBacktestRunner` + `ScheduleLaunchModal`）。
4. 路由改造（避让 `:strategyId` + 旧路径重定向）。
5. 修复 `template`→`templateId` 参数 bug。

**Phase B · 创建去向导化**
6. `StartPointModal`（空白/导入/AI/Fork）接入 `/strategy/new`。
7. 首访引导浮层。

**Phase C · L2 状态重构（内部，UI 不变）**
8. 拆 `useStrategyWorkspaceState` → 各 slice hook。
9. `workspaceStore` → slice-creator（`v6`）。
10. Gallery/Detail/Workspace 路由级懒加载分包。

**Phase D · 清理**
11. 删除 `StrategyLibraryPage`/`StrategyTemplateColumns`/`StrategyTemplateEditModal`。
12. i18n 迁移与废弃键清理。

## 9. 风险与回滚

- **Phase A/B** 与旧页并存，可用 feature flag 切换，随时回滚到 Library。
- **Phase C** 为纯内部重构，按 slice 逐个迁移；每迁一个跑 e2e 冒烟（Playwright 回测流程）即可回退单个 slice。
- 路由重定向保证外链不断裂。

## 10. 待三方讨论的开放问题（靶点）

1. **头号分歧**：L2 到底「重构」还是「重写」？是否存在重写才能达到、feature-slice 重构达不到的更优终点？（本 ADR 主张重构支配重写，欢迎反证）
2. 创建流：起点 Modal（本 v2）vs 4 步向导（GLM）——哪个对真实新手更友好？是否需要 A/B？
3. Detail 的回测/部署「轻量默认档」边界：默认档到什么程度就该强制跳 Workspace？
4. 路由：`/strategy/view/:id` vs 直接复用/合并 `StrategySharePage`（Detail 与公开分享页是否应统一为同一页的登录/未登录两态）？
5. ~~`ListStrategyCards` 的 sparkline 数据来源~~ → **已裁决（§12 A4）**：Phase A = 「最近一次成功回测」；「用户置顶基准回测」延后为后续增强，不进 Phase A。
6. **L3 回测→质量门管线的编排**：维持客户端编排（灵活）还是引入服务端自动链接 / `EvaluationSession`（自动、可审计）？见 §11。

---

## 11. L3 深潜：回测 → 质量门管线的架构评估

本节精确定义 §2.5 中「L3 复用」的范围，并评估这条管线的编排是否最优。**结论：功能块切分正确，但编排层不是最优。** 与 L2 同源——零件对，编排糙。

### 11.1 管线的 5 个功能块（前端 `useBacktestRunner` 编排）

| # | 功能块 | 后端调用 | 交互模型 | 产出 |
|---|--------|----------|----------|------|
| 1 | 代码验证 & 参数/维度抽取 | `codeAssistApi.validateExtended` | 请求/响应 | `parameterEntries` / `sweepDimensions` / `strategyDirectives` |
| 2 | 参数配置 | 本地 | — | 标准参数 + 日期区间 + 策略参数值 |
| 3 | 回测执行 | `StrategyRuntimeService.StartBacktestRun` + `watchBacktestRun` | SSE watch | `backtest_run_id` + metrics + trades + 防篡改快照 |
| 4 | 参数寻优 | `StrategyExperimentService.Submit` | 异步 job | experiment id（内部批量跑回测） |
| 5 | 质量门评估 | `GateService.RunEvaluation`（**服务端 7 门内部管线**） | SSE stream | 逐门 `GateResult` + `GatePipelineSummary` |

数据流：`1 → 2 → 3 → 5`（主线，5 依赖 3 的 `backtest_run_id`）；`4` 为并行支线（吃 1/2 的输出，内部反复触发 3）。

### 11.2 已经做对的（最优部分，须保留）

- **块职责清晰、后端服务边界合理**（`codeAssist` / `strategyRuntime` / `strategyExperiment` / `gate` 四个独立领域服务）。
- **质量门本身是服务端 7 门管线 + SSE 流式**，编排内聚在服务端，客户端只订阅——这一段是最优形态。
- **`backtest_run_id` + 防篡改快照 = 事实上的共享工件**：不仅质量门按它取快照（要求 `SUCCEEDED`），**Marketplace 发布 `PublishStrategyRequest` 也复用同一 `backtest_run_id` 读同一快照**。即「一次可信回测」已是跨 回测/质量门/发布 三处的共享、可审计锚点。**这是既有的良好设计，L3 复用必须保留它。**
- 全推送（块 3/5 SSE，块 4 异步 job），符合 push-first。

### 11.3 不最优的 3 处（编排层，改进目标）

1. **「回测成功 → 质量门」的因果链只活在客户端胶水里。** 服务端不自动串联：前端须等回测 SSE 到达 terminal，再手动 `runGate(backtest_run_id)`。用户在两步之间离开页面则链断。管线语义托管在客户端，脆。
2. **寻优（块 4）是孤岛。** experiment 结果不回流到主线的 `metrics`/gate 视图，也不产出一个可直接被质量门/发布消费的 `backtest_run_id`。「寻优 → 选最优 → 对最优跑质量门 / 发布」这个本该闭环的流程被切成三段，靠人肉搬运。
3. **前端 `useBacktestRunner`（270 行）把编排与纯 UI 状态混在一起**（`panelHeight`/`dragging`/`activeTab`/`modalOpen`），且接缝处类型松（`result: any`、`executionAssumptions: any`、`updateExtractedParams` 里 `typeof pj === 'string'` 再 `JSON.parse`）。

> 注：§2.5 曾说「缺统一评估身份」——经核查需**修正**：run 级身份已由 `backtest_run_id` + 快照提供（且被 gate 与 marketplace 复用）。真正缺的不是身份，而是 **(a) 服务端自动链接** 与 **(b) 寻优结果晋升为可 gate/publish 的 run**。

### 11.4 更优编排（分两层，按收益/成本排序）

**A. 前端重构（低成本，随 L2 Phase C 一起做）**
- 拆 `useBacktestRunner` → `useBacktestParams`（配参）+ `useBacktestRun`（执行+SSE）+ 既有 `useTuning`/`useGateEvaluation`；UI 状态（`panelHeight`/`dragging`/`activeTab`）挪进 store slice。
- runner 退化为薄编排器，对外暴露单一 `pipeline` 对象；接缝处用 proto DTO 取代 `any`/`JSON.parse`。

**B. 协议/后端（高收益，需产品拍板 — §10 Q6）**
- **服务端自动链接**：`StartBacktestRun` 增加可选 `auto_gate` 标志；回测成功后服务端直接续跑质量门，并在**同一条 SSE 流**上先推 metrics、后推逐门结果。消除客户端胶水与断链。
- **寻优结果晋升**：experiment 完成后，把「最优候选」落成一个正式 `backtest_run_id`（带防篡改快照），使其可直接进入质量门 / marketplace 发布，闭合支线。
- **不要过度设计**：块保持可独立调用（保留「用户自主决定何时 gate / 重复 gate / 独立寻优」的灵活性）。B 只是为**常规路径**加服务端自动链接 + 复用既有 run 工件身份，**不是**把整条管线焊死成刚性后端流程。

### 11.5 L3 复用范围裁决

- **保留复用（不动）**：`GateService` 7 门管线、`StrategyRuntimeService` 回测、`backtest_run_id` + 防篡改快照工件、SSE 管线。
- **前端重构（随 Phase C）**：`useBacktestRunner` 拆分。
- **待拍板（§10 Q6）**：服务端 `auto_gate` 自动链接 + 寻优结果晋升为 run。若采纳，属 L3 的**增量增强**，非推倒。

---

## 12. 待 Opus 澄清的架构盲点（2026-07-27 Cascade 补充）

以下 4 个问题在 §11 中未覆盖或未定论，需 Opus 明确后方可进入实施。

### Q1. Marketplace 质量门 vs 7-Gate Pipeline 的关系——全文未提及

§11 通篇只讨论 7-gate pipeline（`ai/gate_pipeline.go`），**完全未提 marketplace 质量门**（`marketplace/quality.go` 中检查 sharpe/drawdown/trades/winRate 阈值）。两套系统独立存在，检查项不同：

| | 7-Gate Pipeline | Marketplace 质量门 |
|---|---|---|
| 文件 | `ai/gate_pipeline.go` | `marketplace/quality.go` |
| 检查项 | Compliance/LookAhead/WalkForward/DeflatedSharpe/MonteCarlo/Paper/Correlation | sharpe/drawdown/totalTrades/winRate 阈值 |
| 用途 | PromoteToLive | PublishStrategy |
| 触发 | 手动 `RunEvaluation` | `Publish` 时自动 |
| 输入 | `backtest_run_id` → equity curve | DB 中的 BacktestSnapshot proto |

**需澄清**：
- 用户通过 7-gate 后想发布到市场，还要再过一遍 marketplace 质量门吗？
- 如果是，两套系统是否应该统一为一条管线？
- 如果否，§11.4 B 的 `auto_gate` 是否应该同时预检 marketplace 质量门，让用户回测完就知道「能不能发布」？

### Q2. `auto_gate` 的 SSE 流合并方式

§11.4 B 说「同一条 SSE 流上先推 metrics、后推逐门结果」，但当前 `WatchBacktestRun`（`BacktestRunUpdate`）和 `RunEvaluation`（`GateEvaluationUpdate`）是两个独立 SSE 流，消息类型完全不同。

**需澄清**：
- 是新建一个统一流类型（如 `BacktestPipelineUpdate`），还是扩展 `BacktestRunUpdate` 加 gate 结果字段？
- 前者更干净但 breaking change，后者兼容但消息类型膨胀。倾向哪个？

### Q3. 寻优结果晋升的具体机制

§11.4 B 说「把最优候选落成一个正式 `backtest_run_id`」，但未定义具体实现路径。

**需澄清**：
- 方案 A：experiment 完成后自动调 `StartBacktestRun` 用最优参数跑一次——简单但多跑一次回测，结果可能因时间窗口不同而有偏差。
- 方案 B：新增 RPC 让 experiment 直接产出 run——复杂但确定性更好，不重复计算。
- 倾向哪个？或有第三方案？

### Q4. `ListStrategyCards` 的 sparkline 数据源——已定还是未定？

§4.7 proto 注释写「来自最近一次成功回测」，但 §10 Q5 又列为开放问题「取最近一次成功回测还是用户置顶的基准回测」。

**需澄清**：
- 这到底定了没有？如果未定，Phase A 实施时按哪个做？
- 如果是「用户置顶的基准回测」，需要新增「置顶回测」的 UI 和后端逻辑，Phase A 范围是否要扩大？

---

## 13. 答复（Cascade，2026-07-27，均经后端代码核查）

**先更正 §12 一处前提**：核查 `marketplace/publish.go` 后确认——当前 `PublishStrategy` **只调 `ValidateBacktestQuality`（marketplace 质量门），不要求过 7-gate**；7-gate 仅把关 `PromoteToLive`。故两者是**两个正交转换（上架 vs 上实盘）上的两道独立闸**，非重复。

### A1（答 Q1）— 不合并，分层

| | 7-Gate Pipeline | Marketplace 质量门 |
|---|---|---|
| 把关转换 | PromoteToLive（上实盘） | PublishStrategy（上架） |
| 语义 | 统计稳健性/防过拟合/前视/相关性，安全导向，**不可豁免**，需 paper 数据 | 配置化指标阈值，**admin 可豁免**，无需 paper |

合并会两头错（逼发布跑 14 天 paper，或让上实盘可豁免）。裁决：
- **保持两服务独立**（不同转换、不同严格度、相反豁免语义）。
- `auto_gate`（§11.4 B）跑的是 **7-gate**；回测刚完成时 Paper/Correlation 两门自然 skip，实际覆盖 5 道统计门 = 即时"过拟合/前视"体检。
- 「过了 7-gate 想发布还要过 marketplace 质量门吗」→ **要**，二者正交；反之发布也不要求 7-gate。
- **优化**：`auto_gate` 可顺带以只读方式调 `ValidateBacktestQuality` 做**发布可行性预览**，但权威判定仍留在 `Publish()`，不强制合并。

### A2（答 Q2）— 扩展 `BacktestRunUpdate`，不新建流

在 `BacktestRunUpdate` 上加**可选字段 `GateEvaluationUpdate gate_update`**，回测成功后在**同一订阅**续推。理由：客户端已持有一条 `watchBacktestRun` 订阅，新建统一流类型等于再开一条流；加 optional 字段**向后兼容**（旧客户端忽略未知字段）、无新 RPC、膨胀极小。→ **选「扩展」。**

### A3（答 Q3）— 定稿：接回已断的链路，不重跑、不新增 RPC

**已核查 experiment worker（`experiment_scoring.go` + `strategy_experiment_repository.go`），事实如下**：
1. 每个候选在 `runSingleBacktest` 里**已经**通过 `backtestRepo.Create` 落成一条真实 `backtest_runs` 行并拿到 `runID`，走常规回测执行路径（快照由同一路径生成）。
2. `strategy_experiment_candidates` 表**本就设计了 `backtest_run_id` 列**（见 `CreateCandidate` 的 INSERT 语句）。
3. **但链路当前是断的**：`runSingleBacktest` 只返回 `*ai.ScoredResult`，把 `runID` 丢弃 → `candidateResult` 不带 runID → 落库候选的 `backtest_run_id` 为空。

**故最优解既非 A（重跑）也非 B（新 RPC），而是「接回断链」**——零重算、零新增快照：
1. `runSingleBacktest` / `backtestAndScore` 把 in-sample 那次的 `runID` 透传进 `candidateResult`（新增字段 `BacktestRunID`）。
2. `CreateCandidate` 落库时填入该 `backtest_run_id`（列已存在，无需迁移）。
3. **晋升 = 纯查表**：`best_candidate_id` → 其 `backtest_run_id` → 即可直接进入质量门 / marketplace 发布。

**GLM 施工时须验证的唯一点**：experiment 子跑（`AccountID = uuid.Nil`，但 `UserID` 已设）在回测完成路径里**确实写入了 `backtest_snapshot`**。若快照写入被某条件（如 `account_id != nil` 或"用户发起"来源）门控，则移除该门控，使实验子跑与用户回测一视同仁地生成快照。`fetchSnapshotByRunID` 只按 `run_id + user_id + status=SUCCEEDED` 校验，`UserID` 已满足，故所有权无碍。

### A4（答 Q4）— Phase A = 「最近一次成功回测」，置顶基准延后

矛盾由 Cascade 造成，现已裁决并同步修正 §10 Q5：**Phase A 用「最近一次成功回测」**（数据现成、零新增 UI/后端）；「用户置顶基准回测」延后为后续增强，**不扩大 Phase A 范围**。

---

## 14. 定稿决议（Cascade，2026-07-27）— GLM 施工依据

§10 的 6 个开放问题在此逐条定稿。**GLM 按本节实施，无需再等三方讨论**；如实施中发现本节与代码事实冲突，回报 Cascade 复核，不得自行降级方案。

> **⚠ 版本对齐（2026-07-24 三方最终共识 + 已实现）**：本 §14/§15 的部分早期规格已被三方最终共识取代并**已实现部署**。凡本 ADR 与 `0027-decision-matrix.md`（§决策汇总 / §F / §G / §H / §I）冲突处，**一律以 decision-matrix 为权威**。已根据最终共识订正的关键点：
> 1. **创建流去 Modal**：不做 `StartPointModal`。创建 = Gallery/Workspace 直接入口（[New]/[Fork]/[AI Generate] → Workspace；Import EA → Drawer）+ 可跳过首访浮层。
> 2. **Detail 收窄**：仅 Overview + Code（只读）+ 一个 [Open in Workspace]，**不做回测/部署 Tab**。
> 3. **L1 复用而非删除**：Gallery = 演进 `StrategyLibraryPage` 骨架（Table→Card Grid），**不新建后删除**（`StrategyLibraryPage.tsx` 已并入 `StrategyGalleryPage.tsx`）。
> 4. **补充**：Gallery 卡片含 [Publish to Market]；侧边栏 "Strategies" → `/strategy`；`ListStrategyCards` 增 `owner_id`/`is_marketplace_published`；访问控制见 §15.10。

### 14.1 §10 开放问题终裁

| # | 问题 | 终裁 | 约束 |
|---|------|------|------|
| 1 | L2 重构 vs 重写 | **重构**（feature-slice） | 终点即最优；禁止推倒 37 组件/17 hooks |
| 2 | 创建流 | **无 Modal**：Gallery/Workspace 直接入口（[New]/[Fork]/[AI Generate] → Workspace；Import EA → Drawer） | 可跳过首访浮层；砍向导也砍 StartPointModal（三方 2026-07-24 共识） |
| 3 | Detail 边界 | **Overview + Code 只读 + 一个 [Open in Workspace]**；不做回测/部署 | 任何回测/调参/部署统一在 Workspace（Detail 不内嵌） |
| 4 | 路由 | Phase A 用 **`/strategy/view/:id`**，与 `StrategySharePage` 保持独立 | 「Detail 与分享页统一」延后评估，不进 Phase A |
| 5 | sparkline 数据源 | **最近一次成功回测**（见 A4） | 置顶基准延后 |
| 6 | 回测→质量门编排 | **采纳服务端 `auto_gate`**（见 A2/§11.4 B），列为 L3 增量增强 | 排在 Phase A-D 之后，见 14.2 |

### 14.2 实施顺序

> **实施状态（2026-07-27 核查）**：Phase A-D + E1/E2 均已完成部署。以下为原始计划，保留作为历史记录。

- **Phase A（✅ 已完成）**：`ListStrategyCards` RPC + Gallery（**演进 `StrategyLibraryPage` 骨架**，Table→Card Grid）+ Detail（Overview+Code只读）+ 路由改造 + 修复 `template→templateId` 参数 bug（§2.3）+ 访问控制（§15.10）。
- **Phase B（✅ 已完成）**：创建入口接线（Gallery [New]/[Fork]/[AI Generate] → Workspace，Import EA Drawer）+ 可跳过首访浮层。**不做 StartPointModal。**
- **Phase C（✅ 已完成）**：L2 feature-slice 重构（拆 `useStrategyWorkspaceState` → 61 行组合根，`workspaceStore` → slice-creator `v6`，路由级懒加载）。`StrategyWorkspacePage` 73 行（≤80 目标）。
- **Phase D（✅ 已完成）**：清理 `StrategyTemplateColumns` / `StrategyTemplateEditModal` + i18n 迁移。**`StrategyLibraryPage` 不删**——其骨架已并入 `StrategyGalleryPage`。
- **L3 增量增强（✅ 已完成）**：
  - E1：`StartBacktestRun` 加 `auto_gate` 标志 + `BacktestRunUpdate` 加可选 `gate_update` 字段（A2）+ marketplace 质量门发布可行性预览（§15.7 E1）。MQL 策略 Compliance/LookAhead 显示 Skipped。
  - E2：接回寻优 `backtest_run_id` 断链 + 验证实验子跑写快照（A3）。`runSingleBacktest` 返回 `runID`，`backtestAndScore` 写入 `candidateResult.BacktestRunID`，`CreateCandidate` 落库。

### 14.3 施工纪律（复用既有代码，勿重造）

- **必复用**：`useBacktestRunner`（唯一回测实现）、`ScheduleLaunchModal`（部署）、`ImportEAPanel`、`StrategyChat`/`CodeAssist`、`PriceChart`、marketplace `CompareModal`（对比）。
- **必保留（L3 不动）**：`GateService` 7 门管线、`StrategyRuntimeService` 回测、`backtest_run_id` + 防篡改快照工件、SSE 管线、`marketplace/quality.go`（与 7-gate 保持独立，见 A1）。
- **红线**：proto-only（无 REST）；Decimal 走 string（禁 float64 参与金额/指标）；旧路径 `<Navigate replace>` 保外链不断；每个 slice 迁移后跑 Playwright 回测冒烟再合并。

---

## 15. 施工规格明细（消除歧义 — 施工方照此实现，不做决策）

本节把所有仍需判断的点定死。**凡本节已给出的，一律照做；凡本节标注「⚠ 须回报 Cascade」的，停并回报，不得自行选择。**

### 15.1 路由改造清单（`frontend/src/routes/AppRoutes.tsx`）

以现有 `mainRoutes` 为基准，做且仅做以下改动：

| 动作 | 路径 | element | 说明 |
|------|------|---------|------|
| 新增 | `strategy` | `<StrategyGalleryPage/>`（lazy） | Gallery 首页 |
| 新增 | `strategy/view/:id` | `<StrategyDetailPage/>`（lazy） | Detail 枢纽 |
| 新增 | `strategy/:id/edit` | `<StrategyWorkspacePage/>`（lazy） | 编辑模式，读 `:id` 自动加载 |
| 新增 | `strategy/new` | `<StrategyWorkspacePage/>`（lazy，空白起点） | 创建入口；**不弹 Modal**（见 §15.5） |
| 改为重定向 | `strategy/library` | `<Navigate to="/strategy" replace/>` | Gallery 取代 |
| 改为重定向 | `strategy/workspace` | `<Navigate to="/strategy/new" replace/>` | 兼容旧书签 |
| 保留不动 | `strategy/live`、`strategy/market-tools`、`strategy/schedules/:id/logs` | 原样 | |
| 保留不动 | `/strategy/:strategyId` → `<StrategySharePage/>` | **必须仍在所有 `strategy/*` 静态段之后匹配** | 避让冲突 |

- `:id/edit` 与 `:strategyId` 消歧：`strategy/:id/edit` 是两段路径，`strategy/:strategyId` 是一段，React Router 精确匹配即可区分；但 `strategy/view/:id`、`strategy/new` 等**必须声明在 `strategy/:strategyId` 之前**。
- ⚠ 若发现 `:id/edit` 被 `:strategyId` 抢匹配，回报 Cascade（不得改用 query param 绕过）。
- **侧边栏导航**：Strategy 菜单组第 2 项由 "Strategy Library"（`/strategy/library`）改为 **"Strategies"（`/strategy`）**；4 项保持不变（Strategies / Workspace / Live / Market Tools）。

### 15.2 `ListStrategyCards` 完整契约

> **已实现**（2026-07-27）— 以下为实际落地版本。原始设计使用 enum 类型和 `repeated double` sparkline，实现中修正为 string filter/sort（更灵活）和 `repeated string` sparkline（遵循精度红线）。

**Proto（实际实现，见 `strategy_template_entity.proto` + `strategy_template_requests.proto`）**：

```proto
message StrategyCard {
  string id = 1;                    // template id
  string user_id = 2;               // template owner user id
  string name = 3;
  string description = 4;
  repeated string tags = 5;
  bool is_system = 6;
  bool is_public = 7;
  int32 use_count = 8;
  google.protobuf.Timestamp created_at = 9;
  repeated string sparkline = 10;   // equity curve decimal strings (≤32 points)
  string win_rate = 11;             // decimal string
  string max_drawdown = 12;
  string profit_factor = 13;
  string sharpe_ratio = 14;
  int32 running_schedules = 15;
  string backtest_run_id = 16;      // empty if no successful backtest
  bool is_marketplace_published = 17;
}
message ListStrategyCardsRequest {
  string filter = 1;  // "all" | "mine" | "preset"
  string sort = 2;    // "recent" | "return" | "risk" | "usage"
  string search = 3;
  int32 limit = 4;
  int32 offset = 5;
}
message ListStrategyCardsResponse { repeated StrategyCard cards = 1; int32 total = 2; }
```

**与原始设计的偏差（均为改进）**：
- `filter`/`sort` 用 `string` 而非 enum — 更灵活，后端按字符串匹配，前端无需导入 enum
- `sparkline` 用 `repeated string` 而非 `repeated double` — 遵循项目精度红线（禁 float64 参与指标运算）
- `user_id` 替代 `owner_id` — 与 DB 列名一致
- `backtest_run_id` 字段（空串=未回测）替代 `has_backtest` bool — 信息量更大，可直接用于发起质量门
- 无 `updated_at_ms`，用 `created_at` Timestamp — 与模板实体一致

**后端实现（`internal/connect/strategy/` 内新增 handler，复用现有 `StrategyService`）**：
- 单查询 join：`strategy_templates` + 每模板「最近一次 `SUCCEEDED` 的 `backtest_runs`」（按 `finished_at DESC LIMIT 1`）+ 在跑 `strategy_schedules`（`status=ACTIVE`）计数。
- KPI 取自该 run 的 `backtest_snapshot`（`BacktestSnapshot` proto）字段映射：`win_rate←WinRate`、`max_drawdown←MaxDrawdown`、`profit_factor←ProfitFactor`（若快照无该字段则空串）、`sharpe←SharpeRatio`。**全部以 decimal string 原样透传，禁止 float64 运算。**
- `equity_sparkline`：取该 run 已计算的权益曲线序列（与 `AccountAnalytics`/`EquityPoint` 同源），**服务端等距降采样到 ≤32 点**；无曲线 → 返回空数组且 `has_backtest` 仍可为 true（卡片只画 KPI）。
- `sort`：`RETURN`/`DRAWDOWN` 按快照对应指标排序（NULL 值排最后）；`USAGE` 按 `running_schedules DESC`；`UPDATED` 按 `updated_at_ms DESC`。
- ⚠ 若 `backtest_runs` 未存权益曲线序列（只存 `proto_response`/`backtest_snapshot`），回报 Cascade 定 sparkline 来源，勿自行造数据。

### 15.3 Gallery 页规格（`StrategyGalleryPage.tsx` + `StrategyCard.tsx`）

- 数据：`useQuery(['strategyCards', filter, sort, search], listStrategyCards)`。搜索输入 300ms debounce。
- 卡片内容与顺序：名称（粗体）→ tags（≤3 个 `Tag`，多余显示 `+N`）→ sparkline（`has_backtest=false` 时替换为灰底"未回测"占位）→ 一行 KPI（胜率 / 回撤 / PF / 夏普，数值缺失显示 `—`）→ 底部 `running_schedules>0` 时绿色徽标。
- **卡片 actions（按 §15.10 访问控制矩阵裁剪按钮集）**：`[详情]`（`navigate('/strategy/view/'+id)`，所有可见卡片都有）、`[部署]`（`ScheduleLaunchModal`，仅 `isOwner || (已发布 && 已购买)`）、`[Fork]`（仅 `isOwner || isSystem`）、`[Publish to Market]`/`[Unpublish]`（仅 `isOwner` 非系统；复用 `PublishToMarketModal`，按 `is_marketplace_published` 切换）、`[Delete]`（仅 `isOwner`）。系统模板仅 `[详情]`+`[Fork]`。
- 工具栏：`Input`（搜索）+ `Segmented`（filter 四值）+ `Select`（sort 四值）+ 右侧 `[创建]`（`navigate('/strategy/new')`）。
- 响应式：`Row gutter=[16,16]`，`Col` `xs=24 sm=12 lg=8 xxl=6`（手机1/平板2/桌面3/大屏4）。
- 状态：loading→骨架卡 ×8；empty→复用 `strategy.gallery.empty`；error→重试按钮。

### 15.4 Detail 页规格（`StrategyDetailPage.tsx`）

**三方最终共识：Detail 只做「浏览深入」，不做回测/部署**（回测/调参/部署统一在 Workspace）。

- 数据：`getTemplate(id)` + 该模板最近成功 run 的 metrics（复用 `strategyRuntimeApi`，只读展示）。
- 权限：从 `useAuthStore` 取当前用户 → `isOwner = template.userId === currentUserId`；`canEdit = isOwner || isSystem`（详见 §15.10）。
- **两 Tab（`概览 / 代码`）**：
  - **概览**：描述 + 完整权益曲线（复用现有权益曲线组件）+ 交易统计表 + 参数说明表（只读，来自 `template.parameters`）。
  - **代码**：`StrategyCodeEditor` 只读，含 `CodeExplainPanel`。**Code Tab 仅 `isOwner || isSystem` 时渲染**（别人的已发布模板隐藏 Code，代码不出平台）。
- **头部按钮**（单一动作入口）：`isSystem` → `[Fork & Edit]`（→ `/strategy/:id/edit`）；`isOwner` → `[Open in Workspace]`（→ `/strategy/:id/edit`）；别人的已发布模板 → 无编辑/Fork 入口。
- Detail **无回测面板、无部署表单、无可编辑参数**；一切迭代动作跳 Workspace。

### 15.5 创建入口规格（**无 Modal** — 三方最终共识）

**不实现 `StartPointModal`。** 创建是 Gallery/Workspace 的直接入口，各入口进入 Workspace 后设置对应初始态：

| 入口 | 位置 | 行为 |
|------|------|------|
| `[New]` | Gallery 工具栏 / 侧边栏 | `navigate('/strategy/new')` → Workspace 空白（`code=''`，落在 code tab） |
| `[Fork]` | Gallery 卡片（`isOwner\|\|isSystem`） | 复制该模板代码进 Workspace，`strategyId` 置空（另存为新模板） |
| `[AI Generate]` | Gallery 工具栏 / Workspace | 进 Workspace 并展开右侧 `StrategyChat`，聚焦输入框 |
| Import EA | Workspace 内 | 打开 `ImportEAPanel` **Drawer**（非独立页/Modal），导入结果写入 `code` |

- 首访引导浮层（复用现有 `WorkspaceTour`）在首次进入 Workspace 时触发，可跳过并记 localStorage `alphaforge_ws_tour_done`。

### 15.6 feature-slice 边界映射（Phase C）

拆 `useStrategyWorkspaceState`（现返回 10 域大对象）为独立 slice，映射如下，**行为不变、仅搬迁**：

| 新 slice hook | 吸收现有 | 归属 store slice（`workspace` v6） |
|---------------|----------|-----------------------------------|
| `useAccountSlice`（已存在） | account/symbol/timeframe/accountInfo | `accountId/symbol/timeframe` |
| `useCodeSlice` | `useStrategyCode` + save/draft | `code/codeName/strategyId`（不持久化 code，同现规则） |
| `useBacktestRun` | `useBacktestRunner` 的 run/status/metrics/trades 部分 | — |
| `useBacktestParams` | `useBacktestRunner` 的 params/date/strategyParams 部分 | — |
| `useTuning`（已存在）/`useGateEvaluation`（已存在） | 原样 | — |
| `useQuickTradeData`（已存在）/`useHistoryState`（已存在）/`useAIWorkflow`（已存在） | 原样 | — |
| `useLayoutSlice` | 现 `workspaceStore` 全部 UI 字段 + `useBacktestRunner` 的 `panelHeight/dragging/activeTab` | `centerTab/rightTab/*Collapsed/rightPanelWidth/panelHeight` |

- `StrategyWorkspacePage` 退化为组合根（≤80 行），按 §4.6 组装。跨 slice 用选择器读取，删除现有 `useWorkspaceEffects` 里的手动 rewire `useEffect`，改为 slice 内 `subscribe`。
- 接缝去 `any`：`handleValidationResult`/`executionAssumptions` 用 `ValidateExtendedResult` 与对应 proto 类型；`updateExtractedParams` 只接受 `ExtractedParam[]`，删除 `JSON.parse(string)` 分支（调用方统一传数组）。

### 15.7 L3 增量增强 proto 与代码改动点

**E1 — `auto_gate`（`proto/ant/v1/backtest_run_*.proto` + gate 复用 + marketplace 质量门预览）**
- `StartBacktestRunRequest` 加 `bool auto_gate = N;`（下一个可用号）。
- `BacktestRunUpdate` 加两个可选字段（向后兼容，旧客户端忽略）：
  - `optional GateEvaluationUpdate gate_update = N;` — 7-gate 逐门结果。
  - `optional MarketplaceQualityPreview quality_preview = N;` — marketplace 质量门发布可行性预览。
- 新增 proto message：
  ```proto
  message MarketplaceQualityPreview {
    bool publishable = 1;              // true = 当前 snapshot 能通过 marketplace 质量门
    repeated QualityViolation violations = 2;  // 不通过的原因（空=通过）
  }
  message QualityViolation {
    string metric = 1;    // e.g. "sharpe_ratio"
    string actual = 2;    // e.g. "0.3"
    string threshold = 3; // e.g. "1.0"
  }
  ```
- 服务端：run 到达 `SUCCEEDED` 且 `auto_gate=true` 时：
  1. 从该 run 的权益曲线**推导 `DailyReturns`**，喂入 `ai.Pipeline`，逐门结果经 `gate_update` 续推到同一流。
  2. 从该 run 的 `backtest_snapshot`（BYTEA）unmarshal 出 `BacktestSnapshot` proto，调 `marketplace.ValidateBacktestQuality`（只读，不写 DB），结果经 `quality_preview` 续推到同一流。
  - 这样用户回测完立刻知道：7-gate 过没过 + 能不能发布，无需手动尝试 Publish。
  - 权威判定仍在 `Publish()`，预览仅做展示。
- **⚠ 关键规则（消歧）——无 DSL 表达式时的 gate 处理**：本模块为 MQL/代码策略，无 DSL `expression`。改 `ai/gate_pipeline.go`：当 `input.Expression == ""` 时，`evalCompliance` 与 `evalLookAhead` 返回 `Skipped:true`（**跳过而非失败**，与 `evalPaper`/`evalCorrelation` 的 skip 语义一致）。故 `auto_gate` 对 MQL 策略实际评估 WalkForward/DeflatedSharpe/MonteCarlo 三门，其余 skip。此改动不影响 agent 生成的带表达式策略（仍全量评估）。
- 前端 `useGateEvaluation` 增加消费 `gate_update` 的分支；不再需要回测完成后手动 `runGate`（手动入口保留供重复评估）。前端在回测结果面板展示 `quality_preview`（通过→绿色「可发布」徽标；不通过→列出 violations）。

**E2 — 寻优 run 断链接回（见 A3）**
- `experiment_scoring.go`：`runSingleBacktest` 返回值增加 `runID`；`backtestAndScore` 写入 `candidateResult.BacktestRunID`。
- `strategy_experiment_worker.go` 落库处：把 `candidateResult.BacktestRunID` 传入 `CreateCandidate`（列已存在）。
- 验证实验子跑写 `backtest_snapshot`（见 A3 末尾）；若未写则移除门控。
- 晋升读取：`best_candidate_id → candidate.backtest_run_id`，前端在寻优结果面板对该 run 提供 `[跑质量门]`/`[发布]` 按钮。

### 15.8 i18n

- 新增 key 组：`strategy.gallery.*`（title/搜索占位/筛选四值/排序四值/empty/未回测/创建；卡片 actions：详情·部署·Fork·发布·下架·删除）、`strategy.detail.*`（两 Tab 名 概览/代码、Fork&Edit、Open in Workspace）。**无 `strategy.start.*`**（创建无 Modal）。
- 走现有 textproto + gen keys 流程（`gen/ant/v1/i18n/`）；旧 `strategy.library.*` 保留至 Phase D 再清。**新增 key 必须中英双语齐全**，不得留 `defaultValue` 兜底作为最终态。

### 15.9 各 Phase 验收标准（Definition of Done）

- **Phase A DoD**：`/strategy` 显示卡片（Gallery 由 LibraryPage 骨架演进）；`/strategy/view/:id` 可分享直达；`/strategy/library`、`/strategy/workspace` 重定向生效；侧边栏第 2 项为 "Strategies"；`template→templateId` bug 修复；访问控制生效（§15.10）；`go build ./...` + `check-file-lines --strict` 通过；Playwright 冒烟（登录→Gallery→Detail）绿。
- **Phase B DoD**：Gallery [New]/[Fork]/[AI Generate] 进 Workspace 各自初始态正确；Import EA Drawer 可用；引导浮层可跳过并记忆。**确认无 StartPointModal。**
- **Phase C DoD**：`useStrategyWorkspaceState` 删除，各 slice 独立；`StrategyWorkspacePage ≤80 行`；Gallery/Detail 不打包 Monaco（构建产物验证 chunk 分离）；Workspace 全功能回归（Playwright 覆盖回测/调参/质量门/AI/保存）。
- **Phase D DoD**：`StrategyTemplateColumns`/`StrategyTemplateEditModal` 删除（**保留 `StrategyGalleryPage`**）；`strategy.library.*` 废弃 key 清理；无死引用（`tsc` + 构建通过）。
- **E1/E2 DoD**：`auto_gate=true` 的回测在同一流推出 7-gate 结果 + marketplace 质量门预览（`quality_preview`）；MQL 策略 Compliance/LookAhead 显示 Skipped 而非 Failed；前端展示「可发布/不可发布」及 violations；寻优候选 `backtest_run_id` 落库非空，胜出候选可直接发起质量门与发布。

### 15.10 策略代码访问控制（强制 — “代码不出平台”）

三方审计（matrix §G）发现：Detail Code Tab / [Edit] / [Deploy] / `:id/edit` 若不鉴权，任何登录用户可查看/编辑/部署他人策略代码。**本节为强制需求，前后端纵深防御。**

**访问控制矩阵**（`isOwner = template.userId === currentUserId`）：

| 操作 | 系统模板 | 我的模板 | 别人已发布 | 别人未发布 |
|------|---------|---------|-----------|-----------|
| Gallery 可见 | ✅ 所有人 | ✅ 仅我（“我的”筛） | ✅ 所有人 | ❌ 不出现 |
| 看 Overview | ✅ | ✅ 仅我 | ✅ | — |
| 看 Code Tab | ✅（开源） | ✅ 仅我 | ❌ **隐藏 Code Tab** | — |
| [Edit]/[Open in WS] | ❌（→[Fork & Edit]） | ✅ 仅我 | ❌ 隐藏 | — |
| [Fork] | ✅ | ✅ | ❌ 已发布不可 Fork | — |
| [Deploy] | ✅ | ✅ | ✅ **仅购买者** | — |
| [Publish] | ❌ | ✅ 仅我 | — | — |
| [Delete] | ❌ | ✅ 仅我 | ❌ | — |
| Workspace 编辑 | ❌ | ✅ 仅我 | ❌ | — |

**前端实现**：
- `StrategyDetailPage`：`useAuthStore` 取 userId；`isOwner`；Code Tab 仅 `isOwner || isSystem` 渲染；头部按钮 `isSystem→[Fork & Edit]` / `isOwner→[Open in Workspace]` / 其他→无。
- `StrategyCard`：`[Deploy]` 仅 `isOwner || (isPublished && hasPurchased)`；`[Fork]` 仅 `isOwner || isSystem`（按 §15.3）。
- `StrategyWorkspacePage`：加载模板时校验 `isOwner || isSystem`，否则拒加载并 redirect 回 Gallery。

**后端纵深防御（不可省）**：
- `getTemplate`：非 owner 且非公开 → 错误；`updateTemplate`/`deleteTemplate`：非 owner（delete 含非 admin）→ `PermissionDenied`。

> 实现状态（matrix §G 已标记已实现）：`StrategyDetailPage.tsx:28-31/86/158`、`StrategyCard.tsx:94-96`。GLM 新增代码需与此矩阵一致；若发现不一致以本节为准。
