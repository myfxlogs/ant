
import { SUMMARY_PERIODS_ALL_KEY, SUMMARY_PERIODS_MONTH_KEY, SUMMARY_PERIODS_TODAY_KEY, SUMMARY_PERIODS_WEEK_KEY, SUMMARY_PERIODS_YEAR_KEY } from '@/gen/ant/v1/i18n/analytics_keys';

export const COLORS = ['#D4AF37', '#2196F3', '#00A651', '#E53935', '#9C27B0', '#FF9800'];

export function periodOptions(t: (key: string, opts?: Record<string, unknown>) => string) {
  return [
    { value: 'today', label: t(SUMMARY_PERIODS_TODAY_KEY) },
    { value: 'week', label: t(SUMMARY_PERIODS_WEEK_KEY) },
    { value: 'month', label: t(SUMMARY_PERIODS_MONTH_KEY) },
    { value: 'year', label: t(SUMMARY_PERIODS_YEAR_KEY) },
    { value: 'all', label: t(SUMMARY_PERIODS_ALL_KEY) },
  ];
}
