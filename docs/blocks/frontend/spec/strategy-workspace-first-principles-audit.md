# 策略工作台第一性原理审计

> **日期**: 2026-08-05
> **审计**: Claude Code
> **范围**: `StrategyWorkspacePage` + 全部 tab/modal/drawer + 用户流程
> **关联**: `docs/plan/mql-ea-compatibility-proposal.md`（批次 1 修复完成后的 UI 审视）

---

## 0. 产品第一性原理

从 `CLAUDE.md` 产品定位推导出策略工作台应遵循的 6 条设计原则：

| 原则 | 推导 | 来源 |
|------|------|------|
| **P1: 代码是重心** | 开发者的一切活动围绕策略代码展开 | 策略市场定位 |
| **P2: AI 是工具，不是目的地** | AI 用于辅助编码，不是替代编码视图 | "AI 持续迭代策略"是核心差异化 |
| **P3: 动作是瞬间，不是位置** | 导入、回测、部署是离散动作，不是持久状态 | 用户心智模型 |
| **P4: 反馈与创作同屏** | 回测结果、AI 建议、校验错误应在编码时可见 | 缩短反馈循环 |
| **P5: 导航与创作分离** | 浏览策略列表 ≠ 编辑一个策略，不同模式 | 关注点分离 |
| **P6: 渐进披露** | 新用户看到最简单界面，高级功能逐层展开 | 降低门槛 |

---

## 1. 当前架构

### 1.1 页面结构

```
StrategyWorkspacePage
├── WorkspaceToolbar          [Account] [Symbol] [TF] [Strategy Name] [Save Status]
├── WorkspaceCenterColumn
│   ├── Tab Bar (5 tabs)
│   │   ├── 🤖 AI Assistant   (chat)
│   │   ├── 📝 Code           (code)
│   │   ├── 📥 Import MQL     (import)
│   │   ├── 📊 Backtest       (backtest)
│   │   └── 📋 Strategies     (strategies)
│   ├── Tab Content (每次只显示一个)
│   └── BottomPanelSection    [Positions] [History] [Backtest] + QuickTrade
└── WorkspaceDrawers
    ├── BacktestParamsModal
    ├── SaveTemplateModal
    ├── IndicatorCatalogDrawer
    ├── BacktestHistoryDrawer
    ├── VersionHistoryDrawer
    ├── AISettingsModal
    ├── PublishToMarketModal   (Strategies tab 内)
    ├── DeployScheduleModal    (gallery/marketplace 触发)
    ├── TradePasswordModal     (部署密码)
    └── ScheduleHealthModal
```

### 1.2 用户流程

1. 入口：Gallery (`/strategy`) → "New Strategy" (`/strategy/new`) 或 "AI Generate" (`/strategy/new?ai=1`)
2. AI 路径：chat tab → agent 流式生成 → "Apply to Editor" → code tab
3. 代码路径：code tab → 编辑 → Save
4. 导入路径：import tab → 粘贴 MQL → Analyze → Import → code tab
5. 回测：Run Backtest 按钮 → BacktestParamsModal → 确认 → backtest tab
6. 部署：strategies tab → Publish to Market | Live page → Create Schedule

### 1.3 路由

```
/strategy                → StrategyGalleryPage
/strategy/new            → StrategyWorkspacePage
/strategy/:id/edit       → StrategyWorkspacePage
/strategy/view/:id       → StrategyDetailPage
/strategy/live           → LiveStrategyPage
```

侧栏菜单：Strategies (`/strategy`)、Workspace (`/strategy/new`)、Live Monitor (`/strategy/live`)

---

## 2. 逐项评估

### 2.1 五个 Center Tab

#### 🤖 AI Assistant Tab — ❌ 违反 P1, P2, P4

**现象**：AI 聊天是独立全宽 tab，切过去后代码编辑器消失。

**根因**：把 AI 当成"目的地"而非"工具"。AI 的核心用途是"帮我改这段代码"——当前设计下用户必须记住代码内容，切到 AI tab 描述需求，AI 产出代码后 apply，再切回 code tab 看结果。反馈循环被 tab 切换打断。

**加剧因素**：`workspaceStore` 默认 `centerTab = 'chat'`，新用户进入 `/strategy/new` 第一眼看到的是聊天界面，不是代码编辑器。

**第一性原理要求**：AI 面板应是可折叠的右侧面板，与代码编辑器同屏。用户在代码中选中一段→右侧 AI 面板直接上下文感知→AI 产出 diff→用户预览→apply。

#### 📥 Import MQL Tab — ❌ 违反 P3

**现象**：独立的持久 tab，占据 20% 导航空间。

**根因**：导入是一次性动作。用户粘贴 MQL → 分析 → 导入 → 代码进入编辑器。此后 tab 无任何用途。把一次性动作做成持久位置是类别错误。

**第一性原理要求**：Import 应是 modal。新用户无代码时首次进入 workspace 自动弹出；老用户通过工具栏 "Import" 按钮触发。

#### 📊 Backtest Tab — ⚠️ 部分违反 P1, P4

**现象**：回测配置和结果在单独 tab，切过去看不到代码。

**缓解因素**：BottomPanel 跨 tab 显示回测指标（equity curve 等）；回测参数配置已通过 `BacktestParamsModal` 完成（modal 是正确的交互模式）。

**第一性原理要求**：Backtest tab 本身可取消——参数配置已有 modal，结果展示已有 BottomPanel。如果用户需要看完整 trade 列表，应通过 BottomPanel 扩展或独立 drawer 实现。

#### 📋 Strategies Tab — ❌ 违反 P5

**现象**：编辑一个策略时，tab 中展示"所有策略的列表"。

**根因**：导航（浏览策略）和创作（编辑策略）混在一个视图。相当于 Google Docs 编辑器里有一个 tab 显示"你的所有文档"。

**叠加问题**：
- 侧栏已有 "Strategies" 入口 (`/strategy`) → 与 workspace 内部 tab 概念重叠
- Strategies tab 内部 "Create" 按钮调用 `tpl.openCreate()` 设置 `editOpen` 状态，但**没有 modal 渲染此状态** → 点击后无事发生（dead path）

**第一性原理要求**：Strategies tab 应从 workspace 中移除。策略列表通过侧栏导航到 `/strategy`（gallery 页面）访问。

#### 📝 Code Tab — ✅ 正确，但地位过低

**问题**：这是唯一符合第一性原理的 tab，但被降级为"5 个之一"。它应该是**唯一持久视图**，其他一切围绕它。

### 2.2 Bottom Panel — ✅ 符合 P4

```
[ Positions ] [ History ] [ Backtest ]
```

跨所有 tab 持久存在，显示实时数据。体现"反馈与创作同屏"。QuickTrade 侧面板（desktop）提供快速下单能力。

**唯一不足**：上层 tab 切换会改变上下文，削弱底部面板"始终可见"的好处。

### 2.3 Modal/Drawer — 全部 ✅

| 组件 | 判定 | 理由 |
|------|------|------|
| BacktestParamsModal | ✅ | 动作前参数配置，modal 正确 |
| TradePasswordModal | ✅ | 安全步骤，modal 正确 |
| PublishToMarketModal | ✅ | 偶尔执行的复杂表单，modal 正确 |
| DeployScheduleModal | ✅ | 配置型动作，modal 正确 |
| SaveTemplateModal | ✅ | 简单表单，modal 可接受 |
| IndicatorCatalogDrawer | ✅ | 参考资料，drawer 正确 |
| VersionHistoryDrawer | ✅ | 偶尔查阅，drawer 正确 |
| BacktestHistoryDrawer | ✅ | 历史记录浏览，drawer 正确 |
| AISettingsModal | ✅ | 设置，modal 正确 |

**结论：全部 modal/drawer 设计正确。问题只出在顶层 tab 结构。**

### 2.4 Toolbar — ✅

Account/Symbol/Timeframe 选择器 + Strategy Name + Save Status + Save/Backtest/Copy/Send to AI 按钮。设计正确。

### 2.5 死代码

以下组件已定义但**无任何活跃代码引用**（15+ 文件）：

| 死代码 | 位置 |
|--------|------|
| `WorkspaceCodePanel` | `components/workspace/` |
| `WorkflowBar` / `ExecutionPanel` / `useExecutionPanel` | `components/strategy/` |
| `AIPanel` / `CodeEditorPanel` | `components/editor/` |
| `BacktestRunsCard` / `BacktestResultsCard` | `pages/strategy/` 和 `components/strategy/` |
| `CoverageReportView` / `DiffView` | `components/strategy/` |
| `PlanPanel` / `ChatMessageItem` / `StepProgress` | `components/strategy/` |
| `editor/*` 全部 6 个文件 | `components/editor/` |
| `CodeAssist` / `AICodeReviseChat` | `components/strategy/` |

保留死代码增加维护负担、误导新开发者、增大 bundle 体积。

---

## 3. 判定汇总

| # | 元素 | 判定 | 违反原则 | 严重度 |
|---|------|------|---------|--------|
| 1 | AI Assistant Tab | ❌ | P1, P2, P4 | 高 |
| 2 | Import MQL Tab | ❌ | P3 | 高 |
| 3 | 默认 tab = chat | ❌ | P1, P6 | 高 |
| 4 | Strategies Tab | ❌ | P5 | 中 |
| 5 | Strategies "Create" 无响应 | 🐛 | — | 中 |
| 6 | Backtest Tab | ⚠️ | P1, P4 | 中 |
| 7 | 15+ 死代码文件 | ⚠️ | — | 低 |
| 8 | Code Tab | ✅ | — | — |
| 9 | Bottom Panel | ✅ | — | — |
| 10 | 全部 9 个 Modal/Drawer | ✅ | — | — |
| 11 | Toolbar | ✅ | — | — |

---

## 4. 改进方案

### 4.1 目标架构

```
┌──────────────────────────────────────────────────────────┐
│ Toolbar: [Account] [Symbol] [TF]     [Import] [Save] [Backtest] [Deploy] │
├──────────────────────────────────┬───────────────────────┤
│                                  │  AI Assistant         │
│  Code Editor                     │  (可折叠右侧面板)      │
│  (始终可见)                       │  - 与代码同屏          │
│                                  │  - 上下文感知          │
│                                  │  - 产出 diff→预览→apply│
├──────────────────────────────────┴───────────────────────┤
│ Bottom Panel: [Positions] [History] [Backtest]           │
│ (可折叠，展示实时数据 + 回测结果)                           │
└──────────────────────────────────────────────────────────┘

Import MQL → Modal（新用户无代码时首次进入自动弹出）
Strategies → 侧栏 /strategy（gallery 页面）已覆盖
Backtest → BacktestParamsModal + BottomPanel 已覆盖
```

### 4.2 改造分三批

#### 批次 A — 止血（改动最小，收益最大）

| # | 改动 | 文件 | 效果 |
|---|------|------|------|
| A1 | 默认 tab `'chat'` → `'code'` | `workspaceStore.ts` | 新用户第一眼看到代码编辑器 |
| A2 | Import MQL tab → modal（首次无代码自动弹出） | `WorkspaceCenterColumn.tsx` + `ImportEAPanel.tsx` | 消除死 tab |
| A3 | 移除 Strategies tab | `WorkspaceCenterColumn.tsx` | 消除导航/创作混淆 |
| A4 | 工具栏策略名旁加下拉 → 最近策略 + "View All" | `WorkspaceToolbar.tsx` | 弥补 A3 移除的策略切换入口 |
| A5 | 修复 "Create" 按钮死胡同 | `StrategiesTab.tsx` | 或随 A3 一并清理 |

#### 批次 B — 结构优化（中等改动）

| # | 改动 | 文件 | 效果 |
|---|------|------|------|
| B1 | AI 从全宽 tab 改为右侧可折叠面板 | `WorkspaceCenterColumn.tsx` + `StrategyChat.tsx` | 代码+AI 同屏，缩短反馈循环 |
| B2 | 移除 Backtest tab；Tuning/Gate 移至 BottomPanel "Advanced"→全屏 drawer | `WorkspaceCenterColumn.tsx` + `BottomPanelSection.tsx` | 消除冗余 tab，功能不丢 |
| B3 | 移除 `?ai=1` 参数逻辑，AI 面板默认折叠 | `useTemplateSlice.ts` | AI 面板由用户主动打开 |

#### 批次 C — 清理（低优先级）

| # | 改动 | 文件 | 效果 |
|---|------|------|------|
| C1 | 删除 15+ 死代码文件 | 见 §2.5 | 减少维护负担 |
| C2 | 删除 `BacktestRunsCard` | `pages/strategy/BacktestRunsCard.tsx` | 无人引用 |

### 4.3 不纳入改造的项

以下设计保持不变，因为它们已经符合第一性原理：
- Toolbar（account/symbol 选择器 + 操作按钮）
- Bottom Panel（Positions/History/Backtest 数据）
- 全部 9 个 Modal/Drawer
- 路由结构（`/strategy/new`、`/strategy/:id/edit` 等）

### 4.4 对抗性审视

在定稿前，对 5 个关键决策做挑战性验证：

**Q1: AI → 右侧面板，是否有更好的替代？**
- 底部面板？❌ 底部已被 Positions/History/Backtest 占据，再加 AI 会过载
- 独立 tab？❌ 即当前方案，已判违反 P1/P2/P4
- 悬浮小窗？⚠️ 可行但不如侧面板稳定，代码编辑时需要固定参照
- **结论：右侧可折叠面板是最优解** — 左到右阅读流（代码→AI），代码始终可见

**Q2: Import → Modal 首次自动弹出，会干扰老用户吗？**
- 只在无代码时触发。有代码的用户不会看到。无代码时弹出是合理的——用户总得从某处开始
- **结论：无风险**

**Q3: 移除 Strategies tab 后，用户如何切换策略？**
- 原方案有缺口。补充：工具栏策略名旁加下拉箭头→最近策略列表+"View All"链接到 `/strategy`（批次 A4）
- **结论：补上策略切换入口后，方案完整**

**Q4: 移除 Backtest tab 后，Tuning 和 Gate 功能放哪？**
- BacktestPanel 有三个子 tab：Results（已有 BottomPanel 覆盖）、Tuning（参数优化）、Gate（风控评估）
- 方案：BottomPanel 的 Backtest 子 tab 中加 "Advanced: Tuning | Gate" 链接→全屏 drawer
- **结论：功能不丢，不占顶级 tab，交互路径清晰**

**Q5: 默认改为 code 后，AI Generate 入口是否丢失？**
- Gallery 页面已有 "AI Generate" 按钮（`?ai=1`）
- Code tab 工具栏有 "Send to AI" 按钮（紫色，醒目）
- **结论：AI 入口不丢，且用户在有代码后使用 AI 是更自然的时机**

**综合结论：方案在对抗审视后仍成立。发现 1 个缺口（策略切换）已补入批次 A4。**

---

## 5. 回测结果展示 — 最优解

### 5.1 现状问题

回测结果分散在三处：
- 代码区上方状态条（最近一次指标摘要 + Re-run）
- 底部面板 Backtest 子 tab（指标卡片 + "Tuning & Gate →"）
- Tuning & Gate 全屏抽屉（完整结果/参数优化/风控）

用户回测后要看完整结果需切换多个位置，下一步操作入口分散。

### 5.2 第一性原理

回测是验证，不是终点。用户的下一步取决于结果：
- 不满意 → 调参数 / 改代码 / 问 AI
- 满意 → 部署实盘 / 发布市场
- 不确定 → 分析交易明细 / 看曲线

最优解：回测完成后，结果和下一步动作在**一处呈现，一键可达**。

### 5.3 目标设计

```
┌── 代码编辑器 ──────────────────────────────┐
│  // MACD Sample source...                  │
├── 回测结果（回测完成后自动展开，可拖拽调整高度）─┤
│  📈 Equity Curve (200px)                   │
│  +12.3% · DD 8.2% · Win 62% · 45 trades   │
│                                            │
│  [🔧 调参数] [💬 AI分析] [📋 交易明细]        │  ← 不满意时
│  [🚀 部署实盘] [📢 发布市场]                  │  ← 满意时
└────────────────────────────────────────────┘
```

### 5.4 施工项（批次 D）

| # | 改动 | 效果 |
|---|------|------|
| D1 | 回测完成后 BottomPanel 自动展开为完整结果视图 | 不用手动点 |
| D2 | 底部面板增加 equity chart + 指标卡片行（从 BacktestPanel Results 迁入） | 结果完整可见 |
| D3 | 底部面板增加操作按钮行（调参数 / AI 分析 / 交易明细 / 部署 / 发布） | 一步到下一步 |
| D4 | 移除代码区上方回测状态条（信息已在底部面板） | 去重 |

Tuning / Gate 仍通过底部面板按钮进入全屏抽屉，不丢失功能。

---

## 6. 修订历史

| 日期 | 修订 |
|------|------|
| 2026-08-05 | v4 — 新增 §5 回测结果展示最优解：回测完成后底部面板自动展开为完整结果+操作入口，一批（D）4 项 |
| 2026-08-05 | v3 — 全三批实施完毕：A（5项止血）+ B（3项结构优化）+ C（14文件死代码清理，修正探索 agent 误判 8 项） |
| 2026-08-05 | v2 — 对抗性审视：验证 5 个关键决策，发现 1 个缺口（策略切换入口）补入批次 A4；批次 B2 细化 Tuning/Gate 去向 |
| 2026-08-05 | v1 — 初始审计：逐 tab/modal 评估，6 条第一性原理推导，3 批改进方案 |
