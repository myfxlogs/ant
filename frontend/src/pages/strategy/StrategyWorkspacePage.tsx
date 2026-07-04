import { Grid, Select } from 'antd';
import { useTranslation } from 'react-i18next';
import { CHART_ERROR_KEY, SELECT_SYMBOL_HINT_KEY, TITLE_KEY } from '@/gen/ant/v1/i18n/strategy_workspace_keys';
import { useStrategyWorkspaceState } from './hooks/useStrategyWorkspaceState';
import WorkspaceTemplateManager from './components/workspace/WorkspaceTemplateManager';
import WorkspaceToolbar from './components/workspace/WorkspaceToolbar';
import BacktestResultsCard from '@/components/strategy/BacktestResultsCard';
import QuickTradePanel from '@/components/chart/QuickTradePanel';
import ChartBottomPanel from '@/components/chart/ChartBottomPanel';
import SymbolPicker from '@/components/chart/SymbolPicker';
import { useAuthStore } from '@/stores/authStore';
import PriceChart from '@/components/chart/PriceChart';
import BacktestRunDrawer from '@/components/strategy/BacktestRunDrawer';
import BacktestHistoryModal from './components/workspace/BacktestHistoryModal';
import WorkspaceErrorBoundary from './components/workspace/WorkspaceErrorBoundary';
import MobileGuard from './components/workspace/MobileGuard';
import RightPanel from './components/workspace/RightPanel';
import { useWorkspaceSession } from './hooks/useWorkspaceSession';
import { SaveTemplateWrapper } from './WorkspaceLayout';

const COL_BORDER = '1px solid var(--ant-color-border)';
const TIMEFRAMES = ['1m', '5m', '15m', '30m', '1h', '4h', '1d', '1w'];

export default function StrategyWorkspacePage() {
  const { t } = useTranslation();
  const screens = Grid.useBreakpoint();
  const ws = useStrategyWorkspaceState();
  const userId = useAuthStore(s => s.user?.id);
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

          {/* Section: Symbol & Timeframe */}
          <div style={{ borderTop: COL_BORDER, padding: 10, flexShrink: 0 }}>
            <div style={{ fontSize: 10, fontWeight: 700, color: 'var(--ant-color-text-tertiary)', textTransform: 'uppercase', letterSpacing: 0.5, marginBottom: 6 }}>
              🔍 {t('strategy.workspace.symbol', 'Symbol')}
            </div>
            <SymbolPicker accountId={ws.account.accountId} value={ws.account.symbol} onChange={ws.account.setSymbol} style={{ width: '100%' }} />
            <div style={{ fontSize: 10, fontWeight: 700, color: 'var(--ant-color-text-tertiary)', textTransform: 'uppercase', letterSpacing: 0.5, margin: '8px 0 4px' }}>
              {t('strategy.workspace.timeframe', 'Timeframe')}
            </div>
            <Select size="small" style={{ width: '100%' }} value={ws.account.timeframe} onChange={ws.account.setTimeframe}
              options={TIMEFRAMES.map(tf => ({ value: tf, label: tf.toUpperCase() }))} />
          </div>
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

            {/* Quick Trade floating overlay (top-right) */}
            {ws.account.symbol && (
              <div style={{
                position: 'absolute', top: 48, right: 12, zIndex: 10,
                width: 210,
              }}>
                <div style={{
                  background: 'var(--ant-color-bg-elevated)', border: '1px solid var(--ant-color-border)',
                  borderRadius: 8, boxShadow: '0 4px 16px rgba(0,0,0,0.4)', overflow: 'hidden',
                }}>
                  <div
                    onClick={() => ws.layout.setQuickTradeCollapsed(!ws.layout.quickTradeCollapsed)}
                    style={{
                      padding: '8px 12px', cursor: 'pointer', userSelect: 'none',
                      display: 'flex', alignItems: 'center', justifyContent: 'space-between',
                      fontSize: 12, fontWeight: 700, color: 'var(--ant-color-text)',
                      borderBottom: ws.layout.quickTradeCollapsed ? 'none' : '1px solid var(--ant-color-border)',
                      background: 'var(--ant-color-bg-layout)',
                    }}
                  >
                    <span>⚡ Quick Trade</span>
                    <span style={{ fontSize: 10, color: 'var(--ant-color-text-tertiary)' }}>
                      {ws.layout.quickTradeCollapsed ? '▼' : '▲'}
                    </span>
                  </div>
                  {!ws.layout.quickTradeCollapsed && (
                    <div style={{ padding: 10 }}>
                      <QuickTradePanel
                        accountId={ws.account.accountId} symbol={ws.account.symbol}
                        accountMeta={ws.account.selectedAccountMeta}
                        allPositions={[]}
                        positions={[]}
                        recentTrades={[]}
                        onClosePosition={ws.quickTrade.handleClosePosition}
                        onToggleAllPositions={() => ws.layout.setBottomPanelCollapsed(false)}
                      />
                    </div>
                  )}
                </div>
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
          onRunBacktest={ws.backtest.run}
          backtestStatus={ws.backtest.status}
          metrics={ws.backtest.metrics}
          backtestStatus2={ws.backtest.status}
          code={ws.code.code}
          onCodeChange={ws.code.setCode}
          onSaveCode={() => ws.code.setSaveModalOpen(true)}
          onRunBacktest2={ws.backtest.run}
        />
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
