# Backend Security Audit Report

**Date**: 2026-07-22  
**Scope**: Authentication, JWT, Cookie, SQL injection, Business logic, Secret management, Frontend security  
**Status**: Completed — 1 bug fixed, 4 informational findings documented

---

## Summary

| Severity | Count | Status |
|----------|-------|--------|
| 🔴 Critical | 0 | — |
| 🟠 High | 1 | ✅ Fixed |
| 🟡 Medium | 2 | Documented |
| 🟢 Low / Info | 5 | Documented |

---

## Findings

### 🟠 H-1: SQL Syntax Bug in Date Range Filters (Fixed)

**Files**:
- `backend/internal/repository/connection_log_repository.go:54-55`
- `backend/internal/repository/order_history_repository.go:86-87`
- `backend/internal/repository/operation_log_repository.go:66-67`
- `backend/internal/repository/auto_trading_settings.go:132-133`

**Root cause**: The `addFilter` closure generates `AND %s = $%d`. When called with operator-suffixed column names like `addFilter("created_at >=", val)`, the generated SQL becomes `AND created_at >= = $N` — a double `=`, which is invalid SQL and causes a query error whenever a user filters by date range.

**Impact**: Any API request with `startDate` or `endDate` filter parameters returns a 500 error. This affects connection logs, order history, operation logs, and trading logs endpoints.

**Fix applied**: Replaced `addFilter("col >=", val)` calls with inline parameterized SQL: `baseQ += fmt.Sprintf(" AND col >= $%d", idx); args = append(args, val); idx++`.

**Build verification**: `go build ./...` passes.

---

### 🟡 M-1: Access Token in URL Query Parameter

**File**: `backend/internal/interceptor/auth.go:112`

```go
if t := strings.TrimSpace(r.URL.Query().Get("access_token")); t != "" {
    hdr.Set("Authorization", "Bearer "+t)
}
```

**Risk**: EventSource (SSE) cannot set custom headers, so the access token is passed as `?access_token=` in the URL. URL parameters are logged by nginx, browser history, and potentially cached by proxies.

**Mitigation**: This is an inherent limitation of the EventSource API. The access token has a 15-minute TTL, limiting exposure. For a future fix, consider switching SSE auth to cookie-based or using the `Authorization` header via `fetch()` + `ReadableStream` instead of `EventSource`.

---

### 🟡 ~~M-2: Rate Limiter Trusts X-Forwarded-For Without Validation~~ (Fixed)

**File**: `backend/internal/interceptor/ratelimit.go:119-131`

**Status**: ✅ Fixed (2026-08-01)

**Fix**: Changed `extractClientIPFromHeader` to prefer `X-Real-IP` (set by nginx to `$remote_addr`, the actual TCP peer IP, overwriting any client-supplied value) over `X-Forwarded-For` (which uses `$proxy_add_x_forwarded_for` that appends to client-supplied values). This prevents rate limiter bypass via XFF header forgery.

---

### 🟢 L-1: JWT Uses HS256 (Symmetric) Instead of RS256 (Asymmetric)

**File**: `backend/internal/connect/user/auth_token.go:68`

```go
token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
```

**Risk**: HS256 uses a shared secret for both signing and verification. If the secret leaks, tokens can be forged. RS256 would separate signing key from verification key.

**Mitigation**: For a single-service monolith, HS256 is acceptable. The `jwtSecret` is loaded from env var `JWT_SECRET` and validated at startup. If the service splits into multiple processes, migrate to RS256.

---

### 🟢 L-2: No JWT `iss` or `aud` Claims

**File**: `backend/internal/connect/user/auth_token.go:55-63`

The JWT claims include `sub`, `iat`, `exp`, `jti`, and custom `user_id`/`token_version`, but omit `iss` (issuer) and `aud` (audience).

**Risk**: If multiple services share the same JWT secret, a token issued for one service could be used on another.

**Mitigation**: For a single-service deployment, this is not a risk. Add `iss` and `aud` if the architecture evolves to multi-service.

---

### 🟢 L-3: Refresh Token Rotation Without Revoke-on-Reuse

**File**: `backend/internal/connect/user/auth_token.go:97-120` (RefreshToken)

The refresh token flow issues a new refresh token on each refresh, but does not revoke the old refresh token if it's reused. The `token_version` check only invalidates tokens on explicit logout/password reset.

**Risk**: If a refresh token is stolen, the attacker can use it indefinitely until the user changes their password or logs out. The 7-day TTL provides a long window.

**Mitigation**: Consider implementing a refresh token blacklist or one-time-use refresh tokens (rotate + revoke old token on each refresh). This requires tracking consumed refresh tokens.

---

### 🟢 L-4: SystemAISecret Logs Masked Secret at Info Level

**File**: `backend/internal/connect/ai/system_ai_handler.go:124`

```go
s.log.Info("UpdateSystemAISecret", zap.String("provider_id", req.Msg.ProviderId), zap.String("secret", maskedSecret))
```

**Risk**: The masking is adequate (≥50% masked, min 4 chars), but logging even partially visible API keys at Info level in production is unnecessary. The prefix/suffix portions could help an attacker narrow down the key.

**Mitigation**: Change to Debug level, or remove the secret from the log entirely and only log the provider_id + success/failure.

---

### 🟢 L-5: No CSRF Protection for Cookie-Based Refresh

**File**: `backend/internal/connect/user/auth_token.go:88` (makeRefreshCookie)

The refresh token cookie uses `SameSite=Strict` which provides strong CSRF protection. However, the `RefreshTokenFromCookie` endpoint relies solely on the cookie for authentication.

**Risk**: `SameSite=Strict` blocks cross-site requests, so CSRF is effectively mitigated. The only residual risk is from same-site subdomain attacks, which would require a subdomain compromise.

**Mitigation**: Current implementation is adequate. If SameSite needs to be relaxed to `Lax` in the future, add a CSRF token or custom header check.

---

## Areas Reviewed — No Issues Found

### Authentication & JWT
- ✅ JWT signing method validation rejects non-HMAC algorithms (`auth.go:189-191`)
- ✅ Password hashing uses Argon2id (preferred) with bcrypt fallback (`password.go:14-21`)
- ✅ Password minimum length enforced (≥8) on registration and reset
- ✅ User enumeration prevented in `ForgotPassword` — always returns success (`auth_handler.go:91-93`)
- ✅ Token version increment on logout and password reset invalidates all sessions
- ✅ Disabled user check on both login and token refresh
- ✅ Email verification gate configurable via `REQUIRE_EMAIL_VERIFICATION`

### Cookie Security
- ✅ `HttpOnly` flag set — prevents JavaScript access
- ✅ `SameSite=Strict` — prevents CSRF
- ✅ `Secure` flag configurable via `COOKIE_SECURE` env var (default: true)
- ✅ `Path=/` scoped correctly
- ✅ `Max-Age` set to match refresh token TTL

### SQL Injection
- ✅ All user-facing queries use parameterized placeholders (`$1`, `$2`, ...)
- ✅ Dynamic filter builders use `fmt.Sprintf` only for column names and placeholder indices, never for user values
- ✅ Partition names validated with regex (`^[a-z_][a-z0-9_]*$`) before DDL
- ✅ Partition DDL uses hardcoded table names, not user input
- ✅ Admin search uses `ILIKE $N` with parameterized values, not string concatenation

### Business Logic & Race Conditions
- ✅ Wallet operations use `FOR UPDATE` row locking within transactions
- ✅ Idempotency keys prevent double-credit on retries (`wallet_repo.go:114-177`)
- ✅ Advisory locks serialize ledger chain operations (`wallet_repo.go:234-237`)
- ✅ Marketplace purchase uses single transaction with `FOR UPDATE` on buyer wallet
- ✅ Subscription renewal checks balance before charging, within transaction
- ✅ Deposit confirmation is atomic (insert + credit in one transaction)
- ✅ CHECK constraint on wallet balance prevents negative balances

### Secret Management
- ✅ MT passwords encrypted with AES-256-GCM via `secrets.Client` before storage
- ✅ Master key loaded from env var, required at startup (fatal if missing)
- ✅ Key rotation supported via `EnvMasterKey.Rotate()`
- ✅ AI provider secrets encrypted with `secretbox.Box` (AES-256-GCM)
- ✅ Plaintext credential backfill migrates legacy data on startup, then column can be dropped
- ✅ `.env.example` uses `CHANGE_ME` placeholders, no real secrets

### Frontend Security
- ✅ No `dangerouslySetInnerHTML` usage anywhere
- ✅ No `innerHTML`, `document.write`, or `eval()` calls
- ✅ Access token stored in memory only (Zustand), not persisted to localStorage
- ✅ Only `user` profile persisted to localStorage (no tokens)
- ✅ Refresh token in httpOnly cookie — not accessible via JavaScript
- ✅ Nginx security headers: `X-Content-Type-Options`, `X-Frame-Options: DENY`, `X-XSS-Protection`, `Referrer-Policy`, `Permissions-Policy`, `HSTS`, `Content-Security-Policy`
- ✅ CSP restricts `script-src` to `'self' 'unsafe-inline' 'unsafe-eval'` (note: `'unsafe-inline'` and `'unsafe-eval'` could be tightened if Vite is configured for nonce-based CSP)

### Rate Limiting
- ✅ Login, register, forgot-password, and AI endpoints rate-limited per IP
- ✅ Nginx additional rate limiting: 30r/s for API, 5r/m for login
- ✅ In-memory rate limiter has cleanup loop (10-minute TTL for idle entries)

---

## Fixed in This Audit

| ID | File | Fix |
|----|------|-----|
| H-1 | `connection_log_repository.go` | Replaced `addFilter("created_at >=", val)` with inline `fmt.Sprintf(" AND created_at >= $%d", idx)` |
| H-1 | `order_history_repository.go` | Same fix for `open_time >=` / `open_time <=` |
| H-1 | `operation_log_repository.go` | Same fix for `created_at >=` / `created_at <=` |
| H-1 | `auto_trading_settings.go` | Same fix for `created_at >=` / `created_at <=` |
| H-1 | `admin_repo_logs.go` | Same fix — `applyFilter("created_at >=", ...)` produced double-equals; added `applyRangeFilter` with direct `>=`/`<=` operators (2026-08-01) |

---

## Comprehensive Code Audit (2026-08-01)

### float64 Price Calculation Compliance

**Status**: ✅ Clean

- mtapi.io gRPC proto defines financial fields as `float64` (external constraint, cannot change)
- All adapter code (`mt4/`, `mt5/`) immediately converts to `decimal.Decimal` at the boundary via `decimal.NewFromFloat()`
- All internal types (`MTAccountInfo`, `ProfitUpdate`, `OrderUpdate`, `OrderRecord`) use `decimal.Decimal`
- All proto API types use `string` for monetary values (balance, equity, profit, margin, etc.)
- `InexactFloat64()` calls in analytics are for statistical ratios (Sharpe, win rate, profit factor) — acceptable, no monetary calculation

### SQL Injection

**Status**: ✅ Clean

- All user-supplied values use parameterized queries (`$1`, `$2`, ...)
- `buildAffectedCountQuery` in `admin_repo_users_delete.go` interpolates table/column names from `information_schema` (DB internal metadata, not user input) — safe
- `partition_mgr.go` uses hardcoded table names (`"md_bars"`, `"close_ts_unix_ms"`) and validates dynamic partition names with `validPartitionName` regex before DDL — safe
- `market_data_pg.go` dynamic `DISTINCT ON` clause uses hardcoded column lists, not user input — safe

### Authorization

**Status**: ✅ Clean

- Admin RPCs protected by `AdminInterceptor` which checks `IsAdmin()` for every request (both unary and streaming)
- User RPCs consistently call `parseUserID()` to extract and validate user identity
- `WalletServer.resolveTargetUser()` correctly enforces admin check before allowing cross-user access
- `DepositServer.requireAdmin()` follows fail-closed pattern when `platformSvc` is nil
- Account CRUD operations verify user ownership via `UserOwnsAccount()`

### Logic Bugs

**Status**: ✅ Fixed

- **H-1 in `admin_repo_logs.go`**: Same double-equals SQL syntax bug as the original 4 files. `applyFilter("created_at >=", val)` produced `AND created_at >= = $N` (invalid SQL). Fixed by adding `applyRangeFilter` with direct `>=`/`<=` operators. Verified with 24/24 admin page E2E tests + 8/8 date-range filter regression tests.

---

## Recommendations (Prioritized)

1. ~~**P2**: Tighten CSP — remove `'unsafe-inline'` and `'unsafe-eval'` from `script-src`~~ ✅ Done (2026-08-01) — removed `'unsafe-inline'`, kept `'unsafe-eval'` for chart libs
2. ~~**P2**: Add trusted proxy validation for `X-Forwarded-For` in rate limiter~~ ✅ Done (2026-08-01) — prefer `X-Real-IP` over XFF
3. **P3**: Implement refresh token one-time-use (rotate + revoke old token on each refresh)
4. **P3**: Reduce `UpdateSystemAISecret` log level from Info to Debug
5. **P4**: Add `iss` and `aud` JWT claims for future multi-service readiness
6. **P4**: Migrate JWT from HS256 to RS256 if service decomposition is planned
