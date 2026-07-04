import { Grid } from 'antd';
import { useTranslation } from 'react-i18next';
import { CHART_ERROR_KEY, SELECT_SYMBOL_HINT_KEY } from '@/gen/ant/v1/i18n/strategy_workspace_keys';
import { useStrategyWorkspaceState } from './hooks/useStrategyWorkspaceState';
import WorkspaceTemplateManager from './components/workspace/WorkspaceTemplateManager';
import WorkspaceToolbar from './components/workspace/WorkspaceToolbar';
import BacktestResultsCard from '@/components/strategy/BacktestResultsCard';
import ChartBottomPanel from '@/components/chart/ChartBottomPanel';
import { useAuthStore } from '@/stores/authStore';
import PriceChart from '@/components/chart/PriceChart';
import BacktestHistoryDrawer from './components/workspace/BacktestHistoryDrawer';
import WorkspaceErrorBoundary from './components/workspace/WorkspaceErrorBoundary';
import MobileGuard from './components/workspace/MobileGuard';
import RightPanel from './components/workspace/RightPanel';
import { useWorkspaceSession } from './hooks/useWorkspaceSession';
import { SaveTemplateWrapper } from './WorkspaceLayout';

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

  if (!screens.lg) return <MobileGuard />;

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: 'calc(100vh - 112px)' }}>
      {/* ═══ TOP TOOLBAR (44px single line) ═══ */}
      <WorkspaceToolbar
        accounts={ws.account.activeAccounts} accountId={ws.account.accountId} onAccountChange={ws.account.handleAccountChange}
        symbol={ws.account.symbol} onSymbolChange={ws.account.setSymbol}
        accountInfo={ws.account.accountInfo} positionCount={ws.quickTrade.positionCount}
        busy={ws.backtest.submitting || ws.tuning.running}
        positionsCount={ws.quickTrade.allPositions.length}
        onTogglePositionsPanel={() => ws.layout.setPositionsPanelVisible(!ws.layout.positionsPanelVisible)}
        strategyName={ws.templates.list.find((t: any) => t.id === ws.templates.selectedId)?.name}
        saveStatus={ws.code.code && ws.code.lastValidatedCode && ws.code.code !== ws.code.lastValidatedCode ? 'modified' : ws.code.lastSavedId ? 'saved' : 'none'}
      />

      {/* ═══ BODY: Three-column layout ═══ */}
      <div style={{ display: 'flex', flex: '1 1 auto', overflow: 'hidden', minHeight: 0 }}>

        {/* ── LEFT COLUMN: Strategy list + backtest summary (260px, collapsible) ── */}
        <div style={{
          width: 260, minWidth: 260, flexShrink: 0, borderRight: COL_BORDER,
          display: 'flex', flexDirection: 'column', overflow: 'hidden',
          marginLeft: ws.layout.leftSidebarCollapsed ? -260 : 0,
          transition: 'margin-left 0.25s ease',
          background: 'var(--ant-color-bg-container)',
        }}>
          {/* Section: Strategy templates */}
          <div style={{ padding: '8px 12px 2px', fontSize: 10, fontWeight: 700, color: 'var(--ant-color-text-tertiary)', textTransform: 'uppercase', letterSpacing: 0.5, flexShrink: 0 }}>
            📋 {t('strategy.workspace.templates', 'Strategies')}
          </div>
          <div style={{ padding: '4px 12px', flexShrink: 0 }}>
            <WorkspaceTemplateManager
              templates={ws.templates.list}
              loading={ws.templates.loading}
              loadedTemplate={ws.templates.list.find((t: any) => t.id === ws.templates.selectedId) || null}
              onLoad={ws.templates.onSelect}
              onSaveAs={() => ws.code.setSaveModalOpen(true)}
            />
          </div>

          {/* Section: Backtest summary */}
          <div style={{ flexShrink: 0 }}>
            <BacktestResultsCard metrics={ws.backtest.metrics} status={ws.backtest.status} />
          </div>

          <div style={{ flex: 1 }} />
        </div>

        {/* Left sidebar toggle button */}
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
          Strategies
        </div>

        {/* ── CENTER COLUMN: Chart full height + bottom panel ── */}
        <div style={{ flex: '1 1 0', minWidth: 0, position: 'relative', overflow: 'hidden', display: 'flex', flexDirection: 'column' }}>
          <div style={{ flex: '1 1 0', minHeight: 0, position: 'relative', display: 'flex', flexDirection: 'column' }}>
            {ws.account.symbol ? (
              <WorkspaceErrorBoundary fallback={<div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '100%', color: 'var(--ant-color-text-tertiary)' }}>{t(CHART_ERROR_KEY)}</div>}>
                <PriceChart
                  symbol={ws.account.symbol} timeframe={ws.account.timeframe} onTimeframeChange={ws.account.setTimeframe}
                  accountId={ws.account.accountId}
                  trades={ws.backtest.chartTrades}
                />
              </WorkspaceErrorBoundary>
            ) : (
              <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '100%', color: 'var(--ant-color-text-secondary)', fontSize: 14 }}>
                {t(SELECT_SYMBOL_HINT_KEY, 'Select a trading account and symbol to view chart')}
              </div>
            )}

          </div>

          {/* Bottom panel: Positions + History */}
          <ChartBottomPanel
            positions={ws.quickTrade.allPositions}
            recentTrades={ws.quickTrade.qtRecentTrades}
            onClosePosition={ws.quickTrade.handleClosePosition}
            collapsed={ws.layout.bottomPanelCollapsed}
            onToggleCollapsed={() => ws.layout.setBottomPanelCollapsed(!ws.layout.bottomPanelCollapsed)}
          />
        </div>

        {/* ── RIGHT COLUMN: Tabbed panel (380px) — Chat / Results / Code ── */}
        <RightPanel
          tab={ws.layout.rightTab}
          onTabChange={ws.layout.setRightTab}
          symbol={ws.account.symbol}
          timeframe={ws.account.timeframe}
          sessionId={sessionId}
          accountId={ws.account.accountId}
          onApplyCode={(code) => { ws.code.setCode(code); ws.layout.setRightTab('code'); }}
          onValidateResult={(result) => ws.backtest.runner.handleValidationResult(result)}
          code={ws.code.code}
          onCodeChange={ws.code.setCode}
          onSaveCode={() => ws.code.setSaveModalOpen(true)}
          backtest={{ metrics: ws.backtest.metrics, status: ws.backtest.status, run: ws.backtest.run }}
          quickTrade={{
            accountId: ws.account.accountId,
            symbol: ws.account.symbol,
            accountMeta: ws.account.selectedAccountMeta,
            collapsed: ws.layout.quickTradeCollapsed,
            onToggle: () => ws.layout.setQuickTradeCollapsed(!ws.layout.quickTradeCollapsed),
            onClosePosition: ws.quickTrade.handleClosePosition,
          }}
        />
      </div>

      <SaveTemplateWrapper ws={ws} />
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
