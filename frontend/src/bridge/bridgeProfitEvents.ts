/**
 * Handles SSE profit updates and flushes them to TanStack Query cache.
 *
 * State is module-scoped to batch updates across the SSE stream lifetime.
 * Call `cleanupProfitBridge()` when the SSE stream is torn down to prevent
 * stale timeouts and phantom cache updates after reconnect.
 */
import type { QueryClient } from '@tanstack/react-query';
import type { ProfitUpdate, OrderProfitItem } from '@/adapters/dataAdapter';
import { queryKeys } from '@/queries/queryKeys';
import type { Account } from '@/types/account';
import type { Position } from '@/types/trading';

const THROTTLE_MS = 300;
let profitTimeout: number | null = null;
let profitLastFlush = 0;
const pendingProfit = new Map<string, ProfitUpdate>();

/** Release module-scoped state. Call when SSE stream stops / reconnects. */
export function cleanupProfitBridge() {
  if (profitTimeout !== null) {
    window.clearTimeout(profitTimeout);
    profitTimeout = null;
  }
  profitLastFlush = 0;
  pendingProfit.clear();
}

function flushProfitUpdates(queryClient: QueryClient) {
  profitTimeout = null;
  profitLastFlush = Date.now();

  const pick = (v: unknown): number | undefined => {
    if (typeof v === 'number' && Number.isFinite(v)) return v;
    if (typeof v === 'string' && v !== '') { const n = Number(v); return Number.isFinite(n) ? n : undefined; }
    return undefined;
  };

  for (const [accId, profit] of pendingProfit) {
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

    queryClient.setQueryData<Account[]>(queryKeys.accounts.list(), (old) =>
      (old ?? []).map((a) =>
        a.id === accId
          ? {
              ...a,
              ...(pick(profit.balance) !== undefined ? { balance: pick(profit.balance) } : {}),
              ...(pick(profit.equity) !== undefined ? { equity: pick(profit.equity) } : {}),
              ...(pick(profit.profit) !== undefined ? { profit: pick(profit.profit) } : {}),
            }
          : a,
      ),
    );

    const orders: OrderProfitItem[] = Array.isArray(profit.orders) ? profit.orders : [];
    if (orders.length > 0) {
      queryClient.setQueryData<Position[]>(
        queryKeys.positions.byAccount(accId),
        (old = []) => {
          let changed = false;
          const next = old.map((p) => {
            const o = orders.find((x) => Number(x.ticket) === Number(p.ticket));
            if (o) {
              changed = true;
              return { ...p, currentPrice: Number(o.currentPrice) || p.currentPrice, profit: Number(o.profit) || p.profit };
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

export function handleProfitUpdate(queryClient: QueryClient, profit: ProfitUpdate) {
  if (!profit?.accountId) return;
  pendingProfit.set(profit.accountId, profit);

  if (profitTimeout !== null) return;
  const now = Date.now();
  const elapsed = now - profitLastFlush;
  const delay = elapsed >= THROTTLE_MS ? 0 : THROTTLE_MS - elapsed;
  // Clear any stale timeout to prevent double-fire after reconnect.
  if (profitTimeout !== null) window.clearTimeout(profitTimeout);
  profitTimeout = window.setTimeout(() => flushProfitUpdates(queryClient), delay);
}
