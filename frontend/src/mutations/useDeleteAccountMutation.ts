import { useQueryClient } from '@tanstack/react-query';
import { useRpcMutation } from '@/hooks/useRpcMutation';
import { accountApi } from '@/client/account';
import { useAccountStore } from '@/stores/accountStore';
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
        const prevZustand = useAccountStore.getState().accounts;
        // Optimistic: remove from both stores immediately.
        queryClient.setQueryData<Account[]>(queryKeys.accounts.list(), (old) =>
          (old ?? []).filter((a) => a.id !== vars.id),
        );
        useAccountStore.getState().removeAccount(vars.id);
        return { prevTq, prevZustand };
      },
      onError: (_err, _vars, context) => {
        if (context?.prevTq) queryClient.setQueryData(queryKeys.accounts.list(), context.prevTq);
        if (context?.prevZustand) useAccountStore.getState().setAccounts(context.prevZustand);
      },
      onSettled: (_data, _err, vars) => {
        queryClient.invalidateQueries({ queryKey: queryKeys.accounts.list() });
        queryClient.removeQueries({ queryKey: queryKeys.accounts.detail(vars.id) });
        queryClient.removeQueries({ queryKey: queryKeys.accounts.financials(vars.id) });
        queryClient.removeQueries({ queryKey: queryKeys.positions.byAccount(vars.id) });
      },
    },
  );
}
