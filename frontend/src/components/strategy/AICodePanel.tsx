import type { ValidateExtendedResult } from '@/client/codeAssist';
import StrategyChat from './StrategyChat';

interface Props {
  symbol?: string;
  timeframe?: string;
  sessionId?: string;
  accountId?: string;
  onApply: (code: string, previousCode?: string) => void;
  onValidateResult?: (result: ValidateExtendedResult) => void;
}

export default function AICodePanel({ symbol, timeframe, sessionId, accountId, onApply, onValidateResult }: Props) {
  return (
    <StrategyChat
      symbol={symbol}
      timeframe={timeframe}
      sessionId={sessionId}
      accountId={accountId}
      onApplyCode={(code) => onApply(code)}
      onValidateResult={onValidateResult}
    />
  );
}
