# 施工提示词：VM-AUDIT-2026-08-27 批次 3（P2 架构加固）

## 立项背景
- 触发：VM 管线审计批次 1/2 已验收通过（`f6f83191`，✅done）；本批为架构加固，3 个 P2 架构问题。
- 证据链：spec `docs/spec/vm-audit-2026-08-27-spec.md` §3「-6」「-7」「-8」；registry `VM-AUDIT-2026-08-27-6/7/8`。
- BUG-6：两条 live 路径缓存逻辑不一致导致 BUG-1 漂移；提取 `compileForLive` helper 统一。
- BUG-7：`recoverFromOutcomeUnknown` 用 `time.Sleep` 不可取消；改用 `select + ctx.Done()`。
- BUG-8：`PositionCache.Subscribe` goroutine 无 panic recovery；加 `defer recover()`。

## 设计 SSOT 声明
- 唯一真相源：`docs/spec/vm-audit-2026-08-27-spec.md` §3 对应章节。本提示词与 spec 冲突时以 spec 为准。

## 约束与目标
- 每步必须有真实 mutation RED→restore→GREEN 对抗证明（nil panic / 另一条错误 / callback-only / "任意 error" 均不算证据）。
- REUSE 优先：`CompileMQLCached`/`CompilePythonCached`（`interp_runner.go:77/102`）、`select + ctx.Done()` 模式（`vm_execute.go:19-24`）、panic recovery 模式（`interp_runner.go:126-130`）。

## 边界 / 不做
- 不重构 VM 执行模型、不部署、不 push、不改 git config、禁 `--no-verify`。
- 不动批次 1/2 已验收的代码。
- BUG-6 与 BUG-1 有重叠：-1 修复后 BUG-1 已解决，-6 是架构加固防止未来漂移。

## 正文

### S1：提取 compileForLive helper（BUG-6）
- 目标：统一 4 个 live 路径的缓存加载逻辑，防止未来新增路径漏验证 SourceHash。
- 坐标：新建 `backend/internal/connect/strategy/vm_live_compile.go` 或在 `vm_live_dispatch.go` 内提取。
- 落点：新增 helper：
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
  4 个调用点全部改用 `compileForLive`：
  - `executeVMLive`（`vm_live_dispatch.go:26`）：`CompileMQLCached(...)` → `compileForLive(req.StrategyCode, cachedBytecode, false)`
  - `executePythonVMLive`（`vm_live_dispatch.go:54`）：`CompilePythonCached(...)` → `compileForLive(req.StrategyCode, cachedBytecode, true)`
  - `NewVMLiveSessionCached`（`vm_live_session.go:45`）：`CompileMQLCached(...)` → `compileForLive(source, cachedBytecode, false)`
  - `NewPythonVMLiveSessionCached`（`vm_live_session.go:66`）：`CompilePythonCached(...)` → `compileForLive(source, cachedBytecode, true)`（丢弃 bytecode 返回值）
  注意：4 个调用点的 `GetBytecode` DB 查询（`importedRepo.GetBytecode`）保留不变——`compileForLive` 只替换编译入口，不替换缓存读取。

### S2：recoverFromOutcomeUnknown 改用可取消的 select（BUG-7）
- 目标：session 关闭后 recovery goroutine 立即退出，不浪费 10s。
- 坐标：`backend/internal/connect/strategy/mutation_recovery.go:41-47`。
- 落点：函数签名加 `ctx context.Context` 作为第一个参数：
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
  `mutation_coordinator.go:200` 和 `:271` 两处 `go s.recoverFromOutcomeUnknown(...)` 调用，在最前面加 `ctx`：
  ```go
  go s.recoverFromOutcomeUnknown(ctx, cfg, activeSess, barrier, spec.expectedTicket, spec.action, verify, conf)
  ```
  ```go
  go s.recoverFromOutcomeUnknown(ctx, cfg, activeSess, barrier, effectiveTicket, spec.action, verify, conf)
  ```
  Context 来源已确认：`coordinateMutation` 的 `ctx` 参数（`mutation_coordinator.go:74-77`）来自 `live_runner.go:208` 的 `runCtx`，是 session 级别的 run context。`SessionRegistry.Stop`（`:306`）调用 `sess.cancel()` 时 `runCtx` 被取消。不需要修改 `ActiveSession` 或 `LiveStrategyConfig`。

### S3：PositionCache.Subscribe goroutine 加 panic recovery（BUG-8）
- 目标：防止 `c.put` panic 导致整个进程崩溃。
- 坐标：`backend/internal/connect/strategy/position_cache.go:54-67`。
- 落点：goroutine 开头（`defer unsub()` 之后）加 panic recovery：
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
  注意：`defer` 顺序——`defer unsub()` 先声明（后执行），`defer recover()` 后声明（先执行）。`c.log`（`position_cache.go:32`）+ `accountID`（函数参数 `:49`）+ `zap` import（`:106` 已用）均可用。

### T1：对抗测试 TestCompileForLive_PythonBranch（BUG-6）
- 突变 `compileForLive` 的 `isPython` 分支（Python 路径误调 `CompileMQLCached`）→ Python 策略 live 执行编译失败 → RED → 恢复 → GREEN。
- 验证所有 4 个调用点都用 `compileForLive`（grep 确认无手写 `CompileMQLFromBytecode` + fallback 模式残留）。

### T2：对抗测试 TestRecoverFromOutcomeUnknown_CancelledByContext（BUG-7）
- 启动 recovery goroutine with `ctx, cancel := context.WithCancel(context.Background())`，立即 `cancel()` → 验证 goroutine 在 <100ms 内退出（而非等 10s）。可用 atomic counter 或 done channel 验证 goroutine 退出。
- 突变：恢复 `time.Sleep(conf.recoveryDelay)` → 测试 RED（goroutine 等 10s 才退出，测试超时或 counter 不在 100ms 内增加）→ 恢复 → GREEN。

### T3：对抗测试 TestPositionCache_SubscribePanicRecovery（BUG-8）
- mock hub 发送一个会让 `put` panic 的 snapshot（构造一个触发 nil deref 的场景——可以在 `put` 中加一个 test-only hook，或 mock hub 发送一个特殊 snap 触发 `c.put` 内的 nil deref）→ 验证 goroutine 不崩溃（log 记录 panic）+ 后续正常 snapshot 仍被处理。
- 突变：删除 `defer recover()` → 测试 RED（goroutine panic 导致 test runner 检测到 goroutine panic，或后续 snapshot 不被处理）→ 恢复 → GREEN。

## 验收标准
1. 先红后绿：T1/T2/T3 必须先 RED（突变）再 GREEN（恢复），证据留测试文件内。
2. 机检五件套：`go build ./...` / `gofmt -l .` / `go vet ./...` / `go test ./tools/mql2go/... -count=1` / `go test ./internal/connect/strategy -count=1`。
3. race×3：`go test -race ./tools/mql2go/... -count=1` ×3 + `go test -race ./internal/connect/strategy -count=1` ×3。
4. check-file-lines：`cd backend && go run ./tools/check-file-lines --strict`（0 errors）。
5. `git diff --check` clean。
6. BUG-6 门禁追加：`grep -r "CompileMQLFromBytecode" backend/internal/connect/strategy/` 应返回 0 匹配（所有缓存加载都走 `compileForLive`）。
7. 收工：更新 registry 三条目（-6/-7/-8）状态为 `🟦open（施工完成，待独立复审）` + STATE.md 施工表；不得自标 ✅done。

## 尾部
勿部署，停手等 Devin CLI 复审。禁 `--no-verify`。
