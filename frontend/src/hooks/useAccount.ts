import { useCallback } from 'react';
import { useAccountStore } from '@/stores/accountStore';
import { accountApi } from '@/client/account';
import { getErrorMessage } from '@/utils/error';
import { showSuccess, showError } from '@/utils/message';
import i18n from '@/i18n';

export function useAccount() {
  const accounts = useAccountStore((state) => state.accounts);
  const currentAccount = useAccountStore((state) => state.currentAccount);
  const loading = useAccountStore((state) => state.loading);
  const setAccounts = useAccountStore((state) => state.setAccounts);
  const setCurrentAccount = useAccountStore((state) => state.setCurrentAccount);
  const setLoading = useAccountStore((state) => state.setLoading);
  const setEnablingAccount = useAccountStore((state) => state.setEnablingAccount);
  const addAccount = useAccountStore((state) => state.addAccount);
  const updateAccount = useAccountStore((state) => state.updateAccount);
  const removeAccount = useAccountStore((state) => state.removeAccount);

  const fetchAccounts = useCallback(async (force = false) => {
    const currentAccounts = useAccountStore.getState().accounts;
    // 如果已有数据且不是强制刷新，不显示 loading，直接返回
    const hasData = currentAccounts && currentAccounts.length > 0;
    if (hasData && !force) {
      return currentAccounts;
    }
    
    setLoading(true);
    try {
      const accountList = await accountApi.list();
      const accounts = Array.isArray(accountList) ? accountList : [];
      setAccounts(accounts as any);
      return accounts;
    } catch (error) {
      showError(getErrorMessage(error, i18n.t('accounts.messages.fetchListFailed')));
      setAccounts([]);
      return [];
    } finally {
      setLoading(false);
    }
  }, [setLoading, setAccounts]);

  const fetchAccount = useCallback(async (id: string, showLoading = true) => {
    if (showLoading) {
      setLoading(true);
    }
    try {
      const account = await accountApi.get(id);
      setCurrentAccount(account as any);
      return account;
    } catch (error) {
      showError(getErrorMessage(error, i18n.t('accounts.messages.fetchAccountFailed')));
      return null;
    } finally {
      if (showLoading) {
        setLoading(false);
      }
    }
  }, [setLoading, setCurrentAccount]);

  const createAccount = useCallback(async (data: {
    login: string;
    password: string;
    mtType: string;
    brokerCompany: string;
    brokerServer: string;
    brokerHost: string;
  }) => {
    setLoading(true);
    try {
      const account = await accountApi.create(data);
      addAccount(account as any);
      showSuccess(i18n.t('accounts.messages.createdSuccess'));
      return account;
    } catch (error) {
      showError(getErrorMessage(error, i18n.t('accounts.messages.createFailed')));
      throw error;
    } finally {
      setLoading(false);
    }
  }, [setLoading, addAccount]);

  const bindAccount = useCallback(async (data: {
    login: string;
    password: string;
    mtType: string;
    brokerCompany: string;
    brokerServer: string;
    brokerHost: string;
  }) => {
    return createAccount(data);
  }, [createAccount]);

  const connectAccount = useCallback(async (id: string) => {
    try {
      const result = await accountApi.connect(id);
      if (result.success) {
        showSuccess(i18n.t('accounts.messages.connectSuccess'));
      }
      const account = await accountApi.get(id);
      updateAccount(account as any);
      return account;
    } catch (error) {
      showError(getErrorMessage(error, i18n.t('accounts.messages.connectFailed')));
      throw error;
    }
  }, [updateAccount]);

  const disconnectAccount = useCallback(async (id: string) => {
    try {
      await accountApi.disconnect(id);
      const account = await accountApi.get(id);
      updateAccount(account as any);
    } catch (error) {
      showError(getErrorMessage(error, i18n.t('accounts.messages.disconnectFailed')));
      throw error;
    }
  }, [updateAccount]);

  const disableAccount = useCallback(async (id: string) => {
    try {
      // Optimistic UI: reflect disabled intent immediately (MT-official-like).
      useAccountStore.getState().updateAccountStatus(id, 'disconnected');
      await accountApi.update({ id, isDisabled: true });
      const account = await accountApi.get(id);
      updateAccount(account as any);
    } catch (error) {
      showError(getErrorMessage(error, i18n.t('accounts.messages.disableFailed')));
      // Rollback by refetching account state.
      try {
        const account = await accountApi.get(id);
        updateAccount(account as any);
      } catch {
        // ignore
      }
      throw error;
    }
  }, [updateAccount]);

  const enableAccount = useCallback(async (id: string) => {
    setEnablingAccount(id);
    try {
      // Optimistic UI: reflect enabling intent immediately (MT-official-like).
      useAccountStore.getState().updateAccountStatus(id, 'connecting');
      await accountApi.update({ id, isDisabled: false });
      const account = await accountApi.get(id);
      updateAccount(account as any);
      return account;
    } finally {
      setEnablingAccount(null);
    }
  }, [updateAccount, setEnablingAccount]);

  const deleteAccount = useCallback(async (id: string) => {
    try {
      await accountApi.delete(id);
      removeAccount(id);
      showSuccess(i18n.t('accounts.messages.deleted'));
    } catch (error) {
      showError(getErrorMessage(error, i18n.t('accounts.messages.deleteFailed')));
      throw error;
    }
  }, [removeAccount]);

  return {
    accounts,
    currentAccount,
    loading,
    fetchAccounts,
    fetchAccount,
    createAccount,
    bindAccount,
    connectAccount,
    disconnectAccount,
    disableAccount,
    enableAccount,
    deleteAccount,
    setCurrentAccount,
  };
}
