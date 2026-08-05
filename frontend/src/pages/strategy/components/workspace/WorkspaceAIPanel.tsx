import { useCallback, useState } from 'react';
import { Button, Spin, Progress } from 'antd';
import { useTranslation } from 'react-i18next';
import { RobotOutlined, BarChartOutlined } from '@ant-design/icons';
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
  activeTab: 'ai' | 'backtest';
  onTabChange: (tab: 'ai' | 'backtest') => void;
  onClose: () => void;
  btSummary?: BtSummary;
  recentSummaries: RecentSummary[];
  backtestStatus?: string;
  backtestMetrics?: { totalReturn?: number; maxDrawdown?: number; sharpeRatio?: number; winRate?: number; totalTrades?: number } | null;
  onRunBacktest?: () => void;
}

function fmtPct(v: number | undefined): string {
  if (v == null) return '—';
  return `${v >= 0 ? '+' : ''}${v.toFixed(1)}%`;
}

export default function WorkspaceAIPanel({ activeTab, onTabChange, onClose, btSummary, recentSummaries, backtestStatus, backtestMetrics, onRunBacktest }: Props) {
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
      <div style={{ width: aiPanelWidth, flexShrink: 0, display: 'flex', flexDirection: 'column', borderLeft: '1px solid var(--ant-color-border)' }}>
        {/* Header: tabs + close */}
        <div style={{
          display: 'flex', alignItems: 'center', flexShrink: 0,
          borderBottom: '1px solid var(--ant-color-border)', height: 34,
        }}>
          {tabs.map(({ key, icon, label }) => (
            <div
              key={key}
              onClick={() => onTabChange(key)}
              style={{
                padding: '0 14px', height: '100%', display: 'flex', alignItems: 'center', gap: 5,
                cursor: 'pointer', fontSize: 12, fontWeight: 600,
                color: activeTab === key ? '#58a6ff' : 'var(--ant-color-text-secondary)',
                borderBottom: activeTab === key ? '2px solid #58a6ff' : '2px solid transparent',
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
            <div style={{ padding: 16 }}>
              {/* Running state */}
              {backtestStatus === 'running' && (
                <div style={{ textAlign: 'center', padding: '40px 0' }}>
                  <Spin size="large" />
                  <div style={{ marginTop: 16, fontSize: 14, fontWeight: 600 }}>
                    {t('strategy.workspace.backtestRunning', { defaultValue: 'Backtest Running...' })}
                  </div>
                  <Progress percent={99} status="active" showInfo={false} style={{ marginTop: 16 }} />
                </div>
              )}
              {/* Completed state */}
              {backtestStatus === 'completed' && backtestMetrics && (
                <>
                  {/* Metrics cards */}
                  <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 8, marginBottom: 12 }}>
                    {[
                      { label: t('strategy.workspace.totalReturn', { defaultValue: 'Total Return' }), value: fmtPct(backtestMetrics.totalReturn), color: (backtestMetrics.totalReturn ?? 0) >= 0 ? '#3fb950' : '#f85149' },
                      { label: t('strategy.workspace.maxDrawdown', { defaultValue: 'Max Drawdown' }), value: fmtPct(backtestMetrics.maxDrawdown), color: '#f85149' },
                      { label: t('strategy.workspace.sharpeRatio', { defaultValue: 'Sharpe' }), value: backtestMetrics.sharpeRatio?.toFixed(2) ?? '—', color: undefined },
                      { label: t('strategy.workspace.winRate', { defaultValue: 'Win Rate' }), value: backtestMetrics.winRate != null ? `${backtestMetrics.winRate.toFixed(0)}%` : '—', color: undefined },
                      { label: t('strategy.workspace.totalTrades', { defaultValue: 'Total Trades' }), value: String(backtestMetrics.totalTrades ?? '—'), color: undefined },
                    ].map((m, i) => (
                      <div key={i} style={{ background: 'var(--ant-color-fill-quaternary)', borderRadius: 6, padding: '10px 12px', textAlign: 'center' }}>
                        <div style={{ fontSize: 16, fontWeight: 700, color: m.color }}>{m.value}</div>
                        <div style={{ fontSize: 10, color: 'var(--ant-color-text-tertiary)', marginTop: 2 }}>{m.label}</div>
                      </div>
                    ))}
                  </div>
                  {/* Action buttons */}
                  <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
                    {onRunBacktest && (
                      <Button block onClick={onRunBacktest}>
                        🔄 {t('strategy.workspace.reRun', { defaultValue: 'Re-run Backtest' })}
                      </Button>
                    )}
                  </div>
                </>
              )}
              {/* Idle / error / no-data */}
              {backtestStatus !== 'running' && backtestStatus !== 'completed' && (
                <div style={{ textAlign: 'center', padding: '40px 0', color: 'var(--ant-color-text-secondary)', fontSize: 13 }}>
                  {backtestStatus === 'error'
                    ? t('strategy.workspace.backtestError', { defaultValue: 'Backtest failed. Check the code and try again.' })
                    : t('strategy.workspace.noBacktestYet', { defaultValue: 'Run a backtest to see results here.' })}
                </div>
              )}
            </div>
          )}
        </div>
      </div>
    </>
  );
}
