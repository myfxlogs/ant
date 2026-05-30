/**
 * Maps SSE stream events to TanStack Query cache operations.
 * Replaces the data-mapping logic previously in ConnectProvider.
 */
import type { QueryClient } from '@tanstack/react-query';
import type { OrderUpdate, ProfitUpdate, OrderProfitItem } from '@/adapters/dataAdapter';
import type { AccountStatusEvent } from '@/gen/ant/v1/stream_event_account_pb';
import { queryKeys } from '@/queries/queryKeys';
import type { Position } from '@/types/trading';
import type { Account } from '@/types/account';

const THROTTLE_MS = 300;
let profitTimeout: number | null = null;
let profitLastFlush = 0;
const pendingProfit = new Map<string, ProfitUpdate>();

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

function flushProfitUpdates(queryClient: QueryClient) {
  profitTimeout = null;
  profitLastFlush = Date.now();

  const pick = (v: unknown): number | undefined =>
    typeof v === 'number' && Number.isFinite(v) ? v : undefined;

  for (const [accId, profit] of pendingProfit) {
    // Update financials cache
    queryClient.setQueryData<Record<string, number>>(
      queryKeys.accounts.financials(accId),
      (old) => {
        const base = old ?? {};
        const bal = pick(profit.balance);
        const eq = pick(profit.equity);
        const pr = pick(profit.profit);
        const mg = pick(profit.margin);
        const fm = pick(profit.freeMargin);
        const ml = pick(profit.marginLevel);
        const cr = pick(profit.credit);
        return {
          ...base,
          ...(bal !== undefined ? { balance: bal } : {}),
          ...(eq !== undefined ? { equity: eq } : {}),
          ...(pr !== undefined ? { profit: pr } : {}),
          ...(mg !== undefined ? { margin: mg } : {}),
          ...(fm !== undefined ? { freeMargin: fm } : {}),
          ...(ml !== undefined ? { marginLevel: ml } : {}),
          ...(cr !== undefined ? { credit: cr } : {}),
        };
      },
    );

    // Patch account list cache with live financials
    queryClient.setQueryData<Account[]>(queryKeys.accounts.list(), (old) =>
      (old ?? []).map((a) =>
        a.id === accId
          ? {
              ...a,
              ...(pick(profit.balance) !== undefined ? { balance: pick(profit.balance) } : {}),
              ...(pick(profit.equity) !== undefined ? { equity: pick(profit.equity) } : {}),
              ...(pick(profit.profit) !== undefined ? { profit: pick(profit.profit) } : {}),
              status: 'connected' as const,
            }
          : a,
      ),
    );

    // Patch existing position currentPrices (only rows that already exist)
    const orders: OrderProfitItem[] = Array.isArray(profit.orders) ? profit.orders : [];
    if (orders.length > 0) {
      queryClient.setQueryData<Position[]>(
        queryKeys.positions.byAccount(accId),
        (old = []) => {
          let changed = false;
          const next = old.map((p) => {
            const o = orders.find(
              (x) => Number(x.ticket) === Number(p.ticket),
            );
            if (o) {
              changed = true;
              return {
                ...p,
                currentPrice: Number(o.currentPrice) || p.currentPrice,
                profit: Number(o.profit) || p.profit,
              };
            }
            return p;
          });
          return changed ? next : old;
        },
      );
    }
  }
  pendingProfit.clear();
}

export function handleProfitUpdate(
  queryClient: QueryClient,
  profit: ProfitUpdate,
) {
  if (!profit?.accountId) return;
  pendingProfit.set(profit.accountId, profit);

  if (profitTimeout) return;
  const now = Date.now();
  const elapsed = now - profitLastFlush;
  const delay = elapsed >= THROTTLE_MS ? 0 : THROTTLE_MS - elapsed;
  profitTimeout = window.setTimeout(() => flushProfitUpdates(queryClient), delay);
}

export function handleOrderUpdate(
  queryClient: QueryClient,
  order: OrderUpdate,
) {
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

export function handleAccountStatus(
  queryClient: QueryClient,
  status: AccountStatusEvent,
) {
  if (!status.accountId) return;
  const s = String(status.status || '');
  let mapped = s;
  if (s === 'enabled') mapped = 'connecting';
  if (s === 'disabled') mapped = 'disconnected';

  queryClient.setQueryData<Account[]>(queryKeys.accounts.list(), (old = []) =>
    old.map((a) =>
      a.id === status.accountId ? { ...a, status: mapped } : a,
    ),
  );
}

export function handlePositionSnapshot(
  queryClient: QueryClient,
  accountId: string,
  positions: OrderUpdate[],
) {
  const mapped = positions.map(mapOrderToPosition);
  queryClient.setQueryData<Position[]>(
    queryKeys.positions.byAccount(accountId),
    (old = []) => {
      const existingByTicket = new Map(old.map((p) => [p.ticket, p]));
      return mapped.map((pos) => {
        const oldPos = existingByTicket.get(pos.ticket);
        return oldPos
          ? { ...pos, currentPrice: oldPos.currentPrice ?? pos.openPrice }
          : { ...pos, currentPrice: pos.openPrice };
      });
    },
  );
}
