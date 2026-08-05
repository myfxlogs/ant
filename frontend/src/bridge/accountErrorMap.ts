import i18n from '@/i18n';
import {
  BIND_ERRORS_BROKER_UNAVAILABLE_KEY,
  BIND_ERRORS_CONNECTION_FAILED_KEY,
  BIND_ERRORS_INVALID_CREDENTIALS_KEY,
  BIND_ERRORS_TIMEOUT_KEY,
} from '@/gen/ant/v1/i18n/accounts_keys';

/**
 * Maps raw MT gateway connection errors to user-friendly i18n messages.
 * Falls back to the raw message when no pattern matches.
 */
export function toFriendlyAccountError(raw: string): string {
  const lower = raw.toLowerCase();

  // Invalid account / wrong password
  if (
    lower.includes('invalid account') ||
    lower.includes('code=65') ||
    lower.includes('invalid_credentials') ||
    lower.includes('wrong password') ||
    lower.includes('invalid password') ||
    lower.includes('not authorized')
  ) {
    return i18n.t(BIND_ERRORS_INVALID_CREDENTIALS_KEY);
  }

  // Broker unreachable / connection refused
  if (
    lower.includes('connection refused') ||
    lower.includes('no route to host') ||
    lower.includes('no such host') ||
    lower.includes('dial tcp') ||
    lower.includes('econnrefused')
  ) {
    return i18n.t(BIND_ERRORS_BROKER_UNAVAILABLE_KEY);
  }

  // Timeout
  if (
    lower.includes('timeout') ||
    lower.includes('deadline exceeded') ||
    lower.includes('context deadline')
  ) {
    return i18n.t(BIND_ERRORS_TIMEOUT_KEY);
  }

  // Generic connection failure
  if (
    lower.includes('connect:') ||
    lower.includes('dial:') ||
    lower.includes('login:')
  ) {
    return i18n.t(BIND_ERRORS_CONNECTION_FAILED_KEY);
  }

  // Unknown — show raw message
  return raw;
}
