import { Row, Col, Card, Statistic } from 'antd';
import { RiseOutlined, FallOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { LineChart, Line, XAxis, YAxis, Tooltip as RechartsTooltip, ResponsiveContainer } from 'recharts';
import {
  ANNUAL_RETURN_KEY, EQUITY_CURVE_KEY, MAX_DRAWDOWN_KEY, SHARPE_KEY,
  TOTAL_RETURN_KEY, TOTAL_TRADES_KEY, WIN_RATE_KEY,
} from '@/gen/ant/v1/i18n/strategy_backtest_keys';
import type { BacktestMetrics } from './useBacktestRunner';

const metricStyle = { fontSize: 14, fontFamily: 'monospace' as const };

function pct(v: number | undefined): string { if (v == null) return '-'; return (v * 100).toFixed(2) + '%'; }
function num(v: number | undefined, d = 2): string { if (v == null) return '-'; return v.toFixed(d); }

export default function BacktestMetricCards({ metrics }: { metrics: BacktestMetrics }) {
  const { t } = useTranslation();
  return (
    <>
      <Row gutter={[12, 12]}>
        <Col span={8}>
          <Card size="small">
            <Statistic title={t(TOTAL_RETURN_KEY, 'Total Return')} value={pct(metrics.totalReturn)}
              prefix={metrics.totalReturn != null && metrics.totalReturn >= 0
                ? <RiseOutlined style={{ color: 'var(--color-success)' }} /> : <FallOutlined style={{ color: 'var(--color-danger)' }} />}
              valueStyle={metricStyle} />
          </Card>
        </Col>
        <Col span={8}>
          <Card size="small">
            <Statistic title={t(ANNUAL_RETURN_KEY, 'Annual Return')} value={pct(metrics.annualReturn)} valueStyle={metricStyle} />
          </Card>
        </Col>
        <Col span={8}>
          <Card size="small">
            <Statistic title={t(MAX_DRAWDOWN_KEY, 'Max Drawdown')} value={pct(metrics.maxDrawdown)} valueStyle={{ ...metricStyle, color: 'var(--color-danger)' }} />
          </Card>
        </Col>
        <Col span={8}>
          <Card size="small">
            <Statistic title={t(SHARPE_KEY, 'Sharpe')} value={num(metrics.sharpeRatio)} valueStyle={metricStyle} />
          </Card>
        </Col>
        <Col span={8}>
          <Card size="small">
            <Statistic title={t(WIN_RATE_KEY, 'Win Rate')} value={pct(metrics.winRate)} valueStyle={metricStyle} />
          </Card>
        </Col>
        <Col span={8}>
          <Card size="small">
            <Statistic title={t(TOTAL_TRADES_KEY, 'Total Trades')} value={metrics.totalTrades ?? '-'} valueStyle={metricStyle} />
          </Card>
        </Col>
      </Row>

      {(metrics as BacktestMetrics & { equityCurve?: { time: string; equity: number }[] }).equityCurve?.length ? (
        <Card size="small" title={t(EQUITY_CURVE_KEY, 'Equity Curve')} style={{ marginTop: 12 }}>
          <ResponsiveContainer width="100%" height={150}>
            <LineChart data={(metrics as BacktestMetrics & { equityCurve?: { time: string; equity: number }[] }).equityCurve!}>
              <XAxis dataKey="time" hide />
              <YAxis width={60} tick={{ fontSize: 11 }} />
              <RechartsTooltip />
              <Line type="monotone" dataKey="equity" stroke="#1890ff" dot={false} strokeWidth={1.5} />
            </LineChart>
          </ResponsiveContainer>
        </Card>
      ) : null}
    </>
  );
}
