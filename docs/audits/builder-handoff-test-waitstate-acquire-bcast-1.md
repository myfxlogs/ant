# 施工派工单：TEST-WAITSTATE-ACQUIRE-BCAST-1

> **状态**：设计实查完成（Devin CLI，2026-09-18），待施工 → 独立复审
> **registry**：`docs/audits/tech-debt-registry.md` 行 211（QS-2.4-F1）
> **前置**：VM-FUNC-FATAL-DELAY-1 ✅done（`de6f672c+6ef18536`）

---

## 1. 缺陷实证（Devin CLI 独立核实）

`internal/connect/strategy/trade_barrier.go` 状态迁移点全扫——`b.state =` 共 9 处，**仅 `Acquire:168`（idle→submitting）缺 `cond.Broadcast()`**，其余 :199/:204/:236/:268/:284/:337/:356/:370 全带广播（含 `Release` idle→:377）。condvar 契约断裂不对称。

**后果链**：`WaitState(ctx, barrierSubmitting)`（:391-413）若在 Acquire 前进入 `cond.Wait()`（:411）→ Acquire 迁移无广播 → 睡到下一次广播（:205 acceptedUnconfirmed）→ 此刻 `state != submitting` 永不匹配 → 睡到 ctx 超时返回当前 state。

**现存调用点两站均是此破形态**：
- `live_order_reentry_r4_redo_test.go:191-199`：goroutine `WaitState(waitCtx, barrierSubmitting)` 欲在 submitting 窗口注入 order update——实际睡到 ctx 超时或下迁移才醒，注入点漂移（事件迟到落 acceptedUnconfirmed 直配路径，**非其意图的 cache 路径**）；`:195` 注释"会被唤醒"与实现不符（误导）。
- `mutation_coordinator_test.go:1237`：同 `WaitState(waitCtx, barrierSubmitting)` defensive sync，忽略返回值。

**判别难点（自审计）**：返回值**不可判别**——无广播时 waiter 睡到 ctx 超时返回的恰是 `barrierSubmitting`（届时 state 正是它）。唯一判别量=**延迟**（有广播 ~µs 醒；无广播睡到 ctx 超时）。

## 2. 设计裁定

**D1 修复**：`Acquire` `b.state = barrierSubmitting`（:168）后加 `b.cond.Broadcast()`（锁内，同其余迁移点形态）+ 注释引本债 ID。副作用评估：非 submitting waiter 多醒一次重检后再睡，无害；submitting waiter 提前醒至**意图窗口**——r4 测试事件注入点修正为 submitting 窗口（cache 路径，与其注释意图一致）。

**D2 注释修正**：`r4_redo_test.go:195` "会被唤醒"注释在修复后才成立——措辞对齐（"Acquire 迁移带 Broadcast，cond.Wait() 会被唤醒"）。

**D3 边界**：`WaitState` 等值语义不改（瞬态 state 竞态为 condvar 固有，ctx 兜底）；`Acquire` 非 idle 早退路径（:166）不广播（无迁移）；其余 8 迁移点已广播不动。

**D4 依赖面**：两调用点语义修正（提前醒至意图窗口），均忽略返回值——`mutation_coordinator_test.go:1237`/`r4_redo_test.go:197` 需全量回归确认无隐性时序依赖。

## 3. 施工步骤

### S1：trade_barrier.go Acquire 加广播

`:168` `b.state = barrierSubmitting` 后加 `b.cond.Broadcast()`（在 `return true` 前、锁内）+ 行注释 `// TEST-WAITSTATE-ACQUIRE-BCAST-1: all state transitions must broadcast`。

### S2：r4 测试注释修正

`live_order_reentry_r4_redo_test.go:195` 措辞对齐 D2。

### S3：新测试（追加 `trade_barrier_test.go`——已核实存在，Acquire 用例先例 :26/:40/`b.Acquire("client-1", 12345, "open")` 签名序一致）

```go
// TestTradeBarrier_AcquireBroadcastsSubmitting：
// ctx, _ := context.WithTimeout(bg, 5*time.Second)
// done := make(chan tradeBarrierState, 1)
// go func() { done <- b.WaitState(ctx, barrierSubmitting) }()
// time.Sleep(50*time.Millisecond)  // 保证 waiter 已进入 cond.Wait（宽限防 flake）
// if !b.Acquire("c1", 1, "open") { t.Fatal("Acquire should succeed on idle barrier") }
// select {
// case s := <-done:
//     if s != barrierSubmitting { t.Fatalf("got %s want submitting", s) }
// case <-time.After(2 * time.Second):
//     t.Fatal("WaitState did not wake within 2s — Acquire broadcast missing")
// }
// 判别：无广播时 waiter 睡到 5s ctx 超时 > 2s done 超时 → RED（边距 3s 防 flake）。
```

### S4：mutation 证据

1. 删 `b.cond.Broadcast()` → S3 用例 RED（done 超时——waiter 睡到 ctx）
2. Broadcast 放锁外（`defer` 后）→ 语义等价（waiter 重检）——不可观测变异，注明豁免或跳过；改测 `WaitState(confirmed)` 早醒无害用例覆盖。

## 4. 验收门

```bash
cd backend && go build ./...
go test ./internal/connect/strategy/
go test -race -count=3 ./internal/connect/strategy/
go vet ./internal/connect/strategy/
gofmt -l internal/connect/strategy/
go run ./tools/check-file-lines --strict
git diff --check
```

## 5. 文件/规模预估

| 文件 | 改动 |
|---|---|
| `internal/connect/strategy/trade_barrier.go` | +2 行（广播+注释） |
| `internal/connect/strategy/live_order_reentry_r4_redo_test.go` | 注释 ~1 行 |
| `internal/connect/strategy/<test>.go` | 新用例 ~25 行 |

`trade_barrier.go` 447→~449 行——check-lines 关注（当前 0 errors；若本文件已在警告级，+2 不致 🔴，但如实报输出级别）。

## 6. 回报格式

`[施工完成:TEST-WAITSTATE-ACQUIRE-BCAST-1] @<commit-hash>` + 机检五件套 + mutation 证据。勿部署、勿 push、禁 `--no-verify`。
