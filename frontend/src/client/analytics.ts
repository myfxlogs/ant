import { analyticsClient, economicDataClient } from './connect';
import i18n from '@/i18n';
import { EquityCurvePeriod } from '../gen/ant/v1/analytics_pb';
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
};
