import { Card, Row, Col, Statistic, Table, Tag, Empty, Spin } from 'antd';
import { RiseOutlined, FallOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { LineChart, Line, XAxis, YAxis, Tooltip, ResponsiveContainer } from 'recharts';

interface BacktestMetrics {
  totalReturn?: number;
  annualReturn?: number;
  maxDrawdown?: number;
  sharpeRatio?: number;
  winRate?: number;
  totalTrades?: number;
  equityCurve?: Array<{ time: number; equity: number }>;
  trades?: Array<{
    id: string; time: number; side: string; price: number;
    volume: number; pnl?: number;
  }>;
}

type BacktestStatus = 'idle' | 'running' | 'completed' | 'error';

interface Props {
  status: BacktestStatus;
  metrics: BacktestMetrics | null;
  errorMessage?: string;
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

export default function WorkspaceBacktestPanel({ status, metrics, errorMessage }: Props) {
  const { t } = useTranslation();

  if (status === 'idle') {
    return (
      <Empty
        description={t('strategy.workspace.backtestEmpty', 'Run a backtest to see results')}
        style={{ padding: 48 }}
      />
    );
  }

  return (
    <div className="space-y-4" style={{ padding: '0 4px' }}>
      {/* Status */}
      <div style={{ marginBottom: 12 }}>
        {status === 'running' && (
          <Tag color="processing" icon={<Spin size="small" />}>
            {t('strategy.workspace.backtestRunning', 'Backtest running...')}
          </Tag>
        )}
        {status === 'completed' && (
          <Tag color="success">{t('strategy.workspace.backtestCompleted', 'Completed')}</Tag>
        )}
        {status === 'error' && (
          <Tag color="error">
            {errorMessage || t('strategy.workspace.backtestError', 'Backtest failed')}
          </Tag>
        )}
      </div>

      {/* Metrics */}
      {metrics && (
        <>
          <Row gutter={[12, 12]}>
            <Col span={8}>
              <Card size="small">
                <Statistic
                  title={t('strategy.backtest.totalReturn', 'Total Return')}
                  value={pct(metrics.totalReturn)}
                  prefix={metrics.totalReturn != null && metrics.totalReturn >= 0
                    ? <RiseOutlined style={{ color: '#26a69a' }} />
                    : <FallOutlined style={{ color: '#ef5350' }} />}
                  valueStyle={metricStyle}
                />
              </Card>
            </Col>
            <Col span={8}>
              <Card size="small">
                <Statistic
                  title={t('strategy.backtest.annualReturn', 'Annual Return')}
                  value={pct(metrics.annualReturn)}
                  valueStyle={metricStyle}
                />
              </Card>
            </Col>
            <Col span={8}>
              <Card size="small">
                <Statistic
                  title={t('strategy.backtest.maxDrawdown', 'Max Drawdown')}
                  value={pct(metrics.maxDrawdown)}
                  valueStyle={{ ...metricStyle, color: '#ef5350' }}
                />
              </Card>
            </Col>
            <Col span={8}>
              <Card size="small">
                <Statistic
                  title={t('strategy.backtest.sharpe', 'Sharpe')}
                  value={num(metrics.sharpeRatio)}
                  valueStyle={metricStyle}
                />
              </Card>
            </Col>
            <Col span={8}>
              <Card size="small">
                <Statistic
                  title={t('strategy.backtest.winRate', 'Win Rate')}
                  value={pct(metrics.winRate)}
                  valueStyle={metricStyle}
                />
              </Card>
            </Col>
            <Col span={8}>
              <Card size="small">
                <Statistic
                  title={t('strategy.backtest.totalTrades', 'Total Trades')}
                  value={metrics.totalTrades ?? '-'}
                  valueStyle={metricStyle}
                />
              </Card>
            </Col>
          </Row>

          {/* Equity curve */}
          {metrics.equityCurve && metrics.equityCurve.length > 0 && (
            <Card size="small" title={t('strategy.backtest.equityCurve', 'Equity Curve')} style={{ marginTop: 12 }}>
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

          {/* Trade log */}
          {metrics.trades && metrics.trades.length > 0 && (
            <Card size="small" title={t('strategy.backtest.tradeLog', 'Trade Log')} style={{ marginTop: 12 }}>
              <Table
                dataSource={metrics.trades}
                rowKey="id"
                size="small"
                pagination={{ pageSize: 20, size: 'small' }}
                columns={[
                  {
                    title: t('strategy.backtest.tradeTime', 'Time'),
                    dataIndex: 'time',
                    render: (v: number) => v ? new Date(v * 1000).toLocaleString() : '-',
                  },
                  {
                    title: t('strategy.backtest.tradeSide', 'Side'),
                    dataIndex: 'side',
                    width: 60,
                  },
                  {
                    title: t('strategy.backtest.tradePrice', 'Price'),
                    dataIndex: 'price',
                    width: 80,
                  },
                  {
                    title: t('strategy.backtest.tradeVolume', 'Volume'),
                    dataIndex: 'volume',
                    width: 80,
                    render: (v: number) => v?.toFixed(2),
                  },
                  {
                    title: 'PnL',
                    dataIndex: 'pnl',
                    width: 80,
                    render: (v: number) => v != null ? (
                      <span style={{ color: v >= 0 ? '#26a69a' : '#ef5350' }}>
                        {v >= 0 ? '+' : ''}{v.toFixed(2)}
                      </span>
                    ) : '-',
                  },
                ]}
              />
            </Card>
          )}
        </>
      )}
    </div>
  );
}
