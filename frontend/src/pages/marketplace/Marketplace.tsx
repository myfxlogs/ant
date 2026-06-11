import { useState, useCallback } from 'react';
import {
  Card, Input, Select, Tag, Button, Space, Typography, Row, Col,
  Tooltip, message, Tabs,
  Grid,
} from 'antd';
import {
  SearchOutlined, PlusOutlined,
  ShopOutlined, BookOutlined,
} from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { marketplaceClient } from '@/client/connect';
import { useRpcQuery } from '@/hooks/useRpcQuery';
import { StatusResult } from '@/components/common/StatusResult';
import { useAuthStore } from '@/stores/authStore';
import PublishStrategyModal from './PublishStrategyModal';
import StrategyCard from './StrategyCard';
import StrategyDetailDrawer from './StrategyDetailDrawer';
import type { PublishedStrategy } from '@/gen/ant/v1/marketplace_service_pb';

const { Title, Text } = Typography;
const ASSET_CLASSES = ['forex', 'crypto', 'commodity', 'index', 'stock', 'other'] as const;

type TabKey = 'marketplace' | 'subscriptions';

export default function MarketplacePage() {
  const { t } = useTranslation();
  const { user } = useAuthStore();
  const screens = Grid.useBreakpoint();
  const isMobile = !screens.sm;
  const userId = user?.id || '';
  const [activeTab, setActiveTab] = useState<TabKey>('marketplace');
  const [searchText, setSearchText] = useState('');
  const [assetFilter, setAssetFilter] = useState('');
  const [sortBy, setSortBy] = useState('newest');
  const [publishOpen, setPublishOpen] = useState(false);
  const [detailStrategy, setDetailStrategy] = useState<PublishedStrategy | null>(null);

  const { data: strategies = [], isLoading, error, refetch } = useRpcQuery(
    ['marketplace', 'published', userId, assetFilter, searchText, sortBy],
    async () => {
      const resp = await marketplaceClient.listPublished({
        userId, limit: 100,
        assetClass: assetFilter || undefined,
        keyword: searchText || undefined,
        sortBy: sortBy || undefined,
      });
      return (resp.strategies || []) as PublishedStrategy[];
    },
  );

  const { data: subscriptions = [], refetch: refetchSubs } = useRpcQuery(
    ['marketplace', 'subscriptions', userId],
    async () => {
      if (!userId) return [];
      const resp = await marketplaceClient.listSubscriptions({ userId });
      return resp.subscriptions || [];
    },
    { enabled: !!userId },
  );

  const handleSubscribe = useCallback(async (publisherUserId: string, strategyId: string) => {
    if (!userId) { message.warning(t('marketplace.messages.loginFirst')); return; }
    try {
      await marketplaceClient.subscribe({ userId, publisherUserId, strategyId, kind: 'copy_trade' });
      message.success(t('marketplace.messages.subscribed'));
      refetchSubs();
    } catch { message.error(t('marketplace.messages.subscribeFailed')); }
  }, [userId, t, refetchSubs]);

  const handleUnsubscribe = useCallback(async (subscriptionId: string) => {
    try {
      await marketplaceClient.unsubscribe({ userId, subscriptionId });
      message.success(t('marketplace.messages.unsubscribed'));
      refetchSubs();
    } catch { message.error(t('marketplace.messages.unsubscribeFailed')); }
  }, [userId, t, refetchSubs]);

  const isSubscribed = (strategyId: string) => subscriptions.some(s => s.strategyId === strategyId);
  const getSubId = (strategyId: string) => subscriptions.find(s => s.strategyId === strategyId)?.subscriptionId;

  return (
    <div className="min-h-screen" style={{ background: 'var(--color-bg-secondary)', padding: isMobile ? '16px 12px 80px' : '24px 24px 80px' }}>
      <div className="max-w-7xl mx-auto">
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: 24, flexWrap: 'wrap', gap: 16 }}>
          <div>
            <Title level={3} style={{ margin: 0 }}><ShopOutlined style={{ marginRight: 8 }} />{t('marketplace.title')}</Title>
            <Text type="secondary">{t('marketplace.subtitle')}</Text>
          </div>
          <Button type="primary" icon={<PlusOutlined />} onClick={() => setPublishOpen(true)} style={{ borderRadius: 8 }}>{t('marketplace.publish')}</Button>
        </div>
        <Tabs activeKey={activeTab} onChange={k => setActiveTab(k as TabKey)} items={[
          {
            key: 'marketplace',
            label: <span><ShopOutlined /> {t('marketplace.tabs.marketplace')}</span>,
            children: (
              <div>
                <Row gutter={[12, 12]} style={{ marginBottom: 20 }}>
                  <Col xs={24} sm={12} md={8}>
                    <Input prefix={<SearchOutlined />} placeholder={t('marketplace.searchPlaceholder')} value={searchText}
                      onChange={e => setSearchText(e.target.value)} style={{ width: '100%', borderRadius: 8 }} allowClear />
                  </Col>
                  <Col xs={12} sm={6} md={4}>
                    <Select value={assetFilter || undefined} onChange={v => setAssetFilter(v || '')} allowClear
                      placeholder={t('marketplace.filterByClass')} style={{ width: '100%' }}
                      options={ASSET_CLASSES.map(c => ({ value: c, label: t(`marketplace.assetClass.${c}`, { defaultValue: c }) }))} />
                  </Col>
                  <Col xs={12} sm={6} md={4}>
                    <Select value={sortBy} onChange={v => setSortBy(v)} style={{ width: '100%' }}
                      options={[
                        { value: 'newest', label: t('marketplace.sort.newest') },
                        { value: 'popular', label: t('marketplace.sort.popular') },
                        { value: 'performance', label: t('marketplace.sort.performance') },
                      ]} />
                  </Col>
                </Row>
                <StatusResult loading={isLoading} error={error instanceof Error ? error.message : undefined} onRetry={refetch}
                  empty={strategies.length === 0 && !isLoading} emptyText={t('marketplace.empty')}>
                  <Row gutter={[16, 16]}>
                    {strategies.map(s => (
                      <StrategyCard
                        key={s.publishId || s.strategyId}
                        strategy={s}
                        isSubscribed={isSubscribed(s.strategyId)}
                        subscriptionId={getSubId(s.strategyId)}
                        userId={userId}
                        onSubscribe={handleSubscribe}
                        onUnsubscribe={handleUnsubscribe}
                        onOpenDetail={setDetailStrategy}
                        onRefresh={refetch}
                      />
                    ))}
                  </Row>
                </StatusResult>
              </div>
            ),
          },
          {
            key: 'subscriptions',
            label: <span><BookOutlined /> {t('marketplace.tabs.subscriptions')}</span>,
            children: (
              <StatusResult empty={subscriptions.length === 0} emptyText={t('marketplace.noSubscriptions')}>
                <Row gutter={[16, 16]}>
                  {subscriptions.map(sub => {
                    const pub = strategies.find(s => s.strategyId === sub.strategyId);
                    return (
                      <Col xs={24} sm={12} lg={8} key={sub.subscriptionId}>
                        <Card size="small" style={{ borderRadius: 12, borderColor: '#D4AF37' }}
                          actions={[<Button key="unsub" type="link" size="small" danger onClick={() => handleUnsubscribe(sub.subscriptionId)}>{t('marketplace.card.unsubscribe')}</Button>]}>
                          <Text strong>{pub?.strategyName || pub?.title || sub.strategyId.slice(0, 8)}</Text><br />
                          <Tag>{sub.kind}</Tag>
                          <Tag color={sub.active ? 'green' : 'default'}>{sub.active ? t('common.active') : t('common.inactive')}</Tag>
                          {pub?.winRate != null && <Tag color="blue">{t('marketplace.card.winRate')}: {(pub.winRate * 100).toFixed(0)}%</Tag>}
                        </Card>
                      </Col>
                    );
                  })}
                </Row>
              </StatusResult>
            ),
          },
        ]} />
        <PublishStrategyModal open={publishOpen} userId={userId} onClose={() => setPublishOpen(false)} onPublished={refetch} />

        <StrategyDetailDrawer
          strategy={detailStrategy}
          userId={userId}
          isMobile={isMobile}
          onClose={() => setDetailStrategy(null)}
          onRefresh={refetch}
        />
      </div>
    </div>
  );
}
