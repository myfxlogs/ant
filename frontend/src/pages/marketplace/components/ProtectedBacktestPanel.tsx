import { useState, useCallback, useEffect, useRef } from 'react';
import { Button, InputNumber, DatePicker, Select, Card, Row, Col, Statistic, Tag, Empty, Typography } from 'antd';
import { RiseOutlined, FallOutlined, LockOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import dayjs from 'dayjs';
import { LineChart, Line, XAxis, YAxis, Tooltip, ResponsiveContainer } from 'recharts';
import { marketplaceClient } from '@/client/connect';
import { create } from '@bufbuild/protobuf';
import { RunMarketBacktestRequestSchema } from '@/gen/ant/v1/marketplace_service_pb';
import { TradeDirection } from '@/gen/ant/v1/backtest_execution_config_pb';
import { BacktestRunStatus } from '@/gen/ant/v1/backtest_run_pb';

const { Text } = Typography;

interface Props {
  strategyId: string;
  defaultSymbol?: string;
  defaultTimeframe?: string;
}

interface Metrics {
  totalReturn?: number;
  annualReturn?: number;
  maxDrawdown?: number;
  sharpeRatio?: number;
  winRate?: number;
  totalTrades?: number;
  equityCurve?: Array<{ time: number; equity: number }>;
}

function pct(v: number | undefined): string {
  if (v == null) return '-';
  return (v * 100).toFixed(2) + '%';
}

export default function ProtectedBacktestPanel({ strategyId, defaultSymbol = 'EURUSD', defaultTimeframe = '1h' }: Props) {
  const { t } = useTranslation();
  const [symbol, setSymbol] = useState(defaultSymbol);
  const [timeframe, setTimeframe] = useState(defaultTimeframe);
  const [initialCapital, setInitialCapital] = useState(10000);
  const [commission, setCommission] = useState(0.001);
  const [leverage, setLeverage] = useState(1);
  const [dateRange, setDateRange] = useState<[dayjs.Dayjs, dayjs.Dayjs]>([dayjs().subtract(3, 'month'), dayjs()]);
  const [submitting, setSubmitting] = useState(false);
  const [status, setStatus] = useState<'idle' | 'running' | 'completed' | 'error'>('idle');
  const [metrics, setMetrics] = useState<Metrics | null>(null);
  const [errorMsg, setErrorMsg] = useState('');
  const [progress, setProgress] = useState('');
  const abortRef = useRef<AbortController | null>(null);

  useEffect(() => () => { abortRef.current?.abort(); }, []);

  const run = useCallback(async () => {
    setSubmitting(true);
    setStatus('running');
    setErrorMsg('');
    setMetrics(null);
    setProgress('Initializing...');

    const ac = new AbortController();
    abortRef.current?.abort();
    abortRef.current = ac;

    try {
      const msg = create(RunMarketBacktestRequestSchema, {
        strategyId,
        symbol,
        timeframe,
        startDateMs: BigInt(dateRange[0].valueOf()),
        endDateMs: BigInt(dateRange[1].valueOf()),
        initialCapital: String(initialCapital),
        executionConfig: {
          commission: String(commission),
          slippage: '0',
          leverage: String(leverage),
          tradeDirection: TradeDirection.BOTH,
          strictMode: true,
        },
      });

      // Single streaming call — replaces the old two-step (start + watch).
      const stream = marketplaceClient.runMarketBacktest(msg, { signal: ac.signal });
      for await (const update of stream) {
        const run = update.run;
        if (!run) continue;

        // Status display
        const st = run.status;
        if (st === BacktestRunStatus.SUCCEEDED || run.isTerminal) {
          if (st === BacktestRunStatus.SUCCEEDED) {
            setStatus('completed');
            if (update.metrics) {
              setMetrics({
                totalReturn: Number(update.metrics.totalReturn),
                annualReturn: Number(update.metrics.annualReturn),
                maxDrawdown: Number(update.metrics.maxDrawdown),
                sharpeRatio: Number(update.metrics.sharpeRatio),
                winRate: Number(update.metrics.winRate),
                totalTrades: Number(update.metrics.totalTrades),
                equityCurve: (update.equityCurve || []).map((v: string, i: number) => ({ time: i, equity: Number(v) })),
              });
            }
          } else if (st === BacktestRunStatus.FAILED || st === BacktestRunStatus.CANCELED) {
            setStatus('error');
            setErrorMsg(run.error || 'Backtest failed');
          }
          break;
        }
        if (st === BacktestRunStatus.RUNNING) {
          setProgress('Running...');
        }
      }
    } catch (e: unknown) {
      if (e instanceof Error && e.name === 'AbortError') return;
      setStatus('error');
      setErrorMsg(e instanceof Error ? e.message : String(e));
    } finally {
      setSubmitting(false);
    }
  }, [strategyId, symbol, timeframe, initialCapital, commission, leverage, dateRange]);

  return (
    <div style={{ padding: '12px 0' }}>
      <div style={{ marginBottom: 12, padding: '8px 12px', background: '#fffbe6', border: '1px solid #ffe58f', borderRadius: 6, display: 'flex', alignItems: 'center', gap: 8 }}>
        <LockOutlined style={{ color: '#fa8c16' }} />
        <Text type="secondary" style={{ fontSize: 12 }}>
          {t('marketplace.backtest.protected')}
        </Text>
      </div>

      <Row gutter={[8, 8]} style={{ marginBottom: 12 }}>
        <Col span={8}>
          <Select value={symbol} onChange={setSymbol} style={{ width: '100%' }}
            options={['EURUSD', 'GBPUSD', 'USDJPY', 'XAUUSD', 'BTCUSD'].map(s => ({ value: s, label: s }))} />
        </Col>
        <Col span={4}>
          <Select value={timeframe} onChange={setTimeframe} style={{ width: '100%' }}
            options={['M1', 'M5', 'M15', 'H1', 'H4', 'D1'].map(tf => ({ value: tf, label: tf }))} />
        </Col>
        <Col span={6}>
          <DatePicker.RangePicker value={dateRange} onChange={(v) => v && setDateRange(v as [dayjs.Dayjs, dayjs.Dayjs])} style={{ width: '100%' }} />
        </Col>
        <Col span={6}>
          <Button type="primary" loading={submitting} onClick={run} block>
            {status === 'running' ? progress : t('marketplace.backtest.run')}
          </Button>
        </Col>
      </Row>

      <Row gutter={[8, 8]} style={{ marginBottom: 12 }}>
        <Col span={6}>
          <Text type="secondary" style={{ fontSize: 11 }}>{t('marketplace.backtest.capital')}</Text>
          <InputNumber value={initialCapital} onChange={v => setInitialCapital(v || 10000)} min={100} step={1000} style={{ width: '100%' }} size="small" />
        </Col>
        <Col span={6}>
          <Text type="secondary" style={{ fontSize: 11 }}>{t('marketplace.backtest.commission')}</Text>
          <InputNumber value={commission} onChange={v => setCommission(v || 0)} min={0} step={0.0001} style={{ width: '100%' }} size="small" />
        </Col>
        <Col span={6}>
          <Text type="secondary" style={{ fontSize: 11 }}>{t('marketplace.backtest.leverage')}</Text>
          <InputNumber value={leverage} onChange={v => setLeverage(v || 1)} min={1} max={500} step={1} style={{ width: '100%' }} size="small" />
        </Col>
      </Row>

      {status === 'running' && <Tag color="processing" style={{ marginBottom: 12 }}>{progress}</Tag>}
      {status === 'completed' && <Tag color="success" style={{ marginBottom: 12 }}>{t('marketplace.backtest.completed')}</Tag>}
      {status === 'error' && <Tag color="error" style={{ marginBottom: 12 }}>{errorMsg}</Tag>}
      {status === 'idle' && <Empty description={t('marketplace.backtest.idle')} style={{ padding: 16 }} />}

      {metrics && status === 'completed' && (
        <>
          <Row gutter={[8, 8]}>
            <Col span={8}>
              <Card size="small"><Statistic title={t('marketplace.backtest.totalReturn')} value={pct(metrics.totalReturn)}
                prefix={metrics.totalReturn != null && metrics.totalReturn >= 0 ? <RiseOutlined style={{ color: '#26a69a' }} /> : <FallOutlined style={{ color: '#ef5350' }} />}
                valueStyle={{ fontSize: 16, fontFamily: 'monospace' }} /></Card>
            </Col>
            <Col span={8}>
              <Card size="small"><Statistic title={t('marketplace.backtest.maxDrawdown')} value={pct(metrics.maxDrawdown)}
                valueStyle={{ fontSize: 16, fontFamily: 'monospace', color: '#ef5350' }} /></Card>
            </Col>
            <Col span={8}>
              <Card size="small"><Statistic title={t('marketplace.backtest.sharpe')} value={metrics.sharpeRatio?.toFixed(2) ?? '-'}
                valueStyle={{ fontSize: 16, fontFamily: 'monospace' }} /></Card>
            </Col>
            <Col span={8}>
              <Card size="small"><Statistic title={t('marketplace.backtest.winRate')} value={pct(metrics.winRate)}
                valueStyle={{ fontSize: 16, fontFamily: 'monospace' }} /></Card>
            </Col>
            <Col span={8}>
              <Card size="small"><Statistic title={t('marketplace.backtest.totalTrades')} value={metrics.totalTrades ?? '-'}
                valueStyle={{ fontSize: 16, fontFamily: 'monospace' }} /></Card>
            </Col>
          </Row>

          {metrics.equityCurve && metrics.equityCurve.length > 0 && (
            <Card size="small" title={t('marketplace.backtest.equityCurve')} style={{ marginTop: 8 }}>
              <ResponsiveContainer width="100%" height={180}>
                <LineChart data={metrics.equityCurve}>
                  <XAxis dataKey="time" hide />
                  <YAxis width={50} tick={{ fontSize: 10 }} />
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
