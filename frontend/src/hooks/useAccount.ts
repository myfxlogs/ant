import { MESSAGES_DELETE_FAILED_KEY, MESSAGES_DISABLE_FAILED_KEY, MESSAGES_DISCONNECT_FAILED_KEY, MESSAGES_ENABLE_FAILED_KEY, MESSAGES_FETCH_ACCOUNT_FAILED_KEY, MESSAGES_FETCH_LIST_FAILED_KEY } from '@/gen/ant/v1/i18n/accounts_keys';
import { MESSAGES_DELETED_KEY } from '@/gen/ant/v1/i18n/ai_settings_keys';

import { useCallback, useState } from 'react'
;
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { accountApi } from '@/client/account';
import type { Account } from '@/types/account';
import { getErrorMessage } from '@/utils/error';
import { showSuccess, showError } from '@/utils/message';
import { queryKeys } from '@/queries/queryKeys';
import i18n from '@/i18n';

export function useAccount() {
  const queryClient = useQueryClient();

  // Single source of truth: TanStack Query cache, shared with MainLayout.
  const { data: accounts, isLoading: _queryLoading } = useQuery<Account[]>({
    queryKey: queryKeys.accounts.list(),
    queryFn: () => accountApi.list(),
  });

  // Local transient state (was Zustand store — Task C removed it).
  const [loading, setLoading] = useState(false);
  const [_enablingAccount, setEnablingAccount] = useState<string | null>(null);

  const fetchAccounts = useCallback(async (force = false) => {
    const cached = queryClient.getQueryData<Account[]>(queryKeys.accounts.list());
    const hasData = Array.isArray(cached) && cached.length > 0;
    if (hasData && !force) {
      return cached;
    }

    setLoading(true);
    try {
      const accountList = await accountApi.list();
      const list: Account[] = Array.isArray(accountList) ? accountList : [];
      queryClient.setQueryData(queryKeys.accounts.list(), list);
      return list;
    } catch (error) {
      showError(getErrorMessage(error, i18n.t(MESSAGES_FETCH_LIST_FAILED_KEY)));
      return [];
    } finally {
      setLoading(false);
    }
  }, [setLoading, queryClient]);

  const fetchAccount = useCallback(async (id: string, showLoading = true) => {
    if (showLoading) setLoading(true);
    try {
      const account = await accountApi.get(id);
      return account;
    } catch (error) {
      showError(getErrorMessage(error, i18n.t(MESSAGES_FETCH_ACCOUNT_FAILED_KEY)));
      return null;
    } finally {
      if (showLoading) setLoading(false);
    }
  }, [setLoading]);

  const createAccount = useCallback(async (data: {
    login: string; password: string; mtType: string;
    brokerCompany: string; brokerServer: string; brokerHost: string;
  }) => {
    setLoading(true);
    try {
      const account = await accountApi.create(data);
      queryClient.invalidateQueries({ queryKey: queryKeys.accounts.list() });
      return account;
    } catch (_e) {
      throw _e;
    } finally {
      setLoading(false);
    }
  }, [setLoading, queryClient]);

  const connectAccount = useCallback(async (id: string) => {
    try {
      await accountApi.connect(id);
      const account = await accountApi.get(id);
      queryClient.setQueryData<Account[]>(queryKeys.accounts.list(), (old) =>
        old?.map((a) => a.id === id ? account : a));
      return account;
    } catch (_e) {
      throw _e;
    }
  }, [queryClient]);

  const disconnectAccount = useCallback(async (id: string) => {
    try {
      await accountApi.disconnect(id);
      const account = await accountApi.get(id);
      queryClient.setQueryData<Account[]>(queryKeys.accounts.list(), (old) =>
        old?.map((a) => a.id === id ? account : a));
    } catch (error) {
      showError(getErrorMessage(error, i18n.t(MESSAGES_DISCONNECT_FAILED_KEY)));
      throw error;
    }
  }, [queryClient]);

  const disableAccount = useCallback(async (id: string) => {
    try {
      // Optimistic: set status to disconnected in TQ cache.
      queryClient.setQueryData<Account[]>(queryKeys.accounts.list(), (old) =>
        old?.map((a) => a.id === id ? { ...a, status: 'disconnected' } : a));
      await accountApi.disconnect(id);
      const account = await accountApi.get(id);
      queryClient.setQueryData<Account[]>(queryKeys.accounts.list(), (old) =>
        old?.map((a) => a.id === id ? account : a));
    } catch (error) {
      showError(getErrorMessage(error, i18n.t(MESSAGES_DISABLE_FAILED_KEY)));
      // Rollback by refetching.
      try {
        const account = await accountApi.get(id);
        queryClient.setQueryData<Account[]>(queryKeys.accounts.list(), (old) =>
          old?.map((a) => a.id === id ? account : a));
      } catch { /* ignore */ }
      throw error;
    }
  }, [queryClient]);

  const enableAccount = useCallback(async (id: string) => {
    setEnablingAccount(id);
    try {
      // Optimistic: set status to connecting in TQ cache.
      queryClient.setQueryData<Account[]>(queryKeys.accounts.list(), (old) =>
        old?.map((a) => a.id === id ? { ...a, status: 'connecting' } : a));
      await accountApi.reconnect(id);
      const account = await accountApi.get(id);
      queryClient.setQueryData<Account[]>(queryKeys.accounts.list(), (old) =>
        old?.map((a) => a.id === id ? account : a));
      return account;
    } catch (error) {
      showError(getErrorMessage(error, i18n.t(MESSAGES_ENABLE_FAILED_KEY)));
      // Rollback.
      try {
        const account = await accountApi.get(id);
        queryClient.setQueryData<Account[]>(queryKeys.accounts.list(), (old) =>
          old?.map((a) => a.id === id ? account : a));
      } catch { /* ignore */ }
      throw error;
    } finally {
      setEnablingAccount(null);
    }
  }, [setEnablingAccount, queryClient]);

  const deleteAccount = useCallback(async (id: string, password: string) => {
    try {
      await accountApi.delete(id, password);
      queryClient.setQueryData<Account[]>(queryKeys.accounts.list(), (old) =>
        old?.filter((a) => a.id !== id));
      showSuccess(i18n.t(MESSAGES_DELETED_KEY));
    } catch (error) {
      const msg = getErrorMessage(error, '');
      showError(msg || i18n.t(MESSAGES_DELETE_FAILED_KEY));
      throw error;
    }
  }, [queryClient]);

  return {
    accounts: accounts ?? [],
    loading,
    fetchAccounts,
    fetchAccount,
    createAccount,
    connectAccount,
    disconnectAccount,
    disableAccount,
    enableAccount,
    deleteAccount,
  };
}
