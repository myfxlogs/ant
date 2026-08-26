# Builder Handoff — LIVE-ORDER-REENTRY-1 R4 返工（复审退回项）

> 日期：2026-08-26 ｜ 审计方：Devin CLI（项目第一负责人） ｜ 施工方：Devin IDE
> 设计 SSOT：`docs/spec/live-order-reentry-r4-spec.md`（不变）
> 原施工提示词：`docs/audits/builder-handoff-live-order-reentry-r4-2026-08-26.md`
> 复审退回记录：本文件 §0 证据链

## 0. 立项背景

**触发**：LIVE-ORDER-REENTRY-1 R4 复审 conditional pass，2 项退回。

**证据链**：
- 复审 C 洁净：`live_order_reentry_r4_redo_test.go` 有 2 处 `time.Sleep`（:79 轮询 10ms、:199 轮询 1ms），违反确定性测试纪律。原 6 处 `time.Sleep` 已全部清除，但新增测试引入了新的 `time.Sleep`。
- 复审 D 正确性 S3：`FullBrokerPath` 测试的 `WaitState` 无法通过 `time.Sleep(0)` 突变证明必要性（`dispatchLiveSignal` 同步阻塞模型使 `WaitState` vs `time.Sleep(0)` 差异不明显）。`Recovery_CloseConfirmed` 的对抗证明成功。

**设计 SSOT 声明**：`docs/spec/live-order-reentry-r4-spec.md`（D1-D3 不变）

**约束与目标**：
- 只修复 2 项退回，不重做已通过的 S1/S2
- `live_order_reentry_r4_redo_test.go` 的 2 处 `time.Sleep` 改为 `WaitState` 同步
- `FullBrokerPath` 测试的 `WaitState` 必须可通过对抗证明（或重构测试使 `WaitState` 成为必需）

**边界/不做**：
- 不改已通过的 S1（`mutation_coordinator.go:259-271`）和 S2（adapter wrapper + RealParse 测试）
- 不改 `trade_barrier.go` 的 `WaitState` 方法（已通过复审）
- 不改 `mutation_coordinator.go` 生产代码
- 不部署（D-COMMIT-SCOPE-001 部署闸仍有效）
- 不 commit/push/deploy
- 禁 `--no-verify`

---

## S1 — 删除 live_order_reentry_r4_redo_test.go 的 2 处 time.Sleep

**目标**：2 处 `time.Sleep` 改为 `WaitState` 确定性同步。

**精确坐标**：
- 文件：`backend/internal/connect/strategy/live_order_reentry_r4_redo_test.go`

### S1a — :79 的轮询循环（TestLIVE_ORDER_REENTRY_1_R4_OpenMutationWithTicket_NoRecovery）

**当前代码**（:63-80）：
```go
deadline := time.After(700 * time.Millisecond)
for {
    select {
    case <-deadline:
        if state := sess.barrier.State(); state != barrierOutcomeUnknown {
            t.Fatalf("OpenMutationWithTicket: post-wait state=%s, want outcome_unknown (no recovery for open)", state)
        }
        if !sess.IsCircuitOpen() {
            t.Fatal("OpenMutationWithTicket: circuit breaker should stay open (no recovery)")
        }
        return
    default:
    }
    if sess.barrier.State() != barrierOutcomeUnknown {
        t.Fatalf("OpenMutationWithTicket: recovery ran unexpectedly, state=%s (should stay outcome_unknown for open)", sess.barrier.State())
    }
    time.Sleep(10 * time.Millisecond)
}
```

**问题**：`time.Sleep(10ms)` 轮询 barrier 状态，违反确定性纪律。

**落点**：这个测试的语义是"在 700ms 内 barrier 必须保持 outcomeUnknown，不得被 recovery 释放"。改为：
```go
// 等待 recoveryDelay + readAfterWriteTimeout + 余量，确认 barrier 保持 outcomeUnknown
noRecoveryCtx, noRecoveryCancel := context.WithTimeout(context.Background(), 700*time.Millisecond)
finalState := sess.barrier.WaitState(noRecoveryCtx, barrierIdle) // 只有 recovery 释放 barrier 才会返回 barrierIdle
noRecoveryCancel()
if finalState != barrierOutcomeUnknown {
    t.Fatalf("OpenMutationWithTicket: recovery ran unexpectedly, state=%s (should stay outcome_unknown for open)", finalState)
}
if !sess.IsCircuitOpen() {
    t.Fatal("OpenMutationWithTicket: circuit breaker should stay open (no recovery)")
}
```

**原理**：`WaitState(ctx, barrierIdle)` 会阻塞直到 barrier 变为 `barrierIdle`（recovery 释放）或 ctx 超时。如果 recovery 不启动（正确行为），ctx 超时后返回当前状态 `barrierOutcomeUnknown`。如果 recovery 错误启动，barrier 会变为 `barrierIdle`，`WaitState` 提前返回，测试断言失败。

### S1b — :199 的轮询循环（TestLIVE_ORDER_REENTRY_1_R4_AdapterLabelPipeline_RealParse_FullPath_MT4）

**当前代码**（:194-201）：
```go
go func() {
    for i := 0; i < 1000; i++ {
        if sess.barrier.State() == barrierSubmitting {
            break
        }
        time.Sleep(time.Millisecond)
    }
    publishOrderUpdate(broker, cfg.AccountID, 42, strategyMagic(cfg.ScheduleID), update.UpdateType)
}()
```

**问题**：`time.Sleep(1ms)` 轮询 barrier 是否进入 `barrierSubmitting`，违反确定性纪律。

**落点**：改为 `WaitState`：
```go
go func() {
    waitCtx, waitCancel := context.WithTimeout(context.Background(), 2*time.Second)
    sess.barrier.WaitState(waitCtx, barrierSubmitting)
    waitCancel()
    publishOrderUpdate(broker, cfg.AccountID, 42, strategyMagic(cfg.ScheduleID), update.UpdateType)
}()
```

**对抗证明（S1a）**：
- 突变 `mutation_coordinator.go:259` 的 `if spec.action != actionOpen` → `if true`（恢复旧逻辑）→ recovery 启动 → barrier 变为 `barrierIdle` → `WaitState` 提前返回 → `finalState == barrierIdle` → 测试断言 `finalState != barrierOutcomeUnknown` 失败 → RED
- 恢复 → GREEN

**对抗证明（S1b）**：
- 突变 `WaitState` 为 `time.Sleep(0)` → barrier 还未进入 `barrierSubmitting` → `publishOrderUpdate` 在错误时机发布 → barrier 可能未确认 → 测试可能 RED（但依赖时序，不保证确定性）
- 更可靠的对抗：突变 `WaitState` 为立即返回 `barrierIdle`（跳过等待）→ `publishOrderUpdate` 在 barrier 进入 submitting 前发布 → broker 订阅者未就绪 → 测试 RED 或 flaky

---

## S2 — FullBrokerPath 测试的 WaitState 对抗证明

**目标**：使 `FullBrokerPath` 测试的 `WaitState` 可通过对抗证明，或重构测试使 `WaitState` 成为必需。

**问题分析**：
- `mutation_coordinator_test.go:1226` 的 `FullBrokerPath` 测试用 `WaitState(ctx, barrierSubmitting)` 等待 barrier 进入 submitting
- 但 `dispatchLiveSignal` 是同步阻塞调用，主 goroutine 在 `dispatchLiveSignal` 内部阻塞等待 barrier 状态变化
- goroutine 中的 `WaitState` 和主 goroutine 的 `dispatchLiveSignal` 并行执行，`time.Sleep(0)` 后 goroutine 立即 publish，此时 barrier 可能已在 submitting（因为 `dispatchLiveSignal` 已开始执行）
- 因此 `WaitState` vs `time.Sleep(0)` 差异不明显

**精确坐标**：
- 文件：`backend/internal/connect/strategy/mutation_coordinator_test.go:1226-1230`（FullBrokerPath 的 WaitState）
- 文件：`backend/internal/connect/strategy/live_order_reentry_r4_redo_test.go:194-201`（RealParse_FullPath_MT4 的轮询，S1b 已处理）

**落点**：
方案——保留 `WaitState`，但在测试注释中明确说明其作用是"防御性同步"而非"必需同步"，并补充一个对抗证明说明：

在 `mutation_coordinator_test.go` 的 `FullBrokerPath` 测试注释中加：
```go
// WaitState here is defensive synchronization: it ensures the goroutine
// publishes after the barrier enters submitting. In the current synchronous
// dispatchLiveSignal model, time.Sleep(0) would also work because the main
// goroutine blocks inside dispatchLiveSignal. However, WaitState is correct
// regardless of dispatch model (sync or async), making the test robust to
// future refactors. The adversarial proof for WaitState necessity is in
// TestLIVE_ORDER_REENTRY_1_R4_Recovery_CloseConfirmed (recovery goroutine
// is truly async — WaitState is required there).
```

**或者**重构测试使 `WaitState` 成为必需——但这需要改 `dispatchLiveSignal` 的调用方式（如改为异步），超出返工 scope。

**推荐方案**：保留 `WaitState` + 加注释说明 + 引用 `Recovery_CloseConfirmed` 的对抗证明。不重构测试（避免 scope 扩大）。

**对抗证明**：
- `Recovery_CloseConfirmed` 的 `WaitState` → `time.Sleep(0)` 突变已验证 RED（recovery goroutine 异步，0ms 时未完成）→ 证明 `WaitState` 对异步场景必需
- `FullBrokerPath` 的 `WaitState` 是防御性同步，对抗证明引用 `Recovery_CloseConfirmed`

---

## 验收标准

1. **S1a 对抗证明**：突变 `if spec.action != actionOpen` → `if true` → `TestLIVE_ORDER_REENTRY_1_R4_OpenMutationWithTicket_NoRecovery` RED → restore GREEN
2. **S1b**：`live_order_reentry_r4_redo_test.go` 的 :199 轮询改为 `WaitState`
3. **S2**：`FullBrokerPath` 测试注释说明 `WaitState` 的防御性角色 + 引用 `Recovery_CloseConfirmed` 对抗证明
4. **`grep "time.Sleep" live_order_reentry_r4_redo_test.go`** 返回 0 行
5. **门禁全绿**：
   - `go build ./...`
   - `go test ./internal/connect/strategy -count=1`
   - `go test -race ./internal/connect/strategy -count=1` ×3
   - `go vet ./internal/connect/strategy/...`
   - `go run ./tools/check-file-lines --strict`（0 errors）
   - `git diff --check`
6. **不破坏已通过的 S1/S2 测试**：原有 R4 测试仍全 GREEN

## 红队自审（施工方完工前必答）

1. S1a 的 `WaitState(ctx, barrierIdle)` 在 recovery 不启动时是否正确超时返回 `barrierOutcomeUnknown`？
2. S1b 的 `WaitState(ctx, barrierSubmitting)` 在 `dispatchLiveSignal` 阻塞时是否会死锁？（不会——`dispatchLiveSignal` 内部会改变 barrier 状态，`WaitState` 的 `cond.Wait()` 会被唤醒）
3. `FullBrokerPath` 注释是否明确说明 `WaitState` 是防御性同步？
4. `grep "time.Sleep" live_order_reentry_r4_redo_test.go` 是否返回 0 行？

## 回填纪律

1. registry `LIVE-ORDER-REENTRY-1-R4-REVIEW`（:71）：更新返工结果
2. `handover-audit-plan.md` 变更日志加一行
3. **不自行宣告完成**——停手等 Devin CLI 复审

## 范围约束

One task = one scope：只动 `backend/internal/connect/strategy/live_order_reentry_r4_redo_test.go`（S1a/S1b 删除 time.Sleep）+ `backend/internal/connect/strategy/mutation_coordinator_test.go`（S2 注释）。不改生产代码、不改已通过的 S1/S2 实现。

## 固定尾部

**勿部署，停手等 Devin CLI 复审。** 禁 `--no-verify`。禁 commit/push/deploy。只 add 本任务文件，禁 `git add -A`。
