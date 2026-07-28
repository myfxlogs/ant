# ADR-0027 · 策略模块前端重构 — Gallery + Detail + Guided Create

- **状态**：**Superseded** by `0027-v2-strategy-module-redesign.md`（Cascade 修订版，已实现）。本文件保留作为历史参考。
- **日期**：2026-07-23
- **决策者**：Team
- **关联**：ADR-0024，ADR-0021

## 1. 背景

### 1.1 现状（2026-07-23 截止）

策略模块四个二级页面：Workspace（编辑+回测+AI）、Library（模板表格，今天刚实现）、Live（实盘）、Market Tools。

**今日新增**：

- `StrategyLibraryPage.tsx`（187 行 Table 列表：搜索/筛选/Open/删除）——此为 Library 的首次实现
- MQL Import Drawer 已接入 Workspace Code Tab（`ImportEAPanel` — 粘贴 MQL → 分析/翻译/桥接 → Apply 写入编辑器）

Workspace 侧边栏已无 `TemplateManagerContent`（此前移除），Library 页面是唯一的模板管理入口。

### 1.2 问题

| 问题 | 严重度 | 说明 |
| ---- | ------ | ---- |
| 模板管理分裂 | 🔴 | Library 页面和 Workspace 侧边栏同时管理模板 |
| 表格不适合浏览 | 🔴 | 纯文字表格，无性能数据、无可视化 |
| 巨型 Modal 编辑 | 🟡 | 1280px Modal 塞 Metadata+Code+AI+Validation |
| 代码优先非结果优先 | 🟡 | 用户先看到 MQL 代码，回测结果藏在 Tab 后 |
| 无引导式创建 | 🟡 | 点创建直接丢进复杂 Workspace |
| 移动端不可用 | 🟡 | 表格和三栏布局小屏不可用 |

### 1.3 用户画像

- **交易员**（主要）：不写代码，关注收益曲线、胜率、回撤
- **策略开发者**（次要）：写 MQL，需要编辑器+AI+回测
- **管理员**：管理系统模板

## 2. 决策

采用 Gallery + Detail + Guided Create 三层架构。

### 2.1 路由结构

```text
/strategy              → Gallery（卡片网格）
/strategy/:id          → Detail（Tab：概览/代码/回测/部署）
/strategy/create       → Guided Create（Stepper 引导）
/strategy/workspace    → Workspace（编辑+回测，从 Detail "Fork & Edit" 跳入）
/strategy/live         → Live Monitor（不变）
/strategy/market-tools → Market Tools（不变）
```

### 2.2 Gallery 页面

替代当前 Library 表格。

**卡片内容**：名称+标签、描述、迷你权益曲线 sparkline、胜率/回撤/收益因子、运行中调度数、详情/部署按钮。

**顶部工具栏**：搜索、筛选（全部/我的/预设/社区）、排序（收益/回撤/时间/使用次数）、创建按钮。

**响应式**：桌面4列、平板2列、手机1列。

**数据来源**：`listTemplates()` + 最近回测摘要 + schedules 统计。

### 2.3 Detail 页面

替代当前编辑 Modal。

**概览 Tab**：描述、完整权益曲线、交易统计（胜率/盈亏比/回撤/PF/夏普）、参数说明表格。

**代码 Tab**：语法高亮只读查看。系统模板纯只读；用户模板有 "Fork & Edit" → 跳转 Workspace。含代码解释面板。

**回测 Tab**：选品种/周期/时间范围 → 运行 → 结果展示（权益曲线+交易列表+统计）。复用现有 BacktestParamsModal + WorkspaceBacktestPanel 逻辑。

**部署 Tab**：选账户 → 设参数 → 上线调度。复用现有 ScheduleLaunchModal 逻辑。

### 2.4 Guided Create 页面

替代当前"点创建→丢进 Workspace"。

**Step 1 选择起点**：空白 / 导入 EA / AI 生成 / Fork 现有模板（4 个大卡片选择）

**Step 2 编写代码**：代码编辑器 + 实时验证 + AI 辅助（精简版 Workspace，只保留编辑+验证+AI）

**Step 3 回测验证**：选参数 → 运行 → 查看结果（必须通过验证才能进入下一步）

**Step 4 保存**：填名称/描述/可见性 → 完成 → 跳转 Detail 页面

顶部 Stepper 显示进度，每步可回退。

### 2.5 Workspace 简化

- 移除侧边栏 `TemplateManagerContent`（Gallery 已接管浏览/管理）
- 专注代码编辑 + 回测 + AI 辅助
- 从 Detail "Fork & Edit" 或 Guided Create Step 2 跳入时自动加载代码
- 保留版本历史、指标目录、导入 EA 等 Drawer 功能

## 3. 备选方案

| 方案 | 优点 | 缺点 | 否决理由 |
| ---- | ---- | ---- | -------- |
| 保持单页 Tab 整合 | 路由简单 | 单页过重，无法分享链接，状态管理复杂 | 职责不清晰 |
| 仅优化表格 | 改动小 | 表格本质不适合策略浏览 | 治标不治本 |
| 完全推倒重做 Workspace | 彻底 | 风险大，Workspace 编辑功能成熟 | 保留 Workspace 编辑能力 |

## 4. 后果

### 正面

- 职责清晰：Gallery 浏览、Detail 深入、Workspace 编辑
- 路由可分享（`/strategy/:id` 直接发链接）
- 性能数据前置，交易员友好
- 引导式创建，新用户友好
- 移动端友好（卡片网格自适应）

### 负面

- 新增 2 个路由 + 3 个页面组件
- 需要后端新增模板回测摘要 RPC（或前端聚合）
- Workspace 需移除侧边栏模板管理，影响现有用户习惯

### 中性

- i18n 需新增 `strategy.gallery.*` 和 `strategy.detail.*` key
- 现有 `strategy.library.*` key 可逐步迁移或保留兼容

## 5. 实施计划

### Phase 1：Gallery + Detail（核心）

1. 新建 `StrategyGalleryPage.tsx` — 卡片网格 + 搜索/筛选/排序
2. 新建 `StrategyDetailPage.tsx` — Tab 切换（概览/代码/回测/部署）
3. 新建 `StrategyCard.tsx` — 单个策略卡片组件（sparkline + 指标）
4. 路由调整：`/strategy` → Gallery，`/strategy/:id` → Detail
5. 后端：新增 `GetTemplateStats` RPC 返回最近回测摘要
6. i18n：新增 `strategy.gallery.*`、`strategy.detail.*` key

### Phase 2：Guided Create

1. 新建 `StrategyCreatePage.tsx` — 4 步 Stepper
2. Step 2 复用 CodeEditorPanel + AIPanel + ValidationResults
3. Step 3 复用 BacktestParamsModal + 回测结果展示
4. 路由：`/strategy/create`

### Phase 3：Workspace 精简

1. 移除 `TemplateManagerContent` 及相关导入
2. 清理 `useLibraryTemplates` 中 Workspace 侧边栏专用逻辑
3. Workspace 入口仅从 Detail "Fork & Edit" 或 Guided Create 跳入

### Phase 4：清理

1. 删除 `StrategyLibraryPage.tsx`（被 Gallery 替代）
2. 删除 `StrategyTemplateEditModal.tsx`（被 Detail + Guided Create 替代）
3. 删除 `StrategyTemplateColumns.tsx`（表格列定义不再需要）
4. 迁移 i18n key，清理废弃 key

## 6. 技术约束

- 前端框架：React + Ant Design + react-router-dom
- 状态管理：Zustand（workspaceStore）+ React Query（数据获取）
- 图表：复用现有权益曲线组件
- Sparkline：轻量 SVG 或 recharts `<Sparkline>`
- i18n：遵循现有 textproto + map.json 流程
- proto-only：所有 API 走 ConnectRPC，无 REST
