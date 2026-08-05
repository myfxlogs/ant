import { useCallback, useState } from 'react';
import { Button } from 'antd';
import { useTranslation } from 'react-i18next';
import { AI_ASSISTANT_KEY } from '@/gen/ant/v1/i18n/strategy_workspace_keys';
import { useWorkspaceStore } from '@/stores/workspaceStore';
import StrategyChat from '@/components/strategy/StrategyChat';
import { useWsAccount, useWsCode } from '../../WorkspaceContext';

interface BtSummary {
  totalReturn: number;
  maxDrawdown: number;
  sharpeRatio: number;
  winRate: number;
  totalTrades: number;
}

interface RecentSummary {
  templateName: string;
  totalReturn: number;
  totalTrades: number;
  startedAt: string;
}

interface Props {
  onClose: () => void;
  btSummary?: BtSummary;
  recentSummaries: RecentSummary[];
}

export default function WorkspaceAIPanel({ onClose, btSummary, recentSummaries }: Props) {
  const { t } = useTranslation();
  const aiPanelWidth = useWorkspaceStore(s => s.aiPanelWidth);
  const setAiPanelWidth = useWorkspaceStore(s => s.setAiPanelWidth);
  const account = useWsAccount();
  const code = useWsCode();

  const [aiDragging, setAiDragging] = useState(false);
  const handleAiResize = useCallback((e: React.MouseEvent) => {
    e.preventDefault();
    setAiDragging(true);
    const startX = e.clientX;
    const startW = aiPanelWidth;
    const onMove = (ev: MouseEvent) => {
      const delta = startX - ev.clientX;
      setAiPanelWidth(Math.max(300, Math.min(600, startW + delta)));
    };
    const onUp = () => {
      setAiDragging(false);
      document.removeEventListener('mousemove', onMove);
      document.removeEventListener('mouseup', onUp);
    };
    document.addEventListener('mousemove', onMove);
    document.addEventListener('mouseup', onUp);
  }, [aiPanelWidth, setAiPanelWidth]);

  return (
    <>
      <div
        onMouseDown={handleAiResize}
        style={{
          width: 4, cursor: 'col-resize', flexShrink: 0,
          background: aiDragging ? '#58a6ff' : 'var(--ant-color-border)',
          transition: aiDragging ? 'none' : 'background 0.15s',
        }}
      />
      <div style={{ width: aiPanelWidth, flexShrink: 0, display: 'flex', flexDirection: 'column', borderLeft: '1px solid var(--ant-color-border)' }}>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '4px 10px', borderBottom: '1px solid var(--ant-color-border)', flexShrink: 0 }}>
          <span style={{ fontSize: 12, fontWeight: 600 }}>🤖 {t(AI_ASSISTANT_KEY)}</span>
          <Button size="small" type="text" onClick={onClose}
            style={{ fontSize: 14, padding: '0 4px', lineHeight: 1 }}>✕</Button>
        </div>
        <div style={{ flex: '1 1 0', minHeight: 0 }}>
          <StrategyChat
            symbol={account.symbol}
            timeframe={account.timeframe}
            accountId={account.accountId}
            onApplyCode={c => { code.setCode(c); }}
            currentCode={code.code}
            lastBacktest={btSummary}
            recentBacktests={recentSummaries}
          />
        </div>
      </div>
    </>
  );
}
