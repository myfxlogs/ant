/**
 * Proto→Frontend mappers for analytics response types.
 * Base type mappers (mapTradeStats, mapRiskMetrics, etc.) live in analyticsMappersBase.ts.
 *
 * Proto monetary fields are now string (decimal strings like "123.45")
 * to preserve precision across the wire — no float64 rounding at proto boundary.
 */
import type {
  AccountAnalyticsResponse,
  GetRecentTradesResponse,
  GetMonthlyPnLResponse,
  GetMonthlyAnalysisResponse,
  GetAttributionAnalysisResponse,
  GetRollingMetricsResponse,
  GetMonthlyDetailResponse,
} from '../gen/ant/v1/analytics_pb';
import { fromBinary } from '@bufbuild/protobuf';
import { MonthlyAnalysisPointsSchema } from '../gen/ant/v1/market_events_pb';
import { deepConvertBigIntToNumber } from '@/adapters/dataAdapter';
import type {
  AccountAnalyticsData,
  RecentTradesData,
  MonthlyPnLData,
  MonthlyAnalysisData,
  AttributionAnalysisData,
  RollingMetricsData,
  MonthlyDetailData,
} from './analytics';
import {
  toNum, mapTradeStats, mapRiskMetrics, mapSymbolStat,
  mapEquityPoint, mapDailyPnL, mapHourlyStat, mapTradeRecord,
} from './analyticsMappersBase';

export function mapAccountAnalytics(r: AccountAnalyticsResponse): AccountAnalyticsData {
  const c = deepConvertBigIntToNumber(r);
  return {
    tradeStats: c.tradeStats ? mapTradeStats(c.tradeStats) : {
      totalTrades: 0, winRate: 0, profitFactor: 0, averageProfit: 0, averageLoss: 0,
      largestWin: 0, largestLoss: 0, maxConsecutiveWins: 0, maxConsecutiveLosses: 0,
      averageHoldingTime: '-', netProfit: 0, totalDeposit: 0, totalWithdrawal: 0, netDeposit: 0,
    },
    riskMetrics: c.riskMetrics ? mapRiskMetrics(c.riskMetrics) : {
      maxDrawdownPercent: 0, sharpeRatio: 0, sortinoRatio: 0, calmarRatio: 0,
      volatility: 0, averageDailyReturn: 0,
    },
    symbolStats: (c.symbolStats || []).map(mapSymbolStat),
    equityCurve: (c.equityCurve || []).map(mapEquityPoint),
    dailyPnl: (c.dailyPnl || []).map(mapDailyPnL),
    hourlyStats: (c.hourlyStats || []).map(mapHourlyStat),
  };
}

export function mapRecentTradesResponse(r: GetRecentTradesResponse): RecentTradesData {
  const c = deepConvertBigIntToNumber(r);
  return {
    trades: (c.trades || []).map(mapTradeRecord),
    total: Number(r.total),
  };
}

export function mapMonthlyPnLResponse(r: GetMonthlyPnLResponse): MonthlyPnLData {
  const c = deepConvertBigIntToNumber(r);
  return {
    monthlyPnl: (c.monthlyPnl || []).map((item) => ({
      month: item.month,
      profit: toNum(item.profit),
      trades: Number(item.trades),
    })),
  };
}

export function mapMonthlyAnalysisResponse(r: GetMonthlyAnalysisResponse): MonthlyAnalysisData {
  let points: unknown[] = [];
  if (r.data && r.data.length > 0) {
    try {
      const decoded = fromBinary(MonthlyAnalysisPointsSchema, r.data);
      points = (decoded.points || []).map(p => ({ ...p, change: Number(p.change), profit: Number(p.profit), lots: Number(p.lots), pips: Number(p.pips) }));
    } catch { /* keep empty on decode error */ }
  }
  return { years: r.years || [], data: points };
}

export function mapAttributionAnalysis(r: GetAttributionAnalysisResponse): AttributionAnalysisData {
  const c = deepConvertBigIntToNumber(r);
  return {
    symbolPnls: (c.symbolPnls || []).map((s) => ({
      symbol: s.symbol,
      netProfit: toNum(s.netProfit),
      totalTrades: Number(s.totalTrades),
      winRate: s.winRate,
      profitFactor: s.profitFactor,
      tradeSharePercent: s.tradeSharePercent,
    })),
    direction: {
      longProfit: toNum(c.direction?.longProfit),
      longTrades: Number(c.direction?.longTrades || 0),
      longWinRate: c.direction?.longWinRate || 0,
      shortProfit: toNum(c.direction?.shortProfit),
      shortTrades: Number(c.direction?.shortTrades || 0),
      shortWinRate: c.direction?.shortWinRate || 0,
    },
    tradeDistribution: {
      profitBuckets: (c.tradeDistribution?.profitBuckets || []).map((b) => ({
        label: b.label,
        minValue: toNum(b.minValue),
        maxValue: toNum(b.maxValue),
        count: Number(b.count),
      })),
    },
    hourlyPnl: (c.hourlyPnl || []).map((h) => ({
      hour: h.hour,
      profit: toNum(h.profit),
      trades: Number(h.trades),
      winRate: h.winRate,
    })),
  };
}

export function mapRollingMetrics(r: GetRollingMetricsResponse): RollingMetricsData {
  const c = deepConvertBigIntToNumber(r);
  return {
    rollingSharpe: (c.rollingSharpe || []).map((p) => ({
      date: p.date,
      value: p.value,
    })),
    drawdownEvents: (c.drawdownEvents || []).map((e) => ({
      startDate: e.startDate,
      endDate: e.endDate,
      durationDays: e.durationDays,
      depthPercent: e.depthPercent,
      recoveryDate: e.recoveryDate,
    })),
    monthlyWinRates: (c.monthlyWinRates || []).map((m) => ({
      month: m.month,
      winRate: m.winRate,
      totalTrades: Number(m.totalTrades),
    })),
    equityCurve: (c.equityCurve || []).map(mapEquityPoint),
    drawdownCurve: (c.drawdownCurve || []).map((p) => ({
      date: p.date,
      drawdownPercent: p.drawdownPercent,
    })),
  };
}

function mapMonthlyMetrics(c: { metrics?: { netReturn?: string | null; returnPercent?: number; totalTrades?: bigint | number; winRate?: number; profitFactor?: number; bestTrade?: string | null; worstTrade?: string | null } }): MonthlyDetailData['metrics'] {
  const m = c.metrics;
  return {
    netReturn: toNum(m?.netReturn),
    returnPercent: m?.returnPercent ?? 0,
    totalTrades: Number(m?.totalTrades ?? 0),
    winRate: m?.winRate ?? 0,
    profitFactor: m?.profitFactor ?? 0,
    bestTrade: toNum(m?.bestTrade),
    worstTrade: toNum(m?.worstTrade),
  };
}

function mapMonthlyBonus(c: { bonus?: { riskRatio?: number; symbolPopularity?: { symbol?: string; trades?: bigint | number; sharePercent?: number }[]; symbolRisks?: { symbol?: string; riskRatio?: number }[]; symbolHoldingSplit?: { symbol?: string; bullsSeconds?: number; shortTermSeconds?: number }[] } }): MonthlyDetailData['bonus'] {
  if (!c.bonus) return undefined;
  return {
    riskRatio: c.bonus.riskRatio ?? 0,
    symbolPopularity: (c.bonus.symbolPopularity ?? []).map((s: { symbol?: string; trades?: bigint | number; sharePercent?: number }) => ({
      symbol: s.symbol || '', trades: Number(s.trades), sharePercent: s.sharePercent ?? 0,
    })),
    symbolRisks: (c.bonus.symbolRisks ?? []).map((r: { symbol?: string; riskRatio?: number }) => ({
      symbol: r.symbol || '', riskRatio: r.riskRatio ?? 0,
    })),
    symbolHoldingSplit: (c.bonus.symbolHoldingSplit ?? []).map((h: { symbol?: string; bullsSeconds?: number; shortTermSeconds?: number }) => ({
      symbol: h.symbol || '', bullsSeconds: h.bullsSeconds ?? 0, shortTermSeconds: h.shortTermSeconds ?? 0,
    })),
  };
}

export function mapMonthlyDetail(r: GetMonthlyDetailResponse): MonthlyDetailData {
  const c = deepConvertBigIntToNumber(r);
  return {
    metrics: mapMonthlyMetrics(c),
    symbolPnls: (c.symbolPnls ?? []).map((s) => ({
      symbol: s.symbol, netProfit: toNum(s.netProfit),
      trades: Number(s.trades), winRate: s.winRate,
    })),
    holdingStats: {
      averageHours: c.holdingStats?.averageHours ?? 0,
      medianHours: c.holdingStats?.medianHours ?? 0,
      maxHours: c.holdingStats?.maxHours ?? 0,
      minHours: c.holdingStats?.minHours ?? 0,
    },
    bonus: mapMonthlyBonus(c),
  };
}
