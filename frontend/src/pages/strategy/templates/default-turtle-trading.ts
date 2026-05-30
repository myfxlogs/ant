import type { DefaultTemplateItem } from '../StrategyTemplatePage.defaults';

const tpl: DefaultTemplateItem = {
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
    createdAt: new Date().toISOString(),
};

export default tpl;