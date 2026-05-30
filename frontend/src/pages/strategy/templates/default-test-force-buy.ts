import type { DefaultTemplateItem } from '../StrategyTemplatePage.defaults';

const tpl: DefaultTemplateItem = {
id: 'default-test-force-buy',
    nameKey: 'strategy.defaultTemplates.forceBuy.name',
    descriptionKey: 'strategy.defaultTemplates.forceBuy.description',
    name: 'Force BUY (Test)',
    description: 'Used to validate order pipeline: always returns buy; reads lot from context/params as volume',
    code: `# Force BUY (test)
# Reads 'lot' from context/params and always emits a BUY signal.

lot = None
try:
    if 'lot' in context:
        lot = context.get('lot')
    if lot is None and isinstance(context.get('params'), dict):
        lot = context.get('params', {}).get('lot')
except Exception:
    lot = None

try:
    lot = float(lot) if lot is not None else 0.01
except Exception:
    lot = 0.01

if lot <= 0:
    lot = 0.01

signal = {
    'signal': 'buy',
    'symbol': symbol,
    'price': close[-1] if len(close) > 0 else None,
    'volume': lot,
    'confidence': 0.5,
    'reason': 'force buy for pipeline test',
    'risk_level': 'high',
}`, 
    isPublic: true,
    useCount: 0,
    createdAt: new Date().toISOString(),
  }
};

export default tpl;