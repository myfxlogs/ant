// StrategyDetailDrawer.tsx — Detail drawer for a published strategy with rating, comments, and reply.
// Extracted from Marketplace.tsx.

import { useState, useCallback } from 'react';
import { Drawer, Space, Tag, Rate, Typography, Divider, List, Input, Button, message } from 'antd';
import { SendOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { marketplaceClient } from '@/client/connect';
import type { PublishedStrategy, CommentItem } from '@/gen/ant/v1/marketplace_service_pb';

const { Text, Paragraph } = Typography;

interface Props {
  strategy: PublishedStrategy | null;
  userId: string;
  isMobile: boolean;
  onClose: () => void;
  onRefresh: () => void;
}

export default function StrategyDetailDrawer({ strategy: detailStrategy, userId, isMobile, onClose, onRefresh }: Props) {
  const { t } = useTranslation();
  const [comments, setComments] = useState<CommentItem[]>([]);
  const [commentsLoading, setCommentsLoading] = useState(false);
  const [commentText, setCommentText] = useState('');
  const [commentSubmitting, setCommentSubmitting] = useState(false);

  const loadComments = useCallback(async (s: PublishedStrategy) => {
    setComments([]);
    setCommentText('');
    setCommentsLoading(true);
    try {
      const resp = await marketplaceClient.listComments({ strategyId: s.publishId, limit: 50, offset: 0 });
      setComments((resp.comments || []) as CommentItem[]);
    } catch { /* non-critical */ }
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
      const resp = await marketplaceClient.listComments({ strategyId: detailStrategy.publishId, limit: 50, offset: 0 });
      setComments((resp.comments || []) as CommentItem[]);
    } catch { message.error(t('marketplace.messages.commentFailed')); }
    setCommentSubmitting(false);
  }, [commentText, detailStrategy, userId, t]);

  return (
    <Drawer
      title={detailStrategy?.strategyName || detailStrategy?.title || detailStrategy?.strategyId?.slice(0, 8) || t('marketplace.card.details')}
      open={!!detailStrategy}
      onClose={onClose}
      afterOpenChange={(open) => { if (open && detailStrategy) loadComments(detailStrategy); }}
      width={isMobile ? '100%' : 480}
      styles={{ body: { paddingBottom: 80 } }}
    >
      {detailStrategy && (
        <>
          {detailStrategy.description && (
            <Paragraph style={{ color: 'var(--color-text-secondary)' }}>{detailStrategy.description}</Paragraph>
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
                  onRefresh();
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
  );
}
