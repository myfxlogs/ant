import { aiClient, aiPrimaryClient } from '../connect';
import {
  toAgentView,
  toConversationRole,
  mapConversationSummary,
  protoDate,
} from './types';
import type {
  ChatResult,
  ConversationSummary,
  ConversationDetail,
  AIAgentDefinitionView,
} from './types';
import type { ConversationMessage as ProtoConversationMessage } from '../../gen/ant/v1/ai_conversation_pb';

export const aiApi = {
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
};
