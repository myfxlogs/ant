import type { QueryClient } from '@tanstack/react-query';
import { queryKeys } from '@/queries/queryKeys';

export interface UserSummaryData {
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
  updatedAt?: unknown;
}

export function handleUserSummary(
  queryClient: QueryClient,
  summary: Partial<UserSummaryData>,
) {
  queryClient.setQueryData<UserSummaryData>(
    queryKeys.userSummary.all,
    (old) => ({ ...(old ?? getDefaultSummary()), ...summary }),
  );
}

function getDefaultSummary(): UserSummaryData {
  return {
    totalBalance: 0, totalEquity: 0, totalProfit: 0,
    accountCount: 0, connectedCount: 0,
    pnlToday: 0, pnlWeek: 0, pnlMonth: 0,
    tradesToday: 0, tradesWeek: 0, tradesMonth: 0,
    winRate: 0, profitFactor: 0,
    maxDrawdownPercent: 0, maxConsecutiveWins: 0, maxConsecutiveLosses: 0,
  };
}
