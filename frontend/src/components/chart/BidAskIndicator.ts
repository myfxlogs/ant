import { registerIndicator, type KLineData } from 'klinecharts';
import type { IndicatorCreate } from 'klinecharts';

// Module-level bid/ask store — written by SSE handler, read by indicator calc.
// Key = bar open-time in unix seconds, matching KLineData.timestamp / 1000.
const liveMap = new Map<number, { bid: number; ask: number }>();

/** Called from PriceChart onBar to feed live bid/ask into the indicator. */
export function setBidAsk(tsSec: number, bid: number, ask: number) {
  const cur = liveMap.get(tsSec);
  liveMap.set(tsSec, {
    bid: bid > 0 ? bid : cur?.bid ?? 0,
    ask: ask > 0 ? ask : cur?.ask ?? 0,
  });
}

/** Clear all stored bid/ask (e.g. on symbol/account change). */
export function clearBidAsk() {
  liveMap.clear();
}

const BIDASK_INDICATOR: IndicatorCreate = {
  name: 'BIDASK',
  shortName: 'B/A',
  precision: 5,
  shouldOhlc: false,
  figures: [
    { key: 'bid', title: 'Bid', type: 'line' },
    { key: 'ask', title: 'Ask', type: 'line' },
  ],
  styles: {
    lines: [
      { color: '#ef5350', size: 1.5, style: 'solid' as any, smooth: false, dashedValue: [2, 2] as any },
      { color: '#26a69a', size: 1.5, style: 'solid' as any, smooth: false, dashedValue: [2, 2] as any },
    ],
  },
  calc: (list: KLineData[]) => {
    if (!list || list.length === 0) return [];
    return list.map((k: any) => {
      const ts = k.timestamp != null ? Math.floor(k.timestamp / 1000) : 0;
      const ba = ts > 0 ? liveMap.get(ts) : undefined;
      const b = ba?.bid != null ? ba.bid : k.close;
      const a = ba?.ask != null ? ba.ask : k.close;
      return { bid: b ?? 0, ask: a ?? 0 };
    });
  },
  draw: () => true,
};

try { registerIndicator(BIDASK_INDICATOR); } catch { /* ok */ }

export default BIDASK_INDICATOR;
