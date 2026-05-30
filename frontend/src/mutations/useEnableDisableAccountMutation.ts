import { useQueryClient } from '@tanstack/react-query';
import { useRpcMutation } from '@/hooks/useRpcMutation';
import { accountApi } from '@/client/account';
import { queryKeys } from '@/queries/queryKeys';
import type { Account } from '@/types/account';

interface ToggleVars {
  id: string;
  isDisabled: boolean;
}

/**
 * Optimistic update for enable/disable toggle.
 * Updates Query cache immediately, rolls back on error.
 */
export function useEnableDisableAccountMutation() {
  const queryClient = useQueryClient();

  return useRpcMutation<Account, Error, ToggleVars>(
    ({ id, isDisabled }) => accountApi.update({ id, isDisabled }),
    {
      onMutate: async ({ id, isDisabled }) => {
        await queryClient.cancelQueries({ queryKey: queryKeys.accounts.list() });
        const previous = queryClient.getQueryData<Account[]>(
          queryKeys.accounts.list(),
        );
        queryClient.setQueryData<Account[]>(
          queryKeys.accounts.list(),
          (old = []) =>
            old.map((a) =>
              a.id === id
                ? { ...a, isDisabled, status: isDisabled ? 'disconnected' : a.status }
                : a,
            ),
        );
        return { previous };
      },
      onError: (_err, _vars, ctx) => {
        if (ctx?.previous) {
          queryClient.setQueryData(queryKeys.accounts.list(), ctx.previous);
        }
      },
      onSettled: () => {
        queryClient.invalidateQueries({ queryKey: queryKeys.accounts.list() });
      },
    },
  );
}
