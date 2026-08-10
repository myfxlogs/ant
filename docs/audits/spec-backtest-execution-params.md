# Spec: 回测执行假设参数端到端接线

> **Status**: 🟦 设计完成，待施工（已纳入复审 5 点修订）  
> **Date**: 2026-08-10（复审 2026-08-11）  
> **Scope**: 将 `signal_timing` / `fill_rule` / `simulation_mode` 从 UI 状态完整接线到后端引擎，替代硬编码

## 1. 问题分析

### 1.1 当前状态

| 参数 | UI | 前端→后端 | 后端 Config | 引擎行为 | ExecutionAssumptions 响应 |
|------|-----|----------|------------|---------|------------------------|
| `signal_timing` | ✅ 可选 | ⚠️ 间接（映射为 `strictMode` bool） | ✅ `Config.StrictMode` | ❌ **引擎不读** | ⚠️ 硬编码 `next_bar_open`，仅根据 `StrictMode` 覆盖字符串 |
| `fill_rule` | ✅ 可选 | ❌ 不传 | ❌ 无字段 | ❌ N/A | ❌ 硬编码 `bar_close` |
| `simulation_mode` | ✅ 可选 | ❌ 不传 | ❌ 无字段 | ❌ N/A | ❌ 硬编码 `KLINE_RANGE` |

### 1.2 核心发现：`StrictMode` 引擎不生效

`backtest.Config.StrictMode` 定义在 `@/opt/ant/backend/strategy/backtest/types.go:25`，注释为 "if true, skip bars with missing data"。但 `@/opt/ant/backend/strategy/backtest/engine.go` 的 `Engine.Run()` **从未检查 `StrictMode`**。引擎始终在当前 bar 执行信号（`dispatchSignal` at `bar.Close`），无论 `StrictMode` true/false。

`buildBacktestResponse`（`@/opt/ant/backend/internal/connect/strategy/backtest_worker_vm.go:318-323`）根据 `StrictMode` 设置响应中的 `SignalTiming` 字符串，但引擎行为不变 → **响应谎报执行时机**。

### 1.3 影响范围

- **所有回测**：`ExecutionAssumptions.SignalTiming` 谎报 `next_bar_open`（当 `strictMode=true`），实际引擎在信号 bar 的 close 价成交
- **用户信任**：用户看到 "Next Bar Open" 但回测结果基于 "Same Bar Close" 语义 → 结果不可复现
- **fill_rule**：用户选择被完全忽略

## 2. 设计方案

### 2.1 架构决策

**决策 1：`signal_timing` 替代 `strict_mode`，不保留 bool**

`strict_mode` 语义模糊（proto 注释说 "next-bar-open"，types.go 注释说 "skip bars with missing data"）。用 `signal_timing` string（`"next_bar_open"` / `"same_bar_close"`）直接表达意图。`strict_mode` 字段保留向后兼容，但从 `signal_timing` 派生：`strict_mode = (signal_timing == "next_bar_open")`。

**决策 2：`fill_rule` 控制成交价计算方式**

| fill_rule | 成交价 | 说明 |
|-----------|-------|------|
| `bar_close` | `bar.Close`（无 spread） | 理想化：信号 bar 收盘价直接成交 |
| `market` | `bar.Close ± spread`（buy=ask, sell=bid） | 现实化：含买卖价差成本 |
| `limit` | 指定价格（pending order 语义） | 挂单成交，已有 `checkPendingOrders` 逻辑 |

当前引擎 market order 在 `OrderSend` 中调 `applySpreadToFill`（`@/opt/ant/backend/strategy/backtest/broker.go:108-110`），即所有 market order 都加 spread。`fill_rule=bar_close` 时需跳过 spread；`fill_rule=market` 时保持现有行为。

**决策 3：`simulation_mode` 仅传参+响应透传，DATASET 模式 UI 保持 disabled**

DATASET 模式需要历史 tick 数据回放引擎，是独立大功能。本次仅完成 `KLINE_RANGE` 的端到端透传（proto → config → 响应），DATASET 保持 disabled。

**决策 4：引擎实现 `next_bar_open` 信号延迟执行**

`signal_timing=next_bar_open` 时，信号不在当前 bar 执行，而是队列化，在下一根 bar 的 `Open` 价执行。这是回测引擎的标准 "no look-ahead" 语义。

> **复审修订 1**：延迟执行不仅影响 order 类信号（ActionBuy/Sell），也影响 ActionClose/CloseAll。这些走 `PositionClose` → `broker.currentPrice`（循环头 `SetBarPrice(bar.Close)` 设置）。若不修改，延迟平仓会在下一 bar 的 close 价平仓而非 open 价——语义错误。**修法**：执行延迟信号前先 `SetBarPrice(bar.Open)`，执行完毕再恢复 `SetBarPrice(bar.Close)`。

### 2.2 改动范围

```
proto/ant/v1/backtest_execution_config.proto     — 加 signal_timing + fill_rule + simulation_mode 字段
backend/strategy/backtest/types.go               — Config 加 SignalTiming + FillRule + SimulationMode
backend/strategy/backtest/engine.go              — 实现 next_bar_open 延迟执行 + fill_rule 成交价 + 延迟平仓 open 价基准
backend/strategy/backtest/broker.go              — OrderSend 接受 fillRule 参数控制 spread
backend/internal/connect/strategy/backtest_worker.go — backtestParams 加新字段 + 旧快照回退从 StrictMode 派生
backend/internal/connect/strategy/backtest_worker_vm.go — buildBacktestConfig 传新字段 + buildBacktestResponse 读 config
backend/internal/connect/strategy/backtest_execution.go — ExecuteBacktestDirect 同步
backend/internal/connect/strategy/strategy_backtest_validate.go — validateBacktestRequest 拒绝 unimplemented 值
frontend/src/pages/strategy/components/workspace/BacktestParamsModal.tsx — BacktestModalResult 加新字段 + handleConfirm 传全参
frontend/src/pages/strategy/components/workspace/ExecutionAssumptionsSelectors.tsx — 无改动（UI 已就绪）
frontend/src/pages/strategy/components/workspace/WorkspaceDrawers.tsx — 传新字段到 executionConfig
frontend/src/pages/strategy/hooks/useStrategyWorkspaceState.ts — handleRunBacktest overrides 类型扩展
frontend/src/components/backtest/useBacktestRunner.ts — executionConfig 加新字段
frontend/src/client/strategyRuntime.ts — startBacktestRun 传新字段到 proto
```

## 3. 详细设计

### 3.1 Proto 改动

`proto/ant/v1/backtest_execution_config.proto`:

```protobuf
message BacktestExecutionConfig {
  string commission = 1;
  string slippage = 2;
  string leverage = 3;
  TradeDirection trade_direction = 4;
  bool strict_mode = 5;           // 保留向后兼容；= (signal_timing == "next_bar_open")
  StrategyConfig strategy_config = 6;
  string swap_rate = 7;
  string margin_call_level = 8;
  string signal_timing = 9;       // 新增: "next_bar_open" | "same_bar_close"
  string fill_rule = 10;          // 新增: "bar_close" | "market" | "limit"
  string simulation_mode = 11;    // 新增: "KLINE_RANGE" | "DATASET"
}
```

字段号 9/10/11 不冲突。`strict_mode` 保留，从 `signal_timing` 派生，确保旧 `ConfigSnapshot` 二进制兼容（`UnmarshalOptions{DiscardUnknown: true}` 已在 `extractBacktestParams` 中使用）。

### 3.2 后端 Config + 引擎

`backend/strategy/backtest/types.go` — `Config` 加 3 字段：

```go
type Config struct {
    // ... existing fields ...
    StrictMode      bool            // 保留，从 SignalTiming 派生
    SignalTiming    string          // "next_bar_open" | "same_bar_close"，空默认 "next_bar_open"
    FillRule        string          // "bar_close" | "market" | "limit"，空默认 "bar_close"
    SimulationMode  string          // "KLINE_RANGE" | "DATASET"，空默认 "KLINE_RANGE"
    // ...
}
```

`backend/strategy/backtest/engine.go` — `Engine.Run()` 改动：

**next_bar_open 实现**：在 `Run` 循环中，信号不立即 `dispatchSignal`，而是存入 `pendingSignals []sdk.Signal`。在下一根 bar 开头，先执行 `pendingSignals`（在 `bar.Open` 价），再处理当前 bar 的信号。

> **复审修订 1（延迟平仓价格基准）**：执行延迟信号前必须 `SetBarPrice(bar.Open)`，因为 `ActionClose`/`CloseAll` 走 `PositionClose` → `broker.currentPrice`。若仍为 `bar.Close`，延迟平仓在下一 bar close 价平仓而非 open 价——语义错误。执行完延迟信号后恢复 `SetBarPrice(bar.Close)`。

```go
// Run 循环改动伪码：
var pendingSignals []sdk.Signal

for i := 1; i < len(e.bars); i++ {
    bar := e.bars[i]
    e.broker.SetBar(i)
    e.broker.SetBarTime(time.UnixMilli(bar.Timestamp))
    e.broker.SetBarPrice(bar.Close)

    // 1. 执行上一根 bar 的延迟信号（next_bar_open 模式）
    if len(pendingSignals) > 0 {
        // 复审修订 1：延迟平仓需在 open 价执行，先切换 currentPrice
        e.broker.SetBarPrice(bar.Open)
        for _, sig := range pendingSignals {
            e.dispatchSignalAtPrice(sig, bar.Open)  // 在本 bar 的 open 价成交
        }
        e.broker.SetBarPrice(bar.Close)  // 恢复
        pendingSignals = nil
    }

    // 2. 正常流程：checkPendingOrders, checkSLTP
    e.advanceExtraBars(btCtx, bar.Timestamp)
    e.checkPendingOrders(bar)
    e.checkSLTP(bar)

    // 3. 运行策略
    sig, err := e.runStrategySignal(btCtx, bar)
    if sig != nil {
        if e.config.SignalTiming == "same_bar_close" {
            // same_bar_close: 立即在当前 bar close 成交
            e.dispatchSignal(sig, bar)
        } else {
            // next_bar_open: 延迟到下一根 bar open
            pendingSignals = append(pendingSignals, *sig)
        }
    }

    // 4. equity, margin call...
}
```

> **注意**：`SignalTiming` 在 `buildBacktestConfig` 中已确定最终值（非空），引擎不需要处理空值 fallback。

**fill_rule 实现**：`SimBroker.OrderSend` 已对 market order 调 `applySpreadToFill`。改为根据 `FillRule` 决定：

```go
// broker.go OrderSend 中：
if req.Type == sdk.OrderMarket {
    if b.config.FillRule == "bar_close" || b.config.FillRule == "" {
        // bar_close: 不加 spread，用原始价格
    } else {
        // market: 加 spread（现有行为）
        rec.Price = b.applySpreadToFill(rec.Price, req.Side == sdk.SideBuy)
    }
}
```

**dispatchSignalAtPrice**：新增方法，与 `dispatchSignal` 相同但用指定价格而非 `bar.Close`：

```go
func (e *Engine) dispatchSignalAtPrice(sig *sdk.Signal, price decimal.Decimal) {
    // 与 dispatchSignal 相同，但 price 参数替代 bar.Close
    // ...
    if sig.Price.IsPositive() {
        price = sig.Price  // 信号自带价格优先
    }
    // ...
}
```

### 3.3 后端 backtestParams + worker

`backtest_worker.go` — `backtestParams` struct 加字段：

```go
type backtestParams struct {
    // ... existing fields ...
    signalTiming    string  // "next_bar_open" | "same_bar_close"
    fillRule        string  // "bar_close" | "market" | "limit"
    simulationMode  string  // "KLINE_RANGE" | "DATASET"
}
```

`extractBacktestParams` — 从 `ConfigSnapshot` 解析新字段：

```go
if len(run.ConfigSnapshot) > 0 {
    var ec antv1.BacktestExecutionConfig
    opts := proto.UnmarshalOptions{DiscardUnknown: true}
    if err := opts.Unmarshal(run.ConfigSnapshot, &ec); err == nil {
        // ... existing fields ...
        if ec.GetSignalTiming() != "" {
            p.signalTiming = ec.GetSignalTiming()
        }
        if ec.GetFillRule() != "" {
            p.fillRule = ec.GetFillRule()
        }
        if ec.GetSimulationMode() != "" {
            p.simulationMode = ec.GetSimulationMode()
        }
    }
}
```

`backtest_worker_vm.go` — `buildBacktestConfig` 传新字段：

> **复审修订 2（旧快照回退默认值）**：`signalTiming` 为空时，从 `params.strictMode`（来自 `run.StrictMode`）派生：`true` → `next_bar_open`，`false` → `same_bar_close`。两者皆无才默认 `next_bar_open`。这防止旧 `strict_mode=false` 的 run 重跑时静默翻转为 `next_bar_open`。

```go
func (s *StrategyExecutionServer) buildBacktestConfig(params backtestParams, run *repository.BacktestRun) backtest.Config {
    signalTiming := params.signalTiming
    if signalTiming == "" {
        // 复审修订 2：从 StrictMode 派生，不静默翻转
        if params.strictMode {
            signalTiming = "next_bar_open"
        } else {
            signalTiming = "same_bar_close"
        }
    }
    fillRule := params.fillRule
    if fillRule == "" {
        fillRule = "bar_close"
    }
    simulationMode := params.simulationMode
    if simulationMode == "" {
        simulationMode = "KLINE_RANGE"
    }
    cfg := backtest.Config{
        // ... existing fields ...
        StrictMode:     signalTiming == "next_bar_open",  // 从 SignalTiming 派生
        SignalTiming:   signalTiming,
        FillRule:       fillRule,
        SimulationMode: simulationMode,
    }
    // ...
}
```

`buildBacktestResponse` — 从 config 读替代硬编码：

> **复审修订 4（MtfFallbackReason 诚实化）**：1m 子分辨率不存在，`same_bar_close` 时不应声称 MTF fallback。删除 `MtfFallbackReason` 或改为诚实文案 `"same_bar_close: signals execute at bar close price"`。

```go
resp.ExecutionAssumptions = &antv1.ExecutionAssumptions{
    SimulationMode:   cfg.SimulationMode,
    SignalTiming:     cfg.SignalTiming,
    FillRule:         cfg.FillRule,
    ActualCommission: cfg.Commission.String(),
    // ...
}
// 复审修订 4：不设 MtfFallbackReason——1m 子分辨率不存在，无需解释
```

### 3.4 前端改动

**`BacktestParamsModal.tsx`** — `BacktestModalResult` 加字段：

```typescript
export interface BacktestModalResult {
  params: StandardParams;
  startDate: string;
  endDate: string;
  timeframe: string;
  strategyParams?: Record<string, string>;
  signalTiming: 'next_bar_open' | 'same_bar_close';  // 新增
  fillRule: 'bar_close' | 'market' | 'limit';        // 新增
  simulationMode: 'KLINE_RANGE' | 'DATASET';          // 新增
}
```

`handleConfirm` 传全参：

```typescript
onConfirm({
  params: { ...params, strictMode: signalTiming === 'next_bar_open' },
  startDate, endDate, timeframe,
  strategyParams: strategyParamValues,
  signalTiming,
  fillRule,
  simulationMode,
});
```

**`WorkspaceDrawers.tsx`** — 传新字段到 `executionConfig`：

```typescript
backtest.run({
  params: result.strategyParams,
  executionConfig: {
    commission: p.commission,
    slippage: p.slippage,
    leverage: p.leverage,
    tradeDirection: p.tradeDirection,
    strictMode: p.strictMode,
    signalTiming: result.signalTiming,     // 新增
    fillRule: result.fillRule,             // 新增
    simulationMode: result.simulationMode, // 新增
  },
  timeframe: result.timeframe,
});
```

**`useBacktestRunner.ts`** — `executionConfig` 类型扩展：

```typescript
executionConfig?: {
  commission: number;
  slippage: number;
  leverage: number;
  tradeDirection: 'long' | 'short' | 'both';
  strictMode: boolean;
  signalTiming?: 'next_bar_open' | 'same_bar_close';
  fillRule?: 'bar_close' | 'market' | 'limit';
  simulationMode?: 'KLINE_RANGE' | 'DATASET';
};
```

**`strategyRuntime.ts`** — `startBacktestRun` 传新字段到 proto message：

```typescript
executionConfig: params.executionConfig ? create(BacktestExecutionConfigSchema, {
  commission: String(params.executionConfig.commission),
  slippage: String(params.executionConfig.slippage),
  leverage: String(params.executionConfig.leverage),
  tradeDirection: /* existing */,
  strictMode: params.executionConfig.strictMode,
  signalTiming: params.executionConfig.signalTiming ?? 'next_bar_open',
  fillRule: params.executionConfig.fillRule ?? 'bar_close',
  simulationMode: params.executionConfig.simulationMode ?? 'KLINE_RANGE',
}) : undefined,
```

### 3.5a `validateBacktestRequest` — API 边界拒绝未实现值

> **复审修订 3**：`fill_rule=limit` 和 `simulation_mode=DATASET` 被接受并透传到响应，但引擎不实现——违反诚实性红线。必须在 `validateBacktestRequest` 拒绝。

```go
// strategy_backtest_validate.go — validateBacktestRequest 末尾追加：
if cfg := req.Msg.GetExecutionConfig(); cfg != nil {
    if cfg.GetFillRule() == "limit" {
        return connect.NewError(connect.CodeInvalidArgument,
            fmt.Errorf("fill_rule=limit is not yet implemented"))
    }
    if cfg.GetSimulationMode() == "DATASET" {
        return connect.NewError(connect.CodeInvalidArgument,
            fmt.Errorf("simulation_mode=DATASET is not yet implemented"))
    }
}
```

### 3.5b `backtest_execution.go` — `ExecuteBacktestDirect` 同步

`ExecuteBacktestDirect`（智能调优路径）也构造 `backtestParams`，需同步加默认值：

```go
params := backtestParams{
    // ... existing ...
    signalTiming:   "next_bar_open",
    fillRule:       "bar_close",
    simulationMode: "KLINE_RANGE",
}
```

`inheritBacktestRunConfig` 也需从 `ConfigSnapshot` 继承新字段（与 `extractBacktestParams` 相同逻辑）。

### 3.6 Marketplace / Agent 硬编码 `strict=true` 影响审计

> **复审修订 5**：`marketplace/backtest.go:72` 和 `agent/strategy_gen_helpers.go:118` 硬编码 `strict=true`。引擎修复后，这些路径真实运行 `next_bar_open`（而非之前谎报的 `next_bar_open` 但实际 `same_bar_close`）。

**影响分析**：
- **Marketplace 校验回测**（`marketplace/backtest.go:72`）：`strictTrue := true` → 引擎修复后信号延迟到下一 bar open 执行。之前引擎忽略 `StrictMode`，信号在当前 bar close 执行。结果会变化（成交价从 close 变为 next open），但这是 bug fix——之前的结果基于错误语义。
- **Agent 生成验证**（`agent/strategy_gen_helpers.go:118`）：`StrictMode: ptr.Bool(true)` → 同上。Agent 生成的策略回测验证结果会变化，但更准确。
- **风险实际范围**：`fill_rule=bar_close` 默认影响 ≈0（worker 路径从不设置 `cfg.Spread`，仅日志，fallback 到 slippage 默认 0）。结果变化风险实际只来自 `next_bar_open` 延迟执行。
- **结论**：无需修改 marketplace/agent 代码——`strict=true` 语义正确（这些路径确实应使用 `next_bar_open`）。引擎修复使其行为与声明一致。

## 4. 测试计划

### 4.1 后端单元测试

| 测试 | 文件 | 验证 |
|------|------|------|
| `TestEngine_NextBarOpen_DefersExecution` | `engine_test.go` | signal_timing=next_bar_open → 信号在下一 bar 的 Open 价成交，非当前 bar Close |
| `TestEngine_SameBarClose_ImmediateExecution` | `engine_test.go` | signal_timing=same_bar_close → 信号在当前 bar Close 成交（现有行为不变） |
| `TestEngine_FillRule_BarClose_NoSpread` | `broker_test.go` | fill_rule=bar_close → market order 成交价 = bar.Close（无 spread） |
| `TestEngine_FillRule_Market_WithSpread` | `broker_test.go` | fill_rule=market → buy 成交价 = bar.Close + spread |
| `TestBuildBacktestResponse_ReadsConfig` | `backtest_worker_vm_test.go` | ExecutionAssumptions 从 Config 读取，不硬编码 |
| `TestExtractBacktestParams_NewFields` | `backtest_worker_test.go` | ConfigSnapshot 含新字段 → backtestParams 正确提取 |
| `TestEngine_NextBarOpen_CloseAtOpenPrice` | `engine_test.go` | 复审 1：next_bar_open 模式 ActionClose 在下一 bar Open 价平仓，非 Close |
| `TestBuildBacktestConfig_OldSnapshotStrictFalse` | `backtest_worker_test.go` | 复审 2：旧快照无 signal_timing + strict_mode=false → SignalTiming=same_bar_close |
| `TestValidateBacktestRequest_RejectsUnimplemented` | `strategy_backtest_validate_test.go` | 复审 3：fill_rule=limit / simulation_mode=DATASET → InvalidArgument |

### 4.2 对抗证明

- 删除 `pendingSignals` 逻辑 → `TestEngine_NextBarOpen_DefersExecution` 必红（信号在当前 bar 成交而非下一 bar）
- 硬编码 `FillRule: "market"` → `TestEngine_FillRule_BarClose_NoSpread` 必红（成交价含 spread）
- 删除 `SetBarPrice(bar.Open)` 延迟平仓切换 → `TestEngine_NextBarOpen_CloseAtOpenPrice` 必红（平仓在 close 价而非 open 价）
- 硬编码 `signalTiming = "next_bar_open"` fallback → `TestBuildBacktestConfig_OldSnapshotStrictFalse` 必红（旧 strict=false 静默翻转为 next_bar_open）

### 4.3 回归

- `go build ./...` + `go test ./strategy/backtest/...` + `go test ./internal/connect/strategy/...`
- 前端 `npm run build` + `vitest run`
- `check-file-lines --strict`

## 5. 施工顺序

1. **Proto** → `make proto`（加字段 + 重新生成 Go + TS）
2. **后端 Config + Engine** → `types.go` + `engine.go` + `broker.go`（引擎核心改动）
3. **后端 worker** → `backtest_worker.go` + `backtest_worker_vm.go` + `backtest_execution.go`（接线层）
4. **后端测试** → 写 6 个测试 + 对抗证明
5. **前端** → `BacktestParamsModal.tsx` + `WorkspaceDrawers.tsx` + `useBacktestRunner.ts` + `strategyRuntime.ts`
6. **验证** → `go build` + `go test` + `npm run build` + `check-file-lines`
7. **部署** → `docker compose build backend && docker compose up -d backend` + 前端 `docker cp`

## 6. 风险

| 风险 | 缓解 |
|------|------|
| next_bar_open 改变现有回测结果 | 这是 bug fix（引擎谎报行为），不是行为变更。strictMode=true 的回测结果会变（从错误的 same_bar_close 变为正确的 next_bar_open） |
| ConfigSnapshot 旧数据无新字段 | proto3 默认空字符串 + `DiscardUnknown: true` + 代码空值 fallback 默认值 |
| engine.go 行数超标 | 当前 495 行，加 next_bar_open 逻辑约 +20 行 → ~515 行，需提取 `dispatchSignalAtPrice` 到独立函数或文件 |
| `dispatchSignal` 重复 | 提取共享核心逻辑，`dispatchSignal` 和 `dispatchSignalAtPrice` 共用 |
