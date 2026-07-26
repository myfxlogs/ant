# MT Gateway Audit Report

## Scope

Comprehensive review of the MT Gateway / mthub subsystem:
- `internal/mthub/service.go` — MtHubService facade, session management, market data routing
- `internal/mthub/service_orders.go` — PlaceOrder pipeline (kill switch, guard, ownership, idempotency, reconcile gate, rate limit, OMS, risk gate, broker submit)
- `internal/mthub/service_orders_close.go` — CloseOrder pipeline
- `internal/mthub/service_orders_delete.go` — DeleteOrder (cancel pending)
- `internal/mthub/service_orders_modify.go` — ModifyOrder (SL/TP modification)
- `internal/mthub/order_types.go` — OrderExecutor interface, order/symbol/bar types
- `internal/mthub/types.go` — Hub, Session, event brokers
- `internal/mthub/idempotency.go` — Three-layer idempotency (PG advisory lock, Redis SETNX, broker magic)
- `internal/mthub/state_cache.go` — In-memory OMS state + position cache with Redis backup
- `internal/mthub/derived_state.go` — Tier-2 derived quantities (PnL, exposure, VaR)
- `internal/mthub/reconciliation.go` — Event-driven reconciliation loop
- `internal/mthub/reconcile_gate.go` — Reconcile-before-accept gate
- `internal/mthub/oms_writer.go` — OMS 16-state machine writer
- `internal/mthub/trade_event_store.go` — NATS JetStream event store
- `internal/connect/system/mthub_service.go` — ConnectRPC handlers
- `internal/risk/gate.go` — Risk gate implementation

## Findings

### M1 — Order intents missing Source field, bypassing all Gate safety checks 🔴 CRITICAL

**File**: `backend/internal/mthub/service_orders.go:180` (PlaceOrder), `backend/internal/mthub/service_orders_close.go:58` (CloseOrder)

**Problem**: `orderRequestToIntent` constructs an `antv1.OrderIntent` but does not set the `Source` field. Proto3 defaults unset fields to zero value, so `Source` defaults to `ORDER_INTENT_SOURCE_UNSPECIFIED` (0).

The risk Gate (`gate.go:115-158`) evaluates three critical safety checks **only when `Source == ORDER_INTENT_SOURCE_LIVE`**:
- **R10 Kill Switch**: blocks all live orders when global kill switch is engaged
- **R11 Autotrade**: blocks live orders when per-user autotrade is disabled
- **D6-A Fail-Closed**: blocks live orders when `AccountState` is nil (provider not connected) or equity is negative (sentinel)

Because `Source` was `UNSPECIFIED` (not `LIVE`), **none of these checks fired** for any real order placed through mthub. This means:
1. A user with autotrade disabled could still place live orders
2. Orders could be placed even when the account state provider was disconnected
3. The global kill switch had no effect on mthub-routed orders

Additionally, `UserId` was not set on the intent, meaning even if R11 had fired, `autotrade("")` would not correctly identify the user.

**Fix**: Set `Source: antv1.OrderIntentSource_ORDER_INTENT_SOURCE_LIVE` and `UserId` from context in both PlaceOrder and CloseOrder intent construction.

```diff
// service_orders.go — orderRequestToIntent
+		Source:    antv1.OrderIntentSource_ORDER_INTENT_SOURCE_LIVE,
// service_orders.go — PlaceOrder gate section
+		intent.UserId = usermgr.GetUserID(ctx)
// service_orders_close.go — closeIntent
+		UserId:    usermgr.GetUserID(ctx),
+		Source:    antv1.OrderIntentSource_ORDER_INTENT_SOURCE_LIVE,
```

**Risk if unfixed**: Complete bypass of kill switch, autotrade enforcement, and fail-closed safety for all live trading through the platform. This is the most severe finding — it affects every order on every account.

### M2 — Nil logger panics in PlaceOrder error paths 🟡 MEDIUM

**File**: `backend/internal/mthub/service_orders.go:90,115,160,164,258`

**Problem**: Multiple `s.logger.Warn/Error` calls in PlaceOrder's gate evaluation, idempotency SetTicket, margin precheck, and event store publish paths are called without nil-checking `s.logger`. If `SetLogger` was not called (e.g., in test setups or partial wiring), these would panic with nil pointer dereference.

CloseOrder already had nil checks for its logger calls, but PlaceOrder did not.

**Fix**: Added `s.logger != nil` guards to all logger calls in PlaceOrder paths.

```diff
-		if stateErr != nil {
+		if stateErr != nil && s.logger != nil {
-		} else {
-			s.logger.Warn(...)
+		} else if s.logger != nil {
+			s.logger.Warn(...)
-	if err := s.eventStore.Publish(ctx, ev); err != nil {
+	if err := s.eventStore.Publish(ctx, ev); err != nil && s.logger != nil {
```

**Risk if unfixed**: Runtime panic (nil pointer dereference) if logger is not injected, crashing the order placement path.

## Verified Safe (No Issues Found)

- **Account ownership**: All ConnectRPC handlers (`PlaceOrder`, `CloseOrder`, `OpenedOrders`, `OrderHistory`, `SymbolParams`, `SymbolList`) call `validateAccountAccess` which checks `interceptor.GetUserID` + `platform.UserOwnsAccount`. No cross-user access possible.
- **Service-layer ownership**: `PlaceOrder`, `CloseOrder`, `DeleteOrder`, `ModifyOrder` all check `accountOwnerVerifier` when `userID != ""`.
- **Kill switch**: Checked first in all order operations (`killSwitch.IsEngaged()`).
- **Guard (3-rule safety net)**: Kill switch + duplicate protection + max lot size, checked before gate.
- **Idempotency (3-layer)**: PG advisory xact lock (auto-release on commit) + Redis SETNX with 96h TTL + broker magic hash. `CheckAndSet` → `SetTicket` flow correctly updates ticket after broker placement.
- **Close/Modify idempotency**: Uses `defer s.idem.DeleteKey` — intentional to allow retry after success (close is idempotent at broker level).
- **DeleteOrder idempotency**: Skipped — cancel is safe to retry (comment documents this).
- **Reconcile gate**: Blocks new orders during reconciliation. `EnterReconciling` on connect/reconnect, `MarkReconciled` after position pull.
- **Rate limiter**: Per-user `AllowOrder` check before broker submission.
- **OMS state machine**: 16-state with validated transitions. `Transition` uses `UPDATE ... WHERE state = $current` for optimistic concurrency. `RowsAffected() == 0` detects concurrent conflicts.
- **OMS insert idempotency**: `ON CONFLICT (id) DO NOTHING` with deterministic UUID from `IdempotencyKey(accountID, clientID)`.
- **Hub session management**: `sync.RWMutex` protected. `WaitSession` channel pattern eliminates polling. `EnsureSession` checks `IsExpired` (4h default).
- **Event store**: NATS JetStream with `Nats-Msg-Id` header for at-least-once dedup. UTF-8 sanitization for MT4 non-UTF-8 fields.
- **State cache**: `sync.RWMutex` protected. Terminal states evicted to prevent unbounded growth. Redis backup with 7-day TTL.
- **Position cache**: Correct net volume calculation with side multiplier. Weighted average price on position increase, unchanged on reduction.
- **Reconciliation**: Event-driven (no polling). Compares ticket existence only (state enums differ between OMS and broker). Ghost/orphan detection with logging.
- **Derived state**: 5s recalculation cycle. Parametric VaR with concentration factor. All decimal arithmetic.
- **Decimal precision**: All order/symbol/bar types use `decimal.Decimal`. No float64 in financial calculations.
- **ConnectRPC validation**: `PlaceOrder` validates volume positive, price/SL/TP parse as decimal. `CloseOrder` validates lots parse as decimal.
- **Trade event store**: Proto serialization with UTF-8 sanitization fallback. Trace header injection from context.

## Architecture Compliance

- ✅ No REST endpoints (ConnectRPC only)
- ✅ No WebSocket
- ✅ No JSON for data persistence (proto for NATS, PG for orders)
- ✅ `decimal.Decimal` for all financial calculations
- ✅ No `//nolint` or `// @ts-ignore`
- ✅ Push-first: event-driven reconciliation, SSE for order/account updates
- ✅ MT4/MT5 adapters not sharing code (via `OrderExecutor` interface abstraction)

## Reuse Preflight

- **M1**: REUSE: `antv1.OrderIntentSource_ORDER_INTENT_SOURCE_LIVE` @ `gen/proto/ant/v1/risk_gate.pb.go:31` (existing proto enum, was already used in tests and live_runner)
- **M2**: REUSE: `s.logger != nil` guard pattern @ `service_orders_close.go:67` (CloseOrder already had this pattern, PlaceOrder was inconsistent)

## Migrations

No migrations required.

## Deployment

- `go build ./...` ✅
- `go test ./internal/mthub/... ./internal/risk/...` ✅
- `docker compose build backend` ✅
- `docker compose up -d backend` ✅
- Container health: `healthy` ✅

## Residual Risks

- **CloseOrder idempotency delete-after-success**: The `defer s.idem.DeleteKey` pattern means a successful close can be retried (broker will reject "position already closed"). This is intentional — close is naturally idempotent at the broker level, and the delete allows retry after network failures where the broker succeeded but the response was lost.
- **ModifyOrder no gate**: Gate rules (MaxLotSize, MaxPositionCount, etc.) are designed for position opening. SL/TP modification doesn't change exposure/margin/concentration. Documented in code comment.
- **FNV64a → int32 magic collision**: 50% collision probability after ~77K unique clientIDs. Acceptable as Layer 3 (broker-side dedup) — primary protection is Layer 1+2. Documented in code.
- **Derived state VaR**: Parametric estimate (not historical simulation). Uses 1% daily vol assumption. Full historical VaR is M10-BASE-D7 (future work). Documented in code.
