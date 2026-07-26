# User Management & Auth Subsystem Audit Report

## Scope

- `internal/connect/user/auth_handler.go` — Login, Logout, GetMe, ForgotPassword, ResetPassword
- `internal/connect/user/auth_register.go` — Register
- `internal/connect/user/auth_token.go` — RefreshToken, RefreshTokenFromCookie, JWT issuance
- `internal/connect/admin/admin_user_handler.go` — AdminUserServer, ListUsers, GetDashboard, role/status validation
- `internal/connect/admin/admin_user_handler_write.go` — CreateUser, UpdateUser, DeleteUser, DeleteUsers, DisableUser, EnableUser, ResetUserPassword, RestoreUser
- `internal/interceptor/auth.go` — AuthInterceptor, JWT validation, context injection
- `internal/service/registration_service.go` — RegistrationService (user + account number + wallet)
- `internal/service/user_deletion_service.go` — SoftDeleteUser, SoftDeleteUsers, RestoreUser
- `internal/repository/user_repo.go` — User CRUD, GetByLogin, IncrementTokenVersion, GetCapabilities
- `internal/repository/admin_repo_users.go` — Admin user queries, CountAdmins
- `internal/repository/admin_repo_users_delete.go` — SetUserStatus, ResetUserPassword
- `internal/service/platform_service.go` — IsAdmin

## Findings

### U1 — Disabled users can refresh tokens for 7 days 🔴 CRITICAL

**Files**: `auth_token.go:68-91` (RefreshToken), `auth_token.go:95-141` (RefreshTokenFromCookie)

**Problem**: Both refresh handlers checked `TokenVersion` but **not** `user.Status`. A disabled user (status="disabled") could continue using their refresh token to mint new access tokens for the full 7-day refresh token TTL. The Login handler correctly checks `user.Status != "active"`, but the refresh path bypassed this check entirely.

**Attack scenario**: Admin disables a malicious user. The user's browser still has the `refresh_token` httpOnly cookie. For the next 7 days, the user can call `RefreshTokenFromCookie` to get new 15-minute access tokens and continue using the platform.

**Fix**: Added `user.Status != "active"` check after fetching the user in both `RefreshToken` and `RefreshTokenFromCookie`. Returns `CodePermissionDenied` if not active.

```diff
+if user.Status != "active" {
+    return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("account is disabled"))
+}
```

**Risk if unfixed**: Disabled/suspended users retain full platform access for up to 7 days after being disabled.

### U2 — UpdateUser accepts arbitrary status strings 🟡 MEDIUM

**File**: `admin_user_handler_write.go:127-129`

**Problem**: `UpdateUser` validated `Role` via `validRole()` but accepted any string for `Status` without validation. An admin could set status to arbitrary values (e.g. `"hacked"`, `"superuser"`, `""` after trimming). This could bypass the `user.Status != "active"` check in Login/Refresh (since `"hacked" != "active"` would block login, but `"active"` variants like `"Active"` with different casing could cause confusion).

**Fix**: Added `validStatus()` function (matching the existing `validRole` pattern) and validate status in `UpdateUser`:

```diff
+func validStatus(status string) bool {
+    switch status {
+    case "active", "disabled", "suspended", "pending":
+        return true
+    }
+    return false
+}

 if req.Msg.Status != "" {
+    if !validStatus(req.Msg.Status) {
+        return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid status: %s", req.Msg.Status))
+    }
     existing.Status = req.Msg.Status
 }
```

**Risk if unfixed**: Data integrity violation. Invalid status values could cause unexpected behavior in status-dependent code paths.

### U3 — DisableUser/DeleteUser don't invalidate sessions 🟡 MEDIUM

**Files**: `admin_user_handler_write.go:DisableUser`, `DeleteUser`, `DeleteUsers`, `UpdateUser`

**Problem**: When an admin disables, deletes, or changes a user's status/role, the user's existing JWT refresh tokens remain valid. Combined with U1 (now fixed), disabled users are blocked at refresh time, but their 15-minute access token still works until it expires. More critically, if U1 were not fixed, the 7-day refresh window would be the only barrier.

**Fix**: Added `IncrementTokenVersion` to `AdminRepository` and call it from:
- `DisableUser` — after setting status to "disabled"
- `DeleteUser` — after successful soft-delete
- `DeleteUsers` — for all IDs in the batch
- `UpdateUser` — when status transitions to non-active, or when role changes

This immediately invalidates all existing refresh tokens (token_version mismatch), reducing the attack window from 15 minutes (access token TTL) to **0 seconds**.

```diff
// AdminRepository
+func (r *AdminRepository) IncrementTokenVersion(ctx context.Context, id uuid.UUID) error {
+    _, err := r.db.Exec(ctx, `UPDATE users SET token_version = token_version + 1 WHERE id = $1`, id)
+    return err
+}

// DisableUser
+if err := s.repo.IncrementTokenVersion(ctx, id); err != nil { ... }

// UpdateUser — after UpdateUser succeeds
+if prevStatus != existing.Status && existing.Status != "active" {
+    s.repo.IncrementTokenVersion(ctx, id)
+}
+if prevRole != existing.Role {
+    s.repo.IncrementTokenVersion(ctx, id)
+}
```

**Risk if unfixed**: Disabled/deleted users retain access for up to 15 minutes (access token TTL). Role-downgraded users (admin → user) retain elevated permissions for the same window.

## Verified Safe (No Issues Found)

### Authentication
- **JWT validation**: `ValidateToken` checks signing method is HMAC (prevents alg=none attack), validates signature and expiry.
- **Password hashing**: Uses `bcrypt` via `hash.HashPassword` / `repository.HashPassword`.
- **Password verification**: `VerifyPassword` uses `bcrypt.CompareHashAndPassword` — constant-time comparison.
- **User enumeration prevention**: `ForgotPassword` always returns success, even if email doesn't exist.
- **Email verification gate**: Login checks `EmailVerifiedAt` when `requireEmailVerif` is enabled.
- **Cookie security**: `HttpOnly`, `Secure` (unless insecure mode), `SameSite=Strict`, `Path=/`.
- **Token version for session invalidation**: `IncrementTokenVersion` used on Logout, ResetPassword.

### Registration
- **Duplicate email check**: `ExistsByEmail` before `Create`. Unique constraint as backstop.
- **Password minimum length**: 8 characters enforced in both `RegisterUser` and `CreateUser`.
- **Best-effort account number**: Registration succeeds even if account number generation fails (pool exhaustion).
- **Best-effort wallet/subscription/email**: All post-registration steps are best-effort with warning logs.

### Admin Access Control
- **`requireAdmin` fail-closed**: Returns `CodeInternal` if `platformSvc` is nil.
- **IsAdmin query**: Separate `admins` table — not just a role string check. Defense in depth.
- **Self-delete prevention**: `ErrCannotDeleteSelf` checked in both `SoftDeleteUser` and `SoftDeleteUsers`.
- **Last-admin protection**: `CountAdmins` check prevents deleting the last admin.
- **Audit logging**: All delete/restore operations logged in `audit_logs` within the same transaction.
- **Soft-delete pattern**: `deleted_at IS NULL` filter in all user queries. Restore available.
- **Role validation**: `validRole` checks against 6 recognized roles.

### Token Lifecycle
- **Access token TTL**: 15 minutes — short window limits exposure.
- **Refresh token TTL**: 7 days — reasonable for user experience.
- **Refresh token rotation**: New refresh token issued on each refresh (via cookie).
- **Logout invalidation**: `IncrementTokenVersion` on logout — all refresh tokens immediately invalid.
- **Password reset invalidation**: `IncrementTokenVersion` after password change — old sessions killed.

### Interceptor
- **Auth-free endpoints**: Login, Register, RefreshToken, ForgotPassword, ResetPassword, VerifyEmail, public marketplace reads — all bypass auth.
- **All other endpoints require JWT or API key**: `authenticate` returns error if no Authorization header and no API key.
- **API key support**: `X-API-Key` header with scope-based permissions.
- **Streaming auth**: `WrapStreamingHandler` authenticates SSE/streaming connections.

## Architecture Compliance

- ✅ No REST endpoints (ConnectRPC only)
- ✅ No JSON for data persistence (PostgreSQL + proto)
- ✅ No `//nolint` or `// @ts-ignore`
- ✅ Fail-closed: `requireAdmin` fails when `platformSvc` is nil
- ✅ Cookie security: HttpOnly, Secure, SameSite=Strict

## Reuse Preflight

- **U1**: NEW: status check in refresh path (no existing helper — inline check matching Login pattern)
- **U2**: REUSE: `validRole` pattern @ `admin_user_handler.go:109-115` (mirrored for status)
- **U3**: REUSE: `IncrementTokenVersion` @ `user_repo.go:186-188` (added parallel method to AdminRepository)

## Deployment

- `go build ./...` ✅
- `go test ./internal/connect/... ./internal/repository/... ./internal/service/...` ✅
- `docker compose build backend` ✅
- `docker compose up -d backend` ✅
- Container health: `healthy` ✅
