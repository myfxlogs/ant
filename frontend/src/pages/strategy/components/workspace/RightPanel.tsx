import StrategyChat from '@/components/strategy/StrategyChat';
import QuickTradePanel from '@/components/chart/QuickTradePanel';

export interface QuickTradeState {
  accountId: string;
  symbol: string;
  accountMeta?: { brokerCompany: string; brokerServer: string; mtType: 'MT4' | 'MT5'; leverage: number } | null;
  collapsed: boolean;
  onToggle: () => void;
  onClosePosition?: (ticket: number, volume?: number) => void;
}

interface Props {
  // Chat
  symbol?: string;
  timeframe?: string;
  sessionId?: string;
  accountId?: string;
  onApplyCode: (code: string) => void;
  onValidateResult?: (result: import('@/client/codeAssist').ValidateExtendedResult) => void;

  // QuickTrade
  quickTrade?: QuickTradeState;
}

export default function RightPanel(props: Props) {
  return (
    <div style={{
      width: 380, minWidth: 380, flexShrink: 0,
      background: 'var(--ant-color-bg-container)',
      borderLeft: '1px solid var(--ant-color-border)',
      display: 'flex', flexDirection: 'column', overflow: 'hidden',
    }}>
      {/* Quick Trade — compact form at top */}
      {props.quickTrade && props.quickTrade.symbol && (
        <div style={{ flexShrink: 0, borderBottom: '1px solid var(--ant-color-border)' }}>
          <div
            onClick={props.quickTrade.onToggle}
            style={{
              padding: '6px 12px', cursor: 'pointer', userSelect: 'none',
              display: 'flex', alignItems: 'center', justifyContent: 'space-between',
              fontSize: 12, fontWeight: 700, color: 'var(--ant-color-text)',
              background: 'var(--ant-color-bg-layout)',
            }}
          >
            <span>⚡ Quick Trade</span>
            <span style={{ fontSize: 10, color: 'var(--ant-color-text-tertiary)' }}>
              {props.quickTrade.collapsed ? '▼' : '▲'}
            </span>
          </div>
          {!props.quickTrade.collapsed && (
            <div style={{ padding: 8, maxHeight: 300, overflowY: 'auto' }}>
              <QuickTradePanel
                accountId={props.quickTrade.accountId}
                symbol={props.quickTrade.symbol}
                accountMeta={props.quickTrade.accountMeta}
                allPositions={[]}
                positions={[]}
                recentTrades={[]}
                onClosePosition={props.quickTrade.onClosePosition}
              />
            </div>
          )}
        </div>
      )}

      {/* Chat — always visible, no tab switching */}
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
        <StrategyChat
          symbol={props.symbol}
          timeframe={props.timeframe}
          sessionId={props.sessionId}
          accountId={props.accountId}
          onApplyCode={props.onApplyCode}
          onValidateResult={props.onValidateResult}
        />
      </div>
    </div>
  );
}
