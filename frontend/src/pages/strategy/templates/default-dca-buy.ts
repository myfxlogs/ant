import type { DefaultTemplateItem } from '../StrategyTemplatePage.defaults';

const tpl: DefaultTemplateItem = {
id: 'default-dca-buy',
    nameKey: 'strategy.defaultTemplates.dcaBuy.name',
    descriptionKey: 'strategy.defaultTemplates.dcaBuy.description',
    name: 'DCA Buy',
    description: 'Buy a fixed small lot at regular time intervals; long-term averaging.',
    code: `# DCA Buy (single-symbol)
# Inputs: buy_amount (or lot), interval_hours, timeframe
# Requires context['bar_time_ms'] ideally; fallback to bar index cadence if missing.

params = context.get('params', {}) if isinstance(context.get('params'), dict) else {}
lot = params.get('lot') or params.get('buy_amount') or 0.01
try:
    lot = float(lot)
except Exception:
    lot = 0.01
interval_hours = params.get('interval_hours') or 24
try:
    interval_hours = int(interval_hours)
except Exception:
    interval_hours = 24

now_ms = context.get('bar_time_ms') if isinstance(context, dict) else None
runtime = context.get('runtime') if isinstance(context.get('runtime'), dict) else {}
should_buy = False
state_last_ms = runtime.get('last_dca_buy_ms')

if now_ms is not None:
    if state_last_ms is None:
        should_buy = True
    else:
        should_buy = (int(now_ms) - int(state_last_ms)) >= interval_hours * 3600 * 1000
else:
    # Fallback: use bar index cadence when bar_time_ms is unavailable.
    N = max(1, interval_hours)
    should_buy = (len(close) % N) == 0

if should_buy and now_ms is not None:
    runtime['last_dca_buy_ms'] = int(now_ms)

signal = {
    'signal': 'buy' if should_buy else 'hold',
    'symbol': symbol,
    'price': close[-1] if len(close) > 0 else None,
    'volume': lot if should_buy else None,
    'reason': 'interval reached' if should_buy else 'waiting for next interval',
}
`,
    isPublic: true,
    tags: ['passive', 'DCA'],
    useCount: 0,
    createdAt: new Date().toISOString(),
  }
};

export default tpl;