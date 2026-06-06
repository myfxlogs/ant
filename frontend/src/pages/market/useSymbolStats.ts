import { useState, useEffect, useRef } from 'react';
import { marketClient } from '@/client/connect';
import type { GetSymbolStatsResponse } from '@/gen/ant/v1/market_service_pb';

export interface StatState {
  bid: string;
  ask: string;
  spread: string;
  loading: boolean;
}

export function useSymbolStats(symbol: string) {
  const [stats, setStats] = useState<StatState>({ bid: '-', ask: '-', spread: '-', loading: false });
  const tickAbortRef = useRef<AbortController | null>(null);

  useEffect(() => {
    if (!symbol) {
      setStats({ bid: '-', ask: '-', spread: '-', loading: false });
      return;
    }
    let cancelled = false;
    setStats((s) => ({ ...s, loading: true }));

    marketClient.getSymbolStats({ canonical: symbol, broker: '' })
      .then((res: GetSymbolStatsResponse) => {
        if (cancelled) return;
        setStats({ bid: res.currentBid || '-', ask: res.currentAsk || '-', spread: res.spread || '-', loading: false });
      })
      .catch(() => { if (cancelled) return; setStats({ bid: '-', ask: '-', spread: '-', loading: false }); });

    const ac = new AbortController();
    tickAbortRef.current = ac;
    (async () => {
      try {
        const stream = marketClient.streamTicks({ canonicals: [symbol], broker: '' }, { signal: ac.signal });
        for await (const tick of stream) {
          if (cancelled || ac.signal.aborted) break;
          if (tick.canonical !== symbol) continue;
          setStats((s) => {
            const bid = tick.bid || s.bid;
            const ask = tick.ask || s.ask;
            const bidN = parseFloat(bid); const askN = parseFloat(ask);
            const spread = (bidN && askN) ? (askN - bidN).toFixed(5) : s.spread;
            return { ...s, bid, ask, spread, loading: false };
          });
        }
      } catch { /* stream ended */ }
    })();

    return () => { cancelled = true; ac.abort(); };
  }, [symbol]);

  return stats;
}
