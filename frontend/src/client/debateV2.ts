import type { DebateV2Session as DebateV2SessionMessage } from '../gen/debate_v2_messages_pb';
import type { TemplateParameter } from '../gen/strategy_messages_pb';
import { debateV2Client } from './connect';
import { apiBaseUrl } from './transport';

// Client for the redesigned multi-expert debate flow (v2).
// Backend: DebateV2Service ConnectRPC and backend/internal/service/debate_v2_service.go.

export type V2StepKey = 'intent' | 'code' | 'done' | `agent:${string}`;

export interface V2Message {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  kind?: 'kickoff' | string;
}

export interface V2Step {
  stepKey: V2StepKey;
  agentKey?: string;
  agentName?: string;
  messages: V2Message[];
}

export interface V2Code {
  text: string;
  python: string;
}

export interface V2Usage {
  promptTokens: number;
  completionTokens: number;
  totalTokens: number;
}

export interface V2TemplateParameter {
  name: string;
  type: string;
  default?: string;
  min?: string;
  max?: string;
  step?: string;
  label?: string;
  description?: string;
  options?: string[];
}

export interface V2Session {
  id: string;
  title: string;
  status: string;
  currentStep: V2StepKey;
  agents: string[];
  steps: V2Step[];
  paramSchema?: V2TemplateParameter[];
  code?: V2Code | null;
  // Provider + model that the next chat in the current step would use.
  provider?: string;
  model?: string;
  // Cumulative token usage across the whole session.
  usage?: V2Usage;
  createdAt: string;
  updatedAt: string;
}

function toParam(p: TemplateParameter): V2TemplateParameter {
  return {
    name: p.name,
    type: p.type,
    default: p.default || undefined,
    min: p.min || undefined,
    max: p.max || undefined,
    step: p.step || undefined,
    label: p.label || undefined,
    description: p.description || undefined,
    options: p.options || undefined,
  };
}

function fromParam(p: V2TemplateParameter): Partial<TemplateParameter> {
  return {
    name: p.name,
    type: p.type,
    default: p.default || '',
    min: p.min || '',
    max: p.max || '',
    step: p.step || '',
    label: p.label || '',
    description: p.description || '',
    options: p.options || [],
  };
}

function toSession(s: DebateV2SessionMessage): V2Session {
  return {
    id: s.id,
    title: s.title,
    status: s.status,
    currentStep: s.currentStep as V2StepKey,
    agents: s.agents || [],
    steps: (s.steps || []).map((step) => ({
      stepKey: step.stepKey as V2StepKey,
      agentKey: step.agentKey || undefined,
      agentName: step.agentName || undefined,
      messages: (step.messages || []).map((m) => ({
        id: m.id,
        role: m.role === 'user' ? 'user' : 'assistant',
        content: m.content,
        kind: m.kind || undefined,
      })),
    })),
    paramSchema: (s.paramSchema || []).map(toParam),
    code: s.code ? { text: s.code.text, python: s.code.python } : null,
    provider: s.provider || undefined,
    model: s.model || undefined,
    usage: s.usage ? {
      promptTokens: s.usage.promptTokens,
      completionTokens: s.usage.completionTokens,
      totalTokens: s.usage.totalTokens,
    } : undefined,
    createdAt: s.createdAt,
    updatedAt: s.updatedAt,
  };
}

export const debateV2Api = {
  start: async (input: { agents: string[]; title?: string; locale?: string }): Promise<V2Session> => {
    const data = await debateV2Client.startDebateV2({ agents: input.agents || [], title: input.title || '', locale: input.locale || '' });
    return toSession(data);
  },

  chat: async (input: { sessionId: string; message: string; locale?: string }): Promise<V2Session> => {
    const data = await debateV2Client.chatDebateV2({ sessionId: input.sessionId, message: input.message, locale: input.locale || '' });
    return toSession(data);
  },

  advance: async (input: { sessionId: string; locale?: string }): Promise<V2Session> => {
    const data = await debateV2Client.advanceDebateV2({ sessionId: input.sessionId, locale: input.locale || '' });
    return toSession(data);
  },

  /** Unary submit when the next step needs LLM (agent kickoff or code); pair with waitAdvanceJob. */
  startAdvanceJob: async (input: { sessionId: string; locale?: string }): Promise<{ jobId: string; sessionId: string }> => {
    const data = await debateV2Client.startDebateV2AdvanceJob({ sessionId: input.sessionId, locale: input.locale || '' });
    return { jobId: data.jobId, sessionId: data.sessionId };
  },

  getAdvanceJob: async (jobId: string): Promise<{ phase: string; message: string; sessionId: string }> => {
    const data = await debateV2Client.getDebateV2AdvanceJob({ jobId });
    return { phase: data.phase, message: data.message || '', sessionId: data.sessionId };
  },

  /** Phase 1: persist user message + job_id (no LLM). Then open SSE (`waitChatJob`), then `runChatJob`. */
  prepareChatJob: async (input: { sessionId: string; message: string; locale?: string }): Promise<{ jobId: string; sessionId: string }> => {
    const data = await debateV2Client.prepareDebateV2ChatJob({
      sessionId: input.sessionId,
      message: input.message,
      locale: input.locale || '',
    });
    return { jobId: data.jobId, sessionId: data.sessionId };
  },

  /** Phase 2: start streaming LLM after EventSource is subscribed. */
  runChatJob: async (input: { jobId: string }): Promise<void> => {
    await debateV2Client.runDebateV2ChatJob({ jobId: input.jobId });
  },

  getChatJob: async (jobId: string): Promise<{ phase: string; message: string; sessionId: string }> => {
    const data = await debateV2Client.getDebateV2ChatJob({ jobId });
    return { phase: data.phase, message: data.message || '', sessionId: data.sessionId };
  },

  back: async (input: { sessionId: string; locale?: string }): Promise<V2Session> => {
    const data = await debateV2Client.backDebateV2({ sessionId: input.sessionId, locale: input.locale || '' });
    return toSession(data);
  },

  setParams: async (input: { sessionId: string; params: V2TemplateParameter[]; locale?: string }): Promise<V2Session> => {
    const data = await debateV2Client.setDebateV2Params({ sessionId: input.sessionId, params: (input.params || []).map(fromParam), locale: input.locale || '' });
    return toSession(data);
  },

  listSessions: async (input: { locale?: string } = {}): Promise<V2Session[]> => {
    const data = await debateV2Client.listDebateV2Sessions({ locale: input.locale || '', limit: 50 });
    return (data.sessions || []).map(toSession);
  },

  getSession: async (id: string, locale?: string): Promise<V2Session> => {
    const data = await debateV2Client.getDebateV2Session({ sessionId: id, locale: locale || '' });
    return toSession(data);
  },

  deleteSession: async (id: string): Promise<void> => {
    await debateV2Client.deleteDebateV2Session({ sessionId: id });
  },

  rejectCode: async (input: { sessionId: string; feedback: string; locale?: string }): Promise<V2Session> => {
    const data = await debateV2Client.rejectDebateV2Code({ sessionId: input.sessionId, feedback: input.feedback, locale: input.locale || '' });
    return toSession(data);
  },

  /** Reject + regenerate strategy; same SSE as advance (`waitAdvanceJob` + advance-jobs stream). */
  startRejectCodeJob: async (input: { sessionId: string; feedback: string; locale?: string }): Promise<{ jobId: string; sessionId: string }> => {
    const data = await debateV2Client.startDebateV2RejectCodeJob({
      sessionId: input.sessionId,
      feedback: input.feedback,
      locale: input.locale || '',
    });
    return { jobId: data.jobId, sessionId: data.sessionId };
  },
};

/**
 * Waits for background advance/chat job: **SSE only** for chunks and completion (event-driven).
 * On transport failure: exponential backoff **reconnect** to the same SSE URL; no periodic Unary polling.
 * After repeated failures or overall deadline: **one** `Get*Job` call to read terminal phase only.
 */
export type WaitAdvanceJobOptions = {
	/** Fired for each LLM delta (`event: chunk`), QuantDinger-style incremental UI. */
	onChunk?: (delta: string) => void;
};

type JobPollRow = { phase: string; message: string };

/** Wait for debate v2 job: chunks and completion are **only** from SSE (event-driven). No periodic Unary polling. */
async function waitDebateV2Job(
	jobId: string,
	streamSubpath: 'advance-jobs' | 'chat-jobs',
	poll: (id: string) => Promise<JobPollRow>,
	failedDefault: string,
	opts?: WaitAdvanceJobOptions,
): Promise<void> {
	const token = typeof localStorage !== 'undefined' ? localStorage.getItem('access_token') : null;
	if (!token) {
		return Promise.reject(new Error('missing access_token'));
	}
	const url = `${apiBaseUrl}/antrader/sse/debate-v2/${streamSubpath}/${encodeURIComponent(jobId)}/stream?access_token=${encodeURIComponent(token)}`;

	const deadlineMs = 15 * 60_000;
	const deadlineAt = Date.now() + deadlineMs;
	const maxReconnectBeforeReconcile = 14;

	return new Promise((resolve, reject) => {
		let settled = false;
		let es: EventSource | null = null;
		let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
		let reconnectAttempt = 0;
		let sseErrorStreak = 0;

		const clearReconnect = () => {
			if (reconnectTimer != null) {
				clearTimeout(reconnectTimer);
				reconnectTimer = null;
			}
		};

		const finish = (fn: () => void) => {
			if (settled) return;
			settled = true;
			clearReconnect();
			es?.close();
			es = null;
			fn();
		};

		/** Single Unary read to align terminal state after SSE path is exhausted (not a poll loop). */
		const reconcileOnce = async () => {
			try {
				const st = await poll(jobId);
				if (st.phase === 'completed') finish(() => resolve());
				else if (st.phase === 'failed') finish(() => reject(new Error(st.message || failedDefault)));
				else finish(() => reject(new Error('debate job stream unavailable — check SSE path and proxy buffering')));
			} catch (e) {
				finish(() => reject(e instanceof Error ? e : new Error(String(e))));
			}
		};

		const scheduleReconnect = () => {
			clearReconnect();
			if (settled) return;
			if (Date.now() >= deadlineAt) {
				void reconcileOnce();
				return;
			}
			if (sseErrorStreak >= maxReconnectBeforeReconcile) {
				sseErrorStreak = 0;
				void reconcileOnce();
				return;
			}
			const delayMs = Math.min(500 * 1.55 ** reconnectAttempt, 30_000);
			reconnectAttempt += 1;
			reconnectTimer = setTimeout(open, delayMs);
		};

		const open = () => {
			clearReconnect();
			if (settled) return;
			if (Date.now() >= deadlineAt) {
				void reconcileOnce();
				return;
			}
			es?.close();
			es = new EventSource(url);

			es.addEventListener('open', () => {
				reconnectAttempt = 0;
				sseErrorStreak = 0;
			});

			es.onmessage = (ev) => {
				try {
					const o = JSON.parse(ev.data) as { event?: string; message?: string; content?: string };
					if (o.event === 'chunk' && typeof o.content === 'string' && o.content && opts?.onChunk) {
						opts.onChunk(o.content);
						return;
					}
					if (o.event === 'completed') {
						finish(() => resolve());
					} else if (o.event === 'failed') {
						finish(() => reject(new Error(o.message || failedDefault)));
					}
				} catch {
					/* ignore malformed chunk */
				}
			};

			es.onerror = () => {
				if (settled) return;
				sseErrorStreak += 1;
				es?.close();
				es = null;
				scheduleReconnect();
			};
		};

		open();
	});
}

export function waitAdvanceJob(jobId: string, opts?: WaitAdvanceJobOptions): Promise<void> {
	return waitDebateV2Job(jobId, 'advance-jobs', (id) => debateV2Api.getAdvanceJob(id), 'code generation failed', opts);
}

/** Intent / agent chat async job + SSE (same event shape as advance jobs). */
export function waitChatJob(jobId: string, opts?: WaitAdvanceJobOptions): Promise<void> {
	return waitDebateV2Job(jobId, 'chat-jobs', (id) => debateV2Api.getChatJob(id), 'chat failed', opts);
}
