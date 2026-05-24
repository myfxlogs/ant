import { useCallback, useEffect, useMemo, useState } from 'react';
import { message as antMessage } from 'antd';
import { useTranslation } from 'react-i18next';
import { toFriendlyAIError, type AIAgentDefinitionView } from '@/client/ai';
import { debateV2Api, type V2Session, type V2Step, type V2Usage, waitAdvanceJob, waitChatJob } from '@/client/debateV2';

// useDebateFlow: advance uses StartAdvanceJob + SSE; chat uses PrepareChatJob → waitChatJob (SSE) → RunChatJob.

export type ChatMessage = {
	id: string;
	role: 'user' | 'assistant';
	content: string;
	isLoading?: boolean;
	/** 'kickoff' means a hidden system-handoff user turn; UI hides it. */
	kind?: 'kickoff';
};

/**
 * StepKey mirrors the backend step naming with an additional UI-only
 * 'agent_selection' sentinel used before the session exists.
 */
export type StepKey = 'agent_selection' | 'intent' | `agent:${string}` | 'code';

interface StepState {
	messages: ChatMessage[];
	extractedPrompt: string;
	promptDraft: string;
}

interface CodeState {
	text: string;
	python: string;
	loading: boolean;
	elapsedSeconds: number;
}

function emptyStep(): StepState {
	return { messages: [], extractedPrompt: '', promptDraft: '' };
}

function stepKeyForAgent(agent: AIAgentDefinitionView): StepKey {
	return `agent:${agent.agentKey || agent.type}`;
}

function mkId(prefix: string): string {
	return `${prefix}-${Date.now()}-${Math.floor(Math.random() * 1e6)}`;
}

function friendlyError(e: unknown): string {
	return toFriendlyAIError(e);
}

export interface UseDebateFlowResult {
	currentStep: StepKey;
	stepIndex: number;
	stepLabels: Array<{ key: StepKey; label: string }>;
	selectedAgents: AIAgentDefinitionView[];
	sending: boolean;

	/** True while a unary RPC that calls the LLM is in flight (chat / advance / reject). */
	modelWaitActive: boolean;
	/** Monotonic seconds since modelWait started; 0 when inactive. */
	modelWaitElapsedSeconds: number;

	stepState: (key: StepKey) => StepState;
	updatePromptDraft: (key: StepKey, text: string) => void;

	setSelectedAgents: (agents: AIAgentDefinitionView[]) => void;
	startFlow: () => Promise<void>;
	sendMessage: (text: string) => Promise<void>;
	advance: () => Promise<void>;
	back: () => Promise<void>;
	reset: () => void;
	rejectCode: (feedback: string) => Promise<void>;
	/** When the server is on the code step but no code arrived (e.g. edge 524), go back one step and advance again to re-run generation. */
	retryCodeGeneration: () => Promise<void>;

	code: CodeState;
	/** Raw LLM stream during async advance (agent kickoff / code gen); cleared when idle. */
	advanceStreamPreview: string;
	sessionId: string;
	provider: string;
	model: string;
	usage: V2Usage;
}

function beginModelWait(
	setStartedAt: (n: number) => void,
	setElapsed: (n: number) => void,
) {
	setStartedAt(Date.now());
	setElapsed(0);
}

function endModelWait(
	setStartedAt: (v: null) => void,
	setElapsed: (n: number) => void,
) {
	setStartedAt(null);
	setElapsed(0);
}

export function useDebateFlow(): UseDebateFlowResult {
	const { t, i18n } = useTranslation();
	const locale = i18n.language || 'zh-CN';

	const [selectedAgents, setSelectedAgentsRaw] = useState<AIAgentDefinitionView[]>([]);
	const [session, setSession] = useState<V2Session | null>(null);
	const [pending, setPending] = useState<{ stepKey: StepKey; user?: ChatMessage; loading: ChatMessage } | null>(null);
	const [codeLoading, setCodeLoading] = useState(false);
	const [sending, setSending] = useState(false);
	const [forceSelection, setForceSelection] = useState(false);

	const [modelWaitStartedAt, setModelWaitStartedAt] = useState<number | null>(null);
	const [modelWaitElapsedSeconds, setModelWaitElapsedSeconds] = useState(0);
	/** While Advance RPC is in flight, show the target step immediately (server session updates only after the call returns). */
	const [optimisticDisplayStep, setOptimisticDisplayStep] = useState<StepKey | null>(null);
	const [advanceStreamPreview, setAdvanceStreamPreview] = useState('');
	const [chatStreamPreview, setChatStreamPreview] = useState('');

	/** Catch-up SSE sends full prefix; live sends deltas — avoid double text. */
	const mergeAdvanceStreamChunk = useCallback((delta: string) => {
		if (!delta) return;
		setAdvanceStreamPreview((prev) => {
			if (prev === delta) return prev;
			if (prev === '') return delta;
			if (delta.startsWith(prev)) return delta;
			return prev + delta;
		});
	}, []);

	const mergeChatStreamChunk = useCallback((delta: string) => {
		if (!delta) return;
		setChatStreamPreview((prev) => {
			if (prev === delta) return prev;
			if (prev === '') return delta;
			if (delta.startsWith(prev)) return delta;
			return prev + delta;
		});
	}, []);

	const modelWaitActive = useMemo(() => {
		if (modelWaitStartedAt == null) return false;
		return sending || codeLoading || !!pending?.loading?.isLoading;
	}, [modelWaitStartedAt, sending, codeLoading, pending]);

	useEffect(() => {
		if (modelWaitStartedAt == null || !modelWaitActive) return;
		const tick = () =>
			setModelWaitElapsedSeconds(Math.max(0, Math.floor((Date.now() - modelWaitStartedAt) / 1000)));
		tick();
		const timer = window.setInterval(tick, 1000);
		return () => window.clearInterval(timer);
	}, [modelWaitStartedAt, modelWaitActive]);

	const sessionStep: StepKey = useMemo(() => {
		if (forceSelection || !session) return 'agent_selection';
		const raw = session.currentStep || 'intent';
		if (raw === 'done') return 'code';
		return raw as StepKey;
	}, [session, forceSelection]);

	const currentStep: StepKey = optimisticDisplayStep ?? sessionStep;

	const effectiveAgents = useMemo<AIAgentDefinitionView[]>(() => {
		if (!session) return selectedAgents;
		const byKey = new Map(selectedAgents.map((a) => [a.agentKey || a.type, a]));
		return (session.agents || []).map<AIAgentDefinitionView>((key) => {
			const match = byKey.get(key);
			if (match) return match;
			return {
				id: '',
				agentKey: key,
				type: key,
				name: key,
				identity: '',
				inputHint: '',
				enabled: true,
				position: 0,
				providerId: '',
				modelOverride: '',
			};
		});
	}, [session, selectedAgents]);

	const stepLabels = useMemo(() => {
		const list: Array<{ key: StepKey; label: string }> = [
			{ key: 'agent_selection', label: t('ai.debate.v2.steps.agentSelection', { defaultValue: 'Choose experts' }) },
			{ key: 'intent', label: t('ai.debate.v2.steps.intent', { defaultValue: 'Clarify intent' }) },
		];
		for (const a of effectiveAgents) {
			const builtin = ['style', 'signals', 'risk', 'macro', 'sentiment', 'portfolio', 'execution', 'code'].includes(a.type);
			const label = builtin
				? t(`ai.settings.agent.types.${a.type}`, { defaultValue: a.type })
				: a.name || a.type;
			list.push({ key: stepKeyForAgent(a), label });
		}
		list.push({ key: 'code', label: t('ai.debate.v2.steps.code', { defaultValue: 'Generate code' }) });
		return list;
	}, [effectiveAgents, t]);

	const stepIndex = useMemo(() => {
		const i = stepLabels.findIndex((s) => s.key === currentStep);
		return i < 0 ? 0 : i;
	}, [stepLabels, currentStep]);

	const stepsByKey = useMemo<Record<string, StepState>>(() => {
		const out: Record<string, StepState> = {};
		if (!session) return out;
		for (const s of (session.steps || []) as V2Step[]) {
			const messages: ChatMessage[] = (s.messages || []).map((m) => ({
				id: m.id,
				role: m.role,
				content: m.content,
				kind: m.kind === 'kickoff' ? 'kickoff' : undefined,
			}));
			out[s.stepKey] = { messages, extractedPrompt: '', promptDraft: '' };
		}
		return out;
	}, [session]);

	const stepState = useCallback(
		(key: StepKey): StepState => {
			const base = stepsByKey[key] || emptyStep();
			if (!pending || pending.stepKey !== key) return base;
			const loadingMsg: ChatMessage = {
				...pending.loading,
				content: chatStreamPreview,
				isLoading: true,
			};
			return {
				...base,
				messages: [...base.messages, ...(pending.user ? [pending.user] : []), loadingMsg],
			};
		},
		[stepsByKey, pending, chatStreamPreview],
	);

	const code: CodeState = useMemo(() => {
		const c = session?.code;
		const stream = advanceStreamPreview.trim();
		return {
			text: stream || c?.text || '',
			python: c?.python || '',
			loading: codeLoading,
			elapsedSeconds: codeLoading ? modelWaitElapsedSeconds : 0,
		};
	}, [session, codeLoading, modelWaitElapsedSeconds, advanceStreamPreview]);

	const setSelectedAgents = useCallback((agents: AIAgentDefinitionView[]) => {
		setSelectedAgentsRaw(agents);
	}, []);

	const updatePromptDraft = useCallback((_key: StepKey, _text: string) => {}, []);

	const reset = useCallback(() => {
		setSession(null);
		setSelectedAgentsRaw([]);
		setPending(null);
		setCodeLoading(false);
		setSending(false);
		setForceSelection(true);
		setOptimisticDisplayStep(null);
		setAdvanceStreamPreview('');
		setChatStreamPreview('');
		endModelWait(setModelWaitStartedAt, setModelWaitElapsedSeconds);
	}, []);

	const startFlow = useCallback(async () => {
		setSending(true);
		try {
			const keys = selectedAgents.map((a) => a.agentKey || a.type);
			const resp = await debateV2Api.start({ agents: keys, locale });
			setSession(resp);
			setForceSelection(false);
		} catch (e) {
			antMessage.error(friendlyError(e));
		} finally {
			setSending(false);
		}
	}, [selectedAgents, locale]);

	const sendMessage = useCallback(
		async (text: string) => {
			const trimmed = String(text || '').trim();
			if (!trimmed || !session) return;
			if (sending) return;
			if (sessionStep === 'agent_selection' || sessionStep === 'code') return;

			const userMsg: ChatMessage = { id: mkId('u'), role: 'user', content: trimmed };
			const loadingMsg: ChatMessage = { id: mkId('a'), role: 'assistant', content: '', isLoading: true };
			beginModelWait(setModelWaitStartedAt, setModelWaitElapsedSeconds);
			setChatStreamPreview('');
			setPending({ stepKey: sessionStep, user: userMsg, loading: loadingMsg });
			setSending(true);
			try {
				const { jobId } = await debateV2Api.prepareChatJob({ sessionId: session.id, message: trimmed, locale });
				const waitP = waitChatJob(jobId, { onChunk: mergeChatStreamChunk });
				await debateV2Api.runChatJob({ jobId });
				await waitP;
				setSession(await debateV2Api.getSession(session.id, locale));
			} catch (e) {
				antMessage.error(friendlyError(e));
				try {
					setSession(await debateV2Api.getSession(session.id, locale));
				} catch {
					// ignore resync failure
				}
			} finally {
				setChatStreamPreview('');
				setPending(null);
				setSending(false);
				endModelWait(setModelWaitStartedAt, setModelWaitElapsedSeconds);
			}
		},
		[session, sending, sessionStep, locale, mergeChatStreamChunk],
	);

	const advance = useCallback(async () => {
		if (!session) return;
		if (sending) return;
		const sessionId = session.id;
		const idx = stepLabels.findIndex((s) => s.key === sessionStep);
		const nextStepKey = idx >= 0 ? stepLabels[idx + 1]?.key : undefined;
		if (nextStepKey) {
			setOptimisticDisplayStep(nextStepKey);
		}
		const willEnterCode = nextStepKey === 'code';
		const willAsyncAdvance =
			willEnterCode || (typeof nextStepKey === 'string' && nextStepKey.startsWith('agent:'));
		if (willEnterCode) {
			setCodeLoading(true);
		}
		beginModelWait(setModelWaitStartedAt, setModelWaitElapsedSeconds);
		setSending(true);
		setAdvanceStreamPreview('');
		try {
			if (willAsyncAdvance) {
				const { jobId } = await debateV2Api.startAdvanceJob({ sessionId, locale });
				await waitAdvanceJob(jobId, { onChunk: mergeAdvanceStreamChunk });
				setSession(await debateV2Api.getSession(sessionId, locale));
			} else {
				setSession(await debateV2Api.advance({ sessionId, locale }));
			}
		} catch (e) {
			antMessage.error(friendlyError(e));
			try {
				const synced = await debateV2Api.getSession(sessionId, locale);
				setSession(synced);
			} catch {
				// ignore resync failure
			}
		} finally {
			setAdvanceStreamPreview('');
			setCodeLoading(false);
			setSending(false);
			endModelWait(setModelWaitStartedAt, setModelWaitElapsedSeconds);
			setOptimisticDisplayStep(null);
		}
	}, [session, sending, sessionStep, stepLabels, locale, mergeAdvanceStreamChunk]);

	const rejectCode = useCallback(async (feedback: string) => {
		if (!session) return;
		const sessionId = session.id;
		const fb = String(feedback || '').trim();
		if (!fb) return;
		setCodeLoading(true);
		beginModelWait(setModelWaitStartedAt, setModelWaitElapsedSeconds);
		setSending(true);
		setAdvanceStreamPreview('');
		try {
			const { jobId } = await debateV2Api.startRejectCodeJob({ sessionId, feedback: fb, locale });
			await waitAdvanceJob(jobId, { onChunk: mergeAdvanceStreamChunk });
			setSession(await debateV2Api.getSession(sessionId, locale));
		} catch (e) {
			antMessage.error(friendlyError(e));
			try {
				setSession(await debateV2Api.getSession(sessionId, locale));
			} catch {
				// ignore
			}
		} finally {
			setAdvanceStreamPreview('');
			setCodeLoading(false);
			setSending(false);
			endModelWait(setModelWaitStartedAt, setModelWaitElapsedSeconds);
		}
	}, [session, locale, mergeAdvanceStreamChunk]);

	const retryCodeGeneration = useCallback(async () => {
		if (!session?.id || sending) return;
		const sessionId = session.id;
		setSending(true);
		setCodeLoading(true);
		beginModelWait(setModelWaitStartedAt, setModelWaitElapsedSeconds);
		try {
			let s = await debateV2Api.getSession(sessionId, locale);
			setSession(s);
			if (s.currentStep !== 'code') {
				antMessage.warning(
					t('ai.debate.v2.retryCodeWrongStep', {
						defaultValue: 'Session is not on the code step. The page was refreshed from the server.',
					}),
				);
				return;
			}
			const hasBody = Boolean((s.code?.python || '').trim() || (s.code?.text || '').trim());
			if (hasBody) {
				antMessage.info(
					t('ai.debate.v2.retryCodeAlreadyHave', {
						defaultValue: 'Code is already present.',
					}),
				);
				return;
			}
			await debateV2Api.back({ sessionId, locale });
			s = await debateV2Api.getSession(sessionId, locale);
			setSession(s);
			setAdvanceStreamPreview('');
			const { jobId } = await debateV2Api.startAdvanceJob({ sessionId, locale });
			await waitAdvanceJob(jobId, { onChunk: mergeAdvanceStreamChunk });
			setSession(await debateV2Api.getSession(sessionId, locale));
		} catch (e) {
			antMessage.error(friendlyError(e));
			try {
				setSession(await debateV2Api.getSession(sessionId, locale));
			} catch {
				// ignore
			}
		} finally {
			setAdvanceStreamPreview('');
			setCodeLoading(false);
			setSending(false);
			endModelWait(setModelWaitStartedAt, setModelWaitElapsedSeconds);
		}
	}, [session, sending, locale, t, mergeAdvanceStreamChunk]);

	const back = useCallback(async () => {
		if (!session) return;
		setSending(true);
		try {
			const next = await debateV2Api.back({ sessionId: session.id, locale });
			setSession(next);
		} catch (e) {
			antMessage.error(friendlyError(e));
		} finally {
			setSending(false);
			setOptimisticDisplayStep(null);
		}
	}, [session, locale]);

	const usage: V2Usage = session?.usage || { promptTokens: 0, completionTokens: 0, totalTokens: 0 };

	return {
		currentStep,
		stepIndex,
		stepLabels,
		selectedAgents,
		sending,
		modelWaitActive,
		modelWaitElapsedSeconds,
		stepState,
		updatePromptDraft,
		setSelectedAgents,
		startFlow,
		sendMessage,
		advance,
		back,
		reset,
		rejectCode,
		retryCodeGeneration,
		code,
		advanceStreamPreview,
		sessionId: session?.id || '',
		provider: session?.provider || '',
		model: session?.model || '',
		usage,
	};
}
