/** Dark theme for klinecharts v9.8.12 — matches Styles interface shape. */
const DARK_THEME = {
  grid: {
    show: true,
    horizontal: { show: true, color: 'rgba(255,255,255,0.06)', style: 'dashed' as const, size: 1, dashedValue: [4, 4] },
    vertical: { show: true, color: 'rgba(255,255,255,0.06)', style: 'dashed' as const, size: 1, dashedValue: [4, 4] },
  },
  candle: {
    type: 'candle_solid' as const,
    bar: {
      upColor: '#26a69a', downColor: '#ef5350', noChangeColor: '#888888',
      upBorderColor: '#26a69a', downBorderColor: '#ef5350', noChangeBorderColor: '#888888',
      upWickColor: '#26a69a', downWickColor: '#ef5350', noChangeWickColor: '#888888',
    },
    priceMark: {
      show: true,
      high: { show: true, color: '#26a69a', textOffset: 4 },
      low: { show: true, color: '#ef5350', textOffset: 4 },
      last: {
        show: false,
        upColor: '#26a69a', downColor: '#ef5350', noChangeColor: '#888888',
        line: { show: true, style: 'dashed' as const, dashedValue: [4, 4], size: 1 },
        text: { show: true, color: '#d1d5db', size: 12 },
      },
    },
    tooltip: {
      showRule: 'always' as const,
      showType: 'standard' as const,
      defaultValue: '-',
      text: { color: '#d1d5db', size: 12, marginLeft: 6, marginRight: 6, marginTop: 4, marginBottom: 4 },
      icons: [],
    },
  },
  indicator: {
    bars: [
      { upColor: '#26a69a', downColor: '#ef5350', noChangeColor: '#888888' },
    ],
    lastValueMark: {
      show: false,
      text: { show: true, color: '#d1d5db', size: 10 },
    },
    tooltip: {
      showRule: 'always' as const,
      showType: 'standard' as const,
      defaultValue: '-',
      text: { color: '#d1d5db', size: 12 },
      showName: true,
      showParams: true,
      icons: [],
    },
  },
  xAxis: {
    show: true,
    axisLine: { show: true, color: 'rgba(255,255,255,0.1)' },
    tickLine: { show: false, length: 4 },
    tickText: { show: true, color: '#d1d5db', size: 11, marginStart: 4, marginEnd: 4 },
    size: 'auto' as const,
  },
  yAxis: {
    show: true,
    axisLine: { show: true, color: 'rgba(255,255,255,0.1)' },
    tickLine: { show: false, length: 4 },
    tickText: { show: true, color: '#d1d5db', size: 11, marginStart: 4, marginEnd: 4 },
    size: 'auto' as const,
    type: 'normal' as const,
    position: 'right' as const,
  },
  crosshair: {
    show: true,
    horizontal: {
      show: true,
      line: { show: true, color: 'rgba(255,255,255,0.3)', style: 'dashed' as const, size: 1, dashedValue: [4, 4] },
      text: { show: true, color: '#ffffff', size: 11, paddingTop: 2, paddingBottom: 2, paddingLeft: 6, paddingRight: 6, backgroundColor: 'rgba(0,0,0,0.8)', borderRadius: 3 },
    },
    vertical: {
      show: true,
      line: { show: true, color: 'rgba(255,255,255,0.3)', style: 'dashed' as const, size: 1, dashedValue: [4, 4] },
      text: { show: true, color: '#ffffff', size: 11, paddingTop: 2, paddingBottom: 2, paddingLeft: 6, paddingRight: 6, backgroundColor: 'rgba(0,0,0,0.8)', borderRadius: 3 },
    },
  },
};

export default DARK_THEME;
