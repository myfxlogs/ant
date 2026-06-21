import { useNavigate } from 'react-router-dom';
import { Card, Row, Col, Statistic, Table, Tag, Typography, Empty, Button, Space } from 'antd';
import { ShopOutlined, StarOutlined, SendOutlined, PlusOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next'
import { TABLE_NAME_KEY } from '@/gen/ant/v1/i18n/strategy_templates_keys';
;
import { useMarketplaceCtx } from '../MarketplaceContext';
import type { PublishedStrategy } from '@/gen/ant/v1/marketplace_service_pb';

const { Text } = Typography;

export default function AuthorTab() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const m = useMarketplaceCtx();
  const { myPublished, authorStats } = m;

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <Space>
          <Text strong style={{ fontSize: 15 }}>{t('marketplace.author.myStrategies', 'My Published Strategies')}</Text>
        </Space>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => navigate('/strategy/library')}>
          {t('marketplace.author.publishNew', 'Publish New Strategy')}
        </Button>
      </div>

      <Row gutter={[12, 12]} style={{ marginBottom: 20 }}>
        <Col xs={12} sm={6}>
          <Card size="small" style={{ background: '#f6ffed', borderRadius: 12, border: 'none' }}>
            <Statistic title={t('marketplace.author.published')} value={authorStats.published} prefix={<SendOutlined />} />
          </Card>
        </Col>
        <Col xs={12} sm={6}>
          <Card size="small" style={{ background: '#e6f7ff', borderRadius: 12, border: 'none' }}>
            <Statistic title={t('marketplace.author.subscribers')} value={authorStats.totalSubscribers} prefix={<ShopOutlined />} />
          </Card>
        </Col>
        <Col xs={12} sm={6}>
          <Card size="small" style={{ background: '#fff7e6', borderRadius: 12, border: 'none' }}>
            <Statistic title={t('marketplace.author.avgRating')} value={authorStats.avgRating.toFixed(1)} prefix={<StarOutlined />} />
          </Card>
        </Col>
      </Row>

      {myPublished.length === 0 ? (
        <Empty description={t('marketplace.author.empty')}>
          <Button type="primary" onClick={() => navigate('/strategy/library')}>
            {t('marketplace.author.goToLibrary', 'Go to Strategy Library')}
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
            { title: t('marketplace.card.winRate', 'Win Rate'), key: 'winRate', render: (_: unknown, row: PublishedStrategy) => row.winRate != null ? `${(row.winRate * 100).toFixed(0)}%` : '-' },
            { title: t('marketplace.author.subscribers'), dataIndex: 'totalSubscribers', key: 'subscribers' },
            { title: t('marketplace.author.avgRating'), key: 'rating', render: (_: unknown, row: PublishedStrategy) => Number(row.avgRating || 0).toFixed(1) },
          ]}
          locale={{ emptyText: t('marketplace.author.noPublished') }}
        />
      )}
    </div>
  );
}
