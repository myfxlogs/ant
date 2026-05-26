import { analyticsClient, economicDataClient } from './connect';
import i18n from '@/i18n';

export type { AccountAnalytics, Summary, RiskMetrics, SymbolStats, TradeRecord, MonthlyPnL } from '../gen/ant/v1/api_pb';

const analyticsService = analyticsClient;

// Analytics service currently uses a stub client (backend handlers not yet
// fully implemented). The interfaces below document the expected response
// shapes once the backend is available.
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

interface SymbolStatsResponse {
  stats?: unknown[];
}

interface MonthlyAnalysisBonusResponse {
  riskRatio?: number;
  symbolPopularity?: Array<{ symbol: string; trades: number; sharePercent: number }>;
  symbolRiskRatios?: Array<{ symbol: string; riskRatio: number }>;
  symbolHoldingSplit?: Array<{ symbol: string; bullsSeconds: number; shortTermSeconds: number }>;
  averageHoldingSeconds?: number;
  totalTrades?: number;
}

export const analyticsApi = {
  getAccountAnalytics: async (accountId: string) => {
    return await analyticsService.getAccountAnalytics({ accountId });
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

  getMonthlyAnalysisBonus: async (accountId: string, year: number, month: number) => {
    const response = await analyticsService.getMonthlyAnalysisBonus({
      accountId,
      year,
      month,
    }) as unknown as MonthlyAnalysisBonusResponse;
    const rows = response.symbolPopularity ?? [];
    const risks = response.symbolRiskRatios ?? [];
    const holds = response.symbolHoldingSplit ?? [];
    return {
      riskRatio: response.riskRatio ?? 0,
      symbolPopularity: rows.map((r) => ({
        symbol: r.symbol,
        trades: r.trades,
        sharePercent: r.sharePercent,
      })),
      symbolRisks: risks.map((r) => ({
        symbol: r.symbol,
        riskRatio: r.riskRatio,
      })),
      symbolHoldingSplit: holds.map((r) => ({
        symbol: r.symbol,
        bullsSeconds: r.bullsSeconds,
        shortTermSeconds: r.shortTermSeconds,
      })),
      averageHoldingSeconds: response.averageHoldingSeconds ?? 0,
      totalTrades: response.totalTrades ?? 0,
    };
  },

  getSummary: async (accountId: string) => {
    return await analyticsService.getSummary({ accountId });
  },

  getRiskMetrics: async (accountId: string) => {
    return await analyticsService.getRiskMetrics({ accountId });
  },

  getSymbolStats: async (accountId: string) => {
    const response = await analyticsService.getSymbolStats({ accountId }) as unknown as SymbolStatsResponse;
    return response.stats;
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
