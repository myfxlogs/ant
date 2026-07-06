import { useCallback, useMemo, useState } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { showError } from '@/utils/message';
import { getErrorMessage } from '@/utils/error';
import { analyticsApi } from '@/client/analytics';
import { queryKeys } from '@/queries/queryKeys';
import type { TradeRecordItem, AccountAnalyticsData, RecentTradesData } from '@/client/analytics';

interface AccountMonthlyPnLItem {
  month?: string; monthNum?: number; month_num?: number;
  profit: number; trades: number;
}

/**
 * Analytics data loading. History trades are backed by TanStack Query so the
 * SSE bridge can push close events directly into the cache via setQueryData —
 * no CustomEvent side-channel needed.
 */
export function useAccountAnalytics(
  id: string | undefined,
  _isDataReceived: boolean,
  chartPeriod: 'day' | 'week' | 'month' | 'all',
) {
  const queryClient = useQueryClient();

  // Main analytics data (equity curve, stats, distributions).
  // staleTime=0 — analytics depend on trade_records which may be empty
  // until SyncHistory completes; always fetch fresh on mount.
  const analyticsQ = useQuery<AccountAnalyticsData>({
    queryKey: queryKeys.analytics.detail(id!, chartPeriod),
    queryFn: () => analyticsApi.getAccountAnalytics(id!, chartPeriod),
    enabled: !!id,
    staleTime: 0,
    refetchOnMount: true,
    retry: 2,
  });

  // Recent trades (history) — initial load + SSE bridge appends via setQueryData.
  // refetchOnMount ensures a fresh fetch on every navigation, not just when
  // the query key changes (TanStack Query may serve stale cache otherwise).
  const recentTradesQ = useQuery<RecentTradesData>({
    queryKey: queryKeys.analytics.recentTrades(id!),
    queryFn: () => analyticsApi.getRecentTrades(id!, 1, 10),
    enabled: !!id,
    staleTime: 0,           // always fetch fresh on mount
    refetchOnMount: true,
    retry: 2,
  });

  // Monthly PnL (static, long cache)
  const monthlyPnLQ = useQuery({
    queryKey: queryKeys.analytics.monthlyPnL(id!, new Date().getFullYear()),
    queryFn: () => analyticsApi.getMonthlyPnL(id!, new Date().getFullYear()),
    enabled: !!id,
    staleTime: 5 * 60_000,
  });

  // Monthly analysis (static, long cache)
  const monthlyAnalysisQ = useQuery({
    queryKey: queryKeys.analytics.monthlyAnalysis(id!),
    queryFn: () => analyticsApi.getMonthlyAnalysis(id!),
    enabled: !!id,
    staleTime: 5 * 60_000,
  });

  const analytics = analyticsQ.data ?? null;
  const analyticsLoading = analyticsQ.isLoading;
  const analyticsError = analyticsQ.error ? getErrorMessage(analyticsQ.error, '') : null;

  const historyTrades: TradeRecordItem[] = recentTradesQ.data?.trades ?? [];
  const historyTotal = recentTradesQ.data?.total ?? 0;
  const historyLoading = recentTradesQ.isLoading;

  // Setters update the TanStack Query cache directly so SSE bridge writes
  // (setQueryData on close) and pagination writes (via AccountTradeTabs)
  // share the same single source of truth.
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

  const [historyPage, setHistoryPage] = useState(1);

  const monthlyPnL: AccountMonthlyPnLItem[] = useMemo(() =>
    (monthlyPnLQ.data?.monthlyPnl ?? []).map((item) => ({
      month: String(item.month), monthNum: item.month, month_num: item.month,
      profit: item.profit, trades: item.trades,
    })), [monthlyPnLQ.data?.monthlyPnl]);

  const monthlyAnalysisYears: number[] = useMemo(() =>
    monthlyAnalysisQ.data?.years ?? [], [monthlyAnalysisQ.data?.years]);

  const monthlyAnalysisData: unknown[] = useMemo(() => {
    const raw = monthlyAnalysisQ.data?.data;
    return Array.isArray(raw) ? raw : [];
  }, [monthlyAnalysisQ.data?.data]);

  // ── Derived chart data ──
  const derived = useMemo(() => {
    const DAY_NAMES = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];
    const curve = analytics?.equityCurve?.map((point) => {
      const raw = point.date || '';
      let label: string;
      if (raw.indexOf(' ') > 0) {
        label = raw.slice(raw.indexOf(' ') + 1, raw.indexOf(' ') + 6);
      } else if (chartPeriod === 'week') {
        try { label = DAY_NAMES[new Date(raw + 'T00:00:00').getDay()]; } catch { label = raw; }
      } else {
        const parts = raw.split('-');
        label = parts.length >= 3 ? `${parseInt(parts[1], 10)}/${parseInt(parts[2], 10)}` : raw;
      }
      return { date: label, equity: point.equity, balance: point.balance, profit: point.profit };
    }) || [];
    const profitByMonth = monthlyPnL.map((m) => ({
      month: String(m?.month ?? m?.monthNum ?? m?.month_num ?? ''),
      profit: m.profit, trades: Number(m.trades),
    })).filter((m) => m.month);
    const dailyPnl = (analytics?.dailyPnl || []).map((d) => ({
      day: d.day, date: d.date, profit: d.pnl, trades: d.trades, lots: d.lots,
      balance: d.balance, profitFactor: d.profitFactor,
      maxFloatingLossAmount: d.maxFloatingLossAmount, maxFloatingLossRatio: d.maxFloatingLossRatio,
      maxFloatingProfitAmount: d.maxFloatingProfitAmount, maxFloatingProfitRatio: d.maxFloatingProfitRatio,
    }));
    const symbolDist = (analytics?.symbolStats || []).slice(0, 6).map((s) => ({
      name: s.symbol, value: Math.round(s.tradeSharePercent), profit: s.profit,
    }));
    return {
      equityChartData: curve,
      profitByMonthData: profitByMonth,
      symbolDistributionData: symbolDist,
      dailyPnLData: dailyPnl,
      hourlyData: (analytics?.hourlyStats || []).map((h) => ({
        ...h, hourLabel: `${String(h.hour).padStart(2, '0')}:00`,
      })),
      tradeStats: analytics?.tradeStats || { totalTrades: 0, winRate: 0, profitFactor: 0 },
      riskMetrics: analytics?.riskMetrics || { maxDrawdownPercent: 0, sharpeRatio: 0 },
      monthlyAnalysisYears,
      monthlyAnalysisData,
    };
  }, [analytics, monthlyPnL, chartPeriod, monthlyAnalysisYears, monthlyAnalysisData]);

  return {
    analyticsLoading, analyticsError,
    historyTrades, historyTotal, historyPage, historyLoading,
    setHistoryTrades, setHistoryTotal, setHistoryPage,
    handleRefresh: () => {
      analyticsQ.refetch();
      recentTradesQ.refetch();
      monthlyPnLQ.refetch();
      monthlyAnalysisQ.refetch();
    },
    handleRetry: () => {
      analyticsQ.refetch();
      recentTradesQ.refetch();
    },
    ...derived,
  };
}
