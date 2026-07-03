import { Suspense, lazy } from 'react';
import { Grid, Tabs } from 'antd';
import { LineChartOutlined, CodeOutlined, ExperimentOutlined, BarChartOutlined, ThunderboltOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next'
import { CHART_ERROR_KEY, SELECT_SYMBOL_HINT_KEY, TITLE_KEY } from '@/gen/ant/v1/i18n/strategy_workspace_keys';
import { useStrategyWorkspaceState } from './hooks/useStrategyWorkspaceState';
import WorkspaceCodePanel from './components/workspace/WorkspaceCodePanel';
import BacktestPanel from '@/components/backtest/BacktestPanel';
import WorkspaceTemplateManager from './components/workspace/WorkspaceTemplateManager';
import WorkspaceToolbar from './components/workspace/WorkspaceToolbar';
import MiniPositionsTable from './components/workspace/MiniPositionsTable';
import StrategyChat from '@/components/strategy/StrategyChat';
import QuickTradePanel from '@/components/chart/QuickTradePanel';
import { useAuthStore } from '@/stores/authStore';
import PriceChart from '@/components/chart/PriceChart';
import BacktestRunDrawer from '@/components/strategy/BacktestRunDrawer';
import BacktestHistoryModal from './components/workspace/BacktestHistoryModal';
import WorkspaceErrorBoundary from './components/workspace/WorkspaceErrorBoundary';
import WorkspaceBacktestPanel from './components/workspace/WorkspaceBacktestPanel';
import MobileGuard from './components/workspace/MobileGuard';
import { useWorkspaceSession } from './hooks/useWorkspaceSession';
import { SaveTemplateWrapper } from './WorkspaceLayout';
import type { CenterTab, RightTab } from '@/stores/workspaceStore';

const SaveTemplateModal = lazy(() => import('@/components/strategy/SaveTemplateModal'));

const COL_BORDER = '1px solid var(--ant-color-border)';

export default function StrategyWorkspacePage() {
  const { t } = useTranslation();
  const screens = Grid.useBreakpoint();
  const ws = useStrategyWorkspaceState();
  const userId = useAuthStore(s => s.user?.id);
  const { sessionId, chatHistory } = useWorkspaceSession(
    userId,
    ws.account.symbol,
    ws.account.timeframe,
    ws.code.lastSavedId,
  );

  if (!screens.lg) return <MobileGuard />;

  const centerTab = ws.layout.centerTab as CenterTab;
  const rightTab = ws.layout.rightTab as RightTab;

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: 'calc(100vh - 112px)' }}>
      {/* Title bar */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '8px 0 12px' }}>
        <h2 style={{ margin: 0, fontSize: 16, fontWeight: 600 }}>{t(TITLE_KEY, 'Strategy Workspace')}</h2>
      </div>

      {/* ═══ TOP TOOLBAR ═══ */}
      <WorkspaceToolbar
        accounts={ws.account.activeAccounts} accountId={ws.account.accountId} onAccountChange={ws.account.handleAccountChange}
        symbol={ws.account.symbol} onSymbolChange={ws.account.setSymbol}
        accountInfo={ws.account.accountInfo} positionCount={ws.quickTrade.positionCount}
        busy={ws.backtest.submitting || ws.tuning.running}
        positionsCount={ws.quickTrade.allPositions.length}
        onTogglePositionsPanel={() => ws.layout.setPositionsPanelVisible(!ws.layout.positionsPanelVisible)}
      />

      {/* ═══ BODY: Three-column flex layout ═══ */}
      <div style={{ display: 'flex', flex: '1 1 auto', overflow: 'hidden', minHeight: 0 }}>

        {/* ── LEFT COLUMN (260px): Strategy List ── */}
        <div style={{ width: 260, minWidth: 260, flexShrink: 0, borderRight: COL_BORDER, display: 'flex', flexDirection: 'column', overflow: 'hidden', padding: 12 }}>
          <WorkspaceTemplateManager
            templates={ws.templates.list}
            loading={ws.templates.loading}
            loadedTemplate={ws.templates.list.find((t: any) => t.id === ws.templates.selectedId) || null}
            onLoad={ws.templates.onSelect}
            onSaveAs={() => ws.code.setSaveModalOpen(true)}
          />
        </div>

        {/* ── CENTER COLUMN (flex:1): Tabs + AI Chat ── */}
        <div style={{ flex: '1 1 0', minWidth: 0, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
          {/* Tabbed content area (flex:1) */}
          <div style={{ flex: '1 1 0', minHeight: 0, overflow: 'hidden', display: 'flex', flexDirection: 'column' }}>
            <Tabs
              className="center-tabs"
              activeKey={centerTab}
              onChange={(k) => ws.layout.setCenterTab(k as CenterTab)}
              size="small"
              items={[
                {
                  key: 'design',
                  label: <span><LineChartOutlined /> {t('strategy.workspace.design', 'Design')}</span>,
                  children: (
                    <div style={{ flex: 1, overflow: 'hidden', display: 'flex', flexDirection: 'column' }}>
                      {ws.account.symbol ? (
                        <WorkspaceErrorBoundary fallback={<div style={{ display:'flex',alignItems:'center',justifyContent:'center',height:'100%',color:'var(--ant-color-text-tertiary)' }}>{t(CHART_ERROR_KEY)}</div>}>
                          <PriceChart
                            symbol={ws.account.symbol} timeframe={ws.account.timeframe} onTimeframeChange={ws.account.setTimeframe}
                            accountId={ws.account.accountId}
                            trades={ws.backtest.chartTrades}
                          />
                        </WorkspaceErrorBoundary>
                      ) : (
                        <div style={{
                          display: 'flex', alignItems: 'center', justifyContent: 'center',
                          height: '100%', color: 'var(--ant-color-text-secondary)',
                          border: '1px dashed var(--ant-color-border)', borderRadius: 8, margin: 12,
                        }}>
                          {t(SELECT_SYMBOL_HINT_KEY, 'Select a trading account and symbol to view chart')}
                        </div>
                      )}
                    </div>
                  ),
                },
                {
                  key: 'code',
                  label: <span><CodeOutlined /> {t('strategy.workspace.code', 'Code')}</span>,
                  children: (
                    <div style={{ flex: 1, overflow: 'auto', padding: 12 }}>
                      <WorkspaceCodePanel
                        code={ws.code.code} onCodeChange={ws.code.setCode}
                        validating={ws.code.validating} onValidate={ws.code.validateCode}
                        validationResult={ws.code.validationResult}
                        onRunBacktest={ws.backtest.run} backtestSubmitting={ws.backtest.submitting}
                        canSave={ws.code.canSave} onSave={ws.code.handleSave} onCopy={ws.code.handleCopy}
                        onAskAI={ws.ai.askForValidation}
                        onAutoFix={ws.ai.autoFix} autoFixing={ws.ai.autoFixing}
                        autoFixDebug={ws.ai.autoFixDebug} onDismissDebug={ws.ai.dismissDebug}
                      />
                    </div>
                  ),
                },
                {
                  key: 'backtest',
                  label: <span><ExperimentOutlined /> {t('strategy.workspace.backtest', 'Backtest')}</span>,
                  children: (
                    <div style={{ flex: 1, overflow: 'auto', padding: 12 }}>
                      <BacktestPanel
                        runner={ws.backtest.runner}
                        inputs={{ strategyCode: ws.code.code, accountId: ws.account.accountId, symbol: ws.account.symbol, timeframe: ws.account.timeframe, templateId: ws.templates.selectedId || undefined }}
                        templates={ws.templates}
                        collapsed={ws.backtest.btCollapsed ?? false}
                        onToggleCollapsed={() => ws.backtest.setBtCollapsed?.(!ws.backtest.btCollapsed)}
                        onOpenHistory={ws.history.open}
                        onAIOptimize={ws.ai.optimize}
                        code={ws.code.code}
                        onApplyTunedParams={(code: string) => { ws.code.setCode(code); }}
                      />
                    </div>
                  ),
                },
              ]}
            />
          </div>

          {/* AI Chat panel (fixed height at bottom of center column) */}
          <div style={{ height: 300, flexShrink: 0, borderTop: COL_BORDER, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
            <StrategyChat
              symbol={ws.account.symbol}
              timeframe={ws.account.timeframe}
              sessionId={sessionId}
              accountId={ws.account.accountId}
              onApplyCode={(code) => ws.code.setCode(code)}
              onValidateResult={(result) => ws.backtest.runner.handleValidationResult(result)}
              onRunBacktest={ws.backtest.run}
              backtestStatus={ws.backtest.status}
            />
          </div>
        </div>

        {/* ── RIGHT COLUMN (340px): Results / Quick Trade ── */}
        <div style={{ width: 340, minWidth: 340, flexShrink: 0, borderLeft: COL_BORDER, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
          <Tabs
            className="center-tabs"
            activeKey={rightTab}
            onChange={(k) => ws.layout.setRightTab(k as RightTab)}
            size="small"
            items={[
              {
                key: 'results',
                label: <span><BarChartOutlined /> {t('strategy.workspace.results', 'Results')}</span>,
                children: (
                  <div style={{ flex: 1, overflow: 'auto', padding: 12 }}>
                    <WorkspaceBacktestPanel
                      status={ws.backtest.status}
                      metrics={ws.backtest.metrics}
                      executionAssumptions={ws.backtest.executionAssumptions}
                      errorMessage={ws.backtest.error}
                      onAIOptimize={ws.ai.optimize}
                      subTab={ws.backtest.activeTab}
                      onSubTabChange={ws.backtest.setActiveTab}
                      tuneMethod={ws.tuning.method}
                      onTuneMethodChange={ws.tuning.setMethod}
                      sweepDimensions={ws.tuning.sweepDimensions}
                      onToggleDimension={ws.tuning.toggleDimension}
                      enabledSweepDims={ws.tuning.enabledDims}
                      cartesianSize={ws.tuning.cartesianSize}
                      code={ws.code.code}
                      onApplyTunedParams={(code: string) => { ws.code.setCode(code); }}
                      tuningRunning={ws.tuning.running}
                      canRunTuning={!!ws.code.code}
                      onRunTuning={ws.tuning.run}
                      gateLoading={ws.gate.loading}
                      gateGates={ws.gate.gates}
                      gateSummary={ws.gate.summary}
                      gateError={ws.gate.error}
                      onRunGate={() => ws.gate.run(ws.backtest.runId, () => ws.layout.setCenterTab('backtest'))}
                    />
                  </div>
                ),
              },
              {
                key: 'quicktrade',
                label: <span><ThunderboltOutlined /> {t('strategy.workspace.quickTrade', 'Quick Trade')}</span>,
                children: (
                  <div style={{ flex: 1, overflow: 'auto', padding: 12 }}>
                    <QuickTradePanel
                      accountId={ws.account.accountId} symbol={ws.account.symbol}
                      accountMeta={ws.account.selectedAccountMeta}
                      allPositions={ws.quickTrade.allPositions}
                      positions={ws.quickTrade.qtPositions}
                      recentTrades={ws.quickTrade.qtRecentTrades}
                      onClosePosition={ws.quickTrade.handleClosePosition}
                      onToggleAllPositions={() => ws.layout.setPositionsPanelVisible(!ws.layout.positionsPanelVisible)}
                    />
                  </div>
                ),
              },
            ]}
          />

          {/* Positions overlay panel (within right column) */}
          {ws.layout.positionsPanelVisible && (
            <div style={{
              position: 'absolute', right: 340, top: 0, bottom: 0, zIndex: 100,
              width: 360, overflowY: 'auto',
              background: 'var(--ant-color-bg-container)',
              borderLeft: COL_BORDER,
              boxShadow: '-4px 0 24px rgba(0,0,0,0.2)',
              display: 'flex', flexDirection: 'column',
            }}>
              <div style={{
                display: 'flex', justifyContent: 'space-between', alignItems: 'center',
                padding: '8px 14px', flexShrink: 0,
                borderBottom: COL_BORDER,
              }}>
                <span style={{ fontSize: 13, fontWeight: 700 }}>
                  {t('strategy.workspace.openPositions', { count: ws.quickTrade.allPositions.length, defaultValue: 'Open Positions' })}
                </span>
                <span onClick={() => ws.layout.setPositionsPanelVisible(false)} role="button" tabIndex={0}
                  onKeyUp={e => e.key === 'Enter' && ws.layout.setPositionsPanelVisible(false)}
                  style={{ cursor: 'pointer', fontSize: 16, lineHeight: 1 }}>✕</span>
              </div>
              <div style={{ flex: 1, overflowY: 'auto', padding: '8px 14px' }}>
                {ws.quickTrade.allPositions.length > 0 ? (
                  <MiniPositionsTable positions={ws.quickTrade.allPositions} onClosePosition={ws.quickTrade.handleClosePosition} />
                ) : (
                  <div style={{ textAlign: 'center', padding: 40, color: 'var(--ant-color-text-tertiary)', fontSize: 13 }}>
                    {t('strategy.workspace.noOpenPositions', 'No open positions')}
                  </div>
                )}
              </div>
            </div>
          )}
        </div>
      </div>

      <SaveTemplateWrapper ws={ws} />
      <BacktestHistoryModal
        open={ws.history.modalOpen}
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
        onClose={ws.history.closeModal}
      />
      <BacktestRunDrawer
        open={ws.history.drawerOpen} runId={ws.history.runId}
        onClose={ws.history.close}
        onCancel={ws.history.close}
      />
    </div>
  );
}
