
import { createConnectTransport } from '@connectrpc/connect-web';
import { ConnectError, Code, type Interceptor } from '@connectrpc/connect';
import { useAuthStore } from '@/stores/authStore';
import { ensureFreshToken, refreshAccessToken } from '@/utils/tokenLifecycle';
import { isAuthFreeProcedure, getLang, handleTransportError } from './transportErrorHandler';

const envApiUrl = import.meta.env.VITE_API_URL as string | undefined;
const envStreamUrl = import.meta.env.VITE_STREAM_URL as string | undefined;

const defaultApiUrl = (() => {
  if (typeof window === 'undefined') return 'http://127.0.0.1:8080';
  return window.location.origin;
})();

const rawApiUrl = envApiUrl || defaultApiUrl;
const API_URL = rawApiUrl.replace(/\/+$/, '');

const rawStreamUrl = envStreamUrl || API_URL;
const STREAM_URL = rawStreamUrl.replace(/\/+$/, '');


// Token refresh + lifecycle now lives in @/utils/tokenLifecycle.
// transport.ts only:
//   - calls ensureFreshToken() before each authed request (proactive)
//   - falls back to refreshAccessToken() inside the 401 retry path (reactive safety net)

function getAccessToken(): string | null {
  return useAuthStore.getState().accessToken;
}

function procedureHint(req: unknown): { key: string; label: string } {
  const r = req as {
    service?: { typeName?: string };
    method?: { name?: string };
    url?: string;
    spec?: { procedure?: string };
  };
  const svc = String(r.service?.typeName || '').trim();
  const method = String(r.method?.name || '').trim();
  const label = (svc && method ? `${svc}.${method}` : svc || method).trim();
  const fallback = String(r.url || r.spec?.procedure || '').toLowerCase();
  const key = (label || fallback).toLowerCase();
  return { key, label };
}

const interceptors: Interceptor[] = [
  // Reactive 401 safety net — runs first so it can retry once after a refresh.
  // With ensureFreshToken() preflight in the next interceptor, this path
  // should rarely be hit (server restart, secret rotation, clock skew, etc.).
  (next) => async (req) => {
    try {
      return await next(req);
    } catch (error: unknown) {
      if (error instanceof ConnectError && error.code === Code.Unauthenticated) {
        const proc = procedureHint(req).key;
        if (isAuthFreeProcedure(proc)) {
          throw error;
        }
        // Skip refresh retry if user is already not authenticated (refresh just failed).
        if (!useAuthStore.getState().isAuthenticated) {
          throw error;
        }
        const newToken = await refreshAccessToken();
        if (newToken) {
          req.header.set('Authorization', `Bearer ${newToken}`);
          return next(req);
        }
      }
      throw error;
    }
  },
  (next) => async (req) => {
    const { key: proc, label: procLabel } = procedureHint(req);
    const isAuthFree = isAuthFreeProcedure(proc);

    let token = getAccessToken();
    if (!isAuthFree) {
      token = await ensureFreshToken();
    }
    if (token && !isAuthFree) {
      req.header.set('Authorization', `Bearer ${token}`);
    }

    const lang = getLang();
    if (lang) {
      req.header.set('Accept-Language', lang);
    }

    try {
      return await next(req);
    } catch (error: unknown) {
      handleTransportError(error, proc, procLabel);
      throw error; // unreachable — handleTransportError always throws
    }
  },
];

export const transport = createConnectTransport({
  baseUrl: API_URL,
  useBinaryFormat: true,
  interceptors,
});

export const streamTransport = createConnectTransport({
  baseUrl: STREAM_URL,
  useBinaryFormat: true,
  interceptors,
});
