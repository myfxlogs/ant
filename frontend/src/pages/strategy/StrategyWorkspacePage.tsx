import { useState, useEffect } from 'react';
import { Grid } from 'antd';
import { WorkspaceProvider, useWsAccount, useWsCode, useWsTemplates, useWsBacktest, useWsTuning, useWsLayout, useWsQuickTrade } from './WorkspaceContext';
import { useWorkspaceResize } from './hooks/useWorkspaceResize';
import { useWorkspaceStore } from '@/stores/workspaceStore';
import WorkspaceToolbar from './components/workspace/WorkspaceToolbar';
import RightPanel from './components/workspace/RightPanel';
import WorkspaceCenterColumn from './components/workspace/WorkspaceCenterColumn';
import WorkspaceDrawers from './components/workspace/WorkspaceDrawers';
import WorkspaceTour from './components/workspace/WorkspaceTour';

function WorkspaceInner({ isMobile }: { isMobile: boolean }) {
  const [btModalOpen, setBtModalOpen] = useState(false);
  const [indicatorDrawerOpen, setIndicatorDrawerOpen] = useState(false);
  const [importDrawerOpen, setImportDrawerOpen] = useState(false);
  const [versionHistoryOpen, setVersionHistoryOpen] = useState(false);
  const setCenterTab = useWorkspaceStore(s => s.setCenterTab);
  const centerTab = useWorkspaceStore(s => s.centerTab);
  const { rightPanelWidth, startResize } = useWorkspaceResize();

  // On mobile, redirect 'design' (chart) tab to 'code' since chart is desktop-only.
  useEffect(() => {
    if (isMobile && centerTab === 'design') {
      setCenterTab('code');
    }
  }, [isMobile, centerTab, setCenterTab]);

  const account = useWsAccount();
  const code = useWsCode();
  const templates = useWsTemplates();
  const backtest = useWsBacktest();
  const tuning = useWsTuning();
  const layout = useWsLayout();
  const quickTrade = useWsQuickTrade();

  const strategyName = templates.list.find((t2: { id: string; name?: string }) => t2.id === templates.selectedId)?.name || code.loadedTemplate?.name || '';
  const saveStatus: 'modified' | 'saved' | 'none' = code.code && code.lastValidatedCode && code.code !== code.lastValidatedCode ? 'modified' : code.lastSavedId ? 'saved' : 'none';

  // Header: 56px mobile, 64px desktop. Content padding-top matches in MainLayout.
  const headerHeight = isMobile ? 56 : 64;

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: `calc(100vh - ${headerHeight}px)` }}>
      <WorkspaceToolbar
        accounts={account.activeAccounts} accountId={account.accountId} onAccountChange={account.handleAccountChange}
        symbol={account.symbol} onSymbolChange={account.setSymbol}
        accountInfo={account.accountInfo} positionCount={quickTrade.positionCount}
        busy={backtest.submitting || tuning.running}
        onTogglePositionsPanel={() => layout.setPositionsPanelVisible(!layout.positionsPanelVisible)}
        strategyName={strategyName}
        saveStatus={saveStatus}
      />
      <div style={{ display: 'flex', flex: '1 1 auto', overflow: 'hidden', minHeight: 0 }}>
        <WorkspaceCenterColumn
          isMobile={isMobile}
          btModalOpen={btModalOpen}
          setBtModalOpen={setBtModalOpen}
          setIndicatorDrawerOpen={setIndicatorDrawerOpen}
          setImportDrawerOpen={setImportDrawerOpen}
          onShowVersionHistory={() => setVersionHistoryOpen(true)}
        />
        {!isMobile && (
          <>
            <div onMouseDown={startResize} style={{ width: 5, flexShrink: 0, cursor: 'col-resize', background: 'var(--ant-color-border)' }}
              onMouseEnter={e => (e.currentTarget.style.background = '#58a6ff')}
              onMouseLeave={e => (e.currentTarget.style.background = 'var(--ant-color-border)')} />
            <RightPanel symbol={account.symbol} timeframe={account.timeframe} accountId={account.accountId}
              onApplyCode={c => { code.setCode(c); setCenterTab('code'); }}
              onValidateResult={result => backtest.runner.handleValidationResult(result)}
              width={rightPanelWidth} currentCode={code.code} />
          </>
        )}
      </div>
      <WorkspaceDrawers btModalOpen={btModalOpen} setBtModalOpen={setBtModalOpen}
        indicatorDrawerOpen={indicatorDrawerOpen} setIndicatorDrawerOpen={setIndicatorDrawerOpen}
        importDrawerOpen={importDrawerOpen} setImportDrawerOpen={setImportDrawerOpen}
        versionHistoryOpen={versionHistoryOpen} setVersionHistoryOpen={setVersionHistoryOpen} />
      <WorkspaceTour />
    </div>
  );
}

export default function StrategyWorkspacePage() {
  const screens = Grid.useBreakpoint();
  const isMobile = !screens.lg;
  return <WorkspaceProvider><WorkspaceInner isMobile={isMobile} /></WorkspaceProvider>;
}
