import { analyticsClient, economicDataClient } from './connect';
import i18n from '@/i18n';

export type { AccountAnalytics, Summary, RiskMetrics, SymbolStats, TradeRecord, MonthlyPnL } from '../gen/api_pb';

const analyticsService = analyticsClient;

export const analyticsApi = {
  getAccountAnalytics: async (accountId: string) => {
    return await analyticsService.getAccountAnalytics({ accountId });
  },

  getTradeRecords: async (accountId: string, _params?: { from?: string; to?: string }) => {
    const response: any = await analyticsService.getRecentTrades({
      accountId,
      page: 1,
      pageSize: 100,
    });
    return response.trades;
  },

  getRecentTrades: async (accountId: string, page?: number, pageSize?: number) => {
    const response: any = await analyticsService.getRecentTrades({
      accountId,
      page: page || 1,
      pageSize: pageSize || 10,
    });
    return {
      trades: response.trades,
      total: response.total,
    };
  },

  getMonthlyPnL: async (accountId: string, year?: number) => {
    const response: any = await analyticsService.getMonthlyPnL({ 
      accountId, 
      year: year || new Date().getFullYear() 
    });
    return { 
      monthlyPnl: response.monthlyPnl
    };
  },

  getMonthlyAnalysis: async (accountId: string) => {
    const response: any = await (analyticsService as any).getMonthlyAnalysis({
      accountId,
    });
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
    });
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
    const response: any = await analyticsService.getSymbolStats({ accountId });
    return response.stats;
  },

  getEconomicCalendar: async (params?: { from?: string; to?: string; country?: string; symbol?: string; importance?: string }) => {
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
