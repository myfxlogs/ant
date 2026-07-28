import { CLIENT_ERRORS_CONTENT_BLOCKED_KEY, CLIENT_ERRORS_CONTEXT_TOO_LONG_KEY, CLIENT_ERRORS_EDGE_GATEWAY_TIMEOUT_KEY, CLIENT_ERRORS_FORBIDDEN_KEY, CLIENT_ERRORS_INSUFFICIENT_BALANCE_KEY, CLIENT_ERRORS_INVALID_MODEL_ID_KEY, CLIENT_ERRORS_NETWORK_UNREACHABLE_KEY, CLIENT_ERRORS_PROVIDER_INTERNAL_ERROR_KEY, CLIENT_ERRORS_RATE_LIMITED_KEY, CLIENT_ERRORS_REGION_NOT_SUPPORTED_KEY, CLIENT_ERRORS_REQUEST_FAILED_KEY, CLIENT_ERRORS_UNAUTHORIZED_KEY } from '@/gen/ant/v1/i18n/ai_core_keys';

import i18n from '@/i18n';

export function unwrapProviderMessage(raw: string): string {
  const start = raw.indexOf('{');
  if (start < 0) return raw;
  const body = raw.slice(start);
  try {
    const obj = JSON.parse(body) as { error?: { message?: unknown } | unknown; message?: unknown };
    const errObj = typeof obj?.error === 'object' && obj?.error !== null ? obj.error as { message?: unknown } : null;
    const inner = errObj?.message ?? obj?.message ?? obj?.error ?? '';
    const innerStr = typeof inner === 'string' ? inner : typeof inner === 'object' && inner !== null ? JSON.stringify(inner) : '';
    return innerStr.trim() || raw;
  } catch {
    return raw;
  }
}

export function pickErrorText(raw: unknown): string {
  if (raw == null) return '';
  if (typeof raw === 'string') return raw;
  if (typeof raw === 'number' || typeof raw === 'boolean') return String(raw);
  if (raw instanceof Error) return raw.message;
  if (typeof raw === 'object') {
    const o = raw as { rawMessage?: unknown; message?: unknown };
    if (typeof o.rawMessage === 'string' && o.rawMessage.trim()) return o.rawMessage;
    if (typeof o.message === 'string' && o.message.trim()) return o.message;
  }
  return String(raw);
}

interface ErrorPattern {
  i18nKey: string;
  lower?: string[];
  raw?: string[];
  regex?: RegExp;
  compound?: (lower: string, msg: string) => boolean;
  extractModel?: boolean;
}

const PATTERNS: ErrorPattern[] = [
  {
    i18nKey: CLIENT_ERRORS_INSUFFICIENT_BALANCE_KEY,
    lower: [
      'insufficient_quota', 'insufficient quota', 'insufficient_balance', 'insufficient balance',
      'insufficient credits', 'never purchased credits', 'purchase more at', 'status 402', ' 402',
      'credit_balance_too_low', 'exceeded your current quota', 'exceeded your quota',
      'billing_not_active', 'arrearage', 'overdue-payment', 'overdue payment',
      'product is not activated', 'product not activated', 'not activated, please confirm',
    ],
    raw: ['余额不足', '額度不足', '账户欠费', '帳號欠費', '试用已结束', '试用额度', '試用已結束', '未开通', '未激活', '未開通', '尚未开通', '请先开通'],
    compound: (lower) => (lower.includes('please activate') && lower.includes('product')),
  },
  {
    i18nKey: CLIENT_ERRORS_RATE_LIMITED_KEY,
    lower: ['status 429', ' 429', 'too many requests', 'rate_limit', 'rate limit', 'tpm limit', 'rpm limit'],
    raw: ['请求过于频繁', '限流'],
  },
  {
    i18nKey: CLIENT_ERRORS_UNAUTHORIZED_KEY,
    lower: ['invalid_api_key', 'invalid api key', 'unauthorized', ' 401', 'status 401'],
    raw: ['密钥无效', '鉴权失败'],
  },
  {
    i18nKey: CLIENT_ERRORS_FORBIDDEN_KEY,
    lower: ['forbidden', ' 403', 'status 403'],
  },
  {
    i18nKey: CLIENT_ERRORS_INVALID_MODEL_ID_KEY,
    lower: [
      'model_not_found', 'model not found', 'model does not exist', 'invalid model id',
      'model_deprecated', 'model deprecated',
    ],
    raw: ['模型不存在', '模型已下线', '模型已停用'],
    compound: (lower) => (lower.includes('the model `') && lower.includes('does not exist')),
    extractModel: true,
  },
  {
    i18nKey: CLIENT_ERRORS_CONTEXT_TOO_LONG_KEY,
    lower: ['context_length_exceeded', 'maximum context length', 'request too large', 'payload too large'],
    raw: ['上下文超长', '内容过长'],
    compound: (lower) => (lower.includes('context length') && lower.includes('exceed')),
  },
  {
    i18nKey: CLIENT_ERRORS_CONTENT_BLOCKED_KEY,
    lower: ['content_filter', 'content policy', 'safety_block'],
    raw: ['内容审核', '内容违规', '敏感内容'],
    compound: (lower) => (lower.includes('blocked') && lower.includes('safety')),
  },
  {
    i18nKey: CLIENT_ERRORS_REGION_NOT_SUPPORTED_KEY,
    lower: ['not supported in your region', 'country, region', 'unsupported_country_region_territory'],
  },
  {
    i18nKey: CLIENT_ERRORS_EDGE_GATEWAY_TIMEOUT_KEY,
    regex: /\b524\b|\b523\b|\b522\b|\b521\b|\b520\b/,
  },
  {
    i18nKey: CLIENT_ERRORS_PROVIDER_INTERNAL_ERROR_KEY,
    lower: ['status 5', ' 500', ' 502', ' 503', ' 504', 'overloaded', 'service unavailable', 'internal server error'],
  },
  {
    i18nKey: CLIENT_ERRORS_NETWORK_UNREACHABLE_KEY,
    lower: [
      'context deadline exceeded', 'client.timeout exceeded', 'timeout exceeded while awaiting headers',
      'i/o timeout', 'timeout', 'connection refused', 'no such host', 'dial tcp', 'econnrefused', 'etimedout',
    ],
    compound: (lower) => (lower.includes('failed to send request') && lower.includes('chat/completions')),
  },
];

function matchPattern(p: ErrorPattern, lower: string, msg: string): boolean {
  if (p.lower?.some(s => lower.includes(s))) return true;
  if (p.raw?.some(s => msg.includes(s))) return true;
  if (p.regex?.test(lower)) return true;
  if (p.compound?.(lower, msg)) return true;
  return false;
}

export function toFriendlyAIError(raw: unknown): string {
  const rawMsg = pickErrorText(raw).trim();
  if (!rawMsg) return i18n.t(CLIENT_ERRORS_REQUEST_FAILED_KEY);
  const msg = unwrapProviderMessage(rawMsg);
  const lower = msg.toLowerCase();

  for (const p of PATTERNS) {
    if (matchPattern(p, lower, msg)) {
      if (p.extractModel) {
        const m = msg.match(/(?:Invalid model id|model `?)([\w./:-]+)/i);
        const model = m?.[1] ? `（${m[1]}）` : '';
        return i18n.t(p.i18nKey, { model });
      }
      return i18n.t(p.i18nKey);
    }
  }

  return msg;
}

export const toFriendlyAIChatError = toFriendlyAIError;
