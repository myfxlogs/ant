import { useState, useCallback, useEffect } from 'react';
import { Modal, Descriptions, Tag, Button, Typography, Space, Divider, Rate, Tabs, Alert, Empty, message, Tooltip } from 'antd';
import { ShoppingCartOutlined, DownloadOutlined, ThunderboltOutlined, WarningOutlined, ExperimentOutlined, RocketOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { useStrategyDiscussion } from '../hooks/useStrategyDiscussion';
import { marketplaceClient } from '@/client/connect';
import { useAuthRequired } from '@/hooks/useAuthRequired';
import { useAuthStore } from '@/stores/authStore';
import type { PublishedStrategy } from '@/gen/ant/v1/marketplace_service_pb';
import LivePerformanceTab from './LivePerformanceTab';
import ShareButtons from './ShareButtons';
import { DecayBadge } from './DecayBadge';
import { strategyVersionApi } from '@/client/strategy';
import type { StrategyVersionInfo } from '@/gen/ant/v1/strategy_runtime_pb';
import { VersionHistoryTab, versionHistoryTabLabel } from './VersionHistoryTab';
import { DiscussionSection, priceText } from './StrategyDetailModalHelpers';
import DeployScheduleModal from '@/pages/strategy/components/DeployScheduleModal';

const { Text, Paragraph } = Typography;

interface Props {
  strategy: PublishedStrategy | null;
  open: boolean;
  isPurchased: boolean;
  isOwner: boolean;
  onClose: () => void;
  onGetFree: (s: PublishedStrategy) => void;
  onBuy: (s: PublishedStrategy) => void;
  onRunBacktest?: (s: PublishedStrategy) => void;
}

export default function StrategyDetailModal({ strategy, open, isPurchased, isOwner, onClose, onGetFree, onBuy, onRunBacktest }: Props) {
  const { t } = useTranslation();
  const d = useStrategyDiscussion();
  const requireAuth = useAuthRequired();
  const isAuthed = useAuthStore(s => !!s.accessToken);

  const [commentText, setCommentText] = useState('');
  const [commentSubmitting, setCommentSubmitting] = useState(false);
  const [perfTab, setPerfTab] = useState('backtest');
  const [trialLoading, setTrialLoading] = useState(false);
  const [versions, setVersions] = useState<StrategyVersionInfo[]>([]);
  const [versionsLoading, setVersionsLoading] = useState(false);
  const [deployOpen, setDeployOpen] = useState(false);

  // Load discussion data when a new strategy is opened
  useEffect(() => {
    if (strategy && open) {
      d.load(strategy.strategyId);
      if (isAuthed) {
        setVersionsLoading(true);
        strategyVersionApi.list(strategy.strategyId, 20, 0)
          .then(r => setVersions(r.versions || []))
          .catch(() => setVersions([]))
          .finally(() => setVersionsLoading(false));
      } else {
        setVersions([]);
      }
    }
  }, [strategy?.strategyId, open, isAuthed]); // eslint-disable-line react-hooks/exhaustive-deps

  const doComment = useCallback(async () => {
    const text = commentText.trim();
    if (!text || !strategy) return;
    setCommentSubmitting(true);
    try {
      await d.handleComment(strategy.strategyId, text);
      setCommentText('');
    } finally {
      setCommentSubmitting(false);
    }
  }, [commentText, strategy, d]);

  const handleStartTrial = useCallback(async () => {
    if (!strategy) return;
    if (!requireAuth()) return;
    setTrialLoading(true);
    try {
      const resp = await marketplaceClient.startTrial({ strategyId: strategy.strategyId });
      if (resp.alreadyTried) {
        message.info(t('marketplace.trial.alreadyTried'));
      } else {
        const expires = new Date(Number(resp.expiresAtMs));
        message.success(t('marketplace.trial.started') + ' ' + expires.toLocaleDateString());
      }
    } catch {
      message.error(t('marketplace.trial.failed'));
    } finally {
      setTrialLoading(false);
    }
  }, [strategy, t, requireAuth]);

  if (!strategy) return null;

  const name = strategy.strategyName || strategy.title || 'Unknown';
  const isFree = String(strategy.priceModel || '').toLowerCase() === 'free' || !Number(strategy.priceAmount);

  return (
    <Modal title={<span>{name} <DecayBadge decayStatus={strategy.decayStatus} /></span>}
      open={open} onCancel={onClose} width={720} footer={null} destroyOnClose>
      <DecayBadge decayStatus={strategy.decayStatus} showDescription />

      {/* Basic info */}
      <Descriptions column={2} size="small" bordered style={{ marginBottom: 16 }}>
        <Descriptions.Item label={t('marketplace.detail.author')}>{String(strategy.publisherUserId || '').slice(0, 12)}</Descriptions.Item>
        <Descriptions.Item label={t('marketplace.detail.price')}><Tag color={isFree ? 'green' : 'gold'}>{priceText(strategy, t)}</Tag></Descriptions.Item>
        <Descriptions.Item label={t('marketplace.detail.assetClass')}>{t(`marketplace.publish.assetClass.${strategy.assetClass}`)}</Descriptions.Item>
        <Descriptions.Item label={t('marketplace.detail.riskLevel')}>{t(`marketplace.publish.riskLevel.${strategy.riskLevel}`)}</Descriptions.Item>
        <Descriptions.Item label={t('marketplace.detail.subscribers')}>{String(strategy.totalSubscribers || 0)}</Descriptions.Item>
        <Descriptions.Item label={t('marketplace.detail.avgRating')}>
          <Rate disabled allowHalf value={d.ratingAvg || Number(strategy.avgRating || 0)} style={{ fontSize: 14 }} />
          <Text type="secondary" style={{ marginLeft: 4, fontSize: 12 }}>({d.ratingCount || strategy.ratingCount || 0})</Text>
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
          <Paragraph type="secondary" style={{ marginTop: 4, whiteSpace: 'pre-wrap' }}>{strategy.description}</Paragraph>
        </div>
      )}

      {/* Risk disclaimer */}
      {strategy.disclaimer && (
        <Alert type="warning" showIcon icon={<WarningOutlined />} style={{ marginBottom: 16 }}
          message={t('marketplace.detail.riskDisclaimer')} description={strategy.disclaimer} />
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

      <DiscussionSection
        t={t}
        strategyId={strategy.strategyId}
        comments={d.comments}
        commentsTotal={d.commentsTotal}
        commentsLoading={d.commentsLoading}
        ratings={d.ratings}
        ratingsLoading={d.ratingsLoading}
        userRating={d.userRating}
        onRate={d.handleRate}
        commentText={commentText}
        setCommentText={setCommentText}
        commentSubmitting={commentSubmitting}
        onComment={doComment}
      />

      {/* Performance tabs */}
      <Tabs activeKey={perfTab} onChange={setPerfTab} size="small" style={{ marginBottom: 16 }} items={[
        { key: 'backtest', label: t('marketplace.detail.backtestTab'), children: (
          <div>
            {strategy.backtestSnapshot ? (
              <Descriptions column={3} size="small">
                <Descriptions.Item label={t('marketplace.backtest.totalReturn')}>{strategy.backtestSnapshot.totalReturn || '-'}</Descriptions.Item>
                <Descriptions.Item label={t('marketplace.backtest.maxDrawdown')}>{strategy.backtestSnapshot.maxDrawdown || '-'}</Descriptions.Item>
                <Descriptions.Item label={t('marketplace.backtest.sharpe')}>{strategy.backtestSnapshot.sharpeRatio || '-'}</Descriptions.Item>
                <Descriptions.Item label={t('marketplace.backtest.winRate')}>{strategy.backtestSnapshot.winRate || '-'}</Descriptions.Item>
                <Descriptions.Item label={t('marketplace.backtest.totalTrades')}>{strategy.backtestSnapshot.totalTrades || '-'}</Descriptions.Item>
                <Descriptions.Item label={t('marketplace.backtest.symbol')}>{strategy.backtestSnapshot.symbol || '-'}</Descriptions.Item>
              </Descriptions>
            ) : (
              <Empty description={t('marketplace.detail.noBacktest')} />
            )}
          </div>
        )},
        { key: 'live', label: t('marketplace.detail.liveTab'), children: (
          <LivePerformanceTab strategyId={strategy.strategyId} isOwner={isOwner} />
        )},
        { key: 'versions', label: versionHistoryTabLabel(t), children: (
          <VersionHistoryTab versions={versions} versionsLoading={versionsLoading} isPurchased={isPurchased} />
        )},
      ]} />

      <Divider />

      {/* Action buttons */}
      <div style={{ textAlign: 'right', display: 'flex', gap: 8, justifyContent: 'space-between', alignItems: 'center' }}>
        <ShareButtons strategyId={strategy.strategyId} title={name} />
        <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
        {isOwner ? (
          <>
            <Tag color="blue" style={{ padding: '4px 16px', fontSize: 14 }}>{t('marketplace.card.yourStrategy')}</Tag>
            <Button type="primary" icon={<ThunderboltOutlined />} onClick={() => onRunBacktest?.(strategy)}>
              {t('marketplace.detail.runBacktest')}
            </Button>
          </>
        ) : isPurchased ? (
          <>
            <Tag color="green" style={{ padding: '4px 16px', fontSize: 14 }}>{t('marketplace.card.owned')}</Tag>
            <Button icon={<ThunderboltOutlined />} onClick={() => onRunBacktest?.(strategy)}>
              {t('marketplace.detail.runBacktest')}
            </Button>
            <Button type="primary" icon={<RocketOutlined />} onClick={() => setDeployOpen(true)}>
              {t('strategy.templates.actions.deploy', { defaultValue: 'Deploy' })}
            </Button>
          </>
        ) : isFree ? (
          <Button type="primary" icon={<DownloadOutlined />} size="large" onClick={() => onGetFree(strategy)} disabled={strategy.decayStatus === 'decayed'}>
            {t('marketplace.detail.getFree')}
          </Button>
        ) : (
          <>
            <Button icon={<ExperimentOutlined />} size="large" loading={trialLoading} onClick={handleStartTrial} disabled={strategy.decayStatus === 'decayed'}>
              {t('marketplace.trial.start')}
            </Button>
            <Tooltip title={strategy.decayStatus === 'decayed' ? t('marketplace.decay.descDecayed') : ''}>
              <Button type="primary" icon={<ShoppingCartOutlined />} size="large" onClick={() => onBuy(strategy)} disabled={strategy.decayStatus === 'decayed'}>
                {t('marketplace.detail.buyNow')}
              </Button>
            </Tooltip>
          </>
        )}
        </div>
      </div>

      <DeployScheduleModal
        open={deployOpen}
        templateId={strategy.strategyId}
        templateName={name}
        onClose={() => setDeployOpen(false)}
      />
    </Modal>
  );
}
