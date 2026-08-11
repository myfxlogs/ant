import { useQueryClient } from '@tanstack/react-query';
import { useRpcMutation } from '@/hooks/useRpcMutation';
import { accountApi } from '@/client/account';
import { queryKeys } from '@/queries/queryKeys';
import type { Account } from '@/types/account';

export function useDeleteAccountMutation() {
  const queryClient = useQueryClient();
  return useRpcMutation(
    ({ id, password }: { id: string; password?: string }) =>
      accountApi.delete(id, password),
    {
      onMutate: async (vars) => {
        await queryClient.cancelQueries({ queryKey: queryKeys.accounts.list() });
        const prevTq = queryClient.getQueryData<Account[]>(queryKeys.accounts.list());
        // Optimistic: remove from TQ cache immediately.
        queryClient.setQueryData<Account[]>(queryKeys.accounts.list(), (old) =>
          (old ?? []).filter((a) => a.id !== vars.id),
        );
        return { prevTq };
      },
      onError: (_err, _vars, context) => {
        const ctx = context as { prevTq?: Account[] } | undefined;
        if (ctx?.prevTq) queryClient.setQueryData(queryKeys.accounts.list(), ctx.prevTq);
      },
      onSettled: () => {
        queryClient.invalidateQueries({ queryKey: queryKeys.accounts.list() });
      },
    },
  );
}
