import { useQueryClient } from '@tanstack/react-query';
import { useRpcMutation } from '@/hooks/useRpcMutation';
import { accountApi } from '@/client/account';
import { queryKeys } from '@/queries/queryKeys';
import { trackFunnelEvent, FunnelEvents } from '@/utils/analytics';

export function useConnectAccountMutation() {
  const queryClient = useQueryClient();
  return useRpcMutation(accountApi.connect, {
    onSuccess: (_data, vars) => {
      trackFunnelEvent(FunnelEvents.FIRST_MT_BIND);
      queryClient.invalidateQueries({ queryKey: queryKeys.accounts.list() });
      queryClient.invalidateQueries({
        queryKey: queryKeys.accounts.detail(vars),
      });
    },
  });
}
