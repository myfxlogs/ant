import { useEffect, useRef, useState } from 'react';
import type { Chart } from 'klinecharts';
import { marketApi, isMTSessionError, type KlineData } from '@/client/market';
import { subscribeEvents } from '@/client/stream';
import type { BarUpdateEvent } from '@/gen/ant/v1/stream_pb';
import { setBidAsk, clearBidAsk, setBidAskPrecision } from './BidAskIndicator';

const INITIAL_BARS = 300;
const MAX_CACHE_ENTRIES = 20;

/** Convert internal bar (time=seconds) → klinecharts format (timestamp=ms). Shared export for PriceChart. */
export function toChartBar(bar: KlineData) {
  return { timestamp: bar.time * 1000, open: bar.open, high: bar.high, low: bar.low, close: bar.close, volume: bar.volume };
}

function mergeBar(prev: KlineData[], bar: KlineData): { merged: KlineData[]; changed: KlineData } {
  const idx = prev.findIndex(b => b.time === bar.time);
  if (idx >= 0) {
    const old = prev[idx], copy = [...prev];
    copy[idx] = {
      ...old,
      high: Math.max(old.high, bar.high),
      low: Math.min(old.low ?? old.high, bar.low ?? bar.high),  // guard undefined low
      close: bar.close,
      volume: bar.volume,
    };
    return { merged: copy, changed: copy[idx] };
  }
  const merged = [...prev, bar].sort((a, b) => a.time - b.time);
  return { merged, changed: bar };
}

function shouldProcessEvent(ev: BarUpdateEvent, accountId: string, cancelledRef: React.MutableRefObject<boolean>, symbolRef: React.MutableRefObject<string>, timeframeRef: React.MutableRefObject<string>, loadingMore: React.MutableRefObject<boolean>): boolean {
  if (cancelledRef.current) return false;
  if (ev.accountId !== accountId || ev.symbol !== symbolRef.current || ev.period !== timeframeRef.current) return false;
  if (loadingMore.current) return false;
  const barTime = ev.openTime ? Number(ev.openTime.seconds ?? 0n) : 0;
  return barTime !== 0;
}

function updateBidAsk(ev: BarUpdateEvent, barTime: number, precisionRef: React.MutableRefObject<number>, setBidAsk: (t: number, b: number, a: number) => void, setLatestBid: (s: string) => void, setLatestAsk: (s: string) => void) {
  const b = Number(ev.bid || '0'), a = Number(ev.ask || '0');
  if (b > 0 || a > 0) {
    setBidAsk(barTime * 1000, b, a);
    const d = precisionRef.current;
    setLatestBid(b > 0 ? b.toFixed(d) : '');
    setLatestAsk(a > 0 ? a.toFixed(d) : '');
  }
}

function processBarEvent(
  ev: BarUpdateEvent, accountId: string,
  cancelledRef: React.MutableRefObject<boolean>,
  symbolRef: React.MutableRefObject<string>,
  timeframeRef: React.MutableRefObject<string>,
  loadingMore: React.MutableRefObject<boolean>,
  precisionRef: React.MutableRefObject<number>,
  barsRef: React.MutableRefObject<KlineData[]>,
  chartRef: React.MutableRefObject<Chart | null>,
  setBidAsk: (t: number, b: number, a: number) => void,
  setLatestBid: (s: string) => void,
  setLatestAsk: (s: string) => void,
) {
  if (!shouldProcessEvent(ev, accountId, cancelledRef, symbolRef, timeframeRef, loadingMore)) return;

  const barTime = ev.openTime ? Number(ev.openTime.seconds ?? 0n) : 0;

  updateBidAsk(ev, barTime, precisionRef, setBidAsk, setLatestBid, setLatestAsk);

  const bar: KlineData = {
    time: barTime, open: Number(ev.open ?? '0'), high: Number(ev.high ?? '0'),
    low: Number(ev.low ?? '0'), close: Number(ev.close ?? '0'), volume: Number(ev.volume ?? 0),
  };
  const { merged, changed } = mergeBar(barsRef.current, bar);
  barsRef.current = merged;
  chartRef.current?.updateData(toChartBar(changed));
}

export function useChartData(
  symbol: string, timeframe: string, accountId: string | undefined,
  chartRef: React.MutableRefObject<Chart | null>,
) {
  const [bars, setBars] = useState<KlineData[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [streamActive, setStreamActive] = useState(false);
  const [latestBid, setLatestBid] = useState<string>('');
  const [latestAsk, setLatestAsk] = useState<string>('');
  const precisionRef = useRef(5);
  const barsRef = useRef<KlineData[]>([]);
  const loadingMore = useRef(false);
  const loadedAll = useRef(false);
  const cancelledRef = useRef(false);
  const unsubRef = useRef<(() => void) | null>(null);
  // Client-side bar cache: avoid re-fetching recently viewed symbol+timeframe combos
  const barCache = useRef<Map<string, KlineData[]>>(new Map());
  // Refs for current symbol/timeframe to avoid closure staleness in SSE handler
  const symbolRef = useRef(symbol);
  const timeframeRef = useRef(timeframe);
  symbolRef.current = symbol;
  timeframeRef.current = timeframe;

  // ── Effect 1: SSE subscription (re-created only on accountId change) ──
  useEffect(() => {
    if (!accountId) return;
    cancelledRef.current = false;
    unsubRef.current?.();
    unsubRef.current = subscribeEvents([], {
      onBar: (ev: BarUpdateEvent) => processBarEvent(ev, accountId, cancelledRef, symbolRef, timeframeRef, loadingMore, precisionRef, barsRef, chartRef, setBidAsk, setLatestBid, setLatestAsk),
    });
    return () => { cancelledRef.current = true; unsubRef.current?.(); unsubRef.current = null; };
  // eslint-disable-next-line react-hooks/exhaustive-deps -- chartRef is a ref  | REF: rd.md#part-0.2-hooks-deps
  }, [accountId]);

  // ── Effect 2: subscribeBars (only on symbol or account change) ──
  useEffect(() => {
    if (!symbol || !accountId) return;
    setStreamActive(false);
    marketApi.subscribeBars({ accountId, symbol })
      .then(() => { if (!cancelledRef.current) setStreamActive(true); })
      .catch((err: Error) => {
        if (!cancelledRef.current) {
          const msg = isMTSessionError(err)
            ? 'MT session lost — real-time updates paused. Check terminal connection.'
            : `Live stream unavailable: ${err.message || 'subscription failed'}`;
          setError(msg);
        }
      });
  }, [symbol, accountId]);

  // ── Effect 3: fetch initial bars (symbol/timeframe/account change) ──
  useEffect(() => {
    if (!symbol || !accountId) return;
    const cacheKey = `${symbol}|${timeframe}`;
    const cached = barCache.current.get(cacheKey);
    if (cached && cached.length > 0) {
      barsRef.current = cached; setBars(cached); setLoading(false);
      return;
    }

    let cancelled = false;
    setLoading(true); setError(null);
    loadedAll.current = false; loadingMore.current = false;
    barsRef.current = [];  // clear stale bars from previous symbol
    clearBidAsk();

    marketApi.getSymbolParams(accountId, [marketApi.resolveSymbol(symbol)]).then((infos) => {
      if (!cancelled && infos.length > 0 && infos[0].digits != null) {
        setBidAskPrecision(infos[0].digits);
        precisionRef.current = infos[0].digits;
      }
    }).catch(() => {});

    marketApi.getKlines({ symbol: marketApi.resolveSymbol(symbol), timeframe, count: INITIAL_BARS, accountId })
      .then((data) => {
        if (!cancelled) { barsRef.current = data ?? []; setBars(data ?? []); barCache.current.set(cacheKey, data ?? []); if (barCache.current.size > MAX_CACHE_ENTRIES) { const oldest = barCache.current.keys().next().value; if (oldest) barCache.current.delete(oldest); } setLoading(false); }
      })
      .catch((err: Error) => {
        if (!cancelled) {
          const msg = isMTSessionError(err)
            ? 'MT session lost — chart data unavailable. Check terminal connection.'
            : (err.message || 'Failed to load chart data');
          setError(msg); setLoading(false);
        }
      });

    return () => { cancelled = true; };
  }, [symbol, timeframe, accountId]);

  return { bars, loading, error, streamActive, barsRef, loadingMore, loadedAll, latestBid, latestAsk };
}
