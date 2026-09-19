// MobileStrategyChat — 移动端视图下的 StrategyChat 接线包装。
// 自 WorkspaceCenterColumn 抽出（LOWPRI-SWEEP-2 CQ-12 max-lines）。
import StrategyChat from '@/components/strategy/StrategyChat';

interface Props {
  symbol: string;
  timeframe: string;
  accountId: string;
  currentCode: string;
  lastBacktest?: { totalReturn: string; maxDrawdown: string; sharpeRatio: string; winRate: string; totalTrades: number };
  recentBacktests: Array<{ templateName: string; totalReturn: number; totalTrades: number; startedAt: string }>;
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
