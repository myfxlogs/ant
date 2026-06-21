import { registerOverlay } from 'klinecharts';

// Module-level state for real-time bid/ask.
let latest: { bid: number; ask: number } | null = null;
let precision = 5;

export function setBidAsk(_tsSec: number, bid: number, ask: number) {
  latest = {
    bid: bid > 0 ? bid : latest?.bid ?? 0,
    ask: ask > 0 ? ask : latest?.ask ?? 0,
  };
}

export function clearBidAsk() {
  latest = null;
  precision = 5;
}

export function setBidAskPrecision(digits: number) {
  if (digits > 0) precision = digits;
}

function fmt(p: number): string {
  return p > 0 ? p.toFixed(precision) : '';
}

registerOverlay({
  name: 'bidask',
  totalStep: 1,
  lock: true,
  needDefaultPointFigure: false,
  needDefaultXAxisFigure: false,
  needDefaultYAxisFigure: false,

  createPointFigures: ({ coordinates }) => {
    if (!coordinates?.length) return [];

    // Build price→y linear mapping from visible candles.
    const coords = coordinates as Array<{ price?: number; x?: number; y?: number }>;
    let pMin = Infinity, pMax = -Infinity, yAtPmin = 0, yAtPmax = 0;
    for (const c of coords) {
      const p = c.price ?? 0;
      if (p <= 0) continue;
      if (p < pMin) { pMin = p; yAtPmin = c.y ?? 0; }
      if (p > pMax) { pMax = p; yAtPmax = c.y ?? 0; }
    }
    if (!isFinite(pMin) || !isFinite(pMax) || pMin === pMax) return [];

    // price↑ → y↓ (top of chart = higher price)
    const priceSpan = pMax - pMin;
    const ySpan = yAtPmin - yAtPmax; // positive if y grows downward
    const priceToY = (price: number) => yAtPmin - ((price - pMin) / priceSpan) * ySpan;

    const xMin = coords[0]?.x ?? 0;
    const xMax = coords[coords.length - 1]?.x ?? 0;
    if (xMax <= xMin) return [];

    const figures: any[] = [];

    const drawPriceLine = (price: number, color: string) => {
      if (!(price > 0)) return;
      const y = priceToY(price);
      // Horizontal dashed line
      figures.push({
        type: 'line',
        attrs: { coordinates: [{ x: xMin, y }, { x: xMax, y }] },
        styles: { color, size: 1, style: 'dash' as any, dashedValue: [4, 3] as any },
        ignoreEvent: true,
      });
      // Right-side label pill
      const label = fmt(price);
      if (!label) return;
      const labelW = label.length * 7 + 10;
      const rx = xMax - labelW;
      figures.push({
        type: 'rect',
        attrs: { x: rx, y: y - 7, width: labelW, height: 14 },
        styles: { color, style: 'fill' as any },
        ignoreEvent: true,
      });
      figures.push({
        type: 'text',
        attrs: { x: rx + 5, y, text: label, align: 'left', baseline: 'middle' },
        styles: { color: '#ffffff', size: 10, font: 'sans-serif' },
        ignoreEvent: true,
      });
    };

    if (latest && (latest.bid > 0 || latest.ask > 0)) {
      drawPriceLine(latest.bid, '#ef5350');
      drawPriceLine(latest.ask, '#26a69a');
    } else {
      // Fallback: gray line at last candle's close
      const last = coords[coords.length - 1];
      const p = last?.price;
      if (p && p > 0) drawPriceLine(p, '#888888');
    }

    return figures;
  },
});
