import { useMemo } from 'react';
import { Card, Table, Row, Col, Statistic, DatePicker } from 'antd';
import { IconTrendingUp, IconTrendingDown } from '@tabler/icons-react';
import { useRpcQuery } from '@/hooks/useRpcQuery';
import { adminApi } from '@/client/admin';
import { StatusResult } from '@/components/common/StatusResult';
import { useTranslation } from 'react-i18next';

const { RangePicker } = DatePicker;

interface PlatformData {
  platform: string;
  accounts?: number;
  orders?: number;
  volume?: number;
}

interface TradingMonitorSummary {
  overview?: {
    totalUsers?: number;
    activeUsers?: number;
    totalAccounts?: number;
    connectedAccounts?: number;
  };
  trading?: {
    totalOrders?: number;
    closedOrders?: number;
    totalVolume?: number;
    netProfit?: number;
    totalProfit?: number;
    totalLoss?: number;
    pendingOrders?: number;
  };
  byPlatform?: Record<string, PlatformData>;
}

export default function TradingMonitor() {
  const { t } = useTranslation();

  const { data: summary, isLoading, error, refetch } = useRpcQuery(
    ['admin', 'tradingSummary'],
    () => adminApi.getTradingSummary() as Promise<TradingMonitorSummary>,
  );

  const handleDateChange = (dates: [Date | null, Date | null] | null) => {
    // Dates trigger a re-fetch of the same query. In a real implementation,
    // the date range would be passed as query params.
    refetch();
  };

  const platformColumns = useMemo(() => [
    { title: t('admin.trading.platform'), dataIndex: 'platform', key: 'platform' },
    { title: t('admin.trading.accounts'), dataIndex: 'accounts', key: 'accounts' },
    { title: t('admin.trading.orders'), dataIndex: 'orders', key: 'orders' },
    {
      title: t('admin.trading.volume'),
      dataIndex: 'volume',
      key: 'volume',
      render: (value: number) => value?.toFixed(2) || '0.00',
    },
  ], [t]);

  const platformData = useMemo(() =>
    summary?.byPlatform
      ? Object.entries(summary.byPlatform).map(([platform, data]) => ({ platform, ...data }))
      : [],
    [summary],
  );

  return (
    <StatusResult loading={isLoading} error={error?.message} onRetry={() => refetch()} empty={!isLoading && !error && !summary}>
      <div className="space-y-6">
        <div className="flex justify-between items-center">
          <h1 className="text-2xl font-bold" style={{ color: '#141D22' }}>{t('admin.trading.title')}</h1>
          <RangePicker onChange={(dates) => handleDateChange(dates as [Date | null, Date | null] | null)} />
        </div>

        <Row gutter={[16, 16]}>
          <Col xs={12} sm={6}>
            <Card>
              <Statistic title={t('admin.trading.totalUsers')} value={summary?.overview?.totalUsers || 0} />
            </Card>
          </Col>
          <Col xs={12} sm={6}>
            <Card>
              <Statistic title={t('admin.trading.activeUsers')} value={summary?.overview?.activeUsers || 0} />
            </Card>
          </Col>
          <Col xs={12} sm={6}>
            <Card>
              <Statistic title={t('admin.trading.totalAccounts')} value={summary?.overview?.totalAccounts || 0} />
            </Card>
          </Col>
          <Col xs={12} sm={6}>
            <Card>
              <Statistic title={t('admin.trading.connectedAccounts')} value={summary?.overview?.connectedAccounts || 0} />
            </Card>
          </Col>
        </Row>

        <Row gutter={[16, 16]}>
          <Col xs={12} sm={6}>
            <Card>
              <Statistic title={t('admin.trading.totalOrders')} value={summary?.trading?.totalOrders || 0} />
            </Card>
          </Col>
          <Col xs={12} sm={6}>
            <Card>
              <Statistic title={t('admin.trading.closedOrders')} value={summary?.trading?.closedOrders || 0} />
            </Card>
          </Col>
          <Col xs={12} sm={6}>
            <Card>
              <Statistic title={t('admin.trading.totalVolume')} value={summary?.trading?.totalVolume || 0} precision={2} />
            </Card>
          </Col>
          <Col xs={12} sm={6}>
            <Card>
              <Statistic
                title={t('admin.trading.netProfit')}
                value={summary?.trading?.netProfit || 0}
                precision={2}
                valueStyle={{ color: (summary?.trading?.netProfit || 0) >= 0 ? '#52c41a' : '#ff4d4f' }}
                prefix={(summary?.trading?.netProfit || 0) >= 0 ? <IconTrendingUp size={16} /> : <IconTrendingDown size={16} />}
              />
            </Card>
          </Col>
        </Row>

        <Card title={t('admin.trading.byPlatform')}>
          <Table scroll={{ x: "max-content" }} columns={platformColumns} dataSource={platformData} rowKey="platform" pagination={false} />
        </Card>

        <Card title={t('admin.trading.profitStats')}>
          <Row gutter={[16, 16]}>
            <Col xs={12} sm={8}>
              <Statistic
                title={t('admin.trading.totalProfit')}
                value={summary?.trading?.totalProfit || 0}
                precision={2}
                valueStyle={{ color: '#52c41a' }}
                prefix={<IconTrendingUp size={16} />}
              />
            </Col>
            <Col xs={12} sm={8}>
              <Statistic
                title={t('admin.trading.totalLoss')}
                value={Math.abs(summary?.trading?.totalLoss || 0)}
                precision={2}
                valueStyle={{ color: '#ff4d4f' }}
                prefix={<IconTrendingDown size={16} />}
              />
            </Col>
            <Col xs={12} sm={8}>
              <Statistic title={t('admin.trading.pendingOrders')} value={summary?.trading?.pendingOrders || 0} />
            </Col>
          </Row>
        </Card>
      </div>
    </StatusResult>
  );
}
