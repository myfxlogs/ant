import { useCallback } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { useAccountStore } from '@/stores/accountStore';
import { accountApi } from '@/client/account';
import type { Account } from '@/types/account';
import { getErrorMessage } from '@/utils/error';
import { showSuccess, showError } from '@/utils/message';
import { queryKeys } from '@/queries/queryKeys';
import i18n from '@/i18n';

export function useAccount() {
  const queryClient = useQueryClient();

  // Single source of truth: TanStack Query cache, shared with MainLayout.
  const { data: accounts } = useQuery<Account[]>({
    queryKey: queryKeys.accounts.list(),
    queryFn: () => accountApi.list(),
  });

  // UI-only transient state from Zustand.
  const currentAccount = useAccountStore((s) => s.currentAccount);
  const loading = useAccountStore((s) => s.loading);
  const setCurrentAccount = useAccountStore((s) => s.setCurrentAccount);
  const setLoading = useAccountStore((s) => s.setLoading);
  const setEnablingAccount = useAccountStore((s) => s.setEnablingAccount);

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
      showError(getErrorMessage(error, i18n.t('accounts.messages.fetchListFailed')));
      return [];
    } finally {
      setLoading(false);
    }
  }, [setLoading, queryClient]);

  const fetchAccount = useCallback(async (id: string, showLoading = true) => {
    if (showLoading) setLoading(true);
    try {
      const account = await accountApi.get(id);
      setCurrentAccount(account);
      return account;
    } catch (error) {
      showError(getErrorMessage(error, i18n.t('accounts.messages.fetchAccountFailed')));
      return null;
    } finally {
      if (showLoading) setLoading(false);
    }
  }, [setLoading, setCurrentAccount]);

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
      showError(getErrorMessage(error, i18n.t('accounts.messages.disconnectFailed')));
      throw error;
    }
  }, [queryClient]);

  const disableAccount = useCallback(async (id: string) => {
    try {
      // Optimistic: set status to disconnected in TQ cache.
      queryClient.setQueryData<Account[]>(queryKeys.accounts.list(), (old) =>
        old?.map((a) => a.id === id ? { ...a, status: 'disconnected' } : a));
      await accountApi.update({ id, isDisabled: true });
      const account = await accountApi.get(id);
      queryClient.setQueryData<Account[]>(queryKeys.accounts.list(), (old) =>
        old?.map((a) => a.id === id ? account : a));
    } catch (error) {
      showError(getErrorMessage(error, i18n.t('accounts.messages.disableFailed')));
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
      await accountApi.update({ id, isDisabled: false });
      const account = await accountApi.get(id);
      queryClient.setQueryData<Account[]>(queryKeys.accounts.list(), (old) =>
        old?.map((a) => a.id === id ? account : a));
      return account;
    } catch (error) {
      showError(getErrorMessage(error, i18n.t('accounts.messages.enableFailed')));
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
      showSuccess(i18n.t('accounts.messages.deleted'));
    } catch (error) {
      const msg = getErrorMessage(error, '');
      showError(msg || i18n.t('accounts.messages.deleteFailed'));
      throw error;
    }
  }, [queryClient]);

  return {
    accounts: accounts ?? [],
    currentAccount,
    loading,
    fetchAccounts,
    fetchAccount,
    createAccount,
    connectAccount,
    disconnectAccount,
    disableAccount,
    enableAccount,
    deleteAccount,
    setCurrentAccount,
  };
}
