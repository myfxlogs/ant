/** Dark theme configuration for klinecharts v9.8 — shared across chart instances. */
const DARK_THEME = {
  backgroundColor: '#131722',
  grid: {
    show: true,
    horizontal: { show: true, color: 'rgba(255,255,255,0.06)', style: 'dashed' as const, size: 1, dashedValue: [] },
    vertical: { show: true, color: 'rgba(255,255,255,0.06)', style: 'dashed' as const, size: 1, dashedValue: [] },
  },
  candle: {
    bar: {
      upColor: '#26a69a', downColor: '#ef5350', noChangeColor: '#888888',
      upBorderColor: '#26a69a', downBorderColor: '#ef5350', noChangeBorderColor: '#888888',
      upWickColor: '#26a69a', downWickColor: '#ef5350', noChangeWickColor: '#888888',
    },
    priceMark: { show: true, high: { show: true, color: '#26a69a', textMargin: 4 }, low: { show: true, color: '#ef5350', textMargin: 4 }, last: { show: false } },
  },
  xAxis: {
    axisLine: { show: true, color: 'rgba(255,255,255,0.1)' },
    tickText: { color: '#d1d5db' },
  },
  yAxis: {
    axisLine: { show: true, color: 'rgba(255,255,255,0.1)' },
    tickText: { color: '#d1d5db' },
    highLowPrice: { show: true, color: '#888888', size: 10 },
  },
  crosshair: {
    show: true,
    horizontal: {
      show: true,
      line: { show: true, color: 'rgba(255,255,255,0.3)', style: 'dashed' as const, size: 1, dashedValue: [] },
      text: { show: true, color: '#ffffff', size: 11, padding: [2, 6] as [number, number], backgroundColor: 'rgba(0,0,0,0.8)', borderRadius: 3 },
    },
    vertical: {
      show: true,
      line: { show: true, color: 'rgba(255,255,255,0.3)', style: 'dashed' as const, size: 1, dashedValue: [] },
      text: { show: true, color: '#ffffff', size: 11, padding: [2, 6] as [number, number], backgroundColor: 'rgba(0,0,0,0.8)', borderRadius: 3 },
    },
  },
  tooltip: {
    showRule: 'always' as const,
    showType: 'standard' as const,
    labels: ['Time', 'Open', 'High', 'Low', 'Close', 'Volume'],
    text: { color: '#d1d5db', size: 12, margin: 6 },
  },
  indicator: {
    bars: { upColor: '#26a69a', downColor: '#ef5350' },
  },
};

export default DARK_THEME;
