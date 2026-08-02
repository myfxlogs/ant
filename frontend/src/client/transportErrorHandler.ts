import { ConnectError, Code } from '@connectrpc/connect';
import { Modal, message } from 'antd';
import i18n from '@/i18n';
import { isLikelyStreamTransportFailure, isStreamAuthFailure, isStreamServiceProcedure } from '@/utils/streamErrors';
import { AI_INSUFFICIENT_BALANCE } from '@/utils/aiErrorCodes';
import { translateMaybeI18nKey, getConnectErrorMessage, connectCodeToI18nKey } from '@/utils/error';
import { TOKEN_EXPIRED_KEY } from '@/gen/ant/v1/i18n/errors_keys';
import { useAuthStore } from '@/stores/authStore';

let hasShownConnectionError = false;
let hasShownBalanceError = false;
let lastBizErrorAt = 0;

const AUTH_FREE_KEYWORDS = ['login', 'register', 'refreshtoken', 'verifyemail', 'resendverification', 'forgotpassword', 'resetpassword', 'verifymtidentity'];

export function isAuthFreeProcedure(proc: string): boolean {
  if (!proc.includes('authservice')) return false;
  return AUTH_FREE_KEYWORDS.some(kw => proc.includes(kw));
}

export function getLang(): string {
  try { const v = localStorage.getItem('alphaforge_lang'); if (v) return v; } catch { /* ignore */ }
  return i18n.language || 'en';
}

function handleInsufficientBalance(error: ConnectError): boolean {
  if (!(error.code === 9 && error.message.includes(AI_INSUFFICIENT_BALANCE))) return false;
  if (!hasShownBalanceError) {
    hasShownBalanceError = true;
    Modal.error({
      title: i18n.t('errors.ai.insufficient_balance_title', { defaultValue: 'Insufficient Balance' }),
      content: i18n.t('errors.ai.insufficient_balance', { defaultValue: 'Your AI wallet balance is insufficient. Please top up before continuing.' }),
      onOk: () => { hasShownBalanceError = false; },
    });
  }
  return true;
}

function handleConnectionError(error: Error): void {
  if (!error.message.includes('Failed to fetch')) return;
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
}

function showBizError(error: unknown, procName: string): void {
  const now = Date.now();
  if (now - lastBizErrorAt <= 800) return;
  lastBizErrorAt = now;
  const rawMsg = error instanceof ConnectError ? String(error.rawMessage ?? '').trim() : '';
  const msgPart = error instanceof ConnectError
    ? String(error.message || '').trim()
    : error instanceof Error
      ? String(error.message || '').trim()
      : String(error).trim();
  const translated = translateMaybeI18nKey(rawMsg, '') || translateMaybeI18nKey(msgPart, '') || msgPart;
  const content = (error instanceof ConnectError && connectCodeToI18nKey[error.code])
    ? getConnectErrorMessage(error.code, translated.trim())
    : translated.trim();
  if (content) {
    message.error(procName ? `${procName}: ${content}` : content);
  }
}

function shouldSuppressError(error: unknown, proc: string): boolean {
  if (error instanceof ConnectError && (error.code === Code.InvalidArgument || error.code === Code.AlreadyExists)) return true;
  if (isAuthFreeProcedure(proc)) return true;
  if (error instanceof ConnectError && error.code === Code.Unauthenticated && !useAuthStore.getState().isAuthenticated) return true;
  return false;
}

export function handleTransportError(error: unknown, proc: string, procLabel: string): void {
  if (error instanceof ConnectError && handleInsufficientBalance(error)) throw error;
  if (error instanceof ConnectError && error.code === 12) throw error;
  if (error instanceof Error && (error.message.includes('aborted') || error.message.includes('abort'))) throw error;

  if (error instanceof Error && error.message.includes('Failed to fetch')) {
    handleConnectionError(error);
  } else {
    if (handleStreamError(error, proc)) throw error;
    if (shouldSuppressError(error, proc)) throw error;
    showBizError(error, procLabel);
  }
  throw error;
}

function handleStreamError(error: unknown, proc: string): boolean {
  if (!isStreamServiceProcedure(proc)) return false;
  if (isLikelyStreamTransportFailure(error)) return true;
  if (isStreamAuthFailure(error)) {
    message.error(i18n.t(TOKEN_EXPIRED_KEY));
    return true;
  }
  return false;
}
