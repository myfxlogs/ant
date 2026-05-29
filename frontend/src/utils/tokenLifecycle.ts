/**
 * Proactive access-token lifecycle.
 *
 * Replaces the previous "wait for 401, then refresh" reactive design with a
 * proactive scheme that keeps the access token fresh while the user is alive,
 * and idles silently when they are not.
 *
 * Layers (innermost first):
 *   1. ensureFreshToken()  — request-time preflight; awaits a refresh if the
 *                            current token has < EARLY_REFRESH_MS to live.
 *   2. startTokenScheduler() — periodic check (every 30s) refreshing tokens
 *                              EARLY_REFRESH_MS before expiry, but only when
 *                              the page is visible AND user has been active
 *                              within IDLE_THRESHOLD_MS.
 *   3. visibilitychange handler — on tab returning to visible, eagerly run
 *                                 ensureFreshToken().
 *   4. Reactive 401 interceptor (kept in transport.ts) — last-resort safety
 *      net for unanticipated invalidation (e.g. server restart, secret rotation).
 *
 * The refresh itself is a single-flight POST to /api/auth/refresh that reads
 * the httpOnly refresh_token cookie and returns a new access token.
 */

import { useAuthStore } from '@/stores/authStore';

// Refresh as soon as the access token has less than this many ms left.
const EARLY_REFRESH_MS = 2 * 60 * 1000; // 2 minutes
// Block requests until refresh finishes if exp is closer than this.
const REQUEST_PREFLIGHT_MS = 30 * 1000; // 30 seconds
// Background scheduler tick.
const SCHEDULER_TICK_MS = 30 * 1000; // 30 seconds
// Don't auto-refresh in the background if user has been idle longer than this.
const IDLE_THRESHOLD_MS = 30 * 60 * 1000; // 30 minutes

let refreshPromise: Promise<string | null> | null = null;
let schedulerTimer: number | null = null;
let lastUserActivity = Date.now();
let listenersAttached = false;

/** Decode a JWT and return its `exp` claim in epoch milliseconds, or 0 if unavailable. */
export function getTokenExpiryMs(token: string | null): number {
  if (!token) return 0;
  const parts = token.split('.');
  if (parts.length !== 3) return 0;
  try {
    const payload = JSON.parse(atob(parts[1].replace(/-/g, '+').replace(/_/g, '/')));
    if (typeof payload?.exp !== 'number') return 0;
    return payload.exp * 1000;
  } catch {
    return 0;
  }
}

/** Resolve the URL for the cookie-based refresh endpoint. */
function refreshUrl(): string {
  const envApi = import.meta.env.VITE_API_URL as string | undefined;
  const base = (envApi || (typeof window !== 'undefined' ? window.location.origin : '')).replace(/\/+$/, '');
  return `${base}/api/auth/refresh`;
}

/**
 * Single-flight refresh. All concurrent callers share one network round-trip.
 * Returns the new access token, or null on failure (cookie missing/expired).
 */
export function refreshAccessToken(): Promise<string | null> {
  if (refreshPromise) return refreshPromise;
  refreshPromise = (async () => {
    try {
      const res = await fetch(refreshUrl(), { method: 'POST', credentials: 'include' });
      if (!res.ok) return null;
      const data = await res.json();
      const newAccess: string | undefined = data?.access_token;
      if (!newAccess) return null;
      useAuthStore.getState().setAccessToken(newAccess);
      return newAccess;
    } catch {
      return null;
    } finally {
      refreshPromise = null;
    }
  })();
  return refreshPromise;
}

/**
 * Request-time preflight. Call this immediately before issuing any
 * authenticated request. If the current token is expired or about to expire,
 * we await a refresh (single-flight) and return the new token; otherwise
 * we return the existing token unchanged.
 *
 * Returns null only when there is no token at all OR the refresh failed
 * (caller should then surface a normal Unauthenticated error).
 */
export async function ensureFreshToken(): Promise<string | null> {
  const current = useAuthStore.getState().accessToken;
  if (!current) return null;
  const exp = getTokenExpiryMs(current);
  // If we cannot decode exp, fall back to optimistic use; the reactive 401
  // interceptor will pick up actual invalidation.
  if (exp === 0) return current;
  const remaining = exp - Date.now();
  if (remaining > REQUEST_PREFLIGHT_MS) return current;
  const refreshed = await refreshAccessToken();
  return refreshed ?? null;
}

function bumpUserActivity(): void {
  lastUserActivity = Date.now();
}

function isUserActive(): boolean {
  return Date.now() - lastUserActivity < IDLE_THRESHOLD_MS;
}

async function schedulerTick(): Promise<void> {
  if (typeof document !== 'undefined' && document.visibilityState !== 'visible') return;
  if (!isUserActive()) return;
  const token = useAuthStore.getState().accessToken;
  if (!token) return;
  const exp = getTokenExpiryMs(token);
  if (exp === 0) return;
  const remaining = exp - Date.now();
  if (remaining > EARLY_REFRESH_MS) return;
  await refreshAccessToken();
}

function onVisibilityChange(): void {
  if (typeof document === 'undefined') return;
  if (document.visibilityState !== 'visible') return;
  bumpUserActivity();
  // Eagerly verify token; refresh if near-expiry.
  void ensureFreshToken();
}

/**
 * Install activity listeners + start the periodic refresh scheduler.
 * Idempotent — calling more than once has no effect.
 */
export function startTokenScheduler(): void {
  if (listenersAttached) return;
  listenersAttached = true;

  if (typeof window !== 'undefined') {
    const opts: AddEventListenerOptions = { passive: true };
    window.addEventListener('mousemove', bumpUserActivity, opts);
    window.addEventListener('keydown', bumpUserActivity, opts);
    window.addEventListener('click', bumpUserActivity, opts);
    window.addEventListener('focus', bumpUserActivity, opts);
  }
  if (typeof document !== 'undefined') {
    document.addEventListener('visibilitychange', onVisibilityChange);
  }
  if (typeof window !== 'undefined') {
    schedulerTimer = window.setInterval(() => { void schedulerTick(); }, SCHEDULER_TICK_MS);
  }
}

/** For tests / explicit teardown. */
export function stopTokenScheduler(): void {
  if (!listenersAttached) return;
  listenersAttached = false;
  if (typeof window !== 'undefined') {
    window.removeEventListener('mousemove', bumpUserActivity);
    window.removeEventListener('keydown', bumpUserActivity);
    window.removeEventListener('click', bumpUserActivity);
    window.removeEventListener('focus', bumpUserActivity);
  }
  if (typeof document !== 'undefined') {
    document.removeEventListener('visibilitychange', onVisibilityChange);
  }
  if (schedulerTimer !== null && typeof window !== 'undefined') {
    window.clearInterval(schedulerTimer);
    schedulerTimer = null;
  }
}
