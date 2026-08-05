import ChartBottomPanel from '@/components/chart/ChartBottomPanel';
import QuickTradeSidePanel from './QuickTradeSidePanel';
import type { QuickTradePosition, RecentTrade } from '@/pages/strategy/hooks/useStrategyWorkspaceState';

interface Props {
  isMobile: boolean;
  collapsed: boolean;
  onToggleCollapsed: () => void;
  positions: QuickTradePosition[];
  recentTrades: RecentTrade[];
  onClosePosition: (ticket: number, volume?: number) => void;
  backtestMetrics?: { totalReturn?: number; maxDrawdown?: number; sharpeRatio?: number; winRate?: number; totalTrades?: number } | null;
  backtestStatus?: string;
  onOpenAdvancedBacktest?: () => void;
  panelHeight?: number;
  onResizeStart?: (e: React.MouseEvent) => void;
  dragging?: boolean;
  accountId: string;
  symbol: string;
  accountMeta?: { leverage?: number; balance?: number; equity?: number };
  qtPositions: QuickTradePosition[];
}

export default function BottomPanelSection({
  isMobile, collapsed, onToggleCollapsed,
  positions, recentTrades, onClosePosition,
  backtestMetrics, backtestStatus, onOpenAdvancedBacktest,
  panelHeight, onResizeStart, dragging,
  accountId, symbol, accountMeta, qtPositions,
}: Props) {
  if (isMobile) return null;

  if (collapsed) {
    return (
      <ChartBottomPanel
        positions={positions}
        recentTrades={recentTrades}
        onClosePosition={onClosePosition}
        collapsed={true}
        onToggleCollapsed={onToggleCollapsed}
        backtestMetrics={backtestMetrics}
        backtestStatus={backtestStatus}
        onOpenAdvancedBacktest={onOpenAdvancedBacktest}
      />
    );
  }

  return (
    <div style={{ flexShrink: 0, display: 'flex', borderTop: '1px solid var(--ant-color-border)' }}>
      <div style={{ flex: '1 1 0', minWidth: 0 }}>
        <ChartBottomPanel
          positions={positions}
          recentTrades={recentTrades}
          onClosePosition={onClosePosition}
          collapsed={false}
          onToggleCollapsed={onToggleCollapsed}
          backtestMetrics={backtestMetrics}
          backtestStatus={backtestStatus}
          onOpenAdvancedBacktest={onOpenAdvancedBacktest}
          panelHeight={panelHeight}
          onResizeStart={onResizeStart}
          dragging={dragging}
        />
      </div>
      {symbol && (
        <QuickTradeSidePanel
          accountId={accountId}
          symbol={symbol}
          accountMeta={accountMeta}
          allPositions={positions}
          positions={qtPositions}
          recentTrades={recentTrades}
          onClosePosition={onClosePosition}
        />
      )}
    </div>
  );
}
