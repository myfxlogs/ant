import { useQueryClient } from '@tanstack/react-query';
import { useRpcMutation } from '@/hooks/useRpcMutation';
import { accountApi } from '@/client/account';
import { queryKeys } from '@/queries/queryKeys';

export function useDeleteAccountMutation() {
  const queryClient = useQueryClient();
  return useRpcMutation(
    ({ id, password }: { id: string; password?: string }) =>
      accountApi.delete(id, password),
    {
      onSuccess: (_data, vars) => {
        queryClient.removeQueries({
          queryKey: queryKeys.accounts.detail(vars.id),
        });
        queryClient.invalidateQueries({ queryKey: queryKeys.accounts.list() });
      },
    },
  );
}
