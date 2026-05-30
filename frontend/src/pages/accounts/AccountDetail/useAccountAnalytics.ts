import { useEffect, useState, useCallback, useMemo, useRef } from 'react';
import { showError } from '@/utils/message';
import { getErrorMessage } from '@/utils/error';
import { analyticsApi } from '@/client/analytics';
import type { TradeRecordItem, AccountAnalyticsData } from '@/client/analytics';

interface AccountMonthlyPnLItem {
  month?: string; monthNum?: number; month_num?: number;
  profit: number; trades: number;
}

interface PositionChangeOrder {
  ticket: number; symbol: string; type: string; volume: number;
  openPrice: number; closePrice: number; profit: number;
  openTime: number; closeTime: number; swap: number; commission: number; comment: string;
}
interface PositionChangeDetail { action: string; order: PositionChangeOrder; }

/**
 * Analytics data loading + SSE position-change side-channel handler.
 * Keeps analytics state, history pagination, and chart data transforms in one place.
 */
export function useAccountAnalytics(
  id: string | undefined,
  isDataReceived: boolean,
  chartPeriod: 'day' | 'week' | 'month' | 'all',
) {
  const [analyticsLoading, setAnalyticsLoading] = useState(false);
  const [analyticsError, setAnalyticsError] = useState<string | null>(null);
  const [analytics, setAnalytics] = useState<AccountAnalyticsData | null>(null);
  const [monthlyPnL, setMonthlyPnL] = useState<AccountMonthlyPnLItem[]>([]);
  const [monthlyAnalysisYears, setMonthlyAnalysisYears] = useState<number[]>([]);
  const [monthlyAnalysisData, setMonthlyAnalysisData] = useState<unknown[]>([]);
  const [historyTrades, setHistoryTrades] = useState<TradeRecordItem[]>([]);
  const [historyTotal, setHistoryTotal] = useState(0);
  const [historyPage, setHistoryPage] = useState(1);
  const [historyLoading, setHistoryLoading] = useState(false);

  const lastReloadRef = useRef(0);
  const RELOAD_MIN_MS = 5000;
  const idRef = useRef(id);
  idRef.current = id;

  const loadAllData = useCallback(async (accountId: string) => {
    setAnalyticsLoading(true);
    setAnalyticsError(null);
    try {
      const [analyticsData, tradesData, monthlyData, monthlyAnalysisResp] = await Promise.all([
        analyticsApi.getAccountAnalytics(accountId, chartPeriod),
        analyticsApi.getRecentTrades(accountId, 1, 10),
        analyticsApi.getMonthlyPnL(accountId, new Date().getFullYear()),
        analyticsApi.getMonthlyAnalysis(accountId),
      ]);
      setAnalytics(analyticsData);
      setHistoryTrades(tradesData.trades);
      setHistoryTotal(tradesData.total);
      setHistoryPage(1);
      setMonthlyAnalysisYears(monthlyAnalysisResp.years);
      setMonthlyAnalysisData(Array.isArray(monthlyAnalysisResp.data) ? monthlyAnalysisResp.data : []);
      setMonthlyPnL(monthlyData.monthlyPnl.map((item) => ({
        month: String(item.month), monthNum: item.month, month_num: item.month,
        profit: item.profit, trades: item.trades,
      })));
    } catch (error) {
      setAnalyticsError(getErrorMessage(error, '加载分析数据失败'));
    } finally {
      setAnalyticsLoading(false);
    }
  }, [chartPeriod]);

  // chartPeriod-only refresh
  const loadForPeriod = useCallback(async (accountId: string) => {
    try {
      setAnalytics(await analyticsApi.getAccountAnalytics(accountId, chartPeriod));
    } catch { /* keep existing */ }
  }, [chartPeriod]);

  // Initial load when SSE data arrives
  useEffect(() => {
    if (!id || !isDataReceived) return;
    loadAllData(id).catch((err) => showError(getErrorMessage(err, '加载分析数据失败')));
    const timer = setTimeout(() => { loadAllData(id).catch(() => {}); }, 5000);
    return () => clearTimeout(timer);
  }, [id, isDataReceived]); // eslint-disable-line react-hooks/exhaustive-deps

  // chartPeriod change
  useEffect(() => {
    if (!id) return;
    loadForPeriod(id);
  }, [id, chartPeriod, loadForPeriod]);

  // Position-change side-channel (SSE → throttle analytics reload)
  useEffect(() => {
    if (!id) return;
    const throttled = () => {
      const now = Date.now();
      if (now - lastReloadRef.current < RELOAD_MIN_MS) return;
      lastReloadRef.current = now;
      loadAllData(idRef.current!).catch(() => {});
    };
    const handler = (event: Event) => {
      const detail = (event as CustomEvent<PositionChangeDetail>).detail;
      const { action, order } = detail;
      if (action === 'PositionClose' && order) {
        const newTrade: TradeRecordItem = {
          ticket: order.ticket, symbol: order.symbol, type: order.type,
          volume: order.volume, openPrice: order.openPrice, closePrice: order.closePrice,
          profit: order.profit, openTime: String(order.openTime ?? ''),
          closeTime: String(order.closeTime ?? ''), swap: order.swap || 0,
          commission: order.commission || 0, comment: order.comment || '',
        };
        setHistoryTrades((prev) => {
          const idx = prev.findIndex((t) => t.ticket === order.ticket);
          if (idx >= 0) { const u = [...prev]; u[idx] = { ...prev[idx], ...newTrade }; return u; }
          return [newTrade, ...prev];
        });
        throttled();
      } else if (action === 'PositionOpen' || action === 'PendingOpen') {
        throttled();
      }
    };
    window.addEventListener('position-change', handler);
    return () => window.removeEventListener('position-change', handler);
  }, [id]); // eslint-disable-line react-hooks/exhaustive-deps

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
    handleRefresh: () => id && loadAllData(id),
    handleRetry: () => id && loadAllData(id),
    ...derived,
  };
}
