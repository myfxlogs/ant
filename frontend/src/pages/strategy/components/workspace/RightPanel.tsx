import StrategyChat from '@/components/strategy/StrategyChat';

interface Props {
  symbol?: string;
  timeframe?: string;
  sessionId?: string;
  accountId?: string;
  onApplyCode: (code: string) => void;
  onValidateResult?: (result: import('@/client/codeAssist').ValidateExtendedResult) => void;
}

export default function RightPanel(props: Props) {
  return (
    <div style={{
      width: 380, minWidth: 380, flexShrink: 0,
      background: 'var(--ant-color-bg-container)',
      borderLeft: '1px solid var(--ant-color-border)',
      display: 'flex', flexDirection: 'column', overflow: 'hidden',
    }}>
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
