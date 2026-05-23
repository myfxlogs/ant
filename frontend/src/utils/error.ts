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
