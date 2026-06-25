import { Button, Tag, Row, Col, Card, Statistic, Empty, Spin } from 'antd';
import { RiseOutlined, FallOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { LineChart, Line, XAxis, YAxis, Tooltip, ResponsiveContainer } from 'recharts';
import {
  BACKTEST_COMPLETED_KEY, BACKTEST_EMPTY_KEY, BACKTEST_ERROR_KEY, BACKTEST_RUNNING_KEY,
  EXEC_ASSUMPTIONS_KEY, EXEC_ASSUMPTIONS_FIELDS_COMMISSION_KEY,
  EXEC_ASSUMPTIONS_FIELDS_DIRECTION_KEY, EXEC_ASSUMPTIONS_FIELDS_FILL_RULE_KEY,
  EXEC_ASSUMPTIONS_FIELDS_LEVERAGE_KEY, EXEC_ASSUMPTIONS_FIELDS_MODE_KEY,
  EXEC_ASSUMPTIONS_FIELDS_MTF_FALLBACK_KEY, EXEC_ASSUMPTIONS_FIELDS_SLIPPAGE_KEY,
  EXEC_ASSUMPTIONS_FIELDS_TIMING_KEY,
} from '@/gen/ant/v1/i18n/strategy_workspace_keys';
import {
  ANNUAL_RETURN_KEY, EQUITY_CURVE_KEY, MAX_DRAWDOWN_KEY, SHARPE_KEY,
  TOTAL_RETURN_KEY, TOTAL_TRADES_KEY, WIN_RATE_KEY,
} from '@/gen/ant/v1/i18n/strategy_backtest_keys';
import type { BacktestStatus, BacktestMetrics } from './useBacktestRunner';

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
function assumeVal(t: any, v: string | undefined): string {
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
  executionAssumptions: any;
  errorMsg: string;
  onAIOptimize?: () => void;
}

export default function BacktestResultsTab({ status, metrics, executionAssumptions, errorMsg, onAIOptimize }: Props) {
  const { t } = useTranslation();

  return (
    <div>
      {status === 'idle' && (
        <Empty description={t(BACKTEST_EMPTY_KEY, 'Run a backtest to see results')} style={{ padding: 24 }} />
      )}

      <div style={{ marginBottom: 8 }}>
        {status === 'running' && (
          <Tag color="processing" icon={<Spin size="small" />}>{t(BACKTEST_RUNNING_KEY)}</Tag>
        )}
        {status === 'completed' && (
          <Tag color="success">{t(BACKTEST_COMPLETED_KEY)}</Tag>
        )}
        {status === 'completed' && onAIOptimize && metrics && (
          <Button size="small" type="dashed" onClick={onAIOptimize} style={{ marginLeft: 8, fontSize: 11 }}>
            🤖 AI Optimize
          </Button>
        )}
        {status === 'error' && (
          <Tag color="error">{errorMsg || t(BACKTEST_ERROR_KEY, 'Backtest failed')}</Tag>
        )}
      </div>

      {/* Execution Assumptions */}
      {executionAssumptions && status === 'completed' && (
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
                  <Tooltip />
                  <Line type="monotone" dataKey="equity" stroke="#1890ff" dot={false} strokeWidth={1.5} />
                </LineChart>
              </ResponsiveContainer>
            </Card>
          )}
        </>
      )}
    </div>
  );
}
