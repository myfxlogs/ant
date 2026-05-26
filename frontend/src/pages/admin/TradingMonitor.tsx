import { useMemo, useState } from 'react';
import { Card, Table, Row, Col, Statistic, DatePicker } from 'antd';
import { IconTrendingUp, IconTrendingDown } from '@tabler/icons-react';
import { useRpcQuery } from '@/hooks/useRpcQuery';
import { adminApi } from '@/client/admin';
import type { TradingSummary } from '@/client/admin';
import { StatusResult } from '@/components/common/StatusResult';
import { useTranslation } from 'react-i18next';
import dayjs from 'dayjs';

const { RangePicker } = DatePicker;

function int(v: bigint | number | undefined): number {
  if (typeof v === 'bigint') return Number(v);
  return v ?? 0;
}

export default function TradingMonitor() {
  const { t } = useTranslation();

  const [dates, setDates] = useState<[string, string]>(() => {
    const end = dayjs().format('YYYY-MM-DD');
    const start = dayjs().subtract(30, 'day').format('YYYY-MM-DD');
    return [start, end];
  });

  const { data: summary, isLoading, error, refetch } = useRpcQuery(
    ['admin', 'tradingSummary', ...dates],
    () => adminApi.getTradingSummary({ startDate: dates[0], endDate: dates[1] }) as Promise<TradingSummary>,
  );

  const handleDateChange = (vals: [Date | null, Date | null] | null) => {
    if (vals?.[0] && vals?.[1]) {
      setDates([
        dayjs(vals[0]).format('YYYY-MM-DD'),
        dayjs(vals[1]).format('YYYY-MM-DD'),
      ]);
    }
  };

  const platformColumns = useMemo(() => [
    { title: t('admin.trading.platform'), dataIndex: 'platform', key: 'platform' },
    { title: t('admin.trading.accounts'), dataIndex: 'accounts', key: 'accounts', render: (v: bigint | number) => int(v) },
    { title: t('admin.trading.orders'), dataIndex: 'orders', key: 'orders', render: (v: bigint | number) => int(v) },
    {
      title: t('admin.trading.volume'),
      dataIndex: 'volume',
      key: 'volume',
      render: (value: number) => value?.toFixed(2) || '0.00',
    },
  ], [t]);

  const platformData = useMemo(() =>
    summary?.byPlatform
      ? Object.entries(summary.byPlatform).map(([platform, data]) => ({
          platform,
          accounts: int(data.accounts),
          orders: int(data.orders),
          volume: data.volume,
        }))
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
              <Statistic title={t('admin.trading.totalUsers')} value={int(summary?.overview?.totalUsers)} />
            </Card>
          </Col>
          <Col xs={12} sm={6}>
            <Card>
              <Statistic title={t('admin.trading.activeUsers')} value={int(summary?.overview?.activeUsers)} />
            </Card>
          </Col>
          <Col xs={12} sm={6}>
            <Card>
              <Statistic title={t('admin.trading.totalAccounts')} value={int(summary?.overview?.totalAccounts)} />
            </Card>
          </Col>
          <Col xs={12} sm={6}>
            <Card>
              <Statistic title={t('admin.trading.connectedAccounts')} value={int(summary?.overview?.connectedAccounts)} />
            </Card>
          </Col>
        </Row>

        <Row gutter={[16, 16]}>
          <Col xs={12} sm={6}>
            <Card>
              <Statistic title={t('admin.trading.totalOrders')} value={int(summary?.trading?.totalOrders)} />
            </Card>
          </Col>
          <Col xs={12} sm={6}>
            <Card>
              <Statistic title={t('admin.trading.closedOrders')} value={int(summary?.trading?.closedOrders)} />
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
              <Statistic title={t('admin.trading.pendingOrders')} value={int(summary?.trading?.pendingOrders)} />
            </Col>
          </Row>
        </Card>
      </div>
    </StatusResult>
  );
}
