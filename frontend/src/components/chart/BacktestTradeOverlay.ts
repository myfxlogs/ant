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

interface ChartCoord {
  x: number;
  y: number;
  timestamp?: number;
}

interface TradeEntry {
  side: string;
  openTime: number;
  closeTime?: number;
  openPrice?: number;
  closePrice?: number;
  pnl?: number;
  volume?: number;
}

interface ChartFigure {
  type: string;
  attrs: { coordinates: ChartCoord[] };
  styles: Record<string, unknown>;
  ignoreEvent: boolean;
}

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

  createPointFigures: ({ overlay, coordinates }): ChartFigure[] => {
    const trades = ((overlay as { extendData?: TradeEntry[] }).extendData) ?? [];
    if (!trades.length || !coordinates?.length) return [];

    const byTs = buildTsMap(coordinates as ChartCoord[]);
    const figures: ChartFigure[] = [];
    for (const t of trades) {
      collectTradeFigures(figures, t, byTs);
    }
    return figures;
  },
});

function buildTsMap(coordinates: ChartCoord[]): Map<number, ChartCoord> {
  const byTs = new Map<number, ChartCoord>();
  for (const c of coordinates) {
    const ts = c?.timestamp;
    if (ts != null) byTs.set(ts, c);
  }
  return byTs;
}

function makeArrowPolygon(x: number, y: number, isBuy: boolean, color: string): ChartFigure {
  return {
    type: 'polygon',
    attrs: isBuy
      ? { coordinates: [
          { x: x - ARROW_SIZE, y: y + ARROW_SIZE + 4 },
          { x, y: y + 4 },
          { x: x + ARROW_SIZE, y: y + ARROW_SIZE + 4 },
        ] }
      : { coordinates: [
          { x: x - ARROW_SIZE, y: y - ARROW_SIZE - 4 },
          { x, y: y - 4 },
          { x: x + ARROW_SIZE, y: y - ARROW_SIZE - 4 },
        ] },
    styles: { color, style: 'fill' as unknown },
    ignoreEvent: true,
  };
}

function makeExitArrowPolygon(x: number, y: number, isBuy: boolean, color: string): ChartFigure {
  return {
    type: 'polygon',
    attrs: isBuy
      ? { coordinates: [
          { x: x - ARROW_SIZE, y: y - ARROW_SIZE - 4 },
          { x, y: y - 4 },
          { x: x + ARROW_SIZE, y: y - ARROW_SIZE - 4 },
        ] }
      : { coordinates: [
          { x: x - ARROW_SIZE, y: y + ARROW_SIZE + 4 },
          { x, y: y + 4 },
          { x: x + ARROW_SIZE, y: y + ARROW_SIZE + 4 },
        ] },
    styles: { color, style: 'fill' as unknown },
    ignoreEvent: true,
  };
}

function addExitFigures(figures: ChartFigure[], x: number, y: number, exitPoint: ChartCoord, isBuy: boolean, color: string, fill: string): void {
  const ex = exitPoint.x ?? 0;
  const ey = exitPoint.y ?? 0;

  figures.push({
    type: 'polygon',
    attrs: { coordinates: [{ x, y }, { x, y: ey }, { x: ex, y: ey }, { x: ex, y }] },
    styles: { color: fill, style: 'fill' as unknown, borderColor: color, borderSize: 1, borderStyle: 'dash' as unknown },
    ignoreEvent: true,
  });
  figures.push(makeExitArrowPolygon(ex, ey, isBuy, color));
}

function collectTradeFigures(figures: ChartFigure[], t: TradeEntry, byTs: Map<number, ChartCoord>): void {
  const openTs = typeof t.openTime === 'number' ? t.openTime : 0;
  const closeTs = typeof t.closeTime === 'number' ? t.closeTime : 0;
  const openMs = openTs > 1e10 ? openTs : openTs * 1000;
  const closeMs = closeTs > 1e10 ? closeTs : closeTs * 1000;

  const entryPoint = findNearest(byTs, openMs);
  if (!entryPoint) return;

  const isBuy = t.side?.toLowerCase() === 'buy';
  const color = isBuy ? BUY_COLOR : SELL_COLOR;
  const x = entryPoint.x ?? 0;
  const y = entryPoint.y ?? 0;

  figures.push(makeArrowPolygon(x, y, isBuy, color));

  const exitPoint = closeMs ? findNearest(byTs, closeMs) : null;
  if (exitPoint) {
    addExitFigures(figures, x, y, exitPoint, isBuy, color, isBuy ? BUY_FILL : SELL_FILL);
  }
}

/** Find the coordinate whose timestamp is closest to the target (within 1 bar). */
function findNearest(byTs: Map<number, ChartCoord>, targetMs: number): ChartCoord | null {
  // Exact match first.
  const exact = byTs.get(targetMs);
  if (exact) return exact;

  // Fallback: find nearest within 5 minutes (300,000 ms).
  let best: ChartCoord | null = null;
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
