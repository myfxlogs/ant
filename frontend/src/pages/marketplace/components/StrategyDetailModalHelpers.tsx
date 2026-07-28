import React from 'react';
import { Divider, Typography, Rate, Spin, List, Input, Button } from 'antd';
import { UserOutlined } from '@ant-design/icons';
import type { TFunction } from 'i18next';

const { Text, Paragraph } = Typography;
const { TextArea } = Input;

function fmtTime(ts: { seconds?: bigint | number } | undefined | null): string {
  if (!ts) return '';
  const s = typeof ts.seconds === 'bigint' ? Number(ts.seconds) : (ts.seconds ?? 0);
  if (!s) return '';
  return new Date(s * 1000).toLocaleString();
}

interface DiscussionSectionProps {
  t: TFunction;
  strategyId: string;
  comments: Array<{ id: string; userId?: string; userName?: string; content: string; createdAt?: { seconds?: bigint | number } }>;
  commentsTotal: number;
  commentsLoading: boolean;
  ratings: Array<{ id: string; userId?: string; rating: number; createdAt?: { seconds?: bigint | number } }>;
  ratingsLoading: boolean;
  userRating: number;
  onRate: (strategyId: string, val: number) => void;
  commentText: string;
  setCommentText: (v: string) => void;
  commentSubmitting: boolean;
  onComment: () => void;
}

export function DiscussionSection({
  t, strategyId,
  comments, commentsTotal, commentsLoading,
  ratings, ratingsLoading,
  userRating,
  onRate,
  commentText, setCommentText, commentSubmitting, onComment,
}: DiscussionSectionProps) {
  return (
    <>
      <Divider />

      {/* ── Rating section ── */}
      <div style={{ marginBottom: 16 }}>
        <Text strong>{t('marketplace.detail.yourRating')}</Text>
        <div style={{ marginTop: 4 }}>
          <Rate
            value={userRating}
            onChange={(val) => onRate(strategyId, val)}
            style={{ fontSize: 20 }}
          />
          {userRating > 0 && (
            <Text type="secondary" style={{ marginLeft: 8 }}>
              {t('marketplace.messages.rated')}
            </Text>
          )}
        </div>
        <Spin spinning={ratingsLoading} size="small">
          {ratings.length > 0 && (
            <div style={{ marginTop: 8, maxHeight: 120, overflowY: 'auto' }}>
              {ratings.map((r) => (
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
        <Text strong>{t('marketplace.detail.comments')} ({commentsTotal})</Text>

        <Spin spinning={commentsLoading}>
          {comments.length === 0 && !commentsLoading ? (
            <Paragraph type="secondary" style={{ marginTop: 8, textAlign: 'center' }}>
              {t('marketplace.detail.noComments')}
            </Paragraph>
          ) : (
            <List
              style={{ marginTop: 8 }}
              dataSource={comments}
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

        <div style={{ marginTop: 12, display: 'flex', gap: 8 }}>
          <TextArea
            rows={2}
            value={commentText}
            onChange={(e) => setCommentText(e.target.value)}
            placeholder={t('marketplace.detail.commentPlaceholder')}
            onKeyDown={(e) => {
              if (e.key === 'Enter' && !e.shiftKey) {
                e.preventDefault();
                onComment();
              }
            }}
            style={{ flex: 1 }}
          />
          <Button
            type="primary"
            loading={commentSubmitting}
            disabled={!commentText.trim()}
            onClick={onComment}
            style={{ alignSelf: 'flex-end' }}
          >
            {t('marketplace.detail.comments')}
          </Button>
        </div>
      </div>
    </>
  );
}

export function priceText(s: { priceModel?: string; priceAmount?: string | number }, t: (k: string, opts?: Record<string, unknown>) => string): string {
  const model = String(s.priceModel || '').toLowerCase();
  const amount = Number(s.priceAmount || 0);
  if (model === 'free' || !amount) return t('marketplace.card.free');
  if (model === 'subscription') return t('marketplace.detail.rentPrice', { amount: amount.toFixed(0) });
  return t('marketplace.detail.buyPrice', { amount: amount.toFixed(0) });
}

export { fmtTime };
