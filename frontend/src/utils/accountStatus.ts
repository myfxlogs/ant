import type { Account } from '@/types/account';

// Migration 187 replaced the is_disabled column with the account_status state machine.
// An account is disabled when its status is disconnected or frozen.
const disabledStatusValues = new Set(['disconnected', 'frozen']);

export function isTradingAccountEnabled(account: Account | null | undefined): boolean {
  if (!account) return false;
  // Derive from account_status (the source of truth after migration 187).
  const status = String(account.status || account.accountStatus || '').trim().toLowerCase();
  if (disabledStatusValues.has(status)) return false;
  // Legacy: also check isDisabled for backward compat with older proto responses.
  if (account.isDisabled === true) return false;
  return true;
}
