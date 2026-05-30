import { useState } from 'react';
import {
  Card, Input, Select, Tag, Button, Space, Typography, Row, Col, Modal,
  Form, InputNumber, Empty, Descriptions, Tooltip, message, Spin, Tabs,
} from 'antd';
import {
  SearchOutlined, PlusOutlined, ExperimentOutlined,
  ShopOutlined, BookOutlined, FilterOutlined, PieChartOutlined,
} from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { marketplaceClient } from '@/client/connect';
import { useRpcQuery } from '@/hooks/useRpcQuery';
import { StatusResult } from '@/components/common/StatusResult';
import { useAuthStore } from '@/stores/authStore';

const { Title, Text, Paragraph } = Typography;

const ASSET_CLASSES = ['forex', 'crypto', 'commodity', 'index', 'stock', 'other'] as const;
const RISK_COLORS: Record<string, string> = { low: '#00A651', medium: '#FF9800', high: '#E53935' };
const RISK_BG: Record<string, string> = { low: 'rgba(0,166,81,0.1)', medium: 'rgba(255,152,0,0.1)', high: 'rgba(229,57,53,0.1)' };

interface MarketStrategy {
  publishId: string;
  strategyId: string;
  strategyName: string;
  publisherUserId: string;
  publishedAt?: string;
  riskLevel?: string; // backend will provide real risk data when available
}

type TabKey = 'marketplace' | 'subscriptions';

export default function MarketplacePage() {
  const { t } = useTranslation();
  const { user } = useAuthStore();
  const userId = user?.id || '';
  const [activeTab, setActiveTab] = useState<TabKey>('marketplace');
  const [searchText, setSearchText] = useState('');
  const [assetFilter, setAssetFilter] = useState('');
  const [publishOpen, setPublishOpen] = useState(false);
  const [publishLoading, setPublishLoading] = useState(false);
  const [publishForm] = Form.useForm();

  const { data: strategies = [], isLoading, error, refetch } = useRpcQuery(
    ['marketplace', 'published', userId],
    async () => {
      const resp = await marketplaceClient.listPublished({ userId, limit: 100 });
      return (resp.strategies || []) as unknown as MarketStrategy[];
    },
  );

  const { data: subscriptions = [], refetch: refetchSubs } = useRpcQuery(
    ['marketplace', 'subscriptions', userId],
    async () => {
      if (!userId) return [];
      const resp = await marketplaceClient.listSubscriptions({ userId });
      return (resp.subscriptions || []) as unknown as { subscriptionId: string; targetUserId: string; strategyId: string; kind: string; active: boolean; createdAt?: string }[];
    },
    { enabled: !!userId },
  );

  const handleSubscribe = async (publisherUserId: string, strategyId: string) => {
    if (!userId) { message.warning('Please log in first'); return; }
    try {
      await marketplaceClient.subscribe({ userId, publisherUserId, strategyId, kind: 'copy_trade' });
      message.success(t('marketplace.messages.subscribed', { defaultValue: 'Subscribed!' }));
      refetchSubs();
    } catch {
      message.error(t('marketplace.messages.subscribeFailed', { defaultValue: 'Subscribe failed' }));
    }
  };

  const handleUnsubscribe = async (subscriptionId: string) => {
    try {
      await marketplaceClient.unsubscribe({ userId, subscriptionId });
      message.success(t('marketplace.messages.unsubscribed', { defaultValue: 'Unsubscribed' }));
      refetchSubs();
    } catch {
      message.error(t('marketplace.messages.unsubscribeFailed', { defaultValue: 'Unsubscribe failed' }));
    }
  };

  const handlePublish = async (values: { strategyId: string; title: string }) => {
    setPublishLoading(true);
    try {
      await marketplaceClient.publishStrategy({ userId, strategyId: values.strategyId });
      message.success(t('marketplace.messages.published', { defaultValue: 'Strategy published!' }));
      setPublishOpen(false);
      publishForm.resetFields();
      refetch();
    } catch {
      message.error(t('marketplace.messages.publishFailed', { defaultValue: 'Publish failed' }));
    } finally {
      setPublishLoading(false);
    }
  };

  const isSubscribed = (strategyId: string) =>
    subscriptions.some(s => s.strategyId === strategyId);

  const filtered = strategies.filter(s => {
    if (searchText) {
      const q = searchText.toLowerCase();
      if (!(s.strategyName || '').toLowerCase().includes(q) && !s.strategyId.toLowerCase().includes(q)) return false;
    }
    // asset filter: currently strategies don't carry asset_class from backend
    // Future: add to proto and backend filtering
    return true;
  });

  const renderStrategyCard = (s: MarketStrategy) => {
    const isSub = isSubscribed(s.strategyId);
    const metaKey = `marketplace.strategies.${s.strategyId}`;
    const riskLevel = s.riskLevel || '-'; // backend will provide real risk data when available

    return (
      <Col xs={24} sm={12} lg={8} xl={6} key={s.publishId || s.strategyId}>
        <Card
          hoverable
          size="small"
          style={{ borderRadius: 12, height: '100%', borderColor: isSub ? '#D4AF37' : '#E5E7EB' }}
          actions={[
            isSub ? (
              <Tooltip key="sub" title={t('marketplace.card.unsubscribeHint', { defaultValue: 'Click to unsubscribe' })}>
                <Button type="link" size="small" onClick={() => {
                  const sub = subscriptions.find(x => x.strategyId === s.strategyId);
                  if (sub) handleUnsubscribe(sub.subscriptionId);
                }} style={{ color: '#D4AF37' }}>
                  {t('marketplace.card.subscribed', { defaultValue: 'Subscribed' })} ✓
                </Button>
              </Tooltip>
            ) : (
              <Button key="sub" type="link" size="small" onClick={() => handleSubscribe(s.publisherUserId, s.strategyId)}>
                {t('marketplace.card.subscribe', { defaultValue: 'Subscribe' })}
              </Button>
            ),
            <Button key="detail" type="link" size="small">
              {t('marketplace.card.details', { defaultValue: 'Details' })}
            </Button>,
          ]}
        >
          <div style={{ marginBottom: 8 }}>
            <Text strong style={{ fontSize: 15 }}>{s.strategyName || s.strategyId.slice(0, 8)}</Text>
          </div>
          <Space size={4} wrap style={{ marginBottom: 8 }}>
            <Tag color="blue">{t('marketplace.card.forex', { defaultValue: 'Forex' })}</Tag>
            {riskLevel !== '-' && (
              <Tag color={RISK_COLORS[riskLevel] || 'default'} style={{ background: RISK_BG[riskLevel] || undefined, border: 'none' }}>
                {t(`marketplace.risk.${riskLevel}`, { defaultValue: riskLevel })}
              </Tag>
            )}
          </Space>
          <div style={{ fontSize: 12, color: '#8A9AA5', marginBottom: 8 }}>
            <PieChartOutlined /> {t('marketplace.card.subscribers', { defaultValue: 'Subscribers', count: 0 })}: 0 &nbsp;
            <ExperimentOutlined /> {t('marketplace.card.winRate', { defaultValue: 'Win rate' })}: --%
          </div>
          <div style={{ fontSize: 11, color: '#B0BEC5' }}>
            {t('marketplace.card.by', { defaultValue: 'By' })} {s.publisherUserId ? s.publisherUserId.slice(0, 8) + '...' : 'unknown'}
            {s.publishedAt && ` · ${new Date(s.publishedAt).toLocaleDateString()}`}
          </div>
        </Card>
      </Col>
    );
  };

  return (
    <div className="min-h-screen" style={{ background: '#F5F7F9', padding: '24px 24px 80px' }}>
      <div className="max-w-7xl mx-auto">
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: 24, flexWrap: 'wrap', gap: 16 }}>
          <div>
            <Title level={3} style={{ margin: 0 }}>
              <ShopOutlined style={{ marginRight: 8 }} />
              {t('marketplace.title', { defaultValue: 'Strategy Marketplace' })}
            </Title>
            <Text type="secondary">
              {t('marketplace.subtitle', { defaultValue: 'Discover, subscribe, and earn from trading strategies' })}
            </Text>
          </div>
          <Button type="primary" icon={<PlusOutlined />} onClick={() => setPublishOpen(true)} style={{ borderRadius: 8 }}>
            {t('marketplace.publish', { defaultValue: 'Publish Strategy' })}
          </Button>
        </div>

        <Tabs
          activeKey={activeTab}
          onChange={k => setActiveTab(k as TabKey)}
          items={[
            {
              key: 'marketplace',
              label: <span><ShopOutlined /> {t('marketplace.tabs.marketplace', { defaultValue: 'Marketplace' })}</span>,
              children: (
                <div>
                  <div style={{ display: 'flex', gap: 12, marginBottom: 20, flexWrap: 'wrap' }}>
                    <Input prefix={<SearchOutlined />} placeholder={t('marketplace.searchPlaceholder', { defaultValue: 'Search strategies...' })}
                      value={searchText} onChange={e => setSearchText(e.target.value)}
                      style={{ maxWidth: 360, borderRadius: 8 }} allowClear />
                    <Select
                      value={assetFilter || undefined}
                      onChange={v => setAssetFilter(v || '')}
                      allowClear
                      placeholder={t('marketplace.filterByClass', { defaultValue: 'All asset classes' })}
                      style={{ minWidth: 180 }}
                      options={ASSET_CLASSES.map(c => ({ value: c, label: t(`marketplace.assetClass.${c}`, { defaultValue: c }) }))}
                    />
                  </div>
                  <StatusResult loading={isLoading} error={error instanceof Error ? error.message : undefined} onRetry={refetch}
                    empty={filtered.length === 0 && !isLoading}
                    emptyText={t('marketplace.empty', { defaultValue: 'No strategies published yet' })}>
                    <Row gutter={[16, 16]}>
                      {filtered.map(renderStrategyCard)}
                    </Row>
                  </StatusResult>
                </div>
              ),
            },
            {
              key: 'subscriptions',
              label: <span><BookOutlined /> {t('marketplace.tabs.subscriptions', { defaultValue: 'My Subscriptions' })}</span>,
              children: (
                <StatusResult
                  empty={subscriptions.length === 0}
                  emptyText={t('marketplace.noSubscriptions', { defaultValue: 'No active subscriptions' })}>
                  <Row gutter={[16, 16]}>
                    {subscriptions.map(sub => {
                      const pub = strategies.find(s => s.strategyId === sub.strategyId);
                      return (
                        <Col xs={24} sm={12} lg={8} key={sub.subscriptionId}>
                          <Card size="small" style={{ borderRadius: 12, borderColor: '#D4AF37' }}
                            actions={[
                              <Button key="unsub" type="link" size="small" danger onClick={() => handleUnsubscribe(sub.subscriptionId)}>
                                {t('marketplace.card.unsubscribe', { defaultValue: 'Unsubscribe' })}
                              </Button>,
                            ]}>
                            <Text strong>{pub?.strategyName || sub.strategyId.slice(0, 8)}</Text>
                            <br />
                            <Tag>{sub.kind}</Tag>
                            <Tag color={sub.active ? 'green' : 'default'}>{sub.active ? 'Active' : 'Inactive'}</Tag>
                            <div style={{ fontSize: 11, color: '#B0BEC5', marginTop: 4 }}>
                              {t('marketplace.card.subscribedSince', { defaultValue: 'Since' })}: {sub.createdAt ? new Date(sub.createdAt).toLocaleDateString() : '--'}
                            </div>
                          </Card>
                        </Col>
                      );
                    })}
                  </Row>
                </StatusResult>
              ),
            },
          ]}
        />

        <Modal
          title={t('marketplace.publishModal.title', { defaultValue: 'Publish to Marketplace' })}
          open={publishOpen}
          onCancel={() => setPublishOpen(false)}
          footer={null}
          destroyOnClose
        >
          <Form form={publishForm} layout="vertical" onFinish={handlePublish}>
            <Form.Item name="strategyId" label={t('marketplace.publishModal.strategyId', { defaultValue: 'Strategy ID' })}
              rules={[{ required: true, message: t('marketplace.publishModal.strategyIdRequired', { defaultValue: 'Please enter strategy ID' }) }]}>
              <Input placeholder="e.g. abc123-def4-..." />
            </Form.Item>
            <Form.Item name="title" label={t('marketplace.publishModal.title', { defaultValue: 'Display Title' })}>
              <Input placeholder={t('marketplace.publishModal.titlePlaceholder', { defaultValue: 'My Awesome Strategy' })} />
            </Form.Item>
            <Form.Item style={{ marginBottom: 0 }}>
              <Button type="primary" htmlType="submit" loading={publishLoading} block>
                {t('marketplace.publishModal.submit', { defaultValue: 'Publish' })}
              </Button>
            </Form.Item>
          </Form>
        </Modal>
      </div>
    </div>
  );
}
