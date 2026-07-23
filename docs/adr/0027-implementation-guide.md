# ADR-0027 落地实施指南

- **状态**：三方共识已达成（见 `0027-decision-matrix.md`），按本文施工。
- **总原则**：每 Phase 可独立上线、可独立回滚。Gallery 先上线，Workspace 重构最后做。

---

## Phase A — Gallery + Detail（用户可见，高收益优先）

### A1. 后端：`ListStrategyCards` RPC

| 文件 | 操作 |
|------|------|
| `proto/ant/v1/strategy.proto` | 新增 `StrategyCard`、`ListStrategyCardsRequest/Response`、`rpc ListStrategyCards` |
| `backend/internal/connect/strategy/` | 新增 handler：JOIN templates + 最近回测摘要 + running_schedules 计数 |

**契约**：sparkline = 最近一次成功回测的 `equity` 数组；KPI = win_rate/max_drawdown/profit_factor/sharpe（一律 decimal→string）。无 N+1，单次 RPC 返回全部卡片数据。

**验收**：`grpcurl ListStrategyCards` 返回 12 个系统模板 + 用户模板，每张卡有 sparkline 和 KPI。

### A2. 前端：Gallery 页面

| 文件 | 操作 |
|------|------|
| `StrategyGalleryPage.tsx` | **新建**。拿 `StrategyLibraryPage.tsx` 当骨架：保留搜索/筛选（All/My/Preset）/分页。把 `<Table>` 替换为 `<Row gutter={[16,16]}>` 卡片网格。数据源先用 `listTemplates`+前端聚合，后切 `ListStrategyCards` |
| `StrategyCard.tsx` | **新建**。单张策略卡片：名称+标签+描述、sparkline（recharts mini chart）、KPI 行（胜率/回撤/PF/夏普）、running_schedules 数、actions：[Detail] [Quick Deploy] [Fork] [Publish/Unpublish] [Delete] |
| `StrategyLibraryPage.tsx` | **改为骨架**。搜索/筛选/分页逻辑提取为 `useStrategyFilter` hook，被 Gallery 复用。后续 Phase D 可考虑删除 |

**卡片 actions 规则**：
- 系统模板：`[Detail]` 仅此一个按钮
- 我的模板（未发布）：`[Detail] [Quick Deploy] [Fork] [Publish] [Delete]`
- 我的模板（已发布）：`[Detail] [Quick Deploy] [Fork] [Unpublish] [Delete]`
- [Quick Deploy] → 拉起 `ScheduleLaunchModal`（选账户 → 上线）
- [Publish] → 拉起 `PublishToMarketModal`

**响应式**：桌面 4 列、平板 2 列、手机 1 列。

**验收**：Gallery 显示所有模板为卡片，sparkline 和 KPI 正确，点击 [Detail] 跳转 `/strategy/view/:id`。

### A3. 前端：Detail 页面

| 文件 | 操作 |
|------|------|
| `StrategyDetailPage.tsx` | **新建**。2 个 Tab：Overview + Code（只读） |
| `OverviewTab.tsx` | **新建**。完整权益曲线（recharts）+ 交易统计表（胜率/盈亏比/回撤/PF/夏普）+ 参数说明 |

**Overview Tab**：权益曲线 + KPI 完整版 + 参数表。不需要后端新 RPC——从 `getTemplate` + 查询最新回测摘要即可。

**Code Tab**：`<pre>` + highlight.js（~20KB）做语法高亮，**不 import Monaco**。系统模板纯只读；用户模板显示 [Fork & Edit] 按钮 → `/strategy/:id/edit`。

**页面底部**：`[Open in Workspace]` 按钮（统一入口）。

**验收**：Detail 显示完整权益曲线和代码，点击 Fork & Edit 跳转 Workspace 并加载代码。

### A4. 路由 & 导航

| 文件 | 操作 |
|------|------|
| `AppRoutes.tsx` | `/strategy/library` → `<Navigate to="/strategy">`；新增 `/strategy/view/:id` → `<StrategyDetailPage>`；新增 `/strategy/:id/edit` → `<StrategyWorkspacePage>`；`/strategy` → `<StrategyGalleryPage>` |
| `MainLayout.tsx` | 侧边栏第 2 项 "Strategy Library" → "Strategies"，路由 `/strategy/library` → `/strategy` |

**路由表（最终）**：
```
/strategy                  → Gallery（卡片网格）
/strategy/view/:id         → Detail（Overview + Code 只读）
/strategy/:id/edit         → Workspace（编辑模式，加载指定模板）
/strategy/live             → Live Monitor（不变）
/strategy/market-tools     → Market Tools（不变）
/strategy/library          → 301 → /strategy
/strategy/workspace        → 301 → /strategy（无参数 = New）
```

**验收**：旧链接不 404，新路由全部可用。

### A5. 修复参数 bug

`StrategyLibraryPage.tsx` 传 `?template=`, Workspace 读 `?templateId=`。统一为 `templateId`。

---

## Phase B — 创建流（Gallery 即入口）

| 文件 | 操作 |
|------|------|
| `StrategyGalleryPage.tsx` | 工具栏加 [AI Generate] 按钮 → 跳转 `/strategy/:id/edit?ai=1`（Workspace 自动打开 AI 面板） |
| `StrategyWorkspacePage.tsx` | 接收 `?ai=1` 参数 → 自动打开 AI 面板；接收 `templateId` 参数 → 加载模板 |
| `StrategyGalleryPage.tsx` | [New] 按钮 → 跳转 `/strategy/:id/edit`（新模板 draft）|
| 卡片 [Fork] | → 调用 `createTemplateDraft` 复制模板 → 跳转 `/strategy/:newId/edit` |
| `StartPointModal.tsx` | **不建**（三方否决） |

**首访引导浮层**：新用户首次进 Workspace → 3-4 个 tooltip 指向代码编辑器、AI 面板、回测按钮、保存按钮。可跳过。

---

## Phase C — L2 状态重构（内部，UI 不变）

### C1. 拆分上帝 hook

| 当前（283行单体） | 目标（各域独立 hook） |
|-------------------|----------------------|
| `useStrategyWorkspaceState()` | `useCodeSlice()` / `useBacktestSlice()` / `useTuningSlice()` / `useAISlice()` / `useChartSlice()` / `useQuickTradeSlice()` / `useLayoutSlice()` / `useHistorySlice()` |

**迁移顺序（拓扑序）**：
1. `useCodeSlice` — 最独立，无跨域依赖
2. `useBacktestSlice` — 依赖 Code（code 变 → reset status）
3. `useAISlice` — 依赖 Code + Backtest（AI context）
4. `useTuningSlice` — 依赖 Backtest

**迁移方法**：Zustand slice-creator 模式。跨域协调用 selector 订阅（`useWorkspace(s => s.code)`），取代现在的 `useEffect` rewire。每迁一个 slice，跑 Playwright e2e 冒烟。

### C2. Workspace 页面退化

`StrategyWorkspacePage.tsx` → 纯组合根（~50 行）：渲染 7 个 `<SliceComponent/>`，不做任何业务逻辑。

### C3. 懒加载分包

```tsx
const StrategyWorkspacePage = React.lazy(() => import('./StrategyWorkspacePage'));
```

Gallery/Detail 的入口 chunk **不 import** Monaco editor（~600KB）、回测引擎、SSE 管线。仅 `/strategy/:id/edit` 路由才加载重包。交易员首屏预计 ~300KB。

---

## Phase D — 清理

| 文件 | 操作 |
|------|------|
| `StrategyLibraryPage.tsx` | 评估是否可完全删除（Gallery 已替代）|
| `StrategyTemplateEditModal.tsx` | **删除**（被 Detail + Workspace 替代）|
| `StrategyTemplateColumns.tsx` | **删除**（Table 列定义不再需要）|
| `TemplateManagerContent.tsx` | **删除**（旧 Workspace 侧边栏模板管理）|
| i18n | `strategy.library.*` → `strategy.gallery.*` 迁移；废弃 key 清理 |

---

## 分阶段依赖

```
Phase A（Gallery + Detail）
  ├── A1 后端 RPC ─────── 可并行
  ├── A2 Gallery 前端 ─── 依赖 A1 或降级
  ├── A3 Detail 前端 ──── 独立
  ├── A4 路由 ─────────── 依赖 A2+A3
  └── A5 bug fix ──────── 独立

Phase B（创建流）
  └── 依赖 Phase A2（Gallery 已上线）

Phase C（L2 重构）
  └── 不依赖 A/B，可独立开始

Phase D（清理）
  └── 依赖 A+B+C 全部上线
```

---

## 回滚策略

- **Phase A/B**：Gallery/Detail 与 Library 可并存（feature flag `USE_GALLERY`）。出问题切回 Library。
- **Phase C**：按 slice 逐个迁移。每迁一个跑 e2e，单个 slice 可独立回退。
