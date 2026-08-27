# Spec：VM 管线全面审计修复（VM-AUDIT-2026-08-27）

> **状态**：🟦open（待施工 + Devin CLI 验收）
> **审计方**：Devin CLI（独立审计，2026-08-27）
> **基线**：HEAD `68f31692`（工作树干净）
> **registry 条目**：`VM-AUDIT-2026-08-27-1` ~ `VM-AUDIT-2026-08-27-8`（`docs/audits/tech-debt-registry.md`）

## 1. 问题陈述

Devin CLI 对 VM 管线（MQL→AST→Bytecode 执行引擎 + live/backtest 调度 + mutation coordinator + trade barrier + position cache）做全面审计，发现 5 个 BUG + 3 个架构问题。本 spec 是修复方案 SSOT。

审计范围（10 个组件、~5500 行）：
- VM 核心：`vm.go` / `vm_execute.go` / `vm_helpers.go`
- VM Runner（编译入口）：`interp_runner.go`
- 编译器：`compile.go`
- 交易 builtins：`vm_builtin_trade.go`
- Live session：`vm_live_session.go` / `vm_live_handlers.go` / `vm_live_dispatch.go`
- Live dispatch：`live_dispatch.go` / `live_runner.go` / `live_runner_events.go`
- Mutation coordinator：`mutation_coordinator.go` / `mutation_recovery.go`
- Trade barrier：`trade_barrier.go`
- Position cache：`position_cache.go`
- Backtest worker：`backtest_worker_vm.go`

## 2. 设计决策

### D1：分 3 批施工，按安全优先级

| 批次 | ID | 优先级 | 理由 |
|------|----|--------|------|
| 1 | VM-AUDIT-2026-08-27-1 (Python live SourceHash) + VM-AUDIT-2026-08-27-2 (fatalError 重置) | P1 | 缓存安全 + 可用性，直接影响生产 live |
| 2 | VM-AUDIT-2026-08-27-3 (stack depth) + VM-AUDIT-2026-08-27-4 (popN 检查) + VM-AUDIT-2026-08-27-5 (dispatch default) | P2/P3 | VM 鲁棒性，防御性 |
| 3 | VM-AUDIT-2026-08-27-6 (compileForLive helper) + VM-AUDIT-2026-08-27-7 (recovery ctx) + VM-AUDIT-2026-08-27-8 (PositionCache panic) | P2 | 架构改进，防止未来漂移 |

**理由**：批次 1 是当前生产 live 的真实风险（缓存污染 + 策略永久停止）；批次 2 是防御性加固；批次 3 是架构改进防止 BUG-1 类漂移再次发生。

### D2：不重构核心架构

本 spec 只修复审计发现的具体问题，不重构 VM 执行模型、barrier 状态机、mutation coordinator 协议。这些组件经多轮返工（VM-CACHE-INTEGRITY / VM-TRADE-CONTEXT / LIVE-ORDER-REENTRY-1 R4）已成熟。

### D3：每批独立验收

每批施工完成后 Devin CLI 独立复审（mutation RED→restore→GREEN + 门禁全绿 + check-lines 0 errors），通过后再派下一批。

## 3. 修复方案（按 ID）

### VM-AUDIT-2026-08-27-1：Python live 路径不验证 SourceHash（P1 缓存安全）

**根因**：`executePythonVMLive` 和 `NewPythonVMLiveSessionCached` 用 `CompileMQLFromBytecode` 直接加载缓存，跳过 SourceHash 验证。`CompilePythonCached` 已存在且测试覆盖，但 live 路径未使用它。

**影响**：Python 策略源码修改后，live 路径仍用旧 bytecode 执行新源码——缓存污染攻击面。违反 VM-CACHE-INTEGRITY-2 不变量。

**修复**：

- **S1**：`backend/internal/connect/strategy/vm_live_dispatch.go:46-79` `executePythonVMLive` — 把当前 `:54-66` 的手写缓存加载逻辑（`var strategy *mql2go.VMRunner` + `CompileMQLFromBytecode` + `CompilePython` fallback）替换为 `CompilePythonCached` 三值返回：
  ```go
  strategy, bcData, err := mql2go.CompilePythonCached(req.StrategyCode, cachedBytecode)
  if err != nil {
      return nil, fmt.Errorf("compile Python: %w", err)
  }
  ```
  同时把 `:68-77` 的 SaveBytecode 块从 `mql2go.MarshalBytecode(strategy.Bytecode())` 改为直接用 `bcData`（镜像 `executeVMLive:26-38` 的 MQL 路径模式：`if bcData != nil ... SaveBytecode(ctx, sid, bcData)`）。**注意**：`CompilePythonCached` 在 cache hit 时返回 `cachedBytecode`（输入），cold compile 时返回新 marshal 的 data——两种情况 `bcData` 都非 nil，SaveBytecode 写回是幂等无害的（与 MQL 路径行为一致）。

- **S2**：`backend/internal/connect/strategy/vm_live_session.go:66-79` `NewPythonVMLiveSessionCached` — 把当前 `:67-78` 的手写缓存加载逻辑替换为 `CompilePythonCached`，**丢弃 bytecode 返回值**（该函数返回 `(*VMLiveSession, error)` 二值，bytecode 持久化由调用方 `initVMSession` 在 `live_runner_events.go:130-138` 通过 `MarshalBytecode(vmSess.strategy.Bytecode())` 处理，不在本函数职责内）：
  ```go
  func NewPythonVMLiveSessionCached(source string, cachedBytecode []byte) (*VMLiveSession, error) {
      runner, _, err := mql2go.CompilePythonCached(source, cachedBytecode)
      if err != nil {
          return nil, fmt.Errorf("compile Python: %w", err)
      }
      runner.SetSignalMode(true)
      return &VMLiveSession{strategy: runner}, nil
  }
  ```

**对抗证明**：
- 突变 `CompilePythonCached` 的 SourceHash 检查（`r.Bytecode().SourceHash == hashSource(source)` 改为 `true`）→ 用 source A 编译、source B 调用 `CompilePythonCached(sourceB, bcDataA)` → 应返回 recompiled runner（SourceHash 不匹配）但突变后返回 cached runner（SourceHash 匹配假）→ 测试断言 runner 的 `Bytecode().SourceHash == hashSource(sourceB)` 失败（RED）→ 恢复 → GREEN。
- 新增测试 `TestExecutePythonVMLive_SourceHashVerification`（在 `vm_live_dispatch_test.go` 或新文件）：mock importedRepo 返回 source A 的 bytecode，调用 `executePythonVMLive` with source B → 验证重新编译（不是用缓存）。

**门禁**：`go build ./...` / `go test ./internal/connect/strategy -count=1` / `go test -race ./internal/connect/strategy -count=1` ×3 / `go vet` / `check-file-lines --strict` / `git diff --check`。

**REUSE**：`CompilePythonCached` REUSE `interp_runner.go:102`（已存在，VM-CACHE-INTEGRITY-2 已测试覆盖）。

---

### VM-AUDIT-2026-08-27-2：runEvent 不重置 fatalError（P1 可用性）

**根因**：`vm.go:187-217` `runEvent` 重置 stack/caches/callDepth/signal/pc/lastIndicators/ticks，但不重置 `vm.fatalError`。`runLoop` 在 `vm_execute.go:14-16` 顶部检查 `if vm.fatalError != "" { return error }`——所以一旦事件 N 设置 fatalError，事件 N+1 的 `runLoop` 立即返回错误不执行任何指令。VMLiveSession 路径复用同一 VM 实例，一次 builtin 错误（如 broker 临时超时通过 `setStackError` 或 `callBuiltin` 设置 fatalError）后，后续所有事件立即返回错误不执行策略。

**影响**：生产 live 策略永久停止，即使 broker 恢复也不自愈——只能重建 session。`executeVMLive` 路径不受影响（每次新 VM）。**注意**：`setStackError`（`vm_helpers.go:31-34`）也写 `fatalError`，所以栈下溢同样触发此问题——只需重置 `fatalError` 一个字段即可覆盖两种来源。

**修复**：
- **S1**：`backend/tools/mql2go/vm.go:189` `runEvent` 的 reset 块内（`vm.stack = vm.stack[:0]` 同处），加 `vm.fatalError = ""`。精确插入点：在 `vm.signal = nil`（`:192`）之后、`vm.pc = entryPC`（`:193`）之前，或在 reset 块末尾 `vm.ticks = 0`（`:207`）之后——位置不影响正确性，只需在 `vm.runLoop(ctx)` 调用（`:216`）之前。

**对抗证明**：
- 新增测试 `TestVM_FatalErrorResetBetweenEvents`（`vm_test.go` 或新文件）：
  1. 编译一个调用会触发 fatalError 的 builtin 的 MQL EA（如 `iADX` with `MODE_PLUSDI` → `vm_builtin_indicators.go:197` 设置 `fatalError = "iADX:MODE_PLUSDI not supported"`）→ 第一次 `RunOnBar` 返回 error（fatalError 设置）。
  2. 突变：删除 `vm.fatalError = ""` 行 → 第二次 `RunOnBar`（即使不调用 iADX）仍立即返回 error（`runLoop` 顶部 `:14-16` 检查 fatalError 非空 → 直接返回）→ RED。
  3. 恢复 → 第二次 `RunOnBar` 正常执行（fatalError 已重置）→ GREEN。
- **注意**：不能用 "broker 返回 error" 作为 fatalError 触发条件——live VM 的 trade builtins 在 signal mode 下只记录信号不调 broker，broker error 不经过 `callBuiltin` → 不设 fatalError。必须用 builtin Go error 路径（`vm_helpers.go:232`）或 `setStackError` 路径（`vm_helpers.go:33`）触发。

**门禁**：同 VM-AUDIT-2026-08-27-1 + `go test ./tools/mql2go/... -count=1` ×3 race。

**REUSE**：`runEvent` 已有的重置模式（stack/caches 等），新增一行同模式。

---

### VM-AUDIT-2026-08-27-3：executeCallUser 缺少 MaxStackDepth 检查（P2 安全）

**根因**：`vm_execute.go:332-358` `executeCallUser` 的内联循环检查 ticks/context/MaxTicks，但不检查 `len(vm.stack) > MaxStackDepth`。外层 `runLoop` 在 `:33-36` 检查 MaxStackDepth，但用户函数内的长循环不回到外层 `runLoop`，栈可以增长到 MaxTicks（10M）个 entry 才被 MaxTicks 停止——此时已消耗 ~80-160MB 内存。

**影响**：恶意/有 bug 的 EA 可通过用户函数内的大量 push 操作绕过栈深度限制（4096），导致栈增长到 MaxTicks（10M）才停止，产生 ~80-160MB 内存尖峰。不是无界 OOM（MaxTicks 仍兜底），但远超 MaxStackDepth=4096 的设计意图。

**修复**：
- **S1**：`vm_execute.go` `executeCallUser` 内联循环，在 `vm.ticks > MaxTicks` 检查（`:348-352`）之后、`vm.execute(ins2)`（`:353`）之前加：
  ```go
  if len(vm.stack) > MaxStackDepth {
      vm.locals = oldLocals
      vm.callDepth--
      return fmt.Errorf("strategy exceeded max stack depth (%d)", len(vm.stack))
  }
  ```
  **注意**：必须恢复 `vm.locals = oldLocals` + `vm.callDepth--`（与其他错误退出路径一致：`:349-351` MaxTicks、`:354-356` execute error）。

**对抗证明**：
- 新增测试 `TestVM_CallUserStackDepthLimit`：**直接构造 bytecode**（不走 MQL 编译器——编译器会平衡 push/pop，正常 MQL 无法触发此场景）——构造一个 user function（有 `Funcs` entry + `OP_CALL_USER` 调用），函数体内是一个循环不断 `OP_PUSH_CONST` 但不 pop，超过 MaxStackDepth=4096 → 调用 → 应返回 "strategy exceeded max stack depth" error。
- 突变：删除新增的 stack 检查 → 测试 RED（栈增长到 MaxTicks=10M 才停止，错误信息是 "strategy exceeded instruction limit" 而非 "max stack depth"）→ 恢复 → GREEN。
- **注意**：这是 defense-in-depth 测试——正常 MQL 编译器生成的 bytecode push/pop 平衡，不会触发此路径。此测试针对编译器 bug 或恶意 bytecode 注入场景。

**门禁**：同上。

**REUSE**：`MaxStackDepth` 常量 REUSE `vm.go:20`。错误信息格式 REUSE `vm_execute.go:35`。

---

### VM-AUDIT-2026-08-27-4：popN 栈不足时 callBuiltin 仍执行（P2 语义）

**根因**：`vm_helpers.go:54-63` `popN` 在 `n > len(vm.stack)` 时设置 fatalError 但返回部分结果。`vm_execute.go:116-125` `OP_CALL_BUILTIN` 调用 `popN` 后直接 `callBuiltin(args)`——虽然 fatalError 会在循环顶部捕获，但当前 builtin 仍会用错误参数执行。

**影响**：栈下溢时 builtin 用部分/空参数执行，可能产生副作用（如 OrderSend 用空 symbol/volume 调用 broker）。

**修复**：
- **S1**：`vm_execute.go:117-120` `OP_CALL_BUILTIN` case，在 `args := vm.popN(nArgs)`（`:118`）之后、`result := vm.callBuiltin(ins.A, args)`（`:119`）之前加 fatalError 检查，**提前 return 跳过 callBuiltin**：
  ```go
  case OP_CALL_BUILTIN:
      nArgs := int(ins.B)
      args := vm.popN(nArgs)
      if vm.fatalError != "" {          // NEW: popN 栈下溢 → 不执行 builtin
          return fmt.Errorf("VM fatal: %s", vm.fatalError)
      }
      result := vm.callBuiltin(ins.A, args)
      vm.push(result)
      if vm.fatalError != "" {          // EXISTING: callBuiltin 内部错误（:123-125 保留）
          return fmt.Errorf("VM fatal: %s", vm.fatalError)
      }
  ```
  **注意**：`:123-125` 已有的 fatalError 检查（callBuiltin 返回后）保留不变。新增的检查是 popN 后、callBuiltin 前的 early return，防止栈下溢时 builtin 用部分参数执行产生副作用。

**对抗证明**：
- 新增测试 `TestVM_PopNStackUnderflowStopsBuiltin`：**直接构造 bytecode**（不走 MQL 编译器——编译器会保证参数数量正确）——构造一条 `OP_CALL_BUILTIN` 指令，`ins.B=3`（需要 3 参数），但栈上只有 1 个值 → 调用 → 验证 builtin handler 不被调用（用 mock builtin 注册一个计数器）+ 返回 error 含 "stack error"。
- 突变：删除新增的 fatalError 检查（popN 后的 early return）→ builtin 被调用（计数器增加）→ RED → 恢复 → GREEN。
- **注意**：这是 defense-in-depth 测试——正常 MQL 编译器生成的 bytecode 参数数量与 builtin 签名匹配，不会触发此路径。此测试针对编译器 bug 或恶意 bytecode 注入场景。

**门禁**：同上。

**REUSE**：fatalError 检查模式 REUSE `vm_execute.go:123-125`（OP_CALL_BUILTIN 末尾已有的检查）。

---

### VM-AUDIT-2026-08-27-5：VMLiveSession.dispatch default 误处理（P3 语义）

**根因**：`vm_live_session.go:164-171` default 分支：`if bctx := req.GetBarContext(); bctx != nil` 则当 bar 处理。未知请求类型 + 恰好有 bar_context（proto 字段在未知 enum 下仍可能携带旧数据）→ 误当 bar 事件执行策略。当前 `else` 分支已返回 error，但 `if bctx != nil` 分支是问题所在。

**修复**：
- **S1**：`vm_live_session.go:164-171` default 分支——删除 `if bctx != nil` 条件分支，整个 default 直接返回 error：
  ```go
  default:
      return &antv1.ExecuteLiveResponse{Success: false, Error: fmt.Sprintf("unknown request type: %s", req.GetRequestType())}
  ```
  **注意**：需要 `import "fmt"`（已存在于文件中，因为 `Start` 函数用了 `fmt.Errorf`）。

**对抗证明**：
- 新增测试 `TestVMLiveSession_UnknownRequestType`：构造 `ExecuteLiveRequest` with unknown RequestType（如 `REQUEST_TYPE_UNSPECIFIED`）+ non-nil BarContext → 验证返回 `Success: false` + Error 含 "unknown request type"（不是执行 bar 事件返回 success）。
- 突变：恢复旧 default 分支（`if bctx != nil { resp = vmHandleBar(...) }`）→ 测试 RED（返回 `Success: true` 而非 `false`）→ 恢复 → GREEN。

**门禁**：同上。

**REUSE**：错误响应格式 REUSE `vm_live_session.go:169`。

---

### VM-AUDIT-2026-08-27-6：提取 compileForLive helper 统一缓存逻辑（P2 架构）

**根因**：两条 live 路径（RPC 单次 `executeVMLive`/`executePythonVMLive` + Long-running `VMLiveSession`）各自实现缓存加载逻辑，导致 BUG-1（Python 路径漏验证 SourceHash）。**现状**：`executeVMLive`（MQL 路径，`:26`）已正确使用 `CompileMQLCached`；`NewVMLiveSessionCached`（MQL session，`:46-61`）也已正确使用 `CompileMQLCached`。只有 Python 两个路径（`executePythonVMLive` + `NewPythonVMLiveSessionCached`）手写缓存逻辑——VM-AUDIT-2026-08-27-1 修复后这两处也改用 `CompilePythonCached`。本 ID 是在 -1 修复基础上提取共享 helper，防止未来新增 live 路径再次漏验证。

**修复**：
- **S1**：在 `vm_live_dispatch.go` 或新文件 `vm_live_compile.go` 提取 helper：
  ```go
  // compileForLive compiles source for live execution with bytecode cache.
  // isPython selects the Python vs MQL compiler front-end.
  // Returns (runner, bytecodeData, error). bytecodeData is non-nil on both
  // cache hit (returns cachedBytecode input) and cold compile (fresh marshal).
  func compileForLive(source string, cachedBytecode []byte, isPython bool) (*mql2go.VMRunner, []byte, error) {
      if isPython {
          return mql2go.CompilePythonCached(source, cachedBytecode)
      }
      return mql2go.CompileMQLCached(source, cachedBytecode)
  }
  ```
- **S2**：4 个调用点全部改用 `compileForLive`（替换 `CompileMQLCached`/`CompilePythonCached` 调用）：
  - `executeVMLive`（`vm_live_dispatch.go:26`）：`CompileMQLCached(...)` → `compileForLive(req.StrategyCode, cachedBytecode, false)`
  - `executePythonVMLive`（`vm_live_dispatch.go:54`）：手写缓存逻辑 → `compileForLive(req.StrategyCode, cachedBytecode, true)`（删除 `:54-66` 的 `var strategy` + `CompileMQLFromBytecode` + `CompilePython` fallback）
  - `NewVMLiveSessionCached`（`vm_live_session.go:45`）：`CompileMQLCached(...)` → `compileForLive(source, cachedBytecode, false)`
  - `NewPythonVMLiveSessionCached`（`vm_live_session.go:66-79`）：手写缓存逻辑 → `compileForLive(source, cachedBytecode, true)`（丢弃 bytecode 返回值）
  - **注意**：4 个调用点的 `GetBytecode` DB 查询（`importedRepo.GetBytecode`）保留不变——`compileForLive` 只替换编译入口，不替换缓存读取。只有 Python 两个路径的手写 `CompileMQLFromBytecode` + fallback 块被删除。

**对抗证明**：
- 突变 `compileForLive` 的 `isPython` 分支（Python 路径误调 `CompileMQLCached`）→ Python 策略 live 执行编译失败 → RED → 恢复 → GREEN。
- 验证所有 4 个调用点都用 `compileForLive`（grep 确认无手写 `CompileMQLFromBytecode` + fallback 模式残留）。

**门禁**：同上 + `grep -r "CompileMQLFromBytecode" backend/internal/connect/strategy/` 应返回 0 匹配（所有缓存加载都走 `compileForLive` → `CompileMQLCached`/`CompilePythonCached`，不再直接调 `CompileMQLFromBytecode`）。

**REUSE**：`CompileMQLCached` / `CompilePythonCached` REUSE `interp_runner.go:77/102`。

**注**：本 ID 与 VM-AUDIT-2026-08-27-1 有重叠——-1 修复后 BUG-1 已解决，本 ID 是架构加固防止未来漂移。建议在批次 1 完成 -1 后立即做 -6（趁热打铁），或留到批次 3。

---

### VM-AUDIT-2026-08-27-7：recoverFromOutcomeUnknown 用 time.Sleep 不可取消（P2 架构）

**根因**：`mutation_recovery.go:47` `time.Sleep(conf.recoveryDelay)`（默认 10s）不可取消。session 关闭后 goroutine 仍 sleep 完整 10s，延迟资源释放。

**Context 来源已确认**：`coordinateMutation` 的 `ctx` 参数（`mutation_coordinator.go:74-77`）来自 `live_runner.go:208` 的 `runCtx, runCancel := context.WithCancel(ctx)`，是 session 级别的 run context（不是 per-event）。`runCtx` 在 `live_runner.go:306/317/328` 传给 `handleBar/handleTick/handleTrade`，再传到 `dispatchCloseAll/submitOrder` → `coordinateMutation`。当 `SessionRegistry.Stop`（`:306`）调用 `sess.cancel()` 时，`runCtx` 被取消。因此直接传 `coordinateMutation` 的 `ctx` 给 `recoverFromOutcomeUnknown` 即可——不需要修改 `ActiveSession` 或 `LiveStrategyConfig`。

**修复**：
- **S1**：`mutation_recovery.go:41-47` 函数签名加 `ctx context.Context` 作为第一个参数：
  ```go
  func (s *StrategyExecutionServer) recoverFromOutcomeUnknown(
      ctx context.Context,
      cfg LiveStrategyConfig, activeSess *ActiveSession,
      barrier *TradeBarrier, ticket int64, action mutationAction,
      verify func(orders []*mthub.OrderRecord) bool,
      conf confirmationConfig,
  ) {
      select {
      case <-time.After(conf.recoveryDelay):
      case <-ctx.Done():
          return
      }
      // ... rest unchanged
  ```
- **S2**：`mutation_coordinator.go:200` 和 `:271` 两处 `go s.recoverFromOutcomeUnknown(...)` 调用，在最前面加 `ctx`：
  ```go
  go s.recoverFromOutcomeUnknown(ctx, cfg, activeSess, barrier, spec.expectedTicket, spec.action, verify, conf)
  ```
  ```go
  go s.recoverFromOutcomeUnknown(ctx, cfg, activeSess, barrier, effectiveTicket, spec.action, verify, conf)
  ```

**对抗证明**：
- 新增测试 `TestRecoverFromOutcomeUnknown_CancelledByContext`：启动 recovery goroutine with `ctx, cancel := context.WithCancel(context.Background())`，立即 `cancel()` → 验证 goroutine 在 <100ms 内退出（而非等 10s）。可用 atomic counter 或 done channel 验证 goroutine 退出。
- 突变：恢复 `time.Sleep(conf.recoveryDelay)` → 测试 RED（goroutine 等 10s 才退出，测试超时或 counter 不在 100ms 内增加）→ 恢复 → GREEN。

**门禁**：同上。

**REUSE**：`select + ctx.Done()` 模式 REUSE `vm_execute.go:19-24`（runLoop 的 ctx 检查）和 `position_cache.go:57-59`（Subscribe 的 ctx.Done）。

---

### VM-AUDIT-2026-08-27-8：PositionCache.Subscribe goroutine 无 panic recovery（P2 架构）

**根因**：`position_cache.go:54-67` goroutine 无 `defer recover()`。如果 `c.put` panic（snap 字段 nil 等），整个进程崩溃。

**`c.log` 已确认存在**：`PositionCache` 结构体（`:21-34`）有 `log *zap.Logger` 字段（`:32`），`NewPositionCache`（`:36-44`）初始化为 `zap.NewNop()` if nil。`c.log.Error(...)` 调用安全。`accountID` 是 `Subscribe` 函数参数（`:49`），在 goroutine 闭包中捕获，可用。

**修复**：
- **S1**：`position_cache.go:54` goroutine 开头（`defer unsub()` 之后）加 panic recovery：
  ```go
  go func() {
      defer unsub()
      defer func() {
          if r := recover(); r != nil {
              c.log.Error("PositionCache: subscribe goroutine panicked",
                  zap.String("account", accountID),
                  zap.Any("panic", r))
          }
      }()
      for {
          // ... rest unchanged
      }
  }()
  ```
  **注意**：`defer` 顺序——`defer unsub()` 先声明（后执行），`defer recover()` 后声明（先执行）。这样 panic 先被 recover，然后 unsub 执行。如果反过来，panic 时 unsub 先执行（取消订阅），recover 后 goroutine 退出——也可以，但 recover 先更安全（确保 panic 被捕获后再清理）。需要 `import "go.uber.org/zap"`（已存在于文件中，`:106-107` 已用 `zap.String`）。

**对抗证明**：
- 新增测试 `TestPositionCache_SubscribePanicRecovery`：mock hub 发送一个会让 `put` panic 的 snapshot（如构造一个触发 nil deref 的场景——可以在 `put` 中加一个 test-only hook，或 mock hub 发送一个特殊 snap 触发 `c.put` 内的 nil deref）→ 验证 goroutine 不崩溃（log 记录 panic）+ 后续正常 snapshot 仍被处理。
- 突变：删除 `defer recover()` → 测试 RED（goroutine panic 导致 test runner 检测到 goroutine panic，或后续 snapshot 不被处理）→ 恢复 → GREEN。

**门禁**：同上。

**REUSE**：panic recovery 模式 REUSE `interp_runner.go:126-130`（`CompilePython` 的 `defer func() { if r := recover()... }`）。

## 4. 验收标准

每批施工完成后，Devin CLI 独立复审：
1. **mutation RED→restore→GREEN**：每个 ID 的 S1 必须有真实对抗证明
2. **门禁全绿**：`go build ./...` / `go test ./tools/mql2go/... -count=1` / `go test -race ./tools/mql2go/... -count=1` ×3 / `go test ./internal/connect/strategy -count=1` / `go test -race ./internal/connect/strategy -count=1` ×3 / `go vet ./...` / `check-file-lines --strict`（0 errors）/ `git diff --check`
3. **file-lines**：新增文件不超限（300/450/800 红线）
4. **复用核对**：`bash scripts/cap.sh` 多关键词查重
5. **状态诚实**：施工方不得自标 ✅done，只标 `🟦open（施工完成，待独立复审）`

## 5. 不做

- 不重构 VM 执行模型、barrier 状态机、mutation coordinator 协议（D2）
- 不部署（D-COMMIT-SCOPE-001 部署闸仍有效）
- 不改写历史审计事实
- 不批量验收（D3：每批独立验收）

## 6. 风险评估

| ID | 风险 | 缓解 |
|----|------|------|
| 1 | Python live 路径改动可能影响现有 Python 策略 | `CompilePythonCached` 已有测试覆盖，改动是调用入口替换不改逻辑 |
| 2 | fatalError 重置可能掩盖真实 bug（错误被吞） | fatalError 每次事件返回错误给调用方，调用方已有 RecordError + 日志；重置只是允许下一次事件重新尝试 |
| 3 | stack depth 检查可能误判正常深度递归 | MaxStackDepth=4096 远超正常 EA 需求；MaxCallDepth=256 已限制递归深度 |
| 7 | recovery ctx 传递可能需要改动 LiveStrategyConfig | 已确认不需要——`coordinateMutation` 的 `ctx` 参数就是 session 级 `runCtx`，直接传入即可 |
