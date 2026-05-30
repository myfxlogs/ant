import { useQueryClient } from '@tanstack/react-query';
import { useRpcMutation } from '@/hooks/useRpcMutation';
import { aiApi } from '@/client/ai';
import { queryKeys } from '@/queries/queryKeys';
import type { Conversation } from '@/stores/aiMessageSender';

export function useDeleteConversationMutation() {
  const queryClient = useQueryClient();
  return useRpcMutation<void, Error, string>(
    (id) => aiApi.deleteConversation(id),
    {
      onSuccess: (_data, id) => {
        queryClient.setQueryData<Conversation[]>(
          queryKeys.ai.conversations.list(),
          (old = []) => old.filter((c) => c.id !== id),
        );
        queryClient.removeQueries({
          queryKey: queryKeys.ai.conversations.detail(id),
        });
      },
    },
  );
}
