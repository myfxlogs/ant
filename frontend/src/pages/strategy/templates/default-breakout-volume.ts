import type { DefaultTemplateItem } from '../StrategyTemplatePage.defaults';

const tpl: DefaultTemplateItem = {
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
    createdAt: new Date().toISOString(),
};

export default tpl;