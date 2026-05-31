import { useQueryClient } from '@tanstack/react-query';
import { useRpcMutation } from '@/hooks/useRpcMutation';
import { accountApi } from '@/client/account';
import { useAccountStore } from '@/stores/accountStore';
import { queryKeys } from '@/queries/queryKeys';

export function useDeleteAccountMutation() {
  const queryClient = useQueryClient();
  return useRpcMutation(
    ({ id, password }: { id: string; password?: string }) =>
      accountApi.delete(id, password),
    {
      onSuccess: (_data, vars) => {
        // Remove from Zustand (Dashboard reads this immediately)
        useAccountStore.getState().removeAccount(vars.id);
        // Remove from TanStack Query cache
        queryClient.removeQueries({ queryKey: queryKeys.accounts.detail(vars.id) });
        queryClient.removeQueries({ queryKey: queryKeys.accounts.financials(vars.id) });
        queryClient.removeQueries({ queryKey: queryKeys.positions.byAccount(vars.id) });
        queryClient.removeQueries({
          predicate: (query) =>
            Array.isArray(query.queryKey) &&
            query.queryKey[0] === 'analytics' &&
            query.queryKey[1] === vars.id,
        });
        queryClient.invalidateQueries({ queryKey: queryKeys.accounts.list() });
      },
    },
  );
}
