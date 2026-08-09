# Spec: 智能调优实验策略可追溯性 + 调优回测配置继承

> 状态：🟦 待审核（2026-08-10）

## 问题一：策略可追溯性缺失

用户从回测历史发起智能调优后，实验记录无法直观关联到来源策略。

### 现状数据链路

```
backtest_runs (id=331d0e60, name=空, strategy_id=空)
  → strategy_experiments (base_template_id=空, strategy_code=有, 无 strategy_name)
  → candidates (24行, 全 total_trades=0)
```

### 缺失

| 表 | 缺失字段 | 影响 |
|----|---------|------|
| `strategy_experiments` | `strategy_name` | 实验列表不知是哪个策略 |
| `strategy_experiments` | `backtest_run_id` | 无法回溯来源回测 |

### 根因

`useTuning.ts:runTuning` 参数无 `strategyName`/`backtestRunId`，proto `SubmitStrategyExperimentRequest` 也无这些字段。

## 问题二：调优回测配置硬编码 → 0 开单

### 现象

24 个候选全部 `total_trades=0`/`score=0`/`grade=E`。同一策略在正常回测中有开单记录。

### 根因

`backtest_execution.go:31-39` `ExecuteBacktestDirect`（调优专用路径）硬编码回测配置：

```go
params := backtestParams{
    initialCapital: "10000",
    commission:     "0.001",
    slippage:       "0",
    leverage:       "1",    // ← BUG：正常回测用 1000
    tradeDir:       antv1.TradeDirection_TRADE_DIRECTION_BOTH,
    strictMode:     true,
}
```

正常回测 `backtest_runs` 表中 leverage=1000，调优路径硬编码 leverage=1。

**算账**：XAUUSDm ~$2000，0.1 手 = 100 oz → 名义 $20,000。
- leverage=1000 → 保证金 $20 → 可开单
- leverage=1 → 保证金 $20,000 > 资金 $10,000 → 保证金不足 → 0 笔交易

`total_return=-215.42` 是 swap/commission 在无交易状态下的账面损耗。

### 对比

| 参数 | 正常回测 | 调优（硬编码） |
|------|---------|--------------|
| leverage | 1000 | **1** |
| commission | 0.001 | 0.001 |
| slippage | 0 | 0 |
| swapRate | 有（config_snapshot） | **无** |
| marginCallLevel | 有（config_snapshot） | **无** |

## 修复方案

`backtest_run_id` 一字段同时解决两个问题：追溯显示 + 配置继承。两个入口都必然有 `backtest_run_id`（调优前必须先跑回测），因此只需一个数据源——从 `backtest_runs` 表读取配置，无需 `exec_config` 列或前端传配置。

### 两个入口分析

两个入口最终都通过 `BacktestPanel.tsx:152` 的 `onRunTuning` 调用 `runner.tuning.runTuning`，且 `runner.runId` 都**非空**：

**入口 A：跑完回测 → 切 Tuning tab**

用户编写/导入策略 → 运行回测（必选 symbol、填参数）→ `runner.run()` 设置 `runId` → 得到回测结果 → 切到 Tuning tab → 调优。`runner.runId` 非空。

**入口 B：回测历史 → 加载 → 切 Tuning tab**

`BacktestHistoryDrawer` 选中已完成回测 → `loadRunById(id)` 设置 `runId` + `runMeta` → 展示回测结果 → 切到 Tuning tab → 调优。`runner.runId` 非空。

**结论**：两个入口的 `runner.runId` 都非空，统一处理，无需分支。

### DB migration

```sql
ALTER TABLE strategy_experiments
  ADD COLUMN IF NOT EXISTS strategy_name TEXT DEFAULT '',
  ADD COLUMN IF NOT EXISTS backtest_run_id UUID;
```

只加两列。不存冗余配置——运行时从 `backtest_runs` 表读取。

### proto `strategy_experiment.proto`

```protobuf
// SubmitStrategyExperimentRequest 加:
string strategy_name = 12;
string backtest_run_id = 13;

// StrategyExperiment 加:
string strategy_name = 19;
string backtest_run_id = 20;
```

`backtest_run_id` 为 required（前端保证非空传入）。无需 `execution_config` 字段——后端从 `backtest_runs` 表读取配置。

### 后端

**repo + handler（问题一：追溯）**：

- `repository/strategy_experiment_repository.go`：struct 加 `StrategyName string` + `BacktestRunID *uuid.UUID`；Create/Get/List/Cancel SQL 加两列
- `connect/strategy/experiment_handler.go`：submit 读 proto 新字段填入 struct；experimentToProto 输出

**ExecuteBacktestDirect（问题二：配置继承）**：

- `connect/strategy/backtest_execution.go`：`ExecuteBacktestDirect` 签名加 `backtestRunID string` 参数
- 当 `backtestRunID` 非空时，从 `backtest_runs` 表读取 `leverage/commission/slippage/strict_mode/config_snapshot`（含 swapRate/marginCallLevel），替代硬编码默认值
- `connect/strategy/experiment_scoring.go`：`runSingleBacktest` 传 `exp.BacktestRunID` 给 `ExecuteBacktestDirect`
- `connect/strategy/strategy_experiment_worker.go`：`backtestAndScore` 传 `exp.BacktestRunID` 给 `runSingleBacktest`

### 前端

**`client/strategyExperiment.ts`**：`SubmitStrategyExperimentParams` 加 `strategyName?: string` + `backtestRunId?: string`

**`hooks/useTuning.ts`**：`runTuning` 参数加 `strategyName?`/`backtestRunId?`，传入 `strategyExperimentApi.submit`

**`components/backtest/BacktestPanel.tsx:152`**（两个入口共享的调用点）：
- `onRunTuning` 回调加入 `runner.runId`（两个入口都非空）和策略名
- 策略名优先级：`runner.runMeta.name`（非空）→ 模板名 → `"未命名策略"`
- `canRun` 加 `runner.runId` 非空检查（无回测结果时禁用调优按钮）

**`hooks/useStrategyWorkspaceState.ts:44`**：
- `handleRunTuning` 传入 `backtestRunId: btCtx.runId`（非空）

**`BatchTuningPanel.tsx`**：实验列表加策略名列

**`SmartTuningPanel.tsx`**：结果区显示策略名

### 策略名来源优先级

1. `backtest_runs.name`（非空时，通过 `backtest_run_id` JOIN 读取）
2. 策略模板名（有 templateId 时）
3. `"未命名策略"` fallback

### 不做

- ❌ 不改 backtest_runs 表
- ❌ 不改实验评分流程（Score/OOS/过拟合机器不变）
- ❌ 不加 strategy_id 外键
- ❌ 不在 experiments 表存冗余回测配置（leverage/commission 等）——运行时从 backtest_runs 读取
