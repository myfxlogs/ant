import { useState } from 'react';
import { Grid } from 'antd';
import { useTranslation } from 'react-i18next';
import { WorkspaceProvider, useWsAccount, useWsBacktest, useWsTuning, useWsLayout, useWsQuickTrade } from './WorkspaceContext';
import WorkspaceToolbar from './components/workspace/WorkspaceToolbar';
import WorkspaceCenterColumn from './components/workspace/WorkspaceCenterColumn';
import WorkspaceDrawers from './components/workspace/WorkspaceDrawers';
import WorkspaceTour from './components/workspace/WorkspaceTour';
import { MT_SESSION_LOST_KEY } from '@/gen/ant/v1/i18n/strategy_workspace_keys';

function WorkspaceInner({ isMobile }: { isMobile: boolean }) {
  const { t } = useTranslation();
  const [btModalOpen, setBtModalOpen] = useState(false);
  const [indicatorDrawerOpen, setIndicatorDrawerOpen] = useState(false);
  const [versionHistoryOpen, setVersionHistoryOpen] = useState(false);

  const account = useWsAccount();
  const backtest = useWsBacktest();
  const tuning = useWsTuning();
  const layout = useWsLayout();
  const quickTrade = useWsQuickTrade();
  const [mtError, setMtError] = useState<string | null>(null);

  // Header: 56px mobile, 64px desktop. Content padding-top matches in MainLayout.
  const headerHeight = isMobile ? 56 : 64;

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: `calc(100vh - ${headerHeight}px)` }}>
      <WorkspaceToolbar
        accounts={account.activeAccounts} accountId={account.accountId} onAccountChange={account.handleAccountChange}
        symbol={account.symbol} onSymbolChange={account.setSymbol}
        accountInfo={account.accountInfo} positionCount={quickTrade.positionCount}
        busy={backtest.submitting || tuning.running}
        mtError={mtError}
        onMTErrorChange={(hasError) => setMtError(hasError ? t(MT_SESSION_LOST_KEY) : null)}
        onToggleBottomPanel={() => layout.setBottomPanelCollapsed(!layout.bottomPanelCollapsed)}
      />
      <div style={{ display: 'flex', flex: '1 1 auto', overflow: 'hidden', minHeight: 0 }}>
        <WorkspaceCenterColumn
          isMobile={isMobile}
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
