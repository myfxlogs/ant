import { useState, useCallback, useMemo } from 'react';
import { message } from 'antd';
import { useTranslation } from 'react-i18next';
import { marketplaceClient } from '@/client/connect';
import { walletApi } from '@/client/wallet';
import { useRpcQuery } from '@/hooks/useRpcQuery';
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
    if (priceFilter === 'free') list = list.filter(s => !s.priceAmount || s.priceAmount === 0);
    if (priceFilter === 'paid') list = list.filter(s => s.priceAmount && s.priceAmount > 0);
    return list;
  }, [allStrategies, priceFilter]);

  // Estimate total — API returns at most pageSize items
  const total = allStrategies.length < pageSize ? (page - 1) * pageSize + allStrategies.length : page * pageSize + 1;

  // ── Purchases ──
  const { data: purchases = [], isLoading: purchasesLoading, refetch: refetchPurchases } = useRpcQuery(
    ['marketplace', 'purchases', userId],
    async (): Promise<PurchasedItem[]> => {
      if (!userId) return [];
      const resp = await marketplaceClient.listSubscriptions({ userId });
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

  const authorStats = useMemo(() => ({
    published: myPublished.length,
    totalSubscribers: myPublished.reduce((sum: number, s: PublishedStrategy) => sum + (s.totalSubscribers || 0), 0),
    avgRating: myPublished.length > 0
      ? myPublished.reduce((sum: number, s: PublishedStrategy) => sum + (s.avgRating || 0), 0) / myPublished.length
      : 0,
  }), [myPublished]);

  // ── Backtest drawer ──
  const [backtestDrawerOpen, setBacktestDrawerOpen] = useState(false);
  const [backtestStrategyId, setBacktestStrategyId] = useState('');
  const handleRunBacktest = useCallback((s: PublishedStrategy) => {
    setDetailOpen(false); // close detail
    setBacktestStrategyId(s.strategyId);
    setBacktestDrawerOpen(true);
  }, []);

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

  const handleBuy = useCallback(async (strategy: PublishedStrategy) => {
    if (!userId) { message.warning(t('marketplace.messages.loginFirst')); return; }
    try {
      const wallet = await walletApi.getWallet(userId);
      const balance = wallet?.balance || '0';
      setWalletBalance(balance);
      setPaymentStrategy(strategy);
      setPaymentModalOpen(true);
    } catch {
      message.error(t('marketplace.payment.purchaseFailed', 'Purchase failed. Please try again.'));
    }
  }, [userId, t]);

  const handleConfirmPayment = useCallback(async () => {
    if (!paymentStrategy) return;
    setPaymentLoading(true);
    try {
      await marketplaceClient.purchaseStrategy({
        userId,
        strategyId: paymentStrategy.strategyId,
        publisherUserId: paymentStrategy.publisherUserId,
      });
      message.success(t('marketplace.payment.purchaseSuccess', 'Purchase successful! Strategy added to your library.'));
      setPaymentModalOpen(false);
      setPaymentStrategy(null);
      refetchPurchases();
    } catch (err: unknown) {
      const msg = String((err as { message?: string })?.message || '');
      if (msg.includes('insufficient balance') || msg.includes('insufficient_balance')) {
        message.error(t('marketplace.payment.insufficientBalance', 'Insufficient balance'));
        // Refresh wallet balance so the UI shows the latest amount.
        try {
          const wallet = await walletApi.getWallet(userId);
          if (wallet?.balance != null) setWalletBalance(wallet.balance);
        } catch { /* ignore refresh errors */ }
      } else if (msg.includes('already subscribed') || msg.includes('already_exists')) {
        message.info(t('marketplace.payment.alreadyPurchased', 'You already own this strategy.'));
        setPaymentModalOpen(false);
        setPaymentStrategy(null);
      } else {
        message.error(t('marketplace.payment.purchaseFailed', 'Purchase failed. Please try again.'));
      }
    } finally {
      setPaymentLoading(false);
    }
  }, [paymentStrategy, userId, t, refetchPurchases, refetch]);

  const handleCancelPayment = useCallback(() => {
    setPaymentModalOpen(false);
    setPaymentStrategy(null);
  }, []);

  return {
    strategies, loading, error, purchases, purchasesLoading, myPublished, authorStats,
    activeTab, setActiveTab, searchText, setSearchText,
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
