import type { DefaultTemplateItem } from '../StrategyTemplatePage.defaults';

const tpl: DefaultTemplateItem = {
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
    createdAt: new Date().toISOString(),
  }
};

export default tpl;