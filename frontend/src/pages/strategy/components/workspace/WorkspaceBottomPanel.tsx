// WorkspaceBottomPanel — BottomPanelSection 的 workspace 状态接线包装。
// 自 WorkspaceCenterColumn 抽出（LOWPRI-SWEEP-2 CQ-12 max-lines）。
import BottomPanelSection from './BottomPanelSection';

interface Props {
  isMobile: boolean;
  accountId: string;
  symbol: string;
  bottomPanelCollapsed: boolean;
  onToggleBottomPanelCollapsed: () => void;
  bottomPanelUserResized: boolean;
  bottomPanelHeight?: number;
  onResizeStart: (e: React.MouseEvent) => void;
  dragging: boolean;
  allPositions: unknown[];
  qtRecentTrades: unknown[];
  handleClosePosition: (args: unknown) => void;
  qtPositions: unknown[];
  quickTradeCollapsed: boolean;
  onToggleQuickTrade: () => void;
  accountMeta?: unknown;
}

export default function WorkspaceBottomPanel(p: Props) {
  return (
    <BottomPanelSection
      isMobile={p.isMobile}
      collapsed={p.bottomPanelCollapsed}
      onToggleCollapsed={p.onToggleBottomPanelCollapsed}
      positions={p.allPositions}
      recentTrades={p.qtRecentTrades}
      onClosePosition={p.handleClosePosition}
      panelHeight={p.bottomPanelUserResized ? p.bottomPanelHeight : undefined}
      onResizeStart={p.onResizeStart}
      dragging={p.dragging}
      accountId={p.accountId}
      symbol={p.symbol}
      accountMeta={p.accountMeta ?? undefined}
      qtPositions={p.qtPositions}
      quickTradeCollapsed={p.quickTradeCollapsed}
      onToggleQuickTrade={p.onToggleQuickTrade}
      backtestContent={p.isMobile ? null : undefined}
    />
  );
}
