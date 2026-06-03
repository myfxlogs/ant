<script setup>
import { ref, reactive, computed, onMounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { theme } from 'ant-design-vue'
import KlineChart from '@/components/chart/KlineChart.vue'
import QuickTradePanel from '@/components/workspace/QuickTradePanel.vue'

const { t } = useI18n()

// ---- layout ----
const codePanelVisible = ref(true)
const quickTradeVisible = ref(true)

// ---- Chart / symbol state ----
const accounts = ref([])
const accountId = ref('')
const symbol = ref('')
const market = ref('')
const timeframe = ref('1H')

// ---- Quick Trade state ----
const accountInfo = ref(null)
const qtPositions = ref([])
const qtRecentTrades = ref([])
const qtSubmitting = ref(false)

const handlePlaceOrder = (payload) => {
  qtSubmitting.value = true
  console.log('[QuickTrade] place-order:', payload)
  // TODO: call tradingApi.orderSend(payload) via ConnectRPC
  setTimeout(() => { qtSubmitting.value = false }, 3000)
}

const handleClosePosition = (pos) => {
  console.log('[QuickTrade] close-position:', pos)
  // TODO: call tradingApi.closeOrder(pos.ticket) via ConnectRPC
  const idx = qtPositions.value.findIndex(p => p.ticket === pos.ticket)
  if (idx >= 0) qtPositions.value.splice(idx, 1)
}

// Mock data loader — replace with real ConnectRPC calls
const loadAccountData = () => {
  // Always load demo data (remove guard when real API is connected)
  const activeId = accountId.value || 'demo'
  // TODO: replace with AccountService.GetAccount / MtHubService.GetAccountStatus
  accountInfo.value = {
    balance: 15000, equity: 16250, freeMargin: 12800,
    leverage: 100, currency: 'USD',
  }
  // TODO: replace with MtHubService.OpenedOrders
  if (symbol.value) {
    qtPositions.value = [
      { ticket: 1001, side: 'long', volume: '0.5', openPrice: 72300, markPrice: 72726, profit: 213, leverage: 100 },
      { ticket: 1002, side: 'short', volume: '0.3', openPrice: 73200, markPrice: 72726, profit: 142, leverage: 100 },
    ]
  } else {
    qtPositions.value = []
  }
  // TODO: replace with AnalyticsService.GetRecentTrades
  qtRecentTrades.value = [
    { ticket: 999, symbol: 'EURUSD', side: 'long', closePrice: 1.0865, profit: 45.20, closeTime: '2026-06-03T14:22:00Z' },
    { ticket: 998, symbol: 'BTC/USDT', side: 'short', closePrice: 72150, profit: -120.50, closeTime: '2026-06-03T13:45:00Z' },
    { ticket: 997, symbol: 'XAUUSD', side: 'long', closePrice: 2350.80, profit: 310.00, closeTime: '2026-06-03T11:30:00Z' },
    { ticket: 996, symbol: 'GBPUSD', side: 'short', closePrice: 1.2720, profit: -55.30, closeTime: '2026-06-02T18:15:00Z' },
    { ticket: 995, symbol: 'EURUSD', side: 'long', closePrice: 1.0890, profit: 12.80, closeTime: '2026-06-02T10:00:00Z' },
  ]
}

// ---- Watchlist ----
const watchlist = ref([])
const selectedWatchlistKey = ref('')

// ---- Indicator state ----
const indicatorDropdownVisible = ref(false)
const activeIndicators = ref([])
const availableIndicators = [
  'SMA', 'EMA', 'RSI', 'MACD', 'BB', 'ATR', 'CCI', 'W%R', 'MFI', 'ADX', 'OBV', 'ADOSC', 'AD', 'KDJ',
]

const toggleIndicator = (name) => {
  const idx = activeIndicators.value.indexOf(name)
  if (idx >= 0) {
    activeIndicators.value.splice(idx, 1)
  } else {
    activeIndicators.value.push(name)
  }
}

const indicatorToolbarSummary = computed(() => {
  const n = activeIndicators.value.length
  return n > 0 ? `Indicators (${n})` : 'INDICATORS ▼'
})

// ---- Backtest params ----
const btInitialCapital = ref(10000)
const btLeverage = ref(1)
const btCommission = ref(0.001)
const btSlippage = ref(0.0005)
const btStartDate = ref('')
const btEndDate = ref('')
const datePreset = ref('3M')
const btTradeDirection = ref('both')
const btHighPrecision = ref(false)
const paramsPanelExpanded = ref(true)
const backtestRunning = ref(false)
const backtestStatus = ref('idle')
const backtestMetrics = ref(null)
const backtestError = ref('')
const backtestSubTab = ref('results')

// ---- Smart Tuning ----
const tuneMode = ref('structured')
const structuredTuneMethod = ref('grid')
const tuningRunning = ref(false)
const sweepDimensions = ref([
  { key: 'length', label: 'Period / Length', source: 'code', enabled: true, values: [10, 14, 20, 30, 50, 100] },
  { key: 'mult', label: 'Multiplier', source: 'code', enabled: true, values: [1.5, 2.0, 2.5, 3.0] },
  { key: 'stopLoss', label: 'Stop Loss %', source: 'risk', enabled: false, values: [2, 5, 8, 10, 15] },
  { key: 'takeProfit', label: 'Take Profit %', source: 'risk', enabled: false, values: [3, 5, 10, 15, 20] },
  { key: 'maxPositions', label: 'Max Positions', source: 'risk', enabled: false, values: [1, 3, 5, 10] },
])

// ---- Code ----
const code = ref('')

// ---- AI Generate ----
const aiPrompt = ref('')
const aiGenerating = ref(false)

// ---- computed ----
const enabledSweepDims = computed(() => sweepDimensions.value.filter(d => d.enabled))
const cartesianSize = computed(() => enabledSweepDims.value.reduce((acc, d) => acc * d.values.length, 1))

// ---- date presets ----
const datePresets = [
  { key: '1M', label: '1M', months: 1 },
  { key: '3M', label: '3M', months: 3 },
  { key: '6M', label: '6M', months: 6 },
  { key: '1Y', label: '1Y', months: 12 },
]

const applyDatePreset = (preset) => {
  datePreset.value = preset.key
  const end = new Date()
  const start = new Date()
  start.setMonth(start.getMonth() - preset.months)
  btStartDate.value = start.toISOString().slice(0, 10)
  btEndDate.value = end.toISOString().slice(0, 10)
}

// ---- actions ----
const handleAccountChange = (val) => {
  accountId.value = val
  symbol.value = ''
}

const handleSymbolChange = (val) => { symbol.value = val }

const handleRunBacktest = () => {
  backtestRunning.value = true
  backtestStatus.value = 'running'
  backtestSubTab.value = 'results'
  setTimeout(() => {
    backtestStatus.value = 'completed'
    backtestRunning.value = false
    backtestMetrics.value = {
      totalReturn: 0.1523, annualReturn: 0.0812, maxDrawdown: -0.0635,
      sharpeRatio: 1.84, winRate: 0.58, totalTrades: 47,
    }
  }, 3000)
}

const handleRunTuning = () => {
  tuningRunning.value = true
  setTimeout(() => { tuningRunning.value = false }, 3000)
}

const handleGenerateCode = () => {
  if (!aiPrompt.value.trim()) return
  aiGenerating.value = true
  setTimeout(() => {
    code.value = `# AI-generated strategy based on: "${aiPrompt.value}"\n# TODO: integrate with codeAssistApi.revise\n\ndef initialize():\n    pass\n\ndef on_bar(bar):\n    # Strategy logic here\n    pass\n`
    aiGenerating.value = false
  }, 1500)
}

const toggleSweepDimension = (key) => {
  const dim = sweepDimensions.value.find(d => d.key === key)
  if (dim) dim.enabled = !dim.enabled
}

const formatSweepValues = (values) => values.join(', ')

onMounted(() => {
  // Fetch accounts, watchlist, etc. in real app
  loadAccountData()
})

watch(accountId, () => { loadAccountData() })
</script>

<template>
  <!-- Dark theme scope: ConfigProvider with darkAlgorithm -->
  <a-config-provider
    :theme="{
      algorithm: theme.darkAlgorithm,
      token: {
        colorBgContainer: '#161a20',
        colorBgElevated: '#1c2128',
        colorBorder: '#262b31',
        colorPrimary: '#2f6fed',
        colorBgLayout: '#0b0e11',
        colorText: '#e6e8eb',
        colorTextSecondary: '#8b949e',
        colorTextTertiary: '#5c636b',
        borderRadius: 8,
      },
    }"
  >
    <div class="ide-workspace">
      <!-- ═══ TOP TOOLBAR ═══ -->
      <div class="ide-toolbar">
        <!-- Watchlist group -->
        <div class="ide-toolbar-group">
          <span class="ide-toolbar-label">Watchlist</span>
          <div class="ide-toolbar-row">
            <a-select
              v-model:value="accountId"
              size="small"
              style="width:200px"
              placeholder="Select account"
              show-search
              :filter-option="false"
              @change="handleAccountChange"
            >
              <a-select-option v-for="a in accounts" :key="a.id" :value="a.id">
                {{ a.brokerServer }} · {{ a.login }}
              </a-select-option>
            </a-select>
            <a-select
              v-model:value="symbol"
              size="small"
              style="width:120px"
              placeholder="Symbol"
              show-search
              @change="handleSymbolChange"
            >
              <a-select-option value="BTC/USDT">BTC/USDT</a-select-option>
              <a-select-option value="EURUSD">EURUSD</a-select-option>
              <a-select-option value="GBPUSD">GBPUSD</a-select-option>
              <a-select-option value="XAUUSD">XAUUSD</a-select-option>
            </a-select>
          </div>
        </div>

        <!-- Timeframe group -->
        <div class="ide-toolbar-group ide-toolbar-group--tf">
          <span class="ide-toolbar-label">Timeframe</span>
          <a-radio-group v-model:value="timeframe" size="small" button-style="solid" class="ide-tf-seg">
            <a-radio-button value="1m">1m</a-radio-button>
            <a-radio-button value="5m">5m</a-radio-button>
            <a-radio-button value="15m">15m</a-radio-button>
            <a-radio-button value="1H">1H</a-radio-button>
            <a-radio-button value="4H">4H</a-radio-button>
            <a-radio-button value="1D">1D</a-radio-button>
            <a-radio-button value="1W">1W</a-radio-button>
          </a-radio-group>
        </div>

        <!-- Indicator group -->
        <div class="ide-toolbar-group ide-toolbar-group--indicator">
          <span class="ide-toolbar-label">Indicator</span>
          <a-dropdown v-model:open="indicatorDropdownVisible" placement="bottomLeft" :trigger="['click']">
            <a-button size="small" class="ide-indicator-btn">
              <span class="ide-indicator-trigger-text">{{ indicatorToolbarSummary }}</span>
            </a-button>
            <template #overlay>
              <div class="ide-indicator-overlay" @click.stop>
                <div class="ide-indicator-tags">
                  <span
                    v-for="ind in availableIndicators" :key="ind"
                    class="ide-indicator-tag"
                    :class="{ active: activeIndicators.includes(ind) }"
                    @click="toggleIndicator(ind)"
                  >{{ ind }}</span>
                </div>
              </div>
            </template>
          </a-dropdown>
        </div>

        <!-- Spacer -->
        <div style="flex:1;"></div>

        <!-- Actions -->
        <div class="ide-toolbar-actions">
          <a-tooltip :title="codePanelVisible ? 'Hide Code' : 'Show Code'">
            <a-button size="small" class="ide-toolbar-btn" @click="codePanelVisible = !codePanelVisible">
              &lt;/&gt;
            </a-button>
          </a-tooltip>
          <a-button
            size="small"
            type="primary"
            class="ide-toolbar-qt-btn"
            @click="quickTradeVisible = !quickTradeVisible"
          >
            ⚡ Quick Trade
          </a-button>
        </div>
      </div>

      <!-- ═══ THREE-COLUMN BODY ═══ -->
      <div class="ide-body">
        <!-- ═══ LEFT: Code Panel ═══ -->
        <div v-if="codePanelVisible" class="ide-col-left">
          <div class="ide-code-panel">
            <!-- Code panel header -->
            <div class="ide-code-header">
              <div class="ide-code-header-left">
                <span class="ide-code-title">&lt;/&gt; Python Strategy</span>
                <a-tag v-if="true" color="green" class="ide-code-badge">Purchased</a-tag>
              </div>
              <div class="ide-code-header-actions">
                <a-tooltip title="Save">
                  <a-button size="small" type="text" class="ide-code-action-btn">💾</a-button>
                </a-tooltip>
                <a-tooltip title="Validate">
                  <a-button size="small" type="text" class="ide-code-action-btn">✅</a-button>
                </a-tooltip>
                <a-tooltip title="Copy">
                  <a-button size="small" type="text" class="ide-code-action-btn">📋</a-button>
                </a-tooltip>
                <a-tooltip title="Run">
                  <a-button size="small" type="text" class="ide-code-action-btn">▶️</a-button>
                </a-tooltip>
              </div>
            </div>

            <!-- Code editor -->
            <div class="ide-code-body">
              <textarea
                v-model="code"
                class="ide-code-editor"
                placeholder="# Write your Python strategy code here..."
                spellcheck="false"
              ></textarea>
            </div>

            <!-- AI Generate panel -->
            <div class="ide-ai-panel">
              <div class="ide-ai-header">
                <span class="ide-ai-title">🤖 AI Generate</span>
              </div>
              <div class="ide-ai-body">
                <a-textarea
                  v-model:value="aiPrompt"
                  :rows="3"
                  placeholder="Describe the indicator you want to generate..."
                  class="ide-ai-textarea"
                />
                <a-button
                  type="primary"
                  block
                  size="small"
                  :loading="aiGenerating"
                  :disabled="!aiPrompt.trim()"
                  class="ide-ai-generate-btn"
                  @click="handleGenerateCode"
                >
                  ⚡ Generate Code
                </a-button>
              </div>
            </div>
          </div>
        </div>

        <!-- Code rail (when collapsed) -->
        <div v-if="!codePanelVisible" class="ide-code-rail" @click="codePanelVisible = true">
          <span class="ide-code-rail-label">&lt;/&gt;</span>
        </div>

        <!-- ═══ MIDDLE: Chart + Backtest ═══ -->
        <div class="ide-col-mid">
          <!-- Indicator tags row -->
          <div class="ide-indicator-tags-row">
            <span
              v-for="ind in activeIndicators" :key="ind"
              class="ide-indicator-chip"
              :class="{ active: true }"
            >
              {{ ind }}
              <span class="ide-indicator-chip-close" @click="toggleIndicator(ind)">×</span>
            </span>
            <span v-if="activeIndicators.length === 0" class="ide-indicator-tags-hint">
              Select indicators from the toolbar above
            </span>
          </div>

          <!-- Chart area -->
          <div class="ide-chart-area">
            <!-- OHLC info line -->
            <div class="ide-ohlc-line" v-if="symbol">
              <span class="ide-ohlc-item"><b>O</b> <span class="ide-ohlc-val">1.0850</span></span>
              <span class="ide-ohlc-item"><b>H</b> <span class="ide-ohlc-val">1.0872</span></span>
              <span class="ide-ohlc-item"><b>L</b> <span class="ide-ohlc-val">1.0831</span></span>
              <span class="ide-ohlc-item"><b>C</b> <span class="ide-ohlc-val up">1.0865</span></span>
              <span class="ide-ohlc-item"><b>V</b> <span class="ide-ohlc-val">12,847</span></span>
            </div>

            <KlineChart
              :symbol="symbol"
              :market="market"
              :timeframe="timeframe"
              theme="dark"
              :active-indicators="activeIndicators"
              :realtime-enabled="true"
              :show-bs-markers="true"
            />
          </div>

          <!-- ═══ Backtest Section ═══ -->
          <div class="ide-backtest-section">
            <!-- Backtest Params Card -->
            <div class="ide-bt-params-card">
              <div class="ide-bt-params-header" @click="paramsPanelExpanded = !paramsPanelExpanded">
                <div class="ide-bt-params-title">
                  <span>⚡</span>
                  <span>Backtest Parameters</span>
                </div>
                <div class="ide-bt-params-actions" @click.stop>
                  <a-button size="small" class="ide-bt-settings-btn" title="Settings">⚙</a-button>
                  <a-button
                    type="primary" size="small"
                    :loading="backtestRunning"
                    :disabled="!code || !symbol"
                    class="ide-bt-run-btn"
                    @click="handleRunBacktest"
                  >
                    ▶ Run Backtest
                  </a-button>
                  <span class="ide-bt-collapse-icon">{{ paramsPanelExpanded ? '▲' : '▼' }}</span>
                </div>
              </div>

              <div v-show="paramsPanelExpanded" class="ide-bt-params-body">
                <div class="ide-bt-params-grid">
                  <!-- Date Range -->
                  <div class="ide-bt-param-group">
                    <div class="ide-bt-param-label">Date Range</div>
                    <div class="ide-bt-date-presets">
                      <a-button
                        v-for="p in datePresets" :key="p.key"
                        size="small"
                        :type="datePreset === p.key ? 'primary' : 'default'"
                        @click="applyDatePreset(p)"
                      >{{ p.label }}</a-button>
                    </div>
                    <a-row :gutter="8" style="margin-top:6px;">
                      <a-col :span="12">
                        <a-date-picker v-model:value="btStartDate" size="small" style="width:100%" placeholder="Start" />
                      </a-col>
                      <a-col :span="12">
                        <a-date-picker v-model:value="btEndDate" size="small" style="width:100%" placeholder="End" />
                      </a-col>
                    </a-row>
                  </div>

                  <!-- Capital & Leverage -->
                  <div class="ide-bt-param-group">
                    <div class="ide-bt-param-label">Capital & Leverage</div>
                    <a-row :gutter="8">
                      <a-col :span="12">
                        <div class="ide-bt-field-label">Initial Capital</div>
                        <a-input-number v-model:value="btInitialCapital" size="small" style="width:100%" :min="100" :step="1000" :precision="2" />
                      </a-col>
                      <a-col :span="12">
                        <div class="ide-bt-field-label">Leverage</div>
                        <a-input-number v-model:value="btLeverage" size="small" style="width:100%" :min="1" :max="125" :step="1" />
                      </a-col>
                    </a-row>
                    <a-row :gutter="8" style="margin-top:6px;">
                      <a-col :span="12">
                        <div class="ide-bt-field-label">Commission %</div>
                        <a-input-number v-model:value="btCommission" size="small" style="width:100%" :min="0" :max="10" :step="0.01" :precision="4" />
                      </a-col>
                      <a-col :span="12">
                        <div class="ide-bt-field-label">Slippage %</div>
                        <a-input-number v-model:value="btSlippage" size="small" style="width:100%" :min="0" :max="10" :step="0.01" :precision="4" />
                      </a-col>
                    </a-row>
                  </div>

                  <!-- Trade Direction & High Precision -->
                  <div class="ide-bt-param-group">
                    <div class="ide-bt-param-label">Trade Direction</div>
                    <a-radio-group v-model:value="btTradeDirection" size="small" button-style="solid" class="ide-bt-dir-seg">
                      <a-radio-button value="long">↑ Long</a-radio-button>
                      <a-radio-button value="short">↓ Short</a-radio-button>
                      <a-radio-button value="both">Both</a-radio-button>
                    </a-radio-group>
                    <div style="margin-top:8px;display:flex;align-items:center;gap:8px;">
                      <a-switch v-model:checked="btHighPrecision" size="small" />
                      <span class="ide-bt-field-label" style="margin:0;">High-Precision M1</span>
                    </div>
                  </div>
                </div>
              </div>
            </div>

            <!-- Backtest Results / Smart Tuning Tabs -->
            <div class="ide-bt-tabs">
              <div class="ide-bt-tabs-nav">
                <span
                  class="ide-bt-tab"
                  :class="{ active: backtestSubTab === 'results' }"
                  @click="backtestSubTab = 'results'"
                >Backtest Results</span>
                <span
                  class="ide-bt-tab"
                  :class="{ active: backtestSubTab === 'tuning' }"
                  @click="backtestSubTab = 'tuning'"
                >Smart Tuning</span>
              </div>

              <div class="ide-bt-tabs-body">
                <!-- Backtest Results -->
                <div v-show="backtestSubTab === 'results'" class="ide-bt-results">
                  <div v-if="backtestStatus === 'idle'" class="ide-bt-empty">
                    <p>Run a backtest to see results</p>
                  </div>

                  <div v-if="backtestStatus === 'running'" class="ide-bt-running">
                    <a-spin size="default" />
                    <p class="ide-bt-running-text">Backtest running...</p>
                  </div>

                  <div v-if="backtestStatus === 'completed' && backtestMetrics" class="ide-bt-metrics">
                    <div class="ide-bt-metrics-grid">
                      <div class="ide-bt-metric-card" :class="{ positive: backtestMetrics.totalReturn >= 0, negative: backtestMetrics.totalReturn < 0 }">
                        <div class="ide-bt-metric-label">Total Return</div>
                        <div class="ide-bt-metric-value">{{ (backtestMetrics.totalReturn * 100).toFixed(2) }}%</div>
                      </div>
                      <div class="ide-bt-metric-card">
                        <div class="ide-bt-metric-label">Annual Return</div>
                        <div class="ide-bt-metric-value">{{ (backtestMetrics.annualReturn * 100).toFixed(2) }}%</div>
                      </div>
                      <div class="ide-bt-metric-card negative">
                        <div class="ide-bt-metric-label">Max Drawdown</div>
                        <div class="ide-bt-metric-value">{{ (backtestMetrics.maxDrawdown * 100).toFixed(2) }}%</div>
                      </div>
                      <div class="ide-bt-metric-card">
                        <div class="ide-bt-metric-label">Sharpe</div>
                        <div class="ide-bt-metric-value">{{ backtestMetrics.sharpeRatio.toFixed(2) }}</div>
                      </div>
                      <div class="ide-bt-metric-card" :class="{ positive: backtestMetrics.winRate >= 0.5 }">
                        <div class="ide-bt-metric-label">Win Rate</div>
                        <div class="ide-bt-metric-value">{{ (backtestMetrics.winRate * 100).toFixed(2) }}%</div>
                      </div>
                      <div class="ide-bt-metric-card">
                        <div class="ide-bt-metric-label">Total Trades</div>
                        <div class="ide-bt-metric-value">{{ backtestMetrics.totalTrades }}</div>
                      </div>
                    </div>

                    <!-- Equity curve placeholder -->
                    <div class="ide-bt-equity-placeholder">
                      <div class="ide-bt-equity-inner">
                        📈 Equity Curve ({{ backtestMetrics.totalTrades }} trades)
                      </div>
                    </div>
                  </div>
                </div>

                <!-- Smart Tuning -->
                <div v-show="backtestSubTab === 'tuning'" class="ide-bt-tuning">
                  <div class="ide-tuning-header">
                    <span class="ide-tuning-icon">🧪</span>
                    <div>
                      <div class="ide-tuning-title">Smart Tuning</div>
                      <div class="ide-tuning-subtitle">Automatically search the optimal strategy parameters...</div>
                    </div>
                  </div>

                  <!-- Structured Search -->
                  <div class="ide-tuning-section">
                    <div class="ide-tuning-section-title">Structured scan (no LLM)</div>
                    <div class="ide-tuning-method-row">
                      <a-radio-group v-model:value="structuredTuneMethod" size="small" button-style="solid" class="ide-tuning-method-seg">
                        <a-radio-button value="grid">Grid</a-radio-button>
                        <a-radio-button value="random">Random</a-radio-button>
                      </a-radio-group>
                      <a-button
                        type="primary" size="small"
                        :loading="tuningRunning"
                        :disabled="!code || !symbol"
                        class="ide-tuning-run-btn"
                        @click="handleRunTuning"
                      >
                        ▶ Run Smart Tuning
                      </a-button>
                    </div>

                    <!-- Sweep dimensions -->
                    <div class="ide-tuning-dims">
                      <div class="ide-tuning-dims-header">
                        <span>📐 Sweep Dimensions</span>
                        <span class="ide-tuning-dims-stats">
                          {{ enabledSweepDims.length }}/{{ sweepDimensions.length }} enabled ·
                          Cartesian: {{ cartesianSize.toLocaleString() }}
                        </span>
                      </div>
                      <label
                        v-for="d in sweepDimensions" :key="d.key"
                        class="ide-tuning-dim-row"
                        :class="{ disabled: !d.enabled }"
                      >
                        <input type="checkbox" :checked="d.enabled" @change="toggleSweepDimension(d.key)" />
                        <span class="ide-tuning-dim-name">{{ d.label }}</span>
                        <span class="ide-tuning-dim-badge" :class="'badge--' + d.source">{{ d.source.toUpperCase() }}</span>
                        <span class="ide-tuning-dim-count">×{{ d.values.length }}</span>
                        <span class="ide-tuning-dim-values">{{ formatSweepValues(d.values) }}</span>
                      </label>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- ═══ RIGHT: Quick Trade ═══ -->
        <div v-show="quickTradeVisible" class="ide-col-right">
          <QuickTradePanel
            :symbol="symbol"
            :market="market"
            :timeframe="timeframe"
            :account-id="accountId"
            :account-info="accountInfo"
            :positions="qtPositions"
            :recent-trades="qtRecentTrades"
            @close="quickTradeVisible = false"
            @place-order="handlePlaceOrder"
            @close-position="handleClosePosition"
          />
        </div>
      </div>
    </div>
  </a-config-provider>
</template>

<!-- ═══ DARK THEME CSS ═══ -->
<style>
/* ── V31 Dark Palette ── */
:root {
  --ide-bg: #0b0e11;
  --ide-card: #161a20;
  --ide-card-inner: #1c2128;
  --ide-border: #262b31;
  --ide-text: #e6e8eb;
  --ide-text-secondary: #8b949e;
  --ide-text-tertiary: #5c636b;
  --ide-primary: #2f6fed;
  --ide-up: #26a69a;
  --ide-down: #ef5350;
}

/* ── App Root ── */
body {
  background: var(--ide-bg) !important;
}

/* ── Main layout ── */
.ide-workspace {
  height: 100vh; width: 100vw;
  display: flex; flex-direction: column;
  overflow: hidden;
  background: var(--ide-bg);
  color: var(--ide-text);
}

/* ── TOP TOOLBAR ── */
.ide-toolbar {
  flex-shrink: 0;
  display: flex; align-items: flex-end; gap: 10px;
  padding: 8px 12px 10px;
  background: var(--ide-card);
  border-bottom: 1px solid var(--ide-border);
}
.ide-toolbar-group {
  display: flex; flex-direction: column; gap: 4px; min-width: 0;
  padding: 5px 10px 7px;
  border-radius: 8px;
  background: var(--ide-card-inner);
  border: 1px solid var(--ide-border);
}
.ide-toolbar-label {
  font-size: 9px; font-weight: 700; letter-spacing: 0.06em;
  text-transform: uppercase; color: var(--ide-text-tertiary); line-height: 1;
}
.ide-toolbar-row { display: flex; gap: 6px; }
.ide-toolbar-group--tf { flex: 0 0 auto; }
.ide-toolbar-group--indicator { flex: 1 1 200px; min-width: 160px; }
.ide-toolbar-actions { display: flex; align-items: center; gap: 8px; flex-shrink: 0; padding-bottom: 2px; }
.ide-toolbar-btn {
  border-radius: 6px !important; width: 30px; height: 30px !important; padding: 0 !important;
  font-size: 13px;
}
.ide-toolbar-qt-btn {
  border-radius: 6px !important; font-weight: 600; height: 30px !important;
  padding: 0 14px !important; display: inline-flex !important; align-items: center; gap: 6px;
  background: var(--ide-primary) !important; border-color: var(--ide-primary) !important;
  box-shadow: 0 2px 8px rgba(47,111,237,0.3);
}

/* ── TF segmented ── */
.ide-tf-seg .ant-radio-button-wrapper {
  padding: 0 8px; font-size: 11px; height: 28px; line-height: 26px;
  border-color: var(--ide-border); color: var(--ide-text-secondary);
  background: var(--ide-card-inner);
}
.ide-tf-seg .ant-radio-button-wrapper:first-child { border-radius: 6px 0 0 6px; }
.ide-tf-seg .ant-radio-button-wrapper:last-child { border-radius: 0 6px 6px 0; }
.ide-tf-seg .ant-radio-button-wrapper-checked:not(.ant-radio-button-wrapper-disabled) {
  background: var(--ide-primary) !important; border-color: var(--ide-primary) !important;
  color: #fff !important; box-shadow: 0 1px 4px rgba(47,111,237,0.4);
}
.ide-tf-seg .ant-radio-button-wrapper:not(:first-child)::before { background: var(--ide-border) !important; }

/* ── Indicator button ── */
.ide-indicator-btn {
  width: 100%; display: inline-flex; align-items: center; justify-content: space-between;
  height: 28px; padding: 0 10px; border-radius: 6px;
  background: var(--ide-card-inner) !important;
  border-color: var(--ide-border) !important; color: var(--ide-text-secondary) !important;
}
.ide-indicator-trigger-text { flex: 1; text-align: left; font-size: 11px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.ide-indicator-overlay { padding: 10px 12px; min-width: 260px; background: var(--ide-card) !important; border: 1px solid var(--ide-border); border-radius: 8px; }
.ide-indicator-tags { display: flex; flex-wrap: wrap; gap: 4px; }
.ide-indicator-tag {
  padding: 3px 8px; border-radius: 4px; font-size: 10px; font-weight: 600;
  cursor: pointer; user-select: none;
  background: var(--ide-card-inner); color: var(--ide-text-secondary);
  border: 1px solid var(--ide-border);
  transition: all 0.15s;
}
.ide-indicator-tag:hover { border-color: var(--ide-primary); color: var(--ide-primary); }
.ide-indicator-tag.active { background: rgba(47,111,237,0.15); border-color: var(--ide-primary); color: var(--ide-primary); }

/* ── THREE-COLUMN BODY ── */
.ide-body {
  flex: 1 1 0; display: flex; overflow: hidden; min-height: 0;
}

/* ── LEFT: Code ── */
.ide-col-left {
  width: 25%; min-width: 280px; max-width: 420px; flex-shrink: 0;
  display: flex; flex-direction: column;
  border-right: 1px solid var(--ide-border);
  background: var(--ide-card);
}
.ide-code-panel { display: flex; flex-direction: column; height: 100%; overflow: hidden; }
.ide-code-header {
  flex-shrink: 0;
  display: flex; align-items: center; justify-content: space-between;
  padding: 8px 12px;
  border-bottom: 1px solid var(--ide-border);
  background: var(--ide-card-inner);
}
.ide-code-header-left { display: flex; align-items: center; gap: 8px; }
.ide-code-title { font-size: 12px; font-weight: 700; color: var(--ide-text); }
.ide-code-badge { font-size: 9px; font-weight: 600; }
.ide-code-header-actions { display: flex; gap: 2px; }
.ide-code-action-btn {
  border: none !important; background: transparent !important;
  color: var(--ide-text-tertiary) !important; font-size: 13px;
  width: 26px; height: 26px; padding: 0 !important;
}
.ide-code-action-btn:hover { color: var(--ide-text) !important; background: rgba(255,255,255,0.06) !important; }

.ide-code-body { flex: 1 1 0; overflow: hidden; }
.ide-code-editor {
  width: 100%; height: 100%; min-height: 200px;
  border: none; padding: 12px; resize: none;
  font-family: 'Menlo', 'Monaco', 'Consolas', monospace; font-size: 12px;
  line-height: 1.55; color: var(--ide-text);
  background: var(--ide-card);
}
.ide-code-editor::placeholder { color: var(--ide-text-tertiary); }
.ide-code-editor:focus { outline: none; }

/* AI Generate */
.ide-ai-panel {
  flex-shrink: 0;
  border-top: 1px solid var(--ide-border);
  background: var(--ide-card-inner);
}
.ide-ai-header {
  padding: 8px 12px;
  border-bottom: 1px solid var(--ide-border);
}
.ide-ai-title { font-size: 11px; font-weight: 700; color: var(--ide-text); }
.ide-ai-body { padding: 10px 12px; display: flex; flex-direction: column; gap: 8px; }
.ide-ai-textarea textarea {
  background: var(--ide-card) !important;
  border-color: var(--ide-border) !important;
  color: var(--ide-text) !important;
  font-size: 11px;
}
.ide-ai-textarea textarea::placeholder { color: var(--ide-text-tertiary) !important; }
.ide-ai-generate-btn {
  border-radius: 6px !important; font-weight: 600 !important;
  background: var(--ide-primary) !important; border-color: var(--ide-primary) !important;
  box-shadow: 0 2px 8px rgba(47,111,237,0.25);
}

/* Code rail (collapsed) */
.ide-code-rail {
  flex: 0 0 36px; width: 36px; min-width: 36px;
  display: flex; align-items: center; justify-content: center;
  cursor: pointer; border-right: 1px solid var(--ide-border);
  background: var(--ide-card);
}
.ide-code-rail:hover { background: var(--ide-card-inner); }
.ide-code-rail-label {
  writing-mode: vertical-rl; font-size: 10px; font-weight: 600;
  color: var(--ide-text-secondary); letter-spacing: 0.04em;
}

/* ── MIDDLE: Chart + Backtest ── */
.ide-col-mid {
  flex: 1 1 0; min-width: 0;
  display: flex; flex-direction: column;
  overflow: hidden;
}

/* Indicator tags row */
.ide-indicator-tags-row {
  flex-shrink: 0;
  display: flex; flex-wrap: wrap; gap: 4px;
  padding: 6px 10px;
  background: var(--ide-card);
  border-bottom: 1px solid var(--ide-border);
  min-height: 32px; align-items: center;
}
.ide-indicator-chip {
  padding: 2px 6px; border-radius: 3px; font-size: 10px; font-weight: 600;
  background: rgba(47,111,237,0.12); color: var(--ide-primary);
  border: 1px solid rgba(47,111,237,0.25);
  display: inline-flex; align-items: center; gap: 4px;
}
.ide-indicator-chip-close { cursor: pointer; font-size: 12px; opacity: 0.6; }
.ide-indicator-chip-close:hover { opacity: 1; }
.ide-indicator-tags-hint { font-size: 10px; color: var(--ide-text-tertiary); }

/* Chart area */
.ide-chart-area {
  flex: 1 1 0; min-height: 0; display: flex; flex-direction: column;
  position: relative;
}

/* OHLC info line */
.ide-ohlc-line {
  position: absolute; top: 4px; left: 50%; transform: translateX(-50%); z-index: 10;
  display: flex; gap: 12px; padding: 3px 14px;
  background: rgba(22,26,32,0.92); border: 1px solid var(--ide-border);
  border-radius: 6px; font-size: 10px;
}
.ide-ohlc-item { color: var(--ide-text-secondary); }
.ide-ohlc-item b { color: var(--ide-text-tertiary); margin-right: 2px; }
.ide-ohlc-val { color: var(--ide-text); font-weight: 600; }
.ide-ohlc-val.up { color: var(--ide-up); }

/* ── Backtest Section ── */
.ide-backtest-section {
  flex-shrink: 0;
  border-top: 1px solid var(--ide-border);
  background: var(--ide-card);
  max-height: 45%; overflow-y: auto;
}

/* Backtest params card */
.ide-bt-params-card {
  border-bottom: 1px solid var(--ide-border);
}
.ide-bt-params-header {
  display: flex; align-items: center; justify-content: space-between;
  padding: 9px 14px; cursor: pointer; user-select: none;
  background: var(--ide-card-inner);
}
.ide-bt-params-header:hover { background: rgba(47,111,237,0.06); }
.ide-bt-params-title { display: flex; align-items: center; gap: 8px; font-size: 12px; font-weight: 700; color: var(--ide-text); }
.ide-bt-params-actions { display: flex; align-items: center; gap: 8px; }
.ide-bt-settings-btn { border-radius: 6px !important; }
.ide-bt-run-btn {
  border-radius: 6px !important; font-weight: 600 !important;
  background: var(--ide-primary) !important; border-color: var(--ide-primary) !important;
  box-shadow: 0 2px 8px rgba(47,111,237,0.25);
}
.ide-bt-collapse-icon { font-size: 9px; color: var(--ide-text-tertiary); cursor: pointer; padding: 4px; }

.ide-bt-params-body { padding: 12px 14px; }
.ide-bt-params-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 14px; }
.ide-bt-param-group { display: flex; flex-direction: column; gap: 4px; }
.ide-bt-param-label { font-size: 9px; font-weight: 700; text-transform: uppercase; color: var(--ide-text-tertiary); margin-bottom: 2px; }
.ide-bt-field-label { font-size: 9px; font-weight: 600; color: var(--ide-text-tertiary); text-transform: uppercase; margin-bottom: 2px; }
.ide-bt-date-presets { display: flex; gap: 4px; }

/* Trade Direction segmented */
.ide-bt-dir-seg .ant-radio-button-wrapper {
  padding: 0 10px; font-size: 11px; height: 26px; line-height: 24px;
  border-color: var(--ide-border); color: var(--ide-text-secondary);
  background: var(--ide-card-inner);
}
.ide-bt-dir-seg .ant-radio-button-wrapper:first-child { border-radius: 6px 0 0 6px; }
.ide-bt-dir-seg .ant-radio-button-wrapper:last-child { border-radius: 0 6px 6px 0; }
.ide-bt-dir-seg .ant-radio-button-wrapper-checked:not(.ant-radio-button-wrapper-disabled) {
  border-color: var(--ide-up) !important; color: var(--ide-up) !important;
  background: rgba(38,166,154,0.12) !important;
}
.ide-bt-dir-seg .ant-radio-button-wrapper[value="short"].ant-radio-button-wrapper-checked {
  border-color: var(--ide-down) !important; color: var(--ide-down) !important;
  background: rgba(239,83,80,0.12) !important;
}
.ide-bt-dir-seg .ant-radio-button-wrapper[value="both"].ant-radio-button-wrapper-checked {
  border-color: var(--ide-primary) !important; color: var(--ide-primary) !important;
  background: rgba(47,111,237,0.12) !important;
}
.ide-bt-dir-seg .ant-radio-button-wrapper:not(:first-child)::before { background: var(--ide-border) !important; }

/* Backtest Tabs */
.ide-bt-tabs {
  display: flex; flex-direction: column;
}
.ide-bt-tabs-nav {
  display: flex; gap: 0;
  padding: 0 14px;
  background: var(--ide-card-inner);
  border-bottom: 1px solid var(--ide-border);
}
.ide-bt-tab {
  padding: 8px 16px; font-size: 12px; font-weight: 600;
  cursor: pointer; user-select: none;
  color: var(--ide-text-tertiary);
  border-bottom: 2px solid transparent;
  transition: all 0.15s;
}
.ide-bt-tab:hover { color: var(--ide-text-secondary); }
.ide-bt-tab.active {
  color: var(--ide-primary);
  border-bottom-color: var(--ide-primary);
}
.ide-bt-tabs-body { flex: 1; overflow-y: auto; }

/* Results */
.ide-bt-results { padding: 14px; }
.ide-bt-empty { display: flex; align-items: center; justify-content: center; min-height: 80px; }
.ide-bt-empty p { font-size: 12px; color: var(--ide-text-tertiary); }
.ide-bt-running { display: flex; flex-direction: column; align-items: center; justify-content: center; min-height: 80px; gap: 10px; }
.ide-bt-running-text { font-size: 12px; color: var(--ide-text-secondary); }

.ide-bt-metrics-grid {
  display: grid; grid-template-columns: repeat(auto-fit, minmax(100px, 1fr)); gap: 8px; margin-bottom: 12px;
}
.ide-bt-metric-card {
  background: var(--ide-card-inner); border-radius: 8px; padding: 10px 8px; text-align: center;
  border: 1px solid var(--ide-border);
}
.ide-bt-metric-label { font-size: 9px; color: var(--ide-text-tertiary); margin-bottom: 4px; text-transform: uppercase; letter-spacing: 0.03em; font-weight: 600; }
.ide-bt-metric-value { font-size: 16px; font-weight: 700; color: var(--ide-text); line-height: 1.2; }
.ide-bt-metric-card.positive .ide-bt-metric-value { color: var(--ide-up); }
.ide-bt-metric-card.negative .ide-bt-metric-value { color: var(--ide-down); }

.ide-bt-equity-placeholder { margin: 8px 0; }
.ide-bt-equity-inner { text-align: center; padding: 30px; border-radius: 8px; background: var(--ide-card-inner); color: var(--ide-text-tertiary); font-size: 13px; border: 1px solid var(--ide-border); }

/* Smart Tuning */
.ide-bt-tuning { padding: 14px; }
.ide-tuning-header { display: flex; align-items: center; gap: 10px; margin-bottom: 12px; padding: 10px 12px; border-radius: 6px; background: var(--ide-card-inner); border: 1px solid var(--ide-border); }
.ide-tuning-icon { font-size: 18px; }
.ide-tuning-title { font-size: 13px; font-weight: 700; color: var(--ide-text); }
.ide-tuning-subtitle { font-size: 10px; color: var(--ide-text-tertiary); margin-top: 2px; }
.ide-tuning-section { padding: 12px; border-radius: 6px; border: 1px solid var(--ide-border); background: var(--ide-card-inner); }
.ide-tuning-section-title { font-size: 11px; font-weight: 600; color: var(--ide-text-secondary); margin-bottom: 10px; }
.ide-tuning-method-row { display: flex; align-items: center; gap: 10px; margin-bottom: 12px; }

.ide-tuning-method-seg .ant-radio-button-wrapper {
  padding: 0 12px; font-size: 11px; height: 26px; line-height: 24px;
  border-color: var(--ide-border); color: var(--ide-text-secondary);
  background: var(--ide-card);
}
.ide-tuning-method-seg .ant-radio-button-wrapper:first-child { border-radius: 6px 0 0 6px; }
.ide-tuning-method-seg .ant-radio-button-wrapper:last-child { border-radius: 0 6px 6px 0; }
.ide-tuning-method-seg .ant-radio-button-wrapper-checked {
  background: rgba(47,111,237,0.15) !important; border-color: var(--ide-primary) !important; color: var(--ide-primary) !important;
}
.ide-tuning-method-seg .ant-radio-button-wrapper:not(:first-child)::before { background: var(--ide-border) !important; }

.ide-tuning-run-btn { border-radius: 6px !important; font-weight: 600 !important; }

.ide-tuning-dims { margin-top: 10px; }
.ide-tuning-dims-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 6px; font-size: 10px; color: var(--ide-text-secondary); }
.ide-tuning-dims-stats { color: var(--ide-text-tertiary); font-size: 10px; }
.ide-tuning-dim-row {
  display: flex; align-items: center; gap: 8px; padding: 3px 8px;
  font-size: 10px; border-radius: 4px; margin-bottom: 2px;
  cursor: pointer; user-select: none;
  background: rgba(47,111,237,0.06);
}
.ide-tuning-dim-row.disabled { opacity: 0.4; background: transparent; }
.ide-tuning-dim-row input[type="checkbox"] { cursor: pointer; }
.ide-tuning-dim-name { flex: 1; font-weight: 500; color: var(--ide-text); }
.ide-tuning-dim-badge { font-size: 8px; font-weight: 700; padding: 1px 6px; border-radius: 3px; }
.ide-tuning-dim-badge.badge--code { background: rgba(47,111,237,0.2); color: var(--ide-primary); }
.ide-tuning-dim-badge.badge--risk { background: rgba(250,140,22,0.2); color: #fa8c16; }
.ide-tuning-dim-count { color: var(--ide-primary); font-weight: 700; }
.ide-tuning-dim-values { color: var(--ide-text-tertiary); font-size: 9px; }

/* ── RIGHT: Quick Trade ── */
.ide-col-right {
  width: 25%; min-width: 300px; max-width: 420px; flex-shrink: 0;
  border-left: 1px solid var(--ide-border);
  background: var(--ide-card);
  overflow-y: auto;
}

/* ── Ant Design dark overrides ── */
.ide-workspace .ant-select:not(.ant-select-customize-input) .ant-select-selector {
  background: var(--ide-card) !important; border-color: var(--ide-border) !important; color: var(--ide-text) !important;
}
.ide-workspace .ant-select-arrow { color: var(--ide-text-tertiary) !important; }
.ide-workspace .ant-select-dropdown { background: var(--ide-card) !important; border-color: var(--ide-border) !important; }
.ide-workspace .ant-select-item {
  color: var(--ide-text-secondary) !important;
}
.ide-workspace .ant-select-item-option-active { background: rgba(47,111,237,0.08) !important; }
.ide-workspace .ant-select-item-option-selected { background: rgba(47,111,237,0.12) !important; color: var(--ide-primary) !important; }
.ide-workspace .ant-btn-default {
  background: var(--ide-card-inner) !important; border-color: var(--ide-border) !important; color: var(--ide-text-secondary) !important;
}
.ide-workspace .ant-btn-default:hover { color: var(--ide-text) !important; border-color: var(--ide-primary) !important; }
.ide-workspace .ant-input-number {
  background: var(--ide-card) !important; border-color: var(--ide-border) !important; color: var(--ide-text) !important;
}
.ide-workspace .ant-input-number-input { color: var(--ide-text) !important; }
.ide-workspace .ant-picker {
  background: var(--ide-card) !important; border-color: var(--ide-border) !important; color: var(--ide-text) !important;
}
.ide-workspace .ant-picker input { color: var(--ide-text) !important; }
.ide-workspace .ant-picker-suffix { color: var(--ide-text-tertiary) !important; }
.ide-workspace .ant-switch { background: var(--ide-card-inner) !important; }
.ide-workspace .ant-switch-checked { background: var(--ide-primary) !important; }
.ide-workspace .ant-radio-button-wrapper {
  background: var(--ide-card-inner); border-color: var(--ide-border); color: var(--ide-text-secondary);
}
.ide-workspace .ant-spin-dot-item { background: var(--ide-primary) !important; }

/* ── Responsive ── */
@media (max-width: 1200px) {
  .ide-col-left { width: 35%; min-width: 240px; }
  .ide-col-right { width: 30%; min-width: 260px; }
  .ide-bt-params-grid { grid-template-columns: repeat(2, 1fr); }
}
@media (max-width: 900px) {
  .ide-col-left { width: 100%; max-width: none; }
  .ide-col-mid { order: -1; }
  .ide-col-right { width: 100%; max-width: none; }
  .ide-body { flex-direction: column; }
  .ide-bt-params-grid { grid-template-columns: 1fr; }
}
</style>
