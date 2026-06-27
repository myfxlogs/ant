import type { QueryClient } from '@tanstack/react-query';
import { queryKeys } from '@/queries/queryKeys';

/**
 * Canonical user summary type matching what backend SubscribeUserSummary actually sends.
 * Fields 7-17 (pnl_today..max_consecutive_losses) are defined in proto but not yet populated
 * by the backend — reserved for future analytics integration.
 */
export interface UserSummaryData {
  totalBalance: number | string;
  totalEquity: number | string;
  totalProfit: number | string;
  accountCount: number;
  connectedCount: number;
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
    totalBalance: 0,
    totalEquity: 0,
    totalProfit: 0,
    accountCount: 0,
    connectedCount: 0,
  };
}
