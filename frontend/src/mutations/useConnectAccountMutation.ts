import { useQueryClient } from '@tanstack/react-query';
import { useRpcMutation } from '@/hooks/useRpcMutation';
import { accountApi } from '@/client/account';
import { queryKeys } from '@/queries/queryKeys';

export function useConnectAccountMutation() {
  const queryClient = useQueryClient();
  return useRpcMutation(accountApi.connect, {
    onSuccess: (_data, vars) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.accounts.list() });
      queryClient.invalidateQueries({
        queryKey: queryKeys.accounts.detail(vars),
      });
    },
  });
}
