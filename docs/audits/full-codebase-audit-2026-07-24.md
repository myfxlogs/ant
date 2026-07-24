# Full Codebase Audit Report — 2026-07-24

> Scope: backend `internal/` + frontend `src/` — logic holes, business anomalies, security risks, compatibility issues.
> Method: Automated grep + code_search + manual review of key paths (auth, marketplace, strategy, SSE, sweep, risk).

---

## Summary

| Severity | Count | Status |
|----------|-------|--------|
| 🔴 Critical | 3 | C1+C2 fixed, C3 needs fix |
| 🟡 Medium | 7 | M4+M6 fixed, rest pending |
| 🟢 Low / Info | 4 | Awareness only |

---

## 🔴 Critical

### C1. `SetInsecureCookies(true)` hardcoded in production

**File**: `cmd/server/handlers_user.go:38`
**Risk**: 🔴 Critical — Security
**Impact**: Refresh token cookie sent without `Secure` flag over HTTP. If TLS is terminated at Cloudflare (as in this deployment), the cookie travels plaintext between Cloudflare edge and the Docker container if no internal TLS exists.
**Trigger**: Any production deployment.
**Current code**:
```go
authServer.SetInsecureCookies(true) // no TLS in Docker deployment
```
**Fix**: Read from config (`cfg.CookieSecure`), default `true` for production. Only set `false` when `cfg.AppURL` is `localhost` or explicitly flagged dev.

### C2. `requireAdmin` bypasses when `platformSvc == nil` (wallet + deposit)

**File**: `connect/user/wallet_handler.go:39-41`, `connect/user/deposit_handler.go:230-231`
**Risk**: 🔴 Critical — Security
**Impact**: If `platformSvc` is nil (misconfiguration, partial startup, test wiring leaking to prod), **all admin checks are bypassed** — any authenticated user can perform admin wallet/deposit operations.
**Trigger**: `platformSvc` nil at runtime.
**Current code**:
```go
if s.platformSvc == nil {
    s.log.Warn("requireAdmin: platformSvc is nil — admin check bypassed (ensure this is dev/test only)")
    return actorID, nil  // ← bypasses admin check!
}
```
**Fix**: Fail-closed: return `PermissionDenied` when `platformSvc == nil`. Never silently bypass admin checks.

---

## 🟡 Medium

### M6. `secretCache` (sync.Map) never evicts expired entries

**File**: `service/systemai/chat.go:120-134`, `service/systemai/service.go:68`
**Risk**: 🟡 Medium — Memory leak
**Impact**: `secretCache` is a `sync.Map` storing `secretCacheEntry{secret, expiresAt}` per `userID|providerID` key. Expired entries are never deleted — they're only overwritten when the same key is accessed again. With N users × M providers, the map grows monotonically. Each entry holds a decrypted API key string (~100 bytes). For 1000 users × 7 providers = 7000 entries ≈ 700KB — not catastrophic, but unbounded.
**Trigger**: Long-running server with many users accessing AI features.
**Fix**: Add a periodic cleanup goroutine that scans and deletes expired entries, or use a TTL cache library.

### M7. SSE per-stream subscription creates N goroutines per connection

**File**: `connect/system/stream_handler.go:131`, `connect/system/stream_handler.go:245-260`
**Risk**: 🟡 Medium — Resource scaling
**Impact**: Each SSE `SubscribeEvents` call creates:
- 1 `SubscribeUserOrderEvents` channel + goroutine
- N `SubscribeAccountProfit` channels (one per account)
- N `SubscribePositionSnapshots` channels
- N `SubscribeAccountStatus` channels
- N `SubscribeBarUpdates` channels + N forwarding goroutines
- N `SubscribeBarDrops` channels + N forwarding goroutines

For a user with 10 accounts, that's ~50 channels + ~20 goroutines per SSE connection. With 100 concurrent SSE clients, that's 5000 channels and 2000 goroutines.
**Trigger**: High concurrent SSE connections with many accounts per user.
**Fix**: Acceptable for current scale. At higher scale, consider a shared fan-out model.

---

## Resource Exhaustion Root Cause Analysis

### C3. `StateCache.orders` map grows unbounded — no eviction

**File**: `mthub/state_cache.go:58,122-131`
**Risk**: 🔴 Critical — Memory leak (OOM)
**Impact**: `StateCache.ApplyEvent()` adds/updates entries in `c.orders[ev.Ticket]` but **never deletes** them. Closed orders, cancelled orders, expired orders — all remain in memory forever. The map grows monotonically with every trade event across all accounts.

Over 7 days of replay (the `cacheReplayWindow`), a single active account can generate thousands of order events (open, fill, modify, close, delete). With multiple accounts, this map grows without bound.

**Redis side**: `persistToRedis` writes each order with a 7-day TTL, so Redis entries expire. But `LoadFromRedis` on restart loads all non-expired keys back into the in-memory map — and then the map only grows from there.

**Trigger**: Long-running server with active trading accounts. After days/weeks of operation, the `orders` map consumes all available memory → OOM kill.

**Fix**: Add `RemoveOrder(ticket)` and call it when order state reaches a terminal state (closed/deleted/cancelled). Also add a periodic sweep that removes entries older than `cacheReplayWindow`.

**This is the most likely cause of the server cache resource exhaustion you observed.**

---

### Other contributing factors

| Factor | File | Impact |
|--------|------|--------|
| `secretCache` no eviction | `systemai/service.go:68` | Slow growth, ~700KB per 1K users |
| Per-SSE goroutine fan-out | `stream_handler.go:131` | 20 goroutines + 50 channels per SSE client |
| `publishedCache` bounded at 256 | `marketplace/publish.go:62` | Has lazy eviction |
| Redis maxmemory 200mb + allkeys-lru | `docker-compose.yml:178` | Redis will evict, but may evict idempotency keys |
| Redis idempotency TTL 96h | `mthub/idempotency.go:23` | 96h keys accumulate; high order volume can pressure 200mb |

### M1. `json.Marshal`/`json.Unmarshal` used in non-exempted paths

**Files**: `service/systemai/chat.go`, `connect/ai/code_assist_handler.go`, `connect/system/analytics_report_gen.go`, `ai/param_proposer.go`, `ai/clarification.go`
**Risk**: 🟡 Medium — Architecture violation (AGENTS.md prohibits `encoding/json` for data serialization)
**Impact**: These are all LLM API integration paths — parsing external JSON responses from OpenAI-compatible APIs. Not persistence/transport, but still violates the zero-tolerance rule.
**Note**: `chain/tron_grid.go` and `chain/tron_scan.go` parse external API responses — same category. `repository/schedule_proto_migrate.go` and `repository/notification_proto_migrate.go` are legacy migration code (exempted as one-time conversions).
**Fix**: Replace with `protojson` for proto types, or use a structured decoder for external API responses. Alternatively, document these as "external API boundary" exemptions in AGENTS.md.

### M2. `unsafe.Pointer` for atomic snapshot swap in `PlatformAggregator`

**File**: `risksvc/platform_aggregator.go:42,69,134,141`
**Risk**: 🟡 Medium — Go race safety
**Impact**: Uses `atomic.StorePointer`/`atomic.LoadPointer` with `unsafe.Pointer` for lock-free snapshot reads. This is a standard Go pattern for atomic pointer swaps, but `unsafe.Pointer` usage requires careful GC alignment. The current implementation is correct **if** `PlatformExposure` is never mutated after creation (which appears to be the case — `Recalculate` creates a new struct).
**Trigger**: Only under high concurrency with GC pressure.
**Fix**: Consider using `atomic.Value` (type-safe, no `unsafe`) or `sync.RWMutex` for the snapshot. Low priority since current code is functionally correct.

### M3. `float64` in risk calculations (`vol_target_sizer.go`)

**File**: `risksvc/vol_target_sizer.go:62,74,84,103`
**Risk**: 🟡 Medium — Precision
**Impact**: `riskBudgetPct` (float64), `AnnualVol` (float64), `HoldingDays` (float64→math.Sqrt), and `RiskUsed` output (float64) are used alongside `decimal.Decimal` in the same calculation. The `decimal.NewFromFloat()` conversion can introduce binary float imprecision.
**Trigger**: When ATR fallback path is used (ATR ≤ 0) or when computing `riskUsed` output.
**Note**: These are non-price intermediate values (risk ratios, annualized volatility, holding days) — not price calculations. `math.Sqrt` has no `decimal.Decimal` equivalent. Broker-sourced account data (Balance/Equity/Profit) uses `float64` because mtapi.io proto defines them as `double` — this is correct, not a violation.
**Fix**: Document these as "non-price intermediate values" exempt from the `decimal.Decimal` rule (similar to `winRate` / `sharpeRatio` which are inherently float64 statistics).

### M4. SSE keepalive goroutine may leak if `Close()` not called

**File**: `interceptor/sse_keepalive.go:67-83,114-120`
**Risk**: 🟡 Medium — Goroutine leak
**Impact**: `keepaliveLoop()` runs in a goroutine and exits on `<-w.done`. `Close()` is only called if `w.wrote` is true. If the handler returns without writing (e.g., error before first `Write`), the goroutine never exits. However, the goroutine only writes to the `ResponseWriter` on ticker fire, and if the handler has returned, the write will fail silently (Go HTTP server closes the connection).
**Trigger**: Handler error before first `Write` call + long-lived connection reuse.
**Fix**: Call `Close()` in a `defer` in the middleware wrapper, or use `r.Context().Done()` as an additional exit channel.

### M5. `BrokerLimitUsage` map uses `float64` for ratio

**File**: `risksvc/platform_aggregator.go:108,129`
**Risk**: 🟡 Medium — Precision
**Impact**: `BrokerLimitUsage` is `map[string]float64` — computed as `brokerMargins[broker].Div(limit).InexactFloat64()`. This is a ratio (0~1), not a price, so precision loss is negligible. But it violates the "no float64" rule.
**Fix**: Change to `map[string]decimal.Decimal` or document as exempt (ratio/statistic, not price).

---

## 🟢 Low / Info

### L1. `UpdateTemplate` handler doesn't check ownership before update

**File**: `connect/strategy/strategy_template_handlers.go:156-199`
**Risk**: 🟢 Low — Defense in depth
**Impact**: The handler calls `s.svc.GetTemplate(ctx, id, uid)` which returns the template if `user_id = $2 OR is_public OR is_system`. A non-owner could potentially call `UpdateTemplate` on a public template. However, `s.svc.UpdateTemplate` has `WHERE id=$1 AND user_id=$12` — so the DB write is safe. The issue is that the handler returns the existing template data to the non-owner before the failed update, which is an information leak (code, parameters).
**Trigger**: Authenticated user calls `UpdateTemplate` on another user's public template.
**Fix**: Add explicit ownership check in the handler after `GetTemplate`: if `existing.UserID != nil && *existing.UserID != uid && !existing.IsSystem` → return `PermissionDenied`.

### L2. `userID()` returns `uuid.Nil` silently on auth failure

**File**: `connect/strategy/strategy_handler.go:59-71`
**Risk**: 🟢 Low — Error handling
**Impact**: `userID()` logs a warning and returns `uuid.Nil` instead of an error. Callers like `ListTemplates` and `ListStrategyCards` pass this to the service layer, which would query with `uuid.Nil` as the user ID — returning empty results. Not a security issue (no data leaked), but a confusing UX.
**Fix**: Use `userIDRequire()` (which returns an error) in handlers that need authentication. `userID()` is only appropriate for optional-auth contexts.

### L3. `HasActiveTrial` swallows errors

**File**: `marketplace/trial.go:71-96`
**Risk**: 🟢 Low — Error handling
**Impact**: Parse errors and DB errors return `false, nil` — the caller treats it as "no active trial". If the DB is temporarily unavailable, users could be denied trial access silently.
**Fix**: Return the error to the caller for proper handling.

### L4. `notifyTrialExpiring` uses `context.WithoutCancel(ctx)`

**File**: `marketplace/trial.go:101`
**Risk**: 🟢 Info — Design choice
**Impact**: The notification goroutine outlives the request context. This is intentional (fire-and-forget notification), but if the server is shutting down, the goroutine has no cancellation signal.
**Fix**: Acceptable as-is. Could use a dedicated background context with server lifecycle cancellation.

---

## Architecture Compliance

| Rule | Status | Notes |
|------|--------|-------|
| No REST endpoints | ✅ | Only healthz/readyz/livez found |
| No WebSocket | ✅ | SSE only |
| No JSON persistence | ⚠️ | M1: LLM API integration uses `encoding/json` |
| No float64 in prices | ⚠️ | M3/M5: Risk calculations use float64 for ratios/statistics (non-price). Broker-sourced data uses float64 (mtapi proto `double` — correct design, not a violation). Internal financial calculations (wallet/settlement/pricing) use `decimal.Decimal`. |
| No `//nolint`/`@ts-ignore` | ✅ | None found |
| ConnectRPC + SSE only | ✅ | All external APIs use ConnectRPC |
| Push-first architecture | ✅ | PG LISTEN, SSE streams, no polling for real-time data |
| Ownership checks | ✅ | DB layer enforces `user_id` in WHERE clauses |
| Admin interceptor | ✅ | Separate `AdminInterceptor` for admin-only RPCs |
| Idempotency keys | ✅ | Marketplace purchase/refund/renew all use idem keys |

---

## Fixes Applied

| ID | Fix | Files Modified | Status |
|----|-----|----------------|--------|
| C1 | `SetInsecureCookies` from `cfg.CookieSecure` (env `COOKIE_SECURE`, default `true`) | `config/config.go`, `cmd/server/handlers_user.go` | ✅ |
| C2 | `requireAdmin` fail-closed when `platformSvc == nil` | `connect/user/wallet_handler.go`, `connect/user/deposit_handler.go` | ✅ |
| C3 | `StateCache.orders` evicts terminal states (CLOSED/FILLED/CANCELLED/EXPIRED/FAILED/REJECTED) | `mthub/state_cache.go`, `mthub/state_cache_test.go` | ✅ |
| L1 | `UpdateTemplate` ownership check before mutation | `connect/strategy/strategy_template_handlers.go` | ✅ |
| M4 | SSE keepalive `defer kw.Close()` to prevent goroutine leak | `interceptor/sse_keepalive.go` | ✅ |

## Remaining (Recommended, Non-blocking)

| ID | Action | Priority |
|----|--------|----------|
| M1 | Document LLM API boundary exemption for `encoding/json` in AGENTS.md | Low |
| M2 | Replace `unsafe.Pointer` with `atomic.Value` in `PlatformAggregator` | Low |
| M3 | Document float64 exemption for risk ratios/statistics (non-price values) | Low |
| M5 | Change `BrokerLimitUsage` to `decimal.Decimal` or document as exempt | Low |
| M6 | Add periodic cleanup for `secretCache` expired entries | Low |
| M7 | Shared fan-out for SSE subscriptions at scale | Low |
