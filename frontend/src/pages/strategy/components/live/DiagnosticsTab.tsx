import { useMemo } from 'react';
import { Descriptions, Typography, Empty, Badge, Space } from 'antd';
import { useTranslation } from 'react-i18next';
import type { ActiveStrategy, StrategyDiagnostics, DiagIndicatorSeries } from '@/gen/ant/v1/strategy_runtime_pb';

const { Text } = Typography;

interface Props {
  active?: ActiveStrategy;
}

function formatLastEval(ms: number | bigint, t: (k: string, o?: Record<string, unknown>) => string): string {
  if (!ms || ms === 0n) return t('strategy.live.diag.never');
  const msNum = typeof ms === 'bigint' ? Number(ms) : ms;
  const agoSec = Math.floor((Date.now() - msNum) / 1000);
  if (agoSec < 0) return t('strategy.live.diag.never');
  if (agoSec < 60) return `${agoSec}s ${t('strategy.live.diag.ago')}`;
  if (agoSec < 3600) return `${Math.floor(agoSec / 60)}m ${t('strategy.live.diag.ago')}`;
  if (agoSec < 86400) return `${Math.floor(agoSec / 3600)}h ${t('strategy.live.diag.ago')}`;
  return `${Math.floor(agoSec / 86400)}d ${t('strategy.live.diag.ago')}`;
}

function deriveState(diag: StrategyDiagnostics | undefined, errorCount: number): 'active' | 'starvation' | 'noeval' | 'error' {
  if (errorCount > 0) return 'error';
  if (!diag || diag.evalCount === 0n) return 'noeval';
  const msNum = typeof diag.lastEvalAt === 'bigint' ? Number(diag.lastEvalAt) : diag.lastEvalAt;
  if (msNum > 0) {
    const agoSec = Math.floor((Date.now() - msNum) / 1000);
    if (agoSec > 300) return 'starvation';
  }
  return 'active';
}

function StateBadge({ state, t }: { state: ReturnType<typeof deriveState>; t: (k: string, o?: Record<string, unknown>) => string }) {
  switch (state) {
    case 'active':
      return <Space><Badge status="success" /><Text>{t('strategy.live.diag.state.active')}</Text></Space>;
    case 'starvation':
      return <Space><Badge status="warning" /><Text>{t('strategy.live.diag.state.dataStarvation')}</Text></Space>;
    case 'noeval':
      return <Space><Badge status="default" /><Text type="secondary">{t('strategy.live.diag.state.noEvaluations')}</Text></Space>;
    case 'error':
      return <Space><Badge status="error" /><Text type="danger">{t('strategy.live.diag.state.error')}</Text></Space>;
  }
}

function Sparkline({ values, color = '#1677ff' }: { values: string[]; color?: string }) {
  const nums = values.map(Number).filter(n => !isNaN(n));
  if (nums.length < 2) return <Text type="secondary" style={{ fontSize: 11 }}>{nums.length === 1 ? nums[0].toFixed(4) : '-'}</Text>;
  const min = Math.min(...nums);
  const max = Math.max(...nums);
  const range = max - min || 1;
  const w = 120;
  const h = 28;
  const points = nums.map((v, i) => {
    const x = (i / (nums.length - 1)) * w;
    const y = h - ((v - min) / range) * h;
    return `${x.toFixed(1)},${y.toFixed(1)}`;
  }).join(' ');
  return (
    <svg width={w} height={h} style={{ display: 'block', verticalAlign: 'middle' }}>
      <polyline points={points} fill="none" stroke={color} strokeWidth={1.5} />
    </svg>
  );
}

function IndicatorRow({ series }: { series: DiagIndicatorSeries }) {
  const latest = series.values.length > 0 ? series.values[series.values.length - 1] : '-';
  return (
    <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '4px 0', borderBottom: '1px solid var(--ant-color-border-secondary, #f0f0f0)' }}>
      <Space direction="vertical" size={0}>
        <Text style={{ fontSize: 12 }} strong>{series.key}</Text>
        <Text type="secondary" style={{ fontSize: 11 }}>{latest}</Text>
      </Space>
      <Sparkline values={series.values} />
    </div>
  );
}

export default function DiagnosticsTab({ active }: Props) {
  const { t } = useTranslation();
  const diag = active?.diagnostics;
  const state = useMemo(() => deriveState(diag, active?.errorCount ?? 0), [diag, active?.errorCount]);

  if (!active) {
    return <Empty description={t('strategy.live.noActive')} />;
  }

  return (
    <div>
      <Descriptions size="small" column={3} bordered>
        <Descriptions.Item label={t('strategy.live.diag.evalCount')}>
          <Text strong>{diag?.evalCount?.toString() ?? '0'}</Text>
        </Descriptions.Item>
        <Descriptions.Item label={t('strategy.live.diag.barCount')}>
          {diag?.barCount?.toString() ?? '0'}
        </Descriptions.Item>
        <Descriptions.Item label={t('strategy.live.diag.tickCount')}>
          {diag?.tickCount?.toString() ?? '0'}
        </Descriptions.Item>
        <Descriptions.Item label={t('strategy.live.diag.windowBars')}>
          {diag?.windowBars ?? 0}
        </Descriptions.Item>
        <Descriptions.Item label={t('strategy.live.diag.ordersTotal')}>
          {diag?.ordersTotal ?? 0}
        </Descriptions.Item>
        <Descriptions.Item label={t('strategy.live.diag.lastEval')}>
          {formatLastEval(diag?.lastEvalAt ?? 0n, t)}
        </Descriptions.Item>
        <Descriptions.Item label={t('strategy.live.status', { defaultValue: 'Status' })} span={3}>
          <StateBadge state={state} t={t} />
        </Descriptions.Item>
      </Descriptions>

      <div style={{ marginTop: 12 }}>
        <Text type="secondary" style={{ fontSize: 12 }}>{t('strategy.live.diag.indicators')}</Text>
        {diag?.indicators && diag.indicators.length > 0 ? (
          <div style={{ marginTop: 4 }}>
            {diag.indicators.map(s => <IndicatorRow key={s.key} series={s} />)}
          </div>
        ) : (
          <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t('strategy.live.diag.noIndicators')} style={{ margin: '8px 0' }} />
        )}
      </div>
    </div>
  );
}
