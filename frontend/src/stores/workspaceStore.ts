/**
 * workspaceStore — persists workspace UI state across sessions.
 * Uses Zustand persist middleware with slice-creator pattern (ADR-0027 Phase C).
 * Only persists layout/navigation state; never persists code or results.
 */
import { create } from 'zustand';
import { persist, createJSONStorage } from 'zustand/middleware';

export type CenterTab = 'chat' | 'code' | 'import' | 'backtest';

// ── Slice interfaces ──────────────────────────────────────────────

export interface AccountSlice {
  accountId: string;
  symbol: string;
  timeframe: string;
  setAccountId: (v: string) => void;
  setSymbol: (v: string) => void;
  setTimeframe: (v: string) => void;
}

export interface LayoutSlice {
  centerTab: CenterTab;
  leftSidebarCollapsed: boolean;
  bottomPanelCollapsed: boolean;
  quickTradeCollapsed: boolean;
  positionsPanelVisible: boolean;
  setCenterTab: (v: CenterTab) => void;
  setLeftSidebarCollapsed: (v: boolean) => void;
  setBottomPanelCollapsed: (v: boolean) => void;
  setQuickTradeCollapsed: (v: boolean) => void;
  setPositionsPanelVisible: (v: boolean) => void;
}

export interface CodeSlice {
  currentCode: string;
  currentCodeName: string;
  setCurrentCode: (v: string) => void;
  setCurrentCodeName: (v: string) => void;
}

interface HydrationSlice {
  _hasHydrated: boolean;
}

type WorkspaceState = AccountSlice & LayoutSlice & CodeSlice & HydrationSlice;

// ── Slice creators ────────────────────────────────────────────────

function createAccountSlice(set: (partial: Partial<WorkspaceState>) => void): AccountSlice {
  return {
    accountId: '',
    symbol: '',
    timeframe: '1h',
    setAccountId: (v) => set({ accountId: v }),
    setSymbol: (v) => set({ symbol: v }),
    setTimeframe: (v) => set({ timeframe: v }),
  };
}

function createLayoutSlice(set: (partial: Partial<WorkspaceState>) => void): LayoutSlice {
  return {
    centerTab: 'chat',
    leftSidebarCollapsed: true,
    bottomPanelCollapsed: true,
    quickTradeCollapsed: true,
    positionsPanelVisible: false,
    setCenterTab: (v) => set({ centerTab: v }),
    setLeftSidebarCollapsed: (v) => set({ leftSidebarCollapsed: v }),
    setBottomPanelCollapsed: (v) => set({ bottomPanelCollapsed: v }),
    setQuickTradeCollapsed: (v) => set({ quickTradeCollapsed: v }),
    setPositionsPanelVisible: (v) => set({ positionsPanelVisible: v }),
  };
}

function createCodeSlice(set: (partial: Partial<WorkspaceState>) => void): CodeSlice {
  return {
    currentCode: '',
    currentCodeName: '',
    setCurrentCode: (v) => set({ currentCode: v }),
    setCurrentCodeName: (v) => set({ currentCodeName: v }),
  };
}

// ── Combined store ────────────────────────────────────────────────

export const useWorkspaceStore = create<WorkspaceState>()(
  persist(
    (set) => ({
      ...createAccountSlice(set),
      ...createLayoutSlice(set),
      ...createCodeSlice(set),
      _hasHydrated: false,
    }),
    {
      name: 'ant-workspace-v7',
      storage: createJSONStorage(() => localStorage),
      partialize: (state) => ({
        accountId: state.accountId,
        symbol: state.symbol,
        timeframe: state.timeframe,
        centerTab: state.centerTab,
        leftSidebarCollapsed: state.leftSidebarCollapsed,
        bottomPanelCollapsed: state.bottomPanelCollapsed,
        quickTradeCollapsed: state.quickTradeCollapsed,
        positionsPanelVisible: state.positionsPanelVisible,
      }),
      onRehydrateStorage: () => () => {
        queueMicrotask(() => {
          useWorkspaceStore.setState({ _hasHydrated: true });
        });
      },
    },
  ),
);
