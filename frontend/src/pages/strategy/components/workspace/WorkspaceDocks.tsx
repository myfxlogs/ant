// WorkspaceDocks — 主编辑器视图右侧的停靠面板（AI / 回测）。
// 自 WorkspaceCenterColumn 抽出（LOWPRI-SWEEP-2 CQ-12 max-lines）。
import WorkspaceAIPanel from './WorkspaceAIPanel';

interface Props {
  dock: 'ai' | 'backtest';
  btSummary?: { totalReturn: string; maxDrawdown: string; sharpeRatio: string; winRate: string; totalTrades: number };
  recentSummaries: Array<{ templateName: string; totalReturn: number; totalTrades: number; startedAt: string }>;
  onSwitchToBacktest: () => void;
  onClose: () => void;
}

export default function WorkspaceDocks({ dock, btSummary, recentSummaries, onSwitchToBacktest, onClose }: Props) {
  const panel = (activeTab: 'ai' | 'backtest', onTabChange: () => void) => (
    <div style={{ width: 420, flexShrink: 0, borderLeft: '1px solid var(--ant-color-border)', display: 'flex', flexDirection: 'column', minHeight: 0 }}>
      <WorkspaceAIPanel
        activeTab={activeTab}
        onTabChange={onTabChange}
        onClose={onClose}
        btSummary={btSummary}
        recentSummaries={recentSummaries}
      />
    </div>
  );

  if (dock === 'ai') {
    return panel('ai', onSwitchToBacktest);
  }
  return panel('backtest', onClose);
}
