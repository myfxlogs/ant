# Subscription & Billing Subsystem Audit Report

## Scope

- `internal/service/subscription_service.go` — Subscribe, ChangePlan, CancelSubscription, subscribeFree
- `internal/service/subscription_renewal.go` — Daily renewal loop, auto-renew, expiry
- `internal/service/subscription_service_proto.go` — Proto conversion, EnsureFreeSubscription, GetUsageSummary
- `internal/service/wallet_service.go` — WalletService: AdjustBalance, Freeze/Complete/CancelWithdrawal
- `internal/repository/subscription_repository.go` — Plan CRUD, subscription CRUD, FOR UPDATE locking
- `internal/repository/wallet_repo.go` — Wallet CRUD, AdjustBalanceTx, ledger chain (hash chain + idempotency)
- `internal/repository/wallet_freeze_repo.go` — Freeze/Complete/Cancel withdrawal operations
- `internal/connect/subscription/handler.go` — RPC handlers: ListPlans, Subscribe, Cancel, ChangePlan, GetUsageSummary
- `internal/model/subscription.go` — SubscriptionPlan, UserPlatformSubscription models

## Findings

### S1 — `subscribeFree` silently ignores plan lookup error, can downgrade paid users 🟡 MEDIUM

**File**: `subscription_service.go:181`

**Problem**: In `subscribeFree`, when checking if the user's existing subscription is already free, the code called `GetPlanByID` but silently ignored the error (`existingPlan, _ := ...`). If a transient DB error occurred during the plan lookup, `existingPlan` would be nil, and the code would fall through to deactivate the user's existing (potentially paid) subscription and create a new free one. This means a momentary DB blip could downgrade a paying user to the free tier.

**Attack scenario**: No attacker needed — a transient DB connection timeout during `subscribeFree` (called by `EnsureFreeSubscription` during registration, or by `Subscribe` when selecting the free plan) could cause a paid user's subscription to be deactivated.

**Fix**: Check the error and return it instead of proceeding:

```diff
-existingPlan, _ := s.repo.GetPlanByID(ctx, existing.PlanID)
+existingPlan, err := s.repo.GetPlanByID(ctx, existing.PlanID)
+if err != nil {
+    return nil, fmt.Errorf("subscription: get existing plan: %w", err)
+}
```

**Risk if unfixed**: Transient DB errors during free plan subscription could deactivate paid subscriptions.

## Verified Safe (No Issues Found)

### Subscription Lifecycle
- **Atomic subscribe**: `Subscribe` uses a single DB transaction: deactivate old + charge wallet + create new subscription. If any step fails, all roll back.
- **FOR UPDATE lock**: `GetActiveSubscriptionTx` uses `FOR UPDATE` to prevent concurrent subscribe/change-plan races.
- **Free plan no-charge**: `subscribeFree` skips wallet operations for free plans (price <= 0).
- **Free plan idempotent**: If user already on free plan, returns existing subscription without creating a new one.
- **Free plan period**: Set to 100 years (`now.AddDate(100, 0, 0)`) — effectively permanent. No renewal needed.
- **Cancel = no auto-renew**: `CancelSubscription` sets `auto_renew=false` and records `cancelled_at`. Subscription remains active until period end. No refund.
- **Plan validation**: `Subscribe` and `ChangePlan` validate plan exists and is active (`is_active = true` in query).
- **billingCycle validation**: Invalid values default to "monthly" (safe default).

### Proration (ChangePlan)
- **Prorated credit**: Remaining time ratio × old price = credit amount.
- **Net charge**: New price - proration credit. If positive, charge wallet. If negative, credit wallet.
- **Insufficient balance check**: Manual check before charge + DB CHECK constraint as backstop.
- **Period reset**: New period starts now, ends in 1 month or 1 year based on billing cycle.
- **No existing subscription**: Falls through to `Subscribe` (which creates new subscription).

### Auto-Renewal
- **Daily ticker**: Runs every 24h with 5-minute timeout per run.
- **Expiry first**: Non-auto-renew subscriptions with expired period are set to `expired` status.
- **Auto-renew second**: Auto-renew subscriptions are charged and period extended.
- **Renewal failure → expire**: If wallet charge fails (insufficient balance), subscription is expired (fail-safe).
- **Idempotent renewal**: Wallet charge uses `idem_key = "sub-renewal-"+subID.String()` — if renewal runs twice for the same period, the second charge is rejected as idempotent replay.
- **Free plan renewal**: Just extends the period without charging.

### Wallet Operations
- **Hash chain (R8)**: Every wallet transaction has `entry_hash = SHA256(prev_hash || seq || wallet_id || tx_type || amount || balance_before || balance_after || idem_key)`. Tamper-evident ledger.
- **Idempotency (R7)**: `idem_key` unique constraint prevents double-credit. `AdjustBalanceTx` checks idem_key before balance update.
- **Advisory lock (R8)**: `pg_advisory_xact_lock(20826)` serializes chain operations, preventing race on `prev_hash`.
- **Balance CHECK constraint**: DB enforces `balance >= 0` — `isCheckViolation` caught and returned as `ErrInsufficientBalance`.
- **FOR UPDATE lock**: `GetByUserIDTx` locks wallet row before balance update.
- **Outbox pattern**: `ledger_outbox` + `NOTIFY ledger_outbox` for external ledger mirror synchronization.
- **Credential change ledger**: `WriteCredentialChangeLedger` records password/whitelist changes in the hash chain with zero amount — tamper-evident audit trail.

### Withdrawal Flow (Freeze/Complete/Cancel)
- **Freeze**: `balance -= amount, frozen_balance += amount` atomically. `WHERE balance >= amount` prevents overdraft.
- **Complete**: `frozen_balance -= amount` only. Balance unchanged (funds already left on-chain).
- **Cancel**: `frozen_balance -= amount, balance += amount`. Funds returned to balance.
- **Undo on idempotent replay**: If `ledgerChainInsert` detects duplicate `idem_key`, the balance/frozen update is undone to prevent inconsistency.
- **Idempotent**: All three operations use unique `idem_key` per withdrawal.

### Access Control
- **User-scoped**: All subscription operations use authenticated `userID` from context.
- **No admin override**: No admin endpoint to directly modify subscriptions (by design — users self-manage).
- **Plan visibility**: `ListPlans` returns only `is_active = true` plans.

### Data Precision
- **decimal.Decimal throughout**: All price and balance calculations use `decimal.Decimal`. No float64.
- **NUMERIC columns**: `price_monthly`, `price_yearly`, `balance`, `frozen_balance`, `amount` all stored as PostgreSQL `NUMERIC`.
- **String passthrough**: Amounts passed as strings between layers, parsed by PostgreSQL.

## Architecture Compliance

- ✅ No REST endpoints (ConnectRPC only)
- ✅ No JSON for data persistence (PostgreSQL + proto)
- ✅ No float64 in price calculations (decimal.Decimal throughout)
- ✅ No `//nolint` or `// @ts-ignore`
- ✅ Hash chain ledger (R8) for tamper-evidence
- ✅ Idempotency keys (R7) for all wallet operations
- ✅ Advisory locks for chain serialization

## Reuse Preflight

- **S1 fix**: REUSE: `GetPlanByID` @ `subscription_repository.go:74` (error propagation, no new function)

## Deployment

- `go build ./...` ✅
- `go test ./internal/service/... ./internal/repository/... ./internal/connect/...` ✅
- `docker compose build backend` ✅
- `docker compose up -d backend` ✅
- Container health: `healthy` ✅

## Notes

- `ChangePlan` to the same plan will charge the user a net amount (new period price - proration credit). This is technically correct but surprising UX. A same-plan check could be added but is a product decision, not a security issue.
- `ChangePlan` silently ignores `decimal.NewFromString` errors for old/new prices (`oldPrice, _ := ...`). Since plan prices are admin-controlled and stored as NUMERIC in PostgreSQL, these will always be valid decimals. Theoretical issue only.
- Renewal loop runs in a single goroutine with 24h ticker. No concurrent execution risk in practice. The subscription row is not locked with `FOR UPDATE` during renewal, but the wallet is locked, and the idempotency key prevents double-charge.
