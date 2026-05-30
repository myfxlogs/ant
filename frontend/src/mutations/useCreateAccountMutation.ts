import { useQueryClient } from '@tanstack/react-query';
import { useRpcMutation } from '@/hooks/useRpcMutation';
import { accountApi } from '@/client/account';
import { queryKeys } from '@/queries/queryKeys';
import type { Account } from '@/types/account';

interface CreateAccountVars {
  login: string;
  password: string;
  mtType: string;
  brokerCompany: string;
  brokerServer: string;
  brokerHost: string;
}

export function useCreateAccountMutation() {
  const queryClient = useQueryClient();
  return useRpcMutation<Account, Error, CreateAccountVars>(
    (vars) => accountApi.create(vars),
    {
      onSuccess: () => {
        queryClient.invalidateQueries({ queryKey: queryKeys.accounts.list() });
      },
    },
  );
}
