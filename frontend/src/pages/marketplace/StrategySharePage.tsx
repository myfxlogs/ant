import { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { Card, Typography, Tag, Rate, Space, Button, Spin, Statistic, Row, Col, Empty, Divider } from 'antd';
import { ShopOutlined, UserOutlined, ArrowRightOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import Seo from '@/components/common/Seo';
import { marketplacePublicClient } from '@/client/connect';
import type { GetStrategyPublicInfoResponse } from '@/gen/ant/v1/marketplace_service_pb';

const { Title, Paragraph, Text } = Typography;

export default function StrategySharePage() {
  const { strategyId } = useParams<{ strategyId: string }>();
  const navigate = useNavigate();
  const { t } = useTranslation();
  const [data, setData] = useState<GetStrategyPublicInfoResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(false);

  useEffect(() => {
    if (!strategyId) return;
    setLoading(true);
    marketplacePublicClient.getStrategyPublicInfo({ strategyId })
      .then(resp => setData(resp))
      .catch(() => setError(true))
      .finally(() => setLoading(false));
  }, [strategyId]);

  if (loading) {
    return (
      <div style={{ minHeight: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
        <Spin size="large" />
      </div>
    );
  }

  if (error || !data) {
    return (
      <div style={{ minHeight: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
        <Empty description={t('sharePage.notFound')} />
      </div>
    );
  }

  const priceLabel = data.priceModel === 'free' ? 'Free' :
    data.priceModel === 'subscription' ? `$${data.priceAmount}/mo` :
    `$${data.priceAmount}`;

  return (
    <>
      <Seo
        title={data.title}
        description={data.description?.slice(0, 160) || 'Strategy on AlphaForge Marketplace'}
        path={`/strategy/${strategyId}`}
        ogType="article"
      />
      <div style={{ minHeight: '100vh', background: '#f6f8fa', padding: '40px 24px' }}>
        <div style={{ maxWidth: 800, margin: '0 auto' }}>
          <Card style={{ borderRadius: 12 }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: 16 }}>
              <Space direction="vertical" size={4}>
                <Title level={3} style={{ margin: 0 }}>{data.title}</Title>
                <Space size={8}>
                  <Text type="secondary"><UserOutlined /> {data.publisherName || 'Publisher'}</Text>
                  {data.assetClass && <Tag>{data.assetClass}</Tag>}
                  {data.timeframe && <Tag>{data.timeframe}</Tag>}
                  {data.riskLevel && <Tag color={data.riskLevel === 'high' ? 'red' : data.riskLevel === 'medium' ? 'orange' : 'green'}>{data.riskLevel}</Tag>}
                </Space>
              </Space>
              <Tag color="#D4AF37" style={{ fontSize: 14, fontWeight: 600, padding: '4px 12px' }}>{priceLabel}</Tag>
            </div>

            {data.description && (
              <Paragraph type="secondary">{data.description}</Paragraph>
            )}

            {data.avgRating > 0 && (
              <div style={{ marginBottom: 16 }}>
                <Rate disabled value={data.avgRating} allowHalf />
                <Text style={{ marginLeft: 8 }}>{data.avgRating.toFixed(1)} ({data.ratingCount} ratings)</Text>
              </div>
            )}

            <Divider />

            <Row gutter={24}>
              <Col span={6}>
                <Statistic title="Subscribers" value={data.totalSubscribers} />
              </Col>
              <Col span={6}>
                <Statistic title="Rating" value={data.avgRating.toFixed(1)} suffix={`/ 5`} />
              </Col>
              {data.backtest && (
                <>
                  <Col span={6}>
                    <Statistic
                      title={t('sharePage.backtestReturn')}
                      value={data.backtest.totalReturn || '—'}
                    />
                  </Col>
                  <Col span={6}>
                    <Statistic
                      title={t('sharePage.maxDrawdown')}
                      value={data.backtest.maxDrawdown || '—'}
                    />
                  </Col>
                </>
              )}
            </Row>

            {data.liveTotalReturn && (
              <>
                <Divider />
                <Title level={5}>{t('sharePage.livePerformance')}</Title>
                <Row gutter={24}>
                  <Col span={8}>
                    <Statistic title={t('sharePage.liveReturn')} value={data.liveTotalReturn} />
                  </Col>
                  <Col span={8}>
                    <Statistic title={t('sharePage.liveMaxDD')} value={data.liveMaxDrawdown} />
                  </Col>
                  <Col span={8}>
                    <Statistic title="Sharpe" value={data.liveSharpeRatio} />
                  </Col>
                </Row>
                {data.trackingSince && (
                  <Text type="secondary" style={{ fontSize: 12 }}>Tracking since {data.trackingSince}</Text>
                )}
              </>
            )}

            {data.codeSnippet && (
              <>
                <Divider />
                <Title level={5}>{t('sharePage.codePreview')}</Title>
                <pre style={{
                  background: '#f5f5f5', padding: 16, borderRadius: 8,
                  fontSize: 12, overflow: 'auto', maxHeight: 300,
                }}>
                  {data.codeSnippet}
                </pre>
              </>
            )}

            <Divider />

            <div style={{ textAlign: 'center' }}>
              <Space size="large">
                <Button
                  type="primary"
                  size="large"
                  icon={<ShopOutlined />}
                  onClick={() => navigate('/register')}
                >
                  Sign Up to Access This Strategy
                </Button>
                <Button
                  size="large"
                  icon={<ArrowRightOutlined />}
                  onClick={() => navigate('/marketplace')}
                >
                  Browse Marketplace
                </Button>
              </Space>
            </div>
          </Card>
        </div>
      </div>
    </>
  );
}
