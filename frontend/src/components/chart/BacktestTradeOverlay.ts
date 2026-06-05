/**
 * BacktestTradeOverlay — renders buy/sell markers and fill zones on klinecharts.
 *
 * Registered once at app init via registerOverlay(). Instantiated per backtest
 * run via chart.createOverlay({name: 'backtest_trades', extendData: trades}).
 *
 * Trades format (from BacktestMetrics):
 *   { side: 'buy'|'sell', openPrice: number, closePrice: number,
 *     openTime: number, closeTime: number, pnl: number }
 */
import { registerOverlay } from 'klinecharts';

const ARROW_SIZE = 6;
const BUY_COLOR = '#26a69a';
const SELL_COLOR = '#ef5350';
const BUY_FILL = 'rgba(38,166,154,0.08)';
const SELL_FILL = 'rgba(239,83,80,0.08)';

registerOverlay({
  name: 'backtest_trades',
  totalStep: 1,
  lock: true,
  needDefaultPointFigure: false,
  needDefaultXAxisFigure: false,
  needDefaultYAxisFigure: false,

  createPointFigures: ({ overlay, coordinates, bounding, barSpace, yAxis }) => {
    const trades: any[] = (overlay as any).extendData || [];
    if (!trades.length || !coordinates?.length) return [];
    const figures: any[] = [];

    for (const t of trades) {
      const entryPoint = coordinates.find(
        (c: any) => c?.dataIndex === t.entryIndex
      );
      const exitPoint = coordinates.find(
        (c: any) => c?.dataIndex === t.exitIndex
      );
      if (!entryPoint) continue;

      const isBuy = t.side === 'buy';
      const color = isBuy ? BUY_COLOR : SELL_COLOR;
      const y = (entryPoint as any).y ?? 0;
      const x = (entryPoint as any).x ?? 0;

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

      // Fill zone between entry and exit (if closed).
      if (exitPoint) {
        const ex = (exitPoint as any).x ?? 0;
        const ey = (exitPoint as any).y ?? 0;
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

        // Exit marker.
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
