import { useQuery } from '@tanstack/react-query';
import { queryKeys } from './queryKeys';
import { tradingApi } from '@/client/trading';
import type { Position } from '@/types/trading';

/**
 * SSE-backed positions query. SSE updates the cache directly;
 * the RPC fires once on mount as initial fill.
 */
export function usePositionsQuery(accountId: string) {
  return useQuery<Position[]>({
    queryKey: queryKeys.positions.byAccount(accountId),
    queryFn: async () => {
      const result = await tradingApi.getPositions(accountId);
      return result as unknown as Position[];
    },
    enabled: !!accountId,
    staleTime: 120_000, // SSE keeps position state fresh
    retry: 2,
    refetchOnWindowFocus: false,
    refetchInterval: (query) => {
      // Only poll if SSE hasn't updated the cache recently.
      // SSE updates come via bridgeStreamEvents → setQueryData every few seconds.
      const last = query.state.dataUpdatedAt;
      return Date.now() - last > 90_000 ? 60_000 : false;
    },
  });
}
