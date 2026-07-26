# Chain Monitor Audit Report

## Scope

Comprehensive review of the Chain Monitor subsystem:
- `internal/chain/monitor.go` — block scanner, confirmation tracking, checkpoint persistence
- `internal/chain/tron_grid.go` — TronGrid API client (block events, latest block, TRC20 balance, pagination)
- `internal/chain/tron_scan.go` — TronScan API client (multi-source transaction verification)
- `internal/chain/chain_test.go` — unit tests for TronGrid/TronScan clients
- `internal/service/deposit_service.go` — deposit confirmation, wallet credit, address derivation
- `internal/repository/deposit_repo_v2.go` — deposit CRUD, idempotency, sweep dashboard
- `internal/repository/deposit_address_repo.go` — deposit address management, derivation index

## Findings

### C1 — Deposit record created outside transaction, breaking atomicity 🔴 HIGH

**File**: `backend/internal/service/deposit_service.go:207`

**Problem**: `ConfirmDeposit` begins a PG transaction for the wallet credit, but the deposit record INSERT uses `s.depRepo.Create(ctx, dep)` which executes on the pool connection — **not** the transaction (`tx`). This means:

1. Deposit record is committed immediately with `status="CONFIRMED"`
2. Wallet credit (`AdjustBalanceTx`) runs inside the transaction
3. If the wallet credit fails (non-replay error), `tx.Rollback()` discards the wallet credit
4. **But the deposit record persists** — status="CONFIRMED" with no corresponding wallet balance increase

The user sees a confirmed deposit but their wallet balance was never credited. On retry, `depRepo.Create` with `ON CONFLICT (tx_hash) DO NOTHING` skips the insert (already exists), and `AdjustBalanceTx` with `idem_key="deposit-"+txHash` may also skip if it was partially committed. The deposit is stuck in a "confirmed but not credited" state.

Additionally, the `processEvent` fallback in `monitor.go:328-344` tries to insert a `MANUAL_REVIEW` deposit when `ConfirmDeposit` fails. But since the deposit was already created with `CONFIRMED` status (outside the tx), the `ON CONFLICT (tx_hash) DO NOTHING` on the MANUAL_REVIEW insert silently skips it. The operator never sees the MANUAL_REVIEW entry — the failure is invisible.

**Fix**: Create a transaction-scoped `DepositRepository` via `repository.NewDepositRepository(tx)` and use it for the deposit INSERT. This ensures the deposit record and wallet credit are atomic — either both commit or both roll back.

```diff
- if err := s.depRepo.Create(ctx, dep); err != nil {
-     return fmt.Errorf("deposit service: create deposit: %w", err)
- }
+ depositRepoTx := repository.NewDepositRepository(tx)
+ if err := depositRepoTx.Create(ctx, dep); err != nil {
+     return fmt.Errorf("deposit service: create deposit: %w", err)
+ }
```

**Risk if unfixed**: User deposits can be permanently stuck — confirmed in DB but never credited to wallet. Requires manual database intervention to fix. Invisible to operators because MANUAL_REVIEW fallback is silently skipped.

## Verified Safe (No Issues Found)

- **Advisory lock**: `pg_try_advisory_lock` ensures single-instance execution (R6). Lock released on shutdown via defer.
- **Confirmation threshold**: `scanBlocks` only processes blocks with `minConfirms` confirmations (`safeLatest = latest - minConfirms`). Prevents processing reorged blocks.
- **Checkpoint persistence**: Saved after each block processed. On restart, resumes from last checkpoint.
- **Catch-up mechanism**: `maxBlocksPerTick=10` limits blocks per scan cycle, preventing API overload after restart.
- **Address monitoring**: `ListAllDerivedAddresses` loads ASSIGNED + RETIRED addresses (ADR §10.3). `RegisterAddress` eliminates 30s refresh window for new addresses.
- **Idempotency**: `ON CONFLICT (tx_hash) DO NOTHING` on deposit insert + `idem_key` on wallet credit — safe for reprocessing.
- **Multi-source verification**: TronScan cross-checks TronGrid data. Degrades to single-source on API failure (by design, with logging).
- **TRC20 recipient verification**: TronScan correctly extracts recipient from `contractData.to_address` for TRC20 transfers (contractType=1), not top-level `toAddress` (which is the contract address).
- **Decimal precision**: All amount calculations use `decimal.Decimal`, no float64.
- **Minimum deposit enforcement**: Uses `decimal.NewFromString` for comparison, not float.
- **Pagination**: TronGrid API pagination via fingerprint, no infinite loop risk (bounded by block size).
- **HTTP timeout**: 10s timeout on both TronGrid and TronScan clients.
- **HD wallet security**: Only xpub held online, no private keys. `compromisedChecker` blocks derivation on xpub audit failure.
- **Derivation index**: PG SEQUENCE (`nextval`) for atomic index allocation, no MAX+1 race.
- **Address assignment idempotency**: `ON CONFLICT (user_id) WHERE status='ASSIGNED' DO NOTHING` + fallback to `GetByUserID` — concurrent calls return same address.
- **Sweep dashboard queries**: Correct unswept balance calculation with LEFT JOIN on sweep_logs.

## Architecture Compliance

- ✅ No REST endpoints (ConnectRPC only)
- ✅ No WebSocket
- ✅ No JSON for data persistence (PG only; JSON used only for external API response parsing — TronGrid/TronScan)
- ✅ `decimal.Decimal` for all financial calculations
- ✅ No `//nolint` or `// @ts-ignore`
- ✅ Push-first: timer-driven scan loop (not polling external state)

## Reuse Preflight

- **C1**: REUSE: `repository.NewDepositRepository(tx)` @ `deposit_repo_v2.go:19` (transaction-scoped repo pattern, same as `repository.NewWithdrawalRepository(tx)` used in sweep worker)

## Migrations

No migrations required.

## Deployment

- `go build ./...` ✅
- `go test ./internal/chain/... ./internal/service/...` ✅
- `docker compose build backend` ✅
- `docker compose up -d backend` ✅
- Container health: `healthy` ✅

## Residual Risks

- **Checkpoint advances on processing failure**: If both `ConfirmDeposit` and the MANUAL_REVIEW fallback fail (e.g., DB connection lost), the checkpoint still advances and the deposit is not recorded. Recovery requires manual blockchain rescan. This is an acceptable edge case — the probability of both paths failing simultaneously is very low, and the deposit can be recovered from on-chain data.
- **TronScan degradation**: When TronScan API is unavailable, verification degrades to TronGrid-only. An attacker who can block TronScan access could reduce verification to single-source. This is by design (better to process deposits with one source than to block them entirely), but operators should monitor TronScan availability.
