# Marketplace Subsystem Audit Report

## Scope

- `internal/marketplace/publish.go` — Publish, Unpublish, ListPublished, GetPublisherStats
- `internal/marketplace/purchase.go` — PurchaseStrategy, SetPricing, lookupExistingPurchase
- `internal/marketplace/bundle_purchase.go` — PurchaseBundle, DeleteBundle
- `internal/marketplace/service_subscription.go` — Subscribe, Unsubscribe, RenewSubscriptions, StartRenewalLoop, CanAccessCode
- `internal/marketplace/refund.go` — RefundPurchase, refundPurchaseTx
- `internal/marketplace/refund_request.go` — CreateRefundRequest, ProcessRefundRequest, ListRefundRequests
- `internal/marketplace/settlement.go` — SettleExpired, retryFailedReversals, createFrozenSettlementTx
- `internal/marketplace/trial.go` — StartTrial, HasActiveTrial
- `internal/marketplace/coupon.go` — ValidateCoupon, validateCouponTx, CreateCoupon, ListCoupons, DisableCoupon
- `internal/marketplace/types.go` — Service struct, constants, idem key prefixes
- `internal/connect/marketplace/marketplace_handler.go` — Server struct, checkAdmin
- `internal/connect/marketplace/marketplace_handler_subs.go` — PublishStrategy, Subscribe, Unsubscribe, PurchaseStrategy, ListPublished
- `internal/connect/marketplace/marketplace_handler_refund.go` — RequestRefund, AdminListRefundRequests, AdminProcessRefund
- `internal/connect/marketplace/marketplace_handler_social.go` — SetStrategyPricing, UnpublishStrategy, GetPublisherStats

## Findings

### M1 — `Publish` doesn't verify strategy template ownership 🔴 CRITICAL

**File**: `publish.go:182-261`

**Problem**: The `Publish` method accepts a `StrategyID` and `UserID` but never verifies that the user actually owns the strategy template with that ID. Any authenticated user can publish any other user's strategy to the marketplace, potentially selling another user's strategy under their own name and collecting revenue.

**Attack scenario**: User A has a high-quality strategy template. User B knows the strategy ID (e.g., from API responses, shared backtest results, or guessing). User B calls `PublishStrategy` with User A's `strategy_id`. The strategy is published to the marketplace with User B as the publisher. When buyers purchase it, User B receives the revenue, not User A.

**Fix**: Added ownership check before the transaction — query `strategy_templates` to verify `user_id` matches `params.UserID`:

```go
var templateOwnerID string
err = s.pg.QueryRow(ctx,
    `SELECT user_id::text FROM strategy_templates WHERE id = $1`,
    params.StrategyID,
).Scan(&templateOwnerID)
if err != nil {
    return "", fmt.Errorf("marketplace: strategy template not found: %w", err)
}
if templateOwnerID != params.UserID {
    return "", fmt.Errorf("marketplace: not the strategy owner")
}
```

**Risk if unfixed**: Strategy theft — any user can sell any other user's strategies.

### M2 — `retryFailedReversals` uses different idem keys, causing double-debit 🟡 MEDIUM

**File**: `settlement.go:166-290`

**Problem**: When a refund reversal partially fails (e.g., publisher debit fails but platform fee debit succeeds), the settlement is marked with `reversal_failed = true`. The `retryFailedReversals` method retries both debits with **different idem keys** (`IdemKeyRevRetry`/`IdemKeyFeeRevRetry` vs original `IdemKeyRev`/`IdemKeyFeeRev`). Since the idem keys are different, the already-succeeded platform fee debit is not detected as idempotent replay — it gets debited again, causing a **double-debit** of the platform fee wallet.

**Attack scenario**: No attacker needed — occurs naturally when:
1. Buyer refunds a purchase after settlement (settlement status = 'settled')
2. Publisher wallet has insufficient balance for reversal debit
3. Platform fee reversal succeeds (system wallet always has balance)
4. Settlement marked `reversal_failed = true`
5. Next `SettleExpired` call triggers `retryFailedReversals`
6. Publisher debit retries with new idem key → succeeds (if publisher now has balance)
7. Platform fee debit retries with new idem key → **debited again** (double-debit)

**Fix**: Select `purchase_id` from the settlement row and use the **original idem keys** (`IdemKeyRev+purchaseID`, `IdemKeyFeeRev+purchaseID`) in the retry. This way:
- Already-succeeded debits are rejected as `ErrIdempotentReplay` (no double-debit)
- Failed debits succeed (idem key was never persisted because the original transaction was rolled back)

**Risk if unfixed**: Platform fee wallet slowly drained by duplicate debits. Publisher wallet unaffected (publisher debit failed originally, so no prior transaction exists).

## Verified Safe (No Issues Found)

### Purchase Flow
- **Atomic purchase**: `PurchaseStrategy` uses single DB transaction: idempotency check → strategy lookup → wallet charge → subscription insert → frozen settlement → subscriber count. All roll back on any failure.
- **Idempotency**: `idempotencyKey` checked inside transaction. Unique constraint on `idempotency_key` prevents duplicate purchases. Race fallback via `lookupExistingPurchase`.
- **Self-purchase guard**: `uid == pid` check prevents buying own strategy.
- **Existing subscription guard**: Checks for active subscription before creating new one.
- **Strategy status**: Only `status = 'published'` strategies can be purchased.
- **Price validation**: `priceModel` must be `once` or `subscription`, `priceDec.IsPositive()` required.
- **Coupon validation**: `validateCouponTx` uses `FOR UPDATE` lock, preventing TOCTOU race. Coupon consumption is atomic with purchase.
- **Coupon edge cases**: Percentage capped at 100%, fixed discount capped at purchase amount, final amount floored at 0.

### Bundle Purchase
- **Atomic**: Single transaction for bundle lookup, wallet charge, all subscription inserts, settlement, counter increments.
- **Proration**: `effectivePrice = bundlePrice * (remainingCount / totalCount)` — users only pay for strategies they don't already own.
- **Idempotency**: Checked via wallet transaction idem_key and subscription idempotency_key prefix.
- **Self-purchase guard**: `uid == publisherID` check.
- **Bundle lock**: `FOR UPDATE` on bundle row prevents TOCTOU.
- **Settlement**: Uses `bundle_id` for refund lookup compatibility.

### Refund Flow
- **Atomic**: `refundPurchaseTx` runs in single transaction: lock subscription → check active schedules → refund buyer → handle settlement → deactivate subscription → decrement counter.
- **Active schedule guard (I3)**: Rejects refund if buyer has active live schedules for the strategy.
- **Settlement handling**: Frozen → mark refunded (no wallet debits). Settled → reverse publisher + platform credits. Already refunded → error.
- **Partial reversal tracking**: `reversal_failed` flag + `reversal_failure_note` for reconciliation.
- **FOR UPDATE**: Subscription and settlement rows locked.
- **Idempotency**: `IdemKeyRefund+sid` prevents double-refund.

### Refund Request Flow
- **Refund window**: Publisher-configurable `refund_window_days` from settlement row, default 7 days.
- **Duplicate prevention**: Atomic check-then-insert with `WHERE NOT EXISTS` prevents concurrent duplicate pending requests.
- **Admin approval**: `ProcessRefundRequest` requires admin, checks `status = 'pending'`, uses `FOR UPDATE`.
- **Atomic execution**: Refund execution + status update in same transaction.

### Settlement (Lazy)
- **No polling**: Triggered lazily by publisher dashboard views, earnings queries, or new purchases.
- **SKIP LOCKED**: `FOR UPDATE SKIP LOCKED` prevents concurrent settlement of the same row.
- **Idempotent**: Each settlement credit uses `IdemKeySettle+settlementID` — duplicate settlement attempts are rejected.
- **Per-settlement credits**: Each settlement is credited individually for hash chain integrity.
- **Error resilience**: Failed credits are logged and skipped; remaining settlements in batch still process.

### Subscription Renewal
- **Daily ticker**: 24h interval with midnight alignment + jitter. 5-minute timeout per run.
- **Single goroutine**: No concurrent renewal risk.
- **Idempotent charge**: `IdemKeyRenewBuy+subID` prevents double-charge.
- **Fail-safe**: Insufficient balance → subscription deactivated (not retried forever).
- **Tiered fee rate**: Uses `getEffectiveFeeRateTx` for current tier, not stale publish-time rate.
- **Frozen settlement**: Renewal creates frozen settlement, same as initial purchase.

### Trial
- **One per user per strategy**: `ON CONFLICT (user_id, strategy_id) DO UPDATE` returns existing trial.
- **Published + paid only**: Free strategies don't offer trials.
- **Publisher-configurable**: `trial_days` from DB, default 7.
- **Lazy expiry**: Expired trials marked as expired on read.

### Access Control
- **PublishStrategy**: Authenticated user from context. Now verifies template ownership (M1 fix).
- **SetStrategyPricing**: Admin-only at handler level, publisher-only at service level.
- **UnpublishStrategy**: Publisher or admin.
- **PurchaseStrategy**: Authenticated user.
- **RequestRefund**: Authenticated user, validates subscription ownership.
- **AdminListRefundRequests / AdminProcessRefund**: Admin-only via `checkAdmin`.
- **CreateCoupon / DisableCoupon**: Admin-only (handler-level check, service-level adminID validation).

### Data Precision
- **decimal.Decimal throughout**: All price, fee, and amount calculations use `decimal.Decimal`. No float64.
- **NUMERIC columns**: `price_amount`, `provider_amount`, `platform_fee`, `amount` all stored as PostgreSQL `NUMERIC`.
- **StringFixed(2)**: Amounts formatted to 2 decimal places for wallet operations.

## Architecture Compliance

- ✅ No REST endpoints (ConnectRPC only)
- ✅ No JSON for data persistence (PostgreSQL + proto)
- ✅ No float64 in price calculations (decimal.Decimal throughout)
- ✅ No `//nolint` or `// @ts-ignore`
- ✅ Hash chain ledger for all wallet operations
- ✅ Idempotency keys for all financial operations
- ✅ Push-first: lazy settlement (no polling/cron), notification triggers on events

## Reuse Preflight

- **M1 fix**: NEW: ownership check query (no existing `strategy_templates` ownership check in marketplace package)
- **M2 fix**: REUSE: `IdemKeyRev`/`IdemKeyFeeRev` constants @ `types.go:136-137` (changed retry to use original keys instead of `IdemKeyRevRetry`/`IdemKeyFeeRevRetry`)

## Deployment

- `go build ./...` ✅
- `go test ./internal/marketplace/... ./internal/connect/marketplace/...` ✅
- `docker compose build backend` ✅
- `docker compose up -d backend` ✅
- Container health: `healthy` ✅

## Notes

- `SetStrategyPricing` handler requires admin, but `SetPricing` service method requires publisher ownership. Effective policy: only an admin who is also the publisher can set pricing. This may be intentional (admin oversight) or a design issue, but not a security vulnerability.
- `Subscribe` for non-published strategies uses client-supplied `publisherUserID` as fallback. Limited risk since free strategies don't involve money, and self-subscribe is blocked.
- `bundle_purchase.go:208`: `sid, _ := uuid.Parse(sidStr)` silently ignores parse error. Theoretical only — strategy IDs from `marketplace_bundle_items` are always valid UUIDs.
- `IdemKeyRevRetry`/`IdemKeyFeeRevRetry` constants in `types.go:138-139` are now unused. Kept for backward compatibility with any existing wallet transactions that may have used them.
