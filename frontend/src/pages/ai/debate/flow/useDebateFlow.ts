import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import type { AIAgentDefinitionView } from '@/client/ai';
import type { V2Session, V2Step, V2Usage } from '@/client/debateV2';
import type { ChatMessage, StepKey, StepState, CodeState, UseDebateFlowResult } from './types';
import {
  endModelWait,
  emptyStep,
  stepKeyForAgent,
  createStreamMerger,
} from './helpers';
import {
  createStartFlow,
  createSendMessage,
  createAdvance,
  createRejectCode,
  createRetryCodeGen,
  createBack,
} from './actions';

export type { ChatMessage, StepKey } from './types';

export function useDebateFlow(): UseDebateFlowResult {
  const { t, i18n } = useTranslation();
  const locale = i18n.language || 'zh-CN';

  const [selectedAgents, setSelectedAgentsRaw] = useState<AIAgentDefinitionView[]>([]);
  const [session, setSession] = useState<V2Session | null>(null);
  const [pending, setPending] = useState<{ stepKey: StepKey; user?: ChatMessage; loading: ChatMessage } | null>(null);
  const [codeLoading, setCodeLoading] = useState(false);
  const [sending, setSending] = useState(false);
  const sendingRef = useRef(false);
  const [forceSelection, setForceSelection] = useState(false);

  const [modelWaitStartedAt, setModelWaitStartedAt] = useState<number | null>(null);
  const [modelWaitElapsedSeconds, setModelWaitElapsedSeconds] = useState(0);
  const [optimisticDisplayStep, setOptimisticDisplayStep] = useState<StepKey | null>(null);
  const [advanceStreamPreview, setAdvanceStreamPreview] = useState('');
  const [chatStreamPreview, setChatStreamPreview] = useState('');

  const mergeAdvanceStreamChunk = useCallback(createStreamMerger(setAdvanceStreamPreview), []);
  const mergeChatStreamChunk = useCallback(createStreamMerger(setChatStreamPreview), []);

  const modelWaitActive = useMemo(() => {
    if (modelWaitStartedAt == null) return false;
    return sending || codeLoading || !!pending?.loading?.isLoading;
  }, [modelWaitStartedAt, sending, codeLoading, pending]);

  useEffect(() => {
    if (modelWaitStartedAt == null || !modelWaitActive) return;
    const tick = () => setModelWaitElapsedSeconds(Math.max(0, Math.floor((Date.now() - modelWaitStartedAt) / 1000)));
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
      return { id: '', agentKey: key, type: key, name: key, identity: '', inputHint: '', enabled: true, position: 0, providerId: '', modelOverride: '' };
    });
  }, [session, selectedAgents]);

  const stepLabels = useMemo(() => {
    const list: Array<{ key: StepKey; label: string }> = [
      { key: 'agent_selection', label: t('ai.debate.v2.steps.agentSelection', { defaultValue: 'Choose experts' }) },
      { key: 'intent', label: t('ai.debate.v2.steps.intent', { defaultValue: 'Clarify intent' }) },
    ];
    for (const a of effectiveAgents) {
      const builtin = ['style', 'signals', 'risk', 'macro', 'sentiment', 'portfolio', 'execution', 'code'].includes(a.type);
      const label = builtin ? t(`ai.settings.agent.types.${a.type}`, { defaultValue: a.type }) : a.name || a.type;
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
        id: m.id, role: m.role, content: m.content,
        kind: m.kind === 'kickoff' ? 'kickoff' : undefined,
      }));
      out[s.stepKey] = { messages, extractedPrompt: '', promptDraft: '' };
    }
    return out;
  }, [session]);

  const stepState = useCallback((key: StepKey): StepState => {
    const base = stepsByKey[key] || emptyStep();
    if (!pending || pending.stepKey !== key) return base;
    const loadingMsg: ChatMessage = { ...pending.loading, content: chatStreamPreview, isLoading: true };
    return { ...base, messages: [...base.messages, ...(pending.user ? [pending.user] : []), loadingMsg] };
  }, [stepsByKey, pending, chatStreamPreview]);

  const code: CodeState = useMemo(() => {
    const c = session?.code;
    const stream = advanceStreamPreview.trim();
    return { text: stream || c?.text || '', python: c?.python || '', loading: codeLoading, elapsedSeconds: codeLoading ? modelWaitElapsedSeconds : 0 };
  }, [session, codeLoading, modelWaitElapsedSeconds, advanceStreamPreview]);

  const setSelectedAgents = useCallback((agents: AIAgentDefinitionView[]) => { setSelectedAgentsRaw(agents); }, []);
  const updatePromptDraft = useCallback((_key: StepKey, _text: string) => {}, []);

  const actionCtx = useMemo(() => ({
    session, selectedAgents, sessionStep, stepLabels, locale, t,
    sendingRef, setSession, setSending, setCodeLoading, setOptimisticDisplayStep,
    setAdvanceStreamPreview, setChatStreamPreview, setPending, setForceSelection,
    mergeAdvanceStreamChunk, mergeChatStreamChunk,
    modelWaitSetters: { setStartedAt: setModelWaitStartedAt, setElapsed: setModelWaitElapsedSeconds },
  }), [session, selectedAgents, sessionStep, stepLabels, locale, t,
      mergeAdvanceStreamChunk, mergeChatStreamChunk]);

  const startFlow = useCallback(createStartFlow(actionCtx), [actionCtx]);
  const sendMessage = useCallback(createSendMessage(actionCtx), [actionCtx]);
  const advance = useCallback(createAdvance(actionCtx), [actionCtx]);
  const rejectCode = useCallback(createRejectCode(actionCtx), [actionCtx]);
  const retryCodeGeneration = useCallback(createRetryCodeGen(actionCtx), [actionCtx]);
  const back = useCallback(createBack(actionCtx), [actionCtx]);

  const reset = useCallback(() => {
    setSession(null); setSelectedAgentsRaw([]); setPending(null); setCodeLoading(false);
    sendingRef.current = false; setSending(false); setForceSelection(true);
    setOptimisticDisplayStep(null); setAdvanceStreamPreview(''); setChatStreamPreview('');
    endModelWait(setModelWaitStartedAt as unknown as (v: null) => void, setModelWaitElapsedSeconds);
  }, []);

  const usage: V2Usage = session?.usage || { promptTokens: 0, completionTokens: 0, totalTokens: 0 };

  return {
    currentStep, stepIndex, stepLabels, selectedAgents, sending,
    modelWaitActive, modelWaitElapsedSeconds,
    stepState, updatePromptDraft,
    setSelectedAgents, startFlow, sendMessage, advance, back, reset, rejectCode, retryCodeGeneration,
    code, advanceStreamPreview,
    sessionId: session?.id || '', provider: session?.provider || '', model: session?.model || '', usage,
  };
}
