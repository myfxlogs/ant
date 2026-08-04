import { useWorkspaceStore } from '@/stores/workspaceStore';

export function useLayoutSlice() {
  const leftSidebarCollapsed = useWorkspaceStore(s => s.leftSidebarCollapsed);
  const setLeftSidebarCollapsed = useWorkspaceStore(s => s.setLeftSidebarCollapsed);
  const bottomPanelCollapsed = useWorkspaceStore(s => s.bottomPanelCollapsed);
  const setBottomPanelCollapsed = useWorkspaceStore(s => s.setBottomPanelCollapsed);
  const quickTradeCollapsed = useWorkspaceStore(s => s.quickTradeCollapsed);
  const setQuickTradeCollapsed = useWorkspaceStore(s => s.setQuickTradeCollapsed);
  const positionsPanelVisible = useWorkspaceStore(s => s.positionsPanelVisible);
  const setPositionsPanelVisible = useWorkspaceStore(s => s.setPositionsPanelVisible);

  return {
    leftSidebarCollapsed,
    setLeftSidebarCollapsed,
    bottomPanelCollapsed,
    setBottomPanelCollapsed,
    quickTradeCollapsed,
    setQuickTradeCollapsed,
    positionsPanelVisible,
    setPositionsPanelVisible,
  };
}
