import type { TFunction } from 'i18next';
import type { AIAgentDefinitionView } from '@/client/ai';

/**
 * 系统内置的 8 个量化交易 Agent 默认身份定义。
 *
 * - 每个 Agent 的 agentKey 固定（用于前后端识别，避免重复）。
 * - identity 按量化交易行业常见职责划分撰写，作为 system-prompt 片段；
 *   用户可在设置页自由修改。
 * - inputHint 用于提示使用者"喂给该 Agent 什么信息效果最好"。
 *
 * 覆盖类型：style / signals / risk / macro / sentiment / portfolio / execution / code
 */

// 自 060 起 Agent 不再依附「ai_config_profiles 档」；模板里也不再包含
// id / providerId / modelOverride，由调用方在合并 / 提交时按需补齐。
export type DefaultAgentTemplate = Omit<AIAgentDefinitionView, 'id' | 'providerId' | 'modelOverride'>;

export function getDefaultAgentTemplates(t: TFunction): AIAgentDefinitionView[] {
	const base: DefaultAgentTemplate[] = [
		{
			agentKey: 'default-style',
			type: 'style',
			name: t('ai.settings.agent.types.style'),
			identity: t('ai.settings.agent.defaults.style.identity'),
			inputHint: t('ai.settings.agent.defaults.style.inputHint'),
			enabled: true,
			position: 0,
		},
		{
			agentKey: 'default-signals',
			type: 'signals',
			name: t('ai.settings.agent.types.signals'),
			identity: t('ai.settings.agent.defaults.signals.identity'),
			inputHint: t('ai.settings.agent.defaults.signals.inputHint'),
			enabled: true,
			position: 1,
		},
		{
			agentKey: 'default-risk',
			type: 'risk',
			name: t('ai.settings.agent.types.risk'),
			identity: t('ai.settings.agent.defaults.risk.identity'),
			inputHint: t('ai.settings.agent.defaults.risk.inputHint'),
			enabled: true,
			position: 2,
		},
		{
			agentKey: 'default-macro',
			type: 'macro',
			name: t('ai.settings.agent.types.macro'),
			identity: t('ai.settings.agent.defaults.macro.identity'),
			inputHint: t('ai.settings.agent.defaults.macro.inputHint'),
			enabled: true,
			position: 3,
		},
		{
			agentKey: 'default-sentiment',
			type: 'sentiment',
			name: t('ai.settings.agent.types.sentiment'),
			identity: t('ai.settings.agent.defaults.sentiment.identity'),
			inputHint: t('ai.settings.agent.defaults.sentiment.inputHint'),
			enabled: true,
			position: 4,
		},
		{
			agentKey: 'default-portfolio',
			type: 'portfolio',
			name: t('ai.settings.agent.types.portfolio'),
			identity: t('ai.settings.agent.defaults.portfolio.identity'),
			inputHint: t('ai.settings.agent.defaults.portfolio.inputHint'),
			enabled: true,
			position: 5,
		},
		{
			agentKey: 'default-execution',
			type: 'execution',
			name: t('ai.settings.agent.types.execution'),
			identity: t('ai.settings.agent.defaults.execution.identity'),
			inputHint: t('ai.settings.agent.defaults.execution.inputHint'),
			enabled: true,
			position: 6,
		},
		{
			agentKey: 'default-code',
			type: 'code',
			name: t('ai.settings.agent.types.code'),
			identity: t('ai.settings.agent.defaults.code.identity'),
			inputHint: t('ai.settings.agent.defaults.code.inputHint'),
			enabled: true,
			position: 7,
		},
	];

	return base.map((tpl) => ({ ...tpl, id: '', providerId: '', modelOverride: '' }));
}

/**
 * 将系统默认 8 个 Agent 合并进当前 agents 列表：
 * - 命中同 agentKey：按默认模板覆盖 identity/inputHint/name/type/position（保留已有 id）
 * - 未命中：追加
 * - 用户自己新增的非默认 agentKey：原样保留
 */
export function mergeWithDefaultAgentTemplates(
	current: AIAgentDefinitionView[],
	t: TFunction,
): AIAgentDefinitionView[] {
	const defaults = getDefaultAgentTemplates(t);
	const byKey = new Map<string, AIAgentDefinitionView>();
	for (const a of current) byKey.set(a.agentKey, a);
	for (const d of defaults) {
		const existing = byKey.get(d.agentKey);
		byKey.set(d.agentKey, {
			...existing,
			...d,
			id: existing?.id || '',
			// 保留用户已选的 provider/model，不被默认模板覆盖。
			providerId: existing?.providerId || '',
			modelOverride: existing?.modelOverride || '',
		});
	}
	return Array.from(byKey.values()).sort((a, b) => (a.position ?? 0) - (b.position ?? 0));
}
