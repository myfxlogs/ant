# Strategy Execution Audit Report

## Scope

Comprehensive review of the Strategy Execution subsystem:
- LiveRunner (`live_runner.go`) — real-time strategy execution engine
- Live dispatch (`live_dispatch.go`) — signal routing, order submission, circuit breaker
- Live runner events (`live_runner_events.go`) — bar/tick/trade event handling
- SimBroker (`broker.go`) — simulated broker for backtesting
- Backtest engine (`engine.go`) — bar-by-bar simulation, SL/TP checking, equity tracking
- Backtest worker (`backtest_worker_vm.go`) — VM-based backtest execution
- Schedule engine (`schedule_engine.go`) — timer-driven schedule dispatch
- Session registry (`session_registry.go`) — active session tracking, conflict detection
- Strategy active handlers (`strategy_active_handlers.go`) — Start/Stop/Watch/List RPCs
- Metrics calculation (`metrics.go`) — return, drawdown, Sharpe, win rate
- Fill model (`fill_model.go`) — cost decomposition
- Stale run cleanup (`strategy_run_repo.go`) — crash recovery

## Findings

### S6 — Missing session ownership check in StopStrategy/GetActiveStrategy/WatchStrategySignals 🔴 HIGH

**File**: `backend/internal/connect/strategy/strategy_active_handlers.go:42-132`

**Problem**: Three RPC handlers — `GetActiveStrategy`, `StopStrategy`, and `WatchStrategySignals` — accepted a `run_id` parameter and operated on the corresponding session without verifying that the session belongs to the requesting user. Any authenticated user could:
- **Stop** any other user's running strategy by providing their `run_id`
- **View** another user's active strategy details (symbol, account, signals)
- **Stream** another user's real-time trading signals via SSE

This is a cross-user authorization bypass — a user can disrupt another user's live trading or observe their trading activity.

**Fix**: Added `sess.UserID != uid` ownership check in all three handlers. Returns `CodePermissionDenied` with a generic "not found" message (to avoid leaking existence of other users' sessions).

**Risk if unfixed**: Malicious user can stop competitors' strategies mid-trade, observe their signals for front-running, or cause financial loss by disrupting active positions.

### S7 — Double swap deduction in checkSLTP 🔴 HIGH

**File**: `backend/strategy/backtest/engine.go:310-311`

**Problem**: When a position is closed by Stop Loss or Take Profit in `checkSLTP`, the code calls `applySwap(pos, days)` which subtracts `pos.Swap` from both `equity` and `balance` (broker.go:473-474). Then the close logic at line 310-311 subtracts `pos.Swap` again:
```go
e.broker.equity = e.broker.equity.Add(pos.Profit).Sub(pos.Swap)  // ← second subtraction
e.broker.balance = e.broker.balance.Add(pos.Profit).Sub(pos.Swap)  // ← second subtraction
```

This means swap is deducted **twice** from equity and balance for every SL/TP close. Over a backtest with many trades, this compounds — making strategies appear less profitable than they actually are, with artificially lower equity curves and worse metrics (return, drawdown, Sharpe).

Note: `PositionClose` (manual close by strategy) does NOT have this bug — it only adds profit without subtracting swap (because `applySwap` already did). The inconsistency means SL/TP-closed trades are penalized more than manually-closed trades.

**Fix**: Removed the `.Sub(pos.Swap)` from the close logic in `checkSLTP`, matching the behavior of `PositionClose`.

**Risk if unfixed**: All backtests with overnight positions (where swap > 0) produce incorrect equity curves. Strategy evaluation is biased against strategies that hold positions overnight, potentially rejecting profitable strategies during auto-gate evaluation.

## Verified Safe (No Issues Found)

- **Risk gate enforcement**: `RunLiveStrategy` fails-stop if `s.gate == nil`; all orders pass through `MtHubService.PlaceOrder` (single chokepoint, D6-A)
- **Circuit breaker**: `IsCircuitOpen()` suppresses new orders after consecutive failures; `SetCircuitOpen(false)` resets on success
- **Session conflict detection**: `SessionRegistry.Register` atomically rejects duplicate account sessions under mutex lock
- **Stale run cleanup**: `CleanupStaleRuns` on startup marks orphaned runs as 'stopped' — crash recovery is correct
- **Bar ordering validation**: Engine validates chronological bar order before running (prevents silent garbage from unsorted data)
- **Future data leakage prevention**: Multi-symbol bar advancement stops at current timestamp; `BarsTF` aggregates only visible bars
- **Margin check**: `OrderSend` validates `equity >= margin` before opening positions
- **Commission handling**: Applied once at order open (not at close) — consistent modeling choice
- **Partial close**: `PositionClose` correctly reduces volume and records partial deals
- `PositionCloseBy`: Correctly handles opposite-side position netting with proper index ordering
- **Schedule engine**: Timer-driven (not polling); `reconcileOnStartup` fills missing `next_run_at`; `Stop()` waits for all goroutines
- **AutoTrade cache**: TTL-based cache (30s) prevents per-dispatch DB queries
- **Quota enforcement**: `StartStrategy` checks strategy count and live strategy limits before launching
- **Pre-created run records**: Both `StartStrategy` and `ScheduleEngine.dispatch` pre-create run records before launching, with proper cleanup on conflict
- **Fill model**: Backtest mode forces non-zero commission/slippage/spread defaults to prevent unrealistic results
- **Signal persistence**: `persistSignal` is best-effort (doesn't block dispatch on DB failure)
- **Context detachment**: Order dispatch uses `context.WithoutCancel(ctx)` to prevent parent cancellation from interrupting broker calls
- **Metrics calculation**: Correctly uses net profit (gross - commission - swap) for win/loss classification; Sharpe annualized by trades-per-year

## Architecture Compliance

- ✅ ConnectRPC + SSE only (no REST, no WebSocket)
- ✅ Push-first architecture (bar/tick/trade channels, no polling)
- ✅ No JSON serialization for data exchange
- ✅ decimal.Decimal for all price calculations (no float64 in financial logic)
- ✅ No hardcoded secrets
- ✅ No `//nolint` or `// @ts-ignore`

## Reuse Preflight

- **S6**: REUSE: `userIDRequire` @ `strategy_active_handlers.go` (added ownership check using existing session's UserID field)
- **S7**: REUSE: `applySwap` @ `broker.go:462` (removed duplicate subtraction, relying on existing swap application)

## Migrations

No migrations required.

## Deployment

- `go build ./...` ✅
- `go test ./strategy/backtest/... ./internal/connect/strategy/...` ✅
- `docker compose build backend` ✅
- `docker compose up -d backend` ✅
- Container health: `healthy` ✅

## Residual Risks

- **SL+TP same-bar ambiguity**: When both SL and TP are triggered within the same bar, the code uses the last-checked price (SL for BUY, SL for SELL). This is conservative (worst-case) but not necessarily accurate. This is a known modeling limitation of bar-based backtesting — resolving it would require tick-level data. Not a bug, but a documented limitation.
- **`dispatchCloseAll` closes by symbol, not magic**: If multiple strategies trade the same symbol on the same account (currently prevented by session registry's per-account lock), `close_all` would close all strategies' positions. Low risk given current architecture.
