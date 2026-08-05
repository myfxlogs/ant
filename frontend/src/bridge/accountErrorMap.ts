import i18n from '@/i18n';
import {
  BIND_ERRORS_BROKER_UNAVAILABLE_KEY,
  BIND_ERRORS_CONNECTION_FAILED_KEY,
  BIND_ERRORS_INVALID_CREDENTIALS_KEY,
  BIND_ERRORS_TIMEOUT_KEY,
} from '@/gen/ant/v1/i18n/accounts_keys';

const ERROR_PATTERNS: { patterns: string[]; key: string }[] = [
  {
    patterns: ['invalid account', 'code=65', 'invalid_credentials', 'wrong password', 'invalid password', 'not authorized'],
    key: BIND_ERRORS_INVALID_CREDENTIALS_KEY,
  },
  {
    patterns: ['connection refused', 'no route to host', 'no such host', 'dial tcp', 'econnrefused'],
    key: BIND_ERRORS_BROKER_UNAVAILABLE_KEY,
  },
  {
    patterns: ['timeout', 'deadline exceeded', 'context deadline'],
    key: BIND_ERRORS_TIMEOUT_KEY,
  },
  {
    patterns: ['connect:', 'dial:', 'login:'],
    key: BIND_ERRORS_CONNECTION_FAILED_KEY,
  },
];

export function toFriendlyAccountError(raw: string): string {
  const lower = raw.toLowerCase();
  for (const group of ERROR_PATTERNS) {
    if (group.patterns.some(p => lower.includes(p))) {
      return i18n.t(group.key);
    }
  }
  return raw;
}
