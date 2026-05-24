export const DEFAULT_TEMPLATES: any[] = [
  {
    id: 'default-ma-cross',
    nameKey: 'strategy.defaultTemplates.maCross.name',
    descriptionKey: 'strategy.defaultTemplates.maCross.description',
    name: 'MA Crossover',
    description: 'Buy on fast MA cross above slow MA; sell on cross below',
    code: `# MA crossover strategy
# Available variables: close, open, high, low, volume, symbol
# Return: signal dict

# Parameters
fast_period = 10
slow_period = 20

# Data length check
if len(close) < slow_period + 1:
    signal = {
        'signal': 'hold',
        'symbol': symbol,
        'price': close[-1] if len(close) > 0 else None,
        'confidence': 0.0,
        'reason': 'insufficient data',
        'risk_level': 'low',
    }
else:
    # Compute moving averages
    maFast = np.mean(close[-fast_period:])
    maSlow = np.mean(close[-slow_period:])
    ma_fast_prev = np.mean(close[-fast_period-1:-1])
    ma_slow_prev = np.mean(close[-slow_period-1:-1])

    # Detect crossover
    if maFast > maSlow and ma_fast_prev <= ma_slow_prev:
        action = 'buy'
        reason = 'bullish crossover'
        risk_level = 'medium'
    elif maFast < maSlow and ma_fast_prev >= ma_slow_prev:
        action = 'sell'
        reason = 'bearish crossover'
        risk_level = 'medium'
    else:
        action = 'hold'
        reason = 'no signal'
        risk_level = 'low'

    # Result
    signal = {
        'signal': action,
        'symbol': symbol,
        'price': close[-1],
        'confidence': 0.7,
        'reason': reason,
        'risk_level': risk_level,
        'maFast': round(maFast, 5),
        'maSlow': round(maSlow, 5)
    }`,
    isPublic: true,
    useCount: 0,
    createdAt: new Date().toISOString() as any,
  },
  {
    id: 'default-rsi-oversold',
    nameKey: 'strategy.defaultTemplates.rsiOversold.name',
    descriptionKey: 'strategy.defaultTemplates.rsiOversold.description',
    name: 'RSI Oversold Bounce',
    description: 'Enter long when RSI bounces from oversold; exit on overbought.',
    code: `# RSI Oversold Bounce
import numpy as np

def rsi(series, period=14):
    if len(series) < period + 1:
        return None
    deltas = np.diff(series)
    up = np.where(deltas > 0, deltas, 0.0)
    down = np.where(deltas < 0, -deltas, 0.0)
    roll_up = np.convolve(up, np.ones(period), 'valid') / period
    roll_down = np.convolve(down, np.ones(period), 'valid') / period
    rs = roll_up / (roll_down + 1e-12)
    rsi_vals = 100 - (100 / (1 + rs))
    return rsi_vals

period = int(context.get('params', {}).get('rsi_period', 14))
oversold = float(context.get('params', {}).get('oversold', 30))
overbought = float(context.get('params', {}).get('overbought', 70))

vals = rsi(close, period)
signal = {'signal': 'hold', 'symbol': symbol, 'price': close[-1] if len(close)>0 else None}
if vals is not None and len(vals) >= 2:
    r0, r1 = float(vals[-2]), float(vals[-1])
    if r0 <= oversold and r1 > oversold:
        signal.update({'signal': 'buy', 'reason': f'RSI bounce {r0:.1f}->{r1:.1f}'})
    elif r1 >= overbought:
        signal.update({'signal': 'close', 'reason': f'RSI overbought {r1:.1f}'})
return signal
`,
    isPublic: true,
    tags: ['mean-reversion', 'RSI'],
    useCount: 0,
    createdAt: new Date().toISOString() as any,
  },
  {
    id: 'default-bollinger-squeeze',
    nameKey: 'strategy.defaultTemplates.bollingerSqueeze.name',
    descriptionKey: 'strategy.defaultTemplates.bollingerSqueeze.description',
    name: 'Bollinger Squeeze Breakout',
    description: 'Trade breakouts after Bollinger Band squeezes.',
    code: `# Bollinger Band Squeeze Breakout
import numpy as np

period = int(context.get('params', {}).get('bb_period', 20))
bb_std = float(context.get('params', {}).get('bb_std', 2.0))
squeeze_threshold = float(context.get('params', {}).get('squeeze_threshold', 0.05))

signal = {'signal': 'hold', 'symbol': symbol, 'price': close[-1] if len(close)>0 else None}
if len(close) >= period:
    window = close[-period:]
    m = float(np.mean(window))
    sd = float(np.std(window) + 1e-12)
    upper = m + bb_std * sd
    lower = m - bb_std * sd
    width = (upper - lower) / (m + 1e-12)
    # Breakout conditions
    if width < squeeze_threshold:
        if close[-1] > upper:
            signal.update({'signal': 'buy', 'reason': 'Upper band breakout after squeeze'})
        elif close[-1] < lower:
            signal.update({'signal': 'sell', 'reason': 'Lower band breakout after squeeze'})
return signal
`,
    isPublic: true,
    tags: ['volatility', 'bollinger', 'breakout'],
    useCount: 0,
    createdAt: new Date().toISOString() as any,
  },
  {
    id: 'default-macd-divergence',
    nameKey: 'strategy.defaultTemplates.macdDivergence.name',
    descriptionKey: 'strategy.defaultTemplates.macdDivergence.description',
    name: 'MACD Divergence',
    description: 'Enter on bullish/bearish MACD divergence.',
    code: `# MACD Divergence (simple heuristic)
import numpy as np

fast_p = int(context.get('params', {}).get('fast_period', 12))
slow_p = int(context.get('params', {}).get('slow_period', 26))
sig_p = int(context.get('params', {}).get('signal_period', 9))

def ema(x, p):
    k = 2/(p+1)
    out = []
    prev = None
    for v in x:
        prev = v if prev is None else (v*k + prev*(1-k))
        out.append(prev)
    return np.array(out)

if len(close) >= slow_p + sig_p + 3:
    macd = ema(close, fast_p) - ema(close, slow_p)
    signal_line = ema(macd, sig_p)
    hist = macd - signal_line
    # Heuristic: price makes lower low but hist makes higher low => bullish divergence
    p0, p1 = float(close[-3]), float(close[-1])
    h0, h1 = float(hist[-3]), float(hist[-1])
    sig = {'signal': 'hold', 'symbol': symbol, 'price': close[-1]}
    if p1 < p0 and h1 > h0:
        sig.update({'signal': 'buy', 'reason': 'bullish MACD divergence'})
    elif p1 > p0 and h1 < h0:
        sig.update({'signal': 'sell', 'reason': 'bearish MACD divergence'})
    return sig
return {'signal': 'hold', 'symbol': symbol, 'price': close[-1] if len(close)>0 else None}
`,
    isPublic: true,
    tags: ['trend', 'MACD', 'divergence'],
    useCount: 0,
    createdAt: new Date().toISOString() as any,
  },
  {
    id: 'default-breakout-volume',
    nameKey: 'strategy.defaultTemplates.breakoutVolume.name',
    descriptionKey: 'strategy.defaultTemplates.breakoutVolume.description',
    name: 'Volume Breakout',
    description: 'Enter on price high breakout with above-average volume; ATR stop.',
    code: `# Volume Breakout with ATR stop (simplified)
import numpy as np

lookback = int(context.get('params', {}).get('lookback', 20))
vol_mult = float(context.get('params', {}).get('volume_multiplier', 1.5))
atr_mult = float(context.get('params', {}).get('atr_multiplier', 2.0))

def atr(h, l, c, p=14):
    if len(c) < p+1:
        return None
    trs = []
    for i in range(1, len(c)):
        tr = max(h[i]-l[i], abs(h[i]-c[i-1]), abs(l[i]-c[i-1]))
        trs.append(tr)
    return float(np.mean(trs[-p:]))

signal = {'signal': 'hold', 'symbol': symbol, 'price': close[-1] if len(close)>0 else None}
if len(close) >= lookback + 1:
    recent_high = float(np.max(close[-lookback:]))
    avg_vol = float(np.mean(volume[-lookback:]) + 1e-12)
    cur_vol = float(volume[-1])
    if close[-1] > recent_high and cur_vol > vol_mult * avg_vol:
        my_atr = atr(high, low, close, 14) or 0.0
        signal.update({'signal': 'buy', 'reason': 'volume breakout', 'stop_loss': close[-1] - atr_mult*my_atr if my_atr>0 else 0.0})
return signal
`,
    isPublic: true,
    tags: ['breakout', 'volume', 'ATR'],
    useCount: 0,
    createdAt: new Date().toISOString() as any,
  },
  {
    id: 'default-bb-mean-reversion',
    nameKey: 'strategy.defaultTemplates.bbMeanReversion.name',
    descriptionKey: 'strategy.defaultTemplates.bbMeanReversion.description',
    name: 'BB Mean Reversion',
    description: 'Buy at lower band, exit at upper band.',
    code: `# Bollinger Band Mean Reversion
import numpy as np

period = int(context.get('params', {}).get('bb_period', 20))
bb_std = float(context.get('params', {}).get('bb_std', 2.0))

signal = {'signal': 'hold', 'symbol': symbol, 'price': close[-1] if len(close)>0 else None}
if len(close) >= period:
    window = close[-period:]
    m = float(np.mean(window))
    sd = float(np.std(window) + 1e-12)
    lower = m - bb_std * sd
    upper = m + bb_std * sd
    if close[-1] <= lower:
        signal.update({'signal': 'buy', 'reason': 'touch lower band'})
    elif close[-1] >= upper:
        signal.update({'signal': 'close', 'reason': 'touch upper band'})
return signal
`,
    isPublic: true,
    tags: ['mean-reversion', 'bollinger'],
    useCount: 0,
    createdAt: new Date().toISOString() as any,
  },
  {
    id: 'default-turtle-trading',
    nameKey: 'strategy.defaultTemplates.turtleTrading.name',
    descriptionKey: 'strategy.defaultTemplates.turtleTrading.description',
    name: 'Turtle Trading',
    description: 'Enter on N-day high, exit on M-day low; ATR position sizing omitted.',
    code: `# Turtle Trading (simplified without ATR sizing)
import numpy as np

entry_p = int(context.get('params', {}).get('entry_period', 20))
exit_p = int(context.get('params', {}).get('exit_period', 10))

signal = {'signal': 'hold', 'symbol': symbol, 'price': close[-1] if len(close)>0 else None}
if len(close) >= max(entry_p, exit_p):
    hh = float(np.max(high[-entry_p:]))
    ll = float(np.min(low[-exit_p:]))
    if close[-1] > hh:
        signal.update({'signal': 'buy', 'reason': 'break N-high'})
    elif close[-1] < ll:
        signal.update({'signal': 'sell', 'reason': 'break M-low'})
return signal
`,
    isPublic: true,
    tags: ['trend', 'turtle'],
    useCount: 0,
    createdAt: new Date().toISOString() as any,
  },
  {
    id: 'default-grid-trading',
    nameKey: 'strategy.defaultTemplates.gridTrading.name',
    descriptionKey: 'strategy.defaultTemplates.gridTrading.description',
    name: 'Grid Trading',
    description: 'Place buy/sell orders at regular grid levels within a price range; range-bound friendly.',
    code: `# Grid Trading (single-symbol, stateless approximation using pending orders idea)
# Inputs (from params or context): grid_count, lower_price, upper_price, lot

grid_count = int(context.get('params', {}).get('grid_count', 10)) if isinstance(context.get('params'), dict) else 10
lower = float(context.get('params', {}).get('lower_price', 0)) if isinstance(context.get('params'), dict) else 0.0
upper = float(context.get('params', {}).get('upper_price', 0)) if isinstance(context.get('params'), dict) else 0.0
lot = context.get('params', {}).get('lot', 0.01) if isinstance(context.get('params'), dict) else 0.01
try:
    lot = float(lot)
except Exception:
    lot = 0.01

price = close[-1] if len(close) > 0 else None
signal = {
    'signal': 'hold',
    'symbol': symbol,
    'price': price,
    'volume': lot,
    'reason': 'no grid or out of range',
}
if price is not None and upper > lower and grid_count >= 2:
    step = (upper - lower) / (grid_count - 1)
    # find nearest grid below and above
    idx = int(max(0, min(grid_count - 1, (price - lower) // step)))
    level = lower + idx * step
    # simple rule: if price below level by < half step -> buy; if above next level by < half step -> sell
    half = step * 0.5
    if price < level + half:
        signal.update({'signal': 'buy', 'reason': 'near lower grid'})
    elif price > level + step - half:
        signal.update({'signal': 'sell', 'reason': 'near upper grid'})
    else:
        signal.update({'signal': 'hold', 'reason': 'between grid levels'})
`,
    isPublic: true,
    tags: ['grid', 'market-making'],
    useCount: 0,
    createdAt: new Date().toISOString() as any,
  },
  {
    id: 'default-dca-buy',
    nameKey: 'strategy.defaultTemplates.dcaBuy.name',
    descriptionKey: 'strategy.defaultTemplates.dcaBuy.description',
    name: 'DCA Buy',
    description: 'Buy a fixed small lot at regular time intervals; long-term averaging.',
    code: `# DCA Buy (single-symbol)
# Inputs: buy_amount (or lot), interval_hours, timeframe
# Requires context['bar_time_ms'] ideally; fallback to bar index cadence if missing.

params = context.get('params', {}) if isinstance(context.get('params'), dict) else {}
lot = params.get('lot') or params.get('buy_amount') or 0.01
try:
    lot = float(lot)
except Exception:
    lot = 0.01
interval_hours = params.get('interval_hours') or 24
try:
    interval_hours = int(interval_hours)
except Exception:
    interval_hours = 24

now_ms = context.get('bar_time_ms') if isinstance(context, dict) else None
runtime = context.get('runtime') if isinstance(context.get('runtime'), dict) else {}
should_buy = False
state_last_ms = runtime.get('last_dca_buy_ms')

if now_ms is not None:
    if state_last_ms is None:
        should_buy = True
    else:
        should_buy = (int(now_ms) - int(state_last_ms)) >= interval_hours * 3600 * 1000
else:
    # Fallback: use bar index cadence when bar_time_ms is unavailable.
    N = max(1, interval_hours)
    should_buy = (len(close) % N) == 0

if should_buy and now_ms is not None:
    runtime['last_dca_buy_ms'] = int(now_ms)

signal = {
    'signal': 'buy' if should_buy else 'hold',
    'symbol': symbol,
    'price': close[-1] if len(close) > 0 else None,
    'volume': lot if should_buy else None,
    'reason': 'interval reached' if should_buy else 'waiting for next interval',
}
`,
    isPublic: true,
    tags: ['passive', 'DCA'],
    useCount: 0,
    createdAt: new Date().toISOString() as any,
  },
  {
    id: 'default-pairs-trading',
    nameKey: 'strategy.defaultTemplates.pairsTrading.name',
    descriptionKey: 'strategy.defaultTemplates.pairsTrading.description',
    name: 'Pairs Trading (Placeholder)',
    description: 'Requires multi-symbol engine; placeholder only.',
    code: `# Pairs trading placeholder
signal = {
  'signal': 'hold',
  'reason': '需要多品种引擎（占位）',
}
`,
    isPublic: true,
    tags: ['multi-symbol-required'],
    useCount: 0,
    createdAt: new Date().toISOString() as any,
  },
  {
    id: 'default-momentum-rotation',
    nameKey: 'strategy.defaultTemplates.momentumRotation.name',
    descriptionKey: 'strategy.defaultTemplates.momentumRotation.description',
    name: 'Momentum Rotation (Placeholder)',
    description: 'Requires multi-symbol engine (N>=3); placeholder only.',
    code: `# Momentum rotation placeholder
signal = {
  'signal': 'hold',
  'reason': '需要多品种引擎（占位）',
}
`,
    isPublic: true,
    tags: ['multi-symbol-required'],
    useCount: 0,
    createdAt: new Date().toISOString() as any,
  },
  {
    id: 'default-test-force-buy',
    nameKey: 'strategy.defaultTemplates.forceBuy.name',
    descriptionKey: 'strategy.defaultTemplates.forceBuy.description',
    name: 'Force BUY (Test)',
    description: 'Used to validate order pipeline: always returns buy; reads lot from context/params as volume',
    code: `# Force BUY (test)
# Reads 'lot' from context/params and always emits a BUY signal.

lot = None
try:
    if 'lot' in context:
        lot = context.get('lot')
    if lot is None and isinstance(context.get('params'), dict):
        lot = context.get('params', {}).get('lot')
except Exception:
    lot = None

try:
    lot = float(lot) if lot is not None else 0.01
except Exception:
    lot = 0.01

if lot <= 0:
    lot = 0.01

signal = {
    'signal': 'buy',
    'symbol': symbol,
    'price': close[-1] if len(close) > 0 else None,
    'volume': lot,
    'confidence': 0.5,
    'reason': 'force buy for pipeline test',
    'risk_level': 'high',
}`, 
    isPublic: true,
    useCount: 0,
    createdAt: new Date().toISOString() as any,
  },
  {
    id: 'default-rsi',
    nameKey: 'strategy.defaultTemplates.rsi.name',
    descriptionKey: 'strategy.defaultTemplates.rsi.description',
    name: 'RSI Overbought/Oversold',
    description: 'Buy when RSI < 30; sell when RSI > 70',
    code: `# RSI overbought/oversold strategy
# Available variables: close, open, high, low, volume, symbol
# Return: signal dict

# Parameters
period = 14
oversold = 30
overbought = 70

# Compute RSI
def calculate_rsi(prices, period):
    deltas = np.diff(prices)
    gains = np.where(deltas > 0, deltas, 0)
    losses = np.where(deltas < 0, -deltas, 0)
    avgGain = np.mean(gains[-period:])
    avgLoss = np.mean(losses[-period:])
    if avgLoss == 0:
        return 100
    rs = avgGain / avgLoss
    return 100 - (100 / (1 + rs))

if len(close) < period + 1:
    rsi = None
else:
    rsi = calculate_rsi(close, period)

# Generate signal
if rsi is None:
    action = 'hold'
    reason = 'insufficient data'
    risk_level = 'low'
elif rsi < oversold:
    action = 'buy'
    reason = f'RSI={rsi:.2f} oversold: buy signal'
    risk_level = 'medium'
elif rsi > overbought:
    action = 'sell'
    reason = f'RSI={rsi:.2f} overbought: sell signal'
    risk_level = 'medium'
else:
    action = 'hold'
    reason = f'RSI={rsi:.2f} no signal'
    risk_level = 'low'

# Result
signal = {
    'signal': action,
    'symbol': symbol,
    'price': close[-1],
    'confidence': 0.6,
    'reason': reason,
    'risk_level': risk_level,
    'rsi': round(rsi, 2) if rsi is not None else None
}`,
    isPublic: true,
    useCount: 0,
    createdAt: new Date().toISOString() as any,
  },
  {
    id: 'default-macd',
    nameKey: 'strategy.defaultTemplates.macd.name',
    descriptionKey: 'strategy.defaultTemplates.macd.description',
    name: 'MACD Crossover',
    description: 'Buy on MACD bullish crossover; sell on bearish crossover',
    code: `# MACD crossover strategy
# Available variables: close, open, high, low, volume, symbol
# Return: signal dict

# Parameters
fast_period = 12
slow_period = 26
signal_period = 9

# Compute EMA
def ema(prices, period):
    multiplier = 2 / (period + 1)
    ema_val = prices[0]
    for price in prices[1:]:
        ema_val = (price - ema_val) * multiplier + ema_val
    return ema_val

# Compute MACD
if len(close) < slow_period + signal_period + 2:
    signal = {
        'signal': 'hold',
        'symbol': symbol,
        'price': close[-1] if len(close) > 0 else None,
        'confidence': 0.0,
        'reason': 'insufficient data',
        'risk_level': 'low',
    }
else:
    ema_fast = ema(close, fast_period)
    ema_slow = ema(close, slow_period)
    macd_line = ema_fast - ema_slow

    # Simplified signal line calculation
    macd_history = []
    for i in range(slow_period, len(close)):
        ef = ema(close[:i+1], fast_period)
        es = ema(close[:i+1], slow_period)
        macd_history.append(ef - es)

    signal_line = ema(macd_history[-signal_period*2:], signal_period)

    # Detect crossover
    macd_prev = macd_history[-2]
    signal_prev = signal_line

    if macd_line > signal_line and macd_prev <= signal_prev:
        action = 'buy'
        reason = 'MACD bullish crossover'
        risk_level = 'medium'
    elif macd_line < signal_line and macd_prev >= signal_prev:
        action = 'sell'
        reason = 'MACD bearish crossover'
        risk_level = 'medium'
    else:
        action = 'hold'
        reason = 'no signal'
        risk_level = 'low'

    # Result
    signal = {
        'signal': action,
        'symbol': symbol,
        'price': close[-1],
        'confidence': 0.65,
        'reason': reason,
        'risk_level': risk_level,
        'macd': round(macd_line, 5),
        'signal_line': round(signal_line, 5)
    }`,
    isPublic: true,
    useCount: 0,
    createdAt: new Date().toISOString() as any,
  },
];
