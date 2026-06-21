import { createContext, useContext } from 'react';
import type { PublishedStrategy } from '@/gen/ant/v1/marketplace_service_pb';
import type { PriceFilter, SortBy, TabKey, PurchasedItem } from './hooks/useMarketplace';

export interface MarketplaceCtx {
  // Data
  strategies: PublishedStrategy[];
  loading: boolean;
  error: unknown;
  purchases: PurchasedItem[];
  purchasesLoading: boolean;
  myPublished: PublishedStrategy[];
  authorStats: { published: number; totalSubscribers: number; avgRating: number };
  // UI state
  activeTab: TabKey; setActiveTab: (t: TabKey) => void;
  searchText: string; setSearchText: (v: string) => void;
  priceFilter: PriceFilter; setPriceFilter: (f: PriceFilter) => void;
  sortBy: SortBy; setSortBy: (s: SortBy) => void;
  page: number; pageSize: number; total: number;
  setPage: (p: number) => void; setPageSize: (ps: number) => void;
  // Actions
  refetch: () => void;
  isPurchased: (id: string) => boolean;
  handleGetFree: (s: PublishedStrategy) => void;
  handleBuy: (s: PublishedStrategy) => void;
  handleRunBacktest: (s: PublishedStrategy) => void;
  // Detail
  openDetail: (s: PublishedStrategy) => void;
  closeDetail: () => void;
  detailStrategy: PublishedStrategy | null;
  detailOpen: boolean;
  // Backtest drawer
  backtestDrawerOpen: boolean; setBacktestDrawerOpen: (v: boolean) => void;
  backtestStrategyId: string;
  // Payment
  paymentModalOpen: boolean;
  paymentLoading: boolean;
  paymentStrategy: PublishedStrategy | null;
  walletBalance: string;
  handleConfirmPayment: () => void;
  handleCancelPayment: () => void;
}

const Ctx = createContext<MarketplaceCtx | null>(null);

export function MarketplaceProvider({ value, children }: { value: MarketplaceCtx; children: React.ReactNode }) {
  return <Ctx.Provider value={value}>{children}</Ctx.Provider>;
}

export function useMarketplaceCtx() {
  const c = useContext(Ctx);
  if (!c) throw new Error('useMarketplaceCtx must be used within MarketplaceProvider');
  return c;
}
