// MobileStrategyChat — 移动端视图下的 StrategyChat 接线包装。
// 自 WorkspaceCenterColumn 抽出（LOWPRI-SWEEP-2 CQ-12 max-lines）。
import StrategyChat from '@/components/strategy/StrategyChat';
import type { BacktestSummary } from '@/client/agentGen';

interface Props {
  symbol: string;
  timeframe: string;
  accountId: string;
  currentCode: string;
  lastBacktest?: BacktestSummary;
  recentBacktests: BacktestSummary[];
  onApplyCode: (code: string) => void;
}

export default function MobileStrategyChat({ symbol, timeframe, accountId, currentCode, lastBacktest, recentBacktests, onApplyCode }: Props) {
  return (
    <StrategyChat
      symbol={symbol}
      timeframe={timeframe}
      accountId={accountId}
      onApplyCode={onApplyCode}
      currentCode={currentCode}
      lastBacktest={lastBacktest}
      recentBacktests={recentBacktests}
    />
  );
}
