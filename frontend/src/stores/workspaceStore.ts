/**
 * workspaceStore — persists workspace UI state across sessions.
 * Uses Zustand persist middleware (matching authStore pattern).
 * Only persists layout/navigation state; never persists code or results.
 */
import { create } from 'zustand';
import { persist, createJSONStorage } from 'zustand/middleware';

export type CenterTab = 'design' | 'code' | 'backtest';
export type RightTab = 'results' | 'quicktrade';

interface WorkspaceState {
  accountId: string;
  symbol: string;
  timeframe: string;
  centerTab: CenterTab;
  rightTab: RightTab;
  positionsPanelVisible: boolean;
  _hasHydrated: boolean;
  setAccountId: (v: string) => void;
  setSymbol: (v: string) => void;
  setTimeframe: (v: string) => void;
  setCenterTab: (v: CenterTab) => void;
  setRightTab: (v: RightTab) => void;
  setPositionsPanelVisible: (v: boolean) => void;
}

export const useWorkspaceStore = create<WorkspaceState>()(
  persist(
    (set) => ({
      accountId: '',
      symbol: '',
      timeframe: '1h',
      centerTab: 'design' as CenterTab,
      rightTab: 'results' as RightTab,
      positionsPanelVisible: false,
      _hasHydrated: false,
      setAccountId: (v) => set({ accountId: v }),
      setSymbol: (v) => set({ symbol: v }),
      setTimeframe: (v) => set({ timeframe: v }),
      setCenterTab: (v) => set({ centerTab: v }),
      setRightTab: (v) => set({ rightTab: v }),
      setPositionsPanelVisible: (v) => set({ positionsPanelVisible: v }),
    }),
    {
      name: 'ant-workspace-state',
      storage: createJSONStorage(() => localStorage),
      partialize: (state) => ({
        accountId: state.accountId,
        symbol: state.symbol,
        timeframe: state.timeframe,
        centerTab: state.centerTab,
        rightTab: state.rightTab,
        positionsPanelVisible: state.positionsPanelVisible,
      }),
      onRehydrateStorage: () => (state) => {
        if (state) state._hasHydrated = true;
      },
    },
  ),
);
