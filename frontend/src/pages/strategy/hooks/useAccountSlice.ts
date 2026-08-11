import { useMemo, useCallback } from 'react';
import { useAccount } from '@/hooks/useAccount';
import { marketApi } from '@/client/market';
import { useWorkspaceStore } from '@/stores/workspaceStore';
import type { AccountMeta } from '@/components/chart/QuickTradePanel';

export interface AccountSlice {
  activeAccounts: ReturnType<typeof useAccount>['accounts'];
  accountId: string;
  setAccountId: (v: string) => void;
  symbol: string;
  setSymbol: (v: string) => void;
  timeframe: string;
  setTimeframe: (v: string) => void;
  handleAccountChange: (id: string) => void;
  selectedAccountMeta: AccountMeta | null;
  fetchAccounts: () => void;
}

export function useAccountSlice(): AccountSlice {
  const { accounts: allAccounts, fetchAccounts } = useAccount();
  const activeAccounts = useMemo(() => (allAccounts || []).filter(a => !a.isDisabled), [allAccounts]);
  const accountId = useWorkspaceStore(s => s.accountId);
  const setAccountId = useWorkspaceStore(s => s.setAccountId);
  const symbol = useWorkspaceStore(s => s.symbol);
  const setSymbol = useWorkspaceStore(s => s.setSymbol);
  const timeframe = useWorkspaceStore(s => s.timeframe);
  const setTimeframe = useWorkspaceStore(s => s.setTimeframe);

  const selectedAccountMeta = useMemo(() => {
    const a = activeAccounts.find(a => a.id === accountId);
    if (!a) return null;
    return { brokerCompany: a.brokerCompany, brokerServer: a.brokerServer, mtType: (a.mtType === 'MT5' ? 'MT5' : 'MT4') as 'MT4' | 'MT5', leverage: a.leverage ?? 0 };
  }, [activeAccounts, accountId]);

  const handleAccountChange = useCallback((id: string) => {
    setAccountId(id); setSymbol(''); marketApi.clearSymbolCache();
  }, [setAccountId, setSymbol]);

  return { activeAccounts, accountId, setAccountId, symbol, setSymbol, timeframe, setTimeframe, handleAccountChange, selectedAccountMeta, fetchAccounts };
}
