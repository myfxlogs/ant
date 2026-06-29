import { TIMEFRAME_MAX_MONTHS } from '@/pages/strategy/hooks/backtestParamHelpers';

export type BacktestStatus = 'idle' | 'running' | 'completed' | 'error';

export interface ChartTrade {
  side: string;
  openTime: number;
  openPrice: number;
  closeTime?: number;
  closePrice?: number;
  pnl?: number;
  volume?: number;
}

export interface BacktestMetrics {
  totalReturn?: number; annualReturn?: number; maxDrawdown?: number;
  sharpeRatio?: number; winRate?: number; totalTrades?: number;
  profitFactor?: number;
  winningTrades?: number; losingTrades?: number;
  averageProfit?: number; averageLoss?: number;
}

export interface ExtractedParam {
  name: string; type: string; default: string; label: string;
}

export interface StandardParams {
  initialCapital: number;
  leverage: number;
  lotSize: number;
  commission: number;
  slippage: number;
  tradeDirection: string;
  strictMode: boolean;
}

export const FACTORY_DEFAULTS: StandardParams = {
  initialCapital: 10000, leverage: 1, lotSize: 0.01,
  commission: 0.001, slippage: 0.0,
  tradeDirection: 'both', strictMode: true,
};

const DEFAULTS_KEY = 'ant_backtest_defaults';

export function getTimeframeWarning(timeframe: string, presetMonths: number): string | null {
  const maxMonths = (TIMEFRAME_MAX_MONTHS as Record<string, number>)[timeframe];
  if (maxMonths && presetMonths > maxMonths) {
    return `${timeframe} timeframe recommends max ${maxMonths} month${maxMonths > 1 ? 's' : ''}`;
  }
  return null;
}

export function loadSavedDefaults(): StandardParams | null {
  try { const raw = localStorage.getItem(DEFAULTS_KEY); return raw ? JSON.parse(raw) : null; }
  catch { return null; }
}

export function saveDefaults(vals: StandardParams) {
  try { localStorage.setItem(DEFAULTS_KEY, JSON.stringify(vals)); } catch { /* quota */ }
}

export function removeDefaults() {
  try { localStorage.removeItem(DEFAULTS_KEY); } catch { /* ignore */ }
}

export interface BacktestRunnerInputs {
  strategyCode: string;
  accountId: string;
  symbol: string;
  timeframe: string;
  templateId?: string;
  strategyId?: string;
}
