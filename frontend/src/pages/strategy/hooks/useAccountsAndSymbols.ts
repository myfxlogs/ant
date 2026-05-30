import { useState, useCallback } from 'react';
import { accountApi } from '@/client/account';
import { marketApi, type SymbolInfo } from '@/client/market';

interface AccountLike { id: string; [key: string]: unknown; }
type SymbolOption = { value: string; label: string };

export function useAccountsAndSymbols() {
  const [accounts, setAccounts] = useState<AccountLike[]>([]);
  const [symbols, setSymbols] = useState<SymbolOption[]>([]);
  const [symbolsLoading, setSymbolsLoading] = useState(false);

  const fetchAccounts = useCallback(async () => {
    try {
      const data = await accountApi.list();
      setAccounts((data as AccountLike[]) || []);
    } catch {
      setAccounts([]);
    }
  }, []);

  const loadSymbols = useCallback(async (accountId: string) => {
    if (!accountId) { setSymbols([]); return; }
    setSymbolsLoading(true);
    try {
      const list = await marketApi.getSymbols(accountId);
      const seen = new Set<string>();
      const opts = (list || [])
        .map((s: SymbolInfo) => String(s?.symbol || '').trim())
        .filter((v) => v)
        .filter((v) => { if (seen.has(v)) return false; seen.add(v); return true; })
        .map((v) => ({ value: v, label: v }));
      setSymbols(opts);
    } catch {
      setSymbols([]);
    } finally {
      setSymbolsLoading(false);
    }
  }, []);

  return { accounts, symbols, symbolsLoading, fetchAccounts, loadSymbols };
}
