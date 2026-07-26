# Frontend Security Audit Report (W2)

## Scope

- `frontend/src/stores/authStore.ts` — Token persistence
- `frontend/src/client/transport.ts` — Request interceptors, auth handling
- `frontend/src/utils/tokenLifecycle.ts` — Token refresh lifecycle
- `frontend/src/utils/streamErrors.ts` — SSE error classification
- `frontend/src/utils/getAccessToken.ts` — Token access utility
- `frontend/src/client/auth.ts` — Auth API client
- `frontend/src/hooks/useAuth.ts` — Auth hook
- `frontend/nginx.conf` — Frontend container nginx
- `nginx/nginx.conf` — External reverse proxy nginx
- `backend/internal/server/server.go` — HTTP server config
- `backend/internal/interceptor/auth.go` — Auth interceptor
- `backend/internal/connect/user/auth_token.go` — Cookie configuration

## Findings

### W2-1 — access_token accepted via URL query parameter 🟢 LOW

**File**: `backend/internal/interceptor/auth.go:112`

**Problem**: `UserIDFromHTTP` accepts `access_token` from URL query parameters as a fallback for EventSource (which cannot set custom headers). URL query parameters are logged in nginx access logs, browser history, and proxy logs, exposing the token.

**Status**: The function is defined but never called (dead code). No handler currently uses this path. Risk is latent, not active.

**Recommendation**: Remove `UserIDFromHTTP` or replace query-param auth with a short-lived single-use token exchange if EventSource support is needed in the future.

### W2-2 — HTTP server missing ReadHeaderTimeout 🟢 LOW

**File**: `backend/internal/server/server.go:15-21`

**Problem**: The HTTP server had `ReadTimeout: 15s` but no `ReadHeaderTimeout`. While `ReadTimeout` provides partial protection, `ReadHeaderTimeout` is the specific mitigation against Slowloris attacks (slow header sending to exhaust connections).

**Fix**: Added `ReadHeaderTimeout: 10 * time.Second`.

### W2-3 — Missing HSTS and CSP headers in external nginx 🟢 LOW

**File**: `nginx/nginx.conf:57-62`

**Problem**: The external nginx config had `X-Content-Type-Options`, `X-Frame-Options`, `X-XSS-Protection`, `Referrer-Policy`, and `Permissions-Policy` headers, but was missing:
- `Strict-Transport-Security` (HSTS) — critical for HTTPS deployments to prevent protocol downgrade
- `Content-Security-Policy` (CSP) — restricts resource loading to prevent XSS data exfiltration

**Fix**: Added HSTS (`max-age=31536000; includeSubDomains`) and CSP (`default-src 'self'` with `unsafe-inline`/`unsafe-eval` for script-src needed by Vite dev + build, `frame-ancestors 'none'`).

## Verified Safe (No Issues Found)

- **Token storage**: `accessToken` stored in memory only (Zustand state), NOT persisted to localStorage. `partialize` only saves `user` object. Refresh token in httpOnly cookie.
- **Cookie security**: `HttpOnly; SameSite=Strict; Secure` (Secure disabled only in dev mode). `CookieSecure` config defaults to `true`.
- **XSS vectors**: Only one `dangerouslySetInnerHTML` usage (SmartTuningPanel.tsx:135) — input is an i18n translation key with a numeric interpolation, no user-controlled content. No `eval()`, `new Function()`, `innerHTML`, or `document.write` usage.
- **Password fields**: All password inputs use `Input.Password` (masked). Passwords stored in React state, sent via ConnectRPC POST body (proto serialized), never in URL params.
- **API key handling**: AI settings API key uses `Input.Password`. Backend stores encrypted. Admin gateway table shows only `hasApiKey` boolean, not the key itself.
- **localStorage usage**: Only non-sensitive data (theme, language, watchlist, backtest defaults, onboarding dismiss flag, LLM translation cache). No tokens, passwords, or API keys.
- **SSE error handling**: `isStreamAuthFailure` correctly excludes "missing request message" (transport abort), preventing false "session expired" toasts on page refresh.
- **Metrics/Prometheus/AlertManager endpoints**: Only proxied by frontend nginx (internal container), not by external nginx (public entry point). Not reachable from the internet.
- **No CORS configuration**: Same-origin only (frontend served by nginx, API proxied). No `Access-Control-Allow-Origin` needed or set.
- **Source size limits**: MQL/Python strategy source limited to 500KB (`MaxSourceSize`).
- **HTTP response body limits**: External API responses (ZhipuAI, DeepSeek) read with `io.LimitReader`.

## Architecture Compliance

- ✅ No WebSocket (ConnectRPC + SSE only)
- ✅ No JSON for data persistence (proto for API, PG for storage)
- ✅ No `// @ts-ignore` or `# noqa`

## Reuse Preflight

- **W2-2**: NEW: `ReadHeaderTimeout` field on `http.Server` struct (Go stdlib, no custom code)
- **W2-3**: NEW: `Strict-Transport-Security` and `Content-Security-Policy` headers (nginx config, no code)

## Migrations

No migrations required.

## Deployment

- `go build ./...` ✅
- `go test ./internal/mthub/... ./internal/connect/... ./internal/risk/...` ✅
- nginx config changes require `docker compose restart nginx` (or rebuild nginx container)
