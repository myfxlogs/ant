# Strategy Workspace 对齐 QuantDinger Indicator IDE (v31) 落地方案

> **⚠️ 注意：** 本文档中部分代码路径已按 ADR-0021 更新：
> - `PythonStrategyServer` → `StrategyRuntimeService`
> - `python_strategy_handler.go` → `strategy_runtime_handler.go`
> - `client/pythonStrategy.ts` → `client/strategyRuntime.ts`
> - Python 策略执行器 → Go SDK（`backend/strategy/sdk/`）

生成时间: 2026-06-01
目标: 将 `/opt/ant/reference/QuantDinger/docs/screenshots/v31.png`（QuantDinger Indicator IDE）的能力，落地到 ant 现有 `StrategyWorkspacePage`。
适用对象: 交给 AI 按阶段实施。每个阶段含 proto / backend / frontend 改动点与验收标准。

---

## 0. 截图功能拆解（v31.png）

QuantDinger Indicator IDE 由四个区域组成：

| 区域 | 截图内容 | ant 现状 |
|------|---------|---------|
| 顶部工具栏 | Watchlist(BTC/USDT) + Timeframe + Indicator 下拉 + Quick Trade 按钮 | 部分（账户/品种/周期在 Chart Tab 内） |
| 左侧 | 代码编辑器 + AI Generate（自然语言生成指标） | 代码编辑器 ✓ / AI 仅 revise+explain，无 generate |
| 中部图表 | K 线 + 指标叠加(SMA/EMA/RSI/MACD/BB/ATR...) + 买卖点 B/S 标记 + 成交量 | 仅裸 K 线 + 成交量，无指标、无信号标记 |
| 中下回测面板 | Backtest Parameters（日期区间/资金/杠杆/手续费/滑点/交易方向/高精度M1）+ Backtest Results / Smart Tuning 双 Tab | 仅展示结果，无参数面板；硬编码 capital=10000；**且 startBacktestRun 调用缺 mode 参数（bug）** |
| 右侧 | Quick Trade 面板（Long/Short, Market/Limit, 数量%, 杠杆, 保证金模式, TP/SL） | 占位符 "coming soon" |

**关键差异**: QuantDinger 面向加密交易所（Gate、Cross/Isolated 杠杆），ant 面向 MT4/MT5 外汇。下单语义需映射：`Long/Short → buy/sell`、数量用手数(lots)、杠杆/保证金为账户级只读、TP/SL 直接复用 `PlaceOrder` 的 stop_loss/take_profit。

---

## 1. 现有可复用能力盘点（重要：避免重复造轮子）

| 能力 | 位置 | 复用方式 |
|------|------|---------|
| 下单 (含风控/OMS/幂等/成本估算) | `backend/internal/mthub/service.go` `PlaceOrder` → `MtHubServer.PlaceOrder` | 直接复用，前端 `tradingApi.orderSend()` 已就绪 |
| 平仓 | `MtHubServer.CloseOrder` / `tradingApi.orderClose()` | 直接复用 |
| 下单表单参考实现 | `frontend/src/pages/trading/components/PlaceOrderForm.tsx` | 作为 Quick Trade 蓝本 |
| K 线图 | `frontend/src/components/chart/PriceChart.tsx` (klinecharts) | 扩展指标/标记 |
| 异步回测 + SSE 监听 | `PythonStrategyServer.StartBacktestRun` + `watchBacktestRun` | 扩展参数 |
| 代码校验/参数抽取 | `codeAssistApi.validateExtended()` | 直接复用 |
| AI 改写/解释 | `codeAssistApi.revise()/explain()` | 扩展 generate |
| 模板 CRUD | `strategyApi.*Template*` | 直接复用 |

MT 网关下单链路（invariant #8）: 业务代码经 `mthub` (L5) → adapter (L2) → mtapi.io，**不可直调 mt4client/mt5client**。`PlaceOrder` 已遵循此约束。

---

## 2. 实施阶段总览

| 阶段 | 模块 | 复杂度 | 依赖 | 说明 |
|------|------|-------|------|------|
| P0 | 修复回测调用 bug + 接线 | 低 | 无 | 先修 `startBacktestRun` 缺 `mode` |
| P1 | Quick Trade 面板 | 中 | 无（后端就绪） | 纯前端，复用 `tradingApi` |
| P2 | 回测参数面板 | 中 | proto 扩展 | 日期/资金/杠杆/手续费/滑点/方向/高精度 |
| P3 | 图表指标 + 买卖点标记 | 中 | P2（trades） | klinecharts indicator + overlay |
| P4 | AI Generate（自然语言生成） | 低-中 | 无 | 扩展 codeAssist |
| P5 | Smart Tuning（参数寻优） | 高 | P2 | 新后端服务，可独立里程碑 |

---

## P0. 修复回测调用 bug 并补全接线

**问题**: `StrategyWorkspacePage.handleRunBacktest` 调用
```ts
pythonStrategyApi.startBacktestRun({ code, accountId, symbol, timeframe, initialCapital: 10000 })
```
但 `startBacktestRun` 的类型签名要求 `mode: 'KLINE_RANGE' | 'DATASET'`（见 `frontend/src/client/pythonStrategy.ts:127`）。缺失 `mode` 会导致类型错误/运行时 mode 为 undefined → 后端默认 `KLINE_RANGE` 但 `from/to` 为空。

**修复** (`frontend/src/pages/strategy/StrategyWorkspacePage.tsx`):
```ts
const result = await pythonStrategyApi.startBacktestRun({
  code, accountId, symbol, timeframe,
  initialCapital: backtestParams.initialCapital,
  mode: 'KLINE_RANGE',
  from: backtestParams.from,   // Date
  to: backtestParams.to,       // Date
});
```
（`backtestParams` 由 P2 的参数面板提供；P0 阶段可先用默认最近 3 个月。）

**验收**: 回测可正常发起，SSE 收到状态流转，无类型报错。

---

## P1. Quick Trade 面板（替换占位符）

### 设计
将 `WorkspaceChartTab.tsx` 中 "Trade panel — coming soon" 占位符替换为真实下单面板。MT 语义映射：

| QuantDinger | ant (MT) |
|-------------|----------|
| Long / Short | side = buy / sell |
| Market / Limit | orderType = market / limit |
| Amount 10/25/50/75/100% | 按可用保证金(free margin)估算手数，或直接手数输入 |
| Leverage / Margin mode | 账户级只读，展示即可（MT 不支持下单时改杠杆） |
| Take Profit / Stop Loss | stop_loss / take_profit |

### 新建组件 `frontend/src/pages/strategy/components/workspace/WorkspaceQuickTrade.tsx`
```tsx
import { useState } from 'react';
import { Card, Radio, InputNumber, Button, Space, Segmented, Statistic, message } from 'antd';
import { useTranslation } from 'react-i18next';
import { tradingApi } from '@/client/trading';
import { getErrorMessage } from '@/utils/error';

interface Props {
  accountId: string;
  symbol: string;
  lastPrice?: number;       // 来自图表最新价（P3 可回填）
  freeMargin?: number;      // 来自账户 SSE（可选）
}

export default function WorkspaceQuickTrade({ accountId, symbol, lastPrice, freeMargin }: Props) {
  const { t } = useTranslation();
  const [side, setSide] = useState<'buy' | 'sell'>('buy');
  const [orderType, setOrderType] = useState<'market' | 'limit'>('market');
  const [volume, setVolume] = useState(0.01);
  const [price, setPrice] = useState<number | undefined>(undefined);
  const [sl, setSl] = useState<number | undefined>(undefined);
  const [tp, setTp] = useState<number | undefined>(undefined);
  const [submitting, setSubmitting] = useState(false);

  const submit = async () => {
    if (!accountId || !symbol) {
      message.warning(t('strategy.workspace.quickTradeNeedSymbol', 'Select account and symbol first'));
      return;
    }
    setSubmitting(true);
    try {
      const type = orderType === 'market'
        ? side
        : `${side}_limit`;
      const res = await tradingApi.orderSend({
        accountId, symbol, type, volume,
        price: orderType === 'limit' ? price : undefined,
        stopLoss: sl, takeProfit: tp,
        comment: 'workspace-quicktrade',
      });
      if (res.error) {
        message.error(res.message || res.error);
      } else {
        message.success(t('strategy.workspace.orderSent', 'Order submitted'));
      }
    } catch (e) {
      message.error(getErrorMessage(e, 'Order failed'));
    } finally {
      setSubmitting(false);
    }
  };

  const pctButtons = [10, 25, 50, 75, 100];
  const applyPct = (pct: number) => {
    // 简化：按 freeMargin 粗略折算手数（精确折算需 symbol contract size / margin rate）。
    if (!freeMargin) return;
    const est = Math.max(0.01, Math.floor((freeMargin * pct / 100) / 1000) * 0.01);
    setVolume(est);
  };

  return (
    <Card size="small" title={t('strategy.workspace.quickTrade', 'Quick Trade')}>
      <Space direction="vertical" style={{ width: '100%' }} size="small">
        {lastPrice != null && (
          <Statistic value={lastPrice} precision={5} valueStyle={{ fontSize: 18 }} />
        )}
        <Radio.Group value={side} onChange={(e) => setSide(e.target.value)} buttonStyle="solid" style={{ width: '100%' }}>
          <Radio.Button value="buy" style={{ width: '50%', textAlign: 'center', color: '#52c41a' }}>
            {t('strategy.workspace.long', 'Long')}
          </Radio.Button>
          <Radio.Button value="sell" style={{ width: '50%', textAlign: 'center', color: '#ff4d4f' }}>
            {t('strategy.workspace.short', 'Short')}
          </Radio.Button>
        </Radio.Group>
        <Segmented
          block
          value={orderType}
          onChange={(v) => setOrderType(v as 'market' | 'limit')}
          options={[
            { label: t('trading.market', 'Market'), value: 'market' },
            { label: t('trading.limit', 'Limit'), value: 'limit' },
          ]}
        />
        <div>
          <span style={{ fontSize: 12, color: '#8c8c8c' }}>{t('trading.volume', 'Volume (lots)')}</span>
          <InputNumber min={0.01} step={0.01} value={volume} onChange={(v) => setVolume(v || 0.01)} style={{ width: '100%' }} />
        </div>
        {freeMargin != null && (
          <Space size={4}>
            {pctButtons.map((p) => (
              <Button key={p} size="small" onClick={() => applyPct(p)}>{p}%</Button>
            ))}
          </Space>
        )}
        {orderType === 'limit' && (
          <div>
            <span style={{ fontSize: 12, color: '#8c8c8c' }}>{t('trading.price', 'Price')}</span>
            <InputNumber value={price} onChange={(v) => setPrice(v ?? undefined)} style={{ width: '100%' }} />
          </div>
        )}
        <Space>
          <InputNumber placeholder="SL" value={sl} onChange={(v) => setSl(v ?? undefined)} />
          <InputNumber placeholder="TP" value={tp} onChange={(v) => setTp(v ?? undefined)} />
        </Space>
        <Button type="primary" block danger={side === 'sell'} loading={submitting} onClick={submit}>
          {side === 'buy' ? t('strategy.workspace.long', 'Long') : t('strategy.workspace.short', 'Short')}
        </Button>
      </Space>
    </Card>
  );
}
```

### 接线 `WorkspaceChartTab.tsx`
将占位 div 替换为 `<WorkspaceQuickTrade accountId={accountId} symbol={symbol} />`。

**验收**:
- 选定账户+品种后可下市价/限价单，SL/TP 生效。
- 下单走 `tradingApi.orderSend` → `MtHubServer.PlaceOrder`，风控/OMS 链路不变。
- 卖单按钮 danger 样式，错误以 toast 提示。

**注意**: 杠杆/保证金模式在 MT 下单时不可变更，UI 只读展示或隐藏，避免误导。

---

## P2. 回测参数面板（Backtest Parameters）

### proto 扩展 `proto/ant/v1/backtest_run_start.proto`
当前 `StartBacktestRunRequest` 无手续费/滑点/杠杆/方向字段。新增（向后兼容，全部 optional/默认 0）：
```proto
message StartBacktestRunRequest {
  string code = 1;
  string account_id = 2;
  string symbol = 3;
  string timeframe = 4;
  double initial_capital = 5;
  BacktestRunMode mode = 6;
  optional google.protobuf.Timestamp from = 7;
  optional google.protobuf.Timestamp to = 8;
  optional string dataset_id = 9;
  optional string template_id = 10;
  optional string template_draft_id = 11;
  repeated string extra_symbols = 12;
  // v31: 回测参数面板
  double commission = 13;        // 手续费率，默认 0.001
  double slippage = 14;          // 滑点，默认 0
  int32 leverage = 15;           // 杠杆，默认 1
  string trade_direction = 16;   // "long" | "short" | "both"，默认 "long"
  bool high_precision = 17;      // 高精度 M1 多周期回测
}
```
重新生成：`buf generate`（后端 Go + 前端 TS）。

### backend `backend/internal/connect/strategy/python_strategy_handler.go`
`StartBacktestRun` 当前只写 code/symbol/tf/capital/mode/from/to/extra/dataset/template 到 `repository.BacktestRun`。需：
1. `repository.BacktestRun` 增加字段 `Commission/Slippage/Leverage/TradeDirection/HighPrecision`（含迁移，见下）。
2. handler 透传：
```go
run.Commission = f64Ptr(req.Msg.Commission)
run.Slippage = f64Ptr(req.Msg.Slippage)
run.Leverage = i32Ptr(req.Msg.Leverage)
run.TradeDirection = strPtr(req.Msg.TradeDirection)
run.HighPrecision = boolPtr(req.Msg.HighPrecision)
// 默认值兜底
if req.Msg.Commission <= 0 { run.Commission = f64Ptr(0.001) }
if req.Msg.Leverage <= 0 { run.Leverage = i32Ptr(1) }
if req.Msg.TradeDirection == "" { run.TradeDirection = strPtr("long") }
```
3. 执行回测的 worker（消费 `backtest_runs` PENDING 的组件）需把这些参数传给 Go 策略执行器（对齐 QuantDinger `BacktestService.run(commission, slippage, leverage, trade_direction)`，参考 `reference/.../services/backtest.py:1961`）。

### 数据库迁移 `backend/migrations/12X_backtest_run_params.up.sql`
```sql
ALTER TABLE backtest_runs
  ADD COLUMN IF NOT EXISTS commission DOUBLE PRECISION DEFAULT 0.001,
  ADD COLUMN IF NOT EXISTS slippage DOUBLE PRECISION DEFAULT 0,
  ADD COLUMN IF NOT EXISTS leverage INTEGER DEFAULT 1,
  ADD COLUMN IF NOT EXISTS trade_direction VARCHAR(10) DEFAULT 'long',
  ADD COLUMN IF NOT EXISTS high_precision BOOLEAN DEFAULT FALSE;
```
对应 `.down.sql` 删除这些列。`sqlc generate` 重新生成 repository。

### 前端新建 `WorkspaceBacktestParams.tsx`
```tsx
// 日期区间预设 1M/3M/6M/1Y + 自定义 RangePicker
// 初始资金 InputNumber、杠杆 InputNumber
// 手续费率 / 滑点 InputNumber
// 交易方向 Segmented [Long | Short | Both]
// 高精度 M1 Switch
// 暴露 onChange(params) 给父页面，handleRunBacktest 使用
```
状态提升到 `StrategyWorkspacePage`，`handleRunBacktest` 透传给 `startBacktestRun`（client 同步加 commission/slippage/leverage/tradeDirection/highPrecision 字段）。

**验收**: 修改参数后回测结果随之变化（手续费影响净收益、方向限制开仓方向、日期区间决定 K 线范围）。

---

## P3. 图表指标叠加 + 买卖点标记

klinecharts 内置 SMA/EMA/MACD/RSI/BOLL/VOL 等指标，无需自研计算。

### 扩展 `PriceChart.tsx`
1. 新增 props:
```ts
indicators?: string[];                  // 如 ['MA','MACD','VOL']
markers?: Array<{ time: number; side: 'buy' | 'sell'; price: number }>;
```
2. 指标：
```ts
// 主图叠加
chart.createIndicator('MA', true, { id: 'candle_pane' });
chart.createIndicator('BOLL', true, { id: 'candle_pane' });
// 副图
chart.createIndicator('VOL');
chart.createIndicator('MACD');
chart.createIndicator('RSI');
```
按 `indicators` 动态 create/remove，记录返回的 paneId 以便移除。
3. 买卖点标记（klinecharts overlay）:
```ts
import { registerOverlay } from 'klinecharts';
// 用内置 'simpleAnnotation' 或自定义 overlay 绘制 B/S
for (const m of markers) {
  chart.createOverlay({
    name: 'simpleAnnotation',
    extendData: m.side === 'buy' ? 'B' : 'S',
    points: [{ timestamp: m.time * 1000, value: m.price }],
    styles: { text: { color: m.side === 'buy' ? '#26a69a' : '#ef5350' } },
  });
}
```

### 顶部指标工具栏（对齐截图 SMA/EMA/RSI/MACD/BB...）
在 `WorkspaceChartTab` 顶部加一排可多选 Tag/Checkbox，控制 `indicators` 数组传入 `PriceChart`。

### 买卖点数据来源
回测完成后，`backtestMetrics.trades`（含 time/side/price）→ 转为 `markers` 传入 `PriceChart`，在 K 线上叠加 B/S。

**验收**:
- 勾选指标后图表叠加对应曲线/副图。
- 回测完成后买卖点出现在对应 K 线位置。

---

## P4. AI Generate（自然语言生成指标）

截图左下 "AI Generate" 输入自然语言 → 生成指标代码。现有 `codeAssistApi.revise()` 是「改写已有代码」，需「从零生成」。

### 方案 A（零后端改动，推荐先做）
复用 `reviseCode`，传空 `code` + 用户指令作为 `instruction`，把返回的 `python` 填入编辑器。前端在 AI 面板加 "Generate" 模式：
```ts
const { python } = await codeAssistApi.revise({ code: '', instruction: prompt, locale });
if (python) setCode(python);
```

### 方案 B（语义更清晰，需后端）
新增 RPC `GenerateCode(GenerateCodeRequest{ prompt, locale }) → { python, text }`，在 codeAssist 服务内复用现有 LLM 客户端，使用「生成」系统提示词。proto + handler + client 三处新增。

**验收**: 输入「EMA9 上穿 EMA20 金叉做多」→ 生成可校验通过的策略代码。

---

## P5. Smart Tuning（参数寻优，结构化扫描）

截图底部 "Smart Tuning"：Structured scan (no LLM)，Grid/Random，Run Smart Tuning。ant 当前**无此能力**。参考 QuantDinger `reference/.../services/experiment/runner.py`（grid/random 扫描 + OOS 验证 + 评分）。

### 架构（独立里程碑，复杂度高）
1. **参数空间来源**: 复用 `codeAssistApi.validateExtended()` 返回的 `parameters`（key/type/default/suggested），作为可调参数。
2. **proto**: 新增 `StrategyTuningService`：
```proto
service StrategyTuningService {
  rpc StartTuning(StartTuningRequest) returns (StartTuningResponse);   // 返回 jobId
  rpc WatchTuning(WatchTuningRequest) returns (stream TuningUpdate);   // SSE 进度+排名
}
message StartTuningRequest {
  string code = 1; string account_id = 2; string symbol = 3; string timeframe = 4;
  string method = 5;             // "grid" | "random"
  int32 max_combinations = 6;    // random 上限
  map<string, ParamRange> param_ranges = 7;
  // 复用回测参数
  double initial_capital = 8; double commission = 9; int32 leverage = 10;
  string trade_direction = 11;
  google.protobuf.Timestamp from = 12; google.protobuf.Timestamp to = 13;
}
message ParamRange { double min = 1; double max = 2; double step = 3; }
message TuningUpdate {
  int32 done = 1; int32 total = 2;
  repeated TuningCandidate top = 3;   // 按评分排序的 top-K
}
```
3. **backend** `internal/service/tuning/`:
   - 生成参数组合（grid 笛卡尔积 / random 采样，受 `max_combinations` 限制）。
   - 对每个组合调用现有回测执行器（复用 P2 的 worker），收集 metrics。
   - 评分（夏普/收益/回撤加权）+ 排序，流式推送 top-K。
   - **OOS 防过拟合**（可选）：训练/测试 7:3 切分，参考 runner.py `_compute_oos_window`。
4. **前端** Backtest 面板加 "Smart Tuning" Tab：方法选择(Grid/Random)、参数范围编辑（基于 validateExtended 的 parameters 自动生成行）、Run 按钮、进度条 + 结果排行榜（点击某行回填参数）。

**验收**:
- 选 Grid + 2 个参数范围 → 跑完所有组合，排行榜按评分排序。
- 点击 top 候选可将参数应用到代码/回测。

**风险**: 组合爆炸需限流（max_combinations）、并发回测需队列与资源控制。建议作为 P2/P3 之后的独立里程碑。

---

## 3. 落地顺序建议

```
P0 (修 bug)  →  P1 (Quick Trade, 纯前端)  →  P2 (回测参数, proto+迁移)
            →  P3 (图表指标/标记, 依赖 P2 trades)  →  P4 (AI Generate)
            →  P5 (Smart Tuning, 独立里程碑)
```

P0/P1 可立即交付（后端已就绪）；P2 起涉及 proto 与迁移，需 `buf generate` + `sqlc generate`；P5 工作量最大，建议拆为单独里程碑。

---

## 4. 全局注意事项

- **MT vs 加密差异**: 杠杆/保证金模式在 MT 下单时只读；UI 不要提供「下单时改杠杆」的误导控件。
- **invariant #8**: 下单必须经 `mthub`，禁止直调 `mt4client/mt5client`。
- **proto 改动向后兼容**: 新增字段一律 optional/默认值，避免破坏既有调用。
- **前端状态源**: workspace 现用本地 useState + `useAccount`（Zustand）。新增组件沿用现有约定，避免引入第三套状态。
- **i18n**: 所有新文案走 `t('strategy.workspace.*')`，补齐 `zh-cn/zh-tw/en/ja/vi` 资源。
- **测试**: P1 下单建议用 mock `tradingApi` 做组件测试；P2 回测参数透传加 handler 单测；P5 参数组合生成器加单测。

---

## 5. 文件改动清单（速查）

| 阶段 | 新增/修改文件 |
|------|--------------|
| P0 | `frontend/src/pages/strategy/StrategyWorkspacePage.tsx`（修 startBacktestRun 调用） |
| P1 | 新增 `components/workspace/WorkspaceQuickTrade.tsx`；改 `WorkspaceChartTab.tsx` |
| P2 | `proto/ant/v1/backtest_run_start.proto`；`backend/.../python_strategy_handler.go`；`backend/migrations/12X_backtest_run_params.up/down.sql`；`backend/.../queries` + sqlc；前端 `client/pythonStrategy.ts`、新增 `WorkspaceBacktestParams.tsx`、改主页面 |
| P3 | `frontend/src/components/chart/PriceChart.tsx`（indicators/markers）；改 `WorkspaceChartTab.tsx`（指标工具栏）；主页面把 trades→markers |
| P4 | 方案A：改 AI 面板组件；方案B：codeAssist proto+handler+client |
| P5 | 新增 `proto/ant/v1/strategy_tuning*.proto`；`backend/internal/service/tuning/`；新增前端 Smart Tuning Tab |
