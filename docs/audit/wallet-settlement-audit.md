# Wallet & Settlement Audit Report

## Scope

Comprehensive review of the Wallet & Settlement subsystem:
- Wallet repository (balance updates, idempotency, hash chain, advisory locks)
- Wallet freeze/unfreeze operations (withdrawal lifecycle)
- Wallet service layer (transaction management)
- Deposit service (on-chain deposit crediting)
- Withdrawal builder (unsigned bundle construction)
- Withdrawal repository (request lifecycle, whitelist)
- Sweep worker (batch lifecycle, crash recovery, broadcasting)
- Sweep repository (bundle persistence)
- Sweep broadcaster (chain confirmation, retry)
- Reconciliation (internal + on-chain balance verification)
- Wallet handler (ConnectRPC API)

## Findings

### W1 — freezeOp missing undo on idempotent replay 🟡 MEDIUM

**File**: `backend/internal/repository/wallet_freeze_repo.go:92-97`

**Problem**: `freezeOp` updates `balance`/`frozen_balance` first, then calls `ledgerChainInsert` for hash chain + idempotency. If `ledgerChainInsert` returns `ErrIdempotentReplay` (concurrent race on same `idemKey`), the balance/frozen changes were **not rolled back** — inconsistent with `AdjustBalanceTx` which has undo logic for the same scenario.

**Fix**: Added undo logic matching `AdjustBalanceTx`'s pattern — reverses the balance/frozen changes based on operation type (freeze/cancel/complete) before returning the pre-update wallet state.

**Risk if unfixed**: Under concurrent idempotent replay, wallet balance/frozen could be double-modified. The caller's `defer tx.Rollback()` would eventually undo it, but the returned wallet state would be incorrect.

### W2 — WithdrawalBuilder non-atomic bundle save + withdrawal link 🟡 MEDIUM

**File**: `backend/internal/service/withdrawal_builder.go:186-192`

**Problem**: `SaveUnsignedBundle` and `UpdateWithdrawalBundle` were executed as separate non-transactional operations. If the first succeeded but the second failed, an orphaned `PENDING_SIGN` bundle would exist with no linked withdrawal — coldsign could sign it and broadcast a transfer with no corresponding withdrawal request.

**Fix**: Wrapped both operations in a single DB transaction via `saveBundleAndLinkWithdrawal`. Added `SaveUnsignedBundleTx` to `BundleSaver` interface and `BundleRepository`.

**Risk if unfixed**: Orphaned sweep bundles that could be signed and broadcast without a corresponding withdrawal request.

### W3 — Admin AdjustBalance txType misclassification 🟢 LOW

**File**: `backend/internal/connect/user/wallet_handler.go:146-148`

**Problem**: Negative admin adjustments were classified as `tx_type = "withdrawal"` instead of `tx_type = "adjustment"`. This pollutes reconciliation queries that filter by `tx_type = 'deposit'` vs other types, and misclassifies admin manual operations as user withdrawals.

**Fix**: Always use `tx_type = "adjustment"` for admin balance adjustments regardless of sign.

**Risk if unfixed**: Reconciliation categorization errors; admin debits counted as withdrawals in analytics.

## Verified Safe (No Issues Found)

- **Hash chain integrity**: Global `pg_advisory_xact_lock` serializes chain appends; `ORDER BY seq DESC LIMIT 1` correctly fetches previous hash; `computeEntryHash` includes all fields.
- **Idempotency (R7)**: Dual-check pattern (pre-check + unique constraint) in both `AdjustBalanceTx` and `ledgerChainInsert`; `ErrIdempotentReplay` handled gracefully by callers.
- **Row-level locking**: `FOR UPDATE` on `user_wallets` prevents concurrent balance modifications.
- **Balance CHECK constraint**: `balance >= 0` enforced at DB level.
- **Deposit crediting**: `ON CONFLICT (tx_hash) DO NOTHING` on deposit INSERT + idem_key `deposit-{txHash}` on wallet credit = fully idempotent.
- **Sweep worker lifecycle**: PostgreSQL advisory lock ensures single-instance; D4 TOCTOU check inside transaction prevents duplicate `PENDING_SIGN` bundles; stale bundle expiry + `MANUAL_REVIEW` for confirmed failures.
- **Broadcaster ordering**: delegate → confirm → transfer → confirm → undelegate → confirm; per-leg status tracking in `sweep_logs` for crash recovery.
- **Reconciliation**: `decimal.Decimal` throughout (no float64); asymmetric thresholds (shortage vs surplus); includes cold wallet + all derived addresses (ASSIGNED + RETIRED).
- **Withdrawal whitelist**: 24h cooldown before activation; credential change log for audit trail.
- **Xpub security**: Compromised checker blocks address derivation on audit mismatch; only xpub held online (no private keys).

## Architecture Compliance

- ✅ ConnectRPC + SSE only (no REST, no WebSocket)
- ✅ `decimal.Decimal` for all monetary calculations (no float64)
- ✅ PostgreSQL for persistence (no JSON serialization)
- ✅ Proto for data exchange
- ✅ Push-first: no polling in reconcile (uses `time.Ticker` which is acceptable for periodic batch reconciliation, not latency-sensitive)
- ✅ Advisory locks for concurrency control

## Reuse Preflight

- **W1**: REUSE: `freezeOp` @ `wallet_freeze_repo.go:38` (extended existing function)
- **W2**: REUSE: `SaveUnsignedBundleTx` @ `sweep/repo.go:34` (refactored from `SaveUnsignedBundle`); REUSE: `repository.NewWithdrawalRepository` @ `withdrawal_repo.go:20`
- **W3**: No new code — removed misclassification

## Migrations

No migrations required for W1-W3.

## Deployment

- `docker compose build backend` ✅
- `docker compose up -d backend` ✅
- Container health: `healthy` ✅
- `go build ./...` ✅
- `check-file-lines --strict`: 0 errors ✅

## Residual Risks

None — all three findings addressed. The remaining system is robust:
- Hash chain + advisory locks provide tamper-evidence and serialization
- Idempotency keys prevent double-crediting/debiting
- Atomic transactions ensure consistency
- Reconciliation provides ongoing monitoring
