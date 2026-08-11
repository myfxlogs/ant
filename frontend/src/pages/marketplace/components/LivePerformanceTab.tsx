import { useState, useEffect, useCallback } from 'react';
import { Card, Row, Col, Statistic, Empty, Spin, Typography, Button, Select, Alert } from 'antd';
import { LineChart, Line, XAxis, YAxis, Tooltip, ResponsiveContainer, CartesianGrid } from 'recharts';
import { useTranslation } from 'react-i18next';
import { marketplaceClient } from '@/client/connect';
import { accountApi } from '@/client/account';

const { Text } = Typography;

interface Props {
  strategyId: string;
  isOwner: boolean;
}

interface LivePoint {
  date: string;
  dailyPnl: string;
  dailyReturn: string;
  equity: string;
  drawdown: string;
  totalTrades: number;
  winningTrades: number;
}

interface LiveSummary {
  totalReturn: string;
  annualReturn: string;
  maxDrawdown: string;
  sharpeRatio: string;
  winRate: string;
  totalTrades: number;
  avgMonthlyReturn: string;
  trackingSince: string;
  lastUpdated: string;
}

export default function LivePerformanceTab({ strategyId, isOwner }: Props) {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [points, setPoints] = useState<LivePoint[]>([]);
  const [summary, setSummary] = useState<LiveSummary | null>(null);
  const [accounts, setAccounts] = useState<{label: string; value: string}[]>([]);
  const [selectedAccount, setSelectedAccount] = useState<string>('');
  const [linking, setLinking] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const resp = await marketplaceClient.getLivePerformance({ strategyId, limit: 90 });
      setPoints((resp.points || []).map(p => ({
        date: p.date, dailyPnl: p.dailyPnl, dailyReturn: p.dailyReturn,
        equity: p.equity, drawdown: p.drawdown,
        totalTrades: p.totalTrades, winningTrades: p.winningTrades,
      })));
      if (resp.summary) {
        setSummary({
          totalReturn: resp.summary.totalReturn, annualReturn: resp.summary.annualReturn,
          maxDrawdown: resp.summary.maxDrawdown, sharpeRatio: resp.summary.sharpeRatio,
          winRate: resp.summary.winRate, totalTrades: resp.summary.totalTrades,
          avgMonthlyReturn: resp.summary.avgMonthlyReturn,
          trackingSince: resp.summary.trackingSince, lastUpdated: resp.summary.lastUpdated,
        });
      } else {
        setSummary(null);
      }
    } catch {
      setPoints([]);
      setSummary(null);
      setError(t('marketplace.live.loadError', 'Failed to load live performance data'));
    } finally {
      setLoading(false);
    }
  }, [strategyId, t]);

  useEffect(() => { load(); }, [load]);

  useEffect(() => {
    if (!isOwner) return;
    accountApi.list().then(accs => {
      setAccounts(accs.map(a => ({
        label: `${a.login} (${a.brokerCompany || a.mtType})`,
        value: a.id,
      })));
    }).catch(() => {});
  }, [isOwner]);

  const handleLink = useCallback(async () => {
    if (!selectedAccount) return;
    setLinking(true);
    try {
      await marketplaceClient.linkLiveAccount({ strategyId, accountId: selectedAccount });
      load();
    } finally {
      setLinking(false);
    }
  }, [strategyId, selectedAccount, load]);

  const chartData = points.map(p => ({
    date: p.date,
    equity: Number(p.equity),
    pnl: Number(p.dailyPnl),
  })).reverse();

  const hasData = points.length > 0;

  if (loading) return <Spin style={{ display: 'block', padding: 40 }} />;

  if (error) {
    return (
      <div style={{ padding: 40 }}>
        <Alert
          type="error"
          message={error}
          showIcon
          action={<Button size="small" onClick={load}>{t('common.retry', { defaultValue: 'Retry' })}</Button>}
        />
      </div>
    );
  }

  if (!hasData && !isOwner) {
    return <Empty description={t('marketplace.live.noData')} style={{ padding: 40 }} />;
  }

  if (!hasData && isOwner) {
    return (
      <div style={{ padding: 16 }}>
        <Empty description={t('marketplace.live.linkAccount')} />
        <div style={{ marginTop: 16, display: 'flex', gap: 8, justifyContent: 'center' }}>
          <Select
            placeholder={t('marketplace.live.selectAccount')}
            style={{ width: 240 }}
            value={selectedAccount || undefined}
            onChange={setSelectedAccount}
            options={accounts}
          />
          <Button type="primary" loading={linking} disabled={!selectedAccount} onClick={handleLink}>
            {t('marketplace.live.link')}
          </Button>
        </div>
      </div>
    );
  }

  return (
    <div style={{ padding: '8px 0' }}>
      {summary && (
        <Row gutter={[8, 8]} style={{ marginBottom: 12 }}>
          <Col span={6}>
            <Card size="small"><Statistic title={t('marketplace.live.totalReturn')} value={summary.totalReturn} valueStyle={{ fontSize: 16, fontFamily: 'monospace' }} /></Card>
          </Col>
          <Col span={6}>
            <Card size="small"><Statistic title={t('marketplace.live.maxDrawdown')} value={summary.maxDrawdown} valueStyle={{ fontSize: 16, fontFamily: 'monospace', color: '#ef5350' }} /></Card>
          </Col>
          <Col span={6}>
            <Card size="small"><Statistic title={t('marketplace.live.sharpe')} value={summary.sharpeRatio || '-'} valueStyle={{ fontSize: 16, fontFamily: 'monospace' }} /></Card>
          </Col>
          <Col span={6}>
            <Card size="small"><Statistic title={t('marketplace.live.winRate')} value={summary.winRate || '-'} valueStyle={{ fontSize: 16, fontFamily: 'monospace' }} /></Card>
          </Col>
        </Row>
      )}

      <Card size="small" title={t('marketplace.live.equityCurve')}>
        <ResponsiveContainer width="100%" height={200}>
          <LineChart data={chartData}>
            <CartesianGrid strokeDasharray="3 3" />
            <XAxis dataKey="date" tick={{ fontSize: 10 }} />
            <YAxis width={60} tick={{ fontSize: 10 }} />
            <Tooltip />
            <Line type="monotone" dataKey="equity" stroke="#1890ff" dot={false} strokeWidth={1.5} />
          </LineChart>
        </ResponsiveContainer>
      </Card>

      {summary && (
        <Text type="secondary" style={{ fontSize: 11, display: 'block', marginTop: 8 }}>
          {t('marketplace.live.trackingSince')}: {summary.trackingSince}
          {' | '}
          {t('marketplace.live.lastUpdated')}: {summary.lastUpdated}
        </Text>
      )}
    </div>
  );
}
