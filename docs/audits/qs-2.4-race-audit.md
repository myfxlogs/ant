# QS-2.4 VM 管线 data race 审计报告

> 施工产物（只审不改，零生产/测试代码改动）。handoff：`docs/audits/builder-handoff-qs-2.4.md`。
> 审计日期：2026-09-16；基线 commit：QS-2.2 `174b8405` 之后（goleak 门禁已绿）。

## 1. race×3 三树证据（原样输出摘录 + 结论）

实跑命令（`cd backend`，`-race` 真实启用，非 `-count=1` 冒充）：

```
$ go test -race -count=3 ./tools/mql2go/...
ok  	alphaforge/tools/mql2go	49.935s
ok  	alphaforge/tools/mql2go/interp	1.057s

$ go test -race -count=3 ./internal/connect/strategy/...
ok  	alphaforge/internal/connect/strategy	301.289s

$ go test -race -count=3 ./strategy/...
ok  	alphaforge/strategy/backtest	8.828s
ok  	alphaforge/strategy/indicators	27.683s
ok  	alphaforge/strategy/runner	1.078s
ok  	alphaforge/strategy/sdk	1.049s
```

- **`WARNING: DATA RACE` 出现次数：0**（三树全量输出 grep 确认）。
- 结论：三树 race×3 全绿。
- 覆盖核实：`./strategy/` 顶层包无 `_test.go`，4 个子包共 29 个测试文件（backtest 21 / runner 5 / indicators 2 / sdk 1）——race 覆盖真实非空转；顶层 `strategy` 包本身无测试编译进 run（覆盖空洞，见 §5）。
- 与 QS-2.2 关系：goleak 证"无 goroutine 泄漏"，race 证"已执行交错下无数据竞争"——两者正交。本审计=**未发现 race**，不等于证明无 race（检测器只覆盖被测试驱动的交错）。

## 2. PositionCache 审计表（`position_cache.go` 239 行）

字段：`mu`/`snapshots`/`financialsReceivedAt`/`positionsReceivedAt`（全非导出）；`log` 构造后不可变。字段外部零触碰已复核（仓内 grep 仅 `position_cache.go` 命中，`vm_audit_2026_08_27_batch3_test.go:150` 仅为注释引用）。

| 审计点 | 结论 | 证据 |
|--------|------|------|
| a) `GetFresh*` 在 RUnlock 后读 `snap.*` 是否安全 | **安全**：retained 快照对象发布后永不被原地改。`put` 两条路径均以拷贝存新指针（positions-only：`merged := *current` :94 + slice 深拷 :95-96 → 存 `&merged` :109；financials：`merged := *snap` :120 + slice 深拷 :124-125 → 存 `&merged` :134）；`Unsubscribe` 只删 map 键不碰对象（:149-151）。因此 RUnlock 后持有的 `snap` 指针指向的对象无写者。 | :85-139, :149-151, :165-178/:184-202/:208-231 |
| b) 返回 `*PositionSnapshot` 裸指针——调用方是否原地写 | **全部 6 个调用方只读，无原地写**：①`active_session_proto.go:94` `GetSnapshot` → `enrichFromPositionCache` :112-153 只读（len/range/IsZero/UnixMilli/Sub）；②同文件 :158/:164 `GetFresh*` 仅取 bool 丢弃指针；③`live_context.go:82-132` 读字段构造新 proto 对象；④`strategy_schedule_positions.go:41-64` 读 `snap.Positions` 构造新 proto 列表；⑤`account_provider.go:55-83` 读字段构造新 `risk.AccountState`（`snap.Equity` 比较峰值，不写回）。 | 见左各 file:line |
| c) `Subscribe` goroutine 共享访问 | **无遗漏**：`c.put` 内部自锁（:85）；recover 存在（:59-65）；`defer unsub()` :55；退出路径 ctx.Done :68 / ch 关闭 :71；goroutine 只触碰 `c.put`/`c.log`（不可变）。生命周期=ctx。 | :49-78 |

## 3. TradeBarrier 审计表（`trade_barrier.go` 447 行）

配对图：`cond.Broadcast()`×10（:200/:205/:237/:269/:285/:303/:341/:357/:377/:397）vs `cond.Wait()`×2（:320 `WaitConfirmed`、:411 `WaitState`）。

| 审计点 | 结论 | 证据 |
|--------|------|------|
| a) 每处 Broadcast 是否在 `b.mu` 持锁内 | **10/10 全部持锁**：`NotifyBrokerAccepted` :186 Lock+defer → :200/:205；`NotifyConfirmationEvent` :222 → :237；`NotifyDeterministicRejected` :263 → :269；`NotifyOutcomeUnknown` :276 → :285；`WaitConfirmed` watcher goroutine :302-304 `Lock→Broadcast→Unlock`；`Reconcile` :331 → :341；`ConfirmByAuthoritativeRead` :350 → :357；`Release` :369-378 手动 `Lock…Broadcast:377…Unlock:378`；`WaitState` watcher :396-398。 | 见左各行号 |
| b) 两处 `cond.Wait()` 持锁 for-loop + 醒后重检 | **正确范式**：`WaitConfirmed` :291 Lock → for {:310 isTerminal 检查 → :315 ctx.Err 检查 → :320 Wait}；`WaitState` :404 Lock → for {:406 target/ctx 检查 → :411 Wait}。醒后均重检条件。 | :288-322, :391-413 |
| c) `NotifyOutcomeUnknown` 幂等分支不 Broadcast 是否饿死等待方 | **不饿死**：①首次迁移一定走 :284-285 Broadcast——已在 `cond.Wait` 的等待方被唤醒；②其后到达的等待方在 `WaitConfirmed` :292/:310 或 `WaitState` :406 入口处即见终态（outcomeUnknown ∈ isTerminal :60）直接返回，无需广播。幂等 early-return（:281-283）只在已迁终态后发生，此时不存在"错过了唯一一次广播"的等待方。 | :275-286, :59-61, :292/:310, :406 |
| d) 持 `b.mu` 时是否调用再取 `b.mu` 的路径 | **无双重取锁**：所有 Broadcast 所在方法体内无外部回调、无对 `State()`/`Ticket()`/`Acquire()` 等再取 mu 方法的调用；`cacheEvent`（:246-257）在持锁内被调但自身不碰 `mu`；watcher goroutine 的 `Lock→Broadcast→Unlock`（:302-304/:396-398）为独立短暂临界区，与 cond.Wait 释放-重取语义兼容。 | :185-413 全段 |

## 4. 发现项清单

| # | 位置 | 证据 | 严重度建议 | 建议债务标题 |
|---|------|------|-----------|-------------|
| F1 | `trade_barrier.go` `Acquire` :163-175 无 `cond.Broadcast()` | `WaitState(ctx, barrierSubmitting)` 若在 `Acquire` 完成前进入 `cond.Wait`（:411），其后唤醒只能依赖 :200/:205 等后续广播——彼时 state 已越过 `submitting` → 条件永不命中 → 睡到 ctx 超时。现存两调用方（`mutation_coordinator_test.go:1237`、`live_order_reentry_r4_redo_test.go:197`）忽略返回值且带 2s ctx，行为正确但每交错多耗 ~2s | P3（测试侧时序 footgun，非功能缺陷） | TEST-WAITSTATE-ACQUIRE-BCAST-1：`Acquire` 无 Broadcast 致 WaitState(submitting) 前序等待退化为 ctx 超时 |
| F2 | `position_cache.go` `put` :120 `merged := *snap` 浅拷贝 + `broker_types.go:143` `retained := *merged` 浅拷贝 | retained 快照的 `Positions`/`PendingOrders` slice 与发布管道 `merged` 对象共享底层数组（:120 分支当 `snap.PositionsAuthoritative` 时 :121-129 不执行深拷）；broker 侧 `latest` retained 同样共享。**当前安全**：`mergePositionSnapshot` 所有写入路径均 `append(nil, ...)` 新建 slice（broker_types.go:90-91/:112-113），发布后无原地写。但属跨文件 immutable 约定——任一方未来加原地写即静默 race | P3（健壮性约定，建议注释钉住或改深拷贝） | SNAPSHOT-SLICE-ALIAS-1：PositionCache/broker retained 快照与发布对象共享 slice 底层数组依赖 immutable 约定 |

## 5. 覆盖空洞与局限

1. **race≠证明无竞争**：`-race` 只检测被测试驱动的交错；未被执行路径上的共享写仍可能存在。本审计结论为"未发现"，非"证明无"。
2. **`./strategy` 顶层包无测试**（0 测试文件编译进 run）；4 子包覆盖见 §1。
3. **F2 的 immutable 约定跨两文件**（`position_cache.go` + `mthub/broker_types.go`），单侧修改即可破坏——已通过 F2 记录。
4. **`WaitConfirmed`/`WaitState` watcher 生命周期已证有界**（QS-2.2 goleak）；本审计补证其 Broadcast 配对正确。
5. **未覆盖面**：`ActiveSession`/`SessionRegistry`/`mutationCoordinator` 其他共享字段（如 `sessionDiag`）不在本单坐标范围；VM 单 goroutine 事件模型本身无内部并发，共享面在 watcher/Subscribe/REST 读侧——已覆盖。
6. mql2go 树 49.9s / strategy 树 301.3s / strategy 子树合计 ~38.6s，`-count=3` 三迭代全部执行（`ok` 行即证据）。
