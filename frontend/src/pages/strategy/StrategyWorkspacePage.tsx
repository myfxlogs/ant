import { Suspense, lazy, useRef, useCallback } from 'react';
import { Collapse, Grid } from 'antd';
import { DoubleRightOutlined, DoubleLeftOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next'
import { BACKTEST_RESULTS_LABEL_KEY, CHART_ERROR_KEY, CODE_KEY, COMPLETED_STATUS_KEY, NO_OPEN_POSITIONS_KEY, OPEN_POSITIONS_KEY, RUNNING_STATUS_KEY, SELECT_SYMBOL_HINT_KEY, SMART_TUNING_KEY, TEMPLATE_TITLE_KEY, TITLE_KEY } from '@/gen/ant/v1/i18n/strategy_workspace_keys';
;
import { useStrategyWorkspaceState, DATE_PRESETS } from './hooks/useStrategyWorkspaceState';
import WorkspaceCodePanel from './components/workspace/WorkspaceCodePanel';
import WorkspaceBacktestPanel from './components/workspace/WorkspaceBacktestPanel';
import WorkspaceTemplateManager from './components/workspace/WorkspaceTemplateManager';
import WorkspaceToolbar from './components/workspace/WorkspaceToolbar';
import BacktestParamsCard from './components/workspace/BacktestParamsCard';
import MiniPositionsTable from './components/workspace/MiniPositionsTable';
import AICodePanel from '@/components/strategy/AICodePanel';
import { useAuthStore } from '@/stores/authStore';
import PriceChart from '@/components/chart/PriceChart';
import BacktestRunDrawer from '@/components/strategy/BacktestRunDrawer';
import BacktestHistoryModal from './components/workspace/BacktestHistoryModal';
import WorkspaceErrorBoundary from './components/workspace/WorkspaceErrorBoundary';
import MobileGuard from './components/workspace/MobileGuard';
import { useWorkspaceSession } from './hooks/useWorkspaceSession';
import { CODE_PANEL_WIDTH, POSITIONS_PANEL_WIDTH, C, QuickTradeSection, SaveTemplateWrapper } from './WorkspaceLayout';

const SaveTemplateModal = lazy(() => import('@/components/strategy/SaveTemplateModal'));

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

  // Click-outside: auto-collapse code panel when clicking main content area
  const codePanelRef = useRef<HTMLDivElement>(null);
  const handleContentClick = useCallback((e: React.MouseEvent) => {
    if (!ws.layout.codePanelVisible) return;
    if (codePanelRef.current && !codePanelRef.current.contains(e.target as Node)) {
      ws.layout.setCodePanelVisible(false);
    }
  }, [ws.layout.codePanelVisible, ws.layout.setCodePanelVisible]);

  // The workspace uses fixed-width panels (750+520 px) and is unusable on narrow screens.
  if (!screens.lg) return <MobileGuard />;

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: 'calc(100vh - 112px)', background: '#fff' }}>
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
        codePanelVisible={ws.layout.codePanelVisible} onToggleCodePanel={() => ws.layout.setCodePanelVisible(!ws.layout.codePanelVisible)}
        onCloseCodePanel={() => ws.layout.setCodePanelVisible(false)}
        quickTradeVisible={ws.layout.quickTradeVisible} onToggleQuickTrade={() => ws.layout.setQuickTradeVisible(!ws.layout.quickTradeVisible)}
      />

      {/* ═══ BODY: Chart + Backtest (code overlays chart when open) ═══ */}
      <div style={{ display: 'flex', flex: '1 1 auto', overflow: 'hidden', minHeight: 0, position: 'relative' }}>
        {/* ── Code toggle strip (always in flow) ── */}
        <div onClick={() => ws.layout.setCodePanelVisible(!ws.layout.codePanelVisible)} role="button" tabIndex={0}
          onKeyUp={(e) => e.key === 'Enter' && ws.layout.setCodePanelVisible(!ws.layout.codePanelVisible)}
          style={{
            width: 32, minWidth: 32, flex: '0 0 32px', display: 'flex', flexDirection: 'column',
            alignItems: 'center', justifyContent: 'center', gap: 8, cursor: 'pointer', zIndex: 10,
            background: ws.layout.codePanelVisible
              ? 'linear-gradient(180deg, #1677ff 0%, #0958d9 100%)'
              : 'linear-gradient(180deg, #f8fafc 0%, #eef2f7 100%)',
            borderRight: '1px solid #e2e8f0',
            padding: '14px 0', transition: 'background 0.2s',
          }}>
          {ws.layout.codePanelVisible
            ? <DoubleLeftOutlined style={{ fontSize: 14, color: '#fff' }} />
            : <DoubleRightOutlined style={{ fontSize: 14 }} />
          }
          <span style={{ fontSize: 10, writingMode: 'vertical-rl', fontWeight: 500,
            color: ws.layout.codePanelVisible ? '#fff' : 'inherit' }}>{t(CODE_KEY)}</span>
        </div>

        {/* ── Expanded code panel (overlays chart, stays within workspace) ── */}
        {ws.layout.codePanelVisible && (
          <div ref={codePanelRef} style={{
            position: 'absolute', left: 32, top: 0, bottom: 0, zIndex: 100,
            width: CODE_PANEL_WIDTH, overflowY: 'auto',
            background: '#fcfdfd', borderRight: '1px solid ' + C.border,
            boxShadow: '4px 0 24px rgba(0,0,0,0.1)',
            padding: '12px 14px', display: 'flex', flexDirection: 'column', gap: 12,
          }}>
            <AICodePanel accountId={ws.account.accountId} symbol={ws.account.symbol} timeframe={ws.account.timeframe} sessionId={sessionId} onApply={(code, prevCode) => { ws.code.setCode(code); }} />
          </div>
        )}

        {/* ── Positions overlay panel (right side, next to Quick Trade) ── */}
        {ws.layout.positionsPanelVisible && (
          <div style={{
            position: 'absolute', right: ws.layout.quickTradeVisible ? 310 : 0, top: 0, bottom: 0, zIndex: 100,
            width: POSITIONS_PANEL_WIDTH, overflowY: 'auto',
            background: '#fcfdfd', borderLeft: '1px solid ' + C.border,
            boxShadow: '-4px 0 24px rgba(0,0,0,0.1)',
            display: 'flex', flexDirection: 'column',
          }}>
            <div style={{
              display: 'flex', justifyContent: 'space-between', alignItems: 'center',
              padding: '8px 14px', flexShrink: 0,
              background: 'linear-gradient(180deg, #fff 0%, #f1f5f9 100%)',
              borderBottom: '1px solid ' + C.border,
            }}>
              <span style={{ fontSize: 13, fontWeight: 700, color: '#0f172a' }}>
                {t(OPEN_POSITIONS_KEY, { count: ws.quickTrade.allPositions.length })}
              </span>
              <span onClick={() => ws.layout.setPositionsPanelVisible(false)} role="button" tabIndex={0}
                onKeyUp={e => e.key === 'Enter' && ws.layout.setPositionsPanelVisible(false)}
                style={{ cursor: 'pointer', color: '#94a3b8', fontSize: 16, lineHeight: 1 }}>✕</span>
            </div>
            <div style={{ flex: 1, overflowY: 'auto', padding: '8px 14px' }}>
              {ws.quickTrade.allPositions.length > 0 ? (
                <MiniPositionsTable positions={ws.quickTrade.allPositions} onClosePosition={ws.quickTrade.handleClosePosition} />
              ) : (
                <div style={{ textAlign: 'center', padding: 40, color: '#8c8c8c', fontSize: 13 }}>
                  {t(NO_OPEN_POSITIONS_KEY)}
                </div>
              )}
            </div>
          </div>
        )}

        {/* ── MIDDLE: Chart + Backtest ── */}
        <div onClick={handleContentClick} style={{ flex: '1 1 0', minWidth: 0, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
          <div style={{ flex: '1 1 0', minHeight: 0, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
            {ws.account.symbol ? (
              <WorkspaceErrorBoundary fallback={<div style={{ display:'flex',alignItems:'center',justifyContent:'center',height:'100%',color:'#8c8c8c' }}>{t(CHART_ERROR_KEY)}</div>}>
                <PriceChart
                  symbol={ws.account.symbol} timeframe={ws.account.timeframe} onTimeframeChange={ws.account.setTimeframe}
                  accountId={ws.account.accountId}
                  trades={ws.backtest.chartTrades}
                />
              </WorkspaceErrorBoundary>
            ) : (
              <div style={{
                display: 'flex', alignItems: 'center', justifyContent: 'center',
                height: '100%', color: '#6b7280',
                border: '1px dashed rgba(0,0,0,0.12)', borderRadius: 8, margin: 12,
              }}>
                {t(SELECT_SYMBOL_HINT_KEY, 'Select a trading account and symbol to view chart')}
              </div>
            )}
          </div>

          <div style={{ flexShrink: 0, borderTop: '1px solid #e8e8e8', overflowY: 'auto' }}>
            <div style={{ display: 'flex' }}>
              {/* Left: Backtest Params */}
              <div style={{ flex: 1, minWidth: 0 }}>
                <BacktestParamsCard
                  templates={ws.templates}
                  initialCapital={ws.backtest.initialCapital} onInitialCapitalChange={ws.backtest.setInitialCapital}
                  leverage={ws.backtest.leverage} onLeverageChange={ws.backtest.setLeverage}
                  commission={ws.backtest.commission} onCommissionChange={ws.backtest.setCommission}
                  slippage={ws.backtest.slippage} onSlippageChange={ws.backtest.setSlippage}
                  startDate={ws.backtest.startDate} onStartDateChange={ws.backtest.setStartDate}
                  endDate={ws.backtest.endDate} onEndDateChange={ws.backtest.setEndDate}
                  tradeDirection={ws.backtest.tradeDirection} onTradeDirectionChange={ws.backtest.setTradeDirection}
                  strictMode={ws.backtest.strictMode} onStrictModeChange={ws.backtest.setStrictMode}
                  canRun={Boolean(ws.code.code && ws.account.symbol)}
                  running={ws.backtest.submitting} onRunBacktest={ws.backtest.run}
                  datePresets={DATE_PRESETS} datePresetKey={ws.backtest.datePreset}
                  onApplyDatePreset={ws.backtest.applyDatePreset}
                  expanded={ws.backtest.paramsExpanded} onExpandedChange={ws.backtest.setParamsExpanded}
                  strategyDirectives={ws.backtest.strategyDirectives}
                  onApplyPreset={ws.backtest.applyPreset}
                  timeframeWarning={ws.backtest.getTimeframeWarning(ws.account.timeframe, DATE_PRESETS.find(p => p.key === ws.backtest.datePreset)?.months ?? 3)}
                  onApplyDefaults={ws.backtest.applyDefaults}
                  onOpenHistory={ws.history.open}
                />
              </div>

              {/* Right: Backtest Results */}
              <div style={{ flex: 1, minWidth: 0, borderLeft: '1px solid #e8e8e8', background: '#fafbfc', overflowY: 'auto' }}>
                <div onClick={() => ws.backtest.setResultsExpanded(!ws.backtest.resultsExpanded)} role="button" tabIndex={0}
                  onKeyUp={e => e.key === 'Enter' && ws.backtest.setResultsExpanded(!ws.backtest.resultsExpanded)}
                  style={{
                    display: 'flex', alignItems: 'center', justifyContent: 'space-between',
                    padding: '8px 14px', cursor: 'pointer', userSelect: 'none',
                    background: 'linear-gradient(180deg, #f8fafc 0%, #f1f5f9 100%)',
                  }}>
                  <span style={{ fontSize: 12, fontWeight: 700, color: '#262626' }}>
                    {ws.tuning.subTab === 'tuning' ? t(SMART_TUNING_KEY) : t(BACKTEST_RESULTS_LABEL_KEY)}
                    {ws.backtest.status === 'running' && <span style={{ color: '#1890ff', marginLeft: 8, fontSize: 11 }}>{t(RUNNING_STATUS_KEY)}</span>}
                    {ws.backtest.status === 'completed' && <span style={{ color: '#26a69a', marginLeft: 8, fontSize: 11 }}>{t(COMPLETED_STATUS_KEY)}</span>}
                  </span>
                  <span style={{ fontSize: 10, color: C.muted }}>{ws.backtest.resultsExpanded ? '▲' : '▼'}</span>
                </div>
                {ws.backtest.resultsExpanded && (
                  <div style={{ padding: '8px 14px' }}>
                    <WorkspaceBacktestPanel
                      status={ws.backtest.status} metrics={ws.backtest.metrics}
                      executionAssumptions={ws.backtest.executionAssumptions}
                      errorMessage={ws.backtest.error}
                      onAIOptimize={ws.ai.optimize}
                      code={ws.code.code} onApplyTunedParams={ws.ai.applyTunedParams}
                      subTab={ws.tuning.subTab} onSubTabChange={ws.tuning.setSubTab}
                      tuneMethod={ws.tuning.method} onTuneMethodChange={ws.tuning.setMethod}
                      sweepDimensions={ws.tuning.sweepDimensions} onToggleDimension={ws.tuning.toggleDimension}
                      enabledSweepDims={ws.tuning.enabledDims} cartesianSize={ws.tuning.cartesianSize}
                      tuningRunning={ws.tuning.running} canRunTuning={Boolean(ws.code.code && ws.account.symbol)}
                      onRunTuning={ws.tuning.run}
                      gateLoading={ws.gate.loading} gateGates={ws.gate.gates}
                      gateSummary={ws.gate.summary} gateError={ws.gate.error}
                      onRunGate={ws.gate.run}
                    />
                  </div>
                )}
              </div>
            </div>
          </div>
        </div>

        <QuickTradeSection visible={ws.layout.quickTradeVisible} ws={ws} />
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
