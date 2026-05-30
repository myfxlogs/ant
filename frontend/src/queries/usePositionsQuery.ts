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
      return result as Position[];
    },
    enabled: !!accountId,
    staleTime: 120_000, // SSE keeps position state fresh
    retry: 2,
    refetchOnWindowFocus: false,
  });
}
