import type { DefaultTemplateItem } from '../StrategyTemplatePage.defaults';

const tpl: DefaultTemplateItem = {
id: 'default-pairs-trading',
    nameKey: 'strategy.defaultTemplates.pairsTrading.name',
    descriptionKey: 'strategy.defaultTemplates.pairsTrading.description',
    name: 'Pairs Trading (Placeholder)',
    description: 'Requires multi-symbol engine; placeholder only.',
    code: `# Pairs trading placeholder
signal = {
  'signal': 'hold',
  'reason': '需要多品种引擎（占位）',
}
`,
    isPublic: true,
    tags: ['multi-symbol-required'],
    useCount: 0,
    createdAt: new Date().toISOString(),
};

export default tpl;