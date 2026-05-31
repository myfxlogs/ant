import type { TFunction } from 'i18next';
import type { AIAgentDefinitionView } from '@/client/ai';

/**
 * 8 built-in quantitative trading Agent default identity definitions.
 *
 * - Each Agent's agentKey is fixed (for frontend/backend identification, avoiding duplicates).
 * - identity is written according to common quant trading industry role divisions, used as system-prompt fragments;
 *   Users can freely modify on the settings page.
 * - inputHint prompts the user on what information works best to feed to this Agent.
 *
 * Coverage types: style / signals / risk / macro / sentiment / portfolio / execution / code
 */

// Since 060, Agents no longer depend on ai_config_profiles; templates also no longer contain
// id / providerId / modelOverride — filled by caller on merge/submit as needed.
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
 * Merge 8 system default Agents into the current agents list:
 * - Matching agentKey: overwrite identity/inputHint/name/type/position with defaults (preserve existing id)
 * - No match: append
 * - User-added non-default agentKey: keep as-is
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
			// Preserve user-selected provider/model — don't overwrite with defaults.
			providerId: existing?.providerId || '',
			modelOverride: existing?.modelOverride || '',
		});
	}
	return Array.from(byKey.values()).sort((a, b) => (a.position ?? 0) - (b.position ?? 0));
}
