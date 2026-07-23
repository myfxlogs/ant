# ADR-0027 策略模块重构 · 三方讨论对照表

- **用途**：Cascade / Claude / GLM 三方评审用。逐条对齐「GLM 版」与「Cascade v2 版」的立场，收敛出最终决策。
- **三份提案**：
  - GLM 版：`0027-strategy-gallery-redesign.md`
  - Cascade v2：`0027-v2-strategy-module-redesign.md`
  - **Claude 意见**：已填入本文各行的「决策理由/附加意见」列（标注 [Claude]），并在 §F 补充了三项两份文档均未覆盖的问题。
- **填写方式**：每行在「最终决策」列填结论，「决策理由/附加意见」列记录三方发言。

---

## A. 宏观分歧（架构层）

| # | 议题 | GLM 0027 | Cascade v2 | 关键权衡 | 最终决策 | 决策理由 / 附加意见 |
|---|------|----------|-----------|----------|----------|---------------------|
| A1 | **L2 Workspace：重构 vs 重写** | 未覆盖（仅删侧边栏） | **内部重构** 为 feature-slice；论证重构支配重写 | 重写终点与重构相同，但要重造 37 组件+17 hooks+SSE/回测联动 | ☑ 重构 | **[Claude]** 投重构。今天深入看过 workspace 代码——hook 需要拆，但组件是成熟的。Cascade §4.6 论证正确：feature-slice 是必然终点。反证门槛：能举证「重写才能达到、Feature-slice 达不到的更优终点」才推翻。 |
| A2 | **重做范围** | Gallery/Detail/Create 全新，Workspace 精简 | **L1 重写 + L2 重构 + L3 复用** 的组合最优 | 全模块推倒 vs 分层处理 | ☑ 组合（L1重写+L2重构+L3复用） | **[Claude]** L1 推倒是正确的（表格不适合浏览），L2 重构是对的（上帝 hook），L3 复用是必须的（引擎外溢到 Live/Marketplace）。建议：L1 重写时复用刚建的 LibraryPage 骨架（搜索/筛选/分页已跑通），将 Table 替换为 Card Grid 而非从零开始。 |
| A3 | **上帝 hook / 单体 store** | 保留 `useStrategyWorkspaceState`(283行) + `workspaceStore` | 拆成各域 slice hook + slice-creator store（`v6`） | 一体化状态耦合 vs 分片自治 | ☑ 拆（Feature-slice） | **[Claude]** 283 行单一 hook 返回 10 域大对象，域间用 useEffect 手动 rewire——这是当前 workspace 最明显的技术债务。拆为 Code/Backtest/Tuning/AI/Chart/QuickTrade/Layout/History 各域独立 hook + store slice，组件按需订阅。Workspace 页面退化为组合根（~50 行）。 |
| A4 | **懒加载分包** | 未明确 | Gallery/Detail 不 import Monaco/回测引擎；仅 `:id/edit` 加载重包 | 交易员首屏体积 | ☑ 采纳 | **[Claude]** 交易员是主要用户，他们不进 Workspace。Gallery 首屏不应加载 Monaco editor（~600KB）和回测引擎。这是高收益低风险优化。 |

---

## B. UX / 交互分歧

| # | 议题 | GLM 0027 | Cascade v2 | 关键权衡 | 最终决策 | 决策理由 / 附加意见 |
|---|------|----------|-----------|----------|----------|---------------------|
| B1 | **创建流** | 4 步 Stepper 向导页 | 砍向导 → 起点 Modal + 非线性 Workspace + 首访引导浮层 | 锁步引导 vs 非线性迭代循环 | ☑ 比 Cascade 更激进：**不要 Modal** | **[Claude]** 策略创作是「写→测→改→问 AI→再测」的非线性循环。向导与之对抗（尤其「必须回测通过才保存」锁定）。但我主张比 Cascade 更简化：**砍掉 Modal 本身**——Gallery 卡片上的 [Fork] 按钮就是 Fork 现有模板的入口；[New] 按钮直接进 Workspace 空白模板；Import EA 用已加好的 Drawer。Gallery 列表已呈现"Fork 现有"的所有选项，Modal 是多余的一步。 |
| B2 | **Detail 回测/部署** | 独立 4 Tab（含完整回测/部署） | 轻量默认档，深度迭代跳 Workspace | 功能完整 vs 避免三处重复实现 | ☑ 比 Cascade 更激进：**Detail 不做回测/部署** | **[Claude]** Detail 只做 Overview（KPI+完整权益曲线）+ Code（只读）。回测和部署都属于 Workspace。理由：①若在 Detail 跑轻量回测想看详情→必须跳 Workspace，认知断裂；②卡片已有 sparkline+KPI，Detail Overview 展示完整版已满足深入需求；③一个[Open in Workspace]按钮消除心智负担。唯一例外：卡片上的[Quick Deploy]按钮直接拉 ScheduleLaunchModal。 |
| B3 | **Gallery 增强** | 卡片网格 | 卡片 + **多选 Compare 对比** | 交易员横向对比决策 | ☑ 加 Compare（Phase A 后续） | **[Claude]** 交易员靠对比做决策。但不应阻塞 Phase A 上线——卡片网格先出，Compare 作为增量。复用 marketplace `CompareModal` 模式。 |
| B4 | **一键部署** | Detail Deploy Tab | 卡片/Detail 一键部署（默认参数+选账户） | 交易员是否需进 IDE 才能部署 | ☑ 采纳（卡片按钮） | **[Claude]** 策略卡片的 actions 应包含[Quick Deploy]按钮——直接拉 `ScheduleLaunchModal`，选账户即可上线。这是交易员最高频操作。无需进 Detail 或 Workspace。 |

---

## C. 硬约束 / 缺陷（非观点，须处理）

| # | 事项 | 现状（已核查） | 处理方案 | 最终决策 | 备注 |
|---|------|---------------|----------|----------|------|
| C1 | **路由冲突** | `/strategy/:strategyId` 已被 `StrategySharePage` 占用 | v2：Detail 用 `/strategy/view/:id`，避让 | ☑ view/:id | **[Claude]** 确认冲突存在（代码核查）。Detail 需要登录、Share 无需登录——合并增加未登录态分支复杂度。分开更干净。 |
| C2 | **参数 bug** | Library 传 `?template=`，Workspace 读 `?templateId=` → 打开不加载模板 | 统一为 `templateId`，随 L1 重做修复 | ☑ 确认修复 | **[Claude]** 已核实代码：`StrategyLibraryPage.tsx:69` 传 `?template=${tpl.id}`，`useStrategyWorkspaceState` 读 `'templateId'`。参数名不一致导致当前 Library 的 [Open in Workspace] 不工作。随 L1 重做一并修正。 |
| C3 | **缺聚合 RPC** | 卡片摘要需 N+1 调 `listBacktestRuns` | 新增 `ListStrategyCards`（反范式化，一次返回，Decimal 走 string） | ☑ 采纳 `ListStrategyCards` | **[Claude]** 同 Cascade §4.7。一次 RPC 返回卡片所需全部字段（sparkline + KPI + running_schedules），消除 N+1。Decimal 走 string（项目精度红线）。 |

---

## D. §10 开放问题（需三方拍板）

| # | 问题 | 选项 | 最终决策 | 讨论记录 |
|---|------|------|----------|----------|
| Q1 | L2 重构 vs 重写（= A1） | 重构 / 重写 | ☑ **重构** | **[Claude]** 见 A1。反证门槛清晰：举证更优终点才推翻。 |
| Q2 | 创建流（= B1） | 起点Modal / 向导 / A/B | ☑ **Gallery 即入口，无 Modal** | **[Claude]** 比 Cascade 更简化。Gallery 卡片的 [Fork] 和 [New] 按钮即入口，无需中间 Modal。Import EA 用已加好的 Drawer。 |
| Q3 | Detail 回测/部署边界（= B2） | 轻量档阈值定义 | ☑ **Detail 不做回测/部署** | **[Claude]** 消除边界问题本身。Detail = Overview + Code 只读。一个 [Open in Workspace] 按钮统一入口。 |
| Q4 | Detail 与公开分享页 | 独立 `view/:id` / 统一为同页登录·未登录两态 | ☑ **独立** | **[Claude]** `view/:id` = 需登录的 Detail；`/:strategyId` = 公开的 Share。分开更干净。 |
| Q5 | sparkline 数据来源 | 最近一次成功回测 / 用户置顶基准回测 | ☑ **最近一次成功回测** | **[Claude]** 最简实现，语义清晰。用户重跑回测即刷新卡片数据。 |

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

### F2. 侧边栏导航

Gallery 替代 Library 后：

```
Strategy
├── Strategy Workspace    ← 编辑+回测+AI（开发者）
├── Live Monitor          ← 实盘监控
├── Market Tools          ← 行情分析
```

**[Claude] 建议**：Gallery 作为 `/strategy` 的默认首页，不在侧边栏单独列出。理由：它是"浏览入口"，不是"工具"。Workspace/Live/Market Tools 是工具。Gallery 是到达这些工具的路径。

或者保留一个导航项 "Strategies"——指向 Gallery——表示"策略库"。命名用 "Strategies" 而非 "Gallery"（Gallery 是 UI 模式，不是用户语言）。

### F3. Marketplace 发布入口

GLM 和 Cascade 版都未提及策略「发布到市场」的操作。`strategy_library` i18n 已有 `publish`/`publishSuccess`/`unpublish` 等 key。

**建议**：Gallery 卡片的 actions 应包含 [Publish to Market]（对非系统、非已发布模板）。这是策略生命周期的完整闭环——Library 只管理，Marketplace 是分发渠道。不能漏。

---

## 决策汇总（讨论结束后填写）

| 层 | 最终操作 | 负责人 | Claude 投票 | 备注 |
|----|----------|--------|-------------|------|
| L1 消费/CRUD | ☐ 重写 | | ☑ 重写（但复用LibraryPage骨架） | Table→Card Grid；数据切 ListStrategyCards |
| L2 创作 Workspace | ☐ 重构 | | ☑ 重构（Feature-slice） | 拆上帝hook；UI组件保留 |
| L3 引擎/调度/SSE | ☐ 复用 | | ☑ 复用 | 回测引擎/SSE/ScheduleLaunch 不动 |
| 创建流 | ☐ 起点Modal | | ☑ 无Modal（Gallery即入口） | [Fork]按钮+[New]按钮+ImportEA Drawer |
| 后端 RPC | ☐ ListStrategyCards | | ☑ 采纳 | 反范式化，一次返回卡片全字段 |
| Detail 边界 | ☐ 轻量 | | ☑ Overview+Code只读 | 不做回测/部署；统一跳Workspace |
| Gallery Compare | ☐ 加 | | ☑ Phase A 后续 | 不阻塞首版上线 |
| 路由 | ☐ view/:id | | ☑ view/:id | 避开 :strategyId 冲突 |
| Marketplace 入口 | 未覆盖 | | ☑ 卡片[Publish to Market] | 策略生命周期闭环 |
| 侧边栏导航 | 未覆盖 | | ☑ "Strategies" → Gallery | 或 Gallery 作为 /strategy 首页 |

---

## 决策汇总（讨论结束后填写）

| 层 | 最终操作 | 负责人 | 备注 |
|----|----------|--------|------|
| L1 消费/CRUD | ☐ 重写　☐ 其他 | | |
| L2 创作 Workspace | ☐ 重构　☐ 重写 | | |
| L3 引擎/调度/SSE | ☐ 复用　☐ 其他 | | |
| 创建流 | ☐ 起点Modal　☐ 向导 | | |
| 后端 RPC | ☐ ListStrategyCards | | |
