import type { DefaultTemplateItem } from '../StrategyTemplatePage.defaults';

const tpl: DefaultTemplateItem = {
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
    createdAt: new Date().toISOString(),
  }
};

export default tpl;