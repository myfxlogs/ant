import { useState, useCallback, useMemo } from 'react';
import { message } from 'antd';
import { useTranslation } from 'react-i18next';
import { marketplaceClient } from '@/client/connect';
import { walletApi } from '@/client/wallet';
import { useRpcQuery } from '@/hooks/useRpcQuery';
import { useAuthRequired } from '@/hooks/useAuthRequired';
import { useAuthStore } from '@/stores/authStore';
import type { PublishedStrategy, SubscriptionItem } from '@/gen/ant/v1/marketplace_service_pb';
import type { MarketplaceCtx } from '../MarketplaceContext';

export interface PurchasedItem extends SubscriptionItem {
  /** Alias for subscriptionId */
  id: string;
  /** Alias for createdAt */
  purchasedAt: SubscriptionItem['createdAt'];
}

export type PriceFilter = 'all' | 'free' | 'paid';
export type SortBy = 'score' | 'newest' | 'popular' | 'rating' | 'price_asc' | 'price_desc';
export type TabKey = 'market' | 'leaderboard' | 'purchases' | 'author' | 'bundles' | 'optimization' | 'fees';

export function useMarketplace(): Omit<MarketplaceCtx, 'compareIds' | 'toggleCompare'> {
  const { t } = useTranslation();
  const { user, isAuthenticated } = useAuthStore();
  const userId = user?.id || '';
  const requireAuth = useAuthRequired();

  const [activeTab, setActiveTab] = useState<TabKey>('market');
  const [searchText, setSearchText] = useState('');
  const [priceFilter, setPriceFilter] = useState<PriceFilter>('all');
  const [sortBy, setSortBy] = useState<SortBy>('score');
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [detailStrategy, setDetailStrategy] = useState<PublishedStrategy | null>(null);
  const [detailOpen, setDetailOpen] = useState(false);

  // ── Payment state ──
  const [paymentModalOpen, setPaymentModalOpen] = useState(false);
  const [paymentLoading, setPaymentLoading] = useState(false);
  const [paymentStrategy, setPaymentStrategy] = useState<PublishedStrategy | null>(null);
  const [walletBalance, setWalletBalance] = useState('0');

  // ── Market listing (server-side pagination via API) ──
  // NOTE: userId is NOT passed — the market shows ALL published strategies.
  // Author tab filters client-side via myPublished (by publisherUserId).
  const { data: allStrategies = [], isLoading: loading, error, refetch } = useRpcQuery(
    ['marketplace', 'published', searchText, sortBy, page, pageSize],
    async () => {
      const resp = await marketplaceClient.listPublished({
        limit: pageSize, offset: (page - 1) * pageSize,
        keyword: searchText || undefined,
        sortBy: sortBy || undefined,
      });
      return (resp.strategies || []) as PublishedStrategy[];
    },
  );

  // Client-side price filter + cache-warmed list
  const strategies = useMemo(() => {
    let list = allStrategies;
    if (priceFilter === 'free') list = list.filter(s => !s.priceAmount || s.priceAmount === '0');
    if (priceFilter === 'paid') list = list.filter(s => s.priceAmount && Number(s.priceAmount) > 0);
    return list;
  }, [allStrategies, priceFilter]);

  // Estimate total — API returns at most pageSize items
  const total = allStrategies.length < pageSize ? (page - 1) * pageSize + allStrategies.length : page * pageSize + 1;

  // ── Purchases ──
  const { data: purchases = [], isLoading: purchasesLoading, refetch: refetchPurchases } = useRpcQuery(
    ['marketplace', 'purchases', userId],
    async (): Promise<PurchasedItem[]> => {
      if (!userId) return [];
      const resp = await marketplaceClient.listSubscriptions({});
      return (resp.subscriptions || []).map((s: SubscriptionItem): PurchasedItem => ({
        ...s,
        id: s.subscriptionId,
        purchasedAt: s.createdAt,
      }));
    },
    { enabled: !!userId },
  );

  // ── Author: my published strategies ──
  const { data: myPublished = [] } = useRpcQuery(
    ['marketplace', 'myPublished', userId],
    async (): Promise<PublishedStrategy[]> => {
      if (!userId) return [];
      const resp = await marketplaceClient.listPublished({
        userId, limit: 100,
        sortBy: 'newest',
      });
      return (resp.strategies || []) as PublishedStrategy[];
    },
    { enabled: !!userId },
  );

  // ── Publisher stats (server-side aggregation) ──
  const { data: authorStats } = useRpcQuery(
    ['marketplace', 'publisherStats', userId],
    async () => {
      if (!userId) return {
        published: 0, totalSubscribers: 0, totalRevenue: '0', monthlyRevenue: '0',
        avgRating: 0, topStrategyTitle: '',
        revenueTrend: [], subscriberTrend: [], strategyBreakdown: [],
      };
      const resp = await marketplaceClient.getPublisherStats({});
      return {
        published: resp.totalPublished || 0,
        totalSubscribers: resp.totalSubscribers || 0,
        totalRevenue: resp.totalRevenue || '0',
        monthlyRevenue: resp.monthlyRevenue || '0',
        avgRating: resp.avgRating || 0,
        topStrategyTitle: resp.topStrategyTitle || '',
        revenueTrend: resp.revenueTrend || [],
        subscriberTrend: resp.subscriberTrend || [],
        strategyBreakdown: resp.strategyBreakdown || [],
      };
    },
    { enabled: !!userId },
  );

  // ── Backtest drawer ──
  const [backtestDrawerOpen, setBacktestDrawerOpen] = useState(false);
  const [backtestStrategyId, setBacktestStrategyId] = useState('');
  const handleRunBacktest = useCallback((s: PublishedStrategy) => {
    if (!requireAuth()) return;
    setDetailOpen(false); // close detail
    setBacktestStrategyId(s.strategyId);
    setBacktestDrawerOpen(true);
  }, [requireAuth]);

  // ── Detail ──
  const openDetail = useCallback((s: PublishedStrategy) => {
    setDetailStrategy(s); setDetailOpen(true);
  }, []);
  const closeDetail = useCallback(() => {
    setDetailStrategy(null); setDetailOpen(false);
  }, []);

  // ── Purchase / Get (Set for O(1) lookup) ──
  const purchasedIds = useMemo(() => new Set(purchases.map((p: PurchasedItem) => p.strategyId)), [purchases]);
  const isPurchased = useCallback((strategyId: string) => purchasedIds.has(strategyId), [purchasedIds]);
  const ownedIds = useMemo(() => new Set(myPublished.map((s: PublishedStrategy) => s.strategyId)), [myPublished]);
  const isOwner = useCallback((strategyId: string) => ownedIds.has(strategyId), [ownedIds]);

  const handleGetFree = useCallback(async (strategy: PublishedStrategy) => {
    if (!requireAuth()) return;
    try {
      await marketplaceClient.subscribe({
        userId,
        publisherUserId: strategy.publisherUserId,
        strategyId: strategy.strategyId,
        kind: 'subscription',
      });
      message.success(t('marketplace.messages.subscribed'));
      refetchPurchases();
    } catch { message.error(t('marketplace.messages.subscribeFailed')); }
  }, [requireAuth, userId, t, refetchPurchases]);

  const handleBuy = useCallback(async (strategy: PublishedStrategy) => {
    if (!requireAuth()) return;
    try {
      const wallet = await walletApi.getWallet(userId);
      const balance = wallet?.balance || '0';
      setWalletBalance(balance);
      setPaymentStrategy(strategy);
      setPaymentModalOpen(true);
    } catch {
      message.error(t('marketplace.payment.purchaseFailed'));
    }
  }, [requireAuth, userId, t]);

  const handleConfirmPayment = useCallback(async (couponCode?: string) => {
    if (!paymentStrategy) return;
    setPaymentLoading(true);
    try {
      await marketplaceClient.purchaseStrategy({
        userId,
        strategyId: paymentStrategy.strategyId,
        publisherUserId: paymentStrategy.publisherUserId,
        couponCode: couponCode || undefined,
      });
      message.success(t('marketplace.payment.purchaseSuccess'));
      setPaymentModalOpen(false);
      setPaymentStrategy(null);
      refetchPurchases();
    } catch (err: unknown) {
      const msg = String((err as { message?: string })?.message || '');
      if (msg.includes('insufficient balance') || msg.includes('insufficient_balance')) {
        message.error(t('marketplace.payment.insufficientBalance'));
        // Refresh wallet balance so the UI shows the latest amount.
        try {
          const wallet = await walletApi.getWallet(userId);
          if (wallet?.balance != null) setWalletBalance(wallet.balance);
        } catch { /* ignore refresh errors */ }
      } else if (msg.includes('already subscribed') || msg.includes('already_exists')) {
        message.info(t('marketplace.payment.alreadyPurchased'));
        setPaymentModalOpen(false);
        setPaymentStrategy(null);
      } else {
        message.error(t('marketplace.payment.purchaseFailed'));
      }
    } finally {
      setPaymentLoading(false);
    }
  }, [paymentStrategy, userId, t, refetchPurchases]);

  const handleCancelPayment = useCallback(() => {
    setPaymentModalOpen(false);
    setPaymentStrategy(null);
  }, []);

  return {
    strategies, loading, error, purchases, purchasesLoading, myPublished,
    authorStats: authorStats ?? {
      published: 0, totalSubscribers: 0, avgRating: 0,
      totalRevenue: '0', monthlyRevenue: '0', topStrategyTitle: '',
      revenueTrend: [], subscriberTrend: [], strategyBreakdown: [],
    },
    isAuthenticated, activeTab, setActiveTab, searchText, setSearchText,
    priceFilter, setPriceFilter, sortBy, setSortBy,
    page, pageSize, total, setPage, setPageSize,
    refetch, isPurchased, isOwner, handleGetFree, handleBuy, handleRunBacktest,
    openDetail, closeDetail, detailStrategy, detailOpen,
    // Backtest drawer
    backtestDrawerOpen, setBacktestDrawerOpen, backtestStrategyId,
    // Payment
    paymentModalOpen, paymentLoading, paymentStrategy, walletBalance,
    handleConfirmPayment, handleCancelPayment,
  };
}
