# 设计 SSOT — VM-LIVE-SYNC-DISPATCH-1（R1：signal-mode 假票号哨兵根治）

> 状态：v2（独立复审修订：S5 增 affectedTickets；dispatchResponse 防双发改用 sessionSyncDispatched 实测标记而非 cfg.Mode 判定；T9/M4 补录）
> 触发：registry R1 —— signal-mode `OrderSend` 返回 `IntVal(1)` 假票号；策略存 ticket 做 OrderSelect/OrderClose 必然失配
> 实查坐标：`tools/mql2go/vm_builtin_trade.go:70-88`、`tools/mql2go/vm_builtin_trade_signals.go`（10 处 signalMode 分支）、`internal/connect/strategy/live_context.go:145-198`、`live_dispatch.go`、`mutation_coordinator.go`、`vm_live_session.go`、`vm_live_handlers.go`、`strategy/runner/broker.go`

---

## 0. 实查定论（全部已读码坐实）

1. **信号管线**：signalMode builtin → `vm.signal`（单槽位）→ `r.OnTick/OnBar` 返回 → `vmSignalResponse` → `resp.Signals` → `dispatchResponse` → `dispatchLiveSignal` → `coordinateMutation`。**`coordinateMutation` 本就是同步阻塞**在事件循环 goroutine 上，`mutationResult.ticket` 拿到真票号——但此刻 VM 的 `OrderSend` 早已返回 `IntVal(1)`，真票号只进日志/DB，永远回不到策略变量。
2. **`runner.OrderExecutor` 接口存在但生产从未接线**（`broker.go:15` 注释 "set by LiveRunner"，全仓仅测试赋值）——`ctx.Broker().OrderSend` 的同步通道在 live 是死的，signalMode 是唯一通路。
3. **同类潜伏缺陷 A（丢单）**：`vm.signal` 单槽位 last-wins——同一事件内第二次 `OrderSend`/任何第二次信号发射**静默覆盖前一个信号，订单整单丢失且无任何记录**。
4. **同类潜伏缺陷 B（空转）**：`ActionCancelAll` → proto `"cancel_all"` 在 `dispatchLiveSignal` 的 switch 中无 case，落 `default` 空转——`OrderDelete` 全部挂单时静默无效。
5. **记录链**：`dispatchLiveSignal` 内做 `InsertScheduleRunLog("received")` + `log.Info` + `persistSignal`（DB 审计行）；`dispatchResponse` 做 `RecordSignal` + shadow verifier。**这些必须原样保留**。
6. **同事件可见性**：MQL4 真语义是 `OrderSend→OrderSelect(ticket)` 同事件立即可查。当前 `OrderSelect` 走 `ctx.livePositions`（事件前 backfill 快照）——同步成交后新仓位不在快照内，同事件 OrderSelect 会查不到。须由 dispatcher 在 confirmed 后把 broker 事实注入 runner live state。

## 1. 设计决策：同步信号派发（sync dispatch），非 executor 接线

**方案 B（选定）**：signalMode 分支保持发 `vm.signal`（响应/审计链不动），新增 `vm.syncDispatch` 回调——builtin 在分支内**同步调用** dispatcher，dispatcher 就是 `dispatchSignalSync`（= `dispatchLiveSignal` 的改造版，返回 `mutationResult`）。回调在同一事件循环 goroutine 上执行 `coordinateMutation`——与今天完全相同的阻塞语义，只是从"事件返回后"移进"builtin 调用栈内"。

**为何不选 executor 接线（方案 A）**：`ctx.Broker()` 的 executor 通道生产从未接线，接线要求 live executor 实现全部 8 个方法（含 Positions/Account/SymbolInfo 查询——会绕开已验证的 push 快照链路改成 RPC 拉取），且要逐 builtin 验证非 signalMode 路径（今天只有 backtest 走）。方案 B 是**最小真实改动**：信号仍是协调单元，barrier/熔断/读回验证/审计全复用。

**语义恢复**：MQL4 `OrderSend` 是同步的——返回值=broker 真票号或 -1。本设计把 live 路径恢复为该语义（与 backtest `SimBroker` 同步语义对齐，live/backtest 一致性提升）。

## 2. 组件设计

### S1 — VM 层（`tools/mql2go`）

```go
// vm.go
syncDispatch func(*sdk.Signal) (int64, error)  // nil = 异步信号模式（paper）
// SetSyncDispatcher 挂到 VM 与 VMRunner
```

10 处 signalMode 分支统一改造（提取公共 helper `vm.emitSignal(sig)`）：

```go
func (vm *VM) emitSignal(sig *sdk.Signal) (int64, error) {
    vm.signal = sig               // 响应/审计链不变
    vm.invalidateOrderCaches()
    if vm.syncDispatch == nil {
        return 1, nil             // paper：无 broker 票号，保留哨兵（登记残留）
    }
    ticket, err := vm.syncDispatch(sig)
    if err != nil {
        return 0, err
    }
    if sig.OrderTicket == 0 {
        sig.OrderTicket = ticket  // open 类：真票号写入已发射信号 → 审计行带真票号
    }
    return ticket, nil
}
```

各分支返回约定：
- `OrderSend`：`(ticket, err)` → ok `IntVal(ticket)`；err `IntVal(-1)` + `vm.lastError`（映射见 S3）
- `OrderClose`/`OrderModify`/`OrderDelete`/`OrderCloseBy`/CTrade 系/`CloseAll`：`BoolVal(err == nil)`；err 时 `vm.lastError`

**丢单修复**：`vm.signal` 单槽保留（响应兼容），但**每个信号都已同步执行**——同事件第二次 OrderSend 不再依赖信号槽，执行语义不再丢单。（响应侧 `resp.Signals` 仍只带最后一个信号做 RecordSignal——与今天相同，无回退；而 DB 审计 `persistSignal` 在 sync dispatch 内按次执行，**审计覆盖率反而提升**。）

### S2 — Runner/Session 层

- `Runner.SetSyncDispatcher` / `VMRunner.SetSyncDispatcher` / `VMLiveSession.SetSyncDispatcher` 透传挂到 VM。
- `VMLiveSession` 加 `SyncDispatched() bool`（dispatcher 已挂 = live 同步模式）。

### S3 — Server 层（`internal/connect/strategy`）

**`dispatchSignalSync(ctx, cfg, bar, sig *sdk.Signal, activeSess) (int64, error)`**：
- `sdk.Signal` → proto：复用 `vmSignalToProto` 逻辑（提取包内函数 `sdkSignalToProto`）
- 复用 `dispatchLiveSignal` 的 preamble：`InsertScheduleRunLog("received")` + `log.Info` + `persistSignal`
- action switch → 各 dispatcher **返回 `mutationResult`**（`submitOrder`/`dispatchCloseOrder`/`dispatchModifyOrder`/`dispatchCancelOrder`/`dispatchCloseAll` 改返回值）
- 结果映射：confirmed→`(ticket, nil)`；deterministicRejected→`(0, ErrSignalRejected)`；outcomeUnknown/idle→`(0, ErrSignalUnknown)`——builtin 侧 `vm.lastError = 146`（ERR_TRADE_CONTEXT_BUSY，无 broker retcode 可得时的诚实泛化）+ `IntVal(-1)`/`BoolVal(false)`
- **confirmed 后注入 runner live state**（见 S4）
- 顺带修潜伏缺陷 B：`cancel_all` 加 case → `dispatchCancelAll`（与 close_all 同构，遍历挂单逐个 cancel 经 coordinator；任 outcomeUnknown 停后续）
- paper 模式永不走此路径（dispatcher 只在 live 挂）

**接线点**：`handleBar`/`handleTick`/`handleTrade` 在 `SendEvent` 前（live 模式）：

```go
if cfg.Mode == modeLive {
    if vmSess, ok := (*session).(*VMLiveSession); ok {
        vmSess.SetSyncDispatcher(func(sig *sdk.Signal) (int64, error) {
            return s.dispatchSignalSync(ctx, cfg, bar, sig, activeSess)  // bar 按事件闭包
        })
    }
}
```

**`dispatchResponse` 防双发（v2 修订——实现用实测标记而非模式判定）**：`dispatchResponse` 增参 `alreadyDispatched bool`，调用点传 `sessionSyncDispatched(*session)`——`VMLiveSession.SyncDispatched()` 是该 session 实际挂了 sync dispatcher 的**实测标记**（wireSyncDispatch 每事件前置安装后置位）。比 `cfg.Mode` 判定精确：非 VM session/未接线 session 不会误跳过。信号已在事件内同步执行时 `continue` 跳过 `dispatchLiveSignal`，`RecordSignal` + shadow verifier 原样保留——**绝不二次执行**（live 信号经 syncDispatch 外的路径到达此处属 bug，跳过即 fail-closed）。

### S4 — 同事件仓位可见性（confirmed 事实注入）

`dispatchSignalSync` confirmed 后把 **broker 事实**注入 runner live state（`Runner` 新增方法，均 `ctx.mu` 保护）：
- open confirmed：`AppendLivePosition(sdk.Position{Ticket, Symbol, Side, Volume, OpenPrice, ...})`（来自 `mthub.OrderRecord`）
- close confirmed：`RemoveLivePosition(ticket)`
- pending open confirmed：`AppendLivePendingOrder(sdk.PendingOrder{...})`
- cancel confirmed：`RemoveLivePendingOrder(ticket)`
- modify confirmed：`UpdateLivePosition(ticket, sl, tp)` / pending 同理

依据：coordinator 的 read-after-write 已证明这些是 broker 事实——注入不是推测。效果是 `OrderSend→OrderSelect(ticket)` 同事件即可查（真 MQL4 语义）。

`mutationSpec` 需要带回 `*mthub.OrderRecord`（open）以便取字段——`brokerCall` 已返回 record.Ticket，扩展为返回 record 本体或填 `result.record`。

### S5 — `mutationResult` 扩展（v2 增 affectedTickets——复审抓出的缺口）

```go
type mutationResult struct {
    state  tradeBarrierState
    ticket int64
    record *mthub.OrderRecord // open confirmed 时的 broker 事实（可 nil）
    // affectedTickets: close_all/cancel_all 批内每个【单条已确认】的票号。
    // 批次即便中途 outcome_unknown 早退，已确认条目仍是 broker 事实，
    // runner live state 可诚实移除。
    affectedTickets []int64
}
```

**复审缺口实录**：v1 只覆盖单票号 action 的注入。`close_all`/`cancel_all` 批确认后 `dispatchSignalSync` 拿不到"哪些票号已删"——同事件 `OrderSelect`/`OrdersTotal` 会读到幽灵仓位。修法：`dispatchCloseAll`/`dispatchCancelAll` 循环内累积已确认票号（confirmed 才 append，deterministicRejected 不 append），正常返回与 unknown 早退路径都随 result 带出；`dispatchSignalSync` 在 `barrierConfirmed` 走 `applyConfirmedToRunner`（单票号 action），`affectedTickets` 独立于 state 逐票移除（幂等）。

## 3. 边界（不做）

- paper 模式假票号哨兵保留（paper 无 broker 票号概念；登记为已知残留，后续可让 PaperEngine 分配纸面票号）
- 不动 `resp.Signals` proto 结构、不动 `dispatchResponse` 的记录语义、不动 paper 派发
- R4 CloseBy dispatch 不扩（CloseBy 信号仍无 live dispatch——registry 已记）
- `ExecutedTicket` proto 语义不变
- 不动 timeframe 归一化（另案）

## 4. 测试矩阵（先红后绿 + mutation）

| # | 断言 |
|---|------|
| T1 | signalMode+syncDispatch stub 返回 ticket=5678 → `OrderSend` 返回 5678（非 1）；`sig.OrderTicket` 回填 5678 |
| T2 | stub 返回 err → `OrderSend` 返回 -1 且 `vm.lastError` 非 0；`OrderClose` 返回 false |
| T3 | syncDispatch=nil（paper 等价）→ `OrderSend` 返回 1（残留哨兵行为不变） |
| T4 | 同事件两次 `OrderSend` → dispatcher 被调两次（无丢单） |
| T5 | server 侧 `dispatchSignalSync`：mock mthub confirmed open → 返回真 ticket + `AppendLivePosition` 生效（`Positions()` 同事件可见） |
| T6 | mock deterministic reject → err 返回，无仓位注入 |
| T7 | `cancel_all` 信号 → dispatchCancelAll 被调（潜伏缺陷 B 修复覆盖） |
| T8 | live dispatchResponse 不二次执行（mthub PlaceOrder 调用计数=1） |
| T9 | `close_all` confirmed → affectedTickets 移除 runner 仓位（无幽灵仓位残留） |
| M1 | builtin 删掉 syncDispatch 调用直接 `return IntVal(1)` → T1 红（已实证） |
| M2 | dispatchResponse 恢复调用 dispatchLiveSignal（双发）→ T8 红（计数=2）（已实证） |
| M3 | 删 `applyConfirmedToRunner` 调用 → T5 红（confirmed 仓位不进 runner）（已实证） |
| M4 | 删 affectedTickets 移除循环 → T9 红（幽灵仓位残留）（已实证） |

## 5. 风险与回归面

- **执行位置前移**：mutation 从"事件返回后"移到"builtin 调用栈内"——同一 goroutine、同一 barrier、同一 coordinator，阻塞语义不变。`dispatchLiveSignal` 的 panic-recover 在 sync 路径同样需要（dispatcher 内 recover → err 返回 builtin）。
- **事件时长**：OnTick 现在包含 broker RTT + push 确认等待——与 LIVE-ORDER-REENTRY-1 设计意图一致（"event loop blocks until deterministic outcome"），tick 流由 mthub 缓冲承受。
- **VM fatal 面**：sync dispatch 的 err 是**业务失败**（→-1/false），非 Go error——不触发 fatalError，策略可续跑重试（真 MQL 语义）。
- **paper 零变化**。
