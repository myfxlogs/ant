/**
 * BacktestTradeOverlay — renders buy/sell markers and fill zones on klinecharts.
 *
 * Registered once at app init via registerOverlay(). Instantiated per backtest
 * run via chart.createOverlay({name: 'backtest_trades', extendData: trades}).
 *
 * Trades format (from BacktestMetrics.trades):
 *   { side: 'buy'|'sell', openPrice: number, closePrice?: number,
 *     openTime: number, closeTime?: number, pnl?: number }
 *
 * Matching: trades are matched to candles by openTime/closeTime timestamp.
 * klinecharts adds .timestamp to each Coordinate at runtime (> type def).
 */
import { registerOverlay } from 'klinecharts';

const ARROW_SIZE = 6;
const BUY_COLOR = '#26a69a';
const SELL_COLOR = '#ef5350';
const BUY_FILL = 'rgba(38,166,154,0.12)';
const SELL_FILL = 'rgba(239,83,80,0.10)';

registerOverlay({
  name: 'backtest_trades',
  totalStep: 1,
  lock: true,
  needDefaultPointFigure: false,
  needDefaultXAxisFigure: false,
  needDefaultYAxisFigure: false,

  createPointFigures: ({ overlay, coordinates }) => {
    const trades: any[] = (overlay as any).extendData || [];
    if (!trades.length || !coordinates?.length) return [];
    const figures: any[] = [];

    // Build timestamp → coordinate lookup.
    // klinecharts adds .timestamp (ms) to each coordinate at runtime.
    const byTs = new Map<number, any>();
    for (const c of coordinates) {
      const ts = (c as any).timestamp;
      if (ts != null) byTs.set(ts, c);
    }

    for (const t of trades) {
      // Match by timestamp (ms). openTime/closeTime may be seconds or ms.
      const openTs = typeof t.openTime === 'number' ? t.openTime : 0;
      const closeTs = typeof t.closeTime === 'number' ? t.closeTime : 0;
      // Normalize: if timestamps look like seconds (< 1e10), convert to ms.
      const openMs = openTs > 1e10 ? openTs : openTs * 1000;
      const closeMs = closeTs > 1e10 ? closeTs : closeTs * 1000;

      const entryPoint = findNearest(byTs, openMs);
      if (!entryPoint) continue;

      const isBuy = t.side?.toLowerCase() === 'buy';
      const color = isBuy ? BUY_COLOR : SELL_COLOR;
      const y = entryPoint.y ?? 0;
      const x = entryPoint.x ?? 0;

      // Entry arrow marker.
      figures.push({
        type: 'polygon',
        attrs: isBuy
          ? {
              coordinates: [
                { x: x - ARROW_SIZE, y: y + ARROW_SIZE + 4 },
                { x, y: y + 4 },
                { x: x + ARROW_SIZE, y: y + ARROW_SIZE + 4 },
              ],
            }
          : {
              coordinates: [
                { x: x - ARROW_SIZE, y: y - ARROW_SIZE - 4 },
                { x, y: y - 4 },
                { x: x + ARROW_SIZE, y: y - ARROW_SIZE - 4 },
              ],
            },
        styles: { color, style: 'fill' as any },
        ignoreEvent: true,
      });

      // Exit marker + fill zone (if trade closed).
      const exitPoint = closeMs ? findNearest(byTs, closeMs) : null;
      if (exitPoint) {
        const ex = exitPoint.x ?? 0;
        const ey = exitPoint.y ?? 0;
        const fill = isBuy ? BUY_FILL : SELL_FILL;

        figures.push({
          type: 'polygon',
          attrs: {
            coordinates: [
              { x, y },
              { x, y: ey },
              { x: ex, y: ey },
              { x: ex, y },
            ],
          },
          styles: {
            color: fill,
            style: 'fill' as any,
            borderColor: color,
            borderSize: 1,
            borderStyle: 'dash' as any,
          },
          ignoreEvent: true,
        });

        // Exit marker (inverted direction vs entry).
        figures.push({
          type: 'polygon',
          attrs: isBuy
            ? {
                coordinates: [
                  { x: ex - ARROW_SIZE, y: ey - ARROW_SIZE - 4 },
                  { x: ex, y: ey - 4 },
                  { x: ex + ARROW_SIZE, y: ey - ARROW_SIZE - 4 },
                ],
              }
            : {
                coordinates: [
                  { x: ex - ARROW_SIZE, y: ey + ARROW_SIZE + 4 },
                  { x: ex, y: ey + 4 },
                  { x: ex + ARROW_SIZE, y: ey + ARROW_SIZE + 4 },
                ],
              },
          styles: { color, style: 'fill' as any },
          ignoreEvent: true,
        });
      }
    }
    return figures;
  },
});

/** Find the coordinate whose timestamp is closest to the target (within 1 bar). */
function findNearest(byTs: Map<number, any>, targetMs: number): any | null {
  // Exact match first.
  const exact = byTs.get(targetMs);
  if (exact) return exact;

  // Fallback: find nearest within 5 minutes (300,000 ms).
  let best: any = null;
  let bestDist = Infinity;
  for (const [ts, c] of byTs) {
    const dist = Math.abs(ts - targetMs);
    if (dist < bestDist) {
      bestDist = dist;
      best = c;
    }
  }
  // Only accept if within reasonable range (1 bar max = 1 day).
  return bestDist <= 86_400_000 ? best : null;
}
