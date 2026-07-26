# Marketplace Audit — Risk Review (M1-M12)

**Date**: 2026-07-26  
**Scope**: `backend/internal/marketplace/` + `backend/internal/connect/marketplace/`  
**Commit**: `88a7351c`

---

## Risk Summary

| ID | Severity | Impact | Trigger | Consequence | Fix |
|----|----------|--------|---------|-------------|-----|
| M1 | P0 | All purchases | Any unique constraint violation | False positive error masking real DB errors | `pgconn.PgError` SQLSTATE 23505 |
| M2 | P0 | Re-subscribe flow | User unsubscribes then re-subscribes | `INSERT` fails on unique constraint, user locked out | `ON CONFLICT DO UPDATE SET active=true` |
| M3 | P1 | Pricing security | Non-owner calls `SetPricing` | Any user can change any strategy's pricing | Ownership check in service layer |
| M4 | P1 | Idempotency | Empty string idempotency key | Partial unique index collision on empty string | Convert empty → `NULL` |
| M5 | P1 | Subscriber count | `Subscribe` called for marketplace strategy | Counter never increments, metrics wrong | `total_subscribers + 1` in tx |
| M6 | P1 | Bundle refund | Refund non-first subscription in bundle | Settlement not found, refund skips settlement reversal | `bundle_id` column + fallback lookup |
| M7 | P1 | Renewal fees | Publisher qualifies for lower tier | Renewal charges wrong (higher) fee rate | `getEffectiveFeeRateTx` in renewal |
| M8 | P2 | Query performance | `ListSubscriptions` / `CanAccessCode` | `::text` cast bypasses UUID index, full table scan | Direct UUID comparison |
| M9 | P2 | Reconciliation | Publisher balance insufficient for reversal | Failed reversal silently swallowed, no audit trail | `reversal_failed` + `reversal_failure_note` columns |
| M10 | P2 | API contract | `AdminProcessRefund` error | Returns HTTP 200 with `Success=false`, client can't distinguish | `connect.NewError(CodeInternal, err)` |
| M11 | P2 | Bundle overcharge | User already owns some bundle strategies | Full bundle price charged for duplicate subscriptions | Filter existing subs before charge |
| M12 | P2 | Server stability | Invalid/empty userID in admin handler | `uuid.MustParse` panics, server crashes | `checkAdmin` helper with `uuid.Parse` |

---

## Architecture Compliance

- **No REST endpoints added** — all fixes within existing ConnectRPC handlers
- **No WebSocket** — unchanged
- **No JSON serialization** — all data via proto/SQL
- **No float64 in price calculations** — all use `decimal.Decimal`
- **No `//nolint` / `@ts-ignore`** — clean
- **Push-first preserved** — no polling added
- **MT adapters untouched** — no cross-adapter sharing

## Reuse Preflight

- `REUSE: getEffectiveFeeRateTx @ fee_tier.go:143` (M7)
- `REUSE: isUniqueViolation pattern @ wallet_repo.go:177` (M1)
- `REUSE: walletRepo.AdjustBalanceTx @ wallet_repo.go` (M9)
- `NEW: checkAdmin helper @ marketplace_handler.go:105` (M12) — no existing safe admin check helper found

## Migrations

| Migration | Purpose | Reversible |
|-----------|---------|------------|
| `240_settlement_bundle_id` | Add `bundle_id UUID` to `marketplace_settlements` | ✅ down.sql |
| `241_settlement_reversal_tracking` | Add `reversal_failed BOOLEAN` + `reversal_failure_note TEXT` | ✅ down.sql |

## Regression Test Results

```
go build ./...                    — PASS
go test ./internal/marketplace/   — PASS (0.019s)
go test ./internal/connect/marketplace/ — PASS (0.040s)
check-file-lines --strict         — 0 🔴, 28 🟡 (all pre-existing)
docker compose build backend      — PASS
docker compose up -d backend      — PASS (container started)
```

## Residual Risk — All Resolved (R1-R3)

- ~~**M6 bundle_id lookup**: Uses `LIKE` prefix match~~ → **R1 Fixed**: Added `bundle_id UUID` column to `user_subscriptions` (migration 242). Refund now uses exact `WHERE bundle_id = $1` lookup instead of fragile LIKE prefix matching.
- ~~**M11 prorate**: Charges full price for remaining strategies~~ → **R2 Fixed**: Bundle price is now prorated as `bundlePrice * (remainingCount / totalCount)` using `decimal.Decimal` arithmetic. Buyer only pays for strategies they don't already own.
- ~~**M9 reversal tracking**: Records failure but does not retry~~ → **R3 Fixed**: Added `retryFailedReversals()` method called lazily from `SettleExpired()`. Retries failed publisher/platform debits with new idempotency keys (`IdemKeyRevRetry`/`IdemKeyFeeRevRetry`). Clears `reversal_failed` flag on success. No cron/timer needed — piggybacks on existing lazy settlement trigger.

### Additional Migrations (R1-R3)

| Migration | Purpose | Reversible |
|-----------|---------|------------|
| `242_subscriptions_bundle_id` | Add `bundle_id UUID` to `user_subscriptions` for exact settlement lookup | ✅ down.sql |
