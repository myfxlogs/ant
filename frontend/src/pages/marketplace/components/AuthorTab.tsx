import { Card, Row, Col, Statistic, Table, Tag, Typography, Empty } from 'antd';
import { ShopOutlined, DollarOutlined, StarOutlined, SendOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import type { PublishedStrategy } from '@/gen/ant/v1/marketplace_service_pb';

const { Text } = Typography;

interface Props {
  published: PublishedStrategy[];
  stats: { published: number; totalSubscribers: number; avgRating: number };
}

export default function AuthorTab({ published, stats }: Props) {
  const { t } = useTranslation();

  if (published.length === 0) {
    return (
      <Empty
        description={t('marketplace.author.empty', '还没有发布策略。去策略库发布一个吧。')}
      />
    );
  }

  return (
    <div>
      {/* Stats */}
      <Row gutter={[12, 12]} style={{ marginBottom: 20 }}>
        <Col xs={12} sm={6}>
          <Card size="small" style={{ background: '#f6ffed', borderRadius: 12, border: 'none' }}>
            <Statistic title={t('marketplace.author.published')} value={stats.published} prefix={<SendOutlined />} />
          </Card>
        </Col>
        <Col xs={12} sm={6}>
          <Card size="small" style={{ background: '#e6f7ff', borderRadius: 12, border: 'none' }}>
            <Statistic title={t('marketplace.author.subscribers')} value={stats.totalSubscribers} prefix={<ShopOutlined />} />
          </Card>
        </Col>
        <Col xs={12} sm={6}>
          <Card size="small" style={{ background: '#fff7e6', borderRadius: 12, border: 'none' }}>
            <Statistic title={t('marketplace.author.avgRating')} value={stats.avgRating.toFixed(1)} prefix={<StarOutlined />} />
          </Card>
        </Col>
      </Row>

      {/* Published list */}
      <Table
        rowKey="publishId"
        dataSource={published}
        pagination={{ pageSize: 10 }}
        size="small"
        columns={[
          {
            title: t('strategy.templates.table.name'),
            dataIndex: 'strategyName',
            key: 'name',
            render: (n: string, row: any) => <Text strong>{n || row.title || 'Unknown'}</Text>,
          },
          {
            title: t('marketplace.detail.price'),
            key: 'price',
            render: (_: unknown, row: any) => {
              const free = !row.priceAmount || row.priceModel === 'free';
              return <Tag color={free ? 'green' : 'gold'}>{free ? t('marketplace.card.free') : `¥${row.priceAmount}`}</Tag>;
            },
          },
          {
            title: t('marketplace.author.subscribers'),
            dataIndex: 'totalSubscribers', key: 'subscribers',
          },
          {
            title: t('marketplace.author.avgRating'),
            key: 'rating',
            render: (_: unknown, row: any) => Number(row.avgRating || 0).toFixed(1),
          },
        ]}
        locale={{ emptyText: t('marketplace.author.noPublished') }}
      />
    </div>
  );
}
