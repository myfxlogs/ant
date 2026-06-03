# Strategy Workspace 视觉复刻 v31.png（像素级 UI）落地方案

生成时间: 2026-06-03
目标: 把 `/opt/ant/reference/QuantDinger/docs/screenshots/v31.png`（QuantDinger Indicator IDE）**视觉上几乎一模一样**地复刻到 `http://localhost:8023/strategy/workspace`。
范围: **仅 UI 外观/布局/交互骨架**。不涉及后端引擎升级。数据未就绪的控件按"视觉占位"实现（样式齐全、行为 stub 或 disabled），保证整页看起来完整。
适用对象: 交给 AI 落地。

---

## 0. 关键认知

- v31 截图最左侧是 QuantDinger 的**应用级导航栏**（AI Asset Analysis / Indicator IDE / ...）。ant 有自己的 app shell 导航，**不复刻这条侧栏**，只复刻其右侧的**工作区内容区**。
- 当前 ant 工作区 `StrategyWorkspacePage.tsx` 已有骨架，但与 v31 有两处根本差异：
  1. **浅色主题**（`background: '#fff'`）→ v31 是**深色主题**。
  2. **Tab 切换式**（Chart & Trade / Backtest & Results / AI 三个 tab）→ v31 是**单屏三栏全展示**（代码 | 图表+回测参数 | Quick Trade 同屏可见）。
- 现有可复用组件: `WorkspaceCodePanel`、`WorkspaceChartTab`、`WorkspaceBacktestPanel`、`WorkspaceTemplateManager`、`QuickTradePanel`、`PriceChart`、`SymbolPicker`。复刻以**重排布局 + 换深色皮肤 + 补缺失区块**为主，不必推倒重来。
- 技术栈: React 19 + Ant Design + klinecharts。沿用项目现有约定（AntD 组件 + 内联样式 + i18n `t()`），见 `ant-ui-patterns` skill。

---

## 0.5 前置步骤（关键）：解除 1360px 宽度上限，让 workspace 全宽

> **这是布局复刻能否成立的前提，必须最先做。** 不做这一步，无论屏幕多大、三栏怎么写，workspace 都只会用 1360px 居中、两侧大片留白，永远做不出 v31 的铺满观感。

### 根因（已实测核实）
- v31.png 实测 **3827 × 1930**（≈4K，宽高比 ~2:1），用满全宽做三栏。
- ant 的 `MainLayout` 用 `ContentContainer` 包裹**所有路由**，而该容器写死 `maxWidth: 1360`：
  - `frontend/src/components/layout/ContentContainer.tsx`：`maxWidth: 1360, margin: '0 auto'`
  - `frontend/src/components/layout/MainLayout.tsx:70-74`：`<Content><ContentContainer><Outlet/></ContentContainer></Content>`
- 左侧固定侧栏 240px（`AppSidebar`）。
- 结论：**问题不是 4K、也不是架构能力不足，而是这一个 `maxWidth: 1360`**。它对仪表盘/表单是合理的可读性上限，但对 workspace 这种终端式三栏是错误约束。

### 各分辨率可容性（去掉上限后，可用宽 = 100vw - 240px）
| 屏幕 | 可用宽 | 三栏(代码25%/图表50%/QT25%) | 结论 |
|------|-------|------------------------------|------|
| 1440 | ~1200 | 300 / 600 / 300 | 偏紧可用 |
| 1920 | ~1680 | 420 / 840 / 420 | 舒适 |
| 2560 | ~2320 | 580 / 1160 / 580 | 宽松 |
| 3840(4K) | ~3600 | ≈ v31 | 与截图一致 |

### 方案 A（采用）：给 ContentContainer 加 `fluid` 开关，由路由控制
**不要全局删 1360**（会破坏仪表盘等普通页的可读性），仅让 workspace 这类全宽页绕过上限。

1. 改 `frontend/src/components/layout/ContentContainer.tsx`：
```tsx
import type { ReactNode } from 'react';

interface ContentContainerProps {
  children: ReactNode;
  fluid?: boolean;           // 全宽页（如 workspace）跳过 1360 上限
}

export default function ContentContainer({ children, fluid }: ContentContainerProps) {
  return (
    <div style={{ maxWidth: fluid ? 'none' : 1360, margin: '0 auto', width: '100%' }}>
      {children}
    </div>
  );
}
```

2. 改 `frontend/src/components/layout/MainLayout.tsx`：用 `useLocation` 判断当前路由是否属于"全宽白名单"，传 `fluid`。
```tsx
import { Outlet, useLocation } from 'react-router-dom';

const FULL_BLEED_ROUTES = ['/strategy/workspace'];   // 全宽页白名单，后续可扩展

// 组件内：
const location = useLocation();
const isFluid = FULL_BLEED_ROUTES.includes(location.pathname);
// ...
<Content className="pt-14 sm:pt-16 px-0">
  <ContentContainer fluid={isFluid}>
    <Outlet />
  </ContentContainer>
</Content>
```

### 配套
- workspace 根容器：全宽后用 `width: '100%'`，高度保持 `calc(100vh - <appHeader>)`；三栏各自 `overflow:auto`，确保外层不出现横向滚动条。
- 移动端（`isMobile`，<992px）：不强求三栏，可降级为纵向堆叠或保留原 Tab 形式（v31 是桌面终端形态）。

---

## 1. 目标布局（单屏三栏）

```
┌────────────────────────────────────────────────────────────────────────────┐
│ 顶部工具栏:  [Watchlist▼ BTC/USDT] [TF: 1m 5m 15m 1H 4H 1D 1W] [Indicator▼]   [⚡Quick Trade] │
├──────────────┬──────────────────────────────────────────────┬───────────────┤
│ 代码区(左,25%)│ 图表区(中,50%)                                  │ Quick Trade   │
│              │ [SMA EMA RSI MACD BB ATR CCI W%R MFI ADX ...]   │ (右,25%)      │
│ [Purchased]  │ OHLC info line                                  │ BTC/USDT $price│
│ 1 # code...  │ ┌──────────────────────────────────────────┐   │ Exchange:Gate │
│ 2 ...        │ │  K线 + EMA叠加 + B/S买卖点标记              │   │ [Long][Short] │
│ (代码编辑器)  │ │  volume 副图 / RSI 副图 / MACD 副图          │   │ Market|Limit  │
│              │ └──────────────────────────────────────────┘   │ Amount % ████ │
│ ──────────── │ ┌─ Backtest Parameters ───────[⚙][▶Run Backtest]│ Leverage ──── │
│ AI Generate  │ │ DateRange[1M 3M 6M 1Y]  Capital  Leverage     │ Margin:Cross  │
│ [textarea]   │ │ Commission Slippage   TradeDir[Long Short Both]│ TP / SL       │
│ [Generate]   │ │ □ High-Precision M1                            │               │
│              │ ├─ Tabs: [Backtest Results] [Smart Tuning] ──────┤               │
│              │ │ Smart Tuning: Structured scan  Grid|Random [Run]│               │
└──────────────┴──────────────────────────────────────────────┴───────────────┘
```

栏宽: 左 `minmax(280px, 25%)` / 中 `1fr` / 右 `minmax(300px, 25%)`。三栏均可独立滚动，整页不出现外层滚动条（高度 `calc(100vh - <appHeader>)`）。

**布局改造点**: 把现在的 `<Tabs>`（chart/backtest/ai）改为三栏 flex/grid 同屏。回测参数 + 结果/SmartTuning 放到**中栏图表下方**，不再是独立 tab。

---

## 2. 深色主题调色板（从 v31 提取）

| 用途 | 色值 | 说明 |
|------|------|------|
| 页面底色 | `#0b0e11` | 最外层工作区背景 |
| 面板/卡片底 | `#161a20` | 代码区/图表区/参数卡/QuickTrade |
| 卡片内层底 | `#1c2128` | 输入框、子卡片 |
| 边框 | `#262b31` | 卡片/分割线 |
| 主文字 | `#e6e8eb` | 标题/数值 |
| 次文字 | `#8b949e` | label/说明 |
| 弱文字 | `#5c636b` | 占位符 |
| 强调蓝(按钮) | `#2f6fed` | Quick Trade / Run Backtest / Generate |
| 涨/Long/Buy | `#26a69a`（候选 `#16c784`） | 阳线、Long 按钮、Ask 线 |
| 跌/Short/Sell | `#ef5350`（候选 `#f6465d`） | 阴线、Short 按钮、Bid 线 |
| TF 选中态 | `#2f6fed` 填充 + 白字 | 时间周期 segmented 选中 |
| B 标记 | 绿底白字小方块 | 买点 |
| S 标记 | 红底白字小方块 | 卖点 |

实现方式（**作用域隔离**，不污染全站浅色主题）:
- 最外层工作区包一个 `<ConfigProvider theme={{ algorithm: theme.darkAlgorithm, token: {...} }}>`，token 用上表色值（`colorBgContainer/#161a20`、`colorBorder/#262b31`、`colorPrimary/#2f6fed`、`colorBgLayout/#0b0e11` 等）。
- 容器根 div 设 `background:#0b0e11; color:#e6e8eb`。
- klinecharts 已有 `DARK_THEME`（见 `PriceChart.tsx`），与深色背景天然契合。

---

## 3. 分区构建规格

### 3.1 顶部工具栏
对应文件: `WorkspaceChartTab.tsx` 顶部 control row（现仅 Watchlist 组）。

补齐三组 + 右侧按钮，均用 `.ide-toolbar-group` 样式（深色化）：
- **Watchlist**: 账户 `Select` + `SymbolPicker`（已有，保留）。
- **Timeframe**: `Segmented` options `['1m','5m','15m','30m','1h','4h','1d','1w']`，标签显示 `1m 5m 15m 1H 4H 1D 1W`，绑定 `timeframe`（把现在藏在 PriceChart 里的周期切换提到工具栏）。
- **Indicator**: `Select`（可多选/下拉），展示如 "EMA9/EMA20 crossover"。控制图表叠加哪些指标（见 3.3）。
- 右侧: `[⚡ Quick Trade]` 主色按钮，切换右栏显隐（已有 `quickTradeVisible`）。

### 3.2 代码区（左栏）
对应文件: `WorkspaceCodePanel.tsx`（深色化）。
- 顶部徽标 `Purchased (read only)`（绿色 Tag）当模板为只读时显示。
- 工具条小图标（保存/校验/复制/运行）保留。
- 代码编辑器深色（若用 textarea/Monaco，套深色样式；行号灰 `#5c636b`）。
- **AI Generate 面板**（v31 代码区下方）: 新增区块——标题 `AI Generate`，多行 `Input.TextArea` 占位 "Describe the indicator you want to generate"，下方 `[Generate Code]` 主色按钮。功能可复用 `codeAssistApi.revise`（空 code + prompt），属低成本真功能。

### 3.3 图表区（中栏上半）
对应文件: `PriceChart.tsx` + `WorkspaceChartTab.tsx`。
- **指标标签行**: 图表上方一排 `Segmented`/`Tag` 多选：`SMA EMA RSI MACD BB ATR CCI W%R MFI ADX OBV ADOSC AD KDJ`。点击在 klinecharts 上 `createIndicator`/`removeIndicator`（klinecharts 内置大部分；不支持的先做纯视觉 Tag）。
- **OHLC info line**: 图表顶部显示 `Time / Open / High / Low / Close / Volume`（klinecharts 自带 tooltip 或自绘）。
- **K 线 + 叠加**: 阳绿阴红（DARK_THEME 已配）。EMA 等主图叠加。
- **B/S 买卖点标记**: 视觉用 klinecharts overlay（绿 B / 红 S 方块）。数据来源（回测 trades）**当前后端不返回逐笔**——视觉复刻阶段可用 **mock/示例标记**占位，并在代码注释标注 `TODO: 接回测逐笔`（见 `strategy-workspace-v31-implementation.md` 的引擎缺口）。

### 3.4 回测参数卡（中栏下半）
新增组件: `components/workspace/WorkspaceBacktestParams.tsx`。深色卡片，右上角 `[⚙]` 和 `[▶ Run Backtest]`（主色）。
- **Date Range**: `Segmented [1M 3M 6M 1Y]` + 下方自定义日期区间（`RangePicker` 深色）。
- **Capital**: `Initial Capital`（InputNumber）、`Commission`（InputNumber）。
- **Leverage**: InputNumber。
- **Trade Direction**: `Segmented [Long | Short | Both]`（Long 选中绿色）。
- **High-Precision M1**: `Switch` + 说明文字。
- 标注: Commission/Slippage/Leverage/Direction 当前后端引擎不读 → **视觉齐全，提交时仅传引擎支持的字段**；其余作为 UI 占位（可加 tooltip "coming soon"）。

### 3.5 Backtest Results / Smart Tuning 双 Tab（参数卡下）
- `Tabs [Backtest Results | Smart Tuning]`（深色 pill 样式，复用现有 `.ide-workspace-tabs` 改深色）。
- **Backtest Results**: 复用 `WorkspaceBacktestPanel`（深色化）。
- **Smart Tuning**（新增视觉区块 `WorkspaceSmartTuning.tsx`）: 标题 `Smart Tuning` + 说明 "Automatically search the optimal strategy parameters..."；`Structured scan (no LLM)` 区：`Segmented [Grid | Random]` + `[▶ Run Smart Tuning]` 按钮。**后端不存在**→ 纯视觉，按钮 disabled 或点击提示 "coming soon"。

### 3.6 Quick Trade（右栏）
对应文件: `QuickTradePanel.tsx`（已存在，深色化 + 补 v31 视觉元素）。
- 顶部: symbol `BTC/USDT` + 实时价 `$72,726`（大字）。
- `Exchange` 下拉（v31 是 Gate）——MT 场景无交易所概念：**视觉保留下拉，选项为账户/券商名**，或标注占位。
- `[↑ Long]` 绿 / `[↓ Short]` 红 大按钮。
- `Segmented [Market | Limit]`。
- `Amount (USDT)` 输入 + `[10% 25% 50% 75% 100%]` 按钮组。
- `Leverage` 滑块（MT 只读/占位）。
- `Margin mode [Cross | Isolated]`（MT 无对应 → 视觉占位）。
- `Take profit / Stop loss` 两个输入。
- 真功能部分（buy/sell/market/limit/lots/SL/TP）走已有 `tradingApi.orderSend`；crypto 专属控件（Exchange/Leverage/Margin）为视觉占位，标注 `TODO/disabled`。

---

## 4. 真功能 vs 视觉占位（一览，避免误判）

| 区块/控件 | 状态 | 落地方式 |
|----------|------|---------|
| 工具栏 Watchlist/TF/Indicator | 真 | 已有/小改 |
| 代码编辑器 + 校验/保存/复制/运行 | 真 | 已有 |
| AI Generate | 真 | 复用 `codeAssistApi.revise` |
| 图表 + 指标叠加 | 真 | klinecharts 内置 |
| 图表 B/S 标记 | **视觉占位** | mock 标记，待引擎返回逐笔 |
| 回测 Date Range / Capital / Run | 真 | 已有回测链路 |
| 回测 Commission/Slippage/Leverage/Direction/HighPrecision | **视觉占位** | UI 齐全，引擎暂不读 |
| Smart Tuning | **视觉占位** | 纯 UI，disabled |
| Quick Trade Long/Short/Market/Limit/Lots/TP/SL | 真 | `tradingApi.orderSend` |
| Quick Trade Exchange/Leverage/Margin mode | **视觉占位** | MT 无对应，disabled |

> 视觉占位控件务必**样式做满**（与 v31 一致），仅行为为 stub/disabled，并加 `title`/tooltip 说明，避免用户误以为故障。

---

## 5. 文件改动清单

| 文件 | 改动 |
|------|------|
| `frontend/src/components/layout/ContentContainer.tsx` | **前置**：加 `fluid` 开关，`maxWidth: fluid ? 'none' : 1360` |
| `frontend/src/components/layout/MainLayout.tsx` | **前置**：`useLocation` + 全宽白名单 → 给 `ContentContainer` 传 `fluid` |
| `frontend/src/pages/strategy/StrategyWorkspacePage.tsx` | 去 Tabs，改单屏三栏布局；外层套深色 `ConfigProvider`；根容器深色 + `width:100%` |
| `frontend/src/pages/strategy/components/workspace/WorkspaceChartTab.tsx` | 工具栏补 Timeframe `Segmented` + Indicator `Select`；图表上方指标标签行；深色化 |
| `frontend/src/pages/strategy/components/workspace/WorkspaceCodePanel.tsx` | 深色化；底部新增 AI Generate 面板 |
| `frontend/src/components/chart/PriceChart.tsx` | 指标叠加 props；B/S overlay（先 mock）；OHLC info line；深色已具备 |
| `frontend/src/components/chart/QuickTradePanel.tsx` | 深色化；补 Exchange/Amount%/Leverage/Margin/TP/SL 视觉元素 |
| **新增** `components/workspace/WorkspaceBacktestParams.tsx` | 回测参数卡（DateRange/Capital/Commission/Leverage/Direction/HighPrecision） |
| **新增** `components/workspace/WorkspaceSmartTuning.tsx` | Smart Tuning 视觉区块 |
| `components/workspace/WorkspaceBacktestPanel.tsx` | 深色化，置于 Backtest Results tab |

---

## 6. 验收标准（视觉）

- 整页深色，配色与 v31 接近（底 `#0b0e11`、卡 `#161a20`、强调蓝 `#2f6fed`、涨绿跌红）。
- 单屏三栏同时可见：左代码+AI Generate / 中图表+回测参数+结果/SmartTuning / 右 Quick Trade。
- 顶部工具栏含 Watchlist、Timeframe segmented、Indicator 下拉、Quick Trade 按钮。
- 图表上方有指标标签行，图上有 B/S 标记（mock 可），OHLC info line。
- 回测参数卡含日期区间预设、资金、手续费、杠杆、交易方向、High-Precision 开关。
- 有 Backtest Results / Smart Tuning 双 Tab。
- Quick Trade 含 Long/Short、Market/Limit、Amount%、Leverage、Margin mode、TP/SL。
- 与 v31 并排对比，布局/配色/控件位置基本一致；视觉占位控件外观完整、无报错。
- 全站其他页面浅色主题不受影响（深色作用域隔离）。

---

## 7. 实施顺序建议

0. **前置：解除 1360 宽度上限**（`ContentContainer` 加 `fluid` + `MainLayout` 白名单）——不做这步后面全白搭。
1. **深色主题 + 三栏布局骨架**（StrategyWorkspacePage 重排 + ConfigProvider）——确立整体观感。
2. **工具栏补全**（TF/Indicator）+ 图表指标标签行。
3. **回测参数卡 + Smart Tuning Tab**（视觉）。
4. **Quick Trade 深色化 + 补视觉元素**。
5. **代码区 AI Generate** + B/S mock 标记 + OHLC line 收尾。

> 功能层（B/S 真数据、回测真参数、Smart Tuning 后端）见 `strategy-workspace-v31-implementation.md` 的 Tier 2/3，本文档不含。
