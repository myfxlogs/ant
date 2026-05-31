type AgentKey = 'style' | 'signals' | 'risk' | 'code';

export interface AgentPromptOverride {
	name?: string;
	identity?: string;
}

export function buildAgentPrompts(
	baseInfo: string,
	t: (key: string, opts?: Record<string, any>) => string,
	overrides?: Partial<Record<AgentKey, AgentPromptOverride>>,
): Record<AgentKey, { title: string; prompt: string }> {
	const buildFor = (key: AgentKey): { title: string; prompt: string } => {
		const ov = overrides?.[key];
		const title = ov?.name || t(`ai.agentPrompts.${key}.title`);
		// Default templates are preserved; user-defined identity takes priority, with baseInfo appended as context.
		const defaultPrompt = t(`ai.agentPrompts.${key}.prompt`, { baseInfo });
		const identity = (ov?.identity || '').trim();
		const prompt = identity
			? `${identity}\n\n${baseInfo}`
			: defaultPrompt;
		return { title, prompt };
	};

	return {
		style: buildFor('style'),
		signals: buildFor('signals'),
		risk: buildFor('risk'),
		code: buildFor('code'),
	};
}

export type { AgentKey, AgentPromptOverride };
