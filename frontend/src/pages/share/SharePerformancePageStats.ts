import type { ShareData } from './SharePerformancePageHelpers';
import { toNum, fmt, avgHoldingText } from './SharePerformancePageHelpers';
import type { TFunction } from 'i18next';

export function buildKpiCards(t: TFunction, data: ShareData) {
  const green = '#52c41a', red = '#ff4d4f';
  const money = (n: number, d = 2) => Number.isFinite(n) ? n.toLocaleString('en', { minimumFractionDigits: d, maximumFractionDigits: d }) : '-';
  const signed = (n: number) => `${n >= 0 ? '+' : ''}${money(n)}`;
  const ts = data.tradeStats;
  return [
    { label: t('sharePage.winRate'), value: `${fmt(toNum(data.winRate), 1)}%`, color: '#1677ff', icon: null },
    { label: t('sharePage.profitFactor'), value: toNum(data.profitFactor) > 0 ? fmt(toNum(data.profitFactor), 2) : 'N/A', color: '#eb2f96', icon: null },
    { label: t('sharePage.maxDrawdown'), value: `${fmt(toNum(data.maxDrawdown), 2)}%`, color: '#fa8c16', icon: null },
    { label: t('sharePage.sharpeRatio'), value: fmt(toNum(data.sharpeRatio), 2), color: '#a0d911', icon: null },
    { label: t('sharePage.totalTrades'), value: String(data.totalTrades || 0), color: '#722ed1', icon: null },
    { label: t('sharePage.totalVolume'), value: fmt(toNum(data.totalVolume), 1), color: '#13c2c2', icon: null },
    { label: t('sharePage.avgHolding'), value: avgHoldingText(toNum(data.avgHoldingMs)), color: '#2f54eb', icon: null },
    { label: t('sharePage.bestTrade'), value: ts ? signed(toNum(ts.bestTrade)) : '-', color: green, icon: null },
    { label: t('sharePage.worstTrade'), value: ts ? signed(toNum(ts.worstTrade)) : '-', color: red, icon: null },
    { label: t('sharePage.avgWin'), value: ts ? signed(toNum(ts.avgWin)) : '-', color: green, icon: null },
    { label: t('sharePage.avgLoss'), value: ts ? signed(toNum(ts.avgLoss)) : '-', color: red, icon: null },
    { label: `Win / Loss`, value: ts ? `${ts.winningTrades} / ${ts.losingTrades}` : '-', color: '#1677ff', icon: null },
  ];
}
