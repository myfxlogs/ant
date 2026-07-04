/**
 * workspaceStore — persists workspace UI state across sessions.
 * Uses Zustand persist middleware (matching authStore pattern).
 * Only persists layout/navigation state; never persists code or results.
 */
import { create } from 'zustand';
import { persist, createJSONStorage } from 'zustand/middleware';

export type CenterTab = 'design' | 'code' | 'backtest';
export type RightTab = 'chat' | 'code';

// Persisted to localStorage so code survives page refresh during a session.

interface WorkspaceState {
  accountId: string;
  symbol: string;
  timeframe: string;
  centerTab: CenterTab;
  rightTab: RightTab;
  leftSidebarCollapsed: boolean;
  bottomPanelCollapsed: boolean;
  quickTradeCollapsed: boolean;
  positionsPanelVisible: boolean;
  currentCode: string;
  currentCodeName: string;
  rightPanelWidth: number;
  _hasHydrated: boolean;
  setAccountId: (v: string) => void;
  setSymbol: (v: string) => void;
  setTimeframe: (v: string) => void;
  setCenterTab: (v: CenterTab) => void;
  setRightTab: (v: RightTab) => void;
  setLeftSidebarCollapsed: (v: boolean) => void;
  setBottomPanelCollapsed: (v: boolean) => void;
  setQuickTradeCollapsed: (v: boolean) => void;
  setPositionsPanelVisible: (v: boolean) => void;
  setCurrentCode: (v: string) => void;
  setCurrentCodeName: (v: string) => void;
  setRightPanelWidth: (v: number) => void;
}

export const useWorkspaceStore = create<WorkspaceState>()(
  persist(
    (set) => ({
      accountId: '',
      symbol: '',
      timeframe: '1h',
      centerTab: 'design',
      rightTab: 'chat',
      leftSidebarCollapsed: true,
      bottomPanelCollapsed: true,
      quickTradeCollapsed: true,
      positionsPanelVisible: false,
      currentCode: '',
      currentCodeName: '',
      rightPanelWidth: 380,
      _hasHydrated: false,
      setAccountId: (v) => set({ accountId: v }),
      setSymbol: (v) => set({ symbol: v }),
      setTimeframe: (v) => set({ timeframe: v }),
      setCenterTab: (v) => set({ centerTab: v }),
      setRightTab: (v) => set({ rightTab: v }),
      setLeftSidebarCollapsed: (v) => set({ leftSidebarCollapsed: v }),
      setBottomPanelCollapsed: (v) => set({ bottomPanelCollapsed: v }),
      setQuickTradeCollapsed: (v) => set({ quickTradeCollapsed: v }),
      setPositionsPanelVisible: (v) => set({ positionsPanelVisible: v }),
      setCurrentCode: (v) => set({ currentCode: v }),
      setCurrentCodeName: (v) => set({ currentCodeName: v }),
      setRightPanelWidth: (v) => set({ rightPanelWidth: Math.max(280, Math.min(600, v)) }),
    }),
    {
      name: 'ant-workspace-v5',
      storage: createJSONStorage(() => localStorage),
      partialize: (state) => ({
        accountId: state.accountId,
        symbol: state.symbol,
        timeframe: state.timeframe,
        centerTab: state.centerTab,
        rightTab: state.rightTab,
        leftSidebarCollapsed: state.leftSidebarCollapsed,
        bottomPanelCollapsed: state.bottomPanelCollapsed,
        quickTradeCollapsed: state.quickTradeCollapsed,
        positionsPanelVisible: state.positionsPanelVisible,
        rightPanelWidth: state.rightPanelWidth,
      }),
      onRehydrateStorage: () => (state) => {
        if (state) state._hasHydrated = true;
      },
    },
  ),
);
