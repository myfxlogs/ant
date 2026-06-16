// analyticsApi.ts — API methods for analytics.
import { analyticsClient, analyticsStreamClient, economicDataClient } from './connect';
import i18n from '@/i18n';
import { EquityCurvePeriod } from '../gen/ant/v1/analytics_pb';
import type { GenerateReportChunk } from '../gen/ant/v1/analytics_pb';
import {
  mapAccountAnalytics,
  mapRecentTradesResponse,
  mapMonthlyPnLResponse,
  mapMonthlyAnalysisResponse,
  mapAttributionAnalysis,
  mapRollingMetrics,
  mapMonthlyDetail,
} from './analyticsMappers';
import type {
  AccountAnalyticsData,
  RecentTradesData,
  MonthlyPnLData,
  MonthlyAnalysisData,
  AttributionAnalysisData,
  RollingMetricsData,
  MonthlyDetailData,
  ReportCallbacks,
} from './analyticsTypes';

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
    return mapAttributionAnalysis(res);
  },

  getRollingMetrics: async (accountId: string): Promise<RollingMetricsData> => {
    const res = await analyticsClient.getRollingMetrics({ accountId });
    return mapRollingMetrics(res);
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
    return mapMonthlyDetail(res);
  },
};
