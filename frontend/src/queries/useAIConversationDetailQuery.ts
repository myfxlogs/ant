import { useRpcQuery } from '@/hooks/useRpcQuery';
import { queryKeys } from './queryKeys';
import { aiApi } from '@/client/ai';
import type { Message } from '@/stores/aiMessageSender';

export function useAIConversationDetailQuery(id: string) {
  return useRpcQuery<Message[]>(
    queryKeys.ai.conversations.detail(id),
    async () => {
      const detail = await aiApi.getConversation(id);
      return detail.messages.map((m) => ({
        id: m.id,
        role: m.role as 'user' | 'assistant',
        content: m.content,
        timestamp: new Date(m.createdAt),
      }));
    },
    { enabled: !!id },
  );
}
