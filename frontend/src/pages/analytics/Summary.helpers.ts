import { COLORS } from './Summary.constants';

type TradeStatsLike = {
  totalTrades?: number;
  winningTrades?: number;
  losingTrades?: number;
  buyTrades?: number;
  sellTrades?: number;
} | null;

type SymbolStatLike = {
  symbol: string;
  trades: number;
} | null;

export const getEquityCurveData = (equityCurve: any[]) => {
  return (equityCurve || []).map((p: any) => ({
    date: String(p?.date || ''),
    equity: Number(p?.equity || 0),
  }));
};

export const getMonthlyData = (monthlyPnL: any[]) => {
  return (monthlyPnL || []).map((m: any) => ({
    month: String(m?.month || ''),
    profit: Number(m?.profit || 0),
    trades: Number(m?.trades || 0),
  }));
};

export const getSymbolPieData = (symbolStats: any[]) => {
  return (symbolStats || []).slice(0, 5).map((s: SymbolStatLike, index: number) => ({
    name: (s as any)?.symbol,
    value: (s as any)?.trades,
    color: COLORS[index % COLORS.length],
  }));
};

export const getDirectionPieData = (t: (key: string, opts?: Record<string, any>) => string, tradeStats: TradeStatsLike) => {
  const buyTrades = Number((tradeStats as any)?.buyTrades || 0);
  const sellTrades = Number((tradeStats as any)?.sellTrades || 0);
  return [
    { name: t('analytics.summary.direction.buy'), value: buyTrades, color: '#00A651' },
    { name: t('analytics.summary.direction.sell'), value: sellTrades, color: '#E53935' },
  ];
};

export const getProfitPieData = (t: (key: string, opts?: Record<string, any>) => string, tradeStats: TradeStatsLike) => {
  return [
    { name: t('analytics.summary.profit.win'), value: (tradeStats as any)?.winningTrades || 0, color: '#00A651' },
    { name: t('analytics.summary.profit.loss'), value: (tradeStats as any)?.losingTrades || 0, color: '#E53935' },
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
