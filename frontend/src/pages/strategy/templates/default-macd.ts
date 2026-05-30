import type { DefaultTemplateItem } from '../StrategyTemplatePage.defaults';

const tpl: DefaultTemplateItem = {
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
    createdAt: new Date().toISOString(),
};

export default tpl;