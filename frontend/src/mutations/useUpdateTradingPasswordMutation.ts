import { useQueryClient } from '@tanstack/react-query';
import { useRpcMutation } from '@/hooks/useRpcMutation';
import { accountApi } from '@/client/account';
import { queryKeys } from '@/queries/queryKeys';

interface UpdatePasswordVars {
  id: string;
  newPassword: string;
  oldPassword?: string;
}

interface UpdatePasswordResult {
  success: boolean;
  hasTradePermission: boolean;
  isInvestor: boolean;
  message: string;
}

export function useUpdateTradingPasswordMutation() {
  const queryClient = useQueryClient();
  return useRpcMutation<UpdatePasswordResult, Error, UpdatePasswordVars>(
    (vars) => accountApi.updateTradingPassword(vars.id, vars.newPassword, vars.oldPassword),
    {
      onSuccess: (_data, vars) => {
        queryClient.invalidateQueries({
          queryKey: queryKeys.accounts.detail(vars.id),
        });
      },
    },
  );
}
