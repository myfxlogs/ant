import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useWorkspaceStore } from '@/stores/workspaceStore';
import StrategyChat from '@/components/strategy/StrategyChat';
import WorkspaceSidebar from './WorkspaceSidebar';
import WorkspaceAIPanel from './WorkspaceAIPanel';
import WorkspaceCenterTabBar from './WorkspaceCenterTabBar';
import CodeEditorArea from './CodeEditorArea';
import MobileSidebarDrawer from './MobileSidebarDrawer';
import BottomPanelSection from './BottomPanelSection';
import { useWsAccount, useWsCode, useWsTemplates, useWsBacktest, useWsQuickTrade, useWsLayout, useWsHistory } from '../../WorkspaceContext';
import { useSidebarActions } from './useSidebarActions';

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
  const sidebarActions = useSidebarActions(code, history);

  // ── Sidebar ──────────────────────────────────────────────────────────
  const leftSidebarCollapsed = useWorkspaceStore(s => s.leftSidebarCollapsed);
  const setLeftSidebarCollapsed = useWorkspaceStore(s => s.setLeftSidebarCollapsed);
  const [sidebarDrawerOpen, setSidebarDrawerOpen] = useState(false);

  // ── Right panel tab: 'ai' | 'backtest' | null — mode-driven layout ──
  const [rightPanelTab, setRightPanelTab] = useState<'ai' | 'backtest' | null>(null);
  // Auto-switch to backtest tab when backtest starts
  const prevBtStatusRef = useRef(backtest.status);
  useEffect(() => {
    if (backtest.status === 'running' && prevBtStatusRef.current !== 'running') {
      setRightPanelTab('backtest');
    }
    prevBtStatusRef.current = backtest.status;
  }, [backtest.status]);
  // Mode-driven: collapse bottom panel when right panel is active, restore when it closes.
  useEffect(() => {
    layout.setBottomPanelCollapsed(rightPanelTab === 'backtest' || rightPanelTab === 'ai');
  }, [rightPanelTab, layout]);

  // Auto-expand bottom panel when account is selected.
  const prevAccountIdRef = useRef(account.accountId);
  useEffect(() => {
    if (account.accountId && !prevAccountIdRef.current) {
      layout.setBottomPanelCollapsed(false);
    }
    prevAccountIdRef.current = account.accountId;
  }, [account.accountId, layout]);

  // Auto-expand quick trade panel when symbol is selected.
  const prevSymbolRef = useRef(account.symbol);
  useEffect(() => {
    if (account.symbol && !prevSymbolRef.current) {
      layout.setQuickTradeCollapsed(false);
    }
    prevSymbolRef.current = account.symbol;
  }, [account.symbol, layout]);

  // ── Bottom panel resize ──────────────────────────────────────────────
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

  // ── Auto-restore last backtest on first visit ────────────────────────
  const restoreRef = useRef(false);
  useEffect(() => {
    if (restoreRef.current) return;
    if (!account.accountId) return;
    if (backtest.runner.status !== 'idle') return;
    restoreRef.current = true;
    backtest.runner.restoreLastRun(account.accountId, templates.selectedId || undefined);
  }, [account.accountId, backtest.runner, templates.selectedId]);

  // ── Import MQL inline (replaces empty state area, never modal) ────────
  const [importMode, setImportMode] = useState(false);

  // ── New strategy handler (shared by desktop + mobile sidebar) ────────
  const handleNewStrategy = useCallback(() => {
    templates.onSelect('');
    code.setCode('');
    code.setStrategyId(undefined);
    code.setValidationResult(null);
    code.setLastValidatedCode('');
    setCenterTab('code');
  }, [templates, code, setCenterTab]);

  // ── Shared sidebar props (desktop + mobile) ──────────────────────────
  const sidebarProps = useMemo(() => ({
    templates: templates.list,
    loading: templates.loading,
    selectedId: templates.selectedId || '',
    onSelect: (id: string) => templates.onSelect(id),
    onDeleteTemplate: sidebarActions.onDeleteTemplate,
    onRenameTemplate: sidebarActions.onRenameTemplate,
    onBatchDeleteTemplates: sidebarActions.onBatchDeleteTemplates,
    backtestRuns: (history.runs as Array<{ id: string; startedAt?: string; totalReturn?: number; totalTrades?: number; templateName?: string; templateId?: string; name?: string }>) || [],
    runsLoading: history.loading,
    onOpenHistory: (tid?: string) => history.open(tid),
    onDeleteRun: history.onDeleteRun,
    onBatchDeleteRuns: sidebarActions.onBatchDeleteRuns,
    onRenameRun: sidebarActions.onRenameRun,
    onImport: () => setImportMode(true),
    onNew: handleNewStrategy,
    autoExpandHistory: history.autoExpandHistory,
  }), [templates, sidebarActions, history, handleNewStrategy, setImportMode]);

  // ── Backtest context for AI ───────────────────────────────────────────
  const btSummary = backtest.metrics?.totalTrades != null
    ? { totalReturn: backtest.metrics.totalReturn, maxDrawdown: backtest.metrics.maxDrawdown, sharpeRatio: backtest.metrics.sharpeRatio, winRate: backtest.metrics.winRate, totalTrades: backtest.metrics.totalTrades }
    : undefined;
  const recentSummaries = (history.runs as Array<{ templateName?: string; totalReturn?: number; totalTrades?: number; startedAt?: string }>)
    ?.slice(0, 10).map(r => ({ templateName: r.templateName || '', totalReturn: r.totalReturn ?? 0, totalTrades: r.totalTrades ?? 0, startedAt: r.startedAt || '' })) || [];

  // Desktop: if centerTab is 'chat', open AI panel instead
  useEffect(() => {
    if (!isMobile && centerTab === 'chat') {
      setRightPanelTab('ai');
      setCenterTab('code');
    }
  }, [isMobile, centerTab, setCenterTab]);

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
        rightPanelTab={rightPanelTab}
        setRightPanelTab={setRightPanelTab}
        code={code}
        account={account}
        templates={templates}
      />

      {/* Main area: sidebar + content */}
      <div style={{ flex: '1 1 0', minHeight: 0, display: 'flex', flexDirection: 'row' }}>
        {/* Sidebar: persistent on desktop, drawer overlay on mobile */}
        {!isMobile && (
          <WorkspaceSidebar
            {...sidebarProps}
            collapsed={leftSidebarCollapsed}
            onToggle={() => setLeftSidebarCollapsed(!leftSidebarCollapsed)}
          />
        )}

        {/* Content */}
        <div style={{ flex: '1 1 0', minWidth: 0, display: 'flex', flexDirection: 'column' }}>
          {/* AI Chat — full width tab (mobile only; desktop uses right panel) */}
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

          {/* Code + optional right panel (desktop) */}
          <div style={{ flex: '1 1 0', minHeight: 0, display: centerTab === 'code' ? 'flex' : 'none', flexDirection: 'row' }}>
            {!isMobile && rightPanelTab ? (
              <WorkspaceAIPanel
                activeTab={rightPanelTab}
                onTabChange={setRightPanelTab}
                onClose={() => setRightPanelTab(null)}
                btSummary={btSummary}
                recentSummaries={recentSummaries}
              />
            ) : (
              <CodeEditorArea
                code={code.code || ''}
                importMode={importMode}
                isMobile={isMobile}
                templateCount={templates.list.length}
                onSetImportMode={setImportMode}
                onSetCode={code.setCode}
                onSetCenterTab={setCenterTab}
                onSetRightPanelTab={setRightPanelTab}
                onSelectFirstTemplate={() => templates.onSelect(templates.list[0]?.id || '')}
                onStrategyIdChange={(id) => { if (id) code.setStrategyId(id); }}
              />
            )}
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

      {/* Bottom panel: Positions | History | Backtest  +  Quick Trade on the right (desktop only) */}
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
      />
    </div>
  );
}
