import { Button, Tag, Empty, Spin, Table, Skeleton, Progress, Typography, Tooltip } from 'antd';
import { StopOutlined, WarningOutlined, RobotOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import {
  EXEC_ASSUMPTIONS_KEY, EXEC_ASSUMPTIONS_FIELDS_COMMISSION_KEY,
  EXEC_ASSUMPTIONS_FIELDS_DIRECTION_KEY, EXEC_ASSUMPTIONS_FIELDS_FILL_RULE_KEY,
  EXEC_ASSUMPTIONS_FIELDS_LEVERAGE_KEY, EXEC_ASSUMPTIONS_FIELDS_MODE_KEY,
  EXEC_ASSUMPTIONS_FIELDS_MTF_FALLBACK_KEY, EXEC_ASSUMPTIONS_FIELDS_SLIPPAGE_KEY,
  EXEC_ASSUMPTIONS_FIELDS_TIMING_KEY,
} from '@/gen/ant/v1/i18n/strategy_workspace_keys';
import {
  TRADE_PRICE_KEY, TRADE_SIDE_KEY, TRADE_VOLUME_KEY,
} from '@/gen/ant/v1/i18n/strategy_backtest_keys';
import {
  CLOSE_PRICE_KEY, LONG_KEY, PNL_KEY, SHORT_KEY,
} from '@/gen/ant/v1/i18n/strategy_backtest_params_keys';
import type { BacktestStatus, BacktestMetrics, ChartTrade } from './useBacktestRunner';
import type { BacktestBlindSpotItem } from './backtestRunnerWatch';
import type { GateEvaluationUpdate, MarketplaceQualityPreview } from '@/gen/ant/v1/backtest_run_query_pb';
import type { GateResult } from '@/gen/ant/v1/ai_gate_pb';
import { GatePreview } from './BacktestSections';
import { DiagnosticPanel } from './DiagnosticPanel';
import BacktestMetricCards from './BacktestMetricCards';

const _ASSUMPTION_MAP: Record<string, string> = {
  MT_LIVE: 'strategy.backtest.assumptions.mtLive',
  MT_DATASET: 'strategy.backtest.assumptions.mtDataset',
  next_bar_open: 'strategy.backtest.assumptions.nextBarOpen',
  same_bar_close: 'strategy.backtest.assumptions.sameBarClose',
  market: 'strategy.backtestParams.market',
  limit: 'strategy.backtestParams.limit',
  both: 'strategy.backtestParams.both',
  long: 'strategy.backtestParams.long',
  short: 'strategy.backtestParams.short',
};
function assumeVal(t: unknown, v: string | undefined): string {
  if (!v) return '-';
  const key = _ASSUMPTION_MAP[v];
  return key ? t(key, v) : v;
}

const ASSUMPTION_TOOLTIPS: Record<string, string> = {
  simulationMode: 'How price data is simulated: KLINE_RANGE uses high-low range of each bar; MT_LIVE uses real MT tick data',
  signalTiming: 'When the strategy signal is evaluated relative to bar close: next_bar_open means signals fire on the next bar open after bar closes',
  fillRule: 'How orders are filled: bar_close fills at bar close price; market fills at current market price',
  tradeDirection: 'Which trade directions are allowed: long (buy only), short (sell only), or both',
  commission: 'Per-trade commission rate as a percentage of trade value',
  slippage: 'Simulated price deviation from expected fill price, as a percentage',
  leverage: 'Account leverage multiplier applied to position sizing',
  mtfFallback: 'Reason for multi-timeframe data fallback when primary timeframe data is unavailable',
};

function AssumptionField({ label, value, tooltip }: { label: string; value: React.ReactNode; tooltip: string }) {
  return (
    <Tooltip title={tooltip} placement="top">
      <div style={{ cursor: 'help' }}>
        <span style={{ color: '#8c8c8c', borderBottom: '1px dashed #d9d9d9' }}>{label}:</span> <strong>{value}</strong>
      </div>
    </Tooltip>
  );
}

interface Props {
  status: BacktestStatus;
  metrics: BacktestMetrics | null;
  executionAssumptions: unknown;
  errorMsg: string;
  onAIOptimize?: () => void;
  onOpenHistory?: () => void;
  trades: ChartTrade[];
  panelHeight: number;
  onCancel?: () => void;
  gateUpdate?: GateEvaluationUpdate | null;
  gateResults?: GateResult[];
  qualityPreview?: MarketplaceQualityPreview | null;
  blindSpots?: BacktestBlindSpotItem[];
  strategyId?: string;
  onAIFix?: (blindSpots: BacktestBlindSpotItem[]) => void;
  aiFixing?: boolean;
  coverageScore?: number;
  totalBlocks?: number;
  recognizedBlocks?: number;
  runMeta?: { symbol?: string; timeframe?: string; createdAt?: string; name?: string } | null;
}

export default function BacktestResultsTab({ status, metrics, executionAssumptions, errorMsg, onAIOptimize, trades, panelHeight, onCancel, gateUpdate, gateResults, qualityPreview, blindSpots, strategyId, onAIFix, aiFixing, coverageScore, totalBlocks, recognizedBlocks, runMeta }: Props) {
  const { t } = useTranslation();

  const buys = trades.filter((tr) => (tr.side || '').toLowerCase() === 'buy');
  const sells = trades.filter((tr) => (tr.side || '').toLowerCase() === 'sell');
  const buyPnl = buys.reduce((s, tr) => s + (tr.pnl || 0), 0);
  const sellPnl = sells.reduce((s, tr) => s + (tr.pnl || 0), 0);
  const buyVol = buys.reduce((s, tr) => s + (tr.volume || 0), 0);
  const sellVol = sells.reduce((s, tr) => s + (tr.volume || 0), 0);

  return (
    <div>
      {/* Header: status + meta + actions */}
      <div style={{
        display: 'flex', alignItems: 'center', gap: 16, marginBottom: 10,
        padding: '8px 14px', borderRadius: 6,
        background: 'linear-gradient(180deg, #f0f5ff 0%, #e6f0ff 100%)',
        border: '1px solid #d6e4ff',
      }}>
        {/* Status */}
        <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
          <span style={{ fontSize: 12, color: '#595959', fontWeight: 700 }}>{t('strategy.backtest.status', { defaultValue: 'Status' })}</span>
          {status === 'running' && <Tag color="processing" icon={<Spin size="small" />}>{t(BACKTEST_RUNNING_KEY)}</Tag>}
          {status === 'completed' && <Tag color="success" style={{ fontSize: 13, padding: '2px 8px' }}>{t(BACKTEST_COMPLETED_KEY)}</Tag>}
          {status === 'degraded' && <Tag color="warning" icon={<WarningOutlined />}>{t(BACKTEST_DEGRADED_KEY)}</Tag>}
          {status === 'error' && <Tag color="error">{errorMsg || t(BACKTEST_ERROR_KEY, 'Backtest failed')}</Tag>}
        </div>

        {/* Symbol */}
        {runMeta?.symbol && (
          <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
            <span style={{ fontSize: 12, color: '#595959', fontWeight: 700 }}>{t('strategy.backtest.symbol', { defaultValue: 'Symbol' })}</span>
            <Tag style={{ fontSize: 13, padding: '2px 8px' }}>{runMeta.symbol}</Tag>
          </div>
        )}

        {/* Timeframe */}
        {runMeta?.timeframe && (
          <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
            <span style={{ fontSize: 12, color: '#595959', fontWeight: 700 }}>{t('strategy.backtest.timeframe', { defaultValue: 'Period' })}</span>
            <Tag style={{ fontSize: 13, padding: '2px 8px' }}>{runMeta.timeframe}</Tag>
          </div>
        )}

        {/* Created at */}
        {runMeta?.createdAt && (
          <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
            <span style={{ fontSize: 12, color: '#595959', fontWeight: 700 }}>{t('strategy.backtest.createdAt', { defaultValue: 'Created' })}</span>
            <Typography.Text type="secondary" style={{ fontSize: 12 }}>{new Date(runMeta.createdAt).toLocaleString()}</Typography.Text>
          </div>
        )}

        <div style={{ flex: 1 }} />

        {/* Actions */}
        {status === 'completed' && onAIOptimize && metrics && (
          <Button type="primary" icon={<RobotOutlined />} onClick={onAIOptimize} size="small">
            {t('strategy.backtest.aiOptimize', { defaultValue: 'AI Optimize' })}
          </Button>
        )}
        {status === 'running' && onCancel && (
          <Button size="small" danger icon={<StopOutlined />} onClick={onCancel}>
            {t('common.cancel', { defaultValue: 'Cancel' })}
          </Button>
        )}
      </div>

      {status === 'running' && (
        <div style={{ padding: '12px 0' }}>
          <Progress strokeColor={{ from: '#108ee9', to: '#87d068' }} percent={100} status="active" showInfo={false} />
          <Skeleton active paragraph={{ rows: 4 }} style={{ marginTop: 12 }} />
        </div>
      )}

      {status === 'idle' && !metrics && (
        <Empty description={t(BACKTEST_EMPTY_KEY, 'Run a backtest to see results')} style={{ padding: 24 }} />
      )}

      {/* DEGRADED — diagnostic panel with severity grouping + AI fix + silence */}
      {status === 'degraded' && blindSpots && blindSpots.length > 0 && (
        <DiagnosticPanel
          blindSpots={blindSpots}
          strategyId={strategyId}
          onAIFix={onAIFix}
          aiFixing={aiFixing}
          coverageScore={coverageScore}
          totalBlocks={totalBlocks}
          recognizedBlocks={recognizedBlocks}
        />
      )}

      {/* Execution Assumptions */}
      {executionAssumptions && (status === 'completed' || status === 'degraded') && (
        <div style={{
          marginBottom: 12, padding: '8px 12px', border: '1px solid #e6f4ff', borderRadius: 8,
          background: 'linear-gradient(180deg, #f8fbff 0%, #f4f9ff 100%)',
        }}>
          <div style={{ fontSize: 12, fontWeight: 600, color: '#1677ff', marginBottom: 6 }}>{t(EXEC_ASSUMPTIONS_KEY)}</div>
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, minmax(0, 1fr))', gap: '6px 12px', fontSize: 12 }}>
            <AssumptionField label={t(EXEC_ASSUMPTIONS_FIELDS_MODE_KEY)} value={assumeVal(t, executionAssumptions.simulationMode)} tooltip={ASSUMPTION_TOOLTIPS.simulationMode} />
            <AssumptionField label={t(EXEC_ASSUMPTIONS_FIELDS_TIMING_KEY)} value={assumeVal(t, executionAssumptions.signalTiming)} tooltip={ASSUMPTION_TOOLTIPS.signalTiming} />
            <AssumptionField label={t(EXEC_ASSUMPTIONS_FIELDS_FILL_RULE_KEY)} value={assumeVal(t, executionAssumptions.fillRule)} tooltip={ASSUMPTION_TOOLTIPS.fillRule} />
            <AssumptionField label={t(EXEC_ASSUMPTIONS_FIELDS_DIRECTION_KEY)} value={assumeVal(t, executionAssumptions.tradeDirection)} tooltip={ASSUMPTION_TOOLTIPS.tradeDirection} />
            <AssumptionField label={t(EXEC_ASSUMPTIONS_FIELDS_COMMISSION_KEY)} value={executionAssumptions.actualCommission != null ? (executionAssumptions.actualCommission * 100).toFixed(4) + '%' : '-'} tooltip={ASSUMPTION_TOOLTIPS.commission} />
            <AssumptionField label={t(EXEC_ASSUMPTIONS_FIELDS_SLIPPAGE_KEY)} value={executionAssumptions.actualSlippage != null ? (executionAssumptions.actualSlippage * 100).toFixed(4) + '%' : '-'} tooltip={ASSUMPTION_TOOLTIPS.slippage} />
            <AssumptionField label={t(EXEC_ASSUMPTIONS_FIELDS_LEVERAGE_KEY)} value={(executionAssumptions.actualLeverage || '-') + 'x'} tooltip={ASSUMPTION_TOOLTIPS.leverage} />
            {executionAssumptions.mtfFallbackReason && (
              <div style={{ gridColumn: '1 / -1' }}><span style={{ color: '#fa8c16' }}>{t(EXEC_ASSUMPTIONS_FIELDS_MTF_FALLBACK_KEY)}:</span> <strong>{assumeVal(t, executionAssumptions.mtfFallbackReason)}</strong></div>
            )}
          </div>
        </div>
      )}

      {status === 'completed' && (gateUpdate || qualityPreview) && (
        <GatePreview gateUpdate={gateUpdate} gateResults={gateResults} qualityPreview={qualityPreview} />
      )}

      {metrics && (
        <BacktestMetricCards metrics={metrics} />
      )}

      {/* Trade detail table */}
      {trades.length > 0 && (
        <div style={{ marginTop: 12 }}>
          <div style={{ display: 'flex', gap: 24, marginBottom: 12, fontSize: 14, padding: '6px 12px', borderRadius: 6, background: '#fafafa', border: '1px solid #f0f0f0' }}>
            <span>🟢 {t(LONG_KEY)}: <b style={{ fontSize: 15 }}>{buys.length}</b> {t(TRADE_VOLUME_KEY)} <b style={{ fontSize: 15 }}>{buyVol.toFixed(2)}</b> {t(PNL_KEY)} <b style={{ color: buyPnl >= 0 ? '#26a69a' : '#e57373', fontSize: 15 }}>{buyPnl >= 0 ? '+' : ''}{buyPnl.toFixed(2)}</b></span>
            <span>🔴 {t(SHORT_KEY)}: <b style={{ fontSize: 15 }}>{sells.length}</b> {t(TRADE_VOLUME_KEY)} <b style={{ fontSize: 15 }}>{sellVol.toFixed(2)}</b> {t(PNL_KEY)} <b style={{ color: sellPnl >= 0 ? '#26a69a' : '#e57373', fontSize: 15 }}>{sellPnl >= 0 ? '+' : ''}{sellPnl.toFixed(2)}</b></span>
          </div>
          <Table dataSource={trades.map((tr, i) => ({ ...tr, key: i }))}
            pagination={{ pageSize: 30, size: 'small' }} scroll={{ y: panelHeight - 180, x: 'max-content' }}
            columns={[
              { title: 'Ticket', dataIndex: 'ticket', width: 80 },
              { title: t(TRADE_SIDE_KEY, 'Side'), dataIndex: 'side', width: 60,
                render: (v: string) => <Tag color={(v || '').toLowerCase() === 'buy' ? 'green' : 'red'}>{(v || '').toUpperCase()}</Tag> },
              { title: t(TRADE_VOLUME_KEY, 'Volume'), dataIndex: 'volume', width: 70,
                render: (v: number) => v?.toFixed(2) },
              { title: 'Open Time', dataIndex: 'openTime', width: 140,
                render: (v: number) => v ? new Date(v).toLocaleString() : '-' },
              { title: t(TRADE_PRICE_KEY, 'Open Price'), dataIndex: 'openPrice', width: 90,
                render: (v: number) => v?.toFixed(5) },
              { title: 'Close Time', dataIndex: 'closeTime', width: 140,
                render: (v: number) => v ? new Date(v).toLocaleString() : '-' },
              { title: t(CLOSE_PRICE_KEY), dataIndex: 'closePrice', width: 90,
                render: (v: number) => v?.toFixed(5) ?? '—' },
              { title: t(PNL_KEY), dataIndex: 'pnl', width: 80,
                render: (v: number) => v != null ? (
                  <span style={{ color: v >= 0 ? '#26a69a' : '#ef5350' }}>{v >= 0 ? '+' : ''}{v.toFixed(2)}</span>
                ) : '-' },
              { title: 'Commission', dataIndex: 'commission', width: 80,
                render: (v: number) => v != null ? v.toFixed(2) : '-' },
              { title: 'Reason', dataIndex: 'reason', width: 100,
                render: (v: string) => v || '-' },
            ]} />
        </div>
      )}
    </div>
  );
}
