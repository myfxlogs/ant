import StrategyChat from './StrategyChat';

interface Props {
  symbol?: string;
  timeframe?: string;
  sessionId?: string;
  accountId?: string;
  onApply: (code: string, previousCode?: string) => void;
}

export default function AICodePanel({ symbol, timeframe, sessionId, accountId, onApply }: Props) {
  return (
    <StrategyChat
      symbol={symbol}
      timeframe={timeframe}
      sessionId={sessionId}
      accountId={accountId}
      onApplyCode={(code) => onApply(code)}
    />
  );
}
