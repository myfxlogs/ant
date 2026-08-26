# Builder Handoff — VM 返工批第二批：VM-TRADE-CONTEXT-1/2（交易上下文失真）

> 日期：2026-08-26 ｜ 审计方：Devin CLI（项目第一负责人） ｜ 施工方：Devin IDE
> 设计 SSOT：`docs/spec/vm-revert-redo-spec.md`（D1-D4 不变）
> registry spec：`:109`（TRADE-CONTEXT-1）、`:116`（TRADE-CONTEXT-2）
> 基线 HEAD：`df69cacf`（2026-08-26，第一批 + R4 返工验收通过后）

## 0. 立项背景

**触发**：D-REVERT-SCOPE-DRIFT-001 第二批——revert `830b2c79` 导致 VM-TRADE-CONTEXT-1/2 修复代码丢失，registry 状态从"施工完成待复审"降级回"🟦open（待施工）"。

**证据链**：
- `grep -rn "invalidateOrderCaches" tools/mql2go/` → 0 匹配（应存在于 `vm_helpers.go`）
- `grep -rn "OppositeTicket" strategy/sdk/ tools/mql2go/` → 0 匹配（应存在于 `sdk.Signal` struct）
- `grep -rn "tradeMagic\|tradeDeviation" tools/mql2go/` → 0 匹配（应存在于 VM struct + `ctradeOrder`）
- `SetExpertMagicNumber`/`SetDeviationInPoints` 绑定为 `builtinNoop`（`vm_builtin_impls.go:215-216`）
- `builtinAccountNumber` 硬编码返回 999999（`vm_builtin_string.go:191`）
- `builtinIsTesting` 用 `ServerTime() > 0` heuristic（`vm_builtin_string.go:183`）
- `brokerImpl.Positions` 静默吞 error（`strategy/runner/broker.go:109`）
- `builtinOrderCloseBy` signal mode 只设 `OrderTicket: ticket1`，无 `OppositeTicket`（`vm_builtin_trade_signals.go:43`）

**设计 SSOT 声明**：`docs/spec/vm-revert-redo-spec.md`（D1-D4 设计决策不变）

**约束与目标**：
- 基于 registry 中的原始修复 spec 重新施工，不做架构变更（D3）
- 每批独立验收（D4）
- revert 不可逆，不尝试恢复被 revert 的代码（D1）

**边界/不做**：
- 不改写历史审计事实
- 不扩大到无关功能块
- 不部署（D-COMMIT-SCOPE-001 部署闸仍有效）
- 不 commit/push/deploy（施工方禁外部操作）
- 禁 `--no-verify`
- 不动 broker/handler 业务语义（只动 VM 层 + sdk 层 + runner 层的 fail-closed 检查）

---

## VM-TRADE-CONTEXT-1：VM 交易上下文在同一事件内失真（P1）

> registry spec：`:109`
> 问题：`OrdersTotal`/`OrderSelect` 使用 lazy slice cache，OrderSend/Close/Modify/Delete 成功后未统一失效；`CTrade.SetExpertMagicNumber`/`SetDeviationInPoints` 绑定为 `builtinNoop`，CTrade.Buy/Sell 丢失 magic/deviation。

### S1 — VM struct 加 tradeMagic/tradeDeviation 字段

**目标**：VM 持有 CTrade setter 设置的 magic/deviation，供 `ctradeOrder` 使用。

**精确坐标**：
- 文件：`backend/tools/mql2go/vm.go:28-45`（`type VM struct`）
- `signalMode bool` 字段后（约 :45）加两个字段

**落点**：
```go
// CTrade setter state (VM-TRADE-CONTEXT-1): SetExpertMagicNumber /
// SetDeviationInPoints write here; ctradeOrder reads from here.
tradeMagic     int32
tradeDeviation int32
```

### S2 — invalidateOrderCaches 函数

**目标**：集中清空所有 order/position cache，供 mutation builtin 成功后调用。

**精确坐标**：
- 文件：`backend/tools/mql2go/vm_helpers.go`（新建函数，放在文件末尾或 Stack helpers 前）
- 方法签名：`func (vm *VM) invalidateOrderCaches()`

**落点**：
```go
// invalidateOrderCaches clears all lazy-loaded order/position caches and
// selection state. Called after every successful mutation builtin and at
// the top of runEvent (VM-TRADE-CONTEXT-1).
func (vm *VM) invalidateOrderCaches() {
	vm.cachedPositions = nil
	vm.cachedOrders = nil
	vm.cachedHistory = nil
	vm.currentPos = nil
	vm.currentOrder = nil
}
```

**注意**：`positionsLoaded`/`ordersLoaded`/`historyLoaded` 字段在当前代码中不存在（revert 后丢失），lazy cache 用 `nil` 检查即可（`cachedPositions == nil` → 下次 OrderSelect 重新加载）。不新增 loaded 标志字段，保持最简。

### S3 — runEvent 顶部调用 invalidateOrderCaches

**目标**：每个 event 从干净状态开始。

**精确坐标**：
- 文件：`backend/tools/mql2go/vm.go:182-190`（`func (vm *VM) runEvent`）
- 当前 :184-188 已手动清空 `currentPos`/`currentOrder`/`cachedPositions`/`cachedOrders`/`cachedHistory`

**落点**：将 :184-188 的 5 行手动清空替换为 `vm.invalidateOrderCaches()` 单行调用：
```go
func (vm *VM) runEvent(ctx context.Context, entryPC int32) error {
	// Reset state for this event invocation (VM-TRADE-CONTEXT-1: centralized cache invalidation).
	vm.stack = vm.stack[:0]
	vm.invalidateOrderCaches()
	vm.callDepth = 0
	vm.signal = nil
	vm.pc = entryPC
	// ... (rest unchanged)
```

### S4 — mutation builtin 成功后调用 invalidateOrderCaches

**目标**：所有 mutation builtin 成功后统一失效 cache。

**精确坐标**（两类路径，分别处理）：

#### S4a — broker path（非 signalMode）

在以下函数的 **broker 成功路径**（`err == nil` 后的 return 前）加 `vm.invalidateOrderCaches()`：

- `backend/tools/mql2go/vm_builtin_trade.go`:
  - `builtinOrderSend`（:12）— broker path `return interp.IntVal(int32(result.Ticket)), nil` 前（:75）
- `backend/tools/mql2go/vm_builtin_trade_signals.go`:
  - `builtinOrderClose`（:17）— `return interp.BoolVal(err == nil), nil` 前（:33）→ 改为 `if err != nil { return interp.BoolVal(false), nil }; vm.invalidateOrderCaches(); return interp.BoolVal(true), nil`
  - `builtinOrderCloseBy`（:35）— 同上模式（:49）
  - `builtinOrderModify`（:52）— broker path 成功 return 前
  - `builtinOrderDelete`（:84）— broker path 成功 return 前
  - `builtinCTradePositionClose`（:100）— broker path 成功 return 前
  - `builtinCTradePositionClosePartial`（:117）— broker path 成功 return 前
  - `builtinCTradePositionCloseBy`（:135）— broker path 成功 return 前（:149）
  - `builtinCTradePositionModify`（:152）— broker path 成功 return 前
  - `builtinCTradeOrderDelete`（:172）— broker path 成功 return 前
  - `builtinCloseAll`（:188）— 已有 `vm.cachedPositions = nil; vm.cachedOrders = nil`（:201-202），替换为 `vm.invalidateOrderCaches()`

- `backend/tools/mql2go/vm_builtin_trade.go`:
  - `ctradeOrder`（:565）— broker path `_, err := vm.ctx.Broker().OrderSend(req)` 成功 return 前（:620）

#### S4b — signal path（signalMode）

signal path 不操作 broker，但 **仍需失效 cache**——因为 signal 被外部 dispatch 后 broker 会执行 mutation，同一 event 内后续 OrderSelect 不应返回 stale cache。

在以下函数的 **signal path 成功 return 前**加 `vm.invalidateOrderCaches()`：

- `builtinOrderSend`（:12）signal path—`return interp.IntVal(1), nil` 前（:65）
- `builtinOrderClose`（:17）signal path—`return interp.BoolVal(true), nil` 前（:27）
- `builtinOrderCloseBy`（:35）signal path—`return interp.BoolVal(true), nil` 前（:45）
- `builtinOrderModify`（:52）signal path—成功 return 前
- `builtinOrderDelete`（:84）signal path—成功 return 前
- `builtinCTradePositionClose`（:100）signal path—成功 return 前
- `builtinCTradePositionClosePartial`（:117）signal path—成功 return 前
- `builtinCTradePositionCloseBy`（:135）signal path—成功 return 前
- `builtinCTradePositionModify`（:152）signal path—成功 return 前
- `builtinCTradeOrderDelete`（:172）signal path—成功 return 前
- `builtinCloseAll`（:188）signal path—成功 return 前
- `ctradeOrder`（:565）signal path—`return interp.BoolVal(true), nil` 前（:613）

**对抗证明**：
- 突变：删 `builtinOrderClose` broker path 的 `invalidateOrderCaches()` → bar1 OrderSend 开仓 → bar2 OrdersTotal=1（load cache）→ OrderClose → OrdersTotal 仍=1（stale cache）→ RED
- 恢复 → GREEN

### S5 — builtinOrderSelect 顶部 reset currentPos/currentOrder

**目标**：失败的 OrderSelect 不留旧属性。

**精确坐标**：
- 文件：`backend/tools/mql2go/vm_builtin_trade.go:139`（`func builtinOrderSelect`）
- 当前 :139-141 读取 index/selectBy/pool，无 reset

**落点**：在函数体开头（读取参数前）加：
```go
func builtinOrderSelect(vm *VM, args []interp.Value) (interp.Value, error) {
	// VM-TRADE-CONTEXT-1: reset selection state — failed select must not leave stale currentPos/currentOrder.
	vm.currentPos = nil
	vm.currentOrder = nil

	index := int(argI(args, 0))
	// ... (rest unchanged)
```

**对抗证明**：
- 突变：删 reset → bar1 OrderSend → bar2 OrderSelect(0) 成功 g_ticket>0 → OrderSelect(999) 失败 → OrderTicket() 仍返回旧 ticket → RED
- 恢复 → GREEN

### S6 — SetExpertMagicNumber/SetDeviationInPoints 透传到 VM state

**目标**：CTrade setter 不再是 noop，写入 VM 的 tradeMagic/tradeDeviation。

**精确坐标**：
- 文件：`backend/tools/mql2go/vm_builtin_impls.go:215-216`
- 当前：
```go
builtinRegistry[id("CTrade.SetExpertMagicNumber")].fn = builtinNoop
builtinRegistry[id("CTrade.SetDeviationInPoints")].fn = builtinNoop
```

**落点**：
1. 新增两个 builtin 函数（放在 `vm_builtin_trade.go` 末尾或 `vm_builtin_impls.go` 中）：
```go
func builtinCTradeSetExpertMagicNumber(vm *VM, args []interp.Value) (interp.Value, error) {
	vm.tradeMagic = argI(args, 0)
	return interp.NoneVal(), nil
}

func builtinCTradeSetDeviationInPoints(vm *VM, args []interp.Value) (interp.Value, error) {
	vm.tradeDeviation = argI(args, 0)
	return interp.NoneVal(), nil
}
```

2. `vm_builtin_impls.go:215-216` 改为：
```go
builtinRegistry[id("CTrade.SetExpertMagicNumber")].fn = builtinCTradeSetExpertMagicNumber
builtinRegistry[id("CTrade.SetDeviationInPoints")].fn = builtinCTradeSetDeviationInPoints
```

### S7 — ctradeOrder 使用 tradeMagic/tradeDeviation

**目标**：CTrade.Buy/Sell 等订单操作携带 setter 设置的 magic/deviation。

**精确坐标**：
- 文件：`backend/tools/mql2go/vm_builtin_trade.go:565`（`func ctradeOrder`）
- broker path 的 `req` 构造（:583-591）加 `Magic: vm.tradeMagic, Deviation: vm.tradeDeviation`
- signal path 的 `vm.signal` 构造（:607-613）加 `Magic: vm.tradeMagic, Deviation: vm.tradeDeviation`

**落点**：
```go
// broker path req 构造（:583-591 后加）
req := sdk.OrderRequest{
	Symbol:     symbol,
	Type:       orderType,
	Side:       side,
	Volume:     volume,
	Price:      price,
	StopLoss:   sl,
	TakeProfit: tp,
	Comment:    comment,
	Magic:      vm.tradeMagic,      // VM-TRADE-CONTEXT-1
	Deviation:  vm.tradeDeviation,  // VM-TRADE-CONTEXT-1
}

// signal path vm.signal 构造（:607-613 后加）
vm.signal = &sdk.Signal{
	Action:     action,
	Symbol:     symbol,
	Volume:     volume,
	Price:      price,
	StopLoss:   sl,
	TakeProfit: tp,
	Comment:    comment,
	Magic:      vm.tradeMagic,      // VM-TRADE-CONTEXT-1
	Deviation:  vm.tradeDeviation,  // VM-TRADE-CONTEXT-1
}
```

**对抗证明**：
- 突变：删 signal path 的 `Magic: vm.tradeMagic`（改为 `Magic: 0`）→ OnInit SetExpertMagicNumber(999) → signalMode OnTick CTrade.Buy → sig.Magic=0 ≠ 999 → RED
- 恢复 → GREEN

---

## VM-TRADE-CONTEXT-2：CloseBy signal 丢第二票据 + 账户平台值硬编码 + broker query error 静默吞（P0/P1）

> registry spec：`:116`

### S8 — sdk.Signal 加 OppositeTicket 字段

**目标**：CloseBy signal 携带第二个票据。

**精确坐标**：
- 文件：`backend/strategy/sdk/strategy.go:34`（`type Signal struct`）
- `OrderTicket int64` 字段后（:45）加 `OppositeTicket int64`

**落点**：
```go
OrderTicket  int64           // for modify/close/cancel: which order to act on
OppositeTicket int64         // for CloseBy: the opposite position ticket (VM-TRADE-CONTEXT-2)
```

### S9 — builtinOrderCloseBy / builtinCTradePositionCloseBy signal mode 设置 OppositeTicket

**目标**：CloseBy signal 携带双票据。

**精确坐标**：
- `backend/tools/mql2go/vm_builtin_trade_signals.go:41-45`（`builtinOrderCloseBy` signal path）
- `backend/tools/mql2go/vm_builtin_trade_signals.go:141-145`（`builtinCTradePositionCloseBy` signal path）

**落点**：
```go
// builtinOrderCloseBy signal path（:43-45）
vm.signal = &sdk.Signal{
	Action:         sdk.ActionClose,
	OrderTicket:    ticket1,
	OppositeTicket: ticket2,  // VM-TRADE-CONTEXT-2
}

// builtinCTradePositionCloseBy signal path（:143-145）
vm.signal = &sdk.Signal{
	Action:         sdk.ActionClose,
	OrderTicket:    t1,
	OppositeTicket: t2,  // VM-TRADE-CONTEXT-2
}
```

**对抗证明**：
- 突变：删 `OppositeTicket: ticket2` → signal OppositeTicket=0 ≠ 200 → RED
- 恢复 → GREEN

### S10 — sdk.AccountInfo 加 Login/Company 字段

**目标**：AccountNumber 从 context 注入而非硬编码。

**精确坐标**：
- 文件：`backend/strategy/sdk/broker.go:46`（`type AccountInfo struct`）
- `Mode AccountMode` 字段后（:54）加两个字段

**落点**：
```go
Mode          AccountMode
Login         int64    // account login number, for AccountNumber() (VM-TRADE-CONTEXT-2)
Company       string   // broker company name, for AccountCompany() (VM-TRADE-CONTEXT-2)
```

### S11 — builtinAccountNumber 从 context 读取

**目标**：不再硬编码 999999。

**精确坐标**：
- 文件：`backend/tools/mql2go/vm_builtin_string.go:189-193`（`func builtinAccountNumber`）

**落点**：
```go
func builtinAccountNumber(vm *VM, args []interp.Value) (interp.Value, error) {
	// VM-TRADE-CONTEXT-2: read from context instead of hardcoding 999999.
	if vm.ctx != nil {
		login := vm.ctx.Account().Login
		if login == 0 {
			// Record blind spot — AccountNumber unavailable.
			// (If blind spot recording mechanism exists, use it; otherwise return 0.)
			return interp.IntVal(0), nil
		}
		return interp.IntVal(int32(login)), nil
	}
	return interp.IntVal(0), nil
}
```

**注意**：`vm.ctx.Account()` 返回 `sdk.AccountInfo`。检查 `sdk.Context` 接口是否有 `Account()` 方法——如果没有，需要加。查看 `strategy/sdk/strategy.go` 中 `Context` 接口定义。

**对抗证明**：
- 突变：恢复硬编码 `return interp.IntVal(999999), nil` → Login=12345 → AccountNumber()=999999 ≠ 12345 → RED
- 恢复 → GREEN

### S12 — builtinIsTesting 改为 !vm.signalMode

**目标**：backtest=true/live=false，不再用 ServerTime heuristic。

**精确坐标**：
- 文件：`backend/tools/mql2go/vm_builtin_string.go:182-186`（`func builtinIsTesting`）

**落点**：
```go
func builtinIsTesting(vm *VM, args []interp.Value) (interp.Value, error) {
	// VM-TRADE-CONTEXT-2: backtest mode = !signalMode (signalMode is true for live trading).
	return interp.BoolVal(!vm.signalMode), nil
}
```

**对抗证明**：
- 突变：恢复旧 heuristic `if vm.ctx != nil && vm.ctx.ServerTime() > 0 { return interp.BoolVal(true), nil }` → live mode (signalMode=true) 但 ServerTime>0 → IsTesting()=true ≠ false → RED
- 恢复 → GREEN

### S13 — brokerImpl 加 lastError + Positions/Orders/HistoryOrders/Deals 记录 error

**目标**：broker query error 不再静默吞，记录到 lastError 供 Runner 检查。

**精确坐标**：
- 文件：`backend/strategy/runner/broker.go:13-18`（`type brokerImpl struct`）

**落点**：
1. struct 加字段（:16 `ctx context.Context` 后）：
```go
type brokerImpl struct {
	runner   *Runner
	executor OrderExecutor
	ctx      context.Context
	lastError error  // VM-TRADE-CONTEXT-2: records last broker query error
}
```

2. 新增方法（放在 `setContext` 后）：
```go
// LastError returns the last broker query error (VM-TRADE-CONTEXT-2).
func (b *brokerImpl) LastError() error { return b.lastError }

// resetError clears the last error (called at the start of each event by Runner).
func (b *brokerImpl) resetError() { b.lastError = nil }
```

3. `Positions`（:92）error 路径（:109-112）改为记录 error：
```go
positions, err := b.executor.OpenedOrders(b.orderCtx())
if err != nil {
	b.lastError = fmt.Errorf("broker Positions query: %w", err)
	return nil
}
```

4. `Orders`（:128）同样模式——找到 executor path 的 error 路径，加 `b.lastError = ...`

5. `HistoryOrders`（:163）改为记录 "not available in live mode" error（当 executor != nil 时）：
```go
func (b *brokerImpl) HistoryOrders(from, to int64) []sdk.Position {
	if b.executor != nil {
		b.lastError = fmt.Errorf("HistoryOrders not available in live mode")
	}
	return nil
}
```

6. `Deals`（:170）同样模式：
```go
func (b *brokerImpl) Deals(from, to int64, magic int32) []sdk.Deal {
	if b.executor != nil {
		b.lastError = fmt.Errorf("Deals not available in live mode")
	}
	return nil
}
```

**注意**：harness mode（`executor == nil`）不设 lastError——harness 是 backtest，HistoryOrders/Deals 返回 nil 是正常的。

### S14 — Runner.OnBar/OnTick 检查 broker.LastError() 并 fail-closed

**目标**：broker query error 导致策略执行 fail-closed。

**精确坐标**：
- 文件：`backend/strategy/runner/runner.go:110`（`func (r *Runner) OnBar`）
- 文件：`backend/strategy/runner/runner.go:131`（`func (r *Runner) OnTick`）
- 同样模式适用于 OnTrade（:149）/ OnTimerTick（:163）/ OnBookEvent（:177）/ OnTimer（:191）

**落点**：
1. 在每个 `On*` 方法中，`r.broker.setContext(ctx)` 后加 `r.broker.resetError()`
2. 在 `return r.strategy.OnBar(...)` / `return ts.OnTick(...)` 后（strategy 执行后）检查 `LastError()`：
```go
func (r *Runner) OnBar(ctx context.Context, bars sdk.BarSeries, timeframe string) (*sdk.Signal, error) {
	if r.strategy == nil {
		return nil, nil
	}
	r.ctx.setGoContext(ctx)
	r.broker.setContext(ctx)
	r.broker.resetError()  // VM-TRADE-CONTEXT-2
	r.mu.Lock()
	r.ctx.setBars(bars)
	r.mu.Unlock()
	r.barRev.Add(1)
	sig, err := r.strategy.OnBar(r.ctx, timeframe)
	if err != nil {
		return nil, err
	}
	// VM-TRADE-CONTEXT-2: fail-closed on broker query error.
	if brokerErr := r.broker.LastError(); brokerErr != nil {
		return nil, fmt.Errorf("runner OnBar fail-closed: %w", brokerErr)
	}
	return sig, nil
}
```

3. 对 OnTick/OnTrade/OnTimerTick/OnBookEvent/OnTimer 同样模式——strategy 执行后检查 `LastError()`。

**对抗证明**：
- 突变：恢复 `Positions` 静默吞 error（`_ = err; return nil`）→ executor 返回 error → LastError nil → Runner 不 fail-closed → RED
- 恢复 → GREEN

---

## 新增行为测试

> 文件：`backend/tools/mql2go/vm_trade_context_redo_test.go`（新建）
> 测试用真实 MQL→VM→SimBroker 端到端验证

### VM-TRADE-CONTEXT-1 测试（4 个）

1. **TestOrderCacheInvalidatedAfterClose**：bar1 OrderSend 开仓 → bar2 OrdersTotal=1（load cache）→ OrderClose → OrdersTotal=0（cache invalidated，非 stale 1）
2. **TestCTradeMagicDeviationReachLiveSignal**：OnInit SetExpertMagicNumber(999)+SetDeviationInPoints(77) → signalMode OnTick CTrade.Buy → sig.Magic=999 + sig.Deviation=77 + sig.Action=ActionBuy
3. **TestFailedOrderSelectResetsCurrent**：bar1 OrderSend → bar2 OrderSelect(0) 成功 g_ticket>0 → OrderSelect(999) 失败 → OrderTicket()=0（currentPos reset）
4. **TestInvalidTicketOrderCloseFails**：OrderClose(99999) → broker error → fail-closed 停止执行 → Engine.Run 返回 error + g_after=-1

### VM-TRADE-CONTEXT-2 测试（9 个）

1. **TestSignalMode_OrderCloseBy_BothTickets**：ticket1=100, ticket2=200 → OppositeTicket=200
2. **TestSignalMode_CTradePositionCloseBy_BothTickets**：同上 CTrade path
3. **TestAccountNumber_FromContext**：Login=12345 → AccountNumber()=12345
4. **TestAccountNumber_ZeroLoginReturnsZero**：Login=0 → AccountNumber()=0
5. **TestIsTesting_BacktestMode**：signalMode=false → IsTesting()=true
6. **TestIsTesting_LiveMode**：signalMode=true → IsTesting()=false
7. **TestBrokerImpl_PositionsQueryError_RecordsLastError**：executor error → LastError 非 nil
8. **TestBrokerImpl_OrdersQueryError_RecordsLastError**：同上 Orders
9. **TestBrokerImpl_HistoryOrders_NotAvailable_RecordsError**：live mode → LastError 非 nil

**测试基础设施**：
- 检查是否已有 SimBroker mock——`grep -rn "SimBroker\|simBroker" tools/mql2go/` 
- 检查是否已有 VM test helper（编译 MQL → VMRunner → OnBar/OnTick）——`grep -rn "func.*compileAndRun\|func.*newTestVM\|func.*setupTestVM" tools/mql2go/`
- 如已有测试基础设施（如 `live_mql_order_context_vm_test.go` 中的 helper），REUSE 之；否则新建最小 helper

---

## 验收标准

1. **对抗证明 8 项**（每项 RED→restore→GREEN）：
   - S4: 删 `builtinOrderClose` 的 `invalidateOrderCaches()` → `TestOrderCacheInvalidatedAfterClose` RED
   - S5: 删 `builtinOrderSelect` 顶部 reset → `TestFailedOrderSelectResetsCurrent` RED
   - S7: 删 `ctradeOrder` signal path 的 `Magic: vm.tradeMagic` → `TestCTradeMagicDeviationReachLiveSignal` RED
   - S9: 删 `builtinOrderCloseBy` signal `OppositeTicket` → `TestSignalMode_OrderCloseBy_BothTickets` RED
   - S11: 恢复 `builtinAccountNumber` 硬编码 999999 → `TestAccountNumber_FromContext` RED
   - S12: 恢复 `builtinIsTesting` 旧 heuristic → `TestIsTesting_LiveMode` RED
   - S13/S14: 恢复 `brokerImpl.Positions` 静默吞 error → `TestBrokerImpl_PositionsQueryError_RecordsLastError` RED
   - S13/S14: 恢复 `Runner.OnBar` 不检查 LastError → broker error 不 fail-closed → RED（如可构造端到端测试）

2. **门禁全绿**：
   - `go build ./...`
   - `go test ./tools/mql2go/... -count=1`
   - `go test -race ./tools/mql2go/... -count=1` ×3
   - `go test ./internal/connect/strategy -count=1`
   - `go test -race ./internal/connect/strategy -count=1` ×3
   - `go vet ./...`
   - `go run ./tools/check-file-lines --strict`（0 errors）
   - `git diff --check`

3. **file-lines**：新增文件 `vm_trade_context_redo_test.go` 不超 450 行。如 `vm_builtin_trade.go` 或 `vm_builtin_trade_signals.go` 因加 `invalidateOrderCaches()` 调用导致超限，考虑将部分函数拆分到独立文件。

4. **复用核对**：`bash scripts/cap.sh invalidateOrderCaches` / `cap.sh OppositeTicket` / `cap.sh tradeMagic` / `cap.sh lastError` / `cap.sh SimBroker`

## 红队自审（施工方完工前必答）

1. `invalidateOrderCaches()` 是否在所有 mutation builtin 的**所有成功路径**（broker path + signal path）都调用了？
2. `builtinOrderSelect` 顶部 reset 是否在读取参数前执行？
3. `ctradeOrder` 的 broker path 和 signal path 是否都用了 `vm.tradeMagic`/`vm.tradeDeviation`？
4. `sdk.Context` 接口是否有 `Account()` 方法？如果没有，S11 需要先加接口方法——检查 `strategy/sdk/strategy.go` 中 `Context` 接口定义。
5. `brokerImpl.lastError` 是否在 harness mode（executor==nil）时不被设置？harness 是 backtest，不应有 broker error。
6. `Runner.On*` 方法的 `resetError()` 是否在每个 event 开头调用？`LastError()` 检查是否在 strategy 执行后？
7. `OppositeTicket` 字段是否在 `builtinOrderCloseBy` 和 `builtinCTradePositionCloseBy` 两个函数都设置了？
8. `builtinIsTesting` 的 `!vm.signalMode` 是否正确——backtest 时 signalMode=false → IsTesting=true；live 时 signalMode=true → IsTesting=false？

## 回填纪律

1. registry `VM-TRADE-CONTEXT-1`（:109）和 `VM-TRADE-CONTEXT-2`（:116）：状态改为 `🟦open（施工完成，待独立复审）` + 真实实现 + 对抗证明结果
2. `handover-audit-plan.md` 变更日志加一行
3. **不自行宣告完成**——停手等 Devin CLI 复审

## 范围约束

One task = one scope：只动以下文件：
- `backend/tools/mql2go/vm.go`（S1 struct 字段 + S3 runEvent）
- `backend/tools/mql2go/vm_helpers.go`（S2 invalidateOrderCaches）
- `backend/tools/mql2go/vm_builtin_trade.go`（S4a/S5/S6/S7）
- `backend/tools/mql2go/vm_builtin_trade_signals.go`（S4a/S4b/S9）
- `backend/tools/mql2go/vm_builtin_impls.go`（S6 注册）
- `backend/tools/mql2go/vm_builtin_string.go`（S11/S12）
- `backend/strategy/sdk/strategy.go`（S8 OppositeTicket）
- `backend/strategy/sdk/broker.go`（S10 AccountInfo 字段）
- `backend/strategy/runner/broker.go`（S13 lastError）
- `backend/strategy/runner/runner.go`（S14 fail-closed 检查）
- `backend/tools/mql2go/vm_trade_context_redo_test.go`（新建测试）

不顺手重构、不改无关逻辑、不动 broker/handler 业务语义。

## 固定尾部

**勿部署，停手等 Devin CLI 复审。** 禁 `--no-verify`。禁 commit/push/deploy。只 add 本任务文件，禁 `git add -A`。
