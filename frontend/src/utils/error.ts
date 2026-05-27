import i18n from '@/i18n';

interface ApiErrorResponse {
  code: number;
  message: string;
  request_id?: string;
  timestamp?: string;
}

type LegacyResponseError = {
  response?: {
    data?: Partial<ApiErrorResponse>;
  };
};

export interface ConnectionError {
  type: 'CONNECTION_ERROR';
  message: string;
}

// Maps backend error codes to i18n keys (mirrors backend errorMessages map).
const errorCodeToKey: Record<number, string> = {
  0: 'errors.success',
  1: 'errors.unknown',
  2: 'errors.invalid_parameter',
  3: 'errors.unauthorized',
  4: 'errors.forbidden',
  5: 'errors.not_found',
  6: 'errors.internal',
  7: 'errors.rate_limited',
  8: 'errors.service_unavailable',
  9: 'errors.request_timeout',

  1001: 'errors.user_not_found',
  1002: 'errors.user_already_exists',
  1003: 'errors.invalid_password',
  1004: 'errors.token_expired',
  1005: 'errors.token_invalid',
  1006: 'errors.token_missing',
  1007: 'errors.user_disabled',
  1008: 'errors.email_not_verified',
  1009: 'errors.password_too_weak',
  1010: 'errors.old_password_incorrect',

  2001: 'errors.account_not_found',
  2002: 'errors.account_already_bound',
  2003: 'errors.account_connection_failed',
  2004: 'errors.account_disconnected',
  2005: 'errors.account_auth_failed',
  2006: 'errors.account_timeout',
  2007: 'errors.account_limit_exceeded',
  2008: 'errors.invalid_account_type',
  2009: 'errors.account_not_connected',
  2010: 'errors.platform_not_supported',

  3001: 'errors.order_not_found',
  3002: 'errors.order_rejected',
  3003: 'errors.insufficient_margin',
  3004: 'errors.market_closed',
  3005: 'errors.invalid_order_type',
  3006: 'errors.invalid_volume',
  3007: 'errors.invalid_price',
  3008: 'errors.order_timeout',
  3009: 'errors.position_not_found',
  3010: 'errors.cannot_close_position',
  3011: 'errors.cannot_modify_order',
  3012: 'errors.order_already_filled',
  3013: 'errors.order_already_cancelled',
  3014: 'errors.slippage_exceeded',
  3015: 'errors.symbol_not_subscribed',

  4001: 'errors.symbol_not_found',
  4002: 'errors.no_market_data',
  4003: 'errors.subscription_failed',
  4004: 'errors.unsubscription_failed',
  4005: 'errors.quote_not_available',
  4006: 'errors.history_not_available',
  4007: 'errors.invalid_timeframe',
  4008: 'errors.invalid_time_range',

  5001: 'errors.analytics_not_available',
  5002: 'errors.report_generation_failed',
  5003: 'errors.invalid_date_range',
  5004: 'errors.insufficient_data',

  6001: 'errors.admin_access_denied',
  6002: 'errors.operation_forbidden',
  6003: 'errors.audit_log_not_found',

  7001: 'errors.broker_search_failed',
  7002: 'errors.broker_not_found',
  7003: 'errors.broker_server_unavailable',
};

export function getErrorMessageByCode(code: number, fallback?: string): string {
  const key = errorCodeToKey[code];
  if (key) {
    const translated = i18n.t(key);
    if (translated && translated !== key) return translated;
  }
  return fallback ?? i18n.t('errors.unknown');
}

export function translateMaybeI18nKey(msg: unknown, fallback: string): string {
  const trimmed = String(msg ?? '').trim();
  if (!trimmed) return fallback;
  if (trimmed.includes('.') && !trimmed.includes(' ')) {
    const translated = i18n.t(trimmed);
    return translated && translated !== trimmed ? translated : fallback;
  }
  return trimmed;
}

export function isConnectionError(error: unknown): boolean {
  if (error && typeof error === 'object' && 'message' in error) {
    const errorMsg = (error as Error).message;
    return errorMsg.includes('Failed to fetch');
  }
  return false;
}

export function getErrorMessage(error: unknown, defaultMsg: string): string {
  if (error && typeof error === 'object') {
    const responseError = error as LegacyResponseError;
    if (responseError.response?.data?.message) {
      return translateMaybeI18nKey(responseError.response.data.message, defaultMsg);
    }
    if ('message' in error && typeof (error as Error).message === 'string') {
      const errorMsg = (error as Error).message;
      const trimmed = String(errorMsg || '').trim();
      const maybeTranslated = translateMaybeI18nKey(trimmed, defaultMsg);
      if (maybeTranslated !== trimmed) return maybeTranslated;
      if (errorMsg.includes('Failed to fetch')) {
        return i18n.t('errors.connection_failed.title');
      }

      const lower = errorMsg.toLowerCase();
      if (lower.includes('allocationquota.freetieronly') || lower.includes('free tier') || lower.includes('free-tier only')) {
        return i18n.t('errors.ai.free_tier_exhausted');
      }
      if (lower.includes('[resource_exhausted]') || lower.includes('status 429') || lower.includes('too many requests')) {
        return i18n.t('errors.ai.rate_limited');
      }
      if (lower.includes('status 403') && (lower.includes('quota') || lower.includes('exhaust') || lower.includes('allocation'))) {
        return i18n.t('errors.ai.forbidden_quota');
      }

      return errorMsg;
    }
  }
  return defaultMsg;
}
