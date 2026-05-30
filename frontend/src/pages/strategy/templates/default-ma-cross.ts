import type { DefaultTemplateItem } from '../StrategyTemplatePage.defaults';

const tpl: DefaultTemplateItem = {
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
    createdAt: new Date().toISOString(),
  }
};

export default tpl;