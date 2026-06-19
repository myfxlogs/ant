import { Area, AreaChart, CartesianGrid, ComposedChart, Line, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts';
import { useTranslation } from 'react-i18next'
import { ANALYTICS_CHART_SERIES_BALANCE_KEY, ANALYTICS_CHART_SERIES_EQUITY_KEY, ANALYTICS_CHART_SERIES_PROFIT_KEY, ANALYTICS_EMPTY_EQUITY_CURVE_KEY } from '@/gen/ant/v1/i18n/accounts_keys';

;

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
        {t(ANALYTICS_EMPTY_EQUITY_CURVE_KEY)}
      </div>
    );
  }

  const hasDrawdown = typeof data[0]?.drawdown === 'number';
  const Chart = hasDrawdown ? ComposedChart : AreaChart;

  return (
    <ResponsiveContainer width="100%" height={280}>
      <Chart data={data}>
        <defs>
          <linearGradient id="colorEquityGradient" x1="0" y1="0" x2="0" y2="1">
            <stop offset="5%" stopColor="var(--color-primary)" stopOpacity={0.3} />
            <stop offset="95%" stopColor="var(--color-primary)" stopOpacity={0} />
          </linearGradient>
<linearGradient id="colorBalanceGradient" x1="0" y1="0" x2="0" y2="1">
            <stop offset="5%" stopColor="var(--color-info)" stopOpacity={0.3} />
            <stop offset="95%" stopColor="var(--color-info)" stopOpacity={0} />
          </linearGradient>
          <linearGradient id="colorProfitGradient" x1="0" y1="0" x2="0" y2="1">
            <stop offset="5%" stopColor="var(--color-success)" stopOpacity={0.3} />
            <stop offset="95%" stopColor="var(--color-success)" stopOpacity={0} />
          </linearGradient>
        </defs>
        <CartesianGrid strokeDasharray="3 3" stroke="var(--color-border)" />
        <XAxis dataKey="date" stroke="var(--color-text-muted)" fontSize={11} interval={chartPeriod === 'day' ? 1 : 'preserveStartEnd'} />
        <YAxis yAxisId="left" stroke="var(--color-text-muted)" fontSize={11} />
        {hasDrawdown && <YAxis yAxisId="right" orientation="right" stroke="var(--color-danger)" fontSize={11} reversed domain={[100, 0]} />}
        <Tooltip contentStyle={{ background: 'var(--color-bg-card)', border: 'none', borderRadius: '8px', boxShadow: '0 4px 12px var(--color-shadow)' }} />
        {chartType === 'equity' && <Area yAxisId="left" type="monotone" dataKey="equity" stroke="var(--color-primary)" strokeWidth={2} fillOpacity={1} fill="url(#colorEquityGradient)" name={t(ANALYTICS_CHART_SERIES_EQUITY_KEY)} isAnimationActive={false} />}
        {chartType === 'balance' && <Area yAxisId="left" type="monotone" dataKey="balance" stroke="var(--color-info)" strokeWidth={2} fillOpacity={1} fill="url(#colorBalanceGradient)" name={t(ANALYTICS_CHART_SERIES_BALANCE_KEY)} isAnimationActive={false} />}
        {chartType === 'profit' && <Area yAxisId="left" type="monotone" dataKey="profit" stroke="var(--color-success)" strokeWidth={2} fillOpacity={1} fill="url(#colorProfitGradient)" name={t(ANALYTICS_CHART_SERIES_PROFIT_KEY)} isAnimationActive={false} />}
        {hasDrawdown && <Line yAxisId="right" type="monotone" dataKey="drawdown" stroke="var(--color-danger)" strokeWidth={1.5} dot={false} name="DD%" isAnimationActive={false} />}
      </Chart>
    </ResponsiveContainer>
  );
}
