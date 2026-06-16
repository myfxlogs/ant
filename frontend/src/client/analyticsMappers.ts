/**
 * Proto→Frontend mappers for analytics types.
 * Converts proto types (with bigint, string monetary fields) to
 * frontend-friendly types (with number, required fields with defaults).
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
  TradeStats as ProtoTradeStats,
  RiskMetrics as ProtoRiskMetrics,
  SymbolStat as ProtoSymbolStat,
  EquityPoint as ProtoEquityPoint,
  DailyPnL as ProtoDailyPnL,
  HourlyStat as ProtoHourlyStat,
  TradeRecord as ProtoTradeRecord,
} from '../gen/ant/v1/analytics_pb';
import { deepConvertBigIntToNumber } from '@/adapters/dataAdapter';
import type {
  TradeStats,
  RiskMetrics,
  SymbolStat,
  EquityPoint,
  DailyPnL,
  HourlyStat,
  AccountAnalyticsData,
  TradeRecordItem,
  RecentTradesData,
  MonthlyPnLData,
  MonthlyAnalysisData,
  AttributionAnalysisData,
  RollingMetricsData,
  MonthlyDetailData,
} from './analytics';

function toNum(v: string | undefined | null): number {
  if (v == null || v === '') return 0;
  const n = Number(v);
  return Number.isFinite(n) ? n : 0;
}

export function mapTradeStats(s: ProtoTradeStats): TradeStats {
  return {
    totalTrades: Number(s.totalTrades),
    winRate: s.winRate,
    profitFactor: s.profitFactor,
    averageProfit: toNum(s.averageProfit),
    averageLoss: toNum(s.averageLoss),
    largestWin: toNum(s.largestWin),
    largestLoss: toNum(s.largestLoss),
    maxConsecutiveWins: Number(s.maxConsecutiveWins),
    maxConsecutiveLosses: Number(s.maxConsecutiveLosses),
    averageHoldingTime: s.averageHoldingTime,
    netProfit: toNum(s.netProfit),
    totalDeposit: toNum(s.totalDeposit),
    totalWithdrawal: toNum(s.totalWithdrawal),
    netDeposit: toNum(s.netDeposit),
  };
}

export function mapRiskMetrics(r: ProtoRiskMetrics): RiskMetrics {
  return {
    maxDrawdownPercent: r.maxDrawdownPercent,
    sharpeRatio: r.sharpeRatio,
    sortinoRatio: r.sortinoRatio,
    calmarRatio: r.calmarRatio,
    volatility: r.volatility,
    averageDailyReturn: r.averageDailyReturn,
  };
}

export function mapSymbolStat(s: ProtoSymbolStat): SymbolStat {
  return {
    symbol: s.symbol,
    profit: toNum(s.profit),
    tradeSharePercent: s.tradeSharePercent,
  };
}

export function mapEquityPoint(e: ProtoEquityPoint): EquityPoint {
  return {
    date: e.date,
    equity: toNum(e.equity),
    balance: toNum(e.balance),
    profit: toNum(e.profit),
  };
}

export function mapDailyPnL(d: ProtoDailyPnL): DailyPnL {
  return {
    day: d.day,
    date: d.date,
    pnl: toNum(d.pnl),
    trades: Number(d.trades),
    lots: toNum(d.lots),
    balance: toNum(d.balance),
    profitFactor: d.profitFactor,
    maxFloatingLossAmount: toNum(d.maxFloatingLossAmount),
    maxFloatingLossRatio: d.maxFloatingLossRatio,
    maxFloatingProfitAmount: toNum(d.maxFloatingProfitAmount),
    maxFloatingProfitRatio: d.maxFloatingProfitRatio,
  };
}

export function mapHourlyStat(h: ProtoHourlyStat): HourlyStat {
  return {
    hour: h.hour,
    lots: toNum(h.lots),
    balance: toNum(h.balance),
    profitFactor: h.profitFactor,
    maxFloatingLossAmount: toNum(h.maxFloatingLossAmount),
    maxFloatingLossRatio: h.maxFloatingLossRatio,
    maxFloatingProfitAmount: toNum(h.maxFloatingProfitAmount),
    maxFloatingProfitRatio: h.maxFloatingProfitRatio,
  };
}

export function mapTradeRecord(t: ProtoTradeRecord): TradeRecordItem {
  return {
    ticket: Number(t.ticket),
    symbol: t.symbol,
    type: t.type,
    volume: toNum(t.volume),
    openPrice: toNum(t.openPrice),
    closePrice: toNum(t.closePrice),
    profit: toNum(t.profit),
    openTime: t.openTime,
    closeTime: t.closeTime,
    swap: toNum(t.swap),
    commission: toNum(t.commission),
    comment: t.comment,
  };
}

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
  return {
    years: r.years || [],
    data: r.data ?? [],
  };
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

export function mapMonthlyDetail(r: GetMonthlyDetailResponse): MonthlyDetailData {
  const c = deepConvertBigIntToNumber(r);
  return {
    metrics: {
      netReturn: toNum(c.metrics?.netReturn),
      returnPercent: c.metrics?.returnPercent ?? 0,
      totalTrades: Number(c.metrics?.totalTrades ?? 0),
      winRate: c.metrics?.winRate ?? 0,
      profitFactor: c.metrics?.profitFactor ?? 0,
      bestTrade: toNum(c.metrics?.bestTrade),
      worstTrade: toNum(c.metrics?.worstTrade),
    },
    symbolPnls: (c.symbolPnls ?? []).map((s) => ({
      symbol: s.symbol,
      netProfit: toNum(s.netProfit),
      trades: Number(s.trades),
      winRate: s.winRate,
    })),
    holdingStats: {
      averageHours: c.holdingStats?.averageHours ?? 0,
      medianHours: c.holdingStats?.medianHours ?? 0,
      maxHours: c.holdingStats?.maxHours ?? 0,
      minHours: c.holdingStats?.minHours ?? 0,
    },
    bonus: c.bonus ? {
      riskRatio: c.bonus.riskRatio,
      symbolPopularity: (c.bonus.symbolPopularity ?? []).map((s) => ({
        symbol: s.symbol,
        trades: Number(s.trades),
        sharePercent: s.sharePercent,
      })),
      symbolRisks: (c.bonus.symbolRisks ?? []).map((r) => ({
        symbol: r.symbol,
        riskRatio: r.riskRatio,
      })),
      symbolHoldingSplit: (c.bonus.symbolHoldingSplit ?? []).map((h) => ({
        symbol: h.symbol,
        bullsSeconds: h.bullsSeconds,
        shortTermSeconds: h.shortTermSeconds,
      })),
    } : undefined,
  };
}
