<script setup>
import { ref, onMounted, onBeforeUnmount, watch, nextTick } from 'vue'
import { init, dispose } from 'klinecharts'

const props = defineProps({
  symbol: { type: String, default: '' },
  market: { type: String, default: '' },
  timeframe: { type: String, default: '1H' },
  theme: { type: String, default: 'dark' },
  activeIndicators: { type: Array, default: () => [] },
  realtimeEnabled: { type: Boolean, default: false },
  showBsMarkers: { type: Boolean, default: false },
})

const containerRef = ref(null)
const chart = ref(null)
const loading = ref(false)
const error = ref(null)

// ── DARK THEME ──
const darkThemeStyles = {
  backgroundColor: '#0b0e11',
  grid: {
    show: true,
    horizontal: { show: true, color: '#1a1e25', style: 'dashed', size: 1 },
    vertical: { show: true, color: '#1a1e25', style: 'dashed', size: 1 },
  },
  candle: {
    bar: {
      upColor: '#26a69a', downColor: '#ef5350', noChangeColor: '#5c636b',
      upBorderColor: '#26a69a', downBorderColor: '#ef5350',
      upWickColor: '#26a69a', downWickColor: '#ef5350',
    },
    priceMark: {
      show: true,
      high: { show: true, color: '#26a69a', textMargin: 4 },
      low: { show: true, color: '#ef5350', textMargin: 4 },
      last: { show: true, color: '#e6e8eb', textMargin: 4 },
    },
    tooltip: {
      showRule: 'always',
      showType: 'standard',
      labels: ['Time', 'Open', 'High', 'Low', 'Close', 'Volume'],
    },
  },
  xAxis: {
    axisLine: { show: true, color: '#262b31' },
    tickLine: { show: true, color: '#262b31' },
    tickText: { color: '#5c636b', size: 10 },
  },
  yAxis: {
    axisLine: { show: true, color: '#262b31' },
    tickLine: { show: true, color: '#262b31' },
    tickText: { color: '#5c636b', size: 10 },
  },
  crosshair: {
    show: true,
    horizontal: {
      show: true,
      line: { show: true, color: '#2f6fed', style: 'dashed', size: 1 },
      text: { show: true, color: '#e6e8eb', size: 10, padding: [2, 6], backgroundColor: 'rgba(22,26,32,0.95)', borderRadius: 3 },
    },
    vertical: {
      show: true,
      line: { show: true, color: '#2f6fed', style: 'dashed', size: 1 },
      text: { show: true, color: '#e6e8eb', size: 10, padding: [2, 6], backgroundColor: 'rgba(22,26,32,0.95)', borderRadius: 3 },
    },
  },
}

const lightThemeStyles = {
  backgroundColor: '#ffffff',
  grid: {
    horizontal: { color: '#f0f0f0' },
    vertical: { color: '#f0f0f0' },
  },
  candle: {
    bar: {
      upColor: '#13c2c2', downColor: '#fa541c', noChangeColor: '#888888',
      upBorderColor: '#13c2c2', downBorderColor: '#fa541c',
      upWickColor: '#13c2c2', downWickColor: '#fa541c',
    },
  },
  xAxis: { tickText: { color: '#666' } },
  yAxis: { tickText: { color: '#666' } },
}

const getThemeStyles = () => props.theme === 'dark' ? darkThemeStyles : lightThemeStyles

// ── Mock B/S markers ──
const generateBsMarkers = (data) => {
  if (!props.showBsMarkers || data.length < 20) return []
  const markers = []
  for (let i = 20; i < data.length; i += 15) {
    const isBuy = Math.random() > 0.45
    markers.push({
      timestamp: data[i].timestamp,
      price: isBuy ? data[i].low * 0.998 : data[i].high * 1.002,
      type: isBuy ? 'buy' : 'sell',
    })
  }
  return markers
}

// ── Mock Data ──
const generateMockData = () => {
  const data = []
  const now = new Date()
  let price = 1.0850
  for (let i = 300; i >= 0; i--) {
    const time = new Date(now)
    time.setHours(time.getHours() - i)
    const open = price
    const change = (Math.random() - 0.5) * 0.002
    const close = open + change
    const high = Math.max(open, close) + Math.random() * 0.0005
    const low = Math.min(open, close) - Math.random() * 0.0005
    const volume = Math.floor(Math.random() * 10000) + 1000
    price = close
    data.push({
      timestamp: time.getTime(),
      open: parseFloat(open.toFixed(5)),
      high: parseFloat(high.toFixed(5)),
      low: parseFloat(low.toFixed(5)),
      close: parseFloat(close.toFixed(5)),
      volume,
    })
  }
  return data
}

// ── Indicator configs ──
const indicatorConfigs = {
  SMA: { name: 'SMA', paneId: 'candle_pane' },
  EMA: { name: 'EMA', paneId: 'candle_pane' },
  RSI: { name: 'RSI', paneId: 'rsi_pane' },
  MACD: { name: 'MACD', paneId: 'macd_pane' },
  BB: { name: 'BOLL', paneId: 'candle_pane' },
  ATR: { name: 'ATR', paneId: 'atr_pane' },
  CCI: { name: 'CCI', paneId: 'cci_pane' },
  WR: { name: 'WR', paneId: 'wr_pane' },
  MFI: { name: 'MFI', paneId: 'mfi_pane' },
  ADX: { name: 'ADX', paneId: 'adx_pane' },
  OBV: { name: 'OBV', paneId: 'obv_pane' },
  AD: { name: 'AD', paneId: 'ad_pane' },
  KDJ: { name: 'KDJ', paneId: 'kdj_pane' },
}

const initChart = () => {
  const container = containerRef.value
  if (!container) return

  if (chart.value) {
    try { dispose(container) } catch (e) { /* */ }
    chart.value = null
  }

  try {
    chart.value = init(container, { styles: getThemeStyles() })
  } catch (e) {
    error.value = 'Failed to init chart'
    return
  }

  if (!chart.value) return

  // Create VOL indicator pane
  try {
    chart.value.createIndicator('VOL', false, { height: 96, minHeight: 52 })
  } catch (e) { /* */ }

  // Generate and apply mock data
  const mockData = generateMockData()
  chart.value.applyNewData(mockData)

  // Add mock B/S markers
  const markers = generateBsMarkers(mockData)
  markers.forEach(m => {
    try {
      chart.value.createIndicator('TEXT', false, {
        text: m.type === 'buy' ? 'B' : 'S',
        color: m.type === 'buy' ? '#ffffff' : '#ffffff',
        backgroundColor: m.type === 'buy' ? '#26a69a' : '#ef5350',
        fontSize: 10,
        fontWeight: 'bold',
        paddingLeft: 2, paddingRight: 2,
        paddingTop: 1, paddingBottom: 1,
        point: { timestamp: m.timestamp, value: m.price },
      })
    } catch (e) { /* */ }
  })
}

// ── Watch props ──
watch(() => props.timeframe, () => {
  if (chart.value) {
    chart.value.applyNewData(generateMockData())
  }
})

watch(() => props.theme, () => {
  if (chart.value) {
    try { chart.value.setStyles(getThemeStyles()) } catch (e) { /* */ }
  }
})

onMounted(() => {
  nextTick(() => initChart())
})

onBeforeUnmount(() => {
  if (containerRef.value && chart.value) {
    try { dispose(containerRef.value) } catch (e) { /* */ }
  }
})
</script>

<template>
  <div class="kline-chart-wrapper">
    <!-- Drawing toolbar (left side) -->
    <div class="drawing-toolbar">
      <div
        v-for="tool in [
          { key: 'line', label: 'Trend Line', icon: '╱' },
          { key: 'horizontal', label: 'Horizontal', icon: '—' },
          { key: 'vertical', label: 'Vertical', icon: '|' },
          { key: 'ray', label: 'Ray', icon: '↗' },
          { key: 'fibonacci', label: 'Fibonacci', icon: 'F' },
        ]"
        :key="tool.key"
        class="drawing-tool-btn"
        :title="tool.label"
      >
        {{ tool.icon }}
      </div>
      <a-divider type="vertical" style="margin:4px 0;border-color:rgba(255,255,255,0.12);" />
      <div class="drawing-tool-btn" title="Clear All">✕</div>
    </div>

    <!-- B/S marker legend -->
    <div v-if="showBsMarkers" class="bs-legend">
      <span class="bs-legend-item bs-legend-buy">B Buy</span>
      <span class="bs-legend-item bs-legend-sell">S Sell</span>
    </div>

    <!-- Chart container -->
    <div style="flex:1;min-width:0;position:relative;">
      <div
        v-if="!symbol"
        class="chart-overlay"
      >
        <span>Select a symbol to view chart</span>
      </div>
      <div v-if="loading" class="chart-overlay chart-overlay--loading">
        <a-spin />
      </div>
      <div v-if="error" class="chart-overlay chart-overlay--error">
        {{ error }}
      </div>
      <div ref="containerRef" class="kline-chart-container"></div>
    </div>
  </div>
</template>

<style scoped>
.kline-chart-wrapper {
  display: flex; height: 100%; position: relative;
}
.kline-chart-container {
  width: 100%; height: 100%;
}

/* Drawing toolbar */
.drawing-toolbar {
  display: flex; flex-direction: column; gap: 2px;
  padding: 4px 2px;
  background: rgba(22,26,32,0.95);
  border-radius: 6px;
  border: 1px solid rgba(255,255,255,0.08);
  position: absolute; top: 8px; left: 8px; z-index: 20;
}
.drawing-tool-btn {
  width: 28px; height: 28px;
  display: flex; align-items: center; justify-content: center;
  border-radius: 4px; cursor: pointer;
  font-size: 14px; color: #5c636b;
  transition: all 0.15s;
  user-select: none;
}
.drawing-tool-btn:hover {
  color: #e6e8eb; background: rgba(255,255,255,0.06);
}

/* B/S legend */
.bs-legend {
  position: absolute; top: 8px; right: 8px; z-index: 20;
  display: flex; gap: 8px;
  padding: 4px 8px; border-radius: 4px;
  background: rgba(22,26,32,0.9); border: 1px solid #262b31;
  font-size: 10px; font-weight: 700;
}
.bs-legend-item { display: inline-flex; align-items: center; gap: 3px; }
.bs-legend-buy { color: #26a69a; }
.bs-legend-sell { color: #ef5350; }

/* Overlays */
.chart-overlay {
  position: absolute; inset: 0; display: flex; align-items: center; justify-content: center;
  z-index: 10; color: #5c636b; background: rgba(11,14,17,0.92); font-size: 12px;
}
.chart-overlay--loading { background: rgba(11,14,17,0.5); }
.chart-overlay--error { color: #ef5350; }
</style>
