import { useState, useCallback } from 'react';
import { Grid, Drawer } from 'antd';
import { useTranslation } from 'react-i18next';
import { useStrategyWorkspaceState } from './hooks/useStrategyWorkspaceState';
import WorkspaceToolbar from './components/workspace/WorkspaceToolbar';
import { TITLE_KEY as INDICATOR_CATALOG_TITLE_KEY } from '@/gen/ant/v1/i18n/indicator_catalog_keys';
import { useWorkspaceStore } from '@/stores/workspaceStore';
import BacktestHistoryDrawer from './components/workspace/BacktestHistoryDrawer';
import VersionHistoryDrawer from './components/workspace/VersionHistoryDrawer';
import MobileGuard from './components/workspace/MobileGuard';
import RightPanel from './components/workspace/RightPanel';
import { SaveTemplateWrapper } from './WorkspaceLayout';
import { BacktestParamsModal, type BacktestModalResult } from './components/workspace/BacktestParamsModal';
import WorkspaceCenterColumn from './components/workspace/WorkspaceCenterColumn';
import IndicatorCatalogContent from './components/workspace/IndicatorCatalogContent';

export default function StrategyWorkspacePage() {
  const { t } = useTranslation();
  const screens = Grid.useBreakpoint();
  const ws = useStrategyWorkspaceState();
  const rightPanelWidth = useWorkspaceStore(s => s.rightPanelWidth);
  const setRightPanelWidth = useWorkspaceStore(s => s.setRightPanelWidth);
  const setCenterTab = useWorkspaceStore(s => s.setCenterTab);
  const [btModalOpen, setBtModalOpen] = useState(false);
  const [indicatorDrawerOpen, setIndicatorDrawerOpen] = useState(false);
  const [versionHistoryOpen, setVersionHistoryOpen] = useState(false);

  const startResize = useCallback((e: React.MouseEvent) => {
    e.preventDefault();
    const startX = e.clientX;
    const startW = rightPanelWidth;
    const onMove = (ev: MouseEvent) => { setRightPanelWidth(startW + (startX - ev.clientX)); };
    const onUp = () => {
      document.removeEventListener('mousemove', onMove);
      document.removeEventListener('mouseup', onUp);
      document.body.style.cursor = '';
      document.body.style.userSelect = '';
    };
    document.addEventListener('mousemove', onMove);
    document.addEventListener('mouseup', onUp);
    document.body.style.cursor = 'col-resize';
    document.body.style.userSelect = 'none';
  }, [rightPanelWidth, setRightPanelWidth]);

  if (!screens.lg) return <MobileGuard />;

  const strategyName = ws.templates.list.find((t2: any) => t2.id === ws.templates.selectedId)?.name || ws.code.loadedTemplate?.name || '';
  const saveStatus: 'modified' | 'saved' | 'none' = ws.code.code && ws.code.lastValidatedCode && ws.code.code !== ws.code.lastValidatedCode ? 'modified' : ws.code.lastSavedId ? 'saved' : 'none';

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: 'calc(100vh - 112px)' }}>
      <WorkspaceToolbar
        accounts={ws.account.activeAccounts} accountId={ws.account.accountId} onAccountChange={ws.account.handleAccountChange}
        symbol={ws.account.symbol} onSymbolChange={ws.account.setSymbol}
        accountInfo={ws.account.accountInfo} positionCount={ws.quickTrade.positionCount}
        busy={ws.backtest.submitting || ws.tuning.running}
        onTogglePositionsPanel={() => ws.layout.setPositionsPanelVisible(!ws.layout.positionsPanelVisible)}
        strategyName={strategyName}
        saveStatus={saveStatus}
      />

      <div style={{ display: 'flex', flex: '1 1 auto', overflow: 'hidden', minHeight: 0 }}>
        {/* ── CENTER COLUMN ── */}
        <WorkspaceCenterColumn
          ws={ws}
          btModalOpen={btModalOpen}
          setBtModalOpen={setBtModalOpen}
          setIndicatorDrawerOpen={setIndicatorDrawerOpen}
          onShowVersionHistory={() => setVersionHistoryOpen(true)}
        />

        {/* ── Resize handle ── */}
        <div
          onMouseDown={startResize}
          style={{
            width: 5, flexShrink: 0, cursor: 'col-resize',
            background: 'var(--ant-color-border)',
            transition: 'background 0.15s',
          }}
          onMouseEnter={(e) => (e.currentTarget.style.background = '#58a6ff')}
          onMouseLeave={(e) => (e.currentTarget.style.background = 'var(--ant-color-border)')}
        />

        {/* ── RIGHT COLUMN ── */}
        <RightPanel
          symbol={ws.account.symbol}
          timeframe={ws.account.timeframe}
          accountId={ws.account.accountId}
          onApplyCode={(code) => { ws.code.setCode(code); setCenterTab("code"); }}
          onValidateResult={(result) => ws.backtest.runner.handleValidationResult(result)}
          width={rightPanelWidth}
          currentCode={ws.code.code}
        />
      </div>

      {/* ── Modals & Drawers ── */}
      <BacktestParamsModal
        open={btModalOpen}
        onClose={() => setBtModalOpen(false)}
        code={ws.code.code}
        symbol={ws.account.symbol}
        onConfirm={(result: BacktestModalResult) => {
          const p = result.params;
          ws.backtest.setInitialCapital(p.initialCapital);
          ws.backtest.setLeverage(p.leverage);
          ws.backtest.setCommission(p.commission);
          ws.backtest.setSlippage(p.slippage);
          ws.backtest.setTradeDirection(p.tradeDirection);
          ws.backtest.setStrictMode(p.strictMode);
          ws.backtest.setStartDate(result.startDate);
          ws.backtest.setEndDate(result.endDate);
          if (result.strategyParams) {
            for (const [name, value] of Object.entries(result.strategyParams)) {
              ws.backtest.runner.setParam(name, value);
            }
          }
          ws.backtest.run();
          setCenterTab('backtest');
        }}
      />
      <SaveTemplateWrapper ws={ws} />
      <Drawer title={t(INDICATOR_CATALOG_TITLE_KEY)} open={indicatorDrawerOpen} onClose={() => setIndicatorDrawerOpen(false)} width={640} styles={{ body: { overflowY: 'auto' } }}>
        <IndicatorCatalogContent />
      </Drawer>
      <BacktestHistoryDrawer
        open={ws.history.modalOpen || ws.history.drawerOpen}
        runs={ws.history.runs}
        loading={ws.history.loading}
        page={ws.history.page}
        pageSize={ws.history.pageSize}
        total={ws.history.total}
        selectedRowKeys={ws.history.selectedRowKeys}
        deleting={ws.history.deleting}
        onPageChange={ws.history.onPageChange}
        onSelectionChange={ws.history.setSelectedRowKeys}
        onViewRun={ws.history.onViewRun}
        onDeleteRun={ws.history.onDeleteRun}
        onBatchDelete={ws.history.onBatchDelete}
        onRefresh={ws.history.onRefresh}
        onClose={ws.history.runId ? ws.history.close : ws.history.closeModal}
        runId={ws.history.runId}
      />
      <VersionHistoryDrawer
        open={versionHistoryOpen}
        strategyId={ws.code.strategyId}
        onClose={() => setVersionHistoryOpen(false)}
        onRollback={(sourceCode) => { ws.code.setCode(sourceCode); }}
      />
    </div>
  );
}
