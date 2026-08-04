import { useState } from 'react';
import { Grid } from 'antd';
import { WorkspaceProvider, useWsAccount, useWsTemplates, useWsBacktest, useWsTuning, useWsLayout, useWsQuickTrade, useWsCode } from './WorkspaceContext';
import WorkspaceToolbar from './components/workspace/WorkspaceToolbar';
import WorkspaceCenterColumn from './components/workspace/WorkspaceCenterColumn';
import WorkspaceDrawers from './components/workspace/WorkspaceDrawers';
import WorkspaceTour from './components/workspace/WorkspaceTour';

function WorkspaceInner({ isMobile }: { isMobile: boolean }) {
  const [btModalOpen, setBtModalOpen] = useState(false);
  const [indicatorDrawerOpen, setIndicatorDrawerOpen] = useState(false);
  const [versionHistoryOpen, setVersionHistoryOpen] = useState(false);

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
          onShowVersionHistory={() => setVersionHistoryOpen(true)}
        />
      </div>
      <WorkspaceDrawers btModalOpen={btModalOpen} setBtModalOpen={setBtModalOpen}
        indicatorDrawerOpen={indicatorDrawerOpen} setIndicatorDrawerOpen={setIndicatorDrawerOpen}
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
