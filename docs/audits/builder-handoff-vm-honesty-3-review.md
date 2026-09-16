# 施工派工单：VM-HONESTY-3-REVIEW

> **角色纪律**：你是施工方，不是验收方。完成后保持 `🟦open（施工完成，待独立复审）`，由 Devin CLI 独立复审后才可变更为 `✅done`。禁止扩 scope、提交、push、部署。commit 用 `ANT_ROLE=builder` 前缀。

## 立项背景

**触发**：MQL-HONESTY-3 的对抗测试未真正证明 `IsReliable=false` 由 fatal blind spot 置位——`assessRisk` 对 `<10` trades 本来就返回 `IsReliable=false`，删除 `buildBacktestResponse` 的 fatal-severity loop 仍可能通过。且 `TestHONESTY3_NonFatalBlindSpotKeepsReliable` 当前 `trades=0`（实测 2026-09-16），空转通过，未验证 fatal loop 不误伤非致命 blind spot。

**证据链**（HEAD `bae29d4a` 实拍，2026-09-16 核实）：
- `backtest_worker_helpers.go:171` `IsReliable: m.TotalTrades >= 10`——`assessRisk` 对 <10 trades 直接 false。
- `backtest_worker_vm.go:338` `resp.Risk = assessRisk(result.Metrics)`；`:344-349` HONESTY-3 fatal-severity loop：`for _, bs := range resp.BlindSpots { if bs.Severity == interp.SeverityFatal { resp.Risk.IsReliable = false; break } }`。
- `honesty_fatal_blindspot_test.go:46` `TestHONESTY3_FatalBlindSpotSetsUnreliable`：策略 `iNonExistentIndicator` 顶层调用 → 运行时返 0 → `v>0` 恒假 → 0 trades → assessRisk 已 false。删 fatal loop 仍 GREEN（非对抗）。
- `honesty_fatal_blindspot_test.go:133` `TestHONESTY3_NonFatalBlindSpotKeepsReliable`：MA14 + makeE2EBars(80) 实测 `trades=0`（EMA14 在 20-bar 振荡周期上平滑过度，`ma>maPrev` 不触发）→ 空转通过；且 `OrderClose(OrderTicket(),...)` 缺前置 `OrderSelect` → `OrderTicket()` 返 0。
- `engine.go:133` `allTrades := append(e.trades, e.broker.Trades()...)` + `metrics.go:25` `TotalTrades: int32(len(trades))`——VMRunner 直连 Broker() 的交易**计入** TotalTrades（engine_validation_test:81 `len(result.Trades)==1` 实证）。故"≥10 trades → assessRisk=true"在本 harness 可行。
- `iNonExistentIndicator` 运行时非 fatal（返 0 + recordBlindSpot，engine.Run 不报错——现 Fatal 测试已证），但静态 coverage 判 `SeverityFatal`（`interp/analyze.go:43` iXxx 模式）。死分支放它可解耦"静态 fatal blind spot"与"运行时交易"。
- 非致命 blind spot 确定性触发：`rule_engine.go:257-272` `ruleOrderSelectHistory`（R06，`sevWarningEn`）——源码含 `ORDERSELECT`+`MODE_HISTORY` 即触发，经 `runDiagnostics`→`attachBlindSpots:433` 入 `resp.BlindSpots`（在 fatal loop `:344` 之前）。

**不变量**：本任务**只改测试文件** `honesty_fatal_blindspot_test.go`，零生产代码改动。HONESTY-3 fatal-severity loop（`backtest_worker_vm.go:344-349`）保持不变——它是被测对象，不是被改对象。

## 设计 SSOT

### S1：重构 `TestHONESTY3_FatalBlindSpotSetsUnreliable`（`honesty_fatal_blindspot_test.go:46`）

目标：证明 `IsReliable=false` **仅**由 HONESTY-3 fatal-severity loop 置位，而非 `assessRisk` 的 <10 trades 规则。

策略源（替换 `:48-58` 的 source）：
- MA 交叉产生 **≥10 笔已平仓交易**：`iMA(Symbol(),0,MAPeriod,0,MODE_EMA,PRICE_CLOSE,1)` vs shift 2，`MAPeriod=3`（快线，跟踪 20-bar 振荡紧密），`makeE2EBars(200)`（振荡每 10 bar 反转 → 频繁交叉）。开仓 `if(ma>maPrev && OrdersTotal()==0) OrderSend(Symbol(),OP_BUY,LotSize,Ask,5,0,0,"T",MagicNumber,0,clrGreen);`；平仓 `if(ma<maPrev && OrdersTotal()>0){ if(OrderSelect(0,SELECT_BY_POS,MODE_TRADES)) OrderClose(OrderTicket(),LotSize,Bid,5,clrRed); }`。
- fatal indicator 放入**死分支** `if(1==0){ double v = iNonExistentIndicator(Symbol(),0,14,0,0); }`——静态 coverage 仍扫到该 call 节点 → 记 SeverityFatal blind spot；运行时永不执行 → 不干扰交易。

断言（替换 `:104-122`）：
1. `cov.BlindSpots` 含 ≥1 个 `SeverityFatal`（证明静态 coverage 仍捕获死分支内的 iNonExistentIndicator——若死分支被编译器/coverage 优化掉导致漏捕获，改用 `if(false)` 或运行时恒假变量 `if(MagicNumber==999999)` 等价死分支，重新验证捕获）。
2. `result.Metrics.TotalTrades >= 10`（**关键**：证明 assessRisk 会设 `IsReliable=true`，故 false 只能来自 fatal loop）。
3. `resp.Risk.IsReliable == false`（fatal loop 置位）。
4. `resp.BlindSpots` 含 ≥1 个 `SeverityFatal`。

**对抗证明**：注释 `backtest_worker_vm.go:344-349` fatal-severity loop（或改为 `if false {}`）→ 本测试 RED（`IsReliable=true but fatal coverage blind spots present`，且 trades≥10 故 assessRisk 不再兜底）→ 恢复 → GREEN。

### S2：重构 `TestHONESTY3_NonFatalBlindSpotKeepsReliable`（`honesty_fatal_blindspot_test.go:133`）

目标：证明 HONESTY-3 fatal-severity loop **不误伤**非致命 blind spot。

策略源（替换 `:136-150`）：
- 同 S1 的 MA 交叉（MAPeriod=3, makeE2EBars(200), OrderSelect 前置 OrderClose）产生 ≥10 trades。
- 触发一个**确定性非致命 blind spot**：在源码中包含 `OrderSelect(0, SELECT_BY_POS, MODE_HISTORY)`（放死分支 `if(1==0){ OrderSelect(0,SELECT_BY_POS,MODE_HISTORY); }` 即可——R06 是源码文本扫描，case-insensitive contains，无需运行时执行）→ R06 `sevWarningEn` finding → `resp.BlindSpots` 含 `SeverityWarning` 项。
- **不含**任何 fatal blind spot（无 iXxx 未知指标、无 iCustom、无 OrderSelect+MODE_HISTORY 之外的 fatal 触发）。

断言（替换 `:170-215`）：
1. `result.Metrics.TotalTrades >= 10`。
2. `resp.BlindSpots` 中 **无** `SeverityFatal`（若有，Fatalf 报来源）。
3. `resp.BlindSpots` 中 **有** ≥1 个 `SeverityWarning`（R06_orderselect_history）——证明非致命 blind spot 确实存在并经过 fatal loop。
4. `resp.Risk.IsReliable == true`（**强断言**，替换原容忍 false 的弱逻辑）——fatal loop 见 warning 不翻转。

**对抗证明**：将 `backtest_worker_vm.go:345` `if bs.Severity == interp.SeverityFatal` 改为 `if bs.Severity != interp.SeverityInfo`（让 loop 对 warning 也翻转）→ 本测试 RED（`IsReliable=true` 断言失败，warning blind spot 现被误判）→ 恢复 → GREEN。

### S3：`TestHONESTY3_UnsupportedSilentWrongIsFatal`（`:231`）保持不变

该测试是纯 SeverityForBuiltin 分类断言，不依赖 trades/engine，已对抗有效（mutation：改 SeverityForBuiltin 分类 → RED）。**不改**。

## 约束与目标

- **范围**：仅 `backend/internal/connect/strategy/honesty_fatal_blindspot_test.go`。零生产代码、零其他测试文件。
- **不碰**：`backtest_worker_vm.go`（fatal loop 是被测对象）、`backtest_worker_helpers.go`（assessRisk 是被测对象）、`rule_engine.go`、`honesty_audit_test.go`（mql2go 包内另一组 HONESTY 测试）、`TestHONESTY3_UnsupportedSilentWrongIsFatal`。
- **REUSE**：`buildBacktestResponse` @ `backtest_worker_vm.go:274`；`assessRisk` @ `backtest_worker_helpers.go:93`（IsReliable 阈值 `:171`）；`CompileMQLWithCoverage` @ `interp_runner.go`；`backtest.New` @ `engine.go`；`makeE2EBars` @ `honesty_fatal_blindspot_test.go:18`（本文件既有 helper，扩到 200 bars 仅改调用参数）；`backtestParams` struct @ 本文件既有；MA 交叉 EA 模式参考 `tools/mql2go/e2e_test.go:40-54`（OrderSelect+OrderClose 循环模式）。
- **NEW**：无新文件、无新 helper（makeE2EBars 已存在，仅改 n 参数）。
- **先红后绿**：S1/S2 各 1 项对抗证明（mutation RED→restore GREEN），共 2 项。mutation 临时改 `backtest_worker_vm.go` 验证后**必须恢复**，最终 diff 只含测试文件。
- **门禁**：`go build ./...` ✓ / `go test ./internal/connect/strategy/` ✓（完整 package，3 个 HONESTY3 测试全绿）/ `go test -race ./internal/connect/strategy/` ✓ / `go vet ./internal/connect/strategy/` ✓ / gofmt ✓ / `go run ./tools/check-file-lines --strict` 0 errors / `git diff --check` clean。
- **串行**：本单完成后报 `[施工完成:VM-HONESTY-3-REVIEW] @<hash>`，等 Devin CLI 独立复审。勿部署。

## 边界 / 不做

- 不改 fatal-severity loop 的逻辑或阈值——它是被测对象；对抗证明的 mutation 是临时验证，恢复后 diff 不得含生产代码改动。
- 不引入新 blind spot 类型/新 rule——R06 是既有 rule，仅复用其源码文本触发。
- 不为"≥10 trades"硬编码结果——必须由真实 MA 交叉 + engine.Run 产生；若 MAPeriod=3/200bars 仍 <10 trades，调参（MAPeriod=2、bars=300、或改用 close vs close 比较）直到 ≥10，并在测试日志打印实际 trades 数。
- 不碰 `TestHONESTY3_UnsupportedSilentWrongIsFatal`（已对抗有效）。
- 不改 `makeE2EBars` 的振荡逻辑（既有 (i/10)%2 模式适合触发交叉）。
