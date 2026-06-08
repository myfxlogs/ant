import { useState, useCallback } from 'react';
import {
  Card, Input, Select, Tag, Button, Space, Typography, Row, Col,
  Tooltip, message, Tabs, Rate, Drawer, List, Form, Divider,
} from 'antd';
import {
  SearchOutlined, PlusOutlined, ExperimentOutlined,
  ShopOutlined, BookOutlined, PieChartOutlined,
  StarFilled, SendOutlined,
} from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { marketplaceClient } from '@/client/connect';
import { useRpcQuery } from '@/hooks/useRpcQuery';
import { StatusResult } from '@/components/common/StatusResult';
import { useAuthStore } from '@/stores/authStore';
import PublishStrategyModal from './PublishStrategyModal';
import type { PublishedStrategy, CommentItem } from '@/gen/ant/v1/marketplace_service_pb';

const { Title, Text, Paragraph } = Typography;
const ASSET_CLASSES = ['forex', 'crypto', 'commodity', 'index', 'stock', 'other'] as const;
const RISK_COLORS: Record<string, string> = { low: '#00A651', medium: '#FF9800', high: '#E53935' };
const RISK_BG: Record<string, string> = { low: 'rgba(0,166,81,0.1)', medium: 'rgba(255,152,0,0.1)', high: 'rgba(229,57,53,0.1)' };

type TabKey = 'marketplace' | 'subscriptions';

export default function MarketplacePage() {
  const { t } = useTranslation();
  const { user } = useAuthStore();
  const userId = user?.id || '';
  const [activeTab, setActiveTab] = useState<TabKey>('marketplace');
  const [searchText, setSearchText] = useState('');
  const [assetFilter, setAssetFilter] = useState('');
  const [sortBy, setSortBy] = useState('newest');
  const [publishOpen, setPublishOpen] = useState(false);
  const [detailStrategy, setDetailStrategy] = useState<PublishedStrategy | null>(null);
  const [comments, setComments] = useState<CommentItem[]>([]);
  const [commentsLoading, setCommentsLoading] = useState(false);
  const [commentText, setCommentText] = useState('');
  const [commentSubmitting, setCommentSubmitting] = useState(false);

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

  const handleSubscribe = async (publisherUserId: string, strategyId: string) => {
    if (!userId) { message.warning(t('marketplace.messages.loginFirst')); return; }
    try {
      await marketplaceClient.subscribe({ userId, publisherUserId, strategyId, kind: 'copy_trade' });
      message.success(t('marketplace.messages.subscribed'));
      refetchSubs();
    } catch { message.error(t('marketplace.messages.subscribeFailed')); }
  };

  const handleUnsubscribe = async (subscriptionId: string) => {
    try {
      await marketplaceClient.unsubscribe({ userId, subscriptionId });
      message.success(t('marketplace.messages.unsubscribed'));
      refetchSubs();
    } catch { message.error(t('marketplace.messages.unsubscribeFailed')); }
  };

  const isSubscribed = (strategyId: string) => subscriptions.some(s => s.strategyId === strategyId);

  const openDetail = useCallback(async (s: PublishedStrategy) => {
    setDetailStrategy(s);
    setComments([]);
    setCommentText('');
    setCommentsLoading(true);
    try {
      const resp = await marketplaceClient.listComments({
        strategyId: s.publishId,
        limit: 50,
        offset: 0,
      });
      setComments((resp.comments || []) as CommentItem[]);
    } catch { /* comments load failed — non-critical */ }
    setCommentsLoading(false);
  }, []);

  const submitComment = useCallback(async () => {
    if (!commentText.trim() || !detailStrategy) return;
    setCommentSubmitting(true);
    try {
      await marketplaceClient.commentOnStrategy({
        userId,
        strategyId: detailStrategy.publishId,
        content: commentText.trim(),
      });
      message.success(t('marketplace.messages.commentPosted'));
      setCommentText('');
      // Reload comments.
      const resp = await marketplaceClient.listComments({
        strategyId: detailStrategy.publishId,
        limit: 50,
        offset: 0,
      });
      setComments((resp.comments || []) as CommentItem[]);
    } catch { message.error(t('marketplace.messages.commentFailed')); }
    setCommentSubmitting(false);
  }, [commentText, detailStrategy, userId, t]);

  const renderStrategyCard = (s: PublishedStrategy) => {
    const isSub = isSubscribed(s.strategyId);
    const riskLevel = s.riskLevel || 'medium';
    const assetClass = s.assetClass || 'forex';
    const subscribers = s.totalSubscribers || 0;
    const winRate = s.winRate != null ? `${(s.winRate * 100).toFixed(0)}%` : '--';
    const displayName = s.strategyName || s.title || s.strategyId.slice(0, 8);

    return (
      <Col xs={24} sm={12} lg={8} xl={6} key={s.publishId || s.strategyId}>
        <Card hoverable size="small" style={{ borderRadius: 12, height: '100%', borderColor: isSub ? '#D4AF37' : '#E5E7EB' }}
          actions={[
            isSub ? (
              <Tooltip key="sub" title={t('marketplace.card.unsubscribeHint')}>
                <Button type="link" size="small" onClick={() => {
                  const sub = subscriptions.find(x => x.strategyId === s.strategyId);
                  if (sub) handleUnsubscribe(sub.subscriptionId);
                }} style={{ color: '#D4AF37' }}>{t('marketplace.card.subscribed')} ✓</Button>
              </Tooltip>
            ) : (
              <Button key="sub" type="link" size="small" onClick={() => handleSubscribe(s.publisherUserId, s.strategyId)}>
                {t('marketplace.card.subscribe')}
              </Button>
            ),
            <Button key="detail" type="link" size="small" onClick={() => openDetail(s)}>{t('marketplace.card.details')}</Button>,
          ]}>
          <div style={{ marginBottom: 8 }}>
            <Text strong style={{ fontSize: 15 }}>{displayName}</Text>
            {s.priceModel && s.priceModel !== 'free' && <Tag color="gold" style={{ marginLeft: 6 }}>${s.priceAmount?.toFixed(2)}</Tag>}
          </div>
          <div style={{ marginBottom: 6, display: 'flex', alignItems: 'center', gap: 4 }}
            onClick={e => e.stopPropagation()}>
            <Rate allowHalf value={s.avgRating || 0} style={{ fontSize: 12 }}
              onChange={async (v) => {
                if (!userId) { message.warning(t('marketplace.messages.loginFirst')); return; }
                try {
                  const resp = await marketplaceClient.rateStrategy({ userId, strategyId: s.publishId, rating: v });
                  message.success(t('marketplace.messages.rated'));
                  refetch();
                } catch { message.error(t('marketplace.messages.rateFailed')); }
              }} />
            <span style={{ fontSize: 11, color: '#8c8c8c' }}>
              {s.ratingCount > 0 ? `(${s.ratingCount})` : ''}
            </span>
          </div>
          <Space size={4} wrap style={{ marginBottom: 8 }}>
            <Tag color="blue">{t(`marketplace.assetClass.${assetClass}`, { defaultValue: assetClass })}</Tag>
            <Tag color={RISK_COLORS[riskLevel] || 'default'} style={{ background: RISK_BG[riskLevel] || undefined, border: 'none' }}>
              {t(`marketplace.risk.${riskLevel}`, { defaultValue: riskLevel })}</Tag>
            {s.tags?.slice(0, 2).map(tag => <Tag key={tag}>{tag}</Tag>)}
          </Space>
          {s.description && <Paragraph ellipsis={{ rows: 2 }} style={{ fontSize: 12, color: '#5A6B75', marginBottom: 8 }}>{s.description}</Paragraph>}
          <div style={{ fontSize: 12, color: '#8A9AA5', marginBottom: 8 }}>
            <PieChartOutlined /> {t('marketplace.card.subscribers', { count: subscribers })}: {subscribers} &nbsp;
            <ExperimentOutlined /> {t('marketplace.card.winRate')}: {winRate}
            {s.totalPnl != null && <span style={{ marginLeft: 8, color: s.totalPnl >= 0 ? '#00A651' : '#E53935' }}>PnL: ${s.totalPnl.toFixed(0)}</span>}
          </div>
          <div style={{ fontSize: 11, color: '#B0BEC5' }}>
            {t('marketplace.card.by')} {s.publisherUserId ? s.publisherUserId.slice(0, 8) + '...' : 'unknown'}
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
                <div style={{ display: 'flex', gap: 12, marginBottom: 20, flexWrap: 'wrap' }}>
                  <Input prefix={<SearchOutlined />} placeholder={t('marketplace.searchPlaceholder')} value={searchText}
                    onChange={e => setSearchText(e.target.value)} style={{ maxWidth: 360, borderRadius: 8 }} allowClear />
                  <Select value={assetFilter || undefined} onChange={v => setAssetFilter(v || '')} allowClear
                    placeholder={t('marketplace.filterByClass')} style={{ minWidth: 180 }}
                    options={ASSET_CLASSES.map(c => ({ value: c, label: t(`marketplace.assetClass.${c}`, { defaultValue: c }) }))} />
                  <Select value={sortBy} onChange={v => setSortBy(v)} style={{ minWidth: 160 }}
                    options={[
                      { value: 'newest', label: t('marketplace.sort.newest') },
                      { value: 'popular', label: t('marketplace.sort.popular') },
                      { value: 'performance', label: t('marketplace.sort.performance') },
                    ]} />
                </div>
                <StatusResult loading={isLoading} error={error instanceof Error ? error.message : undefined} onRetry={refetch}
                  empty={strategies.length === 0 && !isLoading} emptyText={t('marketplace.empty')}>
                  <Row gutter={[16, 16]}>{strategies.map(renderStrategyCard)}</Row>
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

        {/* Detail Drawer */}
        <Drawer
          title={detailStrategy?.strategyName || detailStrategy?.title || detailStrategy?.strategyId?.slice(0, 8) || t('marketplace.card.details')}
          open={!!detailStrategy}
          onClose={() => setDetailStrategy(null)}
          width={480}
          styles={{ body: { paddingBottom: 80 } }}
        >
          {detailStrategy && (
            <>
              {detailStrategy.description && (
                <Paragraph style={{ color: '#5A6B75' }}>{detailStrategy.description}</Paragraph>
              )}
              <Space size={4} wrap style={{ marginBottom: 16 }}>
                <Tag color="blue">{t(`marketplace.assetClass.${detailStrategy.assetClass || 'forex'}`, { defaultValue: detailStrategy.assetClass || 'forex' })}</Tag>
                <Tag>{t(`marketplace.risk.${detailStrategy.riskLevel || 'medium'}`, { defaultValue: detailStrategy.riskLevel || 'medium' })}</Tag>
                {detailStrategy.tags?.map(tag => <Tag key={tag}>{tag}</Tag>)}
              </Space>
              <div style={{ marginBottom: 16 }}>
                <Rate allowHalf value={detailStrategy.avgRating || 0} style={{ fontSize: 14 }}
                  onChange={async (v) => {
                    if (!userId) { message.warning(t('marketplace.messages.loginFirst')); return; }
                    try {
                      await marketplaceClient.rateStrategy({ userId, strategyId: detailStrategy.publishId, rating: v });
                      message.success(t('marketplace.messages.rated'));
                      refetch();
                    } catch { message.error(t('marketplace.messages.rateFailed')); }
                  }} />
                <span style={{ fontSize: 11, color: '#8c8c8c', marginLeft: 8 }}>
                  {detailStrategy.ratingCount ? `(${detailStrategy.ratingCount})` : ''}
                </span>
              </div>

              <Divider />

              <Typography.Title level={5}>{t('marketplace.detail.comments')} ({comments.length})</Typography.Title>
              <List
                loading={commentsLoading}
                dataSource={comments}
                locale={{ emptyText: t('marketplace.detail.noComments') }}
                renderItem={(item: CommentItem) => (
                  <List.Item style={{ padding: '8px 0' }}>
                    <List.Item.Meta
                      title={<Text style={{ fontSize: 13 }}>{item.userName || item.userId?.slice(0, 8)}</Text>}
                      description={
                        <>
                          <Text style={{ fontSize: 13, color: '#1A2B3C' }}>{item.content}</Text>
                          <br />
                          <Text style={{ fontSize: 11, color: '#B0BEC5' }}>
                            {item.createdAt ? new Date(item.createdAt.seconds ? Number(item.createdAt.seconds) * 1000 : item.createdAt as any).toLocaleDateString() : ''}
                          </Text>
                        </>
                      }
                    />
                  </List.Item>
                )}
              />

              {userId && (
                <div style={{ marginTop: 16, display: 'flex', gap: 8 }}>
                  <Input.TextArea
                    rows={2}
                    value={commentText}
                    onChange={e => setCommentText(e.target.value)}
                    placeholder={t('marketplace.detail.commentPlaceholder')}
                    onPressEnter={e => {
                      if (!e.shiftKey) {
                        e.preventDefault();
                        submitComment();
                      }
                    }}
                  />
                  <Button
                    type="primary"
                    icon={<SendOutlined />}
                    onClick={submitComment}
                    loading={commentSubmitting}
                    disabled={!commentText.trim()}
                    style={{ alignSelf: 'flex-end' }}
                  />
                </div>
              )}
            </>
          )}
        </Drawer>
      </div>
    </div>
  );
}
