# 施工提示词：VM-AUDIT-2026-08-27 批次 1（P1 缓存安全 + 可用性）

## 立项背景
- 触发：Devin CLI 2026-08-27 对 VM 管线全面审计，发现 2 个 P1 级生产风险。
- 证据链：审计 spec `docs/spec/vm-audit-2026-08-27-spec.md`（SSOT）；registry 条目 `VM-AUDIT-2026-08-27-1`、`VM-AUDIT-2026-08-27-2`（`docs/audits/tech-debt-registry.md`）。
- BUG-1：Python live 路径（`executePythonVMLive` / `NewPythonVMLiveSessionCached`）用 `CompileMQLFromBytecode` 直接加载缓存，跳过 SourceHash 验证 → 源码改后仍跑旧 bytecode。
- BUG-2：`runEvent` 不重置 `vm.fatalError` → 一次 builtin 错误后策略永久停止，不自愈。

## 设计 SSOT 声明
- 唯一真相源：`docs/spec/vm-audit-2026-08-27-spec.md` §3「VM-AUDIT-2026-08-27-1」「VM-AUDIT-2026-08-27-2」。本提示词与 spec 冲突时以 spec 为准。

## 约束与目标
- 只改 spec 指定的代码坐标，不扩大 diff。
- 每步必须有真实 mutation RED→restore→GREEN 对抗证明（nil panic / 另一条错误 / callback-only / "任意 error" 均不算证据）。
- REUSE 优先：`CompilePythonCached`（`interp_runner.go:102`）已存在且有测试覆盖，本任务只替换调用入口，不新建编译逻辑。

## 边界 / 不做
- 不重构 VM 执行模型、不部署、不 push、不改 git config、禁 `--no-verify`。
- 不动 MQL 路径（`executeVMLive` / `NewVMLiveSessionCached` 已正确使用 `CompileMQLCached`，本批不改）。
- 不做批次 2/3 的任何内容（-3/-4/-5/-6/-7/-8）。

## 正文

### S1：executePythonVMLive 改用 CompilePythonCached（BUG-1）
- 目标：消除 Python RPC live 路径的缓存污染。
- 坐标：`backend/internal/connect/strategy/vm_live_dispatch.go:46-79` `executePythonVMLive`。
- 落点：把 `:54-66` 的手写缓存加载（`var strategy *mql2go.VMRunner` + `CompileMQLFromBytecode` + `CompilePython` fallback）替换为：
  ```go
  strategy, bcData, err := mql2go.CompilePythonCached(req.StrategyCode, cachedBytecode)
  if err != nil { return nil, fmt.Errorf("compile Python: %w", err) }
  ```
  把 `:68-77` 的 SaveBytecode 块从 `mql2go.MarshalBytecode(strategy.Bytecode())` 改为直接用 `bcData`，镜像 MQL 路径 `:32` 的 `if bcData != nil ... SaveBytecode(ctx, sid, bcData)` 模式。注意：cache hit 时 `bcData` = 输入 `cachedBytecode`，cold compile 时 = 新 marshal data，两种都非 nil，SaveBytecode 幂等无害（与 MQL 路径行为一致）。

### S2：NewPythonVMLiveSessionCached 改用 CompilePythonCached（BUG-1）
- 目标：消除 Python long-running session 路径的缓存污染。
- 坐标：`backend/internal/connect/strategy/vm_live_session.go:66-79` `NewPythonVMLiveSessionCached`。
- 落点：把 `:67-78` 手写缓存逻辑替换为：
  ```go
  runner, _, err := mql2go.CompilePythonCached(source, cachedBytecode)
  if err != nil { return nil, fmt.Errorf("compile Python: %w", err) }
  runner.SetSignalMode(true)
  return &VMLiveSession{strategy: runner}, nil
  ```
  丢弃 bytecode 返回值——持久化由调用方 `initVMSession`（`live_runner_events.go:130-138`）通过 `MarshalBytecode(vmSess.strategy.Bytecode())` 处理，不在本函数职责内。

### S3：runEvent 重置 fatalError（BUG-2）
- 目标：消除 live session 单次错误后策略永久停止。
- 坐标：`backend/tools/mql2go/vm.go:187-217` `runEvent`。
- 落点：在 reset 块内加一行 `vm.fatalError = ""`。精确插入点：`vm.signal = nil`（`:192`）之后、`vm.pc = entryPC`（`:193`）之前，或 reset 块末尾 `vm.ticks = 0`（`:207`）之后——位置不影响正确性，只需在 `vm.runLoop(ctx)`（`:216`）之前。理由：`runLoop`（`vm_execute.go:14-16`）顶部检查 `fatalError != ""` 直接返回 error；`setStackError`（`vm_helpers.go:31-34`）和 `callBuiltin`（`vm_helpers.go:232`）都写 `fatalError`，重置这一个字段覆盖两种来源。

### T1：对抗测试 TestExecutePythonVMLive_SourceHashVerification（BUG-1）
- 用 source A 编译得到 bytecode，mock importedRepo 返回 A 的 bytecode，调用 `executePythonVMLive` with source B → 断言重新编译（runner 的 `Bytecode().SourceHash == hashSource(sourceB)`，不是 A 的）。
- 突变：把 `CompilePythonCached`（`interp_runner.go:105`）的 `r.Bytecode().SourceHash == hashSource(source)` 改为 `true` → 测试 RED（返回 cached runner，SourceHash 是 A 的）→ 恢复 → GREEN。

### T2：对抗测试 TestVM_FatalErrorResetBetweenEvents（BUG-2）
- 编译一个调用 `iADX` with `MODE_PLUSDI` 的 MQL EA（触发 `vm_builtin_indicators.go:197` 设置 `fatalError = "iADX:MODE_PLUSDI not supported"`）。
- 第一次 `RunOnBar` 返回 error（fatalError 设置）。
- 突变：删除 S3 新增的 `vm.fatalError = ""` 行 → 第二次 `RunOnBar`（不调用 iADX）仍立即返回 error（`runLoop` 顶部 `:14-16` fatalError 非空直接返回）→ RED。
- 恢复 → 第二次 `RunOnBar` 正常执行 → GREEN。
- 注意：不能用 "broker 返回 error" 触发——live VM trade builtins 在 signal mode 只记录信号不调 broker，broker error 不经过 `callBuiltin` → 不设 fatalError。必须用 builtin Go error 路径（`vm_helpers.go:232`）或 `setStackError` 路径（`vm_helpers.go:33`）触发。

## 验收标准
1. 先红后绿：T1/T2 必须先 RED（突变）再 GREEN（恢复），证据留测试文件内。
2. 机检五件套：`go build ./...` / `gofmt -l .` / `go vet ./...` / `go test ./tools/mql2go/... -count=1` / `go test ./internal/connect/strategy -count=1`。
3. race×3：`go test -race ./tools/mql2go/... -count=1` ×3 + `go test -race ./internal/connect/strategy -count=1` ×3。
4. check-file-lines：`cd backend && go run ./tools/check-file-lines --strict`（0 errors）。
5. `git diff --check` clean。
6. 收工：更新 registry 两条目（-1/-2）状态为 `🟦open（施工完成，待独立复审）` + STATE.md 施工表；不得自标 ✅done。

## 尾部
勿部署，停手等 Devin CLI 复审。禁 `--no-verify`。
