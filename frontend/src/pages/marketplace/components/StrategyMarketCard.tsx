import { Card, Tag, Typography, Rate, Space } from 'antd';
import { CrownOutlined, StarFilled } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import type { PublishedStrategy } from '@/gen/ant/v1/marketplace_service_pb';

const { Text, Title } = Typography;

interface Props {
  strategy: PublishedStrategy;
  isPurchased: boolean;
  onOpenDetail: (s: PublishedStrategy) => void;
  onGetFree: (s: PublishedStrategy) => void;
}

function priceLabel(s: PublishedStrategy, t: (k: string) => string): { text: string; color: string } {
  const model = String(s.priceModel || '').toLowerCase();
  const amount = Number(s.priceAmount || 0);
  if (model === 'free' || !amount) return { text: t('marketplace.card.free', '免费'), color: '#52c41a' };
  if (model === 'subscription') return { text: t('marketplace.card.rent', '¥{{amount}}/月', { amount: amount.toFixed(0) }), color: '#fa8c16' };
  return { text: t('marketplace.card.buy', '¥{{amount}}', { amount: amount.toFixed(0) }), color: '#D4AF37' };
}

export default function StrategyMarketCard({ strategy, isPurchased, onOpenDetail, onGetFree }: Props) {
  const { t } = useTranslation();
  const price = priceLabel(strategy, t);
  const name = strategy.strategyName || strategy.title || 'Unknown';
  const rating = Number(strategy.avgRating || 0);
  const ratingCount = Number(strategy.ratingCount || 0);
  const subscribers = Number(strategy.totalSubscribers || 0);
  const winRate = strategy.winRate != null ? (strategy.winRate * 100).toFixed(0) : null;
  const pnl = strategy.totalPnl != null ? Number(strategy.totalPnl).toFixed(0) : null;

  return (
    <Card
      hoverable
      size="small"
      style={{ borderRadius: 12, height: '100%' }}
      onClick={() => onOpenDetail(strategy)}
      extra={
        <Tag color={price.color} style={{ margin: 0, fontWeight: 600, fontSize: 12 }}>
          {price.text}
        </Tag>
      }
    >
      {/* Name */}
      <div style={{ marginBottom: 8 }}>
        <Text strong ellipsis style={{ fontSize: 14, maxWidth: '100%', display: 'block' }}>
          {name}
        </Text>
      </div>

      {/* Author */}
      <Text type="secondary" style={{ fontSize: 12 }}>
        {String(strategy.publisherUserId || '').slice(0, 8)}
      </Text>

      {/* Rating */}
      {rating > 0 && (
        <div style={{ marginTop: 8 }}>
          <Rate disabled value={rating} allowHalf style={{ fontSize: 14 }} />
          <Text style={{ fontSize: 12, marginLeft: 6, color: '#8c8c8c' }}>
            {rating.toFixed(1)} ({ratingCount})
          </Text>
        </div>
      )}

      {/* KPIs */}
      <div style={{ display: 'flex', gap: 12, marginTop: 10, flexWrap: 'wrap' }}>
        {winRate != null && (
          <div>
            <Text style={{ fontSize: 10, color: '#8c8c8c' }}>{t('marketplace.card.winRate')}</Text>
            <div><Text style={{ fontSize: 13, fontWeight: 600, color: '#00A651' }}>{winRate}%</Text></div>
          </div>
        )}
        {pnl != null && (
          <div>
            <Text style={{ fontSize: 10, color: '#8c8c8c' }}>{t('marketplace.card.pnl')}</Text>
            <div><Text style={{ fontSize: 13, fontWeight: 600 }}>{Number(pnl) >= 0 ? '+' : ''}{pnl}</Text></div>
          </div>
        )}
        <div>
          <Text style={{ fontSize: 10, color: '#8c8c8c' }}>{t('marketplace.card.users')}</Text>
          <div><Text style={{ fontSize: 13, fontWeight: 600 }}>{subscribers}</Text></div>
        </div>
      </div>

      {/* Tags */}
      <div style={{ marginTop: 10 }}>
        <Space size={4} wrap>
          {strategy.assetClass && <Tag style={{ fontSize: 10 }}>{String(strategy.assetClass)}</Tag>}
          {strategy.riskLevel && <Tag style={{ fontSize: 10 }}>{String(strategy.riskLevel)}</Tag>}
          {isPurchased && <Tag color="green" style={{ fontSize: 10 }}>{t('marketplace.card.owned')}</Tag>}
        </Space>
      </div>
    </Card>
  );
}
