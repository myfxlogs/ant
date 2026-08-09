import { useCallback, useState } from 'react';
import { Button } from 'antd';
import { useTranslation } from 'react-i18next';
import { RobotOutlined, BarChartOutlined } from '@ant-design/icons';
import { useWorkspaceStore } from '@/stores/workspaceStore';
import StrategyChat from '@/components/strategy/StrategyChat';
import BacktestPanel from '@/components/backtest/BacktestPanel';
import { useAIFix } from '@/components/backtest/useAIFix';
import { useWsAccount, useWsCode, useWsBacktest, useWsHistory, useWsAI, useWsTemplates } from '../../WorkspaceContext';

interface BtSummary {
  totalReturn?: number;
  maxDrawdown?: number;
  sharpeRatio?: number;
  winRate?: number;
  totalTrades: number;
}

interface RecentSummary {
  templateName: string;
  totalReturn: number;
  totalTrades: number;
  startedAt: string;
}

interface Props {
  activeTab: 'ai' | 'backtest';
  onTabChange: (tab: 'ai' | 'backtest') => void;
  onClose: () => void;
  btSummary?: BtSummary;
  recentSummaries: RecentSummary[];
}

export default function WorkspaceAIPanel({ activeTab, onTabChange, onClose, btSummary, recentSummaries }: Props) {
  const { t } = useTranslation();
  const aiPanelWidth = useWorkspaceStore(s => s.aiPanelWidth);
  const setAiPanelWidth = useWorkspaceStore(s => s.setAiPanelWidth);
  const account = useWsAccount();
  const code = useWsCode();
  const backtest = useWsBacktest();
  const templates = useWsTemplates();
  const history = useWsHistory();
  const ai = useWsAI();

  const aiFix = useAIFix({
    strategyId: code.strategyId,
    currentCode: code.code,
    onApplyCode: code.setCode,
    onRerunBacktest: () => backtest.runner.run({
      strategyCode: code.code,
      accountId: account.accountId,
      symbol: account.symbol,
      timeframe: account.timeframe,
      templateId: templates.selectedId || undefined,
      strategyId: code.strategyId,
    }),
  });

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

  const tabs: { key: 'ai' | 'backtest'; icon: React.ReactNode; label: string }[] = [
    { key: 'ai', icon: <RobotOutlined />, label: t('strategy.workspace.aiTab', { defaultValue: 'AI' }) },
    { key: 'backtest', icon: <BarChartOutlined />, label: t('strategy.workspace.backtestTab', { defaultValue: 'Backtest' }) },
  ];

  return (
    <>
      {/* Resize handle */}
      <div
        onMouseDown={handleAiResize}
        style={{
          width: 4, cursor: 'col-resize', flexShrink: 0,
          background: aiDragging ? '#58a6ff' : 'var(--ant-color-border)',
          transition: aiDragging ? 'none' : 'background 0.15s',
        }}
      />
      <div style={{ flex: '1 1 0', minWidth: 0, display: 'flex', flexDirection: 'column', borderLeft: '1px solid var(--ant-color-border)' }}>
        {/* Header: tabs + close */}
        <div style={{
          display: 'flex', alignItems: 'center', flexShrink: 0,
          borderBottom: '1px solid var(--ant-color-border)', height: 40,
          background: 'linear-gradient(180deg, #f0f5ff 0%, #e6f0ff 100%)',
        }}>
          {tabs.map(({ key, icon, label }) => (
            <div
              key={key}
              onClick={() => onTabChange(key)}
              style={{
                padding: '0 16px', height: '100%', display: 'flex', alignItems: 'center', gap: 6,
                cursor: 'pointer', fontSize: 13, fontWeight: 700,
                color: activeTab === key ? '#1677ff' : 'var(--ant-color-text-secondary)',
                borderBottom: activeTab === key ? '3px solid #1677ff' : '3px solid transparent',
                background: activeTab === key ? 'rgba(22, 119, 255, 0.06)' : 'transparent',
                transition: 'all 0.15s',
              }}
            >
              {icon} {label}
            </div>
          ))}
          <div style={{ flex: 1 }} />
          <Button size="small" type="text" onClick={onClose}
            style={{ fontSize: 14, padding: '0 6px', lineHeight: 1, marginRight: 4 }}>✕</Button>
        </div>

        {/* Content */}
        <div style={{ flex: '1 1 0', minHeight: 0, overflow: 'auto' }}>
          {activeTab === 'ai' && (
            <StrategyChat
              symbol={account.symbol}
              timeframe={account.timeframe}
              accountId={account.accountId}
              onApplyCode={c => { code.setCode(c); }}
              currentCode={code.code}
              lastBacktest={btSummary}
              recentBacktests={recentSummaries}
            />
          )}
          {activeTab === 'backtest' && (
            <BacktestPanel
              runner={backtest.runner}
              inputs={{
                strategyCode: code.code,
                accountId: account.accountId,
                symbol: account.symbol,
                timeframe: account.timeframe,
                templateId: templates.selectedId || undefined,
                strategyId: code.strategyId,
              }}
              onOpenHistory={(templateId?: string) => history.open(templateId)}
              onAIOptimize={() => ai.optimize()}
              code={code.code}
              onApplyTunedParams={code.setCode}
              strategyId={code.strategyId}
              onAIFix={aiFix.handleAIFix}
              aiFixing={aiFix.aiFixing}
            />
          )}
        </div>
      </div>
      {aiFix.diffModal}
    </>
  );
}
