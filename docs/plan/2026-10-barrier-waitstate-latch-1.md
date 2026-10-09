# BARRIER-WAITSTATE-LATCH-1：TradeBarrier.WaitState 锁存化——消灭竞态测试类（Devin CLI 设计 SSOT 2026-10-09）

## 立项背景

`TestSubmitOrder_CommentAndDeviationReachExecutor`（`vm_live_parity_test.go:83`）T1/T2 两轮复审各败一次，均判负载型抖动。根因：`WaitState`（`trade_barrier.go:392`）是非锁存**现态**等待——waiter goroutine 必须与同步 submit 路径抢 µs 级窗口，错过 `barrierSubmitting` 瞬态即等到 ctx 超时（2s），测试成败取决于调度运气。同类 8 个调用点（全在 `_test.go`）。修法=锁存化：记录"曾到达"而非只盯"当前"。

## 设计 SSOT

- `TradeBarrier` 加 `visited uint32` 位图字段（state 是 iota 0-5，位图够）。
- 每处 `b.state = X` 赋值点（现 10 处：trade_barrier.go :168 :200 :205 :237 :269 :285 :338 :340 :357 :371）同时置位——做法：统一改 `b.setStateLocked(X)`，新 helper 内 `b.state = s; b.visited |= 1 << uint32(s)`。cond.Broadcast 各点原样不动。
- `Acquire`（:168 所在函数）置位前先 `b.visited = 0` 清零——visited 语义=**本生命周期**曾到达，跨 mutation 不残留。
- 构造初始态即 `barrierIdle`——`NewTradeBarrier` 内 `visited = 1 << uint32(barrierIdle)`。
- `WaitState` 新语义：`visited&(1<<target) != 0`（含当前态）→ 立即返回 `target`；否则 `cond.Wait()` 循环；ctx 到期返回当前 `b.state`。**返回值=target 表示"曾到达"**——兼容全部既有断言：
  - `vm_live_parity:100` / `adapterlabel:237` / `barrier_test:394` / `reentry:202`：错过 submitting 瞬态不再致命，latch 命中即返
  - 负向等待（`reentry:74`、`recovery_test:252` 等 WaitState(idle) 断言"不提前释放"）：buggy 早释放→idle 置位→latch 命中返回 idle→断言红；无释放→超时返回非 idle→断言绿——语义保持
- `1 << uint32(s)` 移位安全（state 值域 0-5）。

## 施工步骤

### S1 `backend/internal/connect/strategy/trade_barrier.go`

- struct 加 `visited uint32` 字段（注释：per-Acquire 生命周期已访状态位图，供 WaitState 锁存判定）。
- `NewTradeBarrier` 置 `visited = 1 << uint32(barrierIdle)`。
- 加 `setStateLocked(s tradeBarrierState)` helper；10 处 `b.state = X` 全改 `b.setStateLocked(X)`；Acquire 处在置位前 `b.visited = 0`。
- `WaitState` 循环体改：`if b.visited&(1<<uint32(target)) != 0 { b.mu.Unlock(); return target }` + `if ctx.Err() != nil { s := b.state; b.mu.Unlock(); return s }` + `b.cond.Wait()`；doc comment 更新为新语义。

### S2 回归+对抗验证

- `go test -count=5 ./internal/connect/strategy` 全绿（连跑 5 遍验 flake 消失）。
- `go test -race -count=1 ./internal/connect/strategy` 绿。
- **mutation 对抗（先红后绿）**：删 S1 中任一 `visited` 置位（如 `setStateLocked` 里去掉 `|=` 行）→ `TestSubmitOrder_CommentAndDeviationReachExecutor` 须在高负载/多次重跑下复现 RED（`-count=20` 提高命中率），恢复后 GREEN。
- 负向语义守护：`live_order_reentry_r4_redo_test.go` / `mutation_coordinator_recovery_test.go` 全绿（latch 不得让负向等待假过）。

## 验收门禁

- build/vet/gofmt/diff-check 净；check-file-lines 0 ERROR（trade_barrier.go 448 行现贴 300 预警线——新增 ~10 行后若破 450 须顺手报实际行数，不破则不动）
- `go test -count=5` + `-race` 全绿；mutation RED→GREEN 证据贴回报
- 全仓 `WaitState` 调用点行为清单核对（8 处）无遗漏

勿部署，停手等 Devin CLI 复审。
