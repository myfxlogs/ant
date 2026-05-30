import { useRpcQuery } from '@/hooks/useRpcQuery';
import { queryKeys } from './queryKeys';
import { accountApi } from '@/client/account';
import type { Account } from '@/types/account';

export function useAccountDetailQuery(id: string) {
  return useRpcQuery<Account>(
    queryKeys.accounts.detail(id),
    () => accountApi.get(id),
    { enabled: !!id },
  );
}
