import { useState } from 'react';
import { Drawer, Button } from 'antd';
import { LayoutOutlined } from '@ant-design/icons';
import ChartBottomPanel from '@/components/chart/ChartBottomPanel';
import QuickTradeSidePanel from './QuickTradeSidePanel';
import type { AccountMeta } from '@/components/chart/QuickTradePanel';
import type { QuickTradePosition, RecentTrade } from '@/pages/strategy/hooks/useStrategyWorkspaceState';

interface Props {
  isMobile: boolean;
  collapsed: boolean;
  onToggleCollapsed: () => void;
  positions: QuickTradePosition[];
  recentTrades: RecentTrade[];
  onClosePosition: (ticket: number, volume?: number) => void;
  panelHeight?: number;
  onResizeStart?: (e: React.MouseEvent) => void;
  dragging?: boolean;
  accountId: string;
  symbol: string;
  accountMeta?: AccountMeta | null;
  qtPositions: QuickTradePosition[];
  quickTradeCollapsed: boolean;
  onToggleQuickTrade: () => void;
}

export default function BottomPanelSection({
  isMobile, collapsed, onToggleCollapsed,
  positions, recentTrades, onClosePosition,
  panelHeight, onResizeStart, dragging,
  accountId, symbol, accountMeta, qtPositions,
  quickTradeCollapsed, onToggleQuickTrade,
}: Props) {
  const [mobileOpen, setMobileOpen] = useState(false);

  if (isMobile) {
    return (
      <>
        <Button
          icon={<LayoutOutlined />}
          onClick={() => setMobileOpen(true)}
          style={{ position: 'fixed', bottom: 16, right: 16, zIndex: 1000 }}
          type="primary"
          shape="circle"
        />
        <Drawer
          open={mobileOpen}
          onClose={() => setMobileOpen(false)}
          placement="bottom"
          height="60vh"
          title={undefined}
          styles={{ body: { padding: 0 } }}
        >
          <ChartBottomPanel
            positions={positions}
            recentTrades={recentTrades}
            onClosePosition={onClosePosition}
            collapsed={false}
            onToggleCollapsed={() => setMobileOpen(false)}
          />
        </Drawer>
      </>
    );
  }

  if (collapsed) {
    return (
      <ChartBottomPanel
        positions={positions}
        recentTrades={recentTrades}
        onClosePosition={onClosePosition}
        collapsed={true}
        onToggleCollapsed={onToggleCollapsed}
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
          panelHeight={panelHeight}
          onResizeStart={onResizeStart}
          dragging={dragging}
        />
      </div>
      {symbol && !quickTradeCollapsed && (
        <QuickTradeSidePanel
          accountId={accountId}
          symbol={symbol}
          accountMeta={accountMeta}
          allPositions={positions}
          positions={qtPositions}
          recentTrades={recentTrades}
          onClosePosition={onClosePosition}
          onCollapse={onToggleQuickTrade}
        />
      )}
      {symbol && quickTradeCollapsed && (
        <div onClick={onToggleQuickTrade} role="button" tabIndex={0}
          onKeyUp={e => e.key === 'Enter' && onToggleQuickTrade()}
          style={{
            width: 48, flexShrink: 0, cursor: 'pointer',
            borderLeft: '1px solid var(--ant-color-border)',
            background: 'var(--ant-color-bg-elevated)',
            display: 'flex', alignItems: 'center', justifyContent: 'center',
            maxHeight: 160, fontSize: 20,
          }} title="Expand Quick Trade">
          ⚡
        </div>
      )}
    </div>
  );
}
