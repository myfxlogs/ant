import { useState, useCallback, useMemo, useRef } from 'react';
import { message } from 'antd';
import { useQueryClient } from '@tanstack/react-query';
import { queryKeys } from '@/queries/queryKeys';
import { useAccountFinancials } from '@/queries/useAccountFinancials';
import { usePositionsQuery } from '@/queries/usePositionsQuery';
import { tradingApi } from '@/client/trading';

export interface QuickTradePosition {
  ticket: number; side: string; volume: number;
  openPrice: number; markPrice?: number; profit: number; leverage?: number;
}

export interface RecentTrade {
  ticket: number; symbol: string; side: string;
  closePrice?: number; price?: number; profit: number; closeTime?: string; created_at?: string;
}

export function useQuickTradeData(accountId: string, symbol: string) {
  const { data: accountInfo, isSuccess: financialsReady } = useAccountFinancials(accountId);
  const { data: rawPositions } = usePositionsQuery(accountId);
  const positionCount = rawPositions?.length ?? 0;

  const allPositions: QuickTradePosition[] = useMemo(() => {
    if (!accountId || !rawPositions) return [];
    return rawPositions.map(p => ({
      ticket: p.ticket, side: p.type.startsWith('buy') ? 'long' : 'short',
      symbol: p.symbol, volume: p.volume || 0, openPrice: p.openPrice || 0,
      markPrice: p.currentPrice, profit: p.profit || 0,
    }));
  }, [accountId, rawPositions]);

  const qtPositions: QuickTradePosition[] = useMemo(() => {
    if (!symbol) return [];
    return (rawPositions || []).filter(p => p.symbol === symbol).map(p => ({
      ticket: p.ticket, side: p.type.startsWith('buy') ? 'long' : 'short',
      volume: p.volume || 0, openPrice: p.openPrice || 0,
      markPrice: p.currentPrice, profit: p.profit || 0, leverage: undefined,
    }));
  }, [symbol, rawPositions]);

  const tradeCacheRef = useRef<Set<string>>(new Set());
  const [qtRecentTrades, setQtRecentTrades] = useState<RecentTrade[]>([]);
  const fetchTradeHistory = useCallback(async () => {
    if (!accountId || !financialsReady) return;
    if (tradeCacheRef.current.has(accountId)) return;
    tradeCacheRef.current.add(accountId);
    try {
      const result = await tradingApi.getOrderHistory({ accountId, pageSize: 5 });
      setQtRecentTrades(result.orders.slice(0, 5).map(o => ({
        ticket: o.ticket, symbol: o.symbol || '', side: o.type || '',
        closePrice: o.closePrice, price: o.openPrice, profit: o.profit || 0,
        closeTime: o.closeTime ? new Date(o.closeTime * 1000).toISOString() : undefined,
        created_at: o.openTime ? new Date(o.openTime * 1000).toISOString() : undefined,
      })));
    } catch (e) { console.warn('fetch trade history failed', e); }
  }, [accountId, financialsReady]);

  const queryClient = useQueryClient();
  const handleClosePosition = useCallback(async (ticket: number, volume?: number) => {
    if (!accountId) return;
    try {
      const result = await tradingApi.orderClose({ accountId, ticket: BigInt(ticket), volume });
      if (result.error) { message.error(result.message || result.error); } else {
        message.success(result.message || 'Position closed');
        queryClient.invalidateQueries({ queryKey: queryKeys.positions.byAccount(accountId) });
      }
    } catch (e: unknown) { message.error((e as Error)?.message || 'Close failed'); }
  }, [accountId, queryClient]);

  return { accountInfo, financialsReady, positionCount, allPositions, qtPositions, qtRecentTrades, handleClosePosition, fetchTradeHistory };
}
