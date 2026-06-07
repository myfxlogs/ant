import { useState, useCallback } from 'react';
import { accountApi } from '@/client/account';
import { marketApi, isMTSessionError, type SymbolInfo } from '@/client/market';

interface AccountLike { id: string; [key: string]: unknown; }
type SymbolOption = { value: string; label: string };

export function useAccountsAndSymbols() {
  const [accounts, setAccounts] = useState<AccountLike[]>([]);
  const [symbols, setSymbols] = useState<SymbolOption[]>([]);
  const [symbolsLoading, setSymbolsLoading] = useState(false);
  const [mtError, setMtError] = useState<string | null>(null);

  const fetchAccounts = useCallback(async () => {
    try {
      const data = await accountApi.list();
      setAccounts((data as AccountLike[]) || []);
    } catch {
      setAccounts([]);
    }
  }, []);

  const loadSymbols = useCallback(async (accountId: string) => {
    if (!accountId) { setSymbols([]); setMtError(null); return; }
    setSymbolsLoading(true);
    setMtError(null);
    try {
      const list = await marketApi.getSymbols(accountId);
      const seen = new Set<string>();
      const opts = (list || [])
        .map((s: SymbolInfo) => String(s?.symbol || '').trim())
        .filter((v) => v)
        .filter((v) => { if (seen.has(v)) return false; seen.add(v); return true; })
        .map((v) => ({ value: v, label: v }));
      setSymbols(opts);
    } catch (e) {
      setSymbols([]);
      if (isMTSessionError(e)) {
        setMtError('MT session not found. The trading terminal may be reconnecting — refresh in a moment.');
      }
    } finally {
      setSymbolsLoading(false);
    }
  }, []);

  return { accounts, symbols, symbolsLoading, mtError, fetchAccounts, loadSymbols };
}
