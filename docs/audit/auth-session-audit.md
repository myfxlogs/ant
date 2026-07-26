# Auth & Session Audit Report

## Scope

Comprehensive review of the Auth & Session subsystem:
- Auth interceptor (JWT validation, public endpoint whitelist, API key auth, streaming handler)
- Admin interceptor (permission enforcement, unary + streaming)
- Login/Register/Logout handlers
- JWT token generation (access + refresh), token version, cookie security
- Token refresh flow (RefreshToken + RefreshTokenFromCookie)
- Email verification (token generation, validation, atomicity)
- Password storage (argon2id + bcrypt backward compat)
- Password reset (ForgotPassword, ResetPassword, VerifyMTIdentity)
- Rate limiting (login/register brute-force protection)
- WebAuthn coldsign verification (assertion, challenge, sign count, whitelist, limits)
- User repository (GetByID, GetByEmail, GetByLogin, GetByAccountNumber)

## Findings

### A6 — GetByLogin/GetByEmail/GetByAccountNumber missing token_version 🔴 HIGH

**File**: `backend/internal/repository/user_repo.go:66-150`

**Problem**: Three user lookup queries (`GetByEmail`, `GetByLogin`, `GetByAccountNumber`) did not SELECT `token_version` from the database. The Go struct field `User.TokenVersion` defaulted to 0 (zero value). The `Login` handler uses `GetByLogin` then calls `issueRefreshToken(user.ID.String(), tokenEmail, user.TokenVersion)` — issuing a refresh token with `TokenVersion=0`.

For a new user, `token_version` starts at 0, so the first login works. But after logout (which calls `IncrementTokenVersion`, bumping DB to 1), the next login still reads `TokenVersion=0` from the struct (not queried). The refresh token has version 0, but the DB has version 1 → `RefreshToken` calls `GetByID` (which DOES query `token_version`), finds mismatch, returns "token revoked". **Users cannot refresh after logout.**

**Fix**: Added `token_version` to the SELECT and Scan in all three query methods.

**Risk if unfixed**: After any logout, all subsequent login sessions have broken refresh tokens. Users are forced to re-login every 15 minutes (access token TTL) with no refresh capability.

### A7 — RefreshToken/RefreshTokenFromCookie ignoring JWT issue errors 🟡 MEDIUM

**File**: `backend/internal/connect/user/auth_token.go:81-82, 117-118`

**Problem**: Both `RefreshToken` and `RefreshTokenFromCookie` used `accessToken, _ := s.issueAccessToken(...)` and `refreshToken, _ := s.issueRefreshToken(...)`, discarding the error. If JWT signing failed (e.g., transient error), empty strings would be returned as tokens, and the client would receive `AccessToken: ""` — causing an auth loop where every subsequent request fails and triggers another refresh.

**Fix**: Properly handle errors from `issueAccessToken` and `issueRefreshToken`, returning `CodeInternal` on failure.

### A8 — ValidateResetToken+ConsumeResetToken non-atomic TOCTOU 🟡 MEDIUM

**File**: `backend/internal/repository/password_reset_repo.go:68-101`

**Problem**: `ValidateResetToken` checked `consumed = FALSE` and `expires_at > NOW()` in a SELECT, then `ConsumeResetToken` separately set `consumed = TRUE` in an UPDATE. Two concurrent requests with the same token could both pass the SELECT check before either reached the UPDATE — allowing a token to be used twice for different passwords.

**Fix**: Combined validate+consume into a single atomic `UPDATE ... SET consumed = TRUE WHERE consumed = FALSE AND expires_at > NOW() RETURNING user_id`. This makes the operation atomic at the DB level — only one concurrent request can succeed. `ConsumeResetToken` is now a redundant no-op but kept for backward compatibility.

## Verified Safe (No Issues Found)

- **JWT validation**: `ValidateToken` explicitly checks `token.Method.(*jwt.SigningMethodHMAC)` — prevents `alg=none` attack
- **Password storage**: argon2id (64MB/3 iterations/2 parallelism) with `subtle.ConstantTimeCompare` + bcrypt backward compat
- **Cookie security**: `HttpOnly` + `SameSite=Strict` + `Secure` (production); `insecure` flag for dev
- **User enumeration prevention**: `ForgotPassword` and `ResendVerification` always return success
- **Password reset**: `IncrementTokenVersion` after reset invalidates all existing sessions
- **Logout**: Correctly increments `TokenVersion` + clears refresh cookie
- **Rate limiting**: Login, Register, ForgotPassword, VerifyMTIdentity all rate-limited per IP
- **Admin interceptor**: Both `WrapUnary` and `WrapStreamingHandler` check admin privileges
- **Auth interceptor whitelist**: All public endpoints correctly whitelisted (login, register, refresh, verify email, resend verification, forgot/reset password, verify MT identity, marketplace public reads)
- **Streaming handler auth**: `WrapStreamingHandler` authenticates all streams (no whitelist bypass)
- **Email verification**: Token stored as SHA-256 hash, consumed atomically in a transaction
- **WebAuthn coldsign**: Self-held public keys (not trusting online server), challenge reconstruction, sign count for replay prevention, whitelist + withdrawal limit checks
- **API key auth**: Validated via `APIKeyValidator` interface, scopes injected into context
- **Client IP extraction**: Takes first IP from `X-Forwarded-For` (original client), falls back to `X-Real-IP`

## Architecture Compliance

- ✅ ConnectRPC only (no REST, no WebSocket)
- ✅ No JSON serialization for data exchange
- ✅ No float64 in auth logic
- ✅ No hardcoded secrets (JWT_SECRET from env)
- ✅ No `//nolint` or `// @ts-ignore`

## Reuse Preflight

- **A6**: REUSE: `GetByEmail`/`GetByLogin`/`GetByAccountNumber` @ `user_repo.go` (fixed existing queries)
- **A7**: REUSE: `issueAccessToken`/`issueRefreshToken` @ `auth_token.go` (added error handling to existing calls)
- **A8**: REUSE: `ValidateResetToken` @ `password_reset_repo.go` (made existing function atomic)

## Migrations

No migrations required — `token_version` column already exists in `users` table (migration 061).

## Deployment

- `go build ./...` ✅
- `go test ./internal/interceptor/... ./internal/connect/user/...` ✅
- `docker compose build backend` ✅
- `docker compose up -d backend` ✅
- Container health: `healthy` ✅

## Residual Risks

None — all three findings addressed. The auth system is robust:
- JWT validation prevents algorithm substitution
- argon2id with constant-time comparison prevents timing attacks
- Token version provides session invalidation
- Atomic token consumption prevents TOCTOU races
- Rate limiting prevents brute-force
- Cookie security prevents XSS-based token theft
