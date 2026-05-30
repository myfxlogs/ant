import { create } from 'zustand';
import { message } from 'antd';
import { aiApi } from '@/client/ai';
import { toFriendlyAIChatError } from '@/client/ai';
import type { ConversationSummary } from '@/client/ai';
import i18n from '@/i18n';
import { sendMessageCore, toConv, type Message, type Conversation } from './aiMessageSender';

interface AIState {
  conversations: Conversation[];
  activeConversationId: string;
  messages: Message[];
  loading: boolean;
  sending: boolean;
  conversationsLoaded: boolean;

  loadConversations: () => Promise<void>;
  sendMessage: (_content: string, _accountId?: string) => Promise<void>;
  sendMessageAndGetResponse: (_content: string, _accountId?: string) => Promise<string>;
  clearMessages: () => void;
  newConversation: () => Promise<void>;
  selectConversation: (_id: string) => Promise<void>;
  deleteConversation: (_id: string) => Promise<void>;
  getReports: (_accountId?: string) => Promise<any[]>;
  generateReport: (_accountId: string, _reportType: string, _period: string) => Promise<any>;
  setLoading: (loading: boolean) => void;
}

export const useAIStore = create<AIState>((set, get) => {
  function buildAccessors() {
    return {
      getSending: () => get().sending,
      getActiveConversationId: () => get().activeConversationId,
      getConversations: () => get().conversations,
      getMessages: () => get().messages,
      setState: (partial: {
        sending?: boolean;
        conversations?: Conversation[];
        activeConversationId?: string;
        messages?: Message[];
        loading?: boolean;
      }) => set(partial),
    };
  }

  return {
    conversations: [],
    activeConversationId: '',
    messages: [],
    loading: false,
    sending: false,
    conversationsLoaded: false,

    loadConversations: async () => {
      try {
        const list = await aiApi.listConversations();
        const convs = list.map(toConv);
        set({ conversations: convs, conversationsLoaded: true });
      } catch (err) {
        console.debug('loadConversations failed', err);
        set({ conversationsLoaded: true });
      }
    },

    sendMessageAndGetResponse: async (content, accountId) => {
      return sendMessageCore(content, accountId, buildAccessors());
    },

    sendMessage: async (content, accountId) => {
      await sendMessageCore(content, accountId, buildAccessors());
    },

    newConversation: async () => {
      try {
        const created = await aiApi.createConversation(i18n.t('ai.store.conversations.newConversationTitle'));
        const conv = toConv(created);
        const cur = get().conversations;
        set({
          conversations: [conv, ...cur],
          activeConversationId: conv.id,
          messages: [],
        });
      } catch {
        message.error(i18n.t('ai.store.messages.createConversationFailed'));
      }
    },

    selectConversation: async (id: string) => {
      set({ activeConversationId: id, messages: [], loading: true });
      try {
        const detail = await aiApi.getConversation(id);
        const msgs: Message[] = detail.messages.map((m) => ({
          id: m.id,
          role: m.role as 'user' | 'assistant',
          content: m.content,
          timestamp: new Date(m.createdAt),
        }));
        set({ messages: msgs, loading: false });
      } catch {
        set({ loading: false });
        message.error(i18n.t('ai.store.messages.loadConversationFailed'));
      }
    },

    deleteConversation: async (id: string) => {
      try {
        await aiApi.deleteConversation(id);
        const cur = get().conversations.filter((c) => c.id !== id);
        const activeId = get().activeConversationId;
        if (activeId === id) {
          set({
            conversations: cur,
            activeConversationId: cur[0]?.id || '',
            messages: [],
          });
          if (cur[0]) {
            get().selectConversation(cur[0].id);
          }
        } else {
          set({ conversations: cur });
        }
      } catch {
        message.error(i18n.t('ai.store.messages.deleteConversationFailed'));
      }
    },

    clearMessages: () => {
      set({ messages: [] });
      message.success(i18n.t('ai.store.messages.clearedLocalOnly'));
    },

    getReports: async (accountId) => {
      try {
        const reports = await aiApi.getReports({ accountId });
        return reports;
      } catch {
        message.error(i18n.t('ai.store.messages.getReportsFailed'));
        return [];
      }
    },

    generateReport: async (accountId, reportType, period) => {
      try {
        const report = await aiApi.generateReport({
          accountId,
          reportType,
          period,
        });
        message.success(i18n.t('ai.store.messages.generateReportSuccess'));
        return report;
      } catch {
        message.error(i18n.t('ai.store.messages.generateReportFailed'));
        return null;
      }
    },

    setLoading: (loading) => set({ loading }),
  };
});
