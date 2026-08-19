---
description: Diagnose and fix frontend "Session expired" / data-loading failures on page reload
---

# Debug Frontend Stream & Auth Issues

## Symptom

- After login everything works; after manual refresh, data does not load or a "Session expired" toast flashes.
- All API requests return 200, no 401, token/cookie look fine.

## Root causes and fixes

### 1. Auth-free endpoints not aligned

Backend `backend/internal/interceptor/auth.go:WrapUnary` and frontend `frontend/src/client/transport.ts:isAuthFree` must both exclude:

- `/refreshtoken`
- `/refreshtokenfromcookie`
- `/verifyemail`
- `/resendverification`

If backend excludes but frontend still tries `ensureFreshToken`, or vice versa, you get 401 or a deadlock (refresh request waits for the token it is fetching).

### 2. `procedureHint` drops the method name

In `frontend/src/client/transport.ts`, the hint must be:

```ts
const procedureHint = `${service.typeName}.${method.name}`.toLowerCase();
```

Using `||` or omitting the method makes the key just the service name (e.g. `ant.v1.authservice`), so `isAuthFree` checks like `proc.includes('refreshtoken')` fail.

### 3. `"missing request message"` misclassified as auth failure

When a server-stream (`SubscribeEvents`, `SubscribeUserSummary`) is aborted by the browser on page refresh, connect-web may emit the **string** `"missing request message"`.

- `typeof error === 'string'`
- `isConnErr === false`, `isErr === false`

`utils/streamErrors.ts` must **not** include `'missing request message'` in `isStreamAuthFailure()`. Move it to `isLikelyStreamTransportFailure()` so refresh aborts are treated as transport errors (silent / retry) instead of token expired.

### 4. Token hydration race

- `authStore` `partialize` only persists `user`, not `accessToken`.
- On reload, wait for Zustand rehydration (`persist.hasHydrated()` / `onFinishHydration()`) before deciding `isAuthenticated`.

## Reproducing with tests

The "Session expired" toast is transient; use Playwright polling:

```ts
for (let i = 0; i < 16; i++) {
  const notices = page.locator('.ant-message-notice');
  if (await notices.count() > 0) return true;
  await page.waitForTimeout(500);
}
```

## Diagnostic logs

Add logging in `transport.ts` interceptor and `utils/streamErrors.ts`:

```ts
console.log({ str: String(error), typeof: typeof error, isStreamAuth: isStreamAuthFailure(error) });
```

## Reference files

- `frontend/src/client/transport.ts`
- `frontend/src/client/tokenLifecycle.ts`
- `frontend/src/utils/streamErrors.ts`
- `backend/internal/interceptor/auth.go`
