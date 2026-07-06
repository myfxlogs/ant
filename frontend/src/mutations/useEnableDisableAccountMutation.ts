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
 * Uses disconnect/reconnect RPCs (migration 187 replaced is_disabled column
 * with the account_status state machine).
 * Updates Query cache immediately, rolls back on error.
 */
export function useEnableDisableAccountMutation() {
  const queryClient = useQueryClient();

  return useRpcMutation<Account, Error, ToggleVars>(
    ({ id, isDisabled }) =>
      isDisabled ? accountApi.disconnect(id) : accountApi.reconnect(id),
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
                ? { ...a, isDisabled, status: isDisabled ? 'disconnected' : 'connecting' }
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
      onSettled: (_data, _err, vars) => {
        queryClient.invalidateQueries({ queryKey: queryKeys.accounts.list() });
        // When re-enabling, force-refetch positions so stale cache is replaced.
        if (!vars?.isDisabled) {
          queryClient.invalidateQueries({ queryKey: queryKeys.positions.byAccount(vars.id) });
        }
      },
    },
  );
}
