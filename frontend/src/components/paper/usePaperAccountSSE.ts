import { useCallback, useEffect, useRef } from 'react';
import { paperTradingStreamClient } from '@/client/connect';
import { strategyActiveApi } from '@/client/strategy';
import type { PaperAccount, PaperAccountUpdate } from '@/gen/ant/v1/paper_trading_pb';

export interface RunningStrategy {
  symbol: string;
  timeframe: string;
  runId?: string;
}

export function usePaperAccountSSE(
  accounts: PaperAccount[],
  setAccounts: React.Dispatch<React.SetStateAction<PaperAccount[]>>,
  setRunning: React.Dispatch<React.SetStateAction<Record<string, RunningStrategy>>>,
) {
  const streamRefs = useRef<Map<string, { abort: () => void }>>(new Map());

  const subscribeAccount = useCallback((accountId: string) => {
    if (streamRefs.current.has(accountId)) return;

    const abort = new AbortController();
    let cancelled = false;
    streamRefs.current.set(accountId, { abort: () => { cancelled = true; abort.abort(); } });

    (async () => {
      try {
        const stream = paperTradingStreamClient.watchPaperAccount(
          { paperAccountId: accountId },
          { signal: abort.signal },
        );
        for await (const update of stream) {
          if (cancelled) break;
          const u = update as PaperAccountUpdate;
          setAccounts(prev =>
            prev.map(a => (a.id === accountId ? { ...u.account! } : a)),
          );
        }
      } catch (err) {
        if (!cancelled) {
          console.warn('[PaperAccountPanel] SSE stream ended for', accountId, err);
        }
      } finally {
        streamRefs.current.delete(accountId);
      }
    })();
  }, [setAccounts]);

  const unsubscribeAccount = useCallback((accountId: string) => {
    const ref = streamRefs.current.get(accountId);
    if (ref) {
      ref.abort();
      streamRefs.current.delete(accountId);
    }
  }, []);

  useEffect(() => {
    for (const a of accounts) {
      subscribeAccount(a.id);
    }
    return () => {
      for (const [id] of streamRefs.current) {
        unsubscribeAccount(id);
      }
    };
  }, [accounts.map(a => a.id).join(','), subscribeAccount, unsubscribeAccount]);

  useEffect(() => {
    const abort = new AbortController();
    (async () => {
      try {
        for await (const event of strategyActiveApi.watchActive('', abort.signal)) {
          const active = (event.strategies || []) as Array<{ accountId: string; mode: string; symbol: string; timeframe: string; runId: string }>;
          const recovered: Record<string, RunningStrategy> = {};
          for (const s of active) {
            if (s.accountId && s.mode === 'paper') {
              recovered[s.accountId] = {
                symbol: s.symbol,
                timeframe: s.timeframe,
                runId: s.runId,
              };
            }
          }
          setRunning(prev => {
            const next = { ...prev };
            for (const [id, val] of Object.entries(recovered)) {
              next[id] = val;
            }
            for (const id of Object.keys(next)) {
              if (!(id in recovered)) {
                delete next[id];
              }
            }
            return next;
          });
        }
      } catch {
        // Stream ended or aborted
      }
    })();
    return () => abort.abort();
  }, [setRunning]);

  return { subscribeAccount, unsubscribeAccount };
}
