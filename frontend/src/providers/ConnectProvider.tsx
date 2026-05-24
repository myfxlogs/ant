import { useEffect, useRef, useState, useCallback, type ReactNode } from 'react';
import { subscribeEvents, subscribeUserSummary } from '@/client/stream';
import { tradingApi } from '@/client/trading';
import type { OrderProfitItem, OrderUpdate, ProfitUpdate } from '@/adapters/dataAdapter';
import { toCamelCase } from '@/adapters/dataAdapter';
import { useAuthStore } from '@/stores/authStore';
import { useTradingStore, type AccountInfo } from '@/stores/tradingStore';
import { useAccountStore } from '@/stores/accountStore';
import { ConnectContext } from './connectContext';
import { getDeviceLocale, getDeviceTimeZone } from '@/utils/date';
import type { Position } from '@/types/trading';
import type { AccountStatusEvent } from '@/gen/stream_event_account_pb';

function normalizePositionSide(raw: string): Position['type'] {
  const u = raw.toLowerCase();
  if (u === 'buy' || u === 'sell' || u === 'buy_limit' || u === 'sell_limit' || u === 'buy_stop' || u === 'sell_stop') {
    return u;
  }
  if (u.includes('sell')) return 'sell';
  return 'buy';
}

export function ConnectProvider({ children }: { children: ReactNode }) {
  const unsubscribeEventsRef = useRef<(() => void) | null>(null);
  const unsubscribeUserSummaryRef = useRef<(() => void) | null>(null);
  const mountedRef = useRef(false);
  const lastAccountIdsRef = useRef<string>('');
  const isConnectingRef = useRef(false);
  const connectSeqRef = useRef(0);
  const isConnectedRef = useRef(false);
  const [isConnected, setIsConnected] = useState(false);
  const [connectionState, setConnectionState] = useState<'connecting' | 'connected' | 'disconnected'>('disconnected');
  const reconnectTimeoutRef = useRef<number | null>(null);
  const reconnectAttemptsRef = useRef(0);
  const connectTimeoutRef = useRef<number | null>(null);
  const profitUpdateTimeoutRef = useRef<number | null>(null);
  const pendingProfitUpdates = useRef<Map<string, ProfitUpdate>>(new Map());
  /** Throttle full getPositions refresh while profit stream updates account cards (row-level profit). */
  const profitPositionsPollRef = useRef<Map<string, number>>(new Map());
  const POSITIONS_POLL_MS = 1000;

  const connect = useCallback(() => {
    // Single-flight: if a connect is already scheduled/in-progress, do not queue another.
    if (connectTimeoutRef.current || isConnectingRef.current) {
      return;
    }

    // Mark connecting immediately to avoid race between multiple connect() calls before the timeout fires.
    isConnectingRef.current = true;
    setConnectionState('connecting');
    const seq = ++connectSeqRef.current;

    connectTimeoutRef.current = setTimeout(() => {
      // Only the latest scheduled connect is allowed to run.
      if (seq !== connectSeqRef.current) {
        return;
      }
      const { isAuthenticated, accessToken } = useAuthStore.getState();
      const { accounts } = useAccountStore.getState();

      if (!isAuthenticated || !accessToken) {
        // Tear down streams when user is not authenticated or no active accounts.
        if (unsubscribeEventsRef.current) {
          unsubscribeEventsRef.current();
          unsubscribeEventsRef.current = null;
        }
        if (unsubscribeUserSummaryRef.current) {
          unsubscribeUserSummaryRef.current();
          unsubscribeUserSummaryRef.current = null;
        }

        // Update cached account statuses to avoid stale UI.
        for (const a of accounts) {
          if (!a?.id) continue;
          useAccountStore.getState().updateAccountStatus(a.id, 'disconnected');
        }

        useTradingStore.getState().setUserSummary({
          totalBalance: 0,
          totalEquity: 0,
          totalProfit: 0,
          accountCount: 0,
          connectedCount: 0,
          pnlToday: 0,
          pnlWeek: 0,
          pnlMonth: 0,
          tradesToday: 0,
          tradesWeek: 0,
          tradesMonth: 0,
          winRate: 0,
          profitFactor: 0,
          maxDrawdownPercent: 0,
          maxConsecutiveWins: 0,
          maxConsecutiveLosses: 0,
        });

        setIsConnected(false);
        isConnectedRef.current = false;
        setConnectionState('disconnected');
        lastAccountIdsRef.current = '';
        isConnectingRef.current = false;
        if (connectTimeoutRef.current) {
          clearTimeout(connectTimeoutRef.current);
          connectTimeoutRef.current = null;
        }
        return;
      }

      // Stable MT-official-like stream: server manages the enabled account set.
      // Client subscribes once with accountIds=[] and does not reconnect on enabled/disabled changes.
      if (!unsubscribeUserSummaryRef.current) {
        unsubscribeUserSummaryRef.current = subscribeUserSummary((summary) => {
          if (!mountedRef.current) return;
          useTradingStore.getState().setUserSummary({
            totalBalance: summary.totalBalance,
            totalEquity: summary.totalEquity,
            totalProfit: summary.totalProfit,
            accountCount: summary.accountCount,
            connectedCount: summary.connectedCount,
            pnlToday: summary.pnlToday,
            pnlWeek: summary.pnlWeek,
            pnlMonth: summary.pnlMonth,
            tradesToday: summary.tradesToday,
            tradesWeek: summary.tradesWeek,
            tradesMonth: summary.tradesMonth,
            winRate: summary.winRate,
            profitFactor: summary.profitFactor,
            maxDrawdownPercent: summary.maxDrawdownPercent,
            maxConsecutiveWins: summary.maxConsecutiveWins,
            maxConsecutiveLosses: summary.maxConsecutiveLosses,
            updatedAt: summary.updatedAt,
          });
        }, () => {});
      }

      if (!unsubscribeEventsRef.current) {
        unsubscribeEventsRef.current = subscribeEvents([], {
          onOrder: (orderEvent: OrderUpdate) => {
            if (!mountedRef.current) return;

            const locale = getDeviceLocale();
            const timeZone = getDeviceTimeZone();

            const store = useTradingStore.getState();

            const accountId = String(orderEvent.accountId || '');
            if (!accountId) return;

            const actionRaw = String(orderEvent.action || '').toLowerCase();
            const typeRaw = String(orderEvent.type || '').toLowerCase();

            const ticket = Number(orderEvent.ticket);
            if (!Number.isFinite(ticket) || ticket <= 0) {
              return;
            }

            const positionPatch: Partial<Position> = {
              ticket,
              symbol: orderEvent.symbol || '',
              type: normalizePositionSide(typeRaw || 'buy'),
              volume: Number(orderEvent.volume || 0),
              openPrice: Number(orderEvent.openPrice || 0),
              sl: Number(orderEvent.stopLoss ?? 0),
              tp: Number(orderEvent.takeProfit ?? 0),
              profit: Number(orderEvent.profit || 0),
              swap: Number(orderEvent.swap ?? 0),
              commission: Number(orderEvent.commission ?? 0),
              comment: orderEvent.comment || '',
              action: orderEvent.action,
              closePrice: Number(orderEvent.closePrice ?? 0),
              closeTime: orderEvent.closeTime
                ? new Date(Number(orderEvent.closeTime) * 1000).toLocaleString(locale, { timeZone })
                : '',
              openTime: orderEvent.openTime
                ? new Date(Number(orderEvent.openTime) * 1000).toLocaleString(locale, { timeZone })
                : '',
            };

            const isClose = actionRaw.includes('close');
            const isDelete = actionRaw.includes('delete');
            const isModify = actionRaw.includes('modify');
            const isOpen = actionRaw.includes('open');

            const dispatchPositionChange = (action: string) => {
              try {
                window.dispatchEvent(
                  new CustomEvent('position-change', {
                    detail: {
                      action,
                      order: {
                        ticket,
                        symbol: positionPatch.symbol,
                        type: positionPatch.type,
                        volume: positionPatch.volume,
                        openPrice: positionPatch.openPrice,
                        closePrice: positionPatch.closePrice,
                        profit: positionPatch.profit,
                        openTime: Number(orderEvent.openTime) || 0,
                        closeTime: Number(orderEvent.closeTime) || 0,
                        swap: positionPatch.swap,
                        commission: positionPatch.commission,
                        comment: positionPatch.comment,
                      },
                    },
                  }),
                );
              } catch {
                // noop
              }
            };

            if (isClose || isDelete) {
              store.removePosition(accountId, ticket);
              dispatchPositionChange('PositionClose');
              return;
            }

            if (isModify) {
              store.updatePosition(accountId, ticket, positionPatch);
              return;
            }

            if (isOpen) {
              const existing = store.positionsMap.get(accountId) || [];
              const old = existing.find((p) => p.ticket === ticket);
              store.addPosition(accountId, {
                ...positionPatch,
                currentPrice: old?.currentPrice ?? positionPatch.openPrice,
              });
              dispatchPositionChange('PositionOpen');
              return;
            }

            if (positionPatch.symbol) {
              const existing = store.positionsMap.get(accountId) || [];
              const old = existing.find((p) => p.ticket === ticket);
              if (old) {
                store.updatePosition(accountId, ticket, {
                  ...positionPatch,
                  currentPrice: old.currentPrice,
                });
              } else {
                store.addPosition(accountId, {
                  ...positionPatch,
                  currentPrice: positionPatch.openPrice,
                });
              }
            }
          },
          onProfit: (profit) => {
            if (!mountedRef.current) return;

            // Receiving profit updates implies the account stream is active.
            if (profit?.accountId) {
              useAccountStore.getState().updateAccountStatus(profit.accountId, 'connected');
            }

            if (profit?.accountId) {
              pendingProfitUpdates.current.set(profit.accountId, profit);
            }

            if (profitUpdateTimeoutRef.current) {
              clearTimeout(profitUpdateTimeoutRef.current);
            }

            profitUpdateTimeoutRef.current = setTimeout(() => {
              if (!mountedRef.current) return;

              const tradingStore = useTradingStore.getState();
              const updates = pendingProfitUpdates.current;

              const pick = (v: unknown): number | undefined => {
                if (typeof v !== 'number') return undefined;
                return Number.isFinite(v) ? v : undefined;
              };
              for (const [accId, profitData] of updates.entries()) {
                const patch: Partial<AccountInfo> = {};
                const bal = pick(profitData.balance);
                const eq = pick(profitData.equity);
                const pr = pick(profitData.profit);
                const pp = pick(profitData.profitPercent);
                const mg = pick(profitData.margin);
                const fm = pick(profitData.freeMargin);
                const ml = pick(profitData.marginLevel);
                const cr = pick(profitData.credit);
                if (bal !== undefined) patch.balance = bal;
                if (eq !== undefined) patch.equity = eq;
                if (pr !== undefined) patch.profit = pr;
                if (pp !== undefined) patch.profitPercent = pp;
                if (mg !== undefined) patch.margin = mg;
                if (fm !== undefined) patch.freeMargin = fm;
                if (ml !== undefined) patch.marginLevel = ml;
                if (cr !== undefined) patch.credit = cr;
                if (Object.keys(patch).length > 0) {
                  tradingStore.setAccountInfoById(accId, patch);
                }
                tradingStore.touchStreamProfitAt(accId);

                // Authoritative open list: order stream + getPositions. Profit `orders` (when present) may be partial (MT5).
                // Only patch rows that already exist; never add from profit. MT5 backend omits orders on profit events.
                const orders: OrderProfitItem[] = Array.isArray(profitData.orders) ? profitData.orders : [];
                const existingRows = tradingStore.positionsMap.get(accId) || [];
                for (const o of orders) {
                  const ticketRaw = o.ticket;
                  const ticket = typeof ticketRaw === 'bigint' ? Number(ticketRaw) : Number(ticketRaw);
                  if (!Number.isFinite(ticket) || ticket <= 0) continue;
                  const old = existingRows.find((p) => Number(p.ticket) === ticket);
                  if (!old) continue;
                  const posPatch: Partial<Position> = {
                    ticket,
                    symbol: String(o.symbol || old.symbol || ''),
                    type: old.type,
                    volume: Number(o.volume) || Number(old.volume) || 0,
                    openPrice: Number(old.openPrice) || 0,
                    currentPrice: Number(o.currentPrice) || Number(old.currentPrice) || 0,
                    profit: Number(o.profit) || 0,
                    openTime: old.openTime || '',
                  };
                  useTradingStore.getState().updatePosition(accId, ticket, posPatch);
                }

                const now = Date.now();
                const lastPoll = profitPositionsPollRef.current.get(accId) ?? 0;
                if (existingRows.length > 0 && now - lastPoll >= POSITIONS_POLL_MS) {
                  profitPositionsPollRef.current.set(accId, now);
                  tradingApi
                    .getPositions(accId)
                    .then((pos) => {
                      const arr = Array.isArray(pos) ? pos : [];
                      useTradingStore.getState().setPositions(accId, toCamelCase<Position[]>(arr), { preferRpcProfit: true });
                    })
                    .catch(() => {});
                }
              }

              pendingProfitUpdates.current.clear();
            }, 100);
          },
          onStatus: (status: AccountStatusEvent) => {
            if (!mountedRef.current) return;
            const accountId = status.accountId;
            const s = String(status.status || '');
            if (!accountId) return;
            let mapped = s;
            if (s === 'enabled') mapped = 'connecting';
            if (s === 'disabled') mapped = 'disconnected';
            if (mapped === 'connected' || mapped === 'connecting' || mapped === 'disconnected') {
              useAccountStore.getState().updateAccountStatus(accountId, mapped);
            }
          },
          onError: () => {},
        });
      }

      setIsConnected(true);
      isConnectedRef.current = true;
      setConnectionState('connected');
      isConnectingRef.current = false;
      reconnectAttemptsRef.current = 0;

      if (connectTimeoutRef.current) {
        clearTimeout(connectTimeoutRef.current);
        connectTimeoutRef.current = null;
      }
    }, 100);
  }, []);

  useEffect(() => {
    mountedRef.current = true;

    const unsubscribeAuth = useAuthStore.subscribe((state, prevState) => {
      if (state.isAuthenticated !== prevState.isAuthenticated) {
        setIsConnected(false);
        isConnectedRef.current = false;
        setConnectionState('disconnected');
        reconnectAttemptsRef.current = 0;
        connect();
      }
    });

    connect();

    let initialPollCount = 0;
    const initialPollInterval = setInterval(() => {
      if (!mountedRef.current) {
        clearInterval(initialPollInterval);
        return;
      }
      
      initialPollCount++;
      const { isAuthenticated } = useAuthStore.getState();
      
      if (isAuthenticated) {
        clearInterval(initialPollInterval);
        connect();
      } else if (initialPollCount >= 10 || isConnectedRef.current) {
        clearInterval(initialPollInterval);
      }
    }, 1000);

    const intervalId = setInterval(() => {
      if (mountedRef.current && !isConnectedRef.current && !isConnectingRef.current) {
        const { isAuthenticated } = useAuthStore.getState();
        if (isAuthenticated) {
          connect();
        }
      } else if (mountedRef.current) {
      }
    }, 30000);

    return () => {
      mountedRef.current = false;

      unsubscribeAuth();
      clearInterval(initialPollInterval);
      clearInterval(intervalId);

      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current);
        reconnectTimeoutRef.current = null;
      }
      if (connectTimeoutRef.current) {
        clearTimeout(connectTimeoutRef.current);
        connectTimeoutRef.current = null;
      }
      if (profitUpdateTimeoutRef.current) {
        clearTimeout(profitUpdateTimeoutRef.current);
        profitUpdateTimeoutRef.current = null;
      }

      if (unsubscribeEventsRef.current) {
        unsubscribeEventsRef.current();
        unsubscribeEventsRef.current = null;
      }
      if (unsubscribeUserSummaryRef.current) {
        unsubscribeUserSummaryRef.current();
        unsubscribeUserSummaryRef.current = null;
      }
    };
  }, [connect]);

  return (
    <ConnectContext.Provider value={{ isConnected, connectionState }}>
      {children}
    </ConnectContext.Provider>
  );
}
