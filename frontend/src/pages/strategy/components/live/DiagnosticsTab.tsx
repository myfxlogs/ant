import { useMemo } from 'react';
import { Descriptions, Typography, Empty, Badge, Space, Tag, Tooltip } from 'antd';
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

function formatAge(ms: number | bigint): string {
  if (!ms || ms === 0n) return '-';
  const msNum = typeof ms === 'bigint' ? Number(ms) : ms;
  if (msNum < 0) return '-';
  if (msNum < 60_000) return `${Math.floor(msNum / 1000)}s`;
  if (msNum < 3_600_000) return `${Math.floor(msNum / 60_000)}m`;
  return `${Math.floor(msNum / 3_600_000)}h`;
}

// deriveState computes the diagnostic state from L1+L3 fields.
// LIVE-DIAG-TRUTH-1 rule 3: VM count != broker count, or positions stale,
// or outcome unknown → warning/error, not green active.
type DiagState = 'active' | 'starvation' | 'noeval' | 'error' | 'warning';

function deriveState(diag: StrategyDiagnostics | undefined, errorCount: number): DiagState {
  if (errorCount > 0) return 'error';
  if (!diag || diag.evalCount === 0n) return 'noeval';
  const msNum = typeof diag.lastEvalAt === 'bigint' ? Number(diag.lastEvalAt) : diag.lastEvalAt;
  if (msNum > 0) {
    const agoSec = Math.floor((Date.now() - msNum) / 1000);
    if (agoSec > 300) return 'starvation';
  }
  // L3: Check for warning conditions (rule 3)
  // Only check data-dependent warnings when data is available (not paper mode)
  if (diag.executionState === 'outcome_unknown') return 'warning';
  if (diag.dataAvailable) {
    if (!diag.positionsFresh) return 'warning';
    if (diag.vmOrdersTotal !== diag.brokerAccountOrders) return 'warning';
  }
  return 'active';
}

function StateBadge({ state, t }: { state: DiagState; t: (k: string, o?: Record<string, unknown>) => string }) {
  switch (state) {
    case 'active':
      return <Space><Badge status="success" /><Text>{t('strategy.live.diag.state.active')}</Text></Space>;
    case 'warning':
      return <Space><Badge status="warning" /><Text type="warning">{t('strategy.live.diag.state.warning')}</Text></Space>;
    case 'starvation':
      return <Space><Badge status="warning" /><Text>{t('strategy.live.diag.state.dataStarvation')}</Text></Space>;
    case 'noeval':
      return <Space><Badge status="default" /><Text type="secondary">{t('strategy.live.diag.state.noEvaluations')}</Text></Space>;
    case 'error':
      return <Space><Badge status="error" /><Text type="danger">{t('strategy.live.diag.state.error')}</Text></Space>;
  }
}

// lifecycleColor returns the tag color for an order lifecycle state.
// signal_generated is NOT green (it's not a fill — rule 1).
function lifecycleColor(lifecycle: string): string {
  switch (lifecycle) {
    case 'order_confirmed': return 'green';
    case 'order_rejected': return 'red';
    case 'order_outcome_unknown': return 'orange';
    case 'order_submitting':
    case 'order_submitted': return 'blue';
    case 'signal_generated': return 'default';
    default: return 'default';
  }
}

function executionStateColor(state: string): string {
  switch (state) {
    case 'idle': return 'default';
    case 'submitting': return 'blue';
    case 'accepted_unconfirmed': return 'blue';
    case 'confirmed': return 'green';
    case 'deterministic_rejected': return 'red';
    case 'outcome_unknown': return 'orange';
    default: return 'default';
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

  // L3: Freshness tags — N/A when data unavailable (paper mode), not Stale
  const dataAvail = diag?.dataAvailable ?? false;
  const finFreshTag = !dataAvail
    ? <Tag color="default">{t('strategy.live.diag.na')}</Tag>
    : diag!.financialFresh
      ? <Tag color="green">{t('strategy.live.diag.fresh')}</Tag>
      : <Tag color="red">{t('strategy.live.diag.stale')}</Tag>;
  const posFreshTag = !dataAvail
    ? <Tag color="default">{t('strategy.live.diag.na')}</Tag>
    : diag!.positionsFresh
      ? <Tag color="green">{t('strategy.live.diag.fresh')}</Tag>
      : <Tag color="red">{t('strategy.live.diag.stale')}</Tag>;

  // L3: VM vs broker mismatch warning (rule 3) — only meaningful when data available
  const vmBrokerMismatch = dataAvail && diag != null && diag.vmOrdersTotal !== diag.brokerAccountOrders;

  return (
    <div>
      <Descriptions size="small" column={3} bordered>
        {/* L1: Evaluation counters */}
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
        <Descriptions.Item label={t('strategy.live.diag.lastEval')}>
          {formatLastEval(diag?.lastEvalAt ?? 0n, t)}
        </Descriptions.Item>
        <Descriptions.Item label={t('strategy.live.status')} span={3}>
          <StateBadge state={state} t={t} />
        </Descriptions.Item>
      </Descriptions>

      {/* L3: Order Truth — VM vs broker vs magic (rule 7: mixed magic must show all three) */}
      <Descriptions size="small" column={3} bordered style={{ marginTop: 12 }} title={t('strategy.live.diag.orderTruth')}>
        <Descriptions.Item label={t('strategy.live.diag.vmOrdersTotal')}>
          <Tooltip title={vmBrokerMismatch ? t('strategy.live.diag.vmBrokerMismatch') : ''}>
            <Text color={vmBrokerMismatch ? 'warning' : undefined}>
              {diag?.vmOrdersTotal ?? 0}
            </Text>
          </Tooltip>
        </Descriptions.Item>
        <Descriptions.Item label={t('strategy.live.diag.brokerAccountOrders')}>
          {dataAvail ? (diag?.brokerAccountOrders ?? 0) : t('strategy.live.diag.na')}
        </Descriptions.Item>
        <Descriptions.Item label={t('strategy.live.diag.strategyMagicOrders')}>
          {dataAvail ? (diag?.strategyMagicOrders ?? 0) : t('strategy.live.diag.na')}
        </Descriptions.Item>
        <Descriptions.Item label={t('strategy.live.diag.pendingBrokerOrders')}>
          {dataAvail ? (diag?.pendingBrokerOrders ?? 0) : t('strategy.live.diag.na')}
        </Descriptions.Item>
        <Descriptions.Item label={t('strategy.live.diag.scheduleMagic')}>
          {diag?.scheduleMagic ?? 0}
        </Descriptions.Item>
        <Descriptions.Item label={t('strategy.live.diag.lastBrokerTicket')}>
          {diag?.lastBrokerTicket?.toString() ?? '0'}
        </Descriptions.Item>
      </Descriptions>

      {/* L3: Execution state + order lifecycle (rules 1, 2) */}
      <Descriptions size="small" column={2} bordered style={{ marginTop: 12 }} title={t('strategy.live.diag.execution')}>
        <Descriptions.Item label={t('strategy.live.diag.executionState')}>
          <Tag color={executionStateColor(diag?.executionState ?? 'idle')}>
            {t(`strategy.live.diag.execState.${diag?.executionState ?? 'idle'}`)}
          </Tag>
        </Descriptions.Item>
        <Descriptions.Item label={t('strategy.live.diag.orderLifecycle')}>
          <Tag color={lifecycleColor(diag?.orderLifecycle ?? 'signal_generated')}>
            {t(`strategy.live.diag.lifecycle.${diag?.orderLifecycle ?? 'signal_generated'}`)}
          </Tag>
        </Descriptions.Item>
      </Descriptions>

      {/* L3: Freshness (rules 3, 5: server-computed, frontend renders only) */}
      <Descriptions size="small" column={3} bordered style={{ marginTop: 12 }} title={t('strategy.live.diag.freshness')}>
        <Descriptions.Item label={t('strategy.live.diag.financialSource')}>
          {diag?.financialSource ? t(`strategy.live.diag.source.${diag.financialSource}`, { defaultValue: diag.financialSource }) : '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('strategy.live.diag.financialAge')}>
          {formatAge(diag?.financialAgeMs ?? 0n)}
        </Descriptions.Item>
        <Descriptions.Item label={t('strategy.live.diag.financialFresh')}>
          {finFreshTag}
        </Descriptions.Item>
        <Descriptions.Item label={t('strategy.live.diag.positionsSource')}>
          {diag?.positionsSource ? t(`strategy.live.diag.source.${diag.positionsSource}`, { defaultValue: diag.positionsSource }) : '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('strategy.live.diag.positionsAge')}>
          {formatAge(diag?.positionsAgeMs ?? 0n)}
        </Descriptions.Item>
        <Descriptions.Item label={t('strategy.live.diag.positionsFresh')}>
          {posFreshTag}
        </Descriptions.Item>
      </Descriptions>

      {/* L2: Indicators */}
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
