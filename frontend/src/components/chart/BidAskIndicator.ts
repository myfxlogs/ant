import { registerIndicator, type KLineData } from 'klinecharts';
import type { IndicatorCreate } from 'klinecharts';

// Module-level state for real-time bid/ask.
const liveMap = new Map<number, { bid: number; ask: number }>();
let latest: { bid: number; ask: number } | null = null;
let precision = 5;

export function setBidAsk(tsSec: number, bid: number, ask: number) {
  const cur = liveMap.get(tsSec);
  const nb = bid > 0 ? bid : cur?.bid ?? 0;
  const na = ask > 0 ? ask : cur?.ask ?? 0;
  liveMap.set(tsSec, { bid: nb, ask: na });
  latest = { bid: nb || latest?.bid || 0, ask: na || latest?.ask || 0 };
}

export function clearBidAsk() {
  liveMap.clear();
  latest = null;
}

export function setBidAskPrecision(digits: number) {
  if (digits > 0) precision = digits;
}

function fmt(p: number): string {
  return p > 0 ? p.toFixed(precision) : '';
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
    const bv = latest?.bid || 0;
    const av = latest?.ask || 0;
    return list.map((k: any) => ({
      bid: bv > 0 ? bv : k.close,
      ask: av > 0 ? av : k.close,
    }));
  },

  draw: ({ ctx, bounding, yAxis, kLineDataList }: any) => {
    const line = (price: number, color: string) => {
      if (!(price > 0)) return;
      const y = yAxis.convertToPixel(price);
      if (y == null || y < 0 || y > bounding.height) return;

      ctx.save();
      ctx.strokeStyle = color;
      ctx.lineWidth = 1;
      ctx.setLineDash([4, 3]);
      ctx.beginPath();
      ctx.moveTo(0, y);
      ctx.lineTo(bounding.width, y);
      ctx.stroke();

      // Right-side label pill.
      const label = fmt(price);
      if (label) {
        ctx.setLineDash([]);
        ctx.font = '10px sans-serif';
        const tw = ctx.measureText(label).width + 8;
        const lx = bounding.width - tw;
        ctx.fillStyle = color;
        ctx.fillRect(lx, y - 7, tw, 14);
        ctx.fillStyle = '#fff';
        ctx.textBaseline = 'middle';
        ctx.fillText(label, lx + 4, y);
      }
      ctx.restore();
    };

    if (latest && (latest.bid > 0 || latest.ask > 0)) {
      line(latest.bid, '#ef5350');
      line(latest.ask, '#26a69a');
    } else {
      // Fallback: gray line at last close.
      const last = kLineDataList?.[kLineDataList.length - 1];
      if (last?.close > 0) line(last.close, '#888888');
    }
    return true; // skip default per-bar rendering
  },
};

try { registerIndicator(BIDASK_INDICATOR); } catch { /* ok */ }

export default BIDASK_INDICATOR;
