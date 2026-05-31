import { message } from 'antd';
import { aiApi } from '@/client/ai';
import { toFriendlyAIChatError } from '@/client/ai';
import type { ConversationSummary } from '@/client/ai';
import i18n from '@/i18n';
import { buildChatContext, saveUserPrefs } from './aiContext';

export interface Message {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  timestamp: Date;
  isLoading?: boolean;
}

export interface Conversation {
  id: string;
  title: string;
  createdAt: Date;
  updatedAt: Date;
  messageCount: number;
}

export function toConv(c: ConversationSummary): Conversation {
  return {
    id: c.id,
    title: c.title,
    createdAt: new Date(c.createdAt),
    updatedAt: new Date(c.updatedAt),
    messageCount: c.messageCount,
  };
}

type StatePatch = {
  sending?: boolean;
  conversations?: Conversation[];
  activeConversationId?: string;
  messages?: Message[];
  loading?: boolean;
};

interface StoreAccessors {
  getSending: () => boolean;
  getActiveConversationId: () => string;
  getConversations: () => Conversation[];
  getMessages: () => Message[];
  setState: (updater: StatePatch | ((prev: { messages: Message[]; conversations: Conversation[] }) => StatePatch)) => void;
}

// Track the active stream controller so we can abort when a new send starts
// or the component unmounts.
let activeAbortController: AbortController | null = null;

/**
 * Core message sending logic shared by sendMessage and sendMessageAndGetResponse.
 * Returns the assistant's response text (empty string on failure).
 */
export async function sendMessageCore(
  content: string,
  accountId: string | undefined,
  accessors: StoreAccessors,
): Promise<string> {
  if (accessors.getSending()) return '';

  // Abort any in-flight streaming request so deltas don't land
  // on the wrong conversation.
  if (activeAbortController) {
    activeAbortController.abort();
    activeAbortController = null;
  }
  activeAbortController = new AbortController();
  const signal = activeAbortController.signal;

  accessors.setState({ sending: true });

  let activeConversationId = accessors.getActiveConversationId();
  let convReady = !!activeConversationId;

  const rememberPrefix = i18n.t('ai.store.prefs.rememberPrefix');
  if (content.trim().startsWith(rememberPrefix)) {
    const next = content.trim().slice(rememberPrefix.length).trim();
    saveUserPrefs(next);
    message.success(i18n.t('ai.store.prefs.rememberedToast'));
    accessors.setState({ sending: false });
    return i18n.t('ai.store.prefs.savedReply');
  }

  if (!activeConversationId) {
    try {
      const created = await aiApi.createConversation(i18n.t('ai.store.conversations.newConversationTitle'));
      const conv = toConv(created);
      const cur = accessors.getConversations();
      activeConversationId = conv.id;
      convReady = true;
      accessors.setState({
        conversations: [conv, ...cur],
        activeConversationId: conv.id,
        messages: [],
      });
    } catch (createErr) {
      console.error('sendMessageCore: createConversation failed', createErr);
      convReady = false;
    }
  }

  const { getMessages } = accessors;
  const userMessage: Message = {
    id: `user-${crypto.randomUUID()}`,
    role: 'user',
    content,
    timestamp: new Date(),
  };
  const aiMessageId = `ai-${crypto.randomUUID()}`;
  const aiMessage: Message = {
    id: aiMessageId,
    role: 'assistant',
    content: '',
    timestamp: new Date(),
    isLoading: true,
  };

  // Functional update — reads latest state, no race with streaming deltas.
  accessors.setState((prev) => ({
    messages: [...prev.messages, userMessage, aiMessage],
  }));

  try {
    let response: { message: string; suggestions: string[] };
    try {
      let acc = '';
      response = await aiApi.chatStreaming(
        {
          message: content,
          context: buildChatContext(),
          accountId,
          conversationId: convReady ? activeConversationId : '',
        },
        (delta) => {
          acc += delta;
          // Functional update avoids race with the outer setState.
          accessors.setState((prev) => ({
            messages: prev.messages.map((m) =>
              m.id === aiMessageId ? { ...m, content: acc, isLoading: true } : m,
            ),
          }));
        },
        { signal },
      );
    } catch (streamErr) {
      // Don't fall back if the request was intentionally aborted.
      if (signal.aborted) {
        console.debug('chatStreaming aborted');
        return '';
      }
      console.error('chatStreaming failed, falling back to non-streaming', streamErr);
      response = await aiApi.chat({
        message: content,
        context: buildChatContext(),
        accountId,
        conversationId: convReady ? activeConversationId : '',
      });
    }

    accessors.setState((prev) => ({
      messages: prev.messages.map((m) =>
        m.id === aiMessageId ? { ...m, content: response.message, isLoading: false } : m,
      ),
    }));

    if (convReady) {
      const list = await aiApi.listConversations();
      accessors.setState({ conversations: list.map(toConv) });
    }
    return response.message || '';
  } catch (e: unknown) {
    console.error('sendMessageCore: send failed', e);
    accessors.setState((prev) => ({
      messages: prev.messages.map((m) =>
        m.id === aiMessageId ? { ...m, content: i18n.t('ai.store.messages.sendFailedInline'), isLoading: false } : m,
      ),
    }));
    message.error(toFriendlyAIChatError(e) || i18n.t('ai.store.messages.sendFailedToast'));
    return '';
  } finally {
    if (activeAbortController?.signal === signal) {
      activeAbortController = null;
    }
    accessors.setState({ sending: false });
  }
}
