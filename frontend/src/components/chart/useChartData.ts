import { useEffect, useRef, useState } from 'react';
import { marketApi, type KlineData } from '@/client/market';
import { subscribeEvents } from '@/client/stream';
import type { BarUpdateEvent } from '@/gen/ant/v1/stream_pb';
import { setBidAsk, clearBidAsk, setBidAskPrecision } from './BidAskIndicator';

const INITIAL_BARS = 300;

export function useChartData(symbol: string, timeframe: string, accountId?: string) {
  const [bars, setBars] = useState<KlineData[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const barsRef = useRef<KlineData[]>([]);
  const loadingMore = useRef(false);
  const loadedAll = useRef(false);
  const unsubRef = useRef<(() => void) | null>(null);

  useEffect(() => {
    if (!symbol || !accountId) return;
    let cancelled = false;

    setLoading(true);
    setError(null);
    loadedAll.current = false;
    loadingMore.current = false;
    clearBidAsk();
    if (unsubRef.current) { unsubRef.current(); unsubRef.current = null; }

    marketApi.getSymbolParams(accountId, [marketApi.resolveSymbol(symbol)]).then((infos) => {
      if (!cancelled && infos.length > 0 && infos[0].digits != null) setBidAskPrecision(infos[0].digits);
    }).catch(() => {});

    marketApi.subscribeBars({ accountId, symbol }).then(() => {
      if (cancelled) return;
      const canonical = marketApi.resolveSymbol(symbol);
      marketApi.getKlines({ symbol: canonical, timeframe, count: INITIAL_BARS, accountId })
        .then((data) => { if (!cancelled) { barsRef.current = data; setBars(data); setLoading(false); } })
        .catch((err: Error) => { if (!cancelled) { setError(err.message || 'Failed to load chart data'); setLoading(false); } });
    });

    unsubRef.current = subscribeEvents([], {
      onBar: (ev: BarUpdateEvent) => {
        if (cancelled) return;

        const b = Number(ev.bid || '0');
        const a = Number(ev.ask || '0');

        if (ev.accountId !== accountId || ev.symbol !== symbol || ev.period !== timeframe) return;

        const barTime = ev.openTime ? Number(ev.openTime.seconds ?? 0n) : 0;
        if (barTime === 0) return;

        if (b > 0 || a > 0) setBidAsk(barTime, b, a);

        const bar: KlineData = {
          time: barTime,
          open: Number(ev.open ?? '0'), high: Number(ev.high ?? '0'),
          low: Number(ev.low ?? '0'), close: Number(ev.close ?? '0'),
          volume: Number(ev.volume ?? 0),
        };

        const prev = barsRef.current;
        const idx = prev.findIndex(b2 => b2.time === barTime);
        let merged: KlineData[];
        if (idx >= 0) {
          const old = prev[idx];
          merged = [...prev];
          merged[idx] = { ...old, high: Math.max(old.high, bar.high), low: Math.min(old.low, bar.low), close: bar.close, volume: bar.volume };
        } else {
          merged = [...prev, bar].sort((x, y) => x.time - y.time);
        }
        barsRef.current = merged;
        setBars(merged);
      },
    });

    return () => { cancelled = true; if (unsubRef.current) { unsubRef.current(); unsubRef.current = null; } };
  }, [symbol, timeframe, accountId]);

  return { bars, loading, error, barsRef, loadingMore, loadedAll };
}
