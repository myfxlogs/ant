import StrategyChat from './StrategyChat';

interface Props {
  symbol?: string;
  timeframe?: string;
  sessionId?: string;
  onApply: (code: string, previousCode?: string) => void;
}

export default function AICodePanel({ symbol, timeframe, sessionId, onApply }: Props) {
  return (
    <StrategyChat
      symbol={symbol}
      timeframe={timeframe}
      sessionId={sessionId}
      onApplyCode={(code) => onApply(code)}
    />
  );
}
