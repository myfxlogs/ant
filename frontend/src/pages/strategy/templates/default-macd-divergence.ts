import type { DefaultTemplateItem } from '../StrategyTemplatePage.defaults';

const tpl: DefaultTemplateItem = {
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
    createdAt: new Date().toISOString(),
  }
};

export default tpl;