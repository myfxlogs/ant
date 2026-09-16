# builder-handoff-qs-3-baseline — VM 性能基线测量（只测不优化）

> v1 @2026-09-16（VM 质量方案 v2 阶段 1/2 全部 ✅done 后的贯穿项收口）

## 立项背景

spec §5，registry QS-3-BASELINE。D-009 已否决 v1 阶段 3 全部优化项——**本单只产数据**：benchmark 基线 + 生产 metric 接线，为"是否/何处优化"提供量化判据。判据（spec 原文）：生产 p99 单事件耗时 > tick 间隔 10% 或 benchmark 某项占比 > 30% 才允许立优化条目。

## 设计 SSOT

- spec §5 + 本文件坐标（D-013 已核实）。
- 既有 metric 惯例：`internal/connect/strategy/metrics.go`（promauto + `ant_strategy_*` 命名 + 显式 buckets），复用不另起体系。
- `vm.ticks`（`vm_execute.go:28/:354` 自增，`runEvent` :214 每事件清零）已存在——指令计数不需新建，只差导出访问器。

## 约束与目标

1. **禁止任何性能优化改动**——benchmark/观测代码之外的生产行为零变更（含不"顺手"改结构体/缓存）。
2. 改动面：`tools/mql2go/`（benchmark 文件 + 最小访问器）+ `internal/connect/strategy/metrics.go`（新 metric）+ 事件调用点接线 + `docs/audits/vm-perf-baseline-2026-09.md`（结果落盘）。
3. 串行，勿部署、勿 push、禁 `--no-verify`；commit 用 `ANT_ROLE=builder git commit` 前缀。
4. 完工按 D-012 自审 + D-014 末行 `[施工完成:QS-3-BASELINE] @<commit-hash>` 自报，停手等复审。

## S1 — benchmark 套件（`tools/mql2go/vm_bench_test.go` 新文件）

四项基准（`go test -bench=. -benchtime=1s -run=^$ ./tools/mql2go/` 出数）：

- **B1 dispatch 循环**：编译一个最小策略（如 `OnTick` 内一次赋值），测 `RunOnTick`/`RunOnBar` 单事件全链路耗时（`runEvent` 重置+runLoop 开销）。
- **B2 builtin 调用**：选高频 builtin（如 `Close(0)` 类序列访问 + 一个纯计算 builtin）单点耗时——区分"builtin 调度开销"与"builtin 本体开销"。
- **B3 Decimal 算术**：`executeArith` 路径加减乘除单 op 耗时（decimal 运算 vs int 运算对照，量化 decimal 惩罚）。
- **B4 典型策略 1000 tick**：编译一个现实形态策略（MA 交叉类：每 tick 若干指标+比较+条件单），驱动 1000 次 `RunOnTick`/`RunOnBar`，总耗时与单事件均值。
- 策略源用既有 `CompileMQL`/`CompilePython`（`compile_py_bool_test.go:29` 用法范式）+ `VMRunner`/直接 `vm.RunOnBar(ctx)`；ctx 用 noop 或最小 stub——benchmark 测 VM 本身不是 ctx 实现，须在报告注明 ctx 形态。
- **结果落盘** `docs/audits/vm-perf-baseline-2026-09.md`：每项 ns/op + B/op + allocs/op + 环境（CPU/go version）+ 占比分析（哪项 >30% 阈值线）。

## S2 — 生产 metric 接线

`internal/connect/strategy/metrics.go` 追加（沿用 promauto + `ant_strategy_*`）：

- `ant_strategy_vm_event_duration_seconds` HistogramVec{event}——VM 单事件耗时，buckets 按 tick 间隔量级设计（如 1ms/5ms/10ms/50ms/100ms/500ms/1s/5s，builders 可按 live 事件周期实况微调并注明理由）。
- `ant_strategy_vm_event_instructions` HistogramVec{event}——每事件指令数（来源 `vm.ticks`，需访问器）。
- `ant_strategy_vm_fatal_total` CounterVec{event}——fatalError 发生计数。

**接线点**（决策方实拍坐标，施工方复核可达性）：live 事件链为 `vm_live_handlers.go` → `runner.Runner.OnBar/OnTick`（`strategy/runner/runner.go:146/:170`）→ `VMRunner.OnBar/OnTick`（`interp_runner.go:301/:315`）→ `vm.RunOnX`。**observe 点定在 `vm_live_handlers.go:53`（`r.OnBar`）与 `:136`（`r.OnTick`）调用帧**——`time.Since(start)` 恰包住 VM 执行（context 装配在前、signal 转换在后，均不计入）；同文件 trade/timer 等其余 `vmHandle*` 站点可按同一范式补 event label。**只接 live 路径**——backtest 每秒数千事件，直方图成本本身即开销（如需另议）。

**stats 透传**（分层约束：metrics.go 在 connect/strategy，`runner` 在 strategy/runner 不可反向 import）：`vm.ticks`/`vm.fatalError` 均非导出（实拍无 getter）——新增最小访问器 `Ticks() int`/`FatalError() string`（注释"QS-3-BASELINE 观测用"）；`VMRunner` 暴露 `LastEventStats()`（或实现可选接口如 `sdk.EventStatsProvider`）；`runner.Runner` 加 `EventStats()` 做 type-assert 透传；`vm_live_handlers` 断言成功后 observe。链路任一环缺接口则 degrade 为只记 duration（报告注明）。

## S3 — 报告与判据声明

`vm-perf-baseline-2026-09.md` 结构：①环境 ②B1-B4 数据表 ③占比分析 + 是否触及 >30% 阈值 ④metric 清单（名称/类型/labels/buckets/接线点）⑤**明确声明：本报告不构成优化立项依据**，触及阈值时另走决策。

## 验收标准

- [ ] `go test -bench=. -run=^$ ./tools/mql2go/` 四项基准出数且原样入报告
- [ ] `go build ./...`、`go test -count=1 ./tools/mql2go/... ./internal/connect/strategy/` 全绿
- [ ] `go test -race -count=3 ./tools/mql2go/ ./internal/connect/strategy/` 通过（goleak 仍绿——新 goroutine 禁止引入）
- [ ] check-lines 零新增错误；gofmt/vet 零新增
- [ ] metric 命名/labels 与既有 `ant_strategy_*` 惯例一致；接线点 file:line 入报告
- [ ] mutation×1：`Ticks()` 返回常数 0 → 报告或一处 pin 测试证明指令数 metric 来源真实（如断言 benchmark 内 ticks>0）；restore→GREEN

## 边界/不做

- 不优化任何东西；不接 backtest 路径 metric；不改 `vm.ticks` 计数语义；不动 prometheus registry 全局配置。
- benchmark 策略源不与生产策略文件耦合（测试内嵌源码即可）。

## D-013 出件自审记录（决策方）

- 坐标回验：`metrics.go` 43 行全读（promauto/`ant_strategy_*` 惯例实拍）；`vm.ticks` 自增点 `vm_execute.go:28/:354` + `runEvent:214` 清零 + `MaxTicks` 上限已核实；`vm.ticks`/`fatalError` 无导出访问器（grep 实拍）；`runEvent` :193-224 事件入口已读；`RunOnBar` 经 `interp_runner.go:304` safeRun 调用已核实。
- 路径可达性：live 事件调用链实拍 `vm_live_handlers.go:53/:136` → `runner.Runner.OnBar/OnTick`（runner.go:146/:170）→ `VMRunner.OnBar/OnTick`（interp_runner.go:301/:315）→ `vm.RunOnX` → `runEvent`——observe 点定在 connect/strategy 层 `r.OnBar/r.OnTick` 调用帧（metrics.go 同包，免分层依赖问题；初稿猜 live_context.go 为误，已按实拍修正）。stats 透传走"VM 访问器→VMRunner→Runner type-assert→handlers"四级，任一级缺失降级为只记 duration。
- 生命周期：`vm.ticks` 每事件清零（:214）——读取必须在事件返回后立即取，接线注释须写明；`fatalError` 同样在 :199 每事件重置。
- 冲突检查：D-009 否决优化项 → 本单明确"只测不优化"并写入边界；与 QS-2.2 goleak 无冲突（metric 无线程）。
- 复用核对：prometheus client_golang v1.23.2 直接依赖已钉版；`emptyBars`/`noopContext`（QS-2.3）可作 benchmark ctx 基座复用。
- **署名**：最终决策：Devin CLI（[角色:决策终] 激活）
