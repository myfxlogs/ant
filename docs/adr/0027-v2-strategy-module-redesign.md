# ADR-0027 v2 · 策略模块前端重构（修订版 — Cascade 提案）

- **状态**：Proposed（与 `0027-strategy-gallery-redesign.md`「GLM 版」并列，供三方评审）
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

```proto
// proto/ant/v1/strategy.proto (新增)
message StrategyCard {
  string id = 1;
  string name = 2;
  repeated string tags = 3;
  bool is_system = 4;
  bool is_public = 5;
  // 反范式化摘要（来自最近一次成功回测）
  repeated double equity_sparkline = 6; // 归一化点，画迷你曲线
  string win_rate = 7;    // decimal string
  string max_drawdown = 8;
  string profit_factor = 9;
  string sharpe = 10;
  int32 running_schedules = 11;
  int64 updated_at_ms = 12;
}
message ListStrategyCardsRequest { string filter = 1; string sort = 2; string search = 3; }
message ListStrategyCardsResponse { repeated StrategyCard cards = 1; }

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
5. `ListStrategyCards` 的 sparkline 数据来源：取「最近一次成功回测」还是「用户置顶的基准回测」？
