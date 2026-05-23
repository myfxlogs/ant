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

const interceptors: Interceptor[] = [
  (next) => async (req) => {
    const proc = procedureHint(req).key;
    const isAuthFree = proc.includes('authservice') && (proc.includes('login') || proc.includes('register'));
    const token = localStorage.getItem('access_token');
    if (token && !isAuthFree) {
      req.header.set('Authorization', `Bearer ${token}`);
    }

    // Attach current UI language so backend can localize responses (e.g. strategy templates).
    const lang = i18n.language || 'en';
    if (lang) {
      req.header.set('Accept-Language', lang);
    }

    try {
      return await next(req);
    } catch (error: unknown) {
      // Ignore abort errors (normal when canceling streams)
      if (error instanceof Error && (error.message.includes('aborted') || error.message.includes('abort'))) {
        throw error;
      }

      // 检测连接错误并显示居中弹窗
      if (error instanceof Error && error.message.includes('Failed to fetch')) {
        if (!hasShownConnectionError) {
          hasShownConnectionError = true;
          Modal.error({
            title: i18n.t('errors.connection_failed.title'),
            content: i18n.t('errors.connection_failed.content'),
            centered: true,
            okText: i18n.t('common.confirm'),
            onOk: () => {
              hasShownConnectionError = false;
            },
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
});

export const streamTransport = createConnectTransport({
  baseUrl: STREAM_URL,
  interceptors,
});
