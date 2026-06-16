import { create } from 'zustand';
import type { Account } from '@/types/account';

/**
 * Lightweight UI-only store for transient account selection and mutation state.
 * Account data (list, financials, status) lives in TanStack Query cache —
 * the single source of truth kept fresh by SSE bridge events.
 *
 * NOTE: setCurrentAccount stores data that has already been camelCase-converted
 * by the API client (accountApi.get). No second conversion needed.
 */
interface AccountState {
  /** Currently selected/viewed account (transient UI selection). */
  currentAccount: Account | null;
  /** Whether a mutation (create/delete/enable/disable) is in progress. */
  loading: boolean;
  /** Account ID currently being enabled (for per-row spinner). */
  enablingAccount: string | null;
  setCurrentAccount: (_account: Account | null) => void;
  setLoading: (_loading: boolean) => void;
  setEnablingAccount: (_accountId: string | null) => void;
}

export const useAccountStore = create<AccountState>((set) => ({
  currentAccount: null,
  loading: false,
  enablingAccount: null,
  setCurrentAccount: (account) => {
    set({ currentAccount: account });
  },
  setLoading: (loading) => {
    set({ loading });
  },
  setEnablingAccount: (accountId) => {
    set({ enablingAccount: accountId });
  },
}));
