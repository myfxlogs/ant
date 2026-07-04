import { useTranslation } from 'react-i18next';

interface BacktestMetrics {
  totalReturn?: number;
  annualReturn?: number;
  maxDrawdown?: number;
  sharpeRatio?: number;
  winRate?: number;
  totalTrades?: number;
  equityCurve?: Array<{ time: number; equity: number }>;
}

interface Props {
  metrics: BacktestMetrics | null;
  status: string;
}

function fmtPct(v: number | undefined, suffix = '%'): string {
  if (v == null) return '—';
  return `${v >= 0 ? '+' : ''}${v.toFixed(1)}${suffix}`;
}

function fmtNum(v: number | undefined, digits = 2): string {
  if (v == null) return '—';
  return v.toFixed(digits);
}

export default function BacktestResultsCard({ metrics, status }: Props) {
  const { t } = useTranslation();

  if (status === 'running') {
    return (
      <div style={{ margin: '8px 10px', background: 'var(--ant-color-bg-base)', border: '1px solid var(--ant-color-border)', borderRadius: 8, padding: 12 }}>
        <div style={{ fontSize: 10, fontWeight: 700, color: 'var(--ant-color-text-tertiary)', textTransform: 'uppercase', marginBottom: 8 }}>
          {t('strategy.workspace.backtest')}
        </div>
        <div style={{ textAlign: 'center', padding: 12, color: 'var(--ant-color-text-secondary)', fontSize: 12 }}>
          {t('strategy.gen.backtesting')}
        </div>
      </div>
    );
  }

  if (!metrics) {
    return (
      <div style={{ margin: '8px 10px', background: 'var(--ant-color-bg-base)', border: '1px solid var(--ant-color-border)', borderRadius: 8, padding: 12 }}>
        <div style={{ fontSize: 10, fontWeight: 700, color: 'var(--ant-color-text-tertiary)', textTransform: 'uppercase', marginBottom: 8 }}>
          {t('strategy.workspace.backtest')}
        </div>
        <div style={{ textAlign: 'center', padding: 12, color: 'var(--ant-color-text-tertiary)', fontSize: 12 }}>
          {t('strategy.workspace.noResults')}
        </div>
      </div>
    );
  }

  const cells = [
    { label: t('strategy.gen.totalReturn'), value: fmtPct(metrics.totalReturn), positive: (metrics.totalReturn ?? 0) >= 0 },
    { label: t('strategy.gen.maxDrawdown'), value: fmtPct(metrics.maxDrawdown), positive: false },
    { label: t('strategy.gen.sharpe'), value: fmtNum(metrics.sharpeRatio), positive: (metrics.sharpeRatio ?? 0) >= 1 },
    { label: t('strategy.gen.winRate'), value: fmtPct(metrics.winRate), positive: undefined },
  ];

  return (
    <div style={{ margin: '8px 10px', background: 'var(--ant-color-bg-base)', border: '1px solid var(--ant-color-border)', borderRadius: 8, padding: 12 }}>
      <div style={{ fontSize: 10, fontWeight: 700, color: 'var(--ant-color-text-tertiary)', textTransform: 'uppercase', marginBottom: 8 }}>
        {t('strategy.workspace.backtestResultsLabel')}
      </div>
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 6 }}>
        {cells.map((c, i) => (
          <div key={i} style={{ background: 'var(--ant-color-bg-elevated)', borderRadius: 5, padding: 8, textAlign: 'center' }}>
            <div style={{ fontSize: 14, fontWeight: 700, color: c.positive === true ? '#3fb950' : c.positive === false ? '#f85149' : 'var(--ant-color-text)' }}>
              {c.value}
            </div>
            <div style={{ fontSize: 9, color: 'var(--ant-color-text-tertiary)', marginTop: 2 }}>{c.label}</div>
          </div>
        ))}
      </div>
      {metrics.equityCurve && metrics.equityCurve.length > 1 && (
        <EquityMiniLine points={metrics.equityCurve} />
      )}
    </div>
  );
}

function EquityMiniLine({ points }: { points: Array<{ time: number; equity: number }> }) {
  const values = points.map(p => p.equity);
  const min = Math.min(...values);
  const max = Math.max(...values);
  const range = max - min || 1;
  const barCount = 24;
  const step = Math.max(1, Math.floor(values.length / barCount));
  const bars: { h: number; up: boolean }[] = [];
  for (let i = 0; i < values.length; i += step) {
    const v = values[i];
    const h = ((v - min) / range) * 100;
    bars.push({ h: Math.max(4, h), up: v >= (values[i - step] ?? v) });
  }

  return (
    <div style={{ display: 'flex', alignItems: 'flex-end', gap: 1, height: 40, marginTop: 8 }}>
      {bars.map((b, i) => (
        <div key={i} style={{
          flex: 1, borderRadius: '1px 1px 0 0', minHeight: 2, height: `${b.h}%`,
          background: b.up ? '#3fb950' : '#f85149',
        }} />
      ))}
    </div>
  );
}
