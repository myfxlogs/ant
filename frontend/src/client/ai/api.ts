import { timestampDate } from '@bufbuild/protobuf/wkt';
import type { Timestamp } from '@bufbuild/protobuf/wkt';
import { aiClient, aiPrimaryClient } from '../connect';
import {
  toAgentView,
  viewToAgentDefinition,
  mapConversationSummary,
  mapWorkflowRunSummary,
  mapWorkflowStep,
} from './types';
import type {
  ChatResult,
  ConversationSummary,
  ConversationDetail,
  WorkflowRunSummary,
  WorkflowStep,
  WorkflowRunDetail,
  AIAgentDefinitionView,
} from './types';
import type { ConversationMessage as ProtoConversationMessage } from '../../gen/ant/v1/ai_conversation_pb';

function protoTs(ts: Timestamp | undefined): Date {
  return ts ? timestampDate(ts) : new Date();
}

function toConversationRole(role: string): 'user' | 'assistant' | 'system' {
  if (role === 'user' || role === 'assistant' || role === 'system') return role;
  return 'user';
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
        createdAt: protoTs(m.createdAt),
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
