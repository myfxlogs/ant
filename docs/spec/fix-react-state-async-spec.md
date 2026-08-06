# 施工 Spec：修复 React setState 异步 + useCallback 闭包旧 state（参数传不进回测）

> 根因详见 `docs/spec/param-pipeline-pitfalls.md` 坑 3。这是 xianhua 参数被忽略的真根因。

## 根因（一句话）

`WorkspaceDrawers.tsx` 的 `onConfirm` 循环调 `setParam`（异步 state 更新）后立即调 `backtest.run()`（useCallback 闭包捕获**旧** state）→ run 发送的 parameterOverrides 是空旧值，不是用户刚填的值。

## 现状代码（必读）

### WorkspaceDrawers.tsx（onConfirm 调用方）
```tsx
onConfirm={(result: BacktestModalResult) => {
  const p = result.params;
  backtest.setInitialCapital(p.initialCapital);
  backtest.setLeverage(p.leverage);       // 异步 setState
  backtest.setCommission(p.commission);
  backtest.setSlippage(p.slippage);
  backtest.setTradeDirection(p.tradeDirection);
  backtest.setStrictMode(p.strictMode);
  backtest.setStartDate(result.startDate);
  backtest.setEndDate(result.endDate);
  if (result.strategyParams) {
    for (const [name, value] of Object.entries(result.strategyParams)) {
      backtest.runner.setParam(name, value);  // 异步 setStrategyParamValues
    }
  }
  backtest.run();  // ← 闭包捕获旧 state！parameterOverrides 还是空、leverage 还是旧值
}}
```

### useBacktestRunner.ts（run 是 useCallback，闭包捕获旧 state）
```ts
const run = useCallback(async (inputs: BacktestRunnerInputs) => {
  ...
  parameterOverrides: strategyParamValues,        // ← 闭包旧值（空）
  executionConfig: { commission, slippage, leverage, ... },  // ← 闭包旧值（默认）
  ...
}, [initialCapital, commission, slippage, leverage, ..., strategyParamValues, t]);
```

## 修复方案（最小改动，不改 run 签名）

**核心思路**：`onConfirm` 已经有同步的 `result.strategyParams` + `result.params`，不需要经过异步 state 中转。直接构造 overrides + executionConfig 传给 run。

### 改 useBacktestRunner.ts

`run` 函数签名加一个**可选的 overrides 参数**（不破坏现有调用）：

```ts
const run = useCallback(async (
  inputs: BacktestRunnerInputs,
  overrides?: {
    params?: Record<string, string>;     // strategyParamValues 的直接传入
    executionConfig?: {                   // executionConfig 的直接传入
      commission: number;
      slippage: number;
      leverage: number;
      tradeDirection: string;
      strictMode: boolean;
    };
  }
) => {
  // 如果 overrides 传入，用它；否则用闭包 state（向后兼容）
  const paramValues = overrides?.params ?? strategyParamValues;
  const cfg = overrides?.executionConfig
    ? { commission: String(overrides.executionConfig.commission),
        slippage: String(overrides.executionConfig.slippage),
        leverage: String(overrides.executionConfig.leverage),
        tradeDirection: overrides.executionConfig.tradeDirection as ...,
        strictMode: overrides.executionConfig.strictMode }
    : { commission: String(commission), slippage: String(slippage), leverage: String(leverage), ... };

  ...
  parameterOverrides: paramValues && Object.keys(paramValues).length > 0
    ? paramValues : undefined,
  executionConfig: cfg,
  ...
}, [t]);  // ← 依赖数组只留 t（不再依赖 strategyParamValues/leverage 等，因为从 overrides 读）
```

### 改 WorkspaceDrawers.tsx（onConfirm 直接传值）

```tsx
onConfirm={(result: BacktestModalResult) => {
  const p = result.params;
  // 仍然 setState（保持 UI 同步，下次打开弹窗时显示正确）
  backtest.setInitialCapital(p.initialCapital);
  backtest.setLeverage(p.leverage);
  ...
  // 但 run 直接传入同步值（不依赖异步 state）
  backtest.run(
    { strategyCode, accountId, symbol, timeframe, ... },
    {
      params: result.strategyParams,
      executionConfig: {
        commission: p.commission,
        slippage: p.slippage,
        leverage: p.leverage,
        tradeDirection: p.tradeDirection,
        strictMode: p.strictMode,
      },
    }
  );
}}
```

## 约束

- 最小改动：只改 `run` 签名（加可选 overrides）+ `WorkspaceDrawers` onConfirm（传 overrides）。不重构其他。
- 向后兼容：overrides 可选，不传时用旧闭包 state（其他调用 run 的地方不受影响）。
- 不改后端（后端注入逻辑已对）。
- `npm run build` 通过。

## 验收

- 弹窗填 lots=0.01 + 杠杆=2000 → run 发送的请求里 parameterOverrides 有 Lots=0.01（有值非空）、executionConfig.leverage="2000"。
- 浏览器 devtools network 或后端 PG 确认。
