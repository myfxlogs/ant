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
      onSettled: () => {
        // Refetch list to confirm with server. Per-account caches
        // (detail, financials, positions) are garbage-collected when
        // the component unmounts on navigation — no need to remove here.
        queryClient.invalidateQueries({ queryKey: queryKeys.accounts.list() });
      },
    },
  );
}
