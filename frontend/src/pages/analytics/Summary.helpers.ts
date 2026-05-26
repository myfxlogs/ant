import { COLORS } from './Summary.constants';

interface TradeStatsLike {
  totalTrades?: number;
  winningTrades?: number;
  losingTrades?: number;
  buyTrades?: number;
  sellTrades?: number;
  winRate?: number;
  profitFactor?: number;
}

interface SymbolStatLike {
  symbol: string;
  trades: number;
}

interface EquityCurvePoint {
  date?: string;
  equity?: number;
}

interface MonthlyPnlItem {
  month?: string;
  profit?: number;
  trades?: number;
}

export const getEquityCurveData = (equityCurve: EquityCurvePoint[]) => {
  return (equityCurve || []).map((p) => ({
    date: String(p?.date || ''),
    equity: Number(p?.equity || 0),
  }));
};

export const getMonthlyData = (monthlyPnL: MonthlyPnlItem[]) => {
  return (monthlyPnL || []).map((m) => ({
    month: String(m?.month || ''),
    profit: Number(m?.profit || 0),
    trades: Number(m?.trades || 0),
  }));
};

export const getSymbolPieData = (symbolStats: SymbolStatLike[]) => {
  return (symbolStats || []).slice(0, 5).map((s, index) => ({
    name: s?.symbol ?? '',
    value: s?.trades ?? 0,
    color: COLORS[index % COLORS.length],
  }));
};

export const getDirectionPieData = (t: (key: string, opts?: Record<string, unknown>) => string, tradeStats: TradeStatsLike | null) => {
  const buyTrades = Number(tradeStats?.buyTrades || 0);
  const sellTrades = Number(tradeStats?.sellTrades || 0);
  return [
    { name: t('analytics.summary.direction.buy'), value: buyTrades, color: '#00A651' },
    { name: t('analytics.summary.direction.sell'), value: sellTrades, color: '#E53935' },
  ];
};

export const getProfitPieData = (t: (key: string, opts?: Record<string, unknown>) => string, tradeStats: TradeStatsLike | null) => {
  return [
    { name: t('analytics.summary.profit.win'), value: tradeStats?.winningTrades || 0, color: '#00A651' },
    { name: t('analytics.summary.profit.loss'), value: tradeStats?.losingTrades || 0, color: '#E53935' },
  ];
};

export const getYearOptions = (t: (key: string, opts?: Record<string, any>) => string) => {
  const yearOptions: { value: number; label: string }[] = [];
  const currentYear = new Date().getFullYear();
  for (let y = currentYear; y >= currentYear - 5; y--) {
    yearOptions.push({ value: y, label: t('analytics.summary.yearOption', { year: y }) });
  }
  return yearOptions;
};
