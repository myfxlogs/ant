import { useRpcQuery } from '@/hooks/useRpcQuery';
import { queryKeys } from './queryKeys';
import { accountApi } from '@/client/account';
import type { Account } from '@/types/account';

export function useAccountDetailQuery(id: string) {
  return useRpcQuery<Account>(
    queryKeys.accounts.detail(id),
    () => accountApi.get(id),
    // staleTime=0 — account status changes frequently (connecting/connected/error);
    // always fetch fresh on mount to avoid showing stale state.
    { enabled: !!id, staleTime: 0 },
  );
}
