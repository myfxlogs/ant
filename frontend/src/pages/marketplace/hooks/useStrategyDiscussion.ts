import { useState, useCallback } from 'react';
import { message } from 'antd';
import { useTranslation } from 'react-i18next';
import { marketplaceClient } from '@/client/connect';
import { useAuthStore } from '@/stores/authStore';
import type { RatingItem, CommentItem } from '@/gen/ant/v1/marketplace_service_pb';

export interface StrategyDiscussion {
  // Ratings
  ratings: RatingItem[];
  ratingAvg: number;
  ratingCount: number;
  ratingsLoading: boolean;
  userRating: number;
  handleRate: (strategyId: string, rating: number) => Promise<void>;
  // Comments
  comments: CommentItem[];
  commentsTotal: number;
  commentsLoading: boolean;
  handleComment: (strategyId: string, content: string) => Promise<void>;
}

/** Per-strategy rating & comment state. Call `load()` when the strategy to discuss changes. */
export function useStrategyDiscussion(onLoaded?: (avg: number, count: number) => void): StrategyDiscussion & { load: (strategyId: string) => void } {
  const { t } = useTranslation();
  const { user } = useAuthStore();
  const userId = user?.id || '';

  const [ratings, setRatings] = useState<RatingItem[]>([]);
  const [ratingAvg, setRatingAvg] = useState(0);
  const [ratingCount, setRatingCount] = useState(0);
  const [ratingsLoading, setRatingsLoading] = useState(false);
  const [userRating, setUserRating] = useState(0);
  const [comments, setComments] = useState<CommentItem[]>([]);
  const [commentsTotal, setCommentsTotal] = useState(0);
  const [commentsLoading, setCommentsLoading] = useState(false);

  const load = useCallback(async (strategyId: string) => {
    // Reset for the new strategy
    setRatings([]); setRatingAvg(0); setRatingCount(0); setUserRating(0);
    setComments([]); setCommentsTotal(0);

    // Fetch ratings
    setRatingsLoading(true);
    try {
      const r = await marketplaceClient.listRatings({ strategyId });
      const items = (r.ratings || []) as RatingItem[];
      setRatings(items);
      const avg = r.avgRating ?? 0;
      const count = r.ratingCount ?? 0;
      setRatingAvg(avg);
      setRatingCount(count);
      setUserRating(items.find((x: RatingItem) => x.userId === userId)?.rating ?? 0);
      onLoaded?.(avg, count);
    } catch { /* non-critical */ }
    finally { setRatingsLoading(false); }

    // Fetch comments
    setCommentsLoading(true);
    try {
      const c = await marketplaceClient.listComments({ strategyId, limit: 20, offset: 0 });
      setComments((c.comments || []) as CommentItem[]);
      setCommentsTotal(c.total ?? 0);
    } catch { /* non-critical */ }
    finally { setCommentsLoading(false); }
  }, [userId, onLoaded]);

  const handleRate = useCallback(async (strategyId: string, rating: number) => {
    if (!userId) { message.warning(t('marketplace.messages.loginFirst')); return; }
    try {
      const resp = await marketplaceClient.rateStrategy({ userId, strategyId, rating });
      setRatingAvg(resp.avgRating ?? 0);
      setRatingCount(resp.ratingCount ?? 0);
      setUserRating(rating);
      message.success(t('marketplace.messages.rated'));
      // Re-fetch the full rating list so other users' ratings stay in sync
      const r = await marketplaceClient.listRatings({ strategyId });
      setRatings((r.ratings || []) as RatingItem[]);
    } catch { message.error(t('marketplace.messages.rateFailed')); }
  }, [userId, t]);

  const handleComment = useCallback(async (strategyId: string, content: string) => {
    if (!userId) { message.warning(t('marketplace.messages.loginFirst')); return; }
    try {
      await marketplaceClient.commentOnStrategy({ userId, strategyId, content });
      message.success(t('marketplace.messages.commentPosted'));
      const c = await marketplaceClient.listComments({ strategyId, limit: 20, offset: 0 });
      setComments((c.comments || []) as CommentItem[]);
      setCommentsTotal(c.total ?? 0);
    } catch { message.error(t('marketplace.messages.commentFailed')); }
  }, [userId, t]);

  return {
    ratings, ratingAvg, ratingCount, ratingsLoading, userRating, handleRate,
    comments, commentsTotal, commentsLoading, handleComment,
    load,
  };
}
