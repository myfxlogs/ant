import { createConnectTransport } from '@connectrpc/connect-web';
import { ConnectError, Code, type Interceptor } from '@connectrpc/connect';
import { Modal, message } from 'antd';
import i18n from '@/i18n';
import { useAuthStore } from '@/stores/authStore';
import { isLikelyStreamTransportFailure, isStreamServiceProcedure } from '@/utils/streamErrors';
import { translateMaybeI18nKey } from '@/utils/error';
import { ensureFreshToken, refreshAccessToken } from '@/utils/tokenLifecycle';

const envApiUrl = import.meta.env.VITE_API_URL as string | undefined;
const envStreamUrl = import.meta.env.VITE_STREAM_URL as string | undefined;

const defaultApiUrl = (() => {
  if (typeof window === 'undefined') return 'http://127.0.0.1:8080';
  return window.location.origin;
})();

const rawApiUrl = envApiUrl || defaultApiUrl;
const API_URL = rawApiUrl.replace(/\/+$/, '');

/** Same origin Connect base URL; also used for EventSource (debate v2 advance jobs). */
export const apiBaseUrl = API_URL;

const rawStreamUrl = envStreamUrl || API_URL;
const STREAM_URL = rawStreamUrl.replace(/\/+$/, '');

let hasShownConnectionError = false;
let lastBizErrorAt = 0;

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
  const label = String(r.service?.typeName || r.method?.name || '').trim();
  const key = String(r.service?.typeName || r.method?.name || r.url || r.spec?.procedure || '').toLowerCase();
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
        if (proc.includes('authservice') && (proc.includes('refreshtoken') || proc.includes('login') || proc.includes('register'))) {
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
    const proc = procedureHint(req).key;
    const isAuthFree = proc.includes('authservice') && (proc.includes('login') || proc.includes('register'));

    // Proactive preflight: if the access token is expired or about to expire,
    // refresh BEFORE issuing the request so the server never sees a 401 and
    // the browser DevTools never logs a red error line.
    let token = getAccessToken();
    if (token && !isAuthFree) {
      token = await ensureFreshToken();
    }
    if (token && !isAuthFree) {
      req.header.set('Authorization', `Bearer ${token}`);
    }

    const lang = i18n.language || 'en';
    if (lang) {
      req.header.set('Accept-Language', lang);
    }

    try {
      return await next(req);
    } catch (error: unknown) {
      if (error instanceof ConnectError && error.code === 12) {
        throw error; // unimplemented
      }

      if (error instanceof Error && (error.message.includes('aborted') || error.message.includes('abort'))) {
        throw error;
      }

      if (error instanceof Error && error.message.includes('Failed to fetch')) {
        if (!hasShownConnectionError) {
          hasShownConnectionError = true;
          Modal.error({
            title: i18n.t('errors.connection_failed.title'),
            content: i18n.t('errors.connection_failed.content'),
            centered: true,
            okText: i18n.t('common.confirm'),
            onOk: () => { hasShownConnectionError = false; },
          });
        }
      } else {
        if (isStreamServiceProcedure(proc) && isLikelyStreamTransportFailure(error)) {
          throw error;
        }
        const now = Date.now();
        if (now - lastBizErrorAt > 800) {
          lastBizErrorAt = now;
          const procName = procedureHint(req).label;
          const rawMsg = error instanceof ConnectError ? String(error.rawMessage ?? '').trim() : '';
          const msgPart =
            error instanceof ConnectError
              ? String(error.message || '').trim()
              : error instanceof Error
                ? String(error.message || '').trim()
                : String(error).trim();
          // Translate i18n keys from the backend (e.g. "errors.user_not_found").
          const translated = translateMaybeI18nKey(rawMsg, '')
            || translateMaybeI18nKey(msgPart, '')
            || msgPart;
          const content = translated.trim();
          if (content) {
            message.error(procName ? `${procName}: ${content}` : content);
          }
        }
      }
      throw error;
    }
  },
];

export const transport = createConnectTransport({
  baseUrl: API_URL,
  interceptors,
});

export const streamTransport = createConnectTransport({
  baseUrl: STREAM_URL,
  interceptors,
});
