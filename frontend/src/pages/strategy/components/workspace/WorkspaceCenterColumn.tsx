import { useCallback, useEffect, useRef, useState } from 'react';
import { useWorkspaceStore } from '@/stores/workspaceStore';
import WorkspaceSidebar, { type WorkspaceSection } from './WorkspaceSidebar';
import NewStrategyPanel from './NewStrategyPanel';
import BacktestHistoryPanel from './BacktestHistoryPanel';
import WorkspaceCenterTabBar from './WorkspaceCenterTabBar';
import CodeEditorArea from './CodeEditorArea';
import MobileSidebarDrawer from './MobileSidebarDrawer';
import MobileStrategyChat from './MobileStrategyChat';
import WorkspaceBottomPanel from './WorkspaceBottomPanel';
import WorkspaceDocks from './WorkspaceDocks';
import { useWsAccount, useWsCode, useWsTemplates, useWsBacktest, useWsQuickTrade, useWsLayout, useWsHistory } from '../../WorkspaceContext';
import { useWorkspaceSidebarProps } from './useWorkspaceSidebarProps';

type CenterView = 'sources' | 'editor' | 'history';
type Dock = 'ai' | 'backtest' | null;



interface Props {
  isMobile?: boolean;
  setBtModalOpen: (v: boolean) => void;
  setIndicatorDrawerOpen: (v: boolean) => void;
  onShowVersionHistory?: () => void;
}

export default function WorkspaceCenterColumn({ isMobile = false, setBtModalOpen, setIndicatorDrawerOpen, onShowVersionHistory }: Props) {
  const centerTab = useWorkspaceStore(s => s.centerTab);
  const setCenterTab = useWorkspaceStore(s => s.setCenterTab);

  const account = useWsAccount();
  const code = useWsCode();
  const templates = useWsTemplates();
  const backtest = useWsBacktest();
  const quickTrade = useWsQuickTrade();
  const layout = useWsLayout();
  const history = useWsHistory();

  const leftSidebarCollapsed = useWorkspaceStore(s => s.leftSidebarCollapsed);
  const setLeftSidebarCollapsed = useWorkspaceStore(s => s.setLeftSidebarCollapsed);
  const leftSidebarWidth = useWorkspaceStore(s => s.leftSidebarWidth);
  const setLeftSidebarWidth = useWorkspaceStore(s => s.setLeftSidebarWidth);
  const [sidebarDrawerOpen, setSidebarDrawerOpen] = useState(false);

  // ── 工作台导航的完整状态：主区视图 + 停靠面板。没有其他隐藏维度。 ──
  const [centerView, setCenterView] = useState<CenterView>('sources');
  const [dock, setDock] = useState<Dock>(null);
  const [importMode, setImportMode] = useState(false);
  const [activeSection, setActiveSection] = useState<WorkspaceSection>('new');

  const prevBtStatusRef = useRef(backtest.status);
  useEffect(() => {
    if (backtest.status === 'running' && prevBtStatusRef.current !== 'running') {
      setDock('backtest');
    }
    prevBtStatusRef.current = backtest.status;
  }, [backtest.status]);
  useEffect(() => {
    layout.setBottomPanelCollapsed(dock !== null);
  }, [dock, layout]);

  // 回测完成等外部事件要求展开历史分区时，主区随之切换
  useEffect(() => {
    if (history.autoExpandHistory) { setCenterView('history'); setDock(null); }
  }, [history.autoExpandHistory]);

  const prevAccountIdRef = useRef(account.accountId);
  useEffect(() => {
    if (account.accountId && !prevAccountIdRef.current) {
      layout.setBottomPanelCollapsed(false);
    }
    prevAccountIdRef.current = account.accountId;
  }, [account.accountId, layout]);

  const prevSymbolRef = useRef(account.symbol);
  useEffect(() => {
    if (account.symbol && !prevSymbolRef.current) {
      layout.setQuickTradeCollapsed(false);
    }
    prevSymbolRef.current = account.symbol;
  }, [account.symbol, layout]);

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

  // 分区头点击：切视图 + 关停靠面板（分区是主区的导航）
  const onSectionChange = useCallback((s: WorkspaceSection) => {
    setActiveSection(s);
    setCenterView(s === 'new' ? 'sources' : 'editor');
    setDock(null);
  }, [setActiveSection, setCenterView, setDock]);

  // sidebarProps + 新建策略/来源选择回调均内聚在 hook（handleNewStrategy/onNewSource 不再外泄到本文件）
  const { sidebarProps, onNewSource } = useWorkspaceSidebarProps({
    activeSection,
    onSectionChange,
    onSetCenterView: setCenterView,
    onSetDock: setDock,
    onSetImportMode: setImportMode,
    onSetCode: code.setCode,
  });

  const backtestHistoryPanel = (
    <BacktestHistoryPanel
      runs={(history.runs as Array<{ id: string; startedAt?: unknown; totalReturn?: number; totalTrades?: number; templateName?: string; name?: string }>) || []}
      loading={history.loading}
      onOpen={(runId: string) => { if (runId) backtest.loadRunById(runId, code.setCode); setDock('backtest'); }}
      onDelete={history.onDeleteRun}
    />
  );



  const btSummary = backtest.metrics?.totalTrades != null
    ? { totalReturn: backtest.metrics.totalReturn, maxDrawdown: backtest.metrics.maxDrawdown, sharpeRatio: backtest.metrics.sharpeRatio, winRate: backtest.metrics.winRate, totalTrades: backtest.metrics.totalTrades }
    : undefined;
  const recentSummaries = (history.runs as Array<{ templateName?: string; totalReturn?: number; totalTrades?: number; startedAt?: string }>)
    ?.slice(0, 10).map(r => ({ templateName: r.templateName || '', totalReturn: r.totalReturn ?? 0, totalTrades: r.totalTrades ?? 0, startedAt: r.startedAt || '' })) || [];

  return (
    <div data-tour="code-editor" style={{ flex: '1 1 0', minWidth: 0, position: 'relative', overflow: 'hidden', display: 'flex', flexDirection: 'column' }}>
      <WorkspaceCenterTabBar
        isMobile={isMobile}
        centerTab={centerTab}
        setCenterTab={setCenterTab}
        setSidebarDrawerOpen={setSidebarDrawerOpen}
        setBtModalOpen={setBtModalOpen}
        setIndicatorDrawerOpen={setIndicatorDrawerOpen}
        onShowVersionHistory={onShowVersionHistory}
        rightPanelTab={dock}
        setRightPanelTab={setDock}
        code={code}
        account={account}
        templates={templates}
      />

      <div style={{ flex: '1 1 0', minHeight: 0, display: 'flex', flexDirection: 'row' }}>
        {!isMobile && (
          <WorkspaceSidebar
            {...sidebarProps}
            collapsed={leftSidebarCollapsed}
            onToggle={() => setLeftSidebarCollapsed(!leftSidebarCollapsed)}
            width={leftSidebarWidth}
            onWidthChange={setLeftSidebarWidth}
          />
        )}

        <div style={{ flex: '1 1 0', minWidth: 0, display: 'flex', flexDirection: 'column' }}>
          {isMobile && (
            <div style={{ flex: '1 1 0', minHeight: 0, display: centerTab === 'chat' ? 'flex' : 'none', flexDirection: 'column' }}>
              <MobileStrategyChat
                symbol={account.symbol}
                timeframe={account.timeframe}
                accountId={account.accountId}
                currentCode={code.code}
                lastBacktest={btSummary}
                recentBacktests={recentSummaries}
                onApplyCode={c => { code.setCode(c); setCenterTab('code'); }}
              />
            </div>
          )}

          <div style={{ flex: '1 1 0', minHeight: 0, display: centerTab === 'code' ? 'flex' : 'none', flexDirection: 'row' }}>
            {dock && (
              <WorkspaceDocks
                dock={dock}
                btSummary={btSummary}
                recentSummaries={recentSummaries}
                onSwitchToBacktest={() => setDock('backtest')}
                onClose={() => setDock(null)}
              />
            )}
            {!dock && centerView === 'sources' && <NewStrategyPanel onNewSource={onNewSource} />}
            {!dock && centerView === 'editor' && (
              <CodeEditorArea
                code={code.code || ''}
                importMode={importMode}
                onSetImportMode={setImportMode}
                onSetCode={code.setCode}
                onStrategyIdChange={(id) => { if (id) code.setStrategyId(id); }}
              />
            )}
            {!dock && centerView === 'history' && backtestHistoryPanel}
          </div>
        </div>
      </div>

      {isMobile && (
        <MobileSidebarDrawer
          {...sidebarProps}
          open={sidebarDrawerOpen}
          onClose={() => setSidebarDrawerOpen(false)}
        />
      )}

      <WorkspaceBottomPanel
        isMobile={!!isMobile}
        accountId={account.accountId}
        symbol={account.symbol}
        bottomPanelCollapsed={layout.bottomPanelCollapsed}
        onToggleBottomPanelCollapsed={() => layout.setBottomPanelCollapsed(!layout.bottomPanelCollapsed)}
        bottomPanelUserResized={layout.bottomPanelUserResized}
        bottomPanelHeight={layout.bottomPanelHeight}
        onResizeStart={handleBpResize}
        dragging={bpDragging}
        allPositions={quickTrade.allPositions}
        qtRecentTrades={quickTrade.qtRecentTrades}
        handleClosePosition={quickTrade.handleClosePosition}
        qtPositions={quickTrade.qtPositions}
        quickTradeCollapsed={layout.quickTradeCollapsed}
        onToggleQuickTrade={() => layout.setQuickTradeCollapsed(!layout.quickTradeCollapsed)}
        accountMeta={account.selectedAccountMeta}
      />
    </div>
  );
}
