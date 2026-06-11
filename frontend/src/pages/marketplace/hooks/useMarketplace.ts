import { useState, useCallback, useMemo } from 'react';
import { message } from 'antd';
import { useTranslation } from 'react-i18next';
import { marketplaceClient } from '@/client/connect';
import { useRpcQuery } from '@/hooks/useRpcQuery';
import { useAuthStore } from '@/stores/authStore';
import type { PublishedStrategy } from '@/gen/ant/v1/marketplace_service_pb';
import type { MarketplaceCtx } from '../MarketplaceContext';

export type PriceFilter = 'all' | 'free' | 'paid';
export type SortBy = 'score' | 'newest' | 'popular' | 'rating' | 'price_asc' | 'price_desc';
export type TabKey = 'market' | 'purchases' | 'author';

export function useMarketplace(): MarketplaceCtx {
  const { t } = useTranslation();
  const { user } = useAuthStore();
  const userId = user?.id || '';

  const [activeTab, setActiveTab] = useState<TabKey>('market');
  const [searchText, setSearchText] = useState('');
  const [priceFilter, setPriceFilter] = useState<PriceFilter>('all');
  const [sortBy, setSortBy] = useState<SortBy>('score');
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [detailStrategy, setDetailStrategy] = useState<PublishedStrategy | null>(null);
  const [detailOpen, setDetailOpen] = useState(false);

  // ── Market listing (server-side pagination via API) ──
  const { data: allStrategies = [], isLoading: loading, error, refetch } = useRpcQuery(
    ['marketplace', 'published', userId, searchText, sortBy, page, pageSize],
    async () => {
      const resp = await marketplaceClient.listPublished({
        userId, limit: pageSize, offset: (page - 1) * pageSize,
        keyword: searchText || undefined,
        sortBy: sortBy || undefined,
      });
      return (resp.strategies || []) as PublishedStrategy[];
    },
  );

  // Client-side price filter + cache-warmed list
  const strategies = useMemo(() => {
    let list = allStrategies;
    if (priceFilter === 'free') list = list.filter(s => !s.priceAmount || s.priceAmount === 0);
    if (priceFilter === 'paid') list = list.filter(s => s.priceAmount && s.priceAmount > 0);
    return list;
  }, [allStrategies, priceFilter]);

  // Estimate total — API returns at most pageSize items
  const total = allStrategies.length < pageSize ? (page - 1) * pageSize + allStrategies.length : page * pageSize + 1;

  // ── Purchases ──
  const { data: purchases = [], isLoading: purchasesLoading, refetch: refetchPurchases } = useRpcQuery(
    ['marketplace', 'purchases', userId],
    async () => {
      if (!userId) return [];
      const resp = await marketplaceClient.listSubscriptions({ userId });
      return (resp.subscriptions || []).map((s: any) => ({
        ...s,
        id: s.subscriptionId,
        strategyId: s.strategyId,
        purchasedAt: s.createdAt,
      }));
    },
    { enabled: !!userId },
  );

  // ── Author: my published strategies ──
  const myPublished = useMemo(() => {
    if (!userId) return [];
    return allStrategies.filter((s: any) => s.publisherUserId === userId);
  }, [allStrategies, userId]);

  const authorStats = useMemo(() => ({
    published: myPublished.length,
    totalSubscribers: myPublished.reduce((sum: number, s: any) => sum + (s.totalSubscribers || 0), 0),
    avgRating: myPublished.length > 0
      ? myPublished.reduce((sum: number, s: any) => sum + (s.avgRating || 0), 0) / myPublished.length
      : 0,
  }), [myPublished]);

  // ── Detail ──
  const openDetail = useCallback((s: PublishedStrategy) => {
    setDetailStrategy(s); setDetailOpen(true);
  }, []);
  const closeDetail = useCallback(() => {
    setDetailStrategy(null); setDetailOpen(false);
  }, []);

  // ── Purchase / Get ──
  const isPurchased = useCallback((strategyId: string) =>
    purchases.some((p: any) => p.strategyId === strategyId),
  [purchases]);

  const handleGetFree = useCallback(async (strategy: PublishedStrategy) => {
    if (!userId) { message.warning(t('marketplace.messages.loginFirst')); return; }
    try {
      await marketplaceClient.subscribe({
        userId,
        publisherUserId: strategy.publisherUserId,
        strategyId: strategy.strategyId,
        kind: 'copy_trade',
      });
      message.success(t('marketplace.messages.subscribed'));
      refetchPurchases();
    } catch { message.error(t('marketplace.messages.subscribeFailed')); }
  }, [userId, t, refetchPurchases]);

  const handleBuy = useCallback(async (_strategy: PublishedStrategy) => {
    message.info(t('marketplace.messages.paymentComingSoon', 'Payment coming soon'));
  }, [t]);

  return {
    strategies, loading, error, purchases, purchasesLoading, myPublished, authorStats,
    activeTab, setActiveTab, searchText, setSearchText,
    priceFilter, setPriceFilter, sortBy, setSortBy,
    page, pageSize, total, setPage, setPageSize,
    refetch, isPurchased, handleGetFree, handleBuy,
    openDetail, closeDetail, detailStrategy, detailOpen,
  };
}
