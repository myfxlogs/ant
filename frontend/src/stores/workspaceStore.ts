/**
 * workspaceStore — persists workspace UI state across sessions.
 * Uses Zustand persist middleware (matching authStore pattern).
 * Only persists layout/navigation state; never persists code or results.
 */
import { create } from 'zustand';
import { persist, createJSONStorage } from 'zustand/middleware';

interface WorkspaceState {
  accountId: string;
  symbol: string;
  timeframe: string;
  codePanelVisible: boolean;
  quickTradeVisible: boolean;
  _hasHydrated: boolean;
  setAccountId: (v: string) => void;
  setSymbol: (v: string) => void;
  setTimeframe: (v: string) => void;
  setCodePanelVisible: (v: boolean) => void;
  setQuickTradeVisible: (v: boolean) => void;
}

export const useWorkspaceStore = create<WorkspaceState>()(
  persist(
    (set) => ({
      accountId: '',
      symbol: '',
      timeframe: '1h',
      codePanelVisible: true,
      quickTradeVisible: true,
      _hasHydrated: false,
      setAccountId: (v) => set({ accountId: v }),
      setSymbol: (v) => set({ symbol: v }),
      setTimeframe: (v) => set({ timeframe: v }),
      setCodePanelVisible: (v) => set({ codePanelVisible: v }),
      setQuickTradeVisible: (v) => set({ quickTradeVisible: v }),
    }),
    {
      name: 'ant-workspace-state',
      storage: createJSONStorage(() => localStorage),
      partialize: (state) => ({
        accountId: state.accountId,
        symbol: state.symbol,
        timeframe: state.timeframe,
        codePanelVisible: state.codePanelVisible,
        quickTradeVisible: state.quickTradeVisible,
      }),
      onRehydrateStorage: () => (state) => {
        if (state) state._hasHydrated = true;
      },
    },
  ),
);
