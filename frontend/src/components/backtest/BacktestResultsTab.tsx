import { Button, Tag, Row, Col, Card, Statistic, Empty, Spin, Table, Skeleton, Progress, Typography } from 'antd';
import { RiseOutlined, FallOutlined, StopOutlined, WarningOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { LineChart, Line, XAxis, YAxis, Tooltip as RechartsTooltip, ResponsiveContainer } from 'recharts';
import {
  BACKTEST_COMPLETED_KEY, BACKTEST_EMPTY_KEY, BACKTEST_ERROR_KEY, BACKTEST_RUNNING_KEY,
  BACKTEST_DEGRADED_KEY,
  EXEC_ASSUMPTIONS_KEY, EXEC_ASSUMPTIONS_FIELDS_COMMISSION_KEY,
  EXEC_ASSUMPTIONS_FIELDS_DIRECTION_KEY, EXEC_ASSUMPTIONS_FIELDS_FILL_RULE_KEY,
  EXEC_ASSUMPTIONS_FIELDS_LEVERAGE_KEY, EXEC_ASSUMPTIONS_FIELDS_MODE_KEY,
  EXEC_ASSUMPTIONS_FIELDS_MTF_FALLBACK_KEY, EXEC_ASSUMPTIONS_FIELDS_SLIPPAGE_KEY,
  EXEC_ASSUMPTIONS_FIELDS_TIMING_KEY,
} from '@/gen/ant/v1/i18n/strategy_workspace_keys';
import {
  ANNUAL_RETURN_KEY, EQUITY_CURVE_KEY, MAX_DRAWDOWN_KEY, SHARPE_KEY,
  TOTAL_RETURN_KEY, TOTAL_TRADES_KEY, WIN_RATE_KEY,
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

function pct(v: number | undefined): string { if (v == null) return '-'; return (v * 100).toFixed(2) + '%'; }
function num(v: number | undefined, d = 2): string { if (v == null) return '-'; return v.toFixed(d); }

const S = { metricStyle: { fontSize: 14, fontFamily: 'monospace' as const } };

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

export default function BacktestResultsTab({ status, metrics, executionAssumptions, errorMsg, onAIOptimize, onOpenHistory, trades, panelHeight, onCancel, gateUpdate, gateResults, qualityPreview, blindSpots, strategyId, onAIFix, aiFixing, coverageScore, totalBlocks, recognizedBlocks, runMeta }: Props) {
  const { t } = useTranslation();

  const buys = trades.filter((tr) => tr.side === 'buy');
  const sells = trades.filter((tr) => tr.side === 'sell');
  const buyPnl = buys.reduce((s, tr) => s + (tr.pnl || 0), 0);
  const sellPnl = sells.reduce((s, tr) => s + (tr.pnl || 0), 0);
  const buyVol = buys.reduce((s, tr) => s + (tr.volume || 0), 0);
  const sellVol = sells.reduce((s, tr) => s + (tr.volume || 0), 0);

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8, flexWrap: 'wrap' }}>
          {status === 'running' && (
            <Tag color="processing" icon={<Spin size="small" />}>{t(BACKTEST_RUNNING_KEY)}</Tag>
          )}
          {status === 'completed' && (
            <Tag color="success">{t(BACKTEST_COMPLETED_KEY)}</Tag>
          )}
          {status === 'degraded' && (
            <Tag color="warning" icon={<WarningOutlined />}>{t(BACKTEST_DEGRADED_KEY)}</Tag>
          )}
          {runMeta?.symbol && <Tag>{runMeta.symbol}</Tag>}
          {runMeta?.timeframe && <Tag>{runMeta.timeframe}</Tag>}
          {runMeta?.createdAt && <Typography.Text type="secondary" style={{ fontSize: 11 }}>{new Date(runMeta.createdAt).toLocaleString()}</Typography.Text>}
          {status === 'completed' && onAIOptimize && metrics && (
            <Button size="small" type="dashed" onClick={onAIOptimize} style={{ fontSize: 11 }}>
              🤖 AI Optimize
            </Button>
          )}
          {status === 'error' && (
            <Tag color="error">{errorMsg || t(BACKTEST_ERROR_KEY, 'Backtest failed')}</Tag>
          )}
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
          {status === 'running' && onCancel && (
            <Button size="small" danger icon={<StopOutlined />} onClick={onCancel}>
              {t('common.cancel', { defaultValue: 'Cancel' })}
            </Button>
          )}
        </div>
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
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(2, minmax(0, 1fr))', gap: '4px 12px', fontSize: 12 }}>
            <div><span style={{ color: '#8c8c8c' }}>{t(EXEC_ASSUMPTIONS_FIELDS_MODE_KEY)}:</span> <strong>{assumeVal(t, executionAssumptions.simulationMode)}</strong></div>
            <div><span style={{ color: '#8c8c8c' }}>{t(EXEC_ASSUMPTIONS_FIELDS_TIMING_KEY)}:</span> <strong>{assumeVal(t, executionAssumptions.signalTiming)}</strong></div>
            <div><span style={{ color: '#8c8c8c' }}>{t(EXEC_ASSUMPTIONS_FIELDS_FILL_RULE_KEY)}:</span> <strong>{assumeVal(t, executionAssumptions.fillRule)}</strong></div>
            <div><span style={{ color: '#8c8c8c' }}>{t(EXEC_ASSUMPTIONS_FIELDS_DIRECTION_KEY)}:</span> <strong>{assumeVal(t, executionAssumptions.tradeDirection)}</strong></div>
            <div><span style={{ color: '#8c8c8c' }}>{t(EXEC_ASSUMPTIONS_FIELDS_COMMISSION_KEY)}:</span> <strong>{executionAssumptions.actualCommission != null ? (executionAssumptions.actualCommission * 100).toFixed(4) + '%' : '-'}</strong></div>
            <div><span style={{ color: '#8c8c8c' }}>{t(EXEC_ASSUMPTIONS_FIELDS_SLIPPAGE_KEY)}:</span> <strong>{executionAssumptions.actualSlippage != null ? (executionAssumptions.actualSlippage * 100).toFixed(4) + '%' : '-'}</strong></div>
            <div><span style={{ color: '#8c8c8c' }}>{t(EXEC_ASSUMPTIONS_FIELDS_LEVERAGE_KEY)}:</span> <strong>{executionAssumptions.actualLeverage || '-'}x</strong></div>
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
        <>
          <Row gutter={[12, 12]}>
            <Col span={8}>
              <Card size="small">
                <Statistic title={t(TOTAL_RETURN_KEY, 'Total Return')} value={pct(metrics.totalReturn)}
                  prefix={metrics.totalReturn != null && metrics.totalReturn >= 0
                    ? <RiseOutlined style={{ color: '#26a69a' }} /> : <FallOutlined style={{ color: '#ef5350' }} />}
                  valueStyle={S.metricStyle} />
              </Card>
            </Col>
            <Col span={8}>
              <Card size="small">
                <Statistic title={t(ANNUAL_RETURN_KEY, 'Annual Return')} value={pct(metrics.annualReturn)} valueStyle={S.metricStyle} />
              </Card>
            </Col>
            <Col span={8}>
              <Card size="small">
                <Statistic title={t(MAX_DRAWDOWN_KEY, 'Max Drawdown')} value={pct(metrics.maxDrawdown)} valueStyle={{ ...S.metricStyle, color: '#ef5350' }} />
              </Card>
            </Col>
            <Col span={8}>
              <Card size="small">
                <Statistic title={t(SHARPE_KEY, 'Sharpe')} value={num(metrics.sharpeRatio)} valueStyle={S.metricStyle} />
              </Card>
            </Col>
            <Col span={8}>
              <Card size="small">
                <Statistic title={t(WIN_RATE_KEY, 'Win Rate')} value={pct(metrics.winRate)} valueStyle={S.metricStyle} />
              </Card>
            </Col>
            <Col span={8}>
              <Card size="small">
                <Statistic title={t(TOTAL_TRADES_KEY, 'Total Trades')} value={metrics.totalTrades ?? '-'} valueStyle={S.metricStyle} />
              </Card>
            </Col>
          </Row>

          {metrics.equityCurve && metrics.equityCurve.length > 0 && (
            <Card size="small" title={t(EQUITY_CURVE_KEY, 'Equity Curve')} style={{ marginTop: 12 }}>
              <ResponsiveContainer width="100%" height={150}>
                <LineChart data={metrics.equityCurve}>
                  <XAxis dataKey="time" hide />
                  <YAxis width={60} tick={{ fontSize: 11 }} />
                  <RechartsTooltip />
                  <Line type="monotone" dataKey="equity" stroke="#1890ff" dot={false} strokeWidth={1.5} />
                </LineChart>
              </ResponsiveContainer>
            </Card>
          )}
        </>
      )}

      {/* Trade detail table */}
      {trades.length > 0 && (
        <div style={{ marginTop: 12 }}>
          <div style={{ display: 'flex', gap: 12, marginBottom: 10, fontSize: 12 }}>
            <span>🟢 {t(LONG_KEY)}: <b>{buys.length}</b> {t(TRADE_VOLUME_KEY)} <b>{buyVol.toFixed(2)}</b> {t(PNL_KEY)} <b style={{ color: buyPnl >= 0 ? '#26a69a' : '#e57373' }}>{buyPnl >= 0 ? '+' : ''}{buyPnl.toFixed(2)}</b></span>
            <span>🔴 {t(SHORT_KEY)}: <b>{sells.length}</b> {t(TRADE_VOLUME_KEY)} <b>{sellVol.toFixed(2)}</b> {t(PNL_KEY)} <b style={{ color: sellPnl >= 0 ? '#26a69a' : '#e57373' }}>{sellPnl >= 0 ? '+' : ''}{sellPnl.toFixed(2)}</b></span>
          </div>
          <Table dataSource={trades.map((tr, i) => ({ ...tr, key: i }))}
            pagination={{ pageSize: 30, size: 'small' }} scroll={{ y: panelHeight - 180, x: 'max-content' }}
            columns={[
              { title: '#', dataIndex: 'key', width: 40 },
              { title: 'Ticket', dataIndex: 'ticket', width: 70 },
              { title: t(TRADE_SIDE_KEY, 'Side'), dataIndex: 'side', width: 60,
                render: (v: string) => <Tag color={v === 'buy' ? 'green' : 'red'}>{v?.toUpperCase()}</Tag> },
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
