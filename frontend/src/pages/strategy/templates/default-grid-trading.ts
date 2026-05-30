import type { DefaultTemplateItem } from '../StrategyTemplatePage.defaults';

const tpl: DefaultTemplateItem = {
id: 'default-grid-trading',
    nameKey: 'strategy.defaultTemplates.gridTrading.name',
    descriptionKey: 'strategy.defaultTemplates.gridTrading.description',
    name: 'Grid Trading',
    description: 'Place buy/sell orders at regular grid levels within a price range; range-bound friendly.',
    code: `# Grid Trading (single-symbol, stateless approximation using pending orders idea)
# Inputs (from params or context): grid_count, lower_price, upper_price, lot

grid_count = int(context.get('params', {}).get('grid_count', 10)) if isinstance(context.get('params'), dict) else 10
lower = float(context.get('params', {}).get('lower_price', 0)) if isinstance(context.get('params'), dict) else 0.0
upper = float(context.get('params', {}).get('upper_price', 0)) if isinstance(context.get('params'), dict) else 0.0
lot = context.get('params', {}).get('lot', 0.01) if isinstance(context.get('params'), dict) else 0.01
try:
    lot = float(lot)
except Exception:
    lot = 0.01

price = close[-1] if len(close) > 0 else None
signal = {
    'signal': 'hold',
    'symbol': symbol,
    'price': price,
    'volume': lot,
    'reason': 'no grid or out of range',
}
if price is not None and upper > lower and grid_count >= 2:
    step = (upper - lower) / (grid_count - 1)
    # find nearest grid below and above
    idx = int(max(0, min(grid_count - 1, (price - lower) // step)))
    level = lower + idx * step
    # simple rule: if price below level by < half step -> buy; if above next level by < half step -> sell
    half = step * 0.5
    if price < level + half:
        signal.update({'signal': 'buy', 'reason': 'near lower grid'})
    elif price > level + step - half:
        signal.update({'signal': 'sell', 'reason': 'near upper grid'})
    else:
        signal.update({'signal': 'hold', 'reason': 'between grid levels'})
`,
    isPublic: true,
    tags: ['grid', 'market-making'],
    useCount: 0,
    createdAt: new Date().toISOString(),
  }
};

export default tpl;