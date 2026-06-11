import { analyticsClient, analyticsStreamClient, economicDataClient } from './connect';
import i18n from '@/i18n';
import { EquityCurvePeriod } from '../gen/ant/v1/analytics_pb';
import type { GenerateReportChunk } from '../gen/ant/v1/analytics_pb';
import {
  mapAccountAnalytics,
  mapRecentTradesResponse,
  mapMonthlyPnLResponse,
  mapMonthlyAnalysisResponse,
} from './analyticsMappers';

export type { TradeRecord, MonthlyPnLItem } from '../gen/ant/v1/analytics_pb';

// ──────────────────────────────────────────────
// Frontend-friendly types (after BigInt→Number conversion)
// ──────────────────────────────────────────────

export interface TradeStats {
  totalTrades: number;
  winRate: number;
  profitFactor: number;
  averageProfit: number;
  averageLoss: number;
  largestWin: number;
  largestLoss: number;
  maxConsecutiveWins: number;
  maxConsecutiveLosses: number;
  averageHoldingTime: string;
  netProfit: number;
  totalDeposit: number;
  totalWithdrawal: number;
  netDeposit: number;
}

export interface RiskMetrics {
  maxDrawdownPercent: number;
  sharpeRatio: number;
  sortinoRatio: number;
  calmarRatio: number;
  volatility: number;
  averageDailyReturn: number;
}

export interface SymbolStat {
  symbol: string;
  profit: number;
  tradeSharePercent: number;
}

export interface EquityPoint {
  date: string;
  equity: number;
  balance: number;
  profit: number;
}

export interface DailyPnL {
  day: string;
  date: string;
  pnl: number;
  trades: number;
  lots: number;
  balance: number;
  profitFactor: number;
  maxFloatingLossAmount: number;
  maxFloatingLossRatio: number;
  maxFloatingProfitAmount: number;
  maxFloatingProfitRatio: number;
}

export interface HourlyStat {
  hour: number;
  lots: number;
  balance: number;
  profitFactor: number;
  maxFloatingLossAmount: number;
  maxFloatingLossRatio: number;
  maxFloatingProfitAmount: number;
  maxFloatingProfitRatio: number;
}

export interface AccountAnalyticsData {
  tradeStats: TradeStats;
  riskMetrics: RiskMetrics;
  symbolStats: SymbolStat[];
  equityCurve: EquityPoint[];
  dailyPnl: DailyPnL[];
  hourlyStats: HourlyStat[];
}

export interface TradeRecordItem {
  ticket: number;
  symbol: string;
  type: string;
  volume: number;
  openPrice: number;
  closePrice: number;
  profit: number;
  openTime: string;
  closeTime: string;
  swap: number;
  commission: number;
  comment: string;
}

export interface RecentTradesData {
  trades: TradeRecordItem[];
  total: number;
}

export interface MonthlyPnLData {
  monthlyPnl: Array<{
    month: number;
    profit: number;
    trades: number;
  }>;
}

export interface MonthlyAnalysisData {
  years: number[];
  data: unknown;
}

const analyticsService = analyticsClient;

function toProtoPeriod(p: 'day' | 'week' | 'month' | 'all'): EquityCurvePeriod {
  switch (p) {
    case 'day':   return EquityCurvePeriod.DAY;
    case 'week':  return EquityCurvePeriod.WEEK;
    case 'month': return EquityCurvePeriod.MONTH;
    case 'all':   return EquityCurvePeriod.ALL;
    default:      return EquityCurvePeriod.ALL;
  }
}

export const analyticsApi = {
  getAccountAnalytics: async (accountId: string, period?: 'day' | 'week' | 'month' | 'all'): Promise<AccountAnalyticsData> => {
    const res = await analyticsService.getAccountAnalytics({
      accountId,
      equityCurvePeriod: toProtoPeriod(period || 'all'),
    });
    return mapAccountAnalytics(res);
  },

  getTradeRecords: async (accountId: string, _params?: { from?: string; to?: string }) => {
    const response = await analyticsService.getRecentTrades({
      accountId,
      page: 1,
      pageSize: 100,
    });
    return mapRecentTradesResponse(response).trades;
  },

  getRecentTrades: async (accountId: string, page?: number, pageSize?: number): Promise<RecentTradesData> => {
    const response = await analyticsService.getRecentTrades({
      accountId,
      page: page || 1,
      pageSize: pageSize || 10,
    });
    return mapRecentTradesResponse(response);
  },

  getMonthlyPnL: async (accountId: string, year?: number): Promise<MonthlyPnLData> => {
    const response = await analyticsService.getMonthlyPnL({
      accountId,
      year: year || new Date().getFullYear(),
    });
    return mapMonthlyPnLResponse(response);
  },

  getMonthlyAnalysis: async (accountId: string): Promise<MonthlyAnalysisData> => {
    const response = await analyticsService.getMonthlyAnalysis({
      accountId,
    });
    return mapMonthlyAnalysisResponse(response);
  },

  getEconomicCalendar: async (params?: {
    from?: string;
    to?: string;
    country?: string;
    symbol?: string;
    importance?: string;
  }) => {
    const lang = i18n.language || 'en';
    const data = await economicDataClient.listEconomicCalendarEvents({
      from: params?.from || '',
      to: params?.to || '',
      country: params?.country || '',
      symbol: params?.symbol || '',
      importance: params?.importance || '',
      lang,
    });
    return (data.events || []).map((event) => ({
      ...event,
      timestamp: Number(event.timestamp),
    }));
  },

  getEconomicIndicators: async () => {
    const data = await economicDataClient.listEconomicIndicators({});
    return data.indicators || [];
  },

  getAttributionAnalysis: async (accountId: string): Promise<AttributionAnalysisData> => {
    const res = await analyticsClient.getAttributionAnalysis({ accountId });
    return {
      symbolPnls: (res.symbolPnls || []).map((s) => ({
        symbol: s.symbol,
        netProfit: s.netProfit,
        totalTrades: Number(s.totalTrades),
        winRate: s.winRate,
        profitFactor: s.profitFactor,
        tradeSharePercent: s.tradeSharePercent,
      })),
      direction: {
        longProfit: res.direction?.longProfit || 0,
        longTrades: Number(res.direction?.longTrades || 0),
        longWinRate: res.direction?.longWinRate || 0,
        shortProfit: res.direction?.shortProfit || 0,
        shortTrades: Number(res.direction?.shortTrades || 0),
        shortWinRate: res.direction?.shortWinRate || 0,
      },
      tradeDistribution: {
        profitBuckets: (res.tradeDistribution?.profitBuckets || []).map((b) => ({
          label: b.label,
          minValue: b.minValue,
          maxValue: b.maxValue,
          count: Number(b.count),
        })),
      },
      hourlyPnl: (res.hourlyPnl || []).map((h) => ({
        hour: h.hour,
        profit: h.profit,
        trades: Number(h.trades),
        winRate: h.winRate,
      })),
    };
  },

  getRollingMetrics: async (accountId: string): Promise<RollingMetricsData> => {
    const res = await analyticsClient.getRollingMetrics({ accountId });
    return {
      rollingSharpe: (res.rollingSharpe || []).map((p) => ({
        date: p.date,
        value: p.value,
      })),
      drawdownEvents: (res.drawdownEvents || []).map((e) => ({
        startDate: e.startDate,
        endDate: e.endDate,
        durationDays: e.durationDays,
        depthPercent: e.depthPercent,
        recoveryDate: e.recoveryDate,
      })),
      monthlyWinRates: (res.monthlyWinRates || []).map((m) => ({
        month: m.month,
        winRate: m.winRate,
        totalTrades: Number(m.totalTrades),
      })),
      equityCurve: (res.equityCurve || []).map((p) => ({
        date: p.date,
        equity: p.equity,
        balance: p.balance,
        profit: p.profit,
      })),
      drawdownCurve: (res.drawdownCurve || []).map((p) => ({
        date: p.date,
        drawdownPercent: p.drawdownPercent,
      })),
    };
  },

  generateReportStream: (
    accountId: string,
    period: string,
    locale: string,
    callbacks: ReportCallbacks,
  ): (() => void) => {
    const abortController = new AbortController();
    (async () => {
      try {
        const stream = analyticsStreamClient.generateReport(
          { accountId, period, locale },
          { signal: abortController.signal },
        );
        for await (const chunk of stream) {
          const c = chunk as GenerateReportChunk;
          if (c.phase) callbacks.onPhase(c.phase);
          if (c.delta) callbacks.onDelta(c.delta);
          if (c.section) callbacks.onSection(c.section);
          if (c.error) callbacks.onError(c.error);
          if (c.done) {
            if (c.summary) callbacks.onSummary(c.summary);
            if (c.findings) callbacks.onFindings(c.findings);
            if (c.recommendations) callbacks.onRecommendations(c.recommendations);
            callbacks.onDone();
          }
        }
      } catch (e: unknown) {
        if (e instanceof Error && e.name === 'AbortError') return;
        callbacks.onError(e instanceof Error ? e.message : 'Stream failed');
        callbacks.onDone();
      }
    })();
    return () => abortController.abort();
  },

  getMonthlyDetail: async (accountId: string, year: number, month: number): Promise<MonthlyDetailData> => {
    const res = await analyticsClient.getMonthlyDetail({ accountId, year, month });
    return {
      metrics: {
        netReturn: res.metrics?.netReturn ?? 0,
        returnPercent: res.metrics?.returnPercent ?? 0,
        totalTrades: Number(res.metrics?.totalTrades ?? 0),
        winRate: res.metrics?.winRate ?? 0,
        profitFactor: res.metrics?.profitFactor ?? 0,
        bestTrade: res.metrics?.bestTrade ?? 0,
        worstTrade: res.metrics?.worstTrade ?? 0,
      },
      symbolPnls: (res.symbolPnls ?? []).map((s) => ({
        symbol: s.symbol,
        netProfit: s.netProfit,
        trades: Number(s.trades),
        winRate: s.winRate,
      })),
      holdingStats: {
        averageHours: res.holdingStats?.averageHours ?? 0,
        medianHours: res.holdingStats?.medianHours ?? 0,
        maxHours: res.holdingStats?.maxHours ?? 0,
        minHours: res.holdingStats?.minHours ?? 0,
      },
    };
  },
};

export interface SymbolPnLData {
  symbol: string;
  netProfit: number;
  totalTrades: number;
  winRate: number;
  profitFactor: number;
  tradeSharePercent: number;
}

export interface DirectionBreakdown {
  longProfit: number;
  longTrades: number;
  longWinRate: number;
  shortProfit: number;
  shortTrades: number;
  shortWinRate: number;
}

export interface TradeBucket {
  label: string;
  minValue: number;
  maxValue: number;
  count: number;
}

export interface AttributionAnalysisData {
  symbolPnls: SymbolPnLData[];
  direction: DirectionBreakdown;
  tradeDistribution: {
    profitBuckets: TradeBucket[];
  };
  hourlyPnl: Array<{ hour: number; profit: number; trades: number; winRate: number }>;
}

export interface RollingMetricsData {
  rollingSharpe: Array<{ date: string; value: number }>;
  drawdownEvents: Array<{
    startDate: string;
    endDate: string;
    durationDays: number;
    depthPercent: number;
    recoveryDate: string;
  }>;
  monthlyWinRates: Array<{ month: string; winRate: number; totalTrades: number }>;
  equityCurve: EquityPoint[];
  drawdownCurve: Array<{ date: string; drawdownPercent: number }>;
}

export interface MonthlyDetailData {
  metrics: {
    netReturn: number;
    returnPercent: number;
    totalTrades: number;
    winRate: number;
    profitFactor: number;
    bestTrade: number;
    worstTrade: number;
  };
  symbolPnls: Array<{
    symbol: string;
    netProfit: number;
    trades: number;
    winRate: number;
  }>;
  holdingStats: {
    averageHours: number;
    medianHours: number;
    maxHours: number;
    minHours: number;
  };
}

export interface ReportCallbacks {
  onPhase: (phase: string) => void;
  onDelta: (delta: string) => void;
  onSection: (section: string) => void;
  onSummary: (text: string) => void;
  onFindings: (text: string) => void;
  onRecommendations: (text: string) => void;
  onError: (error: string) => void;
  onDone: () => void;
}
