# ADR-0027 落地实施指南

- **状态**：**已完成**（2026-07-27）。Phase A-D + E1/E2 全部落地。三方共识见 `0027-decision-matrix.md`，设计依据见 `0027-v2-strategy-module-redesign.md`。
- **策略**：一步到位，不做过渡。Gallery 首版即用 `ListStrategyCards`，Library 同日删除。所有工作在最终形态上一次完成。

## 并行策略

两组人可并行推进，互不阻塞：

```
后端（1 人）               前端（1 人）
────────                   ────────
ListStrategyCards RPC      StrategyCard + Detail（独立，mock 数据）
                           等 RPC 联调 → Gallery
                           路由 + 导航
                           删除 Library + 旧文件
                           Workspace feature-slice 拆分
```

---

## 第一步：修 bug（1 行，立刻做）

| 文件 | 行 | 改法 |
|------|-----|------|
| `StrategyLibraryPage.tsx` | L69 | `?template=` → `?templateId=` |

**验收**：Library 页面 [Open in Workspace] 点后 Workspace 正确加载模板。

---

## 第二步：后端 — `ListStrategyCards` RPC

| 文件 | 操作 |
|------|------|
| `proto/ant/v1/strategy.proto` | 新增 `StrategyCard` message + `ListStrategyCards` RPC |

```proto
message StrategyCard {
  string id = 1;
  string name = 2;
  string description = 3;
  repeated string tags = 4;
  bool is_system = 5;
  bool is_public = 6;
  bool is_published_to_market = 7;
  string asset_class = 8;
  string risk_level = 9;
  // 反范式化摘要（最近一次成功回测）
  repeated double equity_sparkline = 10;  // 归一化点（0-1），画迷你曲线
  string win_rate = 11;     // decimal string
  string max_drawdown = 12;
  string profit_factor = 13;
  string sharpe = 14;
  string total_return = 15;
  int32 total_trades = 16;
  int32 running_schedules = 17;
  int64 updated_at_ms = 18;
}
message ListStrategyCardsRequest {
  string filter = 1;   // "all" / "mine" / "preset"
  string sort = 2;     // "return" / "risk" / "usage" / "recent"
  string search = 3;
}
message ListStrategyCardsResponse {
  repeated StrategyCard cards = 1;
}
```

**Handler 实现**：单次 SQL JOIN `strategy_templates` + 最近成功回测 + `schedules` 计数。Decimal 一律走 string。无 N+1。

**验收**：`grpcurl ListStrategyCards` 返回全部模板，每张卡有 sparkline、KPI、running_schedules。

---

## 第三步：前端 — 最终形态一次性到位

### 3a. 新建 StrategyCard 组件

| 文件 | 操作 |
|------|------|
| `StrategyCard.tsx` | **新建** |

```
卡片内容：
┌─────────────────────────────────┐
│ [tag] Name                      │
│ description (1 行省略)           │
│ ═══════════════════════════════ │
│ ▁▂▃▄▅▆▇ sparkline (recharts)   │
│                                 │
│ Win 72%  DD -8%  PF 2.1  SR 1.5│
│ 3 running                       │
│                                 │
│ [Detail] [Deploy] [Fork] [...]  │
└─────────────────────────────────┘
```

- sparkline：归一化 equity 点，recharts `<Area>` 或 `<Line>`，颜色跟随总收益正负
- KPI 行：4 个指标，紧凑排列
- Actions 行：按模板类型裁剪（见下表）

**卡片 actions 矩阵**：

| 模板类型 | 按钮 |
|---------|------|
| 系统 | `[Detail]` |
| 我的·未发布 | `[Detail] [Deploy] [Fork] [Publish] [Delete]` |
| 我的·已发布 | `[Detail] [Deploy] [Fork] [Unpublish] [Delete]` |

- `[Deploy]` → `ScheduleLaunchModal`，选账户直接上线
- `[Publish]` → `PublishToMarketModal`
- `[Fork]` → `createTemplateDraft` 复制 + 跳转 `/:newId/edit`
- `[Delete]` → `Popconfirm` → `deleteTemplate`

### 3b. 新建 Gallery 页面

| 文件 | 操作 |
|------|------|
| `StrategyGalleryPage.tsx` | **新建** |

```
顶部工具栏：[🔍 搜索] [All | My | Preset] [排序 ▾]      [+ New] [AI Generate]
卡片区：  <Row gutter={[16,16]}>  {cards.map(StrategyCard)}  </Row>
响应式：  <Col xs={24} sm={12} md={8} lg={6}>
```

- 数据源：`useQuery(queryKeys.strategyCards.list, () => strategyApi.listStrategyCards({filter, sort, search}))`
- 搜索/筛选/排序逻辑直接在新页面实现（不继承 LibraryPage，不引入中间抽象层）
- `[+ New]` → 调 `createTemplateDraft` → 跳转 `/:newId/edit`
- `[AI Generate]` → 跳转 `/strategy/workspace?ai=1`（Workspace 自动打开 AI 面板）

### 3c. 新建 Detail 页面

| 文件 | 操作 |
|------|------|
| `StrategyDetailPage.tsx` | **新建** |

**2 个 Tab**：

| Tab | 内容 | 说明 |
|-----|------|------|
| Overview | 完整权益曲线（recharts）+ 交易统计表（胜率/盈亏比/回撤/PF/夏普/年化收益/总交易）+ 参数说明 | `getTemplate` + 最近回测摘要 |
| Code | `<pre>` + highlight.js（~20KB）语法高亮。系统模板只读；用户模板 [Fork & Edit] | **不 import Monaco** |

底部固定栏：`[Open in Workspace]` 按钮 → `/strategy/:id/edit`

### 3d. 路由替换

| 旧路由 | 新路由 | 操作 |
|--------|--------|------|
| `/strategy/library` | — | 删除路由 + `Navigate` 301 → `/strategy` |
| `/strategy/workspace` | — | 删除路由 + `Navigate` 301 → `/strategy`（无参数 = new） |
| — | `/strategy` | **新增** → `StrategyGalleryPage` |
| — | `/strategy/view/:id` | **新增** → `StrategyDetailPage`（`view` 静态段避让 `:strategyId`） |
| — | `/strategy/:id/edit` | **新增** → `StrategyWorkspacePage`（加载 `templateId`） |
| `/strategy/live` | `/strategy/live` | 不变 |
| `/strategy/market-tools` | `/strategy/market-tools` | 不变 |

### 3e. 侧边栏

```
Strategy
├── Strategies           ← /strategy（Gallery，改名）
├── Strategy Workspace  ← /strategy/workspace（保留入口，无参=新建）
├── Live Monitor        ← 不变
├── Market Tools        ← 不变
```

### 3f. 删除

| 文件 | 理由 |
|------|------|
| `StrategyLibraryPage.tsx` | Gallery 替代，不复用（3b 直接内联实现搜索/筛选逻辑，不引入中间抽象） |
| `StrategyTemplateColumns.tsx` | Table 列定义不再需要 |
| `StrategyTemplateEditModal.tsx` | Detail + Workspace 替代 |

---

## 第四步：Workspace 增强 + Feature-slice 拆分

### 4a. 路由参数支持

`StrategyWorkspacePage` 接收两个参数：
- `?templateId=X` → 加载指定模板（替换旧 `?template=` bug）
- `?ai=1` → 自动打开右侧 AI 面板

### 4b. 首访引导浮层

新用户首次进 Workspace → 3-4 个 tooltip（Ant `Tour` 组件），指向代码编辑器、AI 面板、回测按钮、保存按钮。可跳过。`localStorage` 记 `workspace_tour_seen=true`。

### 4c. Feature-slice 拆分

**目标**：283 行 `useStrategyWorkspaceState` → 各域独立 hook + Zustand slice。

| 当前 | 目标 |
|------|------|
| `useStrategyWorkspaceState()`（10 域大对象）| `useCodeSlice()` `useBacktestSlice()` `useTuningSlice()` `useAISlice()` `useChartSlice()` `useQuickTradeSlice()` `useLayoutSlice()` `useHistorySlice()` |

**迁移顺序**：Code（无上游依赖）→ Backtest（依赖 Code）→ AI（依赖 Code+Backtest）→ Tuning（依赖 Backtest）。每迁一个 slice 跑 Playwright e2e 冒烟。

**Store 结构**（Zustand slice-creator）：
```ts
// stores/workspace/index.ts
export const useWorkspace = create<CodeSlice & LayoutSlice & BacktestSlice & /*...*/>()(
  persist(
    (...a) => ({ ...createCodeSlice(...a), ...createLayoutSlice(...a), /*...*/ }),
    { name: 'ant-workspace-v6', partialize: s => ({ centerTab: s.centerTab, rightPanelWidth: s.rightPanelWidth }) },
  ),
);
```

**跨域协调**：selector 订阅取代 useEffect。例：`useWorkspace(s => s.code)` 变化时，backtest slice 自动 `resetStatus()`。

Workspace 页面退化为组合根（~50 行）：
```tsx
export default function StrategyWorkspacePage() {
  return (
    <div>
      <WorkspaceToolbar />
      <InnerLayout>
        <WorkspaceCenterColumn />
        <ResizeHandle />
        <RightPanel />
      </InnerLayout>
      <ModalsAndDrawers />
    </div>
  );
}
```

### 4d. 懒加载

```tsx
// AppRoutes.tsx
const StrategyWorkspacePage = lazy(() => import('./StrategyWorkspacePage'));
```

Gallery/Detail 的入口 chunk **不 import** Monaco（~600KB）、回测引擎、SSE 管线。仅 `/:id/edit` 路由加载 Workspace 重包。

---

## 验收清单

| # | 验收项 | 方式 |
|---|--------|------|
| 1 | `ListStrategyCards` 返回全部模板，含 sparkline 和 KPI | `grpcurl` |
| 2 | Gallery 卡片正确渲染 sparkline/KPI/actions | Playwright |
| 3 | 点击 [Detail] 跳转 `/strategy/view/:id`，Overview+Code Tab 正常 | Playwright |
| 4 | 点击 [Deploy] 拉起 ScheduleLaunchModal | Playwright |
| 5 | 点击 [Fork & Edit] 跳转 Workspace 并加载代码 | Playwright |
| 6 | `/strategy/library` → 301 `/strategy` | curl |
| 7 | 旧链接不 404 | Playwright |
| 8 | Workspace 懒加载——Gallery 首屏不加载 Monaco | Lighthouse |
| 9 | Feature-slice 拆分后 Playwright 回测流程通过 | Playwright |
| 10 | 删除 Library/TemplateColumns/EditModal 后编译通过 | `npm run build` |
