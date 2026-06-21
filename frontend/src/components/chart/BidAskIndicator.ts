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
  name: 'BIDASK',
  totalStep: 1,
  lock: true,
  needDefaultPointFigure: false,
  needDefaultXAxisFigure: false,
  needDefaultYAxisFigure: false,

  createPointFigures: ({ overlay, coordinates }) => {
    if (!latest || (latest.bid <= 0 && latest.ask <= 0)) {
      // Fallback: gray line at last close.
      const lastCoord = coordinates?.[coordinates.length - 1];
      if (!lastCoord) return [];
      const y = lastCoord.y ?? 0;
      return [horizontalLine(y, '#888888', lastCoord.price)];
    }

    const figures: any[] = [];
    const { bid, ask } = latest;

    // Convert price → y-coordinate using adjacent candles
    const priceToY = (price: number): number | null => {
      if (!coordinates?.length) return null;
      // Find two candles that bracket this price
      let lo = coordinates[0], hi = coordinates[coordinates.length - 1];
      for (const c of coordinates) {
        const p = (c as any).price ?? c.close;
        if (p == null) continue;
        if (p <= price) lo = c;
        if (p >= price) { hi = c; break; }
      }
      const loPrice = (lo as any).price ?? (lo as any).close;
      const hiPrice = (hi as any).price ?? (hi as any).close;
      if (loPrice == null || hiPrice == null || loPrice === hiPrice) return null;
      const ratio = (price - loPrice) / (hiPrice - loPrice);
      return (lo.y ?? 0) - ratio * Math.abs((hi.y ?? 0) - (lo.y ?? 0));
    };

    // Simple fallback: use y-axis range from first/last candle
    const lastC = coordinates[coordinates.length - 1];
    const firstC = coordinates[0];
    const lastPrice = (lastC as any).price ?? (lastC as any).close ?? 0;
    const firstPrice = (firstC as any).price ?? (firstC as any).close ?? 0;
    const yRange = Math.abs((lastC.y ?? 0) - (firstC.y ?? 0));
    const priceRange = Math.abs(lastPrice - firstPrice);
    const yPerPrice = priceRange > 0 ? yRange / priceRange : 0;
    const refY = lastC.y ?? 0;
    const refPrice = lastPrice;

    const yFromPrice = (price: number) => refY - (price - refPrice) * yPerPrice;

    const drawPriceLine = (price: number, color: string) => {
      if (!(price > 0)) return;
      const y = yFromPrice(price);
      const xmin = coordinates[0]?.x ?? 0;
      const xmax = coordinates[coordinates.length - 1]?.x ?? 0;
      if (y < 0 || xmax <= 0) return;

      // Horizontal dashed line
      figures.push({
        type: 'line',
        attrs: { coordinates: [{ x: xmin, y }, { x: xmax, y }] },
        styles: { color, size: 1, style: 'dash' as any, dashedValue: [4, 3] as any },
        ignoreEvent: true,
      });

      // Right-side label pill
      const label = fmt(price);
      if (label) {
        // Estimate label dimensions
        const labelW = label.length * 7 + 10;
        const lx = xmax - labelW;
        figures.push({
          type: 'rect',
          attrs: { x: lx, y: y - 7, width: labelW, height: 14 },
          styles: { color, style: 'fill' as any },
          ignoreEvent: true,
        });
        figures.push({
          type: 'text',
          attrs: { x: lx + 5, y, text: label, align: 'left', baseline: 'middle' },
          styles: { color: '#ffffff', size: 10, font: 'sans-serif' },
          ignoreEvent: true,
        });
      }
    };

    if (bid > 0) drawPriceLine(bid, '#ef5350');
    if (ask > 0) drawPriceLine(ask, '#26a69a');

    return figures;
  },
});
