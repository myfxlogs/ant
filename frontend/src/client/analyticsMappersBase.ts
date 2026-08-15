import type {
  TradeStats as ProtoTradeStats,
  RiskMetrics as ProtoRiskMetrics,
  SymbolStat as ProtoSymbolStat,
  EquityPoint as ProtoEquityPoint,
  DailyPnL as ProtoDailyPnL,
  HourlyStat as ProtoHourlyStat,
  TradeRecord as ProtoTradeRecord,
} from '../gen/ant/v1/analytics_pb';
import type {
  TradeStats,
  RiskMetrics,
  SymbolStat,
  EquityPoint,
  DailyPnL,
  HourlyStat,
  TradeRecordItem,
} from './analytics';

export function toNum(v: string | undefined | null): number {
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
    magicNumber: t.magicNumber ?? 0,
  };
}
