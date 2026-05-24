import { CHART_COLORS } from '@/constants/performance';

export type MetricType = 'change' | 'profit' | 'lots' | 'pips';

export type MonthlyAnalysisPoint = {
  year: number;
  month: number;
  change: number;
  profit: number;
  lots: number;
  pips: number;
  trades?: number;
};

export type MonthlyBarRow = MonthlyAnalysisPoint & {
  monthAxisLabel: string;
  value: number;
  isActive: boolean;
};

export function monthFromBarClick(data: unknown, index: number, rows: MonthlyBarRow[]): number | null {
  const payload = (data as { payload?: { month?: number } })?.payload;
  if (typeof payload?.month === 'number' && payload.month >= 1 && payload.month <= 12) {
    return payload.month;
  }
  const row = rows[index];
  if (row && typeof row.month === 'number' && row.month >= 1 && row.month <= 12) {
    return row.month;
  }
  return null;
}

export type BonusPayload = {
  riskRatio: number;
  symbolPopularity: { symbol: string; trades: number; sharePercent: number }[];
  symbolRisks: { symbol: string; riskRatio: number }[];
  symbolHoldingSplit: { symbol: string; bullsSeconds: number; shortTermSeconds: number }[];
  averageHoldingSeconds: number;
  totalTrades: number;
};

export type MonthlyAnalysisCardProps = {
  accountId?: string;
  years: number[];
  data: MonthlyAnalysisPoint[];
  currency?: string;
};

export const monthShortLabels = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];

/** Myfxbook-style pastel bars (rotate by month). */
export const MONTH_BAR_PASTELS = [
  '#9B8FD9',
  '#E879A9',
  '#4DB6AC',
  '#FF9E5E',
  '#A3D977',
  '#7EB6E8',
  '#D4A574',
  '#90CAF9',
  '#CE93D8',
  '#80CBC4',
  '#FFAB91',
  '#B39DDB',
];

export function barCellFill(item: MonthlyBarRow): string {
  const v = item.value;
  const isEmpty = !Number.isFinite(v) || Math.abs(v) < 1e-12;
  if (item.isActive) return '#2B6CB0';
  if (isEmpty) return 'rgba(148, 163, 184, 0.42)';
  return MONTH_BAR_PASTELS[(item.month - 1) % MONTH_BAR_PASTELS.length];
}

export const PIE_COLORS = [
  ...CHART_COLORS,
  '#795548',
  '#607D8B',
  '#3F51B5',
  '#009688',
  '#CDDC39',
];

export function formatSecondsAxis(sec: number): string {
  if (!Number.isFinite(sec) || sec <= 0) return '0';
  if (sec < 60) return `${Math.round(sec)}s`;
  if (sec < 3600) return `${(sec / 60).toFixed(1)}min`;
  return `${(sec / 3600).toFixed(2)}hr`;
}

/** Bar length (axis); raw value shown in tooltip. */
export function riskBarVisual(raw: number): number {
  if (!Number.isFinite(raw)) return 0;
  if (raw >= 999.98) return 100;
  return Math.min(raw, 200);
}
