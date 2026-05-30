import type { DefaultTemplateItem } from '../StrategyTemplatePage.defaults';

const tpl: DefaultTemplateItem = {
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
    createdAt: new Date().toISOString(),
  }
};

export default tpl;