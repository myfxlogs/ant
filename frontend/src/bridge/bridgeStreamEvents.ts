/**
 * Maps SSE order/status/snapshot events to TanStack Query cache.
 * Profit update handlers are in bridgeProfitEvents.ts.
 */
import type { QueryClient } from '@tanstack/react-query';
import type { OrderUpdate } from '@/adapters/dataAdapter';
import type { AccountStatusEvent } from '@/gen/ant/v1/stream_event_account_pb';
import { queryKeys } from '@/queries/queryKeys';
import type { Position } from '@/types/trading';
import type { Account } from '@/types/account';
import type { TradeRecordItem, RecentTradesData } from '@/client/analytics';

function normalizeSide(raw: string): Position['type'] {
  const u = raw.toLowerCase();
  if (['buy', 'sell', 'buy_limit', 'sell_limit', 'buy_stop', 'sell_stop'].includes(u)) return u;
  return u.includes('sell') ? 'sell' : 'buy';
}

function mapOrderToPosition(o: OrderUpdate): Position {
  return {
    ticket: Number(o.ticket),
    symbol: o.symbol || '',
    type: normalizeSide(o.type || 'buy'),
    volume: Number(o.volume || 0),
    openPrice: Number(o.openPrice || 0),
    sl: Number(o.stopLoss ?? 0),
    tp: Number(o.takeProfit ?? 0),
    profit: Number(o.profit || 0),
    swap: Number(o.swap ?? 0),
    commission: Number(o.commission ?? 0),
    comment: o.comment || '',
    action: o.action,
    closePrice: Number(o.closePrice ?? 0),
    closeTime: o.closeTime ? String(o.closeTime) : '',
    openTime: o.openTime ? String(o.openTime) : '',
    currentPrice: 0,
  };
}

export function handleOrderUpdate(queryClient: QueryClient, order: OrderUpdate) {
  const accountId = String(order.accountId || '');
  if (!accountId) return;

  const ticket = Number(order.ticket);
  if (!Number.isFinite(ticket) || ticket <= 0) return;

  const actionRaw = String(order.action || '').toLowerCase();
  const isClose = actionRaw.includes('close');
  const isDelete = actionRaw.includes('delete');
  const isModify = actionRaw.includes('modify');
  const isOpen = actionRaw.includes('open');

  const pos = mapOrderToPosition(order);

  if (isClose || isDelete) {
    queryClient.setQueryData<Position[]>(
      queryKeys.positions.byAccount(accountId),
      (old = []) => old.filter((p) => p.ticket !== ticket),
    );
    queryClient.setQueryData<RecentTradesData>(
      queryKeys.analytics.recentTrades(accountId),
      (old) => {
        const trade: TradeRecordItem = {
          ticket: pos.ticket, symbol: pos.symbol, type: pos.type,
          volume: pos.volume, openPrice: pos.openPrice, closePrice: pos.closePrice ?? 0,
          profit: pos.profit, openTime: pos.openTime, closeTime: pos.closeTime ?? '',
          swap: pos.swap, commission: pos.commission, comment: pos.comment,
        };
        if (!old) return { trades: [trade], total: 1 };
        const filtered = old.trades.filter((t) => t.ticket !== trade.ticket);
        return { trades: [trade, ...filtered], total: old.total + 1 };
      },
    );
	    // Invalidate analytics so charts/stats refresh when new trades arrive.
	    queryClient.invalidateQueries({ queryKey: ["analytics", accountId] });
    return;
  }

  if (isModify) {
    queryClient.setQueryData<Position[]>(
      queryKeys.positions.byAccount(accountId),
      (old = []) => old.map((p) => (p.ticket === ticket ? { ...p, ...pos } : p)),
    );
    return;
  }

  if (isOpen) {
    queryClient.setQueryData<Position[]>(
      queryKeys.positions.byAccount(accountId),
      (old = []) => {
        if (old.some((p) => p.ticket === ticket)) return old;
        return [...old, { ...pos, currentPrice: pos.openPrice }];
      },
    );
    return;
  }

  if (pos.symbol) {
    queryClient.setQueryData<Position[]>(
      queryKeys.positions.byAccount(accountId),
      (old = []) => {
        const idx = old.findIndex((p) => p.ticket === ticket);
        if (idx >= 0) {
          const next = [...old];
          next[idx] = { ...next[idx], ...pos, currentPrice: next[idx].currentPrice };
          return next;
        }
        return [...old, { ...pos, currentPrice: pos.openPrice }];
      },
    );
  }
}

export function handleAccountStatus(queryClient: QueryClient, status: AccountStatusEvent) {
  if (!status.accountId) return;
  const s = String(status.status || '');
  let mapped = s;
  if (s === 'enabled') mapped = 'connecting';
  if (s === 'disabled') mapped = 'disconnected';

  const isDisabled = s === 'disabled' ? true : s === 'enabled' ? false : undefined;
  const patch: Partial<Account> = { status: mapped };
  if (isDisabled !== undefined) patch.isDisabled = isDisabled;
  // Carry error message for disconnected/error statuses so the frontend
  // shows connection failure reasons in real time.
  if (status.message) {
    patch.lastError = status.message;
  } else if (s === 'connected') {
    patch.lastError = '';
  }

  queryClient.setQueryData<Account[]>(queryKeys.accounts.list(), (old = []) =>
    old.map((a) => (a.id === status.accountId ? { ...a, ...patch } : a)),
  );
  // Also update the individual detail query so AccountDetail page sees the change.
  queryClient.setQueryData<Account>(
    queryKeys.accounts.detail(status.accountId),
    (old) => (old ? { ...old, ...patch } : old),
  );
}

export function handlePositionSnapshot(
  queryClient: QueryClient, accountId: string, positions: OrderUpdate[],
) {
  const mapped = positions.map(mapOrderToPosition);
  queryClient.setQueryData<Position[]>(
    queryKeys.positions.byAccount(accountId),
    (old = []) => {
      const existingByTicket = new Map(old.map((p) => [p.ticket, p]));
      return mapped.map((pos) => {
        const oldPos = existingByTicket.get(pos.ticket);
        return oldPos ? { ...pos, currentPrice: oldPos.currentPrice ?? pos.openPrice } : { ...pos, currentPrice: pos.openPrice };
      });
    },
  );
}
