import type { ShareData } from './SharePerformancePageHelpers';
import { toNum, fmt, avgHoldingText, computeMaxDrawdownPct, aggregateBySymbol } from './SharePerformancePageHelpers';
import type { TFunction } from 'i18next';

export interface TradeStats {
  winningTrades: number;
  losingTrades: number;
  bestTrade: number;
  worstTrade: number;
  avgWin: number;
  avgLoss: number;
  winPct: number;
  netProfit: number;
  isPositive: boolean;
  maxDrawdownPct: number;
  bySymbol: ReturnType<typeof aggregateBySymbol>;
}

export function computeTradeStats(data: ShareData): TradeStats {
  const trades = data.trades || [];
  const netProfit = toNum(data.totalReturn);
  const isPositive = netProfit >= 0;
  const profits = trades.map(tr => toNum(tr.profit));
  const winningProfits = profits.filter(p => p > 0);
  const losingProfits = profits.filter(p => p < 0);
  const winningTrades = winningProfits.length;
  const losingTrades = losingProfits.length;
  const bestTrade = profits.length ? Math.max(...profits) : 0;
  const worstTrade = profits.length ? Math.min(...profits) : 0;
  const avgWin = winningTrades ? winningProfits.reduce((a, b) => a + b, 0) / winningTrades : 0;
  const avgLoss = losingTrades ? losingProfits.reduce((a, b) => a + b, 0) / losingTrades : 0;
  const winPct = (winningTrades + losingTrades) > 0 ? Math.round(winningTrades / (winningTrades + losingTrades) * 100) : 0;
  const equity = data.equityCurve || [];
  const maxDrawdownPct = computeMaxDrawdownPct(equity, toNum(data.maxDrawdown));
  const bySymbol = aggregateBySymbol(trades);
  return { winningTrades, losingTrades, bestTrade, worstTrade, avgWin, avgLoss, winPct, netProfit, isPositive, maxDrawdownPct, bySymbol };
}

export function buildKpiCards(t: TFunction, data: ShareData, stats: TradeStats) {
  const green = '#52c41a', red = '#ff4d4f';
  const money = (n: number, d = 2) => Number.isFinite(n) ? n.toLocaleString('en', { minimumFractionDigits: d, maximumFractionDigits: d }) : '-';
  const signed = (n: number) => `${n >= 0 ? '+' : ''}${money(n)}`;
  return [
    { label: t('sharePage.winRate'), value: `${fmt(toNum(data.winRate), 1)}%`, color: '#1677ff', icon: null },
    { label: t('sharePage.profitFactor'), value: toNum(data.profitFactor) > 0 ? fmt(toNum(data.profitFactor), 2) : 'N/A', color: '#eb2f96', icon: null },
    { label: t('sharePage.maxDrawdown'), value: `${fmt(stats.maxDrawdownPct, 2)}%`, color: '#fa8c16', icon: null },
    { label: t('sharePage.sharpeRatio'), value: fmt(toNum(data.sharpeRatio), 2), color: '#a0d911', icon: null },
    { label: t('sharePage.totalTrades'), value: String(data.totalTrades || 0), color: '#722ed1', icon: null },
    { label: t('sharePage.totalVolume'), value: fmt(toNum(data.totalVolume), 1), color: '#13c2c2', icon: null },
    { label: t('sharePage.avgHolding'), value: avgHoldingText(toNum(data.avgHoldingMs)), color: '#2f54eb', icon: null },
    { label: t('sharePage.bestTrade'), value: signed(stats.bestTrade), color: green, icon: null },
    { label: t('sharePage.worstTrade'), value: signed(stats.worstTrade), color: red, icon: null },
    { label: t('sharePage.avgWin'), value: signed(stats.avgWin), color: green, icon: null },
    { label: t('sharePage.avgLoss'), value: signed(stats.avgLoss), color: red, icon: null },
    { label: `Win / Loss`, value: `${stats.winningTrades} / ${stats.losingTrades}`, color: '#1677ff', icon: null },
  ];
}
