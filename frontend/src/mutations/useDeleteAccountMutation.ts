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
      onMutate: async (vars) => {
        // Optimistic: remove from Zustand immediately so UI updates instantly.
        const prev = useAccountStore.getState().accounts;
        useAccountStore.getState().removeAccount(vars.id);
        // Cancel active queries to prevent 404 refetches.
        await queryClient.cancelQueries({ queryKey: queryKeys.accounts.detail(vars.id) });
        await queryClient.cancelQueries({ queryKey: queryKeys.accounts.financials(vars.id) });
        await queryClient.cancelQueries({ queryKey: queryKeys.positions.byAccount(vars.id) });
        return { prev };
      },
      onError: (_err, vars, context) => {
        // Rollback: restore account list if delete failed.
        if (context?.prev) useAccountStore.getState().setAccounts(context.prev);
      },
      onSuccess: (_data, vars) => {
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
