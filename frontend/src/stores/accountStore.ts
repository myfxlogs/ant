import { create } from 'zustand';
import type { Account } from '@/types/account';
import { toCamelCase } from '../adapters/dataAdapter';

interface AccountState {
  accounts: Account[];
  currentAccount: Account | null;
  loading: boolean;
  enablingAccount: string | null;
  setAccounts: (_accounts: Account[]) => void;
  setCurrentAccount: (_account: Account | null) => void;
  setLoading: (_loading: boolean) => void;
  setEnablingAccount: (_accountId: string | null) => void;
  addAccount: (_account: Account) => void;
  updateAccount: (_account: Account) => void;
  updateAccountStatus: (_accountId: string, _status: string) => void;
  removeAccount: (_id: string) => void;
}

export const useAccountStore = create<AccountState>((set) => ({
  accounts: [],
  currentAccount: null,
  loading: false,
  enablingAccount: null,
  setAccounts: (accounts) => {
    const camelAccounts = Array.isArray(accounts) ? toCamelCase(accounts) : [];
    set({ accounts: camelAccounts });
  },
  setCurrentAccount: (account) => {
    const camelAccount = account ? toCamelCase(account) : null;
    set({ currentAccount: camelAccount });
  },
  setLoading: (loading) => {
    set({ loading });
  },
  setEnablingAccount: (accountId) => {
    set({ enablingAccount: accountId });
  },
  addAccount: (account) => {
    set((state) => ({ accounts: [...state.accounts, toCamelCase(account)] }));
  },
  updateAccount: (account) => {
    set((state) => ({
      accounts: state.accounts.map((a) => (a.id === account.id ? toCamelCase(account) : a)),
      currentAccount: state.currentAccount?.id === account.id ? toCamelCase(account) : state.currentAccount,
    }));
  },
  updateAccountStatus: (accountId, status) => {
    set((state) => ({
      accounts: state.accounts.map((a) => 
        a.id === accountId ? { ...a, status } : a
      ),
      currentAccount: state.currentAccount?.id === accountId 
        ? { ...state.currentAccount, status } 
        : state.currentAccount,
    }));
  },
  removeAccount: (id) => {
    set((state) => ({
      accounts: state.accounts.filter((a) => a.id !== id),
      currentAccount: state.currentAccount?.id === id ? null : state.currentAccount,
    }));
  },
}));
