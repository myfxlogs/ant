<script setup>
import { ref, computed, watch } from 'vue'

const props = defineProps({
  symbol: { type: String, default: '' },
  market: { type: String, default: '' },
  timeframe: { type: String, default: '1H' },
  /** MT account id — when set, balance + positions can be fetched */
  accountId: { type: String, default: '' },
  /** Account snapshot from parent: { balance, equity, freeMargin, leverage, currency } */
  accountInfo: { type: Object, default: null },
  /** Open positions for the current symbol: [{ ticket, side, volume, openPrice, markPrice, profit, ... }] */
  positions: { type: Array, default: () => [] },
  /** Recent trades: [{ ticket, symbol, side, volume, price, profit, openTime, closeTime }] */
  recentTrades: { type: Array, default: () => [] },
})

const emit = defineEmits(['close', 'place-order', 'close-position'])

// ---- Trade state ----
const currentPrice = ref(72726)
const exchange = ref('')
const tradeDirection = ref('long')
const orderType = ref('market')
const limitPrice = ref(0)
const amount = ref(0)
const amountPercent = ref(0)
const leverage = ref(1)
const marginMode = ref('cross')
const takeProfit = ref('')
const stopLoss = ref('')
const submitting = ref(false)
const closingTicket = ref(null)
const historyCollapsed = ref(true)

// ---- Computed ----
const priceChangeClass = computed(() => 'up')

const displayPrice = computed(() => {
  if (currentPrice.value > 0) return currentPrice.value
  return 0
})

const freeMargin = computed(() => {
  if (props.accountInfo && props.accountInfo.freeMargin != null) {
    return parseFloat(props.accountInfo.freeMargin) || 0
  }
  return 0
})

const accountLeverage = computed(() => {
  if (props.accountInfo && props.accountInfo.leverage) {
    return parseInt(props.accountInfo.leverage) || 1
  }
  return 1
})

const totalAmount = computed(() => {
  if (amountPercent.value > 0) {
    const base = freeMargin.value > 0 ? freeMargin.value : 10000
    return (amountPercent.value / 100 * base).toFixed(0)
  }
  return amount.value || 0
})

const canSubmit = computed(() => {
  return props.symbol && (amount.value > 0 || amountPercent.value > 0) && !submitting.value
})

const submitLabel = computed(() => {
  const dir = tradeDirection.value === 'long' ? 'Long' : 'Short'
  const sym = props.symbol || ''
  return `${dir} ${sym}`
})

const submitBtnClass = computed(() => {
  return tradeDirection.value === 'long' ? 'qt-btn-long' : 'qt-btn-short'
})

// ---- Watchers ----
watch(() => props.accountInfo, (info) => {
  if (info && info.leverage) {
    leverage.value = parseInt(info.leverage) || 1
  }
})

// ---- Actions ----
const setAmountPercent = (pct) => {
  amountPercent.value = pct
  amount.value = 0
}

const setTradeDirection = (dir) => {
  tradeDirection.value = dir
  if (orderType.value === 'market') {
    handlePlaceOrder(dir)
  }
}

const handleAmountChange = () => {
  amountPercent.value = 0
}

const handlePlaceOrder = (dir) => {
  tradeDirection.value = dir
  if (!canSubmit.value) return

  submitting.value = true
  const payload = {
    symbol: props.symbol,
    side: dir === 'long' ? 'buy' : 'sell',
    orderType: orderType.value,
    volume: String(totalAmount.value),
    price: orderType.value === 'limit' ? String(limitPrice.value) : '',
    stopLoss: stopLoss.value ? String(stopLoss.value) : '',
    takeProfit: takeProfit.value ? String(takeProfit.value) : '',
  }
  emit('place-order', payload)
  // Parent controls submitting reset via prop or timeout
  setTimeout(() => { submitting.value = false }, 5000)
}

const handleClosePosition = (pos) => {
  closingTicket.value = pos.ticket
  emit('close-position', pos)
  setTimeout(() => { closingTicket.value = null }, 5000)
}

const formatPrice = (p) => {
  if (p == null || p === '') return '—'
  const n = Number(p)
  if (isNaN(n)) return String(p)
  if (Math.abs(n) >= 100) return n.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
  if (Math.abs(n) >= 1) return n.toLocaleString('en-US', { minimumFractionDigits: 4, maximumFractionDigits: 4 })
  return n.toLocaleString('en-US', { minimumFractionDigits: 5, maximumFractionDigits: 5 })
}

const formatPnL = (v) => {
  const n = Number(v)
  if (isNaN(n)) return '$0.00'
  const sign = n >= 0 ? '+' : ''
  return sign + '$' + formatPrice(Math.abs(n))
}

const formatTime = (ts) => {
  if (!ts) return ''
  const d = new Date(typeof ts === 'string' ? ts : ts * 1000)
  const mm = String(d.getMonth() + 1).padStart(2, '0')
  const dd = String(d.getDate()).padStart(2, '0')
  const hh = String(d.getHours()).padStart(2, '0')
  const min = String(d.getMinutes()).padStart(2, '0')
  return `${mm}-${dd} ${hh}:${min}`
}

const posSideLabel = (side) => {
  if (!side) return ''
  return side === 'buy' || side === 'long' ? 'LONG' : 'SHORT'
}

const posSideColor = (side) => {
  if (!side) return '#5c636b'
  return side === 'buy' || side === 'long' ? '#26a69a' : '#ef5350'
}
</script>

<template>
  <div class="qt-panel">
    <!-- Header -->
    <div class="qt-header">
      <span class="qt-header-title">⚡ Quick Trade</span>
      <a-button type="text" size="small" class="qt-close-btn" @click="emit('close')">✕</a-button>
    </div>

    <div class="qt-body">
      <!-- Symbol & Price -->
      <div class="qt-symbol-bar" v-if="symbol">
        <div class="qt-symbol-name">{{ symbol }}</div>
        <div class="qt-price" :class="priceChangeClass">${{ formatPrice(displayPrice) }}</div>
      </div>
      <div v-else class="qt-no-symbol">
        <p>Select a symbol to trade</p>
      </div>

      <!-- Account Balance (MT free margin) -->
      <div class="qt-section" v-if="accountInfo">
        <div class="qt-label">Account Balance</div>
        <div class="qt-balance-card">
          <div class="qt-balance-row">
            <span class="qt-balance-label">Free Margin</span>
            <span class="qt-balance-value">${{ formatPrice(freeMargin) }}</span>
          </div>
          <div class="qt-balance-row" v-if="accountInfo.equity != null">
            <span class="qt-balance-label">Equity</span>
            <span class="qt-balance-value">${{ formatPrice(accountInfo.equity) }}</span>
          </div>
          <div class="qt-balance-row" v-if="accountInfo.balance != null">
            <span class="qt-balance-label">Balance</span>
            <span class="qt-balance-value">${{ formatPrice(accountInfo.balance) }}</span>
          </div>
        </div>
      </div>

      <!-- Exchange (visual placeholder for MT) -->
      <div class="qt-section">
        <div class="qt-label">Exchange <a-tag color="default" class="qt-hint-tag">Coming soon</a-tag></div>
        <a-select
          v-model:value="exchange"
          size="small"
          style="width:100%"
          placeholder="Gate (default)"
          disabled
          title="Exchange selection not applicable for MT accounts"
        >
          <a-select-option value="gate">Gate</a-select-option>
        </a-select>
      </div>

      <!-- Long / Short buttons -->
      <div class="qt-section">
        <div class="qt-direction-row">
          <button
            class="qt-dir-btn qt-dir-btn--long"
            :class="{ active: tradeDirection === 'long' }"
            @click="setTradeDirection('long')"
          >↑ Long</button>
          <button
            class="qt-dir-btn qt-dir-btn--short"
            :class="{ active: tradeDirection === 'short' }"
            @click="setTradeDirection('short')"
          >↓ Short</button>
        </div>
      </div>

      <!-- Market / Limit -->
      <div class="qt-section">
        <div class="qt-label">Order Type</div>
        <a-radio-group v-model:value="orderType" size="small" button-style="solid" class="qt-type-seg">
          <a-radio-button value="market">Market</a-radio-button>
          <a-radio-button value="limit">Limit</a-radio-button>
        </a-radio-group>
      </div>

      <!-- Limit Price -->
      <div class="qt-section" v-if="orderType === 'limit'">
        <div class="qt-label">Limit Price</div>
        <a-input-number
          v-model:value="limitPrice"
          size="small"
          style="width:100%"
          :min="0"
          :step="0.01"
          placeholder="Enter limit price"
        />
      </div>

      <!-- Amount -->
      <div class="qt-section">
        <div class="qt-label">Amount (USDT)</div>
        <a-input-number
          v-model:value="amount"
          size="small"
          style="width:100%"
          :min="0"
          :step="100"
          placeholder="0.00"
          @change="handleAmountChange"
        />
        <div class="qt-pct-row">
          <button
            v-for="pct in [10, 25, 50, 75, 100]" :key="pct"
            class="qt-pct-btn"
            :class="{ active: amountPercent === pct }"
            :disabled="freeMargin <= 0"
            @click="setAmountPercent(pct)"
          >{{ pct }}%</button>
        </div>
      </div>

      <!-- Leverage (visual placeholder) -->
      <div class="qt-section">
        <div class="qt-label">
          Leverage
          <a-tag color="default" class="qt-hint-tag" title="Leverage control not available for MT accounts">MT Read-only</a-tag>
        </div>
        <div class="qt-leverage-row">
          <a-slider
            v-model:value="leverage"
            :min="1" :max="125" :step="1"
            class="qt-leverage-slider"
            disabled
            title="Leverage control not available for MT accounts"
          />
          <span class="qt-leverage-val">{{ leverage }}x</span>
        </div>
      </div>

      <!-- Margin Mode (visual placeholder) -->
      <div class="qt-section">
        <div class="qt-label">
          Margin Mode
          <a-tag color="default" class="qt-hint-tag" title="Margin mode not applicable for MT">Coming soon</a-tag>
        </div>
        <a-radio-group v-model:value="marginMode" size="small" button-style="solid" class="qt-type-seg" disabled>
          <a-radio-button value="cross">Cross</a-radio-button>
          <a-radio-button value="isolated">Isolated</a-radio-button>
        </a-radio-group>
      </div>

      <!-- TP / SL -->
      <div class="qt-section">
        <div class="qt-label">Take Profit / Stop Loss</div>
        <a-row :gutter="8">
          <a-col :span="12">
            <a-input-number
              v-model:value="takeProfit"
              size="small"
              style="width:100%"
              placeholder="TP"
              :min="0"
              :step="1"
            />
          </a-col>
          <a-col :span="12">
            <a-input-number
              v-model:value="stopLoss"
              size="small"
              style="width:100%"
              placeholder="SL"
              :min="0"
              :step="1"
            />
          </a-col>
        </a-row>
      </div>

      <!-- Estimated total + Submit -->
      <div class="qt-section">
        <div class="qt-estimate" v-if="symbol && totalAmount > 0">
          Est. total: <b>${{ formatPrice(displayPrice * totalAmount) }}</b>
        </div>
        <a-button
          :type="tradeDirection === 'long' ? 'primary' : 'danger'"
          size="large"
          block
          :loading="submitting"
          :disabled="!canSubmit"
          class="qt-submit-btn"
          :class="submitBtnClass"
          @click="handlePlaceOrder(tradeDirection)"
        >
          <span v-if="tradeDirection === 'long'">↑</span>
          <span v-else>↓</span>
          {{ submitLabel }}
        </a-button>
      </div>

      <!-- ── Current Position ── -->
      <div class="qt-position-section" v-if="symbol">
        <div class="qt-section-header">
          <span>📊 Current Position</span>
          <span v-if="positions.length > 0" class="qt-pos-count">({{ positions.length }})</span>
        </div>

        <template v-if="positions.length > 0">
          <div
            v-for="(pos, idx) in positions" :key="'pos-' + idx"
            class="qt-position-card"
            :class="pos.side === 'buy' || pos.side === 'long' ? 'pos-long' : 'pos-short'"
          >
            <div class="qt-pos-row">
              <span>Side</span>
              <a-tag :color="posSideColor(pos.side)" size="small">{{ posSideLabel(pos.side) }}</a-tag>
            </div>
            <div class="qt-pos-row">
              <span>Size</span>
              <span>{{ pos.volume || pos.size || '—' }}</span>
            </div>
            <div class="qt-pos-row">
              <span>Entry Price</span>
              <span>${{ formatPrice(pos.openPrice || pos.entry_price) }}</span>
            </div>
            <div class="qt-pos-row" v-if="pos.markPrice || pos.mark_price">
              <span>Mark Price</span>
              <span>${{ formatPrice(pos.markPrice || pos.mark_price) }}</span>
            </div>
            <div class="qt-pos-row" v-if="pos.leverage && pos.leverage > 1">
              <span>Leverage</span>
              <span>{{ pos.leverage }}x</span>
            </div>
            <div class="qt-pos-row">
              <span>Unrealized PnL</span>
              <span :class="(pos.profit || pos.unrealized_pnl) >= 0 ? 'qt-green' : 'qt-red'">
                {{ formatPnL(pos.profit || pos.unrealized_pnl) }}
              </span>
            </div>
            <a-button
              type="danger"
              size="small"
              block
              ghost
              :loading="closingTicket === pos.ticket"
              style="margin-top: 6px;"
              @click="handleClosePosition(pos)"
            >
              Close Position
            </a-button>
          </div>
        </template>
        <div v-else class="qt-position-empty">
          <span class="qt-empty-desc">No open positions for {{ symbol || 'this symbol' }}</span>
        </div>
      </div>

      <!-- ── Recent Trades ── -->
      <div class="qt-history-section" v-if="recentTrades.length > 0">
        <a-collapse
          :bordered="false"
          :activeKey="historyCollapsed ? [] : ['history']"
          @change="historyCollapsed = $event.length === 0"
        >
          <a-collapse-panel key="history" :showArrow="false" class="qt-history-panel">
            <template #header>
              <div class="qt-section-header" style="margin:0;padding:0;">
                <span>🕐 Recent Trades</span>
                <span class="qt-history-count">({{ recentTrades.length }})</span>
              </div>
            </template>
            <div class="qt-trade-list">
              <div class="qt-trade-item" v-for="t in recentTrades" :key="t.ticket || t.id">
                <div class="qt-trade-main">
                  <a-tag :color="(t.side === 'buy' || t.side === 'long') ? '#26a69a' : '#ef5350'" size="small">
                    {{ posSideLabel(t.side) }}
                  </a-tag>
                  <span class="qt-trade-symbol">{{ t.symbol }}</span>
                  <span class="qt-trade-amount">${{ formatPrice(t.price || t.closePrice) }}</span>
                </div>
                <div class="qt-trade-meta">
                  <a-tag
                    :color="t.profit >= 0 ? '#26a69a' : '#ef5350'"
                    size="small"
                  >{{ formatPnL(t.profit) }}</a-tag>
                  <span class="qt-trade-time">{{ formatTime(t.closeTime || t.created_at) }}</span>
                </div>
              </div>
            </div>
          </a-collapse-panel>
        </a-collapse>
      </div>
    </div>
  </div>
</template>

<style scoped>
/* ── V31 Dark Palette ── */
.qt-panel {
  display: flex; flex-direction: column; height: 100%;
  background: #161a20;
}

/* Header */
.qt-header {
  flex-shrink: 0;
  display: flex; align-items: center; justify-content: space-between;
  padding: 10px 14px;
  border-bottom: 1px solid #262b31;
  background: #1c2128;
}
.qt-header-title { font-size: 13px; font-weight: 700; color: #e6e8eb; }
.qt-close-btn { color: #5c636b !important; font-size: 14px; }

/* Body */
.qt-body { flex: 1; overflow-y: auto; padding: 12px 14px; display: flex; flex-direction: column; gap: 12px; }

/* Symbol & Price */
.qt-symbol-bar { text-align: center; padding: 6px 0 2px; }
.qt-symbol-name { font-size: 13px; font-weight: 700; color: #e6e8eb; margin-bottom: 4px; }
.qt-price { font-size: 24px; font-weight: 800; line-height: 1.2; }
.qt-price.up { color: #26a69a; }
.qt-price.down { color: #ef5350; }
.qt-no-symbol { padding: 24px 0; text-align: center; }
.qt-no-symbol p { font-size: 12px; color: #8b949e; }

/* Sections */
.qt-section { display: flex; flex-direction: column; gap: 5px; }
.qt-label { font-size: 10px; font-weight: 700; text-transform: uppercase; color: #5c636b; display: flex; align-items: center; gap: 6px; }
.qt-hint-tag { font-size: 8px; line-height: 1; padding: 1px 4px; }

/* ── Balance Card ── */
.qt-balance-card {
  background: #1c2128; border: 1px solid #262b31; border-radius: 6px;
  padding: 8px 10px; display: flex; flex-direction: column; gap: 3px;
}
.qt-balance-row { display: flex; justify-content: space-between; align-items: center; }
.qt-balance-label { font-size: 10px; color: #5c636b; }
.qt-balance-value { font-size: 11px; font-weight: 700; color: #e6e8eb; }

/* Long/Short */
.qt-direction-row { display: flex; gap: 8px; }
.qt-dir-btn {
  flex: 1; padding: 10px 0; border: 1px solid #262b31; border-radius: 8px;
  font-size: 14px; font-weight: 700; cursor: pointer; user-select: none;
  background: #1c2128; color: #8b949e;
  transition: all 0.15s;
}
.qt-dir-btn:hover { filter: brightness(1.1); }
.qt-dir-btn--long.active {
  background: rgba(38,166,154,0.18); border-color: #26a69a; color: #26a69a;
  box-shadow: 0 0 12px rgba(38,166,154,0.15);
}
.qt-dir-btn--short.active {
  background: rgba(239,83,80,0.18); border-color: #ef5350; color: #ef5350;
  box-shadow: 0 0 12px rgba(239,83,80,0.15);
}

/* Order Type segmented */
.qt-type-seg :deep(.ant-radio-button-wrapper) {
  padding: 0 16px; font-size: 11px; height: 28px; line-height: 26px;
  border-color: #262b31; color: #8b949e; background: #1c2128;
}
.qt-type-seg :deep(.ant-radio-button-wrapper:first-child) { border-radius: 6px 0 0 6px; }
.qt-type-seg :deep(.ant-radio-button-wrapper:last-child) { border-radius: 0 6px 6px 0; }
.qt-type-seg :deep(.ant-radio-button-wrapper-checked) {
  background: rgba(47,111,237,0.15) !important; border-color: #2f6fed !important; color: #2f6fed !important;
}
.qt-type-seg :deep(.ant-radio-button-wrapper:not(:first-child)::before) { background: #262b31 !important; }

/* Amount % buttons */
.qt-pct-row { display: flex; gap: 4px; margin-top: 2px; }
.qt-pct-btn {
  flex: 1; padding: 4px 0; border: 1px solid #262b31; border-radius: 4px;
  font-size: 10px; font-weight: 600; cursor: pointer; user-select: none;
  background: #1c2128; color: #8b949e;
  transition: all 0.12s;
}
.qt-pct-btn:hover:not(:disabled) { border-color: #2f6fed; color: #2f6fed; }
.qt-pct-btn.active { background: rgba(47,111,237,0.15); border-color: #2f6fed; color: #2f6fed; font-weight: 700; }
.qt-pct-btn:disabled { opacity: 0.4; cursor: not-allowed; }

/* Leverage */
.qt-leverage-row { display: flex; align-items: center; gap: 10px; }
.qt-leverage-slider { flex: 1; }
.qt-leverage-val { font-size: 12px; font-weight: 700; color: #e6e8eb; min-width: 36px; text-align: right; }

/* Estimate */
.qt-estimate { font-size: 11px; color: #8b949e; text-align: right; }
.qt-estimate b { color: #e6e8eb; }

/* ── Submit Button ── */
.qt-submit-btn {
  border-radius: 8px !important; font-weight: 700 !important; font-size: 14px !important;
  height: 40px !important; margin-top: 2px;
  display: inline-flex !important; align-items: center; gap: 6px;
  transition: all 0.15s;
}
.qt-submit-btn.qt-btn-long {
  background: #26a69a !important; border-color: #26a69a !important;
  box-shadow: 0 2px 12px rgba(38,166,154,0.3);
}
.qt-submit-btn.qt-btn-long:hover { background: #2ecc71 !important; border-color: #2ecc71 !important; }
.qt-submit-btn.qt-btn-short {
  background: #ef5350 !important; border-color: #ef5350 !important;
  box-shadow: 0 2px 12px rgba(239,83,80,0.3);
}
.qt-submit-btn.qt-btn-short:hover { background: #f44336 !important; border-color: #f44336 !important; }

/* ── Position Section ── */
.qt-position-section {
  border-top: 1px solid #262b31; padding-top: 10px;
  display: flex; flex-direction: column; gap: 6px;
}
.qt-section-header {
  display: flex; align-items: center; gap: 6px;
  font-size: 11px; font-weight: 700; color: #e6e8eb;
}
.qt-pos-count { font-size: 10px; color: #8b949e; font-weight: 500; }
.qt-position-empty { padding: 10px 0; text-align: center; }
.qt-empty-desc { font-size: 11px; color: #5c636b; }

.qt-position-card {
  background: #1c2128; border: 1px solid #262b31; border-radius: 6px;
  padding: 8px 10px; display: flex; flex-direction: column; gap: 3px;
}
.qt-position-card.pos-long { border-left: 3px solid #26a69a; }
.qt-position-card.pos-short { border-left: 3px solid #ef5350; }

.qt-pos-row { display: flex; justify-content: space-between; align-items: center; font-size: 10px; }
.qt-pos-row > span:first-child { color: #5c636b; }
.qt-pos-row > span:last-child { color: #e6e8eb; font-weight: 600; }
.qt-green { color: #26a69a !important; }
.qt-red { color: #ef5350 !important; }

/* ── Trade History ── */
.qt-history-section {
  border-top: 1px solid #262b31; padding-top: 10px;
}
.qt-history-count { font-size: 10px; color: #8b949e; font-weight: 500; }
.qt-history-panel {
  background: transparent !important; border: none !important;
}
.qt-trade-list { display: flex; flex-direction: column; gap: 4px; }
.qt-trade-item {
  background: #1c2128; border: 1px solid #262b31; border-radius: 4px;
  padding: 6px 8px;
}
.qt-trade-main { display: flex; align-items: center; gap: 6px; margin-bottom: 4px; }
.qt-trade-symbol { font-size: 10px; font-weight: 600; color: #e6e8eb; }
.qt-trade-amount { font-size: 10px; color: #8b949e; margin-left: auto; }
.qt-trade-meta { display: flex; align-items: center; gap: 6px; }
.qt-trade-time { font-size: 9px; color: #5c636b; margin-left: auto; }

/* ── Ant Design overrides ── */
:deep(.ant-select:not(.ant-select-customize-input) .ant-select-selector) {
  background: #1c2128 !important; border-color: #262b31 !important; color: #e6e8eb !important;
}
:deep(.ant-select-arrow) { color: #5c636b !important; }
:deep(.ant-input-number) {
  background: #1c2128 !important; border-color: #262b31 !important; color: #e6e8eb !important;
}
:deep(.ant-input-number-input) { color: #e6e8eb !important; }
:deep(.ant-slider-rail) { background: #262b31 !important; }
:deep(.ant-slider-track) { background: #2f6fed !important; }
:deep(.ant-slider-handle) { border-color: #2f6fed !important; background: #161a20 !important; }

/* Collapse overrides */
:deep(.ant-collapse) { background: transparent !important; border: none !important; }
:deep(.ant-collapse-item) { border: none !important; }
:deep(.ant-collapse-header) {
  padding: 0 !important; background: transparent !important;
  color: #e6e8eb !important; font-size: 11px;
}
:deep(.ant-collapse-content) {
  background: transparent !important; border-top: none !important;
}
:deep(.ant-collapse-content-box) { padding: 6px 0 0 !important; }
:deep(.ant-collapse-arrow) { display: none !important; }
</style>
