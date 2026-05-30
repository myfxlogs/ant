import { useQuery } from '@tanstack/react-query';
import { queryKeys } from './queryKeys';

export interface UserSummary {
  totalBalance: number;
  totalEquity: number;
  totalProfit: number;
  accountCount: number;
  connectedCount: number;
  pnlToday: number;
  pnlWeek: number;
  pnlMonth: number;
  tradesToday: number;
  tradesWeek: number;
  tradesMonth: number;
  winRate: number;
  profitFactor: number;
  maxDrawdownPercent: number;
  maxConsecutiveWins: number;
  maxConsecutiveLosses: number;
}

const DEFAULT_SUMMARY: UserSummary = {
  totalBalance: 0, totalEquity: 0, totalProfit: 0,
  accountCount: 0, connectedCount: 0,
  pnlToday: 0, pnlWeek: 0, pnlMonth: 0,
  tradesToday: 0, tradesWeek: 0, tradesMonth: 0,
  winRate: 0, profitFactor: 0,
  maxDrawdownPercent: 0, maxConsecutiveWins: 0, maxConsecutiveLosses: 0,
};

/**
 * UserSummary is purely SSE-driven (no RPC).
 * The bridge writes data directly to this cache key.
 */
export function useUserSummaryQuery() {
  return useQuery<UserSummary>({
    queryKey: queryKeys.userSummary.all,
    queryFn: () => DEFAULT_SUMMARY,
    staleTime: Infinity, // SSE-only; never refetch
    retry: false,
    refetchOnWindowFocus: false,
    refetchOnMount: false,
  });
}
