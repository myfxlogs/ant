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
        balance: a.balance ?? 0,
        equity: a.equity ?? 0,
        profit: a.profit ?? 0,
        profitPercent: a.profitPercent ?? 0,
        margin: a.margin ?? 0,
        freeMargin: a.freeMargin ?? 0,
        marginLevel: a.marginLevel ?? 0,
        credit: a.credit ?? 0,
      };
    },
    enabled: !!accountId,
    staleTime: 60_000, // SSE keeps this fresh; RPC is the fallback
    retry: 2,
    refetchOnWindowFocus: false,
  });
}
