import { analyticsClient, economicDataClient } from './connect';
import i18n from '@/i18n';
import { EquityCurvePeriod } from '../gen/ant/v1/analytics_pb';
import { deepConvertBigIntToNumber } from '@/adapters/dataAdapter';

export type { TradeStats, RiskMetrics, SymbolStat, TradeRecord, MonthlyPnLItem } from '../gen/ant/v1/analytics_pb';

const analyticsService = analyticsClient;

// Analytics backed by AnalyticsService handler (backend/internal/connect/system/analytics_handler.go).
// Reads from trade_records PostgreSQL table via AnalyticsRepository.
interface RecentTradesResponse {
  trades?: unknown[];
  total?: number;
}

interface MonthlyPnLResponse {
  monthlyPnl?: unknown[];
}

interface MonthlyAnalysisResponse {
  years?: number[];
  data?: unknown[];
}

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
  getAccountAnalytics: async (accountId: string, period?: 'day' | 'week' | 'month' | 'all') => {
    const res = await analyticsService.getAccountAnalytics({
      accountId,
      equityCurvePeriod: toProtoPeriod(period || 'all'),
    });
    return deepConvertBigIntToNumber(res);
  },

  getTradeRecords: async (accountId: string, _params?: { from?: string; to?: string }) => {
    const response = await analyticsService.getRecentTrades({
      accountId,
      page: 1,
      pageSize: 100,
    }) as unknown as RecentTradesResponse;
    return response.trades;
  },

  getRecentTrades: async (accountId: string, page?: number, pageSize?: number) => {
    const response = await analyticsService.getRecentTrades({
      accountId,
      page: page || 1,
      pageSize: pageSize || 10,
    }) as unknown as RecentTradesResponse;
    return {
      trades: response.trades,
      total: response.total,
    };
  },

  getMonthlyPnL: async (accountId: string, year?: number) => {
    const response = await analyticsService.getMonthlyPnL({
      accountId,
      year: year || new Date().getFullYear(),
    }) as unknown as MonthlyPnLResponse;
    return {
      monthlyPnl: response.monthlyPnl,
    };
  },

  getMonthlyAnalysis: async (accountId: string) => {
    const response = await analyticsService.getMonthlyAnalysis({
      accountId,
    }) as unknown as MonthlyAnalysisResponse;
    return {
      years: response.years || [],
      data: response.data || [],
    };
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
