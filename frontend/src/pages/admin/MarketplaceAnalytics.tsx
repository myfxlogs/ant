import { useState, useEffect, useCallback } from 'react';
import { Card, Row, Col, Statistic, Select, Table, Spin, Empty } from 'antd';
import { useTranslation } from 'react-i18next';
import { marketplaceClient } from '@/client/connect';
import type { MarketplaceAnalytics, TopItem } from '@/gen/ant/v1/marketplace_service_pb';

export default function MarketplaceAnalyticsPage() {
  const { t } = useTranslation();
  const [analytics, setAnalytics] = useState<MarketplaceAnalytics | null>(null);
  const [topStrategies, setTopStrategies] = useState<{ byRev: TopItem[]; bySub: TopItem[] }>({ byRev: [], bySub: [] });
  const [topProviders, setTopProviders] = useState<{ byRev: TopItem[]; byStrat: TopItem[] }>({ byRev: [], byStrat: [] });
  const [loading, setLoading] = useState(true);
  const [period, setPeriod] = useState('30d');

  const fetchAll = useCallback(async (p: string) => {
    setLoading(true);
    try {
      const [a, ts, tp] = await Promise.all([
        marketplaceClient.getMarketplaceAnalytics({ period: p }),
        marketplaceClient.getTopStrategies({}),
        marketplaceClient.getTopProviders({}),
      ]);
      setAnalytics(a);
      setTopStrategies({ byRev: ts.byRevenue || [], bySub: ts.bySubscribers || [] });
      setTopProviders({ byRev: tp.byRevenue || [], byStrat: tp.byStrategies || [] });
    } catch {
      // ignore
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchAll(period);
  }, [period, fetchAll]);

  if (loading) return <Spin style={{ display: 'block', margin: '100px auto' }} />;
  if (!analytics) return <Empty />;

  const topRevCols = [
    { title: '#', dataIndex: 'rank', key: 'rank', width: 40 },
    { title: t('admin.analytics.name', { defaultValue: 'Name' }), dataIndex: 'name', key: 'name', ellipsis: true },
    { title: t('admin.analytics.value', { defaultValue: 'Value' }), dataIndex: 'value', key: 'value' },
  ];

  return (
    <div>
      <Select
        value={period}
        onChange={setPeriod}
        style={{ width: 120, marginBottom: 16 }}
        options={[
          { label: '7 days', value: '7d' },
          { label: '30 days', value: '30d' },
          { label: '90 days', value: '90d' },
          { label: 'All', value: 'all' },
        ]}
      />

      <Row gutter={16} style={{ marginBottom: 24 }}>
        <Col span={4}>
          <Card size="small">
            <Statistic title="GMV" value={analytics.totalGmv} precision={2} />
          </Card>
        </Col>
        <Col span={4}>
          <Card size="small">
            <Statistic title={t('admin.analytics.platformRev', { defaultValue: 'Platform Rev' })} value={analytics.platformRevenue} precision={2} />
          </Card>
        </Col>
        <Col span={4}>
          <Card size="small">
            <Statistic title={t('admin.analytics.providerRev', { defaultValue: 'Provider Rev' })} value={analytics.providerRevenue} precision={2} />
          </Card>
        </Col>
        <Col span={4}>
          <Card size="small">
            <Statistic title={t('admin.analytics.activeBuyers', { defaultValue: 'Active Buyers' })} value={analytics.activeBuyers} />
          </Card>
        </Col>
        <Col span={4}>
          <Card size="small">
            <Statistic title="ARPU" value={analytics.arpu} precision={2} />
          </Card>
        </Col>
        <Col span={4}>
          <Card size="small">
            <Statistic title={t('admin.analytics.refundRate', { defaultValue: 'Refund Rate' })} value={(parseFloat(analytics.refundRate) * 100).toFixed(1)} suffix="%" />
          </Card>
        </Col>
      </Row>

      <Row gutter={16} style={{ marginBottom: 24 }}>
        <Col span={6}>
          <Card size="small">
            <Statistic title={t('admin.analytics.totalTx', { defaultValue: 'Transactions' })} value={analytics.totalTransactions} />
          </Card>
        </Col>
        <Col span={6}>
          <Card size="small">
            <Statistic title={t('admin.analytics.newSubs', { defaultValue: 'New Subscribers' })} value={analytics.newSubscribers} />
          </Card>
        </Col>
        <Col span={6}>
          <Card size="small">
            <Statistic title={t('admin.analytics.totalStrategies', { defaultValue: 'Total Strategies' })} value={analytics.totalStrategies} />
          </Card>
        </Col>
        <Col span={6}>
          <Card size="small">
            <Statistic title={t('admin.analytics.newStrategies', { defaultValue: 'New Strategies' })} value={analytics.newStrategies} />
          </Card>
        </Col>
      </Row>

      <Row gutter={16}>
        <Col span={12}>
          <Card title={t('admin.analytics.topByRevenue', { defaultValue: 'Top Strategies by Revenue' })} size="small">
            <Table dataSource={topStrategies.byRev} columns={topRevCols} rowKey="id" pagination={false} size="small" />
          </Card>
        </Col>
        <Col span={12}>
          <Card title={t('admin.analytics.topBySubs', { defaultValue: 'Top Strategies by Subscribers' })} size="small">
            <Table dataSource={topStrategies.bySub} columns={topRevCols} rowKey="id" pagination={false} size="small" />
          </Card>
        </Col>
      </Row>

      <Row gutter={16} style={{ marginTop: 16 }}>
        <Col span={12}>
          <Card title={t('admin.analytics.topProvidersRev', { defaultValue: 'Top Providers by Revenue' })} size="small">
            <Table dataSource={topProviders.byRev} columns={topRevCols} rowKey="id" pagination={false} size="small" />
          </Card>
        </Col>
        <Col span={12}>
          <Card title={t('admin.analytics.topProvidersStrat', { defaultValue: 'Top Providers by Strategies' })} size="small">
            <Table dataSource={topProviders.byStrat} columns={topRevCols} rowKey="id" pagination={false} size="small" />
          </Card>
        </Col>
      </Row>
    </div>
  );
}
