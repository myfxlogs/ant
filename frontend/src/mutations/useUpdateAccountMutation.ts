import { useQueryClient } from '@tanstack/react-query';
import { useRpcMutation } from '@/hooks/useRpcMutation';
import { accountApi } from '@/client/account';
import { queryKeys } from '@/queries/queryKeys';
import type { Account } from '@/types/account';

interface UpdateAccountVars {
  id: string;
  brokerCompany?: string;
  brokerServer?: string;
  brokerHost?: string;
  isDisabled?: boolean;
}

export function useUpdateAccountMutation() {
  const queryClient = useQueryClient();
  return useRpcMutation<Account, Error, UpdateAccountVars>(
    (vars) => accountApi.update(vars),
    {
      onSuccess: (_data, vars) => {
        queryClient.invalidateQueries({ queryKey: queryKeys.accounts.list() });
        queryClient.invalidateQueries({
          queryKey: queryKeys.accounts.detail(vars.id),
        });
      },
    },
  );
}
