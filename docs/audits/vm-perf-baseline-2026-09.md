# vm-perf-baseline-2026-09 — VM 性能基线测量

> QS-3-BASELINE（spec §5）。**只测不优化**：本报告不构成优化立项依据。
> 判据（spec 原文）：生产 p99 单事件耗时 > tick 间隔 10% 或 benchmark 某项占比 > 30% 才允许立优化条目。
> 测量命令：`go test -bench=. -benchtime=1s -benchmem -run='^$' ./tools/mql2go/`

## 1. 环境

| 项 | 值 |
|----|----|
| CPU | Intel(R) Xeon(R) Gold 6138 @ 2.00GHz（-4，4 核配额） |
| OS/Arch | linux/amd64 |
| Go | go1.26.6 |
| ctx 形态 | `noopContext`（QS-2.3）——bare `NewVM`，`Bars()=emptyBars`、`Broker()=nil`、指标全 0。benchmark 测 VM 本身，不含任何 sdk.Context 实现成本 |
| 代码点 | commit `5ad339a9` 之后工作树（QS-2.3 不变量已落地） |

## 2. B1–B4 数据表（原样机出）

```text
BenchmarkDispatchOnTick-4         10841850        128.2 ns/op        0 B/op        0 allocs/op
BenchmarkDispatchOnBar-4          10390771        115.5 ns/op        0 B/op        0 allocs/op
BenchmarkBuiltinSeriesAccess-4     2580853        435.0 ns/op       96 B/op        1 allocs/op
BenchmarkBuiltinPureCompute-4      1980846        605.3 ns/op      136 B/op        3 allocs/op
BenchmarkArithDecimalAdd-4         3289708        360.5 ns/op       80 B/op        2 allocs/op
BenchmarkArithIntAdd-4             8322873        137.4 ns/op        0 B/op        0 allocs/op
BenchmarkArithDecimalMul-4         2714617        383.6 ns/op       80 B/op        2 allocs/op
BenchmarkArithIntMul-4             7453544        142.5 ns/op        0 B/op        0 allocs/op
BenchmarkArithDecimalDiv-4         1114930       1017   ns/op      184 B/op        7 allocs/op
BenchmarkArithIntDiv-4             7199930        140.9 ns/op        0 B/op        0 allocs/op
BenchmarkStrategy1000Ticks-4             15   67872731 ns/op  10762700 B/op   310014 allocs/op
```

策略形态说明：

- **B1** `OnTick(){ g=1 }` / `OnBar(){ g=1 }`——runEvent 全重置 + runLoop + 1 push + 1 store。
- **B2** `g=Close(0)`（序列访问，noop 空序列）vs `g=MathAbs(-1.5)`（纯计算 builtin）——单事件单 builtin，开销 = dispatch(~120ns) + call_builtin 调度 + builtin 本体。
- **B3** 直测 `executeArith` 单 op（含 push×2 + pop×1 栈操作摊入）。
- **B4** MA 交叉形态：每 tick `10×Close(i) + 30×Close(i)` 循环 + 2 次 decimal 除法 + 比较 + 条件存储，驱动 1000 次 RunOnTick。**单事件均值 ≈ 67.9 µs**（67,872,731/1000），每事件 ~310 allocs / ~10.8KB。

## 3. 占比分析与阈值判定

- **dispatch 地板 ≈ 116–128 ns/事件**——runEvent 重置 + runLoop 框架成本，可忽略。
- **builtin 调用附加 ≈ 310–480 ns/次**（B2 减 B1）——调度+本体合计，其中空序列 Close(0) 已含 1 alloc。
- **decimal 惩罚**：add/mul ≈ **2.6–2.7×** int；**div ≈ 7.2×** int（1017 vs 141 ns），且 div 每次 7 allocs。
- **B4 单事件 67.9 µs**：40 次 builtin 调度 ≈ 12–19 µs（占 18–28%）；decimal 算术（~84 次加 + 2 次除 + 比较）与栈/循环指令占主体。
- **>30% 阈值判定**：benchmark 内无单一项（dispatch / builtin 调度 / 单一算术 op）占单事件耗时 >30%——builtin 调度合计 ~18–28%，decimal 合计虽为主体但分散于多 op。**未触及 benchmark 占比阈值**。生产 p99 判据需 metric 上线后观测（见 §4）。
- 结论：数据仅登记基线。**是否/何处优化不在本单范围**。

## 4. 生产 metric 接线清单（live 路径 only）

| metric | 类型 | labels | buckets / 说明 | 接线点 |
|--------|------|--------|----------------|--------|
| `ant_strategy_vm_event_duration_seconds` | HistogramVec | `event` ∈ {bar, tick, trade, trade_transaction, timer} | 0.001/0.005/0.01/0.05/0.1/0.5/1/5 s | `vm_live_handlers.go` observeVMEvent：`r.OnBar`(原:53)、`r.OnTick`(原:136)、`r.OnTrade`、`r.OnTradeTransaction`、`r.OnTimerTick` 调用帧 |
| `ant_strategy_vm_event_instructions` | HistogramVec | `event` 同上 | 10/1e2/1e3/1e4/1e5/1e6/1e7（上界=MaxTicks） | 同上；来源 `vm.Ticks()` |
| `ant_strategy_vm_fatal_total` | CounterVec | `event` 同上 | — | 同上；`vm.FatalError() != ""` 时 Inc |

- **stats 透传链**（四级，任一缺失降级为仅记 duration）：`vm.Ticks()`/`vm.FatalError()`（vm.go，QS-3-BASELINE 观测访问器）→ `VMRunner.LastEventStats()`（interp_runner.go）→ `Runner.EventStats()` type-assert（strategy/runner/runner.go）→ `observeVMEvent`（vm_live_handlers.go）。
- **生命周期约束（代码注释已写明）**：`vm.ticks`/`vm.fatalError` 每事件入口重置（runEvent :214/:199）——observe 必须在 runner 调用返回后、下一事件前取，接线点满足。
- **只接 live**：backtest 每秒数千事件，直方图自身成本即偏差（handoff 边界）。

## 5. 声明

本报告为测量基线登记，**不构成优化立项依据**。触及阈值（生产 p99 > tick 间隔 10% 或 benchmark 单占比 >30%）时另走决策流程立项。
