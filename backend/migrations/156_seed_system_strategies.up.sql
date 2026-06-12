-- 156_seed_system_strategies.up.sql
-- Seed 12 个系统预设策略（由 admin 管理端后续维护）

INSERT INTO strategy_templates (id, user_id, name, description, code, status, parameters, is_public, is_system, tags, use_count, i18n) VALUES

-- 1. MA Crossover
('00000000-0000-0000-0000-000000000001', NULL,
 'MA Crossover', 'Buy on fast MA cross above slow MA; sell on cross below',
 $code$# MA crossover strategy
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
    maFast = np.mean(close[-fast_period:])
    maSlow = np.mean(close[-slow_period:])
    ma_fast_prev = np.mean(close[-fast_period-1:-1])
    ma_slow_prev = np.mean(close[-slow_period-1:-1])
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
    signal = {
        'signal': action,
        'symbol': symbol,
        'price': close[-1],
        'confidence': 0.7,
        'reason': reason,
        'risk_level': risk_level,
        'maFast': round(maFast, 5),
        'maSlow': round(maSlow, 5)
    }$code$,
 'published', '[]', true, true, ARRAY['trend-following','ma'], 0,
 '{"name":{"en":"MA Crossover","zh":"双均线交叉"}}'),

-- 2. RSI
('00000000-0000-0000-0000-000000000002', NULL,
 'RSI Overbought/Oversold', 'Buy when RSI < 30; sell when RSI > 70',
 $code$# RSI overbought/oversold strategy
# Available variables: close, open, high, low, volume, symbol
# Return: signal dict

period = 14
oversold = 30
overbought = 70

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

signal = {
    'signal': action,
    'symbol': symbol,
    'price': close[-1],
    'confidence': 0.6,
    'reason': reason,
    'risk_level': risk_level,
    'rsi': round(rsi, 2) if rsi is not None else None
}$code$,
 'published', '[]', true, true, ARRAY['mean-reversion','rsi'], 0,
 '{"name":{"en":"RSI Overbought/Oversold","zh":"RSI超买超卖"}}'),

-- 3. MACD
('00000000-0000-0000-0000-000000000003', NULL,
 'MACD Crossover', 'Buy on MACD bullish crossover; sell on bearish crossover',
 $code$# MACD crossover strategy
fast_period = 12
slow_period = 26
signal_period = 9

def ema(prices, period):
    multiplier = 2 / (period + 1)
    ema_val = prices[0]
    for price in prices[1:]:
        ema_val = (price - ema_val) * multiplier + ema_val
    return ema_val

if len(close) < slow_period + signal_period + 2:
    signal = {'signal': 'hold', 'symbol': symbol, 'price': close[-1] if len(close) > 0 else None, 'confidence': 0.0, 'reason': 'insufficient data', 'risk_level': 'low'}
else:
    ema_fast = ema(close, fast_period)
    ema_slow = ema(close, slow_period)
    macd_line = ema_fast - ema_slow
    macd_history = []
    for i in range(slow_period, len(close)):
        ef = ema(close[:i+1], fast_period)
        es = ema(close[:i+1], slow_period)
        macd_history.append(ef - es)
    signal_line = ema(macd_history[-signal_period*2:], signal_period)
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
    signal = {'signal': action, 'symbol': symbol, 'price': close[-1], 'confidence': 0.65, 'reason': reason, 'risk_level': risk_level, 'macd': round(macd_line, 5), 'signal_line': round(signal_line, 5)}$code$,
 'published', '[]', true, true, ARRAY['trend-following','macd'], 0,
 '{"name":{"en":"MACD Crossover","zh":"MACD交叉"}}'),

-- 4. MACD Divergence
('00000000-0000-0000-0000-000000000004', NULL,
 'MACD Divergence', 'Enter on bullish/bearish MACD divergence',
 $code$# MACD Divergence (simple heuristic)
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
    p0, p1 = float(close[-3]), float(close[-1])
    h0, h1 = float(hist[-3]), float(hist[-1])
    sig = {'signal': 'hold', 'symbol': symbol, 'price': close[-1]}
    if p1 < p0 and h1 > h0:
        sig.update({'signal': 'buy', 'reason': 'bullish MACD divergence'})
    elif p1 > p0 and h1 < h0:
        sig.update({'signal': 'sell', 'reason': 'bearish MACD divergence'})
    return sig
return {'signal': 'hold', 'symbol': symbol, 'price': close[-1] if len(close)>0 else None}$code$,
 'published', '[]', true, true, ARRAY['trend','macd','divergence'], 0,
 '{"name":{"en":"MACD Divergence","zh":"MACD背离"}}'),

-- 5. RSI Oversold Bounce
('00000000-0000-0000-0000-000000000005', NULL,
 'RSI Oversold Bounce', 'Enter long when RSI bounces from oversold; exit on overbought',
 $code$# RSI Oversold Bounce
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
return signal$code$,
 'published', '[]', true, true, ARRAY['mean-reversion','rsi'], 0,
 '{"name":{"en":"RSI Oversold Bounce","zh":"RSI超卖反弹"}}'),

-- 6. Bollinger Squeeze
('00000000-0000-0000-0000-000000000006', NULL,
 'Bollinger Squeeze Breakout', 'Trade breakouts after Bollinger Band squeezes',
 $code$# Bollinger Band Squeeze Breakout
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
    if width < squeeze_threshold:
        if close[-1] > upper:
            signal.update({'signal': 'buy', 'reason': 'Upper band breakout after squeeze'})
        elif close[-1] < lower:
            signal.update({'signal': 'sell', 'reason': 'Lower band breakout after squeeze'})
return signal$code$,
 'published', '[]', true, true, ARRAY['volatility','bollinger','breakout'], 0,
 '{"name":{"en":"Bollinger Squeeze Breakout","zh":"布林带挤压突破"}}'),

-- 7. BB Mean Reversion
('00000000-0000-0000-0000-000000000007', NULL,
 'BB Mean Reversion', 'Buy at lower band, exit at upper band',
 $code$# Bollinger Band Mean Reversion
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
return signal$code$,
 'published', '[]', true, true, ARRAY['mean-reversion','bollinger'], 0,
 '{"name":{"en":"BB Mean Reversion","zh":"布林带均值回归"}}'),

-- 8. Volume Breakout
('00000000-0000-0000-0000-000000000008', NULL,
 'Volume Breakout', 'Enter on price high breakout with above-average volume; ATR stop',
 $code$# Volume Breakout with ATR stop (simplified)
import numpy as np

lookback = int(context.get('params', {}).get('lookback', 20))
vol_mult = float(context.get('params', {}).get('volume_multiplier', 1.5))
atr_mult = float(context.get('params', {}).get('atr_multiplier', 2.0))

def atr(h, l, c, p=14):
    if len(c) < p+1: return None
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
return signal$code$,
 'published', '[]', true, true, ARRAY['breakout','volume','atr'], 0,
 '{"name":{"en":"Volume Breakout","zh":"放量突破"}}'),

-- 9. Turtle Trading
('00000000-0000-0000-0000-000000000009', NULL,
 'Turtle Trading', 'Enter on N-day high, exit on M-day low',
 $code$# Turtle Trading (simplified)
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
return signal$code$,
 'published', '[]', true, true, ARRAY['trend','turtle'], 0,
 '{"name":{"en":"Turtle Trading","zh":"海龟交易"}}'),

-- 10. Grid Trading
('00000000-0000-0000-0000-000000000010', NULL,
 'Grid Trading', 'Place buy/sell orders at regular grid levels',
 $code$# Grid Trading
grid_count = int(context.get('params', {}).get('grid_count', 10)) if isinstance(context.get('params'), dict) else 10
lower = float(context.get('params', {}).get('lower_price', 0)) if isinstance(context.get('params'), dict) else 0.0
upper = float(context.get('params', {}).get('upper_price', 0)) if isinstance(context.get('params'), dict) else 0.0
lot = context.get('params', {}).get('lot', 0.01) if isinstance(context.get('params'), dict) else 0.01
try: lot = float(lot)
except Exception: lot = 0.01

price = close[-1] if len(close) > 0 else None
signal = {'signal': 'hold', 'symbol': symbol, 'price': price, 'volume': lot, 'reason': 'no grid or out of range'}
if price is not None and upper > lower and grid_count >= 2:
    step = (upper - lower) / (grid_count - 1)
    idx = int(max(0, min(grid_count - 1, (price - lower) // step)))
    level = lower + idx * step
    half = step * 0.5
    if price < level + half:
        signal.update({'signal': 'buy', 'reason': 'near lower grid'})
    elif price > level + step - half:
        signal.update({'signal': 'sell', 'reason': 'near upper grid'})
    else:
        signal.update({'signal': 'hold', 'reason': 'between grid levels'})$code$,
 'published', '[]', true, true, ARRAY['grid','market-making'], 0,
 '{"name":{"en":"Grid Trading","zh":"网格交易"}}'),

-- 11. DCA Buy
('00000000-0000-0000-0000-000000000011', NULL,
 'DCA Buy', 'Buy a fixed small lot at regular time intervals',
 $code$# DCA Buy (single-symbol)
params = context.get('params', {}) if isinstance(context.get('params'), dict) else {}
lot = params.get('lot') or params.get('buy_amount') or 0.01
try: lot = float(lot)
except Exception: lot = 0.01
interval_hours = params.get('interval_hours') or 24
try: interval_hours = int(interval_hours)
except Exception: interval_hours = 24

now_ms = context.get('bar_time_ms') if isinstance(context, dict) else None
runtime = context.get('runtime') if isinstance(context.get('runtime'), dict) else {}
should_buy = False
state_last_ms = runtime.get('last_dca_buy_ms')

if now_ms is not None:
    if state_last_ms is None: should_buy = True
    else: should_buy = (int(now_ms) - int(state_last_ms)) >= interval_hours * 3600 * 1000
else:
    N = max(1, interval_hours)
    should_buy = (len(close) % N) == 0

if should_buy and now_ms is not None:
    runtime['last_dca_buy_ms'] = int(now_ms)

signal = {'signal': 'buy' if should_buy else 'hold', 'symbol': symbol, 'price': close[-1] if len(close) > 0 else None, 'volume': lot if should_buy else None, 'reason': 'interval reached' if should_buy else 'waiting for next interval'}$code$,
 'published', '[]', true, true, ARRAY['passive','dca'], 0,
 '{"name":{"en":"DCA Buy","zh":"定投买入"}}'),

-- 12. Force BUY (Test)
('00000000-0000-0000-0000-000000000012', NULL,
 'Force BUY (Test)', 'Always returns buy signal for pipeline testing',
 $code$# Force BUY (test)
lot = None
try:
    if 'lot' in context:
        lot = context.get('lot')
    if lot is None and isinstance(context.get('params'), dict):
        lot = context.get('params', {}).get('lot')
except Exception: lot = None
try: lot = float(lot) if lot is not None else 0.01
except Exception: lot = 0.01
if lot <= 0: lot = 0.01

signal = {'signal': 'buy', 'symbol': symbol, 'price': close[-1] if len(close) > 0 else None, 'volume': lot, 'confidence': 0.5, 'reason': 'force buy for pipeline test', 'risk_level': 'high'}$code$,
 'published', '[]', true, true, ARRAY['test'], 0,
 '{"name":{"en":"Force BUY (Test)","zh":"强制买入(测试)"}}')

ON CONFLICT (id) DO NOTHING;
