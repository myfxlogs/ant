export const COLORS = ['#D4AF37', '#2196F3', '#00A651', '#E53935', '#9C27B0', '#FF9800'];

export function periodOptions(t: (key: string, opts?: Record<string, any>) => string) {
  return [
    { value: 'today', label: t('analytics.summary.periods.today') },
    { value: 'week', label: t('analytics.summary.periods.week') },
    { value: 'month', label: t('analytics.summary.periods.month') },
    { value: 'year', label: t('analytics.summary.periods.year') },
    { value: 'all', label: t('analytics.summary.periods.all') },
  ];
}
