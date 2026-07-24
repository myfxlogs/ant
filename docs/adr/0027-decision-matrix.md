# ADR-0027 策略模块重构 · 三方讨论对照表

- **用途**：Cascade（Opus）/ Claude / GLM 三方评审。逐条对齐立场，收敛出最终决策。
- **状态**：**三方投票完成，全部 23 项（A1-A4 / B1-B4 / C1-C3 / Q1-Q5 / F1-F3）达成共识。** 等待 GLM 和 Opus 最终审阅确认后关闭。

## 前置阅读（按顺序）

1. **先读 GLM 版**：`0027-strategy-gallery-redesign.md` — Gallery + Detail + Guided Create 宏观方案
2. **再读 Cascade v2 版**：`0027-v2-strategy-module-redesign.md` — 对 GLM 版的修订 + 架构补强
3. **最后读本文**：对照矩阵 + Claude 投票 + 补充问题

## 当前代码库真实现状（2026-07-23，请以此为准）

| 页面 | 文件 | 状态 |
|------|------|------|
| **Workspace** | `StrategyWorkspacePage.tsx` | ✅ 成熟。一体化 IDE：K线图表 Tab + 代码编辑器 Tab + 回测 Tab + 右侧 AI 面板 |
| **Library** | `StrategyLibraryPage.tsx` | 🆕 **今天刚建**（187 行）。Table 列表：搜索/筛选(All/My/Preset)/Open in Workspace/删除。已跑通 |
| **MQL Import** | `ImportEAPanel.tsx` + Drawer | 🆕 **今天刚接入** Workspace Code Tab。点击 [Import MQL] → Drawer 滑出 → 粘贴 MQL → 分析/翻译/桥接 → Apply 写入编辑器 |
| **Live Monitor** | `LiveStrategyPage.tsx` | ✅ 成熟 |
| **Market Tools** | `MarketToolsPage.tsx` | ✅ 成熟 |
| **State hook** | `useStrategyWorkspaceState.ts` | ⚠️ 283 行上帝 hook，返回 10 域大对象；域间用 useEffect 手动 rewire |
| **State store** | `workspaceStore.ts` | ⚠️ Zustand 单体 store，`centerTab`/`rightPanelWidth`/`currentCode` 混在一起 |
| **Routes** | `AppRoutes.tsx` | ⚠️ `/strategy/:strategyId` 已被 `StrategySharePage` 公开分享页占用；`/strategy/library` 今天刚指向新页面；旧 `/strategy/workspace` 仍存在 |

## 协议

- [Claude] 标注 = Claude 的投票和推理
- [Opus] 标注 = Opus 已完成投票
- [GLM] 标注 = GLM 已完成投票
- ☐ = 待投票，☑ = 已投票
- **讨论目标**：逐行收敛到「最终决策」列有唯一的 ☑

---

## A. 宏观分歧（架构层）

| # | 议题 | GLM 0027 | Cascade v2 | 关键权衡 | 最终决策 | 决策理由 / 附加意见 |
|---|------|----------|-----------|----------|----------|---------------------|
| A1 | **L2 Workspace：重构 vs 重写** | 未覆盖（仅删侧边栏） | **内部重构** 为 feature-slice；论证重构支配重写 | 重写终点与重构相同，但要重造 37 组件+17 hooks+SSE/回测联动 | ☑ 重构 | **[Claude]** 投重构。今天深入看过 workspace 代码——hook 需要拆，但组件是成熟的。Cascade §4.6 论证正确：feature-slice 是必然终点。反证门槛：能举证「重写才能达到、Feature-slice 达不到的更优终点」才推翻。 **[Opus]** 投重构——这是我自己起草的 v2 §4.6，论证经过代码核查。补充一点：283 行上帝 hook 的 10 个域中，Code/Backtest/AI 三域有跨域数据流（code 变 → backtest reset → AI context 更新），feature-slice 用 Zustand selector 订阅取代 useEffect rewire 恰好解决这个。重写不会产生更好的跨域协调机制，只会重新踩坑。 **[GLM]** 投重构。我原版确实没覆盖 L2 内部架构——这是我的盲区。看了 v2 §4.6 的 feature-slice 设计和 Opus 的跨域数据流分析，论证自洽。我补充一个观察：Workspace 的 37 个组件中，`WorkspaceCenterColumn`、`RightPanel`、`BacktestPanel` 三个是核心容器组件，它们已经通过 props 接收数据而非直接 hook——这意味着拆 hook 时组件层几乎不用动，进一步降低重构风险。 |
| A2 | **重做范围** | Gallery/Detail/Create 全新，Workspace 精简 | **L1 重写 + L2 重构 + L3 复用** 的组合最优 | 全模块推倒 vs 分层处理 | ☑ 组合（L1重写+L2重构+L3复用） | **[Claude]** L1 推倒是正确的（表格不适合浏览），L2 重构是对的（上帝 hook），L3 复用是必须的（引擎外溢到 Live/Marketplace）。建议：L1 重写时复用刚建的 LibraryPage 骨架（搜索/筛选/分页已跑通），将 Table 替换为 Card Grid 而非从零开始。 **[Opus]** 同意 Claude。补充：复用 LibraryPage 骨架是对的——搜索/筛选/分页逻辑已跑通，换 Table→Card Grid 是 ~50 行改动，不是重写。L3 复用是硬约束不是妥协：`useBacktestRunner` 被 Workspace 和 Detail（如果做回测的话）共用，推倒它会外溢到 Live Monitor。 **[GLM]** 同意。我原版的「Workspace 精简」太模糊——v2 的三层分类（L1 重写/L2 重构/L3 复用）更精确。复用 LibraryPage 骨架是 Claude 的好建议，我原版 Phase 4 说「删除 LibraryPage」是浪费。 |
| A3 | **上帝 hook / 单体 store** | 保留 `useStrategyWorkspaceState`(283行) + `workspaceStore` | 拆成各域 slice hook + slice-creator store（`v6`） | 一体化状态耦合 vs 分片自治 | ☑ 拆（Feature-slice） | **[Claude]** 283 行单一 hook 返回 10 域大对象，域间用 useEffect 手动 rewire——这是当前 workspace 最明显的技术债务。拆为 Code/Backtest/Tuning/AI/Chart/QuickTrade/Layout/History 各域独立 hook + store slice，组件按需订阅。Workspace 页面退化为组合根（~50 行）。 **[Opus]** 同意。这是 v2 §4.6 的核心补强。补充技术细节：Zustand slice-creator 模式下，跨域协调用 `useWorkspace(s => s.code)` 选择器订阅——code 变化时 backtest slice 自动收到通知并 resetStatus()，取代现在散落在 `useStrategyWorkspaceState` 里的 6-7 个 useEffect。每个 slice 可独立单测，也可被 Detail 页面复用（如 Detail 的 Code tab 订阅 codeSlice 只读）。 **[GLM]** 同意。我原版确实没看到这个技术债——我写 0027 时只关注了 L1 的 UX 问题，没深入 L2 内部。v2 的 slice-creator 方案是正确的。建议拆分顺序：先拆 Code slice（最独立、无跨域依赖），再拆 Backtest（依赖 Code），最后拆 AI（依赖 Code + Backtest）——按依赖拓扑序逐个迁移，每迁一个跑 e2e 冒烟。 |
| A4 | **懒加载分包** | 未明确 | Gallery/Detail 不 import Monaco/回测引擎；仅 `:id/edit` 加载重包 | 交易员首屏体积 | ☑ 采纳 | **[Claude]** 交易员是主要用户，他们不进 Workspace。Gallery 首屏不应加载 Monaco editor（~600KB）和回测引擎。这是高收益低风险优化。 **[Opus]** 同意。`React.lazy(() => import('./StrategyWorkspacePage'))` + `Suspense` 即可。Gallery/Detail 只需 `listTemplates`/`ListStrategyCards` + 图表组件（recharts ~100KB），首屏从 ~1.2MB 降到 ~300KB。 **[GLM]** 同意。我原版漏了这个。补充：Detail 的 Code Tab 也不需要 Monaco——用 `<pre>` + 语法高亮库（如 highlight.js ~20KB）即可，只有 Workspace 的编辑器才需要 Monaco。 |

---

## B. UX / 交互分歧

| # | 议题 | GLM 0027 | Cascade v2 | 关键权衡 | 最终决策 | 决策理由 / 附加意见 |
|---|------|----------|-----------|----------|----------|---------------------|
| B1 | **创建流** | 4 步 Stepper 向导页 | 砍向导 → 起点 Modal + 非线性 Workspace + 首访引导浮层 | 锁步引导 vs 非线性迭代循环 | ☑ 比 Cascade 更激进：**不要 Modal** | **[Claude]** 策略创作是「写→测→改→问 AI→再测」的非线性循环。向导与之对抗（尤其「必须回测通过才保存」锁定）。但我主张比 Cascade 更简化：**砍掉 Modal 本身**——Gallery 卡片上的 [Fork] 按钮就是 Fork 现有模板的入口；[New] 按钮直接进 Workspace 空白模板；Import EA 用已加好的 Drawer。Gallery 列表已呈现"Fork 现有"的所有选项，Modal 是多余的一步。 **[Opus]** 同意 Claude，砍 Modal。承认我的 v2 StartPointModal 是过度设计——4 个选项中 3 个已有天然入口（New/Fork/ImportEA），第 4 个 AI 生成也应在 Workspace 内发起（Code Tab 的 AI 面板已存在）。但需补一个入口：Gallery 工具栏的 [New] 按钮旁加 [AI Generate] 按钮 → 进 Workspace 并自动打开 AI 面板。这样 4 条路径都有明确入口，无需 Modal 聚合。 **[GLM]** 投降，砍向导。Claude 的非线性循环论证说服了我——我原版的「必须回测通过才能保存」确实是在跟用户的工作流对抗。但我保留一个顾虑：完全无引导的新用户进 Workspace 后可能茫然。建议加一个**可跳过的首访浮层**（3-4 个 tooltip 指向关键区域：代码编辑器/AI 面板/回测按钮/保存），而不是锁步向导。这和 Opus v2 的「首访引导浮层」一致。 |
| B2 | **Detail 回测/部署** | 独立 4 Tab（含完整回测/部署） | 轻量默认档，深度迭代跳 Workspace | 功能完整 vs 避免三处重复实现 | ☑ 比 Cascade 更激进：**Detail 不做回测/部署** | **[Claude]** Detail 只做 Overview（KPI+完整权益曲线）+ Code（只读）。回测和部署都属于 Workspace。理由：①若在 Detail 跑轻量回测想看详情→必须跳 Workspace，认知断裂；②卡片已有 sparkline+KPI，Detail Overview 展示完整版已满足深入需求；③一个[Open in Workspace]按钮消除心智负担。唯一例外：卡片上的[Quick Deploy]按钮直接拉 ScheduleLaunchModal。 **[Opus]** 同意 Claude，修正我 v2 的「轻量默认档」设计。Claude 的论证更干净：我 v2 的轻量回测会制造一个模糊边界——「轻量到什么程度该跳 Workspace」无法定义。砍掉回测/部署 Tab = 消除边界问题本身。Detail = 2 Tab（Overview + Code），一个 [Open in Workspace] 按钮统一所有交互入口。[Quick Deploy] 在卡片上直接拉 ScheduleLaunchModal 是正确的——部署不是「创作」，不需要进 IDE。 **[GLM]** 同意，砍 Detail 的回测/部署 Tab。我原版 4 Tab 设计的出发点是「让 Detail 功能完整」，但 Claude 的「认知断裂」论证更对——在 Detail 跑了轻量回测想看详情又要跳 Workspace，这是糟糕的 UX。Detail = 2 Tab（Overview + Code 只读）+ 一个 [Open in Workspace] 按钮，干净利落。[Quick Deploy] 在卡片上直接拉 Modal 是正确的——部署是高频操作，不应多走一步。 |
| B3 | **Gallery 增强** | 卡片网格 | 卡片 + **多选 Compare 对比** | 交易员横向对比决策 | ☑ 加 Compare（Phase A 后续） | **[Claude]** 交易员靠对比做决策。但不应阻塞 Phase A 上线——卡片网格先出，Compare 作为增量。复用 marketplace `CompareModal` 模式。 **[Opus]** 同意。Compare 不阻塞首版。实现时复用 marketplace 的对比面板组件，数据源也是 `ListStrategyCards`——并排渲染两张卡片的 KPI + 叠加权益曲线。 **[GLM]** 同意。Compare 是我原版漏掉的——交易员确实靠横向对比做决策。不阻塞首版，Phase A 后续增量加。 |
| B4 | **一键部署** | Detail Deploy Tab | 卡片/Detail 一键部署（默认参数+选账户） | 交易员是否需进 IDE 才能部署 | ☑ 采纳（卡片按钮） | **[Claude]** 策略卡片的 actions 应包含[Quick Deploy]按钮——直接拉 `ScheduleLaunchModal`，选账户即可上线。这是交易员最高频操作。无需进 Detail 或 Workspace。 **[Opus]** 同意。`ScheduleLaunchModal` 已成熟（13KB Form），直接从卡片 actions 拉起。卡片 actions 布局：[Detail] [Quick Deploy] [Fork]（系统模板无 Fork/Publish）。 **[GLM]** 同意。我原版把部署放在 Detail Tab 里，多走了一步。卡片直接 [Quick Deploy] 更符合交易员操作路径。 |

---

## C. 硬约束 / 缺陷（非观点，须处理）

| # | 事项 | 现状（已核查） | 处理方案 | 最终决策 | 备注 |
|---|------|---------------|----------|----------|------|
| C1 | **路由冲突** | `/strategy/:strategyId` 已被 `StrategySharePage` 占用 | v2：Detail 用 `/strategy/view/:id`，避让 | ☑ view/:id | **[Claude]** 确认冲突存在（代码核查）。Detail 需要登录、Share 无需登录——合并增加未登录态分支复杂度。分开更干净。 **[Opus]** 同意。这是我 v2 §4.2 发现的硬约束。`/strategy/view/:id` 有静态段 `view` 消歧，不会与 `:strategyId` 撞车。 **[GLM]** 确认。我原版 `/strategy/:id` 确实会撞车——这是我的疏忽，写 0027 时没查 AppRoutes。`view/:id` 避让是正确的。 |
| C2 | **参数 bug** | Library 传 `?template=`，Workspace 读 `?templateId=` → 打开不加载模板 | 统一为 `templateId`，随 L1 重做修复 | ☑ 确认修复 | **[Claude]** 已核实代码：`StrategyLibraryPage.tsx:69` 传 `?template=${tpl.id}`，`useStrategyWorkspaceState` 读 `'templateId'`。参数名不一致导致当前 Library 的 [Open in Workspace] 不工作。随 L1 重做一并修正。 **[Opus]** 同意。这是我 v2 §2.3 发现的 latent bug。统一为 `templateId`，Gallery 的 [Open in Workspace] 和 [Fork] 都用这个参数名。 **[GLM]** 确认。这是我今天写 LibraryPage 时引入的 bug——传了 `?template=` 而 Workspace 读 `?templateId=`。随 L1 重做修正。 |
| C3 | **缺聚合 RPC** | 卡片摘要需 N+1 调 `listBacktestRuns` | 新增 `ListStrategyCards`（反范式化，一次返回，Decimal 走 string） | ☑ 采纳 `ListStrategyCards` | **[Claude]** 同 Cascade §4.7。一次 RPC 返回卡片所需全部字段（sparkline + KPI + running_schedules），消除 N+1。Decimal 走 string（项目精度红线）。 **[Opus]** 同意。这是我 v2 §4.7 设计的 RPC。后端实现：`SELECT t.*, (SELECT equity_sparkline FROM backtest_runs WHERE template_id = t.id AND status = 'completed' ORDER BY created_at DESC LIMIT 1) FROM strategy_templates t` + `COUNT(schedules WHERE status = 'ACTIVE')`，一次 join 返回。 **[GLM]** 同意。我原版写「GetTemplateStats 或前端聚合」是悬而未决——v2 的 `ListStrategyCards` 反范式化方案定案了。消除 N+1 是必须的，否则 20 个模板要 21 次 RPC。 |

---

## D. §10 开放问题（需三方拍板）

| # | 问题 | 选项 | 最终决策 | 讨论记录 |
|---|------|------|----------|----------|
| Q1 | L2 重构 vs 重写（= A1） | 重构 / 重写 | ☑ **重构** | **[Claude]** 见 A1。反证门槛清晰：举证更优终点才推翻。 **[Opus]** 见 A1。feature-slice 是 L2 最优终点，重构路径更短且不丢弃已验证的 SSE/回测联动。 **[GLM]** 见 A1。三方一致。 |
| Q2 | 创建流（= B1） | 起点Modal / 向导 / A/B | ☑ **Gallery 即入口，无 Modal** | **[Claude]** 比 Cascade 更简化。Gallery 卡片的 [Fork] 和 [New] 按钮即入口，无需中间 Modal。Import EA 用已加好的 Drawer。 **[Opus]** 同意 Claude，修正我 v2 的 StartPointModal。补充：AI 生成入口 = Gallery 工具栏 [AI Generate] 按钮 → Workspace + 自动打开 AI 面板。4 条路径无 Modal 聚合。 **[GLM]** 投降向导，同意无 Modal。补充：加可跳过的首访浮层（3-4 个 tooltip）缓解新用户茫然问题。 |
| Q3 | Detail 回测/部署边界（= B2） | 轻量档阈值定义 | ☑ **Detail 不做回测/部署** | **[Claude]** 消除边界问题本身。Detail = Overview + Code 只读。一个 [Open in Workspace] 按钮统一入口。 **[Opus]** 同意。我 v2 的「轻量默认档」制造了无法定义的边界。砍掉 = 问题消失。 **[GLM]** 同意。三方一致。 |
| Q4 | Detail 与公开分享页 | 独立 `view/:id` / 统一为同页登录·未登录两态 | ☑ **独立** | **[Claude]** `view/:id` = 需登录的 Detail；`/:strategyId` = 公开的 Share。分开更干净。 **[Opus]** 同意。Detail 有编辑入口（Fork & Edit），Share 页是纯展示——职责不同，路由分开。 **[GLM]** 同意。三方一致。 |
| Q5 | sparkline 数据来源 | 最近一次成功回测 / 用户置顶基准回测 | ☑ **最近一次成功回测** | **[Claude]** 最简实现，语义清晰。用户重跑回测即刷新卡片数据。 **[Opus]** 同意。`ListStrategyCards` 后端查询 `WHERE status='completed' ORDER BY created_at DESC LIMIT 1`，语义清晰。后续可加 `featured_backtest_id` 字段支持置顶，但不是首版需求。 **[GLM]** 同意。三方一致。后续加 `featured_backtest_id` 是可选项，不阻塞首版。 |

---

## E. 共识项（三版已一致，无需辩论）

- 浏览与创作分离的宏观架构（Gallery/Detail 给交易员，Workspace 给开发者）。
- Gallery 结果优先：sparkline + KPI（胜率/回撤/PF/夏普）+ 在跑调度数。
- 响应式卡片网格（桌面4/平板2/手机1）。
- Detail 代码 Tab：系统模板只读，用户模板 Fork & Edit。
- proto-only（无 REST）；价格/指标 Decimal 走 string（禁 float64）。
- 旧路径 `<Navigate replace>` 重定向，外链不断裂。

---

## F. Claude 补充：两份文档均未覆盖的问题

### F1. 刚建的 `StrategyLibraryPage` 怎么处理？

**现状**：`StrategyLibraryPage.tsx`（187 行）今天刚实现——搜索/筛选/分页/CRUD 已跑通。Cascade v2 Phase D 和 GLM Phase 4 都说要"删除"。

**建议**：**不要丢弃。** 拿它当 Gallery 的骨架——搜索/筛选（All/My/Preset）/分页逻辑/`listTemplates` 调用/删除确认已就绪。把 `<Table>` 替换为 `<Row gutter={[16,16]}>` 卡片网格即可。数据源后续从 `listTemplates` + 前端 N+1 聚合 切换到 `ListStrategyCards` RPC。

**收益**：节省 ~150 行重复代码，Gallery 实现周期缩短。

**[Opus]** 同意。LibraryPage 的 `useQuery(queryKeys.templates.list())` + `useMemo` filter 逻辑可直接复用。Gallery = LibraryPage 换 Table→Card Grid + 加 sparkline/KPI 渲染 + 数据源切换。增量演进，不是推倒。

**[GLM]** 同意。我原版 Phase 4 说「删除 LibraryPage」是浪费——Claude 的复用建议更务实。Gallery = LibraryPage 骨架 + Card Grid 替换 Table + ListStrategyCards 替换 listTemplates。

### F2. 侧边栏导航

Gallery 替代 Library 后，侧边栏需更新。

**当前状态（2026-07-23）**：侧边栏 Strategy 菜单组下有 4 项：

```
Strategy
├── Strategy Workspace    ← 代码编辑+回测+AI
├── Strategy Library 🆕   ← 今天加的，指向 /strategy/library（Table 列表）
├── Live Monitor          ← 实盘监控
├── Market Tools          ← 行情分析
```

**Gallery 上线后的变更**：第 2 项从 "Strategy Library" → "Strategies"，指向 `/strategy`（Gallery）。

**[Claude] 建议**：命名用 "Strategies" 而非 "Gallery"——Gallery 是 UI 模式，不是用户语言。如果你主张 Gallery 作为 `/strategy` 默认首页、不单列导航项——这也在讨论范围内，见决策表 A2。

**[Opus]** 同意 "Strategies" 命名。主张 Gallery 作为 `/strategy` 默认首页（不单列导航项）——用户点 "Strategies" 进 Gallery，Gallery 卡片的 [Open in Workspace] 跳转到 Workspace。侧边栏变为：Strategies / Workspace / Live / Market Tools（4 项不变，第 2 项改名+改路由）。

**[GLM]** 同意 "Strategies" 命名和 `/strategy` 默认首页。我原版用 "Gallery" 确实是开发者语言不是用户语言。侧边栏 4 项不变，第 2 项改名+改路由。

### F3. Marketplace 发布入口

GLM 和 Cascade 版都未提及策略「发布到市场」的操作。`strategy_library` i18n 已有 `publish`/`publishSuccess`/`unpublish` 等 key。

**建议**：Gallery 卡片的 actions 应包含 [Publish to Market]（对非系统、非已发布模板）。这是策略生命周期的完整闭环——Library 只管理，Marketplace 是分发渠道。不能漏。

**[Opus]** 同意。卡片 actions 完整集：[Detail] [Quick Deploy] [Fork] [Publish to Market] [Delete]（按模板类型裁剪：系统模板只有 [Detail]；我的模板有全部；已发布的模板 [Publish] 变 [Unpublish]）。复用现有 `PublishToMarketModal` 组件。

**[GLM]** 同意。这是我原版和 v2 都漏掉的——策略生命周期闭环需要 Publish 入口。复用现有 `PublishToMarketModal` 组件，卡片 actions 按模板类型裁剪按钮集。

---

## G. Claude 补充 #2：策略代码访问控制（三方均遗漏，事后补）

### 背景

"代码不出平台"是项目经营层的核心原则（`docs/roadmaps/business-direction.md` §1），但 ADR-0027 三方讨论聚焦在 UX 分工（Gallery/Detail/Workspace），**权限模型是三方共同的遗漏**。

2026-07-23 审计发现：`StrategyDetailPage.tsx` 的 Code Tab 和 [Edit] 按钮对任何人开放，`StrategyCard.tsx` 的 [Deploy] 按钮仅检查 `!isSystem`（所有人可点），`/strategy/:id/edit` 可加载任意模板。这三个漏洞合起来允许任何登录用户查看、编辑、部署任何其他用户的策略代码。

### 访问控制矩阵

| 操作 | 系统模板 | 我的模板 | 别人的已发布模板 | 别人的未发布模板 |
|------|---------|---------|---------------|---------------|
| **Gallery 中可见** | ✅ 所有人 | ✅ 仅我（或"我的"筛选） | ✅ 所有人 | ❌ 不出现 |
| **查看 Overview Tab** | ✅ 任何人 | ✅ 仅我 | ✅ 任何人 | ❌ 不适用 |
| **查看 Code Tab** | ✅ 任何人（开源） | ✅ 仅我 | ❌ **隐藏 Code Tab** | ❌ 不适用 |
| **[Edit] 按钮** | ❌ 无（改为 [Fork & Edit]） | ✅ 仅我 | ❌ 隐藏 | ❌ 不适用 |
| **[Fork]** | ✅ 任何人 | ✅ 仅我 | ❌ 禁止（已发布策略不可 Fork） | ❌ 不适用 |
| **[Deploy]** | ✅ 任何人 | ✅ 仅我 | ✅ **仅购买者**（通过 marketplace 购买后） | ❌ 不适用 |
| **[Publish]** | ❌ 不适用 | ✅ 仅我（从未发布→已发布） | ❌ 不适用 | ❌ 不适用 |
| **[Delete]** | ❌ 不适用 | ✅ 仅我 | ❌ | ❌ 不适用 |
| **Workspace 编辑** | ❌（不可编辑系统模板） | ✅ 仅我 | ❌ 禁止 | ❌ 不适用 |

### 实现要求

**StrategyDetailPage**：
- 从 `useAuthStore` 获取当前用户 ID
- 计算 `isOwner = template.userId === currentUserId`
- Code Tab：仅 `isOwner || template.isSystem` 时渲染
- [Edit] 按钮：`isSystem` → 改为 [Fork & Edit]；`isOwner` → [Edit]；其他 → 不渲染

**StrategyCard**：
- [Deploy]：`isOwner || (isPublished && hasPurchased)` — 已修复部分（Card 已有 `isOwner` 检查）
- [Fork]：仅 `isOwner || isSystem`

**StrategyWorkspacePage**：
- 加载模板时检查 `isOwner || isSystem`，非 owner 且非系统模板 → 拒绝加载，redirect 回 Gallery

**后端防护（纵深防御）**：
- `getTemplate` RPC：非 owner 且非公开模板 → 返回错误
- `updateTemplate` RPC：非 owner → 返回 `PermissionDenied`
- `deleteTemplate` RPC：非 owner 且非 admin → 返回 `PermissionDenied`

### 表决

**[Claude]** 这是三方共同的遗漏。上面的矩阵是 2026-07-23 代码审计发现的，直接确认。

**[GLM]** ☐ 待确认

**[Opus]** ☐ 待确认

---

## 决策汇总（三方共识，23/23 项一致）

| 层 | 最终决策 | 负责人 | 备注 |
|----|----------|--------|------|
| L1 消费/CRUD | **重写**（复用 LibraryPage 骨架） | GLM | Table→Card Grid；数据切 `ListStrategyCards`；复用搜索/筛选/分页 |
| L2 创作 Workspace | **重构**（Feature-slice 拆分） | GLM | 拆 283 行上帝 hook；按 Code→Backtest→AI→Tuning 拓扑序逐域迁移；UI 组件不动 |
| L3 引擎/调度/SSE | **复用** | — | `useBacktestRunner`/SSE/ScheduleLaunch 不动 |
| 创建流 | **无 Modal，Gallery + Workspace 直接入口** | GLM | [New]/[Fork]/[AI Generate] → Workspace；Import EA → Drawer；首访浮层 |
| 后端 RPC | **新增 `ListStrategyCards`** | GLM | 反范式化，一次返回 sparkline + KPI + running_schedules；Decimal=string |
| Detail 页面 | **Overview + Code 只读** | GLM | 不做回测/部署；一个 [Open in Workspace] 按钮 |
| Gallery Compare | **Phase A 后续增量** | GLM | 不阻塞首版；复用 marketplace CompareModal |
| 路由 | **`/strategy/view/:id`** | GLM | 避让 `/:strategyId`（被分享页占用）；旧路径重定向 |
| Marketplace 入口 | **卡片 [Publish to Market]** | GLM | 系统模板无；用户模板有 Publish/Unpublish；复用 PublishToMarketModal |
| 侧边栏导航 | **"Strategies" → `/strategy`** | GLM | Gallery 作为策略默认首页 |
| 修 bug | **`template`→`templateId`** | GLM | 随 L1 重做修正 |
| 懒加载分包 | **Gallery/Detail 不 import Monaco** | GLM | 交易员首屏 ~300KB |
| Gallery Compare | ☑ Phase A 后续 | 同 Claude |
| 路由 | ☑ view/:id | v2 §4.2 |
| Marketplace 入口 | ☑ 卡片 [Publish to Market] | 补充 v2 未覆盖 |
| 侧边栏导航 | ☑ "Strategies" → /strategy | Gallery 作为默认首页 |

### GLM 补充投票

| 层 | GLM 投票 | 备注 |
|----|----------|------|
| L1 消费/CRUD | ☑ 重写（复用 LibraryPage 骨架） | 同 Claude；原版 Phase 4 说删除是浪费 |
| L2 创作 Workspace | ☑ 重构（Feature-slice） | 原版盲区，认同 v2 §4.6；建议按依赖拓扑序拆分 |
| L3 引擎/调度/SSE | ☑ 复用 | 同 Claude/Opus |
| 创建流 | ☑ 无 Modal（Gallery 即入口） | 投降向导；加可跳过首访浮层 |
| 后端 RPC | ☑ ListStrategyCards | 原版「GetTemplateStats 或前端聚合」悬而未决，v2 定案 |
| Detail 边界 | ☑ Overview + Code 只读 | 同 Claude；原版 4 Tab 过度设计 |
| Gallery Compare | ☑ Phase A 后续 | 原版漏掉 |
| 路由 | ☑ view/:id | 原版 /strategy/:id 撞车，我的疏忽 |
| Marketplace 入口 | ☑ 卡片 [Publish to Market] | 原版和 v2 都漏掉 |
| 侧边栏导航 | ☑ "Strategies" → /strategy | 原版用 "Gallery" 是开发者语言 |

---

## 三方收敛决策（最终）

| 层 | 最终操作 | 三方一致 | 备注 |
|----|----------|----------|------|
| L1 消费/CRUD | ☑ 重写（复用 LibraryPage 骨架） | ✅ | Table→Card Grid；数据切 ListStrategyCards |
| L2 创作 Workspace | ☑ 重构（Feature-slice） | ✅ | 拆上帝 hook；UI 组件保留；按依赖拓扑序迁移 |
| L3 引擎/调度/SSE | ☑ 复用 | ✅ | 回测引擎/SSE/ScheduleLaunch 不动 |
| 创建流 | ☑ 无 Modal（Gallery 即入口） | ✅ | [Fork]+[New]+[AI Generate]+ImportEA Drawer；可跳过首访浮层 |
| 后端 RPC | ☑ ListStrategyCards | ✅ | 反范式化，一次返回卡片全字段 |
| Detail 边界 | ☑ Overview + Code 只读 | ✅ | 不做回测/部署；统一跳 Workspace |
| Gallery Compare | ☑ Phase A 后续 | ✅ | 不阻塞首版上线 |
| 路由 | ☑ view/:id | ✅ | 避开 :strategyId 冲突 |
| Marketplace 入口 | ☑ 卡片 [Publish to Market] | ✅ | 策略生命周期闭环 |
| 侧边栏导航 | ☑ "Strategies" → /strategy | ✅ | Gallery 作为默认首页 |

**三方全部分歧已收敛，14 行议题 + 5 个开放问题 + 3 个补充问题全部一致。可以进入实施阶段。**

---

## H. 业务流审计发现的断层（2026-07-24 交付前审计，Claude）

ADR-0027 设计聚焦在「策略创建→管理→发布」的策略模块内部 UX。以下是发布后「策略模块↔市场模块」桥接层的 3 个断层。

### H1. 🔴 重复发布无防护

**现状**：`backend/internal/marketplace/publish.go` 的 `INSERT INTO marketplace_strategies` 没有 `ON CONFLICT` 子句。同一策略 ID 可以重复发布，产生多条市场记录。

**修复**：
```sql
-- publish.go 中 INSERT 改为：
INSERT INTO marketplace_strategies (...) VALUES (...)
ON CONFLICT (strategy_id) WHERE status = 'published' DO NOTHING
```

**验收**：同一策略 ID 调用 Publish 两次 → 第二次返回已有的 publish ID，不创建重复记录。

### H2. 🔴 购买后无法部署

**现状**：买家在市场购买/订阅策略后，`CanAccessCode` 返回 true（后端权限正确），但前端没有任何入口让买家将策略部署到实盘：
- `PurchaseTab.tsx` — 只有「查看详情」和「跑回测」，没有部署按钮
- `StrategyDetailModal.tsx` — 市场侧详情弹窗，没有部署入口
- `StrategyCard.tsx` — Gallery 卡片的 [Deploy] 仅对 `isOwner` 开放

**修复**：
- `PurchaseTab.tsx`：每行加 `[Deploy]` 按钮 → 拉起 `DeployScheduleModal`（复用 `StrategyCard` 已有的组件）
- `StrategyDetailModal.tsx`：已购买用户显示 `[Deploy]` 按钮

**验收**：购买者从「My Purchases」Tab 点击 [Deploy] → 选账户 → 策略上线。

### H3. 🟡 发布后 [Publish] 按钮不消失

**现状**：`StrategyCard.tsx` 判断 `isPublished = card.isPublic` 来决定是否显示 [Publish] 按钮。但 marketplace 发布不更新 `strategy_templates.is_public`——发布后按钮仍然显示，诱导用户重复发布。

**修复**：`ListStrategyCards` 后端查询 JOIN `marketplace_strategies`，返回 `is_marketplace_published` 字段。前端改为 `isPublished = card.isMarketplacePublished`。

**验收**：发布后 StrategyCard 显示 [Unpublish] 替代 [Publish]。

### 表决

**[Claude]** 三项均为 2026-07-24 交付前深度审计发现。确认必须修。

**[GLM]** ☐ 待确认

**[Opus]** ☐ 待确认
