import type { DefaultTemplateItem } from '../StrategyTemplatePage.defaults';

const tpl: DefaultTemplateItem = {
id: 'default-momentum-rotation',
    nameKey: 'strategy.defaultTemplates.momentumRotation.name',
    descriptionKey: 'strategy.defaultTemplates.momentumRotation.description',
    name: 'Momentum Rotation (Placeholder)',
    description: 'Requires multi-symbol engine (N>=3); placeholder only.',
    code: `# Momentum rotation placeholder
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