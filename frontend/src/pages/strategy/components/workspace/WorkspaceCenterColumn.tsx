import { useCallback, useEffect, useRef, useState } from 'react';
import { Button } from 'antd';
import { PlayCircleOutlined, ImportOutlined, RobotOutlined, HistoryOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { useWorkspaceStore } from '@/stores/workspaceStore';
import StrategyCodeEditor from '@/components/strategy/StrategyCodeEditor';
import StrategyChat from '@/components/strategy/StrategyChat';
import ImportEAPanel from '../editor/ImportEAPanel';
import WorkspaceSidebar from './WorkspaceSidebar';
import BottomPanelSection from './BottomPanelSection';
import WorkspaceCenterTabBar from './WorkspaceCenterTabBar';
import WorkspaceAIPanel from './WorkspaceAIPanel';
import WorkspaceCenterModals from './WorkspaceCenterModals';
import { useWsAccount, useWsCode, useWsTemplates, useWsBacktest, useWsQuickTrade, useWsLayout, useWsHistory, useWsAI } from '../../WorkspaceContext';

interface Props {
  isMobile?: boolean;
  btModalOpen: boolean;
  setBtModalOpen: (v: boolean) => void;
  setIndicatorDrawerOpen: (v: boolean) => void;
  onShowVersionHistory?: () => void;
}

export default function WorkspaceCenterColumn({ isMobile = false, setBtModalOpen, setIndicatorDrawerOpen, onShowVersionHistory }: Props) {
  const { t } = useTranslation();
  const centerTab = useWorkspaceStore(s => s.centerTab);
  const setCenterTab = useWorkspaceStore(s => s.setCenterTab);

  const account = useWsAccount();
  const code = useWsCode();
  const templates = useWsTemplates();
  const backtest = useWsBacktest();
  const quickTrade = useWsQuickTrade();
  const layout = useWsLayout();
  const history = useWsHistory();
  const ai = useWsAI();

  // Right panel tab: 'ai' | 'backtest' | null
  const [rightPanelTab, setRightPanelTab] = useState<'ai' | 'backtest' | null>(null);

  // Auto-expand bottom panel + switch right panel to backtest when running/completed
  const prevBtStatusRef = useRef(backtest.status);
  useEffect(() => {
    if (backtest.status === 'running' && prevBtStatusRef.current !== 'running') {
      setRightPanelTab('backtest');
    }
    if (backtest.status === 'completed' && prevBtStatusRef.current === 'running') {
      layout.setBottomPanelCollapsed(false);
      if (layout.bottomPanelHeight < 250) layout.setBottomPanelHeight(300);
    }
    prevBtStatusRef.current = backtest.status;
  }, [backtest.status, layout]);

  // Desktop: if centerTab is 'chat', open AI panel
  useEffect(() => {
    if (!isMobile && centerTab === 'chat') {
      setRightPanelTab('ai'); setCenterTab('code');
    }
  }, [isMobile, centerTab, setCenterTab]);

  const leftSidebarCollapsed = useWorkspaceStore(s => s.leftSidebarCollapsed);
  const setLeftSidebarCollapsed = useWorkspaceStore(s => s.setLeftSidebarCollapsed);
  const [sidebarDrawerOpen, setSidebarDrawerOpen] = useState(false);

  const [bpDragging, setBpDragging] = useState(false);
  const handleBpResize = useCallback((e: React.MouseEvent) => {
    e.preventDefault();
    setBpDragging(true);
    layout.setBottomPanelUserResized(true);
    const startY = e.clientY;
    const startH = layout.bottomPanelHeight;
    const onMove = (ev: MouseEvent) => {
      const delta = startY - ev.clientY;
      layout.setBottomPanelHeight(Math.max(80, Math.min(500, startH + delta)));
    };
    const onUp = () => {
      setBpDragging(false);
      document.removeEventListener('mousemove', onMove);
      document.removeEventListener('mouseup', onUp);
    };
    document.addEventListener('mousemove', onMove);
    document.addEventListener('mouseup', onUp);
  }, [layout]);

  const restoreRef = useRef(false);
  useEffect(() => {
    if (restoreRef.current) return;
    if (!account.accountId) return;
    if (backtest.runner.status !== 'idle') return;
    restoreRef.current = true;
    backtest.runner.restoreLastRun(account.accountId, templates.selectedId || undefined);
  }, [account.accountId, backtest.runner, templates.selectedId]);

  const [btDrawerOpen, setBtDrawerOpen] = useState(false);
  const [importMode, setImportMode] = useState(false);

  const btSummary = backtest.metrics?.totalTrades != null
    ? { totalReturn: backtest.metrics.totalReturn, maxDrawdown: backtest.metrics.maxDrawdown, sharpeRatio: backtest.metrics.sharpeRatio, winRate: backtest.metrics.winRate, totalTrades: backtest.metrics.totalTrades }
    : undefined;
  const recentSummaries = (history.runs as Array<{ templateName?: string; totalReturn?: number; totalTrades?: number; startedAt?: string }>)
    ?.slice(0, 10).map(r => ({ templateName: r.templateName || '', totalReturn: r.totalReturn ?? 0, totalTrades: r.totalTrades ?? 0, startedAt: r.startedAt || '' })) || [];

  useEffect(() => {
    if (centerTab === 'import' || centerTab === 'strategies' || centerTab === 'backtest') {
      setCenterTab('code');
    }
  }, [centerTab, setCenterTab]);

  return (
    <div data-tour="code-editor" style={{ flex: '1 1 0', minWidth: 0, position: 'relative', overflow: 'hidden', display: 'flex', flexDirection: 'column' }}>
      <WorkspaceCenterTabBar
        isMobile={isMobile}
        setBtModalOpen={setBtModalOpen}
        setIndicatorDrawerOpen={setIndicatorDrawerOpen}
        onShowVersionHistory={onShowVersionHistory}
        onMobileSidebarToggle={() => setSidebarDrawerOpen(true)}
        onToggleAI={() => setRightPanelTab(prev => prev === 'ai' ? null : 'ai')}
        aiActive={rightPanelTab === 'ai'}
      />

      <div style={{ flex: '1 1 0', minHeight: 0, display: 'flex', flexDirection: 'row' }}>
        {!isMobile && (
          <WorkspaceSidebar
            templates={templates.list}
            loading={templates.loading}
            selectedId={templates.selectedId || ''}
            onSelect={(id) => templates.onSelect(id)}
            backtestRuns={(history.runs as Array<{ id: string; startedAt?: string; totalReturn?: number; totalTrades?: number; templateName?: string; templateId?: string }>) || []}
            runsLoading={history.loading}
            onOpenHistory={(tid) => history.open(tid)}
            onImport={() => setImportMode(true)}
            onNew={() => { templates.onSelect(''); }}
            collapsed={leftSidebarCollapsed}
            onToggle={() => setLeftSidebarCollapsed(!leftSidebarCollapsed)}
          />
        )}

        <div style={{ flex: '1 1 0', minWidth: 0, display: 'flex', flexDirection: 'column' }}>
          {isMobile && (
            <div style={{ flex: '1 1 0', minHeight: 0, display: centerTab === 'chat' ? 'flex' : 'none', flexDirection: 'column' }}>
              <StrategyChat
                symbol={account.symbol}
                timeframe={account.timeframe}
                accountId={account.accountId}
                onApplyCode={c => { code.setCode(c); setCenterTab('code'); }}
                currentCode={code.code}
                lastBacktest={btSummary}
                recentBacktests={recentSummaries}
              />
            </div>
          )}

          <div style={{ flex: '1 1 0', minHeight: 0, display: centerTab === 'code' ? 'flex' : 'none', flexDirection: 'row' }}>
            <div style={{ flex: '1 1 0', minWidth: 0, display: 'flex', flexDirection: 'column' }}>
              {importMode ? (
                <div style={{ flex: 1, overflow: 'auto', padding: '12px 16px' }}>
                  <div style={{ display: 'flex', alignItems: 'center', marginBottom: 8 }}>
                    <Button size="small" type="text" onClick={() => setImportMode(false)}>
                      ← {t('strategy.workspace.backToEditor', { defaultValue: 'Back' })}
                    </Button>
                  </div>
                  <ImportEAPanel
                    onApplyCode={(c) => { code.setCode(c); setCenterTab('code'); setImportMode(false); }}
                    onStrategyIdChange={(id) => { if (id) code.setStrategyId(id); }}
                  />
                </div>
              ) : code.code ? (
                <StrategyCodeEditor
                  value={code.code}
                  onChange={code.setCode}
                  style={{ flex: 1, borderRadius: 0, border: 'none', minHeight: 0 }}
                />
              ) : (
                <div style={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                  <div style={{ textAlign: 'center', maxWidth: 420, padding: 40 }}>
                    <div style={{ fontSize: 48, marginBottom: 16 }}>📝</div>
                    <div style={{ fontSize: 16, fontWeight: 600, marginBottom: 8, color: 'var(--ant-color-text)' }}>
                      {t('strategy.workspace.emptyTitle', { defaultValue: 'Start building your strategy' })}
                    </div>
                    <div style={{ fontSize: 13, color: 'var(--ant-color-text-secondary)', marginBottom: 24, lineHeight: 1.6 }}>
                      {t('strategy.workspace.emptyDesc', { defaultValue: 'Import an existing MQL EA, pick a template, or let AI generate one for you. All backtesting and deployment happens right here.' })}
                    </div>
                    <div style={{ display: 'flex', gap: 10, justifyContent: 'center', flexWrap: 'wrap' }}>
                      <Button type="primary" icon={<ImportOutlined />} onClick={() => setImportMode(true)}>
                        {t('strategy.workspace.importMql', { defaultValue: 'Import MQL EA' })}
                      </Button>
                      <Button icon={<RobotOutlined />} onClick={() => isMobile ? setCenterTab('chat') : setRightPanelTab('ai')}
                        style={{ background: '#722ed1', borderColor: '#722ed1', color: '#fff' }}>
                        {t('strategy.workspace.aiGenerate', { defaultValue: 'AI Generate' })}
                      </Button>
                      <Button icon={<HistoryOutlined />} onClick={() => templates.onSelect(templates.list[0]?.id || '')}
                        disabled={!templates.list.length}>
                        {t('strategy.workspace.useTemplate', { defaultValue: 'Use Template' })}
                      </Button>
                    </div>
                  </div>
                </div>
              )}
            </div>
            {!isMobile && rightPanelTab && (
              <WorkspaceAIPanel
                activeTab={rightPanelTab}
                onTabChange={setRightPanelTab}
                onClose={() => setRightPanelTab(null)}
                btSummary={btSummary}
                recentSummaries={recentSummaries}
                backtestStatus={backtest.status}
                backtestMetrics={backtest.metrics}
                chartTrades={backtest.chartTrades}
                onRunBacktest={() => setBtModalOpen(true)}
                onOpenAdvanced={() => setBtDrawerOpen(true)}
              />
            )}
          </div>
        </div>
      </div>

      <WorkspaceCenterModals
        btDrawerOpen={btDrawerOpen}
        setBtDrawerOpen={setBtDrawerOpen}
        sidebarDrawerOpen={sidebarDrawerOpen}
        setSidebarDrawerOpen={setSidebarDrawerOpen}
        isMobile={isMobile}
        onImport={() => setImportMode(true)}
      />

      <BottomPanelSection
        isMobile={!!isMobile}
        collapsed={layout.bottomPanelCollapsed}
        onToggleCollapsed={() => layout.setBottomPanelCollapsed(!layout.bottomPanelCollapsed)}
        positions={quickTrade.allPositions}
        recentTrades={quickTrade.qtRecentTrades}
        onClosePosition={quickTrade.handleClosePosition}
        backtestMetrics={backtest.metrics}
        backtestStatus={backtest.status}
        onOpenAdvancedBacktest={() => setBtDrawerOpen(true)}
        onRunBacktest={() => setBtModalOpen(true)}
        onAIOptimize={() => ai.optimize()}
        panelHeight={layout.bottomPanelUserResized ? layout.bottomPanelHeight : undefined}
        onResizeStart={handleBpResize}
        dragging={bpDragging}
        accountId={account.accountId}
        symbol={account.symbol}
        accountMeta={account.selectedAccountMeta}
        qtPositions={quickTrade.qtPositions}
      />
    </div>
  );
}
