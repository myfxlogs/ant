# Sweep Subsystem Audit Report

## Scope

Comprehensive review of the fund consolidation (sweep) pipeline:
- `internal/sweep/builder.go` — Unsigned bundle construction (delegate, transfer, undelegate)
- `internal/sweep/batch_builder.go` — Batch unsigned bundle for multi-address sweep
- `internal/sweep/broadcaster.go` — Signed bundle broadcast with leg-by-leg confirmation
- `internal/sweep/state.go` — State machine: reconfirmation, stuck marking, double-spend prevention
- `internal/sweep/worker.go` — Periodic worker: crash recovery, stale expiry, broadcast resume
- `internal/sweep/admin.go` — Admin RPC entry points (export, import, dashboard, undelegate-only)
- `internal/sweep/repo.go` — Bundle persistence (proto marshal/unmarshal, status transitions)
- `internal/sweep/tron_client.go` — TRON gRPC client (build, broadcast, confirm, expiry)
- `internal/sweep/interfaces.go` — Interface abstractions for testability
- `internal/connect/user/sweep_handler.go` — ConnectRPC handlers for admin sweep operations
- `internal/repository/sweep_log_repo.go` — Sweep log DB operations

## Findings

### S1 — BroadcastingBundle.IsExpired hardcodes 23h, ignoring configurable RawTxExpiryHours 🟡 MEDIUM

**File**: `backend/internal/sweep/repo.go:247-249`

**Problem**: `IsExpired()` used a hardcoded 23h to check if a bundle's raw transactions have expired. However, `RawTxExpiryHours` is configurable via `sweep_raw_tx_expiry_hours` system config (default 23, but admin can set any value). If admin configures a longer expiry (e.g. 48h), the worker would prematurely mark the bundle as EXPIRED at 23h — while the raw transactions are still valid on chain for another 25h. This blocks broadcasting of still-valid transactions, stranding funds until a new bundle is built and cold-signed.

**Fix**: Changed `IsExpired` to accept `expiryHours int` parameter. Updated `resumeBroadcasting` in `worker.go` to load `sweep_raw_tx_expiry_hours` from config and pass it to `IsExpired`.

```diff
// repo.go
-func (b *BroadcastingBundle) IsExpired() bool {
-	expiryMs := b.BuiltAtMs + (23 * time.Hour).Milliseconds()
+func (b *BroadcastingBundle) IsExpired(expiryHours int) bool {
+	if expiryHours <= 0 {
+		expiryHours = 23
+	}
+	expiryMs := b.BuiltAtMs + (time.Duration(expiryHours) * time.Hour).Milliseconds()

// worker.go — resumeBroadcasting
+	rawTxExpiryHours := 23
+	if cfg, err := w.adminRepo.GetConfig(ctx, "sweep_raw_tx_expiry_hours"); ...
+		rawTxExpiryHours = n
+	}
-for _, bb := range bundles {
-	if bb.IsExpired() {
+for _, bb := range bundles {
+	if bb.IsExpired(rawTxExpiryHours) {
```

**Risk if unfixed**: Premature bundle expiration when admin configures longer raw_tx expiry. Funds stranded, require re-build + re-sign cycle.

### S2 — CheckDoubleSpend silently swallows DB error from GetLatestDoneTransferLeg 🟢 LOW

**File**: `backend/internal/sweep/state.go:148-149`

**Problem**: `GetLatestDoneTransferLeg` returns an error when the DB query fails, but the error was silently ignored (`if err == nil && doneLeg != nil`). If the DB is temporarily down, the function proceeds to the TronGrid chain check. If a previous sweep's outgoing transfer exists on chain (legitimate, already DONE), the function returns `true` (double-spend detected), blocking re-sweeping. This is fail-safe (blocks rather than allows), but the DB error was invisible to operators.

**Fix**: Added a `Warn` log when `GetLatestDoneTransferLeg` returns an error, so operators can diagnose DB issues. The fail-safe behavior (proceed to chain check) is preserved.

```diff
doneLeg, err := s.sweepRepo.GetLatestDoneTransferLeg(ctx, addrID)
+if err != nil {
+	s.log.Warn("sweep state: failed to query DONE transfer leg, proceeding to chain check",
+		zap.String("from", fromAddr),
+		zap.Error(err))
+}
if err == nil && doneLeg != nil {
```

**Risk if unfixed**: DB errors during double-spend check are invisible. Operators can't diagnose why re-sweeping is blocked.

## Verified Safe (No Issues Found)

### Architecture & Security
- **R1 watch-only**: Online server never holds private keys. `TronClient` only builds unsigned transactions (`BuildTRC20Transfer`, `BuildDelegateResource`, `BuildUndelegateResource`). Signing happens on air-gapped coldsign machine.
- **R6 single-instance**: Worker acquires PG advisory lock (`sweepAdvisoryLockID = 20260720`) before starting. Distinct from chain.Monitor's lock ID.
- **Admin-only RPCs**: All sweep handlers (`ExportUnsignedSweepBundle`, `ExportBatchUnsignedSweepBundle`, `ImportSignedSweepBundle`, `GetSweepDashboard`, `BuildUndelegateOnlyBundle`, `ListPendingSignBundles`) call `requireAdmin()` which fail-closes if `platformSvc` is nil.
- **Xpub fingerprint verification**: `ImportXpub` checks fingerprint match to prevent key substitution. Fail-closed on mismatch.

### Double-Spend Prevention (ADR §2.7)
- **CheckDoubleSpend**: Verifies no unaccounted outgoing TRC20 transfer before re-sweep. If DONE transfer leg exists, outgoing is expected (previous sweep) — doesn't block.
- **F5 admin override**: `sweep_skip_doublecheck=true` bypasses TronGrid check during outages. Logged at WARN level.
- **Broadcaster re-broadcast guard**: SWEEPING/FAILED legs with tx_hash are chain-checked before re-broadcast. If confirmed on chain, marked DONE or MANUAL_REVIEW. If not found, safe to re-broadcast.
- **MANUAL_REVIEW halt**: Broadcaster never re-broadcasts MANUAL_REVIEW legs — funds may have moved. Returns `ErrManualReview` to stop retry cycle.
- **D13 txid mismatch**: If broadcast returns a different txid than expected, DB is updated with the actual txid for correct chain reconfirmation.

### State Machine
- **ReconfirmSweeping**: Checks SWEEPING and FAILED legs with tx_hash against chain. SUCCESS → DONE, FAILED → MANUAL_REVIEW, unconfirmed → stays.
- **MarkStuckSweeping**: SWEEPING+tx_hash → MANUAL_REVIEW (funds may have moved). SWEEPING without tx_hash → FAILED (stuck before broadcast). PENDING → FAILED (24h timeout matching raw_tx expiry).
- **D14 MANUAL_REVIEW stop**: When broadcaster encounters MANUAL_REVIEW leg, bundle is marked MANUAL_REVIEW to stop 30s retry cycle.

### Bundle Persistence & Crash Recovery (Q3)
- **D4 TOCTOU prevention**: `saveBatchBundleAndLegs` checks PENDING_SIGN inside DB transaction — eliminates race between two concurrent admin RPCs.
- **Atomic save**: Sweep log legs + unsigned bundle saved in single transaction. Rollback on any error.
- **ON CONFLICT handling**: `SaveUnsignedBundle` uses `ON CONFLICT (batch_id) DO UPDATE SET unsigned_bundle` — allows re-export. `SaveBundle` uses `ON CONFLICT DO UPDATE SET signed_bundle, status='BROADCASTING'` — transitions PENDING_SIGN → BROADCASTING. `built_at_ms` preserved on conflict (not in UPDATE SET), so `IsExpired` uses original build time.
- **Crash recovery**: Worker's `resumeBroadcasting` reads BROADCASTING bundles and continues from first unconfirmed leg. Re-broadcast needs no private key.
- **Stale PENDING_SIGN expiry**: `expireStalePendingSign` marks bundles older than 24h as EXPIRED and fails their legs, freeing addresses.

### Transaction Building
- **decimal.Decimal for energy calculation**: `calculateEnergy` uses `decimal.Decimal` for DEM factor and buffer percent. `energyToTRX` uses `decimal.Decimal` for rate conversion.
- **USDT 6 decimals**: `amountSmallest = amountDec.Mul(1_000_000).BigInt()` — correct conversion from human-readable to smallest unit.
- **Configurable fee limit**: Default 1000 TRX (1B SUN), configurable via `sweep_fee_limit`.
- **Configurable energy**: First sweep 130k, subsequent 65k, with DEM factor and buffer. All configurable.
- **Expiry set on all 3 legs**: `SetTxExpiry` called on delegate, transfer, and undelegate raw transactions.

### Batch Operations
- **Batch bundle**: 3N txs (delegate, transfer, undelegate per address). Operational benefit: one USB round-trip for cold signing.
- **Pre-filter**: D4 pre-filter checks PENDING_SIGN before building txs (avoids unnecessary Tron gRPC calls). Authoritative check inside transaction.
- **Double-spend filter**: Each address checked before inclusion in batch.

### Undelegate-Only Recovery (C5)
- **No D4 check**: Undelegate is chain-idempotent (second undelegate fails harmlessly) and doesn't transfer USDT, so duplicate bundles pose no double-spend risk.
- **Energy recovery**: Used when delegate succeeded but transfer failed — operator can reclaim frozen TRX without re-sweeping USDT.

### TronClient
- **Nil checks**: All build methods check `ext.Transaction == nil || ext.RawData == nil` before proceeding.
- **Broadcast validation**: Checks `result.Code != 0` and returns error with message.
- **GetTransactionInfo**: Returns `confirmed=false` with nil error for "not found" — correct for unconfirmed transactions.
- **WaitForConfirmation**: Uses `context.WithTimeout` for per-leg confirmation timeout. Configurable poll interval.

## Architecture Compliance

- ✅ No REST endpoints (ConnectRPC only)
- ✅ No JSON for data persistence (proto marshal for bundles, PostgreSQL for state)
- ✅ `decimal.Decimal` for energy and amount calculations
- ✅ No `//nolint` or `// @ts-ignore`
- ✅ Push-first: event-driven worker cycle, no polling for external state (TronGrid check is on-demand, not polling)
- ✅ Fail-closed: `requireAdmin` fails when `platformSvc` is nil
- ✅ Cold signing architecture: online server is watch-only (R1)

## Reuse Preflight

- **S1**: REUSE: `loadConfig` pattern @ `builder.go:61-120` (already reads `sweep_raw_tx_expiry_hours`)
- **S2**: REUSE: `s.log.Warn` pattern @ `state.go:62-65` (already used for reconfirm errors)

## Migrations

No migrations required.

## Deployment

- `go build ./...` ✅
- `go test ./internal/sweep/...` ✅
- `docker compose build backend` ✅
- `docker compose up -d backend` ✅
- Container health: `healthy` ✅

## Residual Risks

- **Worker uses `time.Ticker` (30s scan interval)**: This is acceptable — the worker is a periodic background recovery loop, not a latency-sensitive data feed. The ticker is for crash recovery and stuck-leg detection, not for real-time data distribution. No push-based alternative exists for "check if DB has stuck legs."
- **TronGrid dependency for double-spend check**: If TronGrid is unavailable, `CheckDoubleSpend` returns an error and the sweep is blocked. This is fail-safe. F5 admin override exists for extended outages.
- **`SaveBundle` INSERT path doesn't set `deposit_address_id`**: If a signed bundle is imported without a prior unsigned bundle (shouldn't happen in normal flow), `deposit_address_id` would be NULL. The `ListPendingSignBundlesForAdmin` query handles NULL via `*uuid.UUID` scan.
- **Batch bundle `firstAddrID` for compatibility**: `saveBatchBundleAndLegs` passes `entries[0].Addr.ID` as `deposit_address_id` for the bundle row. This is a compatibility hack — the bundle actually covers multiple addresses. The per-address legs have correct `deposit_address_id` values.
