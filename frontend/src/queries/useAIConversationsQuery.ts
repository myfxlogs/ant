import { useRpcQuery } from '@/hooks/useRpcQuery';
import { queryKeys } from './queryKeys';
import { aiApi, type ConversationSummary } from '@/client/ai';
import { toConv } from '@/stores/aiMessageSender';
import type { Conversation } from '@/stores/aiMessageSender';

export function useAIConversationsQuery() {
  return useRpcQuery<Conversation[], Error, ConversationSummary[]>(
    queryKeys.ai.conversations.list(),
    () => aiApi.listConversations(),
    { select: (list) => list.map(toConv) },
  );
}
