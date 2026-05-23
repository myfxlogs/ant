const disabledStatusValues = new Set(['disabled', 'disable', 'inactive', 'frozen', 'blocked', 'deleted']);

export function isTradingAccountEnabled(account: any): boolean {
  if (!account) return false;
  if (account.isDisabled === true || account.is_disabled === true) return false;
  const status = String(account.status || account.accountStatus || account.account_status || '').trim().toLowerCase();
  return !disabledStatusValues.has(status);
}
