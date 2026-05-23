import { aiClient, aiPrimaryClient } from './connect';
import i18n from '@/i18n';
import { timestampDate } from '@bufbuild/protobuf/wkt';
import type { Timestamp } from '@bufbuild/protobuf/wkt';
import { create } from '@bufbuild/protobuf';
import { AIAgentDefinitionSchema, type AIAgentDefinition } from '../gen/ai_agent_pb';
import type { ConversationMessage as ProtoConversationMessage, ConversationSummary as ProtoConversationSummary } from '../gen/ai_conversation_pb';
import type { WorkflowRunSummary as ProtoWorkflowRunSummary, WorkflowStep as ProtoWorkflowStep } from '../gen/ai_workflow_entity_pb';

export type { AIReport } from '../gen/ai_pb';

function protoDate(ts: Timestamp | undefined): Date {
  return ts ? timestampDate(ts) : new Date();
}

function toConversationRole(role: string): 'user' | 'assistant' | 'system' {
  if (role === 'user' || role === 'assistant' || role === 'system') return role;
  return 'user';
}

// Shared mapper that converts a raw provider/gateway error message into a
// localized, human-friendly hint. Covers the most common failure modes we see
// across OpenAI-family, Anthropic, and Chinese cloud providers (DeepSeek,
// Zhipu, Qwen, Doubao, Moonshot, ...).
//
// Intentionally pattern-matches on substrings rather than HTTP status alone
// because most providers wrap their JSON inside a transport error and we only
// see the pre-formatted body string by the time it gets here.
// 后端常把 provider 原始 JSON 直接拼在 "API request failed with status N: {...}"
// 后面塞回前端，下面这个小 helper 把里头的 provider message 抠出来，让模式
// 匹配工作在「真消息」上而不是外层包装。失败/无 JSON 时原样返回。
function unwrapProviderMessage(raw: string): string {
  const start = raw.indexOf('{');
  if (start < 0) return raw;
  const body = raw.slice(start);
  try {
    const obj = JSON.parse(body) as { error?: { message?: unknown }; message?: unknown; error?: unknown };
    const inner = obj?.error?.message ?? obj?.message ?? obj?.error ?? '';
    const innerStr = typeof inner === 'string' ? inner : typeof inner === 'object' ? JSON.stringify(inner) : '';
    return innerStr.trim() || raw;
  } catch {
    return raw;
  }
}

function pickErrorText(raw: unknown): string {
  if (raw == null) return '';
  if (typeof raw === 'string') return raw;
  if (typeof raw === 'number' || typeof raw === 'boolean') return String(raw);
  if (raw instanceof Error) return raw.message;
  if (typeof raw === 'object') {
    const o = raw as { rawMessage?: unknown; message?: unknown };
    if (typeof o.rawMessage === 'string' && o.rawMessage.trim()) return o.rawMessage;
    if (typeof o.message === 'string' && o.message.trim()) return o.message;
  }
  return String(raw);
}

export function toFriendlyAIError(raw: unknown): string {
  const rawMsg = pickErrorText(raw).trim();
  if (!rawMsg) return i18n.t('ai.client.errors.requestFailed');
  const msg = unwrapProviderMessage(rawMsg);
  const lower = msg.toLowerCase();

  if (
    lower.includes('insufficient_quota') ||
    lower.includes('insufficient quota') ||
    lower.includes('insufficient_balance') ||
    lower.includes('insufficient balance') ||
    lower.includes('insufficient credits') ||
    lower.includes('never purchased credits') ||
    lower.includes('purchase more at') ||
    lower.includes('status 402') ||
    lower.includes(' 402') ||
    lower.includes('credit_balance_too_low') ||
    lower.includes('exceeded your current quota') ||
    lower.includes('exceeded your quota') ||
    lower.includes('billing_not_active') ||
    lower.includes('arrearage') ||
    lower.includes('overdue-payment') ||
    lower.includes('overdue payment') ||
    msg.includes('余额不足') ||
    msg.includes('額度不足') ||
    msg.includes('账户欠费') ||
    msg.includes('帳號欠費') ||
    msg.includes('试用已结束') ||
    msg.includes('试用额度') ||
    msg.includes('試用已結束') ||
    lower.includes('product is not activated') ||
    lower.includes('product not activated') ||
    lower.includes('not activated, please confirm') ||
    (lower.includes('please activate') && lower.includes('product')) ||
    msg.includes('未开通') ||
    msg.includes('未激活') ||
    msg.includes('未開通') ||
    msg.includes('尚未开通') ||
    msg.includes('请先开通')
  ) {
    return i18n.t('ai.client.errors.insufficientBalance');
  }

  if (
    lower.includes('status 429') ||
    lower.includes(' 429') ||
    lower.includes('too many requests') ||
    lower.includes('rate_limit') ||
    lower.includes('rate limit') ||
    lower.includes('tpm limit') ||
    lower.includes('rpm limit') ||
    msg.includes('请求过于频繁') ||
    msg.includes('限流')
  ) {
    return i18n.t('ai.client.errors.rateLimited');
  }

  if (
    lower.includes('invalid_api_key') ||
    lower.includes('invalid api key') ||
    lower.includes('unauthorized') ||
    lower.includes(' 401') ||
    lower.includes('status 401') ||
    msg.includes('密钥无效') ||
    msg.includes('鉴权失败')
  ) {
    return i18n.t('ai.client.errors.unauthorized');
  }
  if (lower.includes('forbidden') || lower.includes(' 403') || lower.includes('status 403')) {
    return i18n.t('ai.client.errors.forbidden');
  }

  if (
    lower.includes('model_not_found') ||
    lower.includes('model not found') ||
    lower.includes('model does not exist') ||
    lower.includes('invalid model id') ||
    (lower.includes('the model `') && lower.includes('does not exist')) ||
    lower.includes('model_deprecated') ||
    lower.includes('model deprecated') ||
    msg.includes('模型不存在') ||
    msg.includes('模型已下线') ||
    msg.includes('模型已停用')
  ) {
    const m = msg.match(/(?:Invalid model id|model `?)([\w./:-]+)/i);
    const model = m?.[1] ? `（${m[1]}）` : '';
    return i18n.t('ai.client.errors.invalidModelId', { model });
  }

  if (
    lower.includes('context_length_exceeded') ||
    lower.includes('maximum context length') ||
    (lower.includes('context length') && lower.includes('exceed')) ||
    lower.includes('request too large') ||
    lower.includes('payload too large') ||
    msg.includes('上下文超长') ||
    msg.includes('内容过长')
  ) {
    return i18n.t('ai.client.errors.contextTooLong');
  }

  if (
    lower.includes('content_filter') ||
    lower.includes('content policy') ||
    lower.includes('safety_block') ||
    (lower.includes('blocked') && lower.includes('safety')) ||
    msg.includes('内容审核') ||
    msg.includes('内容违规') ||
    msg.includes('敏感内容')
  ) {
    return i18n.t('ai.client.errors.contentBlocked');
  }

  if (
    lower.includes('not supported in your region') ||
    lower.includes('country, region') ||
    lower.includes('unsupported_country_region_territory')
  ) {
    return i18n.t('ai.client.errors.regionNotSupported');
  }

  if (/\b524\b/.test(lower) || /\b523\b/.test(lower) || /\b522\b/.test(lower) || /\b521\b/.test(lower) || /\b520\b/.test(lower)) {
    return i18n.t('ai.client.errors.edgeGatewayTimeout');
  }

  if (
    lower.includes('status 5') ||
    lower.includes(' 500') ||
    lower.includes(' 502') ||
    lower.includes(' 503') ||
    lower.includes(' 504') ||
    lower.includes('overloaded') ||
    lower.includes('service unavailable') ||
    lower.includes('internal server error')
  ) {
    return i18n.t('ai.client.errors.providerInternalError');
  }

  if (
    lower.includes('context deadline exceeded') ||
    lower.includes('client.timeout exceeded') ||
    lower.includes('timeout exceeded while awaiting headers') ||
    (lower.includes('failed to send request') && lower.includes('chat/completions')) ||
    lower.includes('i/o timeout') ||
    lower.includes('timeout') ||
    lower.includes('connection refused') ||
    lower.includes('no such host') ||
    lower.includes('dial tcp') ||
    lower.includes('econnrefused') ||
    lower.includes('etimedout')
  ) {
    return i18n.t('ai.client.errors.networkUnreachable');
  }

  return msg;
}

export const toFriendlyAIChatError = toFriendlyAIError;

export interface ChatResult {
  message: string;
  suggestions: string[];
}

export interface ConversationSummary {
  id: string;
  title: string;
  messageCount: number;
  createdAt: Date;
  updatedAt: Date;
}

export interface ConversationMessage {
  id: string;
  role: 'user' | 'assistant' | 'system';
  content: string;
  createdAt: Date;
}

export interface ConversationDetail extends ConversationSummary {
  messages: ConversationMessage[];
}

export interface WorkflowRunSummary {
  id: string;
  title: string;
  status: string;
  createdAt: Date;
  updatedAt: Date;
  stepCount: number;
}

export interface WorkflowStep {
  id: string;
  runId: string;
  key: string;
  title: string;
  status: string;
  input: string;
  output: string;
  error: string;
  durationMs: number;
  createdAt: Date;
}

export interface WorkflowRunDetail {
  run: WorkflowRunSummary;
  steps: WorkflowStep[];
  contextJson: string;
}

export interface AIAgentDefinitionView {
  id: string;
  agentKey: string;
  type: string;
  name: string;
  identity: string;
  inputHint: string;
  enabled: boolean;
  position: number;
  providerId: string;
  modelOverride: string;
}

function toAgentView(a: AIAgentDefinition): AIAgentDefinitionView {
  return {
    id: a.id || '',
    agentKey: a.agentKey || '',
    type: a.type || '',
    name: a.name || '',
    identity: a.identity || '',
    inputHint: a.inputHint || '',
    enabled: !!a.enabled,
    position: typeof a.position === 'number' ? a.position : 0,
    providerId: a.providerId || '',
    modelOverride: a.modelOverride || '',
  };
}

function mapConversationSummary(c: ProtoConversationSummary): ConversationSummary {
  return {
    id: c.id,
    title: c.title || 'Untitled',
    messageCount: c.messageCount || 0,
    createdAt: protoDate(c.createdAt),
    updatedAt: protoDate(c.updatedAt),
  };
}

function mapWorkflowRunSummary(r: ProtoWorkflowRunSummary): WorkflowRunSummary {
  return {
    id: r.id,
    title: r.title || '',
    status: r.status || '',
    createdAt: protoDate(r.createdAt),
    updatedAt: protoDate(r.updatedAt),
    stepCount: r.stepCount || 0,
  };
}

function mapWorkflowStep(s: ProtoWorkflowStep): WorkflowStep {
  return {
    id: s.id,
    runId: s.runId,
    key: s.key,
    title: s.title || '',
    status: s.status || '',
    input: s.input || '',
    output: s.output || '',
    error: s.error || '',
    durationMs: Number(s.durationMs || 0n),
    createdAt: protoDate(s.createdAt),
  };
}

function viewToAgentDefinition(a: AIAgentDefinitionView): AIAgentDefinition {
  return create(AIAgentDefinitionSchema, {
    id: a.id,
    agentKey: a.agentKey,
    type: a.type,
    name: a.name,
    identity: a.identity,
    inputHint: a.inputHint,
    enabled: a.enabled,
    position: a.position,
    providerId: a.providerId || '',
    modelOverride: a.modelOverride || '',
  });
}

export const aiApi = {
  getReports: async (params?: { accountId?: string; limit?: number }) => {
    const response = await aiClient.getAIReports({
      accountId: params?.accountId || '',
      limit: params?.limit || 10,
    });
    return response.reports;
  },

  generateReport: async (params: { accountId: string; reportType: string; period: string }) => {
    const response = await aiClient.generateReport({
      accountId: params.accountId,
      reportType: params.reportType,
      period: params.period,
    });
    return response.report;
  },

  chat: async (params: {
    message: string;
    context?: string;
    accountId?: string;
    conversationId?: string;
  }): Promise<ChatResult> => {
    const response = await aiClient.chat({
      message: params.message,
      context: params.context || '',
      accountId: params.accountId || '',
      conversationId: params.conversationId || '',
    });
    return {
      message: response.message,
      suggestions: response.suggestions || [],
    };
  },

  chatStreaming: async (
    params: {
      message: string;
      context?: string;
      accountId?: string;
      conversationId?: string;
    },
    onDelta: (delta: string) => void,
    opts?: { signal?: AbortSignal },
  ): Promise<ChatResult> => {
    const req = {
      message: params.message,
      context: params.context || '',
      accountId: params.accountId || '',
      conversationId: params.conversationId || '',
    };
    const stream = await aiClient.chatStream(req, { signal: opts?.signal });
    let full = '';
    for await (const chunk of stream) {
      if (chunk.delta) {
        full += chunk.delta;
        onDelta(chunk.delta);
      }
      if (chunk.errorMessage) {
        throw new Error(chunk.errorMessage);
      }
      if (chunk.done) {
        break;
      }
    }
    return { message: full, suggestions: [] };
  },

  listAgents: async (): Promise<AIAgentDefinitionView[]> => {
    const response = await aiClient.listAgents({});
    return (response.agents || []).map(toAgentView);
  },

  setAgents: async (agents: AIAgentDefinitionView[]): Promise<AIAgentDefinitionView[]> => {
    const response = await aiClient.setAgents({
      agents: agents.map(viewToAgentDefinition),
    });
    return (response.agents || []).map(toAgentView);
  },

  getPrimary: async (): Promise<{ providerId: string; model: string }> => {
    const r = await aiPrimaryClient.getAIPrimary({});
    return { providerId: r.providerId || '', model: r.model || '' };
  },

  setPrimary: async (input: { providerId: string; model: string }): Promise<{ providerId: string; model: string }> => {
    const r = await aiPrimaryClient.setAIPrimary({
      providerId: input.providerId || '',
      model: input.model || '',
    });
    return { providerId: r.providerId || '', model: r.model || '' };
  },

  listConversations: async (): Promise<ConversationSummary[]> => {
    const response = await aiClient.listConversations({});
    return (response.conversations || []).map(mapConversationSummary);
  },

  getConversation: async (id: string): Promise<ConversationDetail> => {
    const response = await aiClient.getConversation({ id });
    const conv = response.conversation;
    const base = conv
      ? mapConversationSummary(conv)
      : {
          id,
          title: '',
          messageCount: 0,
          createdAt: new Date(),
          updatedAt: new Date(),
        };
    return {
      ...base,
      id: conv?.id || id,
      messages: (response.messages || []).map((m: ProtoConversationMessage) => ({
        id: m.id,
        role: toConversationRole(m.role),
        content: m.content,
        createdAt: protoDate(m.createdAt),
      })),
    };
  },

  createConversation: async (title?: string): Promise<ConversationSummary> => {
    const response = await aiClient.createConversation({
      title: title || 'New conversation',
    });
    const c = response.conversation;
    if (!c) {
      throw new Error('createConversation: empty response');
    }
    return mapConversationSummary(c);
  },

  deleteConversation: async (id: string): Promise<boolean> => {
    const response = await aiClient.deleteConversation({ id });
    return !!response.success;
  },

  updateConversationTitle: async (id: string, title: string): Promise<boolean> => {
    const response = await aiClient.updateConversationTitle({ id, title });
    return !!response.success;
  },

  createWorkflowRun: async (params: { title?: string; contextJson?: string }): Promise<WorkflowRunSummary> => {
    const response = await aiClient.createWorkflowRun({
      title: params.title || 'New workflow run',
      contextJson: params.contextJson || '',
    });
    const r = response.run;
    if (!r) {
      throw new Error('createWorkflowRun: empty run');
    }
    return mapWorkflowRunSummary(r);
  },

  appendWorkflowStep: async (params: {
    runId: string;
    key: string;
    title: string;
    status: string;
    input?: string;
    output?: string;
    error?: string;
    durationMs?: number;
  }): Promise<WorkflowStep> => {
    const response = await aiClient.appendWorkflowStep({
      runId: params.runId,
      key: params.key,
      title: params.title,
      status: params.status,
      input: params.input || '',
      output: params.output || '',
      error: params.error || '',
      durationMs: BigInt(params.durationMs ?? 0),
    });
    const s = response.step;
    if (!s) {
      throw new Error('appendWorkflowStep: empty step');
    }
    return mapWorkflowStep(s);
  },

  listWorkflowRuns: async (params?: { limit?: number; offset?: number }): Promise<WorkflowRunSummary[]> => {
    const response = await aiClient.listWorkflowRuns({
      limit: params?.limit || 20,
      offset: params?.offset || 0,
    });
    return (response.runs || []).map(mapWorkflowRunSummary);
  },

  getWorkflowRun: async (id: string): Promise<WorkflowRunDetail> => {
    const response = await aiClient.getWorkflowRun({ id });
    const r = response.run;
    return {
      run: r
        ? mapWorkflowRunSummary(r)
        : {
            id,
            title: '',
            status: '',
            createdAt: new Date(),
            updatedAt: new Date(),
            stepCount: 0,
          },
      steps: (response.steps || []).map(mapWorkflowStep),
      contextJson: response.contextJson || '',
    };
  },
};
