# Strategy Workspace Redesign Spec

**日期**: 2026-07-03
**范围**: 策略工作区布局重构 + 全站暗色主题
**参考**: `/opt/ant/tmp/workspace-layout-mockup.html` (浏览器打开查看设计稿)

---

## 一、布局变更

### 现状 → 目标

| 现状 | 目标 |
|------|------|
| Code panel 是 750px 绝对定位遮罩，盖住图表 | **AI Chat 是 380px 永久侧边栏**，左栏固定，不盖图表 |
| 无历史对话 | **Chat 面板包含可滚动历史对话区**，每轮 AI 回复内嵌回测指标 |
| 回测在底部独立可拖拽面板 | **右栏 tab 切换：Results / Quick Trade** |
| 三栏抢空间 (32+750+flex+320) | **三栏固定比例 (260 + flex + 340)**，1366px 笔记本中栏 766px |
| PlanCard 埋在 AgentGenChat 第 8 层滚动流 | **PlanCard 固定在 Chat 顶部**，方案出来时置顶显示 |
| QuickTrade 独立右栏 | **QuickTrade 保留，在右栏第二个 tab** |

### 目标布局

```
┌─ 左栏 260px ─┬─ 中栏 flex:1 ─────────────────────────┬─ 右栏 340px ─┐
│ 策略列表       │  [设计] [代码] [回测]                    │ [Results] [Quick Trade] │
│  ├ Dual EMA   │                                        │              │
│  ├ Grid EUR   │  Tab=设计: K线图 (TradingView chart)     │ 指标卡片 6 个  │
│  └ + 新建      │  Tab=代码: Monaco 编辑器 (全高)          │ 净值曲线       │
│               │  Tab=回测: 回测详情 (全高)               │ 交易记录       │
│ AI Agents     │                                        │              │
│ 账户/品种/周期  ├────────────────────────────────────────│              │
│               │  AI Chat (固定在底部, 始终可见)          │              │
│               │  ┌─ PlanCard (固定顶部) ────────────┐  │              │
│               │  ├─ 历史对话区 (可滚动, max 40%) ────┤  │              │
│               │  │  You · 14:32                     │  │              │
│               │  │  Agent · 14:32 (含回测指标)       │  │              │
│               │  ├─ 当前流式输出 (max 100px) ────────┤  │              │
│               │  ├─ 输入框 + 发送按钮 (固定底部) ────┘  │              │
└───────────────┴───────────────────────────────────────┴──────────────┘
```

### 各区域尺寸约束

| 区域 | 宽度 | 高度 | 行为 |
|------|------|------|------|
| 左栏 | 260px, `flex-shrink: 0` | 100% | 固定 |
| 中栏 | `flex: 1 1 0`, `min-width: 0` | 100% | 弹性 |
| 右栏 | 340px, `flex-shrink: 0` | 100% | 固定 |
| 中栏 Tab 内容 | 100% | `flex: 1`, `min-height: 0` | 弹性, 放 K线/编辑器/回测 |
| AI Chat 面板 | 100% | `max-height: 40%`, `flex-shrink: 0` | 固定底部 |
| 历史对话区 | 100% | `flex: 1`, `max-height: 280px` | 可滚动 |
| 当前流式输出 | 100% | `max-height: 100px` | 可滚动 |

---

## 二、AI Chat 面板结构

从上到下四层：

1. **PlanCard（固定顶部）**：`plan != null` 时显示。包含策略类型/入场/出场/风控 + [确认生成] [修改] [换方案] 按钮。不跟随滚动。

2. **历史对话区（可滚动）**：已完成的对话轮次。每轮包含：
   - 用户消息（右对齐气泡，背景 `#1c2128`）
   - Agent 回复（左对齐气泡，背景 `#0d1117`，含内嵌回测指标行）
   - 时间戳（10px，颜色 `#484f58`）
   - 数据来源：`useWorkspaceSession` 已有的会话存储

3. **当前流式输出**：`GenerateStrategy` 进行中的实时输出。与历史区视觉分离。

4. **输入框 + 发送按钮（固定底部）**：3 行 textarea + 发送按钮。

---

## 三、暗色主题

### 实现方式

在 `frontend/src/providers/LocaleProvider.tsx` 的 `<ConfigProvider>` 加 `token`：

```tsx
<ConfigProvider
  locale={antdLocale || undefined}
  theme={{
    algorithm: isDark ? darkAlgorithm : defaultAlgorithm,
    token: isDark ? {
      colorBgBase: '#0d1117',
      colorBgContainer: '#161b22',
      colorBgElevated: '#1c2128',
      colorBorder: '#21262d',
      colorBorderSecondary: '#30363d',
      colorPrimary: '#58a6ff',
      colorSuccess: '#3fb950',
      colorError: '#f85149',
      colorWarning: '#d29922',
      colorText: '#c9d1d9',
      colorTextSecondary: '#8b949e',
      colorTextTertiary: '#6e7681',
      colorFill: 'rgba(88,166,255,0.12)',
      borderRadius: 6,
    } : undefined,
  }}
>
```

### 影响范围

- 所有 Ant Design 组件（Button, Card, Table, Select, Modal, Tabs, Tag, Input, Statistic, Progress...）自动继承
- 不需要改任何组件级 CSS

### 不自动覆盖的

- `style={{}}` 行内硬编码的颜色。已知位置：
  - `WorkspaceLayout.tsx`: border `"#e8e8e8"`
  - 所有 `style={{color: '#xxx', background: '#xxx'}}` 写死的值
  - **处理方式**：遇到时用 `token.colorXxx` 或 CSS 变量替换，不在本次强制全改

---

## 四、数据源（全部已有，无需新建）

| UI 元素 | 数据来源 | Proto 字段 |
|---------|---------|-----------|
| 回测指标 (6 StatCard) | `SubmitStrategy` / `GenerateStrategy` 返回 | `AgentBacktestResult.total_return, max_drawdown, sharpe_ratio, win_rate, profit_factor, total_trades` |
| 净值曲线 | 同上 | `AgentBacktestResult.equity_curve[] + equity_times_ms[]` |
| 交易记录 | 同上 | `AgentBacktestResult.trades[]` (ticket, side, volume, open/close price, pnl, reason) |
| 实时账户统计 | SSE 流 | 现有 `useAccountFinancials` 等 hooks |
| K 线图 | 现有 PriceChart 组件 | 现有 market data pipeline |
| 历史对话 | `useWorkspaceSession` | 已有会话持久化 |
| PlanCard | `AgentGenerateStrategyChunk.plan` | 已有，渲染为 `<PlanCard>` |
| Quick Trade | 现有 QuickTradePanel 组件 | 已有，移到右栏 tab |

---

## 五、现有组件改动清单

| 文件 | 改动 |
|------|------|
| `StrategyWorkspacePage.tsx` | **重构** — 三栏 flex 布局替换当前 overlay 模式 |
| `WorkspaceLayout.tsx` | **简化** — 移除 CODE_PANEL_WIDTH / POSITIONS_PANEL_WIDTH 等硬编码常量 |
| `AICodePanel.tsx` | **废弃** — 不再需要 overlay 遮罩，AI Chat 直接渲染在中栏 |
| `AgentGenChat.tsx` | **拆分** — PlanCard、历史对话区、当前流式输出、输入框 四个独立子组件 |
| `WorkspaceCodePanel.tsx` | **保留** — Monaco 编辑器，移到中栏"代码"tab |
| `BacktestPanel.tsx` | **保留** — 移到中栏"回测"tab |
| `QuickTradePanel.tsx` | **保留** — 移到右栏"Quick Trade"tab |
| `PriceChart` | **保留** — 移到中栏"设计"tab |
| `PlanCard.tsx` | **保留** — 改为固定位置渲染，不再埋在滚动流中 |
| `LocaleProvider.tsx` | **加 token** — 50 行 GitHub Dark 色板 |
| `workspaceStore.ts` | **简化** — 移除 `codePanelOpen` 等 overlay 状态，改为 `centerTab` |
| `MemoryPage.tsx` | 已实现 |
| `AdminSettingsPage.tsx` | 已实现 |

### 新增文件

| 文件 | 职责 |
|------|------|
| `ChatHistory.tsx` | 可滚动历史对话列表组件 |
| `ChatInput.tsx` | 输入框 + 发送按钮组件（从 AgentGenChat 拆出） |

---

## 六、验证方式

1. 浏览器打开 `tmp/workspace-layout-mockup.html` 看设计稿
2. 开发环境打开策略工作区，验证：
   - 左栏策略列表可点击切换
   - 中栏 "设计" tab 显示 K 线图 + AI Chat
   - 中栏 "代码" tab 显示 Monaco 编辑器 + AI Chat（Chat 不消失）
   - 中栏 "回测" tab 显示回测详情 + AI Chat
   - 右栏 Results tab 显示回测指标（拉到真实数据）
   - 右栏 Quick Trade tab 显示下单面板
   - AI Chat 中 PlanCard 置顶显示
   - AI Chat 历史对话区可滚动查看之前的对话
   - 全站组件颜色变为 GitHub Dark 色板
   - 1366x768 分辨率下三栏比例合理（中栏 > 700px）
   - 1920x1080 分辨率下布局充裕
