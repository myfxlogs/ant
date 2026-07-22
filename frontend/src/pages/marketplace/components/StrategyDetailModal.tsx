import { useState, useCallback, useEffect } from 'react';
import { Modal, Descriptions, Tag, Button, Typography, Space, Divider, Input, List, Spin, Rate, Tabs, Alert, Empty, message } from 'antd';
import { ShoppingCartOutlined, DownloadOutlined, UserOutlined, ThunderboltOutlined, WarningOutlined, ExperimentOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { useStrategyDiscussion } from '../hooks/useStrategyDiscussion';
import { marketplaceClient } from '@/client/connect';
import { useAuthRequired } from '@/hooks/useAuthRequired';
import { useAuthStore } from '@/stores/authStore';
import type { PublishedStrategy } from '@/gen/ant/v1/marketplace_service_pb';
import LivePerformanceTab from './LivePerformanceTab';
import ShareButtons from './ShareButtons';
import { strategyVersionApi } from '@/client/strategy';
import type { StrategyVersionInfo } from '@/gen/ant/v1/strategy_runtime_pb';
import { VersionHistoryTab, versionHistoryTabLabel } from './VersionHistoryTab';

const { Text, Paragraph } = Typography;
const { TextArea } = Input;

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

function priceText(s: PublishedStrategy, t: (k: string, opts?: Record<string,unknown>) => string): string {
  const model = String(s.priceModel || '').toLowerCase();
  const amount = Number(s.priceAmount || 0);
  if (model === 'free' || !amount) return t('marketplace.card.free', { defaultValue: 'Free' });
  if (model === 'subscription') return t('marketplace.detail.rentPrice', { amount: amount.toFixed(0), defaultValue: `¥${amount.toFixed(0)} / month` });
  return t('marketplace.detail.buyPrice', { amount: amount.toFixed(0), defaultValue: `¥${amount.toFixed(0)} one-time` });
}

function fmtTime(ts: { seconds?: bigint | number } | undefined | null): string {
  if (!ts) return '';
  const s = typeof ts.seconds === 'bigint' ? Number(ts.seconds) : (ts.seconds ?? 0);
  if (!s) return '';
  return new Date(s * 1000).toLocaleString();
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
        message.info(t('marketplace.trial.alreadyTried', { defaultValue: 'You have already used your free trial for this strategy.' }));
      } else {
        const expires = new Date(Number(resp.expiresAtMs));
        message.success(t('marketplace.trial.started', { defaultValue: 'Free trial started! Expires' }) + ' ' + expires.toLocaleDateString());
      }
    } catch {
      message.error(t('marketplace.trial.failed', { defaultValue: 'Failed to start trial' }));
    } finally {
      setTrialLoading(false);
    }
  }, [strategy, t, requireAuth]);

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
          {t(`marketplace.publish.assetClass.${strategy.assetClass}`, { defaultValue: strategy.assetClass || '-' })}
        </Descriptions.Item>
        <Descriptions.Item label={t('marketplace.detail.riskLevel')}>
          {t(`marketplace.publish.riskLevel.${strategy.riskLevel}`, { defaultValue: strategy.riskLevel || '-' })}
        </Descriptions.Item>
        <Descriptions.Item label={t('marketplace.detail.subscribers')}>
          {String(strategy.totalSubscribers || 0)}
        </Descriptions.Item>
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
          <Paragraph type="secondary" style={{ marginTop: 4, whiteSpace: 'pre-wrap' }}>
            {strategy.description}
          </Paragraph>
        </div>
      )}

      {/* Risk disclaimer */}
      {strategy.disclaimer && (
        <Alert
          type="warning"
          showIcon
          icon={<WarningOutlined />}
          style={{ marginBottom: 16 }}
          message={t('marketplace.detail.riskDisclaimer', { defaultValue: 'Risk Disclaimer' })}
          description={strategy.disclaimer}
        />
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

      {/* ── Rating section ── */}
      <div style={{ marginBottom: 16 }}>
        <Text strong>{t('marketplace.detail.yourRating')}</Text>
        <div style={{ marginTop: 4 }}>
          <Rate
            value={d.userRating}
            onChange={(val) => d.handleRate(strategy.strategyId, val)}
            style={{ fontSize: 20 }}
          />
          {d.userRating > 0 && (
            <Text type="secondary" style={{ marginLeft: 8 }}>
              {t('marketplace.messages.rated')}
            </Text>
          )}
        </div>
        <Spin spinning={d.ratingsLoading} size="small">
          {d.ratings.length > 0 && (
            <div style={{ marginTop: 8, maxHeight: 120, overflowY: 'auto' }}>
              {d.ratings.map((r) => (
                <div key={r.id} style={{ display: 'flex', alignItems: 'center', gap: 6, padding: '2px 0' }}>
                  <UserOutlined style={{ fontSize: 11, color: '#8c8c8c' }} />
                  <Text style={{ fontSize: 11, color: '#8c8c8c' }}>{String(r.userId || '').slice(0, 8)}</Text>
                  <Rate disabled value={r.rating} style={{ fontSize: 12 }} />
                  <Text type="secondary" style={{ fontSize: 10 }}>{fmtTime(r.createdAt)}</Text>
                </div>
              ))}
            </div>
          )}
        </Spin>
      </div>

      <Divider />

      {/* ── Comments section ── */}
      <div style={{ marginBottom: 16 }}>
        <Text strong>{t('marketplace.detail.comments')} ({d.commentsTotal})</Text>

        {/* Comment list */}
        <Spin spinning={d.commentsLoading}>
          {d.comments.length === 0 && !d.commentsLoading ? (
            <Paragraph type="secondary" style={{ marginTop: 8, textAlign: 'center' }}>
              {t('marketplace.detail.noComments')}
            </Paragraph>
          ) : (
            <List
              style={{ marginTop: 8 }}
              dataSource={d.comments}
              size="small"
              split={false}
              renderItem={(c) => (
                <List.Item style={{ padding: '6px 0', borderBottom: '1px solid #f0f0f0' }}>
                  <div style={{ width: '100%' }}>
                    <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 2 }}>
                      <Text strong style={{ fontSize: 12 }}>
                        <UserOutlined style={{ marginRight: 4 }} />{c.userName || String(c.userId || '').slice(0, 8)}
                      </Text>
                      <Text type="secondary" style={{ fontSize: 10 }}>{fmtTime(c.createdAt)}</Text>
                    </div>
                    <Text style={{ fontSize: 13, whiteSpace: 'pre-wrap' }}>{c.content}</Text>
                  </div>
                </List.Item>
              )}
            />
          )}
        </Spin>

        {/* Comment input */}
        <div style={{ marginTop: 12, display: 'flex', gap: 8 }}>
          <TextArea
            rows={2}
            value={commentText}
            onChange={(e) => setCommentText(e.target.value)}
            placeholder={t('marketplace.detail.commentPlaceholder')}
            onKeyDown={(e) => {
              if (e.key === 'Enter' && !e.shiftKey) {
                e.preventDefault();
                doComment();
              }
            }}
            style={{ flex: 1 }}
          />
          <Button
            type="primary"
            loading={commentSubmitting}
            disabled={!commentText.trim()}
            onClick={doComment}
            style={{ alignSelf: 'flex-end' }}
          >
            {t('marketplace.detail.comments')}
          </Button>
        </div>
      </div>

      <Divider />

      {/* Performance tabs */}
      <Tabs activeKey={perfTab} onChange={setPerfTab} size="small" style={{ marginBottom: 16 }} items={[
        { key: 'backtest', label: t('marketplace.detail.backtestTab', { defaultValue: 'Backtest' }), children: (
          <div>
            {strategy.backtestSnapshot ? (
              <Descriptions column={3} size="small">
                <Descriptions.Item label={t('marketplace.backtest.totalReturn', { defaultValue: 'Total Return' })}>{strategy.backtestSnapshot.totalReturn || '-'}</Descriptions.Item>
                <Descriptions.Item label={t('marketplace.backtest.maxDrawdown', { defaultValue: 'Max Drawdown' })}>{strategy.backtestSnapshot.maxDrawdown || '-'}</Descriptions.Item>
                <Descriptions.Item label={t('marketplace.backtest.sharpe', { defaultValue: 'Sharpe' })}>{strategy.backtestSnapshot.sharpeRatio || '-'}</Descriptions.Item>
                <Descriptions.Item label={t('marketplace.backtest.winRate', { defaultValue: 'Win Rate' })}>{strategy.backtestSnapshot.winRate || '-'}</Descriptions.Item>
                <Descriptions.Item label={t('marketplace.backtest.totalTrades', { defaultValue: 'Total Trades' })}>{strategy.backtestSnapshot.totalTrades || '-'}</Descriptions.Item>
                <Descriptions.Item label={t('marketplace.backtest.symbol', { defaultValue: 'Symbol' })}>{strategy.backtestSnapshot.symbol || '-'}</Descriptions.Item>
              </Descriptions>
            ) : (
              <Empty description={t('marketplace.detail.noBacktest', { defaultValue: 'No backtest snapshot available' })} />
            )}
          </div>
        )},
        { key: 'live', label: t('marketplace.detail.liveTab', { defaultValue: 'Live Performance' }), children: (
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
            <Tag color="blue" style={{ padding: '4px 16px', fontSize: 14 }}>{t('marketplace.card.yourStrategy', 'Your Strategy')}</Tag>
            <Button type="primary" icon={<ThunderboltOutlined />} onClick={() => onRunBacktest?.(strategy)}>
              {t('marketplace.detail.runBacktest', 'Run Backtest')}
            </Button>
          </>
        ) : isPurchased ? (
          <>
            <Tag color="green" style={{ padding: '4px 16px', fontSize: 14 }}>{t('marketplace.card.owned')}</Tag>
            <Button type="primary" icon={<ThunderboltOutlined />} onClick={() => onRunBacktest?.(strategy)}>
              {t('marketplace.detail.runBacktest', 'Run Backtest')}
            </Button>
          </>
        ) : isFree ? (
          <Button type="primary" icon={<DownloadOutlined />} size="large" onClick={() => onGetFree(strategy)}>
            {t('marketplace.detail.getFree')}
          </Button>
        ) : (
          <>
            <Button icon={<ExperimentOutlined />} size="large" loading={trialLoading} onClick={handleStartTrial}>
              {t('marketplace.trial.start', { defaultValue: 'Free Trial' })}
            </Button>
            <Button type="primary" icon={<ShoppingCartOutlined />} size="large" onClick={() => onBuy(strategy)}>
              {t('marketplace.detail.buyNow')}
            </Button>
          </>
        )}
        </div>
      </div>
    </Modal>
  );
}
