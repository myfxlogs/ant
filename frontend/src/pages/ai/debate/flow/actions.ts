import { message as antMessage } from 'antd';
import type { TFunction } from 'i18next';
import type { AIAgentDefinitionView } from '@/client/ai';
import { debateV2Api, waitAdvanceJob, waitChatJob } from '@/client/debateV2';
import type { V2Session } from '@/client/debateV2';
import type { ChatMessage, StepKey } from './types';
import { beginModelWait, endModelWait, mkId, friendlyError } from './helpers';

interface ActionContext {
  session: V2Session | null;
  selectedAgents: AIAgentDefinitionView[];
  sessionStep: StepKey;
  stepLabels: Array<{ key: StepKey; label: string }>;
  locale: string;
  t: TFunction;
  sendingRef: React.MutableRefObject<boolean>;
  setSession: (s: V2Session | null) => void;
  setSending: (v: boolean) => void;
  setCodeLoading: (v: boolean) => void;
  setOptimisticDisplayStep: (v: StepKey | null) => void;
  setAdvanceStreamPreview: (v: string) => void;
  setChatStreamPreview: (v: string) => void;
  setPending: (p: { stepKey: StepKey; user?: ChatMessage; loading: ChatMessage } | null) => void;
  setForceSelection: (v: boolean) => void;
  mergeAdvanceStreamChunk: (delta: string) => void;
  mergeChatStreamChunk: (delta: string) => void;
  modelWaitSetters: {
    setStartedAt: (n: number) => void;
    setElapsed: (n: number) => void;
  };
}

function getModelWaitSetters(s: { setStartedAt: (n: number) => void; setElapsed: (n: number) => void }) {
  return {
    begin: () => beginModelWait(s.setStartedAt, s.setElapsed),
    end: () => endModelWait(s.setStartedAt as unknown as (v: null) => void, s.setElapsed),
  };
}

export function createStartFlow(ctx: ActionContext) {
  return async () => {
    const keys = ctx.selectedAgents.map((a) => a.agentKey || a.type);
    if (keys.length === 0) {
      antMessage.warning('Please select at least one agent to start the debate.');
      return;
    }
    ctx.sendingRef.current = true; ctx.setSending(true);
    try {
      const resp = await debateV2Api.start({ agents: keys, locale: ctx.locale });
      ctx.setSession(resp);
      ctx.setForceSelection(false);
    } catch (e) {
      antMessage.error(friendlyError(e));
    } finally {
      ctx.sendingRef.current = false; ctx.setSending(false);
    }
  };
}

export function createSendMessage(ctx: ActionContext) {
  return async (text: string) => {
    const trimmed = String(text || '').trim();
    if (!trimmed || !ctx.session) return;
    if (ctx.sendingRef.current) return;
    if (ctx.sessionStep === 'agent_selection' || ctx.sessionStep === 'code') return;

    const userMsg: ChatMessage = { id: mkId('u'), role: 'user', content: trimmed };
    const loadingMsg: ChatMessage = { id: mkId('a'), role: 'assistant', content: '', isLoading: true };
    const mw = getModelWaitSetters(ctx.modelWaitSetters);
    mw.begin();
    ctx.setChatStreamPreview('');
    ctx.setPending({ stepKey: ctx.sessionStep, user: userMsg, loading: loadingMsg });
    ctx.sendingRef.current = true; ctx.setSending(true);
    try {
      const { jobId } = await debateV2Api.prepareChatJob({ sessionId: ctx.session.id, message: trimmed, locale: ctx.locale });
      const waitP = waitChatJob(jobId, { onChunk: ctx.mergeChatStreamChunk });
      await debateV2Api.runChatJob({ jobId });
      await waitP;
      ctx.setSession(await debateV2Api.getSession(ctx.session.id, ctx.locale));
    } catch (e) {
      antMessage.error(friendlyError(e));
      try {
        ctx.setSession(await debateV2Api.getSession(ctx.session.id, ctx.locale));
      } catch {
        mw.end();
        ctx.setPending(null);
      }
    } finally {
      ctx.setChatStreamPreview('');
      ctx.setPending(null);
      ctx.sendingRef.current = false; ctx.setSending(false);
      mw.end();
    }
  };
}

export function createAdvance(ctx: ActionContext) {
  return async () => {
    if (!ctx.session) return;
    if (ctx.sendingRef.current) return;
    const sessionId = ctx.session.id;
    const idx = ctx.stepLabels.findIndex((s) => s.key === ctx.sessionStep);
    const nextStepKey = idx >= 0 ? ctx.stepLabels[idx + 1]?.key : undefined;
    if (nextStepKey) ctx.setOptimisticDisplayStep(nextStepKey);
    const willEnterCode = nextStepKey === 'code';
    const willAsyncAdvance = willEnterCode || (typeof nextStepKey === 'string' && nextStepKey.startsWith('agent:'));
    if (willEnterCode) ctx.setCodeLoading(true);
    const mw = getModelWaitSetters(ctx.modelWaitSetters);
    mw.begin();
    ctx.sendingRef.current = true; ctx.setSending(true);
    ctx.setAdvanceStreamPreview('');
    try {
      if (willAsyncAdvance) {
        const { jobId } = await debateV2Api.startAdvanceJob({ sessionId, locale: ctx.locale });
        await waitAdvanceJob(jobId, { onChunk: ctx.mergeAdvanceStreamChunk });
        ctx.setSession(await debateV2Api.getSession(sessionId, ctx.locale));
      } else {
        ctx.setSession(await debateV2Api.advance({ sessionId, locale: ctx.locale }));
      }
    } catch (e) {
      antMessage.error(friendlyError(e));
      try {
        const synced = await debateV2Api.getSession(sessionId, ctx.locale);
        ctx.setSession(synced);
      } catch (resyncErr) { console.debug('resync failed after advance error', resyncErr); }
    } finally {
      ctx.setAdvanceStreamPreview('');
      ctx.setCodeLoading(false);
      ctx.sendingRef.current = false; ctx.setSending(false);
      mw.end();
      ctx.setOptimisticDisplayStep(null);
    }
  };
}

export function createRejectCode(ctx: ActionContext) {
  return async (feedback: string) => {
    if (!ctx.session) return;
    const sessionId = ctx.session.id;
    const fb = String(feedback || '').trim();
    if (!fb) return;
    ctx.setCodeLoading(true);
    const mw = getModelWaitSetters(ctx.modelWaitSetters);
    mw.begin();
    ctx.sendingRef.current = true; ctx.setSending(true);
    ctx.setAdvanceStreamPreview('');
    try {
      const { jobId } = await debateV2Api.startRejectCodeJob({ sessionId, feedback: fb, locale: ctx.locale });
      await waitAdvanceJob(jobId, { onChunk: ctx.mergeAdvanceStreamChunk });
      ctx.setSession(await debateV2Api.getSession(sessionId, ctx.locale));
    } catch (e) {
      antMessage.error(friendlyError(e));
      try { ctx.setSession(await debateV2Api.getSession(sessionId, ctx.locale)); } catch (resyncErr) {
        console.debug('resync failed after rejectCode error', resyncErr);
      }
    } finally {
      ctx.setAdvanceStreamPreview('');
      ctx.setCodeLoading(false);
      ctx.sendingRef.current = false; ctx.setSending(false);
      mw.end();
    }
  };
}

export function createRetryCodeGen(ctx: ActionContext) {
  return async () => {
    if (!ctx.session?.id || ctx.sendingRef.current) return;
    const sessionId = ctx.session.id;
    ctx.sendingRef.current = true; ctx.setSending(true);
    ctx.setCodeLoading(true);
    const mw = getModelWaitSetters(ctx.modelWaitSetters);
    mw.begin();
    try {
      let s = await debateV2Api.getSession(sessionId, ctx.locale);
      ctx.setSession(s);
      if (s.currentStep !== 'code') {
        antMessage.warning(ctx.t('ai.debate.v2.retryCodeWrongStep', { defaultValue: 'Session is not on the code step. The page was refreshed from the server.' }));
        return;
      }
      const hasBody = Boolean((s.code?.python || '').trim() || (s.code?.text || '').trim());
      if (hasBody) {
        antMessage.info(ctx.t('ai.debate.v2.retryCodeAlreadyHave', { defaultValue: 'Code is already present.' }));
        return;
      }
      await debateV2Api.back({ sessionId, locale: ctx.locale });
      s = await debateV2Api.getSession(sessionId, ctx.locale);
      ctx.setSession(s);
      ctx.setAdvanceStreamPreview('');
      const { jobId } = await debateV2Api.startAdvanceJob({ sessionId, locale: ctx.locale });
      await waitAdvanceJob(jobId, { onChunk: ctx.mergeAdvanceStreamChunk });
      ctx.setSession(await debateV2Api.getSession(sessionId, ctx.locale));
    } catch (e) {
      antMessage.error(friendlyError(e));
      try { ctx.setSession(await debateV2Api.getSession(sessionId, ctx.locale)); } catch (resyncErr) {
        console.debug('resync failed after retryCodeGeneration error', resyncErr);
      }
    } finally {
      ctx.setAdvanceStreamPreview('');
      ctx.setCodeLoading(false);
      ctx.sendingRef.current = false; ctx.setSending(false);
      mw.end();
    }
  };
}

export function createBack(ctx: ActionContext) {
  return async () => {
    if (!ctx.session) return;
    ctx.sendingRef.current = true; ctx.setSending(true);
    try {
      const next = await debateV2Api.back({ sessionId: ctx.session.id, locale: ctx.locale });
      ctx.setSession(next);
    } catch (e) {
      antMessage.error(friendlyError(e));
    } finally {
      ctx.sendingRef.current = false; ctx.setSending(false);
      ctx.setOptimisticDisplayStep(null);
    }
  };
}
