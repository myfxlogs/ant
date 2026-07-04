/**
 * workspaceStore — persists workspace UI state across sessions.
 * Uses Zustand persist middleware (matching authStore pattern).
 * Only persists layout/navigation state; never persists code or results.
 */
import { create } from 'zustand';
import { persist, createJSONStorage } from 'zustand/middleware';

export type CenterTab = 'design' | 'code' | 'backtest';
export type RightTab = 'chat' | 'code';

interface WorkspaceState {
  accountId: string;
  symbol: string;
  timeframe: string;
  rightTab: RightTab;
  leftSidebarCollapsed: boolean;
  bottomPanelCollapsed: boolean;
  quickTradeCollapsed: boolean;
  positionsPanelVisible: boolean;
  _hasHydrated: boolean;
  setAccountId: (v: string) => void;
  setSymbol: (v: string) => void;
  setTimeframe: (v: string) => void;
  setRightTab: (v: RightTab) => void;
  setLeftSidebarCollapsed: (v: boolean) => void;
  setBottomPanelCollapsed: (v: boolean) => void;
  setQuickTradeCollapsed: (v: boolean) => void;
  setPositionsPanelVisible: (v: boolean) => void;
}

export const useWorkspaceStore = create<WorkspaceState>()(
  persist(
    (set) => ({
      accountId: '',
      symbol: '',
      timeframe: '1h',
      rightTab: 'chat',
      leftSidebarCollapsed: true,
      bottomPanelCollapsed: true,
      quickTradeCollapsed: true,
      positionsPanelVisible: false,
      _hasHydrated: false,
      setAccountId: (v) => set({ accountId: v }),
      setSymbol: (v) => set({ symbol: v }),
      setTimeframe: (v) => set({ timeframe: v }),
      setRightTab: (v) => set({ rightTab: v }),
      setLeftSidebarCollapsed: (v) => set({ leftSidebarCollapsed: v }),
      setBottomPanelCollapsed: (v) => set({ bottomPanelCollapsed: v }),
      setQuickTradeCollapsed: (v) => set({ quickTradeCollapsed: v }),
      setPositionsPanelVisible: (v) => set({ positionsPanelVisible: v }),
    }),
    {
      name: 'ant-workspace-v5',
      storage: createJSONStorage(() => localStorage),
      partialize: (state) => ({
        accountId: state.accountId,
        symbol: state.symbol,
        timeframe: state.timeframe,
        rightTab: state.rightTab,
        leftSidebarCollapsed: state.leftSidebarCollapsed,
        bottomPanelCollapsed: state.bottomPanelCollapsed,
        quickTradeCollapsed: state.quickTradeCollapsed,
        positionsPanelVisible: state.positionsPanelVisible,
      }),
      onRehydrateStorage: () => (state) => {
        if (state) state._hasHydrated = true;
      },
    },
  ),
);
