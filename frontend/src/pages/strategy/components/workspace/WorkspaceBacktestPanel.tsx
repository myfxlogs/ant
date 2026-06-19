import { Button, Card, Row, Col, Statistic, Table, Tag, Empty, Spin } from 'antd';
import { RiseOutlined, FallOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next'
import { BACKTEST_COMPLETED_KEY, BACKTEST_EMPTY_KEY, BACKTEST_ERROR_KEY, BACKTEST_RUNNING_KEY, BACKTEST_TAB_KEY, EXEC_ASSUMPTIONS_FIELDS_COMMISSION_KEY, EXEC_ASSUMPTIONS_FIELDS_DIRECTION_KEY, EXEC_ASSUMPTIONS_FIELDS_FILL_RULE_KEY, EXEC_ASSUMPTIONS_FIELDS_LEVERAGE_KEY, EXEC_ASSUMPTIONS_FIELDS_MODE_KEY, EXEC_ASSUMPTIONS_FIELDS_MTF_FALLBACK_KEY, EXEC_ASSUMPTIONS_FIELDS_SLIPPAGE_KEY, EXEC_ASSUMPTIONS_FIELDS_TIMING_KEY, EXEC_ASSUMPTIONS_KEY, GATE_TAB_KEY, TUNING_TAB_KEY } from '@/gen/ant/v1/i18n/strategy_workspace_keys';
import { ANNUAL_RETURN_KEY, EQUITY_CURVE_KEY, MAX_DRAWDOWN_KEY, SHARPE_KEY, TOTAL_RETURN_KEY, TOTAL_TRADES_KEY, TRADE_LOG_KEY, TRADE_PRICE_KEY, TRADE_SIDE_KEY, TRADE_TIME_KEY, TRADE_VOLUME_KEY, WIN_RATE_KEY } from '@/gen/ant/v1/i18n/strategy_backtest_keys';

;
import { LineChart, Line, XAxis, YAxis, Tooltip, ResponsiveContainer } from 'recharts';
import SmartTuningPanel from './SmartTuningPanel';
import GatePanel from './GatePanel';
import type { SweepDimension } from '../../hooks/useBacktestParams';
import type { GateResult, GatePipelineSummary } from '@/gen/ant/v1/ai_gate_pb';

interface BacktestMetrics {
  totalReturn?: number; annualReturn?: number; maxDrawdown?: number;
  sharpeRatio?: number; winRate?: number; totalTrades?: number;
  equityCurve?: Array<{ time: number; equity: number }>;
  trades?: Array<{
    id: string; time: number; side: string; price: number; volume: number; pnl?: number;
  }>;
}

type BacktestStatus = 'idle' | 'running' | 'completed' | 'error';

interface Props {
  status: BacktestStatus;
  metrics: BacktestMetrics | null;
  executionAssumptions?: any;
  errorMessage?: string;
  onAIOptimize?: () => void;
  code?: string;
  onApplyTunedParams?: (code: string) => void;
  // Sub-tab
  subTab: 'results' | 'tuning' | 'gate'; onSubTabChange: (tab: string) => void;
  // Smart tuning
  tuneMethod: 'grid' | 'random'; onTuneMethodChange: (m: string) => void;
  sweepDimensions: SweepDimension[]; onToggleDimension: (key: string) => void;
  enabledSweepDims: SweepDimension[]; cartesianSize: number;
  tuningRunning: boolean; canRunTuning: boolean; onRunTuning: () => void;
  // Gate
  gateLoading: boolean; gateGates: GateResult[];
  gateSummary: GatePipelineSummary | null; gateError: string;
  onRunGate: () => void;
}

function pct(v: number | undefined): string {
  if (v == null) return '-';
  return (v * 100).toFixed(2) + '%';
}

function num(v: number | undefined, decimals = 2): string {
  if (v == null) return '-';
  return v.toFixed(decimals);
}

const metricStyle: React.CSSProperties = { fontSize: 16, fontFamily: 'monospace' };

const tabStyle: React.CSSProperties = {
  padding: '8px 16px', fontSize: 12, fontWeight: 600, cursor: 'pointer',
  color: '#8c8c8c', borderBottom: '2px solid transparent', transition: 'all 0.15s',
};

export default function WorkspaceBacktestPanel({
  status, metrics, executionAssumptions, errorMessage, onAIOptimize,
  subTab, onSubTabChange,
  tuneMethod, onTuneMethodChange,
  sweepDimensions, onToggleDimension,
  enabledSweepDims, cartesianSize,
  code, onApplyTunedParams,
  tuningRunning, canRunTuning, onRunTuning,
  gateLoading, gateGates, gateSummary, gateError, onRunGate,
}: Props) {
  const { t } = useTranslation();

  return (
    <div className="space-y-4">
      {/* Sub-tab navigation */}
      <div style={{ display: 'flex', gap: 0, borderBottom: '1px solid #e8e8e8', marginBottom: 12 }}>
        <div onClick={() => onSubTabChange('results')} style={{
          ...tabStyle,
          color: subTab === 'results' ? '#1890ff' : '#8c8c8c',
          borderBottomColor: subTab === 'results' ? '#1890ff' : 'transparent',
        }}>{t(BACKTEST_TAB_KEY)}</div>
        <div onClick={() => onSubTabChange('tuning')} style={{
          ...tabStyle,
          color: subTab === 'tuning' ? '#1890ff' : '#8c8c8c',
          borderBottomColor: subTab === 'tuning' ? '#1890ff' : 'transparent',
        }}>{t(TUNING_TAB_KEY)}</div>
        <div onClick={() => onSubTabChange('gate')} style={{
          ...tabStyle,
          color: subTab === 'gate' ? '#1890ff' : '#8c8c8c',
          borderBottomColor: subTab === 'gate' ? '#1890ff' : 'transparent',
        }}>{t(GATE_TAB_KEY, 'Gate')}</div>
      </div>

      {/* Results Tab */}
      {subTab === 'results' && (
        <>
          {status === 'idle' && (
            <Empty description={t(BACKTEST_EMPTY_KEY, 'Run a backtest to see results')}
              style={{ padding: 48 }} />
          )}

          {/* Status */}
          <div style={{ marginBottom: 12 }}>
            {status === 'running' && (
              <Tag color="processing" icon={<Spin size="small" />}>
                {t(BACKTEST_RUNNING_KEY)}{metrics?.processedBars != null ? ` — ${metrics.processedBars} bars processed` : '...'}
              </Tag>
            )}
            {status === 'completed' && (
              <Tag color="success">{t(BACKTEST_COMPLETED_KEY)}</Tag>
            )}
            {status === 'completed' && onAIOptimize && metrics && (
              <Button size="small" type="dashed" onClick={onAIOptimize}
                style={{ marginLeft: 8, fontSize: 11 }}>
                🤖 AI Optimize
              </Button>
            )}
            {status === 'error' && (
              <Tag color="error">{errorMessage || t(BACKTEST_ERROR_KEY, 'Backtest failed')}</Tag>
            )}
          </div>

          {/* Execution Assumptions — transparency panel */}
          {executionAssumptions && status === 'completed' && (
            <div style={{
              marginBottom: 12, padding: '8px 12px',
              border: '1px solid #e6f4ff', borderRadius: 8,
              background: 'linear-gradient(180deg, #f8fbff 0%, #f4f9ff 100%)',
            }}>
              <div style={{ fontSize: 10, fontWeight: 600, color: '#1677ff', marginBottom: 6 }}>
                {t(EXEC_ASSUMPTIONS_KEY)}
              </div>
              <div style={{ display: 'grid', gridTemplateColumns: 'repeat(2, minmax(0, 1fr))', gap: '4px 12px', fontSize: 11 }}>
                <div><span style={{ color: '#8c8c8c' }}>{t(EXEC_ASSUMPTIONS_FIELDS_MODE_KEY)}:</span> <strong>{executionAssumptions.simulationMode || '-'}</strong></div>
                <div><span style={{ color: '#8c8c8c' }}>{t(EXEC_ASSUMPTIONS_FIELDS_TIMING_KEY)}:</span> <strong>{executionAssumptions.signalTiming || '-'}</strong></div>
                <div><span style={{ color: '#8c8c8c' }}>{t(EXEC_ASSUMPTIONS_FIELDS_FILL_RULE_KEY)}:</span> <strong>{executionAssumptions.fillRule || '-'}</strong></div>
                <div><span style={{ color: '#8c8c8c' }}>{t(EXEC_ASSUMPTIONS_FIELDS_DIRECTION_KEY)}:</span> <strong>{executionAssumptions.tradeDirection || '-'}</strong></div>
                <div><span style={{ color: '#8c8c8c' }}>{t(EXEC_ASSUMPTIONS_FIELDS_COMMISSION_KEY)}:</span> <strong>{executionAssumptions.actualCommission != null ? (executionAssumptions.actualCommission * 100).toFixed(4) + '%' : '-'}</strong></div>
                <div><span style={{ color: '#8c8c8c' }}>{t(EXEC_ASSUMPTIONS_FIELDS_SLIPPAGE_KEY)}:</span> <strong>{executionAssumptions.actualSlippage != null ? (executionAssumptions.actualSlippage * 100).toFixed(4) + '%' : '-'}</strong></div>
                <div><span style={{ color: '#8c8c8c' }}>{t(EXEC_ASSUMPTIONS_FIELDS_LEVERAGE_KEY)}:</span> <strong>{executionAssumptions.actualLeverage || '-'}x</strong></div>
                {executionAssumptions.mtfFallbackReason && (
                  <div style={{ gridColumn: '1 / -1' }}><span style={{ color: '#fa8c16' }}>{t(EXEC_ASSUMPTIONS_FIELDS_MTF_FALLBACK_KEY)}:</span> <strong>{executionAssumptions.mtfFallbackReason}</strong></div>
                )}
              </div>
            </div>
          )}

          {metrics && (
            <>
              <Row gutter={[12, 12]}>
                <Col span={8}>
                  <Card size="small">
                    <Statistic title={t(TOTAL_RETURN_KEY, 'Total Return')}
                      value={pct(metrics.totalReturn)}
                      prefix={metrics.totalReturn != null && metrics.totalReturn >= 0
                        ? <RiseOutlined style={{ color: '#26a69a' }} />
                        : <FallOutlined style={{ color: '#ef5350' }} />}
                      valueStyle={metricStyle} />
                  </Card>
                </Col>
                <Col span={8}>
                  <Card size="small">
                    <Statistic title={t(ANNUAL_RETURN_KEY, 'Annual Return')}
                      value={pct(metrics.annualReturn)} valueStyle={metricStyle} />
                  </Card>
                </Col>
                <Col span={8}>
                  <Card size="small">
                    <Statistic title={t(MAX_DRAWDOWN_KEY, 'Max Drawdown')}
                      value={pct(metrics.maxDrawdown)}
                      valueStyle={{ ...metricStyle, color: '#ef5350' }} />
                  </Card>
                </Col>
                <Col span={8}>
                  <Card size="small">
                    <Statistic title={t(SHARPE_KEY, 'Sharpe')}
                      value={num(metrics.sharpeRatio)} valueStyle={metricStyle} />
                  </Card>
                </Col>
                <Col span={8}>
                  <Card size="small">
                    <Statistic title={t(WIN_RATE_KEY, 'Win Rate')}
                      value={pct(metrics.winRate)} valueStyle={metricStyle} />
                  </Card>
                </Col>
                <Col span={8}>
                  <Card size="small">
                    <Statistic title={t(TOTAL_TRADES_KEY, 'Total Trades')}
                      value={metrics.totalTrades ?? '-'} valueStyle={metricStyle} />
                  </Card>
                </Col>
              </Row>

              {metrics.equityCurve && metrics.equityCurve.length > 0 && (
                <Card size="small" title={t(EQUITY_CURVE_KEY, 'Equity Curve')} style={{ marginTop: 12 }}>
                  <ResponsiveContainer width="100%" height={200}>
                    <LineChart data={metrics.equityCurve}>
                      <XAxis dataKey="time" hide />
                      <YAxis width={60} tick={{ fontSize: 11 }} />
                      <Tooltip />
                      <Line type="monotone" dataKey="equity" stroke="#1890ff" dot={false} strokeWidth={1.5} />
                    </LineChart>
                  </ResponsiveContainer>
                </Card>
              )}

              {metrics.trades && metrics.trades.length > 0 && (
                <Card size="small" title={t(TRADE_LOG_KEY, 'Trade Log')} style={{ marginTop: 12 }}>
                  <Table dataSource={metrics.trades} rowKey="id" size="small"
                    pagination={{ pageSize: 20, size: 'small' }}
                    columns={[
                      { title: t(TRADE_TRADING_TIME_KEY, 'Time'), dataIndex: 'time',
                        render: (v: number) => v ? new Date(v * 1000).toLocaleString() : '-' },
                      { title: t(TRADE_TRADING_SIDE_KEY, 'Side'), dataIndex: 'side', width: 60 },
                      { title: t(TRADE_TRADING_PRICE_KEY, 'Price'), dataIndex: 'price', width: 80 },
                      { title: t(TRADE_TRADING_VOLUME_KEY, 'Volume'), dataIndex: 'volume', width: 80,
                        render: (v: number) => v?.toFixed(2) },
                      { title: 'PnL', dataIndex: 'pnl', width: 80,
                        render: (v: number) => v != null ? (
                          <span style={{ color: v >= 0 ? '#26a69a' : '#ef5350' }}>{v >= 0 ? '+' : ''}{v.toFixed(2)}</span>
                        ) : '-' },
                    ]} />
                </Card>
              )}
            </>
          )}
        </>
      )}

      {/* Tuning Tab */}
      {subTab === 'tuning' && (
        <SmartTuningPanel
          tuneMethod={tuneMethod} onTuneMethodChange={onTuneMethodChange}
          sweepDimensions={sweepDimensions} onToggleDimension={onToggleDimension}
          enabledSweepDims={enabledSweepDims} cartesianSize={cartesianSize}
          tuningRunning={tuningRunning} canRun={canRunTuning} onRunTuning={onRunTuning}
          code={code} onApplyToCode={onApplyTunedParams}
        />
      )}

      {/* Gate Tab */}
      {subTab === 'gate' && (
        <GatePanel
          loading={gateLoading} gates={gateGates} summary={gateSummary}
          error={gateError} status={status} canRun={status === 'completed'}
          onRun={onRunGate}
        />
      )}
    </div>
  );
}
