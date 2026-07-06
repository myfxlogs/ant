import { useCallback, useState } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { analyticsApi } from '@/client/analytics';
import { queryKeys } from '@/queries/queryKeys';
import type { TradeRecordItem, RecentTradesData } from '@/client/analytics';

/** Recent trades for AccountTradeTabs — backed by TanStack Query so SSE can push directly. */
export function useHistoryTrades(id: string | undefined) {
  const queryClient = useQueryClient();

  const recentTradesQ = useQuery<RecentTradesData>({
    queryKey: queryKeys.analytics.recentTrades(id!),
    queryFn: () => analyticsApi.getRecentTrades(id!, 1, 10),
    enabled: !!id,
    staleTime: 0,
    refetchOnMount: true,
    retry: 2,
  });

  const historyTrades: TradeRecordItem[] = recentTradesQ.data?.trades ?? [];
  const historyTotal = recentTradesQ.data?.total ?? 0;
  const historyLoading = recentTradesQ.isLoading;
  const [historyPage, setHistoryPage] = useState(1);

  const setHistoryTrades = useCallback((trades: TradeRecordItem[]) => {
    queryClient.setQueryData<RecentTradesData>(
      queryKeys.analytics.recentTrades(id!),
      (old) => (old ? { ...old, trades } : { trades, total: trades.length }),
    );
  }, [id, queryClient]);

  const setHistoryTotal = useCallback((total: number) => {
    queryClient.setQueryData<RecentTradesData>(
      queryKeys.analytics.recentTrades(id!),
      (old) => (old ? { ...old, total } : { trades: [], total }),
    );
  }, [id, queryClient]);

  return {
    historyTrades, historyTotal, historyPage, historyLoading,
    setHistoryTrades, setHistoryTotal, setHistoryPage,
    handleRefresh: () => { recentTradesQ.refetch(); },
    handleRetry: () => { recentTradesQ.refetch(); },
  };
}
