/**
 * workspaceStore — persists workspace UI state across sessions.
 * Uses Zustand persist middleware with slice-creator pattern (ADR-0027 Phase C).
 * Only persists layout/navigation state; never persists code or results.
 */
import { create } from 'zustand';
import { persist, createJSONStorage } from 'zustand/middleware';

export type CenterTab = 'chat' | 'code' | 'import' | 'backtest' | 'strategies';

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
  bottomPanelHeight: number;
  bottomPanelUserResized: boolean;
  quickTradeCollapsed: boolean;
  positionsPanelVisible: boolean;
  aiPanelOpen: boolean;
  aiPanelWidth: number;
  setCenterTab: (v: CenterTab) => void;
  setLeftSidebarCollapsed: (v: boolean) => void;
  setBottomPanelCollapsed: (v: boolean) => void;
  setBottomPanelHeight: (v: number) => void;
  setBottomPanelUserResized: (v: boolean) => void;
  setQuickTradeCollapsed: (v: boolean) => void;
  setPositionsPanelVisible: (v: boolean) => void;
  setAiPanelOpen: (v: boolean) => void;
  setAiPanelWidth: (v: number) => void;
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
    centerTab: 'code',
    leftSidebarCollapsed: true,
    bottomPanelCollapsed: false,
    bottomPanelHeight: 160,
    bottomPanelUserResized: false,
    quickTradeCollapsed: true,
    positionsPanelVisible: false,
    aiPanelOpen: false,
    aiPanelWidth: 380,
    setCenterTab: (v) => set({ centerTab: v }),
    setLeftSidebarCollapsed: (v) => set({ leftSidebarCollapsed: v }),
    setBottomPanelCollapsed: (v) => set({ bottomPanelCollapsed: v }),
    setBottomPanelHeight: (v) => set({ bottomPanelHeight: v }),
    setBottomPanelUserResized: (v) => set({ bottomPanelUserResized: v }),
    setQuickTradeCollapsed: (v) => set({ quickTradeCollapsed: v }),
    setPositionsPanelVisible: (v) => set({ positionsPanelVisible: v }),
    setAiPanelOpen: (v) => set({ aiPanelOpen: v }),
    setAiPanelWidth: (v) => set({ aiPanelWidth: v }),
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
      name: 'ant-workspace-v8',
      storage: createJSONStorage(() => localStorage),
      partialize: (state) => ({
        accountId: state.accountId,
        symbol: state.symbol,
        timeframe: state.timeframe,
        centerTab: state.centerTab,
        leftSidebarCollapsed: state.leftSidebarCollapsed,
        bottomPanelCollapsed: state.bottomPanelCollapsed,
        bottomPanelHeight: state.bottomPanelHeight,
        bottomPanelUserResized: state.bottomPanelUserResized,
        quickTradeCollapsed: state.quickTradeCollapsed,
        positionsPanelVisible: state.positionsPanelVisible,
        aiPanelOpen: state.aiPanelOpen,
        aiPanelWidth: state.aiPanelWidth,
      }),
      onRehydrateStorage: () => () => {
        queueMicrotask(() => {
          useWorkspaceStore.setState({ _hasHydrated: true });
        });
      },
    },
  ),
);
