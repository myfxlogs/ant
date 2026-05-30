import { useRpcQuery } from '@/hooks/useRpcQuery';
import { queryKeys } from './queryKeys';
import { accountApi } from '@/client/account';
import type { Account } from '@/types/account';

export function useAccountListQuery() {
  return useRpcQuery<Account[]>(queryKeys.accounts.list(), () =>
    accountApi.list(),
  );
}
