import type { Account } from '@/types/account';

const disabledStatusValues = new Set(['disabled', 'disable', 'inactive', 'frozen', 'blocked', 'deleted']);

export function isTradingAccountEnabled(account: Account | null | undefined): boolean {
  if (!account) return false;
  if (account.isDisabled === true) return false;
  const status = String(account.status || account.accountStatus || '').trim().toLowerCase();
  return !disabledStatusValues.has(status);
}
