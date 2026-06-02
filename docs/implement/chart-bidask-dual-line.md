# K线图表：MT4 风格 Bid/Ask 双线显示 落地方案

生成时间: 2026-06-02
目标: 让 `PriceChart`（含 strategy/workspace Chart Tab）显示 MT4 风格的 **Bid(红)/Ask 两条整宽水平线 + 右轴价签**，停在最新报价处。
参考截图: `/opt/ant/reference/ScreenShot_2026-06-02_150959_775.jpg`
适用对象: 交给 AI 按步骤落地。

---

## 0. 现状（已核实）

双线的基础设施**已存在**，只是画法和数据触达有问题：

- `frontend/src/components/chart/BidAskIndicator.ts`：已注册名为 `BIDASK` 的自定义指标，`registerIndicator` 在模块加载时执行。
- `frontend/src/components/chart/PriceChart.tsx:134`：图表创建时 `chart.createIndicator('BIDASK', true, { id: 'candle_pane' })`，叠加在主图。三处页面共用此组件：`Market`、`Trading`、`StrategyWorkspace`(Chart Tab)。
- `frontend/src/components/chart/useChartData.ts:49`：SSE `onBar` 事件携带 `ev.bid / ev.ask` → 调 `setBidAsk(barTime, bid, ask)` 喂数据；`setBars` 触发图表重绘。

**两个问题**:
1. **画法不对**: 当前 `calc` 给每根 K 线返回一个 bid/ask 点（历史 bar 无报价就回落 `close`），渲染成"贴着收盘价的逐根连线"，而非截图里"贯穿全宽、停在最新报价"的水平双线。
2. **数据前提**: bid/ask 仅在实时 tick 流入时才有，要求账户已连接 + 已 `subscribeBars` + 行情开市。否则退化重叠在 close。

klinecharts 版本: **9.8.12**（v9 指标 `draw` 回调签名为单对象参数，含 `ctx / bounding / yAxis`，`yAxis.convertToPixel(price)` 可用）。

---

## 1. 落地步骤

### 步骤 1（核心）：改写 `BidAskIndicator.ts` 为整宽水平双线

文件: `frontend/src/components/chart/BidAskIndicator.ts`

**1.1 新增"最新报价"状态**（除了按 bar 存，再存一个 `latest`）：
```ts
let latest: { bid: number; ask: number } | null = null;

export function setBidAsk(tsSec: number, bid: number, ask: number) {
  const cur = liveMap.get(tsSec);
  const nb = bid > 0 ? bid : cur?.bid ?? 0;
  const na = ask > 0 ? ask : cur?.ask ?? 0;
  liveMap.set(tsSec, { bid: nb, ask: na });
  latest = { bid: nb || latest?.bid || 0, ask: na || latest?.ask || 0 };
}

export function clearBidAsk() {
  liveMap.clear();
  latest = null;
}
```

**1.2 指标改为只画两条水平线**（`figures: []` + 自定义 `draw`，`return false` 抑制默认逐根线）：
```ts
const BIDASK_INDICATOR: IndicatorCreate = {
  name: 'BIDASK',
  shortName: 'B/A',
  precision: 5,
  shouldOhlc: false,
  figures: [],                       // 去掉逐根 bar 线
  calc: () => [],                    // 数据走 latest，calc 不产点
  draw: ({ ctx, bounding, yAxis }: any) => {
    if (!latest) return false;
    const line = (price: number, color: string) => {
      if (!(price > 0)) return;
      const y = yAxis.convertToPixel(price);
      ctx.save();
      ctx.strokeStyle = color;
      ctx.lineWidth = 1;
      ctx.setLineDash([2, 2]);       // MT4 风格虚线；要实线删掉此行
      ctx.beginPath();
      ctx.moveTo(0, y);
      ctx.lineTo(bounding.width, y);
      ctx.stroke();
      // 右轴价签
      ctx.setLineDash([]);
      const label = price.toFixed(5);
      ctx.font = '10px sans-serif';
      const w = ctx.measureText(label).width + 8;
      ctx.fillStyle = color;
      ctx.fillRect(bounding.width - w, y - 7, w, 14);
      ctx.fillStyle = '#fff';
      ctx.textBaseline = 'middle';
      ctx.fillText(label, bounding.width - w + 4, y);
      ctx.restore();
    };
    line(latest.bid, '#ef5350');     // Bid 红
    line(latest.ask, '#26a69a');     // Ask 青（要白色改 '#e0e0e0'）
    return false;                    // 只画自定义，不画默认 figure
  },
};
```

关键点:
- `draw` 返回 `false` = 只渲染自定义的两条水平线，不渲染默认 figure。
- `figures: []` 移除原贴 close 的逐根连线。
- 精度先硬编码 5 位；后续可由 symbol 的 `digits` 动态传入（见步骤 4 可选项）。

### 步骤 2（必做）：无 tick 时的兜底（避免空白/误导）

收市或账户未连接时 `latest` 为空 → 不画线。为避免"图上完全没有价格参考"，在 `latest` 为空时回落用最后一根 bar 的 `close` 画一条单线（灰色），表示"最新收盘价、无实时报价"。

实现: `draw` 内 `kLineDataList`（v9 `draw` 入参之一）取最后一根的 `close`，`latest` 为空时画单灰线。
```ts
draw: ({ ctx, bounding, yAxis, kLineDataList }: any) => {
  const fallback = () => {
    const last = kLineDataList?.[kLineDataList.length - 1];
    if (!last || !(last.close > 0)) return false;
    // 画灰色单线 last.close（实现同上 line()，color '#888'）
    return false;
  };
  if (!latest) return fallback();
  // ...画 bid/ask 双线...
  return false;
}
```

### 步骤 3（必做）：保证 bid/ask 数据流入

双线依赖实时 tick。两个前提:
1. **账户已连接**: bid/ask 来自 SSE tick。切账户时需触发 `connectAccount`（与 K 线方案 `kline-account-aware-fetch.md` 步骤 3 同一处理点，可合并实现）。
2. **已订阅**: `useChartData` 已调 `marketApi.subscribeBars({ accountId, symbol })`（`useChartData.ts:29`），无需新增；确认切 symbol/账户时正确重订阅（依赖数组 `[symbol, timeframe, accountId]` 已覆盖）。

重绘触发: tick 到达 → `onBar` → `setBidAsk` + `setBars(merged)` → `PriceChart` 的 `applyNewData` 重渲染 → `draw` 执行。**已有链路，无需额外处理。**

### 步骤 4（可选增强）

- **动态精度**: 从 `SymbolPicker`/`symbolParams` 拿该 symbol 的 `digits`，通过模块变量传给指标的 `toFixed`，避免 5 位硬编码对 JPY 等品种不准。
- **点差展示**: 在两线之间或价签旁标注 `spread = ask - bid`（pips）。
- **开关**: 在图表工具栏加一个 "B/A" toggle 控制是否显示双线（默认开）。

---

## 2. 验收标准

- 账户已连接 + 行情开市时，主图出现两条贯穿全宽的水平线：Bid(红)、Ask(青)，停在最新报价，右侧有价签。
- tick 更新时双线实时上下移动。
- 收市/未连接时回落为灰色单线（最后收盘价），不空白、不报错。
- 切换账户/symbol 后双线随新报价刷新（`clearBidAsk` 已在切换时调用，`useChartData.ts:26`）。
- Market / Trading / StrategyWorkspace 三处图表表现一致。

---

## 3. 影响面与风险

- **共用组件**: `BidAskIndicator.ts` 经 `PriceChart` 被三处页面共用，改动同时生效——三处都需回归。
- **canvas 直绘**: `draw` 内直接操作 `ctx`，注意 `ctx.save()/restore()` 配对，避免污染后续绘制状态。
- **性能**: 仅画两条线 + 两个价签，开销极小；不随 bar 数增长。
- **向后兼容**: 指标名仍为 `BIDASK`，`PriceChart` 的 `createIndicator('BIDASK', ...)` 调用不变。

---

## 4. 文件改动清单（速查）

| 步骤 | 文件 | 改动 |
|------|------|------|
| 1 | `frontend/src/components/chart/BidAskIndicator.ts` | 新增 `latest` 状态；`figures:[]` + 自定义 `draw` 画整宽水平双线 + 右轴价签；`return false` |
| 2 | 同上 | `draw` 内 `latest` 为空时用最后 bar `close` 画灰色单线兜底 |
| 3 | 账户选择入口（`StrategyWorkspacePage` 等） | 切账户触发 `connectAccount`（可与 K 线方案合并） |
| 4 | （可选）`BidAskIndicator.ts` / `PriceChart.tsx` | 动态精度、点差标注、显示开关 |

---

## 5. 关键事实备注（供实施者核对）

- 不要新写指标注册/图表创建——`BIDASK` 指标与 `createIndicator` 调用**已存在**，只改 `BidAskIndicator.ts` 内部实现即可。
- 数据已由 SSE `onBar` 喂入（`useChartData.ts:49`），不要另起数据通道。
- klinecharts v9 `draw` 返回 `false` 表示"只画自定义、跳过默认 figure"；`figures: []` 配合可彻底去掉逐根连线。
- 双线"不动"通常是**没有实时 tick**（账户未连接/收市），优先排查数据流，而非指标代码。
