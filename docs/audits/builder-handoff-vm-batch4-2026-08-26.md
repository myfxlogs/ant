# Builder Handoff — VM 返工批第四批：VM-TIMESERIES-SEMANTICS-1 + VM-RUNTIME-FAILCLOSED-1（语义正确性 + fail-closed）

> 日期：2026-08-26 ｜ 审计方：Devin CLI（项目第一负责人） ｜ 施工方：Devin IDE
> 设计 SSOT：`docs/spec/vm-revert-redo-spec.md`（D1-D4 不变）
> registry spec：`:108`（VM-TIMESERIES-SEMANTICS-1）、`:111`（VM-RUNTIME-FAILCLOSED-1）
> 基线 HEAD：`2845a428`（2026-08-26，第三批 VM-COMPILER-SEMANTICS-1 + BT-FUNC-ENTRYPC-FWD 验收通过后）

## 0. 立项背景

**触发**：D-REVERT-SCOPE-DRIFT-001 第四批——revert `830b2c79` 导致 VM-TIMESERIES-SEMANTICS-1 和 VM-RUNTIME-FAILCLOSED-1 修复代码丢失，registry 状态从"施工完成待复审"降级回"🟦open（待施工）"。

**证据链**（基于 HEAD `2845a428` 实测）：

VM-TIMESERIES-SEMANTICS-1：
- `vm_builtin_mql5_ts.go:251` `builtinCopyTime` 用 `int32(s.Time(shift))`（裸 unix_ms），缺少 `/ 1000` 转 unix seconds
- `vm_builtin_mql5_ts.go:40-96` `builtinIHighest`/`builtinILowest` 硬编码 `series.High(i)`，无 `extremeIndex`/`valueAt`/`validSeriesMode` mode 分支
- `vm_builtin_mql5_ts.go:51-56` 越界 guard 只有 `start < 0`，无 `series.Len()==0 || start>=series.Len()` 检查，无 count clamp
- `vm_builtin_mql5_ts.go:23-38` `builtinIBarShift` 无 `exact` 参数处理，始终 `series.Time(i) <= ts`
- ✅ `vm_builtin_mql5_ts.go:145-186` `copyBarData` 方向语义已存在（count>0 chronological, count<0 reverse）

VM-RUNTIME-FAILCLOSED-1：
- `vm_helpers.go:210-234` `callBuiltin` handler 返回 nil error 后无 `fatalError` 检查
- `vm_helpers.go:29-54` `pop`/`popN` 栈下溢返回 `NoneVal`/截断，无 `setStackError` 调用
- `vm_execute.go:116-120` `OP_CALL_BUILTIN` push 后无 `fatalError` 检查
- `engine.go:107-111` `Engine.Run` 策略事件 error 只写 stderr + continue，未 fail-closed return error
- ✅ `vm_execute.go:13-16` `runLoop` 顶部 `fatalError` 检查已存在
- ✅ `vm_helpers.go:218-222` builtin Go error 包装已存在（但返回 NoneVal 非 fail-closed）
- `vm_builtin_indicators.go:195-200` `iADX:MODE_PLUSDI`/`MODE_MINUSDI` 只 `recordBlindSpot` 返回零值，不设 `fatalError`

**设计 SSOT 声明**：`docs/spec/vm-revert-redo-spec.md`（D1-D4 设计决策不变）

**约束与目标**：
- 基于 registry 中的原始修复 spec 重新施工，不做架构变更（D3）
- 每批独立验收（D4）
- revert 不可逆，不尝试恢复被 revert 的代码（D1）

**边界/不做**：
- 不改写历史审计事实
- 不改 proto/gen 文件
- 不改第三批已验收的 `patchUserCalls`/`ClassTypes`/`compileDeclaration` 等

---

## 1. VM-TIMESERIES-SEMANTICS-1（S1-S5）

### S1：CopyTime seconds 转换

**目标**：`builtinCopyTime` 将 `unix_ms` 转为 MQL `datetime`（unix seconds）。

**当前代码**（`vm_builtin_mql5_ts.go:245-254`）：
```go
func builtinCopyTime(vm *VM, args []interp.Value) (interp.Value, error) {
    series, ok := resolveSeries(vm, 0, 1, args)
    if !ok {
        return interp.IntVal(-1), nil
    }
    n := copyBarData(args, series, func(s sdk.BarSeries, shift int) interp.Value {
        return interp.IntVal(int32(s.Time(shift)))  // ❌ 缺少 / 1000
    })
    return interp.IntVal(n), nil
}
```

**施工要求**：`vm_builtin_mql5_ts.go:251` 改为：
```go
return interp.IntVal(int32(s.Time(shift) / 1000)) // VM-TIMESERIES-SEMANTICS-1: unix_ms → unix seconds
```

### S2：iHighest/iLowest mode 分支 + extremeIndex 重构

**目标**：`iHighest`/`iLowest` 按 `ENUM_SERIESMODE`（0-5）选择字段，不硬编码 High/Low。

**当前代码**（`vm_builtin_mql5_ts.go:40-96`）：硬编码 `series.High(i)`，无 mode 分支。

**施工要求**：

① 新增 `validSeriesMode` 函数：
```go
// validSeriesMode returns true if mode is a valid ENUM_SERIESMODE (0-5).
// VM-TIMESERIES-SEMANTICS-1
func validSeriesMode(mode int32) bool {
    return mode >= 0 && mode <= 5
}
```

② 新增 `valueAt` 函数按 mode 选择字段：
```go
// valueAt returns the bar value for the given series mode.
// VM-TIMESERIES-SEMANTICS-1: ENUM_SERIESMODE 0-5 → Open/Low/High/Close/Volume/Time
func valueAt(s sdk.BarSeries, idx int, mode int32) decimal.Decimal {
    switch mode {
    case 0: return s.Open(idx)   // MODE_OPEN
    case 1: return s.Low(idx)    // MODE_LOW
    case 2: return s.High(idx)   // MODE_HIGH
    case 3: return s.Close(idx)  // MODE_CLOSE
    case 4: return decimal.NewFromInt(s.Volume(idx)) // MODE_VOLUME
    case 5: return decimal.NewFromInt(s.Time(idx))   // MODE_TIME
    default: return decimal.Zero
    }
}
```

③ 新增 `extremeIndex` 函数（集中 iHighest/iLowest 逻辑）：
```go
// extremeIndex finds the index of the extreme value in series[start, start+count).
// findMax=true → iHighest, findMax=false → iLowest.
// VM-TIMESERIES-SEMANTICS-1
func extremeIndex(vm *VM, series sdk.BarSeries, mode, start, count int32, findMax bool) int32 {
    if series.Len() == 0 || start < 0 || int(start) >= series.Len() {
        return -1
    }
    if !validSeriesMode(mode) {
        vm.recordBlindSpot(fmt.Sprintf("invalid series mode: %d", mode))
        return -1
    }
    // Clamp count to remaining bars
    remaining := int32(series.Len()) - start
    if count <= 0 || count > remaining {
        count = remaining
    }
    extIdx := start
    extVal := valueAt(series, int(start), mode)
    for i := start + 1; i < start+count; i++ {
        v := valueAt(series, int(i), mode)
        if findMax {
            if v.GreaterThan(extVal) {
                extVal = v
                extIdx = i
            }
        } else {
            if v.LessThan(extVal) {
                extVal = v
                extIdx = i
            }
        }
    }
    return extIdx
}
```

④ 重写 `builtinIHighest`/`builtinILowest` 调用 `extremeIndex`：
```go
func builtinIHighest(vm *VM, args []interp.Value) (interp.Value, error) {
    series, ok := resolveSeries(vm, 0, 1, args)
    if !ok {
        return interp.IntVal(-1), nil
    }
    mode := argI(args, 2)   // ENUM_SERIESMODE
    count := argI(args, 3)
    start := argI(args, 4)
    return interp.IntVal(extremeIndex(vm, series, mode, start, count, true)), nil
}

func builtinILowest(vm *VM, args []interp.Value) (interp.Value, error) {
    series, ok := resolveSeries(vm, 0, 1, args)
    if !ok {
        return interp.IntVal(-1), nil
    }
    mode := argI(args, 2)
    count := argI(args, 3)
    start := argI(args, 4)
    return interp.IntVal(extremeIndex(vm, series, mode, start, count, false)), nil
}
```

**落点**：`vm_builtin_mql5_ts.go:40-96` 重写 + 新增 `validSeriesMode`/`valueAt`/`extremeIndex`

### S3：越界 guard（已含在 S2 extremeIndex 中）

S2 的 `extremeIndex` 已包含越界 guard（`series.Len()==0 || start<0 || start>=series.Len()` → 返回 -1）和 count clamp。无需额外步骤。

### S4：iBarShift exact 参数

**目标**：`builtinIBarShift` 支持 `exact` 参数——exact=true 只精确匹配，exact=false 返回 `time<=ts` 的最近 bar。

**当前代码**（`vm_builtin_mql5_ts.go:23-38`）：
```go
func builtinIBarShift(vm *VM, args []interp.Value) (interp.Value, error) {
    // ...
    ts := int64(argI(args, 2)) * 1000
    for i := 0; i < series.Len(); i++ {
        if series.Time(i) <= ts {  // ❌ 始终非精确
            return interp.IntVal(int32(i)), nil
        }
    }
    return interp.IntVal(-1), nil
}
```

**施工要求**：
```go
func builtinIBarShift(vm *VM, args []interp.Value) (interp.Value, error) {
    series, ok := resolveSeries(vm, 0, 1, args)
    if !ok {
        return interp.IntVal(-1), nil
    }
    ts := int64(argI(args, 2)) * 1000
    // VM-TIMESERIES-SEMANTICS-1: exact parameter
    exact := len(args) > 3 && argI(args, 3) != 0
    for i := 0; i < series.Len(); i++ {
        barTs := series.Time(i)
        if barTs == ts || (!exact && barTs < ts) {
            return interp.IntVal(int32(i)), nil
        }
    }
    return interp.IntVal(-1), nil
}
```

**落点**：`vm_builtin_mql5_ts.go:23-38` 重写

### S5：对抗证明

4 项突变：

1. 删 CopyTime `/1000`（用 `s.Time(shift)` 裸 ms）→ `TestVM_Audit_CopyTime_SecondsConversion` RED（int32 溢出，值 != 预期 unix seconds）
2. 删 `extremeIndex` mode 分支（始终用 Close）→ `TestVM_Audit_IHighest_ModeSelectsCorrectField` RED（MODE_HIGH 结果错误）
3. 删 `extremeIndex` 越界 guard（只保留空序列检查）→ `TestVM_Audit_IHighest_OutOfRangeStart` RED（越界 start 原样返回）
4. 删 `iBarShift` exact 处理（始终 exact=false）→ `TestVM_Audit_IBarShift_ExactTrue` RED（非精确时间匹配了）

---

## 2. VM-RUNTIME-FAILCLOSED-1（S6-S10）

### S6：callBuiltin fatalError defense-in-depth

**目标**：`callBuiltin` 在 builtin handler 返回 nil error 后检查 `vm.fatalError`——若 handler 内部设置了 `fatalError`，`callBuiltin` 不返回 result，而是返回 NoneVal（让 runLoop 顶部检查捕获）。

**当前代码**（`vm_helpers.go:210-234`）：
```go
func (vm *VM) callBuiltin(builtinID int32, args []interp.Value) interp.Value {
    // ...
    if entry.fn != nil {
        result, err := entry.fn(vm, args)
        if err != nil {
            vm.recordBlindSpot(entry.name)
            return interp.NoneVal()
        }
        return result  // ❌ 缺少 fatalError 检查
    }
    // ...
}
```

**施工要求**：在 `return result` 前加 fatalError 检查：
```go
    if entry.fn != nil {
        result, err := entry.fn(vm, args)
        if err != nil {
            vm.recordBlindSpot(entry.name)
            return interp.NoneVal()
        }
        // VM-RUNTIME-FAILCLOSED-1: defense-in-depth — handler may have set
        // fatalError via recordBlindSpot (e.g. iADX:MODE_PLUSDI). Don't push
        // the result to stack; runLoop top check will catch fatalError.
        if vm.fatalError != "" {
            return interp.NoneVal()
        }
        return result
    }
```

### S7：pop/popN setStackError

**目标**：栈下溢时设置 `vm.fatalError`（而非静默返回 NoneVal/截断）。

**当前代码**（`vm_helpers.go:29-54`）：
```go
func (vm *VM) pop() interp.Value {
    if len(vm.stack) == 0 {
        return interp.NoneVal()  // ❌ 无 setStackError
    }
    // ...
}

func (vm *VM) popN(n int) []interp.Value {
    if n > len(vm.stack) {
        n = len(vm.stack)  // ❌ 无 setStackError
    }
    // ...
}
```

**施工要求**：

① 新增 `setStackError` 方法（`vm_helpers.go` 顶部附近）：
```go
// setStackError records a stack underflow as a fatal error.
// VM-RUNTIME-FAILCLOSED-1: runLoop top check will catch and return error.
func (vm *VM) setStackError(msg string) {
    if vm.fatalError == "" {
        vm.fatalError = "stack error: " + msg
    }
}
```

② `pop` underflow 时调用：
```go
func (vm *VM) pop() interp.Value {
    if len(vm.stack) == 0 {
        vm.setStackError("pop from empty stack")
        return interp.NoneVal()
    }
    // ...
}
```

③ `popN` underflow 时调用：
```go
func (vm *VM) popN(n int) []interp.Value {
    if n > len(vm.stack) {
        vm.setStackError(fmt.Sprintf("popN(%d) from stack of %d", n, len(vm.stack)))
        n = len(vm.stack)
    }
    // ...
}
```

### S8：OP_CALL_BUILTIN push 后 fatalError 检查

**目标**：`OP_CALL_BUILTIN` push result 后检查 `fatalError`，形成三层 defense-in-depth。

**当前代码**（`vm_execute.go:116-120`）：
```go
case OP_CALL_BUILTIN:
    nArgs := int(ins.B)
    args := vm.popN(nArgs)
    result := vm.callBuiltin(ins.A, args)
    vm.push(result)  // ❌ 无 fatalError 检查
```

**施工要求**：
```go
case OP_CALL_BUILTIN:
    nArgs := int(ins.B)
    args := vm.popN(nArgs)
    result := vm.callBuiltin(ins.A, args)
    vm.push(result)
    // VM-RUNTIME-FAILCLOSED-1: defense-in-depth — check fatalError after
    // builtin call (callBuiltin may have set it via handler or setStackError).
    if vm.fatalError != "" {
        return fmt.Errorf("VM fatal: %s", vm.fatalError)
    }
```

### S9：Engine.Run fail-closed

**目标**：`Engine.Run` 策略事件 error 时 fail-closed return error（而非 stderr + continue）。

**当前代码**（`engine.go:107-111`）：
```go
sig, err := e.runStrategySignal(btCtx, bar)
if err != nil {
    fmt.Fprintf(os.Stderr, "backtest: OnBar error at bar %d: %v\n", i, err)
    continue  // ❌ 未 fail-closed
}
```

**施工要求**：
```go
sig, err := e.runStrategySignal(btCtx, bar)
if err != nil {
    // VM-RUNTIME-FAILCLOSED-1: fail-closed — stop backtest on strategy error.
    return nil, fmt.Errorf("backtest: strategy event failed at bar %d: %w", i, err)
}
```

**注意**：`fmt.Fprintf(os.Stderr, ...)` 可保留或删除——fail-closed return 已包含 error 信息。建议删除 stderr 写入（return error 已足够，调用方应处理 error）。

### S10：iADX 不支持 mode 设 fatalError（使 S6 对抗证明可触发）

**目标**：`iADX:MODE_PLUSDI`/`MODE_MINUSDI` 等不支持的 mode 设置 `vm.fatalError`（而非只 recordBlindSpot 返回零值），使 S6 的 callBuiltin fatalError 检查能被对抗证明触发。

**当前代码**（`vm_builtin_indicators.go:195-200`）：
```go
case 1: // MODE_PLUSDI (+DI line)
    vm.recordBlindSpot("iADX:MODE_PLUSDI")
    return interp.DecimalVal(decimal.Zero), nil
case 2: // MODE_MINUSDI (-DI line)
    vm.recordBlindSpot("iADX:MODE_MINUSDI")
    return interp.DecimalVal(decimal.Zero), nil
```

**施工要求**：
```go
case 1: // MODE_PLUSDI (+DI line)
    // VM-RUNTIME-FAILCLOSED-1: unsupported mode → fail-closed
    vm.fatalError = "iADX:MODE_PLUSDI not supported"
    vm.recordBlindSpot("iADX:MODE_PLUSDI")
    return interp.DecimalVal(decimal.Zero), nil
case 2: // MODE_MINUSDI (-DI line)
    vm.fatalError = "iADX:MODE_MINUSDI not supported"
    vm.recordBlindSpot("iADX:MODE_MINUSDI")
    return interp.DecimalVal(decimal.Zero), nil
```

**注意**：同样修改 `vm_builtin_indicators_ext.go:300-304` 的 `iADXWilder:MODE_PLUSDI`/`MODE_MINUSDI`。

### S11：对抗证明

4 项突变：

1. 删 `callBuiltin`+`OP_CALL_BUILTIN` 三处 `fatalError` 检查 → `TestVM_Audit_FatalBlindSpotFromHandlerNotPushedToStack` RED（iADX MODE_PLUSDI 后续指令仍执行）
2. 恢复 `pop` 为 silent underflow（不调 `setStackError`）→ `TestVM_Audit_StackUnderflowIsError` RED（栈下溢不报 error）
3. 恢复 `callBuiltin` 吞 builtin error（`return NoneVal(), nil` 不设 fatalError）→ `TestVM_Audit_BuiltinErrorStopsExecution` RED（OrderSend error 后续指令仍执行）
4. 恢复 `Engine.Run` 吞策略事件 error（`_ = err` + continue）→ `TestVM_Audit_BuiltinErrorPropagatesToEngine` RED（Engine.Run 返回成功）

---

## 3. 测试文件

### 新建测试文件

**文件**：`backend/tools/mql2go/vm_timeseries_failclosed_redo_test.go`

**测试函数**（VM-TIMESERIES-SEMANTICS-1，11 个行为测试）：
1. `TestVM_Audit_IHighest_AllSeriesModes` — 5 bars 值递增，6 种 mode 全部 iHighest=shift 0
2. `TestVM_Audit_ILowest_AllSeriesModes` — 同上 iLowest=shift 4
3. `TestVM_Audit_IHighest_ModeSelectsCorrectField` — 3 bars 高/收盘/低在不同位置，mode 影响结果
4. `TestVM_Audit_IHighest_PartialRange` — start=2 count=2 → shift 2（只扫子范围）
5. `TestVM_Audit_IHighest_EmptySeries` — 空序列 → -1
6. `TestVM_Audit_IHighest_OutOfRangeStart` — start=10 Len=3 → -1
7. `TestVM_Audit_IHighest_InvalidMode` — mode=99 → -1 + blind spot
8. `TestVM_Audit_IBarShift_ExactTrue` — 精确匹配=shift 2，非精确时间 → -1
9. `TestVM_Audit_IBarShift_ExactFalse` — 非精确时间 → shift 2（最近 bar），时间在所有 bars 前 → -1
10. `TestVM_Audit_CopyTime_SecondsConversion` — 3 bars → unix seconds 非 ms
11. `TestVM_Audit_CopyClose_Direction` — count=+3 chronological, count=-3 reverse

**测试函数**（VM-RUNTIME-FAILCLOSED-1，4 个行为测试）：
12. `TestVM_Audit_BuiltinErrorStopsExecution` — OrderSend volume=0 → error → 后续指令不执行
13. `TestVM_Audit_InvalidMutationDoesNotChangeCapital` — OrderSend volume=0 → balance 不变 + positions=0
14. `TestVM_Audit_FatalBlindSpotFromHandlerNotPushedToStack` — iADX MODE_PLUSDI → fatalError → 赋值未完成 + 后续不执行
15. `TestVM_Audit_BuiltinErrorPropagatesToEngine` — OrderSend cmd=99 → error 经 VM→Runner→Engine 传播

**测试约束**：
- 全部固定 epoch `time.Date(2024,1,1,...,time.UTC)`，禁 `time.Now()`
- VM-RUNTIME-FAILCLOSED-1 测试需要用 `backtest.Engine`（`TestVM_Audit_BuiltinErrorPropagatesToEngine`）或直接 `VM` + `SimBroker`

**REUSE**：
- `builtinCopyTime`/`builtinIHighest`/`builtinILowest`/`builtinIBarShift`/`copyBarData`/`resolveSeries` @ `vm_builtin_mql5_ts.go`
- `callBuiltin`/`pop`/`popN`/`recordBlindSpot` @ `vm_helpers.go`
- `runLoop` fatalError 检查 @ `vm_execute.go:13-16`
- `CompileMQL`/`NewVM` @ `interp_runner.go`/`vm.go`
- `backtest.New`/`Engine.Run` @ `engine.go`
- `sdk.BarsToSlice` @ `series.go`

---

## 4. 验收

### 机检五件套
```bash
cd backend && go build ./...
cd backend && go test ./tools/mql2go/... -count=1
cd backend && go test -race ./tools/mql2go/... -count=1  # ×3 次
cd backend && go vet ./...
cd backend && go run ./tools/check-file-lines --strict
git diff --check
```

### 对抗证明
- VM-TIMESERIES-SEMANTICS-1：4 项 RED→restore→GREEN
- VM-RUNTIME-FAILCLOSED-1：4 项 RED→restore→GREEN
- 总计 8 项

### 文件清单（预期改动）
| 文件 | 改动类型 |
|------|----------|
| `tools/mql2go/vm_builtin_mql5_ts.go` | S1 (CopyTime /1000) + S2 (extremeIndex/valueAt/validSeriesMode + iHighest/iLowest 重写) + S4 (iBarShift exact) |
| `tools/mql2go/vm_helpers.go` | S6 (callBuiltin fatalError 检查) + S7 (setStackError + pop/popN) |
| `tools/mql2go/vm_execute.go` | S8 (OP_CALL_BUILTIN push 后 fatalError 检查) |
| `tools/mql2go/vm_builtin_indicators.go` | S10 (iADX MODE_PLUSDI/MINUSDI 设 fatalError) |
| `tools/mql2go/vm_builtin_indicators_ext.go` | S10 (iADXWilder MODE_PLUSDI/MINUSDI 设 fatalError) |
| `strategy/backtest/engine.go` | S9 (Engine.Run fail-closed) |
| `tools/mql2go/vm_timeseries_failclosed_redo_test.go` | 新建（15 个测试） |

### REUSE/NEW 标记
- **REUSE**: `callBuiltin`/`pop`/`popN`/`recordBlindSpot`/`runLoop`/`copyBarData`/`resolveSeries`/`CompileMQL`/`NewVM`/`backtest.New`/`Engine.Run`
- **NEW**: `extremeIndex`/`valueAt`/`validSeriesMode` @ `vm_builtin_mql5_ts.go`；`setStackError` @ `vm_helpers.go`

---

## 5. 固定尾部

**勿部署，停手等 Devin CLI 复审。**
**禁 `--no-verify`。**
**收工更新 registry + handover + STATE.md。**
