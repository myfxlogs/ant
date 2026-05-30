import { useQueryClient } from '@tanstack/react-query';
import { useRpcMutation } from '@/hooks/useRpcMutation';
import { aiApi } from '@/client/ai';
import { queryKeys } from '@/queries/queryKeys';
import { toConv } from '@/stores/aiMessageSender';
import type { Conversation } from '@/stores/aiMessageSender';

export function useCreateConversationMutation() {
  const queryClient = useQueryClient();
  return useRpcMutation<Conversation, Error, string | undefined>(
    (title) => aiApi.createConversation(title),
    {
      onSuccess: (created) => {
        const conv = toConv(created);
        queryClient.setQueryData<Conversation[]>(
          queryKeys.ai.conversations.list(),
          (old = []) => [conv, ...old],
        );
      },
    },
  );
}
