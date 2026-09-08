import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { Modal } from 'antd';
import { useTranslation } from 'react-i18next';
import { useWorkspaceStore } from '@/stores/workspaceStore';
import StrategyChat from '@/components/strategy/StrategyChat';
import WorkspaceSidebar, { type WorkspaceSection, type NewSource } from './WorkspaceSidebar';
import NewStrategyPanel from './NewStrategyPanel';
import BacktestHistoryPanel from './BacktestHistoryPanel';
import WorkspaceAIPanel from './WorkspaceAIPanel';
import WorkspaceCenterTabBar from './WorkspaceCenterTabBar';
import CodeEditorArea from './CodeEditorArea';
import MobileSidebarDrawer from './MobileSidebarDrawer';
import BottomPanelSection from './BottomPanelSection';
import MobileBacktestContent from './MobileBacktestContent';
import { useWsAccount, useWsCode, useWsTemplates, useWsBacktest, useWsQuickTrade, useWsLayout, useWsHistory } from '../../WorkspaceContext';
import { useSidebarActions } from './useSidebarActions';
import { COMMON_CANCEL_KEY, COMMON_CONFIRM_KEY, COMMON_UNSAVED_KEY } from '@/gen/ant/v1/i18n/base_keys';
import { SIDEBAR_NEW_STRATEGY_KEY } from '@/gen/ant/v1/i18n/strategy_workspace_keys';

type CenterView = 'sources' | 'editor' | 'history';
type Dock = 'ai' | 'backtest' | null;



interface Props {
  isMobile?: boolean;
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
  const sidebarActions = useSidebarActions(code, history);

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

  const handleNewStrategy = useCallback(() => {
    const hasUnsaved = code.code && code.lastValidatedCode && code.code !== code.lastValidatedCode;
    const doNew = () => {
      templates.onSelect('');
      code.setCode('');
      code.setStrategyId(undefined);
      code.setValidationResult(null);
      code.setLastValidatedCode('');
      code.setLoadedTemplate(null);
      backtest.runner.resetStatus();
      setImportMode(false);
      setDock(null);
      setCenterTab('code');
    };
    if (hasUnsaved) {
      Modal.confirm({
        title: t(SIDEBAR_NEW_STRATEGY_KEY),
        content: t(COMMON_UNSAVED_KEY),
        okText: t(COMMON_CONFIRM_KEY),
        cancelText: t(COMMON_CANCEL_KEY),
        onOk: doNew,
      });
    } else {
      doNew();
    }
  }, [templates, code, backtest, setCenterTab, t]);

  // 来源选择（侧栏菜单项与主区大卡共用）：全部落在编辑器视图，分区保持展开
  const onNewSource = (source: NewSource) => {
    handleNewStrategy();
    if (source === 'ai') { setDock('ai'); return; }
    if (source === 'import') setImportMode(true);
    if (source === 'manual') {
      // 最小脚手架（<20 字符不触发审计），让用户直接落进空白编辑器
      code.setCode('# 新策略\n');
    }
    setCenterView('editor');
  };

  // 分区头点击：切视图 + 关停靠面板（分区是主区的导航）
  const onSectionChange = (s: WorkspaceSection) => {
    setActiveSection(s);
    setCenterView(s === 'new' ? 'sources' : 'editor');
    setDock(null);
  };

  const backtestHistoryPanel = (
    <BacktestHistoryPanel
      runs={(history.runs as Array<{ id: string; startedAt?: unknown; totalReturn?: number; totalTrades?: number; templateName?: string; name?: string }>) || []}
      loading={history.loading}
      onOpen={(runId: string) => { if (runId) backtest.loadRunById(runId, code.setCode); setDock('backtest'); }}
      onDelete={history.onDeleteRun}
    />
  );

  const sidebarProps = useMemo(() => ({
    templates: templates.list,
    loading: templates.loading,
    selectedId: templates.selectedId || '',
    onSelect: (id: string) => { templates.onSelect(id); setImportMode(false); setDock(null); setCenterView('editor'); },
    onDeleteTemplate: sidebarActions.onDeleteTemplate,
    onRenameTemplate: sidebarActions.onRenameTemplate,
    onBatchDeleteTemplates: sidebarActions.onBatchDeleteTemplates,
    backtestRuns: ((history.runs || []) as Array<{ id: string; startedAt?: string; totalReturn?: number; totalTrades?: number; templateName?: string; templateId?: string; name?: string }>),
    runsLoading: history.loading,
    onOpenHistory: (runId?: string) => { if (runId) backtest.loadRunById(runId, code.setCode); setCenterView('history'); setDock('backtest'); },
    onDeleteRun: history.onDeleteRun,
    onBatchDeleteRuns: sidebarActions.onBatchDeleteRuns,
    onRenameRun: sidebarActions.onRenameRun,
    onNew: handleNewStrategy,
    onNewSource,
    activeSection,
    onSectionChange,
    autoExpandHistory: history.autoExpandHistory,
  }), [templates, sidebarActions, history, handleNewStrategy, onNewSource, activeSection, onSectionChange, backtest, code.setCode]);

  const btSummary = backtest.metrics?.totalTrades != null
    ? { totalReturn: backtest.metrics.totalReturn, maxDrawdown: backtest.metrics.maxDrawdown, sharpeRatio: backtest.metrics.sharpeRatio, winRate: backtest.metrics.winRate, totalTrades: backtest.metrics.totalTrades }
    : undefined;
  const recentSummaries = (history.runs as Array<{ templateName?: string; totalReturn?: number; totalTrades?: number; startedAt?: string }>)
    ?.slice(0, 10).map(r => ({ templateName: r.templateName || '', totalReturn: r.totalReturn ?? 0, totalTrades: r.totalTrades ?? 0, startedAt: r.startedAt || '' })) || [];

  const aiDockPanel = (
    <div style={{ width: 420, flexShrink: 0, borderLeft: '1px solid var(--ant-color-border)', display: 'flex', flexDirection: 'column', minHeight: 0 }}>
      <WorkspaceAIPanel
        activeTab="ai"
        onTabChange={() => setDock('backtest')}
        onClose={() => setDock(null)}
        btSummary={btSummary}
        recentSummaries={recentSummaries}
      />
    </div>
  );
  const backtestDockPanel = (
    <div style={{ width: 420, flexShrink: 0, borderLeft: '1px solid var(--ant-color-border)', display: 'flex', flexDirection: 'column', minHeight: 0 }}>
      <WorkspaceAIPanel
        activeTab="backtest"
        onTabChange={() => setDock(null)}
        onClose={() => setDock(null)}
        btSummary={btSummary}
        recentSummaries={recentSummaries}
      />
    </div>
  );

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
            {dock === 'ai' && aiDockPanel}
            {dock === 'backtest' && backtestDockPanel}
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

      <BottomPanelSection
        isMobile={!!isMobile}
        collapsed={layout.bottomPanelCollapsed}
        onToggleCollapsed={() => layout.setBottomPanelCollapsed(!layout.bottomPanelCollapsed)}
        positions={quickTrade.allPositions}
        recentTrades={quickTrade.qtRecentTrades}
        onClosePosition={quickTrade.handleClosePosition}
        panelHeight={layout.bottomPanelUserResized ? layout.bottomPanelHeight : undefined}
        onResizeStart={handleBpResize}
        dragging={bpDragging}
        accountId={account.accountId}
        symbol={account.symbol}
        accountMeta={account.selectedAccountMeta ?? undefined}
        qtPositions={quickTrade.qtPositions}
        quickTradeCollapsed={layout.quickTradeCollapsed}
        onToggleQuickTrade={() => layout.setQuickTradeCollapsed(!layout.quickTradeCollapsed)}
        backtestContent={isMobile ? <MobileBacktestContent /> : null}
      />
    </div>
  );
}
