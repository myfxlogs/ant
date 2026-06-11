import React, { useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import type { MonthlyDetailData } from '@/client/analytics';

type Props = {
  detail: MonthlyDetailData;
  currency: string;
};

const sectionStyle = {
  background: 'var(--color-bg-secondary)',
  border: '1px solid var(--color-border)',
  borderRadius: 8,
  padding: 10,
} as const;

const titleStyle = {
  color: 'var(--color-text)',
  fontSize: 12,
  fontWeight: 600,
  marginBottom: 8,
} as const;

const labelStyle = { color: 'var(--color-text-muted)', fontSize: 10, lineHeight: 1.5 };
const valueStyle = (color = 'var(--color-text)') => ({ color, fontSize: 12, fontWeight: 600, lineHeight: 1.5 });

const MetricRow = React.memo(({ label, value, color }: { label: string; value: string; color?: string }) => (
  <div className="flex justify-between items-baseline">
    <span style={labelStyle}>{label}</span>
    <span style={valueStyle(color)}>{value}</span>
  </div>
));

const SymbolBar = React.memo(({ symbol, profit, widthPct, color }: { symbol: string; profit: string; widthPct: number; color: string }) => (
  <div className="flex items-center gap-1.5 mb-1">
    <span style={{ ...labelStyle, width: 56, flexShrink: 0, textAlign: 'right' }}>{symbol}</span>
    <div className="flex-1 h-3 rounded-sm" style={{ background: 'var(--color-bg-card)', overflow: 'hidden' }}>
      <div className="h-full rounded-sm transition-all" style={{ width: `${Math.max(widthPct, 2)}%`, background: color }} />
    </div>
    <span style={{ ...valueStyle(color), width: 64, flexShrink: 0 }}>{profit}</span>
  </div>
));

export default function MonthlyDrillDown({ detail, currency }: Props) {
  const { t } = useTranslation();
  const { metrics, symbolPnls, holdingStats } = detail;

  const symbolBars = useMemo(() => {
    if (!symbolPnls.length) return null;
    const maxAbs = Math.max(...symbolPnls.map((s) => Math.abs(s.netProfit)), 1);
    return symbolPnls.slice(0, 8).map((s) => ({
      ...s,
      widthPct: (Math.abs(s.netProfit) / maxAbs) * 100,
      color: s.netProfit >= 0 ? 'var(--color-success)' : 'var(--color-danger)',
    }));
  }, [symbolPnls]);

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3" style={{ borderTop: '1px solid var(--color-border)', paddingTop: 12 }}>
      {/* Panel 1: Monthly Metrics */}
      <div style={sectionStyle}>
        <h4 style={titleStyle}>{t('accounts.analytics.monthlyDetail.metricsTitle')}</h4>
        <div className="space-y-0.5">
          <MetricRow label={t('accounts.analytics.monthlyDetail.fields.netReturn')}
            value={`${metrics.netReturn >= 0 ? '+' : ''}${metrics.netReturn.toFixed(2)} ${currency}`}
            color={metrics.netReturn >= 0 ? 'var(--color-success)' : 'var(--color-danger)'} />
          <MetricRow label={t('accounts.analytics.monthlyDetail.fields.totalTrades')}
            value={`${metrics.totalTrades}`} />
          <MetricRow label={t('accounts.analytics.monthlyDetail.fields.winRate')}
            value={`${metrics.winRate.toFixed(1)}%`} />
          <MetricRow label={t('accounts.analytics.monthlyDetail.fields.profitFactor')}
            value={metrics.profitFactor.toFixed(2)} />
          <MetricRow label={t('accounts.analytics.monthlyDetail.fields.bestTrade')}
            value={`+${metrics.bestTrade.toFixed(2)}`} color="var(--color-success)" />
          <MetricRow label={t('accounts.analytics.monthlyDetail.fields.worstTrade')}
            value={`${metrics.worstTrade.toFixed(2)}`} color="var(--color-danger)" />
        </div>
      </div>

      {/* Panel 2: Symbol P&L */}
      <div style={sectionStyle}>
        <h4 style={titleStyle}>{t('accounts.analytics.monthlyDetail.symbolPnLTitle')}</h4>
        {symbolBars && symbolBars.length > 0 ? (
          <div>
            {symbolBars.map((s) => (
              <SymbolBar key={s.symbol} symbol={s.symbol}
                profit={`${s.netProfit >= 0 ? '+' : ''}${s.netProfit.toFixed(2)}`}
                widthPct={s.widthPct} color={s.color} />
            ))}
          </div>
        ) : (
          <div style={{ color: 'var(--color-text-muted)', fontSize: 11, paddingTop: 4 }}>
            {t('accounts.analytics.empty.monthlyProfit')}
          </div>
        )}
      </div>

      {/* Panel 3: Holding Time Stats */}
      <div style={sectionStyle}>
        <h4 style={titleStyle}>{t('accounts.analytics.monthlyDetail.holdingTitle')}</h4>
        <div className="space-y-0.5">
          <MetricRow label={t('accounts.analytics.monthlyDetail.fields.averageHours')}
            value={`${holdingStats.averageHours.toFixed(1)}h`} />
          <MetricRow label={t('accounts.analytics.monthlyDetail.fields.medianHours')}
            value={`${holdingStats.medianHours.toFixed(1)}h`} />
          <MetricRow label={t('accounts.analytics.monthlyDetail.fields.maxHours')}
            value={`${holdingStats.maxHours.toFixed(1)}h`} />
          <MetricRow label={t('accounts.analytics.monthlyDetail.fields.minHours')}
            value={`${holdingStats.minHours.toFixed(1)}h`} />
        </div>
      </div>
    </div>
  );
}
