
import { TOKEN_EXPIRED_KEY } from '@/gen/ant/v1/i18n/errors_keys';

import { createConnectTransport } from '@connectrpc/connect-web';
import { ConnectError, Code, type Interceptor } from '@connectrpc/connect';
import { Modal, message } from 'antd';
import i18n from '@/i18n';
import { useAuthStore } from '@/stores/authStore';
import { isLikelyStreamTransportFailure, isStreamAuthFailure, isStreamServiceProcedure } from '@/utils/streamErrors';
import { AI_INSUFFICIENT_BALANCE } from '@/utils/aiErrorCodes';
import { translateMaybeI18nKey, getConnectErrorMessage, connectCodeToI18nKey } from '@/utils/error';
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
let hasShownBalanceError = false;
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
        if (proc.includes('authservice') && (proc.includes('refreshtoken') || proc.includes('login') || proc.includes('register') || proc.includes('verifyemail') || proc.includes('resendverification') || proc.includes('forgotpassword') || proc.includes('resetpassword') || proc.includes('verifymtidentity'))) {
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
    const isAuthFree = proc.includes('authservice') && (proc.includes('login') || proc.includes('register') || proc.includes('refreshtoken') || proc.includes('verifyemail') || proc.includes('resendverification') || proc.includes('forgotpassword') || proc.includes('resetpassword') || proc.includes('verifymtidentity'));

    // Proactive preflight: refresh if token is missing (page reload) or about to expire.
    // Always call ensureFreshToken() for non-auth requests so it can attempt
    // a cookie-based refresh when accessToken is null but user is authenticated.
    let token = getAccessToken();
    if (!isAuthFree) {
      token = await ensureFreshToken();
    }
    if (token && !isAuthFree) {
      req.header.set('Authorization', `Bearer ${token}`);
    }

    const lang = (() => {
      try { const v = localStorage.getItem('alphaforge_lang'); if (v) return v; } catch {}
      return i18n.language || 'en';
    })();
    if (lang) {
      req.header.set('Accept-Language', lang);
    }

    try {
      return await next(req);
    } catch (error: unknown) {
      // Wallet insufficient balance — require user to acknowledge.
      if (error instanceof ConnectError && error.code === 9 && error.message.includes(AI_INSUFFICIENT_BALANCE)) {
        if (!hasShownBalanceError) {
          hasShownBalanceError = true;
          Modal.error({
            title: i18n.t('errors.ai.insufficient_balance_title', { defaultValue: 'Insufficient Balance' }),
            content: i18n.t('errors.ai.insufficient_balance', { defaultValue: 'Your AI wallet balance is insufficient. Please top up before continuing.' }),
            onOk: () => { hasShownBalanceError = false; },
          });
        }
        throw error;
      }

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
        // StreamService auth failure (expired/missing token) — the reactive
        // refresh interceptor already tried and failed; show friendly re-login prompt.
        if (isStreamServiceProcedure(proc) && isStreamAuthFailure(error)) {
          message.error(i18n.t(TOKEN_EXPIRED_KEY));
          throw error;
        }
        // Skip global toast for user-input validation errors — the caller
        // (e.g. BindAccount, AccountDetail) handles display with friendly messages.
        if (error instanceof ConnectError &&
            (error.code === Code.InvalidArgument || error.code === Code.AlreadyExists)) {
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
          // For known ConnectRPC codes, prefer user-friendly localized message.
          const content = (error instanceof ConnectError && connectCodeToI18nKey[error.code])
            ? getConnectErrorMessage(error.code, translated.trim())
            : translated.trim();
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
  useBinaryFormat: true,
  interceptors,
});

export const streamTransport = createConnectTransport({
  baseUrl: STREAM_URL,
  useBinaryFormat: true,
  interceptors,
});
