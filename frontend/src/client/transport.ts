import { createConnectTransport } from '@connectrpc/connect-web';
import { ConnectError, type Interceptor } from '@connectrpc/connect';
import { Modal, message } from 'antd';
import i18n from '@/i18n';
import { isLikelyStreamTransportFailure, isStreamServiceProcedure } from '@/utils/streamErrors';

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

// Services that are available on the new v2 backend.
// All others silently return empty — no error modals.
const v2Services = new Set([
  'ant.v1.marketplaceservice',
  'ant.v1.mthubservice',
  'ant.v1.marketservice',
]);

/** Narrow Connect request shape for logging / stream heuristics (transport layer only). */
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

// Rewrite old antrader.* → ant.v1.* (M7 proto migration).
function rewriteLegacyPath(url: string): string {
  return url.replace(/\/(antrader\.)/g, '/ant.v1.');
}

// Shared fetch with path rewrite.
function v2Fetch(input: RequestInfo | URL, init?: RequestInit): Promise<Response> {
  if (typeof input === 'string') {
    input = rewriteLegacyPath(input);
  } else if (input instanceof Request) {
    const newUrl = rewriteLegacyPath(input.url);
    if (newUrl !== input.url) {
      input = new Request(newUrl, input);
    }
  }
  return fetch(input, init);
}

const interceptors: Interceptor[] = [
  (next) => async (req) => {
    const proc = procedureHint(req).key;
    const isAuthFree = proc.includes('authservice') && (proc.includes('login') || proc.includes('register'));

    // Silently skip requests for services not yet on v2 backend.
    // Only allow v2-available services + auth (login/register).
    if (!isAuthFree && v2Services.size > 0) {
      const svcPath = proc.replace(/^antrader\./, 'ant.v1.');
      const isV2 = [...v2Services].some(s => svcPath.startsWith(s));
      if (!isV2) {
        // Return an empty-like response: ConnectError with CodeUnimplemented,
        // but suppress the error modal so pages degrade gracefully.
        throw new ConnectError('migrating to ant v2', 12); // CodeUnimplemented = 12
      }
    }

    const token = localStorage.getItem('access_token');
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
        throw error; // unimplemented — don't show error modal
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
          const codePart = error instanceof ConnectError ? `code=${String(error.code)} ` : '';
          const msgPart =
            error instanceof ConnectError
              ? String(error.message || '').trim()
              : error instanceof Error
                ? String(error.message || '').trim()
                : String(error).trim();
          const content = String(rawMsg || `${codePart}${msgPart}` || '').trim();
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
  fetch: v2Fetch,
});

export const streamTransport = createConnectTransport({
  baseUrl: STREAM_URL,
  interceptors,
  fetch: v2Fetch,
});
