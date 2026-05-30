import { create } from 'zustand';
import type { Account } from '@/types/account';
import { toCamelCase } from '../adapters/dataAdapter';

interface AccountState {
  accounts: Account[];
  /** O(1) lookup map maintained in sync with accounts array. */
  accountsById: Map<string, Account>;
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
  patchAccountFinancials: (_accountId: string, _patch: Partial<Account>) => void;
  removeAccount: (_id: string) => void;
}

function buildById(accounts: Account[]): Map<string, Account> {
  const m = new Map<string, Account>();
  for (const a of accounts) m.set(a.id, a);
  return m;
}

export const useAccountStore = create<AccountState>((set) => ({
  accounts: [],
  accountsById: new Map(),
  currentAccount: null,
  loading: false,
  enablingAccount: null,
  setAccounts: (accounts) => {
    const camelAccounts = Array.isArray(accounts) ? toCamelCase(accounts) : [];
    set({ accounts: camelAccounts, accountsById: buildById(camelAccounts) });
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
    set((state) => {
      const camel = toCamelCase(account);
      const next = [...state.accounts, camel];
      const nextById = new Map(state.accountsById);
      nextById.set(camel.id, camel);
      return { accounts: next, accountsById: nextById };
    });
  },
  updateAccount: (account) => {
    set((state) => {
      const camel = toCamelCase(account);
      const next = state.accounts.map((a) => (a.id === camel.id ? camel : a));
      const nextById = new Map(state.accountsById);
      nextById.set(camel.id, camel);
      return {
        accounts: next,
        accountsById: nextById,
        currentAccount: state.currentAccount?.id === camel.id ? camel : state.currentAccount,
      };
    });
  },
  updateAccountStatus: (accountId, status) => {
    set((state) => {
      const nextById = new Map(state.accountsById);
      const existing = nextById.get(accountId);
      if (existing) nextById.set(accountId, { ...existing, status });
      return {
        accounts: state.accounts.map((a) =>
          a.id === accountId ? { ...a, status } : a
        ),
        accountsById: nextById,
        currentAccount: state.currentAccount?.id === accountId
          ? { ...state.currentAccount, status }
          : state.currentAccount,
      };
    });
  },
  removeAccount: (id) => {
    set((state) => {
      const nextById = new Map(state.accountsById);
      nextById.delete(id);
      return {
        accounts: state.accounts.filter((a) => a.id !== id),
        accountsById: nextById,
        currentAccount: state.currentAccount?.id === id ? null : state.currentAccount,
      };
    });
  },
  patchAccountFinancials: (accountId, patch) => {
    set((state) => {
      // O(1) lookup via accountsById map instead of O(n) scan.
      const existing = state.accountsById.get(accountId);
      if (!existing) return {};
      const nextById = new Map(state.accountsById);
      const updated = { ...existing, ...patch };
      nextById.set(accountId, updated);
      // Rebuild accounts array with the single changed entry for O(n) render but O(1) find.
      const nextAccounts = state.accounts.map((a) =>
        a.id === accountId ? updated : a,
      );
      return {
        accounts: nextAccounts,
        accountsById: nextById,
        currentAccount:
          state.currentAccount?.id === accountId
            ? { ...state.currentAccount, ...patch }
            : state.currentAccount,
      };
    });
  },
}));
