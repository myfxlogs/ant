# builder-handoff-qs-2.5 — panic recovery 加固（进程存活 + fail-closed）

> v1 @2026-09-16（随 QS-2.4 验收后发单）

## 立项背景

VM 管线质量方案 v2 阶段 2 第三单（spec §4.3，registry QS-2.5）。**实拍根因已核实**：`dispatchLiveSignal`（`live_dispatch.go:30`）由 VM 事件循环同步调用（`live_context.go:196`），该路径**全链零 recover**——仓内 5 处 recover（backtest_worker:148/position_cache:60/session_registry:373/shadow_verifier:108/experiment_worker:74）均不在此路径。Go 语义：任一 goroutine 未捕获 panic = 整个进程崩溃 → 所有策略会话全灭 + barrier 停在中途态无记录。

## 设计 SSOT

- spec §4.3 + 本文件坐标（D-013 已核实）。
- FAILCLOSED-1（D-009）：recover 不是"吞掉错误继续跑"，是把不确定结果收敛到 `outcomeUnknown` 锁仓 + circuit open。

## 约束与目标

1. 只改 `backend/internal/connect/strategy/` 两函数 + 测试文件；不动 `trade_barrier.go`、不动其他 dispatcher 业务逻辑。
2. 串行，勿部署、勿 push、禁 `--no-verify`；commit 用 `ANT_ROLE=builder git commit` 前缀。
3. 完工按 D-012 自审 + D-014 末行 `[施工完成:QS-2.5] @<commit-hash>` 自报，停手等复审。

## S1 — `coordinateMutation` recover（`mutation_coordinator.go:74`）

**结构**：函数签名改命名返回 `) (result mutationResult)`（源码兼容，既有 `return mutationResult{...}` 不动）。函数首部（`barrier := activeSess.barrier` 之前/紧随）注册 defer recover——**必须最先注册**，使其在 unwind 中最后执行（:116-119 的 confirmUnsub 清理先跑完再进 recover 收敛）。

**两枚标志**：
- `acquired bool`——`barrier.Acquire` :86 成功后置 true。
- `brokerCalled bool`——`:130` `spec.brokerCall(brokerCtx)` 调用**紧前**置 true（以"是否已发出 RPC"为界，spec 原文要求）。

**recover 收敛逻辑**：

```
panic 时：
  log.Error("coordinateMutation: panic recovered — fail-closed", panic值, account, action, broker_called)
  acquired == false          → result = {state: barrierIdle}（barrier 未持有，只记录）
  acquired && !brokerCalled  → barrier.NotifyDeterministicRejected() + barrier.Release()
                              + RecordError → result = {state: barrierDeterministicRejected}
                              （未发 RPC = 提前判定，无 outcome 不确定性）
  acquired && brokerCalled   → barrier.NotifyOutcomeUnknown() + activeSess.SetCircuitOpen(true)
                              + RecordError → result = {state: barrierOutcomeUnknown}
                              （RPC 可能已发 = 结果不可知 → 锁仓 fail-closed）
```

## S2 — `dispatchLiveSignal` recover（`live_dispatch.go:30`）

函数首部注册 defer recover。**收敛规则（关键安全约束，已核实语义）**：

```
panic 时：
  log.Error("dispatchLiveSignal: panic recovered", panic值, account, signal_type)
  activeSess != nil → RecordError + SetCircuitOpen(true)
  barrier 处理必须按 State() 门控——NotifyOutcomeUnknown 实测语义：
    终态(confirmed/deterministicRejected) → 内部 no-op（:278-280 保护）
    outcomeUnknown → no-op（:281-283）
    **idle/submitting/acceptedUnconfirmed → 迁移 outcomeUnknown**
  ⇒ 仅当 barrier.State() ∈ {submitting, acceptedUnconfirmed}（在途）才调 NotifyOutcomeUnknown；
    idle 时**禁止**调用——否则把健康 barrier 误锁，后续 mutation 全部被 Acquire 拒绝（fail-open 反向）。
```

理由：S2 落地后 coordinateMutation 内部 panic 已被自身 recover 收敛——到达 dispatchLiveSignal recover 的 panic 发生在 coordinator 之外（persistSignal/日志/sub-dispatcher 框架码），此时 barrier 在途只可能来自异常路径，锁仓仍是保守正确；idle 则绝不能碰。

## S3 — 行为测试（先红后绿）

新文件 `mutation_panic_test.go`（或就近既有测试文件，按行数红线裁量）：

- **S3a 后置 RPC panic**：`mutationSpec.brokerCall` 注入 `func(ctx) (int64, error) { panic("broker boom") }` → `coordinateMutation` 返回 `{state: barrierOutcomeUnknown}` + `barrier.State()==barrierOutcomeUnknown` + `IsCircuitOpen()==true` + RecordError 有记录。
- **S3b 前置 panic**：`s.mtHub = nil` 触发 `:106 SubscribePositionSnapshots` nil 解引用 panic（或施工方找到更稳的注入点，须说明）→ 返回 `deterministicRejected` + barrier 回 idle（Release 生效）+ circuit **不**开。
- **S3c 未 Acquire panic**：构造使 `Acquire` 前/中 panic 的注入（如 nil cfg 字段路径，若不可行则以 Acquire 失败早退路径代替并注明覆盖等价性）。
- **S3d dispatchLiveSignal idle 守卫**：barrier idle + 注入 dispatcher 外 panic → recover 后 `barrier.State()==barrierIdle`（未被误锁）+ circuit open + 进程未崩。
- **S3e dispatchLiveSignal 在途守卫（可选加分）**：预置 barrier submitting 态 + panic → `outcomeUnknown` 锁仓。
- panic 注入点须使 panic 真实穿过目标函数帧——`panic("x")` 直接写在 brokerCall 闭包内即满足 S3a；S3d 注入点自选但须证明 panic 发生在 `dispatchLiveSignal` 帧内（如 `persistSignal` 依赖项置 nil，或 sub-dispatcher 的 nil 解引用），并在自报中给出注入坐标。

## 验收标准

- [ ] `go build ./...`、`go test -count=1 ./internal/connect/strategy/` 全绿
- [ ] `go test -race -count=3 ./internal/connect/strategy/` 通过（goleak 门禁须仍绿）
- [ ] `go run ./tools/check-file-lines --strict` 本任务文件零新增警告
- [ ] gofmt/go vet 零新增
- [ ] mutation×3：①删 recover → S3a panic 逃逸测试崩 RED；②recover 中删 `NotifyOutcomeUnknown` → barrier 停 submitting → S3a 断言 RED；③删 S2 的 State() 门控（改无条件 NotifyOutcomeUnknown）→ S3d RED；各 restore→GREEN
- [ ] `git diff` 无交接层文件、无 spec 外改动

## 边界/不做

- 不给其他 recover 点动手术；不改 `NotifyOutcomeUnknown` 内部语义；不动 FAILCLOSED-1 的任何正常路径行为；不处理 `WaitConfirmed` 内部 panic（cond/mu 内部不会 panic，越界）。
- panic 后**不重新抛出**（进程存活是本单目标；panic 信息入日志 + RecordError 留痕即可）。

## D-013 出件自审记录（决策方）

- 坐标回验：`dispatchLiveSignal` :30-93 已读；`coordinateMutation` :74-189+ 已读；`mutationResult` :65-68；调用链 `live_context.go:196` 同步调用已核实；recover 现存 5 处均不在此路径（grep 全仓）。
- 路径可达性：`s.mtHub = nil` 注入点可达 `:106`（`SubscribePositionSnapshots` 为接口/指针方法调用，nil receiver panic 前置条件成立——施工方若实测该调用对 nil 容忍须换注入点并报备）。
- 生命周期/时序：defer LIFO 顺序已推演（recover 先注册后执行，confirmUnsub 清理先行）；`NotifyOutcomeUnknown` 迁移语义逐行核实（:278 终态保护 + :281 幂等 + idle→outcomeUnknown 会迁移——S2 门控必要性由此推出，**本单关键安全约束**）。
- 冲突检查：与 QS-1.7-INV"永久锁仓"裁定一致（panic 锁仓同一 fail-closed 语义）；与 QS-2.2 watcher 有界性无冲突。
- 复用核对：`RecordError/SetCircuitOpen/NotifyOutcomeUnknown/NotifyDeterministicRejected/Release` 全为既有 API，零新建。
- **署名**：最终决策：Devin CLI（[角色:决策终] 激活）
