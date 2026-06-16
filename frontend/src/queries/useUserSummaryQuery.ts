import { useQuery } from '@tanstack/react-query';
import type { UserSummaryData } from '@/bridge/bridgeUserSummary';
import { queryKeys } from './queryKeys';

/**
 * UserSummary is purely SSE-driven (no RPC).
 * StreamProvider writes data directly to this cache key via handleUserSummary().
 */
export function useUserSummaryQuery() {
  return useQuery<UserSummaryData>({
    queryKey: queryKeys.userSummary.all,
    staleTime: Infinity, // SSE keeps this fresh; never refetch
    retry: false,
    refetchOnWindowFocus: false,
    refetchOnMount: false,
  });
}
