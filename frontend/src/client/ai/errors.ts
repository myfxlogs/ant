import i18n from '@/i18n';

// Shared mapper that converts a raw provider/gateway error message into a
// localized, human-friendly hint. Covers the most common failure modes we see
// across OpenAI-family, Anthropic, and Chinese cloud providers (DeepSeek,
// Zhipu, Qwen, Doubao, Moonshot, ...).
//
// Intentionally pattern-matches on substrings rather than HTTP status alone
// because most providers wrap their JSON inside a transport error and we only
// see the pre-formatted body string by the time it gets here.

function unwrapProviderMessage(raw: string): string {
  const start = raw.indexOf('{');
  if (start < 0) return raw;
  const body = raw.slice(start);
  try {
    const obj = JSON.parse(body) as { error?: { message?: unknown }; message?: unknown; error?: unknown };
    const inner = obj?.error?.message ?? obj?.message ?? obj?.error ?? '';
    const innerStr = typeof inner === 'string' ? inner : typeof inner === 'object' && inner !== null ? JSON.stringify(inner) : '';
    return innerStr.trim() || raw;
  } catch {
    return raw;
  }
}

function pickErrorText(raw: unknown): string {
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

export function toFriendlyAIError(raw: unknown): string {
  const rawMsg = pickErrorText(raw).trim();
  if (!rawMsg) return i18n.t('ai.client.errors.requestFailed');
  const msg = unwrapProviderMessage(rawMsg);
  const lower = msg.toLowerCase();

  if (
    lower.includes('insufficient_quota') ||
    lower.includes('insufficient quota') ||
    lower.includes('insufficient_balance') ||
    lower.includes('insufficient balance') ||
    lower.includes('insufficient credits') ||
    lower.includes('never purchased credits') ||
    lower.includes('purchase more at') ||
    lower.includes('status 402') ||
    lower.includes(' 402') ||
    lower.includes('credit_balance_too_low') ||
    lower.includes('exceeded your current quota') ||
    lower.includes('exceeded your quota') ||
    lower.includes('billing_not_active') ||
    lower.includes('arrearage') ||
    lower.includes('overdue-payment') ||
    lower.includes('overdue payment') ||
    msg.includes('余额不足') ||
    msg.includes('額度不足') ||
    msg.includes('账户欠费') ||
    msg.includes('帳號欠費') ||
    msg.includes('试用已结束') ||
    msg.includes('试用额度') ||
    msg.includes('試用已結束') ||
    lower.includes('product is not activated') ||
    lower.includes('product not activated') ||
    lower.includes('not activated, please confirm') ||
    (lower.includes('please activate') && lower.includes('product')) ||
    msg.includes('未开通') ||
    msg.includes('未激活') ||
    msg.includes('未開通') ||
    msg.includes('尚未开通') ||
    msg.includes('请先开通')
  ) {
    return i18n.t('ai.client.errors.insufficientBalance');
  }

  if (
    lower.includes('status 429') ||
    lower.includes(' 429') ||
    lower.includes('too many requests') ||
    lower.includes('rate_limit') ||
    lower.includes('rate limit') ||
    lower.includes('tpm limit') ||
    lower.includes('rpm limit') ||
    msg.includes('请求过于频繁') ||
    msg.includes('限流')
  ) {
    return i18n.t('ai.client.errors.rateLimited');
  }

  if (
    lower.includes('invalid_api_key') ||
    lower.includes('invalid api key') ||
    lower.includes('unauthorized') ||
    lower.includes(' 401') ||
    lower.includes('status 401') ||
    msg.includes('密钥无效') ||
    msg.includes('鉴权失败')
  ) {
    return i18n.t('ai.client.errors.unauthorized');
  }
  if (lower.includes('forbidden') || lower.includes(' 403') || lower.includes('status 403')) {
    return i18n.t('ai.client.errors.forbidden');
  }

  if (
    lower.includes('model_not_found') ||
    lower.includes('model not found') ||
    lower.includes('model does not exist') ||
    lower.includes('invalid model id') ||
    (lower.includes('the model `') && lower.includes('does not exist')) ||
    lower.includes('model_deprecated') ||
    lower.includes('model deprecated') ||
    msg.includes('模型不存在') ||
    msg.includes('模型已下线') ||
    msg.includes('模型已停用')
  ) {
    const m = msg.match(/(?:Invalid model id|model `?)([\w./:-]+)/i);
    const model = m?.[1] ? `（${m[1]}）` : '';
    return i18n.t('ai.client.errors.invalidModelId', { model });
  }

  if (
    lower.includes('context_length_exceeded') ||
    lower.includes('maximum context length') ||
    (lower.includes('context length') && lower.includes('exceed')) ||
    lower.includes('request too large') ||
    lower.includes('payload too large') ||
    msg.includes('上下文超长') ||
    msg.includes('内容过长')
  ) {
    return i18n.t('ai.client.errors.contextTooLong');
  }

  if (
    lower.includes('content_filter') ||
    lower.includes('content policy') ||
    lower.includes('safety_block') ||
    (lower.includes('blocked') && lower.includes('safety')) ||
    msg.includes('内容审核') ||
    msg.includes('内容违规') ||
    msg.includes('敏感内容')
  ) {
    return i18n.t('ai.client.errors.contentBlocked');
  }

  if (
    lower.includes('not supported in your region') ||
    lower.includes('country, region') ||
    lower.includes('unsupported_country_region_territory')
  ) {
    return i18n.t('ai.client.errors.regionNotSupported');
  }

  if (/\b524\b/.test(lower) || /\b523\b/.test(lower) || /\b522\b/.test(lower) || /\b521\b/.test(lower) || /\b520\b/.test(lower)) {
    return i18n.t('ai.client.errors.edgeGatewayTimeout');
  }

  if (
    lower.includes('status 5') ||
    lower.includes(' 500') ||
    lower.includes(' 502') ||
    lower.includes(' 503') ||
    lower.includes(' 504') ||
    lower.includes('overloaded') ||
    lower.includes('service unavailable') ||
    lower.includes('internal server error')
  ) {
    return i18n.t('ai.client.errors.providerInternalError');
  }

  if (
    lower.includes('context deadline exceeded') ||
    lower.includes('client.timeout exceeded') ||
    lower.includes('timeout exceeded while awaiting headers') ||
    (lower.includes('failed to send request') && lower.includes('chat/completions')) ||
    lower.includes('i/o timeout') ||
    lower.includes('timeout') ||
    lower.includes('connection refused') ||
    lower.includes('no such host') ||
    lower.includes('dial tcp') ||
    lower.includes('econnrefused') ||
    lower.includes('etimedout')
  ) {
    return i18n.t('ai.client.errors.networkUnreachable');
  }

  return msg;
}

export const toFriendlyAIChatError = toFriendlyAIError;
