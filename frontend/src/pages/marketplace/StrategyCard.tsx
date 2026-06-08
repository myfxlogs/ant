// StrategyCard.tsx — Card component for a single published strategy in the marketplace grid.
// Extracted from Marketplace.tsx.

import { Card, Tag, Button, Space, Typography, Tooltip, Row, Col, Rate, message } from 'antd';
import { ExperimentOutlined, PieChartOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { marketplaceClient } from '@/client/connect';
import type { PublishedStrategy } from '@/gen/ant/v1/marketplace_service_pb';

const { Text, Paragraph } = Typography;
const RISK_COLORS: Record<string, string> = { low: '#00A651', medium: '#FF9800', high: '#E53935' };
const RISK_BG: Record<string, string> = { low: 'rgba(0,166,81,0.1)', medium: 'rgba(255,152,0,0.1)', high: 'rgba(229,57,53,0.1)' };

interface Props {
  strategy: PublishedStrategy;
  isSubscribed: boolean;
  subscriptionId?: string;
  userId: string;
  onSubscribe: (publisherUserId: string, strategyId: string) => void;
  onUnsubscribe: (subscriptionId: string) => void;
  onOpenDetail: (s: PublishedStrategy) => void;
  onRefresh: () => void;
}

export default function StrategyCard({ strategy: s, isSubscribed: isSub, subscriptionId, userId, onSubscribe, onUnsubscribe, onOpenDetail, onRefresh }: Props) {
  const { t } = useTranslation();
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
                if (subscriptionId) onUnsubscribe(subscriptionId);
              }} style={{ color: '#D4AF37' }}>{t('marketplace.card.subscribed')} ✓</Button>
            </Tooltip>
          ) : (
            <Button key="sub" type="link" size="small" onClick={() => onSubscribe(s.publisherUserId, s.strategyId)}>
              {t('marketplace.card.subscribe')}
            </Button>
          ),
          <Button key="detail" type="link" size="small" onClick={() => onOpenDetail(s)}>{t('marketplace.card.details')}</Button>,
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
                await marketplaceClient.rateStrategy({ userId, strategyId: s.publishId, rating: v });
                message.success(t('marketplace.messages.rated'));
                onRefresh();
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
}
