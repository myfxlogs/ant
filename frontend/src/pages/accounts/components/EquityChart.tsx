import { Area, AreaChart, CartesianGrid, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts';
import { useTranslation } from 'react-i18next';

type Props = {
  chartType: 'equity' | 'balance' | 'profit';
  chartPeriod: 'day' | 'week' | 'month' | 'all';
  data: Record<string, unknown>[];
};

export function EquityChart({ chartType, chartPeriod, data }: Props) {
  const { t } = useTranslation();

  if (data.length === 0) {
    return (
      <div className="flex items-center justify-center h-[280px]" style={{ color: 'var(--color-text-muted)' }}>
        {t('accounts.analytics.empty.equityCurve')}
      </div>
    );
  }

  return (
    <ResponsiveContainer width="100%" height={280}>
      <AreaChart data={data}>
        <defs>
          <linearGradient id="colorEquityGradient" x1="0" y1="0" x2="0" y2="1">
            <stop offset="5%" stopColor="#D4AF37" stopOpacity={0.3} />
            <stop offset="95%" stopColor="#D4AF37" stopOpacity={0} />
          </linearGradient>
          <linearGradient id="colorBalanceGradient" x1="0" y1="0" x2="0" y2="1">
            <stop offset="5%" stopColor="#2196F3" stopOpacity={0.3} />
            <stop offset="95%" stopColor="#2196F3" stopOpacity={0} />
          </linearGradient>
          <linearGradient id="colorProfitGradient" x1="0" y1="0" x2="0" y2="1">
            <stop offset="5%" stopColor="#00A651" stopOpacity={0.3} />
            <stop offset="95%" stopColor="#00A651" stopOpacity={0} />
          </linearGradient>
        </defs>
        <CartesianGrid strokeDasharray="3 3" stroke="var(--color-border)" />
        <XAxis dataKey="date" stroke="var(--color-text-muted)" fontSize={11} interval={chartPeriod === 'day' ? 1 : 'preserveStartEnd'} />
        <YAxis stroke="var(--color-text-muted)" fontSize={11} />
        <Tooltip contentStyle={{ background: 'var(--color-bg-card)', border: 'none', borderRadius: '8px', boxShadow: '0 4px 12px rgba(0, 0, 0, 0.1)' }} />
        {chartType === 'equity' && <Area type="monotone" dataKey="equity" stroke="#D4AF37" strokeWidth={2} fillOpacity={1} fill="url(#colorEquityGradient)" name={t('accounts.analytics.chartSeries.equity')} isAnimationActive={false} />}
        {chartType === 'balance' && <Area type="monotone" dataKey="balance" stroke="#2196F3" strokeWidth={2} fillOpacity={1} fill="url(#colorBalanceGradient)" name={t('accounts.analytics.chartSeries.balance')} isAnimationActive={false} />}
        {chartType === 'profit' && <Area type="monotone" dataKey="profit" stroke="#00A651" strokeWidth={2} fillOpacity={1} fill="url(#colorProfitGradient)" name={t('accounts.analytics.chartSeries.profit')} isAnimationActive={false} />}
      </AreaChart>
    </ResponsiveContainer>
  );
}
