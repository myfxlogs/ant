import { useState, useCallback } from 'react';
import { Grid, Button, Drawer } from 'antd';
import { useTranslation } from 'react-i18next';
import { useStrategyWorkspaceState } from './hooks/useStrategyWorkspaceState';
import WorkspaceTemplateManager from './components/workspace/WorkspaceTemplateManager';
import WorkspaceToolbar from './components/workspace/WorkspaceToolbar';
import BacktestResultsCard from '@/components/strategy/BacktestResultsCard';
import { useAuthStore } from '@/stores/authStore';
import { useWorkspaceStore } from '@/stores/workspaceStore';
import BacktestHistoryDrawer from './components/workspace/BacktestHistoryDrawer';
import MobileGuard from './components/workspace/MobileGuard';
import RightPanel from './components/workspace/RightPanel';
import { useWorkspaceSession } from './hooks/useWorkspaceSession';
import { SaveTemplateWrapper } from './WorkspaceLayout';
import { BacktestParamsModal, type BacktestModalResult } from './components/workspace/BacktestParamsModal';
import WorkspaceCenterColumn from './components/workspace/WorkspaceCenterColumn';
import IndicatorCatalogContent from './components/workspace/IndicatorCatalogContent';
import TemplateManagerContent from './components/workspace/TemplateManagerContent';

const COL_BORDER = '1px solid var(--ant-color-border)';

export default function StrategyWorkspacePage() {
  const { t } = useTranslation();
  const screens = Grid.useBreakpoint();
  const ws = useStrategyWorkspaceState();
  const userId = useAuthStore(s => s.user?.id) ?? '';
  const { sessionId } = useWorkspaceSession(
    userId,
    ws.account.symbol,
    ws.account.timeframe,
    ws.code.lastSavedId,
  );

  const rightPanelWidth = useWorkspaceStore(s => s.rightPanelWidth);
  const setRightPanelWidth = useWorkspaceStore(s => s.setRightPanelWidth);
  const [btModalOpen, setBtModalOpen] = useState(false);
  const [indicatorDrawerOpen, setIndicatorDrawerOpen] = useState(false);
  const [tplDrawerOpen, setTplDrawerOpen] = useState(false);

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
        {/* ── LEFT COLUMN ── */}
        <div style={{
          width: 260, minWidth: 260, flexShrink: 0, borderRight: COL_BORDER,
          display: 'flex', flexDirection: 'column', overflow: 'hidden',
          marginLeft: ws.layout.leftSidebarCollapsed ? -260 : 0,
          transition: 'margin-left 0.25s ease',
          background: 'var(--ant-color-bg-container)',
        }}>
          <div style={{ padding: '8px 12px 2px', fontSize: 10, fontWeight: 700, color: 'var(--ant-color-text-tertiary)', textTransform: 'uppercase', letterSpacing: 0.5, flexShrink: 0, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <span>📋 {t('strategy.workspace.templates')}</span>
            <Button size="small" type="text" style={{ fontSize: 10, padding: '0 4px' }} onClick={() => setTplDrawerOpen(true)}>{t('strategy.workspace.manage')}</Button>
          </div>
          <div style={{ padding: '4px 12px', flexShrink: 0 }}>
            <WorkspaceTemplateManager
              templates={ws.templates.list}
              loading={ws.templates.loading}
              loadedTemplate={ws.templates.list.find((t2: any) => t2.id === ws.templates.selectedId) || null}
              onLoad={ws.templates.onSelect}
              onSaveAs={() => ws.code.setSaveModalOpen(true)}
            />
          </div>
          <div style={{ flexShrink: 0 }}>
            <BacktestResultsCard metrics={ws.backtest.metrics} status={ws.backtest.status} />
          </div>
          <div style={{ flex: 1 }} />
        </div>

        {/* Left sidebar toggle */}
        <div
          onClick={() => ws.layout.setLeftSidebarCollapsed(!ws.layout.leftSidebarCollapsed)}
          style={{
            width: 28, height: 80, flexShrink: 0,
            background: 'var(--ant-color-bg-elevated)',
            border: '1px solid var(--ant-color-border)', borderLeft: 'none',
            borderRadius: '0 8px 8px 0', cursor: 'pointer', zIndex: 20,
            display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center',
            color: ws.layout.leftSidebarCollapsed ? 'var(--ant-color-text-tertiary)' : '#58a6ff',
            fontSize: 11, writingMode: 'vertical-rl', letterSpacing: 2,
            alignSelf: 'center',
          }}
        >
          {t('strategy.workspace.templates')}
        </div>

        {/* ── CENTER COLUMN ── */}
        <WorkspaceCenterColumn
          ws={ws}
          btModalOpen={btModalOpen}
          setBtModalOpen={setBtModalOpen}
          setIndicatorDrawerOpen={setIndicatorDrawerOpen}
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
          sessionId={sessionId}
          accountId={ws.account.accountId}
          onApplyCode={(code) => { ws.code.setCode(code); }}
          onValidateResult={(result) => ws.backtest.runner.handleValidationResult(result)}
          width={rightPanelWidth}
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
          ws.backtest.run();
        }}
      />
      <SaveTemplateWrapper ws={ws} />
      <Drawer title={t('indicatorCatalog.title')} open={indicatorDrawerOpen} onClose={() => setIndicatorDrawerOpen(false)} width={640} styles={{ body: { overflowY: 'auto' } }}>
        <IndicatorCatalogContent />
      </Drawer>
      <Drawer title={t('strategy.library.title', 'Strategy Library')} open={tplDrawerOpen} onClose={() => setTplDrawerOpen(false)} width={420} styles={{ body: { padding: 0, overflow: 'hidden' } }}>
        <TemplateManagerContent />
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
    </div>
  );
}
