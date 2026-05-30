/**
 * Proto→Frontend mappers for analytics types.
 * Converts proto types (with bigint, optional nested messages) to
 * frontend-friendly types (with number, required fields with defaults).
 */
import type {
  AccountAnalyticsResponse,
  GetRecentTradesResponse,
  GetMonthlyPnLResponse,
  GetMonthlyAnalysisResponse,
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
} from './analytics';

export function mapTradeStats(s: ProtoTradeStats): TradeStats {
  return {
    totalTrades: Number(s.totalTrades),
    winRate: s.winRate,
    profitFactor: s.profitFactor,
    averageProfit: s.averageProfit,
    averageLoss: s.averageLoss,
    largestWin: s.largestWin,
    largestLoss: s.largestLoss,
    maxConsecutiveWins: Number(s.maxConsecutiveWins),
    maxConsecutiveLosses: Number(s.maxConsecutiveLosses),
    averageHoldingTime: s.averageHoldingTime,
    netProfit: s.netProfit,
    totalDeposit: s.totalDeposit,
    totalWithdrawal: s.totalWithdrawal,
    netDeposit: s.netDeposit,
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
    profit: s.profit,
    tradeSharePercent: s.tradeSharePercent,
  };
}

export function mapEquityPoint(e: ProtoEquityPoint): EquityPoint {
  return {
    date: e.date,
    equity: e.equity,
    balance: e.balance,
    profit: e.profit,
  };
}

export function mapDailyPnL(d: ProtoDailyPnL): DailyPnL {
  return {
    day: d.day,
    date: d.date,
    pnl: d.pnl,
    trades: Number(d.trades),
    lots: d.lots,
    balance: d.balance,
    profitFactor: d.profitFactor,
    maxFloatingLossAmount: d.maxFloatingLossAmount,
    maxFloatingLossRatio: d.maxFloatingLossRatio,
    maxFloatingProfitAmount: d.maxFloatingProfitAmount,
    maxFloatingProfitRatio: d.maxFloatingProfitRatio,
  };
}

export function mapHourlyStat(h: ProtoHourlyStat): HourlyStat {
  return {
    hour: h.hour,
    lots: h.lots,
    balance: h.balance,
    profitFactor: h.profitFactor,
    maxFloatingLossAmount: h.maxFloatingLossAmount,
    maxFloatingLossRatio: h.maxFloatingLossRatio,
    maxFloatingProfitAmount: h.maxFloatingProfitAmount,
    maxFloatingProfitRatio: h.maxFloatingProfitRatio,
  };
}

export function mapTradeRecord(t: ProtoTradeRecord): TradeRecordItem {
  return {
    ticket: Number(t.ticket),
    symbol: t.symbol,
    type: t.type,
    volume: t.volume,
    openPrice: t.openPrice,
    closePrice: t.closePrice,
    profit: t.profit,
    openTime: t.openTime,
    closeTime: t.closeTime,
    swap: t.swap,
    commission: t.commission,
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
      profit: item.profit,
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
