import { Modal, Descriptions, Tag, Button, Typography, Space, Divider, message } from 'antd';
import { ShoppingCartOutlined, DownloadOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import type { PublishedStrategy } from '@/gen/ant/v1/marketplace_service_pb';

const { Text, Paragraph } = Typography;

interface Props {
  strategy: PublishedStrategy | null;
  open: boolean;
  isPurchased: boolean;
  onClose: () => void;
  onGetFree: (s: PublishedStrategy) => void;
  onBuy: (s: PublishedStrategy) => void;
}

function priceText(s: PublishedStrategy, t: (k: string) => string): string {
  const model = String(s.priceModel || '').toLowerCase();
  const amount = Number(s.priceAmount || 0);
  if (model === 'free' || !amount) return t('marketplace.card.free', '免费');
  if (model === 'subscription') return t('marketplace.detail.rentPrice', '¥{{amount}} / 月', { amount: amount.toFixed(0) });
  return t('marketplace.detail.buyPrice', '¥{{amount}} 买断', { amount: amount.toFixed(0) });
}

export default function StrategyDetailModal({ strategy, open, isPurchased, onClose, onGetFree, onBuy }: Props) {
  const { t } = useTranslation();
  if (!strategy) return null;

  const name = strategy.strategyName || strategy.title || 'Unknown';
  const isFree = String(strategy.priceModel || '').toLowerCase() === 'free' || !Number(strategy.priceAmount);

  return (
    <Modal
      title={name}
      open={open}
      onCancel={onClose}
      width={720}
      footer={null}
      destroyOnClose
    >
      {/* Basic info */}
      <Descriptions column={2} size="small" bordered style={{ marginBottom: 16 }}>
        <Descriptions.Item label={t('marketplace.detail.author')}>
          {String(strategy.publisherUserId || '').slice(0, 12)}
        </Descriptions.Item>
        <Descriptions.Item label={t('marketplace.detail.price')}>
          <Tag color={isFree ? 'green' : 'gold'}>{priceText(strategy, t)}</Tag>
        </Descriptions.Item>
        <Descriptions.Item label={t('marketplace.detail.assetClass')}>
          {String(strategy.assetClass || '-')}
        </Descriptions.Item>
        <Descriptions.Item label={t('marketplace.detail.riskLevel')}>
          {String(strategy.riskLevel || '-')}
        </Descriptions.Item>
        <Descriptions.Item label={t('marketplace.detail.subscribers')}>
          {String(strategy.totalSubscribers || 0)}
        </Descriptions.Item>
        <Descriptions.Item label={t('marketplace.detail.avgRating')}>
          {Number(strategy.avgRating || 0).toFixed(1)} ({strategy.ratingCount || 0})
        </Descriptions.Item>
      </Descriptions>

      {/* Performance KPIs */}
      <div style={{ display: 'flex', gap: 16, marginBottom: 16, flexWrap: 'wrap' }}>
        {strategy.winRate != null && (
          <div style={{ flex: 1, minWidth: 100, textAlign: 'center', background: '#f6ffed', padding: '8px 12px', borderRadius: 8 }}>
            <Text type="secondary" style={{ fontSize: 11 }}>{t('marketplace.card.winRate')}</Text>
            <div><Text strong style={{ fontSize: 18, color: '#52c41a' }}>{(strategy.winRate * 100).toFixed(0)}%</Text></div>
          </div>
        )}
        {strategy.totalPnl != null && (
          <div style={{ flex: 1, minWidth: 100, textAlign: 'center', background: '#fff7e6', padding: '8px 12px', borderRadius: 8 }}>
            <Text type="secondary" style={{ fontSize: 11 }}>{t('marketplace.card.pnl')}</Text>
            <div><Text strong style={{ fontSize: 18 }}>{Number(strategy.totalPnl).toFixed(0)}</Text></div>
          </div>
        )}
      </div>

      {/* Description */}
      {strategy.description && (
        <div style={{ marginBottom: 16 }}>
          <Text strong>{t('marketplace.detail.description')}</Text>
          <Paragraph type="secondary" style={{ marginTop: 4, whiteSpace: 'pre-wrap' }}>
            {strategy.description}
          </Paragraph>
        </div>
      )}

      {/* Tags */}
      {(Array.isArray(strategy.tags) && (strategy.tags as string[]).length > 0) && (
        <div style={{ marginBottom: 16 }}>
          <Text strong style={{ marginBottom: 4, display: 'block' }}>{t('marketplace.detail.tags')}</Text>
          <Space wrap>
            {(strategy.tags as string[]).map(tag => (
              <Tag key={tag}>{tag}</Tag>
            ))}
          </Space>
        </div>
      )}

      <Divider />

      {/* Action buttons */}
      <div style={{ textAlign: 'right' }}>
        {isPurchased ? (
          <Tag color="green" style={{ padding: '4px 16px', fontSize: 14 }}>{t('marketplace.card.owned')}</Tag>
        ) : isFree ? (
          <Button type="primary" icon={<DownloadOutlined />} size="large" onClick={() => onGetFree(strategy)}>
            {t('marketplace.detail.getFree')}
          </Button>
        ) : (
          <Button type="primary" icon={<ShoppingCartOutlined />} size="large" onClick={() => onBuy(strategy)}>
            {t('marketplace.detail.buyNow')}
          </Button>
        )}
      </div>
    </Modal>
  );
}
