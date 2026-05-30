import { useQuery } from '@tanstack/react-query';
import { queryKeys } from './queryKeys';
import { accountApi } from '@/client/account';

/**
 * Live financial data backed by SSE (bridgeStreamEvents writes to this cache).
 * The RPC acts as the initial fill + fallback if SSE hasn't arrived yet.
 */
export function useAccountFinancials(accountId: string) {
  return useQuery({
    queryKey: queryKeys.accounts.financials(accountId),
    queryFn: async () => {
      const a = await accountApi.get(accountId);
      return {
        balance: a.balance,
        equity: a.equity,
        profit: a.profit,
        margin: a.margin,
        freeMargin: a.freeMargin,
        marginLevel: a.marginLevel,
        credit: a.credit,
      };
    },
    enabled: !!accountId,
    staleTime: 60_000, // SSE keeps this fresh; RPC is the fallback
    retry: 2,
    refetchOnWindowFocus: false,
  });
}
