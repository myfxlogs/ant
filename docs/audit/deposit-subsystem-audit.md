# Deposit Subsystem Audit Report

## Scope

- `internal/hdwallet/xpub.go` — BIP44 HD wallet derivation (watch-only xpub)
- `internal/service/deposit_service.go` — DepositService: address derivation, ConfirmDeposit
- `internal/chain/monitor.go` — Chain monitor: TronGrid block scanning, event processing
- `internal/chain/tron_grid.go` — TronGrid API client: block events, balance queries
- `internal/chain/tron_scan.go` — TronScan API client: multi-source verification
- `internal/audit/xpub_audit.go` — XpubAuditor: 24h address integrity check
- `internal/repository/deposit_address_repo.go` — DepositAddressRepository: CRUD, sequence-based index
- `internal/repository/deposit_repo_v2.go` — DepositRepository: deposit records, sweep dashboard
- `internal/connect/user/deposit_handler.go` — DepositServer: RPC handlers, ImportXpub, requireAdmin
- `internal/reconcile/reconcile.go` — Reconciler: on-chain vs internal balance verification
- `migrations/204_hd_wallet_deposit.up.sql` — Schema definition
- `migrations/205_hd_wallet_phase_a_cleanup.up.sql` — Schema cleanup (dropped encrypted_privkey)

## Findings

### No bugs found — subsystem is well-designed

After thorough review, no actionable bugs were identified. The deposit subsystem demonstrates strong security architecture with multiple defense layers. All findings below are verified-safe design decisions, not issues.

## Verified Safe (No Issues Found)

### HD Wallet & Address Derivation
- **Watch-only xpub**: Online server holds ONLY the account-level extended public key (m/44'/195'/0'/0). No private keys ever touch the online machine (ADR-0026 R1). The `encrypted_privkey` column from initial migration was dropped in migration 205.
- **BIP32 CKDpub**: `DeriveAddressFromExtKey` uses public-only derivation — no private key needed.
- **PG SEQUENCE for index allocation**: `NextDerivationIndex` uses `nextval('deposit_addr_index_seq')` — atomic, no MAX(index)+1 race condition.
- **Idempotent address assignment**: `InsertDepositAddress` uses `ON CONFLICT (user_id) WHERE status='ASSIGNED' DO NOTHING` — concurrent calls for the same user return the same address.
- **Xpub fingerprint verification**: `ImportXpub` checks SHA-256 fingerprint against `DEPOSIT_XPUB_FINGERPRINT` env var — detects xpub substitution at import time.
- **Xpub integrity audit**: `XpubAuditor` runs every 24h, re-derives all DB addresses from xpub, flags mismatches. When compromised, `GetOrDeriveAddress` blocks new derivation (ADR-0026 §12.1).
- **Xpub hot-reload**: `UpdateXpub` validates, parses, and atomically swaps xpub string + parsed key under RWMutex.
- **Import cross-check**: `ImportDepositAddresses` re-derives each address from xpub and compares — skips mismatches.

### Deposit Confirmation
- **Multi-source verification**: `processEvent` verifies each deposit via TronGrid (confirmed events) + TronScan (independent confirmation). If TronScan disagrees → `MANUAL_REVIEW`.
- **Graceful degradation**: TronScan API failure (network/timeout) → retry once → degrade to TronGrid-only. This is safe because TronGrid events are already filtered by `only_confirmed=true`.
- **Minimum deposit amount**: Enforced using `decimal.Decimal` (no float64). Amounts below `min_deposit_amount` are ignored with a warning log.
- **Atomic credit**: `ConfirmDeposit` uses a single DB transaction: deposit INSERT + wallet credit. If either fails, both roll back.
- **Idempotent deposit**: `tx_hash` is UNIQUE in `deposits` table. `Create` uses `ON CONFLICT (tx_hash) DO NOTHING`. Wallet credit uses `idem_key = "deposit-"+txHash` — `AdjustBalanceTx` returns `ErrIdempotentReplay` for duplicates, which is caught and returned as nil.
- **Double-credit prevention**: If `ConfirmDeposit` fails, fallback inserts a `MANUAL_REVIEW` deposit (not confirmed). Operator must manually review and credit. No automatic retry that could double-credit.

### Chain Monitor
- **Single-instance execution**: PG advisory lock (`pg_try_advisory_lock(20260719)`) prevents multiple monitor instances.
- **Checkpoint persistence**: `last_scanned_block` saved to `system_config` after each block. Restart recovery is automatic.
- **Catch-up scanning**: `maxBlocksPerTick=10` prevents overwhelming TronGrid after a restart gap. Logs catch-up progress.
- **Confirmation wait**: Only scans blocks with `minConfirms` confirmations (`safeLatest = latest - minConfirms`).
- **Refresh interval**: Address map refreshed every 30s. New addresses also registered immediately via `RegisterAddress` callback — eliminates the 30s window.
- **Retired address monitoring**: `ListAllDerivedAddresses` loads ALL addresses (ASSIGNED + RETIRED) — users may send to old addresses.
- **Orphaned address handling**: `user_deposit_addresses.user_id` has `ON DELETE SET NULL`. When user is deleted, `user_id` becomes NULL, and `ListAllDerivedAddresses` filters by `WHERE user_id IS NOT NULL` — address stops being monitored.

### Config Validation
- **Fail-closed for critical config**: `usdt_contract_address` and `min_confirmations` missing → monitor refuses to start (returns error).
- **Default for non-critical config**: `min_deposit_amount` missing → defaults to "1" USDT.
- **Config refresh**: Config loaded once at startup. Not hot-reloaded, but monitor restart picks up changes.

### Access Control
- **Admin-only operations**: `ImportXpub`, `ListDepositAddresses`, `ListManualReviewDeposits`, `ImportDepositAddresses`, sweep handlers all require `requireAdmin`.
- **requireAdmin fail-closed**: Returns `CodeInternal` if `platformSvc` is nil.
- **User-scoped queries**: `ListMyDeposits` uses authenticated `userID` from context — users can only see their own deposits.
- **GetDepositAddress**: Uses authenticated `userID` — users can only get their own address.

### Data Precision
- **decimal.Decimal throughout**: Amount parsing, minimum deposit check, USDT conversion all use `decimal.Decimal`. No float64 in price calculations.
- **USDT 6 decimals**: `convertSunToUSDT` divides by 10^6 and formats with `StringFixed(6)`. Stored as `NUMERIC(20,8)` — no precision loss.
- **Amount string passthrough**: Deposit amounts stored as strings in proto, converted to `NUMERIC` by PostgreSQL. No float conversion.

### Reconciliation
- **On-chain vs internal**: Reconciler compares `SUM(on-chain USDT balances)` against `SUM(confirmed deposits) - SUM(swept amounts)`.
- **Cold wallet included**: Swept funds land in cold wallet — reconciliation includes cold wallet balance.
- **All addresses covered**: Reconciliation queries all derived addresses (ASSIGNED + RETIRED).

### Schema Integrity
- **UNIQUE constraints**: `tx_hash` UNIQUE on deposits (idempotency), `address` UNIQUE on deposit_addresses, `derivation_index` UNIQUE on deposit_addresses.
- **Partial unique index**: `ON CONFLICT (user_id) WHERE status='ASSIGNED'` — one active address per user.
- **Foreign keys**: `user_id` REFERENCES users with `ON DELETE SET NULL` (addresses) / `ON DELETE CASCADE` (deposits). Soft-delete doesn't trigger cascade.
- **Cleanup migration**: Migration 205 dropped `encrypted_privkey` and `wallet_secrets` table — no private key material in online DB.

## Architecture Compliance

- ✅ No REST endpoints (ConnectRPC only)
- ✅ No JSON for data persistence (PostgreSQL + proto). TronGrid/TronScan API responses use `encoding/json` — this is external API protocol constraint, not project choice (exempted per rules).
- ✅ No float64 in price calculations (decimal.Decimal throughout)
- ✅ No `//nolint` or `// @ts-ignore`
- ✅ Push-first: Chain monitor uses polling (TronGrid has no push capability for block events — this is the correct exception per rules)
- ✅ Fail-closed: Missing critical config → monitor refuses to start
- ✅ Advisory lock for single-instance execution

## Reuse Preflight

No new files or functions were created in this audit. All reviewed code was existing.

## Notes

- `GetTRC20Balance` hardcodes `usdtContractMainnet` instead of using configurable `usdt_contract_address`. This is acceptable because USDT on TRON mainnet has a fixed contract address, and the system only runs on mainnet. If testnet support is needed in the future, this should be parameterized.
- Chain monitor uses polling (3s interval) because TronGrid has no push/streaming capability for block events. This is the correct exception to the push-first rule — no streaming equivalent exists.
