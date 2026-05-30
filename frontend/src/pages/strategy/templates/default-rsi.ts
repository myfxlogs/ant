import type { DefaultTemplateItem } from '../StrategyTemplatePage.defaults';

const tpl: DefaultTemplateItem = {
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
    createdAt: new Date().toISOString(),
};

export default tpl;