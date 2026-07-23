import { useNavigate } from 'react-router-dom';
import { Card, Row, Col, Statistic, Table, Tag, Typography, Empty, Button, Space, Tooltip } from 'antd';
import { ShopOutlined, StarOutlined, SendOutlined, PlusOutlined, DollarOutlined, WalletOutlined, RiseOutlined, FallOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { TABLE_NAME_KEY } from '@/gen/ant/v1/i18n/strategy_templates_keys';
import { useMarketplaceCtx } from '../MarketplaceContext';
import { RevenueTrendChart, SubscriberTrendChart } from './AuthorCharts';
import ProviderEarningsPanel from './ProviderEarningsPanel';
import type { PublishedStrategy, StrategyBreakdown } from '@/gen/ant/v1/marketplace_service_pb';

const { Text } = Typography;

export default function AuthorTab() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const m = useMarketplaceCtx();
  const { myPublished, authorStats } = m;

  const breakdown = authorStats.strategyBreakdown || [];
  const subscriberTrend = authorStats.subscriberTrend || [];
  const lastDay = subscriberTrend.length > 0 ? subscriberTrend[subscriberTrend.length - 1] : null;
  const prevDay = subscriberTrend.length > 1 ? subscriberTrend[subscriberTrend.length - 2] : null;
  const newToday = lastDay ? Number(lastDay.newSubscribers || 0) : 0;
  const churnedToday = lastDay ? Number(lastDay.churned || 0) : 0;
  const activeNow = lastDay ? Number(lastDay.active || 0) : authorStats.totalSubscribers;
  const prevActive = prevDay ? Number(prevDay.active || 0) : activeNow;
  const netChange = activeNow - prevActive;

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <Space>
          <Text strong style={{ fontSize: 15 }}>{t('marketplace.author.myStrategies')}</Text>
        </Space>
        <Space>
          <Button icon={<WalletOutlined />} onClick={() => navigate('/wallet')}>
            {t('marketplace.author.wallet')}
          </Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={() => navigate('/strategy')}>
            {t('marketplace.author.publishNew')}
          </Button>
        </Space>
      </div>

      {/* ── Summary stats ── */}
      <Row gutter={[12, 12]} style={{ marginBottom: 20 }}>
        <Col xs={12} sm={6}>
          <Card size="small" style={{ background: '#f6ffed', borderRadius: 12, border: 'none' }}>
            <Statistic title={t('marketplace.author.published')} value={authorStats.published} prefix={<SendOutlined />} />
          </Card>
        </Col>
        <Col xs={12} sm={6}>
          <Card size="small" style={{ background: '#e6f7ff', borderRadius: 12, border: 'none' }}>
            <Statistic title={t('marketplace.author.subscribers')} value={activeNow} prefix={<ShopOutlined />} />
            {netChange !== 0 && (
              <Text type={netChange > 0 ? 'success' : 'danger'} style={{ fontSize: 12 }}>
                {netChange > 0 ? <RiseOutlined /> : <FallOutlined />} {Math.abs(netChange)} {t('marketplace.author.today')}
              </Text>
            )}
          </Card>
        </Col>
        <Col xs={12} sm={6}>
          <Card size="small" style={{ background: '#fff7e6', borderRadius: 12, border: 'none' }}>
            <Statistic title={t('marketplace.author.avgRating')} value={Number(authorStats.avgRating || 0).toFixed(1)} prefix={<StarOutlined />} />
          </Card>
        </Col>
        <Col xs={12} sm={6}>
          <Card size="small" style={{ background: '#f0f5ff', borderRadius: 12, border: 'none' }}>
            <Statistic title={t('marketplace.author.monthlyRevenue')} value={`¥${Number(authorStats.monthlyRevenue || 0).toFixed(2)}`} prefix={<DollarOutlined />} />
          </Card>
        </Col>
      </Row>

      {/* ── Provider earnings & transaction history ── */}
      <ProviderEarningsPanel />

      {/* ── Charts: Revenue trend + Subscriber trend ── */}
      <Row gutter={[12, 12]} style={{ marginBottom: 20 }}>
        <Col xs={24} lg={12}>
          <Card size="small" title={t('marketplace.author.revenueTrend')}>
            <RevenueTrendChart data={authorStats.revenueTrend || []} />
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card size="small" title={t('marketplace.author.subscriberTrend')}>
            <SubscriberTrendChart data={subscriberTrend} />
            <Space size="large" style={{ marginTop: 8 }}>
              <Text type="success">+{newToday} {t('marketplace.author.newSubs')}</Text>
              <Text type="danger">-{churnedToday} {t('marketplace.author.churned')}</Text>
            </Space>
          </Card>
        </Col>
      </Row>

      {/* ── Strategy breakdown table ── */}
      {breakdown.length > 0 && (
        <Card size="small" style={{ marginBottom: 20 }} title={t('marketplace.author.strategyBreakdown')}>
          <Table<StrategyBreakdown>
            rowKey="strategyId"
            dataSource={breakdown}
            pagination={false}
            size="small"
            columns={[
              { title: t('marketplace.author.strategyName'), dataIndex: 'title', key: 'title', render: (v: string) => <Text strong>{v || '-'}</Text> },
              { title: t('marketplace.detail.price'), key: 'price', render: (_: unknown, row: StrategyBreakdown) => (
                <Tag color={row.priceModel === 'free' ? 'green' : 'gold'}>
                  {row.priceModel === 'free' ? t('marketplace.card.free') : `¥${row.priceAmount || '0'}`}
                </Tag>
              )},
              { title: t('marketplace.author.subscribers'), dataIndex: 'totalSubscribers', key: 'subs', width: 100 },
              { title: t('marketplace.author.revenue'), dataIndex: 'revenue', key: 'revenue', width: 100, render: (v: string) => `¥${Number(v || 0).toFixed(2)}` },
              { title: t('marketplace.author.avgRating'), key: 'rating', width: 80, render: (_: unknown, row: StrategyBreakdown) => (
                <Tooltip title={`${row.ratingCount || 0} ratings`}>
                  <span>{Number(row.avgRating || 0).toFixed(1)}</span>
                </Tooltip>
              )},
            ]}
          />
        </Card>
      )}

      {/* ── Published strategies list ── */}
      {myPublished.length === 0 ? (
        <Empty description={t('marketplace.author.empty')}>
          <Button type="primary" onClick={() => navigate('/strategy')}>
            {t('marketplace.author.goToLibrary')}
          </Button>
        </Empty>
      ) : (
        <Table<PublishedStrategy>
          rowKey="publishId"
          dataSource={myPublished}
          pagination={{ pageSize: 10 }}
          size="small"
          columns={[
            { title: t(TABLE_NAME_KEY), dataIndex: 'strategyName', key: 'name', render: (n: string, row: PublishedStrategy) => <Text strong>{n || row.title || 'Unknown'}</Text> },
            { title: t('marketplace.detail.price'), key: 'price', render: (_: unknown, row: PublishedStrategy) => (<Tag color={!row.priceAmount || row.priceModel === 'free' ? 'green' : 'gold'}>{!row.priceAmount || row.priceModel === 'free' ? t('marketplace.card.free') : `¥${row.priceAmount}`}</Tag>) },
            { title: t('marketplace.card.winRate'), key: 'winRate', render: (_: unknown, row: PublishedStrategy) => row.winRate != null ? `${(row.winRate * 100).toFixed(0)}%` : '-' },
            { title: t('marketplace.author.subscribers'), dataIndex: 'totalSubscribers', key: 'subscribers' },
            { title: t('marketplace.author.avgRating'), key: 'rating', render: (_: unknown, row: PublishedStrategy) => Number(row.avgRating || 0).toFixed(1) },
          ]}
          locale={{ emptyText: t('marketplace.author.noPublished') }}
        />
      )}
    </div>
  );
}
