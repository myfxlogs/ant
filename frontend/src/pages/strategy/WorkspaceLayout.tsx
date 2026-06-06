import { lazy, Suspense } from 'react';
import QuickTradePanel from '@/components/chart/QuickTradePanel';

const SaveTemplateModal = lazy(() => import('@/components/strategy/SaveTemplateModal'));

export const CODE_PANEL_WIDTH = 750;
export const POSITIONS_PANEL_WIDTH = 520;
export const C = { border: "#e8e8e8", bg: "#f8fafc", bgAlt: "#f1f5f9", muted: "#8c8c8c" };

interface QuickTradeSectionProps {
  visible: boolean;
  ws: any;
}

export function QuickTradeSection({ visible, ws }: QuickTradeSectionProps) {
  if (!visible) return null;
  return (
    <div style={{
      width: 320, minWidth: 240, flexShrink: 0,
      borderLeft: '1px solid #e8e8e8', background: C.bg,
      display: 'flex', flexDirection: 'column', overflow: 'hidden',
    }}>
      <div style={{
        padding: '10px 14px', display: 'flex', justifyContent: 'space-between', alignItems: 'center',
        background: 'linear-gradient(180deg, #ffffff 0%, #f1f5f9 100%)',
        borderBottom: '1px solid #e8e8e8', flexShrink: 0,
      }}>
        <span style={{ fontSize: 13, fontWeight: 700, color: '#0f172a' }}>⚡ Quick Trade</span>
        <span onClick={() => ws.layout.setQuickTradeVisible(false)} role="button" tabIndex={0}
          onKeyUp={(e) => e.key === 'Enter' && ws.layout.setQuickTradeVisible(false)}
          style={{ cursor: 'pointer', color: '#94a3b8', fontSize: 16, lineHeight: 1 }}>✕</span>
      </div>
      <div style={{ flex: 1, overflowY: 'auto', padding: 12 }}>
        <QuickTradePanel
          accountId={ws.account.accountId} symbol={ws.account.symbol}
          accountMeta={ws.account.selectedAccountMeta}
          allPositions={ws.quickTrade.allPositions}
          positions={ws.quickTrade.qtPositions}
          recentTrades={ws.quickTrade.qtRecentTrades}
          onClosePosition={ws.quickTrade.handleClosePosition}
          onToggleAllPositions={() => ws.layout.setPositionsPanelVisible(!ws.layout.positionsPanelVisible)}
        />
      </div>
    </div>
  );
}

interface SaveTemplateWrapperProps {
  ws: any;
}

export function SaveTemplateWrapper({ ws }: SaveTemplateWrapperProps) {
  return (
    <Suspense fallback={null}>
      <SaveTemplateModal open={ws.code.saveModalOpen} confirmLoading={ws.code.saveLoading} form={ws.code.saveForm}
        onCancel={() => ws.code.setSaveModalOpen(false)} onOk={ws.code.handleSaveModalOk} />
    </Suspense>
  );
}
