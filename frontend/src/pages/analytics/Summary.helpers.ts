import { COLORS } from './Summary.constants';

interface TradeStatsLike {
  totalTrades?: number;
  winRate?: number;
}

interface EquityCurvePoint {
  date?: string;
  equity?: number;
}

interface SymbolStatLike {
  symbol: string;
  tradeSharePercent: number;
}

interface DailyPnlLike {
  day?: string;
  date?: string;
  pnl?: number;
  trades?: number;
  lots?: number;
  balance?: number;
}

export const getEquityCurveData = (equityCurve: EquityCurvePoint[]) => {
  return (equityCurve || []).map((p) => ({
    date: String(p?.date || ''),
    equity: Number(p?.equity || 0),
    balance: Number(p?.balance || 0),
  }));
};

export const getMonthlyData = (dailyPnl: DailyPnlLike[]) => {
  return (dailyPnl || []).map((d) => ({
    month: String(d?.date || d?.day || ''),
    profit: Number(d?.pnl || 0),
    trades: Number(d?.trades || 0),
  }));
};

export const getSymbolPieData = (symbolStats: SymbolStatLike[]) => {
  return (symbolStats || []).slice(0, 5).map((s, index) => ({
    name: s?.symbol ?? '',
    value: s?.tradeSharePercent ?? 0,
    color: COLORS[index % COLORS.length],
  }));
};

export interface DirectionBreakdownLike {
  longTrades?: number;
  shortTrades?: number;
}

export const getDirectionPieData = (t: (key: string, opts?: Record<string, unknown>) => string, direction: DirectionBreakdownLike | null | undefined) => {
  const long = Number(direction?.longTrades || 0);
  const short = Number(direction?.shortTrades || 0);
  if (long === 0 && short === 0) {
    return [
      { name: t('analytics.summary.direction.buy'), value: 0, color: '#00A651' },
      { name: t('analytics.summary.direction.sell'), value: 0, color: '#E53935' },
    ];
  }
  return [
    { name: t('analytics.summary.direction.buy'), value: long, color: '#00A651' },
    { name: t('analytics.summary.direction.sell'), value: short, color: '#E53935' },
  ];
};

export const getProfitPieData = (t: (key: string, opts?: Record<string, unknown>) => string, tradeStats: TradeStatsLike | null) => {
  const total = Number(tradeStats?.totalTrades || 0);
  const winRate = Number(tradeStats?.winRate || 0);
  const wins = Math.round(total * winRate / 100);
  const losses = total - wins;
  return [
    { name: t('analytics.summary.profit.win'), value: wins, color: '#00A651' },
    { name: t('analytics.summary.profit.loss'), value: losses, color: '#E53935' },
  ];
};

export const getYearOptions = (t: (key: string, opts?: Record<string, unknown>) => string) => {
  const yearOptions: { value: number; label: string }[] = [];
  const currentYear = new Date().getFullYear();
  for (let y = currentYear; y >= currentYear - 5; y--) {
    yearOptions.push({ value: y, label: t('analytics.summary.yearOption', { year: y }) });
  }
  return yearOptions;
};
