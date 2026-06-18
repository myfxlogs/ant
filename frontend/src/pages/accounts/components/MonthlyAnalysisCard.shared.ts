import { CHART_COLORS } from '@/constants/performance';

export type MonthlyAnalysisPoint = {
  year: number;
  month: number;
  change: number;
  profit: number;
  lots: number;
  pips: number;
  trades?: number;
  winRate?: number;
};

export type MonthlyBarRow = MonthlyAnalysisPoint & {
  monthAxisLabel: string;
  value: number;
  isActive: boolean;
};

export function monthFromBarClick(data: unknown, index: number, rows: MonthlyBarRow[]): number | null {
  // recharts v3: data is BarRectangleItem with payload containing the chart data entry
  const v3Payload = (data as { payload?: MonthlyBarRow })?.payload;
  if (typeof v3Payload?.month === 'number' && v3Payload.month >= 1 && v3Payload.month <= 12) {
    return v3Payload.month;
  }
  // recharts v2 / direct entry
  const direct = data as MonthlyBarRow | undefined;
  if (typeof direct?.month === 'number' && direct.month >= 1 && direct.month <= 12) {
    return direct.month;
  }
  // Fallback by numeric index into the series array
  const row = rows[index];
  if (row && typeof row.month === 'number' && row.month >= 1 && row.month <= 12) {
    return row.month;
  }
  return null;
}

export type MonthlyWinRatePoint = { month: string; winRate: number; totalTrades: number };

export type MonthlyAnalysisCardProps = {
  accountId?: string;
  years: number[];
  data: MonthlyAnalysisPoint[];
  winRateData?: MonthlyWinRatePoint[];
  currency?: string;
};

export const monthShortLabels = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];

/** Returns translated month abbreviations for the current locale. */
export function getMonthLabels(t: (key: string) => string): string[] {
  return [
    t('common.months.jan'), t('common.months.feb'), t('common.months.mar'),
    t('common.months.apr'), t('common.months.may'), t('common.months.jun'),
    t('common.months.jul'), t('common.months.aug'), t('common.months.sep'),
    t('common.months.oct'), t('common.months.nov'), t('common.months.dec'),
  ];
}

/** Myfxbook-style pastel bar colors — rotating palette per month. */
export const MONTH_BAR_PASTELS = [
  '#B486B4', // Jan — mauve
  '#E28686', // Feb — soft red
  '#5AB4B4', // Mar — teal
  'rgba(255, 183, 137, 0.9)', // Apr — peach
  '#B5D35D', // May — lime
  '#F6D263', // Jun — gold
  '#CCE6FA', // Jul — ice blue
  '#E28686', // Aug — soft red
  '#5AB4B4', // Sep — teal
  'rgba(255, 183, 137, 0.9)', // Oct — peach
  '#B5D35D', // Nov — lime
  '#B486B4', // Dec — mauve
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
